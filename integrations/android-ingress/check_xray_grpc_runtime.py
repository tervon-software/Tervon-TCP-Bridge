#!/usr/bin/env python3
"""Read-only Xray gRPC runtime projection. Never print raw gRPC or credentials."""
import base64
import binascii
import json
import os
import re
import subprocess
import sys
from pathlib import Path

KNOWN_PID = 1253673  # Previously verified Homeserver process, identity guarded.
TCP = {"tcp","raw","ws","grpc","httpupgrade","http","h2","xhttp","splithttp"}

def wire_fields(data):
    if len(data) > 1048576:
        raise ValueError("oversize protobuf")
    i = 0
    out = {}
    def varint():
        nonlocal i
        n = 0
        for shift in range(0,70,7):
            if i >= len(data):
                raise ValueError("truncated protobuf")
            b = data[i]
            i += 1
            n |= (b & 127) << shift
            if b < 128:
                return n
        raise ValueError("invalid varint")
    while i < len(data):
        key = varint()
        field, kind = key >> 3, key & 7
        if not field:
            raise ValueError("zero field")
        if kind == 0:
            val = varint()
        elif kind == 2:
            length = varint()
            if length > len(data) - i:
                raise ValueError("truncated value")
            val = data[i:i+length]
            i += length
        elif kind in (1,5):
            length = 8 if kind == 1 else 4
            if length > len(data) - i:
                raise ValueError("truncated fixed")
            val = data[i:i+length]
            i += length
        else:
            raise ValueError("invalid wire type")
        out.setdefault(field,[]).append((kind,val))
    return out

def one(fields, number, kind, required=True):
    values = fields.get(number,[])
    if not required and not values:
        return None
    if len(values) != 1 or values[0][0] != kind:
        raise ValueError("ambiguous protobuf field")
    return values[0][1]

def typed(obj, expected=None):
    if not isinstance(obj,dict):
        raise ValueError("typed message absent")
    typ, value = obj.get("type"),obj.get("value")
    if not isinstance(typ,str) or not isinstance(value,str):
        raise ValueError("invalid typed message")
    if expected and typ != expected:
        raise ValueError("unrecognized receiver type")
    try:
        return typ,base64.b64decode(value,validate=True)
    except (ValueError,binascii.Error):
        raise ValueError("invalid base64") from None

def extract(inbounds):
    if not isinstance(inbounds,list) or not 1 <= len(inbounds) <= 4096:
        raise ValueError("invalid inbound count")
    seen=set()
    result=[]
    for item in inbounds:
        if not isinstance(item,dict):
            raise ValueError("invalid handler")
        tag=item.get("tag")
        if not isinstance(tag,str) or not tag or tag in seen:
            raise ValueError("ambiguous handler")
        seen.add(tag)
        _,receiver_bytes=typed(item.get("receiverSettings"),"xray.app.proxyman.ReceiverConfig")
        proxy_type,_=typed(item.get("proxySettings"))
        receiver=wire_fields(receiver_bytes)
        port_list=wire_fields(one(receiver,1,2))
        ranges=port_list.get(1,[])
        if len(ranges)!=1 or ranges[0][0]!=2:
            raise ValueError("port range missing or ambiguous")
        port_range=wire_fields(ranges[0][1])
        port=one(port_range,1,0)
        port_to=one(port_range,2,0,False)
        if port_to is None: port_to=port
        if not 1<=port<=65535 or port_to!=port:
            raise ValueError("port must be singleton")
        stream_data=one(receiver,3,2,False)
        network="tcp"
        if stream_data is not None:
            stream=wire_fields(stream_data)
            encoded=one(stream,5,2,False)
            if encoded:
                network=encoded.decode("ascii").lower()
        m=re.fullmatch(r"xray\.proxy\.([a-z0-9_]+)\.inbound\.Config",proxy_type)
        protocol=m.group(1) if m else "unknown"
        result.append((tag,port,protocol,network))
    return result

def blocked(reason):
    print("XRAY_RUNTIME_AUDIT=BLOCKED")
    print("REASON="+reason)
    return 2

