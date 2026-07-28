# Development status

Updated: 2026-07-28

## Completed

- [x] 2026-07-28: Add a channel-neutral native HTTP service type.
- [x] 2026-07-28: Include account, sender, conversation, message, attachment,
  and capability information in native events.
- [x] 2026-07-28: Add explicit native-service user allowlists with default
  deny behavior.
- [x] 2026-07-28: Add structured text and media replies.
- [x] 2026-07-28: Add service-authenticated proactive messages.
- [x] 2026-07-28: Add multi-account outbound selection.
- [x] 2026-07-28: Preserve ACP, CLI, and OpenAI-compatible HTTP behavior.
- [x] 2026-07-28: Add focused native service and authorization tests.

## Planned

- [ ] Add expiring attachment URLs for files larger than the inline limit.
- [ ] Add group-specific allowlists when stable group metadata is available.
- [ ] Add durable proactive-message delivery retries.
- [ ] Add per-service request rate limits.
