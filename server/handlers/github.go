package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

var githubEndpointContextKey = "github_endpoint"
var githubUserinfoContextKey = "github_userinfo_endpoint"
var githubOrgsContextKey = "github_orgs_endpoint"

var githubDefaultEndpoint = oauth2.Endpoint{
	AuthURL:  "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
}

const githubDefaultUserinfoEndpoint = "https://api.github.com/user"
const githubDefaultOrgsEndpoint = "https://api.github.com/user/orgs"
const githubHTTPTimeout = 10 * time.Second
const githubMaxResponseSize = 1 << 20 // 1MB

var githubHTTPClient = &http.Client{Timeout: githubHTTPTimeout}

type githubUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubOrg struct {
	Login string `json:"login"`
}

// checkGitHubAuth validates that GitHub authentication is properly configured.
// Returns false if an error response has been written and the caller should return.
func checkGitHubAuth(ctx *context.Context) bool {
	config := ctx.GetConfig()

	if config.FeatureAuthentication == common.FeatureDisabled {
		ctx.BadRequest("authentication is disabled")
		return false
	}

	if !config.GitHubAuthentication {
		ctx.BadRequest("GitHub authentication is disabled")
		return false
	}

	if config.GitHubAPIClientID == "" || config.GitHubAPISecret == "" {
		ctx.InternalServerError("missing GitHub API credentials", nil)
		return false
	}

	return true
}

