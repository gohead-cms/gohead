package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
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

	_, err := accounts.CreateAdmin("owner", "Owner@Studio.test", "a-long-admin-password")
	require.NoError(t, err)

	// signIn runs the whole flow for code and returns the fragment the
	// browser lands on.
	signIn := func(t *testing.T, code string, tamper func(q url.Values)) url.Values {
		t.Helper()
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, httptest.NewRequest("GET", "/auth/sso/google/start?return=/agents", nil))
		require.Equal(t, http.StatusFound, rr.Code)
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
		frag, err := url.ParseQuery(back.Fragment)
		require.NoError(t, err)
		return frag
	}

	t.Run("providers", func(t *testing.T) {
		list := a.ok("", "GET", "/auth/sso", nil).([]any)
		assert.Equal(t, "google", field(list[0], "id"))
	})

	t.Run("an existing account, by its verified email", func(t *testing.T) {
		idp.ids["code-1"] = map[string]any{"sub": "g-owner", "email": "owner@studio.test", "email_verified": true, "hd": "studio.test"}
		frag := signIn(t, "code-1", nil)
		require.NotEmpty(t, frag.Get("sso_token"), frag.Get("sso_error"))
		assert.Equal(t, "/agents", frag.Get("return"))
		claims, err := auth.ParseJWT(frag.Get("sso_token"))
		require.NoError(t, err)
		assert.Equal(t, "owner", claims.Username)
		assert.Equal(t, "admin", claims.Role)
		u, err := storage.GetUserBySSO("google", "g-owner")
		require.NoError(t, err, "linked")
		assert.Equal(t, "owner", u.Username)
	})

	t.Run("a new account for an allowed Workspace domain", func(t *testing.T) {
		idp.ids["code-2"] = map[string]any{"sub": "g-jane", "email": "jane.doe@studio.test", "email_verified": true, "hd": "studio.test", "name": "Jane"}
		frag := signIn(t, "code-2", nil)
		claims, err := auth.ParseJWT(frag.Get("sso_token"))
		require.NoError(t, err, frag.Get("sso_error"))
		assert.Equal(t, "jane.doe", claims.Username)
		assert.Equal(t, "editor", claims.Role, "the default role, never admin")
		// Again: the same account.
		frag = signIn(t, "code-2", nil)
		claims, err = auth.ParseJWT(frag.Get("sso_token"))
		require.NoError(t, err)
		assert.Equal(t, "jane.doe", claims.Username)
	})

	t.Run("refused", func(t *testing.T) {
		// A personal account whose email is in the domain, without the
		// Workspace claim.
		idp.ids["code-3"] = map[string]any{"sub": "g-x", "email": "x@studio.test", "email_verified": true}
		frag := signIn(t, "code-3", nil)
		assert.Empty(t, frag.Get("sso_token"))
		assert.Contains(t, frag.Get("sso_error"), "no GoHead account for x@studio.test")

		idp.ids["code-4"] = map[string]any{"sub": "g-y", "email": "owner@studio.test", "email_verified": false}
		frag = signIn(t, "code-4", nil)
		assert.Empty(t, frag.Get("sso_token"), "an unverified email links nothing")

		frag = signIn(t, "code-1", func(q url.Values) { q.Set("state", "forged") })
		assert.Contains(t, frag.Get("sso_error"), "try again")

		frag = signIn(t, "nope", nil)
		assert.Contains(t, frag.Get("sso_error"), "refused the code")
	})

	t.Run("the state cannot be used as a session token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, httptest.NewRequest("GET", "/auth/sso/google/start", nil))
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
