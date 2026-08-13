package query

import (
	"errors"
	"strings"
	"unicode"
)

// ParseLogQLSubset accepts a deliberately useful Loki-compatible subset:
// label selector followed by a contains-text pipeline
// This allows existing Grafana/Loki clients to start without a new query language.
func ParseLogQLSubset(q string) (map[string]string, string, error) {
	q = strings.TrimSpace(q)
	labels := map[string]string{}
	if q == "" {
		return labels, "", nil
	}
	if !strings.HasPrefix(q, "{") {
		return labels, q, nil
	}
	end := strings.Index(q, "}")
	if end < 0 {
		return nil, "", errors.New("missing } in selector")
	}
	sel := q[1:end]
	if strings.TrimSpace(sel) != "" {
		for _, p := range splitComma(sel) {
			pair := strings.SplitN(p, "=", 2)
			if len(pair) != 2 {
				return nil, "", errors.New("only exact label matchers are supported")
			}
			key := strings.TrimSpace(pair[0])
			value := strings.TrimSpace(pair[1])
			if !validLabelName(key) || len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
				return nil, "", errors.New("invalid exact label matcher")
			}
			labels[key] = strings.Trim(value, `"`)
		}
	}
	rest := strings.TrimSpace(q[end+1:])
	if rest == "" {
		return labels, "", nil
	}
	pipeEq := string([]byte{'|', '='})
	if strings.HasPrefix(rest, pipeEq) {
		text := strings.TrimSpace(strings.TrimPrefix(rest, pipeEq))
		text = strings.Trim(text, `"`)
		return labels, text, nil
	}
	return nil, "", errors.New("unsupported pipeline operator")
}

func validLabelName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			return false
		}
		if i > 0 && !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '-') {
			return false
		}
	}
	return true
}

func splitComma(s string) []string {
	var out []string
	start := 0
	quoted := false
	for i, r := range s {
		if r == '"' {
			quoted = !quoted
		}
		if r == ',' && !quoted {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
