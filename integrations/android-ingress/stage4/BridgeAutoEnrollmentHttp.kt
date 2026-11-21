package com.tervonvpndpi.app

import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.net.HttpURLConnection
import java.net.URI
import java.net.URL
import javax.net.ssl.HttpsURLConnection

/**
 * Existing-subscription-domain discovery; no hardcoded server address,
 * client account, user-registration secret, or new crypto.
 *
 * HTTPS is authenticated by Android's platform trust store and hostname
 * verification. The discovery server determines whether Bridge is configured
 * and provides its own existing CA and client parameters.
 */
internal class BridgeAutoEnrollmentHttp(
    private val subscriptionUrl: String,
) :
    BridgeAutoCertificateEnrollment.SubscriptionBridgeDiscovery,
    BridgeAutoCertificateEnrollment.EnrollmentTransport {

    private val subscriptionHost: String = run {
        val uri = URI(subscriptionUrl.trim())
        require(uri.scheme.equals("https", ignoreCase = true) &&
                !uri.host.isNullOrBlank() && uri.rawUserInfo == null &&
                uri.port in -1..65535 && uri.port != 0 &&
                uri.rawFragment == null) { "current subscription must be HTTPS" }
        uri.host.lowercase(java.util.Locale.ROOT)
    }

    internal data class BridgeParameters(
        val serverId: String,
        val remoteUrl: String,
        val tlsName: String,
        val service: String,
        val tcpRoutes: Map<Int, String>,
        val clientCaPem: String,
        val serverCaPem: String,
    )

    @Volatile private var discovered: BridgeParameters? = null

    /** Only the existing authenticated HTTPS discovery may supply Bridge parameters. */
    fun discoveredParameters(): BridgeParameters? = discovered

    private val endpoint: URL = run {
        val origin = URI(subscriptionUrl.trim())
        require(origin.scheme.equals("https", ignoreCase = true) &&
                !origin.host.isNullOrBlank() && origin.rawUserInfo == null &&
                origin.port in -1..65535 && origin.port != 0 &&
                origin.rawFragment == null) {
            "current subscription must be an HTTPS URL"
        }
        // The HTTPS subscription can be served on its own port (e.g. 2096).
        // Discovery uses ONLY the subscription host and the existing HTTPS
        // default HTTPS origin. Never send the secret subscription path/query
        // to discovery or try to register against the 3x-ui subscription port.
        URL("https", origin.host, "/.well-known/tervon-bridge")
    }

    /** Stable certificate-store identity is the HTTPS subscription authority, not camouflage SNI. */
    internal fun verifiedServerId(advertisedUrl: String, advertisedName: String): String {
        val remote = URI(advertisedUrl)
        require(remote.scheme == "https" && !remote.host.isNullOrBlank() &&
                remote.rawUserInfo == null && remote.rawQuery == null &&
                remote.rawFragment == null && remote.rawPath == "/v2/stream" &&
                remote.port in -1..65535 && remote.port != 0) {
            "invalid Bridge HTTPS endpoint"
        }
        require(advertisedName.equals(remote.host, ignoreCase = true)) {
            "Bridge TLS name differs from its HTTPS endpoint"
        }
        require(subscriptionHost.matches(Regex("[a-z0-9][a-z0-9.-]{0,127}"))) {
            "invalid subscription server identity"
        }
        return subscriptionHost
    }

    override fun discover(activeSubscription: String): BridgeAutoCertificateEnrollment.TrustedBridge? {
        require(activeSubscription == subscriptionUrl) { "subscription changed during registration" }
        val (code, json) = request("GET", null)
        if (code == HttpURLConnection.HTTP_NOT_FOUND) {
            discovered = null
            return null
        }
        require(code == HttpURLConnection.HTTP_OK) { "Bridge discovery unavailable" }
        val data = JSONObject(json)
        if (!data.optBoolean("available", false)) {
            discovered = null
            return null
        }
        val remote = URI(data.getString("remote_url"))
        require(remote.scheme == "https" && remote.host != null &&
                remote.rawUserInfo == null && remote.rawQuery == null &&
                remote.rawFragment == null && remote.rawPath == "/v2/stream") {
            "invalid server Bridge parameters"
        }
        val defaultService = data.getString("service")
        require(defaultService.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,63}"))) {
            "invalid Bridge service"
        }
        val name = data.getString("server_name")
        require(name.equals(remote.host, ignoreCase = true)) {
            "Bridge TLS identity differs from advertised endpoint"
        }
        // The authenticated server publishes only configured TCP backend ports
        // and their authorized Bridge service. No VPN protocol table exists.
        val routes = linkedMapOf<Int, String>()
        val advertised = data.optJSONObject("tcp_routes")
        if (advertised != null) {
            val keys = advertised.keys()
            while (keys.hasNext()) {
                val portText = keys.next()
                val port = portText.toIntOrNull()
                require(port != null && port in 1..65535 && port.toString() == portText) {
                    "invalid Bridge backend port"
                }
                val service = advertised.getString(portText)
                require(service.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,63}"))) {
                    "invalid Bridge backend service"
                }
                require(routes.put(port, service) == null && routes.size <= 256) {
                    "duplicate or oversized TCP route map"
                }
            }
        }
        val clientCa = data.getString("client_ca")
        val serverCa = data.getString("server_ca")
        require(clientCa.contains("-----BEGIN CERTIFICATE-----") &&
                serverCa.contains("-----BEGIN CERTIFICATE-----")) {
            "incomplete Bridge certificate trust information"
        }
        val params = BridgeParameters(
            verifiedServerId(data.getString("remote_url"), name),
            data.getString("remote_url"), name, defaultService, routes.toMap(), clientCa, serverCa
        )
        discovered = params
        return BridgeAutoCertificateEnrollment.TrustedBridge(params.serverId, params.clientCaPem)
    }

    override fun issue(
        trusted: BridgeAutoCertificateEnrollment.TrustedBridge,
        csrPem: String
    ): BridgeAutoCertificateEnrollment.Issued {
        val params = discovered ?: error("Bridge discovery must succeed before issuance")
        require(trusted.serverId == params.serverId && trusted.clientCaPem == params.clientCaPem) {
            "certificate issuance differs from authenticated HTTPS discovery"
        }
        require(csrPem.length in 64..8192) { "invalid CSR" }
        val (code, response) = request("POST", csrPem.toByteArray(Charsets.US_ASCII))
        require(code == HttpURLConnection.HTTP_OK) { "Bridge enrollment unavailable" }
        val certificate = JSONObject(response).getString("certificate")
        require(certificate.length in 100..32768) { "invalid issued certificate size" }
        // Existing BridgeServerCertificateStore validates signer, clientAuth,
        // expiry and exact device public key before writing this certificate.
        return BridgeAutoCertificateEnrollment.Issued(certificate)
    }

    private fun request(method: String, body: ByteArray?): Pair<Int, String> {
        val connection = endpoint.openConnection() as HttpsURLConnection
        try {
            connection.requestMethod = method
            connection.instanceFollowRedirects = false
            connection.connectTimeout = 7000
            connection.readTimeout = 7000
            connection.setRequestProperty("Accept", "application/json")
            if (body != null) {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/pkcs10")
                connection.outputStream.use { it.write(body) }
            }
            val code = connection.responseCode
            if (code == HttpURLConnection.HTTP_NOT_FOUND) return code to ""
            // Do not read error bodies (may contain server paths or secrets).
            if (code != HttpURLConnection.HTTP_OK) return code to ""
            val text = connection.inputStream.use { input ->
                val output = ByteArrayOutputStream()
                val buf = ByteArray(4096)
                while (true) {
                    val read = input.read(buf)
                    if (read < 0) break
                    require(output.size() + read <= 65536) { "Bridge response too large" }
                    output.write(buf, 0, read)
                }
                output.toString("UTF-8")
            }
            return code to text
        } finally {
            connection.disconnect()
        }
    }
}
