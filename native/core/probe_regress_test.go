package core

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProbeRegression(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	_, chapters, err := d.GetHuangguoChapters(ctx, "hongguo", "7503745441910017049")
	fmt.Printf("DETAIL err=%v chapters=%d\n", err, len(chapters))

	entry, err := d.searchHongguoDramasProgress(ctx, "霸总", nil)
	fmt.Printf("SEARCH err=%v items=%d\n", err, len(entry.Dramas))

	media, err := d.resolveHongguoMultiMedia(ctx, "7503780574541122622")
	if err == nil {
		fmt.Printf("PLAYBACK OK key=%v\n", media.CENCKey != nil)
	} else {
		fmt.Printf("PLAYBACK ERR %v\n", err)
	}
	page, err := d.hongguoDanmaku(ctx, "7503745441910017049", "7503780574541122622", 0, 60000)
	if err == nil {
		fmt.Printf("DANMAKU OK items=%d\n", len(page.Items))
	} else {
		fmt.Printf("DANMAKU ERR %v\n", err)
	}
}
