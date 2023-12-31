// Library for authenticating with Twitch using OAuth2 and the client credentials grant flow
// https://dev.twitch.tv/docs/authentication/getting-tokens-oauth#oauth-client-credentials-flow
package twitchauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Constants for the Twitch API
const (
	// Twitch API URL
	twitchAuthTokenURL = "https://id.twitch.tv/oauth2/token"
	// Regex for Twitch OAuth token
	twitchAuthTokenRegex = `[a-zA-Z0-9]{30}`
)

// TwitchAuth is the struct for the Twitch API
type TwitchAuth struct {
	ClientID       string
	Secret         string
	ExpirationTime time.Time // Time Token was received in time.Time
	Token          token
}

// token is the response from the Twitch API
type token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type TwitchAuthInterface interface {
	NewTokenSet()
	Isexpired()
	TimeTillExpiration()
	String()
}

// Returns Token
func (self *TwitchAuth) GetToken() string {
	if self.Token.AccessToken == "" {
		return ""
	}
	return self.Token.AccessToken
}

// Returns Token Information as string.
func (self *TwitchAuth) String() string {
	return fmt.Sprintf("Token Expired: %v\nExpiration %v\n",
		// Print expiration time in local time
		self.Isexpired(),
		self.TimeTillExpiration())
}

// Returns duration until token expires
func (self *TwitchAuth) TimeTillExpiration() time.Duration {
	return self.ExpirationTime.Sub(time.Now())
}

// returns true if the token is expired
func (self *TwitchAuth) Isexpired() bool {
	return !self.ExpirationTime.After(time.Now())
}

// Obtains a new Token set from the Twitch API
// Token set includes access token, Type, expiration time
func (self *TwitchAuth) NewTokenSet() error {
	re, err := regexp.Compile(twitchAuthTokenRegex)
	if err != nil {
		return fmt.Errorf("Error compiling regex: %v with '%s'", err, twitchAuthTokenRegex)
	}
	var t token
	// Client credentials grant flow
	// https://dev.twitch.tv/docs/authentication/getting-tokens-oauth#oauth-client-credentials-flow
	data := url.Values{}
	data.Set("client_id", self.ClientID)
	data.Set("client_secret", self.Secret)
	data.Set("grant_type", "client_credentials")
	req, err := http.NewRequest("POST", twitchAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("Error creating new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send Request to Twitch API
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error getting token set: %v", err)
	}

	// Close the body when done reading from it
	defer resp.Body.Close()

	// Read response body into a byte slice
	b := make([]byte, resp.ContentLength)
	resp.Body.Read(b)

	// Decode the JSON response into the token struct
	// Return error on failure.
	if err := json.Unmarshal(b, &t); err != nil {
		return fmt.Errorf("Error Decoding Json (%v) response Body: %v", err, string(b))
	}

	// Validate the token,return error if there is AccessToken is blank, or does not match regex
	if t.AccessToken == "" || !re.MatchString(t.AccessToken) {
		return fmt.Errorf("invalid token received %v Response Body: %v", t.AccessToken, string(b))
	}

	// Set the token, and the time that it will expire.
	self.Token = t
	self.ExpirationTime = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
	return nil
}
