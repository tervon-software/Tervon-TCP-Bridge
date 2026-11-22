# Tervon TCP Bridge v1.0.2 — журнал разработки

> **ИСТОРИЧЕСКИЙ ЖУРНАЛ, НЕ РАБОЧИЙ ПЛАН.** Записи ниже отражают состояние *на момент соответствующего этапа* и могут содержать уже устранённые FAIL, PENDING, старые адреса/схемы и опровергнутые предположения. Не использовать их как актуальные команды или текущую архитектуру. Действующий план: [`docs/DEVELOPMENT_PLAN.md`](DEVELOPMENT_PLAN.md). Проверенный реестр M6: [`docs/M6_AUDIT_2026-11-22.md`](M6_AUDIT_2026-11-22.md). Удаление событий из журнала уничтожило бы доказательства последовательности; отменённые выводы исправляются в действующем плане и в новых датированных записях.


**Дата открытия:** 2026-11-22. **Рабочая ветка:** `work/m6-android-ingress`. **Исходный HEAD:** `af72b959e227a849e72f9bc1f99ee20ecae94fec`. **База проверенного релиза:** Go Bridge v1.0.1, исходный commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86`.

**Назначение журнала:** фиксировать каждый выполненный шаг, изменённые файлы, основание решения, точные проверки и результаты. Статусы `СОГЛАСОВАНО`, `РЕАЛИЗОВАНО`, `ТЕСТИРОВАНО`, `PASS`, `ВЫПУЩЕНО` различаются. Нельзя переносить PASS другого коммита на новый.


> Android-записи исходного смешанного журнала перенесены в [Android/V1_0_2_ANDROID_EXTRACT.md](../Android/V1_0_2_ANDROID_EXTRACT.md), без пересмотра фактов.

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

## 2026-11-22 — Этап 3: кандидаты Xray configured inbound (исходники, не runtime PASS)

**После PASS Remnawave TOCTOU на `da7f8e2141218fce407fb5b88ff97773686a019c`** добавлена новая связанная часть этапа 3. PASS предыдущего commit не переносится.

**Сверенный предыдущий механизм:** `integrations/android-ingress/check_live_xray_bridge_routes.py`, `test_check_live_xray_bridge_routes.py`. Текущий Python audit сопоставляет порты по двум JSON-файлам, исключает некоторые non-TCP типы, но **не подтверждает**, что запущенный Xray применил именно этот файл. Повторно реализовывать тот же сервис управления не требуется; новый Go-парсер нужен для передачи разрешённых кандидатов в общий Go reconcile и отличается обязательным доверенным resolver.

**Новые файлы и их коммиты:**
- `integrations/android-ingress/xui/xray_configured.go` — `ParseXrayConfiguredInbounds`, строго ограниченный JSON-парсер Xray `inbounds`; нормализует TCP transport и возвращает `EffectiveInbound` **только как CONFIGURED кандидатов** после вызова внедрённого доверенного `XrayBackendResolver` для каждого TCP-port. Сам Xray JSON не может задать произвольный backend address. Ошибки неизвестного транспорта/дублированных портов/неверных адресов не превращаются в разрешение. Commit `230c8c1cc29a706db670322949deee2b4a6da1cd`.
- `integrations/android-ingress/xui/xray_configured_test.go` — позитивная связь `ParseXrayConfiguredInbounds → ReconcileInbounds` с 3x-ui-подобными данными; negative JSON/port/duplicate/backend resolver tests. Commit `c0a3f1ffcde82d96cf02137b973ff833d1782804`.

**Доказательная граница:** `XRAY_CONFIGURED_CANDIDATES=SOURCE_COMMITTED`; `GO_TEST_NEW_HEAD=NOT_RUN`; `XRAY_PROCESS_APPLIED_CONFIG=NOT_VERIFIED`; `CORE_ROUTE_APPLY_ACK=NOT_IMPLEMENTED`; `DISCOVERY_GEN_MATCH=NOT_IMPLEMENTED`; `STAGE3=IN_PROGRESS`, `RELEASE=NO`. `EffectiveInbound` в типах описывает предполагаемое эффективное состояние, но функция-источник здесь является только файлом. Никакие кандидаты не могут рекламироваться Android без отдельного подтверждения работающего Xray и атомарного применённого policy ядра.

**Следующая проверка:** выполнить прицельные `go test -count=1 -run '^TestParseXrayConfiguredInbounds' -v .`, полный `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` на точном новом HEAD. Только после подтверждённого PASS продолжить runtime-proof и согласованную публикацию.

## 2026-11-22 — Этап 3: Homeserver PASS Go-парсера Xray configured inbound

**Источник доказательств:** полный вывод консоли Homeserver оператора; не CI, не результат чтения реального запущенного Xray.

**Проверенный GitHub commit/HEAD:** `10aa483916b7f88591986009b30630dee6b2c127`; ветка `work/m6-android-ingress`. Оператор выполнил `git pull --ff-only`, подтверждён `Fast-forward` `da7f8e2..10aa483`. Каталог тестов: `integrations/android-ingress/xui`.

**Изменения проверенного кандидата:** `xray_configured.go` (config-only Xray inbound parser с доверенным resolver), `xray_configured_test.go` (валидный TCP и XHTTP, отклонение дубликатов, отсутствующего/некорректного порта, дополнительных JSON-документов и неверного backend).

| Контроль | Фактический вывод / код | Статус |
| --- | --- | --- |
| `go test -count=1 -run '^TestParseXrayConfiguredInbounds' -v .` | Оба теста PASS, включая 5 отрицательных подпроверок; `XRAY_RC=0` | **PASS** |
| `go test -count=1 ./...` | 6 Go packages `ok`; `TEST_RC=0` | **PASS** |
| `go test -race -count=1 ./...` | 6 Go packages `ok`; `RACE_RC=0` | **PASS** |
| `go vet ./...` | Нет диагностических сообщений; `VET_RC=0` | **PASS** |
| Безопасность SSH | `SSH_CONSOLE=OPEN` | **PASS** |

**Точный смысл PASS:** парсер Xray JSON и тестовая связка с `ReconcileInbounds` работают в изолированном Go-модуле на указанном commit. **НЕТ** подтверждения, что текущий работающий Xray применил этот JSON или принимает TCP на любом определённом порту. **НЕТ** подключения к атомарной политике Decapsulator, подтверждения поколения от сервера, динамического discovery, live Remnawave API или успешного интернет-трафика Android. `XRAY_CONFIGURED_PARSER=PASS`, `XRAY_RUNTIME_ACTIVE=NOT_VERIFIED`, `STEP3_OVERALL=IN_PROGRESS`, `V1.0.2_RELEASE=NO`. Команда не содержала вмешательства в работающие службы; их актуальный runtime-статус этим испытанием отдельно не проверялся.

**Следующий шаг:** ревизия имеющегося доверенного механизма проверки реально запущенного Xray и связки Core ACK → Discovery. Не объявлять файл JSON равнозначным процессному runtime, не повторять уже прошедшие Go-проверки без новой причины и не менять действующие службы.

## 2026-11-22 — Этап 3: аудит доказательства live Xray и ACK перед публикацией

**Исходный HEAD после записи результатов парсера:** `b31a86c7d237e00e3892c54ee10ad76885272a26`. Выполнена только GitHub-ревизия файлов (без записи исходников или изменения служб):
- `integrations/android-ingress/check_live_xray_bridge_routes.py` и `test_check_live_xray_bridge_routes.py` — read-only сверка конфигурационных JSON и портов, не доказательство применённой Xray конфигурации;
- `integrations/android-ingress/xui/acl_docker_confirm.go` — ныне работающий `DockerACLConfirmer` проверяет строго старый образ `tervon-bridge-android-ingress:m6-runtime-acl` + label `org.tervon.bridge.hot_acl=m6-v1`, выполняет Docker SIGHUP и ждёт лог `M6_ACL_RELOAD=PASS digest=...`; это **не** ACK route-generation v1.0.2, использовать напрямую нельзя;
- `integrations/android-ingress/xui/cmd/m6-auto-registrar/main.go` — `tcp_routes` получаются при запуске из JSON, а не из применённого поколения ядра;
- `integrations/android-ingress/xui/enrollment_http.go` — discovery отдаёт хранимую `TCPRoutes` без подтверждения поколения.

**Вывод:** текущая последовательность `panel API → Xray JSON → policy overlay` не доказывает, что Xray применил конфигурацию и доступен для подключения из сетевого пространства Decapsulator. Имена localhost/host.docker.internal в тестах не подтверждают реальный сетевой путь. Не считать TCP connect без подтверждения протокола доказательством работоспособного VPN, однако это полезный отдельный проверяемый слой.

**Неразрешённые входные данные:** каким способом на конкретной ноде подтверждается применённая Xray конфигурация (достоверный runtime API, источник применённого конфигурационного поколения или безопасная проверка фактических listener/process), и каким механизмом кандидат ядра v1.0.2 подтверждает уже применённое поколение перед отдачей discovery. Только сравнение с файлом и подтверждение старого `M6_ACL_RELOAD` недостаточны.

**Статусы:** `LIVE_XRAY_PROOF=BLOCKED_PENDING_READ_ONLY_EVIDENCE`; `V102_ROUTE_ACK=NOT_IMPLEMENTED`; `DISCOVERY_ACK_SYNC=NOT_IMPLEMENTED`; `STAGE3=IN_PROGRESS`. До получения данных новые исходники для live-публикации не создавать и действующий Docker не менять. Следующий точечный read-only шаг — посмотреть фактические командные строки Xray/связанный конфиг и открытые TCP-listener, без вывода ключей или содержимого конфигов, без повторных Go-тестов. После этого подготовить согласованную схему runtime-валидатора.

## 2026-11-22 — Этап 3: read-only обследование реально слушающих Xray портов Homeserver

**Источник:** точный текст терминальной диагностики Homeserver, предоставленный оператором. Измерение состояния носит точечный характер (снимок, не непрерывный monitoring).

- `pgrep -x xray`: без вывода. Причина согласуется с тем, что исполняемый процесс по `ss` имеет другое имя (`xray-linux-amd6`), а не точное `xray`. Это не доказательство отсутствия Xray.
- `ss -H -lntp` отобразил TCP LISTEN процесса `xray-linux-amd6`, `pid=1253673`: `*:55535`, `*:55636`, `*:30000`; loopback `127.0.0.1:10811`, `127.0.0.1:11111`, `127.0.0.1:54321` (два FD на последнем). `xray-exporter`, pid 3977 слушает `*:9550`, НЕ считать exporter VPN inbound.
- По предыдущему read-only API snapshot панели: `55535` — VLESS TCP, `55636` — VLESS XHTTP, `30000` — Trojan TCP **по конфигурации панели**, но `ss` не раскрывает VPN-протокол; нельзя приравнивать порт к протоколу без дополнительной сверки конфигурации.
- `systemctl show x-ui`: `ActiveState=active`, `SubState=running`, `MainPID=2091`. PID x-ui отличается от PID Xray, ожидаемо для панели, запускающей дочерний процесс, но фактическая связь parent-child ещё не проверялась.
- `docker ps --filter name=android-ingress`: `android-ingress-android-bridge-decapsulator-1`, image `tervon-bridge-android-ingress:m6-runtime-acl`, status `Up 14 hours`. Это действующий **старый ACL-only** runtime, а не кандидат v1.0.2.
- Команда закончилась `READ_ONLY_CHECK=COMPLETE` и `SSH_CONSOLE=OPEN`; не запускала контейнеры, не изменяла конфигураций и не компилировала Android.

**Вывод и границы:** `XRAY_LISTENER_PRESENT=YES` для трёх портов; `XRAY_PROTOCOL_APPLIED=NOT_CONFIRMED_BY_SS`; `DOCKER_TO_XRAY_BACKEND_REACHABLE=NOT_TESTED`; `VPN_DATA_TRANSFER=NOT_TESTED`; `V102_ROUTING_ACK=NOT_IMPLEMENTED`. Исходная жалоба на отсутствие интернета/ошибку VLESS TCP остаётся открытой. Отсутствие точного `pgrep -x xray` объясняет отсутствие информации в блоках PID/exe, а не отсутствие работающего Xray.

**Следующее действие (только read-only):** установить фактическую сеть существующего Bridge Docker и текущие backend endpoints **из разрешённых маршрутов Bridge**, затем отдельно оценить TCP-доступность backend без смены Docker/Xray/X-ui/Android. Не выводить полный `server.json`, ключи, токены или конфиги Xray. Текущий PID и порты — snapshot, не hardcode для программного компонента. `STEP3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — Этап 3: read-only проверка реальных Docker backend routes и network namespace

**Источник:** точный терминальный вывод Homeserver, присланный оператором после ранее документированного `XRAY_LISTENER_PRESENT=YES` (TCP :55535/:55636/:30000).

- Прочитан действующий `/etc/tervon/bridge-android/config/server.json` без вывода секретов: `MODE=server`, `SERVICE=vpn BACKEND=host.docker.internal:30000`, `SERVICE=vpn-xhttp BACKEND=host.docker.internal:55636`. Эти номера взяты из действующих конфигураций, **не** hardcode исходников.
- `docker inspect` действующего `android-ingress-android-bridge-decapsulator-1`: `NETWORK_MODE=android-ingress_default`, `NETWORK=android-ingress_default`, `IP=172.20.0.2`, `GATEWAY=172.20.0.1`. Результат не устанавливает, в какой IP адрес разрешается `host.docker.internal` **из сетевого пространства контейнера**; DNS/hosts и TCP reachability ещё не подтверждены.
- `ps -p 1253673 -o pid,ppid,comm,stat`: `PID=1253673 PPID=2091 COMM=xray-linux-amd6 STAT=Sl`, подтверждён ребёнок x-ui (MainPID 2091, ранее считан через systemd).
- Ранее наблюдавшийся VLESS TCP :55535 слушает Xray, но **не опубликован** в двух действующих backend маршрутах Bridge: `XRAY_TCP_LISTENER=YES`, `BRIDGE_ROUTE_FOR_55535=NO`. Это объясняет отсутствие разрешённого mapping для локальной активации Bridge VLESS TCP, **но не доказывает** причину потери интернета на XHTTP/Trojan.
- `SSH_CONSOLE=OPEN`. Команда не изменяла службы и не требовала доступа к содержимому Xray secrets.

**Следующая проверка (только read-only):** прочитать mapping `host.docker.internal` внутри существующего контейнера и без изменения службы проверить TCP connect **из того же network namespace**, до реальных backend :30000 и :55636. Если контейнер не имеет Python/sh/curl, допускается использование `nsenter -t <ContainerInitPID> -n` с Python из host; запрет на пересборку образа, установку пакетов, изменённые Docker configs и включение/отключение VPN остаётся. Проверка TCP connect доказывает исключительно достижимость порта, а не корректность VPN, а отсутствие `host.docker.internal` в host `/etc/hosts` не равно отсутствию этого имени в контейнере. Результат фиксировать по каждой паре отдельно, не предполагать PASS.

**Статусы:** `DOCKER_NETWORK_DISCOVERED=YES`, `BACKEND_ROUTES_READ=YES`, `VLESS_TCP_MISSING_ROUTE=YES`, `DOCKER_TO_XRAY_TCP_CONNECT=NOT_TESTED`, `ANDROID_INTERNET=NOT_PASS`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — Этап 3: успешная проверка backend TCP из network namespace Docker; уточнение обязательной API-маршрутизации

**Источник:** вывод Homeserver от оператора по уже согласованной read-only команде `nsenter`. Runtime-измерение ограничено TCP connect, без отправки VPN-трафика.

- `CONTAINER_PID=1756814` — init PID запущенного Android ingress Decapsulator на момент проверки.
- В `/proc/1756814/root/etc/hosts` имя `host.docker.internal` разрешается в `172.17.0.1` (это отличается от Docker gateway `172.20.0.1`, но реально настроенный host alias TCP-доступен из данного namespace).
- `SERVICE=vpn PORT=30000 TCP_CONNECT=YES`.
- `SERVICE=vpn-xhttp PORT=55636 TCP_CONNECT=YES`.
- `TCP_ONLY=YES`, `VPN_PROTOCOL_VERIFIED=NO`, `SSH_CONSOLE=OPEN`.
- Из этого **НЕ** следует VPN dataplane PASS, подтверждение входящего протокола Xray, корректный mTLS от Android или интернет. Не приписывать текущему Docker динамический API sync: действующий образ `m6-runtime-acl` продолжает читать только статический `server.json`.

**Повторное требование оператора:** процесс Decapsulator в Docker должен ориентироваться на актуальные порты inbound через API панели и автоматически знать актуальные маршруты, без ручной синхронизации `server.json`. Поддержка **двух** провайдеров — 3x-ui и Remnawave; порт пользовательского VPN и внешний HTTPS Bridge являются разными понятиями. Уже утверждённый v1.0.2 контракт: API provider → selected node active inbounds → verified applied Xray/listener & backend reachability → atomic `RuntimePolicy` ACL+Routes → generation ACK ядра → discovery publication только применённого поколения. При ошибке API/несовпадении не публиковать новые неподтверждённые маршруты; текущие открытые TCP sessions не рвать.

**Техническая граница доверия:** процесс Docker обязан **использовать** актуальную маршрутизацию, но административные Bearer токены 3x-ui/Remnawave внутри Docker не обязательны и не должны переноситься без отдельной необходимости. Существующий root-owned серверный API adapter/registrar может получать API данные и передавать применяемый проверенный snapshot ядру через имеющийся механизм reload/ACK. Для v1.0.2 старый `DockerACLConfirmer` с `M6_ACL_RELOAD=PASS` не подтверждает route snapshot; требуется `M6_POLICY_RELOAD=PASS`/generation digest именно маршрутов и ACL.

**Итог:** `DOCKER_BACKEND_CONNECT_30000=PASS`; `DOCKER_BACKEND_CONNECT_55636=PASS`; `VLESS_TCP_55535_ROUTE=ABSENT_CURRENT_RUNTIME`; `DYNAMIC_API_ROUTE_SYNC=NOT_IMPLEMENTED_ON_RUNNING_DOCKER`; `VPN_E2E=NOT_TESTED`; `STAGE3=IN_PROGRESS`; `V102_RELEASE=NO`. Команды не изменяли Docker, Xray, 3x-ui, Remnawave или Android.

**Следующий согласованный участок этапа 3:** добиться передачи подтверждённой конфигурации из read-only adapter обоих провайдеров в `RuntimePolicy` кандидата v1.0.2, добавить отдельный route generation ACK и связать discovery с этим ACK; сначала изолированные тесты и документация, затем отдельное разрешение для замены работающего Docker. Без API токенов в Android, без hardcode backend портов.

## 2026-11-22 — Этап 3: confirmed-route discovery snapshot, исходники и тесты (PENDING)

**Входной commit перед изменениями:** `30954af94f74baf14bde748392df70747f88eab8`. Предыдущий подтверждённый проход `go test/race/vet` парсера Xray относится к `10aa483916b7f88591986009b30630dee6b2c127`, не переносится на новый исходник.

**Прочитаны и сверены контракты:** `integrations/android-ingress/xui/enrollment_http.go`, `enrollment_http_test.go`, `cmd/m6-auto-registrar/main.go`, `runtime-routes/policy.go` и `runtime-routes/apply-to-acl-source.py`. Действующий M6 регистратор по-прежнему использует статическую `TCPRoutes`, а `M6_ACL_RELOAD=PASS` доказывает только ACL. В ядре-кандидате v1.0.2 уже предусмотрен `M6_POLICY_RELOAD=PASS digest=...` для полного ACL+routes.

