# Tervon TCP Bridge M6 — Android runtime route integration candidate

> **АРХИВНЫЙ СНИМОК ЭТАПА, НЕ ДЕЙСТВУЮЩАЯ ИНСТРУКЦИЯ (ревизия 2026-11-22).** Датированные статусы, старые примеры адресов/портов, команды cutover/patch и признаки NOT_IMPLEMENTED/PENDING ниже не отражают текущее состояние M6. Не выполнять команды из этого документа без нового утверждённого плана и сверки фактического кода/сервисов. Актуальные факты: [`DEVELOPMENT_PLAN.md`](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/DEVELOPMENT_PLAN.md), [`M6_AUDIT_2026-11-22.md`](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/M6_AUDIT_2026-11-22.md). Android Bridge E2E = FAIL; исторические PASS относятся только к явно названному слою.

**Источник:** проверенный архив TervonVPN DPI `tervonvpndpi-source-20261121.tar.gz`, затем подтверждённый оператором 8-файловый Stage4 transfer и 3-файловый автоматический переключатель. Установка исправленного экрана мониторинга подтверждена отдельно. Репозиторий Bridge: `git@github.com:tervon-software/Tervon-TCP-Bridge.git`, ветка `work/m6-android-ingress`.

## Что находится в комплекте

- `m6-runtime-route.patch`: единый текстовый патч для 7 существующих Kotlin/Flutter файлов + 2 новых Kotlin-файлов.
- `m6_manifest.json`: точные Git blob SHA-1 исходных и результирующих байтов девяти файлов.
- `install_m6_runtime_route.py`: SHA-guarded `--check` / `--apply`, подготовка в изолированной директории, проверка результата и попытка отката при ошибке записи. Дополнительно обновляет **две** ранее перенесённые Kotlin-зависимости из текущего GitHub: `BridgeAutoEnrollmentHttp.kt` и `BridgeClientConfigBuilder.kt`; проверяет их точные blob SHA-1.
- `test_installer_offline.py`: локальная симуляция установщика (в репозитории с архивом проверки).

**Точное число изменяемых файлов: 11.** Никакие конфигурации Linux Bridge, Xray, HEV, Android подпись, версии, systemd, Docker и публичные службы установщик не меняет. Применяется только к исходникам `/opt/tervonvpndpi`. Никакого APK не собирает.

## Строгая установка исходников

Перенесите каталог комплекта на Homeserver без изменения содержимого; синхронизируйте GitHub ветку M6 только SSH. Запускать из каталога распакованного комплекта:

```bash
python3 install_m6_runtime_route.py --repo-root /opt/tervon-tcp-bridge-m6-android-ingress --app-root /opt/tervonvpndpi --check
python3 install_m6_runtime_route.py --repo-root /opt/tervon-tcp-bridge-m6-android-ingress --app-root /opt/tervonvpndpi --apply
python3 install_m6_runtime_route.py --repo-root /opt/tervon-tcp-bridge-m6-android-ingress --app-root /opt/tervonvpndpi --check
```

Состояния `ALREADY_APPLIED` и `READY -> SOURCE_APPLIED -> ALREADY_APPLIED` допустимы. При `BLOCKED` ничего не переписывать вручную.

После исходников — отдельно проверка Flutter+Kotlin компиляции и полного APK на Homeserver. Затем реальные испытания на Android.

## Важные гарантии и ограничения

- Переключатель вручную ON: существующий HTTPS subscription discovery/CSR, сертификат текущего устройства, адрес VLESS backend из **проверенного HTTPS discovery**; несовпадение адреса/порта вызывает direct fallback.
- Переключатель OFF: существующий прямой Xray; никаких запросов регистрации.
- Перед `libv2ray.startLoop` Bridge пытается запустить локальный TCP listener и подготовить mTLS JSON. Только VLESS/TCP конфигурация proxy может быть перенаправлена, остальные типы — прямые.
- Приватный ключ хранится зашифрованным через Android Keystore и передаётся Go Bridge во временном app-private PEM файле согласно существующему механизму; временный PEM удаляется после запуска/ошибки/остановки.
- Из-за существующей `VpnService.Builder.addDisallowedApplication(packageName)` Go-процесс приложения не входит в TUN, что предотвращает VPN loop без новой JNI-передачи файлового дескриптора в `VpnService.protect()`.
- При внезапном выходе запущенного Bridge инициируется возврат к прямому Xray; если восстановление не удалось, VPN останавливается.
- Локальные метрики Go включены только на `127.0.0.1`, исходящий удалённый статус не объявляется подтверждённым без E2E.
- Существующий серверный `m6-auto-registrar` в GitHub теперь требует `--backend-host` и `--backend-port` для выдачи авторитетного сопоставления выбранного VLESS endpoint. Нужны корректные реальные значения профиля; их не следует выдумывать.

## Статусы доказательств

- Патч воспроизведён байт-в-байт на изолированном архиве: PASS.
- Guarded установщик на имитации реального исходного состояния: PASS (11 файлов, preflight→apply→already applied); SHA-проверки реальных GitHub Kotlin источников выполняются только на Homeserver.
- Чистая Kotlin-проверка `BridgeVlessRoutePlanTest`: PASS.
- Компиляция изменённых Android/Flutter исходников на Homeserver: NOT RUN.
- Серверный Go registrar с расширенным контрактом: CODE COMMITTED / TESTS NOT RUN.
- Новый публичный HTTPS ingress Android Bridge и registrar: NOT DEPLOYED, фактический TCP+HTTP2+mTLS Android E2E: NOT RUN.

**Нельзя называть M6 PASS или выпускать рабочий APK по одному только source-stage PASS.** Перед реальным E2E необходимо проверить публичную доступность Bridge и фактическую установку/активацию сертификата в серверном ACL. Работающие Linux службы не изменяются этим комплектом.