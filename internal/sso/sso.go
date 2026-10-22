// Package sso signs people in with an identity provider (OpenID Connect,
// authorization code flow with PKCE), next to passwords: Google for now.
//
// The ID token comes straight from the provider's token endpoint, over TLS,
// in exchange for the code: as OpenID Connect allows in that case, its
// signature is not checked, its claims are (issuer, audience, expiry,
// nonce).
package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Provider is an OpenID Connect identity provider.
type Provider struct {
	ID       string
	Label    string
	AuthURL  string
	TokenURL string
	// Issuers are the accepted "iss" values of ID tokens.
	Issuers      []string
	ClientID     string
	ClientSecret string
	// AllowedDomains: people of these domains get an account on their
	// first sign-in.
	AllowedDomains []string
	// HostedDomain: a domain is the person's only when the provider says
	// so in the "hd" claim (Google Workspace), not from the email alone.
	HostedDomain bool
	DefaultRole  string
	// HTTPClient makes the token request (default http.DefaultClient).
	HTTPClient *http.Client
}

// Google returns the Google provider for an OAuth client.
func Google(clientID, clientSecret string) *Provider {
	return &Provider{
		ID: "google", Label: "Google",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Issuers:      []string{"https://accounts.google.com", "accounts.google.com"},
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HostedDomain: true,
	}
}

var providers = map[string]*Provider{}

// Configure sets the enabled providers (call at startup).
func Configure(list ...*Provider) {
	m := map[string]*Provider{}
	for _, p := range list {
		if p != nil && p.ClientID != "" {
			m[p.ID] = p
		}
	}
	providers = m
}

// Get returns an enabled provider.
func Get(id string) (*Provider, bool) {
	p, ok := providers[id]
	return p, ok
}

// Info is a provider as the login page shows it.
type Info struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Enabled lists the enabled providers.
func Enabled() []Info {
	out := []Info{}
	for _, p := range providers {
		out = append(out, Info{ID: p.ID, Label: p.Label})
	}
	slices.SortFunc(out, func(a, b Info) int { return strings.Compare(a.ID, b.ID) })
	return out
}

// RandomString returns n random bytes, base64url-encoded.
func RandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// challenge is the PKCE S256 challenge of a verifier.
func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// AuthCodeURL is where to send the browser to sign in.
func (p *Provider) AuthCodeURL(redirectURI, state, nonce, verifier string) string {
	q := url.Values{
		"client_id":             {p.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {challenge(verifier)},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	sep := "?"
	if strings.Contains(p.AuthURL, "?") {
		sep = "&"
	}
	return p.AuthURL + sep + q.Encode()
}

// Identity is who signed in.
type Identity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	// HostedDomain is the Google Workspace domain of the account, if any.
	HostedDomain string
}

// Domain returns the domain of the email address.
func (id Identity) Domain() string {
	_, d, _ := strings.Cut(id.Email, "@")
	return strings.ToLower(d)
}

// ErrSignIn is a sign-in the provider or its token refused.
var ErrSignIn = errors.New("sign-in failed")

type idClaims struct {
	Iss           string          `json:"iss"`
	Sub           string          `json:"sub"`
	Aud           json.RawMessage `json:"aud"`
	Exp           int64           `json:"exp"`
	Nonce         string          `json:"nonce"`
	Email         string          `json:"email"`
	EmailVerified any             `json:"email_verified"`
	Name          string          `json:"name"`
	HD            string          `json:"hd"`
}

func (c idClaims) audiences() []string {
	var one string
	if json.Unmarshal(c.Aud, &one) == nil {
		return []string{one}
	}
	var many []string
	_ = json.Unmarshal(c.Aud, &many)
	return many
}

func (c idClaims) verified() bool {
	switch v := c.EmailVerified.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

// Exchange trades the code for the person's identity, checking the ID
// token's issuer, audience, expiry and nonce.
func (p *Provider) Exchange(ctx context.Context, redirectURI, code, verifier, nonce string) (*Identity, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
		"code_verifier": {verifier},
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: could not reach %s: %v", ErrSignIn, p.Label, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var tok struct {
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &tok)
	if resp.StatusCode != http.StatusOK || tok.IDToken == "" {
		msg := tok.Description
		if msg == "" {
			msg = tok.Error
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("%w: %s refused the code: %s", ErrSignIn, p.Label, msg)
	}
	parts := strings.Split(tok.IDToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: malformed ID token", ErrSignIn)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: malformed ID token", ErrSignIn)
	}
	var c idClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("%w: malformed ID token", ErrSignIn)
	}
	switch {
	case !slices.Contains(p.Issuers, c.Iss):
		return nil, fmt.Errorf("%w: unexpected issuer %q", ErrSignIn, c.Iss)
	case !slices.Contains(c.audiences(), p.ClientID):
		return nil, fmt.Errorf("%w: the token is for another application", ErrSignIn)
	case time.Unix(c.Exp, 0).Before(time.Now()):
		return nil, fmt.Errorf("%w: the token expired", ErrSignIn)
	case c.Nonce == "" || c.Nonce != nonce:
		return nil, fmt.Errorf("%w: the token does not answer this sign-in", ErrSignIn)
	case c.Sub == "":
		return nil, fmt.Errorf("%w: no subject in the token", ErrSignIn)
	}
	return &Identity{
		Provider: p.ID, Subject: c.Sub, Email: strings.ToLower(strings.TrimSpace(c.Email)),
		EmailVerified: c.verified(), Name: c.Name, HostedDomain: strings.ToLower(c.HD),
	}, nil
}

// OwnsEmail reports whether the provider vouches that the person holds
// their email address now. For Google: a Workspace account of the
// address's domain (hd), or a Gmail address. A consumer Google account
// created with another address keeps it "verified" even after the address
// changes hands, so it proves nothing about that address.
func (p *Provider) OwnsEmail(id Identity) bool {
	if !id.EmailVerified || id.Email == "" {
		return false
	}
	if !p.HostedDomain {
		return true
	}
	d := id.Domain()
	return (id.HostedDomain != "" && id.HostedDomain == d) || d == "gmail.com" || d == "googlemail.com"
}

// DomainAllowed reports whether the identity belongs to an allowed domain:
// its verified email's domain, and for Google, the Workspace domain Google
// vouches for (a personal account with a matching email is not enough).
func (p *Provider) DomainAllowed(id Identity) bool {
	if !id.EmailVerified || id.Email == "" {
		return false
	}
	domain := id.Domain()
	for _, d := range p.AllowedDomains {
		d = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(d), "@"))
		if d == "" || d != domain {
			continue
		}
		if p.HostedDomain && id.HostedDomain != d {
			continue
		}
		return true
	}
	return false
}
