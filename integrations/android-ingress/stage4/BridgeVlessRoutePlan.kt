package com.tervonvpndpi.app

/**
 * Routing decision only. Never changes VPN configuration or starts Bridge.
 * The caller must establish the authorized mTLS Bridge, check local readiness,
 * and identify the selected VLESS/Trojan TCP or TCP-based XHTTP destination.
 */
internal object BridgeVlessRoutePlan {
    // Xray calls its normal TCP stream "tcp" or "raw"; omitted network also defaults to TCP.
    fun isTcpTransport(network: String?): Boolean = when (network?.lowercase(java.util.Locale.ROOT)) {
        null, "", "tcp", "raw" -> true
        else -> false
    }

    // XHTTP uses TCP when Xray establishes HTTP/1.1 or HTTP/2; H3/QUIC is
    // intentionally outside this integration. The caller must reject H3.
    fun isSupportedTransport(network: String?): Boolean =
        isTcpTransport(network) || network.equals("xhttp", ignoreCase = true) ||
            network.equals("splithttp", ignoreCase = true)

    fun transportKey(network: String?): String =
        if (network.equals("xhttp", true) || network.equals("splithttp", true)) "xhttp" else "tcp"

    data class Endpoint(val host: String, val port: Int)
    data class Route(val remote: Endpoint, val local: Endpoint)

    sealed class Decision {
        object Direct : Decision()
        data class ThroughBridge(val route: Route) : Decision()
        data class Blocked(val reason: Reason) : Decision()
    }

    enum class Reason {
        INVALID_PROFILE, UNSUPPORTED_TRANSPORT, BRIDGE_NOT_READY,
        INVALID_LOCAL_LISTENER, SUBSCRIPTION_NOT_VERIFIED
    }

    fun decide(
        enabled: Boolean,
        protocol: String,
        transport: String,
        activeServer: Endpoint?,
        subscriptionVerified: Boolean,
        localPort: Int,
        bridgeReady: Boolean
    ): Decision {
        if (!enabled) return Decision.Direct
        if (protocol.lowercase(java.util.Locale.ROOT) !in setOf("vless", "trojan") ||
            !isSupportedTransport(transport)) {
            return Decision.Blocked(Reason.UNSUPPORTED_TRANSPORT)
        }
        if (activeServer == null || !validRemote(activeServer)) {
            return Decision.Blocked(Reason.INVALID_PROFILE)
        }
        if (!subscriptionVerified) return Decision.Blocked(Reason.SUBSCRIPTION_NOT_VERIFIED)
        if (localPort !in 1024..65535) {
            return Decision.Blocked(Reason.INVALID_LOCAL_LISTENER)
        }
        if (!bridgeReady) return Decision.Blocked(Reason.BRIDGE_NOT_READY)
        return Decision.ThroughBridge(Route(activeServer, Endpoint("127.0.0.1", localPort)))
    }

    private fun validRemote(endpoint: Endpoint): Boolean {
        val host = endpoint.host.trimEnd('.')
        return host.isNotBlank() && host.length <= 253 &&
            host.none { it.isWhitespace() || it in "/\\@?#" } &&
            endpoint.port in 1..65535
    }
}
