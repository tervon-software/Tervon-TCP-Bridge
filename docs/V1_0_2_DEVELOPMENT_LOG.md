# Tervon TCP Bridge v1.0.2 — журнал разработки

**Дата открытия:** 2026-11-22. **Рабочая ветка:** `work/m6-android-ingress`. **Исходный HEAD:** `af72b959e227a849e72f9bc1f99ee20ecae94fec`. **База проверенного релиза:** Go Bridge v1.0.1, исходный commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86`.

**Назначение журнала:** фиксировать каждый выполненный шаг, изменённые файлы, основание решения, точные проверки и результаты. Статусы `СОГЛАСОВАНО`, `РЕАЛИЗОВАНО`, `ТЕСТИРОВАНО`, `PASS`, `ВЫПУЩЕНО` различаются. Нельзя переносить PASS другого коммита на новый.

## 2026-11-22 — Шаг 0. Фактический аудит и согласованный объём

**Статус:** АУДИТ ПОДТВЕРЖДЁН ПО ПРЕДОСТАВЛЕННЫМ ОПЕРАТОРОМ ВЫВОДАМ; СОГЛАСОВАНА РАЗРАБОТКА; РЕАЛИЗАЦИЯ v1.0.2 ЕЩЁ НЕ НАЧАТА.

**Проверенные сведения Homeserver (оператор, чтение без изменений):**
- Android-facing Docker `android-ingress-android-bridge-decapsulator-1`, образ `tervon-bridge-android-ingress:m6-runtime-acl`, метка `m6-v1`, порт хоста 18443/tcp → контейнер 8443/tcp, `/etc/tervon/bridge-android/config` монтируется только для чтения.
- Два дополнительных диагностических контейнера `m6-acl-mutation-probe` и `m6-acl-isolated-probe` работали на момент аудита; не изменялись.
- `server.json`: маршруты `vpn→30000`, `vpn-xhttp→55636`. В текущем Xray inbound: VLESS TCP/TLS `55535`, VLESS XHTTP/TLS `55636`, Trojan TCP/TLS `30000`. Пользователь пояснил, что Trojan на 30000 — намеренное диагностическое изменение; **не считать дефектом протокола**. TCP inbound 55535 отсутствует в карте Bridge.
- Существующий `m6-auto-registrar.candidate` запущен в systemd; API-клиент в GitHub реализует только чтение клиентов, а runtime registrar получает `tcp_routes` из `server.json` один раз при старте.
- Авторизованный GET `/panel/api/inbounds/list/slim` через локальное HTTPS-подключение с сохранённой проверкой сертификата (внешний hostname:5588 принудительно разрешён в 127.0.0.1) вернул HTTP 200 и три активных inbound. Прямой внешний запрос завершился timeout. Bearer header хранится в root-only файле (0600); токен и закрытый URL панели в GitHub не публикуются.
- Нынешний `m6-runtime-acl` умеет обновлять по SIGHUP **только ACL**; изменения `routes` отклоняются. Наличие новых данных в панели **не** изменяет маршрут работающего Decapsulator.
- Android запускал локальный Encapsulator, но пользователь наблюдал отсутствие интернета; причина E2E не установлена. Локальный listener/MTLS/байтовые счётчики нельзя выдавать за успешную работу интернета.

**Согласованные требования к v1.0.2:**
1. Серверное ядро получает действующие TCP inbound/порты через API 3x-ui и Remnawave, с адаптерами к единой проверенной маршрутной модели; мониторинг появления, изменения, отключения и удаления. API-путь/тип панели и секреты — конфигурация оператора, без hardcoded VPN endpoint. Клиентские данные 3x-ui для этой задачи не используются.
2. Горячая атомарная публикация маршрутов для **новых** TCP-соединений, без обрыва существующих; сохранить TLS 1.3/HTTP2/mTLS, ACL, Padding/Jitter и проверенную TCP-передачу v1.0.1. Не превращать Bridge в открытый TCP-прокси.
3. Сведения API проверяются по фактическому назначению inbound; для Remnawave учитывать принадлежность профиля конкретному Node и каскад. Один Decapsulator на главной ноде допустим; установка на других нодах **не запрещается**.
4. Android позже берёт VPN endpoint из **выбранного профиля подписки**, на этом же домене ищет подтверждённый Bridge; отсутствие проверенного Bridge означает отказ от режима обхода. VPN-порт и HTTPS-порт Bridge могут различаться. На текущем этапе **Android и APK не изменять**.
5. Диагностика **реальных соединений**, серверная и через Android ADB позже: `ЕСТЬ/НЕТ` → последний успешный этап → первый неуспешный этап → подтверждённая причина. Различать локальный запуск, защищённое соединение, серверный backend dial, направление байтов, обратные байты, VPN/DNS и доступность интернета. При недостатке наблюдений — `CAUSE=NOT_DETERMINED`, не гадать. Корреляция одной сессии и безопасные журналы без токенов, UUID пользователей, секретов или содержимого VPN.
6. API недоступен — отдельный диагностический отказ; не путать недоступность API и всего сервера, не превращать отсутствие ответа в команду удалить все маршруты. Политика свежести/срока действия снимка и частота опроса требуют конкретизации и испытаний, а не произвольных значений.
7. Сохранить прежний режим static routes для Linux Bridge без API. Конкретная схема v1.0.2 и версия её релиза не считаются выпущенными до PASS.

**Технические блокеры до развёртывания:** нынешний SIGHUP обрабатывает только ACL; runtime registrar публикует статичную карту; актуальный API-контракт Remnawave и Node mapping надо проверить; конечную политику stale-data и журналов необходимо проверить тестами. Ветки/запущенные службы не изменялись.

**План пакетов (строгая последовательность):**
- **Этап 1:** серверный 3x-ui API inbound-read и безопасная проверка снимка + unit/negative tests; PASS по точному commit.
- **Этап 2:** ядро Go v1.0.2, атомарные runtime routes вместе с ACL, корректные новые/старые потоки; race/integration tests.
- **Этап 3:** server discovery/registrar получает тот же опубликованный снимок; проверка согласованности и отказов.
- **Этап 4:** сквозная серверная диагностика реального потока и ошибок, correlation/byte counts; тесты на сбои.
- **Этап 5:** Remnawave API/Node adapter, standalone и cascade scenario; тесты.
- **Этап 6:** изолированный кандидат на Homeserver, только после точных PASS — отдельно согласованный cutover. Документация/release лишь по подтверждённым результатам; Android — отдельный следующий этап.

**Следующий шаг:** реализация и изолированная проверка **Этапа 1**, без обновления работающих служб. Никаких PASS v1.0.2 пока нет.

## 2026-11-22 — Шаг 1. Исходный адаптер чтения inbound 3x-ui

**Статус:** `SOURCE_COMMITTED`; **`GO_TEST=NOT_RUN`, `GO_RACE=NOT_RUN`, `GO_VET=NOT_RUN`, `LIVE_API_ADAPTER=NOT_TESTED`, `ROUTES_HOT_RELOAD=NOT_IMPLEMENTED`, `v1.0.2_PASS=NO`**.

**Изменённые файлы:**
- `integrations/android-ingress/xui/inbounds.go` — независимый read-only вызов `GET /panel/api/inbounds/list/slim` через существующий HTTPS Bearer-клиент; строго проверяются HTTP-ответ, структура списка, ID, порт, enable, streamSettings, допускается объект или legacy JSON-string. Возвращается **только очищенная** проекция (id/port/protocol/enabled/network/security); клиентские секреты не выводятся. Не меняет маршруты.
- `integrations/android-ingress/xui/inbounds_test.go` — mock HTTPS success и negative tests некорректных/дублированных ID, порта, отсутствующего enable, некорректных настроек, отказа API, неутечки токена/пути.
- Имеющийся `client.go` и путь к данным клиентов не удалялись: удаление legacy-API и замена runtime регистратора — отдельные изменения после подтверждённых тестов; не нарушать текущую систему.

**GitHub commits:** `82a3fba0ecd5372684c9b7c0ae2d7b646b2e0802` (исходник); `47ba046c1e3003340858d2691c914cf6904b8052` (тест); `8273217cc3c3df3a097957b39989452e2b5c6e7c` (исправление fixture двойного JSON escape при повторном чтении исходника).

**Проверки, которые необходимо выполнить на точном HEAD этого пакета:**
```bash
cd /opt/tervon-tcp-bridge-m6-android-ingress/integrations/android-ingress/xui
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```
Перед запуском убедиться, что checkout — нужная ветка, получен новый HEAD без потери локальных изменений. Эти команды не обновляют Docker, 3x-ui, Xray, Nginx, Android и не требуют токена API. Тесты mock HTTP и не подтверждают боевую маршрутизацию.

**Следующий шаг:** получить результаты названных Go-проверок с Homeserver. При FAIL исправить именно этот модуль и повторить только относящиеся к дефекту проверки. При PASS записать точный HEAD и переходить к серверному dynamic-routes механизму по утверждённому плану. Не создавать ложное впечатление, что один API-adapter устранил отказ Android 55535.

## 2026-11-22 — Шаг 1: подтверждённые испытания на Homeserver

**Источник фактов:** непосредственный вывод оператора с Homeserver, не предположение и не CI-сводка.

**Проверенный рабочий каталог:** `/opt/tervon-tcp-bridge-m6-android-ingress/integrations/android-ingress/xui`.
**Ветка:** `work/m6-android-ingress`.
**Исходный HEAD до получения изменений:** `48e4370`; выполнено `git pull --ff-only origin work/m6-android-ingress`, результат `Fast-forward`; **тестируемый HEAD:** `09d663d5aa794793aa2fa23358dc47477209c9f3`.
**Проверка чистоты tracked-файлов перед pull:** `git status --porcelain --untracked-files=no` пустой, иначе скрипт прекратил бы работу. **Наличие untracked-файлов не исследовалось**.
**Пакеты:** 6 (`xui`, `xui/catalog`, `xui/cmd/catalog-keygen`, `xui/cmd/catalog-sign`, `xui/cmd/m6-auto-registrar`, `xui/cmd/m6-first-device`).

| Команда | Результат оператора | Статус |
| --- | --- | --- |
| `go test -count=1 ./...` | Все шесть пакетов `ok`, `TEST_RC=0` | **PASS** |
| `go test -race -count=1 ./...` | Все шесть пакетов `ok`, `RACE_RC=0` | **PASS** |
| `go vet ./...` | Без диагностических сообщений, `VET_RC=0` | **PASS** |

**Границы результата:** это PASS исходного Go-пакета 3x-ui inbound API reader и связанных существующих Go-пакетов на проверенном HEAD. Это **не** live-интеграция нового метода в регистратор; mock TLS-тесты не заменяют проверку runtime-синхронизации с API. Ранее отдельный операторский read-only запрос к реальной панели `/panel/api/inbounds/list/slim` через локально разрешённый HTTPS, с неизменённой проверкой TLS, вернул HTTP 200 / 3 inbound. Текущая Go-команда не проверяла реальный API. `ROUTES_HOT_RELOAD=NOT_IMPLEMENTED`; `DOCKER_E2E=NOT_TESTED`; `ANDROID_E2E=NOT_PASS`; `v1.0.2_RELEASE=NO`.

**Отсутствие затрагивания служб:** тестовая команда не содержит изменения Docker/Xray; операторский итог `DOCKER_CHANGED=NO`, `XRAY_CHANGED=NO`. Подтверждения состава/состояния других служб этот прогон не предоставляет.

**Критерий закрытия Шага 1:** код + позитивные и негативные тесты + Go test/race/vet PASS на идентифицированном commit + документирование в GitHub. **Шаг 1 закрывается после commit настоящей записи.** Следующий согласованный шаг: **Шаг 2 — атомарная runtime-таблица маршрутов для новых соединений**. Перед изменениями проверить точный Go release v1.0.1 и ACL overlay, сохранить контроль отсутствия изменений в действующем Docker. Требования к диагностике: соединение есть/нет, фактический этап и проверяемая причина; подробные сквозные проверки относятся к отдельному утверждённому этапу.

## 2026-11-22 — Шаг 2. Изолированный кандидат атомарных ACL+routes

**Статус: `SOURCE_COMMITTED`, `GO_TEST=PENDING`, `GO_RACE=PENDING`, `GO_VET=PENDING`, `LIVE_DEPLOY=NO`, `ROUTE_SYNC_API=NOT_IMPLEMENTED`.** Этап 2 до подтверждённого PASS не закрыт.

**Основание:** в релизном `prototype/bridge.go` commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86` `Server.Routes` неизменяем в процессе, а M6 `runtime-acl/acl_snapshot.go` держит только ACL. `runtime-acl/apply-to-release-source.py` отклоняет любые изменения `routes` при SIGHUP. Простая подмена map недопустима: новые потоки могут увидеть несогласованные разрешения и backends.

