# M6 — журнал разработки динамической маршрутизации inbound 3x-ui

**Область:** серверная маршрутизация Tervon TCP Bridge M6; не Tervon Federation.  
**Технический контракт:** [M6_3XUI_DYNAMIC_ROUTING.md](M6_3XUI_DYNAMIC_ROUTING.md).  
**Ветка:** `work/m6-android-ingress`. **HEAD до создания журнала:** `cbaab8a994b753e91c3c4298fd2d0a07d15ac014`.  
**Принцип:** ниже исторические события и результаты в их исходной области. Исторический PASS не означает нынешний E2E PASS. Источники — `docs/Server/V1_0_2_DEVELOPMENT_LOG.md`, `docs/Server/M6_SERVER_AUDIT_EXTRACT_2026-11-22.md`, проверенные исходники ветки и операторские результаты 22–24 ноября. Старые документы не являются действующим маршрутизационным контрактом.

## 2026-11-22 — серверная база, исходный конфликт маршрутов

- **УТВЕРЖДЕНО:** Bridge переносит непрозрачные TCP-байты через HTTP/2 + TLS 1.3 + mTLS; 3x-ui API — источник TCP inbound, Xray runtime — обязательное подтверждение; `server.json` — результирующая политика. Отдельная `tcp_routes` отменена. Запрещены статические VPN-имена как источник маршрутов.
- **АУДИТ:** `ListInbounds`, `ReconcileInbounds`, `SyncVerifiedRoutes`, `ConfirmedRouteStore` присутствуют в исходниках, но существование компонентов не доказывает их вызов работающим Registrar. Старый аудит описывает `configuredTCPRoutes()`; это историческое описание, **не** актуальный исходный код после дальнейших изменений.
- **ОПЕРАТОРСКИЕ ДАННЫЕ:** три нужных TCP inbound — VLESS TCP `30000`, VLESS XHTTP/splithttp `55535`, Trojan TCP `55636`. Исторический `server.json` содержал только `vpn → 30000`, `vpn-xhttp → 55636`; маршрута `55535` не было. Наличие listener не означает E2E.
- **СТАТУС:** конфликт API-first ↔ статическая политика установлен; автоматическая синхронизация не подтверждена.

## 2026-11-23 — динамический селектор ядра и испытания исходников

- **РЕАЛИЗОВАНО В ИСХОДНИКАХ:** `RuntimePolicy.Lookup` поддерживает канонический числовой TCP-порт в `Tervon-Service` и ищет backend только в текущем fingerprint-authorized поколении; неизвестные, неоднозначные и неразрешённые значения отклоняются. Старые имена сервисов ещё поддерживались для совместимости, но не являются целевой схемой автоматического discovery.
- **ТЕСТЫ:** три маршрута `30000/55535/55636`, смена поколения, три HTTP/2+mTLS соединения с echo-backend и 116 синтетических отпечатков. Source commit `ee113cf1328ee32da1ff28493a67e919c7314129`; CI `45134204649` — PASS.
- **ГРАНИЦА:** SOURCE_CI PASS, не доказана автоматическая публикация трёх маршрутов в live Docker и Android E2E.

## 2026-11-23 — сборка и установка Android-facing Decapsulator

- **ПРОВЕРКА ИСХОДНИКОВ:** candidate на базе release `96c136702d10b6fb9a16aa2893f6b978e8b14e86`; source `e132609401b90c3c28054590cd2435d191c42dd1`. CI `45137007433` — PASS для runtime-policy, ACL и registrar-source.
- **HOMESERVER BUILD:** source `c1cd963aab7de3eb4c5888db3c1c6d910d711fe2`; Go test/race/vet и сборка подтверждены оператором. Candidate image `tervon-bridge-android-ingress:m6-dynamic-e1326094`, image ID `sha256:26415f7f60e9138a397b15865783dda894a2eb85a2e65f0afa4bd0dc3a9e690e`. Установленная версия Registrar привязана к точному Docker container ID/image; при cutover требуется rebind, иначе enrollment/ACK не работают.
- **CI ДОПУСКА IMAGE:** commit `d9350eaddd5086b7a2d978e280a08bb5e49229d0`, runs `45141157550` и `45141157533` — PASS.
- **CUTOVER ПО ОПЕРАТОРСКОМУ ВЫВОДУ:** source `8cd9a04264c2d03244adcf6c5d5960101c3e2f22`; только Compose service `android-bridge-decapsulator` пересоздан, образ запущен. `M6_DOCKER_CUTOVER=PASS`; SIGHUP `M6_POLICY_RELOAD=PASS`; `M6_POLICY_ACK=PASS` в пределах проверенного скриптом подтверждения; Registrar rebind и active service `M6_REGISTRAR_REBIND=PASS`.
- **ВАЖНО:** ACK этой операции не был независимым доказательством совпадения live политики с тремя inbound. `55535` оставался не подтверждён в политике. Android интернет/E2E — не PASS.

## 2026-11-23 — фильтрация, единый coordinator, API adapter

