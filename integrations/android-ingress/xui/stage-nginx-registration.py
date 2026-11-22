#!/usr/bin/env python3
"""M6: create a candidate nginx vhost; NEVER modify or reload live nginx.

The existing 302 on 'server' scope must be moved into 'location /'.
Only the exact /.well-known/tervon-bridge endpoint is proxied to the
loopback-only approved registrar. No VPN stream proxy is added.
"""
import argparse
from pathlib import Path
import re
import sys

DEFAULT = Path("/etc/nginx/sites-enabled/vp-redirect")
MARKER = "# M6 ANDROID BRIDGE REGISTRATION (isolated candidate only)"
REGISTRATION = "/.well-known/tervon-bridge"


def stage(text: str, port: int) -> str:
    if not (1024 <= port <= 65535):
        raise ValueError("invalid loopback registrar port")
    if MARKER in text or REGISTRATION in text:
        raise ValueError("registration block already present")
    if re.search(r"(?m)^\s*location\b", text):
        raise ValueError("existing location detected; manual review required")
    if len(re.findall(r"(?m)^\s*server_name\s+vp\.xrayvless\.online\s*;", text)) != 1:
        raise ValueError("expected unique vp.xrayvless.online server_name")
    if len(re.findall(r"(?m)^\s*listen\s+443\s+ssl\s*;", text)) != 1:
        raise ValueError("expected 443 ssl listen")
    if len(re.findall(r"(?m)^\s*server\s*\{", text)) != 1:
        raise ValueError("expected one server block")

    lines = text.splitlines(keepends=True)
    depth = 0
    matches = []
    for i, line in enumerate(lines):
        code = line.split("#", 1)[0]
        if re.search(r"(?<!\w)return\s+302\s+", code):
            if depth != 1 or not re.fullmatch(
                r"\s*return\s+302\s+\S.+?;\s*", code
            ):
                raise ValueError("302 not a single server-level return statement")
            matches.append(i)
        depth += code.count("{") - code.count("}")
        if depth < 0:
            raise ValueError("invalid braces")
    if depth != 0 or len(matches) != 1:
        raise ValueError("expected one top-level 302 and balanced braces")
    i = matches[0]
    original = lines[i].rstrip("\r\n")
    original_indent = original[:len(original) - len(original.lstrip())]
    linebreak = "\r\n" if lines[i].endswith("\r\n") else "\n"
    # Preserve original redirect expression exactly, including $variables.
    redirect = original.strip()
    replacement = [
        original_indent + MARKER + linebreak,
        original_indent + "location = " + REGISTRATION + " {" + linebreak,
        original_indent + "    proxy_pass http://127.0.0.1:" + str(port) + ";" + linebreak,
        original_indent + "    proxy_http_version 1.1;" + linebreak,
        original_indent + "    proxy_set_header Host $host;" + linebreak,
        original_indent + "    proxy_set_header X-Forwarded-Proto https;" + linebreak,
        original_indent + "    proxy_read_timeout 30s;" + linebreak,
        original_indent + "    client_max_body_size 16k;" + linebreak,
        original_indent + "}" + linebreak,
        original_indent + "location / {" + linebreak,
        original_indent + "    " + redirect + linebreak,
        original_indent + "}" + linebreak,
    ]
    lines[i:i+1] = replacement
    return "".join(lines)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--source", type=Path, default=DEFAULT)
    p.add_argument("--output", type=Path)
    p.add_argument("--port", type=int, default=18763)
    p.add_argument("--check", action="store_true")
    a = p.parse_args()
    if a.check == bool(a.output):
        p.error("specify exactly one of --check or --output")
    original = a.source.read_text()
    candidate = stage(original, a.port)
    if a.check:
        print("M6_NGINX_REDIRECT_SCOPE=MATCH")
        print("M6_NGINX_LIVE_CONFIG=UNCHANGED")
        return
    output = a.output.resolve()
    source = a.source.resolve()
    if output == source or output.parent == source.parent:
        raise ValueError("output must be in a separate staging directory")
    if output.exists():
        raise FileExistsError("candidate exists; do not overwrite")
    if not output.parent.is_dir():
        raise ValueError("staging directory missing")
    output.write_text(candidate)
    output.chmod(0o600)
    print("M6_NGINX_STAGED=PASS")
    print("M6_NGINX_LIVE_CONFIG=UNCHANGED")


if __name__ == "__main__":
    try:
        main()
    except Exception as err:
        print("M6_NGINX_STAGING=FAIL: " + str(err), file=sys.stderr)
        sys.exit(1)
