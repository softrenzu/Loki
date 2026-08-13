package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) tail(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok { http.Error(w, "streaming unsupported", 500); return }
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	ch, cancel := s.Store.Subscribe(); defer cancel(); t := tenant(r)
	for {
		select {
		case <-r.Context().Done(): return
		case e, ok := <-ch:
			if !ok { return }
			if e.Tenant != t { continue }
			b, _ := json.Marshal(e); fmt.Fprintf(w, "data: %s\n\n", b); fl.Flush()
		}
	}
}
