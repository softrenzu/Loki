package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/softrenzu/Loki/internal/store"
)

func filterFromRequest(r *http.Request) store.Filter {
	params := r.URL.Query()
	f := store.Filter{Tenant: tenant(r), Query: params.Get("q"), Limit: 200}
	if n, err := strconv.Atoi(params.Get("limit")); err == nil && n > 0 && n <= 10000 { f.Limit = n }
	for key, values := range params {
		if strings.HasPrefix(key, "label.") && len(values) > 0 {
			if f.Labels == nil { f.Labels = map[string]string{} }
			f.Labels[strings.TrimPrefix(key, "label.")] = values[0]
		}
	}
	f.From = parseTime(params.Get("from"))
	f.To = parseTime(params.Get("to"))
	return f
}
