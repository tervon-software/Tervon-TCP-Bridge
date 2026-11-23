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

### Запись 006 — Проверенные временные метки записи исследования

- `0ff4a09647029dbfdd9f7a14650d77f2621a5d75` — `2026-11-22 11:15:48 UTC` = **18:15:48 Новосибирск**, дополнение журнала разработки (Запись 005).
- `d3981c3e4ff7fee335fbdcdd727c237b72a24e02` — `2026-11-22 11:15:51 UTC` = **18:15:51 Новосибирск**, дополнение журнала испытаний (JOB-003).
- Источник времени: GitHub `fetch_commit.created_at`. Изменены только docs; новых тестов/Android-сборок не было.

### Запись 007 — recovered Android ARM64 v1.0.2 + diagnostics lineage (read-only)

- **Основание:** прямое требование пользователя восстановить подтверждённую историю Go ARM64-ядра из v1.0.2, диагностических изменений и установленного APK. **Исходный HEAD:** `a29f0d45a83b012e34fc96ac05b3332e3380234e`, `work/m6-android-ingress`. Новая Android-разработка от 18:00 не изменялась; изучалась только история из `docs/V1_0_2_DEVELOPMENT_LOG.md` строки 949–961, 1031–1061.
- **Точная исходная сборка:** `/opt/tervon/Developed/v1.0.2-runtime-policy-validation/prototype` — Linux release v1.0.1 base `96c136702d10b6fb9a16aa2893f6b978e8b14e86` + полный RuntimePolicy overlay tested at `40dc00a24fb43ea05e1adf4a62845cb5546ca1bd`. Первое Android ARM64 исполнение `GOOS=android GOARCH=arm64 CGO_ENABLED=0`, Go 1.25.5, ELF64 PIE AArch64 `/system/bin/linker64`; `/opt/tervon/Developed/v1.0.2-android-arm64/libtervonbridge.so`, SHA256 `d71607005c510ec1d97787f656b4b5580d10d750ec1031b9b18630af266ac487`, `ANDROID_BUILD_RC=0`.
- **Обнаруженный диагностический дефект старого Android:** `BridgeCoreManager.kt` подавлял merged Go stdout/stderr через `/dev/null` и мог блокировать единственный executor на `waitFor()`; Go `Client.ListenAndServe` отбрасывал ошибку `_ = cl.HandleContext(sessionCtx,c)`. Старые мониторинговые данные отражали локальную готовность, а не реальный H2/mTLS/backend.
- **Диагностическая модернизация Go:** guarded patch `/opt/tervon/Developed/tervonvpn-m6-diagnostics-fix/apply_go.py` наложен на изолированную копию той же tested release v1.0.2 ветки; сохраняет ошибку `HandleContext` как фиксированный безопасный marker без исходного raw error/секретов, частота не более одного marker в секунду. Полные `go test -count=1 ./...`, `go test -race -count=1 ./...` (6 packages), `go vet ./...`, Android ARM64 crosscompile `go build -trimpath ./cmd/bridge`: `PASS`, `M6_GO_BATCH_RC=0`. **Новый ELF:** `/opt/tervon/Developed/m6-go-diagnostics-5HZL1E/libtervonbridge.so`, SHA256 `a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd`.
- **Диагностическая модернизация Kotlin/Dart в старом приложении:** `BridgeCoreManager.kt` отдельный daemon drain для merged output, 64-event ring, fixed code categories, bounded/redacted markers и independent process-exit watcher; `BridgeProcessDiagnostics.kt`, `BridgeRuntimeControl.kt`, `BridgeDiagnostics.kt`, `lib/features/home/bridge_monitor_screen.dart`. На Homeserver installer `--check` показал `ALREADY_APPLIED` всех пяти, `ANDROID_SOURCE_VERIFY_RC=0`; `./gradlew --offline :app:compileDebugKotlin` `BUILD SUCCESSFUL`, `KOTLIN_RC=0`. Flutter CLI не найден, `FLUTTER_RC=127`; Flutter тесты `NOT_RUN`, пакет `M6_BATCH_RC=1` по недоступности инструмента.
- **Операторская установка Go:** исходный `libtervonbridge.so` SHA `d71607...` проверен и сохранён под `/opt/tervon/Developed/libtervonbridge-m6-before-diagnostics.so`; новый ELF SHA `a7f693...` атомарно скопирован в `/opt/tervonvpndpi/android/app/src/main/jniLibs/arm64-v8a/libtervonbridge.so`; `M6_ANDROID_CORE_INSTALL=PASS`, `M6_INSTALL_RC=0`; `DOCKER=UNCHANGED`.
- **APK и реальный телефон:** позднее оператор собрал и установил APK с новым ELF. ADB 15:41–15:43 зафиксировал `transport/internet/splithttp` GET/POST `EOF`, локальные `127.0.0.1` connection reset, `TervonBridge: code=UNKNOWN generation=1`; нет интернета. `M6_APK_BUILT_AND_INSTALLED=OPERATOR_REPORTED`, `ANDROID_E2E=FAIL`, `ROOT_CAUSE=UNDETERMINED`, `SESSION_CLASSIFICATION=UNKNOWN`. Логи **не доказывают**, что виноват исключительно Kotlin-wrapper: это гипотеза оператора, требующая проверки новой реализацией.
- **Обязательная граница перезапуска Android:** **Go ARM64 ELF SHA `a7f693...` — подтверждённый установленный артефакт, НЕ удалять/перекомпилировать из-за очистки старых семи Kotlin-файлов.** Старые Kotlin/Dart исходники как база новой обвязки отменены пользователем, но полезные требования к безопасной диагностике сохранить. Точная текущая checksum ELF на Homeserver после описанного теста не считана снова в этой проверке — `CURRENT_HOST_ELF_HASH=NOT_RECHECKED`; рабочие службы не трогать.
- **Результат:** `ANDROID_CORE_BUILD_DIAGNOSTICS=PASS_OPERATOR_RECORD`, `ANDROID_ELF_SOURCE_INSTALL=PASS_OPERATOR_RECORD`, `KOTLIN_DIAGNOSTIC_COMPILE=PASS_OPERATOR_RECORD`, `APK=OPERATOR_BUILT_AND_INSTALLED`, `DEVICE_E2E=FAIL`; **новые тесты с 18:00: NOT_RUN**.

### Запись 008 — Старт чистой Android v1.0.2 реализации, модуль TCP route decision

