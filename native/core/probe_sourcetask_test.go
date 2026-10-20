package core

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProbeSourceTaskUpdate(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	start := time.Now()
	err = engine.updateSource(ctx, "hongguo", "update")
	fmt.Printf("SOURCE-TASK err=%v elapsed=%.1fs items=%d\n", err, time.Since(start).Seconds(), len(engine.nativeCached("hongguo").Items))
}