**Данный изолированный подшаг:**
- `integrations/android-ingress/xui/confirmed_routes.go` — новый `ConfirmedRouteStore` с read-only копией snapshot, полной проверкой цифровых портов/service, mutex для сериализации ACK. `PublishAfterCoreACK` принимает **только** от доверенного control-plane callback подтверждение конкретного полного policy digest; только после его успешного результата разрешает новую таблицу. При исходно неподтверждённой таблице `Snapshot` запрещает публикацию. При неуспехе / тайм-ауте **после попытки** обновления ядра состояние переводится в fail-closed для новых discovery: результат обновления мог быть неоднозначным; существующие TCP-потоки ядра этот код не трогает. Исходник commits `d52a6c7dc9158a17f0e7ba76b7d7b605ea27820d`, `e9ed9f2132970ecd7e1a5af6f0a492148a3b661f`.
- `integrations/android-ingress/xui/enrollment_http.go` — исключительно **optional** `ConfirmedRoutes *ConfirmedRouteStore`. Если не задано — прежнее M6 discovery без изменения. Если задано — HTTP GET `/.well-known/tervon-bridge` возвращает 503 до ACK, затем только подтверждённые `tcp_routes` и дополнительный `policy_generation`; `Cache-Control: no-store` сохранён. Commit `d3cfed09fc4928b43d85f2caf293378d1f9ea147`.
- `integrations/android-ingress/xui/confirmed_routes_test.go` — подтверждение, defensive copies, отказ на некорректном port/digest, fail-closed при ambiguous ACK, параллельные читатели. Commits `30ff016a88e609649f6b07db6e38d491b95f4d31`, `f8bcf7408ebe84b49200decd45b513f671e168fd`.
- `integrations/android-ingress/xui/enrollment_http_test.go` — тест 503 до ACK, публикация после callback ACK и запрет legacy map в v1.0.2, 503 после failed callback. Commits `cba79aa4b0dcaac5c427df34aa385569ac667816`, `2a295b27b0048ad2745b4832806f647323ca4673`.

**Безопасность и ограничения:** digest в коде является **входным параметром**; код `ConfirmedRouteStore` сам не выводит его из эффективного `RuntimePolicy` и не подтверждает его подпись или реальную установку. Только доверенный callback и одно последовательное управление writer может обеспечить соответствие checksum. Прямое использование публичного callback, передающего успех без проверки ядра, ЗАПРЕЩЕНО. Отсутствует интеграция фактических API→Xray→candidate policy→Docker ACK→registrar. В частности этот commit **ещё не переключает** `m6-auto-registrar` на `ConfirmedRouteStore`, и не запускает SIGHUP. Дополнительное поле `policy_generation` используется только после опции `ConfirmedRoutes`. Существующие Android и Docker не затронуты.

**Входные проверки / подтверждение:** опубликованные исходники в GitHub `SOURCE_COMMITTED`, но `GO_TEST=NOT_RUN_ON_HOMESERVER`, `RACE=NOT_RUN`, `VET=NOT_RUN`, `RUNTIME_ACK=NOT_CONNECTED`, `DISCOVERY_V102_LIVE=NOT_CONNECTED`, `STEP3_OVERALL=IN_PROGRESS`, `V102_RELEASE=NO`.

**Следующая проверка на Homeserver:** на точном новом HEAD `go test -count=1 -run '^(TestConfirmedRouteStore|TestDiscoveryV102)' -v .`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`. Если FAIL — исправить только текущий подшаг. Не затрагивать работающие службы.

## 2026-11-22 — Этап 3: Homeserver PASS подтверждённой таблицы и HTTPS discovery

**Источник:** полный операторский терминальный вывод Homeserver. **Точно испытанный GitHub SOURCE_HEAD:** `a1f682f5bedee509368db961296be0ddfd6992e7`; ветка `work/m6-android-ingress`; `git pull --ff-only` выполнен как `Fast-forward 10aa483..a1f682f`. Go-модуль: `integrations/android-ingress/xui`.

**Проверенные изменения:** `confirmed_routes.go`, `confirmed_routes_test.go`, `enrollment_http.go`, `enrollment_http_test.go`; сохранён опциональный legacy static discovery и проверено правило публикации только после доверенного callback ACK.

| Проверка | Доказательство | Статус |
| --- | --- | --- |
| `go test -count=1 -v -run '^(TestConfirmedRouteStore\|TestDiscoveryV102)' .` | PASS `TestConfirmedRouteStoreRequiresCoreACK`, `TestConfirmedRouteStoreRejectsBadGenerationWithoutMutation`, `TestConfirmedRouteStoreConcurrentReaders`, `TestDiscoveryV102OnlyAdvertisesAcknowledgedGeneration`; `ROUTES_RC=0` | **PASS** |
| `go test -count=1 ./...` | Все 6 Go packages `ok`, `TEST_RC=0` | **PASS** |
| `go test -race -count=1 ./...` | Все 6 Go packages `ok`, `RACE_RC=0` | **PASS** |
| `go vet ./...` | `VET_RC=0`, предупреждений нет | **PASS** |
| Состояние SSH | `SSH_CONSOLE=OPEN` | **PASS** |

**Точный смысл результата:** `CONFIRMED_ROUTE_SNAPSHOT=TEST_PASS`, `DISCOVERY_FAIL_CLOSED=TEST_PASS`, `DISCOVERY_GENERATION_FIELD=TEST_PASS` **для приведённого HEAD**, с тестовым доверенным callback. Это не подтверждение работающей синхронизации API 3x-ui/Remnawave → Xray → RuntimePolicy → Docker → discovery.

**Ограничения:** реальный проверяющий Docker route ACK ещё не подключён, `ConfirmedRouteStore` не доказывает самостоятельно связь предоставленного digest с запущенным ядром и не синхронизирует его с фактическим Xray. Смена маршрутов в активном контейнере не выполнялась. `REAL_CORE_ACK=NOT_IMPLEMENTED`; `LIVE_DISCOVERY_SYNC=NOT_CONNECTED`; `XRAY_PROTOCOL_E2E=NOT_VERIFIED`; `ANDROID_INTERNET=NOT_PASS`; `STEP3_OVERALL=IN_PROGRESS`; `RELEASE_V1_0_2=NO`.

**Следующий участок этапа 3:** ревизия существующих digest и Docker ACK контрактов, затем минимальное внедрение доверенного подтверждения поколения реального кандидата ядра в контроллер на изолированном стенде. Перед публикацией verify policy bytes/digest и последствия ошибки подтверждения; не менять действующие Docker, Xray, 3x-ui, Remnawave или Android без отдельного согласования.

## 2026-11-22 — Этап 3: digest ACL + routes для будущего реального ACK

**HEAD перед работой:** `ea807fa0d393880699dd36fca09b65c91600abad` на `work/m6-android-ingress`. Прочитаны `runtime-routes/apply-to-acl-source.py`, `runtime-routes/policy.go`, `xui/acl_docker_confirm.go`, `xui/acl_docker_confirm_test.go`, `xui/confirmed_routes.go`, `prototype/bridge.go`, `prototype/cmd/bridge/main.go`.

**Доказанный контракт по исходникам:** кандидат Bridge v1.0.2 после успешного RuntimePolicy.Replace сериализует JSON объекта с полями в порядке routes, authorized; вложенные backend имеют экспортируемые Go поля Name и Address; затем вычисляет SHA-256 и выдаёт M6_POLICY_RELOAD=PASS digest=<hex>. Старый DockerACLConfirmer знает только M6_ACL_RELOAD и старый образ, его ACK не подтверждает изменение routes.

**Новые исходники:**
- `integrations/android-ingress/xui/policy_generation.go`, commit `ad7c5a7da31c3010cf5850833144f6c6b7e187d4`: PolicyGenerationDigest формирует SHA-256 того же сериализованного объекта routes + authorized, проверяет service/port/backend, требует непустые ACL и routes.
- `integrations/android-ingress/xui/policy_generation_test.go`, commit `aa2010f048d388def321e56d4dff7568c479df93`: тест независимой Go JSON сериализации структуры эквивалентного bridge.Backend, детерминизма и изменения digest при смене маршрута; негативные проверки.

**Ограничения:** текущий код формирует только ожидаемый digest; он не доказывает ACK ядра, корректность live Xray, применённую конфигурацию или достижимость VPN. В рабочем Docker действует старый образ m6-runtime-acl. Имя нового Docker image, labels и порядок cutover не менялись. Реального Docker confirmation нет, API→core→discovery ещё не подключён.

**Статусы:** POLICY_DIGEST_SOURCE=COMMITTED, POLICY_DIGEST_TEST=PENDING, REAL_CORE_ACK=NOT_CONNECTED, LIVE_XRAY_PROOF=NOT_COMPLETE, STAGE3=IN_PROGRESS, V102_RELEASE=NO. Следующая проверка на Homeserver: go test -count=1 -run '^TestPolicyGenerationDigest' -v ., go test -count=1 ./..., go test -race -count=1 ./..., go vet ./... на новом точном HEAD. Без PASS дальше не переходить; сервисы не затрагивать.

## 2026-11-22 — Этап 3: Homeserver PASS расчёта PolicyGenerationDigest

**Источник:** полный вывод команды Homeserver от оператора. **Точно испытанный SOURCE_HEAD:** `b124a7cf6bdbd6b0d314aceeea3b089254218e96`, ветка `work/m6-android-ingress`. Успешный `git pull --ff-only`, `Fast-forward a1f682f..b124a7c`; исходники `integrations/android-ingress/xui/policy_generation.go`, `policy_generation_test.go`. Запуск в subshell сохранил интерактивный SSH.

**Подтверждённые проверки:**
- `go test -count=1 -v -run '^TestPolicyGenerationDigest' .`: PASS `TestPolicyGenerationDigestMatchesCoreJSONContract`, PASS `TestPolicyGenerationDigestRejectsInvalidInput`; `DIGEST_RC=0`.
- `go test -count=1 ./...`: 6 Go packages `ok`; `TEST_RC=0`.
- `go test -race -count=1 ./...`: 6 Go packages `ok`; `RACE_RC=0`.
- `go vet ./...`: диагностических сообщений нет; `VET_RC=0`.
- `SSH_CONSOLE=OPEN`.

**Итог в узкой области:** `POLICY_DIGEST_HELPER_TEST=PASS`, `GO_MODULE_TEST=PASS`, `GO_RACE=PASS`, `GO_VET=PASS` **только на проверенном commit**. Тест проверяет равенство SHA-256 с независимой моделью сериализации структур по формату исходника Go Bridge; не доказывает реальный SIGHUP, Docker ACK, операционную транзакцию или применённый runtime policy.

**Не завершено:** `M6_POLICY_RELOAD_RUNTIME_ACK=NOT_TESTED`, `V102_DOCKER_CANDIDATE=NOT_DEPLOYED`, `API_TO_POLICY_SYNC=NOT_CONNECTED`, `XRAY_RUNTIME_PROOF=NOT_COMPLETE`, `DISCOVERY_LIVE=NOT_CONNECTED`, `ANDROID_E2E=NOT_PASS`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. Работающие службы в проверочной команде не изменялись.

**Следующее утверждённое направление:** изолированная проверка настоящего `M6_POLICY_RELOAD=PASS` от ядра v1.0.2 и точного digest; не тестировать реальную смену конфигурации в производственном Docker и не менять текущую версию, образ или метки.

## 2026-11-22 — Этап 3: тест реального M6_POLICY_RELOAD ACK в отдельном процессе — SOURCE_COMMITTED, HOST_TEST_PENDING

**Ветка / исходный HEAD:** `work/m6-android-ingress`, `1c0ccb9f7a7d94913a262568478680d6cf858406`. Сверены `docs/DEVELOPMENT_PLAN.md`, этот журнал, `prototype/cmd/bridge/main.go`, `runtime-acl/apply-to-release-source.py`, `runtime-routes/apply-to-acl-source.py`, `runtime-routes/policy_stream_test.go`, `xui/policy_generation.go`. До этого завершён и задокументирован Homeserver PASS PolicyGenerationDigest на `b124a7cf6bdbd6b0d314aceeea3b089254218e96`.

**Что добавлено без влияния на работающие службы:**
- `integrations/android-ingress/runtime-routes/policy_ack_integration_test.go`, commit `6575a5a8c145677e03b0886a0a8c66e4c8172845`. Linux Go integration test `TestV102RealProcessSIGHUPPolicyGenerationACK` собирает отдельный Bridge из изолированного Go checkout, создаёт временные тестовые ECDSA cert/key/CA, только loopback listener 127.0.0.1:0, отдельный config с двумя сменяемыми backend адресами и валидным ACL. Отправляет SIGHUP **только запущенному тестом процессу**; требует фактический `M6_POLICY_RELOAD=PASS digest=<expected sha256>`, затем меняет защищённое поле `max_streams` и ожидает `M6_POLICY_RELOAD=FAIL fixed config changed` без нового PASS, после восстановления конфигурации ожидает новый PASS от того же процесса. По завершении теста subprocess останавливается, исходная система не используется. Backend адреса в тесте — loopback заглушки, а не производственные маршруты.
- `integrations/android-ingress/runtime-routes/apply-to-acl-source.py`, commit `5c10564742bc75fa5ee234a94f287db7f0a34b31`: дополнен перенос только этого `_test.go` в `prototype/cmd/bridge` отдельной staged release копии при прежних жёстких guards; не копирует тест в live Docker и не меняет существующий runtime код ядра.

**Граница проверки:** тест подготовлен, но пока **не запускался на Homeserver** против полного staged v1.0.1 release+M6 ACL+v1.0.2 route overlay. Команда Go `go build` выполняется внутри теста только для isolated binary, не для APK и не для производственного Docker. Ожидаемый digest вычисляется из JSON routes + authorized, независимого от текста лога. Нет операции Docker restart/kill и обращения к панели API.

**Статус:** `REAL_PROCESS_ACK_TEST=SOURCE_COMMITTED`, `REAL_PROCESS_ACK_HOMESERVER=NOT_TESTED`, `STAGE3=IN_PROGRESS`, `RELEASE_V1_0_2=NO`. Следующий шаг — только на Homeserver: подготовить чистый изолированный release worktree точного commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86`, применить существующие ACL и routes overlay, выполнить `go test -run '^TestV102RealProcessSIGHUPPolicyGenerationACK$' -count=1 -v ./cmd/bridge` и затем suite/vet. Не применять overlay к текущему рабочему Docker или уже изменённой source copy. PASS записать только после реального вывода.

## 2026-11-22 — Этап 3: реальный subprocess ACK — FAIL, локализована ошибка ожидаемого digest в тесте

**Исходный испытанный HEAD:** `313b4aa84b9aef3b432678b1699f114dbae5b9cc`, `work/m6-android-ingress`. Homeserver выполнил `git pull --ff-only`, создал отдельный `/opt/tervon/Developed/v102-policy-ack-validation` worktree точного release commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86`; оба existing overlay `runtime-acl`, `runtime-routes` проверились и применились к изолированной копии (`M6_RUNTIME_ACL_RELEASE_SOURCE=MATCH`, `M6_V102_ROUTE_SOURCE_CHECK=PASS`). Действующие службы не менялись, SSH остался открыт.

**Фактический исход испытания:** `TestV102RealProcessSIGHUPPolicyGenerationACK` **FAIL** после 5.89 s: ожидалось `M6_POLICY_RELOAD=PASS digest=dc148f47d48c9443dd988722f82f35be3054ee6a939508511a30ca0dfffe6c8e`; **реальный отдельный Bridge subprocess выдал** `M6_POLICY_RELOAD=PASS digest=2e72206a05ad287f679805c0f1c50b45cf4b602092757bd878e749ab1a96cdd8`. Лог `http: TLS handshake error ... EOF` связан с TCP readiness probe без TLS и не содержит доказательства VPN-отказа. `ACK_RC=1`, `TEST_RC=1` (общий тест повторил тот же FAIL; прочие пакеты `ok`), `VET_RC=0`, `SSH_CONSOLE=OPEN`.

**Root cause подтверждается сравнением исходников:** тест рассчитывал SHA-256 от `map[string][]map[string]string`; encoding/json сортирует ключи `Address,Name`, хотя ядро после декодирования конфигурации использует `map[string][]bridge.Backend` с Go struct-полями, сериализуемыми в порядке `Name,Address`. Содержание backend одинаково, но байты JSON и digest различаются. Нельзя считать тестовый FAIL отказом ядра от смены маршрута: фактический процесс сообщил PASS, а ожидаемый digest был неправильно сформирован тестом. **Ядро и существующий проверенный `PolicyGenerationDigest` не изменялись.**

**Минимальное исправление, source commit `01b3e6efc695f133d34a68ede23eaaf6f1c51239`:** только `integrations/android-ingress/runtime-routes/policy_ack_integration_test.go`. Тест использует тип `ackBackend struct{Name,Address string}` вместо map для setup + ожидаемого digest, сохраняя тот же SIGHUP/FAIL/PASS сценарий. Скрипты overlay, core runtime и Docker не изменены.

**Статусы:** `REAL_ACK_OLD_TEST=FAIL`, `ROOT_CAUSE=TEST_BACKEND_JSON_FIELD_ORDER`, `FIX_SOURCE=COMMITTED`, `FIX_HOMESERVER_TEST=PENDING`, `STEP3=IN_PROGRESS`, `RELEASE=NO`.

**Следующая проверка:** обновить HEAD на Homeserver и перенести **только исправленный тест** в уже существующее изолированное source tree `/opt/tervon/Developed/v102-policy-ack-validation/prototype/cmd/bridge/policy_ack_integration_test.go` после точной проверки, что старый staged test совпадает с blob `313b4aa...`. Повторно **не** создавать worktree и **не** применять уже применённые overlays. Запустить прицельный тест и общий go test/vet в этом же изолированном дереве, зафиксировать реальные коды; после FAIL не переходить к Docker ACK интеграции.

## 2026-11-22 — Этап 3: реальный SIGHUP Policy ACK изолированного ядра — PASS

**Источник доказательства:** полный вывод команды Homeserver от оператора; **HEAD исходников проекта:** `bf90fbeef95b410a719aee61d8f785245c796992`, ветка `work/m6-android-ingress`. `git pull --ff-only` выполнил `Fast-forward 313b4aa..bf90fbe`. Ранее созданный изолированный worktree `/opt/tervon/Developed/v102-policy-ack-validation` на **release source commit** `96c136702d10b6fb9a16aa2893f6b978e8b14e86` сохранён; проверка `git show OLD:test | cmp - staged test` прошла до замены только тестового файла. `TEST_SOURCE_UPDATED=YES`, `CORE_SOURCE_UNCHANGED=YES`; никакие overlays повторно не применялись.

**Результаты:**
- `go test -count=1 -v -run '^TestV102RealProcessSIGHUPPolicyGenerationACK$' ./cmd/bridge`: `PASS`, `ok tervon-tcp-bridge/cmd/bridge 0.932s`; `ACK_RC=0`. Этот тест запускает настоящий отдельный subprocess Bridge, проверяет полученный `M6_POLICY_RELOAD=PASS digest=<expected>`, отказ `M6_POLICY_RELOAD=FAIL fixed config changed` и повторный успешный reload при восстановлении разрешённой конфигурации. Сценарий проверяется тестом без публикации ключей/реальных backend.
- `go test -count=1 ./...`: все шесть пакетов `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `jitter`, `padding`, `telemetry` имеют `ok`; `TEST_RC=0`.
- `go vet ./...`: сообщений нет, `VET_RC=0`.
- `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`. Команда не запускала сигналы для production Docker / Xray и не выпускала APK.

