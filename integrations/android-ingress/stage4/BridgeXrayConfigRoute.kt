package com.tervonvpndpi.app

import org.json.JSONObject

/**
 * Applies an ALREADY AUTHORIZED, READY, locally listening Bridge route to one
 * generated Xray VLESS or Trojan TCP-based outbound. It never starts Bridge.
 * Keeps VLESS UUID/Trojan password, flow, transport mode and routing. Preserves HTTP Host
 * and TLS SNI explicitly when Xray would derive them from the old destination.
 */
internal object BridgeXrayConfigRoute {
    fun apply(configJson: String, route: BridgeVlessRoutePlan.Route): String {
        val root = JSONObject(configJson)
        val outbounds = root.getJSONArray("outbounds")
        val matching = (0 until outbounds.length()).map { outbounds.getJSONObject(it) }
            .filter { it.optString("tag") == "proxy" }
        require(matching.size == 1) { "Exactly one proxy outbound required" }
        val proxy = matching.single()
        val protocol = proxy.optString("protocol").lowercase(java.util.Locale.ROOT)
        require(protocol == "vless" || protocol == "trojan") { "Only VLESS/Trojan supported" }
        val stream = proxy.optJSONObject("streamSettings") ?: JSONObject().also {
            proxy.put("streamSettings", it)
        }
        val network = stream.optString("network")
        require(BridgeVlessRoutePlan.isSupportedTransport(network)) { "Only TCP-based VLESS supported" }
        val serverArray = proxy.getJSONObject("settings").getJSONArray(
            if (protocol == "trojan") "servers" else "vnext"
        )
        require(serverArray.length() == 1) { "Exactly one destination required" }
        val destination = serverArray.getJSONObject(0)
        require(destination.getString("address").trimEnd('.').equals(
            route.remote.host.trimEnd('.'), ignoreCase = true
        ) && destination.getInt("port") == route.remote.port) {
            "Current VPN destination does not match the authorized Bridge target"
        }
        require(route.local.host == "127.0.0.1" && route.local.port in 1024..65535) {
            "Bridge route must use verified local IPv4 listener"
        }
        val originalHost = destination.getString("address")
        val xhttp = BridgeVlessRoutePlan.transportKey(network) == "xhttp"
        if (xhttp) {
            // H3/QUIC is not TCP, even when XHTTP itself is the selected network.
            val tlsAlpn = stream.optJSONObject("tlsSettings")?.optJSONArray("alpn")
            require(tlsAlpn == null || (0 until tlsAlpn.length()).none {
                tlsAlpn.optString(it).equals("h3", ignoreCase = true)
            }) { "XHTTP over HTTP/3 is outside TCP scope" }
            val settings = stream.optJSONObject("xhttpSettings") ?: JSONObject().also {
                stream.put("xhttpSettings", it)
            }
            val separate = settings.optJSONObject("downloadSettings")
            if (separate != null) {
                // Only a shared backend is mapped by this single local Bridge
                // listener. Independent download destinations need an explicit
                // second authorized server route, never silently bypass Bridge.
                val downAddress = separate.optString("address", originalHost)
                val downPort = separate.optInt("port", route.remote.port)
                require(downAddress.equals(originalHost, ignoreCase = true) &&
                        downPort == route.remote.port) {
                    "independent XHTTP download backend is not authorized"
                }
                val downAlpn = separate.optJSONObject("tlsSettings")?.optJSONArray("alpn")
                require(downAlpn == null || (0 until downAlpn.length()).none {
                    downAlpn.optString(it).equals("h3", ignoreCase = true)
                }) { "XHTTP download over HTTP/3 is outside TCP scope" }
                if (separate.optString("security") == "tls") {
                    val downTls = separate.optJSONObject("tlsSettings") ?: JSONObject().also {
                        separate.put("tlsSettings", it)
                    }
                    if (downTls.optString("serverName").isEmpty() &&
                        originalHost.matches(Regex("[A-Za-z0-9][A-Za-z0-9.-]{0,252}")) &&
                        !originalHost.matches(Regex("[0-9.]+"))) {
                        downTls.put("serverName", originalHost)
                    }
                }
                if (separate.has("address")) separate.put("address", route.local.host)
                if (separate.has("port")) separate.put("port", route.local.port)
            }
            // XHTTP Host is derived from the destination when not explicitly
            // configured. Preserve that effective host before redirecting.
            if (settings.optString("host").isEmpty()) settings.put("host", originalHost)
        }
        // Xray may derive its TLS SNI from the original destination when
        // serverName was omitted. Retain that identity for the local dial.
        if (stream.optString("security") == "tls") {
            val tls = stream.optJSONObject("tlsSettings") ?: JSONObject().also {
                stream.put("tlsSettings", it)
            }
            if (tls.optString("serverName").isEmpty() &&
                originalHost.matches(Regex("[A-Za-z0-9][A-Za-z0-9.-]{0,252}")) &&
                !originalHost.matches(Regex("[0-9.]+"))) {
                tls.put("serverName", originalHost)
            }
        }
        destination.put("address", route.local.host)
        destination.put("port", route.local.port)
        return root.toString()
    }
}
