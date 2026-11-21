package com.tervonvpndpi.app

/**
 * Routing decision only. Never changes VPN configuration or starts Bridge.
 * The caller must establish the authorized mTLS Bridge, check local readiness,
 * and identify the selected destination of any TCP-capable VPN outbound.
 */
internal object BridgeVlessRoutePlan {
    // Bridge is protocol-blind. Only the Xray stream's network stack matters:
    // TCP-capable transports may carry arbitrary payloads; UDP/QUIC cannot.
    fun isTcpTransport(network: String?): Boolean = when (
        network?.lowercase(java.util.Locale.ROOT)
    ) {
        null, "", "tcp", "raw", "ws", "grpc", "httpupgrade", "http",
        "h2", "xhttp", "splithttp" -> true
        else -> false
    }

    // Compatibility for the already integrated M6 callers/tests.
    fun isSupportedTransport(network: String?): Boolean = isTcpTransport(network)

    fun transportKey(network: String?): String =
        if (network.equals("xhttp", true) || network.equals("splithttp", true))
            "xhttp" else "tcp"

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
        transport: String,
        activeServer: Endpoint?,
        subscriptionVerified: Boolean,
        localPort: Int,
        bridgeReady: Boolean
    ): Decision {
        if (!enabled) return Decision.Direct
        if (!isTcpTransport(transport)) {
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
