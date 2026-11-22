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

### JOB-008 — Прямое исследование текущего приватного исходного архива Android

- **Исходная ветка/HEAD:** `work/m6-android-ingress` @ `52c8e8a01b9fd2cd7231bdfb7f27b72c9b7fb3a1`.
- **Основание:** архив 12 Kotlin файлов из реального Homeserver после JOB-007. Имя `android-bridge-v102-current-source.tar.gz`; SHA256 `02b960f9f9dd5d7f6486cc593217146df966dc05a1b50336008f6a50cb235a16`. Архив и приватный исходный текст не публикуются в GitHub.
- **Вопросы проверки:** архив текущий? Есть ли семь правильных Kotlin и точные зависимости? Какие старые механизмы нельзя переносить? Какие внешние интерфейсы необходимы для замены без повреждения VPN?
- **Среда/метод:** Python `tarfile` read-only `r:gz`, `hashlib.sha256` по каждому regular entry, статическое исследование строк 12 файлов; поиск exact call-sites `BridgeVpnSession.prepare`, `BridgeControlChannel.register`, `params.tcpRoutes`, `BridgeRuntimeControl.register`; без установки и запуска APK.
- **Ожидаемое:** 12 regular entries без symlink/traversal; все 7 target SHA совпадают с JOB-007; реальные вызовы в `VpnForegroundService` и `MainActivity` известны. **Фактически:** `ARCHIVE_TARGET_SHA=PASS_7_OF_7`; `ENTRIES=12`; `REGULAR_PATHS=PASS`; `INTEGRATION_STATIC_CONTRACT=PASS_OBSERVED_DEPS`.
- **Подтверждённые line locations:** `BridgeVpnSession.kt:63` — отменённая `tcpRoutes`; `BridgeControlChannel.kt:15` — регистрация второго runtime channel; `BridgeRuntimeControl.kt:12,40,59,69` — другой MethodChannel и независимые commands; `VpnForegroundService.kt:367,565` — Bridge prepare; `MainActivity.kt:36` — Bridge register. В `VpnForegroundService.kt:578,914` собственное приложение исключено из TUN через `addDisallowedApplication(packageName)`.
- **Результат:** `JOB-008=PASS_READ_ONLY_SOURCE_RECONCILIATION`, `SEVEN_SHA_MATCH=PASS`, `DUPLICATE_RUNTIME_CONTROL=CONFIRMED_SOURCE`, `STATIC_TCP_ROUTES_REFERENCE=CONFIRMED_SOURCE`; `ANDROID_APP_REPLACEMENT=NOT_RUN`, `ANDROID_BUILD=NOT_RUN`, `DEVICE_E2E=NOT_RUN`.
- **Ограничения:** архив не включает все классы Discovery/PKI и Dart consumers, поэтому полный source/API contract не установлен. `git status --short` приложения пока отсутствует при `tracked_dirty=true`. Нельзя применять массовый rewrite без проверки dirty paths. По причине NO INTERNET имеется только прежний device E2E FAIL и гипотезы; static audit не доказывает единственную причину.
- **Время:** фактическое начало/конец исполнения Python отдельно не измерены; `EXEC_TIME=NOT_MEASURED`. Фиксацию работы датировать GitHub commit `created_at`, не подменять этим метку самого теста.

### JOB-008 — GitHub timestamps evidence

- Запись 012: `5924b007c5cce1f8840be48ece9155c1fa1b216a`, GitHub `created_at=2026-11-22 11:56:59 UTC` = **18:56:59 Новосибирск**.
- JOB-008: `68b2689ec7880229fdbb3b1b1c339c12357b944e`, GitHub `created_at=2026-11-22 11:57:14 UTC` = **18:57:14 Новосибирск**.
- Это время документальных коммитов, а не точное время исполнения локального Python. Приложение, Docker, сервер и бинарник не изменены.

### JOB-009 — Read-only dependency archive verification and seven-file repair triage