**Созданные исходники:**
- `integrations/android-ingress/runtime-routes/policy.go` — новый `RuntimePolicy` с `atomic.Pointer`, проверкой ACL+routes и глубокой копией списка backend. `Select` читает один снимок и возвращает связанную пару «право + список backend». `Replace` отклоняет некорректный кандидат, не изменяя предыдущую карту. Порты проходят проверку диапазона 1–65535; это **не** замена проверке соответствия действующему Xray.
- `integrations/android-ingress/runtime-routes/policy_test.go` — изменение/добавление/удаление сервисов, rollback на ошибке, конкурентное чтение при заменах, отсутствие alias входных структур.
- `integrations/android-ingress/runtime-routes/policy_stream_test.go` — E2E-кандидат mTLS HTTP/2 → Go Bridge → две локальные TCP echo backend: существующий поток обязан остаться на backend A, новый после `Replace` должен попасть на B.
- `integrations/android-ingress/runtime-routes/apply-to-acl-source.py` — **source-only** overlay поверх отдельного изолированного дерева релизных v1.0.1 исходников, в которое предварительно применён уже существующий `runtime-acl/apply-to-release-source.py`. Изменяет `prototype/bridge.go` и `cmd/bridge/main.go`, добавляет два `policy*.go` тестовых файла и `policy.go`. Для потоков v1.0.2 используется `RouteLookup`/атомарный policy; старый путь `Routes`/`AuthorizationLookup` остаётся для совместимости. SIGHUP позволяет менять только `authorized` и `routes` (все прочие поля конфигурации обязаны совпадать); публикует `M6_POLICY_RELOAD=PASS digest=...` после применения целого снимка.

