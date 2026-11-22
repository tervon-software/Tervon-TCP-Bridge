# M6 — вариант 2: обновление ACL без перезапуска Android Bridge

> **АРХИВНЫЙ СНИМОК ЭТАПА, НЕ ДЕЙСТВУЮЩАЯ ИНСТРУКЦИЯ (ревизия 2026-11-22).** Датированные статусы, старые примеры адресов/портов, команды cutover/patch и признаки NOT_IMPLEMENTED/PENDING ниже не отражают текущее состояние M6. Не выполнять команды из этого документа без нового утверждённого плана и сверки фактического кода/сервисов. Актуальные факты: [`DEVELOPMENT_PLAN.md`](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/DEVELOPMENT_PLAN.md), [`M6_AUDIT_2026-11-22.md`](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/M6_AUDIT_2026-11-22.md). Android Bridge E2E = FAIL; исторические PASS относятся только к явно названному слою.

**Решение пользователя:** вариант 2, 21.11.2026. Продолжаем тот же M6 — автоматическая регистрация mTLS после включения «Обход DPI». Существующие P-256 CSR, client CA, TLS 1.3, HTTP/2, Padding/Jitter, серверная TLS-идентичность `vp.xrayvless.online` **не меняются**. Не проверяем действительность VLESS-подписки при выпуске Bridge-сертификата; проверка доступа к VPN остаётся за Xray.

## Что меняется

**Только Android-facing кандидат ядра**, полученный из проверенного исходного commit релиза v1.0.1 `96c136702d10b6fb9a16aa2893f6b978e8b14e86`. Этот исходник **не тот** старый `prototype/`, который сейчас лежит в M6-ветке; никаких прямых правок Linux RU↔FI работающих экземпляров.

- `integrations/android-ingress/runtime-acl/acl_snapshot.go`: неизменяемая копия таблицы разрешённых mTLS-отпечатков, опубликованная `atomic.Pointer`. Каждый новый HTTP/2 stream получает актуальный список разрешений, но начатые TCP-потоки не меняются и не закрываются.
- `apply-to-release-source.py`: применяет ограниченную правку только на **чистой копии точного исходного commit v1.0.1**. Добавляет обработчик `SIGHUP`, который повторно читает `server.json`, проверяет существующие поля, требует неизменности **всего кроме `authorized`**, затем атомарно меняет только ACL. Логирует `M6_ACL_RELOAD=PASS digest=...` после публикации снимка; при ошибке прежний список остаётся активным. Не меняет рабочие сервисы.
- `integrations/android-ingress/xui/acl_activation.go`: сериализует регистрацию, обновляет существующий `server.json` с сохранением прав и владельца и требует подтверждения **точного SHA-256** JSON таблицы ACL от работающего процесса; при ошибке инициирует откат. SHA-256 здесь только идентификатор версии таблицы, **не новая криптосистема**.
- `integrations/android-ingress/xui/acl_docker_confirm.go`: отправляет `SIGHUP` **только** запущенному Android-facing контейнеру с правильными образом, меткой поддержки `m6-v1` и read-only mount `/config`. Никакого `docker restart` при выдаче сертификатов. Если до однократного обновления ядра работает прежний образ `:local`, сигнала **не будет** — это обязательная защита от остановки старого Bridge.
- `integrations/android-ingress/runtime-acl/Dockerfile`: собирает **отдельный** образ `tervon-bridge-android-ingress:m6-runtime-acl` с меткой `org.tervon.bridge.hot_acl=m6-v1`, не перезаписывая проверенный `:local`.
- `integrations/android-ingress/xui/cmd/m6-auto-registrar/main.go`: обрабатывает серверные `GET/POST /.well-known/tervon-bridge` только на loopback, через уже существующий CA и callback ACL. Должен быть подключён к Nginx HTTPS 443 (а не к 3x-ui подписке на порту 2096). Серверные адреса являются параметрами запуска, не hardcode Android.

## Проверки до внедрения (НЕ изменяют действующие службы)

Работать в текущем M6 checkout на Homeserver с веткой `work/m6-android-ingress` после `git pull --ff-only` через SSH. Использовать чистую отдельную рабочую копию релиза в `/opt/tervon/Developed`, предварительно проверив отсутствие каталога. Сборка из HEAD M6 `prototype/` **запрещена**, поскольку это не исходник выпущенного v1.0.1.

```bash
set -euo pipefail
M6=/opt/tervon-tcp-bridge-m6-android-ingress
RELEASE=/opt/tervon/Developed/m6-acl-validation-release
test "$(git -C "$M6" branch --show-current)" = "work/m6-android-ingress"
test ! -e "$RELEASE"
git -C "$M6" cat-file -e 96c136702d10b6fb9a16aa2893f6b978e8b14e86^{commit}
git -C "$M6" worktree add --detach "$RELEASE" 96c136702d10b6fb9a16aa2893f6b978e8b14e86
python3 "$M6/integrations/android-ingress/runtime-acl/apply-to-release-source.py" \
  --release-root "$RELEASE" --check
python3 "$M6/integrations/android-ingress/runtime-acl/apply-to-release-source.py" \
  --release-root "$RELEASE" --apply
(
  cd "$RELEASE/prototype"
  gofmt -w bridge.go cmd/bridge/main.go acl_snapshot.go acl_snapshot_test.go
  go test -count=1 -timeout=120s ./...
  go test -race -count=1 -timeout=180s ./...
  go vet ./...
  go build -o /opt/tervon/Developed/m6-acl-validation-bridge ./cmd/bridge
)
(
  cd "$M6/integrations/android-ingress/xui"
  go test -count=1 -timeout=120s ./...
  go test -race -count=1 -timeout=180s ./...
  go vet ./...
  go build -o /opt/tervon/Developed/m6-auto-registrar-validation ./cmd/m6-auto-registrar
)
```

Не вызывать `docker build`, `docker compose`, `docker kill`, `docker restart` и не изменять конфигурацию живого Nginx, пока **отдельные** проверки исходников не прошли и не подтверждён план однократного переключения только Android-facing контейнера.

## Особенность первоначального переключения

Вариант 2 означает **без перезапуска при каждой последующей регистрации**. Но чтобы впервые установить **новый** код, понадобится **одна согласованная замена/перезапуск только Android-facing контейнера** с поддержкой `SIGHUP`. Старый проверенный образ остаётся на месте для отката; Linux Bridge Россия ↔ Финляндия и работающие службы не менять. После обновления регистратор проверяет capability label, применяет ACL только через `SIGHUP` и ждёт подтверждения поколения ACL. При неподтверждённом обновлении Android получает ошибку вместо ложного успеха.

## Текущий статус

**УТВЕРЖДЕНО:** Вариант 2, автоматическое получение mTLS без проверки подписки.

**РЕАЛИЗОВАНО в GitHub:** отдельный исходный пакет горячего ACL, релизный overlay, сервис регистратора, точное подтверждение ACL и тесты.

**НЕ ПОДТВЕРЖДЕНО:** полный Go build/race PASS текущего HEAD (GitHub Actions workflow создан, результат не получен); фактический Nginx 443, работающий регистратор, живая выдача Android CSR.

**НЕ ВЫПОЛНЕНО:** установка нового образа, однократное переключение Android ingress, изменение действующего ACL, выпуск APK, VPN E2E. Серверный TLS SAN `vp.xrayvless.online` ранее проверен на реальном Homeserver — PASS.
