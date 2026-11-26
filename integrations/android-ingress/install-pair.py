#!/usr/bin/env python3
"""Install one role of a two-server Tervon TCP Bridge."""

import argparse
import hashlib
import ipaddress
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import sys

spec = importlib.util.spec_from_file_location("install_m6", Path(__file__).with_name("install-m6.py"))
m6 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m6)

ROLE_CLIENT = "encapsulator"
ROLE_SERVER = "decapsulator"
CONFIG_BASE = Path("/etc")


def require_new(path):
    if path.exists() or path.is_symlink():
        raise ValueError(f"уже существует: {path}")


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def choice(prompt, options):
    value = m6.ask(prompt)
    if value not in options:
        raise ValueError(f"выберите {' или '.join(options)}")
    return value


def service_name():
    service = m6.ask("Имя TCP-сервиса", "relay")
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,63}", service):
        raise ValueError("некорректное имя сервиса")
    return service


def backend_address(mode):
    value = m6.ask("Адрес TCP backend, IP или DNS-имя",
                   "host.docker.internal" if mode == "docker" else "127.0.0.1")
    if ":" in value:
        value = f"[{ipaddress.IPv6Address(value)}]"
    elif not re.fullmatch(r"[A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?", value):
        raise ValueError("некорректный адрес backend")
    return f"{value}:{m6.port('Порт TCP backend', None)}"


def prepare_client(root, check):
    require_new(root)
    if check:
        return
    (root / "config/pki").mkdir(parents=True)
    key = root / "config/pki/client.key"
    csr = root / "client.csr"
    m6.run("openssl", "genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256", "-out", key)
    m6.run("openssl", "req", "-new", "-key", key, "-subj", "/CN=Tervon interserver client", "-out", csr)
    print(f"Передайте Decapsulator только CSR: {csr}")


def issue_client(root, csr):
    config = root / "config/pki"
    private = root / "private"
    exported = root / "export"
    exported.mkdir()
    cert = exported / "client.crt"
    extensions = private / "client-ext.cnf"
    extensions.write_text("basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\n")
    m6.run("openssl", "x509", "-req", "-sha256", "-days", "365", "-in", csr,
           "-CA", config / "client-ca.crt", "-CAkey", private / "client-ca.key",
           "-CAcreateserial", "-extfile", extensions, "-out", cert)
    m6.run("openssl", "verify", "-CAfile", config / "client-ca.crt", "-purpose", "sslclient", cert)
    shutil.copyfile(private / "server-ca.crt", exported / "server-ca.crt")
    der = subprocess.check_output(["openssl", "x509", "-in", str(cert), "-outform", "DER"])
    return hashlib.sha256(der).hexdigest()


def import_client(root, exported, check):
    key = root / "config/pki/client.key"
    cert = exported / "client.crt"
    ca = exported / "server-ca.crt"
    if not key.is_file() or not cert.is_file() or not ca.is_file():
        raise ValueError("отсутствует локальный ключ или полученный комплект сертификатов")
    for dest in ("client.crt", "server-ca.crt"):
        require_new(root / "config/pki" / dest)
    def public(*args):
        return m6.run("openssl", *args, capture_output=True).stdout.strip()
    if public("x509", "-in", cert, "-pubkey", "-noout") != public("pkey", "-in", key, "-pubout"):
        raise ValueError("сертификат клиента выпущен для другого ключа")
    m6.run("openssl", "x509", "-in", cert, "-checkend", "0", "-noout")
    m6.run("openssl", "x509", "-in", ca, "-checkend", "0", "-noout")
    if not check:
        for source in (cert, ca):
            shutil.copyfile(source, root / "config/pki" / source.name)


