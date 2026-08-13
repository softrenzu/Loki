package httpapi

import (
	"fmt"
	"net/http"
)

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintln(w, "# HELP rooomlog_up Whether RooomLog is serving requests")
	fmt.Fprintln(w, "# TYPE rooomlog_up gauge")
	fmt.Fprintln(w, "rooomlog_up 1")
}
