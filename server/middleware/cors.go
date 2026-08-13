package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// CORSPreflight short-circuits OPTIONS preflight requests before the heavier
// Upload/File middleware runs. This prevents unnecessary database lookups and
// avoids failures on password-protected uploads (which would reject an OPTIONS
// request that has no Authorization header).
func CORSPreflight(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodOptions {
			next.ServeHTTP(resp, req)
			return
		}

		origin := ctx.GetConfig().GetCORSOrigin()
		if origin != "" && req.Header.Get("Origin") != "" {
			resp.Header().Set("Access-Control-Allow-Origin", origin)
			resp.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			resp.Header().Set("Access-Control-Max-Age", "86400")
		}

		resp.WriteHeader(http.StatusNoContent)
	})
}
