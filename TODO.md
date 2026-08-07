# Development status

Updated: 2026-07-30

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
- [x] 2026-07-29: Save configuration as owner-writable and service-group-readable
  (`0640`) for read-only local container integrations.
- [x] 2026-07-29: Add a standalone browser administration interface with
  first-run administrator setup and HttpOnly sessions.
- [x] 2026-07-29: Add browser QR login, multiple account status, immediate
  monitor startup, and account removal without container commands.
- [x] 2026-07-29: Record discovered contacts and apply Agent permissions to
  Native, HTTP, ACP, and CLI Agents in the common message dispatcher.
- [x] 2026-07-29: Replace the global chat-command default with a persistent
  default per WeChat account/contact pair.
- [x] 2026-07-29: Include the signed-in account instance in `/new` and `/clear`
  native reset events so Core can select and reset the correct channel session.
- [x] 2026-07-30: Reuse the latest account/contact iLink context token for
  proactive Native messages and return a retryable error when no token exists.

## Planned

- [ ] Add expiring attachment URLs for files larger than the inline limit.
- [ ] Add group-specific allowlists when stable group metadata is available.
- [ ] Add durable proactive-message delivery retries.
- [ ] Add per-service request rate limits.
