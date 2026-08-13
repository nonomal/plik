package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// CLIAuthInitRequest is the request body for the CLI auth init endpoint.
type CLIAuthInitRequest struct {
	Hostname string `json:"hostname"`
}

// CLIAuthInitResponse is the response body for the CLI auth init endpoint.
type CLIAuthInitResponse struct {
	Code      string `json:"code"`
	Secret    string `json:"secret"`
	VerifyURL string `json:"verifyURL"`
	ExpiresIn int    `json:"expiresIn"`
}

// CLIAuthApproveRequest is the request body for the CLI auth approve endpoint.
type CLIAuthApproveRequest struct {
	Code    string `json:"code"`
	Comment string `json:"comment"`
}

// CLIAuthPollRequest is the request body for the CLI auth poll endpoint.
type CLIAuthPollRequest struct {
	Code   string `json:"code"`
	Secret string `json:"secret"`
}

// CLIAuthPollResponse is the response body for the CLI auth poll endpoint.
type CLIAuthPollResponse struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"`
}

// CLIAuthInit initiates a new CLI device authorization session.
// The CLI calls this to get a code and verification URL to display to the user.
func CLIAuthInit(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	if config.FeatureAuthentication == common.FeatureDisabled {
		ctx.BadRequest("authentication is disabled")
		return
	}

	if config.FeatureApiTokens == common.FeatureDisabled {
		ctx.BadRequest("API tokens are disabled")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest(fmt.Sprintf("unable to read request body : %s", err))
		return
	}

	var params CLIAuthInitRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &params); err != nil {
			ctx.BadRequest(fmt.Sprintf("unable to deserialize request body : %s", err))
			return
		}
	}

	// Create session
	session := common.NewCLIAuthSession()

	err = ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	if err != nil {
		ctx.InternalServerError("unable to create CLI auth session", err)
		return
	}

	// Build verification URL
	serverURL := config.GetServerURL()
	verifyURL := fmt.Sprintf("%s/#/cli-auth?code=%s", serverURL.String(), session.Code)
	if params.Hostname != "" {
		verifyURL += "&hostname=" + url.QueryEscape(params.Hostname)
	}

	result := CLIAuthInitResponse{
		Code:      session.Code,
		Secret:    session.Secret,
		VerifyURL: verifyURL,
		ExpiresIn: 300,
	}

	respBytes, err := json.Marshal(result)
	if err != nil {
		ctx.InternalServerError("unable to serialize response", err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	_, _ = resp.Write(respBytes)
}

// CLIAuthApprove is called by the browser (authenticated user) to approve a CLI login session.
// It creates a token using the existing token mechanism and links it to the session.
func CLIAuthApprove(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	if config.FeatureAuthentication == common.FeatureDisabled {
		ctx.BadRequest("authentication is disabled")
		return
	}

	if config.FeatureApiTokens == common.FeatureDisabled {
		ctx.BadRequest("API tokens are disabled")
		return
	}

	// Get user from context (requires authentication)
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest(fmt.Sprintf("unable to read request body : %s", err))
		return
	}

	var params CLIAuthApproveRequest
	if err := json.Unmarshal(body, &params); err != nil {
		ctx.BadRequest(fmt.Sprintf("unable to deserialize request body : %s", err))
		return
	}

	if params.Code == "" {
		ctx.MissingParameter("code")
		return
	}

	// Look up session
	session, err := ctx.GetMetadataBackend().GetCLIAuthSession(params.Code)
	if err != nil {
		ctx.InternalServerError("unable to get CLI auth session", err)
		return
	}

	if session == nil {
		ctx.NotFound("CLI auth session not found or expired")
		return
	}

	if session.IsExpired() {
		// Clean up expired session
		_ = ctx.GetMetadataBackend().DeleteCLIAuthSession(session.Code)
		ctx.NotFound("CLI auth session has expired")
		return
	}

	if session.Status != "pending" {
		ctx.BadRequest("CLI auth session has already been approved")
		return
	}

	// Create a token using the existing mechanism
	token := common.NewToken()
	if params.Comment != "" {
		token.Comment = params.Comment
	} else {
		token.Comment = "CLI login"
	}
	token.UserID = user.ID

	err = ctx.GetMetadataBackend().CreateToken(token)
	if err != nil {
		ctx.InternalServerError("unable to create token : %s", err)
		return
	}

	// Update session with the token
	session.Status = "approved"
	session.Token = token.Token
	err = ctx.GetMetadataBackend().UpdateCLIAuthSession(session)
	if err != nil {
		ctx.InternalServerError("unable to update CLI auth session", err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	_, _ = resp.Write([]byte(`{"status":"approved"}`))
}

// CLIAuthPoll is called by the CLI to check if the user has approved the session.
// Once approved, it returns the token and deletes the session (one-time consumption).
func CLIAuthPoll(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	if config.FeatureAuthentication == common.FeatureDisabled {
		ctx.BadRequest("authentication is disabled")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest(fmt.Sprintf("unable to read request body : %s", err))
		return
	}

	var params CLIAuthPollRequest
	if err := json.Unmarshal(body, &params); err != nil {
		ctx.BadRequest(fmt.Sprintf("unable to deserialize request body : %s", err))
		return
	}

	if params.Code == "" {
		ctx.MissingParameter("code")
		return
	}

	if params.Secret == "" {
		ctx.MissingParameter("secret")
		return
	}

	// Look up session
	session, err := ctx.GetMetadataBackend().GetCLIAuthSession(params.Code)
	if err != nil {
		ctx.InternalServerError("unable to get CLI auth session", err)
		return
	}

	if session == nil {
		ctx.NotFound("CLI auth session not found")
		return
	}

	// Validate secret
	if session.Secret != params.Secret {
		ctx.Unauthorized("invalid secret")
		return
	}

	if session.IsExpired() {
		_ = ctx.GetMetadataBackend().DeleteCLIAuthSession(session.Code)
		ctx.NotFound("CLI auth session has expired")
		return
	}

	if session.Status == "pending" {
		resp.Header().Set("Content-Type", "application/json")
		_, _ = resp.Write([]byte(`{"status":"pending"}`))
		return
	}

	// Session is approved — return token and delete session (one-time consumption)
	result := CLIAuthPollResponse{
		Status: "approved",
		Token:  session.Token,
	}

	// Delete the session after consumption
	_ = ctx.GetMetadataBackend().DeleteCLIAuthSession(session.Code)

	respBytes, err := json.Marshal(result)
	if err != nil {
		ctx.InternalServerError("unable to serialize response", err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	_, _ = resp.Write(respBytes)
}