- **Исходный HEAD:** `a73ba6085cb71d08b99472b3bd8368921ae727c8` (ветка `work/m6-android-ingress`), development record commit `b48c980da64012f1048b827049b4bff5661abbd0`.
- **Входные данные:** `android-bridge-v102-dependencies.tar.gz`, SHA256 `ce9f0403b8e7605477d2088b8a1a40766ba0a55d8710f1a50c5a00da4bf3ff77`; первый архив `android-bridge-v102-current-source.tar.gz`, SHA256 `02b960f9f9dd5d7f6486cc593217146df966dc05a1b50336008f6a50cb235a16`. Не публиковать архивы в GitHub.
- **Вопрос:** можно ли исправить семь неработоспособных Kotlin-файлов минимально с учётом реальных существующих зависимостей, а не переписать весь механизм?
- **Метод:** Python `tarfile.open(path, 'r:gz')`, перечисление объектов, локальный SHA256 архивов, `hashlib.sha256` для каждого содержимого, сравнение пересекающихся путей; статическое чтение текущих классов и call-sites без записи.
- **Результат:** `27` regular Kotlin entries во втором архиве, `12` в первом, пересечений `10`, `HASH_MISMATCH=0`; `DEPENDENCY_ARCHIVE_INTEGRITY=PASS`, `OVERLAP_IDENTICAL=PASS_10_OF_10`. Просмотрен `BridgeAutoEnrollmentHttp.kt` с `tcpRoutes`; `BridgeRuntimeControl` с независимым `MethodChannel`; сохранены source interfaces process/config/diagnostics.
- **Вывод статического анализа:** `BridgeCoreManager.kt` и `BridgeConfigManager.kt` не требуют автоматической полной замены; `BridgeControlChannel.kt`, `BridgeRuntimeControl.kt`, `BridgeXrayConfigRoute.kt` ремонтопригодны минимальными правками при сохранении API; `BridgeVlessRoutePlan.kt` подлежит замене алгоритма на проверенный pure route-policy; `BridgeVpnSession.kt` требует замены устаревшего service binding и корректной интеграции без `tcp_routes`. Этот вывод **не является функциональным PASS**.
- **Подтверждённая блокировка:** `BridgeAutoEnrollmentHttp.kt` за пределами семёрки и current GitHub registrar сохраняют `tcp_routes`, тогда как новый approved discovery contract его отменяет. Нельзя подменять неподтверждённую индивидуальную маршрутизацию одним `params.service` для всех профилей; source-level правки без live contract evidence могут снова сломать VPN.
- **Другие отсутствующие доказательства:** Android Git `tracked_dirty=true`, но точный `git status --short` не получен; Dart consumer код и live Discovery response не проверены. Ни один файл приложения и работающие службы не менялись.
- **Итог:** `JOB-009=PASS_SOURCE_RECONCILIATION`; `IMPLEMENTATION=BLOCKED_PENDING_SAFE_APP_WORKTREE_AND_SERVICE_CONTRACT`; `ANDROID_GRADLE=NOT_RUN`; `DEVICE_E2E=NOT_RUN`. Время локальной проверки отдельно не измерялось; точное время записи — GitHub `created_at` commit.

### JOB-010 — Зависимости deprecated `tcp_routes`; безопасное удаление BLOCKED

- **Исходный HEAD:** `70894656ecf8496c078d4eff91f09bcb986f491b`; dev-log commit `afca8031c5e5039c15e5bbd4cd8716142d066fe2`. Поручение: удалить отменённый route mechanism без риска для исполняемого регистратора и действующего сервера.
- **Read-only source inspection:** `xui/cmd/m6-auto-registrar/main.go` вызывает `configuredTCPRoutes`, которое извлекает mapping из `config/server.json` и направляет `TCPRoutes` в `BridgeEnrollmentHTTP`. `enrollment_http.go` имеет fallback `routes := h.TCPRoutes`, заменяя их на `ConfirmedRoutes.Snapshot()` лишь при injected store; registrar **не задаёт ConfirmedRoutes**. `routes_test.go` и `enrollment_http_test.go` сохраняют legacy tests.
- **Точная проверка предыдущих PASS по docs/V1_0_2_DEVELOPMENT_LOG.md:** Go route sync unit/race/vet PASS (`b08814afdba06f979ab3df774cb4a07e59c00508`); live three-way 3x-ui/Xray gRPC/allowed routes PASS (`92356cba775e61cef09b6e20c4b0142d7850d9e6`); core full policy isolated ACK PASS, Docker v1.0.2 cutover PASS. Однако `LIVE_DISCOVERY_SYNC=NOT_CONNECTED` прямо отмечено в историческом журнале. Никакие из этих прежних PASS не устанавливают реализованный callback в registrar.
- **Вопрос:** удаление legacy функции без риска сохраняет сборку и поведение? **Ответ: NO** по прямым ссылкам исходников. Нельзя удалять прямо сейчас без реального wiring или разрешения на отключение регистратора. Не создавать фиктивную новую структуру или принудительно закрывать уже работающую регистрацию.
- **Статус:** `SOURCE_AUDIT=PASS`; `LEGACY_REMOVAL=BLOCKED_DEPENDENCY`; `GO_CODE_MODIFIED=NO`; `GO_TEST=NOT_RUN_NO_CHANGE`; `LIVE_SERVICES=UNCHANGED`. Следующее допустимое действие — согласовать scope для подключения существующих реальных компонентов управления перед удалением legacy; отсутствие новых тестов не считать FAIL.

