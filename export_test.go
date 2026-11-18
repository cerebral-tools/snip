//go:build !lite

package snip

import (
	"testing"

	"github.com/edouard-claude/snip/internal/trust"
)

// TestLoadFilters_EmbeddedWiring verifies that LoadFilters correctly wires the
// embedded filesystem and returns a non-empty slice of filters. This is the
// regression test for the EmbeddedFS wiring: if the sync.Once assignment is
// ever dropped, this test will fail with an empty slice.
func TestLoadFilters_EmbeddedWiring(t *testing.T) {
	filters, err := LoadFilters(nil)
	if err != nil {
		t.Fatalf("LoadFilters(nil) returned error: %v", err)
	}
	if len(filters) == 0 {
		t.Fatal("LoadFilters(nil) returned empty slice; embedded filters not wired")
	}
	for _, f := range filters {
		if f.Name == "" {
			t.Errorf("filter with empty Name: %+v", f)
		}
		if len(f.Pipeline) == 0 {
			t.Errorf("filter %q has empty Pipeline", f.Name)
		}
	}
}

// TestLoadFilters_Idempotent verifies that calling LoadFilters multiple times
// is safe and produces the same result.
func TestLoadFilters_Idempotent(t *testing.T) {
	filters1, err := LoadFilters(nil)
	if err != nil {
		t.Fatalf("first LoadFilters: %v", err)
	}
	filters2, err := LoadFilters(nil)
	if err != nil {
		t.Fatalf("second LoadFilters: %v", err)
	}
	if len(filters1) != len(filters2) {
		t.Errorf("LoadFilters not idempotent: %d vs %d filters", len(filters1), len(filters2))
	}
}

// TestParseFilter_RoundTrip verifies that ParseFilter parses a minimal YAML
// document and produces a Filter whose Pipeline is reachable through the
// exported alias.
func TestParseFilter_RoundTrip(t *testing.T) {
	yamlDoc := []byte(`
name: test-filter
match:
  command: testcmd
pipeline:
  - action: keep_lines
    pattern: "foo"
`)
	f, err := ParseFilter(yamlDoc)
	if err != nil {
		t.Fatalf("ParseFilter: %v", err)
	}
	if f.Name != "test-filter" {
		t.Errorf("Name = %q, want %q", f.Name, "test-filter")
	}
	if len(f.Pipeline) != 1 {
		t.Errorf("Pipeline length = %d, want 1", len(f.Pipeline))
	}
	if f.Pipeline[0].ActionName != "keep_lines" {
		t.Errorf("ActionName = %q, want %q", f.Pipeline[0].ActionName, "keep_lines")
	}
}

// TestNewRegistry_Match verifies that a registry built from parsed filters can
// find a filter via Match.
func TestNewRegistry_Match(t *testing.T) {
	yamlDoc := []byte(`
name: test-filter
match:
  command: testcmd
  subcommand: sub
pipeline:
  - action: head
    n: 5
`)
	f, err := ParseFilter(yamlDoc)
	if err != nil {
		t.Fatalf("ParseFilter: %v", err)
	}
	reg := NewRegistry([]Filter{*f})
	matched := reg.Match("testcmd", "sub", []string{})
	if matched == nil {
		t.Fatal("Match returned nil for exact command+subcommand")
	}
	if matched.Name != "test-filter" {
		t.Errorf("matched.Name = %q, want %q", matched.Name, "test-filter")
	}
	// A subcommand-scoped filter is indexed only under command:subcommand,
	// so a bare-command lookup correctly misses.
	matched2 := reg.Match("testcmd", "", []string{})
	if matched2 != nil {
		t.Errorf("Match returned non-nil for bare command on subcommand-scoped filter: %q", matched2.Name)
	}
	// Non-matching should return nil
	matched3 := reg.Match("othercmd", "sub", []string{})
	if matched3 != nil {
		t.Errorf("Match returned non-nil for non-matching command: %q", matched3.Name)
	}
}

