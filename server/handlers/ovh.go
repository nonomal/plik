package handlers

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

type ovhError struct {
	ErrorCode string `json:"errorCode"`
	HTTPCode  string `json:"httpCode"`
	Message   string `json:"message"`
}

type ovhUserConsentResponse struct {
	ValidationURL string `json:"validationUrl"`
	ConsumerKey   string `json:"consumerKey"`
}

type ovhUserResponse struct {
	Nichandle string `json:"nichandle"`
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"name"`
}

// maxOVHResponseSize is the maximum size of an OVH API response body (1MB).
const maxOVHResponseSize = 1 << 20

const ovhHTTPTimeout = 10 * time.Second

var ovhHTTPClient = &http.Client{Timeout: ovhHTTPTimeout}

func decodeOVHResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOVHResponseSize))
	if err != nil {
		return nil, fmt.Errorf("unable to read response body : %s", err)
	}

	if resp.StatusCode > 399 {
		// Decode OVH error information from response
		if len(body) > 0 {
			var ovhErr ovhError
			err := json.Unmarshal(body, &ovhErr)
			if err == nil {
				return nil, fmt.Errorf("%s : %s", resp.Status, ovhErr.Message)
			}
			return nil, fmt.Errorf("%s : %s : %s", resp.Status, "unable to deserialize OVH error", string(body))
		}
		return nil, fmt.Errorf("%s", resp.Status)
	}

	return body, nil
}

// checkOvhAuth validates that OVH authentication is properly configured.
// Returns false if an error response has been written and the caller should return.
func checkOvhAuth(ctx *context.Context) bool {
	config := ctx.GetConfig()

	if config.FeatureAuthentication == common.FeatureDisabled {
		ctx.BadRequest("authentication is disabled")
		return false
	}

	if !config.OvhAuthentication {
		ctx.BadRequest("OVH authentication is disabled")
		return false
	}

	if config.OvhAPIKey == "" || config.OvhAPISecret == "" || config.OvhAPIEndpoint == "" {
		ctx.InternalServerError("missing OVH API credentials", nil)
		return false
	}

	return true
}

// OvhLogin return OVH api user consent URL.
func OvhLogin(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	if !checkOvhAuth(ctx) {
		return
	}

	config := ctx.GetConfig()

	// Get redirection URL from the PlikDomain or referrer header
	redirectURL, err := getRedirectURL(ctx, "/auth/ovh/callback")
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	// Prepare auth request
	ovhReqPayload := struct {
		AccessRules []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"accessRules"`
		Redirection string `json:"redirection"`
	}{
		AccessRules: []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		}{{Method: "GET", Path: "/me"}},
		Redirection: redirectURL,
	}
	ovhReqBodyBytes, err := json.Marshal(ovhReqPayload)
	if err != nil {
		ctx.InternalServerError("unable to marshal OVH request body", err)
		return
	}
	u := fmt.Sprintf("%s/auth/credential", config.OvhAPIEndpoint)

	ovhReq, err := http.NewRequest("POST", u, strings.NewReader(string(ovhReqBodyBytes)))
	if err != nil {
		ctx.InvalidParameter("unable to create POST request to %s : %s", u, err)
		return
	}
	ovhReq.Header.Add("X-Ovh-Application", config.OvhAPIKey)
	ovhReq.Header.Add("Content-type", "application/json")

	// Do request
	ovhResp, err := ovhHTTPClient.Do(ovhReq)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", u), err)
		return
	}
	defer ovhResp.Body.Close()
	ovhRespBody, err := decodeOVHResponse(ovhResp)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", u), err)
		return
	}

	var userConsentResponse ovhUserConsentResponse
	err = json.Unmarshal(ovhRespBody, &userConsentResponse)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", u), err)
		return
	}

	// Generate session jwt
	claims := jwt.MapClaims{
		"ovh-consumer-key": userConsentResponse.ConsumerKey,
		"ovh-api-endpoint": config.OvhAPIEndpoint,
		"expire":           time.Now().Add(5 * time.Minute).Unix(),
	}
	session := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	sessionString, err := session.SignedString([]byte(config.OvhAPISecret))
	if err != nil {
		ctx.InternalServerError("unable to sign OVH session cookie", err)
		return
	}

	// Store temporary session jwt in secure cookie
	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.SameSite = http.SameSiteLaxMode
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = sessionString
	ovhAuthCookie.MaxAge = 300 // 5 minutes
	ovhAuthCookie.Path = "/"
	http.SetCookie(resp, ovhAuthCookie)

	_, _ = resp.Write([]byte(userConsentResponse.ValidationURL))
}

