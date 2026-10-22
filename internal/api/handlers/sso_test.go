package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/sso"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIdP is a token endpoint answering with an ID token for the identity
// of the code it gets ("code-<n>" → ids[n]).
type fakeIdP struct {
	*httptest.Server
	ids   map[string]map[string]any
	nonce string
}

func newFakeIdP(t *testing.T) *fakeIdP {
	f := &fakeIdP{ids: map[string]map[string]any{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		claims, ok := f.ids[r.Form.Get("code")]
		if !ok || r.Form.Get("code_verifier") == "" || r.Form.Get("client_secret") != "secret" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		full := map[string]any{"iss": "https://idp.test", "aud": "client", "exp": time.Now().Add(time.Hour).Unix(), "nonce": f.nonce}
		for k, v := range claims {
			full[k] = v
		}
		b, _ := json.Marshal(full)
		token := "e30." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": token, "access_token": "x"})
	}))
	t.Cleanup(f.Close)
	return f
}

func TestSSO(t *testing.T) {
	a := rolesFixture(t)
	prev := Registration
	Registration = RegistrationDisabled
	t.Cleanup(func() { Registration = prev })
	idp := newFakeIdP(t)
	p := &sso.Provider{
		ID: "google", Label: "Google", AuthURL: "https://idp.test/auth", TokenURL: idp.URL,
		Issuers: []string{"https://idp.test"}, ClientID: "client", ClientSecret: "secret",
		AllowedDomains: []string{"studio.test"}, HostedDomain: true, DefaultRole: "editor",
	}
	sso.Configure(p)
	t.Cleanup(func() { sso.Configure() })
	SSOSettings.PublicURL, SSOSettings.UIURL = "https://cms.test", "https://admin.test"
	t.Cleanup(func() { SSOSettings.PublicURL, SSOSettings.UIURL = "", "" })
	a.router.GET("/auth/sso", ListSSOProviders)
	a.router.GET("/auth/sso/:provider/start", StartSSO)
	a.router.GET("/auth/sso/:provider/callback", SSOCallback)
	a.router.POST("/auth/sso/:provider/link", middleware.AuthMiddleware(), middleware.UserSessionOnly(), LinkSSOStart)
	a.router.DELETE("/admin/users/:id/sso", middleware.AuthMiddleware(), middleware.AdminOnly(), UnlinkUserSSO)

	_, err := accounts.CreateAdmin("admin-user", "Owner@Studio.test", "a-long-admin-password")
	require.NoError(t, err)
	editorRole, err := storage.GetRoleByName("editor")
	require.NoError(t, err)
	ed, err := accounts.Build("ed", "ed@studio.test", "ed-password", editorRole)
	require.NoError(t, err)
	require.NoError(t, storage.CreateUser(ed))

	const bind = "bind-value-0123456789"

	// flow runs the sign-in from startPath with the code, and returns the
	// fragment the browser lands on.
	flow := func(t *testing.T, startPath, code string, tamper func(q url.Values)) url.Values {
		t.Helper()
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, httptest.NewRequest("GET", startPath, nil))
		require.Equal(t, http.StatusFound, rr.Code, rr.Body.String())
		loc, err := url.Parse(rr.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "https://cms.test/auth/sso/google/callback", loc.Query().Get("redirect_uri"))
		assert.Equal(t, "S256", loc.Query().Get("code_challenge_method"))
		idp.nonce = loc.Query().Get("nonce")
		cookie := rr.Result().Cookies()[0]
		assert.True(t, cookie.HttpOnly)
		assert.True(t, cookie.Secure)

		q := url.Values{"code": {code}, "state": {loc.Query().Get("state")}}
		if tamper != nil {
			tamper(q)
		}
		req := httptest.NewRequest("GET", "/auth/sso/google/callback?"+q.Encode(), nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		a.router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusFound, rr.Code)
		back, err := url.Parse(rr.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "https://admin.test/login", back.Scheme+"://"+back.Host+back.Path)
		frag, err := url.ParseQuery(back.EscapedFragment())
		require.NoError(t, err)
		return frag
	}
	signIn := func(t *testing.T, code string, tamper func(q url.Values)) url.Values {
		t.Helper()
		return flow(t, "/auth/sso/google/start?return=/agents&bind="+bind, code, tamper)
	}
	username := func(t *testing.T, frag url.Values) string {
		t.Helper()
		require.NotEmpty(t, frag.Get("sso_token"), frag.Get("sso_error"))
		assert.Equal(t, bind, frag.Get("bind"), "results carry the browser's bind value")
		claims, err := auth.ParseJWT(frag.Get("sso_token"))
		require.NoError(t, err)
		return claims.Username
	}

	t.Run("providers", func(t *testing.T) {
		list := a.ok("", "GET", "/auth/sso", nil).([]any)
		assert.Equal(t, "google", field(list[0], "id"))
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, httptest.NewRequest("GET", "/auth/sso/google/start", nil))
		assert.Equal(t, http.StatusBadRequest, rr.Code, "only from the login page (bind)")
	})

	t.Run("an existing account, by an address Google vouches for", func(t *testing.T) {
		idp.ids["code-ed"] = map[string]any{"sub": "g-ed", "email": "ed@studio.test", "email_verified": true, "hd": "studio.test"}
		frag := signIn(t, "code-ed", nil)
		assert.Equal(t, "ed", username(t, frag))
		assert.Equal(t, "/agents", frag.Get("return"))
		u, err := storage.GetUserBySSO("google", "g-ed")
		require.NoError(t, err, "linked")
		assert.Equal(t, "ed", u.Username)
	})

	t.Run("no automatic link to an admin, or on an address Google does not vouch for", func(t *testing.T) {
		idp.ids["code-owner"] = map[string]any{"sub": "g-owner", "email": "owner@studio.test", "email_verified": true, "hd": "studio.test"}
		frag := signIn(t, "code-owner", nil)
		assert.Empty(t, frag.Get("sso_token"))
		assert.Contains(t, frag.Get("sso_error"), "link your Google account")

		// A consumer Google account created with someone's work address.
		require.NoError(t, storage.UnlinkUserSSO(ed.ID))
		idp.ids["code-fake-ed"] = map[string]any{"sub": "g-fake", "email": "ed@studio.test", "email_verified": true}
		frag = signIn(t, "code-fake-ed", nil)
		assert.Empty(t, frag.Get("sso_token"))
		_, err := storage.GetUserBySSO("google", "g-fake")
		assert.Error(t, err, "not linked")
	})

	t.Run("a signed-in account links itself", func(t *testing.T) {
		out := a.ok("admin", "POST", "/auth/sso/google/link?bind="+bind, nil).(map[string]any)
		linkURL, err := url.Parse(out["url"].(string))
		require.NoError(t, err)
		start := linkURL.Path + "?" + linkURL.RawQuery
		assert.Equal(t, "admin-user", username(t, flow(t, start, "code-owner", nil)))
		u, err := storage.GetUserBySSO("google", "g-owner")
		require.NoError(t, err)
		assert.Equal(t, "admin-user", u.Username)

		// Another person's Google account cannot be attached to this one.
		idp.ids["code-other"] = map[string]any{"sub": "g-other", "email": "other@studio.test", "email_verified": true, "hd": "studio.test"}
		out = a.ok("admin", "POST", "/auth/sso/google/link?bind="+bind, nil).(map[string]any)
		linkURL, _ = url.Parse(out["url"].(string))
		frag := flow(t, linkURL.Path+"?"+linkURL.RawQuery, "code-other", nil)
		assert.Empty(t, frag.Get("sso_token"))
		assert.Contains(t, frag.Get("sso_error"), "change your GoHead email first")

		// Now the admin signs in with Google directly; an admin unlinks.
		assert.Equal(t, "admin-user", username(t, signIn(t, "code-owner", nil)))
		a.ok("admin", "DELETE", fmt.Sprintf("/admin/users/%d/sso", u.ID), nil)
		_, err = storage.GetUserBySSO("google", "g-owner")
		assert.Error(t, err)
	})

	t.Run("a new account for an allowed Workspace domain", func(t *testing.T) {
		idp.ids["code-2"] = map[string]any{"sub": "g-jane", "email": "jane.doe@studio.test", "email_verified": true, "hd": "studio.test", "name": "Jane"}
		frag := signIn(t, "code-2", nil)
		assert.Equal(t, "jane.doe", username(t, frag))
		claims, _ := auth.ParseJWT(frag.Get("sso_token"))
		assert.Equal(t, "editor", claims.Role, "the default role, never admin")
		assert.Equal(t, "jane.doe", username(t, signIn(t, "code-2", nil)), "the same account again")
	})

	t.Run("open registration links nothing by email", func(t *testing.T) {
		Registration = RegistrationOpen
		defer func() { Registration = RegistrationDisabled }()
		// (the account was linked in an earlier subtest)
		require.NoError(t, storage.UnlinkUserSSO(ed.ID))
		frag := signIn(t, "code-ed", nil)
		assert.Empty(t, frag.Get("sso_token"))
	})

	t.Run("refused", func(t *testing.T) {
		idp.ids["code-3"] = map[string]any{"sub": "g-x", "email": "x@studio.test", "email_verified": true}
		frag := signIn(t, "code-3", nil)
		assert.Empty(t, frag.Get("sso_token"))
		assert.Contains(t, frag.Get("sso_error"), "no GoHead account for x@studio.test")

		frag = signIn(t, "code-2", func(q url.Values) { q.Set("state", "forged") })
		assert.Contains(t, frag.Get("sso_error"), "try again")
		assert.Empty(t, frag.Get("bind"), "an unchecked state gets no bind value")

		frag = signIn(t, "code-2", func(q url.Values) { q.Set("state", "forged"); q.Set("error", "Your account is suspended") })
		assert.NotContains(t, frag.Get("sso_error"), "suspended", "no text from the URL before the state is checked")

		frag = signIn(t, "nope", nil)
		assert.Contains(t, frag.Get("sso_error"), "refused the code")
	})

	t.Run("the state cannot be used as a session token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, httptest.NewRequest("GET", "/auth/sso/google/start?bind="+bind, nil))
		cookie := rr.Result().Cookies()[0]
		_, err := auth.ParseJWT(cookie.Value)
		assert.Error(t, err)
		loc, _ := url.Parse(rr.Header().Get("Location"))
		assert.False(t, strings.Contains(loc.RawQuery, "client_secret"))
	})

	t.Run("return paths stay in the admin UI", func(t *testing.T) {
		for _, bad := range []string{"https://evil.test", "//evil.test", "evil", "/\\evil"} {
			assert.Equal(t, "/dashboard", safeReturn(bad), bad)
		}
		assert.Equal(t, "/agents?name=x", safeReturn("/agents?name=x"))
	})
}
