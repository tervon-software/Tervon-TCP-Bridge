#!/usr/bin/env bash
# Prepare isolated, TEST-ONLY PKI and Decapsulator config for Android ingress.
# Does not start containers, change services, publish ports or reuse live PKI.
set -Eeuo pipefail
umask 077

if [[ $# -ne 1 || "$1" != /* || "$1" == "/" ]]; then
  echo "USAGE: sudo bash prepare-local-fixture.sh /absolute/new/private/lab-dir" >&2
  exit 2
fi
if (( EUID != 0 )); then
  echo "ERROR: run as root to set restricted ownership" >&2
  exit 2
fi
for cmd in openssl python3 install chown chmod; do
  command -v "$cmd" >/dev/null || { echo "ERROR: missing $cmd" >&2; exit 2; }
done
root="${1%/}"
if [[ -e "$root" || -L "$root" ]]; then
  echo "ERROR: target already exists; refusing to overwrite: $root" >&2
  exit 2
fi
# Prevent accidentally storing the test CA inside the repository checkout.
if [[ "$root" == *"/tervon-tcp-bridge-m6-android-ingress"* ]]; then
  echo "ERROR: choose a private directory outside the Git checkout" >&2
  exit 2
fi

install -d -m 0700 "$root"
install -d -m 0700 "$root/private"
install -d -m 0700 "$root/config" "$root/config/pki"
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$root/private/lab-ca.key"
openssl req -new -x509 -sha256 -days 2 \
  -key "$root/private/lab-ca.key" -out "$root/private/lab-ca.crt" \
  -subj '/CN=Tervon M6 ephemeral lab CA' \
  -addext 'basicConstraints=critical,CA:TRUE,pathlen:0' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign'

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$root/config/pki/server.key"
openssl req -new -sha256 -key "$root/config/pki/server.key" \
  -out "$root/private/server.csr" -subj '/CN=bridge.android.test'
cat > "$root/private/server-ext.cnf" <<'EOF'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=serverAuth
subjectAltName=DNS:bridge.android.test
EOF
openssl x509 -req -sha256 -days 1 \
  -in "$root/private/server.csr" \
  -CA "$root/private/lab-ca.crt" -CAkey "$root/private/lab-ca.key" \
  -CAcreateserial -out "$root/config/pki/server.crt" \
  -extfile "$root/private/server-ext.cnf" >/dev/null 2>&1

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$root/private/lab-client.key"
openssl req -new -sha256 -key "$root/private/lab-client.key" \
  -out "$root/private/lab-client.csr" -subj '/CN=tervon-m6-local-test'
cat > "$root/private/client-ext.cnf" <<'EOF'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=clientAuth
EOF
openssl x509 -req -sha256 -days 1 \
  -in "$root/private/lab-client.csr" \
  -CA "$root/private/lab-ca.crt" -CAkey "$root/private/lab-ca.key" \
  -CAcreateserial -out "$root/private/lab-client.crt" \
  -extfile "$root/private/client-ext.cnf" >/dev/null 2>&1

install -m 0644 "$root/private/lab-ca.crt" "$root/config/pki/client-ca.crt"
openssl x509 -in "$root/private/lab-client.crt" -outform DER \
  | openssl dgst -sha256 -r \
  | awk '{print $1}' > "$root/private/client.sha256"
fingerprint="$(cat "$root/private/client.sha256")"
if [[ ! "$fingerprint" =~ ^[0-9a-f]{64}$ ]]; then
  echo "ERROR: invalid SHA-256 fingerprint" >&2
  exit 1
fi

LAB_CONFIG_ROOT="$root" LAB_CLIENT_FINGERPRINT="$fingerprint" python3 - <<'PY'
import json, os, pathlib
root = pathlib.Path(os.environ["LAB_CONFIG_ROOT"])
fp = os.environ["LAB_CLIENT_FINGERPRINT"]
server = {
    "mode": "server",
    "listen": "0.0.0.0:8443",
    "cert_file": "/config/pki/server.crt",
    "key_file": "/config/pki/server.key",
    "ca_file": "/config/pki/client-ca.crt",
    "max_streams": 0,
    "routes": {"vpn": [{"Name": "existing-vless-tcp", "Address": "host.docker.internal:30000"}]},
    "authorized": {fp: ["vpn"]},
    "padding_enabled": True,
    "padding_rate_enabled": True,
    "jitter_enabled": True,
}
(root / "config/server.json").write_text(json.dumps(server, indent=2) + "\n")
client = {
    "mode": "client",
    "listen": "127.0.0.1:19001",
    "cert_file": str(root / "private/lab-client.crt"),
    "key_file": str(root / "private/lab-client.key"),
    "ca_file": str(root / "private/lab-ca.crt"),
    "server_name": "bridge.android.test",
    "remote_url": "https://127.0.0.1:18443/v2/stream",
    "service": "vpn",
    "max_local_connections": 0,
    "padding_enabled": True,
    "padding_rate_enabled": True,
    "jitter_enabled": True,
}
(root / "private/client.json").write_text(json.dumps(client, indent=2) + "\n")
PY

# Container runs as uid/gid 65532. Only its server key is available to it.
chown root:65532 "$root/config" "$root/config/pki" "$root/config/pki/server.key" "$root/config/server.json"
chmod 0750 "$root/config" "$root/config/pki"
chmod 0640 "$root/config/pki/server.key" "$root/config/server.json"
chmod 0644 "$root/config/pki/server.crt" "$root/config/pki/client-ca.crt"
chmod 0700 "$root" "$root/private"
chmod 0600 "$root/private/"*
openssl verify -CAfile "$root/private/lab-ca.crt" -purpose sslserver "$root/config/pki/server.crt" >/dev/null
openssl verify -CAfile "$root/private/lab-ca.crt" -purpose sslclient "$root/private/lab-client.crt" >/dev/null

echo "M6_LAB_PKI=PASS"
echo "M6_LAB_SERVER_JSON=CREATED"
echo "M6_LAB_CONFIG_DIR=$root/config"
echo "M6_LAB_CLIENT_JSON=$root/private/client.json"
echo "M6_LAB_EXPIRES_WITHIN_DAYS=1"
echo "M6_LAB_RUNTIME=NOT_STARTED"
