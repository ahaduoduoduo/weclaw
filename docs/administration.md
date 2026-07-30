# WeClaw administration

Updated: 2026-07-29

The standalone administration interface is served from the same address as the
health and Native Message APIs. The default address is `0.0.0.0:18011` in the
Docker deployment.

## Administrator

The first browser visit creates one local administrator password. The password
derivation and salt are stored in `~/.weclaw/admin.json`; plaintext is never
stored. Login sessions use an HttpOnly, SameSite cookie and expire after 24
hours. The interface should be published only on a trusted LAN or behind an
authenticated HTTPS reverse proxy.

## WeChat accounts

“Add WeChat account” creates a short-lived QR session. The browser displays a
locally generated PNG and polls only WeClaw for status. After phone
confirmation, WeClaw:

1. saves the returned credential with mode `0600`;
2. creates an iLink client;
3. starts its message monitor immediately;
4. makes the account available to proactive Native messages.

No container command or restart is required. Deleting an account stops its
monitor and removes the credential and sync cursor.

## Contacts and permissions

Every completed inbound user message records the tuple
`(provider_instance_id, sender_id)` in `~/.weclaw/access.json`. This prevents
identical sender IDs on different logged-in accounts from sharing grants.

Agent access is checked before any Native, HTTP, ACP, or CLI invocation:

- `*` makes an Agent available to every contact;
- a grant created by the UI is stored as
  `<provider_instance_id>:<sender_id>`;
- blocked contacts cannot use any Agent;
- an explicit `/agent` or `@agent` command is rejected before the Agent starts
  when permission is missing.

The default Agent selected with `/agent` is stored for that account/contact
pair. Other contacts and the system-wide default are unchanged.

Connected business services may still apply their own roles. For example,
AutoFilm remains public at the WeClaw layer during migration and continues to
approve members inside AutoFilm Core.
