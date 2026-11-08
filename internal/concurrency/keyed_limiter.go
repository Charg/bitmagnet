package concurrency

import (
	"context"
	"fmt"
	"hash/maphash"
	"net/netip"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

const keyedLimiterShards = 64

type KeyedLimiter[K comparable] interface {
	Allow(key K) bool
	Wait(ctx context.Context, key K) error
}

type keyedLimiterShard[K comparable] struct {
	mu  sync.Mutex
	lru *lru.LRU[K, *rate.Limiter]
}

type keyedLimiter[K comparable] struct {
	shards [keyedLimiterShards]keyedLimiterShard[K]
	seed   maphash.Seed
	rl     rate.Limit
	burst  int
}

// NewKeyedLimiter creates a rate limiter per key. The limiters are held in a sharded, expirable LRU cache
// with a total capacity of (approximately) size entries.
func NewKeyedLimiter[K comparable](rl rate.Limit, burst int, size int, ttl time.Duration) KeyedLimiter[K] {
	i := &keyedLimiter[K]{
		seed:  maphash.MakeSeed(),
		rl:    rl,
		burst: burst,
	}

	shardSize := max((size+keyedLimiterShards-1)/keyedLimiterShards, 1)
	for n := range i.shards {
		i.shards[n].lru = lru.NewLRU[K, *rate.Limiter](shardSize, nil, ttl)
	}

	return i
}

func (i *keyedLimiter[K]) Allow(key K) bool {
	return i.getLimiter(key).Allow()
}

func (i *keyedLimiter[K]) Wait(ctx context.Context, key K) error {
	return i.getLimiter(key).Wait(ctx)
}

func (i *keyedLimiter[K]) shardFor(key K) *keyedLimiterShard[K] {
	var h uint64

	switch k := any(key).(type) {
	case netip.Addr:
		b := k.As16()
		h = maphash.Bytes(i.seed, b[:])
	case string:
		h = maphash.String(i.seed, k)
	default:
		h = maphash.String(i.seed, fmt.Sprint(key))
	}

	return &i.shards[h%keyedLimiterShards]
}

func (i *keyedLimiter[K]) getLimiter(key K) *rate.Limiter {
	s := i.shardFor(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.lru.Get(key)
	if !ok {
		l = rate.NewLimiter(i.rl, i.burst)
		s.lru.Add(key, l)
	}

	return l
}
