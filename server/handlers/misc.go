package handlers

import (
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/metadata"
)

// GetVersion return the build information.
func GetVersion(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	bi := common.GetBuildInfo()
	if !ctx.IsAdmin() {
		bi.Sanitize()
	}
	common.WriteJSONResponse(resp, bi)
}

// GetConfiguration return the server configuration
func GetConfiguration(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	common.WriteJSONResponse(resp, ctx.GetConfig())
}

// Logout delete session cookies
func Logout(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	authenticator := ctx.GetAuthenticatorSafe()
	if authenticator == nil {
		return
	}
	common.Logout(resp, authenticator)
}

// GetQrCode return a QRCode for the requested URL
func GetQrCode(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Check params
	urlParam := req.FormValue("url")
	sizeParam := req.FormValue("size")

	// Parse int on size
	sizeInt, err := strconv.Atoi(sizeParam)
	if err != nil {
		sizeInt = 250
	}
	if sizeInt <= 0 {
		ctx.BadRequest("QRCode size must be positive")
		return
	}
	if sizeInt > 1000 {
		ctx.BadRequest("QRCode size must be lower than 1000")
		return
	}

	// Generate QRCode png from url
	qrcode, err := qr.Encode(urlParam, qr.H, qr.Auto)
	if err != nil {
		ctx.InternalServerError("unable to generate QRCode", err)
		return
	}

	// Scale QRCode png size
	qrcode, err = barcode.Scale(qrcode, sizeInt, sizeInt)
	if err != nil {
		ctx.InternalServerError("unable to scale QRCode : %s", err)
		return
	}

	resp.Header().Add("Content-Type", "image/png")
	err = png.Encode(resp, qrcode)
	if err != nil {
		ctx.InternalServerError("unable to encore png : %s", err)
		return
	}
}

// Health is a handler to check for service health
func Health(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	_, _ = io.WriteString(resp, "ok\n")
}

// If a download domain is specified verify that the request comes from this specific domain
func checkDownloadDomain(ctx *context.Context) bool {
	config := ctx.GetConfig()
	req := ctx.GetReq()

	if !config.IsValidDownloadDomain(req.Host) {
		ctx.BadRequest("Invalid download domain %s", req.Host)
		return false
	}

	return true
}

func getRedirectURL(ctx *context.Context, callbackPath string) (redirectURL string, err error) {
	config := ctx.GetConfig()
	req := ctx.GetReq()

	// Prefer PlikDomain (reliable, no header dependency)
	if config.GetPlikDomain() != nil {
		redirectURL = config.PlikDomain
		if config.Path != "" {
			redirectURL += config.Path
		}
		redirectURL += callbackPath
		return redirectURL, nil
	}

	// Fall back to Referer header for backward compatibility
	referer := req.Header.Get("referer")
	if referer == "" {
		return "", common.NewHTTPError("missing referer header", nil, http.StatusBadRequest)
	}

	originURL, err := url.Parse(referer)
	if err != nil {
		return "", common.NewHTTPError("invalid referer header", nil, http.StatusBadRequest)
	}

	redirectURL = fmt.Sprintf("%s://%s", originURL.Scheme, originURL.Host)
	if config.Path != "" {
		redirectURL += config.Path
	}
	redirectURL += callbackPath

	return redirectURL, nil
}

// setCORSHeaders adds CORS headers to download responses when PlikDomain and
// DownloadDomain are both configured, allowing the webapp to fetch file content
// cross-origin (e.g., for the file viewer and E2EE decrypt).
func setCORSHeaders(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if origin := ctx.GetConfig().GetCORSOrigin(); origin != "" && req.Header.Get("Origin") != "" {
		resp.Header().Set("Access-Control-Allow-Origin", origin)
	}
}

func handleHTTPError(ctx *context.Context, err error) {
	if httpError, ok := err.(common.HTTPError); ok {
		ctx.Fail(httpError.Message, httpError.Err, httpError.StatusCode)
	} else {
		ctx.InternalServerError("unexpected error", err)
	}
}

// parseBoolFilter returns a *bool from a query parameter.
// Returns nil if the parameter is absent, enabling optional boolean filtering.
func parseBoolFilter(req *http.Request, key string) *bool {
	if v := req.URL.Query().Get(key); v != "" {
		b := v == "true"
		return &b
	}
	return nil
}

// parseBadgeFilters builds an UploadFilters with the six badge-setting
// query parameters.  Callers may set .User / .Token afterwards.
func parseBadgeFilters(req *http.Request) metadata.UploadFilters {
	return metadata.UploadFilters{
		OneShot:   parseBoolFilter(req, "oneShot"),
		Removable: parseBoolFilter(req, "removable"),
		Stream:    parseBoolFilter(req, "stream"),
		ExtendTTL: parseBoolFilter(req, "extendTTL"),
		Password:  parseBoolFilter(req, "password"),
		E2EE:      parseBoolFilter(req, "e2ee"),
	}
}
