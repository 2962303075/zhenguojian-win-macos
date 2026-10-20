package core

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProbeCatalogBroken(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	page, err := engine.nativeCatalog(ctx, nativeInput{Source: "hongguo", Page: 1, Force: true})
	if err != nil {
		fmt.Printf("CATALOG ERR: %v\n", err)
		return
	}
	fmt.Printf("CATALOG OK items=%d warning=%q\n", len(page.Items), page.Warning)
}
