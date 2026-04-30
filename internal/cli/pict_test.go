package cli

import (
	"fmt"
	"testing"

	"github.com/kevin-burns/c7search/internal/output"
)

// PICT-style pairwise covering test for flag interactions.
//
// Parameters:
//
//	A. apiKeyFlag    : "", "ctx7sk-flag"
//	B. apiKeyEnv     : "", "ctx7sk-env"
//	C. jsonFlag      : false, true
//	D. formatFlag    : "", "text", "md", "json", "garbage"
//	E. noCacheFlag   : false, true
//
// The naive Cartesian product is 2*2*2*5*2 = 80 cases. Pairwise (every
// 2-tuple of parameter values appears in at least one test case) requires
// ~12. The set below is hand-derived using IPO with parameters ordered
// largest-domain-first for tighter packing. The test asserts pairwise
// coverage over the rows so reordering or trimming the set fails fast.
type pictRow struct {
	apiKeyFlag, apiKeyEnv, formatFlag string
	jsonFlag, noCacheFlag             bool
	wantKey                           string        // expected resolveAPIKey
	wantFormat                        output.Format // expected resolveFormat(default=Markdown)
}

var pictRows = []pictRow{
	// A=""        B=""         C=false D=""        E=false
	{apiKeyFlag: "", apiKeyEnv: "", jsonFlag: false, formatFlag: "", noCacheFlag: false,
		wantKey: "", wantFormat: output.FormatMarkdown},
	// A="flag"    B=""         C=true  D="text"    E=true
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "", jsonFlag: true, formatFlag: "text", noCacheFlag: true,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatJSON},
	// A=""        B="env"      C=true  D="md"      E=false
	{apiKeyFlag: "", apiKeyEnv: "ctx7sk-env", jsonFlag: true, formatFlag: "md", noCacheFlag: false,
		wantKey: "ctx7sk-env", wantFormat: output.FormatJSON},
	// A="flag"    B="env"      C=false D="json"    E=true
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "ctx7sk-env", jsonFlag: false, formatFlag: "json", noCacheFlag: true,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatJSON},
	// A=""        B=""         C=true  D="garbage" E=true
	{apiKeyFlag: "", apiKeyEnv: "", jsonFlag: true, formatFlag: "garbage", noCacheFlag: true,
		wantKey: "", wantFormat: output.FormatJSON},
	// A="flag"    B=""         C=false D="md"      E=false
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "", jsonFlag: false, formatFlag: "md", noCacheFlag: false,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatMarkdown},
	// A=""        B="env"      C=false D="garbage" E=false
	{apiKeyFlag: "", apiKeyEnv: "ctx7sk-env", jsonFlag: false, formatFlag: "garbage", noCacheFlag: false,
		wantKey: "ctx7sk-env", wantFormat: output.FormatText},
	// A="flag"    B="env"      C=true  D=""        E=false
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "ctx7sk-env", jsonFlag: true, formatFlag: "", noCacheFlag: false,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatJSON},
	// A=""        B=""         C=false D="text"    E=true
	{apiKeyFlag: "", apiKeyEnv: "", jsonFlag: false, formatFlag: "text", noCacheFlag: true,
		wantKey: "", wantFormat: output.FormatText},
	// A="flag"    B="env"      C=false D=""        E=true
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "ctx7sk-env", jsonFlag: false, formatFlag: "", noCacheFlag: true,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatMarkdown},
	// A=""        B="env"      C=true  D="text"    E=true
	{apiKeyFlag: "", apiKeyEnv: "ctx7sk-env", jsonFlag: true, formatFlag: "text", noCacheFlag: true,
		wantKey: "ctx7sk-env", wantFormat: output.FormatJSON},
	// A="flag"    B=""         C=true  D="json"    E=false
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "", jsonFlag: true, formatFlag: "json", noCacheFlag: false,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatJSON},
	// Pair-fillers — cover the residual pairs the generator flagged after
	// the initial set (formatFlag has 5 levels, the largest domain, so it
	// drives most of the residual).
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "ctx7sk-env", jsonFlag: true, formatFlag: "garbage", noCacheFlag: false,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatJSON},
	{apiKeyFlag: "", apiKeyEnv: "", jsonFlag: false, formatFlag: "text", noCacheFlag: false,
		wantKey: "", wantFormat: output.FormatText},
	{apiKeyFlag: "ctx7sk-flag", apiKeyEnv: "ctx7sk-env", jsonFlag: true, formatFlag: "md", noCacheFlag: true,
		wantKey: "ctx7sk-flag", wantFormat: output.FormatJSON},
	{apiKeyFlag: "", apiKeyEnv: "ctx7sk-env", jsonFlag: true, formatFlag: "json", noCacheFlag: true,
		wantKey: "ctx7sk-env", wantFormat: output.FormatJSON},
}

