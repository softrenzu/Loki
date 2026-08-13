package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/softrenzu/Loki/internal/store"
)

func decodeOTLP(r io.Reader, tenantID string) ([]store.LogEntry, error) {
	var root map[string]any
	dec := json.NewDecoder(r); dec.UseNumber()
	if err := dec.Decode(&root); err != nil { return nil, err }
	var out []store.LogEntry
	resources, _ := root["resourceLogs"].([]any)
	for _, rv := range resources {
		rm := anyMap(rv); resourceAttrs := attrsFrom(anyMap(rm["resource"])["attributes"]); scopes, _ := rm["scopeLogs"].([]any)
		for _, sv := range scopes {
			sm := anyMap(sv); recs, _ := sm["logRecords"].([]any)
			for _, lv := range recs {
				lm := anyMap(lv); attrs := attrsFrom(lm["attributes"]); labels := map[string]string{}; fields := map[string]any{}
				for k, v := range resourceAttrs { fields[k] = v }
				for k, v := range attrs { fields[k] = v }
				if v := fmt.Sprint(resourceAttrs["service.name"]); v != "<nil>" && v != "" { labels["service_name"] = v }
				if v := fmt.Sprint(lm["severityText"]); v != "<nil>" && v != "" { labels["level"] = strings.ToLower(v) }
				e := store.LogEntry{Timestamp: parseAnyInt64(lm["timeUnixNano"]), Tenant: tenantID, Message: fmt.Sprint(anyValue(lm["body"])), Labels: labels, Fields: fields, TraceID: fmt.Sprint(lm["traceId"]), SpanID: fmt.Sprint(lm["spanId"])}
				if e.TraceID == "<nil>" { e.TraceID = "" }
				if e.SpanID == "<nil>" { e.SpanID = "" }
				out = append(out, e)
			}
		}
	}
	return out, nil
}
func anyMap(v any) map[string]any { if m, ok := v.(map[string]any); ok { return m }; return map[string]any{} }
func attrsFrom(v any) map[string]any { out := map[string]any{}; arr, _ := v.([]any); for _, x := range arr { m := anyMap(x); out[fmt.Sprint(m["key"])] = anyValue(m["value"]) }; return out }
func anyValue(v any) any { m := anyMap(v); for _, k := range []string{"stringValue", "boolValue", "intValue", "doubleValue", "bytesValue"} { if x, ok := m[k]; ok { return x } }; if x, ok := m["arrayValue"]; ok { return x }; if x, ok := m["kvlistValue"]; ok { return x }; return v }
func parseAnyInt64(v any) int64 { switch x := v.(type) { case json.Number: n, _ := x.Int64(); return n; case string: n, _ := strconv.ParseInt(x, 10, 64); return n; case float64: return int64(x) }; return 0 }
