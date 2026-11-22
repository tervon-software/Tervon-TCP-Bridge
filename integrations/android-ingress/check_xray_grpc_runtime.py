#!/usr/bin/env python3
"""Read-only Xray gRPC runtime projection. Never print raw gRPC or credentials."""
import json
import os
import re
import subprocess
import sys
from pathlib import Path

KNOWN_PID = 1253673  # Previously verified Homeserver process, identity guarded.
TCP = {"tcp","raw","ws","grpc","httpupgrade","http","h2","xhttp","splithttp"}

def extract(inbounds):
    """Parse the *actual* xray api lsi custom MarshalToJson output.

    Xray's common/reflect/marshal.go expands TypedMessage into a decoded
    object with _TypedMessage_ and renders a singleton PortList as an int.
    It does NOT emit protobuf type/value base64 as protojson would.
    """
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
        if tag=="api":
            result.append((tag,0,"internal","internal"))
            continue
        recv=item.get("receiverSettings")
        if not isinstance(recv,dict) or recv.get("_TypedMessage_")!="xray.app.proxyman.ReceiverConfig":
            raise ValueError("receiver typed schema mismatch")
        port=recv.get("portList")
        if type(port) is not int or not 1<=port<=65535:
            raise ValueError("port is missing, ranged or invalid")
        stream=recv.get("streamSettings")
        if stream is None:
            network="tcp"
        elif isinstance(stream,dict):
            network=stream.get("protocolName") or "tcp"
            if not isinstance(network,str):
                raise ValueError("invalid network type")
            network=network.lower()
        else:
            raise ValueError("invalid stream settings")
        proxy=item.get("proxySettings")
        if not isinstance(proxy,dict):
            raise ValueError("proxy typed schema missing")
        proxy_type=proxy.get("_TypedMessage_")
        if not isinstance(proxy_type,str):
            raise ValueError("proxy type missing")
        m=re.fullmatch(r"xray\.proxy\.([a-z0-9_]+)\.inbound\.Config",proxy_type)
        protocol=m.group(1) if m else "unknown"
        result.append((tag,port,protocol,network))
    return result

def blocked(reason):
    print("XRAY_RUNTIME_AUDIT=BLOCKED")
    print("REASON="+reason)
    return 2

def main():
    stage="PROCESS_SOURCE"
    try:
        proc=Path(f"/proc/{KNOWN_PID}")
        if Path(os.readlink(proc/"exe")).name!="xray-linux-amd64":
            return blocked("PROCESS_CHANGED")
        stage="CONFIG_SOURCE"
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
        stage="CONFIG_READ"
        config=json.loads(config_path.read_text(encoding="utf-8"))
        selected=[v for v in config.get("inbounds",[]) if isinstance(v,dict) and v.get("tag")=="api"]
        if len(selected)!=1 or selected[0].get("listen") not in ("127.0.0.1","::1"):
            return blocked("API_NOT_LOOPBACK_OR_AMBIGUOUS")
        api_port=selected[0].get("port")
        if type(api_port) is not int or not 1<=api_port<=65535:
            return blocked("API_PORT_INVALID")
        host=selected[0]["listen"]
        endpoint=f"[{host}]:{api_port}" if ":" in host else f"{host}:{api_port}"
        stage="GRPC_QUERY"
        result=subprocess.run([str(proc/"exe"),"api","lsi","--server="+endpoint,
                               "--isOnlyTags=false","--timeout=3"],
                              capture_output=True,timeout=8,check=False)
        if result.returncode or len(result.stdout)>8*1024*1024:
            return blocked("GRPC_FULL_QUERY_FAILED")
        stage="GRPC_RESPONSE_JSON"
        reply=json.loads(result.stdout)
        stage="GRPC_PROTOBUF_PROJECTION"
        rows=extract(reply.get("inbounds"))
        stage="REFERENCE_COMPARISON"
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
        return blocked("RUNTIME_"+stage+"_UNAVAILABLE")

if __name__=="__main__":
    sys.exit(main())
