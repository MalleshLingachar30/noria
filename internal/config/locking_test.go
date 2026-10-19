package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStaleLockFileIsNotAStaleOwner(t *testing.T) {
	h := t.TempDir()
	if e := os.WriteFile(filepath.Join(h, "operation.lock"), []byte("pid=999999"), 0600); e != nil {
		t.Fatal(e)
	}
	release, e := Lock(h, "operation")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()
	if next, e := LockContext(ctx, h, "operation"); e == nil {
		next()
		t.Fatal("held OS lock was bypassed")
	}
	release()
	next, e := Lock(h, "operation")
	if e != nil {
		t.Fatal(e)
	}
	next()
}
