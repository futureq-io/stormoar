# Raft Log Storage #

Stormoar uses [Pebble](https://github.com/cockroachdb/pebble) to store Raft logs, Pebble is a RocksDB compatible Key-Value store.

## Compatibility ##

Pebble is a new Key-Value store implemented in Go with bidirectional compatibility with RocksDB.

## Pebble ##

Pebble is used by default, no configuration is required.

## RocksDB ##

Dragonboat removed its RocksDB storage backend in v3.4.

## Use custom storage solution ##

You can extend Stormoar to use your preferred storage solution to store Raft logs -

* implement the ILogDB interface defined in the github.com/futureq-io/stormoar/v4/raftio package
* pass a factory function that creates such a custom Log DB instance to the LogDBFactory field of your NodeHostConfig.Expert instance

## Tan

Tan is an alternative Raft log storage engine. To use it, set the LogDB factory to `tan.Factory` from `github.com/futureq-io/stormoar/v4/plugin/tan`.
