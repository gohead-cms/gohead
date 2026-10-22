package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/internal/sso"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Sign-in with an identity provider (Google). The browser goes to
// /auth/sso/:provider/start, signs in at the provider, comes back to
// /auth/sso/:provider/callback, and lands on the admin UI's login page with
// a GoHead token in the URL fragment (#sso_token=…), which never reaches a
// server.

// SSOSettings are where GoHead and its admin UI are, as browsers reach
// them. Empty: from the request (GoHead serves its UI itself).
var SSOSettings struct {
	PublicURL string
	UIURL     string
}

const (
	ssoCookie   = "gohead_sso"
	ssoStateTTL = 10 * time.Minute
)

type ssoState struct {
	Provider string `json:"p"`
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Return   string `json:"r"`
	jwt.RegisteredClaims
}

func ssoKey() []byte { return auth.DerivedKey("sso-state") }

// publicURL is GoHead's address for redirect URIs.
func publicURL(c *gin.Context) string {
	if SSOSettings.PublicURL != "" {
		return strings.TrimRight(SSOSettings.PublicURL, "/")
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func uiURL(c *gin.Context) string {
	if SSOSettings.UIURL != "" {
		return strings.TrimRight(SSOSettings.UIURL, "/")
	}
	return publicURL(c)
}

func redirectURI(c *gin.Context, provider string) string {
	return publicURL(c) + "/auth/sso/" + provider + "/callback"
}

// safeReturn keeps a path of the admin UI ("/agents"), nothing that leaves
// it.
func safeReturn(p string) string {
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.ContainsAny(p, "\\\r\n") {
		return "/dashboard"
	}
	return p
}

// ListSSOProviders lists the providers people can sign in with.
func ListSSOProviders(c *gin.Context) {
	c.Set("response", sso.Enabled())
	c.Set("status", http.StatusOK)
}

// StartSSO sends the browser to the provider. Query: return, the admin UI
// path to open after signing in.
func StartSSO(c *gin.Context) {
	p, ok := sso.Get(c.Param("provider"))
	if !ok {
		c.Set("response", "Unknown sign-in provider")
		c.Set("status", http.StatusNotFound)
		return
	}
	st := ssoState{
		Provider: p.ID, State: sso.RandomString(24), Nonce: sso.RandomString(24), Verifier: sso.RandomString(48),
		Return: safeReturn(c.Query("return")),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ssoStateTTL)),
			Audience:  jwt.ClaimStrings{"gohead-sso"},
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, st).SignedString(ssoKey())
	if err != nil {
		c.Set("response", "Could not start the sign-in")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	secure := strings.HasPrefix(publicURL(c), "https://")
	http.SetCookie(c.Writer, &http.Cookie{
		Name: ssoCookie, Value: signed, Path: "/auth/sso/", MaxAge: int(ssoStateTTL.Seconds()),
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, p.AuthCodeURL(redirectURI(c, p.ID), st.State, st.Nonce, st.Verifier))
	c.Abort()
}

// SSOCallback finishes the sign-in and sends the browser back to the admin
// UI with a token, or with the reason it failed.
func SSOCallback(c *gin.Context) {
	fail := func(msg string) {
		http.SetCookie(c.Writer, &http.Cookie{Name: ssoCookie, Value: "", Path: "/auth/sso/", MaxAge: -1, HttpOnly: true})
		c.Redirect(http.StatusFound, uiURL(c)+"/login#"+url.Values{"sso_error": {msg}}.Encode())
		c.Abort()
	}
	p, ok := sso.Get(c.Param("provider"))
	if !ok {
		fail("This sign-in provider is not enabled.")
		return
	}
	if e := c.Query("error"); e != "" {
		if e == "access_denied" {
			fail("The sign-in was cancelled.")
		} else {
			fail(p.Label + " refused the sign-in: " + e)
		}
		return
	}
	cookie, err := c.Cookie(ssoCookie)
	if err != nil {
		fail("The sign-in expired or was started in another browser: try again.")
		return
	}
	var st ssoState
	_, err = jwt.ParseWithClaims(cookie, &st, func(*jwt.Token) (any, error) { return ssoKey(), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithAudience("gohead-sso"))
	if err != nil || st.Provider != p.ID || st.State == "" || c.Query("state") != st.State {
		fail("The sign-in expired or was started in another browser: try again.")
		return
	}
	id, err := p.Exchange(c.Request.Context(), redirectURI(c, p.ID), c.Query("code"), st.Verifier, st.Nonce)
	if err != nil {
		logger.Log.WithError(err).WithField("provider", p.ID).Warn("Sign-in with a provider failed")
		fail(strings.TrimPrefix(err.Error(), sso.ErrSignIn.Error()+": "))
		return
	}
	user, created, err := accounts.SignInWith(p, *id)
	if err != nil {
		if errors.Is(err, accounts.ErrNoAccount) {
			logger.Log.WithFields(map[string]any{"provider": p.ID, "email": id.Email}).Warn("Sign-in with a provider: no account")
			fail(strings.TrimPrefix(err.Error(), accounts.ErrNoAccount.Error()+": "))
			return
		}
		logger.Log.WithError(err).WithField("provider", p.ID).Error("Sign-in with a provider failed")
		fail("Could not sign you in.")
		return
	}
	if user.Role.Name == "" {
		fail("Your account has no role: ask an administrator.")
		return
	}
	token, err := auth.GenerateJWT(user.Username, user.Role.Name)
	if err != nil {
		fail("Could not sign you in.")
		return
	}
	logger.Log.WithFields(map[string]any{"username": user.Username, "provider": p.ID, "created": created}).Info("Signed in with a provider")
	http.SetCookie(c.Writer, &http.Cookie{Name: ssoCookie, Value: "", Path: "/auth/sso/", MaxAge: -1, HttpOnly: true})
	c.Redirect(http.StatusFound, uiURL(c)+"/login#"+url.Values{"sso_token": {token}, "return": {st.Return}}.Encode())
	c.Abort()
}