**Область PASS:** `ISOLATED_REAL_PROCESS_ACK=PASS`, `POLICY_DIGEST_MATCH=PASS`, `FIXED_CONFIG_REJECTED=PASS`, `RECOVERY_RELOAD=PASS`, `GO_TEST=PASS`, `GO_VET=PASS` для **изолированного staged-кандидата** релизного ядра v1.0.1 + M6 ACL overlay + v1.0.2 routes overlay с исправленным тестом; исходник ядра после успешного теста не менялся. Это снимает предыдущий FAIL, вызванный разницей структуры/словаря в тестовом JSON. Результат не равен проверке Docker или панели.

**Остаётся:** `PRODUCTION_DOCKER_POLICY_ACK=NOT_TESTED`, `V102_CONTROLLER_DOCKER_ACK=NOT_CONNECTED`, `API_3XUI_REMNAWAVE_TO_POLICY=NOT_CONNECTED`, `XRAY_APPLIED_CONFIG_VALIDATION=INCOMPLETE`, `LIVE_DISCOVERY_GENERATION=NOT_CONNECTED`, `ANDROID_INTERNET=NOT_PASS`, `STAGE3=IN_PROGRESS`, `V102_RELEASE=NO`. Следующий согласованный подшаг: подготовить доверенный confirmer целевого **кандидата** v1.0.2, сверяющий точный целевой контейнер и именно полную `M6_POLICY_RELOAD=PASS digest=...`, без использования прежнего ACL-only ACK и без действий на действующем Docker. Нельзя считать Docker-кандидат установленным до отдельной проверки.

## 2026-11-22 — Этап 3: доверенный Docker policy ACK confirmer для будущего кандидата — SOURCE_COMMITTED, TEST_PENDING

**Начальный HEAD:** `821836dc127b454a34f5daf7c1bae6edca2877f1`, ветка `work/m6-android-ingress`. Перед правкой сверены актуальный план, этот журнал, старый `xui/acl_docker_confirm.go`, его тесты, `xui/confirmed_routes.go` и генератор `M6_POLICY_RELOAD` в `runtime-routes/apply-to-acl-source.py`. Предыдущий `ISOLATED_REAL_PROCESS_ACK=PASS` доказан Homeserver для изолированной сборки и зафиксирован в журнале (tested source HEAD `bf90fbee...`). Продукционный Docker по-прежнему использует ACL-only образ; никакие сервисы не менялись.

**Реализация без вмешательства в runtime:**
- `integrations/android-ingress/xui/policy_docker_confirm.go`, commit `431f7ac4cb1e6b490b9ebb1ddae4ab101c4ace12`: `DockerPolicyConfirmer` требует **полный 64-символьный Docker container ID**, явно заданные доверенным окружением image + label key/value и точную read-only bind-монтировку host config directory → `/config`, проверяет `Running=true` и `inspect.Id`. Действующий `tervon-bridge-android-ingress:m6-runtime-acl` отклоняется. Не вводятся новые version tags, подписи, имена образов или метки. После `docker kill --signal=HUP` только для проверенного целевого ID читает `docker logs --timestamps --since <timestamp>` и принимает исключительно точную строку `M6_POLICY_RELOAD=PASS digest=<expected SHA256>` с временем не раньше моментa сигнала; игнорирует устаревший ACK, `M6_ACL_RELOAD`, другой digest, строки с дополнительными символами. Не использует Docker restart или пересборку.
- `integrations/android-ingress/xui/policy_docker_confirm_test.go`, commit `d7c1d1ee4f399ba3b996f464d9a7353c07b1680e`: отдельный тестовый mock Docker бинарник в temp PATH; позитивный точный кандидат; отрицательные старый/неверный ACK, legacy image, abbreviated container ID, RW bind; отдельный разбор строк со stale timestamps. Настоящий docker tool в этих тестах не вызывается.

**Граница доверия:** confirmer является **только компонентом, не включённым** в `m6-auto-registrar`; caller обязан вычислить digest из фактического atomically-staged файла, сериализовать write→signal→ACK и сверить выбранную ноду/Xray. Callback доверен только при проверенной идентичности кандидата, разрешённом контейнере и выпуске по утверждённому manifests. Для обновления прежних существующих TCP sessions остаётся `RuntimePolicy` ядра. Нет реального Docker candidate deployment, API sync, Android E2E. Метаданные контейнера могут быть изменены привилегированным оператором — эта проверка не заменяет защиту Docker daemon.

**Статус:** `POLICY_DOCKER_CONFIRMER=SOURCE_COMMITTED`, `MOCK_TEST=NOT_RUN_ON_HOMESERVER`, `RACE=NOT_RUN`, `VET=NOT_RUN`, `LIVE_DOCKER=UNCHANGED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

**Следующий checkpoint:** `go test -count=1 -v -run '^(TestPolicyACKInLogs|TestV102DockerPolicyConfirmer)' .`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` в `integrations/android-ingress/xui` на новом точном HEAD. Любой FAIL исправлять в этом же компоненте; после PASS обновить журнал. Не вызывать данный confirmer на производственном Docker.

## 2026-11-22 — Этап 3: Homeserver PASS Docker Policy ACK Confirmer (mock)

**Источник:** предоставленный оператором полный терминальный вывод. `git pull --ff-only origin work/m6-android-ingress` выполнил `Fast-forward bf90fbe..0e7a80b`. **Точно испытанный SOURCE_HEAD:** `0e7a80bc1d2d246ea254c3f3f6bd79cdaa29e45b`; Go-модуль `integrations/android-ingress/xui`.

**Результаты:**
- Прицельно `go test -count=1 -v -run '^(TestPolicyACKInLogs|TestV102DockerPolicyConfirmer)' .`: `TestPolicyACKInLogsExactPostSignalGeneration` PASS; `TestV102DockerPolicyConfirmerMockExactTarget` PASS; `TestV102DockerPolicyConfirmerRejectsLegacyAndWrongACK` PASS; `ACK_RC=0`.
- `go test -count=1 ./...`: все 6 packages `ok`; `TEST_RC=0`.
- `go test -race -count=1 ./...`: все 6 packages `ok`; `RACE_RC=0`.
- `go vet ./...`: без диагностических сообщений; `VET_RC=0`.
- `SSH_CONSOLE=OPEN`.

**Вывод в пределах проверенного:** `DOCKER_POLICY_ACK_MOCK=PASS`, `FULL_GO_MODULE=PASS`, `RACE=PASS`, `VET=PASS` на указанном SHA. Проверены требование явной идентичности контейнера, проверка времени и точного текста `M6_POLICY_RELOAD=PASS digest=...`, отказ от ACL-only и некорректных mount/образов. Тесты используют подменный Docker executable в временном PATH; **реальный Docker-кандидат не опрашивался, не сигнализировался и не обновлялся**.

**Статусы:** `REAL_DOCKER_POLICY_CONFIRM=NOT_TESTED`; `PANEL_API_TO_CORE=NOT_CONNECTED`; `XRAY_RUNTIME_PROOF=INCOMPLETE`; `LIVE_DISCOVERY_SYNC=NOT_CONNECTED`; `ANDROID_INTERNET=NOT_PASS`; `STEP3_OVERALL=IN_PROGRESS`; `V102_RELEASE=NO`.

**Следующее действие:** сверить уже имеющиеся read-only API адаптеры 3x-ui/Remnawave, `ReconcileInbounds`, `PolicyGenerationDigest`, `ConfirmedRouteStore` и Docker confirmer, чтобы минимально связать всю цепочку в изолированном тесте без новых компонентов для той же функции. При выявлении недостаточной доказанности applied Xray не публиковать маршруты и не изменять активные службы.

## 2026-11-22 — Этап 3: изолированная связка panel → verified applied Xray → policy digest → ACK → discovery (source-only)

**Перед работой:** прочитаны `docs/DEVELOPMENT_PLAN.md`, журнал, HEAD `3c721ef0768a4494442952356287338fcf2b842f` ветки `work/m6-android-ingress`; фактические `xui/inbounds.go`, `remnawave.go`, `route_reconcile.go`, `xray_configured.go`, `policy_generation.go`, `confirmed_routes.go`, `policy_docker_confirm.go` и существующий регистратор. Предыдущий mock Docker policy ACK PASS относится к `0e7a80bc1d2d246ea254c3f3f6bd79cdaa29e45b`.

**Факты ревизии:** 3x-ui `Client.ListInbounds` возвращает redacted panel inbound; Remnawave `SelectedNodeInbounds` возвращает выбранные node-scoped `PanelInbound`; `ReconcileInbounds` проверяет protocol/network/port и TCP transport, но не доказывает что Xray действительно применил JSON. `ParseXrayConfiguredInbounds` явно `CONFIGURED_ONLY`. Нельзя использовать его результат как единственное подтверждение applied Xray для автопубликации.

**Source-only изменения:**
- `xui/confirmed_routes.go`, commit `aa5783d7faaa8707d146a899aa3bbbff51ece32a`: `ConfirmedRouteStore.Invalidate()` закрывает новые discovery, когда API/Xray evidence неактуально; сессии ядра не трогает.
- `xui/route_sync.go`, commit `2faa4179630adbcc8a1763d6d86bcbc842018ffa`: `SyncVerifiedRoutes` получает выбранный panel snapshot через доверенный callback, **обязательное** evidence применённого Xray через отдельный доверенный callback, использует существующий `ReconcileInbounds`, требует полное совпадение активных inbound, исключает неоднозначные service → port, проверяет услуги ACL, строит полный policy candidate и вызывает уже проверенные `PolicyGenerationDigest` и `ConfirmedRouteStore.PublishAfterCoreACK`. ACK callback обязан **атомарно записать точную ACL+Routes конфигурацию, получить подтверждение от одобренного кандидата ядра**; на FAIL публикация закрывается. Адрес backend берётся только из trusted effective runtime callback; никакого hardcode реальных портов/доменов и токенов Android.
- `xui/route_sync_test.go`, commit `673574e1be34058e933fd9d1e67591189f6defa6`: проверка двух разных TCP transport, ожидаемого полного policy digest, невыдачи route при неполном Xray, закрытия discovery при ошибках API/ACK, восстановления после нового подтверждения.

**Строгие ограничения:** это интеграционный интерфейс **с тестовыми callback**, не доказательство работающей API→Docker цепочки. Настоящий runtime applied-Xray evidence provider пока не реализован, а JSON parser не заменяет его. Не подключён production registrar, Docker или автоматический polling; нет записи реальной config, SIGHUP или Android E2E. Предполагается единственный сериализованный writer всей цепочки (панели, reload ACL, публикации); нельзя независимо писать ACL и routes в другой задаче и считать digest согласованным. Доверенный apply callback ещё предстоит связать со staged config и реальным `DockerPolicyConfirmer`. Ложный callback в тесте может вернуть success, поэтому `ACK_TEST_PASS` не означает фактический Docker.

**Статусы:** `ROUTE_SYNC_SOURCE=COMMITTED`, `ROUTE_SYNC_HOMESERVER_TEST=PENDING`, `XRAY_APPLIED_EVIDENCE=NOT_IMPLEMENTED`, `DOCKER_APPLY_SYNC=NOT_CONNECTED`, `LIVE_DISCOVERY=NOT_CONNECTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

**Следующий checkpoint:** `go test -count=1 -v -run '^TestSyncVerifiedRoutes' .`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` на точном следующем HEAD. Не изменять работающие службы и не переходить к следующему подшагу при FAIL.

## 2026-11-22 — Этап 3: Homeserver PASS изолированной Route Sync цепочки

**Доказательство:** полный терминальный вывод оператора. Успешный `git pull --ff-only origin work/m6-android-ingress`, `Fast-forward 0e7a80b..b08814a`. **Точный испытанный GitHub SOURCE_HEAD:** `b08814afdba06f979ab3df774cb4a07e59c00508`, ветка `work/m6-android-ingress`, Go-модуль `integrations/android-ingress/xui`.

**Результаты проверок:** `go test -count=1 -v -run '^TestSyncVerifiedRoutes' .`: PASS `TestSyncVerifiedRoutesRequiresAppliedXrayAndCoreACK`, PASS `TestSyncVerifiedRoutesFailClosedWhenAPIOrACKFails`; `SYNC_RC=0`. Полный `go test -count=1 ./...`: 6 пакетов `ok`, `TEST_RC=0`. `go test -race -count=1 ./...`: 6 пакетов `ok`, `RACE_RC=0`. `go vet ./...`: без предупреждений, `VET_RC=0`. `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Подтверждённое:** `ROUTE_SYNC_CALLBACK_UNIT_TEST=PASS`, `ROUTE_SYNC_FAIL_CLOSED_TEST=PASS`, `XUI_MODULE_GO_TEST=PASS`, `XUI_MODULE_RACE=PASS`, `XUI_MODULE_VET=PASS` **только для указанного HEAD**. Тесты доказали согласование выбранных panel inbound с переданным в mock действующим Xray snapshot, построение ожидаемого full policy digest, публикацию только после успешного test callback ACK, закрытие discovery при неполном Xray и сбоях API/ACK. 

**Не доказано и не реализовано:** источник `AppliedXrayEvidence` настоящего запущенного Xray отсутствует (тест передаёт доверенный mock); provider API → Docker apply/ACK не подключён в реально работающем registrar; автоматический polling не включён; Xray live dataplane / Android E2E не испытаны. Нельзя интерпретировать unit-test PASS как реальное API или Docker подтверждение. `XRAY_RUNTIME_PROOF=NOT_IMPLEMENTED`, `REAL_DOCKER_POLICY_CONFIRM=NOT_TESTED`, `LIVE_DISCOVERY_SYNC=NOT_CONNECTED`, `ANDROID_INTERNET=NOT_PASS`, `STEP3_OVERALL=IN_PROGRESS`, `RELEASE_V1_0_2=NO`.

**Следующий согласованный участок этапа 3:** исследование действующего способа применения Xray конфигурации и проверяемого runtime свидетельства, не доверяя одному panel/API JSON. До такого свидетельства не публиковать обновлённые маршруты. Действующие службы не затрагивать.

## 2026-11-22 — Этап 3: ревизия доказательств реально применённой Xray-конфигурации (READ-ONLY, HOST CHECK PENDING)

**Ветка и HEAD на начало ревизии:** `work/m6-android-ingress`, `a1b057b759d84024d6b6309fc2f823d39f8fbe2e`. Сверены `docs/DEVELOPMENT_PLAN.md`, настоящий журнал, `integrations/android-ingress/xui/route_sync.go`, `xray_configured.go`, `inbounds.go`, `remnawave.go`, `route_reconcile.go`, `integrations/android-ingress/check_live_xray_bridge_routes.py`. Предыдущий тест `SyncVerifiedRoutes` PASS только для доверенного тестового callback на `b08814afdba06f979ab3df774cb4a07e59c00508`.

**Что установлено чтением реальных исходников:** `check_live_xray_bridge_routes.py` читает сохранённый Xray JSON и `server.json`, затем выдаёт `CONFIG_MATCH_ONLY`, не узнаёт фактически применённую ревизию Xray. `ParseXrayConfiguredInbounds` намеренно предоставляет только `CONFIGURED_ONLY`. `ReconcileInbounds` сопоставляет порт/протокол/transport и не доказывает runtime state. `ss`/TCP CONNECT подтверждают listener/socket, но не протокол и не происхождение применённой конфигурации. `SyncVerifiedRoutes` требует реального `AppliedXrayEvidence`, который пока не подключён.

**Наблюдения из ранее выполненных оператором команд:** действующий Xray-процесс `xray-linux-amd6` PID `1253673` с parent `x-ui` PID `2091` на момент снимка; TCP listeners `55535`, `55636`, `30000`. Доступность backend из Docker network namespace для `30000` и `55636` подтверждена TCP CONNECT. Это исторический snapshot, а не неизменяемая идентичность процесса или API доказательство применения конфигурации.

**Следующий read-only checkpoint на Homeserver:** найти текущий PID как дочерний процесс x-ui, проверить `/proc/PID/exe`, аргументы запуска **только в безопасной проекции** (наличие `-config`/`-confdir`, без вывода других параметров), путь конфигурации без публикации содержимого, PID/starttime, владелец TCP listeners; проверить наличие локального Xray API listener, если он вообще настроен. Не читать публично токены, inbound credentials, полный `cmdline` и конфиг. Не делать SIGHUP, не запускать `xray api` write-команды, не перезапускать/не устанавливать пакеты, не менять конфигурацию.

**После проверки:** определить допустимое доказательство applied inbound на основе фактического механизма запуска 3x-ui/Remnawave; до этого `XRAY_APPLIED_EVIDENCE=UNPROVEN`, `LIVE_POLICY_UPDATE=BLOCKED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. Изменений исходников и production нет.

## 2026-11-22 — Этап 3: повторная read-only проверка Xray, устранение дублирования проверок

**Источник:** фактический терминальный вывод Homeserver от оператора в текущем диалоге, без изменения служб. **HEAD до фиксации:** `460f345f77962d62802191ef49a9d6efc4b280bd`; ветка `work/m6-android-ingress`.

**Повторная проверка (НЕ новый этап и НЕ новый PASS applied-конфигурации):** `/proc` показывает `PID=1253673`, `PPID=2091`, `EXE_BASENAME=xray-linux-amd64`, `CONFIG_ARGUMENT_TYPE=CONFIG_FILE_FLAG`, `START_TICKS=51919517`. Отдельный `xray-exporter`: `PID=3977`, `PPID=3947`, `START_TICKS=1608`, `CONFIG_ARGUMENT_TYPE=UNKNOWN`, не является Xray VPN-core. `ss -lntp` повторно подтвердил xray-owned публичные TCP listener `30000`, `55636`, `55535` и локальные `127.0.0.1:10811`, `127.0.0.1:11111`, `127.0.0.1:54321` (два FD на 54321); x-ui слушает `2096` и `5588`. `READ_ONLY_CHECK=COMPLETE`; `SSH_CONSOLE=OPEN`. Полная командная строка и содержимое Xray конфигурации не раскрывались: `CONFIG_CONTENT=NOT_READ`, `SECRETS=NOT_PRINTED`.

**Сопоставление с уже документированными результатами:** Xray PID/PPID и три публичных TCP listener **уже были проверены ранее**, поэтому повторное назначение подобной проверки не было обосновано новым вопросом и расходовало время оператора. Новое уточнение — в аргументах запуска обнаружен `-config`, но **не** прочитан его путь и **не** доказано, что Xray применил именно текущий файл/конкретную ревизию; PID и TCP listeners не являются API-level свидетельством применённого inbound. Предыдущие Docker → Xray TCP CONNECT `30000` и `55636` уже имели PASS; их повторять без конкретного основания нельзя.

