# Установка Tervon TCP Bridge v1.0.2 на Linux x86-64

Tervon TCP Bridge передаёт двунаправленные TCP-потоки между двумя узлами через HTTP/2 поверх TLS 1.3 с взаимной проверкой сертификатов (mTLS). **Encapsulator** принимает локальное TCP-подключение. **Decapsulator** проверяет сертификат клиента и передаёт поток разрешённому TCP-сервису. VPN-протокол внутри потока обрабатывает Xray или другой назначенный сервис.

## Какой вариант нужен

| Сценарий | Что используется |
| --- | --- |
| Связать два Linux-узла | Один бинарник на обоих узлах. Маршруты и ACL задаются в JSON на сервере. |
| Подключить Android к 3x-ui с автоматической сменой inbound | Decapsulator и отдельный Registrar. Registrar читает API панели, сверяет inbound с live Xray и применяет маршруты и ACL одним поколением. На этом серверном бинарнике автоматическая интеграция подтверждена для 3x-ui. |

Инструкция ниже предназначена для **новой ручной установки Linux ↔ Linux**. Для схемы 3x-ui см. [контракт M6 и параметры Registrar](M6_3XUI_DYNAMIC_ROUTING.md). Бинарник Decapsulator сам по себе не настраивает 3x-ui, автоматическую выдачу сертификатов Android и синхронизацию inbound.

## 1. Скачать и проверить бинарник

На [странице v1.0.2](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/tag/v1.0.2) актуальная сборка публикуется отдельным файлом `tervon-bridge-linux-amd64-v1.0.2-m6`. Если его ещё нет в списке, публикация актуальной сборки не завершена. Для неё используется отдельная контрольная сумма:

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

Выпустите серверный сертификат с DNS SAN и назначением `serverAuth`, клиентский — с назначением `clientAuth`. Можно использовать один собственный CA или отдельные CA для двух направлений. Подробности есть в [инструкции по mTLS](INSTALL.ru.md).

| Узел | Файлы в `/etc/tervon-tcp-bridge/pki/` |
| --- | --- |
| Decapsulator | `server.crt`, `server.key`, `client-ca.crt` |
| Encapsulator | `client.crt`, `client.key`, `server-ca.crt` |

Установите владельца файлов `root:tervon-bridge` и права `0640`. Приватный ключ CA храните отдельно от этих узлов. Для ACL вычислите SHA256 клиентского сертификата в формате DER:

```bash
openssl x509 -in client.crt -outform DER | sha256sum
```

## 3. Настроить роли и запустить службы

Возьмите файлы из [проверенного коммита исходников](https://github.com/tervon-software/Tervon-TCP-Bridge/tree/6968a9805a9e774c28f2f6494dda0e7fe9323f74):

- На **Decapsulator** используйте [пример server.json](../../examples/server.json). Задайте адрес прослушивания, доступный Encapsulator, и реальный адрес TCP-сервиса в `routes`. Укажите SHA256 клиентского сертификата в `authorized` и разрешите нужный маршрут. Пример слушает `127.0.0.1`. Для прямого межсерверного подключения выберите адрес нужного интерфейса и разрешите вход на порт только от нужных источников.
- На **Encapsulator** используйте [пример client.json](../../examples/client.json). Укажите рабочий HTTPS URL с путём `/v2/stream`, DNS-имя из SAN сервера в `server_name` и разрешённое имя маршрута в `service`. По умолчанию локальный вход доступен на `127.0.0.1:19001`.
- Сохраните JSON на соответствующем узле в `/etc/tervon-tcp-bridge/server.json` или `client.json` с владельцем `root:tervon-bridge` и правами `0640`.

Установите соответствующий [пример systemd-юнита](../../systemd/) в `/etc/systemd/system/tervon-bridge-server.service` или `tervon-bridge-client.service`. Затем на каждом узле выполните команды для нужной роли, сначала на стороне сервера:

```bash
role=server   # На Encapsulator укажите client
sudo systemctl daemon-reload
sudo systemctl enable --now "tervon-bridge-$role.service"
sudo systemctl status "tervon-bridge-$role.service" --no-pager
```

Проверьте реальный двусторонний запрос к целевому сервису через локальный порт Encapsulator. Для HTTP-сервиса можно использовать `curl --fail http://127.0.0.1:19001/`. Для другого TCP-сервиса используйте его обычный клиент.

## 4. Эксплуатация

- Decapsulator применяет `routes` и `authorized` совместно по `SIGHUP`. После успешного применения в журнале появляется `M6_POLICY_RELOAD=PASS digest=…`. Установленные потоки продолжают работать с уже открытым backend-сокетом. Изменения TLS, слушателей и других статических параметров требуют отдельного перезапуска.
- `GET /v2/status` возвращает готовность, версию, время работы и суммарное число принятых потоков TLS 1.3 / HTTP/2 / mTLS. Запрос должен проходить через серверный TLS-вход и предъявлять сертификат из текущего ACL. Ответ не содержит данных отдельных клиентов.
- Локальный `/metrics/v1` можно включить через `metrics_listen` на loopback. Для `/v2/status` это не требуется.
- Для 3x-ui Registrar использует защищённые параметры API и проверяет актуальные порты через live Xray. Изменения списка inbound в настройках панели не требуют пересборки ядра. Настройка Registrar и перенос действующей установки выходят за рамки команды установки на этой странице.

## Происхождение и испытания

Исходная база — `96c136702d10b6fb9a16aa2893f6b978e8b14e86`. RuntimePolicy overlay для этого файла — `b835f41ba37717e550c1787b5effc63492a3d7fa`. Сборка Linux AMD64, Go race-тесты и `go vet` прошли. На действующем сервере подтверждены переключение, точный ACK политики и отображение серверного счётчика в Android. [Журнал разработки](M6_3XUI_ROUTING_DEVELOPMENT_LOG.md)
