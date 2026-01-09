package balance

import (
	"math/rand"
	"sync"
	"time"
)

type RandomBalancer struct {
	servers []string
	rng     *rand.Rand
	mu      sync.Mutex
}

func NewRandomBalancer(servers []string) Balancer {
	return &RandomBalancer{
		servers: servers,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RandomBalancer) Next() string {
	if len(r.servers) == 0 {
		return ""
	}
	r.mu.Lock()
	idx := r.rng.Intn(len(r.servers))
	r.mu.Unlock()
	return r.servers[idx]
}
