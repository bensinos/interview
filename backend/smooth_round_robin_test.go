package backend

import (
	"context"
	"sync"
	"testing"
)

// TestSmoothRRBasic 测试基本权重分布
func TestSmoothRRBasic(t *testing.T) {
	nodes := []*Node{
		{server: "a", weight: 5, current: 0},
		{server: "b", weight: 1, current: 0},
		{server: "c", weight: 1, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes).(*smoothRoundRobinBalancer)

	// 总权重 = 7，预期分布: a:5次, b:1次, c:1次
	expected := map[string]int{
		"a": 5,
		"b": 1,
		"c": 1,
	}

	counts := make(map[string]int)
	for i := 0; i < 7; i++ {
		node := balancer.Next(context.Background())
		counts[node.server]++
	}

	for server, expectedCount := range expected {
		if counts[server] != expectedCount {
			t.Errorf("server %s: expected %d, got %d", server, expectedCount, counts[server])
		}
	}
}

// TestSmoothRRSingleServer 测试单节点
func TestSmoothRRSingleServer(t *testing.T) {
	nodes := []*Node{
		{server: "only", weight: 10, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes)

	for i := 0; i < 100; i++ {
		node := balancer.Next(context.Background())
		if node.server != "only" {
			t.Errorf("expected 'only', got %s", node.server)
		}
	}
}

// TestSmoothRREmptyNodes 测试空节点panic
func TestSmoothRREmptyNodes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty nodes")
		}
	}()

	_ = NewSmoothRRBalancer([]*Node{})
}

// TestSmoothRRZeroWeight 测试零权重
func TestSmoothRRZeroWeight(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero weight")
		}
	}()

	nodes := []*Node{
		{server: "a", weight: 0, current: 0},
	}
	_ = NewSmoothRRBalancer(nodes)
}

// TestSmoothRRNegativeWeight 测试负权重
func TestSmoothRRNegativeWeight(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative weight")
		}
	}()

	nodes := []*Node{
		{server: "a", weight: -1, current: 0},
	}
	_ = NewSmoothRRBalancer(nodes)
}

// TestSmoothRRWeightExceedsMax 测试权重超限
func TestSmoothRRWeightExceedsMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for weight exceeding max")
		}
	}()

	nodes := []*Node{
		{server: "a", weight: maxWeight + 1, current: 0},
	}
	_ = NewSmoothRRBalancer(nodes)
}

// TestSmoothRRTotalWeightExceedsMax 测试总权重超限
func TestSmoothRRTotalWeightExceedsMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for total weight exceeding max")
		}
	}()

	nodes := []*Node{
		{server: "a", weight: maxTotalWeight - 100, current: 0},
		{server: "b", weight: 200, current: 0},
	}
	_ = NewSmoothRRBalancer(nodes)
}

// TestSmoothRRDistribution 测试大样本分布准确性
func TestSmoothRRDistribution(t *testing.T) {
	nodes := []*Node{
		{server: "a", weight: 3, current: 0},
		{server: "b", weight: 2, current: 0},
		{server: "c", weight: 1, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes).(*smoothRoundRobinBalancer)

	// 预期比例: a=50%, b=33.3%, c=16.7%
	iterations := 600
	counts := make(map[string]int)

	for i := 0; i < iterations; i++ {
		node := balancer.Next(context.Background())
		counts[node.server]++
	}

	expected := map[string]int{
		"a": 300, // 3/6 * 600
		"b": 200, // 2/6 * 600
		"c": 100, // 1/6 * 600
	}

	for server, exp := range expected {
		if counts[server] != exp {
			t.Errorf("server %s: expected %d, got %d", server, exp, counts[server])
		}
	}
}

// TestSmoothRRConcurrency 测试并发安全
func TestSmoothRRConcurrency(t *testing.T) {
	nodes := []*Node{
		{server: "a", weight: 5, current: 0},
		{server: "b", weight: 3, current: 0},
		{server: "c", weight: 2, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes).(*smoothRoundRobinBalancer)

	const goroutines = 100
	const callsPerGoroutine = 100

	var wg sync.WaitGroup
	counts := make(map[string]int)
	countsMu := sync.Mutex{}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				node := balancer.Next(context.Background())
				countsMu.Lock()
				counts[node.server]++
				countsMu.Unlock()
			}
		}()
	}

	wg.Wait()

	totalCalls := goroutines * callsPerGoroutine
	expectedRatio := map[string]float64{
		"a": 0.5, // 5/10
		"b": 0.3, // 3/10
		"c": 0.2, // 2/10
	}

	t.Logf("Total calls: %d", totalCalls)
	for server, count := range counts {
		ratio := float64(count) / float64(totalCalls)
		expected := expectedRatio[server]
		diff := ratio - expected
		t.Logf("%s: %d calls (%.2f%%), expected %.2f%%, diff %.4f",
			server, count, ratio*100, expected*100, diff)
	}
}

// TestSmoothRRSmoothness 测试平滑性（连续性）
func TestSmoothRRSmoothness(t *testing.T) {
	nodes := []*Node{
		{server: "a", weight: 4, current: 0},
		{server: "b", weight: 1, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes).(*smoothRoundRobinBalancer)

	// 权重 4:1，预期序列应该是均匀分布，不应该出现连续4次a
	var lastNode string
	consecutiveCount := 0
	maxConsecutive := 0

	for i := 0; i < 20; i++ {
		node := balancer.Next(context.Background())
		t.Logf("iteration %d: selected %s (current=%d)",
			i, node.server, node.current)

		if node.server == lastNode {
			consecutiveCount++
			if consecutiveCount > maxConsecutive {
				maxConsecutive = consecutiveCount
			}
		} else {
			consecutiveCount = 1
			lastNode = node.server
		}
	}

	t.Logf("Max consecutive: %d", maxConsecutive)
	// 权重4:1时，连续4次是合理的（不超过4次即为正常平滑分布）
	if maxConsecutive > 4 {
		t.Errorf("distribution not smooth enough: max consecutive = %d", maxConsecutive)
	}
}

// BenchmarkSmoothRR 性能基准测试
func BenchmarkSmoothRR(b *testing.B) {
	nodes := []*Node{
		{server: "a", weight: 5, current: 0},
		{server: "b", weight: 3, current: 0},
		{server: "c", weight: 2, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes).(*smoothRoundRobinBalancer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.Next(context.Background())
	}
}

// BenchmarkSmoothRRParallel 并发性能基准测试
func BenchmarkSmoothRRParallel(b *testing.B) {
	nodes := []*Node{
		{server: "a", weight: 5, current: 0},
		{server: "b", weight: 3, current: 0},
		{server: "c", weight: 2, current: 0},
	}

	balancer := NewSmoothRRBalancer(nodes).(*smoothRoundRobinBalancer)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			balancer.Next(context.Background())
		}
	})
}