- **ФИЛЬТР:** `FilterBridgePanelInbounds` исключает SOCKS, HTTP proxy, dokodemo-door и неизвестные inbound protocol; разрешает VPN TCP (`vless`, `trojan`, `shadowsocks`), различая inbound protocol и Xray network. Source `6a414dce0a154bec41cf04d2cca7c8e5fa92aae7`; CI `45177282293` и `45177282427` — PASS.
- **ЕДИНЫЙ КООРДИНАТОР:** `RuntimePolicyCoordinator` создан в `xui/route_sync.go`; общий mutex сериализует enrollment и sync в одном процессе, не заменяет межпроцессную блокировку. Commits `7f9a82842031c300e1e77aead57064765a69c676`, `86f118de70a65f91971c25e2c131f6bf73c8e332`. На этом этапе Registrar ещё не подключён.
- **TCP FILTER:** commits `dda4ffcba5f6ab81bbb4bf2a5ad1391af0e6f650`, `20d44a4f69d526dc3588bb7abc26808ed875df80`; UDP/QUIC исключаются, TCP XHTTP остаётся допустимым.
- **3x-ui ADAPTER:** `Client.SelectedBridgeInbounds(ctx)` использует существующие `ListInbounds` и фильтр, без нового API-клиента и жёстких портов. Commits `0457c5251c3152383aa52beebccb5d62bd913c64`, `d8c140c4973dc522226c1f1cb3dd99d7070a52d1`; CI `45187951474` и `45187951809` — PASS. Эти CI — только исходники, не live sync.

## 2026-11-24 — Xray evidence, регрессия, ACL и owner

- **XRAY CONFIGURED FILTER:** `ParseXrayConfiguredInbounds` исключает служебные listener до backend resolver. Commits `818cfe1906df343789a5fe785ae10e52c1bdc301`, `89d1ae5782a0f4f8eee666b3190629496aa9f8f8`. Парсер конфигурации — **CONFIGURED_ONLY**, не доказательство live Xray.
- **РЕГРЕССИЯ И ИСПРАВЛЕНИЕ:** CI `45189009053` и `45189008872` — FAIL из-за экранированного переноса строки в комментарии, скрывшего проверки eligibility/duplicate. Исправлено commit `7387cc1b0e96b859d5554e0a5a176fcfb50c5a62`; последующие CI `45281699543`, `45281699583` — PASS.
- **АВТОМАТИЧЕСКИЙ ACL:** пользователь утвердил, что все уже зарегистрированные fingerprint получают текущий подтверждённый набор TCP route services при добавлении/удалении inbound; новые fingerprint из панели не создаются. `SyncVerifiedRoutes` обновляет ACL в том же поколении, исходную карту не мутирует. Source `5a7a3bdb37b74e148835abc3999d160f676deb40`; тест двух поколений `1b27127d2d50bcf18d60eac0ff9a1a74a49aa9bc`. CI `45282774970` и `45282775087` — PASS для source.
- **РЕШЕНИЕ О ВЛАДЕЛЬЦЕ:** только существующий `m6-auto-registrar` управляет enrollment и verified routes/ACL. Ни второго демона, ни независимого writer. В `main.go` на проверенном source Registrar вызывает `HotACLActivator.Activate` напрямую и читает `configuredServiceNames()` из `server.json` при запуске; coordinator не подключён. `AppliedXrayEvidence` и `RoutePolicyApply` не имеют подтверждённой production-привязки. LIVE SYNC — BLOCKED.

## 2026-11-23–24 — фактическая диагностика маршрутов и ошибок

Эти наблюдения получены из операторских выводов в текущей работе; точные SHA установленного Registrar и момент каждой команды **не верифицированы по одному полному runtime provenance**.

- **ОПЕРАТОРСКОЕ НАБЛЮДЕНИЕ:** Trojan TCP `55636` и VLESS TCP `30000` работали через Bridge; VLESS XHTTP `55535` через Bridge не работал. Не считать это общим PASS всех протоколов.
- **READ-ONLY PANEL/XRAY:** API 3x-ui и Xray gRPC показали соответствие трёх inbound `30000`/`55535`/`55636`; listener Xray `55535` присутствовал и был доступен в отдельной проверке. Это не доказывает публикацию `55535` в Bridge policy.
- **СБОЙ REGISTRAR:** в ходе предыдущих действий Registrar переставал работать, после чего оператор подтвердил восстановление процесса. Не приписывать причину конкретному изменению без журналов процесса и commit.
- **ОШИБКА ДИАГНОСТИКИ:** вспомогательный `docker run --network container:<id> ... host.docker.internal` сообщил `FAIL_OR_UNVERIFIED` для портов, но проверка была некорректной: auxiliary container не унаследовал исходную конфигурацию `extra_hosts`, ошибки были подавлены. **Эти FAIL не являются доказательством недоступности Xray.** Сетевую конфигурацию и Xray на их основании не менять.
- **API CREDENTIAL:** оператор подтвердил формат существующего root-protected файла `Authorization: Bearer …`. `xui.NewClientFromHeader()` уже поддерживает этот формат. Не создавать новую авторизацию, не публиковать токен/URL.
- **ТЕКУЩИЙ НЕПОДТВЕРЖДЁННЫЙ РИСК:** `SyncVerifiedRoutes` инвалидирует snapshot при ошибке чтения API; временная недоступность API и неопределённый исход записи нового поколения должны быть различены. Поведение live Registrar пока не проверено.

