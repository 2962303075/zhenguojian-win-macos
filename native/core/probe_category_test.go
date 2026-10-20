package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestProbeCategoryRaw(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client := d.hongguoClient()
	base, deviceID, installID := client.baseURL, client.state.DeviceID, client.state.InstallID

	payload, _ := json.Marshal(map[string]any{
		"req_scene": "101", "offset": 0, "limit": 18,
		"req_type": "only_content", "need_selector_panel": false, "client_req_type": 3,
		"session_id": "", "filter_ids": "",
		"select_items": map[string]any{
			"genre": []string{"1"}, "sort": []string{"online_time"}, "gender": []string{},
			"category_dim_theme": []string{}, "category_dim_role": []string{}, "category_dim_epoch": []string{},
			"online_time": []string{}, "creation_status": []string{},
		},
	})

	query := url.Values{
		"iid": {installID}, "device_id": {deviceID}, "ac": {"wifi"}, "channel": {"update_64"},
		"aid": {"8662"}, "app_name": {"novelread"}, "version_code": {"71332"},
		"version_name": {"7.1.3.32"}, "device_platform": {"android"}, "os": {"android"},
		"ssmix": {"a"}, "device_type": {"25053RT47C"}, "device_brand": {"Redmi"},
		"language": {"zh"}, "os_api": {"36"}, "os_version": {"16"}, "manifest_version_code": {"71332"},
		"resolution": {"1280*2772"}, "dpi": {"520"}, "update_version_code": {"71332"},
		"host_abi": {"arm64-v8a"}, "dragon_device_type": {"phone"}, "pv_player": {"71332"},
		"compliance_status": {"0"}, "need_personal_recommend": {"1"}, "player_so_load": {"1"},
		"is_android_pad_screen": {"0"},
	}

	for _, mode := range []string{"player-sign", "old-sign"} {
		now := time.Now()
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		q.Set("_rticket", strconv.FormatInt(now.UnixMilli(), 10))
		address := strings.TrimRight(base, "/") + "/reading/distribution/category/landpage/v/?" + q.Encode()
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, address, strings.NewReader(string(payload)))
		req.Header.Set("User-Agent", hongguoAppUserAgent)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-XS-From-Web", "0")
		req.Header.Set("Sdk-Version", "2")
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		if mode == "player-sign" {
			signHongguoPlayerRequest(req, payload, now, installID)
		} else {
			signHongguoRequest(req, payload, now)
		}
		resp, err := d.client.Do(req)
		if err != nil {
			fmt.Printf("[%s] NETERR %v\n", mode, err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		fmt.Printf("[%s] HTTP %d len=%d argus=%v ladon=%v body=%.400s\n", mode, resp.StatusCode, len(body),
			req.Header.Get("X-Argus") != "", req.Header.Get("X-Ladon") != "", strings.TrimSpace(string(body)))
	}
}

func TestProbeCategoryReal(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client := d.hongguoClient()
	base, deviceID, installID := client.baseURL, client.state.DeviceID, client.state.InstallID
	for _, genre := range hongguoAppGenres {
		for _, limit := range []int{18, 54} {
			payload, _ := json.Marshal(map[string]any{
				"req_scene": genre.scene, "offset": 0, "limit": limit,
				"req_type": "only_content", "need_selector_panel": false, "client_req_type": 3,
				"session_id": "", "filter_ids": "",
				"select_items": map[string]any{
					"genre": []string{genre.key}, "sort": []string{"online_time"}, "gender": []string{},
					"category_dim_theme": []string{}, "category_dim_role": []string{}, "category_dim_epoch": []string{},
					"online_time": []string{}, "creation_status": []string{},
				},
			})
			now := time.Now()
			q := url.Values{}
			q.Set("iid", installID)
			q.Set("device_id", deviceID)
			q.Set("aid", "8662")
			q.Set("app_name", "novelread")
			q.Set("version_code", "71332")
			q.Set("version_name", "7.1.3.32")
			q.Set("device_platform", "android")
			q.Set("os", "android")
			q.Set("channel", "update_64")
			q.Set("ac", "wifi")
			q.Set("_rticket", strconv.FormatInt(now.UnixMilli(), 10))
			address := strings.TrimRight(base, "/") + "/reading/distribution/category/landpage/v/?" + q.Encode()
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, address, strings.NewReader(string(payload)))
			req.Header.Set("User-Agent", hongguoAppUserAgent)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("X-XS-From-Web", "0")
			req.Header.Set("Sdk-Version", "2")
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			signHongguoPlayerRequest(req, payload, now, installID)
			resp, err := d.client.Do(req)
			if err != nil {
				fmt.Printf("[%s scene=%s limit=%d] NETERR %v\n", genre.name, genre.scene, limit, err)
				continue
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			fmt.Printf("[%s scene=%s limit=%d] HTTP %d len=%d body=%.200s\n", genre.name, genre.scene, limit, resp.StatusCode, len(body), strings.TrimSpace(string(body)))
		}
	}
}

func TestProbeCategoryViaApp(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	for _, genre := range hongguoAppGenres {
		payload := map[string]any{
			"req_scene": genre.scene, "offset": 0, "limit": 18,
			"req_type": "only_content", "need_selector_panel": false, "client_req_type": 3,
			"session_id": "", "filter_ids": "",
			"select_items": map[string]any{
				"genre": []string{genre.key}, "sort": []string{"online_time"}, "gender": []string{},
				"category_dim_theme": []string{}, "category_dim_role": []string{}, "category_dim_epoch": []string{},
				"online_time": []string{}, "creation_status": []string{},
			},
		}
		result, err := d.hongguoAppRequest(ctx, http.MethodPost, "/reading/distribution/category/landpage/v/", nil, payload)
		if err != nil {
			fmt.Printf("[app %s] ERR %v\n", genre.name, err)
			continue
		}
		data := nestedMap(result, "data")
		rows := anyList(data["video_data"])
		fmt.Printf("[app %s] OK rows=%d hasMore=%v\n", genre.name, len(rows), data["has_more"])
	}
}

func TestProbeCategoryOffsets(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	client := d.hongguoClient()
	for _, offset := range []int{0, 54, 108, 216, 432, 540, 648, 1000} {
		payload := map[string]any{
			"req_scene": "comic_series", "offset": offset, "limit": 54,
			"req_type": "only_content", "need_selector_panel": false, "client_req_type": 3,
			"session_id": "", "filter_ids": "",
			"select_items": map[string]any{
				"genre": []string{"comic_series"}, "sort": []string{"online_time"}, "gender": []string{},
				"category_dim_theme": []string{}, "category_dim_role": []string{}, "category_dim_epoch": []string{},
				"online_time": []string{}, "creation_status": []string{},
			},
		}
		result, err := d.hongguoAppRequest(ctx, http.MethodPost, "/reading/distribution/category/landpage/v/", nil, payload)
		if err != nil {
			fmt.Printf("[offset=%d] ERR %v\n", offset, err)
			continue
		}
		data := nestedMap(result, "data")
		fmt.Printf("[offset=%d] rows=%d hasMore=%v next=%v\n", offset, len(anyList(data["video_data"])), data["has_more"], data["next_offset"])
		_ = client
	}
}

func TestProbeCategoryRate(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	payload := func(offset int) map[string]any {
		return map[string]any{
			"req_scene": "comic_series", "offset": offset, "limit": 54,
			"req_type": "only_content", "need_selector_panel": false, "client_req_type": 3,
			"session_id": "", "filter_ids": "",
			"select_items": map[string]any{
				"genre": []string{"comic_series"}, "sort": []string{"online_time"}, "gender": []string{},
				"category_dim_theme": []string{}, "category_dim_role": []string{}, "category_dim_epoch": []string{},
				"online_time": []string{}, "creation_status": []string{},
			},
		}
	}
	for _, spacing := range []time.Duration{0, 400 * time.Millisecond, 800 * time.Millisecond} {
		ok, fail := 0, 0
		for i := 0; i < 10; i++ {
			if spacing > 0 {
				time.Sleep(spacing)
			}
			if _, err := d.hongguoAppRequest(ctx, http.MethodPost, "/reading/distribution/category/landpage/v/", nil, payload(0)); err == nil {
				ok++
			} else {
				fail++
			}
		}
		fmt.Printf("[spacing=%v] ok=%d fail=%d\n", spacing, ok, fail)
	}
}

func TestProbeGenreReliability(t *testing.T) {
	if os.Getenv("HG_PROBE") == "" {
		t.Skip("probe disabled")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	for _, genre := range hongguoAppGenres {
		ok, fail := 0, 0
		for i := 0; i < 6; i++ {
			payload := map[string]any{
				"req_scene": genre.scene, "offset": 0, "limit": 54,
				"req_type": "only_content", "need_selector_panel": false, "client_req_type": 3,
				"session_id": "", "filter_ids": "",
				"select_items": map[string]any{
					"genre": []string{genre.key}, "sort": []string{"online_time"}, "gender": []string{},
					"category_dim_theme": []string{}, "category_dim_role": []string{}, "category_dim_epoch": []string{},
					"online_time": []string{}, "creation_status": []string{},
				},
			}
			if _, err := d.hongguoAppRequest(ctx, http.MethodPost, "/reading/distribution/category/landpage/v/", nil, payload); err == nil {
				ok++
			} else {
				fail++
			}
		}
		fmt.Printf("[%s scene=%s] ok=%d fail=%d\n", genre.name, genre.scene, ok, fail)
	}
}
