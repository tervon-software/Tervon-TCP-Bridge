package com.tervonvpndpi.app

/**
 * One decision for routing an already selected Xray TCP-capable outbound through
 * a local Bridge listener. Pure policy: no sockets, no Go process, no config IO.
 *
 * A ready local listener does NOT prove remote mTLS, HTTP/2 or backend success.
 * Neither the VPN protocol nor the server's routes/ACL are interpreted here.
 */
internal object BridgeVlessRoutePlan {
    data class Endpoint(val host: String, val port: Int)
    data class Route(val original: Endpoint, val loopback: Endpoint)

    enum class Rejection {
        UNSUPPORTED_TRANSPORT,
        INVALID_ORIGINAL_ENDPOINT,
        UNVERIFIED_BRIDGE,
        INVALID_LOCAL_LISTENER,
        LOCAL_LISTENER_NOT_READY
    }

    sealed interface Decision {
        data object Direct : Decision
        data class ThroughBridge(val route: Route) : Decision
        data class Rejected(val reason: Rejection) : Decision
    }

    private val tcpNetworks = setOf(
        "tcp", "raw", "ws", "grpc", "httpupgrade", "http", "h2", "xhttp", "splithttp"
    )

    /** Xray defaults an absent network to TCP; all unrecognized networks fail closed. */
    fun isTcpCapable(network: String?): Boolean =
        network == null || network.isEmpty() ||
            network.lowercase(java.util.Locale.ROOT) in tcpNetworks

    fun decide(
        enabled: Boolean,
        xrayNetwork: String?,
        original: Endpoint?,
        discoveryVerified: Boolean,
        localListenPort: Int,
        localListenerReady: Boolean
    ): Decision {
        if (!enabled) return Decision.Direct
        if (!isTcpCapable(xrayNetwork)) return Decision.Rejected(Rejection.UNSUPPORTED_TRANSPORT)
        if (original == null || !validEndpoint(original)) {
            return Decision.Rejected(Rejection.INVALID_ORIGINAL_ENDPOINT)
        }
        if (!discoveryVerified) return Decision.Rejected(Rejection.UNVERIFIED_BRIDGE)
        if (localListenPort !in 1024..65535) {
            return Decision.Rejected(Rejection.INVALID_LOCAL_LISTENER)
        }
        if (!localListenerReady) return Decision.Rejected(Rejection.LOCAL_LISTENER_NOT_READY)
        return Decision.ThroughBridge(Route(original, Endpoint("127.0.0.1", localListenPort)))
    }

    private fun validEndpoint(endpoint: Endpoint): Boolean {
        val host = endpoint.host
        return host.isNotEmpty() && host.length <= 253 &&
            host.none { it.isWhitespace() || it.isISOControl() || it in "/\\@?#" } &&
            endpoint.port in 1..65535
    }
}
