# Using Stormoar

This guide is adapted from Dragonboat's usage overview. It assumes familiarity
with Go and introduces the Raft concepts needed to use Stormoar.

## Replicated state machines

Running replicas on separate servers lets a service keep operating when a
server fails. A Raft shard can make progress while a majority of its voting
replicas are online and can communicate. For example, a shard with three voting
replicas can tolerate one failure.

An application submits writes as proposals. Raft assigns each proposal a
position in the log and replicates log entries to the members of the shard.
Once proposals are committed, each replica applies them in log order. Replicas
that start from the same state and apply the same deterministic updates reach
the same state at the same log index. The diagram uses an integer `X` as the
application state.

![Three replicas applying entries from a Raft log to their state machines](rsm.png)

A linearizable read sees the effect of writes that completed before the read
started. If one read finishes before another starts, the second read observes
the same or a later state.

## Terms

| Term | Meaning |
| --- | --- |
| Raft group or shard | An independent replicated state machine identified by a user-assigned 64-bit `ShardID`. Each shard ID must be unique within the deployment. |
| Node or replica | A member of a shard identified by a user-assigned 64-bit `ReplicaID` that is unique within that shard. |
| Initial members | The set of replicas that bootstrap the shard. |
| Leader | The elected replica that coordinates log replication and the quorum checks needed for linearizable reads. |
| Snapshot | A copy of the state machine at a specific log index that can be used to restore or catch up a replica. |
| NodeHost | The process-level handle that starts, stops, and manages local replicas of one or more shards. |

## State machine interfaces

