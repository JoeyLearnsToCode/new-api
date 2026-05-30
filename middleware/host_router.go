package middleware

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func MoeAtelierMiddleware(moeFS embed.FS) gin.HandlerFunc {
	subFS, err := fs.Sub(moeFS, "web/moe-atelier-dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(subFS))
	indexBytes, err := fs.ReadFile(subFS, "index.html")
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		if !strings.HasPrefix(strings.ToLower(c.Request.Host), "imagen") {
			c.Next()
			return
		}

		path := strings.TrimPrefix(c.Request.URL.Path, "/")

		if path != "" {
			f, err := subFS.Open(path)
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				c.Abort()
				return
			}
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
		c.Abort()
	}
}