- **Утверждение:** оператор прямо разрешил начинать новую Android Bridge-обвязку без самовольной смены механизмов. Сохранить Linux/Docker v1.0.2, существующий диагностический Go ARM64 `libtervonbridge.so` и исходники обычного VPN. Старые Stage4 удалены, новый код не основан на них.
- **Исходный HEAD:** `49f8a8f1af45e35df050dcfc21e11d0c51313ab6`, `work/m6-android-ingress`. Перечитаны `docs/DEVELOPMENT_PLAN.md`, `docs/Android_Bridge_V1_0_2_DEVELOPMENT_LOG.md`, `docs/Android_Bridge_V1_0_2_DEVELOPMENT_JOB.md`, `integrations/android-ingress/ENROLLMENT_CONTRACT.ru.md`.
- **Первый минимальный модуль:** pure Kotlin решение о перенаправлении TCP-потока выбранного Xray outbound к локальному `127.0.0.1` listener. Новые файлы **не в старом Stage4**:
  - `integrations/android-bridge-v1.0.2/route/BridgeVlessRoutePlan.kt`, SHA256 `d62fea47cb9ff8852499c49985f212823ce234b3c385083a92acc68d3b3fb8c8`, Git blob `0dee3044f466a400a0fab743bbad8c3b4b4e1b0c`.
  - `integrations/android-bridge-v1.0.2/route/BridgeVlessRoutePlanTest.kt`, SHA256 `394a7419ba329927aa35492fe8c1a4afb5a84266b54c230ec954add0366e6dea`, Git blob `00bef36ab5b02db14f26afd62eea068d1ade37a5`.
- **Проверка:** `kotlinc` 1.9.0 / Java 21 compile и `java -jar` unit test: `ANDROID_BRIDGE_V102_ROUTE_UNIT=PASS`; проверены TCP/raw/ws/gRPC/httpupgrade/http/h2/xhttp/splithttp, null/empty default, запрет UDP/quic/h3/kcp/unknown/whitespace, disabled direct, неподходящий адрес/порт, непроверенный Discovery, неготовый listener, bounds. Локальный `git hash-object` **точно совпал** с GitHub blob SHA обоих файлов: тестировалась именно опубликованная версия, не «эквивалентная».
- **Commit исходников:** `29375777562f8e54f1d78f9dfbacb0d8e6c1bcb3`.
- **Безопасные ограничения:** решение не читает Xray JSON, не запускает Go, не создаёт сокеты и сертификаты, не использует `tcp_routes` или legacy Django/Flutter channels, не интерпретирует VPN протокол. `localListenerReady` — **только локальная готовность**, не TLS/H2/backend/Internet PASS.
- **Статус:** `ROUTE_POLICY=IMPLEMENTED`, `ISOLATED_KOTLIN_UNIT=PASS_EXACT_GIT_BLOBS`; `ACTUAL_APP_SOURCE_INTEGRATION=NOT_RUN`, `ANDROID_GRADLE=NOT_RUN`, `DEVICE_E2E=NOT_RUN`, `SEVEN_OLD_HOME_SERVER_FILES_CLEANUP=NOT_DONE`. Рабочие службы и исходники Android-приложения **не изменялись**.
- **Блокер следующей интеграции:** текущее дерево `/opt/tervonvpndpi` и действующие call-sites не прочитаны; старый архив отменён оператором. Не зашивать архитектуру Android callbacks до подтверждения актуальных файлов. Нельзя отмечать весь M6 как готовый.

### Запись 009 — Точное время создания первого кода и записей

Источник — GitHub `fetch_commit.created_at`, UTC и Новосибирск UTC+7:
- Код + точные unit-файлы: `29375777562f8e54f1d78f9dfbacb0d8e6c1bcb3`, **2026-11-22 11:42:05Z / 18:42:05 (+07)**.
- DEVELOPMENT_LOG запись 008: `49eddf43aa14858fc1d10fed7c09d31a995ed7a1`, **11:42:31Z / 18:42:31 (+07)**.
- DEVELOPMENT_JOB запись JOB-005: `6f149f1c40d4621a13e4dfe7fc1adc5e2d01da02`, **11:42:44Z / 18:42:44 (+07)**.
- Отдельная локальная метка начала/конца unit теста не собиралась, поэтому `TEST_EXECUTION_TIME=NOT_MEASURED`; времена Git commit не выдавать за точное время самого теста.

### Запись 010 — текущий Android app preflight для первого модуля, 22.11.2026

- **Запрос пользователя:** «Продолжай». **Текущий модуль:** M6 Android; после первого pure route-policy unit PASS — чтение актуальных точек подключения в приложении. Не переходить к запуску Go/Xray integration до доказательства действительных call-sites.
- **HEAD перед действием:** `d2c46e380eaa465db7fc369a058005c401c2cae9`, ветка `work/m6-android-ingress`, repo SSH `git@github.com:tervon-software/Tervon-TCP-Bridge.git`.
- **Повторный тест exact source route:** `kotlinc BridgeVlessRoutePlan.kt BridgeVlessRoutePlanTest.kt -include-runtime -d route-test.jar && java -jar route-test.jar` → `ANDROID_BRIDGE_V102_ROUTE_UNIT=PASS`, exit 0. Git blob hashes exact: `0dee3044f466a400a0fab743bbad8c3b4b4e1b0c`, `00bef36ab5b02db14f26afd62eea068d1ade37a5`; JVM Kotlin 1.9.0, JRE 21.
- **Проверка доступности действующего Android source:** GitHub org repo listing показал `CipherElite`, `Xray-core`, `Tervon-Federation`, `Tervon-TCP-Bridge`; отдельный `tervonvpndpi` GitHub repo не обнаружен. Локально доступен только архив старых исходников, явно отменённый оператором; использовать его для нынешних call-sites **запрещено**. Невозможно честно объявить физическую очистку семи файлов или интеграцию без прямого чтения актуального Homeserver source.
- **Новый изолированный диагностический инструмент без модификации приложения:** `integrations/android-bridge-v1.0.2/route/current_app_preflight.py`. Только чтение `/opt/tervonvpndpi`: ровно семь имён Kotlin файлов, SHA-256, пути и номера строк Bridge references в Kotlin/Dart (без содержимого строк), read-only `git branch/HEAD/diff`, SHA-256 `jniLibs/arm64-v8a/libtervonbridge.so` и соответствие подтверждённому ELF `a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd`. Не выводить конфигурацию, секреты, private source. Никаких записей, команд изменения служб или APK.
- **Тесты инструмента:** `python3 -m unittest -v test_current_app_preflight.py`: 3/3 PASS (read-only/no-source-leak, missing root refused, unsafe symlink refused), `python3 -m py_compile` PASS; local source SHA256 `82adb424e068b6d74e40848a16d53df8df5ee59e7a03978ce2b060356438ee97`, test `4c000b8316a9730818e0ed93f0384bb5f4d4593d7f0bfaaf66b0fe3167c4f050`. Git blob SHA exact: `e8ecf08df3e7dd561ce0e4af79e83e729b3e9185`, `fb7d1da713fd2e609b68fa64bf9376f06d85102f`, сверены **до commit**.
- **GitHub commit:** `012faf18a5618071fcf5fb4c1345cad39dde1d00`, GitHub `created_at=2026-11-22T11:46:44Z` = **18:46:44 Новосибирск (+07)**. Изменены только два test/preflight файла в новом каталоге Android Bridge.
- **Доказанные результаты:** `ROUTE_POLICY_UNIT=PASS_REPEAT_EXACT_GIT`; `CURRENT_APP_PREFLIGHT_TOOL=PASS_SYNTHETIC`; `CURRENT_APP_PREFLIGHT_ON_HOMESERVER=NOT_RUN`; `CURRENT_APP_SOURCE_IDENTIFIED=NO`; `APP_SEVEN_FILES_CLEARED=NOT_RUN`; `APK/DEVICE_E2E=NOT_RUN`. Следующий шаг в **том же модуле** — прочитать реальный output preflight Homeserver и сверить source call-sites до source replacement; никаких предположений о действующем Android.

