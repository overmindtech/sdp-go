package auth

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nkeys"
)

var tokenExchangeURLs = []string{
	"http://api-server:8080/api",
	"http://localhost:8080/api",
}

func TestBasicTokenClient(t *testing.T) {
	var c TokenClient

	keys, err := nkeys.CreateUser()

	if err != nil {
		t.Fatal(err)
	}

	c = NewBasicTokenClient("tokeny_mc_tokenface", keys)

	var token string

	token, err = c.GetJWT()

	if err != nil {
		t.Error(err)
	}

	if token != "tokeny_mc_tokenface" {
		t.Error("token mismatch")
	}

	data := []byte{1, 156, 230, 4, 23, 175, 11}

	signed, err := c.Sign(data)

	if err != nil {
		t.Fatal(err)
	}

	err = keys.Verify(data, signed)

	if err != nil {
		t.Error(err)
	}
}

func GetTestOAuthTokenClient(t *testing.T) *OAuthTokenClient {
	var domain string
	var clientID string
	var clientSecret string
	var exists bool

	errorFormat := "environment variable %v not found. Set up your test environment first. See: https://github.com/overmindtech/auth0-test-data"

	// Read secrets form the environment
	if domain, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_DOMAIN"); !exists || domain == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_DOMAIN")
		t.Skip("Skipping due to missing environment setup")
	}

	if clientID, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_CLIENT_ID"); !exists || clientID == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_CLIENT_ID")
		t.Skip("Skipping due to missing environment setup")
	}

	if clientSecret, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_CLIENT_SECRET"); !exists || clientSecret == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_CLIENT_SECRET")
		t.Skip("Skipping due to missing environment setup")
	}

	exchangeURL, err := GetWorkingTokenExchange()

	if err != nil {
		t.Fatal(err)
	}

	flowConfig := ClientCredentialsConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Account:      "org_hdeUXbB55sMMvJLa",
	}

	return NewOAuthTokenClient(
		fmt.Sprintf("https://%v/oauth/token", domain),
		exchangeURL,
		flowConfig,
	)
}

func TestOAuthTokenClient(t *testing.T) {
	c := GetTestOAuthTokenClient(t)

	var err error

	_, err = c.GetJWT()

	if err != nil {
		t.Error(err)
	}

	// Make sure it can sign
	data := []byte{1, 156, 230, 4, 23, 175, 11}

	_, err = c.Sign(data)

	if err != nil {
		t.Fatal(err)
	}

}

func GetWorkingTokenExchange() (string, error) {
	var err error
	errMap := make(map[string]error)

	for _, url := range tokenExchangeURLs {
		if err = testURL(url); err == nil {
			return url, nil
		}
		errMap[url] = err
	}

	var errString string

	for url, err := range errMap {
		errString = errString + fmt.Sprintf("  %v: %v\n", url, err.Error())
	}

	return "", fmt.Errorf("no working token exchanges found:\n%v", err)
}

func testURL(testURL string) error {
	url, err := url.Parse(testURL)

	if err != nil {
		return fmt.Errorf("could not parse NATS URL: %v. Error: %v", testURL, err)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(url.Hostname(), url.Port()), time.Second)

	if err == nil {
		conn.Close()
		return nil
	}

	return err
}