**Коррекция последовательности работы:** перед любой новой командой сверять её цель с записями этого журнала и историей выполненных проверок. Не повторять PID, `ss`, TCP reachability и установленный `-config` без конкретной изменившейся гипотезы. Следующий нерешённый вопрос — **способ подтверждения конкретной применённой Xray-конфигурации** после обновления панелью, а не существование процесса и сокетов. Если текущие документы/исходники не раскрывают безопасный механизм, сначала изучить фактический путь конфигурационного обновления 3x-ui и возможные Xray runtime API/статус **без повторных операторских команд**, затем согласовать строго целевую проверку. Никаких изменений production.

**Статус:** `XRAY_PROCESS_AND_LISTENERS=CONFIRMED_AGAIN`; `CONFIG_CLI_FLAG=CONFIRMED`; `XRAY_APPLIED_GENERATION=UNPROVEN`; `LIVE_POLICY_UPDATE=BLOCKED`; `STAGE3=IN_PROGRESS`; `RELEASE=NO`.

## 2026-11-22 — Этап 3: уточнение Xray applied-state — gRPC HandlerService вместо config.json

**Источник решения:** оператор указал на hot-apply 3x-ui через Xray gRPC и невозможность использовать сохранённый `config.json` как доказательство актуальной runtime-конфигурации. Сверены актуальный план и журнал; Подтверждённый GitHub HEAD до записи: `03317e8ea9ba921659ca3c1e4a561b6c97c9a5db` (ветка `work/m6-android-ingress`). **Никаких действий на Homeserver и исходниках Bridge в этом шаге нет.**

**Проверка upstream (22.11.2026):**
- Xray-core API имеет gRPC `HandlerService` и `StatsService`, а также CLI `xray api`. В текущем upstream присутствуют `HandlerService.ListInbounds`, `GetInboundUsers`, `GetInboundUsersCount`; `ListInboundsResponse.Inbounds` использует `[]*core.InboundHandlerConfig`. Источники: https://pkg.go.dev/github.com/xtls/xray-core/app/proxyman/command , https://xtls.github.io/en/config/api.html , https://github.com/XTLS/Xray-core/blob/main/main/commands/all/api/api.go
- В 3x-ui существует `tryHotApply` с `HandlerService`/`RoutingService` без перезапуска для поддерживаемых секций, а неподдерживаемые изменения могут привести к cold restart. Источник: https://git.xiganglive.com/xinyin025/3x-ui/commit/6b16d8c37a82856d7df7ca681320c8c5b1fc606e
- Issue 3x-ui #6677 подтверждает, что `bin/config.json` может отставать после hot-apply. При этом в обсуждении issue отмечено, что после холодного запуска 3x-ui восстанавливает конфигурацию из БД, поэтому вывод о неизбежной потере изменений при рестарте неверен. Источник: https://github.com/MHSanaei/3x-ui/issues/6677
- Наличие `GetInboundUsers`/`Count` и `ListInbounds` **не гарантировано установленной версией** Xray: ранее возникали различия доступных gRPC методов, включая discussion https://github.com/XTLS/Xray-core/discussions/4149.

**Вывод для утверждённого M6:** предпочтительный read-only источник для `AppliedXrayEvidence` — `HandlerService.ListInbounds` работающего Xray, если установленный бинарник/его API фактически поддерживает метод и возвращает необходимые tag/port/protocol/transport. `GetInboundUsers`/`Count` подтверждают состояние пользователей (не полное содержимое и порт inbound) и не заменяют ListInbounds для динамического выбора backend. `StatsService` — косвенная проверка фактического трафика, не достаточная для доказательства поколения конфигурации. Ответ gRPC может содержать чувствительные сведения и должен быть преобразован в redacted projection без публикации пользователей/ключей в журнал.

**Ограничение:** не установлены версия работающего Xray, реальный список включённых gRPC service/method, фактическое содержимое `ListInbounds`, доступность API только локально и возможность проверить, что вызов обслуживает именно дочерний Xray от 3x-ui. Нельзя выдавать upstream возможности за факт о Homeserver. Ни API, ни конфигурация на Homeserver не изменялись. Следующая строго целевая read-only проверка должна отвечать только на вопрос о поддержке `ListInbounds` и безопасных полях результата; не повторять PID/ss/TCP CONNECT и не использовать методы изменения Xray. При отсутствии подтверждения `XRAY_APPLIED_EVIDENCE=UNPROVEN`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 Этап 3: закреплён точный upstream 3x-ui для интеграции

**Уточнение оператора (источник истины для 3x-ui):** используем **https://github.com/MHSanaei/3x-ui**. Не подменять этот проект форками, альтернативными реализациями 3x-ui или предположениями об их API. Это правило касается всех пользователей/установок MHSanaei/3x-ui в проекте, но **не исключает Remnawave как отдельно утверждённый provider**. Установленная на Homeserver версия (tag/commit) пока не установлена; нельзя приравнивать её к HEAD upstream.

**Фактическая read-only сверка GitHub:** `MHSanaei/3x-ui`, default branch `main`, upstream HEAD на момент запроса `5297fd3c8495d80fab97ae511553fa545c69e913`; Bridge HEAD до записи `ddffd1aa0b184830440410d20018b3c8999645a3`, branch `work/m6-android-ingress`. Открыты фактические `internal/xray/api.go`, `internal/xray/hot_diff.go`, `internal/web/service/xray_hot_apply_persist_test.go`. В `internal/xray/api.go` создаются gRPC clients Xray HandlerService и RoutingService. Upstream тест `TestTryHotApplyWritesConfigFile` явно проверяет запись `config.json` даже в empty-diff hot-apply; поэтому **нельзя объявлять, что MHSanaei/3x-ui всегда оставляет config.json без обновления**. Runtime state gRPC остаётся предпочтительным направлением проверки, но без подтверждения конкретной установленной версии/активного API не считается проверенным.

**Следствие для M6:** дальнейший разбор hot-apply и API endpoints выполнять только по официальному `MHSanaei/3x-ui` и фактической версии установки. Не повторять уже выполненные PID/PPID/ss/TCP CONNECT проверки; не изменять продакшен. `XRAY_LIVE_GRPC_API=NOT_VERIFIED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. Этот commit изменяет только документацию.

## 2026-11-22 — M6 этап 3: подтверждён путь получения applied Xray inbound через официальный 3x-ui / Xray-core (SOURCE REVIEW)

**Требование оператора:** работать исключительно с официальным https://github.com/MHSanaei/3x-ui, не терять прежний контекст, документировать каждый шаг и реальный итог. **Bridge HEAD до этой записи:** `662b2d6d78146dce60c0ee071a2da65ef7591a60`; рабочая ветка `work/m6-android-ingress`. Повторных команд Homeserver и изменений действующих служб нет.

**Исходники сверены по конкретным ревизиям (не по произвольному форку):** `MHSanaei/3x-ui` upstream main `5297fd3c8495d80fab97ae511553fa545c69e913`; его `go.mod` закрепляет `github.com/xtls/xray-core v1.260327.1-0.20261109052709-676eaf66f8f0` (Go `1.27.4`). Открыты `internal/web/service/xray.go` (особенно `RestartXray` / `tryHotApply` / `applyXrayHotDiff` / `persistHotConfig`), `internal/xray/api.go`, `internal/xray/process.go`, `internal/xray/hot_diff.go`, pinned Xray-core `app/proxyman/command/command.proto`, `command.go`, `core/config.proto`, `app/proxyman/config.proto`. Ссылки: https://github.com/MHSanaei/3x-ui/blob/5297fd3c8495d80fab97ae511553fa545c69e913/internal/web/service/xray.go ; https://github.com/XTLS/Xray-core/blob/676eaf66f8f0/app/proxyman/command/command.go .

**Проверенные выводы:**
1. Официальный 3x-ui при изменениях, которые можно применить в живом ядре, вызывает Xray gRPC HandlerService `RemoveInbound`/`AddInbound`/`AlterInbound` и RoutingService, а если diff нельзя безопасно применить — может перейти к холодному перезапуску. Это **не всегда hot-apply**.
2. При успешном hot-apply 3x-ui выполняет `process.SetConfig(effective)` и `persistHotConfig(process)`. `PersistConfig()` атомарно записывает JSON; ошибка записывается в журнал и не означает, что уже применённый runtime откатился. Поэтому файл **не доказательство**, но утверждать «3x-ui никогда не сохраняет файл после hot-apply» также неверно.
3. Pinned Xray-core protobuf `HandlerService.ListInbounds(ListInboundsRequest{isOnlyTags:false})` возвращает `ListInboundsResponse.Inbounds`; серверный обработчик в `command.go` действительно берёт `s.ihm.ListHandlers(ctx)` и из **живых** handler извлекает `Tag`, `ReceiverSettings`, `ProxySettings`. При `isOnlyTags:true` возвращаются **только tags**, что недостаточно для Bridge.
4. `core.InboundHandlerConfig` содержит `tag`, `receiver_settings`, `proxy_settings`; `app/proxyman.ReceiverConfig` внутри typed receiver settings содержит `port_list`, `listen`, `stream_settings`. Значит возможна безопасная проекция **tag/port/protocol/network**, но необходимо корректно десериализовать protobuf TypedMessage, доказать единственность порта/listener и отбрасывать поля credentials/user.
5. `GetInboundUsers` и `GetInboundUsersCount` есть в этой pinned ревизии, но показывают пользователей/число и **не заменяют** проверку port/transport через `ListInbounds`. `StatsService` лишь косвенная проверка трафика.
6. Модуль Bridge `integrations/android-ingress/xui/go.mod` пока `go 1.23.0` и **не содержит внешних Go-зависимостей**. Прямое подключение всей версии `xray-core`, требующей Go `1.27.4`, нарушило бы существующее правило сохранять зависимости и базовую сборку без согласования. Никаких зависимостей, способов сборки и Go-версий самовольно не изменяли.

**Что не доказано:** установленная версия Xray в Homeserver может отличаться от pinned upstream; не подтверждены `ListInbounds` и API endpoint у запущенного экземпляра. По ранее подтверждённым snapshot уже знаем PID/PPID, открытые порты, `-config` и Docker TCP CONNECT; **не повторять** их. Даже `ListInbounds` возвращает runtime конфигурацию handler, но само по себе не доказывает успешный VPN end-to-end traffic.

**Граница следующего решения:** перед реализацией клиента gRPC нужно согласовать способ получения typed protobuf без изменения существующего `go.mod`/Go toolchain: (A) отдельно проверенный read-only helper, использующий поддерживаемый системный gRPC клиент, или (B) узкая согласованная зависимость `grpc-go` + protoc-generated schemas совместимой версии. Не вводить новую библиотеку или бинарник самостоятельно. Первый тест должен быть read-only, с доступом только к loopback API и выводом redacted tag/port/protocol/transport; `Add/Remove/Alter` не вызывать. Сначала доказать поддержку метода у фактически установленной версии.

**Статус:** `UPSTREAM_HANDLER_LIST_INBOUNDS=SOURCE_CONFIRMED`, `HOT_APPLY_BEHAVIOR=SOURCE_CONFIRMED`, `INSTALLED_XRAY_LIST_INBOUNDS=NOT_TESTED`, `APPLIED_XRAY_EVIDENCE=NOT_CONNECTED`, `PRODUCTION_SERVICES=UNCHANGED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 этап 3: снят риск новой gRPC-зависимости при первичной read-only диагностике

**Дополнительная сверка того же pinned Xray-core `676eaf66f8f0`:** открыты `main/commands/all/api/api.go`, `main/commands/all/api/inbounds_list.go` и `main/commands/all/api/shared.go`. Источник: https://github.com/XTLS/Xray-core/blob/676eaf66f8f0/main/commands/all/api/inbounds_list.go .

**Проверенный факт по исходнику:** штатный CLI уже имеет `xray api lsi --server=127.0.0.1:<api-port>` (`cmdListInbounds`); его `executeListInbounds` вызывает `HandlerService.ListInbounds`, причём `isOnlyTags` по умолчанию **false**. Вызов получает `ReceiverSettings` и `ProxySettings`, сериализует proto JSON через `showJSONResponse`. `--timeout` по умолчанию 3 секунды. Команда по коду предназначена для чтения; никаких Add/Remove/Alter не вызывается.

**Коррекция предыдущего блока:** отдельная gRPC-библиотека или CLI grpcurl **не нужны для первоначального подтверждения поддержки `ListInbounds`**, если штатный бинарник Xray, установленный на Homeserver, поддерживает `api lsi`. Не менять `go.mod`, Go toolchain, существующие службы и версионирование. Для **долгосрочного автоматического** адаптера способ вызова штатного CLI либо зависимость согласовать отдельно только после подтверждения installed runtime.

**Безопасность:** полный JSON `api lsi` может содержать `proxySettings` с чувствительными полями; **никогда не выводить его целиком в диалог/лог**, не сохранять без защиты. Будущая read-only проверка должна в локальном процессе сделать JSON parse, оставить только безопасную проекцию `tag, ports, network, type` после строгой валидации typed messages и вернуть только redacted summary; при отсутствии API/метода — `FAIL_CLOSED` без попытки изменить конфигурацию. Адрес Xray API не выдумывать: должен быть получен из установленной действующей конфигурации, а не из публичного порта VPN.

**Статусы:** `PINNED_UPSTREAM_XRAY_API_LSI=SOURCE_CONFIRMED`; `INSTALLED_XRAY_API_LSI=NOT_TESTED`; `NEW_DEPENDENCY_FOR_READ_ONLY_PROBE=NOT_REQUIRED_IF_CLI_SUPPORTED`; `XRAY_APPLIED_EVIDENCE=NOT_CONNECTED`; `STAGE3=IN_PROGRESS`.

## 2026-11-22 — M6 этап 3: checkpoint штатного Xray gRPC ListInbounds на Homeserver — подготовлен, выполнение ожидается

**Источник проверки:** актуальные `docs/DEVELOPMENT_PLAN.md`, этот журнал, `xui/route_sync.go`; актуальный Bridge HEAD до записи `221e0be2c336b36311d810fe38ca2d15d4c6a090`. Ранее по реальному Homeserver подтверждены дочерний Xray PID `1253673`, PPID `2091`, `-config` и слушающие TCP-порты. **Не повторять инвентаризацию процесса, ss или Docker TCP CONNECT.** Официальный 3x-ui источник `MHSanaei/3x-ui`; Xray-core pinned upstream method `api lsi` подтверждён только по исходникам.

**Единственный новый вопрос:** поддерживает ли уже запущенный Xray read-only `HandlerService.ListInbounds` через локальный gRPC API, и возвращает ли непустой список handler tags? Ответ на этот вопрос ещё не получен; это не тест применённого backend и не Android end-to-end.

**Безопасная диагностическая процедура:** использовать точный уже известный PID лишь как источник реального бинарника `/proc/1253673/exe` и `/proc/1253673/cmdline` (guard, не повторное перечисление процессов); считать `-config` путь **только в локальной памяти**, прочитать текущий JSON **без публикации**, найти ровно один `tag=api` с целочисленным портом и буквально loopback `listen` (`127.0.0.1`/`::1`), выполнить `/proc/1253673/exe api lsi --server=<loopback-address> --isOnlyTags=true --timeout=3`. Локально разобрать stdout JSON, не показывать tag strings, полные inbound/proxySettings, stderr или путь к конфигурации; печатать только статус `XRAY_GRPC_LIST_INBOUNDS=PASS`, целое `XRAY_GRPC_TAG_COUNT`, `XRAY_RUNTIME_DETAIL=NOT_TESTED`. Любой отказ/отсутствие API/schema/процесса — `BLOCKED` либо `FAIL` без смены конфигурации. Нельзя включать Xray API или менять работающий процесс только ради теста. Повторные команды не делать, прежде всего сопоставить причину возможного отказа.

**Никаких изменений продукта:** исходники, Go dependencies, Docker, 3x-ui, Xray и APK не менялись; инструкция — подготовленный read-only шаг. `HOST_GRPC_LSI=NOT_TESTED`, `APPLIED_XRAY_EVIDENCE=NOT_CONNECTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. После предоставления вывода оператора записать фактический результат (PASS/FAIL/BLOCKED) отдельно с точным commit.

## 2026-11-22 — M6 этап 3: read-only Xray gRPC ListInbounds на реальном Homeserver — PASS

**Источник результата:** полный вывод read-only команды оператора в текущем диалоге, подготовленной и задокументированной ранее (checkpoint commit `8a9851cfecfcd2a1ac7403453ffc0ef5b3c7ad37`). **GitHub HEAD перед фиксацией:** `8a9851cfecfcd2a1ac7403453ffc0ef5b3c7ad37`; ветка `work/m6-android-ingress`. Работающие службы не затрагивались.

**Реальный результат:** `XRAY_GRPC_LIST_INBOUNDS=PASS`, `XRAY_GRPC_TAG_COUNT=5`, `XRAY_RUNTIME_DETAIL=NOT_TESTED`, `SECRETS=NOT_PRINTED`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Что именно проверено:** текущий Xray, запущенный в составе официального 3x-ui, имеет доступный локальный Xray gRPC API, поддерживает штатную read-only CLI команду `api lsi`, а `HandlerService.ListInbounds` с `isOnlyTags=true` возвращает 5 inbound handler tags. Номер API порта, пути конфигурации, сами tags и credentials в лог не выводились. Подтверждение не опирается на совпадение с config.json.

**Граница PASS:** `XRAY_LIVE_GRPC_LIST_INBOUNDS=PASS` и `XRAY_GRPC_TAG_COUNT=5`. Параметры `ReceiverSettings`, `ProxySettings`, реальные inbound ports, protocol/network, Xray listener binding, сопоставление с API 3x-ui/Remnawave и runtime network dataplane **не проверены**; `XRAY_APPLIED_ROUTE_DETAILS=NOT_TESTED`. Ни `AppliedXrayEvidence`, ни Docker apply, ни HTTPS discovery в production не подключены. `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

**Следующая ровно одна проверка:** безопасная локальная проекция **full** ответа `api lsi --isOnlyTags=false` без публикации raw JSON/proxySettings, private user data, endpoint или Xray API адреса; исследовать фактическую protobuf JSON форму `ReceiverSettings` и `ProxySettings`, прежде чем утверждать, что из них извлечены port/protocol/network. Если поля представлены как `TypedMessage` со встроенным binary payload и пока нет проверенного декодера, вернуться `UNSUPPORTED` и согласовать технический способ парсинга — ничего не додумывать. Ни PID, ни `ss`, ни Docker TCP CONNECT повторно не проверять.

## 2026-11-22 — M6 этап 3: единая read-only gRPC диагностика runtime inbound, исходники опубликованы, HOST TEST PENDING

**Сверка перед работой:** прочитаны `docs/DEVELOPMENT_PLAN.md`, `docs/V1_0_2_DEVELOPMENT_LOG.md`, `xui/route_sync.go` и `xui/xray_configured.go`; последний документированный HOST результат `HandlerService.ListInbounds isOnlyTags=true` = PASS, `XRAY_GRPC_TAG_COUNT=5`. Официальный Xray-core protobuf upstream `676eaf66f8f0` сверён по `common/serial/typed_message.proto`, `app/proxyman/config.proto`, `common/net/port.proto`, `transport/internet/config.proto`, `app/proxyman/command/command.go`. Bridge branch `work/m6-android-ingress`, source HEAD перед записью `4b7981424a73095dbe853de143bfe9ca4aad619d`.

