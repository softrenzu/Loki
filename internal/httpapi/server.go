package httpapi

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/softrenzu/Loki/internal/store"
)

type Metrics struct {
	Ingested atomic.Uint64
	Queries  atomic.Uint64
	Errors   atomic.Uint64
}

type Server struct {
	Store     *store.Store
	Metrics   Metrics
	mux       *http.ServeMux
	retention time.Duration
}

func New(st *store.Store, retention time.Duration) *Server {
	s := &Server{Store: st, mux: http.NewServeMux(), retention: retention}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.cors(s.mux) }
