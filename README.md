# Tervon TCP Bridge

**Статус на 16 ноября 2026:** работающий опытный TCP-мост; **не объявлен production-ready**. Реальная цепочка с TervonVPN и выходом через Германию проверена пользователем. Проверка не заменяет аудит безопасности и нагрузочные измерения на реальных маршрутах.

## Что это

Go-программа с двумя режимами: **Encapsulator** (`mode=client`) принимает локальные TCP-подключения и передаёт байты через HTTPS/HTTP/2 с взаимной TLS-аутентификацией; **Decapsulator** (`mode=server`) соединяется только с заранее разрешённым TCP backend. Внутренние VPN-протоколы Bridge не разбирает.

```text
TervonVPN
  → Homeserver: Xray VLESS XHTTP inbound :55636
  → Xray VLESS TCP TLS outbound (tervon-bridge-fi)
  → Encapsulator 127.0.0.1:55635
  → HTTPS/2 + mTLS
  → finserver Decapsulator
  → finserver Xray VLESS TCP TLS backend :55535
  → Hysteria2 Germany → Internet
```

В этой схеме Xray и Hysteria2 — **внешние компоненты**, а не протоколы, реализованные самим Bridge. Отдельный лабораторный маршрут SOCKS5 использует другой локальный вход и **не является обязательной частью основной схемы**.

## Подтверждённая версия

- Исходники: `prototype/` (Go module `tervon-tcp-bridge`, `go 1.23.0`).
- Исходный коммит рабочего бинарника: `83c98482048d130041e89f545db5885ade090f2b`.
- Homeserver: `/opt/tervon-tcp-bridge/bridge`, Go 1.25.5, `linux/amd64`, `CGO_ENABLED=0`, stripped, около 6.5 MB.
- SHA-256 **конкретного работающего файла**: `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`.
- `go version -m` показывает `vcs.modified=true`; побайтовая воспроизводимость из одного коммита **не подтверждена**.
- В репозитории есть CI, Go-тесты и изолированные сценарии `tests/homeserver/`; зафиксированы успешные loopback-прогоны 1000 сессий и 16 сессий по 120 секунд (см. исторические отчёты).

## Ограничения

Только TCP. Нет собственного UDP, WireGuard, Hysteria2, CONNECT-UDP, динамических произвольных адресов назначения или миграции действующего TCP-соединения при обрыве. Отсутствует гарантия незаметности для DPI. Совместимость Trojan, Shadowsocks и OpenVPN требует отдельных проверок. Программа требует корректных сертификатов и списка разрешённых сервисов.

## Документация

- [Установка и безопасное развёртывание](docs/INSTALL.ru.md)
- [Конфигурация, mTLS и Xray](docs/CONFIGURATION.ru.md)
- [Релиз, контроль целостности и откат](docs/RELEASE.ru.md)
- [Диагностика](docs/TROUBLESHOOTING.ru.md)
- [Проектный устав](docs/PROJECT_CHARTER.md) · [История испытаний](docs/HOMESERVER_INTEGRATION_2026-11-15.md)

## Разработка

```bash
cd prototype
go build ./...
go vet ./...
go test -count=1 -timeout=60s ./...
```

Не запускайте тяжёлые сценарии на работающих службах; испытания должны использовать отдельные адреса, сертификаты и процессы. Git remote — только SSH: `git@github.com:tervon-software/Tervon-TCP-Bridge.git`.

## Лицензия

Файл LICENSE пока не утверждён: публичный исходный код **не означает** автоматического разрешения на использование или распространение. До выбора лицензии не называйте проект лицензированным open source.
