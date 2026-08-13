package httpapi

import (
	"io"
	"net/http"
)

func (s *Server) otlpLogs(w http.ResponseWriter, r *http.Request) {
	entries, err := decodeOTLP(io.LimitReader(r.Body, 32<<20), tenant(r))
	if err != nil { bad(w, err); return }
	for _, e := range entries {
		if err := s.Store.Append(e); err != nil { http.Error(w, err.Error(), 500); return }
		s.Metrics.Ingested.Add(1)
	}
	jsonOut(w, map[string]any{"partialSuccess": map[string]any{"rejectedLogRecords": 0}, "accepted": len(entries)})
}