**Точные commits исходного пакета:** `a9104d2e65698a1818040366d861391cb901d49e` (`policy.go`), `2492358f0e5ecf8273f8991950bbc2ed245ebfca` (`policy_test.go`), `2b3c70d21687de24ac7e22f3d8ef61e2a4af3354` (overlay), `cdbf1199d53ad0b9ad73ffe6c2ded2d0dd61c382` (валидация порта), `94e90545f01f1bae9faf9c1ff1a6a02828c49258` (live-stream test).

**Безопасность/ограничения:**
- В действующий Docker образ, `server.json`, Xray, Nginx, регистратор, Android или Linux RU↔FI Bridge изменения **не** вносились.
- Важный следующий шаг — проверить Go-совместимость с чистым релизом, race, vet, фактические h2/mTLS тесты **на конкретном HEAD**; script `--check` перед `--apply`. Тестовый исходник не означает работающий продукт.
- Регистратор и Docker confirmer пока рассчитаны на `M6_ACL_RELOAD` и старую метку образа; **их нельзя направлять на кандидат v1.0.2, пока не будет согласованной и протестированной интеграции**.
- Само чтение API 3x-ui (Шаг 1) не подключено к новому `RuntimePolicy`; синхронизация, выбор реального адреса backend и точное подтверждение активации маршрута — этап 3 и последующие проверки.
- `M6_POLICY_RELOAD=PASS` — формат будущего подтверждения ядра; не заявлен как уже полученный.

