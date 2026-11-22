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
