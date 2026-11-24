# Stormoar · A multi-group Raft library in Go

Stormoar is a clone of [Dragonboat](https://github.com/lni/dragonboat), the multi-group Raft library created by Lei Ni and contributors.

[![Apache 2.0 license](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Test](https://github.com/futureq-io/stormoar/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/futureq-io/stormoar/actions/workflows/test.yml)
[![Go documentation](https://pkg.go.dev/badge/github.com/futureq-io/stormoar/v4.svg)](https://pkg.go.dev/github.com/futureq-io/stormoar/v4)

## Dragonboat history ##
* 2025-07-17 Added a LLM generated wiki [here](https://deepwiki.com/lni/dragonboat).
* 2022-06-03 Dragonboat began development of v4.0 with API changes. See [CHANGELOG](CHANGELOG.md) for details.
* 2021-01-20 Dragonboat v3.3 has been released, please check [CHANGELOG](CHANGELOG.md) for all changes.

## About ##
Stormoar is a high-performance multi-group [Raft](https://raft.github.io/) [consensus](https://en.wikipedia.org/wiki/Consensus_(computer_science)) library in pure [Go](https://golang.org/).

Consensus algorithms such as Raft provides fault-tolerance by alllowing a system continue to operate as long as the majority member servers are available. For example, a Raft shard of 5 servers can make progress even if 2 servers fail. It also appears to clients as a single entity with strong data consistency always provided. All Raft replicas can be used to handle read requests for aggregated read throughput.

Stormoar handles Raft consensus so applications can focus on their own state machines. Start with the [examples](examples/README.md) and the [API reference](https://pkg.go.dev/github.com/futureq-io/stormoar/v4).

## Features ##
* Easy to use pure-Go APIs for building Raft based applications
* Feature complete and scalable multi-group Raft implementation
* Disk based and memory based state machine support
* Fully pipelined and TLS mutual authentication support, ready for high latency open environment
* Custom Raft log storage and transport support, easy to integrate with latest I/O techs
* Prometheus based health metrics support
* Built-in tool to repair Raft shards that permanently lost the quorum
* [Extensively tested](/docs/test.md) including using [Jepsen](https://aphyr.com/tags/jepsen)'s [Knossos](https://github.com/jepsen-io/knossos) linearizability checker, some results are [here](https://github.com/lni/knossos-data)

All major features covered in Diego Ongaro's [Raft thesis](https://github.com/ongardie/dissertation/blob/master/stanford.pdf) have been supported -
* leader election, log replication, snapshotting and log compaction
* membership change
* pre-vote
* ReadIndex protocol for read-only queries
* leadership transfer
* non-voting member
* witness member
* idempotent update transparent to applications
* batching and pipelining
* disk based state machine

## Performance ##
The figures below are historical benchmarks measured by the Dragonboat project.

For 3-nodes system using mid-range hardware (details [here](docs/test.md)) and in-memory state machine, when RocksDB is used as the storage engine, Dragonboat can sustain at 9 million writes per second when the payload is 16bytes each or 11 million mixed I/O per second at 9:1 read:write ratio. High throughput is maintained in geographically distributed environment. When the RTT between nodes is 30ms, 2 million I/O per second can still be achieved using a much larger number of clients.
![throughput](./docs/throughput.png)

The number of concurrent active Raft groups affects the overall throughput as requests become harder to be batched. On the other hand, having thousands of idle Raft groups has a much smaller impact on throughput.
![nodes](./docs/nodes.png)

Table below shows write latencies in millisecond, Dragonboat has <5ms P99 write latency when handling 8 million writes per second at 16 bytes each. Read latency is lower than writes as the ReadIndex protocol employed for linearizable reads doesn't require fsync-ed disk I/O.

|Ops|Payload Size|99.9% percentile|99% percentile|AVG|
|:-:|:----------:|:--:|:-:|:-:|
|1m|16|2.24|1.19|0.79|
|1m|128|11.11|1.37|0.92|
|1m|1024|71.61|25.91|3.75|
|5m|16|4.64|1.95|1.16|
|5m|128|36.61|6.55|1.96|
|8m|16|12.01|4.65|2.13|

When tested on a single Raft group, Dragonboat can sustain writes at 1.25 million per second when payload is 16 bytes each, average latency is 1.3ms and the P99 latency is 2.6ms. This is achieved when using an average of 3 cores (2.8GHz) on each server.

As visualized below, Stop-the-World pauses caused by Go1.11's GC are sub-millisecond on highly loaded systems. Such very short Stop-the-World pause time is further significantly reduced in Go 1.12. Golang's runtime.ReadMemStats reports that less than 1% of the available CPU time is used by GC on highly loaded system.
![stw](./docs/stw.png)

## Requirements ##

Go 1.27.1 or later. CI tests Linux and macOS.

## Getting started ##

Install the latest tagged Stormoar release.

```sh
go get github.com/futureq-io/stormoar/v4@latest
```

The root Go package is named `dragonboat`. To use `stormoar` in your code, give the import an alias.

```go
import stormoar "github.com/futureq-io/stormoar/v4"
```

[Usage guide](docs/overview.md) · [Storage options](docs/storage.md) · [DevOps guide](docs/devops.md) · [Testing](docs/test.md) · [Examples](examples/README.md)

For upstream Dragonboat, see its [FAQ](https://github.com/lni/dragonboat/wiki/FAQ) and [v3.3 documentation](https://github.com/lni/dragonboat/blob/release-3.3/README.md).

## Contributing ##
Report Stormoar bugs in [GitHub Issues](https://github.com/futureq-io/stormoar/issues/new). See the [contribution guide](CONTRIBUTING.md) for pull requests.

## License ##
Stormoar retains Dragonboat’s Apache License, Version 2.0. See the [license](LICENSE), [notice](NOTICE), and [third-party copyright information](docs/COPYRIGHT).
