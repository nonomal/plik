package handlers

import (
	"bytes"
	gocontext "context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestGitHubLogin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_app_id"
	ctx.GetConfig().GitHubAPISecret = "github_app_secret"

	req, err := http.NewRequest("GET", "/auth/github/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	rr := ctx.NewRecorder(req)
	GitHubLogin(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

	URL, err := url.Parse(string(respBody))
	require.NoError(t, err, "unable to parse github auth url")

	state, err := jwt.Parse(URL.Query().Get("state"), func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Fatalf("Unexpected signing method : %v", token.Header["alg"])
		}

		if expire, ok := token.Claims.(jwt.MapClaims)["expire"]; ok {
			if _, ok = expire.(float64); ok {
				if time.Now().Unix() > (int64)(expire.(float64)) {
					t.Fatal("state expired")
				}
			} else {
				t.Fatal("invalid state expiration date")
			}
		} else {
			t.Fatal("Missing state expiration date")
		}

		return []byte(ctx.GetConfig().GitHubAPISecret), nil
	})
	require.NoError(t, err, "invalid oauth2 state")

	require.Equal(t, origin+"/auth/github/callback", state.Claims.(jwt.MapClaims)["redirectURL"].(string), "invalid state origin")

	// PKCE: code_challenge and method must be present in the auth URL
	require.NotEmpty(t, URL.Query().Get("code_challenge"), "code_challenge must be present in auth URL")
	require.Equal(t, "S256", URL.Query().Get("code_challenge_method"), "code_challenge_method must be S256")

	// PKCE: verifier must be embedded in the state JWT
	require.NotEmpty(t, state.Claims.(jwt.MapClaims)["pkceVerifier"], "pkceVerifier must be present in state JWT")
}

func TestGitHubLoginWithOrgScope(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_app_id"
	ctx.GetConfig().GitHubAPISecret = "github_app_secret"
	ctx.GetConfig().GitHubValidOrganizations = []string{"myorg"}

	req, err := http.NewRequest("GET", "/auth/github/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	rr := ctx.NewRecorder(req)
	GitHubLogin(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	URL, err := url.Parse(string(respBody))
	require.NoError(t, err, "unable to parse github auth url")

	// Verify that read:org scope is included
	scopes := URL.Query().Get("scope")
	require.Contains(t, scopes, "read:org", "read:org scope should be present when orgs are configured")
}

func TestGitHubLoginAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureDisabled
	ctx.GetConfig().GitHubAuthentication = false

	req, err := http.NewRequest("GET", "/auth/github/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestGitHubLoginGitHubAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = false

	req, err := http.NewRequest("GET", "/auth/github/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := ctx.NewRecorder(req)
	GitHubLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "GitHub authentication is disabled")
}

func TestGitHubLoginMissingCredentials(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true

	req, err := http.NewRequest("GET", "/auth/github/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := ctx.NewRecorder(req)
	GitHubLogin(ctx, rr, req)

	context.TestInternalServerError(t, rr, "missing GitHub API credentials")
}

func TestGitHubLoginMissingReferer(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_app_id"
	ctx.GetConfig().GitHubAPISecret = "github_app_secret"

	req, err := http.NewRequest("GET", "/auth/github/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing referer header")
}

func newGitHubCallbackContext(t *testing.T) (*context.Context, string) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_api_client_id"
	ctx.GetConfig().GitHubAPISecret = "github_api_secret"

	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["redirectURL"] = "https://plik.root.gg/auth/github/callback"
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(time.Minute * 5).Unix()

	b64state, err := state.SignedString([]byte(ctx.GetConfig().GitHubAPISecret))
	require.NoError(t, err, "unable to sign state")

	return ctx, b64state
}

func TestGitHubCallback(t *testing.T) {
	ctx, b64state := newGitHubCallbackContext(t)

	ghUser := githubUser{
		Login:     "octocat",
		Name:      "The Octocat",
		Email:     "octocat@github.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/583231",
	}

	// Pre-create user
	user := common.NewUser(common.ProviderGitHub, ghUser.Login)
	user.Login = ghUser.Login
	user.Name = "Old Name"
	user.Email = ghUser.Email
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	oauthToken := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}{
		AccessToken:  "access_token",
		TokenType:    "bearer",
		RefreshToken: "refresh_token",
		ExpiresIn:    int32(time.Now().Add(5 * time.Minute).Unix()),
	}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			responseBody, err := json.Marshal(oauthToken)
			require.NoError(t, err, "unable to marshal oauth token")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user" {
			responseBody, err := json.Marshal(ghUser)
			require.NoError(t, err, "unable to marshal github user")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start API mock server")

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	userinfoEndpoint := oauth2TestEndpoint.AuthURL + "/user"
	req = req.WithContext(gocontext.WithValue(
		gocontext.WithValue(gocontext.TODO(), githubEndpointContextKey, oauth2TestEndpoint),
		githubUserinfoContextKey, userinfoEndpoint))

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	var sessionCookie string
	var xsrfCookie string
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-session" {
			sessionCookie = cookie.Value
		}
		if cookie.Name == "plik-xsrf" {
			xsrfCookie = cookie.Value
		}
	}

	require.NotEqual(t, "", sessionCookie, "missing plik session cookie")
	require.NotEqual(t, "", xsrfCookie, "missing plik xsrf cookie")

	// Verify that user fields were updated on re-login
	updated, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderGitHub, ghUser.Login))
	require.NoError(t, err)
	require.NotNil(t, updated, "missing user")
	require.Equal(t, ghUser.Name, updated.Name, "user name not updated on re-login")
	require.Equal(t, ghUser.AvatarURL, updated.ProfilePicture, "user profile picture not updated on re-login")
}

func TestGitHubCallbackAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureDisabled

	req, err := http.NewRequest("GET", "/auth/github/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestGitHubCallbackGitHubAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled

	req, err := http.NewRequest("GET", "/auth/github/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "GitHub authentication is disabled")
}

func TestGitHubCallbackMissingCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_api_client_id"
	ctx.GetConfig().GitHubAPISecret = "github_api_secret"

	req, err := http.NewRequest("GET", "/auth/github/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing oauth2 authorization code")
}

func TestGitHubCallbackMissingState(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_api_client_id"
	ctx.GetConfig().GitHubAPISecret = "github_api_secret"

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing oauth2 authorization state")
}

func TestGitHubCallbackInvalidState(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_api_client_id"
	ctx.GetConfig().GitHubAPISecret = "github_api_secret"

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state=invalid", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid oauth2 state")
}

func TestGitHubCallbackExpiredState(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().GitHubAuthentication = true
	ctx.GetConfig().GitHubAPIClientID = "github_api_client_id"
	ctx.GetConfig().GitHubAPISecret = "github_api_secret"

	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(-time.Minute * 5).Unix()

	b64state, err := state.SignedString([]byte(ctx.GetConfig().GitHubAPISecret))
	require.NoError(t, err, "unable to sign state")

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid oauth2 state")
}

func TestGitHubCallbackCreateUser(t *testing.T) {
	ctx, b64state := newGitHubCallbackContext(t)

	ghUser := githubUser{
		Login:     "octocat",
		Name:      "The Octocat",
		Email:     "octocat@github.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/583231",
	}

	oauthToken := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}{
		AccessToken:  "access_token",
		TokenType:    "bearer",
		RefreshToken: "refresh_token",
		ExpiresIn:    int32(time.Now().Add(5 * time.Minute).Unix()),
	}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			responseBody, err := json.Marshal(oauthToken)
			require.NoError(t, err, "unable to marshal oauth token")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user" {
			responseBody, err := json.Marshal(ghUser)
			require.NoError(t, err, "unable to marshal github user")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start API mock server")

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	userinfoEndpoint := oauth2TestEndpoint.AuthURL + "/user"
	req = req.WithContext(gocontext.WithValue(
		gocontext.WithValue(gocontext.TODO(), githubEndpointContextKey, oauth2TestEndpoint),
		githubUserinfoContextKey, userinfoEndpoint))

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	var sessionCookie string
	var xsrfCookie string
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "plik-session" {
			sessionCookie = cookie.Value
		}
		if cookie.Name == "plik-xsrf" {
			xsrfCookie = cookie.Value
		}
	}

	require.NotEqual(t, "", sessionCookie, "missing plik session cookie")
	require.NotEqual(t, "", xsrfCookie, "missing plik xsrf cookie")

	user, err := ctx.GetMetadataBackend().GetUser("github:octocat")
	require.NotNil(t, user, "missing user")
	require.Equal(t, ghUser.Email, user.Email, "invalid user email")
	require.Equal(t, ghUser.Name, user.Name, "invalid user name")
	require.Equal(t, ghUser.AvatarURL, user.ProfilePicture, "invalid user profile picture")
}

func TestGitHubCallbackCreateUserNotWhitelisted(t *testing.T) {
	ctx, b64state := newGitHubCallbackContext(t)
	ctx.SetWhitelisted(false)

	ghUser := githubUser{
		Login:     "octocat",
		Name:      "The Octocat",
		Email:     "octocat@github.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/583231",
	}

	oauthToken := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}{
		AccessToken:  "access_token",
		TokenType:    "bearer",
		RefreshToken: "refresh_token",
		ExpiresIn:    int32(time.Now().Add(5 * time.Minute).Unix()),
	}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			responseBody, err := json.Marshal(oauthToken)
			require.NoError(t, err, "unable to marshal oauth token")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user" {
			responseBody, err := json.Marshal(ghUser)
			require.NoError(t, err, "unable to marshal github user")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start API mock server")

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	userinfoEndpoint := oauth2TestEndpoint.AuthURL + "/user"
	req = req.WithContext(gocontext.WithValue(
		gocontext.WithValue(gocontext.TODO(), githubEndpointContextKey, oauth2TestEndpoint),
		githubUserinfoContextKey, userinfoEndpoint))

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "unable to create user from untrusted source IP address")
}

