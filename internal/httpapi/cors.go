package httpapi

import "net/http"

func (s *Server) cors(next http.Handler) http.Handler { return next }