def main():
    try:
        proc=Path(f"/proc/{KNOWN_PID}")
        if Path(os.readlink(proc/"exe")).name!="xray-linux-amd64":
            return blocked("PROCESS_CHANGED")
        args=[x.decode("utf-8") for x in (proc/"cmdline").read_bytes().split(b"\0") if x]
        paths=[]
        for i,arg in enumerate(args):
            if arg in ("-config","--config","-c"):
                if i+1>=len(args):return blocked("CONFIG_ARGUMENT_MISSING")
                paths.append(args[i+1])
            elif arg.startswith(("-config=","--config=")):
                paths.append(arg.split("=",1)[1])
        if len(paths)!=1:return blocked("CONFIG_PATH_AMBIGUOUS")
        config_path=Path(paths[0])
        if not config_path.is_absolute():
            config_path=Path(os.readlink(proc/"cwd"))/config_path
        config=json.loads(config_path.read_text(encoding="utf-8"))
        selected=[v for v in config.get("inbounds",[]) if isinstance(v,dict) and v.get("tag")=="api"]
        if len(selected)!=1 or selected[0].get("listen") not in ("127.0.0.1","::1"):
            return blocked("API_NOT_LOOPBACK_OR_AMBIGUOUS")
        api_port=selected[0].get("port")
        if type(api_port) is not int or not 1<=api_port<=65535:
            return blocked("API_PORT_INVALID")
        host=selected[0]["listen"]
        endpoint=f"[{host}]:{api_port}" if ":" in host else f"{host}:{api_port}"
        result=subprocess.run([str(proc/"exe"),"api","lsi","--server="+endpoint,
                               "--isOnlyTags=false","--timeout=3"],
                              capture_output=True,timeout=8,check=False)
        if result.returncode or len(result.stdout)>8*1024*1024:
            return blocked("GRPC_FULL_QUERY_FAILED")
        reply=json.loads(result.stdout)
        rows=extract(reply.get("inbounds"))
        reference={v.get("tag"):v for v in config.get("inbounds",[]) if isinstance(v,dict)}
        compared=0
        missing=0
        unsupported=0
        internal=0
        print("XRAY_GRPC_TOTAL="+str(len(rows)))
        for tag,port,protocol,network in sorted(rows,key=lambda r:r[1]):
            if tag=="api":
                internal+=1
                continue
            if protocol=="unknown" or network not in TCP:
                unsupported+=1
                continue
            desired=reference.get(tag)
            state="NOT_IN_CONFIG"
            if desired is not None:
                compared+=1
                cfgnet=(desired.get("streamSettings") or {}).get("network") or "tcp"
                cfgport=desired.get("port")
                cfgproto=desired.get("protocol")
                normalize=lambda x:"xhttp" if x=="splithttp" else "tcp" if x=="raw" else x
                good=(type(cfgport) is int and cfgport==port and
                      isinstance(cfgproto,str) and cfgproto.lower()==protocol and
                      isinstance(cfgnet,str) and normalize(cfgnet.lower())==normalize(network))
                state="MATCH" if good else "DIFF"
                missing+=not good
            print(f"XRAY_INBOUND port={port} protocol={protocol} network={network} reference={state}")
        print("XRAY_INTERNAL_API_SKIPPED="+str(internal))
        print("XRAY_UNSUPPORTED_SKIPPED="+str(unsupported))
        print("XRAY_COMPARED_COUNT="+str(compared))
        print("XRAY_REFERENCE_MISMATCH_COUNT="+str(missing))
        print("XRAY_PANEL_API_SELECTION=NOT_TESTED")
        print("XRAY_RUNTIME_DATA_SOURCE=LIVE_GRPC")
        print("XRAY_REFERENCE_DATA_SOURCE=CONFIG_SNAPSHOT_ONLY")
        print("SECRETS=NOT_PRINTED")
        print("RUNNING_SERVICES=UNCHANGED")
        if missing or unsupported or compared==0:
            print("XRAY_RUNTIME_AUDIT=FAIL_INCOMPLETE_OR_DRIFT")
            return 1
        print("XRAY_RUNTIME_AUDIT=PASS_GRPC_PROJECTION")
        return 0
    except (OSError,ValueError,TypeError,KeyError,UnicodeError,
            json.JSONDecodeError,subprocess.TimeoutExpired):
        return blocked("RUNTIME_SCHEMA_OR_SOURCE_UNAVAILABLE")

if __name__=="__main__":
    sys.exit(main())
