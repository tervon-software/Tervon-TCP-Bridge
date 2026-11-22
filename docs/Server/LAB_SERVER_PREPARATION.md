# Laboratory server preparation (no production changes)

Two separate disposable Linux VPS instances are sufficient for initial route testing. The test deployment must not share privileges, firewall chains, services or configuration with production Guard, Tervon Federation, 3x-ui or VPN services.

## Record without publishing sensitive information

- Linux distribution, kernel and CPU architecture.
- CPU count, RAM, network interface MTU and Go version.
- Relative placement of the two test hosts (source-side and destination-side); do not commit live IP addresses, credentials or identifying network inventory.
- Available TCP ports for a temporary HTTPS listener and for an allowlisted backend.
- Whether both hosts can establish normal TLS connections and resolve test DNS names.
- Whether a DNS name and a publicly trusted TLS certificate can be provisioned for the test endpoint.

## Security baseline

- Keep SSH access with public-key authentication and a recovery path.
- Use a dedicated unprivileged service account for the bridge; never run the experimental process as root.
- Do not expose the backend TCP listener publicly; bind it to loopback or a private test interface.
- Keep OS firewall default-deny inbound except explicitly required SSH and test HTTPS.
- Do not disable existing monitoring, firewall or security services.
- Do not publish private keys, token files, subscriber links, packet payloads or production configurations.

## Before any live test

1. Pass CI, identity-to-service authorization and connection lifecycle tests.
2. Review actual test commands and rollback steps.
3. Measure baseline direct TCP and ordinary HTTPS under identical conditions.
4. Start with an isolated echo backend and synthetic traffic, not a production VPN.
5. Capture only aggregate performance and failure counters unless traffic capture is explicitly authorized.

No server changes are required at this stage.