## 2026-11-24 — выделение актуального маршрутизационного контракта

- **СОГЛАСОВАНО:** перестать использовать исторические документы как активное описание маршрутизации; создать отдельный серверный контракт и отдельную хронологию.
- **СОЗДАНО:** `docs/Server/M6_3XUI_DYNAMIC_ROUTING.md`, commit `cbaab8a994b753e91c3c4298fd2d0a07d15ac014`; только документация.
- **ЭТОТ ЖУРНАЛ:** перенос подтверждённой истории без изменения исходников и служб. Старые журналы сохранены как доказательства событий, но текущий контракт — новый документ.

## Открытые задачи и строгие границы

1. Проверить точные существующие Homeserver bindings: base URL 3x-ui, active Xray gRPC evidence, доверенный backend address resolver, целевой Docker container/config, текущий runtime policy digest и SHA установленного Registrar — **без публикации секретов**.
2. Внутри единственного `m6-auto-registrar` подключить уже существующие `SelectedBridgeInbounds`, `RuntimePolicyCoordinator`, `ConfirmedRouteStore`, `RoutePolicyApply` и live `AppliedXrayEvidence`. Не дублировать механизмы.
3. Атомарно применять подтверждённый набор портов и ACL, без `vpn`/`vpn-xhttp` hardcode; при временном сбое API не считать inbound удалённым; при неопределённом ACK не рекламировать неподтверждённое поколение.
4. Комплексные тесты API/Xray/backend/ACK, race, одновременные клиенты и смена поколений; затем согласованные Homeserver и Android E2E. Не назначать произвольные нагрузочные пороги.
5. Документировать commit, конкретные проверки и PASS/FAIL после реализации. **Нынешний статус: SOURCE COMPONENTS PRESENT; LIVE AUTO SYNC NOT PASS; XHTTP 55535 NOT PASS; PRODUCTION UNCHANGED ЭТИМ ДОКУМЕНТАЦИОННЫМ ЭТАПОМ.**

**Файлы, изменённые данным этапом:** только этот новый журнал. **Тесты:** документационная сверка GitHub/операторской истории; код и runtime не тестировались этим этапом. **Воздействие на службы:** NONE.

## 2026-11-24 — устранение потери подтверждённого Discovery при временном отказе источников (SOURCE)

- **УТВЕРЖДЁННЫЙ КОНТРАКТ:** изолировать временные отказы 3x-ui API и Xray evidence; не приравнивать их к подтверждённому удалению inbound. Не публиковать неподтверждённые новые маршруты; неопределённый исход policy apply/ACK по-прежнему закрывает новое Discovery.
- **ИЗМЕНЕНО:** `integrations/android-ingress/xui/route_sync.go`: ошибки вызова `panel(ctx)` и `applied(ctx)` теперь возвращают ошибку синхронизации **без** `ConfirmedRouteStore.Invalidate()`. Ошибки фактического reconciliation, недостающие подтверждённые inbound и неоднозначный policy ACK по-прежнему обрабатываются fail-closed. Старое поколение сохраняется только если оно ранее получило ACK.
- **ТЕСТЫ ИЗМЕНЕНЫ:** `integrations/android-ingress/xui/route_sync_test.go`: после успешного ACK временные сбои panel и Xray read сохраняют подтверждённый snapshot; ошибка ACK его инвалидирует.
- **SOURCE commits:** `a02652f8316b7288d4e3cd9dabd63c548f07f93c` (code), `1d348a1415f1cc6cc4a2dcff5bdba500797307b5` (tests). GitHub Actions на момент записи: runs `45288755257`, `45288755297` запущены на **предыдущем** code commit `a02652f`, поэтому **не доказывают PASS тестового commit**. Полная проверка на финальном SHA ещё НЕ ПОДТВЕРЖДЕНА.
- **Статус:** SOURCE IMPLEMENTED; CI FINAL SHA PENDING; live Registrar не подключён к coordinator, полная автоматическая маршрутизация НЕ PASS. **Влияние на работающие службы: NONE**.

## 2026-11-24 — единый атомарный writer полного поколения (SOURCE ONLY)

- В рамках утверждённого M6 добавлена `xui.ApplyCompletePolicy` в существующий `acl_activation.go`: тот же `atomicACLConfigWrite`, полные `routes + authorized`, проверка точного digest, сохранение остальных полей `server.json`, подтверждение через переданный callback, восстановление предыдущего файла и повторный ACK при неудаче.
- Добавлен `TestApplyCompletePolicyAtomicACKAndRollback` в `route_sync_test.go`: успешный ACK, сохранение других полей, отказ нового ACK и восстановление, отказ при неправильном digest.
- **Коммиты:** `77d2be15c1037e6cf7b09e226a97cb141a392e18` (код), `824c74c5835a3624ea039e35db8bc8a1229c0a48` (тест).
- **Граница:** новый writer — библиотечная функция в существующем процессе, не новый сервис. Он ещё не подключён к `m6-auto-registrar`; live Xray evidence также не подключён. **CI на commit теста ожидается; E2E и release не PASS.** Работающие службы: NONE.

