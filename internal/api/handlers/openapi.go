package handlers

import (
	"net/http"

	"github.com/gohead-cms/gohead/internal/openapi"
	"github.com/gohead-cms/gohead/pkg/logger"

	"github.com/gin-gonic/gin"
)

// APIVersion is reported in the OpenAPI document (set at startup).
var APIVersion = "dev"

// OpenAPISpec serves the OpenAPI 3.1 description of the current content
// model. It is written directly (not wrapped in {"data": …}), as tools
// expect.
func OpenAPISpec(c *gin.Context) {
	doc, err := openapi.Build(APIVersion)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to build OpenAPI document")
		c.Set("response", "Failed to build the OpenAPI document")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, doc)
}

// APIDocs serves Swagger UI for /api/openapi.json. When the spec needs a
// token, the page asks for one and keeps it for the browser session only.
func APIDocs(public bool) gin.HandlerFunc {
	page := swaggerPage(public)
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https://cdn.jsdelivr.net; connect-src 'self'")
		c.Header("X-Frame-Options", "DENY")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
	}
}

func swaggerPage(public bool) string {
	needToken := "true"
	if public {
		needToken = "false"
	}
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>GoHead API</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css" integrity="sha384-wxLW6kwyHktdDGr6Pv1zgm/VGJh99lfUbzSn6HNHBENZlCN7W602k9VkGdxuFvPn" crossorigin="anonymous">
<style>
  body { margin: 0; font-family: system-ui, sans-serif; }
  #login { max-width: 420px; margin: 15vh auto; padding: 24px; border: 1px solid #ddd; border-radius: 8px; }
  #login input { width: 100%; box-sizing: border-box; padding: 8px; margin: 8px 0 12px; font-family: monospace; }
  #login button { padding: 8px 16px; }
  #login p { color: #555; font-size: 14px; }
</style>
</head>
<body>
<form id="login" hidden>
  <h2>GoHead API</h2>
  <p>Paste an API token (<code>ght_…</code>) or a JWT from <code>POST /auth/login</code>. It is kept in this tab only.</p>
  <input id="token" type="password" autocomplete="off" placeholder="ght_…" required>
  <button type="submit">Open the API reference</button>
</form>
<div id="ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js" integrity="sha384-wmyclcVGX/WhUkdkATwhaK1X1JtiNrr2EoYJ+diV3vj4v6OC5yCeSu+yW13SYJep" crossorigin="anonymous"></script>
<script>
(function () {
  var needToken = ` + needToken + `;
  var key = "gohead-api-token";
  function start(token) {
    window.ui = SwaggerUIBundle({
      url: "openapi.json",
      dom_id: "#ui",
      deepLinking: true,
      persistAuthorization: false,
      requestInterceptor: function (req) {
        if (token && !req.headers.Authorization) req.headers.Authorization = "Bearer " + token;
        return req;
      }
    });
  }
  var saved = null;
  try { saved = sessionStorage.getItem(key); } catch (e) {}
  if (!needToken || saved) { start(saved); return; }
  var form = document.getElementById("login");
  form.hidden = false;
  form.addEventListener("submit", function (e) {
    e.preventDefault();
    var t = document.getElementById("token").value.trim();
    try { sessionStorage.setItem(key, t); } catch (e2) {}
    form.hidden = true;
    start(t);
  });
})();
</script>
</body>
</html>`
}
