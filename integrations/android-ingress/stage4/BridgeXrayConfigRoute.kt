package com.tervonvpndpi.app

import org.json.JSONObject

/**
 * Applies an ALREADY AUTHORIZED, READY, locally listening Bridge route to one
 * generated Xray TCP-capable outbound. It never starts Bridge.
 * Does not parse, modify, or classify VPN credentials or payload bytes.
 * Preserves effective HTTP Host and TLS SNI when redirecting the dial target.
 */
internal object BridgeXrayConfigRoute {
    // Choose by the existing Xray destination shape, not by VPN protocol name.
    // Multiple destinations require an explicit trusted route, never guessing.
    fun destination(proxy: JSONObject): JSONObject {
        val settings = proxy.getJSONObject("settings")
        val keys = listOf("vnext", "servers").filter { settings.has(it) }
        val direct = settings.has("address") && settings.has("port")
        require(keys.size + (if (direct) 1 else 0) == 1) {
            "Exactly one TCP destination shape required"
        }
        if (direct) return settings
        val entries = settings.getJSONArray(keys.single())
        require(entries.length() == 1) { "Exactly one TCP destination required" }
        return entries.getJSONObject(0)
    }

    fun apply(configJson: String, route: BridgeVlessRoutePlan.Route): String {
        val root = JSONObject(configJson)
        val outbounds = root.getJSONArray("outbounds")
        val matching = (0 until outbounds.length()).map { outbounds.getJSONObject(it) }
            .filter { it.optString("tag") == "proxy" }
        require(matching.size == 1) { "Exactly one proxy outbound required" }
        val proxy = matching.single()
        val stream = proxy.optJSONObject("streamSettings") ?: JSONObject().also {
            proxy.put("streamSettings", it)
        }
        val network = stream.optString("network")
        require(BridgeVlessRoutePlan.isTcpTransport(network)) { "Only TCP transports supported" }
        val tlsAlpn = stream.optJSONObject("tlsSettings")?.optJSONArray("alpn")
        require(tlsAlpn == null || (0 until tlsAlpn.length()).none {
            tlsAlpn.optString(it).equals("h3", ignoreCase = true)
        }) { "HTTP/3 is not a TCP transport" }
        val destination = destination(proxy)
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
        // Preserve implicit HTTP authority when the original remote hostname
        // would otherwise be replaced by 127.0.0.1 in generated Xray.
        if (network.equals("grpc", true)) {
            val grpc = stream.optJSONObject("grpcSettings") ?: JSONObject().also {
                stream.put("grpcSettings", it)
            }
            if (grpc.optString("authority").isEmpty()) grpc.put("authority", originalHost)
        }
        if (network.equals("ws", true)) {
            val ws = stream.optJSONObject("wsSettings") ?: JSONObject().also {
                stream.put("wsSettings", it)
            }
            val headers = ws.optJSONObject("headers") ?: JSONObject().also {
                ws.put("headers", it)
            }
            if (headers.optString("Host").isEmpty()) headers.put("Host", originalHost)
        }
        if (network.equals("httpupgrade", true)) {
            val upgrade = stream.optJSONObject("httpupgradeSettings") ?: JSONObject().also {
                stream.put("httpupgradeSettings", it)
            }
            if (upgrade.optString("host").isEmpty()) upgrade.put("host", originalHost)
        }
        if (network.equals("http", true) || network.equals("h2", true)) {
            val http = stream.optJSONObject("httpSettings") ?: JSONObject().also {
                stream.put("httpSettings", it)
            }
            if (http.optString("host").isEmpty() && !http.has("host")) {
                http.put("host", org.json.JSONArray().put(originalHost))
            }
        }
        destination.put("address", route.local.host)
        destination.put("port", route.local.port)
        return root.toString()
    }
}