## 2026-11-24 — исправление устаревшего списка сервисов при enrollment

- `RuntimePolicyCoordinator.Activate` теперь под общей блокировкой читает актуальные `routes` из текущего `server.json`, валидирует имена и непустые backends, сортирует и передаёт список существующему `HotACLActivator`. Стартовый `Services` больше не используется **при вызове через координатор**.
- Тест `TestRuntimePolicyCoordinatorEnrollmentUsesCurrentRoutes` проверяет смену маршрута `30000` → `55535` между стартом и регистрацией и точный digest ACK.
- Коммиты `67fda4bd47c65a0cb66bc0c2f134228706cfe0f1`, `a4f665c186bc1613c1218264e27b79e223678c0b`.
- **Ограничение:** сам Registrar пока вызывает `HotACLActivator` напрямую, не координатор. Полная интеграция не PASS; текущие CI ещё выполняются. Runtime и службы не затронуты.

## 2026-11-24 — Registrar подключён к единому владельцу enrollment

- **Изменено:** `integrations/android-ingress/xui/cmd/m6-auto-registrar/main.go`. Существующий HTTPS enrollment теперь вызывает `RuntimePolicyCoordinator.Activate(ctx,fingerprint)` вместо прямого `HotACLActivator.Activate`. Используются прежние CA, проверка Docker image/label, TLS endpoint, ACK и путь конфигурации. Новый процесс или writer не добавлен.
- **Причина:** прямой вызов Activator оставлял статический список сервисов, считанный при старте; coordinator перечитывает актуальные маршруты под общей блокировкой перед регистрацией.
- **SOURCE commit:** `30b8bd9d7ed9b7b99203987af2eaaf452a0f8f6c`. Поведенческий тест координатора `TestRuntimePolicyCoordinatorEnrollmentUsesCurrentRoutes` — commit `a4f665c186bc1613c1218264e27b79e223678c0b`, CI на текущий Registrar commit ещё не подтверждён.
- **Ограничения:** запуск Registrar по-прежнему проверяет существующий `server.json` и требует ненулевых маршрутов; автоматический `Sync()` не включён до подтверждённых live Xray/runtime bindings. Это **не** PASS динамической маршрутизации. Рабочие службы не затронуты.

## 2026-11-24 — удаление startup snapshot сервисов из Registrar

- В `cmd/m6-auto-registrar/main.go` удалены вызов `configuredServiceNames(root)` при запуске и передача статического `Services` в `HotACLActivator`. `RuntimePolicyCoordinator.Activate` теперь единственный источник перечня сервисов для enrollment, перечитывая текущую политику под mutex.
- Не изменены Docker identity, CA, TLS endpoint, слушающий адрес, API HTTP enrollment и работающие службы. Существующий helper `configuredServiceNames` и его тесты пока сохранены; не удалять без ревизии.
- SOURCE commit `a64e19fd33e20972a3bf6187e9e68bb22b84d897`. CI этого commit ещё не подтверждён; live 3x-ui/Xray Sync по-прежнему не подключён. M6 IN_PROGRESS.

## 2026-11-24 — защита Discovery от устаревшего digest после enrollment

- `RuntimePolicyCoordinator.Activate` после успешного изменения ACL инвалидирует `ConfirmedRouteStore` (если он подключён). При ошибке активации также fail-closed: результат применения поколения может быть неопределённым. Это не влияет на уже открытые TCP-потоки.
- Тест `TestCoordinatorEnrollmentInvalidatesPreviousDiscoveryGeneration`: подтверждённый snapshot → enrollment второго fingerprint → snapshot больше не ready; новый ACL сохранён.
- Коммиты `bf77b1dfcce2d916f1445c16e7a0de72f5e9efe4` и `2c9a9310522773749e7760cc7b980cbf9efc59cd`.
- **Важная граница:** текущий Registrar пока не подключает Store и Sync. Это исправление не означает, что после enrollment автоматически публикуется новое поколение. Полная интеграция и E2E не PASS; CI текущего кода ожидается. Работающие службы: NONE.

## 2026-11-24 — Registrar привязан к Store и полному policy writer (без включения Sync)

- `cmd/m6-auto-registrar/main.go`: в существующем `RuntimePolicyCoordinator` добавлены `ConfirmedRouteStore` и `RoutePolicyApply`, использующий `ApplyCompletePolicy` с прежним `DockerPolicyConfirmer` из `HotACLActivator`. Это одна точка записи и тот же точный ACK; дополнительных процессов/слушателей нет.
- Коммит `18967ae9619916b983c6ee366bc70ccbbbbda015`.
- **НЕ ПОДКЛЮЧЕНО:** `Panel`, `Applied` и запуск `Sync`. Не использовать configured-only Xray parser или диагностический скрипт с фиксированным PID как доказательство live runtime. Поэтому автоматическая синхронизация всё ещё НЕ PASS. CI нового commit ожидается. Работающие службы: NONE.

## 2026-11-24 — комплексный source integration test для coordinator→writer→ACK