**Изменения только диагностики (не production):**
- `integrations/android-ingress/check_xray_grpc_runtime.py`, commit `fcdb1f9ab37298ba1378233346b0d60ce0e58999`: выполняет один read-only `Xray api lsi --isOnlyTags=false` к уже известному Xray PID после строгой проверки исполняемого пути и loopback API config. Полный ответ обрабатывает **исключительно в памяти**. Декодирует protobuf `TypedMessage`, `ReceiverConfig.port_list`, `ReceiverConfig.stream_settings.protocol_name`, тип `ProxySettings`. Требует единичный TCP порт; неизвестные/неоднозначные поля не получает права публиковать. Выводит только port/protocol/network и `MATCH/DIFF/NOT_IN_CONFIG` относительно ранее сохранённого локального Xray config. Пропускает служебный `tag=api`; нормализует `splithttp/xhttp`, `raw/tcp`. **Это только сравнение runtime gRPC с config snapshot**, не доказательство выбранных API inbound 3x-ui/Remnawave. Никаких credentials, tags, raw gRPC bytes, API endpoint в stdout. Никаких RPC Add/Remove/Alter.
- `integrations/android-ingress/test_check_xray_grpc_runtime.py`, commit `4b7981424a73095dbe853de143bfe9ca4aad619d`: синтетические protobuf fixtures, точное извлечение port/type/network, запрет диапазона портов, повторного tag, обрывов и отсутствия TypedMessage.

**Проверки автора до commit:** локальный Python syntax и 4 изолированных fixture unit tests прошли на подготовленном варианте парсера (`4 passed`). **Это не тест GitHub HEAD на Homeserver** и не подтверждение реальной формы protojson текущего ядра. Домашняя команда должна сначала выполнить `python3 -m unittest -v test_check_xray_grpc_runtime.py`, затем единственный runtime RPC запрос через скрипт. На любой FAIL/отсутствии метода/отличии protobuf — остановиться, записать реальную причину, не делать повторных сетевых попыток без исправления.

**Критерии:** `PROJECTION_TEST_RC=0`; runtime `XRAY_RUNTIME_AUDIT=PASS_GRPC_PROJECTION` только при полноценном декодировании и отсутствии mismatch/unsupported. `XRAY_PANEL_API_SELECTION=NOT_TESTED` даже при PASS; не путать с актуальным выбором в панели. Допускается `BLOCKED` при несовместимости installed binary; не выдумывать порт API. Процессный PID `1253673` используется **как идентификационный guard по ранее проверенному процессу**, не как hardcoded backend endpoint; если PID изменился — прекратить read-only проверку и не искать/не менять процессы автоматически.

**Статус:** `GRPC_PROJECTION_SOURCE=COMMITTED`; `GRPC_PROJECTION_UNIT_ON_HOST=PENDING`; `XRAY_RUNTIME_AUDIT=PENDING`; `PRODUCTION_XRAY_DOCKER=UNCHANGED`; `STAGE3=IN_PROGRESS`; `RELEASE=NO`. 

## 2026-11-22 — M6 этап 3: Homeserver полный Xray gRPC аудит — BLOCKED, тесты декодера PASS

**Испытанный SOURCE_HEAD:** `7e3dc502549a51b003b563b03c0b42a7d1eee243`, `work/m6-android-ingress`. Оператор выполнил утверждённый один пакет: `git pull --ff-only` и `python3 -m unittest -v test_check_xray_grpc_runtime.py`, затем `python3 check_xray_grpc_runtime.py`. Fast-forward `b08814a..7e3dc50`; четыре теста `test_extract`, `test_reject_duplicate_tag`, `test_reject_invalid_wire_and_missing_fields`, `test_reject_range` — PASS (0.001s); `TEST_RC=0`. Реальная проверка: `XRAY_RUNTIME_AUDIT=BLOCKED`, `REASON=RUNTIME_SCHEMA_OR_SOURCE_UNAVAILABLE`, `RUNTIME_RC=2`; `RUNNING_SERVICES=UNCHANGED`; `SSH_CONSOLE=OPEN`.

**Не делать ложных выводов:** подтверждённый ранее `api lsi --isOnlyTags=true` остаётся PASS (5 handlers), но попытка полного разбора показала недифференцированную ошибку скрипта. Широкий except скрывает конкретный этап (конфиг / полный RPC / protojson / бинарный protobuf), так что **корень BLOCKED не определён**. `XRAY_APPLIED_ROUTE_DETAILS=NOT_CONFIRMED`, production без изменений. Нельзя менять 3x-ui/Xray/Docker на основании этого BLOCKED.

**Исправление в рамках этого же диагностического компонента:** минимально добавить фиксированный, безопасный `FAILED_STAGE` (без печати исключения, конфиг-пути, tags/секретов), исключать service inbound `api` из декодирования proxySettings. Повторная проверка одним пакетом после source commit; при отсутствии подтверждённого результата оставаться BLOCKED. 

## 2026-11-22 — M6 этап 3: фикс локальной диагностики после BLOCKED (SOURCE COMMITTED, HOST TEST PENDING)

**Точная ветка и HEAD перед документацией:** `work/m6-android-ingress`, `2e2ed448fa371d2e04dee5830d2f0f8dfa973c10`. После зафиксированного BLOCKED на `7e3dc502549a51b003b563b03c0b42a7d1eee243` изменён **только** диагностический скрипт и его тест, не Xray/3x-ui/Bridge core.
- `integrations/android-ingress/check_xray_grpc_runtime.py`, commit `d23c0d9880b91a14c2d4cc8ca7368c116e36b9e1`: для исключений теперь безопасная категория `RUNTIME_<FAILED_STAGE>_UNAVAILABLE`, а не общий `RUNTIME_SCHEMA_OR_SOURCE_UNAVAILABLE`. Разделены `PROCESS_SOURCE`, `CONFIG_SOURCE`, `CONFIG_READ`, `GRPC_QUERY`, `GRPC_RESPONSE_JSON`, `GRPC_PROTOBUF_PROJECTION`, `REFERENCE_COMPARISON` (в stdout возвращается только контролируемое фиксированное имя стадии). Внутренний `tag=api` не является VPN service backend и больше не требует `ProxySettings` при protobuf decoding.
- `integrations/android-ingress/test_check_xray_grpc_runtime.py`, commit `2e2ed448fa371d2e04dee5830d2f0f8dfa973c10`: добавлен тест исключения внутреннего API handler при наличии валидного VPN inbound.

**Проверка ожидается:** единый пакет `python3 -m unittest -v test_check_xray_grpc_runtime.py` (ожидается 5 тестов), `python3 -m py_compile check_xray_grpc_runtime.py test_check_xray_grpc_runtime.py`, затем **один** `python3 check_xray_grpc_runtime.py`. Ошибка или `BLOCKED` не являются PASS. Нельзя выводить `stderr`, `raw ListInbounds`, tags/credentials, путь config/API адрес. Нельзя запускать `AddInbound`, `RemoveInbound`, `AlterInbound`; работающие службы не менять. Если снова BLOCKED, классифицированная категория позволяет остановиться и исправить один конкретный этап без бессмысленного повторения всех шагов.

**Статус:** `DIAGNOSTIC_FAILURE_CLASSIFICATION=SOURCE_COMMITTED`, `HOMESERVER_TEST=NOT_RUN`, `XRAY_RUNTIME_DETAILS=BLOCKED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 Stage 3: повторный Homeserver полный Xray gRPC аудит — классифицированный BLOCKED

**Испытанный SOURCE_HEAD:** `9e30299930772653ed1f47ad31a787a944c4669f` в `work/m6-android-ingress`; `git pull --ff-only` fast-forward `7e3dc50..9e30299`. Оператор выполнил единым пакетом `python3 -m py_compile check_xray_grpc_runtime.py test_check_xray_grpc_runtime.py` (`SYNTAX_RC=0`), `python3 -m unittest -v test_check_xray_grpc_runtime.py` (все **5/5 PASS**, `TEST_RC=0`), `python3 check_xray_grpc_runtime.py` (`XRAY_RUNTIME_AUDIT=BLOCKED`, `REASON=RUNTIME_GRPC_PROTOBUF_PROJECTION_UNAVAILABLE`, `RUNTIME_RC=2`). `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Новая установленная граница сбоя:** источник config прочитан, gRPC full `api lsi` успешно вызван и ответ JSON разобран; сбой **в функции `extract()` для бинарных `TypedMessage`/привязанных полей**. На основе этого вывода нельзя назвать конкретное поле, входящий обработчик или тип protobuf mismatch. Синтетические unit-тесты не являются подтверждением реальной формы proto JSON. `LIVE_GRPC=PASS`, `RUNTIME_PROTOBUF_PROJECTION=BLOCKED`, `XRAY_APPLIED_ROUTES=NOT_CONFIRMED`.

**Следующее исправление строго в диагностическом компоненте:** пересмотреть реальный upstream способ сериализации `TypedMessage`, ввести безопасный фиксированный код **подэтапа поля** (без `str(exception)`, raw JSON, tags, credentials) для различения `receiverSettings`, `PortList`, `StreamConfig`, `proxySettings`; один объединённый тестовый прогон. Не трогать production.

## 2026-11-22 — M6 этап 3: установлена и исправлена первопричина protobuf projection BLOCKED в нашем диагностическом коде

**Сверка:** Bridge branch `work/m6-android-ingress`, HEAD до документации `e01882990b9fb4cbcd542162138a82a37df77dd3`; предыдущая подтверждённая Homeserver попытка `9e30299930772653ed1f47ad31a787a944c4669f` вернула `RUNTIME_GRPC_PROTOBUF_PROJECTION_UNAVAILABLE` (5/5 прежних синтетических тестов PASS). Открыты реальные исходники зафиксированного официальным 3x-ui Xray-core `676eaf66f8f0`: `common/reflect/marshal.go`, `common/serial/typed_message.go`, `app/proxyman/config.pb.go`, `transport/internet/config.pb.go`, `core/config.pb.go` и штатный `main/commands/all/api/shared.go`.

**Факт первопричины на уровне исходников:** CLI `xray api lsi` использует **не обычный protojson**, а `common/reflect.MarshalToJson(m, true)`, который `marshalTypedMessage()` раскрывает `TypedMessage` в декодированный JSON с полем `_TypedMessage_` (type), а `serializePortList()` сериализует singleton port в **целое число**. Наш первоначальный `extract()` ошибочно требовал `{"type":"...","value":"<base64>"}` и декодировал бинарные protobuf; он не соответствует формату *данного* CLI. Данное несоответствие в исходнике однозначно объясняет категорию отказа; другие отличия installed binary могут проявиться при следующей проверке.

**Минимальное исправление диагностического компонента:**
- `integrations/android-ingress/check_xray_grpc_runtime.py`, commit `6846d3e2eceb5f52293a13aad75bfab023617f9f`: разбор развёрнутого CLI JSON, проверка `receiverSettings._TypedMessage_ == xray.app.proxyman.ReceiverConfig`, числового `portList`, `streamSettings.protocolName`, `proxySettings._TypedMessage_`, по-прежнему без просмотра users/credentials. Запрещены диапазоны портов/дубликаты/inbound без необходимых атрибутов; `api` обрабатывается как служебный. Удалён собственный бинарный protobuf-декодер из диагностического скрипта, чтобы не дублировать форматизацию Xray.
- `integrations/android-ingress/test_check_xray_grpc_runtime.py`, commit `e01882990b9fb4cbcd542162138a82a37df77dd3`: переписаны unit-fixtures на **реальную схему CLI serializer** (`_TypedMessage_`, числовой `portList`, `protocolName`) и тесты invalid fields/ranges/duplicates/internal API/default network. Ожидается 6 тестов.

**Не выдавать за тестированный результат:** изменение исходников опубликовано, но `python3 -m py_compile`, `unittest` и **реальный gRPC full response на новом HEAD** ещё не исполнены на Homeserver. Не передавать результат проекции в production registrar; состояние на диске лишь `REFERENCE_DATA_SOURCE=CONFIG_SNAPSHOT_ONLY`, а не доказательство current applied Xray. Следующая команда должна единовременно проверить 6 unit-tests, синтаксис и сделать один live `api lsi`, не повторяя PID/ss/Docker проверки. Если при фактическом парсинге всплывёт другой формат, report `BLOCKED` с безопасным `REASON`, без raw output и секретов.

