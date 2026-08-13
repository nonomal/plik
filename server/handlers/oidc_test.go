package handlers

import (
	"bytes"
	gocontext "context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

var oidcOAuth2TestEndpoint = oauth2.Endpoint{
	AuthURL:  "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort),
	TokenURL: "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort) + "/token",
}

var oidcUserinfoTestEndpoint = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort) + "/userinfo"

var oidcTestDiscovery = oidcDiscovery{
	AuthorizationEndpoint: "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort) + "/authorize",
	TokenEndpoint:         "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort) + "/token",
	UserinfoEndpoint:      "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort) + "/userinfo",
}

func makeTestIDToken(claims oidcClaims) string {
	token := jwt.New(jwt.SigningMethodHS256)
	mc := token.Claims.(jwt.MapClaims)
	if claims.Sub != "" {
		mc["sub"] = claims.Sub
	}
	if claims.Email != "" {
		mc["email"] = claims.Email
	}
	if claims.EmailVerified != nil {
		mc["email_verified"] = *claims.EmailVerified
	}
	if claims.Name != "" {
		mc["name"] = claims.Name
	}
	if claims.GivenName != "" {
		mc["given_name"] = claims.GivenName
	}
	if claims.FamilyName != "" {
		mc["family_name"] = claims.FamilyName
	}
	if claims.PreferredUsername != "" {
		mc["preferred_username"] = claims.PreferredUsername
	}
	if claims.Picture != "" {
		mc["picture"] = claims.Picture
	}
	if claims.Locale != "" {
		mc["locale"] = claims.Locale
	}
	// Sign with a dummy key — parseIDTokenClaims skips verification
	signed, _ := token.SignedString([]byte("test-signing-key"))
	return signed
}

// marshalOIDCClaimsForTest produces JSON that includes email_verified,
// which standard json.Marshal skips due to the json:"-" tag.
func marshalOIDCClaimsForTest(claims oidcClaims) []byte {
	data, _ := json.Marshal(claims)
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	if claims.EmailVerified != nil {
		m["email_verified"] = *claims.EmailVerified
	}
	data, _ = json.Marshal(m)
	return data
}

type oidcMockOptions struct {
	userinfo            oidcClaims
	idToken             *oidcClaims
	gotCodeVerifier     *string // if non-nil, receives the code_verifier sent during token exchange
	requireCodeVerifier bool    // if true, returns 400 when code_verifier is absent (simulates PKCE enforcement)
}

func oidcMockHandler(opts oidcMockOptions) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		var responseBody []byte
		switch req.URL.Path {
		case "/.well-known/openid-configuration":
			responseBody, _ = json.Marshal(oidcTestDiscovery)
		case "/token":
			_ = req.ParseForm()
			// Capture code_verifier if the caller wants to inspect it
			if opts.gotCodeVerifier != nil {
				*opts.gotCodeVerifier = req.FormValue("code_verifier")
			}
			// Enforce PKCE: reject exchanges that omit code_verifier
			if opts.requireCodeVerifier {
				if req.FormValue("code_verifier") == "" {
					resp.WriteHeader(http.StatusBadRequest)
					resp.Write([]byte(`{"error":"invalid_grant","error_description":"PKCE code verifier required"}`))
					return
				}
			}
			tokenResp := map[string]any{
				"access_token":  "access_token",
				"token_type":    "Bearer",
				"refresh_token": "refresh_token",
				"expires_in":    300,
			}
			if opts.idToken != nil {
				tokenResp["id_token"] = makeTestIDToken(*opts.idToken)
			}
			responseBody, _ = json.Marshal(tokenResp)
		case "/userinfo":
			responseBody = marshalOIDCClaimsForTest(opts.userinfo)
		default:
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp.Header().Set("Content-Type", "application/json")
		resp.Write(responseBody)
	}
}

func setupOIDCConfig(config *common.Configuration) {
	config.FeatureAuthentication = common.FeatureEnabled
	config.OIDCAuthentication = true
	config.OIDCClientID = "oidc_client_id"
	config.OIDCClientSecret = "oidc_client_secret"
	config.OIDCProviderURL = "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)
}

