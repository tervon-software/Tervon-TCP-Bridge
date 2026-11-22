# Tervon TCP Bridge — Server: свидетельства аудита M6

> Разделено по запросу владельца из ранее единого документа. Все исходные строки сохранены в Server/Android документах, история commit не изменена. Области PASS действуют только на соответствующие испытания.

## 1. Утверждённый неизменный контракт

- **Data plane:** Xray Android → локальный Go Encapsulator (непрозрачные TCP-байты) → HTTP/2 + TLS 1.3 + mTLS → существующий серверный Go Decapsulator → уже определённый серверной системой Xray TCP backend; обратная передача побайтно идентична. Никаких разбора VLESS/Trojan/XHTTP, протокольных маршрутизаторов или замены Host/SNI внутри Bridge.
- **Control plane:** фактические действующие inbound/порты берутся через **существующий 3x-ui API**, сопоставляются с эффективными Xray handlers и подтверждёнными маршрутами Bridge. API, регистрация, сертификаты, авторизация/ACL и серверный Decapsulator **не создаются заново**.
- **server.json** — конфигурация Bridge (политика, routes и ACL), **не первоисточник протоколов/портов 3x-ui**. `tcp_routes` исключена решением пользователя 2026-11-22; не восстанавливать отдельную таблицу портов, включая в Android. Статическое восстановление карты из `server.json` само по себе не удовлетворяет утверждённому API-контракту.
- **Переработка M6:** пользователь разрешил инвентаризацию и последующее удаление/очистку ошибочной Android-обвязки с пересозданием только этой обвязки. **Перечень удаляемых исходных файлов и новая реализация ещё не утверждены пофайлово.** До согласования никакого удаления кода, смены подписи APK, замены Linux Bridge или изменений сервисов.

## 2. Текущая матрица проверки исходников

| Объект | Путь | Доказано по GitHub | Интеграционный статус |
|---|---|---|---|
| Выпущенный Linux Go Bridge v1.0.1 | `prototype/bridge.go`, `prototype/cmd/bridge/main.go` на base `96c136702d10b6fb9a16aa2893f6b978e8b14e86` | Контракт TCP stream, TLS 1.3/mTLS/HTTP2; v1/v2 framing в этом commit | Base Linux исходники, не равны текущему устаревшему файлу на M6-ветке |
| Linux v1.0.2 | `integrations/android-ingress/runtime-acl/runtime_policy.go`, overlay/build scripts и GitHub Release v1.0.2 | Atomic `Routes+ACL`, ACK по digest; Linux AMD64 SHA `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371` | Operator: build/race/vet/isolated runtime/Compose PASS; Android E2E FAIL |
| 3x-ui API inbound | `integrations/android-ingress/xui/inbounds.go` | `Client.ListInbounds()` GET `/panel/api/inbounds/list/slim`, inbound port/protocol/network/enable | Source and isolated tests; действующий read-only API подтверждался отдельно |
| Сверка panel ↔ Xray | `xui/route_reconcile.go`; `xui/route_sync.go` | `ReconcileInbounds`, `SyncVerifiedRoutes` требуют эффективное Xray evidence, policy digest, ACK | **SOURCE_ONLY coordinator seam**, не установлен факт его подключения к работающему регистратору |
| Xray live evidence | `check_xray_grpc_runtime.py`, `check_panel_xray_bridge_routes.py` | Трёхсторонний read-only аудит panel/Xray/старых Bridge routes зафиксирован PASS в журнале | Это аудит, а не автоматическая публикация новой политики |
| Confirmed discovery | `xui/confirmed_routes.go`, `xui/enrollment_http.go` | `ConfirmedRouteStore` поддерживает подтверждённое поколение и fail-closed, есть альтернативный legacy `TCPRoutes` | В `cmd/m6-auto-registrar/main.go` сейчас выбран **legacy** путь |
| Legacy registrar | `xui/cmd/m6-auto-registrar/main.go` | `configuredTCPRoutes()` читает `config/server.json` при старте; публикует `TCPRoutes:ports` | **Подтверждённый source-конфликт** с утверждённым API-first контрактом; фактическая версия развернутого регистратора НЕ СВЕРЕНА |
| Старый release `prototype/bridge.go` на M6 HEAD | `prototype/bridge.go` на `work/m6-android-ingress` | Веточная версия `/v1/stream` отличается от выпущенного Go base `/v1`+`/v2` | **LEGACY SOURCE**; не брать для сборки новых Android/Linux без сверки exact release |
| Android app | Предоставленный `tervonvpn-m6-source-audit.tar.gz`, `/opt/tervonvpndpi` у оператора | Kotlin/Flutter Bridge files, запуск ELF через `ProcessBuilder`, selected profile → local loopback, automatic subscription discovery; список см. ниже | Снимок до последней пользовательской сборки; устройство с новым ядром даёт NO INTERNET |

### Проверка реального устройства, 2026-11-22

Пользователь собрал/установил APK и включил Bridge. ADB Windows: Xray `transport/internet/splithttp` GET/POST `EOF`; повторные read `127.0.0.1:<ephemeral> → 127.0.0.1:<listener>` `connection reset by peer`; Android Bridge `code=UNKNOWN generation=1`. DNS ошибки вторичны/причинность не доказана. **`ANDROID_E2E=FAIL_NO_INTERNET`, `ROOT_CAUSE=UNKNOWN`.** Go ARM64 diagnostic executable SHA `a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd` установлен в исходники; Go test/race/vet и Kotlin compile PASS, но это не functional PASS.

## 7. Непроверенный runtime provenance — перечень блокирующих вопросов

1. На каком точном Git commit и из какого Go source собран **действующий** `m6-auto-registrar`? В GitHub код `configuredTCPRoutes(root)` по-прежнему legacy; наличие `route_sync.go` не означает его вызова в этом процессе.
2. Какой реально effective runtime policy digest применён серверным Bridge и как Android получает актуальный маршрут без отменённой `tcp_routes`? Сравнение с panel/Xray без показа приватных body, client ID, токенов и адресов.
3. Какой exact SHA у исходников Android после последней сборки APK, и какие Bridge-файлы на Homeserver отличаются от изученного snapshot?
4. Подтверждено ли на текущей серверной стороне одно успешное HTTP/2 POST соединение с client certificate и backend dial? Предыдущие отдельные TLS ALPN/Compose проверки не равны E2E.

**Следующий пакет:** строго read-only проверка фактического исходного HEAD/процесса и действующего Android source; затем согласование перечня очистки. Никаких APK, Docker `up`, `kill`, `SIGHUP`, редактирования ACL и `server.json`.