**Проверка на Homeserver (ещё не выполнена):** создать новую отдельную worktree на точном `96c1367...`, применить ACL overlay, затем candidate routes overlay, выполнить `gofmt`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go build ./cmd/bridge` с результатом build без изменений действующего бинарника. При сбое остановиться и исправить текущий этап; без PASS не переходить к следующему.

## 2026-11-22 — Шаг 2: результат первого прогона Homeserver и исправление тестового комплекта

**Операторская проверка на Homeserver:** checkout `/opt/tervon-tcp-bridge-m6-android-ingress`, `work/m6-android-ingress`, `SOURCE_HEAD=96eab1466b4d88e7a84d048a62e29fb62c41fc60`. Отдельная release worktree `/opt/tervon/Developed/v102-route-validation` на `96c136702d10b6fb9a16aa2893f6b978e8b14e86`; оба overlay выполнили `--check` и `--apply` без ошибок.

**Обнаруженный дефект тестового пакета:** `gofmt` сообщил `stat policy_stream_test.go: no such file or directory`; файл `integrations/android-ingress/runtime-routes/policy_stream_test.go` существовал в GitHub, но отсутствовал в `EXTRA` установщика `apply-to-acl-source.py`. Тест `TestV102ExistingStreamKeepsBackendAfterAtomicRouteSwap` **не вошёл** в запущенный Go test. Это **FAIL полноты проверки**, а не ошибка компиляции действующего кандидата.

**Результаты первого прогона (точный `SOURCE_HEAD=96eab146...`):**
- `go test -count=1 ./...`: `TEST_RC=0`; шесть Go packages PASS, но без `policy_stream_test.go`.
- `go test -race -count=1 ./...`: `RACE_RC=0`; шесть Go packages PASS, но без `policy_stream_test.go`.
- `go vet ./...`: `VET_RC=0`.
- `go build -o /opt/tervon/Developed/v102-route-validation/bridge-v102.candidate ./cmd/bridge`: `BUILD_RC=0`.
- Само `gofmt` завершилось ошибкой: её не остановила `set -uo pipefail` (без `set -e`). Поэтому **общий Step 2 PASS не присваивается**, даже при нулевых кодах остальных команд.
- Вывод пользователя: `LIVE_DOCKER_CHANGED=NO`, `LIVE_XRAY_CHANGED=NO`, `ANDROID_CHANGED=NO`. Фактическое состояние служб заново этим прогоном не исследовалось.

**Исправление:** в `integrations/android-ingress/runtime-routes/apply-to-acl-source.py` список `EXTRA` расширен ровно на `policy_stream_test.go`. Commit исходника `2828b8fe12a0bb5b309ff24526ebec61c80eea1f`. Иных изменений кода, transport, ACL, API, Docker или Android нет.

**Следующее контрольное испытание:** использовать уже созданную изолированную `/opt/tervon/Developed/v102-route-validation`, которая содержит ранее staged Go v1.0.2. Проверить совпадение исходных `policy.go` и `policy_test.go` с GitHub, отсутствие `policy_stream_test.go` и перенос **только** недостающего тестового файла из текущего checkout GitHub в изолированное `prototype/`. Запустить `gofmt`, прицельный `go test -run '^TestV102ExistingStreamKeepsBackendAfterAtomicRouteSwap$' -count=1 -v`, затем полный `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go build`; фиксировать каждый код. Без результата **`STEP2=IN_PROGRESS`**, `HOT_ROUTE_RUNTIME_PASS=NOT_CONFIRMED`, `LIVE_DEPLOY=NO`.

## 2026-11-22 — Шаг 2: выяснение причины прерванного повторного прогона

**Источник:** вывод последующей исключительно read-only диагностики, присланный оператором. Указанные ниже сведения описывают реальное состояние отдельной worktree, а не предположение о runtime.

- Исходный checkout: `work/m6-android-ingress`; HEAD `087ebfee9ad99ec5ae2e6624452fdb9c06eb842c`.
- Изолированная worktree `/opt/tervon/Developed/v102-route-validation`: релизный HEAD `96c136702d10b6fb9a16aa2893f6b978e8b14e86`.
- `policy.go=EXISTS`, `policy_test.go=EXISTS`, `policy_stream_test.go=MISSING`.
- `cmp` исходников `policy.go`/`policy_test.go` GitHub против изолированного дерева: оба `DIFFERENT`. Проверка `gofmt -l` не вывела файлов, то есть staged файлы уже отформатированы.
- Предыдущий прогон содержал `cmp ... || exit 1` непосредственно в операторском shell: несовпадение могло прервать интерактивную оболочку/SSH-консоль. **Причина закрытия консоли не доказана системным журналом**, но неуспешный `cmp` подтверждён. Из-за предыдущего `gofmt` прямое побайтовое сравнение с неформатированным репозиторием было некорректным критерием эквивалентности. Это дефект операторской инструкции, а не подтверждённая ошибка Go runtime.
- **Рабочие службы не проверялись и не изменялись в этой read-only диагностике**.

**Корректировка следующей проверки:** все команды, которые могут завершиться ошибкой, запускать в дочерней оболочке (скобки, subshell); не использовать `exit` в интерактивной SSH-оболочке. Текущие `policy.go` и `policy_test.go` не переписывать и не повторять уже завершённые overlay. Сравнить **нормализованные gofmt-версии** исходных файлов с staged (например, `gofmt -d` или `gofmt -s` с выводом в `cmp`), затем перенести только отсутствующий `policy_stream_test.go`, выполнить прицельный live-stream и полный test/race/vet/build с отдельными RC. Если нормализованное сравнение обнаружит семантическое расхождение — блокировать копирование и разобраться; не объявлять PASS.

**Текущий статус:** `STEP2=IN_PROGRESS`, `SOURCE_ONLY`, `NEW_TEST_MISSING_FROM_STAGE`, `E2E_ROUTE_SWAP=NOT_TESTED`, `RELEASE=NO`.

## 2026-11-22 — Шаг 2: повторная проверка с полным комплектом тестов — PASS

**Источник результата:** дословный вывод команды Homeserver, предоставленный оператором. Данная запись заменяет прежний открытый статус шага 2, но не удаляет историю предыдущей неудачной проверки.

**Проверенный GitHub source HEAD:** `087ebfee9ad99ec5ae2e6624452fdb9c06eb842c`, ветка `work/m6-android-ingress`. **База отдельного тестового release worktree:** `96c136702d10b6fb9a16aa2893f6b978e8b14e86`. Испытан **наложенный v1.0.2 исходник** в `/opt/tervon/Developed/v102-route-validation/prototype`, а не исходники рабочей службы.

**Порядок и доказательства:**
1. Проверены два независимых HEAD, значения совпали с ожидаемыми.
2. Выполнено нормализованное сравнение Go-кода: `policy.go=NORMALIZED_MATCH`; `policy_test.go=NORMALIZED_MATCH`. Это подтверждает, что прошлый прямой `cmp` расходился из-за форматирования, а не выявил доказанный семантический дефект.
3. Недостающий `policy_stream_test.go` перенесён **только** в изолированное дерево и обработан `gofmt`.
4. Прицельная проверка `go test -count=1 -timeout=120s -run '^TestV102ExistingStreamKeepsBackendAfterAtomicRouteSwap$' -v .`: `--- PASS: TestV102ExistingStreamKeepsBackendAfterAtomicRouteSwap (0.46s)`; `STREAM_RC=0`. Сценарий проверяет TLS/mTLS HTTP/2 → Bridge → два TCP-backend: старый поток остаётся на A, новый после атомарной замены попадает на B.
5. `go test -count=1 -timeout=180s ./...`: **6 Go packages `ok`**, `TEST_RC=0` (корневой bridge, cmd/bridge, framing, jitter, padding, telemetry).
6. `go test -race -count=1 -timeout=240s ./...`: **6 Go packages `ok`**, `RACE_RC=0`.
7. `go vet ./...`: без диагностики ошибок, `VET_RC=0`.
8. `go build -o /opt/tervon/Developed/v102-route-validation/bridge-v102.candidate ./cmd/bridge`: `BUILD_RC=0`.
9. После дочерней оболочки присутствует `SSH_CONSOLE=OPEN`: операторский сеанс не прерван.

**Итог:** `STEP2_SOURCE_OVERLAY=PASS`, `STEP2_STREAM_SWITCH=PASS`, `STEP2_TEST=PASS`, `STEP2_RACE=PASS`, `STEP2_VET=PASS`, `STEP2_BUILD=PASS`. **Шаг 2 закрыт как изолированный SOURCE + TEST PASS** на приведённых HEAD; после сохранения настоящей записи отдельным документационным commit. Это **НЕ** deployed/integration PASS, **НЕ** доказательство работы TCP VPN на Android. Нет фактической проверки SIGHUP live-config reload на контейнере, сквозной синхронизации API→routes→discovery, соответствия Xray и пользовательского интернета. `RUNTIME_CUTOVER=NOT_DONE`, `ANDROID_E2E=NOT_PASS`, `RELEASE_V1_0_2=NOT_RELEASED`.

**Изменённые исходники шага 2 (в GitHub до испытаний):**
- `integrations/android-ingress/runtime-routes/policy.go`
- `integrations/android-ingress/runtime-routes/policy_test.go`
- `integrations/android-ingress/runtime-routes/policy_stream_test.go`
- `integrations/android-ingress/runtime-routes/apply-to-acl-source.py`

**Ограничения источников:** `SOURCE_HEAD` фиксирует commit GitHub с source-only overlay. Изолированная worktree остаётся на Git commit релиза v1.0.1 с локальными незакоммиченными наложенными файлами; её содержимое проверено нормализованным сравнением указанных файлов и тестами, **но отдельный Git commit этой worktree не создан**. Не утверждать, что реально работающие серверы обновлены.

**Следующий утверждённый этап 3:** интеграция проверенного snapshot с 3x-ui inbound API, фактическим Xray и HTTPS discovery. Прежде чем вносить изменения: сверить актуальный HEAD, зарегистрированные контракты discovery/registrar и правила безопасного обновления. Не запускать и не менять работающие службы без отдельного согласованного внедрения.

## 2026-11-22 — Этап 3: первичная ревизия контрактов перед интеграцией

**Статус:** `READ_ONLY_SOURCE_AUDIT`; `STAGE3_IMPLEMENTATION=NOT_STARTED`; `STAGE3_PASS=NO`. Исследовано актуальное дерево GitHub после подтверждения PASS этапа 2.

**Просмотренные GitHub файлы:** `integrations/android-ingress/xui/cmd/m6-auto-registrar/main.go`, `integrations/android-ingress/xui/enrollment_http.go`, `integrations/android-ingress/xui/client.go`, `integrations/android-ingress/xui/inbounds.go`.

**Фактические точки интеграции:**
1. `cmd/m6-auto-registrar/main.go`, `configuredTCPRoutes(root)` (строка ~112) сейчас читает `config/server.json` и возвращает карту `backend TCP-port → service`. В `main` (строка ~186) результат устанавливается при инициализации, не привязан к live API.
2. `enrollment_http.go`, `Handler.TCPRoutes` (строка ~29) хранит карту портов, а GET discovery отдаёт `tcp_routes` из этой карты (~89). Встроенного versioned update для одновременной смены routes + discovery ещё нет.
3. `xui/inbounds.go` реализует только проверенное чтение `GET /panel/api/inbounds/list/slim`; этот метод **не** включён в исполнение регистратора, не сверяет actual Xray и не обновляет Decapsulator.
4. `xui/client.go` содержит прежний read-only client API, который не заменяет inbound API.

**Критический инвариант для этапа 3:** опубликованные `tcp_routes` должны соответствовать **фактически применённой** политике Decapsulator, а не только данным панели. API без подтверждённого backend в Xray или без подтверждения применения в ядре не даёт права рекламировать порт как доступный. При несогласованности/недоступности данных не открывать произвольный backend и не заявлять `CONNECTED`.

**Следующая работа:** точная ревизия клиентского/серверного lifecycle, затем минимальный механизм публикации согласованного snapshot. Порядок безопасного обновления, валидатор live Xray, срок актуальности API и подтверждение смены политики требуется реализовать и проверить тестами в рамках этапа 3. Рабочие службы, секреты и Android не менялись.

## 2026-11-22 — Этап 3: включение Remnawave и нормализация маршрутов (SOURCE ONLY)

**Уточнение пользователя:** в задаче интеграции нельзя забывать Remnawave API. Поддержка Remnawave относится к общему контракту v1.0.2; в каскаде достаточно одного Decapsulator на главной ноде, но установка Decapsulator на любую другую ноду **не ограничивается**. Android выбирает VPN-ноду по подписке и проверяет Bridge на её домене; без данных/подтверждения Bridge обход DPI не включается. Это не согласие на перезапуск или изменение служб.

**Сверенный GitHub HEAD перед изменениями:** `4db61f2b0cbd07e32ab7a371b54c67396d8e6562`. Предшествующий stage 2 PASS документирован в `5abd1cfe85c493c6286eb3e6852dec5fe6313eab`.

**Публичные официальные источники Remnawave (сверены 2026-11-22):**
- API: `https://docs.rw/api/` (показывает Remnawave API v3.4.7); доступны read-only `GET /api/config-profiles/inbounds`, `GET /api/config-profiles/{uuid}/inbounds`, `GET /api/config-profiles/{uuid}/computed-config`. Эти endpoints показывают, что можно получать профиль и inbound, но **не доказывают**, что все inbound списка активированы на конкретной ноде.
- `https://docs.rw/learn-en/nodes/`: нода выбирает **один Config Profile**, а в его пределах могут быть включены только выбранные inbound; Node является фактическим Xray endpoint, панель сама не выполняет Xray traffic.
- `https://github.com/remnawave/python-sdk/blob/production/remnawave/controllers/config_profiles.py`: подтверждает API пути просмотра профилей и inbound. `https://github.com/remnawave/python-sdk/blob/production/remnawave/__init__.py`: Bearer token используется как заголовок Authorization.
- **Ограничение:** фактическая схема Node UUID → выбранный профиль → активные inbound/фактически применённая Node конфигурация ещё не зафиксирована по реальному API JSON. Нельзя приравнивать `GET /api/config-profiles/inbounds` к активным портам ноды и нельзя пока объявлять Remnawave API adapter готовым. До сверки модели конкретной ноды оператору не предлагать подключать токен/панель в production.

