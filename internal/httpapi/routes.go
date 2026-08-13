package httpapi

import "net/http"

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.ui)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, map[string]any{"status": "ok", "name": "RooomLog"})
	})
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, map[string]any{"ready": true})
	})
	s.addLogRoutes()
}
