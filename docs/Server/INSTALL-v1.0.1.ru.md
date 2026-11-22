# Установка Tervon TCP Bridge v1.0.1 скриптом (Linux x86-64)

**Для чистой первоначальной установки** на двух Linux-серверах. Готовые файлы и проверка SHA256 находятся в [GitHub Release v1.0.1](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/tag/v1.0.1). **Не запускайте установщик поверх работающего Bridge.**

## Что устанавливается

Программа работает в двух режимах:

- **Encapsulator / client** на первом сервере: принимает локальный TCP, устанавливает исходящий TLS 1.3 / HTTP/2 / mTLS канал.
- **Decapsulator / server** на втором сервере: принимает mTLS и перенаправляет только к заранее указанному разрешённому TCP backend.

Режимы используют один и тот же Linux amd64 бинарник. Установщик создаёт системную учётную запись `tervon-bridge`, бинарник `/opt/tervon-tcp-bridge/bridge`, каталог `/etc/tervon-tcp-bridge/pki`, файл `client.json.example` или `server.json.example` и соответствующий `tervon-bridge-*.service`. **Не создаёт сертификаты, не изменяет firewall, не настраивает Xray и не включает службу автоматически.**

Проверенный бинарник Linux v1.0.1 имеет SHA256:

```text
4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b
```

Этот SHA жёстко зафиксирован в установщике; файл с любыми другими байтами будет отвергнут. Публикуется **бинарник без фиксированной квоты TCP-сессий**, проверенный на двух серверах.

## 1. Предварительные условия

Два **новых** Linux x86-64 сервера с systemd, Bash, Python 3, OpenSSL 3, curl, доступом root и достаточными правами для установки. Изоляция от действующей инфраструктуры обязательна. Подготовьте собственный CA (его приватный ключ **не отправлять** на сервера), отдельные клиентский и серверный сертификаты с подходящим EKU и SAN. На Finserver-подобной роли нужен доступный TCP backend, например `127.0.0.1:19000`. В серверном firewall разрешайте только заранее утверждённые источники. Порт `18443` в шаблоне серверного режима намеренно слушает **127.0.0.1**; при прямом внешнем доступе измените его осознанно и настройте firewall, а не отключайте защиту.

**Внимание:** установщик откажется работать, если уже присутствуют Bridge binary, unit, реальные JSON или непустые каталоги установки. Он **не выполняет обновление** и не предназначен для Homeserver/Finserver, где v1.0.1 уже работает.

## 2. Скачать и проверить сам скрипт

Для проверки исходного кода установщика рекомендуется клонировать официальный тег **через SSH**:

```bash
git clone git@github.com:tervon-software/Tervon-TCP-Bridge.git
cd Tervon-TCP-Bridge
git fetch --tags origin
git checkout --detach v1.0.1
bash -n install/install-bridge.sh
less install/install-bridge.sh
```

Не запускайте удалённые скрипты напрямую через `curl | bash`. Git tag указывает на исходники релиза, а исходный commit проверенного бинарника отдельно указан в [заметках к выпуску](RELEASE_NOTES_v1.0.1.ru.md). Не следует считать тег криптографически подписанным без проверки подписи.

## 3. Установить файлы (без запуска службы)

На первом, локальном сервере:

```bash
sudo bash install/install-bridge.sh install client
```

На втором, удалённом сервере:

```bash
sudo bash install/install-bridge.sh install server
```

Скрипт загружает **только** официальный asset `tervon-bridge-linux-amd64` и `SHA256SUMS` из GitHub Release v1.0.1. Контрольный SHA проверяется дважды: против встроенного проверенного SHA и файла Release. Ожидаемый результат:

```text
INSTALL=PASS_FILES_ONLY
VERSION=v1.0.1
BINARY_SHA256=4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b
SERVICE_NOT_STARTED=YES
```

Для установки **без доступа к GitHub**, после ручной передачи проверенных release assets:

```bash
sudo bash install/install-bridge.sh install client \
  --asset ./tervon-bridge-linux-amd64 --checksums ./SHA256SUMS
```

Для роли `server` заменить `client` на `server`.

## 4. Настроить сертификаты

В документации [INSTALL.ru.md](INSTALL.ru.md) описан выпуск отдельных сертификатов для серверного и клиентского назначения. Используйте CA, который контролируете сами. Не копируйте `ca.key` на сервер.

На **client** должны быть доступны:

```text
/etc/tervon-tcp-bridge/pki/client.crt
/etc/tervon-tcp-bridge/pki/client.key
/etc/tervon-tcp-bridge/pki/server-ca.crt
```

На **server**:

