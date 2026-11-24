# Examples

The [single-node example](single-node/main.go) starts a Stormoar shard, proposes
a value, and reads it back through Raft. Run it from the repository root.

```sh
go run ./examples/single-node
```

It uses loopback port 26011 and a temporary data directory. The Test workflow
runs this example against the Stormoar checkout.

The upstream [Dragonboat examples](https://github.com/lni/dragonboat-example)
are historical references for multiple shards, on-disk state machines, and
optimistic locking.