### Запись 011 — Фактический preflight Homeserver от оператора, 2026-11-22

- **Источник:** загруженный оператором вывод `current_app_preflight.py --app-root /opt/tervonvpndpi` после `git pull --ff-only origin work/m6-android-ingress`; Bridge GitHub fast-forward `36864cb..0e1aa67`. Проверка только чтение; `CURRENT_APP_PREFLIGHT=PASS_READ_ONLY`. Фактическое время выполнения команды не предоставлено, `EXECUTION_TIME=UNKNOWN`; за дату использовать timestamp коммита этого журнала, НЕ выдавать его за время preflight.
- **Bridge HEAD на время принятия результата:** `0e1aa67bff5585048f6ae86310359c2cb1ef6789`.
- **Действующий Git TervonVPN:** `/opt/tervonvpndpi`, ветка `feature/status-ping-cleanup`, HEAD `22ff78a9dbabc1544a53b9ac950f78413f25b4fb`, **`tracked_dirty=true`**. До редактирования обязательно получить `git status --short` и source, чтобы не уничтожить чужие незакоммиченные изменения.
- **Подтверждённое ядро:** `android/app/src/main/jniLibs/arm64-v8a/libtervonbridge.so` присутствует, SHA256 **`a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd`**, совпадает с диагностическим Go ARM64 v1.0.2. `GO_CORE_PRESERVE=REQUIRED`.
- **Семь старых файлов, точно один экземпляр каждого в `android/app/src/main/kotlin/com/tervonvpndpi/app/`:**
  1. `BridgeConfigManager.kt` SHA256 `0c1b2815e2c7d866e9870411052c80cfd316b597044ca48fc2f99369e3a80e0d`
  2. `BridgeControlChannel.kt` SHA256 `0e503518ef49e860e0a48be2b25f68dea3705d39d9602e936604c487af750eef`
  3. `BridgeCoreManager.kt` SHA256 `aceadb4b9431a8ac33ece6b60c13547e3917bb0cfb8e5efe59ec02c03e4d369e`
  4. `BridgeRuntimeControl.kt` SHA256 `a6f9695d4cdc6a3d120ac0f685ddd6f2d9ed4a122469e13135e542dbfacdc596`
  5. `BridgeVlessRoutePlan.kt` SHA256 `b2d211029d74d29e9e9fdd4b26dc45046d65c520d6d4f4053c3850b82d6dd26c`
  6. `BridgeVpnSession.kt` SHA256 `5bcf45125cdad9b9220ef7d983e255f2d6a92342b65fc2e2ceccdabf29681838`
  7. `BridgeXrayConfigRoute.kt` SHA256 `d59e1c147507e70b85554d3d2dd31296ca4a81691529310bd99de201a2b09def`
- **Установленные зависимости (только locations, не доказательство поведения):** `VpnForegroundService.kt` содержит 6 упоминаний `BridgeVpnSession`; `MainActivity.kt` — `BridgeControlChannel`; `BridgeRuntimeSettings.kt` — `BridgeVpnSession`; `BridgeClientPemHandoff.kt` — `BridgeConfigManager`; `BridgeCoreManager.kt` — `BridgeProcessDiagnostics` (всего 17 в разных классах) и `BridgeConfigManager`. `BridgeVpnSession` ссылается на route plan, Xray config route, runtime control и core.
- **Вывод:** физически старые семь файлов **не удалены**; полный runtime/API contract этих вызовов по списку номеров строк НЕ восстановлен. Для точной минимальной замены без поломки обычного VPN необходимы реальные тексты семи Kotlin-файлов и вызывающих участков (приватно, без публикации), плюс статус незакоммиченных файлов. Старые архивные исходники больше не брать за основу.
- **Статус:** `CURRENT_APP_PREFLIGHT=PASS_OPERATOR_RECORDED`; `ARM64_DIAGNOSTIC_CORE_SHA=PASS_OPERATOR_RECORDED`; `SEVEN_FILES_LOCATED=7_OF_7`; `CALLSITE_LOCATIONS=FOUND`; `APP_SOURCE_CONTENT=NOT_READ`; `WORKTREE_CLEAN=NO`; `APP_INTEGRATION=BLOCKED_PENDING_CURRENT_SOURCE`; `ANDROID_E2E=NOT_RUN_NEW_IMPL`. Ни один файл приложения, Docker или Go не изменён.

### Запись 012 — Анализ актуального приватного архива Android (22.11.2026, после JOB-007)

- **Источник:** оператор передал в чат `android-bridge-v102-current-source.tar.gz` из действующего `/opt/tervonvpndpi`; архив **не загружать в публичный GitHub**. Архив SHA256 `02b960f9f9dd5d7f6486cc593217146df966dc05a1b50336008f6a50cb235a16`.
- **GitHub исходный HEAD:** `52c8e8a01b9fd2cd7231bdfb7f27b72c9b7fb3a1`; ветка `work/m6-android-ingress`. Git приложения согласно JOB-007: `feature/status-ping-cleanup` @ `22ff78a9dbabc1544a53b9ac950f78413f25b4fb`, `tracked_dirty=true`.
- **Архив проверен без распаковки файлов в приложение:** ровно 12 regular tar entries под `android/app/src/main/kotlin/com/tervonvpndpi/app/`, нет symlink/traversal. Все семь целевых Kotlin SHA256 **совпали 7/7** с read-only Homeserver preflight JOB-007. Остальные пять: `VpnForegroundService.kt`, `MainActivity.kt`, `BridgeRuntimeSettings.kt`, `BridgeClientPemHandoff.kt`, `BridgeProcessDiagnostics.kt`.
- **Фактические call-sites обычного VPN:**
  - `MainActivity.kt:36` регистрирует `BridgeControlChannel.register`.
  - `VpnForegroundService.kt:367` вызывает `BridgeVpnSession.prepare` при hot-switch; `:565` перед `Libv2ray.startLoop`, `:376,381,639,834` — остановка при rollback/error/stop.
  - `BridgeRuntimeSettings.kt:19–20` читает `BridgeVpnSession.active`.
  - `BridgeClientPemHandoff.kt:47,100` зависит от `BridgeConfigManager`.
