package httpapi

import "strings"

func routePath(parts ...string) string { return "/" + strings.Join(parts, "/") }

func (s *Server) addLogRoutes() {
	s.mux.HandleFunc("GET "+routePath("metrics"), s.metrics)
	s.mux.HandleFunc("POST "+routePath("api", "v1", "ingest"), s.ingest)
	s.mux.HandleFunc("GET "+routePath("api", "v1", "search"), s.search)
	s.mux.HandleFunc("GET "+routePath("api", "v1", "tail"), s.tail)
	s.mux.HandleFunc("GET "+routePath("api", "v1", "anomalies"), s.anomalies)
	s.mux.HandleFunc("POST "+routePath("loki", "api", "v1", "push"), s.lokiPush)
	s.mux.HandleFunc("GET "+routePath("loki", "api", "v1", "query_range"), s.lokiQueryRange)
	s.mux.HandleFunc("POST "+routePath("v1", "logs"), s.otlpLogs)
}
