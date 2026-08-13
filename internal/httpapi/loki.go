package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/softrenzu/Loki/internal/query"
	"github.com/softrenzu/Loki/internal/store"
)

func (s *Server) lokiPush(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Streams []struct {
			Stream map[string]string   `json:"stream"`
			Values [][]json.RawMessage `json:"values"`
		} `json:"streams"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 32<<20)).Decode(&p); err != nil { bad(w, err); return }
	for _, st := range p.Streams {
		for _, v := range st.Values {
			if len(v) < 2 { continue }
			var tsS, msg string
			_ = json.Unmarshal(v[0], &tsS); _ = json.Unmarshal(v[1], &msg)
			ts, _ := strconv.ParseInt(tsS, 10, 64)
			fields := map[string]any{}
			if len(v) >= 3 { _ = json.Unmarshal(v[2], &fields) }
			e := store.LogEntry{Timestamp: ts, Tenant: tenant(r), Message: msg, Labels: st.Stream, Fields: fields}
			if err := s.Store.Append(e); err != nil { http.Error(w, err.Error(), 500); return }
			s.Metrics.Ingested.Add(1)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) lokiQueryRange(w http.ResponseWriter, r *http.Request) {
	s.Metrics.Queries.Add(1)
	labels, text, err := query.ParseLogQLSubset(r.URL.Query().Get("query"))
	if err != nil { bad(w, err); return }
	entries := s.Store.Query(store.Filter{Tenant: tenant(r), Query: text, Labels: labels, From: parseTime(r.URL.Query().Get("start")), To: parseTime(r.URL.Query().Get("end")), Limit: parseInt(r.URL.Query().Get("limit"), 1000)})
	type group struct { Labels map[string]string; Values [][]any }
	groups := map[string]group{}
	for _, e := range entries {
		b, _ := json.Marshal(e.Labels); k := string(b); g := groups[k]
		if g.Labels == nil { g.Labels = e.Labels }
		g.Values = append(g.Values, []any{strconv.FormatInt(e.Timestamp, 10), e.Message, e.Fields}); groups[k] = g
	}
	result := make([]map[string]any, 0, len(groups))
	for _, g := range groups { result = append(result, map[string]any{"stream": g.Labels, "values": g.Values}) }
	jsonOut(w, map[string]any{"status": "success", "data": map[string]any{"resultType": "streams", "result": result}})
}