- **Подтверждённые дефекты старой обвязки (исходники, не доказанный единственный root cause отсутствия интернета):**
  1. `BridgeControlChannel.kt:15` регистрирует `BridgeRuntimeControl.register`, который создаёт **второй** `MethodChannel` `com.tervonvpndpi/bridge_runtime` (`BridgeRuntimeControl.kt:12,35–75`) со своими командами конфигурации/запуска/остановки мимо `BridgeVpnSession`. Возникает **конкурирующее управление** одним Go-процессом и несогласованность `vpnRouteIntegrated=false` (`:84`).
  2. `BridgeVpnSession.kt:63–64` выбирает service через `params.tcpRoutes[selected.port]` — **отменённая статическая route map**, несовместима с утверждённым новым контрактом.
  3. `BridgeVpnSession.kt:98–127` объявляет `LOCAL_ROUTE_READY` и меняет Xray JSON после проверки только локального `127.0.0.1` listener; удалённый H2/mTLS, backend и двунаправленные байты не подтверждаются. Это корректная локальная метрика, **не доказательство E2E**.
  4. `BridgeVpnSession.kt:128–133` ловит все исключения и молча возвращает прямую конфигурацию; ошибка скрывается от запускающей стороны, кроме stage code.
  5. `BridgeXrayConfigRoute.kt:27–140` переписывает Xray outbounds и пытается сохранять Host/SNI/authority. Требуется проверить точное соответствие генерируемому Xray JSON для TCP/XHTTP/Trojan и корректную связность, а не копировать старую реализацию.
- **Защитная оговорка:** `VpnForegroundService.kt:578,914` уже вызывает `addDisallowedApplication(packageName)`, исключая собственный пакет из TUN на первичном старте и reconnect. **Нельзя** объявлять отсутствие VPN protect доказанной причиной по этим исходникам. Также не менять HEV, VPN builder, обычный VPN, подписи, Go ELF или Docker без отдельного решения.
- **Недостающие зависимости для компиляции новой обвязки:** архив НЕ включает актуальные `BridgeAutoEnrollmentHttp.kt`, `BridgeSubscriptionBinding.kt`, `BridgeDeviceIdentity.kt`, `BridgeServerCertificateStore.kt`, `BridgeClientConfigBuilder.kt`, `BridgeAutoCertificateEnrollment.kt` и действующие Dart consumers каналов. Их наличие видно только через имена вызовов, точный контракт неизвестен. Нельзя придумывать сигнатуры и обещать compile/functional PASS.
- **Следующая строго согласованная работа:** новая обвязка в **семи** целевых файлах с единственным владельцем lifecycle, сохранением точных внешних `prepare/stop/active/register` call-sites, безопасной диагностикой и сохранением Go ARM64 ELF. Вначале прочитать недостающие интерфейсы и статус незакоммиченных изменений, затем одной пакетной заменой с тестами. **Пока код приложения не изменялся**.
- **Статус:** `PRIVATE_CURRENT_ARCHIVE=VERIFIED`; `SEVEN_SHA_MATCH=PASS`; `CALLSITE_SOURCE_AUDIT=PASS_READ_ONLY`; `OLD_CHANNEL_DUPLICATION=CONFIRMED_SOURCE`; `LEGACY_TCP_ROUTES=CONFIRMED_SOURCE`; `APP_REWRITE=NOT_RUN`; `CURRENT_APP_COMPILE=NOT_RUN`; `NEW_DEVICE_E2E=NOT_RUN`. Время локального анализа неизвестно; timestamp фиксации устанавливается по metadata GitHub commit.

### Запись 013 — GitHub timestamps source audit

- Запись 012: `5924b007c5cce1f8840be48ece9155c1fa1b216a`, GitHub `created_at=2026-11-22 11:56:59 UTC` = **18:56:59 Новосибирск**.
- JOB-008: `68b2689ec7880229fdbb3b1b1c339c12357b944e`, GitHub `created_at=2026-11-22 11:57:14 UTC` = **18:57:14 Новосибирск**.
- Это время документальных коммитов, а не точное время исполнения локального Python. Приложение, Docker, сервер и бинарник не изменены.

### Запись 014 — Новое решение оператора: минимальный ремонт или выборочная замена вместо автоматического переписывания

- **Решение оператора:** «Если можно обойтись простым изменением кода этих файлов, то можно исправить их. Если нет, переписать полностью». Это разрешает **выборочную минимальную коррекцию** в пределах семи утверждённых файлов и полную замену только необходимого механизма, не реабилитирует старые запрещённые методы, не меняет сервер, Go ELF, APK, подпись, рабочие службы. Область M6 не расширять без нового решения.
- **Текущий исходный GitHub HEAD:** `a73ba6085cb71d08b99472b3bd8368921ae727c8` в `work/m6-android-ingress`.
- **Новый приватный архив от оператора:** `android-bridge-v102-dependencies.tar.gz`, SHA256 `ce9f0403b8e7605477d2088b8a1a40766ba0a55d8710f1a50c5a00da4bf3ff77`; 27 regular Kotlin files, прочитаны только локально, не выгружать в публичный GitHub. 10 файлов пересекаются с первым архивом, **SHA совпали 10/10**, расхождений нет.
- **Подтверждённые текущие API:**
  - `BridgeVpnSession.prepare(context,direct):String`, `stop(context)`, `active():Boolean`, `lastFailureCode()` — внешний контракт `VpnForegroundService` и состояния Flutter.
  - `BridgeControlChannel.register(messenger,context)` — entry point `MainActivity`.
  - `BridgeRuntimeControl.launchClient/stopClient/coreState/onUnexpectedExit` — существующий внутренний владелец процесса; старый `register` создаёт второй `MethodChannel com.tervonvpndpi/bridge_runtime` с прямыми `installClientConfig/start/stop` в обход `BridgeVpnSession`.
  - `BridgeCoreManager` читает process stdout/stderr, выделяет безопасный `M6_BRIDGE_SESSION_ERROR=` marker; `BridgeConfigManager` использует `nativeLibraryDir/libtervonbridge.so`, `-config`, app-private JSON; наличие подтверждённого Go ARM64 SHA сохранять.
  - `BridgeControlChannel` уже управляет enrollment через `BridgeAutoCertificateEnrollment`, `BridgeAutoEnrollmentHttp`, `BridgeDeviceIdentity` и хранилище сертификатов; эти классы **существуют** и не требуют переписывания ради замены маршрута.