def install_service(root, app, role, mode, config, host_port):
    directory = root / "config"
    pki = directory / "pki"
    for key in ("cert_file", "key_file", "ca_file"):
        name = config[key]
        config[key] = str(pki / name) if mode == "native" else "/config/pki/" + name
    path = directory / "bridge.json"
    write_json(path, config)
    for f in (root, directory, pki, path, *pki.iterdir()):
        os.chown(f, 0, 65532)
        f.chmod(0o750 if f.is_dir() else 0o640)
    app.mkdir()
    app.chmod(0o755)
    binary = app / "tervon-bridge"
    m6.run("curl", "--proto", "=https", "-fsSL", "--max-time", "120", "--output", binary, m6.ASSET)
    with binary.open("rb") as data:
        if hashlib.file_digest(data, "sha256").hexdigest() != m6.SHA256:
            raise ValueError("SHA-256 ядра не совпал")
    binary.chmod(0o755)

    name = f"tervon-pair-{role}"
    unit = name + ".service"
    if mode == "native":
        capability = "CAP_NET_BIND_SERVICE" if host_port < 1024 else ""
        body = f"""[Unit]
Description=Tervon interserver {role}
After=network-online.target
Wants=network-online.target

[Service]
ExecStart={binary} -config {path}
User=65532
Group=65532
NoNewPrivileges=yes
CapabilityBoundingSet={capability}
AmbientCapabilities={capability}
Restart=on-failure
RestartSec=2s
"""
    else:
        image = name + ":v1.0.2-m6"
        (app / "Dockerfile").write_text("""FROM gcr.io/distroless/static-debian12:nonroot
COPY --chmod=0755 tervon-bridge /tervon-bridge
USER 65532:65532
ENTRYPOINT ["/tervon-bridge","-config","/config/bridge.json"]
""")
        m6.run("docker", "build", "--platform", "linux/amd64", "-t", image, app)
        bind = "127.0.0.1" if role == ROLE_CLIENT else "0.0.0.0"
        compose = app / "compose.json"
        write_json(compose, {"services": {role: {
            "image": image, "container_name": name, "pull_policy": "never",
            "restart": "unless-stopped", "read_only": True, "cap_drop": ["ALL"],
            "security_opt": ["no-new-privileges:true"],
            "ports": [f"{bind}:{host_port}:8443/tcp"],
            "extra_hosts": ["host.docker.internal:host-gateway"],
            "volumes": [f"{directory}:/config:ro"],
        }}})
        docker = shutil.which("docker")
        body = f"""[Unit]
Description=Tervon interserver {role} (Docker)
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart={docker} compose -p {name} -f {compose} up -d --no-build
ExecStop={docker} compose -p {name} -f {compose} stop
"""
    unit_path = m6.write_unit(unit, body)
    m6.run("systemd-analyze", "verify", unit_path)
    m6.run("systemctl", "daemon-reload")
    m6.run("systemctl", "enable", "--now", unit)
    m6.run("systemctl", "is-active", "--quiet", unit)
    if mode == "docker":
        state = m6.run("docker", "inspect", "--format", "{{.State.Running}}", name, capture_output=True)
        if state.stdout.strip() != "true":
            raise ValueError("сервис Compose запущен, контейнер Bridge остановлен")
    print(f"BRIDGE_ROLE_STARTED={role} UNIT={unit}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="проверить ввод без изменений")
    check = parser.parse_args().check
    if os.geteuid() or platform.system() != "Linux" or platform.machine() != "x86_64":
        raise ValueError("нужен root и Linux x86-64")
    if sys.version_info < (3, 11):
        raise ValueError("нужен Python 3.11 или новее")
    os.umask(0o077)
    role = choice("Роль: encapsulator или decapsulator", (ROLE_CLIENT, ROLE_SERVER))
    name = f"tervon-pair-{role}"
    root = CONFIG_BASE / name
    app = Path("/opt") / name
    if role == ROLE_CLIENT and choice("Этап Encapsulator: prepare или activate", ("prepare", "activate")) == "prepare":
        prepare_client(root, check)
        return
    require_new(app)
    require_new(m6.UNITS / (name + ".service"))
    mode = choice("Тип установки: native или docker", ("native", "docker"))
    if mode == "docker":
        m6.run("docker", "compose", "version", stdout=subprocess.DEVNULL)
        for args in (("inspect", "--type", "container", name), ("image", "inspect", name + ":v1.0.2-m6")):
            if subprocess.run(["docker", *args], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
                raise ValueError("контейнер или образ уже существует")
    domain = m6.hostname("DNS-имя Decapsulator")
    service = service_name()
    core_port = 8443
    if role == ROLE_SERVER:
        require_new(root)
        host_port = m6.port("Порт Decapsulator", 18443)
        backend = backend_address(mode)
        csr = Path(m6.ask("Путь к CSR Encapsulator")).resolve(strict=True)
        if not csr.is_file() or csr.stat().st_size > 8192:
            raise ValueError("некорректный CSR")
        m6.run("openssl", "req", "-in", csr, "-verify", "-noout")
        m6.free_listener("0.0.0.0", host_port)
        if check:
            print("BRIDGE_PAIR_PREFLIGHT=PASS")
            return
        if m6.ask("Установить Decapsulator и подписать CSR: yes или no", "no") != "yes":
            return
        m6.run("bash", m6.SOURCE / "prepare-production-pki.sh", root, domain,
               env=dict(os.environ, M6_EMPTY_POLICY="1"))
        fingerprint = issue_client(root, csr)
        config = {
            "mode": "server", "listen": f"0.0.0.0:{host_port if mode == 'native' else core_port}",
            "cert_file": "server.crt", "key_file": "server.key", "ca_file": "client-ca.crt",
            "max_streams": 0, "routes": {service: [{"Name": service, "Address": backend}]},
            "authorized": {fingerprint: [service]},
        }
    else:
        public_port = m6.port("Порт удалённого Decapsulator", 18443)
        host_port = m6.port("Локальный порт Encapsulator (loopback)", 10080)
        exported = Path(m6.ask("Каталог полученных client.crt и server-ca.crt")).resolve(strict=True)
        m6.free_listener("127.0.0.1", host_port)
        import_client(root, exported, True)
        if check:
            print("BRIDGE_PAIR_PREFLIGHT=PASS")
            return
        if m6.ask("Установить и запустить Encapsulator: yes или no", "no") != "yes":
            return
        import_client(root, exported, False)
        bind = f"127.0.0.1:{host_port}" if mode == "native" else f"0.0.0.0:{core_port}"
        config = {
            "mode": "client", "listen": bind, "remote_url": f"https://{domain}:{public_port}/v2/stream",
            "server_name": domain, "service": service, "max_local_connections": 0,
            "cert_file": "client.crt", "key_file": "client.key", "ca_file": "server-ca.crt",
        }
    install_service(root, app, role, mode, config, host_port)
    if role == ROLE_SERVER:
        print(f"Передайте Encapsulator только {root / 'export' / 'client.crt'} и {root / 'export' / 'server-ca.crt'}")
    else:
        print(f"Проверьте обмен с TCP-сервисом через 127.0.0.1:{host_port}")


if __name__ == "__main__":
    try:
        main()
    except (EOFError, KeyboardInterrupt, OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"BRIDGE_PAIR_INSTALL=FAILED: {error}", file=sys.stderr)
        sys.exit(1)
