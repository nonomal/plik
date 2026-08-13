package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestIsBlockedUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		blocked   bool
	}{
		{"Slackbot", "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)", true},
		{"TelegramBot", "TelegramBot (like TwitterBot)", true},
		{"WhatsApp", "WhatsApp/2.23.20.0", true},
		{"Signal", "Signal/6.30.1", true},
		{"Facebook", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatype.php)", true},
		{"Facebot", "Facebot", true},
		{"Discord", "Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)", true},
		{"Skype", "SkypeUriPreview Preview/0.5", true},
		{"LinkedIn", "LinkedInBot/1.0 (compatible; Mozilla/5.0;)", true},
		{"Twitter", "Twitterbot/1.0", true},
		{"Teams", "MicrosoftPreview/2.0 +https://aka.ms/browserpolicydoc", true},
		{"Mattermost", "Mattermost-Bot/1.1", true},
		{"Normal browser", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36", false},
		{"Curl", "curl/7.68.0", false},
		{"Go client", "Go-http-client/1.1", false},
		{"Plik client", "plik_client/1.3.0", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/file/test", &bytes.Buffer{})
			require.NoError(t, err)
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}
			require.Equal(t, tt.blocked, IsBlockedUserAgent(req))
		})
	}
}

func TestBlockBotDownloadOneShotBlocked(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{OneShot: true}
	upload.InitializeForTests()
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/file/test", &bytes.Buffer{})
	require.NoError(t, err)
	req.Header.Set("User-Agent", "Slackbot-LinkExpanding 1.0")

	rr := ctx.NewRecorder(req)
	BlockBotDownload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusNotAcceptable, "link preview bots are not allowed")
}

func TestBlockBotDownloadStreamingBlocked(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{Stream: true}
	upload.InitializeForTests()
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/file/test", &bytes.Buffer{})
	require.NoError(t, err)
	req.Header.Set("User-Agent", "WhatsApp/2.23.20.0")

	rr := ctx.NewRecorder(req)
	BlockBotDownload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusNotAcceptable, "link preview bots are not allowed")
}

func TestBlockBotDownloadNormalUploadAllowed(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	upload.InitializeForTests()
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/file/test", &bytes.Buffer{})
	require.NoError(t, err)
	req.Header.Set("User-Agent", "Slackbot-LinkExpanding 1.0")

	rr := ctx.NewRecorder(req)
	BlockBotDownload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestOK(t, rr)
}

func TestBlockBotDownloadNormalUserAgentOneShotAllowed(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{OneShot: true}
	upload.InitializeForTests()
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/file/test", &bytes.Buffer{})
	require.NoError(t, err)
	req.Header.Set("User-Agent", "curl/7.68.0")

	rr := ctx.NewRecorder(req)
	BlockBotDownload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestOK(t, rr)
}

func TestBlockBotDownloadHEADAllowed(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{OneShot: true}
	upload.InitializeForTests()
	ctx.SetUpload(upload)

	req, err := http.NewRequest("HEAD", "/file/test", &bytes.Buffer{})
	require.NoError(t, err)
	req.Header.Set("User-Agent", "Slackbot-LinkExpanding 1.0")

	rr := ctx.NewRecorder(req)
	BlockBotDownload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestOK(t, rr)
}