- **Минимальная ремонтопригодность семи файлов (предварительная матрица решения, без изменения кода):**
  1. `BridgeVlessRoutePlan.kt` — **заменить старый алгоритм на уже протестированный чистый decision**, сохраняя необходимые сигнатуры call-sites по итоговому согласованию; не дублировать реализацию.
  2. `BridgeControlChannel.kt` — **точечно убрать регистрацию второго управляющего runtime канала**, сохранить действующие `bridge_control` commands и cert enrollment. Перед отключением `bridge_runtime` проверить фактических Dart consumers, которые НЕ входят в архив. Не удалять использующиеся API слепо.
  3. `BridgeRuntimeControl.kt` — **точечно убрать независимые install/start/stop извне**, сохранить внутренний процессный lifecycle и diagnostics. Точное удаление зависит от Dart consumers.
  4. `BridgeCoreManager.kt` — **сохранить** отдельный output drain/exit watcher/diagnostic ring; исправлять только установленный конкретный дефект по тесту. Полная замена не нужна.
  5. `BridgeConfigManager.kt` — **сохранить** защищённые app-private пути, ELF и атомарную запись config; исправлять по результату точного source/Gradle теста, полной замены пока не нужно.
  6. `BridgeXrayConfigRoute.kt` — **точечно корректировать** преобразование Xray JSON после проверки на реальные VLESS TCP, XHTTP, Trojan fixtures; не менять внутренние VPN параметры.
  7. `BridgeVpnSession.kt` — **сосредоточенная замена логики service selection**, удаление `params.tcpRoutes[selected.port]`, согласование реального server Discovery, корректная диагностика первых отказов, защита прямого fallback; сохранить внешний API `prepare/stop/active`. Именно этот файл потенциально потребует самой большой замены.
- **Существенная граница, нельзя решать предположением:** `BridgeAutoEnrollmentHttp.kt:33-41,108-139` (вне утверждённой семёрки) **парсит legacy `tcp_routes`**. Серверный `xui/enrollment_http.go` и `cmd/m6-auto-registrar/main.go` также сохраняют legacy `TCPRoutes`, хотя утверждённый новый контракт его исключил. Простая замена `params.tcpRoutes[selected.port]` на `params.service` даст **один серверный service для всех разных inbound** и может направить трафик не туда. Нельзя править восьмой Android файл или сервер без нового согласования области; прежде проверить **действующий** discovery и какой service он возвращает выбранному профилю.
- **Не получено:** текущий `git status --short` TervonVPN при `tracked_dirty=true`; действующие Dart consumers каналов, ответ живого Discovery; Go->server authenticated E2E trace. Архив не содержит Dart; не утверждать compile PASS по статическому Kotlin review.
- **Статус:** `SECOND_ARCHIVE_VERIFIED`, `OVERLAP_SHA=PASS_10_OF_10`, `MINIMAL_REPAIR_MATRIX=PROPOSED`, `CLIENT_SERVER_SERVICE_BINDING=BLOCKED_BY_UNVERIFIED_CONTRACT`, `APP_SOURCE_MODIFICATION=NOT_RUN`, `ANDROID_DEVICE_E2E=NOT_RUN`. Действующий сервер и приложение не изменялись.

### Запись 015 — Отменённый маршрутизатор в GitHub: stop-line до безопасного удаления

- **Явное поручение оператора:** очистить GitHub от мешающей старой реализации маршрутов, прежде всего `configuredTCPRoutes()` и старой выдачи `tcp_routes`; не затронуть действующие службы и утверждённые серверные механизмы.
- **Исходный HEAD:** `70894656ecf8496c078d4eff91f09bcb986f491b`, `work/m6-android-ingress`. Перечитаны Go исходники `xui/cmd/m6-auto-registrar/main.go`, `routes_test.go`, `xui/enrollment_http.go`, `enrollment_http_test.go`, `xui/route_sync.go`, `confirmed_routes.go`, актуальный план и журнал.
- **Граница доказанных PASS:** `M6_THREE_WAY_READ_ONLY=PASS` сверяет 3x-ui, live Xray и уже разрешённые Bridge routes; `SyncVerifiedRoutes` и `ConfirmedRouteStore` имеют Go unit/race/vet PASS на ранее зафиксированных commit; изолированный core v1.0.2 full-policy SIGHUP/ACK PASS, Docker cutover PASS. **Эти раздельные результаты НЕ подтверждают**, что `m6-auto-registrar` реально вызывает `SyncVerifiedRoutes` или содержит runtime applied-Xray callback: его текущий исходник **не вызывает** этих функций.
- **Прямые зависимости старого кода:** `main.go` вызывает `configuredTCPRoutes(root)`, получает `ports, services`, выбирает `services[0]`, создаёт `HotACLActivator` для сервера и передаёт `TCPRoutes:ports` в `BridgeEnrollmentHTTP`. `enrollment_http.go` принимает legacy `TCPRoutes map[string]string`, по умолчанию публикует их в GET Discovery; только при `ConfirmedRoutes != nil` заменяет snapshot. Старый `routes_test.go` тестирует `configuredTCPRoutes`, `enrollment_http_test.go` тестирует legacy и mixed mode.
- **Блокировка удаления как самостоятельного шага:** буквальное удаление `configuredTCPRoutes`/полей ломает сборку `m6-auto-registrar` либо делает неработоспособными CSR/ACL и Discovery. Полное безопасное удаление требует **согласованного подключения уже имеющегося SyncVerifiedRoutes/ConfirmedRouteStore к исполняемому регистратору** с действительным applied-Xray evidence и Docker policy ACK callback. В GitHub этот production wiring не найден, а пользователь запрещает менять работающие службы и самовольные механизмы.
- **Решение в соответствии с приоритетом утверждённого плана:** оставлены все рабочие исполняемые Go исходники неизменными; **не** вводился fake callback и **не** подменялась маршрутизация чтением `server.json` под другим именем; не обнулялся registrar ради видимости очистки. Статус `LEGACY_CLEANUP=BLOCKED_DEPENDENCY`, `CODE_DELETED=NO`, `SERVICES=UNCHANGED`, `GO_TEST_NOT_RUN_NO_SOURCE_CHANGE`. Требуется отдельное решение об изменении области для привязки existing control plane, либо доказательство текущего подключения в другом проверяемом исходнике. Изменения runtime не предпринимать.

### Запись 016 — Реальная очистка legacy `tcp_routes` (source-only; без cutover)

