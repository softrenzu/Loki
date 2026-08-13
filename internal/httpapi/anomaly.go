package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/softrenzu/Loki/internal/store"
)

func (s *Server) anomalies(w http.ResponseWriter, r *http.Request) {
	hours := parseInt(r.URL.Query().Get("hours"), 1)
	if hours < 1 { hours = 1 }
	if hours > 168 { hours = 168 }
	entries := s.Store.Query(store.Filter{Tenant: tenant(r), From: time.Now().Add(-time.Duration(hours) * time.Hour).UnixNano(), Limit: 10000})
	type pattern struct { Pattern string `json:"pattern"`; Count int `json:"count"`; Rarity float64 `json:"rarity"`; Example string `json:"example"` }
	counts := map[string]*pattern{}
	for _, e := range entries { p := fingerprint(e.Message); x := counts[p]; if x == nil { x = &pattern{Pattern:p, Example:e.Message}; counts[p] = x }; x.Count++ }
	out := make([]pattern, 0, len(counts)); total := float64(max(1, len(entries)))
	for _, x := range counts { x.Rarity = 1 - float64(x.Count)/total; if x.Rarity >= 0.80 || x.Count <= 2 { out = append(out, *x) } }
	for i := 0; i < len(out); i++ { for j := i+1; j < len(out); j++ { if out[j].Rarity > out[i].Rarity { out[i], out[j] = out[j], out[i] } } }
	if len(out) > 100 { out = out[:100] }
	jsonOut(w, map[string]any{"window_hours": hours, "patterns": out})
}
func fingerprint(s string) string { fields := strings.Fields(s); for i, f := range fields { low := strings.ToLower(strings.Trim(f, ",;()[]{}")); if isDynamic(low) { fields[i] = "<*>" } }; return strings.Join(fields, " ") }
func isDynamic(s string) bool { if len(s) < 8 { return false }; digits, hex := 0,0; for _, r := range s { if r >= '0' && r <= '9' { digits++ }; if (r >= 'a' && r <= 'f') || (r >= '0' && r <= '9') || r == '-' { hex++ } }; return digits >= 4 || hex == len(s) }