**Новая общая архитектурная граница:** `3x-ui API` или `Remnawave API` → данные **только выбранной ноды** → сравнение с эффективными inbound Xray → строгие разрешённые маршруты → атомарная ACL+routes политика ядра → подтверждение применения → discovery публикация того же поколения. Одна версия таблицы должна использоваться сервером и discovery; нельзя рекламировать маршрут, который ядро не подтвердило. Исчезновение API отдельно диагностируется; не считать его приказом удалить маршруты.

**Новые SOURCE файлы:**
- `integrations/android-ingress/xui/route_reconcile.go` — независимая `ReconcileInbounds` принимает нормализованные `PanelInbound` (provider предварительно фильтрует по выбранной ноде) и `EffectiveInbound` (локально проверяемая актуальная конфигурация Xray). Совпадение порта+протокола+TCP-транспорта обязательно; disabled, non-TCP, неизвестные, дублированные, некорректные backend исключаются/отклоняются. Возвращаемые `VerifiedRoute` **ещё НЕ разрешено автоматически публиковать**: сначала узел проверяет backend locality и получает подтверждение атомарного применения новой политики ядром. В модуле нет hardcode серверных адресов/портов, изменения API/ACL/Docker отсутствуют.
- `integrations/android-ingress/xui/route_reconcile_test.go` — модель 3x-ui с enabled/disabled TCP и XHTTP; Remnawave модель *уже отфильтрованного активного inbound выбранной ноды*; негативные тесты UDP, несовпадающих протоколов/транспортов, дублей, несовпадения backend, некорректного сервиса.