**Статус:** `ROOT_CAUSE=CLI_CUSTOM_JSON_VS_PROTOJSON` (**SOURCE_CONFIRMED**), `JSON_PARSER_FIX=SOURCE_COMMITTED`, `HOMESERVER_RETEST=PENDING`, `APPLIED_XRAY_EVIDENCE=NOT_CONNECTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 Stage 3: Homeserver gRPC full inbound audit — PARTIAL, aggregate FAIL

**Испытанный SOURCE_HEAD:** `1701aac2ad8b984cd9d6bf1270d89331fbeab953`, ветка `work/m6-android-ingress`, pull `9e30299..1701aac`. Python syntax `SYNTAX_RC=0`; 6 fixture-тестов `PASS`, `TEST_RC=0`; read-only `xray api lsi --isOnlyTags=false` завершился и выдал 5 live inbound: `XRAY_GRPC_TOTAL=5`, `XRAY_INBOUND port=55535 protocol=vless network=tcp reference=MATCH`, `XRAY_INBOUND port=55636 protocol=vless network=splithttp reference=MATCH`; `XRAY_INTERNAL_API_SKIPPED=1`, `XRAY_UNSUPPORTED_SKIPPED=2`, `XRAY_COMPARED_COUNT=2`, `XRAY_REFERENCE_MISMATCH_COUNT=0`, `XRAY_RUNTIME_AUDIT=FAIL_INCOMPLETE_OR_DRIFT`, `RUNTIME_RC=1`. `XRAY_PANEL_API_SELECTION=NOT_TESTED`, `XRAY_RUNTIME_DATA_SOURCE=LIVE_GRPC`, `XRAY_REFERENCE_DATA_SOURCE=CONFIG_SNAPSHOT_ONLY`, `SECRETS=NOT_PRINTED`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Строгое заключение:** runtime live gRPC успешно декодирован для двух VLESS inbound, и они совпали по port/protocol/network с сохранённым config. Два остальных inbound необъяснённо классифицированы скриптом как `UNSUPPORTED`; среди прежних проверенных xray TCP listeners есть Trojan 30000, но **пока нельзя утверждать**, что один из двух отброшенных относится к нему. Полного PASS нет; `RUNTIME_AUDIT=FAIL`, `PRODUCTION_DISCOVERY=NOT_CHANGED`, `XRAY_ALL_REQUIRED_TCP_INBOUNDS=UNPROVEN`.

**Следующая работа в этом же модуле:** исследовать исходники Xray-core для фактических `proxySettings._TypedMessage_` (Trojan и иные inbound), исправить только false-unsupported классификацию; не скрывать несовпадения и неподдерживаемые *целевые TCP inbound*. Не повторять PID/ss/отдельные gRPC ping. Тестировать в одной пакетной проверке, доказать runtime mapping, фиксировать результат здесь.

## 2026-11-22 — M6 Stage 3: классификация двух UNSUPPORTED live inbound — SOURCE FIX, TEST PENDING

**Исходный испытанный HEAD:** `1701aac2ad8b984cd9d6bf1270d89331fbeab953`: из 5 live handlers 2 VLESS MATCH, 1 API, 2 UNSUPPORTED, `RUNTIME_RC=1`. **HEAD перед этим документом:** `e4a003451f3dba499f55d30b024ba48bb19fae15`, branch `work/m6-android-ingress`. Исследованы реальные proto `XTLS/Xray-core` на pinned ref `676eaf66f8f0`: `proxy/trojan/config.proto` (`xray.proxy.trojan.ServerConfig`), `proxy/socks/config.proto` (`xray.proxy.socks.ServerConfig`), `proxy/http/config.proto` (`xray.proxy.http.ServerConfig`), `proxy/dokodemo/config.proto` (`xray.proxy.dokodemo.Config`). Реальный `xray.proxy.vless.inbound.Config` уже декодировался. По логам нельзя доказать, какие именно два UNSUPPORTED были реально получены; это ожидаемые **кандидаты**, не подтверждённая идентичность.

**Изменены только диагностические Python исходники:**
- `check_xray_grpc_runtime.py`, commit `470d86cf1306f082bd51b2c04c26da9551d65822`: добавлено распознавание строго известных типов `trojan/socks/http.ServerConfig` и `dokodemo.Config`; ранее подтверждённые VLESS не меняются. Для иных непонятных TCP inbound печатается лишь безопасный `port=<n> status=UNSUPPORTED_PROTOCOL_OR_TRANSPORT`, raw tags/credentials не публикуются; общий FAIL остаётся.
- `test_check_xray_grpc_runtime.py`, commit `b309e8cef40b26bf9dd34f536a55a99e9f3e5d80`: тестовые fixtures на 3 известных `ServerConfig` и 1 `dokodemo.Config`; ожидается **8 unit test methods**.
- `check_xray_grpc_runtime.py`, commit `e4a003451f3dba499f55d30b024ba48bb19fae15`: исправлена fail-open ошибка: `NOT_IN_CONFIG` теперь увеличивает mismatch count и не позволяет показать PASS без установленного reference. Нет hardcode реальных backend ports, users, hosts или UUID.

**Критерий checkpoint одним пакетом на Homeserver:** проверка точного HEAD, `python3 -m py_compile`, `python3 -m unittest -v test_check_xray_grpc_runtime.py` (8 tests), **один** вызов `python3 check_xray_grpc_runtime.py`; `RUNTIME_RC=0` только при `XRAY_UNSUPPORTED_SKIPPED=0`, `XRAY_REFERENCE_MISMATCH_COUNT=0`, `XRAY_RUNTIME_AUDIT=PASS_GRPC_PROJECTION`. Даже при PASS это не panel API selection и не реальный Bridge E2E. При fail остановиться на конкретном unsupported port / label, не исправлять сетевые службы. `XRAY_PANEL_API_SELECTION=NOT_TESTED`, `LIVE_XRAY_RUNTIME=PREVIOUS_PARTIAL`, `NEXT_HOST_TEST=PENDING`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 этап 3: фактическая live gRPC проекция всех Xray TCP inbound — PASS

**Источник доказательства:** итоговый вывод Homeserver, предоставленный оператором для единого диагностического пакета ветки `work/m6-android-ingress`. **Источник кода перед документированием:** `43d866c0330a0d07e7b061e1c17c006718e1cffc`. Исполнены Python syntax, unit tests и read-only `xray api lsi --isOnlyTags=false` через `integrations/android-ingress/check_xray_grpc_runtime.py`.

**Проверенные результаты:** `SYNTAX_RC=0`, `TEST_RC=0` (дополнительный полный список тестов в данном сообщении не повторён, итог `OK`), `RUNTIME_RC=0`; `XRAY_RUNTIME_AUDIT=PASS_GRPC_PROJECTION`; `XRAY_GRPC_TOTAL=5`. Четыре действующих inbound из gRPC и их сопоставление с config snapshot: `10811 socks/tcp MATCH`, `30000 trojan/tcp MATCH`, `55535 vless/tcp MATCH`, `55636 vless/splithttp MATCH`. `XRAY_INTERNAL_API_SKIPPED=1`; `XRAY_UNSUPPORTED_SKIPPED=0`; `XRAY_COMPARED_COUNT=4`; `XRAY_REFERENCE_MISMATCH_COUNT=0`. `XRAY_RUNTIME_DATA_SOURCE=LIVE_GRPC`, `XRAY_REFERENCE_DATA_SOURCE=CONFIG_SNAPSHOT_ONLY`; `XRAY_PANEL_API_SELECTION=NOT_TESTED`, `SECRETS=NOT_PRINTED`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Строгий смысл PASS:** штатный Xray gRPC перечислил действующие обработчики и успешно декодированы port/protocol/network четырёх TCP inbound; совпадение с локальным снимком конфигурации подтверждено. Это не является доказательством того, что API официального MHSanaei/3x-ui (или Remnawave) в настоящий момент возвращает тот же выбранный набор, и не проверяет Bridge backend traffic, mTLS, Android E2E, применение динамического policy или Docker ACK. Само наличие SOCKS 10811 в числе gRPC inbound **не означает**, что он автоматически разрешён для Bridge; нужен точный выбор из API панели и подтверждённый backend.

**Статусы:** `XRAY_RUNTIME_GRPC_INBOUND_PROJECTION=PASS`; `PY_SYNTAX=PASS`; `PY_UNIT_TEST=PASS`; `XRAY_PROVIDER_SELECTION_RECONCILE=NOT_TESTED`; `BRIDGE_POLICY_APPLY_DOCKER=NOT_CONNECTED`; `STAGE3=IN_PROGRESS`; `V1_0_2_RELEASE=NO`.

**Следующий шаг в текущем этапе:** сверить существующие provider-adapters `xui/inbounds.go` / `xui/remnawave.go` с актуальными Xray live gRPC данными через существующий `ReconcileInbounds`, не добавляя и не разрешая новые службы самостоятельно. Запретить публикацию discovery без согласования выбранного provider inbound и полного policy ACK. Проверки PID/ss/TCP CONNECT не повторять.

## 2026-11-22 — M6 этап 3: source-only аудит provider selection vs Xray runtime; обнаружены два блокера (NO CODE CHANGES)

**Подтверждённая исходная база:** ветка `work/m6-android-ingress`, HEAD до записи `d392708b7a7dab8da30162a1849939d18c7e4d31`. Прочитаны `docs/DEVELOPMENT_PLAN.md`, `docs/V1_0_2_DEVELOPMENT_LOG.md`, `integrations/android-ingress/xui/inbounds.go`, `remnawave.go`, `route_reconcile.go`, `route_sync.go` и их тесты. Последний испытанный live gRPC статус: `XRAY_RUNTIME_AUDIT=PASS_GRPC_PROJECTION`, 5 handlers, из них 4 TCP: socks 10811, trojan 30000, vless/tcp 55535, vless/splithttp 55636; все 4 совпали с config snapshot. Это **НЕ** проверка provider selection.

**Проверка строго официального upstream:** `MHSanaei/3x-ui` commit `5297fd3c8495d80fab97ae511553fa545c69e913`: `internal/web/controller/inbound.go` объявляет GET `/list/slim` и обслуживает через `GetInboundsSlim(user.Id)`; `internal/web/service/inbound.go` возвращает привязанные к пользователю inbound панели и частично сокращает `settings.clients`. `internal/web/controller/api_auth_test.go` подтверждает допустимость Bearer token и применение scope, однако установленная на Homeserver версия и фактический успешный API запрос **НЕ ПРОВЕРЕНЫ**. Источники: https://github.com/MHSanaei/3x-ui/blob/5297fd3c8495d80fab97ae511553fa545c69e913/internal/web/controller/inbound.go ; https://github.com/MHSanaei/3x-ui/blob/5297fd3c8495d80fab97ae511553fa545c69e913/internal/web/controller/api_auth_test.go .

**Блокер A — транспортный alias:** `route_reconcile.go` в `ReconcileInbounds` проверяет `strings.EqualFold(p.Network,x.Network)` буквально; при `p.Network=xhttp`, `x.Network=splithttp` gRPC-подтверждённый inbound **не совпадает**. Код диагностики `check_xray_grpc_runtime.py` уже нормализует эту пару, но production reconcile — нет. Аналогичное потенциальное расхождение `tcp/raw`; считать одинаковыми можно только конкретные известные aliases после отдельного теста.

**Блокер B — отсутствие доказанной policy selection:** существующий 3x-ui `ListInbounds` формирует redacted `InboundSnapshot` из `list/slim`, а `SyncVerifiedRoutes` принимает `RouteEvidenceSource` и требует подтверждения всех входящих `Enabled=true` в переданном provider snapshot. Это API-проекция `enable`, **не явный список разрешённых Bridge services**. В живом Xray обнаружен SOCKS 10811, но это не свидетельство разрешения SOCKS для Bridge. Точный источник решения о разрешённых выбранных inbound и безопасный способ исключения служебных/непредназначенных для Bridge inbound не должны быть придуманы. Не заменять узкую процедуру отбора автоматическим «все включённые»; не добавлять маршруты к SOCKS/посторонним системным портам без согласования.

**Границы проверенного:** source review подтвердил upstream route, токеновые тесты и точные проблемы reconcile. **Фактический 3x-ui panel API на Homeserver не вызывался**, сопоставление выбранного provider набора с live Xray не запускалось, Docker policy ACK и Android E2E не выполнялись. `PROVIDER_SELECTION=NOT_TESTED`, `RUNTIME_MAPPING=PASS_PREVIOUS`, `RECONCILE_XHTTP_ALIAS=SOURCE_FAIL`, `M6_STAGE3=IN_PROGRESS`, `RELEASE=NO`. Продакшен-службы, исходники и конфигурации не менялись.

**Предложение к согласованию перед кодом:** в том же `ReconcileInbounds` минимально нормализовать только подтверждённые network aliases (`xhttp/splithttp`, `tcp/raw`) с регрессионными тестами; отдельно установить из уже утверждённого механизма registrar/selection, какой перечень именно оператор разрешил Bridge. Затем одним пакетом сделать tests/race/vet и read-only, без секретов, API selection ↔ live gRPC сопоставление; только после подтверждённого результата разрешить дальнейшее соединение с существующим policy ACK. Не создавать новый механизм отбора и не менять архитектуру без отдельного согласия оператора.

## 2026-11-22 — M6 этап 3: согласованное минимальное исправление network aliases в существующем ReconcileInbounds (SOURCE COMMITTED)

**Контекст / исходное согласование:** продолжение пользователем непосредственно после предложения исправить только `xhttp/splithttp`, `tcp/raw` и сохранить утверждённую регистрацию. Перед изменениями прочитаны `docs/DEVELOPMENT_PLAN.md`, журнал M6, `xui/route_reconcile.go`, `route_reconcile_test.go`, `route_sync.go`, `inbounds.go`, `remnawave.go` и `cmd/m6-auto-registrar/main.go`. **HEAD до записи:** `465d35be277e1ee9b78076d5c87247f87bc94563`, ветка `work/m6-android-ingress`.

**Подтверждённая картина:** runtime Xray via gRPC `PASS` на прошлой проверке, 4 TCP inbound, 1 служебный. В `ReconcileInbounds` было буквальное сравнение `Network`, поэтому фактически обнаруженный Xray `splithttp` мог не совпасть с panel `xhttp`. Existing `configuredTCPRoutes(root)` в `m6-auto-registrar` берёт соответствия из действующего `config/server.json`, а `SyncVerifiedRoutes` использует `ConfirmedRouteStore` с complete-generation ACK — эти функции **не заменяли**. Уже реализованный provider-selection Remnawave через Node/Profile UUID также не изменён. Прямой live GET официальной 3x-ui панели **не выполнялся**; панельная авторизация/актуальная выборка остаётся отдельным шагом.

**Сделанное:**
1. `integrations/android-ingress/xui/route_reconcile.go`, commit `d58249764961ff2bdb33b95616968b91daa2e9e9`: добавлена локальная `canonicalInboundNetwork()`, которая строго нормализует только `splithttp -> xhttp` и `raw -> tcp` с регистронезависимостью; ранее существующие ограничения по выбранным, Enabled, Protocol, Port, Service и backend Address сохранены. Не расширяли поддерживаемые VPN протоколы, не меняли публикацию маршрутов и ACL.
2. `integrations/android-ingress/xui/route_reconcile_test.go`, commit `465d35be277e1ee9b78076d5c87247f87bc94563`: 8 table subtests на aliases/mismatch/UDP/unknown network, плюс test запрета маршрутов для unselected/disabled inbound.

**Не проверено:** `go test`, `go test -race`, `go vet` именно обновлённого source HEAD ещё **не выполнены**; не утверждать PASS. Первая проверка — единым пакетом в штатном модуле `integrations/android-ingress/xui` на точном HEAD: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`. Статусы `SOURCE_FIX=COMMITTED`, `GO_TEST=PENDING`, `GO_RACE=PENDING`, `GO_VET=PENDING`, `PANEL_PROVIDER_SELECTION_LIVE=NOT_TESTED`, `RUNTIME_GRPC=PREVIOUS_PASS`, `WORKING_SERVICES=UNCHANGED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. Следующий шаг только после подтверждения результатов без повторного PID/ss/gRPC.

## 2026-11-22 — M6 Stage 3: Go regression FAIL caused by source corruption, minimal correction committed

**Tested source HEAD (Homeserver):** `8140756b54c7ee4c9843919e3337b5165b2314ff`, branch `work/m6-android-ingress`; `git pull --ff-only` fast-forward `43d866c..8140756`. Operator ran complete batch in `integrations/android-ingress/xui`: `go test -count=1 ./...` = `GO_TEST_RC=1`, `go test -race -count=1 ./...` = `GO_RACE_RC=1`, `go vet ./...` = `GO_VET_RC=1`. All three fail to compile due to `route_reconcile.go:40:46: string not terminated` and `line 137: unexpected EOF`. Some independent catalog packages PASS, **but overall M6 module FAIL**. `RUNNING_SERVICES=UNCHANGED`; `SSH_CONSOLE=OPEN`.

**Root cause verified directly in committed GitHub source:** previous modification malformed the exact `routeServicePattern` regex declaration near line 40 and duplicated the `package xui` and import block within raw-string content. This was an assistant-generated source-editing defect, unrelated to 3x-ui, Xray, network or Docker. Do not re-run infrastructure diagnostics to address it.

**Minimal fix committed:** `f1898bd9113da42399a9b00fcdef47b4ff74eeac` restores original Go package/imports, model types and `var routeServicePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)` once, retaining only previously approved helper `canonicalInboundNetwork` (`splithttp/xhttp`, `raw/tcp`) and unchanged reconcile behavior otherwise. No deployment, server config or working services affected.

**Preliminary local syntax check:** equivalent minimal package/regexp/helper fragment passes `gofmt -e` using Go 1.23.2, `GO_FRAGMENT_FORMAT_RC=0`, but this is **not** `go test` or `gofmt` of actual complete GitHub file. Actual complete code `go test`, `go test -race`, `go vet` still `PENDING` on fixed commit. Next read-only Homeserver package test should run once in a single batch, using exact source HEAD and optionally `gofmt -e` check first, then tests/race/vet, recording real statuses. `M6_STAGE3=IN_PROGRESS`; `PROVIDER_PANEL_API_LIVE=NOT_TESTED`; `V1_0_2_RELEASE=NO`.

## 2026-11-22 — M6 этап 3: проверка исправления ReconcileInbounds на Homeserver — PASS

**Фактически испытанный источник:** ветка `work/m6-android-ingress`, `SOURCE_HEAD=a52adc3c0895bb4e6a6ada6eae055ec98a3f4482`, получен fast-forward `8140756..a52adc3` по SSH через `git pull --ff-only`; `integrations/android-ingress/xui`. Предыдущий source syntax FAIL (raw-string regex с повторным package/import) исправлен commit `f1898bd9113da42399a9b00fcdef47b4ff74eeac`; минимальная нормализация alias и её тесты сохранены.

**Оператор исполнил единым пакетом:** `gofmt -e route_reconcile.go >/dev/null` — `SYNTAX_RC=0`; `go test -count=1 ./...` — `GO_TEST_RC=0`, **6 пакетов PASS** (`xui`, `catalog`, `cmd/catalog-keygen`, `cmd/catalog-sign`, `cmd/m6-auto-registrar`, `cmd/m6-first-device`); `go test -race -count=1 ./...` — `GO_RACE_RC=0`, те же **6 пакетов PASS**; `go vet ./...` — `GO_VET_RC=0`. `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Критерии:** `M6_RECONCILE_ALIAS_SOURCE=COMMITTED`, `M6_RECONCILE_GO_SYNTAX=PASS`, `M6_RECONCILE_GO_TEST=PASS`, `M6_RECONCILE_GO_RACE=PASS`, `M6_RECONCILE_GO_VET=PASS` **на точно испытанном commit**. Исправление готово по source-уровню и тестированию; в production не устанавливалось. Исходно подтверждённая `XRAY_RUNTIME_GRPC=PASS` остаётся действительной для прошлой проверки. `LIVE_3XUI_PANEL_API_SELECTION=NOT_TESTED`, `LIVE_PROVIDER_RECONCILIATION=NOT_TESTED`, `CORE_POLICY_ACK=NOT_CONNECTED`, `M6_STAGE3=IN_PROGRESS`, `V1_0_2_RELEASE=NO`.

**Следующий согласованный этап без повторных проверок:** read-only сопоставление выбранных пользователем inbound через уже существующий клиент API официального `MHSanaei/3x-ui` с Xray live gRPC projection. Не считать `Enabled` автоматически разрешением на Bridge, не рекламировать SOCKS/невыбранные службы, использовать существующие регистратор `server.json`, `ReconcileInbounds` и ACK-контракт. Никаких изменений действующих служб без отдельного утверждения.

## 2026-11-22 — M6 этап 3: проверка источников provider selection перед живой сверкой (READ-ONLY REVIEW)

**HEAD до документации:** `100f0f986d26409d1455af28721375f3a752b3f1`, branch `work/m6-android-ingress`. Проверены актуальные `docs/DEVELOPMENT_PLAN.md`, журнал, `xui/client.go`, `xui/inbounds.go`, `xui/route_sync.go`, `xui/cmd/m6-auto-registrar/main.go`, `check_live_xray_bridge_routes.py`, а также источники ранее подтверждённого gRPC PASS и Go reconcile PASS.

**Существующие механизмы подтверждены:** `xui.Client.ListInbounds` читает только GET `/panel/api/inbounds/list/slim` по HTTPS с Bearer в защищённом server-side файле; `configuredTCPRoutes(root)` читает только текущие полные маршруты из `config/server.json`; `ReconcileInbounds` сверяет provider-selected и runtime-effective по port/protocol/network; `SyncVerifiedRoutes` требует полный route+ACL generation ACK. **Эти части существуют независимо**, но текущий `m6-auto-registrar` формирует discovery из `server.json`, а не вызывает `ListInbounds` / `SyncVerifiedRoutes`; production автоматическое API↔gRPC согласование пока НЕ ПОДКЛЮЧЕНО. Не дублировать перечисленные функции.

**Восстановлено из предыдущих операторских результатов:** на Homeserver официальный 3x-ui локально ранее отвечал HTTP 200 с Bearer через loopback `curl --resolve`, рабочий header file `/etc/tervon/bridge-android/private/xui-api-header` (`0600`). Известна установленная ранее panel HTTPS endpoint с приватным base-path, который нельзя выводить в публичную документацию или Git. Этот HTTP 200 был проверкой доступности API, **не подтверждением выбранного списка inbound**. Не требовать повторной проверки только ради HTTP 200.

**Существующий `check_live_xray_bridge_routes.py` — CONFIG_ONLY**, читает JSON Xray на диске, поэтому не заменяет уже подтверждённую live `HandlerService.ListInbounds`. Ранее подтверждены gRPC 4 TCP handlers: SOCKS 10811, Trojan 30000, VLESS/TCP 55535, VLESS/XHTTP 55636; (API служебный inbound исключён). Эти портовые значения — только данные прошлого evidence, не жёстко запрограммированные правила.

**Точное следующее действие — одна read-only cross-source проверка:** защищённо получить `list/slim` без вывода response body, отдельно прочитать текущий `server.json` и перечень inbound из gRPC в рамках одного snapshot, сравнить строгие `enable`, `port`, `protocol`, normalized `network`, пересечение с ТОЛЬКО уже разрешёнными Bridge service routes. При отсутствии панели, неверном формате или несовпадении — `FAIL_CLOSED`. **Не** включать SOCKS/любые другие inbound по факту Enabled, не менять `server.json`, Docker, ACL или работающие процессы. Выводить только редактированную таблицу порт/протокол/транспорт/результат; ни токены, ни panel base-path, ни VPN client identifiers. Конкретная сквозная проверка на Homeserver ещё `NOT_RUN` и не может считаться PASS.

**Статусы:** `RECONCILE_SOURCE_TEST=PASS_PREVIOUS`, `XRAY_GRPC_PROJECTION=PASS_PREVIOUS`, `XUI_API_REACHABLE=PASS_PREVIOUS`, `XUI_SELECTED_INBOUND_LIVE=NOT_TESTED`, `AUTO_PROVIDER_RECONCILE=NOT_CONNECTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. Исходники и действующие службы в этом шаге не менялись.

## 2026-11-22 — M6 этап 3: единая read-only сверка 3x-ui API ↔ live Xray gRPC ↔ существующие Bridge routes (SOURCE COMMITTED, HOST TEST PENDING)

**Порядок и контекст:** проверены `docs/DEVELOPMENT_PLAN.md`, текущий журнал, `xui/inbounds.go`, `xui/client.go`, `xui/route_sync.go`, `xui/cmd/m6-auto-registrar/main.go`, текущий `check_xray_grpc_runtime.py`, `check_live_xray_bridge_routes.py`. Ветка `work/m6-android-ingress`; **HEAD до записи** `ff4a491cd10025e7d75bb82d3d48cc2405d9fb20`. Предыдущий source-level `ReconcileInbounds` на `a52adc3c0895bb4e6a6ada6eae055ec98a3f4482`: GO_TEST/RACE/VET PASS; live Xray gRPC projection на `43d866c..`: PASS 4 TCP inbounds; installed panel list/slim и selected Bridge route cross-check ещё не тестировались.

**Изменения исключительно отдельного read-only аудита:**
- `integrations/android-ingress/check_panel_xray_bridge_routes.py`, commits `0dad2994a74592dc1f44b2652386c4796fa0f723`, `ff4a491cd10025e7d75bb82d3d48cc2405d9fb20`: проверяет только пересечение **уже разрешённых** маршрутов `/etc/tervon/bridge-android/config/server.json` с live `GET /panel/api/inbounds/list/slim` и текущим `HandlerService.ListInbounds` (один Xray read-only RPC). Использует уже проверенную Python `extract()` из `check_xray_grpc_runtime.py` и серверный защищённый `/etc/tervon/bridge-android/private/xui-api-header` (права 0600). HTTPS/TLS-сертификат в curl **не отключается**; `--connect-to` сохраняет TLS hostname, соединяясь с loopback, URL с приватным panel prefix передаётся через окружение `M6_PANEL_BASE_URL` и не включается в GitHub. Для каждого разрешённого порта проверяет `enabled=true`, `protocol`, `network` с строго подтверждёнными alias `splithttp/xhttp`, `raw/tcp`, и наличие в действующем Xray. Любой missing/disabled/mismatch/unsupported — FAIL; повреждённый ответ/API/конфигурация — BLOCKED; невыбранные/посторонние inbound **никогда не добавляются**. Ответы 3x-ui/Xray, токен, приватный URL, Xray tags и клиентские credentials не выводятся, только порт/протокол/транспорт/status для разрешённых маршрутов. Ни config.json, ни Docker/3x-ui/Xray/ACL/registrar не изменяются.
- `integrations/android-ingress/test_check_panel_xray_bridge_routes.py`, commit `3c2d093abc9dae3160fa90fc3008300c91b27bfb`: семь локальных offline тестов: matched aliases, disabled, protocol drift, missing panel/runtime, excluded unapproved SOCKS, duplicate panel/bad settings, ambiguous duplicate Bridge ports.
- Важно: это **test-only checker**, не replacement для `xui.ListInbounds`, `ReconcileInbounds`, `SyncVerifiedRoutes` и не production wiring. Не изменены Go dependencies/Go version, реальные backend ports/hosts и способ формирования discovery.

**Проверки:** синтаксис Python и 7 unit-тестов **NOT_RUN на exact HEAD**, live три источника **NOT_RUN**. На Homeserver требуется **одна** пакетная команда для `python3 -m py_compile`, `python3 -m unittest -v test_check_panel_xray_bridge_routes.py` и `M6_PANEL_BASE_URL` через приватный stdin/environment (не писать URL/токен в логи), затем единственный `python3 check_panel_xray_bridge_routes.py`. Считать PASS только `SYNTAX_RC=TEST_RC=RUNTIME_RC=0` и `M6_PANEL_RUNTIME_AUDIT=PASS_READ_ONLY`. Успешный read-only audit **не** означает публикацию policy/ACK, deployment или Android data E2E.

**Статусы:** `THREE_WAY_AUDIT_SOURCE=COMMITTED`, `THREE_WAY_AUDIT_TEST=NOT_RUN`, `PANEL_SELECTED_ROUTES=NOT_TESTED`, `XRAY_RUNTIME=PREVIOUS_PASS`, `GO_RECONCILE=PREVIOUS_PASS`, `POLICY_CORE_ACK=NOT_CONNECTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 этап 3: three-way read-only checker, Python syntax FAIL, corrected source pending retest