- Добавлен `TestCoordinatorSyncAppliesFullGenerationWithRealAtomicWriter` в `xui/route_sync_test.go`: три TCP inbound (VLESS TCP 30000, VLESS XHTTP 55535, Trojan TCP 55636), `RuntimePolicyCoordinator.Sync`, `ApplyCompletePolicy`, атомарный файл, exact digest ACK, подтверждённый `ConfirmedRouteStore` и обновлённый ACL существующего fingerprint. Используется тестовый callback ACK, **не** live Docker.
- Source commit `74a22def3eadf222223558670792c83fe5048b80`; CI ожидается.
- Ограничение: источник live Xray и 3x-ui URL на Homeserver не подтверждены, поэтому production Sync пока не включён. Это не E2E PASS. Рабочие службы: NONE.

## 2026-11-24 — Live Xray и достижимость backend XHTTP из Decapsulator

- Ветка `work/m6-android-ingress`, HEAD GitHub перед записью `ca55b1e6a80d8166167d77b854a267bb22d9e4f3`. Проверка оператором на Homeserver из checkout `7a48737bda553b3f93e819ecea0da1c1daf78a32`; blob диагностического скрипта `66d9879338091b6669a6dd8ecdd2e436f3956de1` совпадает на обоих коммитах.
- **LIVE GRPC PROJECTION PASS.** Xray однозначно найден среди прямых потомков активного `x-ui.service` по PPID и имени исполняемого файла. Один read-only `xray api lsi --isOnlyTags=false` к loopback API, с проверкой идентичности процесса до и после. 5 handlers, один служебный пропущен, 4 совпадения со снимком конфигурации: VLESS TCP 30000, VLESS XHTTP (splithttp) 55535, Trojan TCP 55636, SOCKS TCP 10811. Неподдерживаемых и несовпавших нет. Перезапуск Xray и выбор через API панели в этом прогоне не проверялись.
- **DOCKER_BACKEND_CONNECT_55535=PASS.** Текущий Decapsulator определён через `docker inspect`; прочитан его `/etc/hosts`, затем `nsenter -t <init-pid> -n` выполнил одно TCP-подключение к адресу `host.docker.internal` на порту 55535. Идентичность контейнера повторно сверена после подключения. Предыдущие PASS для 30000 и 55636 сохранены как отдельные результаты, эти порты повторно не проверялись.
- Проверка 55535 подтверждает достижимость TCP. Публикация маршрута, передача XHTTP через Bridge, автоматический `Sync` и live ACK поколения не проверялись. Изменений на Homeserver не вносилось, M6 остаётся IN_PROGRESS.

## 2026-11-24 — Привязка AppliedXrayEvidence к Registrar

- Исходный HEAD `e8fd734f096908f9eb90bd4341f152389d32fc3c`, ветка `work/m6-android-ingress`. Код, тесты и эта запись входят в один commit.
- Существующий Registrar устанавливает `RuntimePolicyCoordinator.Applied` при заданном `--xray-probe`. Путь к доверенному установленному `live_xray_evidence.py` абсолютный. Для него рядом нужны два существующих модуля диагностики: `check_panel_xray_bridge_routes.py` и `check_xray_grpc_runtime.py`. Параметры установки `--xray-owner-unit`, `--xray-executable` и `--xray-backend-alias` задают родительский systemd unit, basename Xray и alias из hosts действующего Decapsulator.
- Probe переиспользует существующий read-only gRPC-опрос и проекцию. Определяет PID Xray по PPID, проверяет владельца loopback API и TCP listeners, проверяет backend TCP из network namespace подтверждённого Decapsulator и повторно сверяет идентичность процессов. Go проверяет проекцию, использует проверенный IP и десятичный порт как имя сервиса. При неоднозначности, смене процесса или недоступности backend источник возвращает ошибку.
- Проверки локально на Linux: 17 Python unit tests PASS; Go 1.23.12 `go test -race -count=1 ./...` и `go vet ./...` PASS. Проверен код привязки, новый probe на Homeserver ещё не запускался.
- Автоматический вызов `Sync`, live ACK и E2E остаются открытыми. В этом шаге выполнена привязка источника, без установки и изменения работающих служб.

## 2026-11-24 — Homeserver: live источник Xray прошёл проверку

- Оператор выполнил read-only прогон `live_xray_evidence.py` из точного GitHub commit `d6a0ba7906021ab1600afaea02d4398aee1f333d` с действующим Decapsulator. Результат `LIVE_XRAY_EVIDENCE=PASS`: VLESS TCP 30000, VLESS XHTTP 55535, Trojan TCP 55636. Прогон проверял live gRPC, принадлежность слушателей и TCP-доступность из сетевого пространства Decapsulator.
- Три порта заданы только как ожидаемые результаты операторской проверки на этом сервере. Источник получает текущие порты из ответа Xray, Registrar получает выбранные inbound из API панели, а координатор сопоставляет их по точному порту, протоколу и транспорту. Конкретные порты не являются настройками Registrar или правилами публикации.
- Проверка относится к live-источнику. Автоматический `Sync`, добавление и удаление inbound с применением новой политики, точный ACK действующего Decapsulator и E2E остаются открытыми. Службы не менялись. M6 IN_PROGRESS.

