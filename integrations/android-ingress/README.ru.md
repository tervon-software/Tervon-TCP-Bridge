# M6 — Android-facing Tervon TCP Bridge: текущее состояние

**Актуально на 2026-11-22.** Этот README описывает только текущий серверный integration scope и **не является командой на развёртывание**. Единственный рабочий план: [docs/DEVELOPMENT_PLAN.md](../../docs/DEVELOPMENT_PLAN.md); конкретный аудит: [docs/M6_AUDIT_2026-11-22.md](../../docs/M6_AUDIT_2026-11-22.md).

## Назначение

Bridge прозрачно переносит двунаправленные TCP-байты между Android Xray и заранее выбранным серверным Xray backend через HTTP/2 + TLS 1.3 + mTLS. Внутренний VLESS/Trojan/XHTTP не анализируется и не меняется. Android должен только подготовить защищённое подключение, запустить ARM64 Go Encapsulator и направить выбранный TCP-поток в локальный listener. Серверный Decapsulator уже имеет собственный механизм routes/ACL; **новый серверный API для Android не создавать**.

## Действующие компоненты в GitHub

- `xui/client.go`, `xui/inbounds.go`: доверенный read-only адаптер 3x-ui API, включая slim inbound.
- `xui/route_reconcile.go`, `xui/route_sync.go`, `xui/confirmed_routes.go`: сверка panel ↔ текущий Xray, полная атомарная политика routes+ACL с digest ACK и подтверждённой таблицей для discovery. Код **есть**, но полный production wiring через текущий регистратор не доказан.
- `xui/enrollment_http.go`, `xui/certificate.go`, `xui/acl_activation.go`, `xui/cmd/m6-auto-registrar/main.go`: существующая регистрация Android CSR/mTLS, activation ACL, HTTPS discovery. Не объявлять «не реализовано»; на устройстве выполнена Android регистрация/сборка, но реальный TCP E2E FAIL.
- `runtime-acl/runtime_policy.go`: точный Go overlay атомарного runtime routes+ACL для выпущенного Linux ядра v1.0.2. Не использовать устаревший `prototype/bridge.go` M6 ветки в качестве релизного source.
- `compose.yaml`, `runtime-acl/compose.hot-acl.yaml`: настроенный отдельный Android-facing Decapsulator. Согласно операторским проверкам 22 ноября: image/cutover/Compose PASS, полноценный Android↔backend E2E **не подтверждён**.
- `stage4/`: многочисленные legacy/source-only версии Android интеграции, установщики и тесты. Они **не являются одним очищенным актуальным Android кодом**. Реальная установленная Android-программа находится в отдельном Homeserver проекте; согласована предстоящая инвентаризация перед удалением ошибочной обвязки.

## Источник серверных портов и сервисов

**Утверждённый контракт:** 3x-ui API → проверка реально применённого Xray → разрешённые маршруты Decapsulator → `tcp_routes` через HTTPS Discovery. `server.json` — runtime-конфигурация Bridge, но не справочник VPN протоколов/портов. В GitHub остаётся **legacy** функция `cmd/m6-auto-registrar/main.go:configuredTCPRoutes()`, которая считывает backend-порты из `server.json` при старте. Не трактовать её как итоговую API-first интеграцию. Нельзя ни скрывать этот конфликт, ни утверждать, что live registrar уже использует новую цепочку, без сверки бинарника/процесса на сервере.

## Последний достоверный статус

- Linux v1.0.2 GitHub Release опубликован; AMD64 asset SHA256 `aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371`.
- ARM64 диагностическое Go ядро v1.0.2: `go test`, `go test -race`, `go vet`, crosscompile PASS, SHA256 `a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd`. Kotlin Bridge compile PASS. Оператор собрал и установил APK на устройство.
- **Android с включённым Bridge: NO INTERNET**; XHTTP GET/POST `EOF`, локальный TCP `reset by peer`, диагностика `UNKNOWN`. Это подтверждённый **E2E FAIL**, а не ошибка `max.ru` или доказательство неработающего серверного ACL.
- Оператор потребовал ревизию и пересоздание старой **Android-обвязки**, с неизменным прозрачным TCP и без нового серверного API. Файлы приложения пока **НЕ удалялись/НЕ заменялись**.

Не выполнять развертывание, перезапуск, исправление backend портов, создание сертификатов или обновление APK только на основании этого документа. Исторические шаги и прежние советы из Git history не являются действующей инструкцией.