// GitHubLogin return GitHub OAuth2 user consent URL.
func GitHubLogin(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !checkGitHubAuth(ctx) {
		return
	}

	config := ctx.GetConfig()

	// Get redirection URL from the PlikDomain or referrer header
	redirectURL, err := getRedirectURL(ctx, "/auth/github/callback")
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	scopes := []string{"read:user", "user:email"}
	if len(config.GitHubValidOrganizations) > 0 {
		scopes = append(scopes, "read:org")
	}

	endpoint := githubDefaultEndpoint
	// For testing purpose
	if customEndpoint := req.Context().Value(githubEndpointContextKey); customEndpoint != nil {
		endpoint = customEndpoint.(oauth2.Endpoint)
	}

	conf := &oauth2.Config{
		ClientID:     config.GitHubAPIClientID,
		ClientSecret: config.GitHubAPISecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	verifier := oauth2.GenerateVerifier()

	/* Generate state */
	claims := jwt.MapClaims{
		"redirectURL":  redirectURL,
		"expire":       time.Now().Add(time.Minute * 5).Unix(),
		"pkceVerifier": verifier,
	}
	state := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	/* Sign state */
	b64state, err := state.SignedString([]byte(config.GitHubAPISecret))
	if err != nil {
		ctx.InternalServerError("unable to sign state", err)
		return
	}

	// Redirect user to GitHub's consent page
	url := conf.AuthCodeURL(b64state, oauth2.S256ChallengeOption(verifier))

	_, _ = resp.Write([]byte(url))
}

// GitHubCallback authenticate GitHub user.
func GitHubCallback(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !checkGitHubAuth(ctx) {
		return
	}

	config := ctx.GetConfig()

	code := req.URL.Query().Get("code")
	if code == "" {
		ctx.MissingParameter("oauth2 authorization code")
		return
	}

	b64state := req.URL.Query().Get("state")
	if b64state == "" {
		ctx.MissingParameter("oauth2 authorization state")
		return
	}

	/* Parse state */
	state, err := jwt.Parse(b64state, func(token *jwt.Token) (any, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method : %v", token.Header["alg"])
		}

		// Verify expiration data
		if expire, ok := token.Claims.(jwt.MapClaims)["expire"]; ok {
			if _, ok = expire.(float64); ok {
				if time.Now().Unix() > (int64)(expire.(float64)) {
					return nil, fmt.Errorf("state has expired")
				}
			} else {
				return nil, fmt.Errorf("invalid expiration date")
			}
		} else {
			return nil, fmt.Errorf("missing expiration date")
		}

		return []byte(config.GitHubAPISecret), nil
	})
	if err != nil {
		ctx.InvalidParameter("oauth2 state : %s", err)
		return
	}

	if _, ok := state.Claims.(jwt.MapClaims)["redirectURL"]; !ok {
		ctx.InvalidParameter("oauth2 state : missing redirectURL")
		return
	}

	if _, ok := state.Claims.(jwt.MapClaims)["redirectURL"].(string); !ok {
		ctx.InvalidParameter("oauth2 state : invalid redirectURL")
		return
	}

	redirectURL := state.Claims.(jwt.MapClaims)["redirectURL"].(string)
	pkceVerifier, _ := state.Claims.(jwt.MapClaims)["pkceVerifier"].(string)

	parsedRedirectURL, err := url.Parse(redirectURL)
	if err != nil || !strings.HasSuffix(parsedRedirectURL.Path, "/auth/github/callback") {
		ctx.InvalidParameter("oauth2 state : invalid redirectURL")
		return
	}

	scopes := []string{"read:user", "user:email"}
	if len(config.GitHubValidOrganizations) > 0 {
		scopes = append(scopes, "read:org")
	}

	endpoint := githubDefaultEndpoint
	// For testing purpose
	if customEndpoint := req.Context().Value(githubEndpointContextKey); customEndpoint != nil {
		endpoint = customEndpoint.(oauth2.Endpoint)
	}

	conf := &oauth2.Config{
		ClientID:     config.GitHubAPIClientID,
		ClientSecret: config.GitHubAPISecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}

	var exchangeOpts []oauth2.AuthCodeOption
	if pkceVerifier != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(pkceVerifier))
	}
	token, err := conf.Exchange(req.Context(), code, exchangeOpts...)
	if err != nil {
		ctx.InternalServerError("unable to exchange GitHub authorization code", err)
		return
	}

	// Fetch user info from GitHub API
	userinfoEndpoint := githubDefaultUserinfoEndpoint
	if customUserinfo := req.Context().Value(githubUserinfoContextKey); customUserinfo != nil {
		userinfoEndpoint = customUserinfo.(string)
	}

	userReq, err := http.NewRequestWithContext(req.Context(), "GET", userinfoEndpoint, nil)
	if err != nil {
		ctx.InternalServerError("unable to create GitHub user request", err)
		return
	}
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := githubHTTPClient.Do(userReq)
	if err != nil {
		ctx.InternalServerError("unable to get user info from GitHub API", err)
		return
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		ctx.InternalServerError("unable to get user info from GitHub API", fmt.Errorf("status %d", userResp.StatusCode))
		return
	}

	var ghUser githubUser
	if err := json.NewDecoder(io.LimitReader(userResp.Body, githubMaxResponseSize)).Decode(&ghUser); err != nil {
		ctx.InternalServerError("unable to parse GitHub user info", err)
		return
	}

	if ghUser.Login == "" {
		ctx.InternalServerError("GitHub user info missing login", nil)
		return
	}

	// Check organization membership if configured
	if len(config.GitHubValidOrganizations) > 0 {
		orgsEndpoint := githubDefaultOrgsEndpoint
		if customOrgs := req.Context().Value(githubOrgsContextKey); customOrgs != nil {
			orgsEndpoint = customOrgs.(string)
		}

		orgsReq, err := http.NewRequestWithContext(req.Context(), "GET", orgsEndpoint, nil)
		if err != nil {
			ctx.InternalServerError("unable to create GitHub orgs request", err)
			return
		}
		orgsReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		orgsReq.Header.Set("Accept", "application/vnd.github+json")

		orgsResp, err := githubHTTPClient.Do(orgsReq)
		if err != nil {
			ctx.InternalServerError("unable to get organizations from GitHub API", err)
			return
		}
		defer orgsResp.Body.Close()

		if orgsResp.StatusCode != http.StatusOK {
			ctx.InternalServerError("unable to get organizations from GitHub API", fmt.Errorf("status %d", orgsResp.StatusCode))
			return
		}

		var orgs []githubOrg
		if err := json.NewDecoder(io.LimitReader(orgsResp.Body, githubMaxResponseSize)).Decode(&orgs); err != nil {
			ctx.InternalServerError("unable to parse GitHub organizations", err)
			return
		}

		goodOrg := false
		for _, org := range orgs {
			for _, validOrg := range config.GitHubValidOrganizations {
				if strings.EqualFold(org.Login, validOrg) {
					goodOrg = true
					break
				}
			}
			if goodOrg {
				break
			}
		}
		if !goodOrg {
			ctx.Forbidden("unauthorized organization")
			return
		}
	}

	// Get user from metadata backend
	user, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderGitHub, ghUser.Login))
	if err != nil {
		ctx.InternalServerError("unable to get user from metadata backend", err)
		return
	}

	if user == nil {
		if ctx.IsWhitelisted() {
			// Create new user
			user = common.NewUser(common.ProviderGitHub, ghUser.Login)
			user.Login = ghUser.Login
			user.Name = ghUser.Name
			user.Email = ghUser.Email
			user.ProfilePicture = ghUser.AvatarURL

			// Save user to metadata backend
			err = ctx.GetMetadataBackend().CreateUser(user)
			if err != nil {
				ctx.InternalServerError("unable to create user : %s", err)
				return
			}
		} else {
			ctx.Forbidden("unable to create user from untrusted source IP address")
			return
		}
	} else {
		// Update existing user fields if changed
		updated := false
		if ghUser.Name != "" && user.Name != ghUser.Name {
			user.Name = ghUser.Name
			updated = true
		}
		if ghUser.Email != "" && user.Email != ghUser.Email {
			user.Email = ghUser.Email
			updated = true
		}
		if ghUser.AvatarURL != "" && user.ProfilePicture != ghUser.AvatarURL {
			user.ProfilePicture = ghUser.AvatarURL
			updated = true
		}
		if updated {
			err = ctx.GetMetadataBackend().UpdateUser(user)
			if err != nil {
				ctx.InternalServerError("unable to update user : %s", err)
				return
			}
		}
	}

	// Set Plik session cookie and xsrf cookie
	sessionCookie, xsrfCookie, err := ctx.GetAuthenticator().GenAuthCookies(user)
	if err != nil {
		ctx.InternalServerError("unable to generate session cookies", err)
		return
	}
	http.SetCookie(resp, sessionCookie)
	http.SetCookie(resp, xsrfCookie)

	http.Redirect(resp, req, config.Path+"/#/login", http.StatusFound)
}
