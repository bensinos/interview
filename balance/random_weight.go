package balance

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	Addr   string
	Weight int
}

type RandomWeightBalancer struct {
	servers atomic.Value
	rng     *rand.Rand
	lock    sync.RWMutex
}

func NewRandomWeightBalancer(servers []*Server) Balancer {
	b := &RandomWeightBalancer{
		servers: atomic.Value{},
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	b.servers.Store(servers)
	return b
}

func (r *RandomWeightBalancer) Next() string {
	// Read server list once to avoid race conditions
	servers := r.servers.Load().([]*Server)
	if len(servers) == 0 {
		return ""
	}

	// Calculate total weight
	totalWeight := 0
	for _, s := range servers {
		totalWeight += s.Weight
	}
	if totalWeight <= 0 {
		return ""
	}

	// Generate random index with lock protection
	r.lock.Lock()
	idx := r.rng.Intn(totalWeight)
	r.lock.Unlock()

	// Find the server based on random index
	for _, s := range servers {
		idx -= s.Weight
		if idx < 0 {
			return s.Addr
		}
	}

	// This should never happen if weights are positive
	// Return first server as fallback
	return servers[0].Addr
}