**Tested source on Homeserver:** `796a105809dc24b756e630fdd9b5c10a30c54080` (branch `work/m6-android-ingress`). Operator's consolidated run: `SYNTAX_RC=1`, `TEST_RC=1`, `RUNTIME_RC=NOT_RUN`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`. Python 3.14 reports `SyntaxError: unexpected character after line continuation character` in `check_panel_xray_bridge_routes.py` at line 108. `unittest` fails during import of the same file; separate `apport_python_hook` traceback is an exception-reporting side effect, **not** a second application defect. No 3x-ui API call, live Xray gRPC request or Bridge configuration read was performed by the checker.

**Exact root cause verified in GitHub:** during previous source-editing operation, literal backslash-n characters `\\n` were written into the Python source inside `get_panel()` where actual newlines were needed. Fault is in our source generation and does not implicate Homeserver/3x-ui/Xray.

**Minimal committed fix:** `5157fec18f685e4e69582cece93151389d3b931c` in `integrations/android-ingress/check_panel_xray_bridge_routes.py`: replace precisely that malformed three-statement span with valid physical lines; no change to route authorization, TLS verification, credentials, API endpoint, production services, policy or Bridge core. Confirmed from GitHub file content that lines now separate correctly.

**Verification status:** fixed HEAD prior to documentation `5157fec18f685e4e69582cece93151389d3b931c` remains `SOURCE_COMMITTED, HOST_RETEST_PENDING`; repeat **once** as a comprehensive batch `python3 -m py_compile`, 7 `unittest` methods, and only on success the read-only 3-way live audit with private HTTPS base URL entered without echoing it. No raw panel responses, private paths, credentials, Xray tags or client IDs in stdout. `XRAY_RUNTIME_GRPC=PREVIOUS_PASS`, `GO_RECONCILE=PREVIOUS_PASS`, `M6_THREE_WAY_LIVE=NOT_TESTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 Stage 3: three-way checker syntax / unit tests PASS, panel URL input rejected before any HTTP or gRPC

**Examined Homeserver source HEAD:** `89869b7d7d7af2dd8e992cfcbf3126e9816f4f7b`, branch `work/m6-android-ingress`; fast-forward pull `796a105..89869b7`. `python3 -m py_compile` `SYNTAX_RC=0`; `python3 -m unittest -v test_check_panel_xray_bridge_routes.py`: **7/7 PASS**, `TEST_RC=0`. Live step returned `M6_PANEL_RUNTIME_AUDIT=BLOCKED`, `REASON=PANEL_URL_INVALID`, `RUNTIME_RC=2`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Exact source classification:** `get_panel()` validates `M6_PANEL_BASE_URL` via `urllib.parse.urlsplit()` (HTTPS, hostname, no embedded credentials, query, fragment, valid port and base path). `PANEL_URL_INVALID` arises before reading protected credential file, invoking curl or querying Xray. The user-provided plain HTTPS origin with port and private base path is structurally valid; the actual characters passed to the hidden `read -s` were not logged, so the precise cause (Markdown brackets, pasted trailing whitespace, etc.) remains **UNPROVEN**. Do not infer unavailable actual input. Enter only raw URL without `/panel` suffix, quotes, brackets, spaces; execute the live checker once with env input and unchanged HEAD. No source change warranted by evidence so far, do not repeat syntax/unit tests solely because of malformed operator input.

**Status:** `THREE_WAY_CHECKER_SYNTAX=PASS`, `THREE_WAY_CHECKER_UNIT=PASS_7_OF_7`, `PANEL_URL_INPUT=BLOCKED`, `LIVE_PANEL_API=NOT_QUERIED_THIS_RUN`, `XRAY_GRPC=NOT_QUERIED_THIS_RUN`, `M6_THREE_WAY_AUDIT=NOT_TESTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`.

## 2026-11-22 — M6 Stage 3: live 3x-ui ↔ Xray gRPC ↔ Bridge routes three-way audit — PASS

**Tested source:** `92356cba775e61cef09b6e20c4b0142d7850d9e6` on `work/m6-android-ingress`; Homeserver fast-forward `89869b7..92356cb`. After prior `SYNTAX_RC=0`, seven unit methods `TEST_RC=0` confirmed on source `89869b7`, the operator reran **only** the previously blocked live audit; no unnecessary repeated Python checks.

**Evidence from operator's consolidated read-only invocation:** `M6_APPROVED_ROUTE port=30000 protocol=trojan network=tcp status=MATCH`; `M6_APPROVED_ROUTE port=55636 protocol=vless network=xhttp status=MATCH`; `M6_APPROVED_ROUTE_COUNT=2`, `M6_MATCHED_ROUTE_COUNT=2`; `M6_PANEL_SOURCE=LIVE_3XUI_API`; `M6_XRAY_SOURCE=LIVE_GRPC`; `M6_ALLOWED_SOURCE=EXISTING_SERVER_JSON`; `M6_PANEL_RUNTIME_AUDIT=PASS_READ_ONLY`; `AUDIT_RC=0`; `M6_PRODUCTION_POLICY_ACK=NOT_TESTED`, `SECRETS=NOT_PRINTED`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`. No broader inbound selection was inferred; SOCKS or other unrelated active inbounds were not advertised as Bridge routes.

**Meaning and remaining boundary:** Exact authorized routes 30000 Trojan/TCP and 55636 VLESS/XHTTP matched enabled panel records and live gRPC runtime records. `M6_THREE_WAY_READ_ONLY=PASS` on tested HEAD, **not** proof of application of complete route+ACL policy generation, Decapsulator ACK, Android HTTPS/mTLS E2E or release. `M6_POLICY_ACK=NOT_TESTED`, `STAGE3=IN_PROGRESS`, `RELEASE=NO`. Do not change running services or publish additional routes.

**Operator UX:** Previous CLI prompt used `read -s` (hidden input) for private panel base URL. Operator questioned this unnecessary concealment: for future *local SSH prompt* use ordinary `read -r -p`, while continue to keep private panel prefix, Bearer token and client credentials out of GitHub, logs and copied session transcripts. No code or production changes requested.

## 2026-11-22 — Коррекция объекта комплексной проверки и сборки по прямому указанию оператора

**Явно уточнено пользователем:** «НЕ APK! Tervon TCP Bridge 1.0.2!!!». Предыдущее направление на Android preflight, Flutter/Kotlin и APK — **ошибочное переключение модулей ассистентом**; Android, мобильную UI и перенос файлов сейчас не вести, текущий Android source mismatch не является блокером для компиляции *серверного Bridge v1.0.2*.

