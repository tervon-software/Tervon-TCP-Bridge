# M6 — смена TLS-имени Android Bridge: bridge.xrayvless.online → vp.xrayvless.online

> **АРХИВНЫЙ СНИМОК ЭТАПА, НЕ ДЕЙСТВУЮЩАЯ ИНСТРУКЦИЯ (ревизия 2026-11-22).** Датированные статусы, старые примеры адресов/портов, команды cutover/patch и признаки NOT_IMPLEMENTED/PENDING ниже не отражают текущее состояние M6. Не выполнять команды из этого документа без нового утверждённого плана и сверки фактического кода/сервисов. Актуальные факты: [`DEVELOPMENT_PLAN.md`](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/DEVELOPMENT_PLAN.md), [`M6_AUDIT_2026-11-22.md`](https://github.com/tervon-software/Tervon-TCP-Bridge/blob/work/m6-android-ingress/docs/M6_AUDIT_2026-11-22.md). Android Bridge E2E = FAIL; исторические PASS относятся только к явно названному слою.

**Решение оператора, 21.11.2026.** Изменяется только имя **серверного Android-facing Bridge**. Приложение Android не содержит фиксированных адресов, использует домен текущей подписки. **Не менять** VLESS TCP :30000, 3x-ui-подписку, её порт 2096, FIN-каскад, существующий Linux Bridge Россия → Финляндия, клиентский CA, серверный CA, UUID или Xray.

## Достоверные исходные данные

- Из документации M6 и результатов предыдущего осмотра Homeserver: отдельный Android ingress Docker использует bind-mount `/etc/tervon/bridge-android/config:/config:ro`, серверный сертификат `/etc/tervon/bridge-android/config/pki/server.crt` (SAN `bridge.xrayvless.online`), ключ `config/pki/server.key`, CA `private/server-ca.crt` / `private/server-ca.key`; listener хоста был `127.0.0.1:18443`. Это снимок, а не подтверждение текущего состояния.
- Отдельная 3x-ui-подписка использует HTTPS `vp.xrayvless.online:2096`; Nginx обслуживает обычный HTTPS **443** с сертификатом `/root/cert/vp.xrayvless.online/{fullchain.pem,privkey.pem}`, ранее применял 302-redirect. Нельзя отправлять discovery-запросы на **2096**, менять подписку или подменять Nginx-сертификат внутренним Bridge CA.
- Проверка DNS 2026-11-21: `vp.xrayvless.online` и `bridge.xrayvless.online` разрешаются в `87.103.198.219` (A). Изменение DNS для этого cutover не требуется; фактическая доступность HTTPS и портов отдельно не проверена.
- М6 discovery-клиент исправлен в `fec81f2eb5b750f89facf90af208dc8e841eee08`: из URL подписки берётся **только домен**; запросы `GET/POST https://<домен>/.well-known/tervon-bridge` идут на HTTPS **443**. Секретный путь и `:2096` в запрос не переносятся. Универсальность без hardcode FQDN сохраняется.

## Подготовленный пакет (GitHub, без автоматического применения)

- `integrations/android-ingress/stage4/stage-server-name-change.py`, коммит `8c9abd4d53c6a9b8e9b79d80e5e156c4173c617b`. `--check` читает текущий server CA/ключ/сертификат, проверяет подпись и совпадение публичных ключей, **ничего не изменяет**. `--stage` сохраняет отдельный *только публичный* кандидат `private/m6-server-name-*/server.crt`, подписанный тем же CA и использующий существующий серверный ключ, SAN нового FQDN. Не перезаписывает текущий `server.crt`, не трогает ключи или Docker.
- `integrations/android-ingress/stage4/activate-server-name-change.py`, коммит `91c546db19ce6fa89e8cd216668b11aaa26c9c71`. `--check` проверяет кандидат и находит ровно один запущенный контейнер **с образом** `tervon-bridge-android-ingress:local` и **read-only bind** указанного каталога в `/config`. `--apply` атомарно заменяет только текущий `server.crt` с сохранением владельца/прав, перезапускает **только этот** контейнер, проверяет его `Running`; если операция завершается ошибкой, возвращает прежний сертификат и пытается вернуть контейнер. Скрипт не меняет Nginx, 3x-ui, Xray, внешние порты, CA или ключи. **Наличие нового SAN в реально поданном TLS-сеансе проверять отдельно.**
- На изолированной P-256/openssl и **имитированной Docker** среде: `M6_SERVER_NAME_STAGING_ISOLATED=PASS`; `M6_CUTOVER_ISOLATED=PASS`; `M6_CUTOVER_ROLLBACK=PASS`. Никакой проверки на реальном Homeserver или его Docker не было.

## Порядок выполнения на Homeserver (после SSH-загрузки утверждённой ветки)

Все GitHub git-операции через SSH `git@github.com:tervon-software/Tervon-TCP-Bridge.git`. Не менять рабочую ветку чужого checkout; использовать уже установленный M6 checkout, сначала подтвердить его `git branch --show-current` и `git rev-parse HEAD`, затем `git pull --ff-only git@github.com:tervon-software/Tervon-TCP-Bridge.git work/m6-android-ingress`, если ветка совпадает.

```bash
cd /opt/tervon-tcp-bridge-m6-android-ingress || exit 1
git branch --show-current
git rev-parse HEAD
python3 integrations/android-ingress/stage4/stage-server-name-change.py \
  --domain vp.xrayvless.online --check
```

При подтверждённой конфигурации:

```bash
sudo python3 integrations/android-ingress/stage4/stage-server-name-change.py \
  --domain vp.xrayvless.online --stage
# Подставить точный путь из строки M6_STAGED_PEM:
CANDIDATE=/etc/tervon/bridge-android/private/m6-server-name-XXXXX/server.crt
sudo python3 integrations/android-ingress/stage4/activate-server-name-change.py \
  --candidate "$CANDIDATE" --domain vp.xrayvless.online --check
# Следующая команда затрагивает ТОЛЬКО Android ingress, требует PASS предыдущей:
sudo python3 integrations/android-ingress/stage4/activate-server-name-change.py \
  --candidate "$CANDIDATE" --domain vp.xrayvless.online --apply
```

После применения отдельно проверить сертификат, предоставляемый Android ingress на фактическом опубликованном локальном порту (по исторической схеме `127.0.0.1:18443`), и SAN через `openssl s_client` / `openssl x509`. **Это не установка HTTP регистратора**: Nginx 443 должен маршрутизировать `/.well-known/tervon-bridge` к обработчику, а обработчик должен активировать fingerprint в ACL, прежде чем ответить успехом. Текущий Go HTTP handler — исходный код, к действующему Nginx/сервису выдачи ещё не привязан.

## Фактический статус

**DOMAIN_DECISION=APPROVED; DNS_MATCH=CONFIRMED; STAGING_AND_CUTOVER_TOOLS=COMMITTED; ISOLATED_TESTS=PASS; HOMESERVER_MIGRATION=NOT_EXECUTED; SERVER_TLS_PRESENTATION=NOT_TESTED; AUTO_ENROLLMENT_ENDPOINT=NOT_DEPLOYED.** Ни одна служба Homeserver не изменялась через ChatGPT. Не выдавать наличие скрипта или тесты с fake Docker за совершённую серверную миграцию.
