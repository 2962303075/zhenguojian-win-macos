import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

import 'app_build.dart';
import 'app_layout.dart';
import 'core_bridge.dart';

class UpdateInfo {
  const UpdateInfo({
    required this.available,
    required this.current,
    required this.version,
    required this.notes,
    required this.pageUrl,
    required this.assetUrl,
    this.error,
  });

  final bool available;
  final String current;
  final String version;
  final String notes;
  final String pageUrl;
  final String assetUrl;
  final String? error;

  static UpdateInfo fromJson(Map<String, dynamic> json) => UpdateInfo(
    available: json['available'] == true,
    current: '${json['current'] ?? appVersion}',
    version: '${json['version'] ?? ''}',
    notes: '${json['notes'] ?? ''}',
    pageUrl: '${json['pageUrl'] ?? 'https://github.com/vjc9wrh9r9-boop/zhenguojian-win-macos/releases/latest'}',
    assetUrl: '${json['assetUrl'] ?? ''}',
    error: json['error'] == null ? null : '${json['error']}',
  );
}

String get _edition => allSourcesEnabled ? 'zhenguojian' : 'hongguojian';

const _checkInterval = Duration(hours: 6);
const _skippedKey = 'updateSkippedVersion';
const _lastCheckKey = 'updateLastCheck';

Future<UpdateInfo?> checkForUpdate(AppRepository repository) async {
  try {
    final raw = await repository.checkForUpdate(appVersion, _edition);
    final info = UpdateInfo.fromJson(raw);
    return info;
  } on AppFailure {
    rethrow;
  } catch (_) {
    return null;
  }
}

/// 启动后静默检查：有新版且未被跳过时弹窗。
Future<void> maybeAutoCheckUpdate(
  BuildContext context,
  AppRepository repository,
) async {
  if (!mountedContext(context)) return;
  try {
    final prefs = await SharedPreferences.getInstance();
    final last = DateTime.tryParse(prefs.getString(_lastCheckKey) ?? '');
    if (last != null && DateTime.now().difference(last) < _checkInterval) {
      return;
    }
    await prefs.setString(_lastCheckKey, DateTime.now().toIso8601String());
    final info = await checkForUpdate(repository);
    if (info == null || !info.available) return;
    if (prefs.getString(_skippedKey) == info.version) return;
    if (!mountedContext(context)) return;
    await showUpdateDialog(context, repository, info);
  } catch (_) {}
}

bool mountedContext(BuildContext context) {
  try {
    return context.mounted;
  } catch (_) {
    return false;
  }
}

Future<void> showUpdateDialog(
  BuildContext context,
  AppRepository repository,
  UpdateInfo info, {
  bool manual = false,
}) async {
  final action = await showDialog<String>(
    context: context,
    builder: (context) {
      final width = MediaQuery.sizeOf(context).width;
      return AlertDialog(
        title: Text(info.available ? '发现新版本 ${info.version}' : '版本检查'),
        content: SizedBox(
          width: width < 520 ? width * 0.8 : 460,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  '当前版本 ${info.current}',
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
                const SizedBox(height: 12),
                if (info.error != null)
                  Text(info.error!, style: const TextStyle(fontSize: 13))
                else if (info.notes.trim().isNotEmpty)
                  Text(info.notes.trim(), style: const TextStyle(fontSize: 13, height: 1.5))
                else if (!info.available)
                  const Text('当前已是最新版本。', style: TextStyle(fontSize: 13)),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, 'later'),
            child: const Text('稍后'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, 'github'),
            child: const Text('去 GitHub 下载'),
          ),
          if (info.available && Platform.isWindows && info.assetUrl.isNotEmpty)
            FilledButton(
              onPressed: () => Navigator.pop(context, 'update'),
              child: const Text('立即更新'),
            ),
        ],
      );
    },
  );
  if (!mountedContext(context)) return;
  switch (action) {
    case 'github':
      await launchUrl(Uri.parse(info.pageUrl), mode: LaunchMode.externalApplication);
    case 'update':
      await _runDownload(context, repository, info);
    case 'later':
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_skippedKey, info.version);
  }
}

Future<void> _runDownload(
  BuildContext context,
  AppRepository repository,
  UpdateInfo info,
) async {
  final navigator = Navigator.of(context, rootNavigator: true);
  final progress = ValueNotifier<double?>(null);
  final statusText = ValueNotifier<String>('正在下载 ${info.version}…');
  unawaited(showDialog<void>(
    context: context,
    barrierDismissible: false,
    builder: (context) => PopScope(
      canPop: false,
      child: AlertDialog(
        title: const Text('正在更新'),
        content: ValueListenableBuilder<double?>(
          valueListenable: progress,
          builder: (context, value, _) => Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              LinearProgressIndicator(value: value),
              const SizedBox(height: 12),
              ValueListenableBuilder<String>(
                valueListenable: statusText,
                builder: (context, text, _) =>
                    Text(text, style: const TextStyle(fontSize: 12)),
              ),
            ],
          ),
        ),
      ),
    ),
  ));

  var closed = false;
  void closeDialog() {
    if (closed) return;
    closed = true;
    navigator.pop();
  }

  try {
    await repository.downloadUpdate(appVersion, _edition);
    while (true) {
      await Future<void>.delayed(const Duration(milliseconds: 500));
      final status = await repository.updateStatus();
      final state = '${status['state']}';
      if (state == 'downloading') {
        final received = (status['received'] as num?)?.toDouble() ?? 0;
        final total = (status['total'] as num?)?.toDouble() ?? 0;
        progress.value = total > 0 ? (received / total).clamp(0.0, 1.0) : null;
        statusText.value = total > 0
            ? '正在下载 ${_size(received)} / ${_size(total)}'
            : '正在下载…';
        continue;
      }
      if (state == 'extracting') {
        progress.value = null;
        statusText.value = '正在解压安装包…';
        continue;
      }
      if (state == 'ready') {
        progress.value = 1;
        statusText.value = '即将关闭并完成更新…';
        break;
      }
      throw AppFailure('${status['message'] ?? '更新失败，请稍后重试'}');
    }
    final applied = await repository.applyUpdate();
    final batch = '${applied['batch'] ?? ''}';
    if (batch.isEmpty) throw AppFailure('更新脚本创建失败');
    await Process.start('cmd', ['/c', batch], mode: ProcessStartMode.detached);
    closeDialog();
    await Future<void>.delayed(const Duration(milliseconds: 300));
    exit(0);
  } catch (error) {
    closeDialog();
    if (mountedContext(context)) {
      await showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('更新未完成'),
          content: Text('$error\n\n可前往 GitHub 手动下载最新版本。'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('知道了'),
            ),
            TextButton(
              onPressed: () async {
                Navigator.pop(context);
                await launchUrl(Uri.parse(info.pageUrl), mode: LaunchMode.externalApplication);
              },
              child: const Text('去 GitHub 下载'),
            ),
          ],
        ),
      );
    }
  }
}

String _size(double bytes) {
  if (bytes >= 1024 * 1024) return '${(bytes / 1048576).toStringAsFixed(1)} MB';
  if (bytes >= 1024) return '${(bytes / 1024).toStringAsFixed(0)} KB';
  return '${bytes.toInt()} B';
}
