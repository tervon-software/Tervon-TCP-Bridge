package com.tervonvpndpi.app

import org.json.JSONObject

/** Source-only focused Android/JVM test. Not copied into the main APK. */
internal object TrojanOutboundParserTest {
    @JvmStatic fun main(args: Array<String>) {
        val pass = "p%40ss%2Bword"
        val base = "trojan://$pass@vpn.example:41773"
        fun outbound(query: String): JSONObject =
            requireNotNull(TrojanOutboundParser.create(base + query))
        val plain = outbound("?security=tls&type=tcp&sni=sni.example&fp=chrome&alpn=h2%2Chttp%2F1.1")
        check(plain.getString("protocol") == "trojan")
        val server = plain.getJSONObject("settings").getJSONArray("servers").getJSONObject(0)
        check(server.getString("address") == "vpn.example")
        check(server.getInt("port") == 41773)
        check(server.getString("password") == "p@ss+word")
        val tls = plain.getJSONObject("streamSettings").getJSONObject("tlsSettings")
        check(tls.getString("serverName") == "sni.example")
        check(tls.getJSONArray("alpn").length() == 2)
        check(!tls.getBoolean("allowInsecure"))
        check(plain.getJSONObject("streamSettings").getString("network") == "tcp")
        check(outbound("?type=raw").getJSONObject("streamSettings").getString("network") == "tcp")
        check(outbound("?type=ws&path=%2Ftun&host=front.example")
            .getJSONObject("streamSettings").getJSONObject("wsSettings")
            .getJSONObject("headers").getString("Host") == "front.example")
        check(outbound("?type=grpc&serviceName=svc")
            .getJSONObject("streamSettings").getJSONObject("grpcSettings")
            .getString("serviceName") == "svc")
        check(TrojanOutboundParser.transport(base + "?type=grpc") == "grpc")
        check(outbound("?type=grpc").getJSONObject("streamSettings")
            .getJSONObject("grpcSettings").getString("serviceName") == "")
        check(outbound("?type=xhttp&path=%2Fstream&mode=stream-one")
            .getJSONObject("streamSettings").getJSONObject("xhttpSettings")
            .getString("mode") == "stream-one")
        check(TrojanOutboundParser.transport(base + "?type=splithttp") == "xhttp")
        check(TrojanOutboundParser.create(base + "?type=quic") == null)
        check(TrojanOutboundParser.create(base + "?alpn=h3") == null)
        check(TrojanOutboundParser.create(base + "?security=none") == null)
        check(TrojanOutboundParser.create("trojan://@vpn.example:443") == null)
        check(TrojanOutboundParser.create("trojan://pwd@vpn.example:0") == null)
        check(TrojanOutboundParser.create("trojan://pwd@vpn.example") == null)
        println("M6_TROJAN_OUTBOUND_PARSER=PASS")
    }
}
