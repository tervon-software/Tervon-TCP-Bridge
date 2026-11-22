#!/usr/bin/env bash
# Create once-only production PKI for the Android-facing Decapsulator.
# Does not start or change a container, open ports, or alter any other service.
set -Eeuo pipefail
umask 077

if (( EUID != 0 )) || [[ $# -ne 2 ]]; then
  echo "Usage (root): bash prepare-production-pki.sh /etc/tervon/bridge-android bridge.your-domain.example" >&2
  exit 2
fi
dir="${1%/}"
name="$2"
if [[ "$dir" != /* || "$dir" == "/" || -e "$dir" || -L "$dir" ]]; then
  echo "ERROR: first argument must be an ABSOLUTE directory that does NOT exist" >&2
  exit 2
fi
if [[ "$dir" == *"/tervon-tcp-bridge-"* || "$dir" == *"/tervon-bridge-m6-lab"* ]]; then
  echo "ERROR: production PKI cannot be placed inside Git or laboratory directories" >&2
  exit 2
fi
PROD_DNS_NAME="$name" python3 - <<'PY'
import os, re, sys
name = os.environ["PROD_DNS_NAME"]
labels = name.split(".")
if len(name) > 253 or len(labels) < 2 or any(
    not 1 <= len(l) <= 63 or
    not re.fullmatch(r"[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?", l)
    for l in labels
):
    sys.exit("ERROR: specify a valid ASCII FQDN that this server actually controls")
PY
for cmd in openssl install chown chmod python3; do
  command -v "$cmd" >/dev/null || { echo "ERROR: missing $cmd" >&2; exit 2; }
done
# Resolve BEFORE creating keys, directories, or server.json. The user-selected
# VPN backend is defined by Xray, not by a fixed TCP port in this installer.
xray_config="${M6_XRAY_CONFIG:-/usr/local/x-ui/bin/config.json}"
backend_port="$(python3 "$(dirname "$0")/resolve_vless_tcp_port.py" --xray-config "$xray_config")" || {
  echo "ERROR: VLESS TCP backend selection failed; production PKI unchanged" >&2
  exit 2
}
[[ "$backend_port" =~ ^[0-9]+$ ]] || {
  echo "ERROR: backend port validation failed" >&2
  exit 2
}

install -d -m 0700 "$dir" "$dir/private" "$dir/config" "$dir/config/pki"
# Separate CAs: server identity and Android device identities have different trust duties.
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$dir/private/server-ca.key"
openssl req -new -x509 -sha256 -days 3650 \
  -key "$dir/private/server-ca.key" -out "$dir/private/server-ca.crt" \
  -subj '/CN=Tervon Android Bridge server CA' \
  -addext 'basicConstraints=critical,CA:TRUE,pathlen:0' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign'
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$dir/private/client-ca.key"
openssl req -new -x509 -sha256 -days 3650 \
  -key "$dir/private/client-ca.key" -out "$dir/config/pki/client-ca.crt" \
  -subj '/CN=Tervon Android Bridge client CA' \
  -addext 'basicConstraints=critical,CA:TRUE,pathlen:0' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign'

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$dir/config/pki/server.key"
openssl req -new -sha256 -key "$dir/config/pki/server.key" \
  -out "$dir/private/server.csr" -subj "/CN=$name"
cat > "$dir/private/server-ext.cnf" <<EOF
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:$name
EOF
openssl x509 -req -sha256 -days 365 \
  -in "$dir/private/server.csr" \
  -CA "$dir/private/server-ca.crt" -CAkey "$dir/private/server-ca.key" \
  -CAcreateserial -out "$dir/config/pki/server.crt" \
  -extfile "$dir/private/server-ext.cnf" >/dev/null 2>&1

PROD_OUTPUT_DIR="$dir" PROD_DNS_NAME="$name" PROD_TCP_BACKEND_PORT="$backend_port" python3 - <<'PY'
import json, os, pathlib
root = pathlib.Path(os.environ["PROD_OUTPUT_DIR"])
conf = {
  "mode": "server",
  "listen": "0.0.0.0:8443",
  "cert_file": "/config/pki/server.crt",
  "key_file": "/config/pki/server.key",
  "ca_file": "/config/pki/client-ca.crt",
  "max_streams": 0,
  "routes": {"vpn": [{"Name":"existing-vless-tcp",
                       "Address":"host.docker.internal:" + os.environ["PROD_TCP_BACKEND_PORT"]}]},
  # Deny-all bootstrap: valid hex satisfies the current parser but matches no actual cert.
  # Enrollment must replace it with approved fingerprints before Android access is enabled.
  "authorized": {"0"*64: ["vpn"]},
  "padding_enabled": True,
  "padding_rate_enabled": True,
  "jitter_enabled": True,
}
(root / "config/server.json").write_text(json.dumps(conf, indent=2) + "\n")
PY

# Container receives config and public client CA, but NO CA private key.
chown root:65532 "$dir/config" "$dir/config/pki" "$dir/config/pki/server.key" "$dir/config/server.json"
chmod 0750 "$dir/config" "$dir/config/pki"
chmod 0640 "$dir/config/pki/server.key" "$dir/config/server.json"
chmod 0644 "$dir/config/pki/server.crt" "$dir/config/pki/client-ca.crt"
chmod 0700 "$dir" "$dir/private"
chmod 0600 "$dir/private/"*
openssl verify -CAfile "$dir/private/server-ca.crt" \
  -purpose sslserver "$dir/config/pki/server.crt" >/dev/null

echo "M6_PRODUCTION_PKI=CREATED"
echo "M6_PRODUCTION_CONFIG=$dir/config"
echo "M6_TCP_BACKEND_SOURCE=XRAY_CONFIG"
echo "M6_PRODUCTION_SERVER_NAME=$name"
echo "M6_PRODUCTION_CLIENT_AUTH=DENY_ALL_UNTIL_ENROLLED"
echo "M6_PRODUCTION_RUNTIME=NOT_MODIFIED"
echo "M6_CERT_RENEWAL_REQUIRED_WITHIN_DAYS=365"