### JOB-011 — Удаление legacy-static source, regression CI

- **Согласовано:** удалить `configuredTCPRoutes`/legacy `tcp_routes` или исправить по утверждённой архитектуре. **HEAD до кода:** `da47e0c1043fdbcb9adb4127c2ab6018219fc8d7`. **Commit кода:** `f5d70c587448449f5765627cd7433e5fdeacb53d`.
- **Вопрос:** осталось ли ошибочное вычисление port→service из runtime `server.json` и публикация отдельного старого поля Discovery; можно ли удалить без объявления локального listener успешным backend E2E?
- **Исходные изменения:** remove `configuredTCPRoutes`; delete его `routes_test.go`; remove `BridgeEnrollmentHTTP.TCPRoutes` и `bridgeDiscoveryResponse.TCPRoutes`; remove static fallback; require `ConfirmedRouteStore`, fail closed on unconfirmed/ambiguous service selection, reject GET/POST until acknowledged generation. Existing certificate issuance/CSR and full Go core untouched.
- **Тесты в source:** существующие enrollment/CSR HTTP тесты изменены для confirmed fixture; дополнена проверка отсутствия JSON key `tcp_routes`, nil store, ambiguous two-service routes and GET/POST 503, failed generation ACK.
- **Go CI:** GitHub push workflow `M6 3X-UI adapter tests`, run `45023624013`, команда `go test -race -count=1 ./...`; runtime source workflow `45023623991`. При создании записи `status=in_progress`, conclusion `UNKNOWN`. **Нельзя ставить PASS до подтверждённого run conclusion.**
- **Ограничение:** `m6-auto-registrar` source после чистки без доверенного injected policy generation отказывается от discovery/enrollment. Изоляция хороша для удаления опасной старой выдачи, но **не годится для production cutover**. Нельзя объявлять «Android готов» и устанавливать регистратор, пока не закрыты source wiring и тесты реального full-policy ACK. Docker/live services и Android не менялись.
- **Статус:** `LEGACY_REMOVAL=SOURCE_COMMITTED`; `CI=RUNNING`; `PRODUCTION_CUTOVER=NOT_RUN`; `ANDROID_E2E=NOT_RUN`.

### JOB-012 — Финальный Go CI после source cleanup

- **Цель:** исправить legacy route path, получить подтверждённый компилирующийся и протестированный source без изменения production.
- **Источник/commit:** `work/m6-android-ingress`, последний code commit `1fa836c724a07717c81253c2d0479cd985312dad`. Обновление действующих инструкций: `1f9245cc3f374587e027fbb6b2bfa18a9b16b32a`.
- **Попытка 1:** code commit `f5d70c...`, GitHub Actions `45023624013` — **FAIL**, ошибка синтаксиса `cmd/m6-auto-registrar/main.go:167,174`. Не считать source PASS. Исправлено commit `11518161d61c53401e3e2144336d975fce515820`.
- **Итоговый код:** удалён deprecated `routes_test.go` и `configuredTCPRoutes`, удалено отдельное поле `tcp_routes` из HTTP response, `BridgeEnrollmentHTTP` требует `ConfirmedRouteStore`, при неточном/неподтверждённом выборе 503; добавлены unit negative cases, `services_test.go` не выводит ports из server.json. Сохранены доверенные существующие crypto/CSR helpers без нового алгоритма.
- **Проверка 1:** GitHub Actions `M6 3X-UI adapter tests` `run_id=45023832666`, `head_sha=1fa836c...`, `go test -race -count=1 ./...`; **COMPLETED / SUCCESS = PASS**.
- **Проверка 2:** GitHub Actions `M6 runtime ACL source check (released v1.0.1)` `run_id=45023832706`, `head_sha=1fa836c...`; **COMPLETED / SUCCESS = PASS**. Это источник-тест, не запуск реального серверного контейнера.
- **Не выполнялось и не утверждается:** `PRODUCTION_REGISTRAR_DEPLOYMENT=NOT_RUN`, `LIVE_API_TO_RUNTIME_ACK=NOT_CONNECTED`, `ANDROID_CLIENT_COMPATIBILITY=NOT_TESTED`, `ANDROID_INTERNET=FAIL_HISTORICAL`. Нельзя развернуть текущую source-сборку регистратора без подтверждённого writer; она fail-closed с HTTP 503 и не поддерживает неоднозначный выбор сервиса.
- **Результат текущей согласованной очистки:** `LEGACY_STATIC_SOURCE_REMOVAL=PASS`, `GO_MODULE_CI=PASS`, `RUNTIME_ACL_SOURCE_WORKFLOW=PASS`; `PRODUCTION_CUTOVER=NO`. Никаких новых live проверок без основания.

