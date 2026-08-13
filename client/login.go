package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/root-gg/plik/plik"
)

type cliAuthInitRequest struct {
	Hostname string `json:"hostname"`
}

type cliAuthInitResponse struct {
	Code      string `json:"code"`
	Secret    string `json:"secret"`
	VerifyURL string `json:"verifyURL"`
	ExpiresIn int    `json:"expiresIn"`
}

type cliAuthPollRequest struct {
	Code   string `json:"code"`
	Secret string `json:"secret"`
}

type cliAuthPollResponse struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"`
}

// login performs the device authorization flow.
// It initiates a session on the server, displays a URL for the user to
// authenticate in their browser, polls for approval, and saves the token.
func login(cfg *CliConfig, client *plik.Client) error {
	// Get hostname for token comment
	hostname, _ := os.Hostname()

	// Step 1: Initiate the CLI auth session
	initReq := cliAuthInitRequest{Hostname: hostname}
	initBody, err := json.Marshal(initReq)
	if err != nil {
		return fmt.Errorf("unable to serialize init request: %s", err)
	}

	resp, err := client.HTTPClient.Post(cfg.URL+"/auth/cli/init", "application/json", bytes.NewBuffer(initBody))
	if err != nil {
		return fmt.Errorf("unable to contact server: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response: %s", err)
	}

	var initResp cliAuthInitResponse
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return fmt.Errorf("unable to parse response: %s", err)
	}

	// Step 2: Build verify URL from client config (not server response)
	// The server's VerifyURL may use its internal address (e.g. 127.0.0.1:8080)
	verifyURL := fmt.Sprintf("%s/#/cli-auth?code=%s", cfg.URL, initResp.Code)
	if hostname != "" {
		verifyURL += "&hostname=" + url.QueryEscape(hostname)
	}

	fmt.Printf("\n  Open this URL in your browser to authenticate:\n\n")
	fmt.Printf("    %s\n\n", verifyURL)
	fmt.Printf("  Your one-time code: %s\n\n", initResp.Code)
	fmt.Printf("  Waiting for authentication...")

	// Best-effort: try to open the browser
	openBrowser(verifyURL)

	// Step 3: Poll for approval
	timeout := time.Duration(initResp.ExpiresIn) * time.Second
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		token, err := pollForToken(client, cfg.URL, initResp.Code, initResp.Secret)
		if err != nil {
			return err
		}

		if token != "" {
			fmt.Printf("\n\n")

			// Step 4: Save token to config
			err := saveToken(cfg, token)
			if err != nil {
				// Still print the token so the user isn't locked out
				fmt.Printf("  Warning: unable to save token to config: %s\n", err)
				fmt.Printf("  Your token is: %s\n", token)
				fmt.Printf("  Add it to your .plikrc manually.\n")
				return nil
			}

			fmt.Printf("  ✓ Authenticated! Token saved to ~/.plikrc\n")
			fmt.Printf("  Token: %s\n\n", token)
			return nil
		}
	}

	fmt.Printf("\n\n")
	return fmt.Errorf("authentication timed out after %s — please try again", timeout)
}