## 2026-11-24 — Автоматическая синхронизация маршрутов и пустое поколение

- Ветка `work/m6-android-ingress`, исходный HEAD `5c2358543f2f786ebb463a21c5d9ab4dc70d8c70`. Registrar запускает существующий `RuntimePolicyCoordinator.Sync` сразу при старте и далее с интервалом `--route-sync-interval`, заданным при установке. Ноль оставляет прежний режим. Второй процесс или writer не добавляется.
- В динамическом режиме Discovery выдаёт доступность только после ACK поколения с непустыми маршрутами. Тот же подтверждённый digest повторно не записывается. При временной ошибке чтения сохраняется последнее подтверждённое поколение.
- Успешный ответ панели без включённых TCP inbound применяется как пустой набор маршрутов и пустые списки разрешений существующих отпечатков. Обновлённый RuntimePolicy overlay принимает и подтверждает его. Восстановление inbound возвращает маршрут и разрешения. TCP-сессия, открытая до переключения, сохраняет свой сокет.
- Изменены серверный RuntimePolicy и его overlay/test, Registrar, координатор, Discovery, обработка полного digest и тесты маршрутизации. Проверено на Linux amd64 с Go 1.23.12. Registrar: `go test -race -count=1 ./...` и `go vet ./...` PASS. Изолированный Decapsulator: patch-функции overlay на точной базе `96c136702d10b6fb9a16aa2893f6b978e8b14e86`; при первом race-прогоне сбой `TestFramedCancelRequestBodyReaderReleasesServerWorker` (closed pipe), повтор этого теста и полный race-прогон PASS. `go vet ./...` и сборка с `-buildvcs=false` для checkout из архива PASS.
- Это проверка исходников и изолированной сборки. Переход на Homeserver и live ACK не выполнялись. Для перехода требуется кандидат Decapsulator с новым overlay и проверенная привязка к Registrar, затем проверка изменения inbound и открытой сессии. Релиз не публиковался. M6 IN_PROGRESS.

## 2026-11-24 — HTTPS loopback к API 3x-ui на Homeserver

- База `a99b783a1ea1de03b7ab8980df76f905f42ad5c6`, ветка `work/m6-android-ingress`. При подготовке перехода на Homeserver оператор подтвердил: прямой HTTPS-запрос истёк по тайм-ауту (`curl RC=28`), loopback с проверкой TLS вернул три inbound. Оба бинарных кандидата на базе исходного HEAD собраны, действующие службы не переключались.
- В защищённый JSON для 3x-ui добавлено необязательное поле `"connect_loopback": true`. Оно направляет TCP на `127.0.0.1` с портом из HTTPS base URL. Проверка сертификата, TLS-имя, HTTP Host, базовый путь и запрет перенаправлений сохранены. Без поля используется прежний HTTPS-транспорт.
- Файлы: `xui/panel_config.go`, `xui/panel_loopback_test.go`. Go 1.23.12: `go test -race -count=1 ./...` и `go vet ./...` PASS. Тест проверяет правильный Host, запрос inbound через TLS и отклонение сертификата при несовпадении имени.
- Следующий шаг перехода: пересобрать только Registrar с этой привязкой, проверить его реальный запрос на Homeserver, сохранить настройки панели и провести согласованное переключение двух компонентов. Рабочие службы и политика пока не менялись.

## 2026-11-24 — Homeserver: переключение на маршруты из API 3x-ui

- Оператор подтвердил `M6_CUTOVER_PREFLIGHT=PASS`, `M6_POLICY_ACK=PASS` и `M6_CUTOVER=PASS`. Registrar из `1f30cfa8` использует защищённые параметры 3x-ui с HTTPS loopback и интервал синхронизации 30 секунд. Decapsulator из `a99b783a` запущен с read-only mount проверенного бинарника; Registrar перепривязан к новому ID контейнера. Для Compose задействован профиль `android-ingress-manual`.
- Проверка после запуска сопоставила точный набор включённых TCP inbound панели с маршрутизацией и разрешениями каждого зарегистрированного отпечатка. Вычисленный digest применённого `server.json` совпал с записью Decapsulator `M6_POLICY_RELOAD=PASS digest=11253e662170a6833e1aa54e0e80cff41ace3522492dd546b908600e30b9fba7`.
- Статус: переход и первый live ACK подтверждены. Изменение inbound после запуска, сохранение уже открытого потока при смене поколения и VPN E2E на Homeserver ещё не подтверждены. M6 IN_PROGRESS.

## 2026-11-24 — Homeserver: три протокола работают через Bridge

- После переключения на маршруты из API 3x-ui оператор подтвердил работу через Bridge всех трёх профилей: VLESS TCP (30000), VLESS XHTTP (55535) и Trojan TCP (55636). Основание: отчёт оператора после `M6_CUTOVER=PASS` и точного ACK, зафиксированных в `38f8380713686a1268f59dd2e40fe43bc3c850e2`. Номера портов относятся к проверенным профилям Homeserver.
- Изменение состава inbound на работающем сервере и непрерывность открытого TCP-потока при смене поколения остаются отдельными проверками M6.
- Дополнительное наблюдение оператора после перехода: при переключении VPN-профилей Bridge оставался включённым, заметного перерыва доступа в интернет не было. Непрерывность отдельной TCP-сессии при изменении серверной политики этим наблюдением не измерялась.

