#!/usr/bin/env python3
"""Install the Android-facing M6 Bridge on a fresh systemd deployment."""

import argparse
import getpass
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import platform
import re
import shutil
import socket
import subprocess
import sys
import tempfile
import time
from urllib.parse import urlsplit

SOURCE = Path(__file__).resolve().parent
ROOT = Path("/etc/tervon/bridge-android")
APP = Path("/opt/tervon-bridge-m6")
UNITS = Path("/etc/systemd/system")
REGISTRAR = "tervon-m6-auto-registrar.service"
NATIVE = "tervon-m6-decapsulator.service"
COMPOSE = "tervon-m6-compose.service"
CONTAINER = "tervon-m6-decapsulator"
IMAGE = "tervon-bridge-android-ingress:v1.0.2"
CORE = "tervon-bridge-linux-amd64-v1.0.2-m6"
SHA256 = "15af985ec45811d55d0ffe9c1cbc9c5c496a888ccd58e83ec6ca711cb09187a6"
ASSET = f"https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.2/{CORE}"
DISCOVERY = "/.well-known/tervon-bridge"
PROBE_FILES = (
    "live_xray_evidence.py", "check_panel_xray_bridge_routes.py", "check_xray_grpc_runtime.py",
)


def run(*args, **kwargs):
    return subprocess.run([str(arg) for arg in args], check=True, text=True, **kwargs)


def ask(label, default=None):
    value = input(f"{label}" + (f" [{default}]" if default else "") + ": ").strip()
    return value or default or ""


def port(label, default):
    value = int(ask(label, str(default) if default is not None else None))
    if not 1 <= value <= 65535:
        raise ValueError(f"{label}: допустимо 1–65535")
    return value


def hostname(label):
    value = ask(label).lower()
    if len(value) > 253 or not re.fullmatch(
        r"(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?",
        value,
    ):
        raise ValueError(f"{label}: укажите DNS-имя")
    return value


def stage_nginx(text, domain, public_port, registrar_port):
    # Tokenize only structure. Preserve the original text, quoting and comments.
    tokens = [
        m for m in re.finditer(r'#.*|"(?:\\.|[^"\\])*"|\x27(?:\\.|[^\x27\\])*\x27|\$\{[^}]+\}|[^\s;{}#]+|[;{}]', text)
        if not m[0].startswith("#")
    ]
    stack, statement, servers = [], [], []
    for token in tokens:
        if token[0] == "{":
            block = (statement, token.end(), [])
            if stack:
                stack[-1][2].append((statement, None))
            stack.append(block)
            statement = []
        elif token[0] == "}":
            if not stack or statement:
                raise ValueError("не удалось разобрать структуру nginx")
            head, start, children = stack.pop()
            if head and head[0][0] == "server":
                servers.append((start, children))
        elif token[0] == ";":
            if stack:
                stack[-1][2].append((statement, token.end()))
            statement = []
        else:
            statement.append(token)
    if stack or statement or DISCOVERY in text:
        raise ValueError("Discovery уже настроен или конфигурация nginx неоднозначна")

    def words(tokens):
        return [m[0].strip("\"'") for m in tokens]

    candidates = []
    for start, children in servers:
        directives = [words(row) for row, _ in children]
        if not any(row and row[0] == "server_name" and domain in row[1:] for row in directives):
            continue
        if any(len(row) > 2 and row[0] == "listen" and "ssl" in row[2:]
               and row[1].rsplit(":", 1)[-1] == str(public_port) for row in directives):
            candidates.append((start, children))
    if len(candidates) != 1:
        raise ValueError("нужен один HTTPS server-блок с указанными доменом и портом")
    start, children = candidates[0]
    redirects = [(row, end) for row, end in children if row and row[0][0] in ("return", "rewrite")]
    if redirects:
        if (len(redirects) != 1 or redirects[0][1] is None
                or words(redirects[0][0])[:2] not in (["return", "301"], ["return", "302"])
                or any(words(row)[:2] == ["location", "/"] for row, _ in children)):
            raise ValueError("неоднозначный редирект nginx на уровне server")
        row, end = redirects[0]
        begin = row[0].start()
        text = text[:begin] + f"location / {{ {text[begin:end]} }}" + text[end:]
    location = (
        f"\n    location = {DISCOVERY} {{\n"
        f"        proxy_pass http://127.0.0.1:{registrar_port};\n"
        "        proxy_http_version 1.1;\n"
        "        proxy_set_header Host $host;\n"
        "        proxy_set_header X-Forwarded-Proto https;\n"
        "        proxy_read_timeout 30s;\n"
        "        client_max_body_size 16k;\n"
        "    }\n"
    )
    return text[:start] + location + text[start:]


