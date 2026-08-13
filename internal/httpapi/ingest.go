package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/softrenzu/Loki/internal/store"
)

func tenant(r *http.Request) string {
	if v := r.Header.Get("X-Scope-OrgID"); v != "" {
		return v
	}
	return "default"
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	var raw any
	dec := json.NewDecoder(io.LimitReader(r.Body, 16<<20))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		bad(w, err)
		return
	}
	var entries []store.LogEntry
	switch v := raw.(type) {
	case map[string]any:
		b, _ := json.Marshal(v)
		var e store.LogEntry
		if err := json.Unmarshal(b, &e); err != nil {
			bad(w, err)
			return
		}
		entries = []store.LogEntry{e}
	case []any:
		b, _ := json.Marshal(v)
		if err := json.Unmarshal(b, &entries); err != nil {
			bad(w, err)
			return
		}
	default:
		bad(w, fmt.Errorf("expected object or array"))
		return
	}
	for i := range entries {
		entries[i].Tenant = tenant(r)
		if err := s.Store.Append(entries[i]); err != nil {
			s.Metrics.Errors.Add(1)
			http.Error(w, err.Error(), 500)
			return
		}
		s.Metrics.Ingested.Add(1)
	}
	w.WriteHeader(http.StatusNoContent)
}