- **Согласование:** прямое указание пользователя «Удаляй их или исправь под актуальные данные». Исходная ветка `work/m6-android-ingress`, HEAD до изменения `da47e0c1043fdbcb9adb4127c2ab6018219fc8d7`.
- **Исходный commit:** `f5d70c587448449f5765627cd7433e5fdeacb53d` (проверить дату через GitHub metadata).
- **Изменены:** `xui/cmd/m6-auto-registrar/main.go`, `xui/enrollment_http.go`, `xui/enrollment_http_test.go`, `xui/confirmed_routes.go`. **Удалён:** `xui/cmd/m6-auto-registrar/routes_test.go`, тестировавший отменённую `configuredTCPRoutes`.
- **Что устранено:** `configuredTCPRoutes` полностью удалена; регистратор больше не выводит карту `backend-port → Bridge service` из `server.json`; поле `TCPRoutes` в HTTP handler и ответное `tcp_routes` удалены, fallback legacy removed. Вместо этого сервис HTTP требует `ConfirmedRouteStore` и проверяет ACK snapshot. При отсутствии подтверждения GET и POST отвечают `503`; для единственного `service` обязателен ровно один подтверждённый маршрут с совпадающим service — многосервисная неоднозначность закрывается (не выдаётся случайный backend).
- **Сохранено:** чтение runtime `server.json` только для проверки имён объявленных сервисов и их ненулевых backend, а также Docker-published TLS endpoint и сертификаты; `HotACLActivator` и CSR код не переписывались, Go ARM64, Linux v1.0.2 и Docker не менялись. Никакой автоматической маршрутизации или новой таблицы не добавлено.
- **Новая safety boundary:** `m6-auto-registrar` теперь создаёт пустой `ConfirmedRouteStore`, но не содержит полноценный API→Xray→policy-ACK coordinator. **Не применять новую сборку регистратора на сервер:** она fail-closed на discovery/enrollment (`503`) до подключения подтверждённого writer, а старый ACL-only activator ещё не переведён на full-policy digest. При нескольких сервисах single `service` response в текущем контракте неоднозначен и тоже fail-closed. Это удаление legacy-кода, **НЕ PRODUCTION CUTOVER**. Действующий живой регистратор не изменён.
- **Тесты:** `test-m6-xui.yml` запущен GitHub Actions автоматически на commit `f5d70c...`, run `45023624013` (`go test -race -count=1 ./...`), также runtime ACL workflow `45023623991`; при составлении записи оба ещё `in_progress`. `TEST_PASS=NOT_CONFIRMED` до фактического CI conclusion. Отрицательные тесты GET/POST без подтверждения и неоднозначных сервисов добавлены в `enrollment_http_test.go`.
- **Граница результата:** `LEGACY_ROUTE_READER=REMOVED_FROM_SOURCE`, `LEGACY_ROUTE_TEST=DELETED`, `LEGACY_DISCOVERY_FIELD=REMOVED_FROM_SOURCE`; `LIVE_SERVICE=UNCHANGED`, `CURRENT_REGISTRAR_REPLACEMENT=NOT_READY`, `CURRENT_ANDROID_DISCOVERY_COMPATIBILITY=NOT_TESTED`. Исторические записи о старой схеме остаются историей, а не текущей инструкцией.

### Запись 017 — Итог очистки legacy source и подтверждённые CI

- **Согласовано и выполнено в source:** удаление `configuredTCPRoutes`/legacy static inbound map и `bridgeDiscoveryResponse.tcp_routes`; новый `confirmed`-only guard не позволяет выдавать неподтверждённую/неоднозначную service selection. Удалён deprecated `cmd/m6-auto-registrar/routes_test.go`; новый `services_test.go` проверяет только непрозрачные имена сервиса, без вычисления inbound/backend портов. `BridgeCoreManager`, Go ARM64, Docker/Linux Release/живые службы без изменений.
- **Commit chain с подтверждённым GitHub временем (Новосибирск +07):** исходная правка `f5d70c587448449f5765627cd7433e5fdeacb53d` (19:41:50) → первый CI FAIL из-за повреждённого `main.go` (`syntax error: unexpected name json`); устранено восстановлением чистого исходника с защищённым splice `11518161d61c53401e3e2144336d975fce515820` (19:43:50); новые service tests `1fa836c724a07717c81253c2d0479cd985312dad` (19:44:19); актуализированы действующие docs/план/контракт `1f9245cc3f374587e027fbb6b2bfa18a9b16b32a` (19:46:23).
- **Комплексный Go CI на точном code commit `1fa836c...`:** GitHub Actions `M6 3X-UI adapter tests` run `45023832666`, `go test -race -count=1 ./...`, **conclusion=success/PASS** (6 Go packages, включая регистратор). Отдельный workflow `M6 runtime ACL source check (released v1.0.1)` run `45023832706` **conclusion=success/PASS**. Ранний compile FAIL run `45023624013` не скрывать; он исправлен в более позднем commit. PASS не относится к Android E2E.
- **Честная эксплуатационная граница:** регистратор собирается и тестируется, но исходная программа пока создаёт пустой `ConfirmedRouteStore`, **не подключая real API→Xray→full policy ACK producer**. GET Discovery / POST CSR на новом бинарнике после запуска вернут 503, пока writer не подаст подтверждённое поколение; для многосервисного состояния `service` также fail-closed вместо произвольного выбора backend. Никакого server cutover/установки/перезапуска не было; старый действующий регистратор и Android клиент могут продолжать работать по старому контракту. **`PRODUCTION_REGISTRAR_READY=NO`**, **`LEGACY_SOURCE_CLEANED=YES`**, **`FULL_END_TO_END=NOT_TESTED`**.
- **Исторические записи в Git истории не удалялись:** удалена текущая устаревшая схема из активных `docs/DEVELOPMENT_PLAN.md`, `integrations/android-ingress/README.ru.md`, `ENROLLMENT_CONTRACT.ru.md`; фактическая история прошлых проверок сохранена.

### Запись 018 — обязательная формула архитектуры Android M6

- **Решение оператора, 2026-11-22:** дословно утверждено «Android TervonVPN → существующий Go Encapsulator → HTTPS/2 + TLS 1.3 + mTLS → серверный Decapsulator → Xray.»; выполнять как единственный архитектурный путь Android M6. Формула внесена в действующий `docs/DEVELOPMENT_PLAN.md`.
- **Ветка/исходный HEAD:** `work/m6-android-ingress` / `b1f7ff82883333d17f42acaa29c4396c3209697d`. Только обновление документации; код, тесты, Android-приложение, Linux Bridge, серверные службы и Docker не изменялись.
- **Смысл:** Android = сертификат + Go Encapsulator + прозрачный TCP поверх существующего HTTPS/2/TLS 1.3/mTLS. Server Decapsulator = решение backend-маршрутов и подключение к Xray. Запрещён перенос серверного control plane и `ConfirmedRouteStore` в Android.
- **Статус:** `ARCHITECTURE=APPROVED_DOCUMENTED`; `CODE_MODIFIED=NO`; `TEST_NOT_RUN_DOCUMENTATION_ONLY`; `ANDROID_E2E=NOT_PASS`. При разработке сперва сверяться с формулой, а не заменять утверждённую архитектуру проектными инициативами.

### Запись 019 — M6: минимальный двухфайловый patch-кандидат Android (не установлен)

