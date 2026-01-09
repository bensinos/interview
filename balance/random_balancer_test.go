package balance

import (
	"sync"
	"testing"
)

func TestRandomBalancer_Basic(t *testing.T) {
	servers := []string{"server1", "server2", "server3"}
	balancer := NewRandomBalancer(servers)

	// 测试返回的服务器都在列表中
	for i := 0; i < 100; i++ {
		got := balancer.Next()
		found := false
		for _, s := range servers {
			if got == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Next() = %v, which is not in server list", got)
		}
	}
}

func TestRandomBalancer_SingleServer(t *testing.T) {
	servers := []string{"only-server"}
	balancer := NewRandomBalancer(servers)

	for i := 0; i < 10; i++ {
		got := balancer.Next()
		if got != "only-server" {
			t.Errorf("Next() = %v, want only-server", got)
		}
	}
}

func TestRandomBalancer_EmptyServers(t *testing.T) {
	servers := []string{}
	balancer := NewRandomBalancer(servers)

	got := balancer.Next()
	if got != "" {
		t.Errorf("Next() with empty servers = %v, want empty string", got)
	}
}

func TestRandomBalancer_Concurrency(t *testing.T) {
	servers := []string{"s1", "s2", "s3", "s4"}
	balancer := NewRandomBalancer(servers)

	const goroutines = 100
	const callsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

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
	serverSet := make(map[string]bool)
	for _, s := range servers {
		serverSet[s] = true
	}

	counts := make(map[string]int)
	for server := range results {
		counts[server]++
		if !serverSet[server] {
			t.Errorf("Got unexpected server: %v", server)
		}
	}

	// 验证所有服务器都被访问过
	for _, s := range servers {
		if counts[s] == 0 {
			t.Errorf("Server %s was never selected", s)
		}
	}

	if len(counts) != len(servers) {
		t.Errorf("Expected %d unique servers, got %d", len(servers), len(counts))
	}

	t.Logf("Distribution: %v", counts)
}

func TestRandomBalancer_Distribution(t *testing.T) {
	servers := []string{"a", "b", "c"}
	balancer := NewRandomBalancer(servers)

	counts := make(map[string]int)
	iterations := 3000

	for i := 0; i < iterations; i++ {
		server := balancer.Next()
		counts[server]++
	}

	// 对于随机分布，每个服务器应该被调用约 1000 次
	// 允许一定的误差范围（比如 +/- 15%）
	expectedPerServer := iterations / len(servers)
	tolerance := expectedPerServer * 15 / 100

	for _, server := range servers {
		count := counts[server]
		if count < expectedPerServer-tolerance || count > expectedPerServer+tolerance {
			t.Errorf("Server %s: expected ~%d calls (tolerance %d), got %d",
				server, expectedPerServer, tolerance, count)
		}
	}

	t.Logf("Distribution: %v", counts)
}

func TestRandomBalancer_ModifyOriginalSlice(t *testing.T) {
	servers := []string{"server1", "server2", "server3"}
	balancer := NewRandomBalancer(servers)

	// 修改原始切片
	servers[0] = "modified"
	servers = append(servers, "server4")

	// RandomBalancer 应该仍然使用原始的切片引用
	// 这是预期的行为，虽然可能导致问题
	for i := 0; i < 10; i++ {
		got := balancer.Next()
		if got == "server4" {
			// 这说明 balancer 使用的是原始切片的引用
			t.Logf("Balancer uses original slice reference (got 'server4')")
		}
	}
}

func BenchmarkRandomBalancer_Next(b *testing.B) {
	servers := make([]string, 10)
	for i := 0; i < 10; i++ {
		servers[i] = "server"
	}
	balancer := NewRandomBalancer(servers)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			balancer.Next()
		}
	})
}

func BenchmarkRandomBalancer_Next_Serial(b *testing.B) {
	servers := []string{"s1", "s2", "s3", "s4", "s5"}
	balancer := NewRandomBalancer(servers)

	for i := 0; i < b.N; i++ {
		balancer.Next()
	}
}
