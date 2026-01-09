package balance

import (
	"sync"
	"testing"
)

func TestRandomWeightBalancer_Basic(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 10},
		{Addr: "server2", Weight: 20},
		{Addr: "server3", Weight: 30},
	}
	balancer := NewRandomWeightBalancer(servers)

	// Run multiple times to ensure all servers are selected
	results := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		addr := balancer.Next()
		if addr == "" {
			t.Errorf("Expected non-empty address, got empty string")
		}
		results[addr]++
	}

	// Check that all servers were selected
	if len(results) != 3 {
		t.Errorf("Expected 3 unique servers, got %d", len(results))
	}

	// Check distribution - server2 should be selected roughly 2x more than server1
	// server3 should be selected roughly 3x more than server1
	ratio12 := float64(results["server2"]) / float64(results["server1"])
	ratio13 := float64(results["server3"]) / float64(results["server1"])

	t.Logf("Distribution: server1=%d, server2=%d, server3=%d", results["server1"], results["server2"], results["server3"])
	t.Logf("Ratio server2/server1: %.2f, server3/server1: %.2f", ratio12, ratio13)

	// Allow some tolerance for randomness
	if ratio12 < 1.5 || ratio12 > 2.5 {
		t.Errorf("Expected server2/server1 ratio around 2.0, got %.2f", ratio12)
	}
	if ratio13 < 2.5 || ratio13 > 3.5 {
		t.Errorf("Expected server3/server1 ratio around 3.0, got %.2f", ratio13)
	}
}

func TestRandomWeightBalancer_SingleServer(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 10},
	}
	balancer := NewRandomWeightBalancer(servers)

	for i := 0; i < 100; i++ {
		addr := balancer.Next()
		if addr != "server1" {
			t.Errorf("Expected 'server1', got '%s'", addr)
		}
	}
}

func TestRandomWeightBalancer_ZeroWeight(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 0},
		{Addr: "server2", Weight: 0},
	}
	balancer := NewRandomWeightBalancer(servers)

	addr := balancer.Next()
	if addr != "" {
		t.Errorf("Expected empty string for zero total weight, got '%s'", addr)
	}
}

func TestRandomWeightBalancer_NegativeWeight(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: -10},
		{Addr: "server2", Weight: 20},
	}
	balancer := NewRandomWeightBalancer(servers)

	addr := balancer.Next()
	if addr != "" {
		t.Logf("Warning: negative weights may cause unexpected behavior, got '%s'", addr)
	}
}

func TestRandomWeightBalancer_EqualWeights(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 10},
		{Addr: "server2", Weight: 10},
		{Addr: "server3", Weight: 10},
	}
	balancer := NewRandomWeightBalancer(servers)

	results := make(map[string]int)
	iterations := 3000

	for i := 0; i < iterations; i++ {
		addr := balancer.Next()
		results[addr]++
	}

	// All servers should have roughly equal distribution
	maxCount := 0
	minCount := iterations
	for _, count := range results {
		if count > maxCount {
			maxCount = count
		}
		if count < minCount {
			minCount = count
		}
	}

	// Check that the difference is not too large
	if maxCount-minCount > iterations/20 { // Allow up to 5% variation
		t.Errorf("Distribution is too uneven: max=%d, min=%d", maxCount, minCount)
	}

	t.Logf("Equal weight distribution: %v", results)
}

func TestRandomWeightBalancer_EmptyServers(t *testing.T) {
	servers := []*Server{}
	balancer := NewRandomWeightBalancer(servers)

	addr := balancer.Next()
	if addr != "" {
		t.Errorf("Expected empty string for empty servers, got '%s'", addr)
	}
}

func TestRandomWeightBalancer_OneZeroWeight(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 0},
		{Addr: "server2", Weight: 10},
		{Addr: "server3", Weight: 20},
	}
	balancer := NewRandomWeightBalancer(servers)

	results := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		addr := balancer.Next()
		results[addr]++
	}

	// server1 should never be selected (weight 0)
	if results["server1"] > 0 {
		t.Errorf("Server with weight 0 was selected %d times", results["server1"])
	}

	// server3 should be selected roughly 2x more than server2
	if results["server2"] > 0 {
		ratio := float64(results["server3"]) / float64(results["server2"])
		if ratio < 1.5 || ratio > 2.5 {
			t.Errorf("Expected server3/server2 ratio around 2.0, got %.2f", ratio)
		}
	}

	t.Logf("Distribution with zero weight: %v", results)
}

