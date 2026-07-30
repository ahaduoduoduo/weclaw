# Repository structure

Updated: 2026-07-30

- `agent/`
  - ACP, CLI, OpenAI-compatible HTTP, and native HTTP service clients.
  - `agent.go` defines both the historical text Agent contract and the
    channel-neutral native message types.
  - `native_service.go` delivers complete normalized events to arbitrary HTTP
    services.
- `api/`
  - Legacy proactive send endpoint, authenticated native `/v1/messages`
    endpoint, dynamic account selection, current context-token enforcement,
    management API, embedded web assets, administrator sessions, and health
    response.
- `cmd/`
  - CLI lifecycle, login, service construction, and outbound policy setup.
- `config/`
  - JSON configuration, environment overrides, aliases, native service
    allowlists, and separate inbound/outbound credentials.
  - `runtime.go` provides synchronized Agent reads and atomic configuration
    updates while the service is running.
  - Saved configuration uses mode `0640` so an explicitly assigned container
    service group can read a shared volume without making the file public.
- `ilink/`
  - WeChat iLink protocol, credentials, monitoring, and message types.
- `messaging/`
  - WeChat message parsing, routing, media encryption/decryption, structured
    native message construction, and reply delivery.
  - `context_tokens.go` shares process-local account/contact context tokens
    between inbound message handling and proactive delivery.
- `management/`
  - Browser administrator authentication, QR login sessions, dynamic account
    monitors, discovered contacts, unified Agent permissions, and per-user
    defaults.
- `web/`
  - Embedded standalone administration interface without a Node runtime.
- `docs/administration.md`
  - Account, Agent, contact permission, persistence, and security behavior.
- `docs/native-services.md`
  - Native service protocol, security model, examples, and ownership boundary.
- `Dockerfile`
  - Multi-stage static Go build and minimal runtime image.

The native protocol is intentionally domain-neutral. Connected services remain
separate repositories and images.