### JOB-013 — Android two-file minimal source patch: локальная проверка

- **Source:** public GitHub `04f9ac6ef8d9f33cbac324740640c07c23257a1d`, `work/m6-android-ingress`; Android Homeserver `feature/status-ping-cleanup` HEAD `22ff78a9dbabc1544a53b9ac950f78413f25b4fb`. Перед правкой получен пользовательский read-only `git status --short`, полный SHA256 семи файлов и Dart consumer grep.
- **Private source material:** `android-bridge-v102-dependencies.tar.gz` 27 Kotlin файлов и `tervonvpn-m6-source-audit.tar.gz` для проверки Dart channel calls. Текущие семь файлов совпали по SHA с архивом. Приватные Kotlin исходники не добавлялись в GitHub.
- **Местные команды:** `python3 -m py_compile /mnt/data/m6_bridge_android_fix.py`; `python3 /mnt/data/m6_bridge_android_fix.py --app-root /mnt/data/m6-analysis` (read-only); `python3 -m unittest -v /mnt/data/test_m6_bridge_android_fix.py`.
- **Expected:** отказ при неверном hash до любой записи; только два утверждённых Kotlin файла; сохранены вызовы `prepare/stop/active`, оба MethodChannel и единственное управление процессом из VPN; не затрагиваются сервер, APK и сертификаты.
- **Actual:** compile script PASS, read-only target SHA PASS; unit **4/4 PASS** (`test_read_only_preflight`, `test_apply_only_two_files`, `test_guard_prevents_any_partial_write`, `test_second_apply_refuses_new_unrecognized_source`). Новые SHA в тестовой копии: `BridgeVpnSession.kt=6b8cfc1fc4f0cdb912be5485f21ba2a9364796dcaf6793bcdf785aa4b3032004`, `BridgeRuntimeControl.kt=79b3d3a1cc4ba1725523f40c0fc32458efc420b9072f0add98f02f46e9cb375f`.
- **Limit:** это проверки патчера на локальном source snapshot, а не Kotlin/Gradle compile или Android device tests. `APP_PATCH_APPLIED=NO`, `KOTLIN_COMPILE=NOT_RUN`, `DEVICE_E2E=NOT_RUN`, `SERVICE_BINDING_NOT_VERIFIED`. Исполняемая часть Android на Homeserver не затронута. Каталог `/mnt/data` локален только этой беседе; пользователь должен явно перенести артефакт на Homeserver перед применением.
- **Status:** `SOURCE_PATCH_CANDIDATE=PREPARED`; `PATCH_LOCAL_TEST=PASS_4_OF_4`; `ANDROID_IMPL=NOT_APPLIED`; `ANDROID_FUNCTIONAL=NOT_PASS`.

### JOB-014 — проверка текущих двух Kotlin файлов и release Kotlin compile (оператор Homeserver)

