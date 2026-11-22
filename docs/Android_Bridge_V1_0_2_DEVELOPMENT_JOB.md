# Android Bridge v1.0.2 — журнал работ, испытаний и результатов

**Начало новой Android-разработки по решению пользователя:** 22.11.2026 18:00, Новосибирск (UTC+07:00).
**Часовой пояс:** Novosibirsk UTC+07:00; для фактических проверок и коммитов записывать одновременно timestamp UTC и UTC+07, не выдавать точку отсчёта за время выполнения.
**Проект:** Tervon TCP Bridge / Android M6.
**Рабочая ветка при старте:** `work/m6-android-ingress`.
**Головной журнал:** `docs/Android_Bridge_V1_0_2_DEVELOPMENT_LOG.md`.

## Формат каждой проверки / job

| Поле | Обязательное содержание |
|---|---|
| Job ID | Уникальный последовательный идентификатор |
| Дата и время | GitHub commit time / timestamp теста UTC и Новосибирск (+07:00), источник |
| Git | SSH origin, ветка, exact source HEAD, dirty status |
| Объект и вопрос | Файлы/модуль, какой ровно риск/требование проверяется |
| Среда | Инструменты, версии, изоляция, устройство, процесс |
| Команды | Точные команды или способ воспроизведения без секретов |
| Ожидаемый результат | Однозначный критерий готовности до выполнения |
| Фактический результат | Код выхода, вывод, измерения, ссылка на evidence |
| Статус | NOT_RUN / PASS / FAIL / BLOCKED; отдельно source, compile, integration, device E2E, release |
| Ограничения | Что тест не доказывает, фактическое влияние на службы |
| Исправления / follow-up | Связанные изменения, новый commit, требуется ли повтор |

**Правила:** не переносить PASS старых Android-файлов/тестов на новое начало; не помечать работу завершённой по компиляции; не запускать повторные проверки без причины; выполнять комплексные пакеты в рамках M6. Не публиковать токены, сертификатные ключи, пользовательские данные, реальные адреса в логах.

## Реестр испытаний

### JOB-000 — исходное состояние на 22.11.2026 18:00 Новосибирск

- **Назначение:** зафиксировать нулевую точку разработки новой Android-обвязки.
- **Состояние:** `NOT_RUN` — новая реализация пока не существует; тесты новой реализации не выполнялись.
- **Операторский факт:** старая Android-обвязка признана неработоспособной; её старые PASS и прежние SHA являются только архивными свидетельствами.
- **Граница:** Linux/серверная часть признана готовой оператором, в область новых испытаний не включается без отдельного решения.
- **Первая будущая проверка:** read-only инвентаризация точных файлов Android для согласованной очистки; при завершении написать отдельную JOB-001 с фактическим временем и результатом.

### JOB-001 — Проверка создания двух журналов GitHub

- **Время подтверждённых операций:** первый файл 2026-11-22 11:05:08 UTC / 18:05:08 Новосибирск; второй 11:05:22 UTC / 18:05:22 Новосибирск. Источник — GitHub `fetch_commit.created_at`.
- **Git:** `work/m6-android-ingress`, до создания `1b8097d9bcc0a596c413976f793420b5f186b709`; два создания `f1d69c119bc5dba8af2efd35509b535217310790`, `65f722184926f90f25ec85dec991dcb86737d652`; первая актуализация LOG `f88da85d7010734d5e090684cd2a1736dfa9b125`.
- **Вопрос:** созданы ли именно оба заказанных документа на правильной ветке?
- **Метод:** GitHub contents `fetch_file` (до создания оба `NOT_FOUND`, после создания найдены), `fetch_commit` для обеих операций.
- **Ожидание:** оба файла доступны, в каждом корректно указана точка отсчёта 22.11.2026 18:00 UTC+7.
- **Результат:** `PASS` только для создания документов и commit-time provenance. **Android compile/E2E: NOT_RUN.**
- **Влияние на службы/приложение:** `NONE`.

### JOB-002 — Граница и атомарная очистка устаревших Android Stage4 GitHub файлов

