package store

import (
	"sync"
	"time"
)

type MemDedupe struct {
	mu      sync.Mutex
	seen    map[string]time.Time
	ttl     time.Duration
	maxKeys int
}

func NewMemDedupe(ttl time.Duration, maxKeys int) *MemDedupe {
	return &MemDedupe{
		seen:    make(map[string]time.Time),
		ttl:     ttl,
		maxKeys: maxKeys,
	}
}

func (d *MemDedupe) MarkSeen(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	d.gcLocked(now)

	if t, ok := d.seen[id]; ok && now.Sub(t) <= d.ttl {
		return false
	}
	d.seen[id] = now
	return true
}

func (d *MemDedupe) gcLocked(now time.Time) {
	for k, t := range d.seen {
		if now.Sub(t) > d.ttl {
			delete(d.seen, k)
		}
	}

	//remove all if exceeding maxKeys
	if d.maxKeys > 0 && len(d.seen) > d.maxKeys {
		d.seen = make(map[string]time.Time)
	}
}
