package httpserver

import (
    "fmt"
    "net/http"
)

func RedocHandler(specPath string) http.HandlerFunc {
    page := fmt.Sprintf(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>API Docs</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="shortcut icon" href="#">
    <style>body { margin: 0; padding: 0; }</style>
  </head>
  <body>
    <redoc spec-url="%s"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
  </body>
</html>`, specPath)
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.WriteHeader(200)
        _, _ = w.Write([]byte(page))
    }
}