- **Начальное состояние Git:** `aa5d3f1233cb71784dadbd7c4a038141a7f33a72`, рабочая ветка `work/m6-android-ingress`, base tree `c696b561bb4e6615a59abf2cd8871430c1564791`.
- **Вопрос:** полностью ли изолирована удаляемая старая Android-инсталляция от Linux/серверного кода?
- **Среда:** GitHub REST Git Tree API, read-only `git/commits`, `git/trees?recursive=1`, затем create_blob/create_tree/create_commit/update_ref.
- **Критерий preflight:** ровно 71 файл с префиксом `integrations/android-ingress/stage4/`, `truncated=false`; ни один path за этим префиксом не удалять. `HEAD` должен оставаться `aa5d3f1233cb71784dadbd7c4a038141a7f33a72` до ref update.
- **Preflight факт:** 226 blob-файлов в дереве; в удаляемом Stage4 71; `truncated=false`; `PREFLIGHT=PASS`.
- **Операция:** одно дерево Git с 71 `sha:null` и изменениями двух журналов; один commit и fast-forward `update_ref` с `expected_sha`.
- **Проверка после коммита:** независимо заново получить HEAD/tree, удостовериться, что Stage4 файлов 0, серверные source paths сохранены и оба журнала доступны. Пока не выполнено, `POST_VERIFY=PENDING`.
- **Новые Android compile/E2E:** `NOT_RUN`; это **удаление устаревшего source из Git**, не восстановление мобильного приложения. Службы не изменяются.
- **Время:** точные временные метки commit будут считаны из GitHub `created_at` после публикации; не заменять их точкой отсчёта 18:00.

### JOB-002 — Итог post-commit verification

- **Точное время коммита:** 2026-11-22 11:08:28 UTC / **18:08:28 Новосибирск**, источник GitHub `fetch_commit.created_at`.
- **Проверяемый commit:** `b4be0648674d86b41fe3a26706c3743d35f40505`.
- **Метод:** отдельный GitHub `compare_commits(branch,branch)` для HEAD; Git Trees `?recursive=1` для файлов; `fetch_commit` для времени и состава изменений.
- **Ожидаемое:** HEAD совпадает, Stage4=0, `truncated=false`, серверные компоненты и оба новых журнала сохраняются.
- **Фактически:** `HEAD_MATCH=PASS`, `STAGE4_FILES_REMAINING=0`, `TREE_COMPLETE=true`, `PROTECTED_PATHS_PRESENT=8/8`, `BLOBS_AFTER=155`.
- **Итог:** `JOB-002=PASS_GITHUB_CLEANUP`. Android Kotlin/Flutter build, ARM64 execution, VPN E2E = `NOT_RUN`. Операции на Homeserver и работающих службах = `NONE`.

### JOB-003 — Read-only release/source/Docker/Android contract research (22.11.2026)

- **Baseline:** `6beb9eae9d9a3ef3110b5e3cbcb9e13d14d242f6`, `work/m6-android-ingress`; выполнение исследовательского чтения GitHub; отдельный wall-clock старта не получен (`TIME_NOT_VERIFIED`). Время запишем по финальному docs commit через GitHub metadata.
- **Вопросы:** опубликован ли Linux v1.0.2; какая точная сборка и функции; какие Docker image/порт и подтверждённые результаты; как Android обязан работать с сервером.
- **Метод и evidence:** GitHub API `fetch_file` по Git refs, `fetch` `/releases/tags/v1.0.2`; проверка release base source `96c1367` отдельно от текущего prototype; старый `docs/V1_0_2_DEVELOPMENT_LOG.md` с последовательными operator results.
- **Ожидаемое:** согласованность release sha, image digest, Compose effective, HTTP2/mTLS/framing route & ACL source; отдельная оговорка отсутствия нового device E2E.
- **Фактически:** Release опубликован `07:53:17Z`, Linux asset SHA `aea59080...` совпадает с Docker-журналом; `runtime_policy.go` подтверждает одну atomic generation; release v1.0.1 source подтверждает `POST /v2/stream`, H2 TLS 1.3 + mTLS, service header, wire framing, padding/jitter. Docker operator `COMPOSE_PERSISTENCE=PASS` с host 18443 -> container 8443. Legacy `TCPRoutes` и `configuredTCPRoutes` всё ещё находятся в GitHub source; пользователь запрещает их возвращение в Android.
- **Статус:** `PASS_READ_ONLY_SOURCE_EVIDENCE`; `PASS_RECORDED_SERVER_DEPLOYMENT`; `BLOCKED_CURRENT_SERVICE_BINDING_INTERFACE` пока не подтверждён точный live registrar discovery; `ANDROID_COMPILE=NOT_RUN`; `ANDROID_E2E=NOT_RUN`. Никакие службы или конфигурации не менялись.

