import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:shared_preferences/shared_preferences.dart';

/// 甜橙短剧用户统计埋点：启动上报到自建统计服务。
/// 全部异步、失败静默，绝不影响 App 正常功能。
abstract final class StatsTracker {
  static const _endpoint = 'http://36.212.212.170:8090/track';
  static const _key = 'stats_device_id';
  static String? _deviceId;

  /// 读取或生成本机设备唯一 ID（首次生成后持久化）。
  static Future<String> deviceId() async {
    final cached = _deviceId;
    if (cached != null) return cached;
    final prefs = await SharedPreferences.getInstance();
    final existing = prefs.getString(_key);
    if (existing != null && existing.length >= 8) {
      return _deviceId = existing;
    }
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));
    final id = _hex(bytes);
    await prefs.setString(_key, id);
    return _deviceId = id;
  }

  static String _hex(List<int> bytes) =>
      bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();

  /// 上报启动（活跃）事件。异步、静默失败，超时 3 秒。
  static void trackLaunch({String platform = '', String version = ''}) {
    final id = _deviceId;
    if (id == null || id.length < 8) return;
    try {
      final client = HttpClient()
        ..connectionTimeout = const Duration(seconds: 3);
      final request = client.postUrl(Uri.parse(_endpoint));
      request.headers.contentType = ContentType.json;
      request.add(
        utf8.encode(
          jsonEncode({
            'device_id': id,
            'platform': platform,
            'version': version,
            'event': 'launch',
          }),
        ),
      );
      request.close().then((response) {
        response
            .drain<void>()
            .then((_) {
              try {
                client.close();
              } catch (_) {}
            })
            .catchError((Object _) {
              try {
                client.close();
              } catch (_) {}
            });
      }).catchError((Object _) {
        try {
          client.close();
        } catch (_) {}
      });
    } catch (_) {
      // 静默：统计不上报不影响使用
    }
  }
}
