# Roadmap

- **M0 — Design baseline:** requirements, architecture, threat model, public development process.
- **M1 — Correctness:** standalone Go encapsulator and decapsulator, full-duplex TCP tests, half-close and cancellation.
- **M2 — Security:** mTLS identity authorization, service allowlist, limits, timeouts and adversarial tests.
- **M3 — Reliability:** two test backends, failure handling, repeatable concurrency and soak tests.
- **M4 — Compatibility:** actual VLESS, Trojan, Shadowsocks and OpenVPN TCP configurations; separately assess UDP carried within TCP-based protocols.
- **M5 — Performance and route measurements:** reproducible direct-TCP vs bridge comparison with topology, versions, throughput, CPU, RAM, RTT, packet loss and connection setup times.

No milestone is considered complete without recorded test evidence. No production changes without a separate deployment decision.
