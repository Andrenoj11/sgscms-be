package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersContentSecurityPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		path             string
		wantPolicy       string
		wantInlineScript bool
	}{
		{
			name:             "API remains restrictive",
			path:             "/api/v1/public/news",
			wantPolicy:       apiContentSecurityPolicy,
			wantInlineScript: false,
		},
		{
			name:             "Swagger permits its embedded assets",
			path:             "/swagger/index.html",
			wantPolicy:       swaggerContentSecurityPolicy,
			wantInlineScript: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeaders())
			router.GET("/*path", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			policy := response.Header().Get("Content-Security-Policy")
			if policy != test.wantPolicy {
				t.Fatalf("unexpected Content-Security-Policy: %q", policy)
			}

			if strings.Contains(policy, "script-src 'self' 'unsafe-inline'") != test.wantInlineScript {
				t.Fatalf("unexpected inline script policy: %q", policy)
			}
		})
	}
}
