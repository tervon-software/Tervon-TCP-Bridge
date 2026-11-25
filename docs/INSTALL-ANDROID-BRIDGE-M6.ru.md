# Установка серверной части Tervon TCP Bridge v1.0.2 M6 для Android

Эта инструкция описывает новую установку Decapsulator и Registrar рядом с действующим Xray под управлением 3x-ui. В примерах подписку обслуживает `sub.example.net`, а внешний транспорт Bridge — `bridge.example.net:18443`. Оба имени в примере указывают на сервер установки. Подставьте свои имена, порты и правила доступа.

## 1. Выбрать выпуск и проверить исходные условия

| Компонент | Что использовать |
| --- | --- |
| Android | TervonVPN DPI с интеграцией Bridge. См. [установку и актуальный APK](ANDROID-v1.0.2.ru.md). |
| Decapsulator Linux amd64 | [tervon-bridge-linux-amd64-v1.0.2-m6](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.2/tervon-bridge-linux-amd64-v1.0.2-m6) и [SHA256SUMS-m6](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.2/SHA256SUMS-m6). |
| Registrar, наблюдение Xray и подготовка PKI | [Исходники M6 на коммите f410e4f469e52789ab5d5e68a93b07bb0573deab](https://github.com/tervon-software/Tervon-TCP-Bridge/tree/f410e4f469e52789ab5d5e68a93b07bb0573deab/integrations/android-ingress). Бинарник Registrar собирается отдельно. |
| Сервер | Linux x86-64, Docker с Compose, Go 1.23 или новее, Python 3, OpenSSL, curl, nginx с рабочим HTTPS для подписки, 3x-ui и Xray с локальным gRPC API. Команда `nsenter` обычно поставляется в util-linux. |

В v1.0.1 ядро использует маршруты и ACL из стартовой конфигурации. Для M6 требуется сборка с атомарным обновлением **маршрутов и ACL** по одному поколению, точным ACK по digest и защищённым `/v2/status`. В релизе v1.0.2 лежат два Linux-бинарника. Для этой инструкции берите файл с суффиксом `-m6`. Переименование старого бинарника этих возможностей не добавляет.

```text
Android → HTTPS подписки → /.well-known/tervon-bridge → Registrar на localhost
Android → TLS 1.3 + HTTP/2 + mTLS → Decapsulator → выбранный TCP inbound Xray

Registrar → HTTPS API 3x-ui + live Xray gRPC + проверка backend из Docker
Registrar → атомарный server.json → SIGHUP Decapsulator → ACK того же digest
```

До установки проверьте, что обычный VPN-профиль работает, DNS подписки и Bridge указывает на нужный хост, внешний TCP-порт Bridge свободен и доступен клиентам, а HTTPS панели и подписки проходит проверку сертификатов. Внешний порт и внутренний порт могут отличаться. Для примера ниже освободите `18443/TCP`, а на loopback — `18763/TCP`. Панель и Xray gRPC оставьте на административном или локальном интерфейсе.

## 2. Скачать бинарник и получить инструменты M6

Команды сервера выполняются из административной оболочки. Пример использует новую директорию `/opt/tervon-bridge-m6`.

```bash
set -euo pipefail
test "$(uname -m)" = x86_64
test ! -e /opt/tervon-bridge-m6
install -d -m 0755 /opt/tervon-bridge-m6
cd /opt/tervon-bridge-m6

git clone git@github.com:tervon-software/Tervon-TCP-Bridge.git source
git -C source checkout --detach f410e4f469e52789ab5d5e68a93b07bb0573deab

base=https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.2
asset=tervon-bridge-linux-amd64-v1.0.2-m6
curl -fL "$base/$asset" -o "$asset"
curl -fL "$base/SHA256SUMS-m6" -o SHA256SUMS-m6
sha256sum -c SHA256SUMS-m6
printf '%s  %s\n' '15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6' "$asset" |
  sha256sum -c -
install -m 0755 "$asset" tervon-bridge
(
  cd source/integrations/android-ingress/xui
  go test ./...
  go build -trimpath -o /opt/tervon-bridge-m6/m6-auto-registrar ./cmd/m6-auto-registrar
)
```

## 3. Подготовить сертификаты и начальную конфигурацию

Для новой установки воспользуйтесь скриптом PKI из закреплённого коммита. Он ожидает новый каталог конфигурации и **однозначный VLESS TCP inbound в стартовом JSON Xray**. При нескольких кандидатах подготовьте отдельный bootstrap JSON только с выбранным действующим VLESS TCP inbound. Передайте путь через `M6_XRAY_CONFIG`. Этот файл используется для начальной политики, а рабочие порты после запуска синхронизируются по API и живому Xray.

```bash
cd /opt/tervon-bridge-m6
M6_XRAY_CONFIG=/usr/local/x-ui/bin/config.json \
  bash source/integrations/android-ingress/prepare-production-pki.sh \
  /etc/tervon/bridge-android bridge.example.net
```

| Файл | Назначение |
| --- | --- |
| `config/pki/server.crt` и `server.key` | Сертификат транспортного сервера и ключ. SAN — имя Bridge, которое получит Android. |
| `config/pki/client-ca.crt` | Публичный CA, которому доверяет Decapsulator при проверке клиентских сертификатов. |
| `private/client-ca.key` | Ключ выдачи, доступный Registrar на хосте. |
| `private/server-ca.crt` | Публичный CA сервера, возвращаемый Android через HTTPS Discovery. |
| `private/server-ca.key` | Закрытый ключ выпуска серверных сертификатов, нужен для обслуживания PKI. |

В контейнер монтируется весь каталог `config` как read-only. Это позволяет ядру увидеть атомарную замену `server.json`. Каталог `private` хранится на хосте с доступом владельца. Скопируйте PKI и конфигурацию в закрытую резервную копию. Срок действия leaf-сертификата можно проверить командой `openssl x509 -in /etc/tervon/bridge-android/config/pki/server.crt -noout -enddate`.

Пример сгенерированного `server.json` (порт `30000` здесь для примера):

```json
{
  "mode": "server",
  "listen": "0.0.0.0:8443",
  "cert_file": "/config/pki/server.crt",
  "key_file": "/config/pki/server.key",
  "ca_file": "/config/pki/client-ca.crt",
  "max_streams": 0,
  "routes": {
    "vpn": [{"Name": "existing-vless-tcp", "Address": "host.docker.internal:30000"}]
  },
  "authorized": {
    "0000000000000000000000000000000000000000000000000000000000000000": ["vpn"]
  },
  "padding_enabled": true,
  "padding_rate_enabled": true,
  "jitter_enabled": true
}
```

Нулевой отпечаток — начальная заглушка. После запуска синхронизации Registrar управляет `routes` и `authorized`. Заранее подготовленные маршруты по именам и список портов вручную для рабочего M6 не нужны. `max_streams: 0` означает отсутствие лимита сессий на уровне этой настройки. Пределы ресурсов следует выбирать по нагрузке.

## 4. Собрать контейнер из проверенного бинарника и запустить Decapsulator

Тег образа ниже принимается Registrar на указанном коммите. При существующей установке с тем же тегом используйте отдельное окно миграции. Для примера Compose публикует один IPv4 binding `0.0.0.0:18443 → 8443/tcp`, который Registrar использует для составления `remote_url`. Для внешнего доступа клиентов разрешите вход на опубликованный TCP-порт в сетевом периметре с учётом правил Docker.

```bash
cd /opt/tervon-bridge-m6
mkdir image
install -m 0755 tervon-bridge image/tervon-bridge
cat > image/Dockerfile <<'DOCKER'
FROM gcr.io/distroless/static-debian12:nonroot
COPY --chmod=0755 tervon-bridge /tervon-bridge
LABEL org.tervon.bridge.hot_acl="m6-v1"
USER 65532:65532
EXPOSE 8443/tcp
ENTRYPOINT ["/tervon-bridge","-config","/config/server.json"]
DOCKER
docker build --platform linux/amd64 \
  -t tervon-bridge-android-ingress:v1.0.2 image

cat > compose.m6.yaml <<'YAML'
services:
  android-bridge-decapsulator:
    image: tervon-bridge-android-ingress:v1.0.2
    pull_policy: never
    restart: unless-stopped
    user: "65532:65532"
    read_only: true
    cap_drop: [ALL]
    security_opt: [no-new-privileges:true]
    pids_limit: 256
    ports: ["0.0.0.0:18443:8443/tcp"]
    extra_hosts: ["host.docker.internal:host-gateway"]
    volumes:
      - /etc/tervon/bridge-android/config:/config:ro
YAML

docker compose -p tervon-android-m6 -f compose.m6.yaml config -q
docker compose -p tervon-android-m6 -f compose.m6.yaml up -d --no-build
cid=$(docker compose -p tervon-android-m6 -f compose.m6.yaml \
  ps -q android-bridge-decapsulator)
test -n "$cid"
docker port "$cid" 8443/tcp
docker inspect --format '{{.Config.Image}} {{.State.Running}} {{index .Config.Labels "org.tervon.bridge.hot_acl"}}' "$cid"
```

Ожидаются `0.0.0.0:18443` и `tervon-bridge-android-ingress:v1.0.2 true m6-v1`. VPN inbound должен быть доступен из сетевого пространства этого контейнера через `host.docker.internal`. Если Xray слушает только `127.0.0.1` хоста, такой backend для контейнера недоступен. Проверьте адрес прослушивания Xray и разрешите доступ только с нужной сетевой стороны. Порт gRPC Xray при этом оставьте локальным.

## 5. Настроить Registrar и API 3x-ui

В этом примере панель принимает HTTPS по `panel.example.net:5588/panel-prefix`. В `base_url` указывается её базовый префикс **до** `/panel/api`. Registrar добавит `/panel/api/inbounds/list/slim` сам. При `connect_loopback: true` TCP идёт на `127.0.0.1` и заданный порт, а HTTP Host и TLS-имя остаются из URL. Сертификат должен проверяться по системному хранилищу доверия.

```bash
(
  set -euo pipefail
  umask 077
  dir=/etc/tervon/bridge-android/private
  read -rsp 'API-токен 3x-ui: ' panel_token </dev/tty
  printf '\n'
  test -n "$panel_token"
  printf 'Authorization: Bearer %s\n' "$panel_token" > "$dir/3x-ui.header"
  unset panel_token
  cat > "$dir/m6-panel.json" <<'JSON'
{
  "type": "3x-ui",
  "base_url": "https://panel.example.net:5588/panel-prefix",
  "credential_file": "/etc/tervon/bridge-android/private/3x-ui.header",
  "connect_loopback": true
}
JSON
  chmod 0600 "$dir/3x-ui.header" "$dir/m6-panel.json"
  curl -fsS --max-time 10 \
    --resolve 'panel.example.net:5588:127.0.0.1' \
    --header "@$dir/3x-ui.header" \
    'https://panel.example.net:5588/panel-prefix/panel/api/inbounds/list/slim' |
    python3 -c 'import json,sys; d=json.load(sys.stdin); assert d.get("success") is True and isinstance(d.get("obj"),list); print("PANEL_API=PASS")'
)
```

Подставьте фактический systemd unit владельца Xray и имя исполняемого файла. Пример использует `x-ui.service` и `xray-linux-amd64`. Скрипт `live_xray_evidence.py` определяет действующий процесс, получает список inbound от Xray и проверяет доступность TCP backend из сетевого пространства контейнера. Для этой схемы Registrar запускается на хосте рядом с Xray и Docker. Пример серверной настройки предназначен для 3x-ui. Для Remnawave нужен Registrar из [ветки M6 с подключённым адаптером](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/a7734397f426266e4a1e95f048644c671f6a90c0/docs/Server/M6_3XUI_DYNAMIC_ROUTING.md#remnawave). В его конфигурации задаётся UUID обслуживаемой ноды. Синхронизация проверена с TLS-имитацией API. Полный прогон с реальной панелью и Android ещё открыт.

Для пробного запуска в отдельной консоли:

```bash
cd /opt/tervon-bridge-m6
cid=$(docker compose -p tervon-android-m6 -f compose.m6.yaml \
  ps -q android-bridge-decapsulator)
test -n "$cid"
./m6-auto-registrar \
  --root /etc/tervon/bridge-android \
  --listen 127.0.0.1:18763 \
  --container-id "$cid" \
  --expected-image tervon-bridge-android-ingress:v1.0.2 \
  --expected-label-key org.tervon.bridge.hot_acl \
  --expected-label-value m6-v1 \
  --panel-config /etc/tervon/bridge-android/private/m6-panel.json \
  --xray-probe /opt/tervon-bridge-m6/source/integrations/android-ingress/live_xray_evidence.py \
  --xray-owner-unit x-ui.service \
  --xray-executable xray-linux-amd64 \
  --xray-backend-alias host.docker.internal \
  --route-sync-interval 30s
```

В рабочей установке поместите эту же команду в systemd, подставив фактический ID. Для постоянной службы пример секций ниже. Перед запуском службы завершите пробный процесс через Ctrl+C. При пересоздании контейнера перед запуском Registrar обновите ID и повторите проверку ACK. Одновременно с каталогом политики должен работать один Registrar.

```ini
[Unit]
Description=Tervon Bridge M6 Registrar
After=docker.service network-online.target
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=/opt/tervon-bridge-m6
ExecStart=/opt/tervon-bridge-m6/m6-auto-registrar \
  --root /etc/tervon/bridge-android \
  --listen 127.0.0.1:18763 \
  --container-id REPLACE_WITH_CONTAINER_ID \
  --expected-image tervon-bridge-android-ingress:v1.0.2 \
  --expected-label-key org.tervon.bridge.hot_acl \
  --expected-label-value m6-v1 \
  --panel-config /etc/tervon/bridge-android/private/m6-panel.json \
  --xray-probe /opt/tervon-bridge-m6/source/integrations/android-ingress/live_xray_evidence.py \
  --xray-owner-unit x-ui.service \
  --xray-executable xray-linux-amd64 \
  --xray-backend-alias host.docker.internal \
  --route-sync-interval 30s
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Сохраните файл как `/etc/systemd/system/tervon-bridge-m6-registrar.service`. После подстановки ID на новой установке выполните `systemctl daemon-reload`, затем `systemctl enable --now tervon-bridge-m6-registrar.service`. Служба использует доступ к Docker, /proc, сетевому namespace контейнера и ключу Client CA. Прямой доступ к ней оставьте на loopback.

## 6. Подключить Discovery к HTTPS подписки

GET и POST используют один путь `/.well-known/tervon-bridge`. GET возвращает адрес, имя TLS и публичные CA. POST принимает CSR. Закрытый клиентский ключ остаётся на Android. После выдачи сертификата Registrar ждёт подтверждения ACL ядром.

**Как подключается Android.** Пользователь сначала импортирует HTTPS-подписку (3x-ui или Remnawave), запускает обычный VPN и проверяет интернет. При включении Bridge приложение обращается к Discovery по домену подписки и при необходимости запрашивает сертификат у Registrar. На Homeserver nginx проксирует оба запроса в Registrar на loopback. Registrar проверяет CSR и подтверждение ACL. Сертификат используется только для mTLS между Android и Decapsulator. Xray обслуживает VPN отдельно от выдачи сертификата и установления mTLS. Endpoint выдачи сертификата отдельно подписку не проверяет.

Вставьте в существующий HTTPS-блок `server` домена подписки. Это пример прямого прокси по штатному шаблону M6 с портом Registrar, подтверждённым на Homeserver:

```nginx
location = /.well-known/tervon-bridge {
    proxy_pass http://127.0.0.1:18763;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto https;
    proxy_read_timeout 30s;
    client_max_body_size 16k;
}
```

Если в этом виртуальном хосте есть редирект на уровне `server`, проверьте, что он не перехватывает Discovery. В штатном шаблоне редирект обычных веб-запросов помещается в `location /`.

Для новой настройки проверьте `nginx -t` и примените конфигурацию через `systemctl reload nginx`. Маршрут данных Android идёт напрямую на транспортный порт Bridge с mTLS. nginx на `443` здесь обслуживает подписку и Discovery.

## 7. Проверки по порядку

1. **Загрузка политики.** Проверьте журнал Registrar и журнал контейнера. После применения новой политики ожидайте `M6_POLICY_RELOAD=PASS digest=…`. Registrar сверяет точный digest и время после сигнала. Ответ Docker об отправке сигнала сам по себе проверку не закрывает.
2. **Discovery.** Проверьте локально, затем повторите с клиента через HTTPS домена подписки. После первого подтверждённого поколения с непустыми маршрутами ожидайте `available: true` и адрес `https://bridge.example.net:18443/v2/stream`.
3. **Android.** Импортируйте рабочую HTTPS-подписку, выберите поддерживаемый TCP-профиль, запустите обычный VPN, проверьте доступ в интернет и включите Bridge через раздел диагностики. Откройте страницу или загрузите файл и проверьте рост полученных байтов и счётчика принятых потоков в мониторинге. «Сервер отвечает» показывает доступность статусного запроса.
4. **Динамические маршруты.** На тестовой установке добавьте новый включённый TCP inbound в 3x-ui. Дождитесь синхронизации и нового ACK. Выберите профиль с новым портом и проведите трафик. При удалении проверьте отказ новых подключений к удалённому порту и сохранение уже установленных потоков на уровне Bridge. Перезапуск самого Xray панелью может оборвать backend-сессии независимо от Bridge.

```bash
journalctl -u tervon-bridge-m6-registrar.service -n 50 --no-pager
docker logs --since 10m "$cid" 2>&1 | grep 'M6_POLICY_RELOAD='
curl -fsS http://127.0.0.1:18763/.well-known/tervon-bridge |
  python3 -c 'import json,sys; d=json.load(sys.stdin); print({k:d.get(k) for k in ("available","remote_url","server_name")})'
```

| Симптом | Что проверить |
| --- | --- |
| Discovery возвращает `available: false` | Наличие включённых совместимых inbound, API 3x-ui, gRPC Xray, backend из Docker и точный ACK. |
| Discovery по HTTPS даёт 403 | Правила доступа виртуального хоста или прокси перед ним. В штатном блоке Discovery ограничений по IP нет. |
| Панель даёт 401 или 404 | Токен, версию API, адрес и префикс панели до `/panel/api`. |
| Registrar отвергает контейнер | Полный актуальный ID, образ, метку и read-only монтирование каталога конфигурации. |
| Android не проходит TLS | Доступность внешнего порта, SAN транспортного сертификата, срок действия, цепочку CA и соответствие имени из Discovery. |
| Android получил сертификат, поток не идёт | Порт в активном профиле и его наличие в политиках, доступность живого backend и счётчик принятых потоков. |
| В белом списке мобильной сети вход не открывается | Проверьте доступность адреса и порта из этой сети. Если IP исключён из списка разрешённых адресов, прямое подключение недоступно. |

Проверенные в проекте варианты — VLESS TCP, VLESS XHTTP поверх TCP и Trojan TCP, включая добавленный после запуска inbound. Значения доменов и портов в этой инструкции — примеры. Проверки 1–4 предназначены для подтверждения именно вашей установки.

## 8. Обновление действующего v1.0.1

Сохраните бинарник, образ, compose, unit, конфигурацию и PKI. Сначала испытайте M6 на отдельном порту и отдельном каталоге конфигурации с отдельным Registrar. После выбора окна переключения сверьте опубликованный порт, адрес Discovery и новый ID контейнера, затем повторите проверки из раздела 7. Смена поколения политики в одном процессе сохраняет открытые потоки. Замена контейнера или процесса может их завершить.

При откате остановите новый Registrar, верните сохранённые бинарник, конфигурацию и связанный с ними способ запуска. Убедитесь, что фактически запущена исходная версия и её обычный маршрут работает. Порядок обновления зависит от способа установки. Команды выше предназначены для нового экземпляра.

## Исходники и дальнейшее чтение

- [Контракт динамической маршрутизации M6](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/f410e4f469e52789ab5d5e68a93b07bb0573deab/docs/Server/M6_3XUI_DYNAMIC_ROUTING.md)
- [Код Registrar на закреплённом коммите](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/f410e4f469e52789ab5d5e68a93b07bb0573deab/integrations/android-ingress/xui/cmd/m6-auto-registrar/main.go)
- [Обычная установка Linux ↔ Linux](INSTALL-v1.0.2.ru.md)
- [Технологии Bridge и границы применимости](TECHNOLOGIES-v1.0.2.ru.md)
