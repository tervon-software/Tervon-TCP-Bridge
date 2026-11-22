# M6 Stage 4 — Android-обвязка Bridge: исходники-кандидаты, не новая инструкция установки

**Актуально на 2026-11-22.** Этот каталог содержит **историю исходников и установщиков**, созданных при старой M6-интеграции. Он не является единственным актуальным исходным деревом Android-приложения и **не разрешает повторно применять последовательность устаревших патчей**.

Рабочий план: [docs/DEVELOPMENT_PLAN.md](../../../docs/DEVELOPMENT_PLAN.md). Инвентаризация GitHub и Android snapshot: [docs/M6_AUDIT_2026-11-22.md](../../../docs/M6_AUDIT_2026-11-22.md).

## Что существует

- Identity/CSR/PKI: `BridgeDeviceIdentity.kt`, `BridgeDeviceCsr.kt`, `BridgeAutoCertificateEnrollment.kt`, `BridgeAutoEnrollmentHttp.kt`, `BridgeServerCertificateStore.kt`, `BridgeClientPemHandoff.kt`. Это **сохранённые для ревизии механизмы**, не повод реализовывать новую регистрацию.
- Xray/Bridge маршрутизация: `BridgeVpnSession.kt`, `BridgeXrayConfigRoute.kt`, `BridgeVlessRoutePlan.kt`, `BridgeClientConfigBuilder.kt`. Они требуют аудита зависимостей до решения об очистке или замене. Название `BridgeVlessRoutePlan` историческое; универсальный TCP не должен зависеть от VLESS/Trojan allowlist.
- Доверие/каталог: `BridgeServerReconciler.kt`, `BridgeSignedServerCatalog.kt`, `BridgeServerCatalogTrust.kt`, `BridgeCatalogTrustAnchors.kt`. Не удалять и не дублировать до выяснения нынешней реально используемой цепочки.
- UI/диагностика: `bridge_settings_screen.dart`, `bridge_monitor_screen.dart`, `BridgeDiagnostics.kt`; актуальный работающий Android-проект содержит и другие bridge файлы. Следует проверить реальные call-site.
- Множество `install_*.py`, `apply_*.py`, `*.patch`: **LEGACY/STAGING**, не применять по старым инструкциям. Некоторые рассчитаны на точные прошлые SHA и закономерно не соответствуют актуальному приложению.

## Что реально проверено на устройстве

22 ноября оператор собрал и установил APK с обновлённым Go ARM64 `libtervonbridge.so` и диагностикой. Kotlin compile PASS, Go test/race/vet + ARM64 build PASS, но **сквозной TCP через Bridge не работает**: XHTTP `EOF`, локальные TCP resets, Bridge `UNKNOWN`, интернет отсутствует. Это `ANDROID_E2E=FAIL`, не `PASS`.

## Следующий утверждённый процесс

1. Сначала инвентаризация фактического Android проекта, GitHub, серверного API и проверенных компонент ядра.
2. Составить зависимость `файл → вызывающий код → сохранение/удаление/очистка` по действующему приложению.
3. **До отдельного пофайлового согласования ничего не удалять и не писать новых файлов обвязки.**
4. При разработке оставлять один путь: Xray TCP → Go Encapsulator → HTTP/2 TLS 1.3 mTLS → существующий Decapsulator → существующий серверный TCP backend.
5. Проверить побайтовую двустороннюю TCP идентичность и работающий VPN интернет, а не только локальный listener или сборку.

**Важно:** реальный исходный Android проект хранится отдельно от публичного репозитория Bridge; не публиковать его приватные файлы и ключи в этом репозитории. Git history сохраняет предыдущие команды и версии этого README для расследования, но они более не являются действующими инструкциями.