def atomic_write(path, content):
    st = path.stat()
    with tempfile.NamedTemporaryFile(dir=path.parent, delete=False, mode="w") as f:
        tmp = Path(f.name)
        f.write(content)
    try:
        os.chown(tmp, st.st_uid, st.st_gid)
        os.chmod(tmp, st.st_mode & 0o777)
        os.replace(tmp, path)
    finally:
        tmp.unlink(missing_ok=True)


def write_unit(name, body):
    path = UNITS / name
    with path.open("x") as f:
        f.write(body + "\n[Install]\nWantedBy=multi-user.target\n")
    return path


def free_listener(host, number):
    with socket.socket() as handle:
        handle.bind((host, number))


def verify_discovery(url, domain, number, server_ca):
    expected = f"https://{domain}:{number}/v2/stream"
    deadline = time.monotonic() + 120
    while time.monotonic() < deadline:
        try:
            response = json.loads(run(
                "curl", "--noproxy", "*", "-fsS", "--max-time", "5", url,
                capture_output=True,
            ).stdout)
            if (response.get("available") is True and response.get("remote_url") == expected
                    and response.get("server_ca") == server_ca):
                return True
        except (ValueError, AttributeError, subprocess.CalledProcessError):
            pass
        time.sleep(2)
    return False


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="проверить ввод и nginx без установки")
    args = parser.parse_args()
    if os.geteuid() or platform.system() != "Linux" or platform.machine() != "x86_64":
        raise ValueError("нужны root и Linux x86-64")
    if sys.version_info < (3, 11):
        raise ValueError("нужен Python 3.11 или новее")
    os.umask(0o077)
    mode = ask("Тип установки: native или docker", "native")
    if mode not in {"native", "docker"}:
        raise ValueError("выберите native или docker")
    for command in ("go", "openssl", "nginx", "systemctl", "systemd-analyze", "curl", "nsenter", "ss", "ps"):
        if not shutil.which(command):
            raise ValueError(f"нет зависимости: {command}")
    if mode == "docker":
        run("docker", "compose", "version", stdout=subprocess.DEVNULL)
        run("docker", "info", "--format", "{{.ServerVersion}}", stdout=subprocess.DEVNULL)
        existing_image = subprocess.run(["docker", "image", "inspect", IMAGE],
                                        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if existing_image.returncode == 0:
            raise ValueError("тег образа уже занят, существующий Docker-образ сохранён")
        existing = run("docker", "ps", "-a", "--filter", f"name=^/{CONTAINER}$",
                       "--format", "{{.Names}}", capture_output=True)
        if existing.stdout.strip():
            raise ValueError("контейнер установщика уже существует")
    if any(p.exists() or p.is_symlink() for p in (ROOT, APP, *(UNITS / u for u in (REGISTRAR, NATIVE, COMPOSE)))):
        raise ValueError("обнаружена существующая установка")
    for unit in (REGISTRAR, NATIVE, COMPOSE):
        state = subprocess.run(["systemctl", "show", unit, "-p", "LoadState", "--value"],
                               capture_output=True, text=True, check=False)
        if state.stdout.strip() != "not-found":
            raise ValueError(f"служба уже существует: {unit}")

    sub = hostname("DNS-имя из URL подписки")
    discovery_port = port("Фактический HTTPS-порт подписки/Discovery", None)
    registration_port = port("Локальный порт Registrar", 18763)
    bridge = hostname("DNS-имя транспорта Bridge")
    bridge_port = port("Внешний TCP-порт транспорта Bridge", 18443)
    if len({discovery_port, registration_port, bridge_port}) != 3:
        raise ValueError("порты Discovery, транспорта и Registrar должны различаться")
    panel = ask("Панель: 3x-ui или remnawave", "3x-ui")
    if panel not in {"3x-ui", "remnawave"}:
        raise ValueError("неизвестная панель")
    base = ask("HTTPS base URL панели")
    parsed = urlsplit(base)
    if (parsed.scheme != "https" or not parsed.hostname or parsed.username is not None
            or parsed.query or parsed.fragment or any(c.isspace() for c in base)
            or (panel == "3x-ui" and "/panel/api" in parsed.path)
            or (panel == "remnawave" and parsed.path not in ("", "/"))):
        raise ValueError("некорректный HTTPS base URL панели")
    _ = parsed.port
    settings = {"type": panel, "base_url": base, "credential_file": str(ROOT / "private/panel.header")}
    if panel == "remnawave":
        node = ask("UUID ноды Remnawave")
        if not re.fullmatch(r"[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}", node):
            raise ValueError("некорректный UUID ноды")
        settings["node_uuid"] = node
    else:
        loopback = ask("API панели через loopback с TLS-проверкой: yes или no", "yes")
        if loopback not in {"yes", "no"}:
            raise ValueError("укажите yes или no")
        settings["connect_loopback"] = loopback == "yes"
    owner = ask("systemd unit владельца Xray", "x-ui.service")
    executable = ask("Имя исполняемого файла Xray", "xray-linux-amd64")
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.@-]*\.service", owner) or not re.fullmatch(
        r"[A-Za-z0-9][A-Za-z0-9_.-]*", executable
    ):
        raise ValueError("проверьте имя службы и бинарника Xray")
    backend = "host.docker.internal" if mode == "docker" else str(ipaddress.ip_address(
        ask("Локальный IP inbound Xray для native", "127.0.0.1")
    ))
    vhost = Path(ask("Путь к существующему HTTPS vhost подписки из nginx -T")).resolve(strict=True)
    original = vhost.read_text()
    nginx_port = port("Порт listen в HTTPS-блоке nginx", discovery_port)
    candidate = stage_nginx(original, sub, nginx_port, registration_port)
    run("nginx", "-t")
    for unit in ("nginx.service", owner):
        run("systemctl", "is-active", "--quiet", unit)
    free_listener("127.0.0.1", registration_port)
    free_listener("0.0.0.0", bridge_port)
    url = f"https://{sub}:{discovery_port}{DISCOVERY}"
    print(f"Режим: {mode}. Discovery: {url}. Bridge: {bridge}:{bridge_port}")
    if args.check:
        print("M6_INSTALL_PREFLIGHT=PASS")
        return 0
    if ask("Установить и перечитать nginx: yes или no", "no") != "yes":
        return 0
    token = getpass.getpass("API-токен панели: ").strip()
    if not token or any(c.isspace() for c in token):
        raise ValueError("некорректный API-токен")

    def root_dir(path):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.mkdir()
        path.chmod(0o755)

    with tempfile.TemporaryDirectory(prefix="tervon-m6-build-") as tmp:
        tmp = Path(tmp)
        core = tmp / "tervon-bridge"
        registrar = tmp / "m6-auto-registrar"
        run("curl", "-fLSs", "--retry", "2", "--output", core, ASSET)
        with core.open("rb") as data:
            digest = hashlib.file_digest(data, "sha256").hexdigest()
        if digest != SHA256:
            raise ValueError("SHA-256 ядра не совпал")
        run("go", "build", "-trimpath", "-o", registrar, "./cmd/m6-auto-registrar", cwd=SOURCE / "xui")
        root_dir(APP)
        for file in (core, registrar):
            shutil.copyfile(file, APP / file.name)
            (APP / file.name).chmod(0o755)
    for name in PROBE_FILES:
        shutil.copyfile(SOURCE / name, APP / name)

    env = dict(os.environ, M6_EMPTY_POLICY="1")
    run("bash", SOURCE / "prepare-production-pki.sh", ROOT, bridge, env=env)
    private = ROOT / "private"
    (private / "panel.header").write_text(f"Authorization: Bearer {token}\n")
    (private / "panel.json").write_text(json.dumps(settings, indent=2) + "\n")
    del token

    conf = ROOT / "config/server.json"
    if mode == "native":
        config = json.loads(conf.read_text())
        config["listen"] = f"0.0.0.0:{bridge_port}"
        for key in ("cert_file", "key_file", "ca_file"):
            config[key] = str(ROOT / config[key].lstrip("/"))
        conf.write_text(json.dumps(config, indent=2) + "\n")
        os.chown(ROOT, 0, 65532)
        ROOT.chmod(0o750)
        capability = "CAP_NET_BIND_SERVICE" if bridge_port < 1024 else ""
        dependency = NATIVE
        target = f"--bridge-unit={NATIVE} --bridge-binary={APP}/tervon-bridge --bridge-binary-sha256={SHA256} --bridge-public-port={bridge_port}"
        write_unit(NATIVE, f"""[Unit]
Description=Tervon M6 native Decapsulator
After=network.target

[Service]
ExecStart={APP}/tervon-bridge -config {conf}
User=65532
Group=65532
CapabilityBoundingSet={capability}
AmbientCapabilities={capability}
Restart=on-failure
RestartSec=2s
NoNewPrivileges=yes
""")
    else:
        dockerfile = APP / "Dockerfile"
        dockerfile.write_text("""FROM gcr.io/distroless/static-debian12:nonroot
COPY --chmod=0755 tervon-bridge /tervon-bridge
LABEL org.tervon.bridge.hot_acl="m6-v1"
USER 65532:65532
ENTRYPOINT ["/tervon-bridge","-config","/config/server.json"]
""")
        run("docker", "build", "--platform", "linux/amd64", "-t", IMAGE, APP)
        compose = APP / "compose.json"
        compose.write_text(json.dumps({"services": {"decapsulator": {
            "image": IMAGE, "container_name": CONTAINER, "pull_policy": "never",
            "restart": "unless-stopped", "user": "65532:65532", "read_only": True,
            "cap_drop": ["ALL"], "security_opt": ["no-new-privileges:true"], "pids_limit": 256,
            "ports": [f"0.0.0.0:{bridge_port}:8443/tcp"],
            "extra_hosts": ["host.docker.internal:host-gateway"],
            "volumes": [f"{ROOT}/config:/config:ro"],
        }}}, indent=2) + "\n")
        docker = shutil.which("docker")
        dependency = COMPOSE
        target = f"--container-name={CONTAINER} --expected-image={IMAGE} --expected-label-key=org.tervon.bridge.hot_acl --expected-label-value=m6-v1"
        write_unit(COMPOSE, f"""[Unit]
Description=Tervon M6 Docker Decapsulator
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart={docker} compose -p tervon-m6 -f {compose} up -d --no-build
ExecStop={docker} compose -p tervon-m6 -f {compose} stop
""")

    write_unit(REGISTRAR, f"""[Unit]
Description=Tervon M6 Registrar
Requires={dependency}
Wants=network-online.target
After={dependency} {owner} network-online.target

[Service]
ExecStart={APP}/m6-auto-registrar --deployment={mode} --root={ROOT} --listen=127.0.0.1:{registration_port} {target} --panel-config={private}/panel.json --xray-probe={APP}/live_xray_evidence.py --xray-owner-unit={owner} --xray-executable={executable} --xray-backend-alias={backend} --route-sync-interval=30s
Restart=on-failure
RestartSec=3s
""")
    run("systemd-analyze", "verify", UNITS / dependency, UNITS / REGISTRAR)
    run("systemctl", "daemon-reload")
    run("systemctl", "enable", "--now", dependency, REGISTRAR)

    if vhost.read_text() != original:
        raise ValueError("nginx vhost изменился во время установки")
    (private / "nginx-vhost.before.conf").write_text(original)
    atomic_write(vhost, candidate)
    try:
        run("nginx", "-t")
        run("systemctl", "reload", "nginx.service")
    except subprocess.CalledProcessError:
        atomic_write(vhost, original)
        run("nginx", "-t")
        run("systemctl", "reload", "nginx.service")
        raise
    if not verify_discovery(url, bridge, bridge_port, (private / "server-ca.crt").read_text()):
        print("M6_INSTALL=UNVERIFIED: внешний Discovery не подтвердил действующие маршруты", file=sys.stderr)
        print(f"Проверьте доступ извне к {url} и журнал {REGISTRAR}.", file=sys.stderr)
        return 1
    print("M6_INSTALL=PASS: внешний HTTPS Discovery подтвердил доступный Bridge")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (EOFError, KeyboardInterrupt, OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"M6_INSTALL=FAILED: {error}", file=sys.stderr)
        sys.exit(1)
