package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func parseTime(v string) int64 {
	if v == "" { return 0 }
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		if n < 1e12 { return n * int64(time.Second) }
		if n < 1e15 { return n * int64(time.Millisecond) }
		return n
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil { return t.UnixNano() }
	return 0
}
func parseInt(v string, d int) int { if n, err := strconv.Atoi(v); err == nil { return n }; return d }
func jsonOut(w http.ResponseWriter, v any) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
func bad(w http.ResponseWriter, err error) { http.Error(w, err.Error(), http.StatusBadRequest) }
func max(a,b int) int { if a>b { return a }; return b }
