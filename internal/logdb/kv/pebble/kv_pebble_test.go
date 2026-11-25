package pebble

import (
	"slices"
	"strings"
	"testing"

	"github.com/cockroachdb/pebble/v2"
	gvfs "github.com/lni/vfs"
	"github.com/stretchr/testify/require"

	"github.com/futureq-io/stormoar/v4/config"
)

func TestAcknowledgedWritesSurviveWALRotationAndCrash(t *testing.T) {
	const (
		dir    = "/db"
		walDir = "/wal"
	)
	fs := gvfs.NewStrictMem()
	cfg := config.GetDefaultLogDBConfig()
	keys := []string{"before-rotation", "after-rotation-1", "after-rotation-2"}

	func() {
		store, err := NewKVStore(cfg, nil, dir, walDir, fs)
		require.NoError(t, err)
		defer func() { require.NoError(t, store.Close()) }()

		db := store.(*KV).db
		require.GreaterOrEqual(t, db.FormatMajorVersion(), pebble.FormatWALSyncChunks)
		require.NoError(t, store.SaveValue([]byte(keys[0]), []byte(keys[0])))
		before, err := fs.List(walDir)
		require.NoError(t, err)

		require.NoError(t, db.Flush())
		after, err := fs.List(walDir)
		require.NoError(t, err)
		require.True(t, slices.ContainsFunc(after, func(name string) bool {
			return strings.HasSuffix(name, ".log") && !slices.Contains(before, name)
		}), "WAL did not rotate: before %v, after %v", before, after)

		for _, key := range keys[1:] {
			require.NoError(t, store.SaveValue([]byte(key), []byte(key)))
		}

		// Freeze durable state before Close releases locks and stops workers.
		fs.SetIgnoreSyncs(true)
	}()
	fs.ResetToSyncedState()
	fs.SetIgnoreSyncs(false)

	reopened, err := NewKVStore(cfg, nil, dir, walDir, fs)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reopened.Close()) })
	for _, key := range keys {
		require.NoError(t, reopened.GetValue([]byte(key), func(value []byte) error {
			require.Equal(t, key, string(value), "recovered value for %q", key)
			return nil
		}))
	}
}
