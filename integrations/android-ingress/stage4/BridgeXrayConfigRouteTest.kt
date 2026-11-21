package com.tervonvpndpi.app

import org.json.JSONArray
import org.json.JSONObject

/**
 * Standalone Android/Kotlin test. Must not be installed into app/src/main.
 * Ensures Bridge changes only its local destination and preserves the original
 * XHTTP stream settings, identity, and direct config on failure.
 */
internal object BridgeXrayConfigRouteTest {
    private val route = BridgeVlessRoutePlan.Route(
        BridgeVlessRoutePlan.Endpoint("vpn.example", 30443),
        BridgeVlessRoutePlan.Endpoint("127.0.0.1", 19001),
    )

    private fun profile(network: String, settings: JSONObject): String {
        val proxy = JSONObject().put("tag", "proxy").put("protocol", "vless")
            .put("settings", JSONObject().put("vnext", JSONArray().put(
                JSONObject().put("address", "vpn.example").put("port", 30443)
                    .put("users", JSONArray().put(JSONObject()
                        .put("id", "00000000-0000-0000-0000-000000000001"))
            )))
            .put("streamSettings", JSONObject().put("network", network)
                .put("security", "tls").put("tlsSettings", JSONObject())
                .put("xhttpSettings", settings))
        return JSONObject().put("outbounds", JSONArray().put(proxy)).toString()
    }

    @JvmStatic
    fun main(args: Array<String>) {
        for (mode in listOf("auto", "packet-up", "stream-up", "stream-one")) {
            val original = profile("xhttp", JSONObject().put("path", "/x-test")
                .put("mode", mode))
            val routed = JSONObject(BridgeXrayConfigRoute.apply(original, route))
                .getJSONArray("outbounds").getJSONObject(0)
            val target = routed.getJSONObject("settings").getJSONArray("vnext").getJSONObject(0)
            check(target.getString("address") == "127.0.0.1")
            check(target.getInt("port") == 19001)
            check(target.getJSONArray("users").getJSONObject(0).getString("id")
                == "00000000-0000-0000-0000-000000000001")
            val stream = routed.getJSONObject("streamSettings")
            check(stream.getJSONObject("xhttpSettings").getString("mode") == mode)
            check(stream.getJSONObject("xhttpSettings").getString("path") == "/x-test")
            check(stream.getJSONObject("xhttpSettings").getString("host") == "vpn.example")
            check(stream.getJSONObject("tlsSettings").getString("serverName") == "vpn.example")
            check(JSONObject(original).getJSONArray("outbounds").getJSONObject(0)
                .getJSONObject("settings").getJSONArray("vnext").getJSONObject(0)
                .getString("address") == "vpn.example")
        }
        // Explicit TLS and HTTP authority must remain unchanged.
        val withHost = profile("splithttp", JSONObject().put("path", "/x")
            .put("host", "front.example"))
        val routed = JSONObject(BridgeXrayConfigRoute.apply(withHost, route))
            .getJSONArray("outbounds").getJSONObject(0).getJSONObject("streamSettings")
        check(routed.getJSONObject("xhttpSettings").getString("host") == "front.example")
        // Same-backend download channel is supported.
        val same = profile("xhttp", JSONObject().put("path", "/x")
            .put("downloadSettings", JSONObject()
                .put("address", "vpn.example").put("port", 30443)
                .put("security", "tls")))
        val updated = JSONObject(BridgeXrayConfigRoute.apply(same, route))
            .getJSONArray("outbounds").getJSONObject(0)
            .getJSONObject("streamSettings").getJSONObject("xhttpSettings")
            .getJSONObject("downloadSettings")
        check(updated.getString("address") == "127.0.0.1")
        check(updated.getInt("port") == 19001)
        check(updated.getJSONObject("tlsSettings").getString("serverName") == "vpn.example")
        // Different download endpoints and H3 must be explicitly rejected.
        val different = profile("xhttp", JSONObject().put("path", "/x")
            .put("downloadSettings", JSONObject().put("address", "other.example")
                .put("port", 30001)))
        check(runCatching { BridgeXrayConfigRoute.apply(different, route) }.isFailure)
        val h3Root = JSONObject(profile("xhttp", JSONObject().put("path", "/x")))
        h3Root.getJSONArray("outbounds").getJSONObject(0).getJSONObject("streamSettings")
            .put("tlsSettings", JSONObject().put("alpn", JSONArray().put("h3")))
        check(runCatching { BridgeXrayConfigRoute.apply(h3Root.toString(), route) }.isFailure)
        // Default missing network is plain TCP.
        val tcp = JSONObject(profile("tcp", JSONObject()))
        tcp.getJSONArray("outbounds").getJSONObject(0).getJSONObject("streamSettings")
            .remove("network")
        check(runCatching { BridgeXrayConfigRoute.apply(tcp.toString(), route) }.isSuccess)
        // Trojan uses servers instead of VLESS vnext. Password and original
        // TLS identity must survive redirect without ever entering logs.
        val trojan = JSONObject(profile("tcp", JSONObject()))
        val tProxy = trojan.getJSONArray("outbounds").getJSONObject(0)
        tProxy.put("protocol", "trojan")
        tProxy.put("settings", JSONObject().put("servers", JSONArray().put(
            JSONObject().put("address", "vpn.example").put("port", 30443)
                .put("password", "secret-test-only"))))
        val trojanRouted = JSONObject(BridgeXrayConfigRoute.apply(trojan.toString(), route))
            .getJSONArray("outbounds").getJSONObject(0)
        val tTarget = trojanRouted.getJSONObject("settings")
            .getJSONArray("servers").getJSONObject(0)
        check(tTarget.getString("password") == "secret-test-only")
        check(tTarget.getString("address") == "127.0.0.1")
        check(tTarget.getInt("port") == 19001)
        check(trojanRouted.getJSONObject("streamSettings")
            .getJSONObject("tlsSettings").getString("serverName") == "vpn.example")
        val trojanXhttp = JSONObject(trojan.toString())
        trojanXhttp.getJSONArray("outbounds").getJSONObject(0)
            .getJSONObject("streamSettings").put("network", "xhttp")
        val tx = JSONObject(BridgeXrayConfigRoute.apply(trojanXhttp.toString(), route))
            .getJSONArray("outbounds").getJSONObject(0)
        check(tx.getJSONObject("settings").getJSONArray("servers")
            .getJSONObject(0).getString("password") == "secret-test-only")
        check(tx.getJSONObject("streamSettings").getJSONObject("xhttpSettings")
            .getString("host") == "vpn.example")
        // VMess uses vnext as VLESS does, but must not be allowlisted by name.
        val vmess = JSONObject(profile("ws", JSONObject()))
        vmess.getJSONArray("outbounds").getJSONObject(0).put("protocol", "vmess")
        val vmessOut = JSONObject(BridgeXrayConfigRoute.apply(vmess.toString(), route))
            .getJSONArray("outbounds").getJSONObject(0)
        check(BridgeXrayConfigRoute.destination(vmessOut).getString("address") == "127.0.0.1")
        check(vmessOut.getJSONObject("streamSettings").getJSONObject("wsSettings")
            .getJSONObject("headers").getString("Host") == "vpn.example")

        // Arbitrary Xray TCP outbound with a servers[] shape and opaque
        // credentials is handled without recognising the VPN protocol.
        val arbitrary = JSONObject(trojan.toString())
        arbitrary.getJSONArray("outbounds").getJSONObject(0)
            .put("protocol", "future-tcp-protocol")
        val arbitraryOut = JSONObject(BridgeXrayConfigRoute.apply(arbitrary.toString(), route))
            .getJSONArray("outbounds").getJSONObject(0)
        check(BridgeXrayConfigRoute.destination(arbitraryOut).getInt("port") == 19001)
        check(BridgeXrayConfigRoute.destination(arbitraryOut).getString("password") == "secret-test-only")

        // UDP-based network and ambiguous destination shapes remain unsafe.
        val quic = JSONObject(vmess.toString())
        quic.getJSONArray("outbounds").getJSONObject(0)
            .getJSONObject("streamSettings").put("network", "quic")
        check(runCatching { BridgeXrayConfigRoute.apply(quic.toString(), route) }.isFailure)
        val ambiguous = JSONObject(vmess.toString())
        val ambiguousProxy = ambiguous.getJSONArray("outbounds").getJSONObject(0)
        ambiguousProxy.getJSONObject("settings").put("servers", JSONArray().put(
            JSONObject().put("address", "vpn.example").put("port", 30443)))
        check(runCatching { BridgeXrayConfigRoute.apply(ambiguous.toString(), route) }.isFailure)
        // Direct address/port pair also carries an opaque TCP destination.
        val directShape = JSONObject(profile("tcp", JSONObject()))
        val directProxy = directShape.getJSONArray("outbounds").getJSONObject(0)
        directProxy.put("protocol", "opaque-tcp")
        directProxy.put("settings", JSONObject().put("address", "vpn.example")
            .put("port", 30443).put("opaqueKey", "keep"))
        val directOut = JSONObject(BridgeXrayConfigRoute.apply(directShape.toString(), route))
            .getJSONArray("outbounds").getJSONObject(0)
        check(BridgeXrayConfigRoute.destination(directOut).getString("address") == "127.0.0.1")
        check(BridgeXrayConfigRoute.destination(directOut).getInt("port") == 19001)
        check(BridgeXrayConfigRoute.destination(directOut).getString("opaqueKey") == "keep")
        val grpcRoot = JSONObject(profile("grpc", JSONObject()))
        val grpcOut = JSONObject(BridgeXrayConfigRoute.apply(grpcRoot.toString(), route))
            .getJSONArray("outbounds").getJSONObject(0)
            .getJSONObject("streamSettings").getJSONObject("grpcSettings")
        check(grpcOut.getString("authority") == "vpn.example")
        println("M6_ANY_TCP_OUTBOUND_REWRITE=PASS")
        println("M6_XHTTP_TCP_ROUTE_IDENTITY=PASS")
        println("M6_TROJAN_TCP_ROUTE_IDENTITY=PASS")
    }
}
