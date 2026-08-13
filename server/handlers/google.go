package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	api_oauth2 "google.golang.org/api/oauth2/v2"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

var googleEndpointContextKey = "google_endpoint"

// checkGoogleAuth validates that Google authentication is properly configured.
// Returns false if an error response has been written and the caller should return.
func checkGoogleAuth(ctx *context.Context) bool {
	config := ctx.GetConfig()

	if config.FeatureAuthentication == common.FeatureDisabled {
		ctx.BadRequest("authentication is disabled")
		return false
	}

	if !config.GoogleAuthentication {
		ctx.BadRequest("Google authentication is disabled")
		return false
	}

	if config.GoogleAPIClientID == "" || config.GoogleAPISecret == "" {
		ctx.InternalServerError("missing Google API credentials", nil)
		return false
	}

	return true
}

// GoogleLogin return google api user consent URL.
func GoogleLogin(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !checkGoogleAuth(ctx) {
		return
	}

	config := ctx.GetConfig()

	// Get redirection URL from the PlikDomain or referrer header
	redirectURL, err := getRedirectURL(ctx, "/auth/google/callback")
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	conf := &oauth2.Config{
		ClientID:     config.GoogleAPIClientID,
		ClientSecret: config.GoogleAPISecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
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
	b64state, err := state.SignedString([]byte(config.GoogleAPISecret))
	if err != nil {
		ctx.InternalServerError("unable to sign state", err)
		return
	}

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL(b64state, oauth2.S256ChallengeOption(verifier))

	_, _ = resp.Write([]byte(url))
}

// GoogleCallback authenticate google user.
func GoogleCallback(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !checkGoogleAuth(ctx) {
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

		return []byte(config.GoogleAPISecret), nil
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
	if err != nil || !strings.HasSuffix(parsedRedirectURL.Path, "/auth/google/callback") {
		ctx.InvalidParameter("oauth2 state : invalid redirectURL")
		return
	}

	conf := &oauth2.Config{
		ClientID:     config.GoogleAPIClientID,
		ClientSecret: config.GoogleAPISecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			api_oauth2.UserinfoEmailScope,
			api_oauth2.UserinfoProfileScope,
		},
		Endpoint: google.Endpoint,
	}

	// For testing purpose
	if customEndpoint := req.Context().Value(googleEndpointContextKey); customEndpoint != nil {
		conf.Endpoint = customEndpoint.(oauth2.Endpoint)
	}

	var exchangeOpts []oauth2.AuthCodeOption
	if pkceVerifier != "" {
		exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(pkceVerifier))
	}
	token, err := conf.Exchange(req.Context(), code, exchangeOpts...)
	if err != nil {
		ctx.InternalServerError("unable to get user info from Google API (1)", err)
		return
	}

	client, err := api_oauth2.New(conf.Client(req.Context(), token))
	if err != nil {
		ctx.InternalServerError("unable to get user info from Google API (2)", err)
		return
	}

	// For testing purpose
	if customEndpoint := req.Context().Value(googleEndpointContextKey); customEndpoint != nil {
		client.BasePath = customEndpoint.(oauth2.Endpoint).AuthURL
	}

	userInfo, err := client.Userinfo.Get().Do()
	if err != nil {
		ctx.InternalServerError("unable to get user info from Google API (3)", err)
		return
	}

	// Intentional: validate domain on every login (not just creation)
	// to revoke access when allowed domains change
	if len(config.GoogleValidDomains) > 0 {
		components := strings.Split(userInfo.Email, "@")
		if len(components) != 2 {
			ctx.Forbidden("invalid email address")
			return
		}
		goodDomain := false
		for _, validDomain := range config.GoogleValidDomains {
			if strings.EqualFold(components[1], validDomain) {
				goodDomain = true
				break
			}
		}
		if !goodDomain {
			ctx.Forbidden("unauthorized domain name")
			return
		}
	}

	// Get user from metadata backend
	user, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderGoogle, userInfo.Email))
	if err != nil {
		ctx.InternalServerError("unable to get user from metadata backend", err)
		return
	}

	if user == nil {
		if ctx.IsWhitelisted() {
			// Create new user
			user = common.NewUser(common.ProviderGoogle, userInfo.Email)
			user.Login = userInfo.Email
			user.Name = userInfo.Name
			user.Email = userInfo.Email
			user.ProfilePicture = userInfo.Picture

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
		if userInfo.Name != "" && user.Name != userInfo.Name {
			user.Name = userInfo.Name
			updated = true
		}
		if userInfo.Email != "" && user.Email != userInfo.Email {
			user.Email = userInfo.Email
			updated = true
		}
		if userInfo.Picture != "" && user.ProfilePicture != userInfo.Picture {
			user.ProfilePicture = userInfo.Picture
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