**Коммиты исходника:** `204f86db95590ed26369197f75df7db8b92bb504`, `4e9460091e940c5338c1e310aeaa652f78679f5a`.

**Статусы:** `STAGE3_RECONCILE_SOURCE=COMMITTED`; `STAGE3_RECONCILE_GO_TEST=NOT_RUN`; `REMNAWAVE_NODE_API_SCHEMA=UNVERIFIED`; `REMNAWAVE_REAL_ADAPTER=NOT_IMPLEMENTED`; `XRAY_LIVE_CONFIG_READER=NOT_CONNECTED`; `CORE_ACK_AND_DISCOVERY=NOT_CONNECTED`; `STAGE3_PASS=NO`; `RELEASE_V1_0_2=NO`. Утверждение, что эти файлы завершили интеграцию 3x-ui или Remnawave, запрещено.

**Следующая проверка:** на точном source HEAD выполнить `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` для `integrations/android-ingress/xui`. При любых FAIL исправить этот пакет, не переходить к API endpoint integration. Не затрагивать живые службы.

## 2026-11-22 — Этап 3: проверка общего согласования 3x-ui / Remnawave на Homeserver

**Источник:** пользовательский полный терминальный вывод Homeserver; не CI и не предположение. **Точно проверенный source HEAD:** `a99bc49f1a45ea73285b49e8cbdac79d2b373998`; рабочая ветка `work/m6-android-ingress`. Перед испытанием успешно выполнен `git pull --ff-only origin work/m6-android-ingress`: `Fast-forward` от `087ebfe` до `a99bc49`. Проверка tracked-изменений и правильной ветки пройдена, иначе операторский subshell остановился бы.

**Проверенные файлы:** `integrations/android-ingress/xui/route_reconcile.go` и `route_reconcile_test.go` вместе с прочими существующими файлами Go-модуля `integrations/android-ingress/xui`.

| Контроль | Подтверждённый результат |
|---|---|
| `go test -count=1 ./...` | `TEST_RC=0`: все 6 Go packages `ok` |
| `go test -race -count=1 ./...` | `RACE_RC=0`: все 6 Go packages `ok` |
| `go vet ./...` | `VET_RC=0`, отсутствие диагностического вывода |
| Безопасность интерактивного сеанса | `SSH_CONSOLE=OPEN` |

**Вывод:** `STAGE3_RECONCILE_SOURCE_TEST=PASS`, `STAGE3_RECONCILE_RACE=PASS`, `STAGE3_RECONCILE_VET=PASS` **исключительно на commit `a99bc49...`**. Включены тестовые модели чтения источников 3x-ui и предварительно отфильтрованных данных Remnawave для выбранной ноды, но это **не означает наличие работающего Remnawave API-адаптера**.

**Оставшиеся задачи этапа 3:** реальный Remnawave API Node→Profile→active inbound adapter, фактическое получение effective Xray, согласование атомарного ACL+Routes ядра с discovery и подтверждённой генерацией маршрутов. `3XUI_LIVE_ROUTE_SYNC=NOT_IMPLEMENTED`, `REMNAWAVE_REAL_API=NOT_TESTED`, `CORE_DISCOVERY_SNAPSHOT_ACK=NOT_IMPLEMENTED`, `ANDROID_E2E=NOT_PASS`, `STAGE3_OVERALL=IN_PROGRESS`, `RELEASE_V1_0_2=NO`. Тестовая команда не включала развёртывания или перезапуска служб; отдельная инструментальная проверка неизменности служб не выполнялась.

**Следующий шаг:** дальнейшая ревизия именно фактического API Remnawave/назначения ноды и реализация изолированного provider adapter. Для внедрения требуется отдельный PASS сквозной связи с ядром. Не менять работающие службы до отдельного разрешения.

## 2026-11-22 — Этап 3: публичная сверка Remnawave Node API (после Go PASS)

**Тип проверки:** чтение официальной публичной API-спецификации и документации; **не вызов пользовательской панели**.

**Подтверждено официальной документацией** (`https://docs.rw/api/`, `https://docs.rw/learn-en/nodes/`, просмотрено 2026-11-22):
- `GET /api/nodes/{uuid}` — чтение записи конкретной ноды.
- `GET /api/config-profiles/{uuid}/inbounds` — чтение inbound выбранного Config Profile.
- По документации Node использует один Config Profile; внутри него включаются выбранные inbound. Поэтому включённые inbound на Node нельзя приравнивать ни к полному списку inbound всех профилей, ни ко всему профилю без дополнительной фильтрации.
- Список API-контроллеров также содержит `GET /api/config-profiles/{uuid}/computed-config`; наличие endpoint подтверждено, но его формат/соответствие фактически запущенному Node Xray **не подтверждены**.
- Реальное соответствие полей JSON ответа Node, поля `activeInbounds`, версии Remnawave и схемы авторизации должно быть проверено на конкретной совместимой версии либо по точной модели официального SDK/схемы, прежде чем подключать реальные автоматические маршруты.

**Техническое решение для этапа 3:** при подключении Remnawave необходимо сначала прочитать **конкретную ноду**, установить назначенный ей профиль и активные inbound, затем сопоставить их с effective Xray; только после атомарного подтверждения ядром публиковать соответствующие маршруты через discovery. `NODE_PROFILE_MAPPING=DESIGN_CONFIRMED`; `NODE_JSON_SCHEMA=NOT_VALIDATED`; `API_ADAPTER_IMPLEMENTATION=NOT_DONE`; `PRODUCTION=UNCHANGED`.

