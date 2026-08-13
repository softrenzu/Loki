package httpapi

import (
	"context"
	"log/slog"
	"time"
)

func (s *Server) StartRetention(ctx context.Context) {
	if s.retention <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				cut := time.Now().Add(-s.retention).UnixNano()
				if n, err := s.Store.DeleteBefore(cut); err != nil {
					slog.Error("retention", "error", err)
				} else if n > 0 {
					slog.Info("retention", "deleted", n)
				}
			}
		}
	}()
}
