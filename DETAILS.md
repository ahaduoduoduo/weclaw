# Repository structure

Updated: 2026-07-28

- `agent/`
  - ACP, CLI, OpenAI-compatible HTTP, and native HTTP service clients.
  - `agent.go` defines both the historical text Agent contract and the
    channel-neutral native message types.
  - `native_service.go` delivers complete normalized events to arbitrary HTTP
    services.
- `api/`
  - Legacy proactive send endpoint, authenticated native `/v1/messages`
    endpoint, account selection, and health response.
- `cmd/`
  - CLI lifecycle, login, service construction, and outbound policy setup.
- `config/`
  - JSON configuration, environment overrides, aliases, native service
    allowlists, and separate inbound/outbound credentials.
- `ilink/`
  - WeChat iLink protocol, credentials, monitoring, and message types.
- `messaging/`
  - WeChat message parsing, routing, media encryption/decryption, structured
    native message construction, and reply delivery.
- `docs/native-services.md`
  - Native service protocol, security model, examples, and ownership boundary.
- `Dockerfile`
  - Multi-stage static Go build and minimal runtime image.

The native protocol is intentionally domain-neutral. Connected services remain
separate repositories and images.
