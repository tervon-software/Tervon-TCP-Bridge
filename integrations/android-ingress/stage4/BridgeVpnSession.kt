package com.tervonvpndpi.app

import android.content.Context
import org.json.JSONObject
import java.net.InetSocketAddress
import java.net.InetAddress
import java.net.ServerSocket
import java.net.Socket

/** M6: existing Go v1.0.1 Bridge + libv2ray VLESS/Trojan TCP or XHTTP-over-TCP.
 * Never changes the persisted direct Xray config or HEV/TUN. Failures use direct.
 */
internal object BridgeVpnSession {
    private var handoff: BridgeClientPemHandoff? = null
    // Local listener/route readiness only; never a remote E2E success signal.
    private var active = false
    private var ownsProcess = false
    @Volatile private var failureCode = "NOT_ATTEMPTED"

    fun lastFailureCode(): String = failureCode

    private fun fallback(code: String, original: String): String {
        failureCode = code
        return original
    }

    @Synchronized
    fun active(): Boolean = active && BridgeRuntimeControl.coreState() == BridgeCoreManager.State.RUNNING

    /** Prepare BEFORE libv2ray startLoop/restart. Call only on the VPN worker. */
    @Synchronized
    fun prepare(context: Context, direct: String): String {
        stop(context)
        failureCode = "NOT_ATTEMPTED"
        val prefs = context.getSharedPreferences("tervon_bridge_settings_v1", Context.MODE_PRIVATE)
        if (!prefs.getBoolean("enabled", false)) return fallback("BYPASS_DISABLED", direct)
        val subscriptionHost = BridgeSubscriptionBinding.host(context) ?: return fallback("SUBSCRIPTION_NOT_SELECTED", direct)
        if (!subscriptionHost.matches(Regex("[a-z0-9][a-z0-9.-]{0,127}"))) return fallback("INVALID_SUBSCRIPTION_HOST", direct)
        try {
            val root = JSONObject(direct)
            val outbounds = root.getJSONArray("outbounds")
            val proxy = (0 until outbounds.length()).map { outbounds.getJSONObject(it) }
                .single { it.optString("tag") == "proxy" }
            val protocol = proxy.optString("protocol").lowercase(java.util.Locale.ROOT)
            if (protocol !in setOf("vless", "trojan")) return fallback("UNSUPPORTED_PROTOCOL", direct)
            val stream = proxy.optJSONObject("streamSettings")
            val network = stream?.optString("network")
            if (!BridgeVlessRoutePlan.isSupportedTransport(network)) return fallback("UNSUPPORTED_TRANSPORT", direct)
            val transport = BridgeVlessRoutePlan.transportKey(network)
            if (transport == "xhttp") {
                // Explicit H3 is UDP/QUIC and cannot enter this TCP bridge.
                val alpn = stream?.optJSONObject("tlsSettings")?.optJSONArray("alpn")
                if (alpn != null && (0 until alpn.length()).any {
                        alpn.optString(it).equals("h3", ignoreCase = true)
                    }) return fallback("XHTTP_H3_UNSUPPORTED", direct)
            }
            val destinations = proxy.getJSONObject("settings").getJSONArray(
                if (protocol == "trojan") "servers" else "vnext"
            )
            if (destinations.length() != 1) return fallback("PROFILE_CONFIG_INVALID", direct)
            val dest = destinations.getJSONObject(0)
            val selected = BridgeVlessRoutePlan.Endpoint(dest.getString("address"), dest.getInt("port"))

            // Re-read the approved subscription host on every start/profile change;
            // never use stale discovery parameters from the preceding profile.
            val origin = "https://$subscriptionHost/"
            val http = BridgeAutoEnrollmentHttp(origin)
            val trusted = http.discover(origin) ?: return fallback("DISCOVERY_NO_BRIDGE", direct)
            val params = http.discoveredParameters() ?: return fallback("DISCOVERY_NO_PARAMETERS", direct)
            if (params.serverId != trusted.serverId) return fallback("DISCOVERY_ID_MISMATCH", direct)
            val serviceKey = if (protocol == "trojan") "trojan_$transport" else transport
            val authorizedService = params.transportServices[serviceKey]
                ?: return fallback("XHTTP_ROUTE_UNAVAILABLE", direct)

            val identity = BridgeDeviceIdentity(context)
            val stored = BridgeServerCertificateStore(context).findValid(
                trusted.serverId, params.clientCaPem, identity.publicIdentity().spkiDer
            ) ?: return fallback("DEVICE_CERTIFICATE_UNAVAILABLE", direct)
            // Obtain local listener ports from the OS, not from manually
            // entered values or fixed per-device defaults.
            val port = automaticLoopbackPort()
            val metricsPort = generateSequence { automaticLoopbackPort() }
                .take(5).firstOrNull { it != port } ?: return fallback("PORT_ALLOCATION_FAILED", direct)
            handoff = BridgeClientPemHandoff.create(context, identity, stored.certificatePem, params.serverCaPem)
            val files = requireNotNull(handoff)
            val config = BridgeClientConfigBuilder.build(BridgeClientConfigBuilder.Inputs(
                remoteUrl = params.remoteUrl,
                serverName = params.tlsName,
                service = authorizedService,
                listenPort = port,
                metricsPort = metricsPort,
                clientCertificatePath = files.clientCertFile.absolutePath,
                clientPrivateKeyPath = files.privateKeyFile.absolutePath,
                serverCaPath = files.serverCaFile.absolutePath,
            ))
            val env = BridgeConfigManager(context)
            if (BridgeRuntimeControl.coreState() == BridgeCoreManager.State.RUNNING) {
                stop(context)
                return fallback("BRIDGE_ALREADY_RUNNING", direct)
            }
            failureCode = "CONFIG_WRITE_FAILED"
            env.writeValidatedConfig(config)
            BridgeRuntimeControl.onUnexpectedExit(context) { unexpectedExit(context.applicationContext) }
            failureCode = "BRIDGE_START_FAILED"
            check(BridgeRuntimeControl.launchClient(context)) { "Bridge process did not start" }
            ownsProcess = true
            var ready = false
            repeat(25) {
                if (!ready) {
                    try {
                        Socket().use { sock -> sock.connect(InetSocketAddress("127.0.0.1", port), 120) }
                        ready = true
                    } catch (_: Exception) { Thread.sleep(80) }
                }
            }
            failureCode = "BRIDGE_LISTENER_UNAVAILABLE"
            check(ready) { "Bridge TCP listener unavailable" }
            // The Go Bridge opens PEM files during startup. Remove handoff after
            // the local listener is confirmed and on every stop/error path.
            files.close()
            handoff = null
            val decision = BridgeVlessRoutePlan.decide(
                enabled = true, protocol = protocol, transport = network ?: "tcp",
                activeServer = selected,
                subscriptionVerified = params.serverId == trusted.serverId,
                localPort = port, bridgeReady = ready
            )
            failureCode = "ROUTE_REJECTED"
            val route = (decision as? BridgeVlessRoutePlan.Decision.ThroughBridge)?.route
                ?: error("VPN TCP route did not pass authorization")
            failureCode = "XRAY_ROUTE_FAILED"
            val routed = BridgeXrayConfigRoute.apply(direct, route)
            prefs.edit().putInt("metricsPort", metricsPort).apply()
            active = true
            failureCode = "LOCAL_ROUTE_READY"
            return routed
        } catch (_: Exception) {
            // Only stage codes may reach the UI; never expose private keys or URLs.
            val stage = failureCode
            stop(context)
            failureCode = if (stage == "NOT_ATTEMPTED") "PROFILE_CONFIG_INVALID" else stage
            return direct
        }
    }

    private fun automaticLoopbackPort(): Int =
        ServerSocket(0, 0, InetAddress.getByName("127.0.0.1")).use { it.localPort }

    private fun unexpectedExit(context: Context) {
        val shouldRecover = synchronized(this) {
            if (!active || !ownsProcess) false else {
                active = false
                ownsProcess = false
                true
            }
        }
        if (!shouldRecover) return
        failureCode = "BRIDGE_PROCESS_EXITED"
        BridgeRuntimeSettings.setEnabled(context, false)
        BridgeSubscriptionBinding.clear(context)
        if (VpnForegroundService.isRunning) {
            VpnForegroundService.restart(context) { ok ->
                if (!ok) VpnForegroundService.stop(context)
            }
        }
    }

    @Synchronized
    fun stop(context: Context) {
        active = false
        if (ownsProcess) {
            BridgeRuntimeControl.onUnexpectedExit(context, null)
            BridgeRuntimeControl.stopClient(context)
            ownsProcess = false
        }
        handoff?.close()
        handoff = null
        BridgeClientPemHandoff.removeAbandoned(context)
    }
}