```text
/etc/tervon-tcp-bridge/pki/server.crt
/etc/tervon-tcp-bridge/pki/server.key
/etc/tervon-tcp-bridge/pki/client-ca.crt
```

Для ключей и сертификатов назначьте владельца `root`, группу `tervon-bridge`, права на приватные ключи не шире `0640`; каталоги `0750`. Убедитесь, что сертификат сервера содержит DNS SAN с именем, указанным в клиентском `server_name`.

## 5. Создать рабочие конфигурации из шаблонов

На сервере **client**:

```bash
sudo cp /etc/tervon-tcp-bridge/client.json.example /etc/tervon-tcp-bridge/client.json
sudoedit /etc/tervon-tcp-bridge/client.json
```

Замените `bridge.example.org` в `remote_url` и `server_name` на реальное имя сертификата сервера. Локальный TCP вход в шаблоне: `127.0.0.1:19001`. Для v1.0.1 используется `/v2/stream`, `padding_enabled`, `padding_rate_enabled`, `jitter_enabled`: `true` на обеих сторонах.

На сервере **server**:

```bash
sudo cp /etc/tervon-tcp-bridge/server.json.example /etc/tervon-tcp-bridge/server.json
sudoedit /etc/tervon-tcp-bridge/server.json
```

Измените `routes.vpn[].Address` на реально доступный локальный TCP backend. В `authorized` замените `REPLACE_WITH_CLIENT_CERT_SHA256_HEX` на точный SHA256 DER-кодировки **клиентского** leaf-сертификата:

```bash
openssl x509 -in client.crt -outform DER | sha256sum
```

Проверяйте совпадение названия `vpn` на обеих сторонах. После редактирования:

```bash
sudo chown root:tervon-bridge /etc/tervon-tcp-bridge/*.json
sudo chmod 0640 /etc/tervon-tcp-bridge/*.json
python3 -m json.tool /etc/tervon-tcp-bridge/server.json >/dev/null  # только server
```

На client вместо server.json проверьте client.json. Не публикуйте конфигурации с реальными endpoint, секретами или закрытыми маршрутами.

## 6. Запустить только выбранный Bridge — отдельное действие

**Сначала** подготовить/запустить разрешённый backend на стороне server, затем убедиться, что сертификаты и сеть корректны. Запустить **только на новой машине**, после настройки:

На сервере **server**:

```bash
sudo bash install/install-bridge.sh activate server
systemctl is-active tervon-bridge-server.service
```

На сервере **client**:

```bash
sudo bash install/install-bridge.sh activate client
systemctl is-active tervon-bridge-client.service
```

`activate` повторно проверяет бинарный SHA, выбранный JSON, отсутствие шаблонных значений, наличие и доступность сертификатов для служебного пользователя. Запуск с уже активной службой отклоняется; перезапуска чужих служб нет. **Успешный systemd Active сам по себе не доказывает VPN-трафик.**

## 7. Проверить передачу данных

```bash
systemctl status tervon-bridge-client.service --no-pager
systemctl status tervon-bridge-server.service --no-pager
journalctl -u tervon-bridge-client.service -n 30 --no-pager
journalctl -u tervon-bridge-server.service -n 30 --no-pager
```

Каждая команда запускается на **соответствующей** машине. Проверьте, что ваш локальный TCP клиент действительно подключается к `127.0.0.1:19001` и backend на сервере принимает его данные. Проверьте свой целевой протокол и сохранность TCP-потока. Если используется отдельная метрика, её порт должен быть привязан только к loopback и не выставлен в Интернет.

## 8. Откат и ограничения

- `install-bridge.sh` намеренно **не обновляет** работающие экземпляры и не перезаписывает конфигурации. Для upgrade/rollback нужен отдельный согласованный plan с резервными копиями и проверкой SHA.
- Установщик не настраивает маршруты Xray, DNS, TLS reverse proxy, брандмауэр, сертификаты и пользовательскую подписку — они зависят от вашей схемы.
- Прежде чем открывать серверный TCP порт вовне, проверьте firewall и доступность только авторизованных клиентов.
- Проверка исходников/CI и успешная установка не являются доказательством гарантированной доступности в любой сети.

## Подтверждение бинарника для официального выпуска

На 2026-11-19 оператор подтвердил одновременный live cutover PASS обоих серверов на SHA `4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b`, настройка `max_sessions=0` на обоих подтверждена независимой телеметрией. В приведённых установщиком примерах `max_local_connections=0` и `max_streams=0`, то есть программного ограничения числа сессий нет. Сохраняются конечные ресурсы ОС, а также mTLS и авторизация. Существующие рабочие серверы этим скриптом не переустанавливайте. Для новых серверов проверяйте, что SHA256 опубликованного asset совпадает с указанным выше.
