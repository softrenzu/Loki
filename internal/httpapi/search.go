package httpapi

import "net/http"

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	s.Metrics.Queries.Add(1)
	f := filterFromRequest(r)
	jsonOut(w, map[string]any{"data": s.Store.Query(f), "stats": s.Store.Stats()})
}
