package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestCLIAuthInit(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	reqBody, err := json.Marshal(CLIAuthInitRequest{Hostname: "test-host"})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/init", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthInit(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)

	var result CLIAuthInitResponse
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	require.NotEmpty(t, result.Code, "missing code")
	require.NotEmpty(t, result.Secret, "missing secret")
	require.NotEmpty(t, result.VerifyURL, "missing verify URL")
	require.Equal(t, 300, result.ExpiresIn, "invalid expiresIn")
	require.Contains(t, result.VerifyURL, result.Code, "verify URL should contain code")
}

func TestCLIAuthInit_AuthDisabled(t *testing.T) {
	config := common.NewConfiguration()
	config.FeatureAuthentication = common.FeatureDisabled
	ctx := newTestingContext(config)

	req, err := http.NewRequest("POST", "/auth/cli/init", bytes.NewBuffer([]byte("{}")))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthInit(ctx, rr, req)
	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestCLIAuthInit_ApiTokensDisabled(t *testing.T) {
	config := common.NewConfiguration()
	config.FeatureApiTokens = common.FeatureDisabled
	ctx := newTestingContext(config)

	reqBody, err := json.Marshal(CLIAuthInitRequest{Hostname: "test-host"})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/init", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthInit(ctx, rr, req)
	context.TestBadRequest(t, rr, "API tokens are disabled")
}

func TestCLIAuthApprove_ApiTokensDisabled(t *testing.T) {
	config := common.NewConfiguration()
	config.FeatureApiTokens = common.FeatureDisabled
	ctx := newTestingContext(config)

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)
	ctx.SetUser(user)

	session := common.NewCLIAuthSession()
	err = ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	reqBody, err := json.Marshal(CLIAuthApproveRequest{Code: session.Code})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthApprove(ctx, rr, req)
	context.TestBadRequest(t, rr, "API tokens are disabled")
}

func TestCLIAuthPoll_Pending(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	// Create a session
	session := common.NewCLIAuthSession()
	err := ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	reqBody, err := json.Marshal(CLIAuthPollRequest{Code: session.Code, Secret: session.Secret})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/poll", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthPoll(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)

	var result CLIAuthPollResponse
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	require.Equal(t, "pending", result.Status)
	require.Empty(t, result.Token)
}

func TestCLIAuthPoll_WrongSecret(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	session := common.NewCLIAuthSession()
	err := ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	reqBody, err := json.Marshal(CLIAuthPollRequest{Code: session.Code, Secret: "wrong-secret"})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/poll", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthPoll(ctx, rr, req)
	context.TestUnauthorized(t, rr, "invalid secret")
}

func TestCLIAuthApprove(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	// Create user
	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)
	ctx.SetUser(user)

	// Create session
	session := common.NewCLIAuthSession()
	err = ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	reqBody, err := json.Marshal(CLIAuthApproveRequest{Code: session.Code})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthApprove(ctx, rr, req)
	context.TestOK(t, rr)

	// Verify session was updated
	updated, err := ctx.GetMetadataBackend().GetCLIAuthSession(session.Code)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "approved", updated.Status)
	require.NotEmpty(t, updated.Token)

	// Verify token was created in DB
	token, err := ctx.GetMetadataBackend().GetToken(updated.Token)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.Equal(t, "CLI login", token.Comment)
	require.Equal(t, user.ID, token.UserID)
}

func TestCLIAuthApprove_Unauthenticated(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	session := common.NewCLIAuthSession()
	err := ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	reqBody, err := json.Marshal(CLIAuthApproveRequest{Code: session.Code})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthApprove(ctx, rr, req)
	context.TestUnauthorized(t, rr, "missing user, please login first")
}

func TestCLIAuthApprove_InvalidCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)
	ctx.SetUser(user)

	reqBody, err := json.Marshal(CLIAuthApproveRequest{Code: "XXXX-YYYY"})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	CLIAuthApprove(ctx, rr, req)
	context.TestNotFound(t, rr, "CLI auth session not found or expired")
}

func TestCLIAuthPoll_Approved(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	// Create user
	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)
	ctx.SetUser(user)

	// Create and approve session
	session := common.NewCLIAuthSession()
	err = ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	// Approve
	approveBody, err := json.Marshal(CLIAuthApproveRequest{Code: session.Code})
	require.NoError(t, err)
	approveReq, err := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(approveBody))
	require.NoError(t, err)
	approveRR := ctx.NewRecorder(approveReq)
	CLIAuthApprove(ctx, approveRR, approveReq)
	context.TestOK(t, approveRR)

	// Poll
	pollBody, err := json.Marshal(CLIAuthPollRequest{Code: session.Code, Secret: session.Secret})
	require.NoError(t, err)
	pollReq, err := http.NewRequest("POST", "/auth/cli/poll", bytes.NewBuffer(pollBody))
	require.NoError(t, err)
	pollRR := ctx.NewRecorder(pollReq)
	CLIAuthPoll(ctx, pollRR, pollReq)
	context.TestOK(t, pollRR)

	respBody, err := io.ReadAll(pollRR.Body)
	require.NoError(t, err)

	var result CLIAuthPollResponse
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	require.Equal(t, "approved", result.Status)
	require.NotEmpty(t, result.Token)

	// Verify session was deleted (one-time consumption)
	deleted, err := ctx.GetMetadataBackend().GetCLIAuthSession(session.Code)
	require.NoError(t, err)
	require.Nil(t, deleted, "session should be deleted after consumption")
}

func TestCLIAuthApprove_AlreadyApproved(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "user1")
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err)
	ctx.SetUser(user)

	session := common.NewCLIAuthSession()
	err = ctx.GetMetadataBackend().CreateCLIAuthSession(session)
	require.NoError(t, err)

	// First approve
	body1, _ := json.Marshal(CLIAuthApproveRequest{Code: session.Code})
	req1, _ := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(body1))
	rr1 := ctx.NewRecorder(req1)
	CLIAuthApprove(ctx, rr1, req1)
	context.TestOK(t, rr1)

	// Second approve
	body2, _ := json.Marshal(CLIAuthApproveRequest{Code: session.Code})
	req2, _ := http.NewRequest("POST", "/auth/cli/approve", bytes.NewBuffer(body2))
	rr2 := ctx.NewRecorder(req2)
	CLIAuthApprove(ctx, rr2, req2)
	context.TestBadRequest(t, rr2, "CLI auth session has already been approved")
}

func TestCLIAuthPoll_MissingCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	reqBody, _ := json.Marshal(CLIAuthPollRequest{Secret: "some-secret"})
	req, _ := http.NewRequest("POST", "/auth/cli/poll", bytes.NewBuffer(reqBody))

	rr := ctx.NewRecorder(req)
	CLIAuthPoll(ctx, rr, req)
	context.TestMissingParameter(t, rr, "code")
}

func TestCLIAuthPoll_MissingSecret(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	reqBody, _ := json.Marshal(CLIAuthPollRequest{Code: "ABCD-EFGH"})
	req, _ := http.NewRequest("POST", "/auth/cli/poll", bytes.NewBuffer(reqBody))

	rr := ctx.NewRecorder(req)
	CLIAuthPoll(ctx, rr, req)
	context.TestMissingParameter(t, rr, "secret")
}
