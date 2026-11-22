package xui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// HotACLActivator updates only the persistent Bridge authorization field.
// Confirm must trigger SIGHUP on the VERIFIED Android-facing container and
// wait for its exact M6_ACL_RELOAD=PASS digest before reporting success.
// Running streams are not touched; no restart is performed.
//
// A single instance serializes enrollments to avoid losing concurrent devices.
// The existing HTTPS registrar should use this as its Activate callback.
type HotACLActivator struct {
	mu      sync.Mutex
	Path    string
	Service string
    Services []string // optional; default remains the existing single Service
	Confirm func(context.Context, string) error
}

func (a *HotACLActivator) Activate(ctx context.Context, fingerprint string) error {
	if a == nil || a.Confirm == nil || a.Service == "" || a.Path == "" {
		return errors.New("Bridge ACL activation not configured")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	fi, err := os.Lstat(a.Path)
	if err != nil || !fi.Mode().IsRegular() {
		return errors.New("Bridge configuration is not a regular file")
	}
	before, err := os.ReadFile(a.Path)
	if err != nil || len(before) > 4<<20 {
		return errors.New("Bridge configuration unavailable")
	}
    services := a.Services
    if len(services) == 0 { services = []string{a.Service} }
    after, err := AddDeviceToACLServices(before, fingerprint, services)
	if err != nil {
		return err
	}
	digest, err := aclDigest(after)
	if err != nil {
		return err
	}
	if err = atomicACLConfigWrite(a.Path, after, fi); err != nil {
		// A directory fsync failure may occur after rename; restore the old
		// file even if publication did not finish, and never signal success.
		if restoreErr := atomicACLConfigWrite(a.Path, before, fi); restoreErr != nil {
			return fmt.Errorf("ACL publication and restoration both failed: %w", restoreErr)
		}
		return fmt.Errorf("ACL publication failed: %w", err)
	}
	// The confirmed digest must come from the running process, not from the
	// file we just wrote. A successful docker kill --signal HUP is NOT an ACK.
	if err = a.Confirm(ctx, digest); err == nil {
		return nil
	}
	// Roll back disk changes if the new snapshot was not acknowledged.
	// A second notification restores in-memory state if the process did load
	// the first change just before the acknowledgment failure.
	rollbackErr := atomicACLConfigWrite(a.Path, before, fi)
	if rollbackErr != nil {
		return fmt.Errorf("ACL not confirmed; persistent rollback failed: %w", rollbackErr)
	}
	priorDigest, hashErr := aclDigest(before)
	if hashErr != nil {
		return errors.New("ACL not confirmed; prior ACL is invalid")
	}
	rollbackCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if secondErr := a.Confirm(rollbackCtx, priorDigest); secondErr != nil {
		return errors.New("ACL not confirmed; previous file restored but runtime rollback unverified")
	}
	return errors.New("new ACL not confirmed; previous ACL restored")
}

func aclDigest(data []byte) (string, error) {
	raw, err := decodeServerConfig(data)
	if err != nil {
		return "", err
	}
	var acl map[string][]string
	if err := json.Unmarshal(raw["authorized"], &acl); err != nil || acl == nil {
		return "", errors.New("missing ACL")
	}
	canonical, err := json.Marshal(acl)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func atomicACLConfigWrite(path string, content []byte, prior os.FileInfo) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".m6-acl-")
	if err != nil {
		return errors.New("cannot stage Bridge ACL")
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(prior.Mode().Perm()); err != nil {
		f.Close()
		return err
	}
	if stat, ok := prior.Sys().(*syscall.Stat_t); ok {
		if err := f.Chown(int(stat.Uid), int(stat.Gid)); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	fd, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer fd.Close()
	return fd.Sync()
}
