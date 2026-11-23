package client

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The fixtures in testdata/ are real Context7 responses recorded on
// 2026-11-23 (search, docs ?type=json, and a 404 error body). They pin
// the wire shape this package decodes, and they guard the move from
// encoding/json (v1) to encoding/json/v2: v2 matches member names
// case-sensitively and rejects duplicate names, so a field that only
// decoded through v1's case-insensitive match would silently stay empty.
// Each fixture must decode to the same value under both versions.
func TestFixtures_DecodeSameUnderV1AndV2(t *testing.T) {
	type errBody struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	cases := []struct {
		file string
		new  func() any
		// nonEmpty fails the test if the decoded value is suspiciously empty,
		// so "both versions decoded nothing" cannot pass.
		nonEmpty func(v any) bool
	}{
		{"search-cobra.json", func() any { return new(searchResponse) },
			func(v any) bool { r := v.(*searchResponse); return len(r.Results) > 0 && r.Results[0].ID != "" }},
		{"context-cobra.json", func() any { return new(docsJSONResponse) },
			func(v any) bool { r := v.(*docsJSONResponse); return len(r.Snippets) > 0 && r.Snippets[0].CodeID != "" }},
		{"error-unknown-lib.json", func() any { return new(errBody) },
			func(v any) bool { return v.(*errBody).Message != "" }},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", c.file))
			if err != nil {
				t.Fatal(err)
			}
			v1, v2 := c.new(), c.new()
			if err := jsonv1.Unmarshal(data, v1); err != nil {
				t.Fatalf("v1: %v", err)
			}
			if err := json.Unmarshal(data, v2); err != nil {
				t.Fatalf("v2: %v", err)
			}
			if !c.nonEmpty(v2) {
				t.Fatalf("v2 decoded an empty value from %s", c.file)
			}
			if !reflect.DeepEqual(v1, v2) {
				t.Fatalf("v1 and v2 decode differently:\nv1: %+v\nv2: %+v", v1, v2)
			}
		})
	}
}
