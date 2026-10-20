package core

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const updateRepo = "vjc9wrh9r9-boop/zhenguojian-win-macos"

var (
	updateVersionPattern = regexp.MustCompile(`(\d+\.\d+\.\d+)\+(\d+)`)
	updateReleasesPage   = "https://github.com/" + updateRepo + "/releases/latest"

	updateMu     sync.Mutex
	updateStatus = struct {
		state    string
		received int64
		total    int64
		message  string
		staging  string
		asset    string
	}{state: "idle"}
)

type updateQuery struct {
	Version string `json:"version"`
	Edition string `json:"edition"`
}

type updateResult struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	PageURL   string `json:"pageUrl"`
	AssetURL  string `json:"assetUrl"`
	Error     string `json:"error,omitempty"`
}

func parseUpdateVersion(value string) (string, int, bool) {
	match := updateVersionPattern.FindStringSubmatch(value)
	if match == nil {
		return "", 0, false
	}
	build, err := strconv.Atoi(match[2])
	if err != nil {
		return "", 0, false
	}
	return match[1], build, true
}

func (engine *nativeEngine) updateCheck(ctx context.Context, raw string) (updateResult, error) {
	var query updateQuery
	_ = json.Unmarshal([]byte(raw), &query)
	result := updateResult{Current: query.Version, PageURL: updateReleasesPage}
	currentMain, currentBuild, ok := parseUpdateVersion(query.Version)
	if !ok {
		result.Error = "当前版本号无法解析"
		return result, nil
	}

	tag, notes, assets, err := engine.fetchLatestRelease(ctx)
	if err != nil {
		result.Error = "无法获取最新版本信息：" + err.Error()
		return result, nil
	}
	latestVersion := ""
	for name := range assets {
		if main, build, ok := parseUpdateVersion(name); ok && strings.HasPrefix(name, query.Edition) {
			latestVersion = fmt.Sprintf("%s+%d", main, build)
			break
		}
	}
	if latestVersion == "" {
		if main, build, ok := parseUpdateVersion(tag); ok {
			latestVersion = fmt.Sprintf("%s+%d", main, build)
		}
	}
	if latestVersion == "" {
		result.Error = "未能确定最新版本号"
		return result, nil
	}
	latestMain, latestBuild, _ := parseUpdateVersion(latestVersion)
	result.Version = latestVersion
	result.Notes = notes
	result.Available = latestBuild > currentBuild || (latestBuild == currentBuild && latestMain > currentMain)
	for name, address := range assets {
		if !strings.HasPrefix(name, query.Edition) {
			continue
		}
		switch runtime.GOOS {
		case "windows":
			if strings.Contains(name, "-windows-x64.zip") {
				result.AssetURL = address
			}
		case "darwin":
			if strings.Contains(name, "-macos-universal.zip") {
				result.AssetURL = address
			}
		}
		if result.AssetURL != "" {
			break
		}
	}
	return result, nil
}

func (engine *nativeEngine) fetchLatestRelease(ctx context.Context) (string, string, map[string]string, error) {
	assets := map[string]string{}
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	api := "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, api, nil)
	if err == nil {
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("User-Agent", "guojian-updater")
		if response, err := engine.updateClient().Do(request); err == nil {
			if response.StatusCode == http.StatusOK {
				var payload struct {
					TagName string `json:"tag_name"`
					Body    string `json:"body"`
					Assets  []struct {
						Name string `json:"name"`
						URL  string `json:"browser_download_url"`
					} `json:"assets"`
				}
				if decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); decodeErr == nil && payload.TagName != "" {
					response.Body.Close()
					for _, asset := range payload.Assets {
						assets[asset.Name] = asset.URL
					}
					return payload.TagName, payload.Body, assets, nil
				}
			}
			response.Body.Close()
		}
	}
	// GitHub API 不可达时退回 releases/latest 重定向解析
	for _, address := range []string{updateReleasesPage, "https://ghfast.top/" + updateReleasesPage} {
		client := engine.updateClient()
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, address, nil)
		if err != nil {
			continue
		}
		request.Header.Set("User-Agent", "guojian-updater")
		response, err := client.Do(request)
		if err != nil {
			continue
		}
		location := response.Header.Get("Location")
		response.Body.Close()
		if location == "" {
			continue
		}
		tag := location[strings.LastIndex(location, "/")+1:]
		if tag != "" {
			return tag, "", assets, nil
		}
	}
	return "", "", assets, errors.New("网络不可达")
}

func (engine *nativeEngine) updateClient() *http.Client {
	client := *engine.downloader.client
	client.Timeout = 0
	return &client
}

