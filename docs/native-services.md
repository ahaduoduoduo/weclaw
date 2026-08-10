# Native message services

Updated: 2026-07-30

Native services let WeClaw connect a WeChat account to any HTTP service without
requiring that service to emulate an LLM API. The protocol carries stable user,
conversation, account, message, and attachment data and supports asynchronous
outbound messages.

## Configuration

```json
{
  "default_agent": "knowledge",
  "api_addr": "0.0.0.0:18011",
  "agents": {
    "knowledge": {
      "type": "native",
      "aliases": ["kb"],
      "endpoint": "http://knowledge-service:3000/v1/conversation/events",
      "api_key": "weclaw-to-service-secret",
      "outbound_token": "service-to-weclaw-secret",
      "allowed_users": ["user_id@im.wechat"],
      "timeout_seconds": 180
    }
  }
}
```

`allowed_users` is used by WeClaw's common dispatcher for every Agent type.
An empty list rejects every sender. The administration interface stores
account-specific grants as `<provider_instance_id>:<sender_id>`. Use `["*"]`
only when every discovered contact may use the Agent.

`api_key` authenticates WeClaw to the service. `outbound_token` authenticates
the service when it calls WeClaw's proactive message endpoint. Use different
random values.

## Inbound event

WeClaw sends `POST` requests to the configured `endpoint` with
`Authorization: Bearer <api_key>`.

```json
{
  "version": "2026-07-01",
  "event_id": "wechat:bot-id:12345",
  "event_type": "message.created",
  "provider": "wechat",
  "provider_instance_id": "bot-id",
  "conversation_id": "user_id@im.wechat",
  "sender_id": "user_id@im.wechat",
  "message_id": "12345",
  "message_type": "text",
  "text": "hello",
  "timestamp": "2026-07-28T12:00:00Z",
  "capabilities": ["text", "image", "video", "file", "typing"]
}
```

Attachments use either a source URL or base64 content:

```json
{
  "type": "image",
  "content_type": "image/jpeg",
  "data_base64": "..."
}
```

Encrypted WeChat CDN attachments are decrypted before delivery. Inline
attachments are limited to 8 MiB.

The service returns zero or more messages:

```json
{
  "messages": [
    {
      "type": "text",
      "text": "accepted"
    },
    {
      "type": "image",
      "media_url": "https://example.test/result.png"
    }
  ]
}
```

A `conversation.reset` event is sent when the user runs `/new` or `/clear`.
The event carries the signed-in account's `provider_instance_id`, so a native
service can reset the correct conversation when one WeClaw instance manages
multiple accounts.

## Proactive messages

Services can send messages after the original request has completed:

```http
POST /v1/messages
Authorization: Bearer service-to-weclaw-secret
Content-Type: application/json
```

```json
{
  "provider_instance_id": "bot-id",
  "to": "user_id@im.wechat",
  "messages": [
    {
      "type": "text",
      "text": "The background task is complete."
    }
  ]
}
```

The target must be in the native service's `allowed_users`. When
`provider_instance_id` is omitted, WeClaw uses the first logged-in account.

WeClaw keeps the latest iLink `context_token` in memory for each account and
contact. Proactive text and media messages use that token. If no current token
is available, `/v1/messages` returns HTTP 409 so the calling service can retry
after the user sends another message. An empty token is never reported as a
successful delivery, and context tokens are not written to the configuration
file or a database.

For proactive batches that contain remote media, WeClaw downloads every media
item before sending the first message. An expired or unavailable media URL
therefore fails the request before any preceding text is delivered repeatedly.

Keep the API on a private container network. Proactive delivery uses the
authenticated `/v1/messages` endpoint.

The configuration file is saved as `0640`. A local service that must discover
native-service credentials may join the file's group and mount the directory
read-only; do not expose the directory through a web server or public volume.

## Service boundaries

WeClaw owns:

- WeChat login and account state.
- Message normalization and deduplication.
- WeChat media encryption and delivery.
- Service selection by aliases.

The connected service owns:

- Its user and role model.
- Conversation state and business data.
- Long-running jobs and retries.
- Domain-specific authorization.

No domain-specific code belongs in WeClaw.
