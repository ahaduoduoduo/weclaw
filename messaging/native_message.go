package messaging

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/fastclaw-ai/weclaw/agent"
	"github.com/fastclaw-ai/weclaw/ilink"
)

const nativeAttachmentLimit = 8 << 20

func buildNativeMessage(
	ctx context.Context,
	client *ilink.Client,
	msg ilink.WeixinMessage,
	text string,
) agent.InboundMessage {
	messageID := strconv.FormatInt(msg.MessageID, 10)
	if msg.MessageID == 0 {
		messageID = NewClientID()
	}
	return agent.InboundMessage{
		Version:            "2026-07-01",
		EventID:            fmt.Sprintf("wechat:%s:%s", client.BotID(), messageID),
		EventType:          "message.created",
		Provider:           "wechat",
		ProviderInstanceID: client.BotID(),
		ConversationID:     msg.FromUserID,
		SenderID:           msg.FromUserID,
		MessageID:          messageID,
		MessageType:        nativeMessageType(msg),
		Text:               text,
		Attachments:        nativeAttachments(ctx, msg),
		Timestamp:          time.Now().UTC(),
		Capabilities:       []string{"text", "image", "video", "file", "typing"},
	}
}

func nativeMessageType(msg ilink.WeixinMessage) string {
	for _, item := range msg.ItemList {
		switch item.Type {
		case ilink.ItemTypeText:
			return "text"
		case ilink.ItemTypeImage:
			return "image"
		case ilink.ItemTypeVoice:
			return "voice"
		case ilink.ItemTypeFile:
			return "file"
		case ilink.ItemTypeVideo:
			return "video"
		}
	}
	return "unknown"
}

func nativeAttachments(ctx context.Context, msg ilink.WeixinMessage) []agent.Attachment {
	var attachments []agent.Attachment
	for _, item := range msg.ItemList {
		var attachment agent.Attachment
		var media *ilink.MediaInfo
		var expectedSize int

		switch item.Type {
		case ilink.ItemTypeImage:
			if item.ImageItem == nil {
				continue
			}
			attachment.Type = "image"
			attachment.ContentType = "image/jpeg"
			attachment.URL = item.ImageItem.URL
			media = item.ImageItem.Media
			expectedSize = item.ImageItem.MidSize
		case ilink.ItemTypeVideo:
			if item.VideoItem == nil {
				continue
			}
			attachment.Type = "video"
			attachment.ContentType = "video/mp4"
			media = item.VideoItem.Media
			expectedSize = item.VideoItem.VideoSize
		case ilink.ItemTypeFile:
			if item.FileItem == nil {
				continue
			}
			attachment.Type = "file"
			attachment.ContentType = "application/octet-stream"
			attachment.FileName = item.FileItem.FileName
			media = item.FileItem.Media
			if size, err := strconv.Atoi(item.FileItem.Len); err == nil {
				expectedSize = size
			}
		default:
			continue
		}

		if attachment.URL == "" && media != nil && media.EncryptQueryParam != "" {
			if expectedSize > nativeAttachmentLimit {
				continue
			}
			data, err := DownloadFileFromCDN(ctx, media.EncryptQueryParam, media.AESKey)
			if err != nil || len(data) > nativeAttachmentLimit {
				continue
			}
			attachment.DataBase64 = base64.StdEncoding.EncodeToString(data)
		}
		if attachment.URL != "" || attachment.DataBase64 != "" {
			attachments = append(attachments, attachment)
		}
	}
	return attachments
}
