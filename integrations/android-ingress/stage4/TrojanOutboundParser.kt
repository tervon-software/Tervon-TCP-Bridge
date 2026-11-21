package com.tervonvpndpi.app

import org.json.JSONArray
import org.json.JSONObject
import java.net.URI
import java.net.URLDecoder
import java.util.Locale

/**
 * Trojan URI -> existing libv2ray/Xray outbound. No new VPN core or credentials.
 * Only TCP-based transports, TLS verification ON, no QUIC/H3 or silent fallback.
 * The subscription password is never logged by this component.
 */
internal object TrojanOutboundParser {
    fun transport(url: String): String? {
        val parsed = parseUrl(url) ?: return null
        if (!(parsed["security"] ?: "tls").equals("tls", ignoreCase = true)) return null
        if (parsed["alpn"].orEmpty().split(",").any { it.equals("h3", ignoreCase = true) }) return null
        val transport = (parsed["type"] ?: "tcp").lowercase(Locale.ROOT)
        if (transport == "grpc" && parsed["servicename"].isNullOrBlank()) return null
        return when (transport) {
            "", "tcp", "raw" -> "tcp"
            "ws", "websocket" -> "ws"
            "grpc" -> "grpc"
            "xhttp", "splithttp" -> "xhttp"
            else -> null
        }
    }

    fun parseUrl(url: String): Map<String, String>? {
        return try {
            if (!url.startsWith("trojan://", ignoreCase = true)) return null
            val uri = URI(url.trim())
            if (!uri.scheme.equals("trojan", true) || uri.rawUserInfo.isNullOrEmpty() ||
                uri.host.isNullOrBlank() || uri.port !in 1..65535) return null
            val rawPassword = uri.rawUserInfo
            val password = decode(rawPassword)
            if (password.isEmpty() || password.length > 4096) return null
            val result = mutableMapOf(
                "password" to password,
                "address" to uri.host,
                "port" to uri.port.toString(),
            )
            val query = uri.rawQuery.orEmpty()
            for (part in query.split("&")) {
                if (part.isEmpty()) continue
                val key = decode(part.substringBefore("=")).lowercase(Locale.ROOT)
                val value = decode(part.substringAfter("=", ""))
                if (key.isNotBlank() && key !in listOf("password", "address", "port")) {
                    result[key] = value
                }
            }
            result
        } catch (_: Exception) { null }
    }

    fun create(url: String): JSONObject? {
        val values = parseUrl(url) ?: return null
        val network = transport(url) ?: return null
        val security = (values["security"] ?: "tls").lowercase(Locale.ROOT)
        if (security != "tls") return null

        val address = values.getValue("address")
        val port = values.getValue("port").toInt()
        val password = values.getValue("password")
        val sni = (values["sni"] ?: address).ifBlank { address }
        val alpn = values["alpn"].orEmpty().split(",").map { it.trim() }.filter { it.isNotEmpty() }
        if (alpn.any { it.equals("h3", ignoreCase = true) }) return null
        val settings = JSONObject()
            .put("servers", JSONArray().put(JSONObject()
                .put("address", address)
                .put("port", port)
                .put("password", password)))
        val tls = JSONObject()
            .put("serverName", sni)
            .put("allowInsecure", false)
        val fingerprint = values["fp"].orEmpty()
        if (fingerprint.isNotEmpty()) tls.put("fingerprint", fingerprint)
        if (alpn.isNotEmpty()) tls.put("alpn", JSONArray(alpn))

        val stream = JSONObject()
            .put("network", network)
            .put("security", "tls")
            .put("tlsSettings", tls)
            .put("sockopt", JSONObject()
                .put("tcpKeepAliveInterval", 15)
                .put("tcpKeepAliveIdle", 30)
                .put("tcpNoDelay", true))
        val path = values["path"].orEmpty().ifBlank { "/" }
        val host = values["host"].orEmpty()
        when (network) {
            "ws" -> {
                val ws = JSONObject().put("path", path)
                if (host.isNotEmpty()) ws.put("headers", JSONObject().put("Host", host))
                stream.put("wsSettings", ws)
            }
            "grpc" -> {
                val service = values["servicename"].orEmpty()
                if (service.isEmpty()) return null
                stream.put("grpcSettings", JSONObject().put("serviceName", service))
            }
            "xhttp" -> {
                val xhttp = JSONObject().put("path", path)
                if (host.isNotEmpty()) xhttp.put("host", host)
                val mode = values["mode"].orEmpty()
                if (mode.isNotEmpty()) xhttp.put("mode", mode)
                stream.put("xhttpSettings", xhttp)
            }
        }
        return JSONObject()
            .put("tag", "proxy")
            .put("protocol", "trojan")
            .put("settings", settings)
            .put("streamSettings", stream)
    }

    private fun decode(value: String): String =
        URLDecoder.decode(value.replace("+", "%2B"), "UTF-8")
}
