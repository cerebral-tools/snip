// Package snip re-exports the declarative filter pipeline as a public library
// surface for downstream Go consumers.
//
// This file is a pure aliasing layer: every identifier is a type alias, a
// function wrapper, or a var binding that forwards to an internal package. No
// logic is copied or reimplemented, and no behavior is changed.
//
// Go's internal rule blocks direct imports of
// github.com/edouard-claude/snip/internal/..., so this thin re-export layer is
// the only supported mechanism for library use.
package snip

import (
	"sync"

	"github.com/edouard-claude/snip/internal/engine"
	"github.com/edouard-claude/snip/internal/filter"
	"github.com/edouard-claude/snip/internal/trust"
)

// Filter is the declarative filter document. Re-exported from internal/filter.
type Filter = filter.Filter

// FilterTest is an inline test case. Re-exported from internal/filter.
type FilterTest = filter.FilterTest

// Match defines which command a filter applies to. Re-exported from internal/filter.
type Match = filter.Match

// MatchSubcommand preserves whether match.subcommand was omitted while
// accepting the legacy scalar YAML form and the list form.
// Re-exported from internal/filter.
type MatchSubcommand = filter.MatchSubcommand

// Inject defines args to inject before execution. Re-exported from internal/filter.
type Inject = filter.Inject

// Action represents a single step in a filter pipeline.
// Re-exported from internal/filter.
type Action = filter.Action

// Pipeline is an ordered sequence of actions. Re-exported from internal/filter.
type Pipeline = filter.Pipeline

// ActionResult is the data passed between pipeline actions.
// Re-exported from internal/filter.
type ActionResult = filter.ActionResult

// ActionFunc is the signature for built-in action implementations.
// Re-exported from internal/filter.
type ActionFunc = filter.ActionFunc

// Registry holds loaded filters indexed for fast command matching.
// Re-exported from internal/filter.
type Registry = filter.Registry

// Store maps absolute file paths to their SHA-256 hashes.
// Re-exported from internal/trust.
type Store = trust.Store

// NewSubcommand returns an explicit single subcommand match for direct Go use.
// Re-exported from internal/filter.
func NewSubcommand(value string) MatchSubcommand {
	return filter.NewSubcommand(value)
}

// NewSubcommands returns an explicit multi-subcommand match for direct Go use.
// Re-exported from internal/filter.
func NewSubcommands(values ...string) MatchSubcommand {
	return filter.NewSubcommands(values...)
}

// ParseFilter parses YAML bytes into a Filter. Re-exported from internal/filter.
func ParseFilter(data []byte) (*Filter, error) {
	return filter.ParseFilter(data)
}

// ValidateFilter checks required fields and action validity.
// Re-exported from internal/filter.
func ValidateFilter(f *Filter) error {
	return filter.ValidateFilter(f)
}

// GetAction returns the ActionFunc for the given action name.
// Re-exported from internal/filter.
func GetAction(name string) (ActionFunc, bool) {
	return filter.GetAction(name)
}

// NewRegistry builds a registry from a list of filters.
// Re-exported from internal/filter.
func NewRegistry(filters []Filter) *Registry {
	return filter.NewRegistry(filters)
}

// LoadEmbedded loads all embedded YAML filter files. Re-exported from internal/filter.
func LoadEmbedded() ([]Filter, error) {
	return filter.LoadEmbedded()
}

// LoadUserFilters loads all YAML files from a directory.
// Re-exported from internal/filter.
func LoadUserFilters(dir string) ([]Filter, error) {
	return filter.LoadUserFilters(dir)
}

// LoadAll loads filters from multiple user directories and embedded filters,
// merging by name. Re-exported from internal/filter.
func LoadAll(userDirs []string) ([]Filter, error) {
	return filter.LoadAll(userDirs)
}

// LoadAllWithStore loads filters like LoadAll but accepts an explicit trust
// store. Re-exported from internal/filter.
func LoadAllWithStore(userDirs []string, store Store) ([]Filter, error) {
	return filter.LoadAllWithStore(userDirs, store)
}

// ApplyPipeline executes filter actions sequentially. Re-exported from
// internal/engine as a documented function wrapper.
func ApplyPipeline(f *Filter, input string) (string, error) {
	return engine.ApplyPipeline(f, input)
}

// loadEmbeddedOnce wires filter.EmbeddedFS to the embedded filter set exactly
// once, safe under concurrent calls.
var loadEmbeddedOnce sync.Once

// LoadFilters is the required entry point for loading filters as a library
// consumer.
//
// filter.LoadAll reads filter.EmbeddedFS, which only Run assigns (via
// embed.go's EmbeddedFilters). A library consumer calling LoadAll directly
// would get zero embedded filters and no error — a silent empty result.
// LoadFilters assigns filter.EmbeddedFS = &EmbeddedFilters before delegating
// to filter.LoadAll(userDirs), matching what Run does, then merges in any
// user-supplied directories.
//
// The embedded-wiring assignment is idempotent and safe under concurrent calls
// (sync.Once).
func LoadFilters(userDirs []string) ([]Filter, error) {
	loadEmbeddedOnce.Do(func() {
		fs := EmbeddedFilters
		filter.EmbeddedFS = &fs
	})
	return filter.LoadAll(userDirs)
}