**Операционные ограничения:** никаких запросов к приватной панели, изменений API токенов, контейнеров, действующих VPN сервисов или Android не выполнялось. Эта запись — продолжение обязательного педантичного журнала, а не PASS Remnawave.

## 2026-11-22 — Этап 3: Remnawave read-only Node/Profile API-адаптер

**GitHub HEAD перед работой:** `84a6a31e6fbccc1731b867508d820307eedf16c6`, ветка `work/m6-android-ingress`. Никаких изменений состояния пользователя/серверных служб в данной операции не выполнялось.

**Сверена схема официального Remnawave Python SDK (ветка `production`):**
- `remnawave/models/nodes.py`: `NodeResponseDto.configProfile`, `activeConfigProfileUuid`, `activeInbounds` — список `InboundsDto`, а `isDisabled` — явный признак отключения Node.
- `remnawave/models/internal_squads.py`: `InboundsDto.uuid`, `profileUuid`, `port`, `type`, `network`.
- `remnawave/models/config_profiles.py`: `InboundDto.uuid`, `profileUuid`, `port`, `type`, `network`, `rawInbound`.
- `remnawave/controllers/nodes.py`: GET `/nodes/{uuid}`; `remnawave/controllers/config_profiles.py`: GET `/config-profiles/{uuid}/inbounds`. Префикс HTTP API — `/api`. Источник: публичный GitHub API `remnawave/python-sdk`, `production` на 2026-11-22.
- Контракт API-ответов предполагает оболочку `response`. Подтверждение фактической версии конкретного сервера остаётся `PENDING`.

**Новые файлы:**
- `integrations/android-ingress/xui/remnawave.go`: read-only `RemnawaveClient` и `SelectedNodeInbounds(ctx,nodeUUID)`. Через HTTPS + Bearer читает конкретную Node, проверяет UUID, отключение, назначенный Config Profile, затем список inbound этого профиля; выбирает **только** UUID в `activeInbounds`. Некорректные/изменившиеся/дублированные UUID, порт, тип, transport приводят к явному отказу. Не включает inbound из других профилей или неактивные. Ошибки безопасные: не раскрываются токен и приватный адрес API. Ключ хранится в отдельном файле и не копируется в GitHub.
- `integrations/android-ingress/xui/remnawave_test.go`: HTTPS mock, активный/неактивный inbound, disabled node, отсутствие профиля, неверный UUID, изменённый порт, ошибочный JSON ответ и отсутствие токена в ошибках.

**Точные commits:** `be8807e81eca43dc8bf14e4514813143c23d77d2` (adapter), `5397e0c3a2fc982b430d8a9ece1839ef97787968` (tests).

**Тесты и доказательная граница:** изолированный прототип адаптера до переноса в GitHub проверен локально с Go 1.23.2 на mock HTTPS: `go test -race -count=1 ./...` → `ok example.com/check 1.108s`; `go vet ./...` → 0. Исполнялся эквивалентный исходник с временным `PanelInbound` stub и лимитом; это **не** проверка полного Go-модуля GitHub и **не** E2E Remnawave. Проверка вновь опубликованного кода на Homeserver ещё НЕ выполнена; прошлый stage 3 PASS на `a99bc49...` не переносится.

**Ограничения:** API-адаптер возвращает `PanelInbound`, НЕ создаёт `VerifiedRoute` без `ReconcileInbounds`, НЕ обновляет ядро и discovery; отсутствует подтверждение фактически применённой Xray-конфигурации. Возможна смена выбранного профиля между двумя GET, так что перед публикацией нужен контроль неизменности generation/повторная верификация Node. Не замещать этим чтением проверку реального Xray. Вариант Remnawave каскада не запрещает установку Decapsulator на другие Node.

**Статус:** `REMNAWAVE_API_ADAPTER=SOURCE_COMMITTED`, `MOCK_TEST_ISOLATED=PASS`, `GITHUB_MODULE_TEST=PENDING`, `LIVE_REMNAWAVE_API=NOT_TESTED`, `CORE_RUNTIME_ACK=NOT_CONNECTED`, `DISCOVERY_GENERATION=NOT_CONNECTED`, `STAGE3=IN_PROGRESS`, `v1.0.2=NOT_RELEASED`.

**Следующий шаг:** запуск на Homeserver Go test/race/vet с точным HEAD и сопоставление результатов с мок-контрактом. Не выполнять деплой, не вызывать приватную Remnawave панель, не трогать Docker/Xray/Android.

## 2026-11-22 — Этап 3: проверка Remnawave API-адаптера на Homeserver — PASS

**Источник результата:** полный вывод команд от оператора Homeserver. **Ветка:** `work/m6-android-ingress`; `git pull --ff-only` выполнил `Fast-forward` `a99bc49..5b65aa1`. **Точный испытанный SOURCE_HEAD:** `5b65aa1affb68d6535091ef6c57a5fd129af657a`. Каталог Go-модуля: `integrations/android-ingress/xui`.

| Испытание | Доказательство из вывода | Итог |
| --- | --- | --- |
| `go test -count=1 -v -run '^TestRemnawave' .` | `TestRemnawaveSelectedNodeFiltersInactive` PASS; `TestRemnawaveRejectsInconsistentNode` PASS с подкейсами disabled/unassigned/missing-selected-inbound/changed-port/malformed-profile; `TestRemnawaveDoesNotExposeAPISecret` PASS; `REMNAWAVE_RC=0` | **PASS** |
| `go test -count=1 ./...` | все 6 packages `ok`; `TEST_RC=0` | **PASS** |
| `go test -race -count=1 ./...` | все 6 packages `ok`; `RACE_RC=0` | **PASS** |
| `go vet ./...` | без diagnostic messages; `VET_RC=0` | **PASS** |
| Безопасность терминальной оболочки | `SSH_CONSOLE=OPEN` | **PASS** |

**Область PASS:** read-only HTTPS-mock Remnawave Node → выбранный Config Profile → активные inbound, обработка негативных условий, компиляция/тесты/race/vet текущего Go-модуля на указанном HEAD. Код и mock подтвердили логику фильтрации, но **не** доступность пользовательской Remnawave панели, **не** реальный JSON её версии и **не** работающую динамическую маршрутизацию.

