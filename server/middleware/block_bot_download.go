package middleware

import (
	"net/http"
	"strings"

	"github.com/root-gg/plik/server/context"
)

// blockedUserAgents contains substrings found in User-Agent headers
// of messaging app link preview bots. These bots fetch URLs to generate
// previews, which would consume one-shot downloads or streaming data.
var blockedUserAgents = []string{
	"Slackbot",
	"TelegramBot",
	"WhatsApp",
	"Signal",
	"facebookexternalhit", // Facebook / Messenger
	"Facebot",             // Facebook / Messenger
	"Discordbot",
	"SkypeUriPreview",
	"Viber",
	"LinkedInBot",
	"Twitterbot", // X / Twitter
	"Wire",
	"Mattermost",
	"Rocket.Chat",
	"Zulip",
	"Teams",            // Microsoft Teams
	"MicrosoftPreview", // Microsoft Teams link preview
}

// IsBlockedUserAgent checks if the request comes from a known
// messaging app link preview bot
func IsBlockedUserAgent(req *http.Request) bool {
	ua := req.Header.Get("User-Agent")
	if ua == "" {
		return false
	}
	for _, blocked := range blockedUserAgents {
		if strings.Contains(ua, blocked) {
			return true
		}
	}
	return false
}

// BlockBotDownload blocks messaging app link preview bots from downloading
// one-shot and streaming files. These bots generate link previews which
// would consume the single download opportunity or the stream data,
// preventing the real recipient from accessing the file.
func BlockBotDownload(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// Only block GET requests (HEAD is harmless for one-shot/stream)
		if req.Method != "GET" {
			next.ServeHTTP(resp, req)
			return
		}

		upload := ctx.GetUpload()
		if upload == nil {
			// Let the handler panic with the appropriate message
			next.ServeHTTP(resp, req)
			return
		}

		if (upload.OneShot || upload.Stream) && IsBlockedUserAgent(req) {
			ctx.NotAcceptable("Messaging app link preview bots are not allowed to download one-shot or streaming files")
			return
		}

		next.ServeHTTP(resp, req)
	})
}
