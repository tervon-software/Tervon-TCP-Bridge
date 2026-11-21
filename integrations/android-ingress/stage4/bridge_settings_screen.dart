import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../../vpn/xray_service.dart';

/// Automatic Bridge enrollment for the current HTTPS subscription.
/// No manual destination fields or credential entry.
class BridgeSettingsScreen extends StatefulWidget {
  const BridgeSettingsScreen({super.key});

  @override
  State<BridgeSettingsScreen> createState() => _BridgeSettingsScreenState();
}

class _BridgeSettingsScreenState extends State<BridgeSettingsScreen> {
  static const _channel = MethodChannel('com.tervonvpndpi/bridge_control');
  static const _background = Color(0xFF0B0E17);
  static const _card = Color(0xFF141825);
  static const _gold = Color(0xFFD4A853);
  bool _enabled = false;
  bool _busy = false;
  String _message = 'Проверка состояния Bridge…';

  @override
  void initState() { super.initState(); _load(); }

  Future<void> _load() async {
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>('getSettings');
      if (!mounted) return;
      setState(() {
        _enabled = data?['enabled'] == true;
        _message = data?['runtimeIntegrated'] == true
            ? 'Bridge: локальный маршрут готов, связь с сервером не подтверждена'
            : 'Bridge: маршрутизация пока не активна';
      });
    } catch (_) {
      if (mounted) setState(() => _message = 'Не удалось прочитать состояние Bridge');
    }
  }

  // Only fixed failure codes are shown. No subscription URL, key or certificate.
  String _bridgeError(String? code) => switch (code) {
    'BYPASS_DISABLED' => 'обход выключен',
    'SUBSCRIPTION_NOT_SELECTED' => 'не определена текущая HTTPS-подписка',
    'INVALID_SUBSCRIPTION_HOST' => 'некорректный адрес подписки',
    'UNSUPPORTED_TRANSPORT' => 'транспорт профиля не поддержан TCP-мостом',
    'XHTTP_H3_UNSUPPORTED' => 'этот режим XHTTP не работает поверх TCP',
    'BRIDGE_SERVICE_UNAVAILABLE' => 'сервер не разрешил TCP-маршрут выбранного подключения',
    'DISCOVERY_NO_BRIDGE' => 'сервер подписки не сообщил о Bridge',
    'DISCOVERY_NO_PARAMETERS' => 'сервер не прислал параметры Bridge',
    'DISCOVERY_ID_MISMATCH' => 'не совпадает сервер подписки',
    'DEVICE_CERTIFICATE_UNAVAILABLE' => 'сертификат устройства недоступен',
    'PORT_ALLOCATION_FAILED' => 'не удалось выделить локальный порт',
    'BRIDGE_ALREADY_RUNNING' => 'предыдущий процесс Bridge ещё работает',
    'CONFIG_WRITE_FAILED' => 'не удалось записать конфигурацию Bridge',
    'BRIDGE_START_FAILED' => 'ядро Bridge не запустилось',
    'BRIDGE_LISTENER_UNAVAILABLE' => 'ядро Bridge не открыло локальный порт',
    'ROUTE_REJECTED' => 'выбранный TCP-маршрут отклонён',
    'XRAY_ROUTE_FAILED' => 'не удалось переключить маршрут Xray',
    'PROFILE_CONFIG_INVALID' => 'не удалось прочитать настройки выбранного профиля',
    'BRIDGE_PROCESS_EXITED' => 'процесс Bridge завершился',
    'BRIDGE_NOT_ACTIVE' => 'Bridge не подтвердил готовность после перезапуска VPN',
    'VPN_RESTART_FAILED' => 'не удалось перезапустить VPN',
    'ENROLLMENT_FAILED' => 'не удалось зарегистрировать устройство',
    _ => 'неизвестная причина',
  };

  Future<void> _toggleBypass(bool enable) async {
    if (_busy) return;
    setState(() { _busy = true; _message = 'Проверка Bridge…'; });
    try {
      String? url;
      if (enable) {
        url = await XrayService.getActiveSubscriptionRaw();
        final parsed = Uri.tryParse(url ?? '');
        if (parsed == null || parsed.scheme != 'https' || parsed.host.isEmpty ||
            parsed.userInfo.isNotEmpty) {
          if (mounted) setState(() =>
              _message = 'Нет выбранной HTTPS-подписки для автоматической регистрации');
          return;
        }
      }
      final data = await _channel.invokeMapMethod<String, dynamic>(
        'setBypassEnabled', {'enabled': enable, if (url != null) 'subscriptionUrl': url});
      if (!mounted) return;
      final state = data?['state']?.toString() ?? 'UNKNOWN';
      final reason = _bridgeError(data?['reasonCode']?.toString());
      setState(() {
        _enabled = data?['enabled'] == true;
        _message = switch (state) {
          'DISABLED' => 'Обход DPI выключен',
          'ACTIVE' => 'Локальный маршрут Bridge готов. Передача через сервер ещё не подтверждена.',
          'PREPARED' => 'Сертификат готов. Bridge включится при следующем подключении VPN.',
          'DIRECT_FALLBACK' => 'Bridge не включился: $reason. Прямой VPN сохранён.',
          'RESTART_FAILED' => 'Не удалось перезапустить VPN: $reason',
          'NO_BRIDGE' => 'На сервере текущей подписки Bridge отсутствует',
          'CANCELLED' => 'Регистрация отменена',
          'ENROLLMENT_FAILED' => 'Регистрация Bridge не выполнена: $reason',
          _ => 'Неизвестный результат регистрации Bridge',
        };
      });
    } on PlatformException catch (e) {
      if (mounted) setState(() {
        _enabled = false;
        _message = 'Автоматическая регистрация: ${e.code}';
      });
    } catch (_) {
      if (mounted) setState(() {
        _enabled = false;
        _message = 'Автоматическая регистрация Bridge не выполнена';
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Widget _flowStep(IconData icon, String title, String subtitle) => Container(
    width: double.infinity,
    padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 14),
    decoration: BoxDecoration(
      color: const Color(0xFF1B2030),
      borderRadius: BorderRadius.circular(12),
      border: Border.all(color: _gold.withOpacity(0.35)),
    ),
    child: Row(children: [
      Icon(icon, color: _gold, size: 26),
      const SizedBox(width: 12),
      Expanded(child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: const TextStyle(
              color: Colors.white, fontWeight: FontWeight.w600)),
          const SizedBox(height: 3),
          Text(subtitle, style: const TextStyle(
              color: Colors.white70, fontSize: 12)),
        ],
      )),
    ]),
  );

  Widget _flowArrow() => const Center(
    child: Padding(
      padding: EdgeInsets.symmetric(vertical: 4),
      child: Icon(Icons.arrow_downward, color: _gold, size: 20),
    ),
  );

  Widget _connectionDiagram() => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      const Text('Как работает автоматический обход',
        style: TextStyle(color: _gold, fontSize: 16,
            fontWeight: FontWeight.w600)),
      const SizedBox(height: 12),
      _flowStep(Icons.smartphone, 'TervonVPN',
          'Использует текущую выбранную HTTPS-подписку'),
      _flowArrow(),
      _flowStep(Icons.verified_user_outlined, 'Регистрация на сервере подписки',
          'Обнаружение Bridge, сертификат устройства и разрешение ACL'),
      _flowArrow(),
      _flowStep(Icons.lock_outline, 'Защищённый Bridge',
          'Параметры подключения выдаются сервером автоматически; TLS 1.3 и mTLS'),
      _flowArrow(),
      _flowStep(Icons.route_outlined, 'TCP-поток',
          'Маршрут выбранного профиля через Bridge; при ошибке — прямое подключение'),
      const SizedBox(height: 10),
      const Text('Адреса, порты и сертификаты вручную указывать не нужно.',
        style: TextStyle(color: Colors.white60, fontSize: 12)),
    ],
  );

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: _background,
      appBar: AppBar(title: const Text('Обход DPI'),
          backgroundColor: _background, foregroundColor: Colors.white),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        Card(color: _card, child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Tervon TCP Bridge v1.0.1',
                style: TextStyle(color: _gold, fontSize: 17)),
            const SizedBox(height: 10),
            const Text('Bridge включается только после регистрации сертификата и проверки локального TCP-маршрута.',
                style: TextStyle(color: Colors.white70)),
            SwitchListTile(
              title: const Text('Обход DPI', style: TextStyle(color: Colors.white)),
              subtitle: const Text('Автоматическая регистрация через текущую HTTPS-подписку; при ошибке — прямой маршрут',
                  style: TextStyle(color: Colors.white60)),
              value: _enabled, onChanged: _busy ? null : _toggleBypass,
            ),
            const SizedBox(height: 12),
            _connectionDiagram(),
          ]),
        )),
        const SizedBox(height: 12),
        Card(color: _card, child: Padding(
          padding: const EdgeInsets.all(16),
          child: SelectableText(_message,
              style: const TextStyle(color: Colors.white70)),
        )),
      ]),
    );
  }
}
