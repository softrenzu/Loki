package httpapi

import "net/http"

func (s *Server) ui(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>RooomLog</title><style>body{font-family:system-ui;max-width:920px;margin:40px auto;padding:0 20px;background:#0b1020;color:#e8edf7}input,button{padding:10px;margin:4px;border-radius:8px;border:1px solid #31405f;background:#111a30;color:#fff}button{background:#274690}code{background:#111a30;padding:2px 6px;border-radius:5px}</style></head><body><h1>RooomLog</h1><p>Hybrid full-text and structured log engine.</p><form method="get" action="/api/v1/search"><input name="q" placeholder="message, trace_id, request_id" size="50"><input name="limit" value="200" size="6"><button type="submit">Search</button></form><p>Health: <a href="/healthz">/healthz</a> &nbsp; Metrics: <a href="/metrics">/metrics</a></p><p>Use <code>X-Scope-OrgID</code> for tenant isolation.</p></body></html>`))
}
