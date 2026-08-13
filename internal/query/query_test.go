package query

import "testing"

func TestParseLogQLSubset(t *testing.T) {
	l, q, e := ParseLogQLSubset(`{app="api",level="error"} ` + string([]byte{'|', '='}) + ` "timeout"`)
	if e != nil || l["app"] != "api" || l["level"] != "error" || q != "timeout" {
		t.Fatalf("bad parse: %#v %q %v", l, q, e)
	}
}
