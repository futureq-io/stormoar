// Copyright 2017-2019 Lei Ni (nilei81@gmail.com) and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vfs

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/errors/oserror"
	pvfs "github.com/cockroachdb/pebble/v2/vfs"

	gvfs "github.com/lni/vfs"
)

// IFS is the vfs interface used by dragonboat.
type IFS = gvfs.FS

// MemFS is a memory backed file system for testing purposes.
type MemFS = gvfs.MemFS

// DefaultFS is a vfs instance using underlying OS fs.
var DefaultFS IFS = gvfs.Default

// MemStrictFS is a vfs instance using memfs.
var MemStrictFS IFS = gvfs.NewStrictMem()

// File is the file interface returned by IFS.
type File = gvfs.File

// NewMemFS creates a in-memory fs.
func NewMemFS() IFS {
	return gvfs.NewStrictMem()
}

// PebbleFS is a wrapper struct that implements the pebble/vfs.FS interface.
type PebbleFS struct {
	fs IFS
}

var _ pvfs.FS = (*PebbleFS)(nil)

// NewPebbleFS creates a new pebble/vfs.FS instance.
func NewPebbleFS(fs IFS) pvfs.FS {
	return &PebbleFS{fs}
}

// GetDiskUsage ...
func (p *PebbleFS) GetDiskUsage(path string) (pvfs.DiskUsage, error) {
	du, err := p.fs.GetDiskUsage(path)
	// lni/vfs returns an unexported error for disk usage on its memory filesystem.
	if err != nil && err.Error() == "vfs: not supported" {
		return pvfs.DiskUsage{}, pvfs.ErrUnsupported
	}
	return pvfs.DiskUsage{
		AvailBytes: du.AvailBytes,
		TotalBytes: du.TotalBytes,
		UsedBytes:  du.UsedBytes,
	}, err
}

// Create ...
func (p *PebbleFS) Create(name string, _ pvfs.DiskWriteCategory) (pvfs.File, error) {
	// Pebble requires a new inode when replacing a file, which may have hard links.
	if err := p.fs.Remove(name); err != nil && !IsNotExist(err) {
		return nil, err
	}
	return wrapPebbleFile(p.fs.Create(name))
}

// Link ...
func (p *PebbleFS) Link(oldname, newname string) error {
	return p.fs.Link(oldname, newname)
}

// Open ...
func (p *PebbleFS) Open(name string, opts ...pvfs.OpenOption) (pvfs.File, error) {
	f, err := wrapPebbleFile(p.fs.Open(name))
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt.Apply(f)
	}
	return f, nil
}

// OpenReadWrite is used by Pebble's optional shared object cache.
func (p *PebbleFS) OpenReadWrite(name string, _ pvfs.DiskWriteCategory, _ ...pvfs.OpenOption) (pvfs.File, error) {
	// The underlying filesystem only provides append mode for existing files.
	return nil, &os.PathError{Op: "open read-write", Path: name, Err: pvfs.ErrUnsupported}
}

// OpenDir ...
func (p *PebbleFS) OpenDir(name string) (pvfs.File, error) {
	f, err := p.fs.OpenDir(name)
	if name == "." && IsNotExist(err) {
		// Pebble syncs the database's parent. lni/vfs MemFS opens its root as "".
		f, err = p.fs.OpenDir("")
	}
	if err != nil {
		return nil, err
	}
	return pebbleFile{File: f, isDir: true}, nil
}

// Remove ...
func (p *PebbleFS) Remove(name string) error {
	return p.fs.Remove(name)
}

// RemoveAll ...
func (p *PebbleFS) RemoveAll(name string) error {
	return p.fs.RemoveAll(name)
}

// Rename ...
func (p *PebbleFS) Rename(oldname, newname string) error {
	return p.fs.Rename(oldname, newname)
}

// ReuseForWrite ...
func (p *PebbleFS) ReuseForWrite(oldname, newname string, _ pvfs.DiskWriteCategory) (pvfs.File, error) {
	return wrapPebbleFile(p.fs.ReuseForWrite(oldname, newname))
}

// MkdirAll ...
func (p *PebbleFS) MkdirAll(dir string, perm os.FileMode) error {
	return p.fs.MkdirAll(dir, perm)
}

// Lock ...
func (p *PebbleFS) Lock(name string) (io.Closer, error) {
	return p.fs.Lock(name)
}

// List ...
func (p *PebbleFS) List(dir string) ([]string, error) {
	return p.fs.List(dir)
}

// Stat ...
func (p *PebbleFS) Stat(name string) (pvfs.FileInfo, error) {
	return wrapPebbleFileInfo(p.fs.Stat(name))
}

// PathBase ...
func (p *PebbleFS) PathBase(path string) string {
	return p.fs.PathBase(path)
}

// PathJoin ...
func (p *PebbleFS) PathJoin(elem ...string) string {
	return p.fs.PathJoin(elem...)
}

// PathDir ...
func (p *PebbleFS) PathDir(path string) string {
	return p.fs.PathDir(path)
}

// Unwrap ends Pebble's wrapper chain at the underlying lni filesystem.
func (p *PebbleFS) Unwrap() pvfs.FS { return nil }

type pebbleFile struct {
	File
	isDir bool
}

func wrapPebbleFile(f File, err error) (pvfs.File, error) {
	if err != nil {
		return nil, err
	}
	return pebbleFile{File: f}, nil
}

func (f pebbleFile) Stat() (pvfs.FileInfo, error) {
	return wrapPebbleFileInfo(f.File.Stat())
}

func (f pebbleFile) SyncTo(_ int64) (bool, error) {
	if f.isDir {
		return false, nil
	}
	err := f.Sync()
	return err == nil, err
}

func (f pebbleFile) SyncData() error              { return f.Sync() }
func (f pebbleFile) Preallocate(_, _ int64) error { return nil }
func (f pebbleFile) Prefetch(_, _ int64) error    { return nil }
func (f pebbleFile) Fd() uintptr                  { return pvfs.InvalidFd }

type pebbleFileInfo struct{ os.FileInfo }

func wrapPebbleFileInfo(info os.FileInfo, err error) (pvfs.FileInfo, error) {
	if err != nil {
		return nil, err
	}
	return pebbleFileInfo{info}, nil
}

func (pebbleFileInfo) DeviceID() pvfs.DeviceID { return pvfs.DeviceID{} }

// IsNotExist returns a boolean value indicating whether the specified error is
// to indicate that a file or directory does not exist.
func IsNotExist(err error) bool {
	return oserror.IsNotExist(err)
}

// IsExist returns a boolean value indicating whether the specified error is to
// indicate that a file or directory already exists.
func IsExist(err error) bool {
	return oserror.IsExist(err)
}

// TempDir returns the directory use for storing temporary files.
func TempDir() string {
	return os.TempDir()
}

// Clean is a wrapper for filepath.Clean.
func Clean(dir string) string {
	return filepath.Clean(dir)
}

// ReportLeakedFD reports leaked file fds.
func ReportLeakedFD(fs IFS, t *testing.T) {
	gvfs.ReportLeakedFD(fs, t)
}