- **Утверждённая архитектура:** `Android TervonVPN → существующий Go Encapsulator → HTTPS/2 + TLS 1.3 + mTLS → серверный Decapsulator → Xray`. Указание пользователя: продолжать внутри текущего M6. Исходный публичный GitHub HEAD `04f9ac6ef8d9f33cbac324740640c07c23257a1d`, branch `work/m6-android-ingress`; Android private app branch `feature/status-ping-cleanup`, HEAD `22ff78a9dbabc1544a53b9ac950f78413f25b4fb` (сообщён read-only Homeserver output).
- **Обязательный preflight:** пользовательский вывод `READ_ONLY_AUDIT=COMPLETE`; SHA256 семи целевых файлов совпали с предоставленным архивом; рабочее дерево Android существенно dirty, не делать reset/clean. Dart использует оба MethodChannel: `bridge_control` и `bridge_runtime`; действующие call-sites `VpnForegroundService.prepare/stop`, `MainActivity.register` подтвердились. Для приватного Android нет write-подключения.
- **Исходная ошибка 1:** `BridgeVpnSession.kt` требует отменённую `params.tcpRoutes[selected.port]`, тогда как существующий клиентский Go config принимает непрозрачный `service` от аутентифицированного Discovery. Согласованная минимальная замена на `params.service` с валидацией допустимого имени: **без Android backend-map**. Важная граница: если один Discovery service не соответствует выбранному профилю, это не доказывает E2E и требует доказанного service contract; нельзя угадывать mapping.
- **Исходная ошибка 2:** Dart `bridge_runtime_screen.dart` действительно вызывает `installClientConfig`/`start`/`stop`, а `BridgeVpnSession` использует тот же process owner. Патч **сохраняет Channel/method names/status** и внутренние `launchClient/stopClient` для VPN, но manual launch/config/stop возвращают явное `BRIDGE_OWNED_BY_VPN` и не вмешиваются в действующее соединение. `vpnRouteIntegrated` отображает `BridgeVpnSession.active()`.
- **Область:** только `BridgeVpnSession.kt` и `BridgeRuntimeControl.kt`. SHA-guard preflight: `5bcf45125cdad9b9220ef7d983e255f2d6a92342b65fc2e2ceccdabf29681838` и `a6f9695d4cdc6a3d120ac0f685ddd6f2d9ed4a122469e13135e542dbfacdc596`. Исходные семь файлов, Go ARM64, сертификаты, сервер, Docker, живые службы и Dart исходники **не изменены**.
- **Artifact-кандидат:** локальный приватный `m6_bridge_android_fix.py` и `test_m6_bridge_android_fix.py` передаются пользователю как вложения беседы; **не размещать код приватного TervonVPN в публичном GitHub**. Скрипт без `--apply` read-only, `--apply` проходит обе SHA-защиты до записи, использует атомарную замену и rollback при ошибке.
- **Фактический локальный результат:** `python3 -m py_compile m6_bridge_android_fix.py` PASS; dry-run на предоставленных семи исходниках PASS; `python3 -m unittest -v test_m6_bridge_android_fix.py`: **4/4 PASS** (read-only, двухфайловая запись, отказ при drift без partial write, повторный запуск отвергнут). `ANDROID_GRADLE=NOT_RUN`, `ANDROID_APK=NOT_BUILT`, `DEVICE_TCP_E2E=NOT_RUN`, `HOMESERVER_APP_MODIFIED=NO`.
- **Важно:** тесты патчера доказывают только SHA-guard/преобразование; не подменяют Android compile и функциональные проверки. Многосервисный binding не считается доказанным на устройстве. Никаких других Kotlin файлов или серверной инфраструктуры менять не разрешено.

### Запись 020 — подтверждённая Kotlin-компиляция на Homeserver (отчёт оператора)

- **Основание:** оператор передал результат одного read-only SHA контроля и Kotlin-компиляции на Homeserver, 2026-11-22. `COMMAND_EXIT_CODE=0`, `SSH_CONSOLE=OPEN`. Отдельная проверка точного времени запуска отсутствует.
- **Приложение:** `/opt/tervonvpndpi`; ветка `feature/status-ping-cleanup`, последний ранее проверенный Git HEAD `22ff78a9dbabc1544a53b9ac950f78413f25b4fb` (новый Git HEAD после сборки не запрашивался). Исходники Kotlin приватные, не загружались в публичный репозиторий.
- **Проверено на Homeserver:** SHA256 `BridgeVpnSession.kt=6b8cfc1fc4f0cdb912be5485f21ba2a9364796dcaf6793bcdf785aa4b3032004`, `BridgeRuntimeControl.kt=79b3d3a1cc4ba1725523f40c0fc32458efc420b9072f0add98f02f46e9cb375f` (обе проверки `sha256sum -c` успешно прошли согласно операторскому `M6_PATCHED_SOURCES=PASS`). `bash ./gradlew :app:compileReleaseKotlin --offline --no-daemon` завершено `BUILD SUCCESSFUL in 19s`, `72 actionable tasks: 5 executed, 67 up-to-date`; `KOTLIN_COMPILE=PASS`.
- **Оговорка о нанесении:** предыдущий запуск патчера `--app-root` без записи сообщил `M6_PATCH=BLOCKED` из-за SHA текущего `BridgeVpnSession.kt`, который уже соответствовал целевому SHA. В данной сессии нет доказательства, когда или каким действием два исправления появились на Homeserver; есть только подтверждённое **текущее содержимое** обоих файлов. Не приписывать их создание последнему запуску патчера.
- **Не проверено:** реальный `service` выбранного профиля, запуск Go Encapsulator из VpnService, сохранение байтов, связь с Decapsulator, APK/device E2E, доступность интернета, самостоятельный JVM unit-test. `ANDROID_E2E=NOT_TESTED`, `M6_RELEASE=NOT_READY`.
- **Службы:** оператор подтвердил `SERVER_SERVICES=UNTOUCHED`; этот этап их не изменял.
- **Текущий документационный HEAD до записи:** `6c451261b4f53592b98dd13604a07ae54aa1036a`. Статус `M6_TWO_FILE_SOURCE_HASH=PASS`, `KOTLIN_COMPILE=PASS`, `DEVICE_TCP_E2E=NOT_RUN`.

### Запись 021 — вызов отсутствующей Android unit-test задачи (не дефект компиляции)

- **Отчёт оператора Homeserver:** `bash ./gradlew :app:testReleaseUnitTest --offline --no-daemon` завершился с `UNIT_TEST_RC=1`, `COMMAND_EXIT_CODE=1`, `BUILD FAILED in 14s`. Gradle указал точную причину: `task 'testReleaseUnitTest' not found in project ':app'`.
- **Диагноз:** ранее ассистент выбрал имя Gradle-задачи без чтения текущего списка задач проекта. Никакие unit tests не исполнялись. Ранее подтверждённая Kotlin-компиляция JOB-014 остаётся PASS; по этому логу нет причины менять Gradle/Kotlin версии или рабочий APK.
- **Граница:** `APK_BUILD=NOT_RUN`, `ANDROID_E2E=NOT_TESTED`, `SERVER_SERVICES=UNTOUCHED`. Следующее действие — только чтение списка фактических задач `:app:tasks --all` и наличия unit-test исходников. Никаких изменений кода/серверных служб.
- **Documentation parent HEAD:** `d06760d20b2e2580bb17657f9210544dcfe9e86b`; дата пользовательской проверки 2026-11-22, точное время на Homeserver не предоставлено.

### Запись 022 — инвентаризация реальных Android Gradle test tasks

