# Native message services

Updated: 2026-07-28

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

`allowed_users` is required for native services. An empty list rejects every
sender. Use `["*"]` only on a trusted private deployment.

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

Keep the API on a private container network. The legacy `/api/send` endpoint is
retained for compatibility and does not provide native service authorization.

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
