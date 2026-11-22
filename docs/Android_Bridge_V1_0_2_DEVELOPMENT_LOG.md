# Android Bridge v1.0.2 — журнал разработки с чистого начала

**Точка отсчёта пользователя:** 22.11.2026 18:00, Новосибирск (UTC+07:00) = 22.11.2026 11:00 UTC. Эта дата — утверждённая граница новой Android-разработки, **не время создания файла GitHub**. Фактическое время каждого последующего commit устанавливается по GitHub metadata.

**Проект:** Tervon TCP Bridge, только Android M6.
**Git SSH:** `git@github.com:tervon-software/Tervon-TCP-Bridge.git`
**Ветка на момент создания:** `work/m6-android-ingress`.
**HEAD до создания журнала:** `1b8097d9bcc0a596c413976f793420b5f186b709`.

## Неизменные правила этого журнала

1. Каждое действие фиксировать отдельно, без пропусков: фактическое время Новосибирска и UTC (по проверяемому источнику), исходный HEAD/commit, файл/механизм, основание, изменение, результат, следующий шаг. Если фактическое время не проверено — `TIME_NOT_VERIFIED`, не придумывать минуты.
2. `ПРЕДЛОЖЕНО`, `СОГЛАСОВАНО`, `РЕАЛИЗОВАНО`, `ПРОТЕСТИРОВАНО`, `PASS`, `FAIL`, `ВЫПУЩЕНО` — разные состояния. Никаких PASS по наличию кода.
3. После каждой значимой операции обновлять этот файл; все тесты с временем, командой/средой, HEAD, ожидаемым/фактическим результатом и ограничениями — в `docs/Android_Bridge_V1_0_2_DEVELOPMENT_JOB.md`.
4. Оператор считает старую Android-обвязку **неработоспособной** и **отменяет её как исходную базу**. Старые Android SHA-guarded патчи, `tcp_routes` и код старой семёрки не переиспользовать без нового согласования. Linux v1.0.2, серверный Bridge/Go-код, API 3x-ui, mTLS/PKI и работающие службы считаются существующими и не изменяются.
5. Строгий цикл внутри текущего M6: подтверждённые действующие входы/зависимости → согласованное очищение → разработка → комплексная проверка → исправления → доказанный PASS → оба журнала → commit. Переход к следующему шагу только после закрытия предыдущего.
6. Изменения версии APK, сборки/подписи, публикации и рабочей конфигурации не производить без отдельного разрешения.

## Записи

### Запись 001 — 22.11.2026, точка отсчёта 18:00 Новосибирск

- **Событие:** новое распоряжение пользователя о перезапуске **только Android-обвязки** v1.0.2 с чистого кода и обязательном полном журналировании.
- **Статус:** `СОГЛАСОВАНО` — завести этот журнал и отдельный журнал испытаний; `ТРЕБУЕТ УТОЧНЕНИЯ` — граница фразы «удалить все файлы в GitHub» (Android-артефакты либо весь репозиторий).
- **Существующие исходники Android:** семь ранее обсуждавшихся файлов признаны пользователем непригодными; **очищение файлов приложения ещё НЕ ВЫПОЛНЕНО**.
- **Серверная часть:** `SERVER_READY=OPERATOR_CONFIRMED`; запрещено переносить очистку на Linux, Go, 3x-ui, серверное PKI и рабочие службы без прямого отдельного решения.
- **Тесты:** `NOT_RUN`; никаких результатов новой Android-реализации нет.
- **Следующее действие:** согласовать точную область удаления; до этого допускаются только подготовка плана и журналов.

### Запись 002 — Создание двух журналов, время GitHub подтверждено

| Действие | UTC | Новосибирск UTC+07 | Commit |
|---|---|---|---|
| Создан `docs/Android_Bridge_V1_0_2_DEVELOPMENT_LOG.md` | 2026-11-22 11:05:08 | 2026-11-22 18:05:08 | `f1d69c119bc5dba8af2efd35509b535217310790` |
| Создан `docs/Android_Bridge_V1_0_2_DEVELOPMENT_JOB.md` | 2026-11-22 11:05:22 | 2026-11-22 18:05:22 | `65f722184926f90f25ec85dec991dcb86737d652` |