// TestApplyPipeline_EndToEnd verifies the full pipeline execution with a
// keep_lines + head pipeline.
func TestApplyPipeline_EndToEnd(t *testing.T) {
	yamlDoc := []byte(`
name: test-pipeline
match:
  command: dummy
pipeline:
  - action: keep_lines
    pattern: "^keep"
  - action: head
    n: 2
`)
	f, err := ParseFilter(yamlDoc)
	if err != nil {
		t.Fatalf("ParseFilter: %v", err)
	}
	input := "keep first\nkeep second\nkeep third\ndrop this\nkeep fourth\n"
	out, err := ApplyPipeline(f, input)
	if err != nil {
		t.Fatalf("ApplyPipeline: %v", err)
	}
	// keep_lines keeps 4 lines, head caps to 2 and appends an overflow marker
	expected := "keep first\nkeep second\n+2 more lines\n"
	if out != expected {
		t.Errorf("ApplyPipeline output mismatch\ngot:      %q\nexpected: %q", out, expected)
	}
}

// TestApplyPipeline_EmptyInput verifies that empty input produces empty output
// without error.
func TestApplyPipeline_EmptyInput(t *testing.T) {
	yamlDoc := []byte(`
name: test-empty
match:
  command: dummy
pipeline:
  - action: head
    n: 5
`)
	f, err := ParseFilter(yamlDoc)
	if err != nil {
		t.Fatalf("ParseFilter: %v", err)
	}
	out, err := ApplyPipeline(f, "")
	if err != nil {
		t.Fatalf("ApplyPipeline: %v", err)
	}
	if out != "\n" {
		t.Errorf("empty input output = %q, want %q", out, "\n")
	}
}

// TestGetAction_KnownAndUnknown verifies that GetAction returns ok for a
// built-in action and false for an unknown one.
func TestGetAction_KnownAndUnknown(t *testing.T) {
	fn, ok := GetAction("head")
	if !ok {
		t.Fatal("GetAction(\"head\") returned false, want true")
	}
	if fn == nil {
		t.Fatal("GetAction(\"head\") returned nil fn")
	}
	fn2, ok2 := GetAction("no-such-action")
	if ok2 {
		t.Fatal("GetAction(\"no-such-action\") returned true, want false")
	}
	if fn2 != nil {
		t.Fatal("GetAction(\"no-such-action\") returned non-nil fn")
	}
}

// TestNewSubcommand_Constructors verifies the exported constructors produce
// expected MatchSubcommand values.
func TestNewSubcommand_Constructors(t *testing.T) {
	s1 := NewSubcommand("log")
	if !s1.IsPresent() {
		t.Error("NewSubcommand: IsPresent() = false, want true")
	}
	if len(s1.Values()) != 1 || s1.Values()[0] != "log" {
		t.Errorf("NewSubcommand values = %v, want [log]", s1.Values())
	}

	s2 := NewSubcommands("log", "commit", "push")
	if !s2.IsPresent() {
		t.Error("NewSubcommands: IsPresent() = false, want true")
	}
	if len(s2.Values()) != 3 {
		t.Errorf("NewSubcommands values length = %d, want 3", len(s2.Values()))
	}
	if s2.Values()[0] != "log" || s2.Values()[1] != "commit" || s2.Values()[2] != "push" {
		t.Errorf("NewSubcommands values = %v, want [log commit push]", s2.Values())
	}
}

// TestFilter_MethodsReachable verifies that methods on aliased types are
// reachable through the exported aliases (compile-time check).
func TestFilter_MethodsReachable(t *testing.T) {
	yamlDoc := []byte(`
name: test-methods
match:
  command: dummy
pipeline:
  - action: head
    n: 1
`)
	f, err := ParseFilter(yamlDoc)
	if err != nil {
		t.Fatalf("ParseFilter: %v", err)
	}
	// These method calls must compile; if they don't, the export is broken.
	_ = f.HasStream("stdout")
	_ = f.Clone()
	_ = f.PipelineActionNames()

	reg := NewRegistry([]Filter{*f})
	_ = reg.Match("dummy", "", []string{})
	_, _ = reg.ShouldInject(f, []string{})
	_ = reg.HasAnyFilter("dummy", "")
	_ = reg.HasAnyFilterForCommand("dummy")
	_ = reg.Commands()

	// Verify Store alias compiles and is usable
	var _ Store = make(trust.Store)
	_, _ = LoadAllWithStore(nil, nil)
}
