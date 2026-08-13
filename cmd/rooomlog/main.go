package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/softrenzu/RooomLog/internal/httpapi"
	"github.com/softrenzu/RooomLog/internal/store"
)

func main() {
	addr := env("ROOOMLOG_ADDR", ":3100")
	data := env("ROOOMLOG_DATA", "./data")
	retentionHours, _ := strconv.Atoi(env("ROOOMLOG_RETENTION_HOURS", "168"))
	retention := time.Duration(retentionHours) * time.Hour
	st, err := store.Open(data)
	if err != nil { slog.Error("open store", "error", err); os.Exit(1) }
	defer st.Close()
	api := httpapi.New(st, retention)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api.StartRetention(ctx)
	srv := &http.Server{Addr: addr, Handler: api.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 0, IdleTimeout: 2 * time.Minute}
	go func() {
		slog.Info("RooomLog listening", "addr", addr, "data", data, "retention", retention)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { slog.Error("server", "error", err); os.Exit(1) }
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	c, cc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cc()
	_ = srv.Shutdown(c)
}
func env(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
