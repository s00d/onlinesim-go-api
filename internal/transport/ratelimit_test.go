package transport

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLimiterGlobalGap(t *testing.T) {
	l := NewLimiter(2) // 500ms
	ctx := context.Background()
	start := time.Now()
	if err := l.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := l.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 400*time.Millisecond {
		t.Fatalf("expected ~500ms gap, got %v", elapsed)
	}
}

func TestLimiterPathInterval(t *testing.T) {
	l := NewLimiter(10) // 100ms global
	l.SetPathInterval("setOperationOk", 200*time.Millisecond)
	ctx := context.Background()
	start := time.Now()
	_ = l.WaitPath(ctx, "setOperationOk")
	_ = l.WaitPath(ctx, "setOperationOk")
	elapsed := time.Since(start)
	if elapsed < 180*time.Millisecond {
		t.Fatalf("expected path gap ~200ms, got %v", elapsed)
	}
}

func TestLimiterDisabled(t *testing.T) {
	l := NewLimiter(0)
	l.SetPathInterval("setOperationOk", time.Second)
	start := time.Now()
	_ = l.WaitPath(context.Background(), "setOperationOk")
	_ = l.WaitPath(context.Background(), "setOperationOk")
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("disabled limiter should not sleep")
	}
}

func TestLimiterContextCancel(t *testing.T) {
	l := NewLimiter(1)
	_ = l.Wait(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := l.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
