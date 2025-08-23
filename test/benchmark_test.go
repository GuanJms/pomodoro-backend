package test

import (
	"testing"

	"pomodoroService/internal/clock"
)

func BenchmarkConcurrentReads(b *testing.B) {
	cr := clock.NewClockRunner()

	// Start a session to have some state to read
	cr.Start()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate multiple clients polling the state
			cr.GetState()
			cr.GetTimeRemaining()
			cr.GetCurrentSession()
			cr.GetTotalSessions()
		}
	})
}

func BenchmarkMixedReadWrite(b *testing.B) {
	cr := clock.NewClockRunner()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				// Every 10th operation is a write (simulating admin operations)
				if cr.GetState() == clock.StateIdle {
					cr.Start()
				} else {
					cr.Stop()
				}
			} else {
				// 90% of operations are reads (simulating user polling)
				cr.GetState()
				cr.GetTimeRemaining()
				cr.GetCurrentSession()
				cr.GetTotalSessions()
			}
			i++
		}
	})
}

func BenchmarkHeavyPolling(b *testing.B) {
	cr := clock.NewClockRunner()

	// Start a session
	cr.Start()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate heavy polling from multiple clients
			cr.GetState()
			cr.GetTimeRemaining()
			cr.GetCurrentSession()
			cr.GetTotalSessions()
			cr.IsRunning()
		}
	})
}
