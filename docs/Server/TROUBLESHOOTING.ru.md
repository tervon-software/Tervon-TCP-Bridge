# Диагностика Tervon TCP Bridge v1.0.1

Проверяйте маршрут по слоям: локальный TCP-сервис → Encapsulator → HTTPS/2 + TLS 1.3 + mTLS → Decapsulator → разрешённый TCP backend. Не смешивайте сетевые ошибки Bridge с проблемами Xray или других прикладных служб.

| Симптом | Что проверить |
|---|---|
| Локальный TCP-клиент не подключается | `listen` на стороне client, активность `tervon-bridge-client.service`, firewall и доступность локального TCP |
| TLS handshake failure | CA, SAN/`server_name`, клиентский сертификат, сроки действия, TLS 1.3, ALPN `h2` |
| HTTP 403 | Отпечаток SHA-256 DER клиентского leaf-сертификата и разрешённый `service` |
| HTTP 502 | Разрешённый `routes` backend недоступен с server |
| HTTP 503 | Проверьте `max_streams`, если администратор задал положительную квоту; в штатном шаблоне v1.0.1 `max_streams=0` |
| HTTP 404 | Путь должен быть `/v2/stream`, проверьте метод и целевой HTTPS endpoint |
| TCP-соединение оборвалось | Сообщения обеих служб, backend и сеть; текущая TCP-сессия автоматически не переносится |
| Bridge подключён, но нет доступа к приложению | Конфигурация TCP backend, маршрутизация и DNS прикладного сервиса |

На **соответствующей** машине используйте команды только для чтения:

```bash
systemctl is-active tervon-bridge-client.service  # на стороне client
systemctl is-active tervon-bridge-server.service  # на стороне server
journalctl -u tervon-bridge-client.service -n 50 --no-pager
journalctl -u tervon-bridge-server.service -n 50 --no-pager
```

Не запускайте второй экземпляр Bridge на работающем порту и не публикуйте приватные JSON, ключи, сертификаты или внутренние адреса. Для полного первоначального развёртывания смотрите [инструкцию v1.0.1](INSTALL-v1.0.1.ru.md).
