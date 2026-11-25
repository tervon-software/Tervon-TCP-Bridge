# Установка Tervon TCP Bridge v1.0.2 на Linux x86-64

Tervon TCP Bridge передаёт двунаправленные TCP-потоки между двумя узлами через HTTP/2 поверх TLS 1.3 с взаимной проверкой сертификатов (mTLS). **Encapsulator** принимает локальное TCP-подключение. **Decapsulator** проверяет сертификат клиента и передаёт поток разрешённому TCP-сервису. VPN-протокол внутри потока обрабатывает Xray или другой назначенный сервис.

Для подключения TervonVPN DPI с Android используйте [отдельную инструкцию установки сервера M6](INSTALL-ANDROID-BRIDGE-M6.ru.md).

## Какой вариант нужен

В обоих вариантах клиенту нужен доступ к адресу и порту Decapsulator. Если сеть разрешает соединения только к определённым IP и исключает IP входного сервера, Bridge по этому маршруту недоступен. См. [границы применимости](TECHNOLOGIES-v1.0.2.ru.md#границы-применимости).

| Сценарий | Что используется |
| --- | --- |
| Связать два Linux-узла | Один бинарник на обоих узлах. Маршруты и ACL задаются в JSON на сервере. |
| Подключить Android к 3x-ui с автоматической сменой inbound | Decapsulator и отдельный Registrar. Registrar читает API панели, сверяет inbound с live Xray и применяет маршруты и ACL одним поколением. На этом серверном бинарнике автоматическая интеграция подтверждена для 3x-ui. |

Инструкция ниже предназначена для **новой ручной установки Linux ↔ Linux**. Для схемы 3x-ui см. [контракт M6 и параметры Registrar](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/Server/M6_3XUI_DYNAMIC_ROUTING.md). Бинарник Decapsulator сам по себе не настраивает 3x-ui, автоматическую выдачу сертификатов Android и синхронизацию inbound.

## 1. Скачать и проверить бинарник

Скачайте [актуальный бинарник Linux x86-64](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.2/tervon-bridge-linux-amd64-v1.0.2-m6) и [SHA256SUMS-m6](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.2/SHA256SUMS-m6) из релиза v1.0.2. Контрольная сумма бинарника:

```text
15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6  tervon-bridge-linux-amd64-v1.0.2-m6
```

На каждом узле, в каталоге загрузки:

```bash
printf '%s  %s\n' \
  '15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6' \
  'tervon-bridge-linux-amd64-v1.0.2-m6' | sha256sum -c -
file 'tervon-bridge-linux-amd64-v1.0.2-m6'
```

Используйте Linux x86-64. Для управления службой ниже нужен systemd. Команды предполагают чистую установку, в которой ещё нет Bridge, его конфигурации и systemd-юнитов.

## 2. Установить файлы и сертификаты

На каждом узле:

```bash
test ! -e /opt/tervon-tcp-bridge/bridge &&
test ! -e /etc/tervon-tcp-bridge &&
sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin tervon-bridge &&
sudo install -d -m 0755 /opt/tervon-tcp-bridge &&
sudo install -d -m 0750 -o root -g tervon-bridge \
  /etc/tervon-tcp-bridge /etc/tervon-tcp-bridge/pki &&
sudo install -m 0755 'tervon-bridge-linux-amd64-v1.0.2-m6' /opt/tervon-tcp-bridge/bridge
```

### Выпустить сертификаты для собственной установки

Если у вас уже есть собственный CA и сертификаты с нужным назначением, используйте их и перейдите к копированию файлов. Для новой установки шаги ниже предполагают OpenSSL 3 и один частный CA для обеих ролей. Замените `bridge.example.org` на DNS-имя Decapsulator, доступное с Encapsulator.

На машине, где будет храниться ключ CA:

```bash
umask 077
openssl req -x509 -newkey rsa:3072 -noenc -sha256 -days 365 \
  -subj '/CN=Tervon Bridge private CA' \
  -addext 'basicConstraints=critical,CA:TRUE' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign' \
  -keyout ca.key -out ca.crt
```

На **Decapsulator** создайте ключ и запрос сертификата:

```bash
umask 077
openssl req -new -newkey rsa:2048 -noenc \
  -subj '/CN=bridge.example.org' -keyout server.key -out server.csr
```

На **Encapsulator** создайте отдельный ключ и запрос:

```bash
umask 077
openssl req -new -newkey rsa:2048 -noenc \
  -subj '/CN=bridge-client' -keyout client.key -out client.csr
```

Передайте на машину CA только `server.csr` и `client.csr`. Создайте там файлы расширений и подпишите оба запроса:

```bash
cat > server.ext <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:bridge.example.org
EXT

cat > client.ext <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=clientAuth
EXT

openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -days 90 -sha256 -extfile server.ext -out server.crt
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
  -CAserial ca.srl -days 90 -sha256 -extfile client.ext -out client.crt

openssl verify -CAfile ca.crt -purpose sslserver \
  -verify_hostname bridge.example.org server.crt
openssl verify -CAfile ca.crt -purpose sslclient client.crt
```

Скопируйте `server.crt` на Decapsulator, `client.crt` на Encapsulator и публичный `ca.crt` на оба узла. Ключ каждой роли остаётся на узле, который его создал. `ca.key` остаётся на машине CA.

На **Decapsulator**:

```bash
sudo install -o root -g tervon-bridge -m 0640 server.crt server.key \
  /etc/tervon-tcp-bridge/pki/
sudo install -o root -g tervon-bridge -m 0640 ca.crt \
  /etc/tervon-tcp-bridge/pki/client-ca.crt
```

На **Encapsulator**:

```bash
sudo install -o root -g tervon-bridge -m 0640 client.crt client.key \
  /etc/tervon-tcp-bridge/pki/
sudo install -o root -g tervon-bridge -m 0640 ca.crt \
  /etc/tervon-tcp-bridge/pki/server-ca.crt
```

Для ACL вычислите SHA256 клиентского сертификата на Encapsulator или машине CA:

```bash
openssl x509 -in client.crt -outform DER | sha256sum
```

Передайте на Decapsulator первые 64 символа вывода и вставьте их вместо заполнителя в `authorized`. У каждого клиента собственный отпечаток.

## 3. Настроить роли и запустить службы

Возьмите файлы из [проверенного коммита исходников](https://github.com/tervon-software/Tervon-TCP-Bridge/tree/6968a9805a9e774c28f2f6494dda0e7fe9323f74):

- На **Decapsulator** используйте [пример server.json](../examples/server.json). Задайте адрес прослушивания, доступный Encapsulator, и реальный адрес TCP-сервиса в `routes`. Укажите SHA256 клиентского сертификата в `authorized` и разрешите нужный маршрут. Пример слушает `127.0.0.1`. Для прямого межсерверного подключения выберите адрес нужного интерфейса и разрешите вход на порт только от нужных источников.
- На **Encapsulator** используйте [пример client.json](../examples/client.json). Укажите рабочий HTTPS URL с путём `/v2/stream`, DNS-имя из SAN сервера в `server_name` и разрешённое имя маршрута в `service`. По умолчанию локальный вход доступен на `127.0.0.1:19001`.
- Сохраните JSON на соответствующем узле в `/etc/tervon-tcp-bridge/server.json` или `client.json` с владельцем `root:tervon-bridge` и правами `0640`.

Из [каталога исходников](https://github.com/tervon-software/Tervon-TCP-Bridge/tree/7c573e52694b67b8318a3f9cdb172c1dabc5e6c9) скачайте `examples/server.json`, `examples/client.json` и нужный шаблон из `systemd/` с сохранением этих путей. В этом каталоге выполните на каждом узле, подставив его роль:

```bash
role=server   # На Encapsulator укажите client
test ! -e "/etc/tervon-tcp-bridge/$role.json" &&
test ! -e "/etc/systemd/system/tervon-bridge-$role.service" &&
sudo install -o root -g tervon-bridge -m 0640 "examples/$role.json" \
  "/etc/tervon-tcp-bridge/$role.json" &&
sudo install -o root -g root -m 0644 "systemd/tervon-bridge-$role.service.example" \
  "/etc/systemd/system/tervon-bridge-$role.service"
```

Отредактируйте JSON перед запуском:

```bash
sudoedit "/etc/tervon-tcp-bridge/$role.json"
sudo python3 -m json.tool "/etc/tervon-tcp-bridge/$role.json" >/dev/null
```

Запустите службы, сначала на Decapsulator:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now "tervon-bridge-$role.service"
sudo systemctl status "tervon-bridge-$role.service" --no-pager
```

Проверьте реальный двусторонний запрос к целевому сервису через локальный порт Encapsulator. Для HTTP-сервиса можно использовать `curl --fail http://127.0.0.1:19001/`. Для другого TCP-сервиса используйте его обычный клиент.

Для отдельной проверки доступности Decapsulator выполните на Encapsulator запрос с действующим клиентским сертификатом. Укажите тот же DNS и порт, что в `remote_url`.

```bash
pki=/etc/tervon-tcp-bridge/pki
sudo curl --fail --show-error --silent --http2 --tlsv1.3 \
  --cert "$pki/client.crt" --key "$pki/client.key" --cacert "$pki/server-ca.crt" \
  https://bridge.example.org:18443/v2/status
```

Ответ со статусом подтверждает доступность HTTPS/mTLS-входа. Для проверки передачи TCP-трафика также нужен запрос через локальный порт Encapsulator.

Если служба не стартует, посмотрите журнал нужной роли:

```bash
sudo journalctl -u "tervon-bridge-$role.service" -n 60 --no-pager
```

Если mTLS работает, а поток не проходит, сверяйте имя `service`, отпечаток клиентского сертификата в `authorized` и доступность адреса из `routes` с Decapsulator.

## 4. Эксплуатация

- Decapsulator применяет `routes` и `authorized` совместно по `SIGHUP`. После успешного применения в журнале появляется `M6_POLICY_RELOAD=PASS digest=…`. Установленные потоки продолжают работать с уже открытым backend-сокетом. Изменения TLS, слушателей и других статических параметров требуют отдельного перезапуска.
- `GET /v2/status` возвращает готовность, версию, время работы и суммарное число принятых потоков TLS 1.3 / HTTP/2 / mTLS. Запрос должен проходить через серверный TLS-вход и предъявлять сертификат из текущего ACL. Ответ не содержит данных отдельных клиентов.
- Локальный `/metrics/v1` можно включить через `metrics_listen` на loopback. Для `/v2/status` это не требуется.
- Для 3x-ui Registrar использует защищённые параметры API и проверяет актуальные порты через live Xray. Изменения списка inbound в настройках панели не требуют пересборки ядра. Настройка Registrar и перенос действующей установки выходят за рамки команды установки на этой странице.

## Происхождение и испытания

Исходная база — `96c136702d10b6fb9a16aa2893f6b978e8b14e86`. RuntimePolicy overlay для этого файла — `b835f41ba37717e550c1787b5effc63492a3d7fa`. Сборка Linux AMD64, Go race-тесты и `go vet` прошли. На действующем сервере подтверждены переключение, точный ACK политики и отображение серверного счётчика в Android. [Журнал разработки](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/Server/M6_3XUI_ROUTING_DEVELOPMENT_LOG.md)