## 2026-11-24 — Homeserver: новый inbound 8891

- Оператор подтвердил `PORT_8891=FOUND_IN_POLICY` и `PORT_8891_ACK=PASS`. Проверка сопоставила маршруты и ACL с API 3x-ui и нашла ACK точного digest текущей политики в журнале действующего Decapsulator. Новый маршрут подтверждён. Передача VPN-трафика через профиль 8891 в этом прогоне не проверялась.
- Последующая проверка оператором: VPN-трафик через Bridge на новом inbound `8891` работает. Вместе с предыдущим точным ACK это подтверждает сквозную работу динамически добавленного маршрута на Homeserver. Отключение inbound и непрерывность ранее открытого потока при смене поколения остаются отдельными проверками.

## 2026-11-24 — Состояние Decapsulator для мониторинга Android

- Согласован первый этап: краткий статус сервера через существующий mTLS-вход, проверка клиентского сертификата по текущему ACL, отображение готовности, версии, времени работы и последнего успешного ответа в Android. Счётчики трафика устройства отложены.
- Homeserver: локальный HTTP-экспорт `/metrics/v1` выключен. Для изолированного кандидата обработчик `GET /v2/status` прочитал состояние из памяти, сохранив штатную точку входа TCP и локальный интерфейс метрик. Исходная ветка Bridge `ce2011c275047dd5950ef8544ea0b9bd4cc48065`, база ядра `96c136702d10b6fb9a16aa2893f6b978e8b14e86`, RuntimePolicy overlay применён до локальных правок.
- Оператор: `go test -race -count=1 ./...`, `go vet ./...` и сборка кандидата PASS. Unit-проверка обработчика покрывает разрешённый сертификат, сертификат вне ACL, отсутствие проверенной цепочки и запрещённый HTTP-метод. SHA256 кандидата `60e098ca66153a3c142956c49398ad0d5f5116de0c4674ad9ec384c77a9818c9`.
- Код проверен в отдельной рабочей копии Homeserver и ещё не перенесён в GitHub overlay. Работающие службы не менялись. Реальный mTLS-запрос к кандидату и интеграция Android не проверялись.

## 2026-11-24 — Homeserver: переключение Decapsulator с `/v2/status`

- Оператор подтвердил `STATUS_PREFLIGHT=PASS`, `STATUS_REGISTRAR_BINDING=PASS`, `STATUS_CUTOVER=PASS`. Проверен SHA256 работающего кандидата `60e098ca66153a3c142956c49398ad0d5f5116de0c4674ad9ec384c77a9818c9`. Binder Registrar соответствует новому контейнеру. После переключения получен точный `STATUS_POLICY_ACK=PASS`, digest `a83d857487f244c86e500634d2aca5de0e3fb96063978f38412de934d9bb8e25`.
- Перед успешным запуском проверены права выполнения кандидата от UID 65532 и добавлено ожидание фактического запуска Registrar после `systemctl start` в операторской процедуре.
- Статус: установка серверного кандидата подтверждена. Ответ `/v2/status` с Android и отображение карточки на устройстве ожидают подтверждения оператора. Проверенный серверный код пока собран из локальных правок изолированного дерева Homeserver, в overlay GitHub записана только документация выполнения.
- Скриншот Android после переключения подтверждает успешный запрос `/v2/status` с телефона: «Сервер отвечает», версия `1.0.2`, время работы `0 мин 28 с`. В поле последнего успешного ответа показано буквальное Dart-выражение вместо даты. Реальный ответ сервера подтверждён, исправление отображения даты и сценарии недоступности/восстановления ещё не проверены.

## 2026-11-24 — Счётчики TLS 1.3 / HTTP/2 / mTLS на сервере и устройстве

