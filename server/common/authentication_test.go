package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionAuthenticator(t *testing.T) {
	setting := GenerateAuthenticationSignatureKey()
	maxAge := 1
	path := "/path"
	sa := &SessionAuthenticator{SignatureKey: setting.Value, SecureCookies: true, SessionTimeout: maxAge, Path: path}

	user := NewUser("local", "user")

	sessionCookie, xsrfCookie, err := sa.GenAuthCookies(user)
	require.NoError(t, err, "unable to generate cookies")
	require.NotNil(t, sessionCookie, "missing session cookie")
	require.NotNil(t, xsrfCookie, "missing xsrf cookie")

	require.NotNil(t, sessionCookie, "missing session cookie")
	require.Equal(t, maxAge, sessionCookie.MaxAge, "invalid session cookie max age")
	require.Equal(t, path, sessionCookie.Path, "invalid session cookie path")
	require.True(t, sessionCookie.Secure, "invalid session cookies not secure")
	require.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite, "invalid session cookie SameSite")

	require.NotNil(t, xsrfCookie, "missing xsrf cookie")
	require.Equal(t, maxAge, xsrfCookie.MaxAge, "invalid xsrf cookie max age")
	require.Equal(t, path, xsrfCookie.Path, "invalid xsrf cookie path")
	require.True(t, xsrfCookie.Secure, "invalid xsrf cookie not secure")
	require.Equal(t, http.SameSiteLaxMode, xsrfCookie.SameSite, "invalid xsrf cookie SameSite")

	uid, xsrf, err := sa.ParseSessionCookie(sessionCookie.Value)
	require.NoError(t, err, "unable to parse session cookie")
	require.Equal(t, user.ID, uid, "invalid user id")
	require.Equal(t, xsrfCookie.Value, xsrf, "invalid xsrf token")

	time.Sleep(time.Second)
	uid, xsrf, err = sa.ParseSessionCookie(sessionCookie.Value)
	require.Error(t, err, "session timeout")
}

func TestLogout(t *testing.T) {
	path := "/path"

	rr := httptest.NewRecorder()
	Logout(rr, &SessionAuthenticator{SecureCookies: true, Path: path})
	require.Equal(t, 2, len(rr.Result().Cookies()), "missing response cookies")

	var sessionCookie *http.Cookie
	var xsrfCookie *http.Cookie

	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == SessionCookieName {
			sessionCookie = cookie
		}
		if cookie.Name == XSRFCookieName {
			xsrfCookie = cookie
		}
	}

	require.NotNil(t, sessionCookie, "missing session cookie")
	require.Equal(t, -1, sessionCookie.MaxAge, "invalid session cookie")
	require.Equal(t, path, sessionCookie.Path, "invalid session cookie path")
	require.True(t, sessionCookie.Secure, "invalid session cookie not secure")
	require.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite, "invalid session cookie SameSite")

	require.NotNil(t, xsrfCookie, "missing xsrf cookie")
	require.Equal(t, -1, xsrfCookie.MaxAge, "invalid xsrf cookie")
	require.Equal(t, path, xsrfCookie.Path, "invalid xsrf cookie path")
	require.True(t, xsrfCookie.Secure, "invalid xsrf cookie not secure")
	require.Equal(t, http.SameSiteLaxMode, xsrfCookie.SameSite, "invalid xsrf cookie SameSite")
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password")
	require.NoError(t, err, "hash password error")

	ok := CheckPasswordHash("password", hash)
	require.True(t, ok)

	ok = CheckPasswordHash("invalid", hash)
	require.False(t, ok)
}

func TestHashPasswordTooLong(t *testing.T) {
	// 72 bytes should work
	password72 := strings.Repeat("a", 72)
	_, err := HashPassword(password72)
	require.NoError(t, err, "72 byte password should be accepted")

	// 73 bytes should fail
	password73 := password72 + "a"
	_, err = HashPassword(password73)
	require.Error(t, err, "73 byte password should be rejected")
	require.Contains(t, err.Error(), "too long")
}