// TestPICT_FlagBehavior drives every PICT row through resolveAPIKey and
// resolveFormat. Each row asserts the documented precedence:
//
//	api key:    flag > env > "" (anonymous)
//	format:     --json  >  --format  >  command default
//	            unknown --format → FormatText (per ParseFormat)
func TestPICT_FlagBehavior(t *testing.T) {
	for i, row := range pictRows {
		row := row
		t.Run(fmt.Sprintf("row=%d", i), func(t *testing.T) {
			t.Setenv("CONTEXT7_API_KEY", row.apiKeyEnv)

			root := newRootCmd()
			args := []string{}
			if row.apiKeyFlag != "" {
				args = append(args, "--api-key", row.apiKeyFlag)
			}
			if row.jsonFlag {
				args = append(args, "--json")
			}
			if row.formatFlag != "" {
				args = append(args, "--format", row.formatFlag)
			}
			if row.noCacheFlag {
				args = append(args, "--no-cache")
			}
			args = append(args, "version") // any leaf will do; we only inspect flags
			root.SetArgs(args)
			// We don't run RunE for a real command here — we just need
			// flags parsed. Use ParseFlags so version doesn't print.
			if err := root.ParseFlags(args); err != nil {
				t.Fatalf("parse: %v", err)
			}

			if got := resolveAPIKey(root); got != row.wantKey {
				t.Errorf("resolveAPIKey: got %q want %q", got, row.wantKey)
			}
			if got := resolveFormat(root, output.FormatMarkdown); got != row.wantFormat {
				t.Errorf("resolveFormat: got %v want %v", got, row.wantFormat)
			}
			if got := noCacheFlag(root); got != row.noCacheFlag {
				t.Errorf("noCacheFlag: got %v want %v", got, row.noCacheFlag)
			}
		})
	}
}

type pictCell struct {
	paramA, paramB int
	valueA, valueB string
}

// TestPICT_PairwiseCoverage asserts that pictRows actually achieves
// pairwise coverage. If someone trims the set, this test fires before
// the behavior tests give a false sense of safety.
func TestPICT_PairwiseCoverage(t *testing.T) {
	t.Parallel()

	levels := [][]string{
		{"", "ctx7sk-flag"},
		{"", "ctx7sk-env"},
		{"false", "true"},
		{"", "text", "md", "json", "garbage"},
		{"false", "true"},
	}
	rowVals := func(r pictRow) []string {
		return []string{
			r.apiKeyFlag,
			r.apiKeyEnv,
			fmt.Sprintf("%v", r.jsonFlag),
			r.formatFlag,
			fmt.Sprintf("%v", r.noCacheFlag),
		}
	}

	required := map[pictCell]struct{}{}
	for i := 0; i < len(levels); i++ {
		for j := i + 1; j < len(levels); j++ {
			for _, va := range levels[i] {
				for _, vb := range levels[j] {
					required[pictCell{i, j, va, vb}] = struct{}{}
				}
			}
		}
	}
	for _, r := range pictRows {
		vs := rowVals(r)
		for i := 0; i < len(vs); i++ {
			for j := i + 1; j < len(vs); j++ {
				delete(required, pictCell{i, j, vs[i], vs[j]})
			}
		}
	}
	if len(required) != 0 {
		t.Errorf("pictRows misses %d pair(s):", len(required))
		for k := range required {
			t.Logf("  missing: P%d=%q × P%d=%q", k.paramA, k.valueA, k.paramB, k.valueB)
		}
	}
}