func (engine *nativeEngine) updateDownload(raw string) (map[string]any, error) {
	var query updateQuery
	_ = json.Unmarshal([]byte(raw), &query)
	if runtime.GOOS != "windows" {
		return nil, errors.New("当前平台请前往 GitHub 下载更新")
	}
	updateMu.Lock()
	if updateStatus.state == "downloading" || updateStatus.state == "extracting" {
		updateMu.Unlock()
		return map[string]any{"state": updateStatus.state}, nil
	}
	updateStatus = struct {
		state    string
		received int64
		total    int64
		message  string
		staging  string
		asset    string
	}{state: "downloading"}
	updateMu.Unlock()

	go engine.runUpdateDownload(query)
	return map[string]any{"state": "downloading"}, nil
}

func (engine *nativeEngine) runUpdateDownload(query updateQuery) {
	setState := func(state, message string, received, total int64, staging, asset string) {
		updateMu.Lock()
		updateStatus.state = state
		updateStatus.message = message
		updateStatus.received = received
		updateStatus.total = total
		updateStatus.staging = staging
		updateStatus.asset = asset
		updateMu.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	info, err := engine.updateCheck(ctx, mustJSON(query))
	if err != nil || info.AssetURL == "" {
		setState("error", "未找到适用于当前平台的安装包", 0, 0, "", "")
		return
	}
	cacheDir := filepath.Join(engine.directory, "update-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		setState("error", "无法创建更新缓存目录", 0, 0, "", "")
		return
	}
	zipPath := filepath.Join(cacheDir, "package.zip")
	_ = os.Remove(zipPath)
	addresses := []string{info.AssetURL, "https://ghfast.top/" + info.AssetURL, "https://gh-proxy.com/" + info.AssetURL}
	var received, total int64
	downloaded := false
	for _, address := range addresses {
		if err := engine.downloadToFile(ctx, address, zipPath, func(done, size int64) {
			setState("downloading", "", done, size, "", info.AssetURL)
		}); err == nil {
			downloaded = true
			received, total = 0, 0
			break
		}
	}
	if !downloaded {
		setState("error", "下载失败，请稍后重试或前往 GitHub 下载", received, total, "", info.AssetURL)
		return
	}
	setState("extracting", "正在解压", received, total, "", info.AssetURL)
	staging := filepath.Join(cacheDir, "staging")
	_ = os.RemoveAll(staging)
	if err := unzipTo(zipPath, staging); err != nil {
		setState("error", "安装包解压失败："+err.Error(), 0, 0, "", info.AssetURL)
		return
	}
	_ = os.Remove(zipPath)
	setState("ready", info.Version, 0, 0, staging, info.AssetURL)
}

func (engine *nativeEngine) downloadToFile(ctx context.Context, address, target string, progress func(int64, int64)) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "guojian-updater")
	response, err := engine.updateClient().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	buffer := make([]byte, 256<<10)
	var done int64
	total := response.ContentLength
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			if _, err := file.Write(buffer[:count]); err != nil {
				return err
			}
			done += int64(count)
			progress(done, total)
		}
		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}

func unzipTo(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, entry := range reader.File {
		path := filepath.Join(target, entry.Name)
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
			return errors.New("安装包内容异常")
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		file, err := os.Create(path)
		if err != nil {
			source.Close()
			return err
		}
		if _, err := io.Copy(file, source); err != nil {
			file.Close()
			source.Close()
			return err
		}
		file.Close()
		source.Close()
	}
	return nil
}

func (engine *nativeEngine) updateStatusValue() map[string]any {
	updateMu.Lock()
	defer updateMu.Unlock()
	return map[string]any{
		"state":    updateStatus.state,
		"received": updateStatus.received,
		"total":    updateStatus.total,
		"message":  updateStatus.message,
		"version":  updateStatus.message,
	}
}

func (engine *nativeEngine) updateApply() (map[string]any, error) {
	if runtime.GOOS != "windows" {
		return nil, errors.New("当前平台请前往 GitHub 下载更新")
	}
	updateMu.Lock()
	staging := updateStatus.staging
	state := updateStatus.state
	updateMu.Unlock()
	if state != "ready" || staging == "" {
		return nil, errors.New("更新尚未下载完成")
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	installDir := filepath.Dir(executable)
	batchPath := filepath.Join(installDir, "update.bat")
	name := filepath.Base(executable)
	content := strings.Join([]string{
		"@echo off",
		"chcp 65001 >nul",
		"timeout /t 1 /nobreak >nul",
		":wait",
		"tasklist /FI \"IMAGENAME eq " + name + "\" 2>nul | find /I \"" + name + "\" >nul",
		"if not errorlevel 1 (",
		"  timeout /t 1 /nobreak >nul",
		"  goto wait",
		")",
		"robocopy \"" + staging + "\" \"" + installDir + "\" /E /R:2 /W:1 /NFL /NDL /NJH /NJS >nul",
		"start \"\" \"" + filepath.Join(installDir, name) + "\"",
		"del \"%~f0\"",
	}, "\r\n")
	if err := os.WriteFile(batchPath, []byte(content), 0o755); err != nil {
		return nil, err
	}
	return map[string]any{"batch": batchPath}, nil
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