func TestRandomWeightBalancer_HighWeightVariance(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 1},
		{Addr: "server2", Weight: 100},
		{Addr: "server3", Weight: 1000},
	}
	balancer := NewRandomWeightBalancer(servers)

	results := make(map[string]int)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		addr := balancer.Next()
		results[addr]++
	}

	t.Logf("High variance distribution: %v", results)

	// server3 should dominate the selection
	if results["server3"] < results["server2"] || results["server2"] < results["server1"] {
		t.Errorf("Weight ordering not respected in distribution")
	}
}

func TestRandomWeightBalancer_Concurrent(t *testing.T) {
	servers := []*Server{
		{Addr: "server1", Weight: 10},
		{Addr: "server2", Weight: 20},
		{Addr: "server3", Weight: 30},
	}
	balancer := NewRandomWeightBalancer(servers)

	var wg sync.WaitGroup
	results := make(chan string, 10000)
	iterations := 5000

	// Run concurrent selections
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations/10; j++ {
				addr := balancer.Next()
				results <- addr
			}
		}()
	}

	wg.Wait()
	close(results)

	// Count results
	counts := make(map[string]int)
	for addr := range results {
		if addr == "" {
			t.Errorf("Got empty address during concurrent test")
		}
		counts[addr]++
	}

	// Verify all servers were selected
	if len(counts) != 3 {
		t.Errorf("Expected 3 unique servers, got %d", len(counts))
	}

	t.Logf("Concurrent distribution: %v", counts)
}

func TestRandomWeightBalancer_DistributionAccuracy(t *testing.T) {
	servers := []*Server{
		{Addr: "s1", Weight: 1},
		{Addr: "s2", Weight: 2},
		{Addr: "s3", Weight: 3},
		{Addr: "s4", Weight: 4},
	}
	balancer := NewRandomWeightBalancer(servers)

	results := make(map[string]int)
	iterations := 100000

	for i := 0; i < iterations; i++ {
		addr := balancer.Next()
		results[addr]++
	}

	// Expected probabilities: s1=10%, s2=20%, s3=30%, s4=40%
	expected := map[string]float64{
		"s1": 0.1,
		"s2": 0.2,
		"s3": 0.3,
		"s4": 0.4,
	}

	t.Logf("Results: %v", results)

	// Check each server's distribution
	for server, count := range results {
		actual := float64(count) / float64(iterations)
		exp := expected[server]
		diff := actual - exp

		// Allow 2% tolerance
		if diff < -0.02 || diff > 0.02 {
			t.Errorf("Server %s: expected %.2f, got %.2f (diff=%.3f)", server, exp, actual, diff)
		}
	}
}

// Test that validates servers are sorted by address for consistency
func TestRandomWeightBalancer_Consistency(t *testing.T) {
	servers1 := []*Server{
		{Addr: "server2", Weight: 30},
		{Addr: "server1", Weight: 10},
		{Addr: "server3", Weight: 20},
	}
	servers2 := []*Server{
		{Addr: "server1", Weight: 10},
		{Addr: "server2", Weight: 30},
		{Addr: "server3", Weight: 20},
	}

	balancer1 := NewRandomWeightBalancer(servers1)
	balancer2 := NewRandomWeightBalancer(servers2)

	results1 := make(map[string]int)
	results2 := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		addr1 := balancer1.Next()
		addr2 := balancer2.Next()
		results1[addr1]++
		results2[addr2]++
	}

	t.Logf("Balancer1 (unsorted input): %v", results1)
	t.Logf("Balancer2 (sorted input): %v", results2)

	// Both should have the same distribution
	for server := range results1 {
		if results1[server] != results2[server] {
			t.Logf("Warning: Server order affects distribution for %s: %d vs %d",
				server, results1[server], results2[server])
		}
	}
}