func TestGitHubCallbackInvalidOrganization(t *testing.T) {
	ctx, b64state := newGitHubCallbackContext(t)
	ctx.GetConfig().GitHubValidOrganizations = []string{"allowed-org"}

	ghUser := githubUser{
		Login:     "octocat",
		Name:      "The Octocat",
		Email:     "octocat@github.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/583231",
	}

	// Pre-create user (simulates previously allowed org)
	user := common.NewUser(common.ProviderGitHub, ghUser.Login)
	user.Login = ghUser.Login
	user.Name = ghUser.Name
	user.Email = ghUser.Email
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	oauthToken := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}{
		AccessToken:  "access_token",
		TokenType:    "bearer",
		RefreshToken: "refresh_token",
		ExpiresIn:    int32(time.Now().Add(5 * time.Minute).Unix()),
	}

	// User belongs to "other-org", not "allowed-org"
	userOrgs := []githubOrg{{Login: "other-org"}}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			responseBody, err := json.Marshal(oauthToken)
			require.NoError(t, err, "unable to marshal oauth token")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user" {
			responseBody, err := json.Marshal(ghUser)
			require.NoError(t, err, "unable to marshal github user")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user/orgs" {
			responseBody, err := json.Marshal(userOrgs)
			require.NoError(t, err, "unable to marshal github orgs")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start API mock server")

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	userinfoEndpoint := oauth2TestEndpoint.AuthURL + "/user"
	orgsEndpoint := oauth2TestEndpoint.AuthURL + "/user/orgs"
	ctx2 := gocontext.WithValue(gocontext.TODO(), githubEndpointContextKey, oauth2TestEndpoint)
	ctx2 = gocontext.WithValue(ctx2, githubUserinfoContextKey, userinfoEndpoint)
	ctx2 = gocontext.WithValue(ctx2, githubOrgsContextKey, orgsEndpoint)
	req = req.WithContext(ctx2)

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "unauthorized organization")
}

func TestGitHubCallbackValidOrganization(t *testing.T) {
	ctx, b64state := newGitHubCallbackContext(t)
	ctx.GetConfig().GitHubValidOrganizations = []string{"allowed-org"}

	ghUser := githubUser{
		Login:     "octocat",
		Name:      "The Octocat",
		Email:     "octocat@github.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/583231",
	}

	oauthToken := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}{
		AccessToken:  "access_token",
		TokenType:    "bearer",
		RefreshToken: "refresh_token",
		ExpiresIn:    int32(time.Now().Add(5 * time.Minute).Unix()),
	}

	// User belongs to "allowed-org"
	userOrgs := []githubOrg{{Login: "other-org"}, {Login: "allowed-org"}}

	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			responseBody, err := json.Marshal(oauthToken)
			require.NoError(t, err, "unable to marshal oauth token")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user" {
			responseBody, err := json.Marshal(ghUser)
			require.NoError(t, err, "unable to marshal github user")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		if req.URL.Path == "/user/orgs" {
			responseBody, err := json.Marshal(userOrgs)
			require.NoError(t, err, "unable to marshal github orgs")
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start API mock server")

	req, err := http.NewRequest("GET", "/auth/github/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	userinfoEndpoint := oauth2TestEndpoint.AuthURL + "/user"
	orgsEndpoint := oauth2TestEndpoint.AuthURL + "/user/orgs"
	ctx2 := gocontext.WithValue(gocontext.TODO(), githubEndpointContextKey, oauth2TestEndpoint)
	ctx2 = gocontext.WithValue(ctx2, githubUserinfoContextKey, userinfoEndpoint)
	ctx2 = gocontext.WithValue(ctx2, githubOrgsContextKey, orgsEndpoint)
	req = req.WithContext(ctx2)

	rr := ctx.NewRecorder(req)
	GitHubCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	user, err := ctx.GetMetadataBackend().GetUser("github:octocat")
	require.NotNil(t, user, "missing user")
	require.Equal(t, ghUser.Email, user.Email, "invalid user email")
	require.Equal(t, ghUser.Name, user.Name, "invalid user name")
	require.Equal(t, ghUser.AvatarURL, user.ProfilePicture, "invalid user profile picture")
}