- **Источник времени:** `mcp__GitHub__fetch_commit`, поле `created_at`, для обоих указанных commit. Перевод в Новосибирск +07:00.
- **Проверено:** оба файла прочитаны по рабочей ветке через GitHub contents API; до создания ответ `NOT_FOUND`; commit изменения подтверждены.
- **Изменения исходников Android:** `NONE`.
- **Удаление файлов:** `NOT_STARTED`; нужно уточнить, что означает «все файлы в GitHub», прежде чем трогать готовый Linux/серверный проект.
- **Программные тесты Android:** `NOT_RUN`, см. JOB-000.

### Запись 003 — утверждённая очистка старой Android Bridge Stage 4, 22.11.2026

- **Распоряжение:** удалить из GitHub все проблемные файлы Bridge-инсталляции. Граница уточнена пользователем; готовые серверная часть, Linux v1.0.2, API 3x-ui, mTLS/PKI и работающие службы не затрагиваются.
- **HEAD до операции:** `aa5d3f1233cb71784dadbd7c4a038141a7f33a72`; **base Git tree:** `c696b561bb4e6615a59abf2cd8871430c1564791`; **ветка:** `work/m6-android-ingress`.
- **Инвентаризация GitHub Git Tree:** 226 blob-файлов, 71 файл целиком под `integrations/android-ingress/stage4/`, `truncated=false`.
- **Изменение:** единым атомарным Git commit удалить **все 71 старый исходный/установочный/тестовый Android Stage4 файл** с записью `sha:null` в Git tree. Архивные исходники, Kotlin/Dart копии, Python installers, Python patch generators, старые runtime-route kits и кандидаты восстановления не использовать вновь.
- **Полный манифест удаления:** 
  - `integrations/android-ingress/stage4/BridgeAutoCertificateAndroidAdapter.kt`
  - `integrations/android-ingress/stage4/BridgeAutoCertificateEnrollment.kt`
  - `integrations/android-ingress/stage4/BridgeAutoCertificateEnrollmentTest.kt`
  - `integrations/android-ingress/stage4/BridgeAutoEnrollmentChannel.kt`
  - `integrations/android-ingress/stage4/BridgeAutoEnrollmentHttp.kt`
  - `integrations/android-ingress/stage4/BridgeAutoEnrollmentHttpBindingTest.kt`
  - `integrations/android-ingress/stage4/BridgeCatalogTrustAnchors.kt`
  - `integrations/android-ingress/stage4/BridgeClientConfigBuilder.kt`
  - `integrations/android-ingress/stage4/BridgeClientConfigBuilderTest.kt`
  - `integrations/android-ingress/stage4/BridgeClientPemHandoff.kt`
  - `integrations/android-ingress/stage4/BridgeDeviceCsr.kt`
  - `integrations/android-ingress/stage4/BridgeDeviceIdentity.kt`
  - `integrations/android-ingress/stage4/BridgeDeviceSelfCheck.kt`
  - `integrations/android-ingress/stage4/BridgeDiagnostics.kt`
  - `integrations/android-ingress/stage4/BridgeServerCatalogTrust.kt`
  - `integrations/android-ingress/stage4/BridgeServerCatalogTrustSelfCheck.kt`
  - `integrations/android-ingress/stage4/BridgeServerCertificateStore.kt`
  - `integrations/android-ingress/stage4/BridgeServerReconciler.kt`
  - `integrations/android-ingress/stage4/BridgeServerReconcilerTest.kt`
  - `integrations/android-ingress/stage4/BridgeSignedServerCatalog.kt`
  - `integrations/android-ingress/stage4/BridgeVlessRoutePlan.kt`
  - `integrations/android-ingress/stage4/BridgeVlessRoutePlanTest.kt`
  - `integrations/android-ingress/stage4/BridgeVpnSession.kt`
  - `integrations/android-ingress/stage4/BridgeXrayConfigRoute.kt`
  - `integrations/android-ingress/stage4/BridgeXrayConfigRouteTest.kt`
  - `integrations/android-ingress/stage4/M6_RUNTIME_ROUTE_PACKAGE.ru.md`
  - `integrations/android-ingress/stage4/README.ru.md`
  - `integrations/android-ingress/stage4/TrojanOutboundParser.kt`
  - `integrations/android-ingress/stage4/TrojanOutboundParserTest.kt`
  - `integrations/android-ingress/stage4/activate-server-name-change.py`
  - `integrations/android-ingress/stage4/apply_m6_recovery_sources.py`
  - `integrations/android-ingress/stage4/apply_m6_session_diagnostics.py`
  - `integrations/android-ingress/stage4/apply_m6_trojan_grpc_fix.py`
  - `integrations/android-ingress/stage4/apply_to_tervonvpndpi.py`
  - `integrations/android-ingress/stage4/bridge_monitor_screen.dart`
  - `integrations/android-ingress/stage4/bridge_server_trust_screen.dart`
  - `integrations/android-ingress/stage4/bridge_settings_screen.dart`
  - `integrations/android-ingress/stage4/fix_bc_license_merge.py`
  - `integrations/android-ingress/stage4/fix_stage4_selfcheck_ui.py`
  - `integrations/android-ingress/stage4/install_auto_enrollment.py`
  - `integrations/android-ingress/stage4/install_bridge_monitoring.py`
  - `integrations/android-ingress/stage4/install_catalog_trust_asset.py`
  - `integrations/android-ingress/stage4/install_cert_store.py`
  - `integrations/android-ingress/stage4/install_m6_auto_discovery_fix.py`
  - `integrations/android-ingress/stage4/install_m6_auto_toggle.py`
  - `integrations/android-ingress/stage4/install_m6_auto_ui.py`
  - `integrations/android-ingress/stage4/install_m6_discovery_scheme.py`
  - `integrations/android-ingress/stage4/install_m6_failure_runtime.py`
  - `integrations/android-ingress/stage4/install_m6_reason_channel.py`
  - `integrations/android-ingress/stage4/install_m6_tcp_raw_fix.py`
  - `integrations/android-ingress/stage4/install_m6_trojan_app.py`
  - `integrations/android-ingress/stage4/install_m6_universal_tcp_source.py`
  - `integrations/android-ingress/stage4/install_m6_xhttp_source.py`
  - `integrations/android-ingress/stage4/install_per_server_catalog.py`
  - `integrations/android-ingress/stage4/install_server_pin_selfcheck.py`
  - `integrations/android-ingress/stage4/install_server_trust_ui.py`
  - `integrations/android-ingress/stage4/install_signed_catalog.py`
  - `integrations/android-ingress/stage4/m6-auto-toggle-source.patch`
  - `integrations/android-ingress/stage4/rebuild/remove_duplicate_runtime_channel.py`
  - `integrations/android-ingress/stage4/remove_manual_trust_ui.py`
  - `integrations/android-ingress/stage4/runtime-route-kit/README.ru.md`
  - `integrations/android-ingress/stage4/runtime-route-kit/install_m6_runtime_route.py`
  - `integrations/android-ingress/stage4/runtime-route-kit/m6-runtime-route.patch`
  - `integrations/android-ingress/stage4/runtime-route-kit/m6_manifest.json`
  - `integrations/android-ingress/stage4/stage-server-name-change.py`
  - `integrations/android-ingress/stage4/stage_m6_trojan_backend.py`
  - `integrations/android-ingress/stage4/stage_m6_xhttp_backend.py`
  - `integrations/android-ingress/stage4/test_install_catalog_trust_asset.py`
  - `integrations/android-ingress/stage4/test_stage_m6_trojan_backend.py`
  - `integrations/android-ingress/stage4/test_stage_m6_xhttp_backend.py`
  - `integrations/android-ingress/stage4/trojan_subscription_test.dart`