**Не закрыто:** `REMNAWAVE_REAL_API=NOT_TESTED`, `NODE_TWO_GET_CONSISTENCY=NOT_IMPLEMENTED`, `EFFECTIVE_XRAY_CHECK=NOT_CONNECTED`, `CORE_POLICY_ACK=NOT_CONNECTED`, `DISCOVERY_GENERATION=NOT_CONNECTED`, `ANDROID_E2E=NOT_PASS`; `STAGE3=IN_PROGRESS`, `v1.0.2=NOT_RELEASED`. Операторская проверка не содержала деплоя/перезапуска; автономного аудита текущего состояния служб не проводилось.

**Следующий шаг:** локально, только в исходниках, предотвратить гонку изменения Remnawave Node между чтением её профиля и чтением inbound (повторная проверка идентичности выбранной ноды, активного профиля и набора выбранных UUID). Добавить негативные HTTP mock-тесты именно этого случая, затем запросить полный Go test/race/vet на новом точном HEAD. Обновлять журнал на каждом изменении, не переносить текущий PASS на новый commit.

## 2026-11-22 — Этап 3: защита от изменения Remnawave Node между HTTP GET

**Отправная точка:** после документированного `REMNAWAVE_RC=0`, `TEST_RC=0`, `RACE_RC=0`, `VET_RC=0` на HEAD `5b65aa1affb68d6535091ef6c57a5fd129af657a`, результат записан в документацию commit `d200450c343d61ee30bbd8428e4ba19eb0603c28`. Этот PASS не переносится на новый код.

**Выявленная ситуация:** Remnawave API-адаптер последовательно делает `GET /api/nodes/{uuid}`, затем `GET /api/config-profiles/{uuid}/inbounds`. Между запросами оператор панели может переназначить ноде Config Profile, выключить ноду или изменить её выбранные inbound. Без повторной проверки адаптер мог вернуть логически устаревший набор.

**Исправление источников:** `integrations/android-ingress/xui/remnawave.go`: в конце `SelectedNodeInbounds` повторно считывается `GET /api/nodes/{uuid}`; функция `remnaNodeSelectionEqual` проверяет UUID ноды, enabled-state, UUID назначенного профиля и полный список активных inbound (UUID, profileUUID, порт, type, network) без зависимости от их порядка. Несовпадение/повторение/пустая структура вызывает отказ без публикации маршрутов. Ошибка не раскрывает API endpoint/token.
**Тесты:** `integrations/android-ingress/xui/remnawave_test.go`: mock TLS-панель возвращает различные данные на первом и втором чтении Node; сценарии disabled, removed inbound, changed profile, changed port должны отклоняться; тест требует ровно 2 чтения Node. Существующие тесты являются регрессией для стабильного ответа.

**Коммиты:** `a083f58729eb192fd6076a0f07aa81d073735e0b` (код), `7f58e8975f6456eeb251be1d71cd25d39f4fc897` (тесты).

**Границы:** повторная проверка уменьшает окно TOCTOU, но не является транзакционным snapshot API и не доказывает неизменность панели после последнего GET. Надёжное runtime применение требует отдельной проверки effective Xray, atomic policy ACK и согласованного discovery. **Этот пакет не развёрнут** и на Homeserver после его изменения ещё не тестировался.

**Статус:** `NODE_RECHECK_SOURCE=COMMITTED`, `NODE_RECHECK_TESTS=COMMITTED`, `NEW_HEAD_TEST=NOT_RUN`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

**Следующий проверочный шаг:** на Homeserver при чистой `work/m6-android-ingress` выполнить `git pull --ff-only`; затем `go test -count=1 -run '^TestRemnawave' -v .`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` с независимыми кодами завершения. Все действия в subshell; работающие службы не затрагивать.

## 2026-11-22 — Этап 3: Homeserver PASS повторной проверки Remnawave Node

**Источник:** полный вывод Homeserver от оператора; не CI. **Ветка:** `work/m6-android-ingress`. Успешный `git pull --ff-only origin work/m6-android-ingress` с `Fast-forward` `5b65aa1..da7f8e2`. **Точный тестируемый SOURCE_HEAD:** `da7f8e2141218fce407fb5b88ff97773686a019c`.

**Результаты по программным проверкам:**
- `go test -count=1 -v -run '^TestRemnawave' .`: `REMNAWAVE_RC=0`; PASS `TestRemnawaveSelectedNodeFiltersInactive`, `TestRemnawaveRejectsInconsistentNode` (disabled/unassigned/missing-selected-inbound/changed-port/malformed-profile), `TestRemnawaveDoesNotExposeAPISecret`, `TestRemnawaveRejectsNodeSelectionDriftBetweenGETs` (disabled/removed-inbound/changed-profile/changed-port).
- `go test -count=1 ./...`: `TEST_RC=0`, все шесть пакетов Go-модуля `integrations/android-ingress/xui` — `ok`.
- `go test -race -count=1 ./...`: `RACE_RC=0`, все шесть пакетов — `ok`.
- `go vet ./...`: `VET_RC=0`, диагностических предупреждений нет.
- Интерактивный сеанс остался доступен: `SSH_CONSOLE=OPEN`.

**Вывод:** `REMNAWAVE_NODE_RECHECK_TEST=PASS`, `REMNAWAVE_NODE_RECHECK_RACE=PASS`, `REMNAWAVE_NODE_RECHECK_VET=PASS` **только на зафиксированном SHA**. Изоляционные HTTP mock-тесты не доказывают работу с реальной Remnawave панелью. Второе чтение Node сокращает риск TOCTOU, но не является атомарной транзакцией между двумя HTTP GET. Перезапусков и изменений работающих служб в команде нет; их состояние отдельно не обследовано.

**Не завершено:** реальная интеграция Xray effective inbound, синхронизация политики ядра и подтверждение поколения, discovery, Android dataplane; `STAGE3=IN_PROGRESS`, `REMNAWAVE_LIVE_API=NOT_TESTED`, `CORE_ACK=NOT_IMPLEMENTED`, `ANDROID_E2E=NOT_PASS`, `V1.0.2_RELEASE=NO`.

**Дальше внутри этапа 3:** проверка актуального фактически применённого Xray inbound перед публикацией маршрута. Сначала изучить существующий валидатор и способ чтения Xray config, не переписывать проверенные механизмы и не затрагивать сервисы.
