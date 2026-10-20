package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProbeUpdateCheck(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, version := range []string{"0.2.71+79", "0.2.70+78"} {
		raw, _ := json.Marshal(updateQuery{Version: version, Edition: "zhenguojian"})
		info, err := engine.updateCheck(ctx, string(raw))
		if err != nil {
			fmt.Printf("[v=%s] ERR %v\n", version, err)
			continue
		}
		fmt.Printf("[v=%s] available=%v latest=%s asset=%.60s err=%s\n", version, info.Available, info.Version, info.AssetURL, info.Error)
	}
}