**Зафиксированный объём:** проверить готовый кандидат **ядра Tervon TCP Bridge Linux v1.0.2** на базе подтверждённого release v1.0.1 source commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86`, интегрированного и действительно тестированного RuntimePolicy с атомарным ACL+routes и точным полным policy ACK; затем при всех PASS собрать отдельный Linux binary в безопасном staging без установки/перезапуска production. Сама ветка M6 `prototype/` не является достоверным исходником выпущенного v1.0.1, так указано в `docs/M6_RUNTIME_ACL_OPTION2_20261121.ru.md`. Старый overlay `runtime-acl/apply-to-release-source.py` обеспечивает **только ACL** и подтверждение `M6_ACL_RELOAD`; он **не равен** полному RuntimePolicy v1.0.2 / `M6_POLICY_RELOAD` и не может быть принят как требуемый результат.

**Подтверждено ранее на Homeserver:** `PYTHON_22=PASS`, `XUI_GO_TEST_RACE_VET=PASS`, `PROTOTYPE_GO_TEST_RACE_VET=PASS`, `PANEL_XRAY_BRIDGE_TWO_ROUTES=PASS`. Этого недостаточно для утверждения выпуска v1.0.2, т.к. нужно доказать применение реального source overlay к точному release v1.0.1, интеграционные Go проверки и согласованный полный ACK (никаких сигналов живым Docker-контейнерам без утверждения). `V1_0_2_LINUX_BINARY=NOT_BUILT`, `PRODUCTION=UNCHANGED`, `RELEASE=NO`.

**Следующий шаг:** определить по актуальной ветке точный источник/инсталлятор full-policy v1.0.2 и проверить его контракт без использования устаревшего ACL-only; при отсутствии complete source integration сначала документировать конкретный блокер и согласовать минимальное исправление, а не запускать сборку другого кода и не подменять тесты выпуска source-level PASS.

## 2026-11-22 — Предрелизная проверка Linux Tervon TCP Bridge v1.0.2: BLOCKED (full RuntimePolicy source absent)

**Запрос пользователя:** комплексно проверить все релизные критерии и при успехе собрать именно Linux Tervon TCP Bridge `1.0.2`. **Проверенный GitHub HEAD:** `caf46e6aaeb05cb346fd7473f89fd977604fe12e`, ветка `work/m6-android-ingress`. Прочитаны утверждённый `docs/DEVELOPMENT_PLAN.md`, этот журнал, `docs/RELEASE.ru.md`, `docs/M6_RUNTIME_ACL_OPTION2_20261121.ru.md`, полный repository tree, `integrations/android-ingress/runtime-acl/apply-to-release-source.py`, `runtime-acl/acl_snapshot.go`, `xui/policy_generation.go`, `xui/route_sync.go` и `prototype/go.mod`.

**Критичный релизный блокер подтверждён по исходникам:** утверждённая v1.0.2 должна собираться на основе точного release-v1.0.1 commit `96c136702d10b6fb9a16aa2893f6b978e8b14e86` с атомарной единой RuntimePolicy generation для `routes + authorized` и реальным точным `M6_POLICY_RELOAD` ACK ядра. Единственный найденный release-source overlay `runtime-acl/apply-to-release-source.py` вызывает `acl.Replace(fresh.Authorized, c.Routes)`, **запрещает** изменения всех полей кроме `authorized` посредством `reflect.DeepEqual` и выводит только `M6_ACL_RELOAD=PASS digest=...`. `acl_snapshot.go` явно гарантирует, что `Routes remain immutable`. `xui/route_sync.go` и `xui/policy_generation.go` — coordinator seam/digest, не реализованный core writer/runtime ACK. В текущем дереве нет подтверждённой release integration атомарного core `routes+ACL` с `M6_POLICY_RELOAD`.

**Результат source readiness gate:** `LINUX_V1_0_2_FULL_POLICY_CORE=FAIL_MISSING`; `LINUX_V1_0_2_INTEGRATION_TEST=NOT_RUN`; `LINUX_V1_0_2_BUILD=NOT_RUN`; `LINUX_V1_0_2_RELEASE=NO`. Ранее подтверждённые `PYTHON_22=PASS`, `XUI_GO_TEST_RACE_VET=PASS`, `PROTOTYPE_GO_TEST_RACE_VET=PASS`, `THREE_WAY_PANEL_XRAY_BRIDGE=PASS` остаются локальными проверками другой функциональности и **не компенсируют** отсутствие full-policy core. Сборка из текущего `prototype/` или ACL-only overlay, ошибочно названная v1.0.2, **запрещена**. Не создавались бинарники, патчи или изменения действующих служб.

**Граница дальнейших действий:** отдельно согласовать реализацию недостающей единой RuntimePolicy только в проверенном release-v1.0.1 source copy, не менять механизм TLS/mTLS, framing, padding/jitter, Docker или процессы. Затем comprehensive Go test/race/vet и интеграционный тест atomic routes+ACL, fail-closed rollback, точного generation ACK; при всех PASS — Linux binary в отдельном staging без установки/подписи/публикации, SHA и commit в журнал. Если пользователь требует оставить утверждённую архитектуру без исходных изменений — остановиться и не выпускать 1.0.2.

## 2026-11-22 — Linux v1.0.2: completion of full RuntimePolicy as source-only release-v1.0.1 overlay (HOST VALIDATION PENDING)

**Explicit user approval:** «завершаем недостающее RuntimePolicy»; work strictly within the Linux Bridge kernel, not Android/APK, without modifying running services or changing the existing TLS/mTLS, framing, Padding/Jitter, build/signing/publishing configuration. Current branch `work/m6-android-ingress`, implementation HEAD prior to this journal `a23acc669f5592fc92a119c2b8a3f17db24add67`. Reviewed actual plan, journal, released `prototype/bridge.go` (blob `45860c4ed5eff7e52321a98e2b4bc6d82e350d98`), `prototype/cmd/bridge/main.go` (blob `b2349a4179d6a23b9a5fec3160d12300be0e23d1`) from exact release base `96c136702d10b6fb9a16aa2893f6b978e8b14e86` and the existing ACL-only overlay and coordinator digest/ACK code.

**Source-only implementation commits:** `bf6e8d1f798964db0bf824115ca6490609926044`, `e8b8d2da2245e5a8ca484d93a3e177dab9dfddf8`, `28a61bd8d99d03b7ba161871a15f16bbfa43719f`, `caae875e23596ea9eba2cd8c08e45f718df0b031`, `d13cf9c88e31a0f4bd25f059d6096050687419f7`, `2ce3b8feda01eb88f0bb378d5357f21d749843f1`, `a23acc669f5592fc92a119c2b8a3f17db24add67`.

**New files (five):** `integrations/android-ingress/runtime-acl/runtime_policy.go`, `runtime_policy_test.go`, `runtime_policy_stream_test.go`, `apply-runtime-policy-to-release-source.py`, `check-and-build-linux-v1.0.2.sh`.

**Design:** keep released v1.0.1 stream framing/TLS and original fallback paths unmodified. Full `bridge.RuntimePolicy` holds **one** `atomic.Pointer` to the immutable pair `routes + mTLS authorized`. Every NEW HTTP/2 stream takes one generation via `PolicyLookup(fingerprint, service)`, which returns both admission and backends; no independently loaded ACL/routes snapshots. Return copied backend list and retain already selected socket for ongoing connections. Validate full candidate before store; rejected update preserves previous snapshot and digest. Digest uses the same Go JSON shape/order `{routes, authorized}` as existing `xui.PolicyGenerationDigest`, SHA256 hex. `SIGHUP` handler of staged **release v1.0.1 main** loads full strict server.json, rejects changes to all static parameters (including TLS, limits, padding/jitter) and applies both routes + authorized atomically; log `M6_POLICY_RELOAD=PASS digest=...` only after successful snapshot publication. Legacy ACL-only overlay `apply-to-release-source.py` remains separate and MUST NOT be combined into this source staging.

**Tests included:** source-level policy digest/validation, atomic replace/invalid rollback, race/concurrent lookup+swap, true TLS1.3 HTTP/2 end-to-end stream continuity where established connection keeps backend A, new connection uses backend B, revoked certificate cannot start new stream but existing stream survives, then readmission. Exact SHA/ACK production coordinator not yet tested with actual running candidate; Go test/race/vet and staging build are pending.

**Source preflight already inspected via GitHub API:** all **six** replacement anchors in untouched release-v1.0.1 bridge.go/main.go exist **exactly once** (policy field, pick, admission, dial, reflect import, core handler). This is only source-anchor inspection, **not** Python `--check`, Go compilation or functional PASS.

**Single gated Homeserver runner:** `integrations/android-ingress/runtime-acl/check-and-build-linux-v1.0.2.sh`. It verifies source branch/SSH remote/clean tracked checkout/origin SHA, exact v1.0.1 release commit, Linux amd64 host and an unused staging directory `/opt/tervon/Developed/v1.0.2-runtime-policy-validation`; creates detached worktree at release commit, runs guarded `--check` then source-only `--apply`, `gofmt`, full Go `test -count=1`, `test -race -count=1`, `vet ./...`; **only if all PASS** builds separate `tervon-bridge-linux-amd64-v1.0.2-candidate`, prints SHA256 and precise candidate source HEAD. No install, no signal, no Docker/systemd, no change to running services. An unsuccessful test leaves candidate unbuilt; partial staged source may remain for inspection, not production.

**Statuses at journal commit:** `FULL_RUNTIME_POLICY=SOURCE_COMMITTED`, `EXACT_RELEASE_ANCHORS=PASS_STATIC_REVIEW`, `GUARDED_RELEASE_APPLY=NOT_RUN_ON_HOST`, `GO_TEST=NOT_RUN_ON_NEW_SOURCE`, `GO_RACE=NOT_RUN_ON_NEW_SOURCE`, `GO_VET=NOT_RUN_ON_NEW_SOURCE`, `REAL_PROCESS_POLICY_ACK=NOT_TESTED`, `V1_0_2_LINUX_BINARY=NOT_BUILT`, `RELEASE=NO`, `PRODUCTION=UNCHANGED`. Do not call it PASS/released or record SHA before tests actually run. Continue same module if failure, no unauthorized architectural changes.

**Дополнительная защита кандидатного source** (commit `ae4a277e8f749514a6d8ec485fe799b5682380fb`): отдельный test `TestM6FullPolicyChangesRoutesAndAuthorizationTogether` проверяет одновременное изменение имени службы, адреса и ACL, отказ старой службе, а также неизменность поколения при несовместимой паре `routes/ACL`. Включён в тот же единственный комплексный прогон. `HOST_TEST=NOT_RUN`, `RELEASE=NO`.

## 2026-11-22 — Linux Bridge v1.0.2 RuntimePolicy: Homeserver gated test/race/vet and candidate build PASS

**Операторский протокол:** branch `work/m6-android-ingress`; точный испытанный SOURCE_HEAD `40dc00a24fb43ea05e1adf4a62845cb5546ca1bd`; `git pull --ff-only` по SSH fast-forward `100880f..40dc00a`; неизменённый release source base `96c136702d10b6fb9a16aa2893f6b978e8b14e86` в отдельном detached worktree `/opt/tervon/Developed/v1.0.2-runtime-policy-validation`. Запуск `bash integrations/android-ingress/runtime-acl/check-and-build-linux-v1.0.2.sh`; `VALIDATION_BUILD_RC=0`.

**Комплексные шаги с реальными результатами:** `SOURCE GATE=PASS`; `M6_FULL_POLICY_SOURCE_PREFLIGHT=PASS` (`--check`), `SOURCE_MODIFIED=NO`; `--apply`: `M6_FULL_POLICY_SOURCE_PREFLIGHT=PASS`, `M6_FULL_POLICY_SOURCE_STAGED=YES`. `go test -count=1 -timeout=240s ./...`: **PASS, 6 пакетов** (`tervon-tcp-bridge`, `cmd/bridge`, `framing`, `jitter`, `padding`, `telemetry`). `go test -race -count=1 -timeout=300s ./...`: **PASS, те же 6 пакетов**. `go vet ./...`: **PASS**. При всех PASS собран только Linux amd64 кандидат, `LINUX_V1_0_2_BUILD=PASS_CANDIDATE_ONLY`.

**Кандидатный бинарник:** `/opt/tervon/Developed/v1.0.2-runtime-policy-validation/tervon-bridge-linux-amd64-v1.0.2-candidate`. `SHA256=aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371`. Относится к overlay на **точный release base**, не к прежнему `prototype/` M6-ветки. Версионные linker flags: `main.buildVersion=1.0.2`, `main.buildCommit=40dc00a24fb43ea05e1adf4a62845cb5546ca1bd`.

**Подтверждено:** атомарное обновление routes+ACL на уровне unit и тестового HTTP/2+TLS 1.3 stream, rollback отвергнутой конфигурации, разные backend для старых и новых соединений, конкурентные обновления под race detector; compile/vet в указанной staging-среде. **Не подтверждено:** реальный `SIGHUP` на отдельном запущенном экземпляре кандидата, реальный `M6_POLICY_RELOAD=PASS digest=...` и подтверждение exact generation ACK от процесса через full route+ACL coordinator. До этого `RELEASE=NO`, `PRODUCTION_DEPLOYMENT=NO`. Настоящие службы не тронуты (`RUNNING_SERVICES=UNCHANGED`, `PRODUCTION=UNCHANGED`, `SSH_CONSOLE=OPEN`).

**Следующий шаг того же модуля:** отдельно в изолированном staging, без сигналов production, проверить живой кандидат на полное чтение JSON при `SIGHUP`, атомарное применение новой пары routes+ACL, отклонение неверной конфигурации с сохранением действующей политики, точный digest/ACK и отсутствие разрыва существующих потоков. Не повторять уже прошедший пакет тестов без изменения кода или установленной причины. Release, установка, перезапуск и публикация только после отдельного подтверждения соответствующих критериев.

## 2026-11-22 — Linux v1.0.2: isolated built-binary RuntimePolicy ACK test prepared (execution pending)

**Operator approved:** run remaining runtime verification to unlock Linux GitHub v1.0.2 release. Inspected current branch and journal, candidate actual full-policy source overlay and older independent `runtime-routes/policy_ack_integration_test.go`. Existing old test's expected fixed-change error differs from candidate's actual `M6_POLICY_RELOAD=FAIL non-policy configuration changed`; do not reuse/mix the old overlay/test. **New read-only-to-production test runner**, commit `e52c76c11ef9c432a4718f8a8a27f1ef2c9074ae`: `integrations/android-ingress/runtime-acl/check-runtime-policy-linux-v1.0.2.py` (blob `2596d7002f07ee233a18ae5aed47cbc75b442ff3`).

**Runtime check contract:** before launching, verify actual executable SHA256 `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371`; use `tempfile.TemporaryDirectory` exclusively below `/opt/tervon/Developed` with 0700 perms; create ephemeral P-256 local test certificate via existing OpenSSL, ephemeral `127.0.0.1` listener, invalid/nonexistent loopback backend addresses (not connected); launch only the checked candidate as a child, send `SIGHUP` **only to its own PID**; test valid combined route+ACL replacement with exact canonical Go JSON generation digest, reject mismatched ACL, reject modified static max-streams, reject malformed JSON, then restore initial policy and verify exact second generation ACK. Count ACKs to reject false-positive success; check process remains alive; always terminate owned child and clean ephemeral files even on failure. No production Bridge, certificates, Xray, Docker/systemd, repository sources, Android app or release assets modified.

**Evidence boundary:** this test demonstrates real process accepted/rejected policy reload with exact generation log; absence of ACK on rejection is not by itself a complete end-to-end proof of retained previous authorization, which is already covered by current source-level integration tests. Core Go test/race/vet and Linux amd64 candidate build previously PASS; no reason to repeat. `HOST_RUNTIME_TEST=NOT_RUN`, `RELEASE=NO`. After operator results, if PASS, record precise stdout and proceed to GitHub release upload workflow only if supported; otherwise fix in same module without touching working services.

## 2026-11-22 — isolated v1.0.2 RuntimePolicy process test: partial PASS, negative-case test expectation corrected

**Operator actual runtime:** source `aa227302464b48680b993775ed284c55db038b35`, `SYNTAX_RC=0`, Linux candidate binary SHA256 `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371` verified, `ISOLATED_PROCESS=PASS`, `ROUTES_AND_ACL_RELOAD=PASS`, `EXACT_GENERATION_ACK=PASS`. After this the test returned `LINUX_V1_0_2_RUNTIME_ACK=FAIL`, `REASON=EXPECTED_ACK_OR_REJECTION_NOT_OBSERVED`, `RUNTIME_RC=1`; `PRODUCTION=UNCHANGED`. This is **NOT a complete runtime PASS** and must not trigger a release.

**Reviewed exact source and root cause of test failure:** original negative fixture `routes_b={"vpn-new":...}`, `authorized=auth_a={"a"*64:["vpn"]}` violates `cmd/bridge/main.go load()` validation (`authorized service has no configured route`). Therefore staged core logs `M6_POLICY_RELOAD=FAIL invalid config` at strict load stage, **not** `FAIL invalid routes or ACL` from `RuntimePolicy.Replace()`. The test awaited the wrong marker and timed out after the prior good generation. No evidence from this output of failed legitimate hot policy update or unexpectedly exiting process.

**Test-only correction:** `check-runtime-policy-linux-v1.0.2.py`, commits `655ec9f451f09cc0b8cf3dc6c37a614d70789b70` and `8013ac022014f06b8f5bf9c20465f985a70ef9a9`. Expect first-stage `FAIL invalid config` for inconsistent ACL service; separately supply two case-variant hex fingerprints `b*64` and `B*64`, both mapped to existing `vpn-new`, which `load()` individually accepts but `RuntimePolicy` rejects as same decoded identity, requiring `M6_POLICY_RELOAD=FAIL invalid routes or ACL`. New malformed-JSON test requires fresh second `FAIL invalid config` event instead of matching the previous rejection. All negative tests assert no extra successful ACK; restoring valid config requires second precise generation digest.

**Status:** `RUNTIME_GOOD_GENERATION=PASS_OBSERVED`, `RUNTIME_NEGATIVE_SUITE=FAIL_TEST_EXPECTATION_AT_OLD_SOURCE`, `CORRECTED_TEST_SOURCE=COMMITTED`, `CORRECTED_TEST_EXECUTION=PENDING`, `LINUX_BINARY=UNCHANGED`, `ANDROID_BINARY=UNCHANGED`, `RELEASE=NO`. Run only corrected isolated test, not Go test/race/vet/build again. No Docker/systemd/production service signals.

## 2026-11-22 — v1.0.2 Linux candidate: isolated real-process RuntimePolicy full suite PASS

**Homeserver operator output:** `SOURCE_HEAD=1ce3bcdabff00d7af8527ca3ac76a785097e2520` after SSH fast-forward `aa22730..1ce3bcd`; `SYNTAX_RC=0`, `CANDIDATE_SHA256=PASS`, `ISOLATED_PROCESS=PASS`, `ROUTES_AND_ACL_RELOAD=PASS`, `EXACT_GENERATION_ACK=PASS`, `INCONSISTENT_POLICY_REJECTED_BY_LOADER=PASS`, `INVALID_POLICY_REJECTED_BY_CORE=PASS`, `NON_POLICY_CHANGE_REJECTED=PASS`, `MALFORMED_CONFIG_REJECTED=PASS`, `RESTORE_GENERATION_ACK=PASS`, `PROCESS_ALIVE_AFTER_RELOAD=PASS`, `ISOLATED_PROCESS_CLEANUP=PASS`, `LINUX_V1_0_2_RUNTIME_ACK=PASS_ISOLATED_PROCESS`, `RUNTIME_RC=0`, `PRODUCTION=UNCHANGED`, `RUNNING_SERVICES=UNCHANGED`, `SSH_CONSOLE=OPEN`.

**Verification scope:** actual previously compiled binary, not rebuilt; immutable Linux amd64 candidate hash `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371`; overlay source commit `40dc00a24fb43ea05e1adf4a62845cb5546ca1bd` atop exact release v1.0.1 base `96c136702d10b6fb9a16aa2893f6b978e8b14e86`. Earlier complete Go test/race/vet plus Linux candidate build PASS documented; Android ARM64 crosscompile PASS SHA `d71607005c510ec1d97787f656b4b5580d10d750ec1031b9b18630af266ac487`. No production deployment or Android on-device v1.0.2 end-to-end validation inferred.

**Release gate:** `LINUX_RUNTIME_POLICY_ACK=PASS`; `LINUX_RELEASE_CANDIDATE_VALIDATED=PASS` for approved isolated tests; **`GITHUB_RELEASE_V1_0_2=NOT_YET_PUBLISHED`**, `INSTALL=NO`, `PRODUCTION=UNCHANGED`. Next operator-approved task remains publishing versioned Linux amd64 asset with precise SHA256, source provenance and reproducibility notes via supported GitHub release workflow; do not invent a Release URL or report publication until actual GitHub response confirms it.

## 2026-11-22 — v1.0.2 Linux GitHub Release packaging prepared; publication via Homeserver CLI pending

**Operator approved:** «Переходим к публикации GitHub Release v1.0.2». Linux AMD64 only. Operator reports Android `libtervonbridge.so` already placed in the Android app source; **no change to that app and no APK build** in this release action. GitHub Releases listed `v1.0.0` and `v1.0.1`, no v1.0.2 at time of check.

**Source/test evidence:** candidate tested from `40dc00a24fb43ea05e1adf4a62845cb5546ca1bd` overlay on release source `96c136702d10b6fb9a16aa2893f6b978e8b14e86`. Linux binary source staging SHA256 `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371`, Go test/race/vet PASS, real isolated-process SIGHUP and digest ACK PASS at runtime test source `1ce3bcdabff00d7af8527ca3ac76a785097e2520` with `RUNTIME_RC=0`. Working services unchanged.

**Release additions:** `docs/RELEASE_NOTES_v1.0.2.ru.md` (commit `c201304551c1dd4ebbc52213fc6e9c57081a0c34`) and `integrations/android-ingress/runtime-acl/publish-linux-v1.0.2.sh` (commit `db79402d0b2a7869c3548d106d5108dbfe2fab94`). Script requires SSH origin, correct branch/clean Git checkout, SHA-confirmed existing Linux executable, the exact v1.0.1 base/staged policy source and up-to-date branch, absence of v1.0.2 tag/release, authenticated `gh`, isolated staging destination. It publishes `tervon-bridge-linux-amd64-v1.0.2`, `SHA256SUMS`, `BUILD_INFO`, release notes to the GitHub Release `v1.0.2`, then downloads published binary and verifies hash. No rebuild, installation, Docker/systemd, Android or live server changes. The release notes explicitly document that the tag contains **overlay recipe and documentation**, not a pre-materialized full patched release tree.

**Status now:** `LINUX_RELEASE_SOURCE_AND_CHECKS=PASS`, `RELEASE_NOTES=COMMITTED`, `PUBLISH_RUNNER=COMMITTED`, **`GITHUB_RELEASE_V1_0_2=NOT_YET_CREATED`**, `ASSETS_UPLOADED=NOT_TESTED`, `PRODUCTION=UNCHANGED`. This assistant's GitHub connector does not provide release-create/upload binary asset action and the Homeserver's GitHub CLI is only available to the operator; must wait for the operator-run command before claiming published PASS. Next confirmed stage after Linux publication: separate Android core integration audit under the approved plan.

## 2026-11-22 — Docker v1.0.2: image build, isolated startup and production cutover reported PASS; persistent Compose override committed

**Operator evidence (Homeserver):** previously published asset `tervon-bridge-linux-amd64-v1.0.2` SHA256 `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371` verified. Existing image `tervon-bridge-android-ingress:m6-runtime-acl` uses `/tervon-bridge` under UID/GID 65532 and read-only rootfs; new Linux v1.0.2 binary dynamically links `libc.so.6`, requiring glibc up to 2.34, whereas the previous image had no `/lib64/ld-linux-x86-64.so.2`. An independent `debian:12-slim` image was built with the unchanged published binary and `ca-certificates`, tagged `tervon-bridge-android-ingress:v1.0.2`. Image ID `sha256:ce484cdc64c56428e89ecc933e64616ec4986004bb8a28bf2ce953b7766353a9`; binary hash inside image matched GitHub asset; dynamic dependency loader resolved successfully; `BUILD_RC=0`.

**Isolated startup:** a network-none, read-only container using existing read-only `/config` mount and UID/GID 65532 remained running; process liveness and absence of published ports confirmed; probe cleaned up. `V1_0_2_ISOLATED_STARTUP=PASS`, `PROBE_RC=0`, production unchanged.

**Production cutover (operator output):** Compose `up -d --no-deps --force-recreate android-bridge-decapsulator` using a temporary image override, preflight preserving live `0.0.0.0:18443` mapping. Running container `android-ingress-android-bridge-decapsulator-1` verified with new image ID, read-only root filesystem, restart policy `unless-stopped`, and local TCP socket accepted on 127.0.0.1:18443. `V1_0_2_DOCKER_CUTOVER=PASS`, `CUTOVER_RC=0`. Old image retained; two M6 probes and other containers not targeted by Compose command.

**Persistent Compose reconciliation:** temporary cutover override was deleted automatically, while committed overlay still selected prior `m6-runtime-acl` image and base Compose defaulted to 127.0.0.1. The existing `integrations/android-ingress/runtime-acl/compose.hot-acl.yaml` was therefore updated at commit `bb6537fa6c2c965348e0d221d93d00373b65ac0b` to select `v1.0.2` and pin the existing live `0.0.0.0:18443:8443/tcp` binding. The base Compose file, container, certificate/configuration mounts, Xray, 3x-ui, Android, and RU/FI Bridge were not changed by this GitHub documentation/config commit.

**Status boundary:** `IMAGE_BUILD=PASS`, `ISOLATED_STARTUP=PASS`, `DOCKER_CUTOVER=PASS` (operator evidence). `PERSISTED_COMPOSE_SOURCE=COMMITTED`, but `COMPOSE_EFFECTIVE_CONFIG_AFTER_PULL=NOT_YET_CHECKED`. Actual client-authenticated mTLS/HTTP2, existing backend data transfer, and live policy reload have **not** been verified in production by these outputs. `PRODUCTION_FUNCTIONAL_PASS=NOT_YET_CONFIRMED`; do not close this deployment stage or claim end-to-end PASS until evidence is collected. Do not rerun Go unit/race/vet or image build without cause. Next: operator pulls the new branch commit, performs read-only effective Compose and live traffic checks; record results in GitHub before declaring complete.

## 2026-11-22 — Docker Compose production-bind duplication: diagnosed and corrected in source

**Operator read-only effective Compose:** after pull to `b028884b983a692eec84ad0025f39197c9ec530e`, `docker compose config --format json` showed two different bindings for the same container port 8443/published 18443: `127.0.0.1:18443` inherited from base Compose and `0.0.0.0:18443` appended by hot ACL override. Thus `len(s["ports"]) == 1` failed and `FINAL_CHECK_RC=1`. This was a Compose list-merge configuration defect, **not evidence of a v1.0.2 runtime failure**. Running production was unchanged by read-only check. The Python Apport excepthook error is secondary to the original assertion failure.

**Source correction:** `integrations/android-ingress/runtime-acl/compose.hot-acl.yaml`, commit `ac1e220d7e23316d516a4898bcb648257bf00274`, replaces inherited port list using `ports: !override` with the existing production mapping `0.0.0.0:18443:8443/tcp`. This requires Docker Compose >= 2.24.4; no unverified assumption about installed Compose version. Base Compose retains its secure localhost default for other uses. No changes to container runtime or service at this commit.

**Gate:** `COMPOSE_CONFIG_FIX=SOURCE_COMMITTED`, `EFFECTIVE_CONFIG_PASS=PENDING_OPERATOR_CHECK`, `PRODUCTION_FUNCTIONAL_PASS=NOT_YET_CONFIRMED`. Next read-only operator check: `docker compose version`, pull commit, `docker compose config` and assert exact single port + image; inspect running container and journal without exposing secrets. If Compose version does not support `!override`, stop and agree a compatible persistent port configuration; do not use `compose up`. Authenticated mTLS/HTTP2 and backend traffic remain separate requirements for full operational PASS.

## 2026-11-22 — Docker v1.0.2: persisted Compose config PASS after fix

**Homeserver operator evidence:** SSH Git fast-forward `b028884..36864cb`, exact `SOURCE_HEAD=36864cbc015058bf17abd52cedfafce5b00c91c7`. Docker Compose version `v5.5.1`. Using base `integrations/android-ingress/compose.yaml` and hot ACL override `integrations/android-ingress/runtime-acl/compose.hot-acl.yaml`, read-only `docker compose ... config --format json` confirmed exactly one published mapping `0.0.0.0:18443 -> 8443/tcp`, image `tervon-bridge-android-ingress:v1.0.2`, read-only filesystem, user `65532:65532`, restart policy `unless-stopped`, and `pull_policy: never`; `COMPOSE_CONFIG=PASS`.

Running main container remained `RUNNING=true` with image ID `sha256:ce484cdc64c56428e89ecc933e64616ec4986004bb8a28bf2ce953b7766353a9`, port bindings `0.0.0.0:18443 -> 8443/tcp`, `RESTART_COUNT=0`. `COMPOSE_PERSISTENCE=PASS`, `FINAL_CHECK_RC=0`, `PRODUCTION=UNCHANGED` during the final read-only check. This verifies successful persistent Compose reconciliation after the earlier duplicate-port FAIL. No repeat of already passed build or isolated startup tests was needed.

**Status:** `LINUX_V1_0_2_DOCKER_IMAGE=PASS`, `DOCKER_CUTOVER=PASS`, `COMPOSE_PERSISTENCE=PASS`; Docker update/configuration is complete. **Not yet evidenced:** authenticated real-client mTLS/HTTP2 sessions, backend application data transfer, and production hot-policy reload; `PRODUCTION_E2E_PASS=NOT_CONFIRMED`. Do not conflate Docker deployment PASS with functional end-to-end PASS. Old `m6-runtime-acl` image preserved for rollback as reported in original cutover output. Follow established plan before further operational tests; Android remains untouched.
