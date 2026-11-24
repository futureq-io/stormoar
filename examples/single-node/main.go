// SPDX-License-Identifier: Apache-2.0

// This example starts a single-replica Raft shard, proposes a value, and reads it back.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	stormoar "github.com/futureq-io/stormoar/v4"
	"github.com/futureq-io/stormoar/v4/config"
	"github.com/futureq-io/stormoar/v4/statemachine"
)

type register struct {
	value string
}

func (r *register) Update(entry statemachine.Entry) (statemachine.Result, error) {
	r.value = string(entry.Cmd)
	return statemachine.Result{}, nil
}

func (r *register) Lookup(any) (any, error) {
	return r.value, nil
}

func (r *register) SaveSnapshot(w io.Writer, _ statemachine.ISnapshotFileCollection, _ <-chan struct{}) error {
	_, err := io.WriteString(w, r.value)
	return err
}

func (r *register) RecoverFromSnapshot(reader io.Reader, _ []statemachine.SnapshotFile, _ <-chan struct{}) error {
	value, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	r.value = string(value)
	return nil
}

func (r *register) Close() error { return nil }

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	const (
		shardID   = 1
		replicaID = 1
		address   = "127.0.0.1:26011"
		message   = "hello, Stormoar"
	)

	dir, err := os.MkdirTemp("", "stormoar-example-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	host, err := stormoar.NewNodeHost(config.NodeHostConfig{
		NodeHostDir:    dir,
		RaftAddress:    address,
		RTTMillisecond: 20,
	})
	if err != nil {
		return err
	}
	defer host.Close()

	members := map[uint64]stormoar.Target{replicaID: address}
	create := func(uint64, uint64) statemachine.IStateMachine { return &register{} }
	if err := host.StartReplica(members, false, create, config.Config{
		ShardID:      shardID,
		ReplicaID:    replicaID,
		ElectionRTT:  10,
		HeartbeatRTT: 1,
		CheckQuorum:  true,
	}); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Assigning the same value twice is safe if startup delays a proposal.
	session := host.GetNoOPSession(shardID)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		_, err := host.SyncPropose(ctx, session, []byte(message))
		if err == nil {
			break
		}
		if !stormoar.IsTempError(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}

	value, err := host.SyncRead(ctx, shardID, nil)
	if err != nil {
		return err
	}
	if value != message {
		return fmt.Errorf("read %q, want %q", value, message)
	}
	fmt.Println(value)
	return nil
}
