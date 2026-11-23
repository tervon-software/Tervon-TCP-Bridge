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
