package pebble

import (
	"testing"
	"time"

	gvfs "github.com/lni/vfs"
	"github.com/stretchr/testify/require"

	"github.com/futureq-io/stormoar/v4/config"
)

func TestWALMinSyncInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, 500 * time.Microsecond} {
		t.Run(interval.String(), func(t *testing.T) {
			cfg := config.GetTinyMemLogDBConfig()
			cfg.PebbleWALMinSyncInterval = interval
			store, err := NewKVStore(cfg, nil, "/db", "/wal", gvfs.NewStrictMem())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, store.Close()) })
			minSyncInterval := store.(*KV).opts.WALMinSyncInterval
			if interval == 0 {
				require.Nil(t, minSyncInterval)
			} else {
				require.NotNil(t, minSyncInterval)
				require.Equal(t, interval, minSyncInterval())
			}
		})
	}
}
