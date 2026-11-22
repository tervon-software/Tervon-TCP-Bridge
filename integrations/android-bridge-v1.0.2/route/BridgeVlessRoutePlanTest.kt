package com.tervonvpndpi.app

private fun assertTrue(condition: Boolean, label: String) { if (!condition) error(label) }
private fun assertEqual(expected: Any?, actual: Any?, label: String) {
    if (expected != actual) error("$label: expected=$expected actual=$actual")
}

fun main() {
    val endpoint = BridgeVlessRoutePlan.Endpoint("selected.example", 443)
    fun choose(
        enabled: Boolean = true,
        network: String? = "tcp",
        server: BridgeVlessRoutePlan.Endpoint? = endpoint,
        verified: Boolean = true,
        localPort: Int = 31234,
        ready: Boolean = true
    ) = BridgeVlessRoutePlan.decide(enabled, network, server, verified, localPort, ready)

    for (network in listOf<String?>(null, "", "tcp", "raw", "ws", "grpc", "httpupgrade", "http", "h2", "xhttp", "splithttp", "XHTTP")) {
        val decision = choose(network = network)
        assertTrue(decision is BridgeVlessRoutePlan.Decision.ThroughBridge, "TCP capable: $network")
        val route = (decision as BridgeVlessRoutePlan.Decision.ThroughBridge).route
        assertEqual(endpoint, route.original, "Original endpoint preserved")
        assertEqual(BridgeVlessRoutePlan.Endpoint("127.0.0.1", 31234), route.loopback, "Local loopback only")
    }
    for (network in listOf("udp", "quic", "h3", "kcp", "unknown", " ")) {
        assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.UNSUPPORTED_TRANSPORT), choose(network = network), "Reject $network")
    }
    assertEqual(BridgeVlessRoutePlan.Decision.Direct, choose(enabled = false, network = "udp"), "Disabled does no work")
    for (bad in listOf("", "bad host", "a/b", "@bad", "x".repeat(254), "a\u0000b")) {
        assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.INVALID_ORIGINAL_ENDPOINT), choose(server = endpoint.copy(host = bad)), "bad hostname")
    }
    for (port in listOf(0, 65536, -1)) {
        assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.INVALID_ORIGINAL_ENDPOINT), choose(server = endpoint.copy(port = port)), "bad remote port")
    }
    assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.INVALID_ORIGINAL_ENDPOINT), choose(server = null), "missing remote")
    assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.UNVERIFIED_BRIDGE), choose(verified = false), "discovery unverified")
    for (port in listOf(0, 1023, 65536)) {
        assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.INVALID_LOCAL_LISTENER), choose(localPort = port), "bad listener")
    }
    assertEqual(BridgeVlessRoutePlan.Decision.Rejected(BridgeVlessRoutePlan.Rejection.LOCAL_LISTENER_NOT_READY), choose(ready = false), "not ready")
    assertTrue(choose(server = endpoint.copy(port = 65535), localPort = 65535) is BridgeVlessRoutePlan.Decision.ThroughBridge, "upper bounds")
    println("ANDROID_BRIDGE_V102_ROUTE_UNIT=PASS")
}