- **Контрольный вопрос:** совпадает ли содержимое двух целевых Kotlin-файлов с локально проверенным patch-кандидатом и компилируется ли Kotlin модуль без выпуска APK?
- **Команда проверки:** `sha256sum -c -` для двух файлов, затем в `/opt/tervonvpndpi/android` команда `bash ./gradlew :app:compileReleaseKotlin --offline --no-daemon`.
- **Ожидаемое:** обе SHA256 совпадают, Gradle exit code 0, сборка не устанавливает APK и не затрагивает сервер.
- **Фактически предоставлено оператором:** `M6_PATCHED_SOURCES=PASS`; `BUILD SUCCESSFUL in 19s`; `72 actionable tasks: 5 executed, 67 up-to-date`; `KOTLIN_COMPILE=PASS`; `ANDROID_E2E=NOT_TESTED`; `SERVER_SERVICES=UNTOUCHED`; `COMMAND_EXIT_CODE=0`; `SSH_CONSOLE=OPEN`. Точная дата/время выполнения команды по серверным часам и новый Git HEAD приложения не представлены. Две целевые SHA256: `6b8cfc1fc4f0cdb912be5485f21ba2a9364796dcaf6793bcdf785aa4b3032004` и `79b3d3a1cc4ba1725523f40c0fc32458efc420b9072f0add98f02f46e9cb375f`.
- **Примечание:** предыдущий dry-run патчера заблокирован потому, что первый Kotlin файл уже имел целевой SHA. Не утверждать, что исправление было применено последней командой; проверено соответствие текущего состояния.
- **Итог:** `SOURCE_HASH_CHECK=PASS`, `KOTLIN_COMPILE=PASS`; `DEVICE_TCP_E2E=NOT_RUN`, `APK_BUILD=NOT_RUN`, `GO_TRANSPORT_INTEGRATION=NOT_CONFIRMED`, `RELEASE=NOT_READY`.

### JOB-015 — Android unit test Gradle task lookup failure (оператор Homeserver)

- **Question:** доступны ли Kotlin/JVM unit tests через `./gradlew :app:testReleaseUnitTest --offline --no-daemon`?
- **Result supplied by operator:** `FAILURE: Build failed`, `Cannot locate tasks that match ':app:testReleaseUnitTest' as task 'testReleaseUnitTest' not found in project ':app'`; `BUILD FAILED in 14s`; `4 actionable tasks: 4 up-to-date`; `UNIT_TEST_RC=1`, `COMMAND_EXIT_CODE=1`, `SSH_CONSOLE=OPEN`. Ошибка — отсутствующий task, **не тестовый assert и не ошибка Kotlin**.
- **Prior result retained:** `KOTLIN_COMPILE=PREVIOUS_PASS`, source hashes PASS по JOB-014. `APK_BUILD=NOT_RUN`, `ANDROID_E2E=NOT_TESTED`, `SERVER_SERVICES=UNTOUCHED`.
- **Warnings:** Kotlin embedded/plugin mismatch и Kotlin/AGP migration notices, но нет данных, что они вызвали этот FAIL. Версии и build config не изменять без отдельного решения.
- **Next:** read-only listing `:app:tasks --all`, inspection of actual `test` source sets and Gradle plugin settings before choosing any test command. Do not invent a test task or count this as unit-test execution.
- **Status:** `UNIT_TEST_TASK=ABSENT`, `ANDROID_UNIT_TESTS=NOT_RUN`, `KOTLIN_COMPILE=PASS_PREVIOUS`, `DEVICE_E2E=NOT_RUN`.

### JOB-016 — поиск доступных Android тестовых задач и тестовых исходников

- **Цель:** выяснить, какой test task реально доступен и есть ли unit/instrumentation источники, прежде чем запускать тесты M6.
- **Оператор:** `:app:tasks --all` с `--offline --no-daemon --console=plain`; `find app/src/test app/src/androidTest`; `grep` Gradle build/plugins. `BUILD SUCCESSFUL in 15s`, `READ_ONLY_INSPECTION=COMPLETE`, `COMMAND_EXIT_CODE=0`.
- **Наблюдение:** `testDebugUnitTest` существует, `testReleaseUnitTest` отсутствует. `ABSENT=app/src/test`; `ABSENT=app/src/androidTest`; `org.jetbrains.kotlin.android` plugin и `android.builtInKotlin=false` используются. Следовательно, запуск test task не обеспечит утверждённое функциональное покрытие Bridge.
- **Результат:** `TEST_TASK_INVENTORY=PASS`, `BRIDGE_UNIT_TESTS=NOT_PRESENT`, `UNIT_TEST_EXECUTION=NOT_RUN`, `KOTLIN_COMPILE=PREVIOUS_PASS`, `DEVICE_TCP_E2E=NOT_RUN`, `SERVER_UNTOUCHED=YES`.