func pollForToken(client *plik.Client, serverURL, code, secret string) (string, error) {
	pollReq := cliAuthPollRequest{Code: code, Secret: secret}
	pollBody, err := json.Marshal(pollReq)
	if err != nil {
		return "", fmt.Errorf("unable to serialize poll request: %s", err)
	}

	resp, err := client.HTTPClient.Post(serverURL+"/auth/cli/poll", "application/json", bytes.NewBuffer(pollBody))
	if err != nil {
		// Transient network error — keep polling
		return "", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("poll error (%d): %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	var pollResp cliAuthPollResponse
	if err := json.Unmarshal(respBody, &pollResp); err != nil {
		return "", nil
	}

	if pollResp.Status == "approved" && pollResp.Token != "" {
		return pollResp.Token, nil
	}

	return "", nil
}

// saveToken performs a surgical edit of ~/.plikrc, updating only the Token
// value in the correct section (top-level or a named profile). All other
// content — comments, whitespace, profile ordering — is preserved verbatim.
func saveToken(cfg *CliConfig, token string) error {
	path := cfg.ConfigPath
	if path == "" {
		path = configFilePath()
	}

	profileName, err := cfg.SingleProfile()
	if err != nil {
		return fmt.Errorf("--login: %s", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read config file: %s", err)
	}

	patched, err := patchToken(data, profileName, token)
	if err != nil {
		return err
	}

	return os.WriteFile(path, patched, 0600)
}

// tokenLineRe matches a TOML Token assignment, regardless of whitespace or
// inline comments:  Token = "value"  # optional comment
var tokenLineRe = regexp.MustCompile(`^(\s*)Token\s*=\s*"[^"]*"(.*)$`)

// sectionHeaderRe matches a TOML table header like [Profiles.work] or [ArchiveOptions].
var sectionHeaderRe = regexp.MustCompile(`^\s*\[`)

// urlLineRe matches a TOML URL assignment like:  URL = "https://..."
var urlLineRe = regexp.MustCompile(`^\s*URL\s*=`)

// patchToken performs an in-place edit of the raw .plikrc bytes, updating only
// the Token value. When profileName is empty, it patches the top-level Token.
// Otherwise it patches Token inside [Profiles.<profileName>].
//
// If a Token line already exists in the target section, its value is replaced
// in-place (preserving any trailing inline comment). If no Token line exists,
// a new one is inserted after the URL line (if present), otherwise right after
// the section header.
func patchToken(data []byte, profileName string, token string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")

	if profileName == "" {
		return patchTopLevelToken(lines, token)
	}
	return patchProfileToken(lines, profileName, token)
}

// patchTopLevelToken patches Token in the top-level section (before the first
// TOML table header).
func patchTopLevelToken(lines []string, token string) ([]byte, error) {
	// Find the boundary: first line that starts a [Table] section.
	firstSection := len(lines)
	for i, line := range lines {
		if sectionHeaderRe.MatchString(line) {
			firstSection = i
			break
		}
	}

	// Look for an existing Token line in the top-level section.
	for i := 0; i < firstSection; i++ {
		if tokenLineRe.MatchString(lines[i]) {
			lines[i] = replaceTokenValue(lines[i], token)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}

	// No Token line found — insert one. Place it after URL if present,
	// otherwise at the start of the file.
	insertAt := 0
	for i := 0; i < firstSection; i++ {
		if urlLineRe.MatchString(lines[i]) {
			insertAt = i + 1
			break
		}
	}

	newLine := fmt.Sprintf("Token = %q", token)
	lines = insertLine(lines, insertAt, newLine)
	return []byte(strings.Join(lines, "\n")), nil
}

// patchProfileToken patches Token inside [Profiles.<name>].
func patchProfileToken(lines []string, profileName string, token string) ([]byte, error) {
	header := fmt.Sprintf("[Profiles.%s]", profileName)

	// Find the profile section header.
	headerIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == header {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		return nil, fmt.Errorf("profile %q not found in config", profileName)
	}

	// Find the end of this profile section (next [Table] header or EOF).
	sectionEnd := len(lines)
	for i := headerIdx + 1; i < len(lines); i++ {
		if sectionHeaderRe.MatchString(lines[i]) {
			sectionEnd = i
			break
		}
	}

	// Look for an existing Token line within this section.
	for i := headerIdx + 1; i < sectionEnd; i++ {
		if tokenLineRe.MatchString(lines[i]) {
			lines[i] = replaceTokenValue(lines[i], token)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}

	// No Token line — insert one after URL if present, otherwise right
	// after the section header.
	insertAt := headerIdx + 1
	for i := headerIdx + 1; i < sectionEnd; i++ {
		if urlLineRe.MatchString(lines[i]) {
			insertAt = i + 1
			break
		}
	}

	newLine := fmt.Sprintf("Token = %q", token)
	lines = insertLine(lines, insertAt, newLine)
	return []byte(strings.Join(lines, "\n")), nil
}

// replaceTokenValue replaces the value in a Token = "..." line, preserving
// leading whitespace and any trailing content (inline comments, etc.).
func replaceTokenValue(line string, token string) string {
	m := tokenLineRe.FindStringSubmatch(line)
	if m == nil {
		return line // shouldn't happen — caller verified match
	}
	return fmt.Sprintf("%sToken = %q%s", m[1], token, m[2])
}

// insertLine inserts a new line at the given index in a slice of lines.
func insertLine(lines []string, at int, newLine string) []string {
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:at]...)
	result = append(result, newLine)
	result = append(result, lines[at:]...)
	return result
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	// Best-effort — ignore errors
	_ = cmd.Start()
}
