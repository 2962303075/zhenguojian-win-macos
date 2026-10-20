package core

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProbeUpdateLoop(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	for round := 0; round < 4; round++ {
		start := time.Now()
		page, err := engine.nativeCatalog(ctx, nativeInput{Source: "hongguo", Page: 1, Force: true})
		fail := 0
		if err != nil {
			fmt.Printf("[round %d] ERR %v\n", round, err)
		} else {
			if page.Warning != "" {
				fail = 1
			}
			fmt.Printf("[round %d] items=%d %.1fs warning=%q\n", round, len(page.Items), time.Since(start).Seconds(), page.Warning)
		}
		_ = fail
		time.Sleep(2 * time.Second)
	}
}
