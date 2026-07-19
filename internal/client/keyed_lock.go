// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"sync"
)

// keyedLock serializes work for the same key while allowing different keys to
// proceed concurrently. Entries are reference counted so organizations that
// are no longer being mutated do not remain in memory for the client's
// lifetime.
type keyedLock struct {
	mu      sync.Mutex
	entries map[string]*keyedLockEntry
}

type keyedLockEntry struct {
	semaphore  chan struct{}
	references int
}

// lock waits until key is available or ctx is canceled. The returned unlock
// function must be called exactly once after a successful lock.
func (l *keyedLock) lock(ctx context.Context, key string) (func(), error) {
	entry := l.retain(key)

	// Prefer a cancellation that happened before lock was called over acquiring
	// an immediately available semaphore.
	select {
	case <-ctx.Done():
		l.releaseReference(key, entry)
		return nil, ctx.Err()
	default:
	}

	select {
	case entry.semaphore <- struct{}{}:
		return func() {
			<-entry.semaphore
			l.releaseReference(key, entry)
		}, nil
	case <-ctx.Done():
		l.releaseReference(key, entry)
		return nil, ctx.Err()
	}
}

func (l *keyedLock) retain(key string) *keyedLockEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.entries == nil {
		l.entries = make(map[string]*keyedLockEntry)
	}

	entry, ok := l.entries[key]
	if !ok {
		entry = &keyedLockEntry{semaphore: make(chan struct{}, 1)}
		l.entries[key] = entry
	}
	entry.references++

	return entry
}

func (l *keyedLock) releaseReference(key string, entry *keyedLockEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.references--
	if entry.references == 0 {
		delete(l.entries, key)
	}
}
