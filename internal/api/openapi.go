package api

import (
	_ "embed"
	"net/http"
)

// openAPISpec describes every endpoint. The contract test validates real
// responses against it, so it cannot drift from the handlers.
//
//go:embed openapi.yaml
var openAPISpec []byte

// handleOpenAPI serves the API description.
func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPISpec)
}