- **Оператор, 2026-11-22:** read-only `:app:tasks --all --offline --no-daemon --console=plain` + проверка исходников тестов + настроек Gradle завершились `COMMAND_EXIT_CODE=0`, `READ_ONLY_INSPECTION=COMPLETE`, Gradle `BUILD SUCCESSFUL in 15s`.
- **Фактически есть:** `testDebugUnitTest`, `test`, `assembleUnitTest`, `check`, `connectedDebugAndroidTest`, `connectedAndroidTest`. `testReleaseUnitTest` отсутствует; это причина предыдущего JOB-015 FAIL. **Новых запусков unit-тестов нет.**
- **Тестовых источников нет:** `app/src/test` отсутствует, `app/src/androidTest` отсутствует. Само наличие Gradle задачи не доказывает покрытие Bridge. Нет оснований выдавать `testDebugUnitTest` за функциональный тест передачи TCP.
- **Настройки:** `com.android.application`, `org.jetbrains.kotlin.android`, `android.builtInKotlin=false`; предупреждения о будущей миграции Kotlin/AGP не требуют изменения текущей сборки в M6.
- **Конкретный оставшийся риск:** Kotlin `BridgeVpnSession.kt` в двухфайловом кандидате получает `service` из доверенного Discovery; это ещё не доказывает точное соответствие сервиса backend выбранного TCP-профиля. Не компенсировать неизвестность новой Android таблицей или изменением серверного Bridge.
- **HEAD документации до этой записи:** `70308154f1d9ea9f2da5a4547a18ac645e2fc565`. `GRADLE_TASK_DISCOVERY=PASS`, `ANDROID_UNIT_TEST_COVERAGE=ABSENT`, `KOTLIN_COMPILE=PREVIOUS_PASS`, `APK_BUILD=NOT_RUN`, `DEVICE_E2E=NOT_RUN`, `SERVER_SERVICES=UNTOUCHED`.

### Запись 023 — исправление регрессии M6 Enrollment без изменения рабочего сервера

- **Прямое указание пользователя:** «Убирай свои художества! Делай как согласовано!» — убрать самовольную зависимость регистрации Android от серверной маршрутизации; не вводить новые механизмы.
- **Ветка/parent HEAD:** `work/m6-android-ingress` / `534272e307bde2fb46fa4f75438e672141a6ba77`. Область исходников: `xui/enrollment_http.go`, `xui/cmd/m6-auto-registrar/main.go`, `xui/enrollment_http_test.go`; документация.
- **Убрано:** обязательный `ConfirmedRouteStore`, чтение `Snapshot()`/проверки поколения на GET Discovery и POST CSR; пустое `new(ConfirmedRouteStore)` в исполняемом регистраторе; публикация `policy_generation` из Discovery. `tcp_routes` не возвращался.
- **Сохранено:** существующие HTTPS endpoint и TLS trust, получение сертификата по PKCS#10 CSR, проверка/ограничение размера CSR, активация клиентского отпечатка через существующий ACL callback до ответа 200, существующие проверочные отказные сценарии. `ConfirmedRouteStore` и серверный route-sync как самостоятельные серверные компоненты не удалены и не изменены.
- **Новая проверка:** в unit-тестах GET Discovery и POST CSR проверяются без серверного route coordinator, отсутствие `tcp_routes` и `policy_generation` и успешный callback ACL. В CI проверять текущую ревизию; прежние PASS не означают готовность новой.
- **Граница:** GitHub source only; действующий сервер, Docker, Xray, Bridge, Android-приложение и сертификаты не трогались. `ENROLLMENT_SOURCE_FIX=IMPLEMENTED`, `CI=AWAITING_RESULT`, `SERVER_DEPLOYMENT=NOT_RUN`, `DEVICE_E2E=NOT_RUN`.

### Запись 024 — подтверждённые CI после устранения связки CSR→route-generation

- **Точный кодовый commit:** `7c2f3d4cb913db9580bb502060fbd87d833bb420` (`work/m6-android-ingress`).
- **GitHub Actions:** `M6 3X-UI adapter tests` run `45034267860` — `completed/success` (PASS), включая Go unit/race; `M6 runtime ACL source check (released v1.0.1)` run `45034267873` — `completed/success` (PASS), включая проверку регистратора и source overlay на Linux release.
- **Результат:** ошибочная обязательная зависимость от `ConfirmedRouteStore` удалена; автоматическая выдача сертификата и вызов существующей ACL-активации сохранены. `tcp_routes` не возвращено.
- **Граница:** CI подтверждает исходный код и тестовые проверки на точном commit, **не** реальную выдачу сертификата устройству, не Android TCP E2E и не серверное развёртывание. Ни одна рабочая служба не изменена. `SOURCE_CI=PASS`, `LIVE_ENROLLMENT=NOT_TESTED`, `ANDROID_E2E=NOT_TESTED`.


### 2026-11-23 M6 — private Android current-source integration of dynamic profile selector

- **Input:** operator-provided \`m6-current-route-source.tar.gz\`, SHA256 \`f59191265fb8bced8d24e549aae66a6568feaf46819687e8b86c9e08d9eb1cb09fa9ff5\`, ten current app-private Kotlin files.
- **Changes prepared locally, not published to public GitHub:** \`BridgeAutoEnrollmentHttp.kt\` no longer requires route \`service\` or deprecated \`tcp_routes\` in certificate Discovery; \`BridgeVpnSession.kt\` passes the *selected Xray TCP outbound endpoint port* as canonical decimal \`service\` to the unchanged \`BridgeClientConfigBuilder\`, and then the existing Go Encapsulator sends the same \`Tervon-Service\` header. No protocol-specific port values or new APIs in Android. Existing CSR, mTLS, local listener and VPN lifecycle retained.
- **Source check:** \`M6_ANDROID_DISCOVERY_SOURCE_CONTRACT=PASS\` (no outdated fields, active Xray endpoint), Kotlin pure route tests \`M6_KOTLIN_ACTIVE_PROFILE_ROUTE=PASS\` (XHTTP 55535, VLESS TCP 30000, Trojan TCP 55636, QUIC/invalid port and OFF); isolated Kotlin compile of Discovery with stubs \`M6_DISCOVERY_KOTLIN_COMPILE=PASS\`. Full Android/Flutter Gradle source compile, installed APK and device Internet/E2E **NOT_RUN**.
- **Matching server code:** GitHub Bridge \`work/m6-android-ingress\` @ \`ee113cf1328ee32da1ff28493a67e919c7314129\`, source CI run \`45134204649\` **PASS**.
- **Deployment dependencies:** Existing live server binary is not updated. Actual core policy and ACL must have all three active inbounds (prior server.json snapshot did not), and automatic 3x-ui→Xray→policy application for newly added/changed inbounds must be verified on the running host. Until matching server-side support is deployed, installing Android source would be an incompatible cutover. The private Kotlin patch is delivered to the operator in an archive and not committed to public Tervon TCP Bridge.
