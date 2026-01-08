package backend

import (
	"fmt"
	"sync/atomic"
)

type Balancer interface {
	Next() string
}

// RoundRobinBalancer
// 简单、高效
// 不能动态调整
type RoundRobinBalancer struct {
	servers []string
	index   uint64
}

func NewRoundRobinBalancer(servers []string) Balancer {
	if len(servers) == 0 {
		panic(fmt.Errorf("new round robin balancer failed: servers is empty"))
	}

	s := make([]string, len(servers))
	copy(s, servers)
	return &RoundRobinBalancer{
		servers: s,
	}
}

func (r *RoundRobinBalancer) Next() string {
	// 1. 原子递增索引值（保证并发安全）
	// 注意：atomic.AddUint64 返回的是增加后的新值
	newVal := atomic.AddUint64(&r.index, 1)

	// 2. 对服务器列表长度取模，实现循环轮询
	// 减 1 是因为我们想要从 0 开始计数，或者直接取模
	idx := (newVal - 1) % uint64(len(r.servers))

	return r.servers[idx]
}