- **Оставить:** `prototype/`, `integrations/android-ingress/xui/`, `runtime-acl/`, `runtime-routes/`, `compose.yaml`, `config` server PKI/Go sources, действующие Linux релизы и оба новых журнала.
- **ВАЖНО:** физические `Bridge*.kt` в `/opt/tervonvpndpi` **этим коммитом не очищаются** — приложение является отдельным исходным деревом Homeserver. Запрещено утверждать обратное.
- **Результат и HEAD удаления:** устанавливаются отдельной проверкой опубликованного Git HEAD после атомарной операции; серверный E2E заново не тестировать.

### Запись 004 — Результат атомарного удаления и GitHub post-verification

- **Точный коммит:** `b4be0648674d86b41fe3a26706c3743d35f40505`.
- **GitHub commit.created_at:** 2026-11-22 11:08:28 UTC = **18:08:28 Новосибирск**.
- **Изменено в коммите:** 71 удалённый файл `integrations/android-ingress/stage4/` и два журнала; полный пофайловый манифест — Запись 003.
- **Независимая post-commit проверка:** GitHub HEAD совпадает с этим SHA; рекурсивное дерево `truncated=false`, осталось 155 blob-файлов, `STAGE4_FILES_REMAINING=0`. Подтверждены сохранённые `prototype/bridge.go`, `xui/inbounds.go`, `xui/enrollment_http.go`, `runtime-acl/runtime_policy.go`, `runtime-routes/policy.go`, `compose.yaml` и оба новых журнала.
- **Результат:** `GITHUB_OLD_ANDROID_STAGE4_REMOVAL=PASS`. **Это не тест Android APK.** Очищение семи файлов в проекте Homeserver `/opt/tervonvpndpi` — `NOT_DONE`: GitHub commit не изменяет другой рабочий каталог.
- **Следующий шаг внутри M6:** очищение/замена ровно семи файлов Android-приложения с сохранением отдельно утверждённых внешних вызовов и без изменения активных служб. Для физического изменения нужен доступ к текущему рабочему дереву приложения; не утверждать выполнение без вывода с Homeserver.

