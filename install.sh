#!/usr/bin/env bash
set -euo pipefail
repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
printf '%s\n' '1) Android TervonVPN DPI → сервер' '2) Linux → Linux'
read -rp 'Сценарий установки [1]: ' scenario
case "${scenario:-1}" in
    1) script=install-m6.py ;;
    2) script=install-pair.py ;;
    *) echo 'Выберите 1 или 2' >&2; exit 2 ;;
esac
exec python3 "$repo/integrations/android-ingress/$script" "$@"