// Remove temporary session cookie
func cleanOvhAuthSessionCookie(resp http.ResponseWriter) {
	ovhAuthCookie := &http.Cookie{}
	ovhAuthCookie.HttpOnly = true
	ovhAuthCookie.Secure = true
	ovhAuthCookie.SameSite = http.SameSiteLaxMode
	ovhAuthCookie.Name = "plik-ovh-session"
	ovhAuthCookie.Value = ""
	ovhAuthCookie.MaxAge = -1
	ovhAuthCookie.Path = "/"
	http.SetCookie(resp, ovhAuthCookie)
}

// OvhCallback authenticate OVH user.
func OvhCallback(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Remove temporary OVH auth session cookie
	cleanOvhAuthSessionCookie(resp)

	if !checkOvhAuth(ctx) {
		return
	}

	config := ctx.GetConfig()

	// Get state from secure cookie
	ovhSessionCookie, err := req.Cookie("plik-ovh-session")
	if err != nil || ovhSessionCookie == nil {
		ctx.MissingParameter("OVH session cookie")
		return
	}

	// Parse session cookie
	ovhAuthCookie, err := jwt.Parse(ovhSessionCookie.Value, func(t *jwt.Token) (any, error) {
		// Verify signing algorithm
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method : %v", t.Header["alg"])
		}

		// Verify expiration date
		if expire, ok := t.Claims.(jwt.MapClaims)["expire"]; ok {
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

		return []byte(config.OvhAPISecret), nil
	})
	if err != nil {
		ctx.InvalidParameter("OVH session cookie : %s", err)
		return
	}

	// Get OVH consumer key from session
	ovhConsumerKey, ok := ovhAuthCookie.Claims.(jwt.MapClaims)["ovh-consumer-key"]
	if !ok {
		ctx.InvalidParameter("OVH session cookie : missing ovh-consumer-key")

		return
	}

	// Get OVH API endpoint
	endpoint, ok := ovhAuthCookie.Claims.(jwt.MapClaims)["ovh-api-endpoint"]
	if !ok {
		ctx.InvalidParameter("OVH session cookie : missing ovh-api-endpoint")
		return
	}

	// Prepare OVH API /me request
	url := endpoint.(string) + "/me"
	ovhReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}

	timestamp := time.Now().Unix()
	ovhReq.Header.Add("X-Ovh-Application", config.OvhAPIKey)
	ovhReq.Header.Add("X-Ovh-Timestamp", fmt.Sprintf("%d", timestamp))
	ovhReq.Header.Add("X-Ovh-Consumer", ovhConsumerKey.(string))

	// Sign request
	h := sha1.New()
	h.Write(fmt.Appendf(nil, "%s+%s+%s+%s+%s+%d",
		config.OvhAPISecret,
		ovhConsumerKey.(string),
		"GET",
		url,
		"",
		timestamp,
	))
	ovhReq.Header.Add("X-Ovh-Signature", fmt.Sprintf("$1$%x", h.Sum(nil)))

	// Do request
	ovhResp, err := ovhHTTPClient.Do(ovhReq)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}
	defer ovhResp.Body.Close()
	ovhRespBody, err := decodeOVHResponse(ovhResp)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}

	// deserialize response
	var userInfo ovhUserResponse
	err = json.Unmarshal(ovhRespBody, &userInfo)
	if err != nil {
		ctx.InternalServerError(fmt.Sprintf("error with OVH API %s", url), err)
		return
	}

	// Get user from metadata backend
	user, err := ctx.GetMetadataBackend().GetUser(common.GetUserID(common.ProviderOVH, userInfo.Nichandle))
	if err != nil {
		ctx.InternalServerError("unable to get user from metadata backend", err)
		return
	}

	if user == nil {
		if ctx.IsWhitelisted() {
			// Create new user
			user = common.NewUser(common.ProviderOVH, userInfo.Nichandle)
			user.Login = userInfo.Nichandle
			user.Name = userInfo.FirstName + " " + userInfo.LastName
			user.Email = userInfo.Email

			// Save user to metadata backend
			err = ctx.GetMetadataBackend().CreateUser(user)
			if err != nil {
				ctx.InternalServerError("unable to create user in metadata backend", err)
				return
			}
		} else {
			ctx.Forbidden("unable to create user from untrusted source IP address")
			return
		}
	} else {
		// Update existing user fields if changed
		updated := false
		name := userInfo.FirstName + " " + userInfo.LastName
		if name != " " && user.Name != name {
			user.Name = name
			updated = true
		}
		if userInfo.Email != "" && user.Email != userInfo.Email {
			user.Email = userInfo.Email
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
