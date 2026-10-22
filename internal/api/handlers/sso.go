package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/internal/sso"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/storage"

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
	// Bind is a random value the admin UI kept for this sign-in: it
	// accepts the result only with it, so nobody can sign a browser into
	// their own account with a crafted link.
	Bind string `json:"b"`
	// Link is the signed-in account that links this identity, if any.
	Link string `json:"l,omitempty"`
	jwt.RegisteredClaims
}

func ssoKey() []byte { return auth.DerivedKey("sso-state") }

func linkKey() []byte { return auth.DerivedKey("sso-link") }

type linkTicket struct {
	Username string `json:"u"`
	Provider string `json:"p"`
	jwt.RegisteredClaims
}

// publicURL is GoHead's address for redirect URIs.
func publicURL(c *gin.Context) string {
	if SSOSettings.PublicURL != "" {
		return strings.TrimRight(SSOSettings.PublicURL, "/")
	}
	scheme := "http"
	// X-Forwarded-Proto counts only from a trusted proxy (ClientIP then
	// differs from the connection's address). Set auth.sso.public_url in
	// production.
	forwarded := c.ClientIP() != c.RemoteIP() && strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	if c.Request.TLS != nil || forwarded {
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

// LinkSSOStart starts linking the signed-in account to a provider: it
// returns the URL to send the browser to (the start URL with a short-lived
// ticket, since a browser navigation carries no Authorization header).
func LinkSSOStart(c *gin.Context) {
	p, ok := sso.Get(c.Param("provider"))
	if !ok {
		c.Set("response", "Unknown sign-in provider")
		c.Set("status", http.StatusNotFound)
		return
	}
	username := c.GetString("username")
	ticket, err := jwt.NewWithClaims(jwt.SigningMethodHS256, linkTicket{
		Username: username, Provider: p.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
			Audience:  jwt.ClaimStrings{"gohead-sso-link"},
		},
	}).SignedString(linkKey())
	if err != nil {
		c.Set("response", "Could not start linking")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	q := url.Values{"link": {ticket}, "bind": {c.Query("bind")}, "return": {safeReturn(c.Query("return"))}}
	c.Set("response", gin.H{"url": publicURL(c) + "/auth/sso/" + p.ID + "/start?" + q.Encode()})
	c.Set("status", http.StatusOK)
}

// UnlinkUserSSO removes a user's link to their identity provider (admin):
// they sign in with their password again.
func UnlinkUserSSO(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}
	if err := storage.UnlinkUserSSO(id); err != nil {
		c.Set("response", "Failed to unlink the account")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Set("response", gin.H{"unlinked": id})
	c.Set("status", http.StatusOK)
}

// StartSSO sends the browser to the provider. Query: bind (a random value
// the admin UI keeps, required), return (the admin UI path to open after
// signing in), link (a ticket from LinkSSOStart).
func StartSSO(c *gin.Context) {
	p, ok := sso.Get(c.Param("provider"))
	if !ok {
		c.Set("response", "Unknown sign-in provider")
		c.Set("status", http.StatusNotFound)
		return
	}
	bind := c.Query("bind")
	if len(bind) < 16 || len(bind) > 128 {
		c.Set("response", "Start the sign-in from GoHead's login page")
		c.Set("status", http.StatusBadRequest)
		return
	}
	var link string
	if t := c.Query("link"); t != "" {
		var lt linkTicket
		_, err := jwt.ParseWithClaims(t, &lt, func(*jwt.Token) (any, error) { return linkKey(), nil },
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithAudience("gohead-sso-link"))
		if err != nil || lt.Provider != p.ID || lt.Username == "" {
			c.Set("response", "The link expired: start again from GoHead")
			c.Set("status", http.StatusBadRequest)
			return
		}
		link = lt.Username
	}
	st := ssoState{
		Provider: p.ID, State: sso.RandomString(24), Nonce: sso.RandomString(24), Verifier: sso.RandomString(48),
		Return: safeReturn(c.Query("return")), Bind: bind, Link: link,
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
	// From here, results go back with the browser's bind value.
	fail = func(msg string) {
		http.SetCookie(c.Writer, &http.Cookie{Name: ssoCookie, Value: "", Path: "/auth/sso/", MaxAge: -1, HttpOnly: true})
		c.Redirect(http.StatusFound, uiURL(c)+"/login#"+url.Values{"sso_error": {msg}, "bind": {st.Bind}}.Encode())
		c.Abort()
	}
	if e := c.Query("error"); e != "" {
		if e == "access_denied" {
			fail("The sign-in was cancelled.")
		} else {
			fail(p.Label + " refused the sign-in.")
		}
		return
	}
	id, err := p.Exchange(c.Request.Context(), redirectURI(c, p.ID), c.Query("code"), st.Verifier, st.Nonce)
	if err != nil {
		logger.Log.WithError(err).WithField("provider", p.ID).Warn("Sign-in with a provider failed")
		fail(strings.TrimPrefix(err.Error(), sso.ErrSignIn.Error()+": "))
		return
	}
	var user *models.User
	created := false
	if st.Link != "" {
		user, err = accounts.LinkSSO(st.Link, *id)
	} else {
		// When anyone can register, an account's email proves nothing.
		user, created, err = accounts.SignInWith(p, *id, Registration != RegistrationOpen)
	}
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
	c.Redirect(http.StatusFound, uiURL(c)+"/login#"+url.Values{"sso_token": {token}, "return": {st.Return}, "bind": {st.Bind}}.Encode())
	c.Abort()
}
