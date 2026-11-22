#!/usr/bin/env python3
import base64
import unittest
from check_xray_grpc_runtime import wire_fields, extract

def v(n):
    out=bytearray()
    while n>127:
        out.append(128|(n&127))
        n >>= 7
    out.append(n)
    return bytes(out)

def field(n,x):return v(n<<3)+v(x)
def msg(n,data):return v((n<<3)|2)+v(len(data))+data

def handler(port,protocol="vless",network="splithttp"):
    ports=msg(1,field(1,port)+field(2,port))
    recv=msg(1,ports)+msg(3,msg(5,network.encode()))
    enc=lambda t,data:{"type":t,"value":base64.b64encode(data).decode()}
    return {"tag":"test-1",
      "receiverSettings":enc("xray.app.proxyman.ReceiverConfig",recv),
      "proxySettings":enc("xray.proxy."+protocol+".inbound.Config",b"")}

class TestRuntimeGRPCProjection(unittest.TestCase):
    def test_extract(self):
        self.assertEqual(extract([handler(55535)]),[("test-1",55535,"vless","splithttp")])
    def test_reject_range(self):
        bad=handler(55535)
        bad["receiverSettings"]["value"]=base64.b64encode(msg(1,msg(1,field(1,1000)+field(2,2000)))).decode()
        with self.assertRaises(ValueError):extract([bad])
    def test_reject_duplicate_tag(self):
        with self.assertRaises(ValueError):extract([handler(55535),handler(30000)])
    def test_reject_invalid_wire_and_missing_fields(self):
        with self.assertRaises(ValueError):wire_fields(b"\x0a\x7f")
        with self.assertRaises(ValueError):extract([{"tag":"test-1"}])

if __name__=="__main__":
    unittest.main()
