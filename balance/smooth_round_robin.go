package balance

import (
	"context"
	"fmt"
	"sync"
)

const (
	maxWeight      = 1_000_000  // 单节点最大权重
	maxTotalWeight = 10_000_000 // 总权重上限
)

type Node struct {
	server  string
	current int // 当前权重
	weight  int // 权重
}

type SmoothBalancer interface {
	Next(ctx context.Context) *Node
}

type smoothRoundRobinBalancer struct {
	nodes []*Node
	lock  sync.RWMutex
}

// NewSmoothRRBalancer
// 下面用挑水来解释，平滑加权轮训的核心：
// 1、每次选取力气最大的节点
// 2、每次迭代时，每个节点的力气=weigth+effectvieWeight
// 3、比较节点自己的力气，是否大于总的力气
// 4、如果大于总的力气，则返回。否则继迭代
// 5、最后，选中的节点，要减掉力气
func NewSmoothRRBalancer(nodes []*Node) SmoothBalancer {
	if len(nodes) == 0 {
		panic(fmt.Errorf("new smooth rr failed: nodes is empty"))
	}
	totalWeight := 0
	for _, node := range nodes {
		if node.weight <= 0 {
			panic(fmt.Errorf("node weight must be positive, got: %d", node.weight))
		}
		if node.weight > maxWeight {
			panic(fmt.Errorf("node weight %d exceeds max %d", node.weight, maxWeight))
		}
		totalWeight += node.weight
	}

	if totalWeight > maxTotalWeight {
		panic(fmt.Errorf("total weight %d exceeds max %d", totalWeight, maxTotalWeight))
	}
	return &smoothRoundRobinBalancer{
		nodes: nodes,
	}
}

func (r *smoothRoundRobinBalancer) Next(ctx context.Context) *Node {
	r.lock.Lock()
	defer r.lock.Unlock()

	var (
		totalWeight = 0
		bestNode    *Node
	)
	for _, node := range r.nodes {
		node.current += node.weight
		totalWeight += node.weight

		if bestNode == nil || node.current > bestNode.current {
			bestNode = node
		}
	}

	if bestNode != nil {
		bestNode.current -= totalWeight
	}
	return bestNode
}
