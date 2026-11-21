package com.tervonvpndpi.app

/** Verifies TCP capability rather than VPN protocol-name allowlists. */
internal object BridgeVlessRoutePlanTest {
    @JvmStatic fun main(args: Array<String>) {
        val remote = BridgeVlessRoutePlan.Endpoint("vpn.example", 41771)
        fun plan(
            enabled: Boolean = true,
            transport: String = "tcp",
            active: BridgeVlessRoutePlan.Endpoint? = remote,
            subscriptionVerified: Boolean = true,
            port: Int = 19001,
            ready: Boolean = true
        ) = BridgeVlessRoutePlan.decide(
            enabled, transport, active, subscriptionVerified, port, ready
        )
        check(plan(enabled = false) is BridgeVlessRoutePlan.Decision.Direct)
        for (tcp in listOf("", "tcp", "raw", "ws", "grpc",
            "httpupgrade", "http", "h2", "xhttp", "splithttp")) {
            check(plan(transport = tcp) is BridgeVlessRoutePlan.Decision.ThroughBridge) {
                "TCP transport incorrectly blocked: $tcp"
            }
        }
        for (udp in listOf("quic", "kcp", "udp", "h3")) {
            check(plan(transport = udp) ==
                BridgeVlessRoutePlan.Decision.Blocked(
                    BridgeVlessRoutePlan.Reason.UNSUPPORTED_TRANSPORT))
        }
        check(plan(active = null) ==
            BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.INVALID_PROFILE))
        check(plan(subscriptionVerified = false) ==
            BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.SUBSCRIPTION_NOT_VERIFIED))
        check(plan(port = 0) ==
            BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.INVALID_LOCAL_LISTENER))
        check(plan(ready = false) ==
            BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.BRIDGE_NOT_READY))
        val result = plan() as BridgeVlessRoutePlan.Decision.ThroughBridge
        check(result.route.remote == remote)
        check(result.route.local == BridgeVlessRoutePlan.Endpoint("127.0.0.1", 19001))
        println("M6_ANY_TCP_ROUTE_POLICY=PASS")
    }
}