### Запись 005 — read-only восстановление Linux v1.0.2 и Android↔Server контракта (22.11.2026)

**Исходный HEAD:** `6beb9eae9d9a3ef3110b5e3cbcb9e13d14d242f6`, `work/m6-android-ingress`; время начала изучения по отдельному системному clock не подтверждено, `TIME_NOT_VERIFIED`. Для созданных Git commits время получаем из metadata post-commit.

**Прочитаны напрямую в GitHub:** `docs/V1_0_2_DEVELOPMENT_LOG.md` (сегодняшние Linux шаги 907–1029 и ARM64/Android шаги 1031–1061), `docs/RELEASE_NOTES_v1.0.2.ru.md`, source `prototype/bridge.go` и `prototype/cmd/bridge/main.go` **строго на release-v1.0.1 SHA `96c136702d10b6fb9a16aa2893f6b978e8b14e86`**, `integrations/android-ingress/runtime-acl/runtime_policy.go`, `apply-runtime-policy-to-release-source.py`, `compose.hot-acl.yaml`, `compose.yaml`, `xui/inbounds.go`, `route_reconcile.go`, `route_sync.go`, `policy_generation.go`, `confirmed_routes.go`, `enrollment_http.go`, `cmd/m6-auto-registrar/main.go`, `ENROLLMENT_CONTRACT.ru.md`. Проверен настоящий GitHub Release API `/releases/tags/v1.0.2`.

**Linux Release:** опубликован `2026-11-22T07:53:17Z` GitHub v1.0.2 (stable, draft=false, prerelease=false), asset `tervon-bridge-linux-amd64-v1.0.2` 10,332,713 bytes, SHA256 `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371`, плюс `SHA256SUMS` и `BUILD_INFO`. Сборка: release base 96c1367 + tested overlay `40dc00a` — НЕ `prototype/` нынешней ветки. `go test`, `go test -race`, `go vet` (6 packages), isolated real-process SIGHUP/routes+ACL/digest ACK и отказы/rollback: **операторский PASS в старом журнале**.