Implement one of the interfaces in the
[`statemachine` package](https://pkg.go.dev/github.com/futureq-io/stormoar/v4/statemachine).
Stormoar calls `Update` to apply committed proposals and `Lookup` to query
application state. A replicated key-value store, for example, implements its
own key-value operations in these methods.

### In-memory state machines

An `IStateMachine` typically keeps its live state in memory. Start it with empty
state on restart. Stormoar restores the latest snapshot with
`RecoverFromSnapshot` and replays subsequent committed entries through
`Update`. Any local files managed by this kind of state machine must also be
safe to recreate during recovery.

Use `IConcurrentStateMachine` when the application can manage concurrent
snapshotting alongside updates. In-memory state machines are a good fit when
the working state fits comfortably in memory. Snapshot creation and replay
consume CPU, I/O, and disk space.

### On-disk state machines

An `IOnDiskStateMachine` keeps its data in application-managed persistent
storage, such as a local key-value database. Its `Open` method returns the last
durably applied log index so Stormoar can resume from the correct position. The
application manages durable updates, the applied index, and synchronization
between queries, updates, and snapshots. This approach accommodates a larger
working state and reduces the work of reconstructing state after restart.

## Starting and stopping replicas

Use `NodeHost.StartReplica`, `StartConcurrentReplica`, or `StartOnDiskReplica`
to start a replica with the corresponding state machine.

- On the first start of a shard, give every initial replica the same initial
  membership map and set `join` to `false`.
- For a replica added by a membership change, omit the initial membership map
  and set `join` to `true` on its first start.
- On restart, omit the initial membership map and set `join` to `false`. The
  replica uses its persisted bootstrap information.

`NodeHost.StopShard` stops the local replica for the shard. Use `StopReplica`
to identify the replica explicitly. Snapshot saves and restores can take
time. Stormoar closes the stop channel passed to these methods when the replica
is stopping. The implementation can respond by returning
`statemachine.ErrSnapshotStopped`.

## Writing data

`Propose` starts a write asynchronously and returns a `RequestState` when the
request is accepted for processing. Wait for its completion to determine the
result. `SyncPropose` waits for the result using the caller's context, which must
have a deadline. A successful proposal has been committed and applied.
A timeout leaves its outcome uncertain. The proposal may still take effect,
so retrying needs to account for duplicate execution.

Both methods take a client session. Use `GetNoOPSession` if the application
handles idempotency itself. On-disk state machines require a no-op session.
For a regular state machine using Stormoar's client-side session protocol,
create a session with `SyncGetSession`. Call `ProposalCompleted` after a
successful proposal to advance the session, and reuse the same session state
when retrying a timed-out proposal. Close the session with `SyncCloseSession`
when finished.

Application updates must be deterministic. If an operation needs a timestamp,
random value, or process-specific identifier, include that value in the
proposal so every replica receives the same input. Enable
`NodeHostConfig.NotifyCommit` if the application needs an additional
notification when a proposal commits, before it is applied.

## Reading data

For linearizable reads, use `SyncRead` or start an asynchronous `ReadIndex`
request. Wait for that request to complete successfully before using its
`RequestState` with `ReadLocalNode`. The ReadIndex protocol confirms the read
against a quorum and waits for the local replica to reach the required log
position. Stormoar then calls the state machine's `Lookup` method.

`NAReadLocalNode` uses byte slices to reduce allocations. It requires the
state machine to implement `statemachine.IExtended` and has the same ReadIndex
requirement. `StaleRead` queries the local state directly and can return data
that is older than the latest committed write.

## Changing shard membership

Use `RequestAddReplica` and `RequestDeleteReplica`, or the corresponding
synchronous methods, to change the membership. Raft commits and replicates
membership changes through the log. The application is responsible for
starting replicas it adds and stopping replicas it removes. A removed replica
ID cannot be added back to the same shard.

Submit one membership change at a time per shard. If multiple operators or
processes can request changes, enable `config.Config.OrderedConfigChange`.
Read the current membership with `SyncGetShardMembership` and pass its
`ConfigChangeID` to the membership operation. If the membership has
changed in the meantime, the request is rejected. Read the membership again
before deciding whether to retry.

After a replica has been permanently removed and stopped, use `RemoveData` or
`SyncRemoveData` to delete its local state. Removing the data of a replica that
still belongs to the shard can make the shard unrecoverable.

## Snapshots

A snapshot records the applied log position and current membership. For
in-memory state machines, it also includes the application state and active
client sessions. Routine snapshots of on-disk state machines contain metadata
because the state is already durable. Snapshots exported for disaster
recovery or streamed to catch up another replica include the application state.

`SaveSnapshot` writes the state for a particular log position to an
`io.Writer`. `RecoverFromSnapshot` restores it from an `io.Reader`. Stormoar
manages Raft metadata and session records. Each replica must capture the
application state associated with the snapshot index, including when updates
continue during a concurrent snapshot.

Snapshots speed up recovery, let a replica catch up without replaying a long
log, and let Stormoar compact earlier log entries. Set
`config.Config.SnapshotEntries` to configure how many applied entries trigger
an automatic snapshot. The interval depends on the application's update rate
and snapshot cost.

Use `RequestSnapshot` or `SyncRequestSnapshot` to request one manually.
`DefaultSnapshotOption` creates a snapshot managed by Stormoar. To export it,
set `SnapshotOption.Exported` and provide an existing `ExportPath`. The
application manages exported files. For a managed snapshot,
`OverrideCompactionOverhead` allows a request to specify either
`CompactionOverhead` or `CompactionIndex`. `RequestCompaction` can reclaim
disk space from entries and snapshot metadata already marked as removed.

The [`tools.ImportSnapshot` function](https://pkg.go.dev/github.com/futureq-io/stormoar/v4/tools#ImportSnapshot)
can repair a shard after a permanent loss of quorum. This is a disaster
recovery operation. Writes committed after the exported snapshot can be lost.
A temporary network partition or restart does not by itself require this
procedure. See the [DevOps guide](devops.md) for operational guidance.

## Gossip and changing addresses

By default, replica targets are stable `RaftAddress` values such as an IP
address or DNS name with a port. When addresses can change between restarts,
enable `NodeHostConfig.DefaultNodeRegistryEnabled` and configure
`NodeHostConfig.Gossip`. In this mode, use the stable NodeHost ID returned by
`NodeHost.ID` as the replica target. The decentralized gossip service maps
NodeHost IDs to the current network addresses used for Raft messages.

## Other NodeHost operations

- Add a non-voting replica to replicate and apply committed entries without
  contributing to elections or the write quorum. This is useful for adding a
  read replica or catching up a new member before promoting it. A successful
  linearizable read on the new replica shows that it applied through that read's
  index.
- Use `RequestLeaderTransfer` to request that leadership move to another
  replica.
- Use `GetNodeHostInfo` to inspect a NodeHost and `SyncGetShardMembership` to
  inspect the current membership of a shard.

See the [API reference](https://pkg.go.dev/github.com/futureq-io/stormoar/v4)
and the [examples](../examples/README.md) for code using these APIs.