- В существующий RuntimePolicy overlay перенесён обработчик `GET /v2/status` из исходников проверенного кандидата Homeserver. В ответ добавлен суммарный счётчик принятых потоков Decapsulator `tls13_h2_mtls_requests`. Чтение статуса не меняет счётчики или окно вычисления скорости. Доступ по текущему ACL и проверенной mTLS-цепочке сохранён.
- Клиентское ядро увеличивает свой счётчик для принятого HTTP/2-потока через TLS 1.3 после проверки HTTP 200, совместимости формата и проверенной TLS-цепочки. Запросы статуса идут отдельно. Серверные и локальные значения различаются по области и времени запуска.
- На точной базе `96c136702d10b6fb9a16aa2893f6b978e8b14e86` применены преобразования overlay. `go test -race -count=1 ./...` и `go vet ./...` PASS. Проверки покрывают выдачу серверного счётчика и подсчёт локального потока в тесте реального HTTP/2+mTLS.
- Интеграция Android UI, пересборка диагностического ARM64 ядра и проверка новых счётчиков на работающих устройствах ещё не выполнены. Рабочий Decapsulator не переключался.
- Homeserver: после сборки из `b835f41ba37717e550c1787b5effc63492a3d7fa` серверный кандидат прошёл race-тесты и `go vet`, SHA256 `15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6`. Android-кандидат с сохранением имеющейся диагностики собран на ARM64 после race-тестов и `go vet`, SHA256 `82e0d7d294e0fb120e75c3bc1dc4664c2057613a1ce6c35ae7210474f1f8b54b`. Правка счётчиков Android UI применена по операторскому выводу. Оба бинарных кандидата подготовлены, установка и рост счётчиков на устройстве ещё не проверялись.
- Скриншоты Android: локальное ядро `1.0.2` показывает 4 потока TLS 1.3 / HTTP/2 / mTLS, 3 TCP-сессии активны; серверный статус отвечает и время последнего ответа отображается в формате даты и времени. Серверная строка счётчика показывает `—`; установка серверного кандидата со счётчиком и рост его значения ещё не подтверждены.
- Homeserver: `TLS_COUNTER_PREFLIGHT=PASS`, `TLS_COUNTER_CUTOVER=PASS`. Запущен Decapsulator с серверным счётчиком (SHA256 `15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6`), Registrar перепривязан, получен `TLS_COUNTER_POLICY_ACK=PASS` для digest `a83d857487f244c86e500634d2aca5de0e3fb96063978f38412de934d9bb8e25`. Отображение и рост серверного показателя в Android после переключения ожидают проверки.
- Оператор подтвердил, что после переключения серверный показатель TLS 1.3 / HTTP/2 / mTLS отображается в Android. Два снимка для количественной проверки роста не предоставлены.

## 2026-11-24 — Подготовка публичной установки Linux v1.0.2

- Обновлено описание в README и добавлена отдельная инструкция по установке Linux x86-64, происхождению актуального бинарника и назначению Registrar. Подтверждённая сборка: `b835f41b`, SHA256 `15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6`.
- Публикация готовится в существующем релизе под отдельным именем `tervon-bridge-linux-amd64-v1.0.2-m6`, чтобы сохранить прежние ссылки и их контрольные суммы. На момент записи загруженный asset ещё не подтверждён. Действующие службы не менялись.
- ПУБЛИКАЦИЯ ПОДТВЕРЖДЕНА. В GitHub Release `v1.0.2` добавлены `tervon-bridge-linux-amd64-v1.0.2-m6` и `SHA256SUMS-m6`, обновлено описание релиза. Оператор: `RELEASE_ASSET_SHA256=PASS`, `RELEASE_UPLOAD=PASS`. Публичный GitHub API подтвердил SHA256 бинарника `15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6` и наличие обновлённого описания. Прежний asset сохранён. Службы не менялись.

## 2026-11-25 — Remnawave в источнике Registrar

- Ветка `work/m6-android-ingress`, база `f410e4f469e52789ab5d5e68a93b07bb0573deab`. Источник выбирает API 3x-ui или Remnawave по защищённой конфигурации. Для Remnawave используется явно заданный UUID ноды. Сверка с live Xray и подтверждение поколения остаются в существующем координаторе.
- Linux, Go 1.23.12, модуль `integrations/android-ingress/xui`. `go test -race -count=1 ./...` и `go vet ./...` прошли. Тест конфигурации отправил HTTPS GET в имитатор Remnawave и проверил выбор только назначенного ноде inbound. `go build -buildvcs=false ./cmd/m6-auto-registrar` прошёл в изолированной копии исходников.
- Живая панель Remnawave и Android end-to-end не проверялись. Работающие службы и релизные артефакты не изменялись.

## 2026-11-26 — Единый установщик и два режима Registrar

- База `dd5c13f4900044a2c8c60eed31f9157a48326fdb`, ветка `work/m6-android-ingress`. Добавлены `./install.sh`, интерактивная подготовка PKI, конфигурации, автозапуска и nginx Discovery. Нативный Registrar подтверждает политику по точному ACK из журнала проверенной службы. Docker использует прежнее подтверждение с возможностью разрешить имя в текущий ID при старте.
- Go 1.23.12: `go test -race -count=1 ./...`, `go vet ./...`, сборка Registrar — PASS. Python: генерация настроек обоих режимов и выбор nginx-блока на 9443 с имитированными системными командами — PASS. Bash syntax, генерация PKI через OpenSSL и проверка hostname в user namespace — PASS. В тесте PKI установка владельца подменена, поскольку namespace отображал только один UID.
- Запуск полного установщика на живом Linux-узле с nginx/Xray, Android end-to-end и тест перезагрузки — ожидают приёмки. Рабочие службы и сборки в релизе сохранены.

## 2026-11-26 — Linux ↔ Linux в едином установщике

- База `5809b69b6aeb61296bf1e1238abe6ba69c4ce6c3`. В меню добавлен межсерверный сценарий. Encapsulator создаёт локальный ключ и CSR. Decapsulator подписывает CSR, связывает его отпечаток с заданным сервисом и возвращает два публичных сертификата.
- Проверки: Bash syntax и Python compile прошли. Локальный TCP echo через два процесса Bridge с сгенерированной PKI — PASS. Команды systemd, загрузка бинарника и смена владельца в изолированном тесте подменялись. Реальный Docker и автозапуск после перезагрузки не проверялись.
