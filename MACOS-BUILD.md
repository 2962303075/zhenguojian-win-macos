# 真果鉴 macOS 版 · 构建指南

> **为什么要在 Mac 上跑一次**：Flutter 的 macOS 应用**只能在 macOS + Xcode 环境编译**（Windows 电脑无法交叉编译出 .app，这是工具链硬限制）。
> 本包已经把全部移植适配做好（新增 macOS 平台工程、Go 核心 macOS 双架构构建脚本、平台分支代码），你在 **任意一台 Mac** 上只需 **一条命令** 即可产出可直接使用的 `真果鉴.app`。

---

## 一、准备环境（一次性，约 30 分钟）

需要：**一台 Mac**（Apple Silicon / Intel 均可，macOS 12 或更新）。

### 1. 安装 Xcode
1. App Store 搜索 **Xcode** 安装（约 7GB，等待时间较长）
2. 装完后打开一次 Xcode 完成首次组件安装，然后在终端执行：
   ```bash
   sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
   xcodebuild -runFirstLaunch
   ```

### 2. 安装 Go（1.24 或更新）
```bash
brew install go
```
（没装 Homebrew 的话到 https://go.dev/dl/ 下载 pkg 双击安装）

### 3. 安装 Flutter 3.47.4（必须与源码要求匹配）
```bash
git clone --depth 1 --branch 3.47.4 https://github.com/flutter/flutter.git ~/flutter
echo 'export PATH="$HOME/flutter/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
flutter --version        # 首次运行会自动下载 Dart SDK，需要几分钟
```
> 国内网络建议追加镜像（构建脚本的 `--cn-mirrors` 参数也会自动启用）：
> ```bash
> echo 'export PUB_HOSTED_URL=https://pub.flutter-io.cn' >> ~/.zshrc
> echo 'export FLUTTER_STORAGE_BASE_URL=https://storage.flutter-io.cn' >> ~/.zshrc
> ```

### 4. 安装 CocoaPods
```bash
brew install cocoapods
```
（如 brew 安装异常，改用 `sudo gem install cocoapods`）

## 二、构建（一条命令）

```bash
unzip guoapp-macos-port.zip
cd guoapp
python3 scripts/build_macos.py --all-sources --cn-mirrors
```

脚本全自动完成五步：
1. 编译 Go 站源核心（arm64 + Intel **双架构**静态库 → `DuanjuCore.xcframework`）
2. `flutter pub get`（国内镜像）
3. `pod install` —— **此步会从 GitHub 下载 FFmpegKit 的 macOS 预编译框架（约 100MB）**，网络不畅时挂个代理，或多次重试
4. `flutter build macos --release`
5. 校验 FFI 符号（DuanjuRequest / DuanjuFree）并打出 zip

**产物**：`dist/macos/zhenguojian-0.2.66+74-macos-universal.zip`

## 三、运行

```bash
unzip dist/macos/zhenguojian-*.zip
xattr -cr zhenguojian.app     # 去掉网络下载的隔离标记（未签名应用的正常一步）
open zhenguojian.app
```
或 Finder 中**右键 → 打开**（首次；之后正常双击）。

**站源解锁方式和 Windows 版完全相同**：左侧「最近观看」连点 **6 次** → 弹窗点「**不使用密码**」→ 全部 23 个站源显示。

## 四、本包相对上游（c344579765/guoapp）改了什么

**新增（macOS 移植，全部是本包新增文件）**

| 文件 | 作用 |
|---|---|
| `macos/` | Flutter macOS 平台工程：已配好网络/文件 entitlements、中文显示名（真果鉴）、窗口标题、局域网互联声明（_zgj-link._tcp） |
| `macos/DuanjuCore/` | Go 核心 pod（静态链接，做法与官方 iOS 工程完全一致） |
| `macos/Podfile` | 声明 DuanjuCore pod + Flutter 插件 |
| `scripts/build_macos.py` | macOS 一键构建脚本（本文件） |

**新增（字幕功能，0.2.65+73 起）**

| 文件 | 作用 |
|---|---|
| 播放器控制条「字幕」按钮 | 打开字幕面板：内嵌字幕轨切换（自动/关闭/指定轨）、加载外挂字幕文件（.srt/.ass/.ssa/.vtt）、字幕延迟 ±0.5s、字幕字号调整 |
| 代码位置 | `lib/player_menu.dart`（字幕面板 UI）、`lib/player_screen.dart`（字幕加载与状态）、`lib/player_controls.dart`（按钮入口） |

**修改（只为 macOS 补平台分支，不影响 Windows / Android / iOS 行为）**

| 文件 | 改动 |
|---|---|
| `lib/core_bridge.dart` | FFI 加载增加 macOS 分支（`DynamicLibrary.process()`，与 iOS 同路径） |
| `lib/main.dart` | window_manager 初始化、设备类型识别（computer）增加 macOS；版本日志更新 |
| `lib/player_screen.dart` | 全屏切换增加 macOS 支持 |
| `lib/app_layout.dart` | 版本号 0.2.65 |

**未改动**：全部业务逻辑、Go 站源实现、Windows / Android / iOS 构建链。可用 `git diff` 查看全部改动。

## 五、已验证 / 未验证（请先读）

**已在原构建机上验证通过：**
- ✅ 代码静态分析零错误（`dart analyze` 全绿）
- ✅ macOS 插件依赖图完整：media_kit（含 macOS 媒体库）、ffmpeg-kit、nsd、window_manager、file_picker 等 12 个插件均有 macOS 实现
- ✅ Go 核心代码含 `!windows` 构建标记（darwin 可编译），核心 ABI 与 iOS 同源
- ✅ 同源代码的 Windows 版已实际构建并运行验证（站源、播放链路正常）

**本机无法验证（因为本机是 Windows、没有 Mac）：**
- ⚠️ macOS 的实际编译与运行 —— 需要你在 Mac 上首次构建时确认
- ⚠️ 如果首次构建遇到报错，**把终端完整输出发回来**，我来针对性修改（这是移植项目的正常流程）

## 六、常见问题

| 现象 | 处理 |
|---|---|
| `pod install` 卡在下载 FFmpegKit | 网络问题：挂代理重试；或先单独构建核心：`python3 scripts/build_macos.py --core-only`，成功后再跑完整构建 |
| 报错缺少 `DuanjuCore.xcframework` | 先执行 `python3 scripts/build_macos.py --core-only` 生成核心，再重试完整构建 |
| 构建报错含 `dependency_overrides` | 本包脚本会自动临时处理；如仍报错请把输出发回 |
| 运行后站源加载失败 | 检查网络；本包 entitlements 已含网络权限，若你手动改过工程请确认 `network.client` 仍在 |
| 下载的剧集写不入选定目录 | 系统设置 → 隐私与安全性 → 文件与文件夹，给真果鉴打开"下载文件夹"权限 |
| 白屏或闪退 | 终端运行 `/path/to/zhenguojian.app/Contents/MacOS/zhenguojian` 查看日志，把输出发回 |
| 想关掉沙盒（遇到下载/文件权限问题时） | 把 `macos/Runner/DebugProfile.entitlements` 与 `Release.entitlements` 中 `<key>com.apple.security.app-sandbox</key>` 的 `<true/>` 改为 `<false/>`，重新构建 |

---

*构建包版本：0.2.66+74（zhenguojian / 全站源 / 含字幕增强）· 移植整理日期 2026-10-14*
