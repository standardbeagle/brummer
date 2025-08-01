package mcp

import (
	"encoding/json"
	"sync"
	"testing"
)

// Benchmarks comparing mutex-based vs lock-free implementations

func BenchmarkMessageQueueMutex_Send(b *testing.B) {
	mq := NewMessageQueue()
	defer mq.Stop()

	payload := json.RawMessage(`{"test": "data"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mq.Send("bench-channel", "test", payload, 3600)
	}
}

func BenchmarkMessageQueueLockFree_Send(b *testing.B) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	payload := json.RawMessage(`{"test": "data"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mq.Send("bench-channel", "test", payload, 3600)
	}
}

func BenchmarkMessageQueueMutex_Receive(b *testing.B) {
	mq := NewMessageQueue()
	defer mq.Stop()

	// Pre-populate with messages
	payload := json.RawMessage(`{"test": "data"}`)
	for i := 0; i < 1000; i++ {
		_, _ = mq.Send("bench-channel", "test", payload, 3600)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mq.Receive("bench-channel", 10, false, 0)
	}
}

func BenchmarkMessageQueueLockFree_Receive(b *testing.B) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	// Pre-populate with messages
	payload := json.RawMessage(`{"test": "data"}`)
	for i := 0; i < 1000; i++ {
		_, _ = mq.Send("bench-channel", "test", payload, 3600)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mq.Receive("bench-channel", 10, false, 0)
	}
}

func BenchmarkMessageQueueMutex_ConcurrentSend(b *testing.B) {
	mq := NewMessageQueue()
	defer mq.Stop()

	payload := json.RawMessage(`{"test": "data"}`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mq.Send("bench-channel", "test", payload, 3600)
		}
	})
}

func BenchmarkMessageQueueLockFree_ConcurrentSend(b *testing.B) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	payload := json.RawMessage(`{"test": "data"}`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mq.Send("bench-channel", "test", payload, 3600)
		}
	})
}

func BenchmarkMessageQueueMutex_Subscribe(b *testing.B) {
	mq := NewMessageQueue()
	defer mq.Stop()

	subs := make([]*Subscription, 0, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sub, _ := mq.Subscribe("bench-channel")
		subs = append(subs, sub)
	}

	// Cleanup
	b.StopTimer()
	for _, sub := range subs {
		_ = mq.Unsubscribe(sub.ID)
	}
}

func BenchmarkMessageQueueLockFree_Subscribe(b *testing.B) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	subs := make([]*Subscription, 0, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sub, _ := mq.Subscribe("bench-channel")
		subs = append(subs, sub)
	}

	// Cleanup
	b.StopTimer()
	for _, sub := range subs {
		_ = mq.Unsubscribe(sub.ID)
	}
}

func BenchmarkMessageQueueMutex_HighContention(b *testing.B) {
	mq := NewMessageQueue()
	defer mq.Stop()

	payload := json.RawMessage(`{"test": "data"}`)
	numChannels := 10

	// Pre-create channels
	for i := 0; i < numChannels; i++ {
		channel := string(rune('A' + i))
		_, _ = mq.Send(channel, "test", payload, 3600)
	}

	b.ResetTimer()

	var wg sync.WaitGroup

	// Multiple goroutines performing mixed operations
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ops := b.N / 4
			channel := string(rune('A' + (id % numChannels)))

			for j := 0; j < ops; j++ {
				switch j % 3 {
				case 0:
					_, _ = mq.Send(channel, "test", payload, 3600)
				case 1:
					_, _ = mq.Receive(channel, 5, false, 0)
				case 2:
					_ = mq.Stats()
				}
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkMessageQueueLockFree_HighContention(b *testing.B) {
	mq := NewMessageQueueLockFree()
	defer mq.Stop()

	payload := json.RawMessage(`{"test": "data"}`)
	numChannels := 10

	// Pre-create channels
	for i := 0; i < numChannels; i++ {
		channel := string(rune('A' + i))
		_, _ = mq.Send(channel, "test", payload, 3600)
	}

	b.ResetTimer()

	var wg sync.WaitGroup

	// Multiple goroutines performing mixed operations
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ops := b.N / 4
			channel := string(rune('A' + (id % numChannels)))

			for j := 0; j < ops; j++ {
				switch j % 3 {
				case 0:
					_, _ = mq.Send(channel, "test", payload, 3600)
				case 1:
					_, _ = mq.Receive(channel, 5, false, 0)
				case 2:
					_ = mq.Stats()
				}
			}
		}(i)
	}

	wg.Wait()
}