func oidcTestState(t *testing.T, secret string) string {
	t.Helper()
	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["redirectURL"] = "https://plik.root.gg/auth/oidc/callback"
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(5 * time.Minute).Unix()
	state.Claims.(jwt.MapClaims)["pkceVerifier"] = "test-pkce-verifier"
	b64state, err := state.SignedString([]byte(secret))
	require.NoError(t, err, "unable to sign state")
	return b64state
}

func oidcCallbackRequest(t *testing.T, state string) *http.Request {
	t.Helper()
	req, err := http.NewRequest("GET", "/auth/oidc/callback?code=code&state="+url.QueryEscape(state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	reqCtx := gocontext.WithValue(gocontext.TODO(), oidcEndpointContextKey, oidcOAuth2TestEndpoint)
	reqCtx = gocontext.WithValue(reqCtx, oidcUserinfoContextKey, oidcUserinfoTestEndpoint)
	req = req.WithContext(reqCtx)
	return req
}

func TestOIDCLogin(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req, err := http.NewRequest("GET", "/auth/oidc/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	origin := "https://plik.root.gg"
	req.Header.Set("referer", origin)

	rr := ctx.NewRecorder(req)
	OIDCLogin(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

	URL, err := url.Parse(string(respBody))
	require.NoError(t, err, "unable to parse OIDC auth url")

	// Verify PKCE challenge parameters are present in the authorization URL
	require.Equal(t, "S256", URL.Query().Get("code_challenge_method"), "expected PKCE S256 method in auth URL")
	require.NotEmpty(t, URL.Query().Get("code_challenge"), "expected non-empty code_challenge in auth URL")

	state, err := jwt.Parse(URL.Query().Get("state"), func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Fatalf("Unexpected signing method: %v", token.Header["alg"])
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

		return []byte(ctx.GetConfig().OIDCClientSecret), nil
	})
	require.NoError(t, err, "invalid oauth2 state")

	require.Equal(t, origin+"/auth/oidc/callback", state.Claims.(jwt.MapClaims)["redirectURL"].(string), "invalid state origin")

	// Verify pkceVerifier is embedded in the state JWT
	verifier, ok := state.Claims.(jwt.MapClaims)["pkceVerifier"].(string)
	require.True(t, ok, "pkceVerifier claim missing from state JWT")
	require.NotEmpty(t, verifier, "pkceVerifier must not be empty")
}

func TestOIDCLoginAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureDisabled
	ctx.GetConfig().OIDCAuthentication = false

	req, err := http.NewRequest("GET", "/auth/oidc/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestOIDCLoginOIDCDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().OIDCAuthentication = false

	req, err := http.NewRequest("GET", "/auth/oidc/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("referer", "http://plik.root.gg")

	rr := ctx.NewRecorder(req)
	OIDCLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "OIDC authentication is disabled")
}

func TestOIDCLoginMissingReferer(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().OIDCAuthentication = true

	req, err := http.NewRequest("GET", "/auth/oidc/login", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCLogin(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing referer header")
}

func TestOIDCCallbackAuthDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureDisabled

	req, err := http.NewRequest("GET", "/auth/oidc/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "authentication is disabled")
}

func TestOIDCCallbackOIDCDisabled(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().FeatureAuthentication = common.FeatureEnabled
	ctx.GetConfig().OIDCAuthentication = false

	req, err := http.NewRequest("GET", "/auth/oidc/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "OIDC authentication is disabled")
}

func TestOIDCCallbackMissingCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	req, err := http.NewRequest("GET", "/auth/oidc/callback", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing oauth2 authorization code")
}

func TestOIDCCallbackMissingState(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	req, err := http.NewRequest("GET", "/auth/oidc/callback?code=code", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing oauth2 authorization state")
}

func TestOIDCCallbackInvalidState(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	req, err := http.NewRequest("GET", "/auth/oidc/callback?code=code&state=state", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid oauth2 state")
}

func TestOIDCCallbackExpiredState(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(-5 * time.Minute).Unix()

	b64state, err := state.SignedString([]byte(ctx.GetConfig().OIDCClientSecret))
	require.NoError(t, err, "unable to sign state")

	req, err := http.NewRequest("GET", "/auth/oidc/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid oauth2 state")
}

func TestOIDCCallbackInvalidRedirectURL(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["redirectURL"] = "https://evil.com/steal"
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(5 * time.Minute).Unix()

	b64state, err := state.SignedString([]byte(ctx.GetConfig().OIDCClientSecret))
	require.NoError(t, err, "unable to sign state")

	req, err := http.NewRequest("GET", "/auth/oidc/callback?code=code&state="+url.QueryEscape(b64state), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid redirectURL")
}

func TestOIDCCallback(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	oidcUser := oidcClaims{
		Sub:   "user123",
		Email: "plik@root.gg",
		Name:  "plik.root.gg",
	}

	user := common.NewUser(common.ProviderOIDC, "user123")
	user.Login = oidcUser.Sub
	user.Name = oidcUser.Name
	user.Email = oidcUser.Email
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

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
}

func TestOIDCCallbackPKCEVerifier(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	oidcUser := oidcClaims{
		Sub:   "pkce-user",
		Email: "pkce@root.gg",
		Name:  "PKCE User",
	}

	// Capture the code_verifier that OIDCCallback sends to the token endpoint
	var gotVerifier string
	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo:        oidcUser,
		gotCodeVerifier: &gotVerifier,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	// The state JWT helper now always embeds pkceVerifier = "test-pkce-verifier"
	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	// The verifier embedded in the state JWT must be forwarded to the token endpoint
	require.Equal(t, "test-pkce-verifier", gotVerifier, "code_verifier not forwarded to token endpoint")
}

// TestOIDCCallbackPKCEEnforced verifies that when the token endpoint enforces PKCE
// (e.g. Keycloak with pkce.code.challenge.method=S256) and the state holds no verifier,
// the callback returns an error. This test would fail if OIDCLogin stopped sending
// code_challenge / if OIDCCallback stopped forwarding code_verifier.
func TestOIDCCallbackPKCEEnforced(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	// Token endpoint enforces PKCE: rejects exchanges that omit code_verifier
	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		requireCodeVerifier: true,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	// State without pkceVerifier → callback sends no code_verifier → token server rejects it
	state := jwt.New(jwt.SigningMethodHS256)
	state.Claims.(jwt.MapClaims)["redirectURL"] = "https://plik.root.gg/auth/oidc/callback"
	state.Claims.(jwt.MapClaims)["expire"] = time.Now().Add(5 * time.Minute).Unix()
	// deliberately omit "pkceVerifier"
	b64state, err := state.SignedString([]byte(ctx.GetConfig().OIDCClientSecret))
	require.NoError(t, err)

	req := oidcCallbackRequest(t, b64state)
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	// The enforcing token server returns 400 → Plik must surface an error, not redirect
	require.NotEqual(t, 302, rr.Code, "callback must not succeed when PKCE is enforced but verifier is absent")
}

func TestOIDCCallbackCreateUser(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	oidcUser := oidcClaims{
		Sub:     "user456",
		Email:   "newuser@root.gg",
		Name:    "New User",
		Picture: "https://example.com/avatar.jpg",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")

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

	user, err := ctx.GetMetadataBackend().GetUser("oidc:user456")
	require.NoError(t, err)
	require.NotNil(t, user, "missing user")
	require.Equal(t, oidcUser.Email, user.Email, "invalid user email")
	require.Equal(t, oidcUser.Name, user.Name, "invalid user name")
	require.Equal(t, oidcUser.Picture, user.ProfilePicture, "invalid user profile picture")
}

func TestOIDCCallbackCreateUserNotWhitelisted(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetWhitelisted(false)
	setupOIDCConfig(ctx.GetConfig())

	oidcUser := oidcClaims{
		Sub:   "user789",
		Email: "blocked@root.gg",
		Name:  "Blocked User",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "unable to create user from untrusted source IP address")
}

func TestOIDCCallbackUpdateUserFields(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	user := common.NewUser(common.ProviderOIDC, "updateuser")
	user.Login = "updateuser"
	user.Name = "Old Name"
	user.Email = "old@root.gg"
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	oidcUser := oidcClaims{
		Sub:               "updateuser",
		Email:             "new@root.gg",
		Name:              "New Name",
		PreferredUsername: "newlogin",
		Picture:           "https://example.com/new-avatar.jpg",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	updated, err := ctx.GetMetadataBackend().GetUser("oidc:updateuser")
	require.NoError(t, err)
	require.NotNil(t, updated, "missing user")
	require.Equal(t, "new@root.gg", updated.Email, "email not updated")
	require.Equal(t, "New Name", updated.Name, "name not updated")
	require.Equal(t, "newlogin", updated.Login, "login not updated")
	require.Equal(t, "https://example.com/new-avatar.jpg", updated.ProfilePicture, "profile picture not updated")
}

func TestOIDCCallbackInvalidDomain(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCValidDomains = []string{"allowed.com"}

	oidcUser := oidcClaims{
		Sub:   "domainuser",
		Email: "user@forbidden.com",
		Name:  "Domain User",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "unauthorized domain name")
}

func TestOIDCCallbackExistingUserInvalidDomain(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCValidDomains = []string{"allowed.com"}

	oidcUser := oidcClaims{
		Sub:   "existinguser",
		Email: "user@revoked.com",
		Name:  "Existing User",
	}

	user := common.NewUser(common.ProviderOIDC, "existinguser")
	user.Login = oidcUser.Sub
	user.Name = oidcUser.Name
	user.Email = oidcUser.Email
	err := ctx.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to create test user")

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "unauthorized domain name")
}

func TestOIDCCallbackDomainValidationNoEmail(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCValidDomains = []string{"allowed.com"}

	oidcUser := oidcClaims{
		Sub:  "noemail_user",
		Name: "No Email User",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "email is required when domain validation is enabled")
}

//go:fix inline
func boolPtr(b bool) *bool { return new(b) }

func TestParseIDTokenClaims(t *testing.T) {
	idClaims := oidcClaims{
		Sub:               "id-sub",
		Email:             "id@example.com",
		EmailVerified:     new(true),
		Name:              "ID Name",
		GivenName:         "ID",
		FamilyName:        "Name",
		PreferredUsername: "iduser",
		Picture:           "https://example.com/pic.jpg",
		Locale:            "en",
	}

	idToken := makeTestIDToken(idClaims)

	token := &oauth2.Token{
		AccessToken: "access",
		TokenType:   "Bearer",
	}
	token = token.WithExtra(map[string]any{
		"id_token": idToken,
	})

	parsed, err := parseIDTokenClaims(token)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "id-sub", parsed.Sub)
	require.Equal(t, "id@example.com", parsed.Email)
	require.NotNil(t, parsed.EmailVerified)
	require.True(t, *parsed.EmailVerified)
	require.Equal(t, "ID Name", parsed.Name)
	require.Equal(t, "ID", parsed.GivenName)
	require.Equal(t, "Name", parsed.FamilyName)
	require.Equal(t, "iduser", parsed.PreferredUsername)
	require.Equal(t, "https://example.com/pic.jpg", parsed.Picture)
	require.Equal(t, "en", parsed.Locale)
}

func TestParseIDTokenClaimsMissing(t *testing.T) {
	token := &oauth2.Token{
		AccessToken: "access",
		TokenType:   "Bearer",
	}

	parsed, err := parseIDTokenClaims(token)
	require.NoError(t, err)
	require.Nil(t, parsed)
}

func TestMergeClaims(t *testing.T) {
	idToken := &oidcClaims{
		Sub:       "id-sub",
		Email:     "id@example.com",
		Name:      "ID Name",
		GivenName: "ID",
		Picture:   "https://example.com/id-pic.jpg",
	}
	userinfo := &oidcClaims{
		Sub:               "userinfo-sub",
		Email:             "userinfo@example.com",
		PreferredUsername: "uiuser",
	}

	merged := mergeClaims(idToken, userinfo)
	require.Equal(t, "userinfo-sub", merged.Sub, "userinfo sub should override")
	require.Equal(t, "userinfo@example.com", merged.Email, "userinfo email should override")
	require.Equal(t, "ID Name", merged.Name, "id_token name should be preserved")
	require.Equal(t, "ID", merged.GivenName, "id_token given_name should be preserved")
	require.Equal(t, "uiuser", merged.PreferredUsername, "userinfo preferred_username should override")
	require.Equal(t, "https://example.com/id-pic.jpg", merged.Picture, "id_token picture should be preserved")
}

func TestMergeClaimsNilIDToken(t *testing.T) {
	userinfo := &oidcClaims{Sub: "sub", Email: "e@e.com"}
	merged := mergeClaims(nil, userinfo)
	require.Equal(t, "sub", merged.Sub)
	require.Equal(t, "e@e.com", merged.Email)
}

func TestMergeClaimsNilUserinfo(t *testing.T) {
	idToken := &oidcClaims{Sub: "sub", Name: "name"}
	merged := mergeClaims(idToken, nil)
	require.Equal(t, "sub", merged.Sub)
	require.Equal(t, "name", merged.Name)
}

func TestMergeClaimsEmailVerifiedOverride(t *testing.T) {
	idToken := &oidcClaims{Sub: "sub", EmailVerified: new(true)}
	userinfo := &oidcClaims{Sub: "sub", EmailVerified: new(false)}
	merged := mergeClaims(idToken, userinfo)
	require.NotNil(t, merged.EmailVerified)
	require.False(t, *merged.EmailVerified, "userinfo email_verified=false should override id_token's true")
}

func TestMergeClaimsNoCopy(t *testing.T) {
	userinfo := &oidcClaims{Sub: "sub", Name: "original", EmailVerified: new(true)}
	merged := mergeClaims(nil, userinfo)
	merged.Name = "mutated"
	require.Equal(t, "original", userinfo.Name, "merge must return a copy, not alias the input")
	*merged.EmailVerified = false
	require.True(t, *userinfo.EmailVerified, "merge must deep-copy EmailVerified pointer")
}

func TestMergeClaimsNoCopyBothPresent(t *testing.T) {
	idToken := &oidcClaims{Sub: "sub", EmailVerified: new(true)}
	userinfo := &oidcClaims{Sub: "sub"}
	merged := mergeClaims(idToken, userinfo)
	require.NotNil(t, merged.EmailVerified)
	require.True(t, *merged.EmailVerified)
	*merged.EmailVerified = false
	require.True(t, *idToken.EmailVerified, "merge must deep-copy EmailVerified from idToken when userinfo has nil")
}

func TestParseIDTokenClaimsEmailVerifiedString(t *testing.T) {
	// Some IdPs (AWS Cognito) return email_verified as string "true"
	token := &oauth2.Token{AccessToken: "access", TokenType: "Bearer"}
	rawJWT := jwt.New(jwt.SigningMethodHS256)
	mc := rawJWT.Claims.(jwt.MapClaims)
	mc["sub"] = "string-ev-user"
	mc["email_verified"] = "true"
	signed, _ := rawJWT.SignedString([]byte("key"))
	token = token.WithExtra(map[string]any{"id_token": signed})

	parsed, err := parseIDTokenClaims(token)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.NotNil(t, parsed.EmailVerified)
	require.True(t, *parsed.EmailVerified, "string 'true' should be parsed as bool true")
}

func TestParseIDTokenClaimsEmailVerifiedNumeric(t *testing.T) {
	token := &oauth2.Token{AccessToken: "access", TokenType: "Bearer"}
	rawJWT := jwt.New(jwt.SigningMethodHS256)
	mc := rawJWT.Claims.(jwt.MapClaims)
	mc["sub"] = "numeric-ev-user"
	mc["email_verified"] = float64(1)
	signed, _ := rawJWT.SignedString([]byte("key"))
	token = token.WithExtra(map[string]any{"id_token": signed})

	parsed, err := parseIDTokenClaims(token)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.NotNil(t, parsed.EmailVerified)
	require.True(t, *parsed.EmailVerified, "numeric 1 should be parsed as true")
}

func TestParseIDTokenClaimsMalformed(t *testing.T) {
	token := &oauth2.Token{AccessToken: "access", TokenType: "Bearer"}
	token = token.WithExtra(map[string]any{"id_token": "not.a.valid-jwt"})

	parsed, err := parseIDTokenClaims(token)
	require.Error(t, err)
	require.Nil(t, parsed)
}

func TestOIDCCallbackIDTokenOnly(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	idTokenClaims := oidcClaims{
		Sub:               "idtoken-user",
		Email:             "idtoken@root.gg",
		Name:              "ID Token User",
		PreferredUsername: "idtokenlogin",
	}

	// Userinfo returns only sub (minimal)
	userinfoClaims := oidcClaims{
		Sub: "idtoken-user",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo: userinfoClaims,
		idToken:  &idTokenClaims,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	user, err := ctx.GetMetadataBackend().GetUser("oidc:idtoken-user")
	require.NoError(t, err)
	require.NotNil(t, user, "missing user")
	require.Equal(t, "idtoken@root.gg", user.Email, "email should come from id_token")
	require.Equal(t, "ID Token User", user.Name, "name should come from id_token")
	require.Equal(t, "idtokenlogin", user.Login, "login should come from id_token preferred_username")
}

func TestOIDCCallbackNameFromGivenFamily(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	idTokenClaims := oidcClaims{
		Sub:        "givenfamily-user",
		Email:      "gf@root.gg",
		GivenName:  "Jean",
		FamilyName: "Dupont",
	}

	// Userinfo returns only sub
	userinfoClaims := oidcClaims{
		Sub: "givenfamily-user",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo: userinfoClaims,
		idToken:  &idTokenClaims,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))

	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "handler returned wrong status code")

	user, err := ctx.GetMetadataBackend().GetUser("oidc:givenfamily-user")
	require.NoError(t, err)
	require.NotNil(t, user, "missing user")
	require.Equal(t, "Jean Dupont", user.Name, "name should be synthesized from given_name + family_name")
	require.Equal(t, "gf@root.gg", user.Email)
}

func TestOIDCCallbackRequireVerifiedEmail(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCRequireVerifiedEmail = true

	idTokenClaims := oidcClaims{
		Sub:           "verified-user",
		Email:         "verified@root.gg",
		EmailVerified: new(true),
		Name:          "Verified User",
	}

	userinfoClaims := oidcClaims{Sub: "verified-user"}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo: userinfoClaims,
		idToken:  &idTokenClaims,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "verified email should allow login")

	user, err := ctx.GetMetadataBackend().GetUser("oidc:verified-user")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, "verified@root.gg", user.Email)
}

func TestOIDCCallbackRequireVerifiedEmailFalse(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCRequireVerifiedEmail = true

	idTokenClaims := oidcClaims{
		Sub:           "unverified-user",
		Email:         "unverified@root.gg",
		EmailVerified: new(false),
		Name:          "Unverified User",
	}

	userinfoClaims := oidcClaims{Sub: "unverified-user"}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo: userinfoClaims,
		idToken:  &idTokenClaims,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "email is not verified")
}

func TestOIDCCallbackRequireVerifiedEmailMissing(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCRequireVerifiedEmail = true

	oidcUser := oidcClaims{
		Sub:   "noverify-user",
		Email: "noverify@root.gg",
		Name:  "No Verify User",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "email is not verified")
}

func TestOIDCCallbackRequireVerifiedEmailDisabled(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	// OIDCRequireVerifiedEmail defaults to false

	oidcUser := oidcClaims{
		Sub:   "anyuser",
		Email: "any@root.gg",
		Name:  "Any User",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{userinfo: oidcUser}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "login should succeed when email verification is not required")
}

func TestOIDCCallbackSubMismatch(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())

	idTokenClaims := oidcClaims{
		Sub:   "idtoken-sub",
		Email: "user@root.gg",
		Name:  "User",
	}

	userinfoClaims := oidcClaims{
		Sub:   "different-sub",
		Email: "user@root.gg",
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo: userinfoClaims,
		idToken:  &idTokenClaims,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	context.TestForbidden(t, rr, "OIDC authentication error")
}

func TestMergeClaimsBothNil(t *testing.T) {
	merged := mergeClaims(nil, nil)
	require.NotNil(t, merged)
	require.Equal(t, "", merged.Sub)
	require.Equal(t, "", merged.Email)
	require.Nil(t, merged.EmailVerified)
}

func TestOIDCClaimsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		wantVerified *bool
		wantSub      string
	}{
		{
			name:         "bool true",
			json:         `{"sub":"s","email_verified":true}`,
			wantVerified: new(true),
			wantSub:      "s",
		},
		{
			name:         "bool false",
			json:         `{"sub":"s","email_verified":false}`,
			wantVerified: new(false),
			wantSub:      "s",
		},
		{
			name:         "string true",
			json:         `{"sub":"s","email_verified":"true"}`,
			wantVerified: new(true),
			wantSub:      "s",
		},
		{
			name:         "string True (case insensitive)",
			json:         `{"sub":"s","email_verified":"True"}`,
			wantVerified: new(true),
			wantSub:      "s",
		},
		{
			name:         "string false",
			json:         `{"sub":"s","email_verified":"false"}`,
			wantVerified: new(false),
			wantSub:      "s",
		},
		{
			name:         "numeric 1",
			json:         `{"sub":"s","email_verified":1}`,
			wantVerified: new(true),
			wantSub:      "s",
		},
		{
			name:         "numeric 0",
			json:         `{"sub":"s","email_verified":0}`,
			wantVerified: new(false),
			wantSub:      "s",
		},
		{
			name:         "null",
			json:         `{"sub":"s","email_verified":null}`,
			wantVerified: nil,
			wantSub:      "s",
		},
		{
			name:         "absent",
			json:         `{"sub":"s"}`,
			wantVerified: nil,
			wantSub:      "s",
		},
		{
			name:         "unexpected type (array)",
			json:         `{"sub":"s","email_verified":[]}`,
			wantVerified: nil,
			wantSub:      "s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var claims oidcClaims
			err := json.Unmarshal([]byte(tt.json), &claims)
			require.NoError(t, err)
			require.Equal(t, tt.wantSub, claims.Sub)
			if tt.wantVerified == nil {
				require.Nil(t, claims.EmailVerified)
			} else {
				require.NotNil(t, claims.EmailVerified)
				require.Equal(t, *tt.wantVerified, *claims.EmailVerified)
			}
		})
	}
}

func TestOIDCCallbackEmailVerifiedFromUserinfo(t *testing.T) {
	ResetOIDCDiscoveryCache()
	ctx := newTestingContext(common.NewConfiguration())
	setupOIDCConfig(ctx.GetConfig())
	ctx.GetConfig().OIDCRequireVerifiedEmail = true

	// id_token has NO email_verified
	idTokenClaims := oidcClaims{
		Sub:   "userinfo-ev-user",
		Email: "ev@root.gg",
		Name:  "EV User",
	}

	// email_verified comes exclusively from userinfo
	userinfoClaims := oidcClaims{
		Sub:           "userinfo-ev-user",
		EmailVerified: new(true),
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, oidcMockHandler(oidcMockOptions{
		userinfo: userinfoClaims,
		idToken:  &idTokenClaims,
	}))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	req := oidcCallbackRequest(t, oidcTestState(t, ctx.GetConfig().OIDCClientSecret))
	rr := ctx.NewRecorder(req)
	OIDCCallback(ctx, rr, req)

	require.Equal(t, 302, rr.Code, "login should succeed when email_verified comes from userinfo")

	user, err := ctx.GetMetadataBackend().GetUser("oidc:userinfo-ev-user")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, "ev@root.gg", user.Email)
}

func TestDiscoverOIDCCache(t *testing.T) {
	ResetOIDCDiscoveryCache()

	var fetchCount atomic.Int32
	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/.well-known/openid-configuration" {
			fetchCount.Add(1)
			responseBody, _ := json.Marshal(oidcTestDiscovery)
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(responseBody)
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, shutdown, err := common.StartAPIMockServerCustomPort(common.APIMockServerDefaultPort, http.HandlerFunc(handler))
	defer shutdown()
	require.NoError(t, err, "unable to start mock server")

	providerURL := "http://127.0.0.1:" + strconv.Itoa(common.APIMockServerDefaultPort)

	// First call: cache miss, should fetch
	d1, err := discoverOIDC(providerURL)
	require.NoError(t, err)
	require.NotNil(t, d1)
	require.Equal(t, int32(1), fetchCount.Load(), "expected one fetch on cache miss")

	// Second call: cache hit, no fetch
	d2, err := discoverOIDC(providerURL)
	require.NoError(t, err)
	require.Equal(t, d1, d2)
	require.Equal(t, int32(1), fetchCount.Load(), "expected no additional fetch on cache hit")

	// Expire cache manually
	oidcDiscoveryMu.Lock()
	oidcDiscoveryCache.fetchedAt = time.Now().Add(-2 * oidcDiscoveryCacheTTL)
	oidcDiscoveryMu.Unlock()

	// Third call: stale-while-revalidate returns stale value, triggers background refresh
	d3, err := discoverOIDC(providerURL)
	require.NoError(t, err)
	require.NotNil(t, d3)

	// Wait for background goroutine to complete the refresh
	require.Eventually(t, func() bool {
		return fetchCount.Load() == int32(2)
	}, 5*time.Second, 10*time.Millisecond, "expected background re-fetch after cache expiry")

	// Fourth call: should return fresh cached value without another fetch
	d4, err := discoverOIDC(providerURL)
	require.NoError(t, err)
	require.NotNil(t, d4)
	require.Equal(t, int32(2), fetchCount.Load(), "expected no additional fetch after refresh")
}
