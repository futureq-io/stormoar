# Examples

The upstream [Dragonboat examples](https://github.com/lni/dragonboat-example)
cover a single Raft shard, multiple shards, on-disk state machines, and
optimistic locking. When adapting them to Stormoar, update Dragonboat v4 import
paths to `github.com/futureq-io/stormoar/v4`. The root package name is
`dragonboat`.

The Test workflow builds a pinned revision of these examples against the
Stormoar checkout.