**Server v1.0.2:** TLS 1.3 + mTLS + ALPN h2; `POST /v2/stream` c заголовками `Tervon-Service`, `Tervon-Wire-Version: 1`, `Tervon-Wire-Capability`. Server-side `RuntimePolicy`: одна immutable atomic generation пары `routes+authorized`; проверяет fingerprint и service, выбирает backend только из доверенной конфигурации, не принимает произвольный backend от Android; SIGHUP меняет только routes+ACL, сохраняет другие настройки и существующие TCP sessions. Исходный v1.0.1 TCP framing/FIN, padding/rate padding/jitter, обратная передача bytes, лимиты, диагностические счётчики сохранены. Android должен пользоваться **существующим Go ARM64 ядром** как TCP Encapsulator, а не изобретать новый HTTP/2 framing.

**Docker:** опубликованный Linux бинарник установлен оператором в отдельный `debian:12-slim` image `tervon-bridge-android-ingress:v1.0.2` (image ID `sha256:ce484cdc64c56428e89ecc933e64616ec4986004bb8a28bf2ce953b7766353a9`). Контейнер `android-ingress-android-bridge-decapsulator-1`, процесс UID:GID 65532, read_only, no-new-privileges, cap_drop ALL, restart unless-stopped, mount `/config:ro`. Docker Compose v5.5.1, base+hot override с `ports: !override`: **один** host port `0.0.0.0:18443 -> 8443/tcp`, `pull_policy:never`. Исторический `COMPOSE_CONFIG=PASS`, `COMPOSE_PERSISTENCE=PASS`, `FINAL_CHECK_RC=0`, `RESTART_COUNT=0`, `DOCKER_CUTOVER=PASS`. Старый образ сохранён для rollback. Эти evidence доказывают deployment/config; они **не** доказывают реальный Android→backend E2E. Оператор подтвердил готовность серверной части, не переоткрывать её разработку.

**Android↔Server договор:** Android выбирает TCP-профиль из действующей подписки; HTTPS host подписки задаёт server discovery; при включении обхода GET trusted discovery, затем если нужно формируется P-256 CSR локально, POST CSR к существующему registrar; private key остаётся на устройстве; сервер подписывает сертификат существующим Client CA и подтверждает применение fingerprint в runtime ACL. Верифицировать server CA / serverName, использовать HTTPS/2 TLS 1.3 mTLS, исходный `/v2/stream` + wire capabilities, согласованный opaque service; Go слушает локальный loopback, Xray outbound перенаправляет TCP dial туда без изменения VPN credentials/Host/SNI/authority/payload; process sockets исключаются из VPN loop; все ошибки разделяются по стадиям. При выключении обхода обычный VPN не изменён.

**КРИТИЧЕСКАЯ несовместимость текущих источников GitHub:** `xui/cmd/m6-auto-registrar/main.go` всё ещё вызывает `configuredTCPRoutes(root)`, считывает `config/server.json` и передаёт `TCPRoutes`; `xui/enrollment_http.go` имеет `TCPRoutes`/legacy response, а `xui/confirmed_routes.go` содержит вспомогательную map. Пользователь отменил `tcp_routes` для целевой Android-архитектуры; не использовать, не восстанавливать и **не делать вывод**, что работающий сервер использует старый код. Действующий deployment registrar identity отдельно НЕ ПОДТВЕРЖДЁН этим GitHub read-only исследованием. Вопрос: каким **реально действующим** механизмом клиент узнаёт корректный opaque service для выбранного TCP inbound **без отдельной `tcp_routes`**? Нельзя придумывать API, включать static port mapping или менять готовый сервер без утверждения; проверить существующий опубликованный интерфейс при новой Android реализации.

**Результат:** `SOURCE_AND_RELEASE_RECONSTRUCTION=PASS_READ_ONLY`, `DOCKER_DEPLOYMENT=PASS_OPERATOR_EVIDENCE`, `SERVER_READY=OPERATOR_CONFIRMED`, `ANDROID_NEW_CODE=NOT_STARTED`, `ANDROID_DEVICE_E2E=NOT_RUN_NEW_BASELINE`. `SERVICE_BINDING_CONTRACT=NEEDS_CURRENT_INTERFACE_EVIDENCE`. Новых исходников/бинарников/служб не меняли, серверных тестов заново не запускали.