### JOB-003 — Уточнение времени записи свидетельств
- `LOG_COMMIT=0ff4a09647029dbfdd9f7a14650d77f2621a5d75`, GitHub `created_at=2026-11-22T11:15:48Z` (Новосибирск 18:15:48).
- `JOB_COMMIT=d3981c3e4ff7fee335fbdcdd727c237b72a24e02`, GitHub `created_at=2026-11-22T11:15:51Z` (Новосибирск 18:15:51).
- Изменены только журналы; Android tests `NOT_RUN`, services `UNCHANGED`.

### JOB-004 — Historical ARM64 core, diagnostic integration and APK device E2E evidence (read-only recovery)

- **Source baseline:** `a29f0d45a83b012e34fc96ac05b3332e3380234e` before record; `work/m6-android-ingress`. **Input:** directly reread `docs/V1_0_2_DEVELOPMENT_LOG.md` lines 949–961 and 1031–1061. Новых команд сборки/установки/ADB **не выполнялось**.
- **Вопрос:** действительно ли v1.0.2 Go ARM64 был собран, снабжён диагностикой, установлен в проект и использован в APK? Что доказал Android тест?
- **Сборка 1 (исторический operator PASS):** Go 1.25.5, `GOOS=android GOARCH=arm64 CGO_ENABLED=0`, AArch64 PIE, SHA `d71607005c510ec1d97787f656b4b5580d10d750ec1031b9b18630af266ac487`.
- **Сборка 2 с диагностикой (operator PASS):** guarded `apply_go.py --check/--apply` на isolated v1.0.2 source; 6 packages `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, Android arm64 `go build -trimpath ./cmd/bridge`, `M6_GO_BATCH_RC=0`. Итоговый `libtervonbridge.so` SHA `a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd`.
- **Встройка (operator PASS):** обновлён `/opt/tervonvpndpi/android/app/src/main/jniLibs/arm64-v8a/libtervonbridge.so` атомарно с SHA-проверкой; до этого сохранён старый ELF. `M6_INSTALL_RC=0`. Kotlin diagnostics `ALREADY_APPLIED`, `./gradlew --offline :app:compileDebugKotlin` success, `KOTLIN_RC=0`; Flutter test `NOT_RUN` (SDK CLI отсутствовал, RC127).
- **Реальный телефон (operator FAIL):** APK установлен; ADB logcat 15:41–15:43: XHTTP/splithttp GET/POST `EOF`, loopback TCP `connection reset by peer`, `TervonBridge: code=UNKNOWN generation=1`; интернет не работает. Точная стадия корневого отказа НЕ УСТАНОВЛЕНА. Предположение о виновности старого Android wrapper не является измеренной root cause.
- **Отличие прошлого от новой разработки:** `JOB-004=PASS_HISTORICAL_EVIDENCE_RETRIEVAL`; `ANDROID_GO_DIAGNOSTICS_INSTALLED=PASS_OPERATOR_RECORD`; `ANDROID_APK_TEST=FAIL_OPERATOR_RECORD`; `NEW_ANDROID_IMPLEMENTATION_TESTS=NOT_RUN`. Никакие службы, исходники, APK или бинарники не менялись этим исследованием.
- **Время:** время исторического теста телефона известно только из фрагмента ADB 15:41–15:43, **часовой пояс этой записи не установлен**; не интерпретировать его как время Новосибирска или точный Git commit. Время нового документационного коммита установить GitHub metadata.

### JOB-005 — Новый Route Policy, точный код GitHub, Kotlin JVM unit test

- **Ветка/исходный HEAD:** `work/m6-android-ingress` / `49f8a8f1af45e35df050dcfc21e11d0c51313ab6`; исходный код commit `29375777562f8e54f1d78f9dfbacb0d8e6c1bcb3`.
- **Вопрос:** корректно ли pure Kotlin принимает только TCP-capable Xray transports и не разрешает bridge route без действительного исходного endpoint, проверенного discovery и локально готового listener? Отсутствует ли старый `tcp_routes`?
- **Среда:** изолированный Kotlin JVM `kotlinc-jvm 1.9.0 (JRE 21.0.12.1)` без Android SDK/Flutter/Go/сети.
- **Команда:** `kotlinc BridgeVlessRoutePlan.kt BridgeVlessRoutePlanTest.kt -include-runtime -d route-test.jar && java -jar route-test.jar` (исполнено на локальных файлах побайтно совпадающих с GitHub Blob).
- **Ожидаемое:** TCP-capable transport, original endpoint, verified discovery, valid local port and listener readiness → local route to 127.0.0.1; unsupported transports and invalid states → typed `Rejected`; disabled → `Direct`.
- **Фактически:** `ANDROID_BRIDGE_V102_ROUTE_UNIT=PASS`, exit 0; positive 12 network inputs; UDP/quic/h3/kcp/unknown/whitespace отрицательные; invalid remote host/port, unverified discovery, invalid/not-ready listener rejected; disabled passes directly; original endpoint preserved.
- **Контроль точного GitHub source:** `git hash-object BridgeVlessRoutePlan.kt=0dee3044f466a400a0fab743bbad8c3b4b4e1b0c` совпал с GitHub create_blob; `git hash-object BridgeVlessRoutePlanTest.kt=00bef36ab5b02db14f26afd62eea068d1ade37a5` совпал. SHA256 src `d62fea47cb9ff8852499c49985f212823ce234b3c385083a92acc68d3b3fb8c8`, test `394a7419ba329927aa35492fe8c1a4afb5a84266b54c230ec954add0366e6dea`.
- **Статус:** `PASS_ISOLATED_EXACT_BLOB`; `ANDROID_APP_GRADLE=NOT_RUN`; `ACTUAL_VPN_LOOP=NOT_TESTED`; `TLS_H2_MTLS=NOT_TESTED`; `DEVICE_E2E=NOT_RUN`. Код не изменял приложение или сервер. Точное время выполнения локального `kotlinc` отдельно не замерялось — `EXECUTION_TIME_NOT_VERIFIED`, timestamp GitHub исходного/docs commit запишем отдельным пунктом.

### JOB-005 — Время проверенных коммитов

Источник — GitHub `fetch_commit.created_at`, UTC и Новосибирск UTC+7:
- Код + точные unit-файлы: `29375777562f8e54f1d78f9dfbacb0d8e6c1bcb3`, **2026-11-22 11:42:05Z / 18:42:05 (+07)**.
- DEVELOPMENT_LOG запись 008: `49eddf43aa14858fc1d10fed7c09d31a995ed7a1`, **11:42:31Z / 18:42:31 (+07)**.
- DEVELOPMENT_JOB запись JOB-005: `6f149f1c40d4621a13e4dfe7fc1adc5e2d01da02`, **11:42:44Z / 18:42:44 (+07)**.
- Отдельная локальная метка начала/конца unit теста не собиралась, поэтому `TEST_EXECUTION_TIME=NOT_MEASURED`; времена Git commit не выдавать за точное время самого теста.

### JOB-006 — Повтор exact-route и read-only preflight актуального Android-приложения

- **Исходный HEAD:** `d2c46e380eaa465db7fc369a058005c401c2cae9`; source commit `012faf18a5618071fcf5fb4c1345cad39dde1d00`; ветка `work/m6-android-ingress`.
- **Цель:** (1) повторить Kotlin route test на exact GitHub-compatible files, (2) проверить read-only preflight прежде чем позволять ему просматривать проект `tervonvpndpi`. Инструмент собирает только относительные пути, номера строк, SHA и Git HEAD, не печатает code/config.
- **Команды и среда:** `kotlinc BridgeVlessRoutePlan.kt BridgeVlessRoutePlanTest.kt -include-runtime -d route-test.jar && java -jar route-test.jar` (Kotlin 1.9.0, Java 21) → `ANDROID_BRIDGE_V102_ROUTE_UNIT=PASS`. `python3 -m unittest -v test_current_app_preflight.py && python3 -m py_compile current_app_preflight.py test_current_app_preflight.py` → **3 tests PASS, py_compile PASS**.
- **Negative tests:** nonexistent Android root raises `NOT_ANDROID_PROJECT_ROOT`; symlink source entry rejected; fixture containing `password = DO_NOT_PRINT` and `secret abc123` does not emit strings; comparison of all fixture files before/after shows identical bytes. Synthetic fake ARM64 core hash correctly marked mismatch.
- **Исходный Git SHA exact:** preflight `e8ecf08df3e7dd561ce0e4af79e83e729b3e9185`; tests `fb7d1da713fd2e609b68fa64bf9376f06d85102f`, совпали с `git hash-object` локальных файлов. SHA256 preflight `82adb424e068b6d74e40848a16d53df8df5ee59e7a03978ce2b060356438ee97`, test `4c000b8316a9730818e0ed93f0384bb5f4d4593d7f0bfaaf66b0fe3167c4f050`.
- **Дата/время коммита:** GitHub `fetch_commit.created_at=2026-11-22T11:46:44Z` = **18:46:44 Новосибирск**. Время локального выполнения теста отдельно не измерялось; `TEST_EXECUTION_TIME=NOT_MEASURED` и не приравнивается к commit time.
- **Результат:** `ROUTE_REPEAT=PASS`; `PREFLIGHT_SYNTHETIC=PASS`; `ACTUAL_HOMESERVER_SCAN=NOT_RUN`; `APP_INTEGRATION=NOT_RUN`; `GO_ELF_HOMESERVER_SHA=NOT_RECHECKED`; `APK_E2E=NOT_RUN`. Живые службы и приложение не изменялись. Следующее — фактическое read-only исследование `/opt/tervonvpndpi` по утверждённой области.

### JOB-007 — Реальный Homeserver Android Source Preflight (результат оператора)

- **Цель:** установить текущие семь файлы/зависимости в фактическом приложении, текущую ветку/HEAD/dirty и точный SHA установленного диагностического Go ARM64; не изменять файлы.
- **Команды оператора:** `cd /opt/tervon-tcp-bridge-m6-android-ingress`; `git pull --ff-only origin work/m6-android-ingress` (fast-forward `36864cb..0e1aa67`); `python3 integrations/android-bridge-v1.0.2/route/current_app_preflight.py --app-root /opt/tervonvpndpi`.
- **Источник:** полный операторский вывод приложения, получен в чате; без секретов. **Метка времени исполнения:** в выводе отсутствует, `TEST_START=UNKNOWN`, `TEST_END=UNKNOWN`. Текущий GitHub docs commit регистрирует обработку, не момент теста.
- **Реальная среда:** app branch `feature/status-ping-cleanup` @ `22ff78a9dbabc1544a53b9ac950f78413f25b4fb`, `tracked_dirty=true`.
- **Наблюдения:** `CURRENT_APP_PREFLIGHT=PASS_READ_ONLY`; семь целевых файлов находятся по одному в `android/app/src/main/kotlin/com/tervonvpndpi/app/`, их точные SHA256 записаны в DEVELOPMENT_LOG запись 011. `libtervonbridge.so` существует, SHA256 `a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd`, `matches_confirmed_diagnostic_core=true`. Locations of call-sites содержат, в частности, `VpnForegroundService.kt`, `MainActivity.kt`, `BridgeRuntimeSettings.kt`, `BridgeClientPemHandoff.kt`, `BridgeProcessDiagnostics.kt`.
- **Критерии:** список должен отражать реальный Android worktree без изменения файлов; Go core checksum должен совпасть; текущие call-sites должны быть выявлены. **Результат:** `PASS_READ_ONLY_INVENTORY`; `PASS_ANDROID_GO_CORE_SHA`; `SEVEN_TARGETS_FOUND=7/7`; **`WORKTREE_DIRTY=YES`**.
- **Ограничения:** source text **не прочитан**, только пути/строки/хеши. Нельзя делать вывод о сигнатурах и фактическом управлении процессом; нельзя удалять Kotlin-файлы без сохранения границ взаимодействия с обычным VPN. `APP_CLEANUP=NOT_RUN`; `NEW_IMPL_APP_BUILD=NOT_RUN`; `DEVICE_E2E=NOT_RUN`.
- **Следующий строго необходимый preflight того же модуля:** read-only `git status --short` плюс приватное получение актуальных Kotlin исходников семи файлов и точек вызова. Никакого применения старых patch source или нового installer без сопоставления кода.
