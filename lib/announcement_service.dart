import 'dart:convert';
import 'dart:io';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

/// 甜橙短剧公告：从自建统计服务拉取，展示给用户（如引导加 QQ 群）。
/// 全部异步、失败静默，绝不影响 App 正常功能。
class Announcement {
  const Announcement({
    required this.enabled,
    this.title = '',
    this.body = '',
    this.buttonText,
    this.link,
    this.date,
  });

  final bool enabled;
  final String title;
  final String body;
  final String? buttonText;
  final String? link;
  final String? date;

  factory Announcement.fromJson(Map<String, dynamic> json) => Announcement(
        enabled: json['enabled'] == true,
        title: (json['title'] as String?) ?? '',
        body: (json['body'] as String?) ?? '',
        buttonText: json['button_text'] as String?,
        link: json['link'] as String?,
        date: json['date'] as String?,
      );
}

abstract final class AnnouncementService {
  static const _endpoint = 'http://36.212.212.170:8090/announcement';
  static const _seenKey = 'announcement_last_seen_date';

  /// 拉取公告；无公告 / 请求失败返回 null。
  static Future<Announcement?> fetch() async {
    try {
      final client = HttpClient()
        ..connectionTimeout = const Duration(seconds: 4);
      final request = await client.getUrl(Uri.parse(_endpoint));
      request.headers.set('Accept', 'application/json');
      final response = await request.close();
      if (response.statusCode != 200) {
        client.close();
        return null;
      }
      final text = await response.transform(utf8.decoder).join();
      client.close();
      final json = jsonDecode(text) as Map<String, dynamic>;
      final announcement = Announcement.fromJson(json);
      return announcement.enabled ? announcement : null;
    } catch (_) {
      return null;
    }
  }

  /// 今日是否已经看过公告。
  static Future<bool> seenToday() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getString(_seenKey) == _today();
    } catch (_) {
      return false;
    }
  }

  /// 标记今日已看过公告（每日最多弹一次）。
  static Future<void> markSeenToday() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_seenKey, _today());
    } catch (_) {}
  }

  static String _today() =>
      DateTime.now().toIso8601String().substring(0, 10);

  /// 打开公告链接（QQ 群 / 外部页面）。
  static Future<void> open(Announcement announcement) async {
    final url = announcement.link;
    if (url == null || url.isEmpty) return;
    try {
      await launchUrl(
        Uri.parse(url),
        mode: LaunchMode.externalApplication,
      );
    } catch (_) {}
  }
}
