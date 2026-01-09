package balance

import (
	"sync"
	"testing"
)

func TestRoundRobinBalancer_Basic(t *testing.T) {
	servers := []string{"server1", "server2", "server3"}
	balancer := NewRoundRobinBalancer(servers)

	tests := []struct {
		index int
		want  string
	}{
		{0, "server1"},
		{1, "server2"},
		{2, "server3"},
		{3, "server1"}, // 循环回第一个
		{4, "server2"},
		{5, "server3"},
	}

	for _, tt := range tests {
		got := balancer.Next()
		if got != tt.want {
			t.Errorf("Next() = %v, want %v", got, tt.want)
		}
	}
}

func TestRoundRobinBalancer_SingleServer(t *testing.T) {
	servers := []string{"only-server"}
	balancer := NewRoundRobinBalancer(servers)

	for i := 0; i < 10; i++ {
		got := balancer.Next()
		if got != "only-server" {
			t.Errorf("Next() = %v, want only-server", got)
		}
	}
}

func TestRoundRobinBalancer_Concurrency(t *testing.T) {
	servers := []string{"s1", "s2", "s3", "s4"}
	balancer := NewRoundRobinBalancer(servers)

	const goroutines = 100
	const callsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 用于收集结果
	results := make(chan string, goroutines*callsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				results <- balancer.Next()
			}
		}()
	}

	wg.Wait()
	close(results)

	// 验证所有返回的服务器都是有效的
	counts := make(map[string]int)
	for server := range results {
		counts[server]++
	}

	totalCalls := goroutines * callsPerGoroutine
	if len(counts) != len(servers) {
		t.Errorf("Expected %d unique servers, got %d", len(servers), len(counts))
	}

	// 验证每个服务器被调用的次数大致相等
	expectedPerServer := totalCalls / len(servers)
	for server, count := range counts {
		if count < expectedPerServer-10 || count > expectedPerServer+10 {
			t.Errorf("Server %s: expected ~%d calls, got %d", server, expectedPerServer, count)
		}
	}
}

func TestRoundRobinBalancer_EvenDistribution(t *testing.T) {
	servers := []string{"x", "y", "z"}
	balancer := NewRoundRobinBalancer(servers)

	counts := make(map[string]int)
	iterations := 3000 // 3个服务器，每个应该被调用1000次

	for i := 0; i < iterations; i++ {
		server := balancer.Next()
		counts[server]++
	}

	for _, server := range servers {
		if counts[server] != 1000 {
			t.Errorf("Server %s: expected 1000 calls, got %d", server, counts[server])
		}
	}
}

// 基准测试
func BenchmarkRoundRobinBalancer_Next(b *testing.B) {
	servers := make([]string, 10)
	for i := 0; i < 10; i++ {
		servers[i] = "server"
	}
	balancer := NewRoundRobinBalancer(servers)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			balancer.Next()
		}
	})
}
