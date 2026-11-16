package sill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/enyhma/hearth"
)

// LoadVault walks dir for markdown notes — via hearth's frontmatter
// parsing and parallel directory scan — and adapts each into an
// EvalContext, all sharing one Vault built from hearth's backlink graph
// so file(), link.asFile(), and link.linksTo() can resolve against any
// other note in the same load, not just "this" one. See
// landmarks/design/engineering/DEC-NqI23Hxjzx.
//
// A note that fails to parse (e.g. an Obsidian template file with
// Templater/Dataview placeholder syntax that isn't valid YAML) is
// skipped and its error returned alongside the rest, not fatal to the
// whole load — see hearth's DEC-LSh8yPXvpn, which this inherits.
func LoadVault(dir string) ([]*EvalContext, []error) {
	notes, errs := hearth.ParseDir([]string{dir})

	// path must agree with buildFileRef's rel path — it's the graph's key
	// space, and Vault.Resolve/LinksTo callers (functions.go) only ever
	// have rel paths (fileRef.Path, FileMeta.Path) to query it with.
	path := func(n *hearth.Note) string { return relPath(n, dir) }
	body := func(n *hearth.Note) string { return n.Body }
	identity := func(n *hearth.Note) []string { return noteIdentity(n, dir) }

	graph := hearth.BuildGraph(notes, path, body, identity)
	vault := &Vault{
		LinksTo:   graph.LinksTo,
		Backlinks: graph.In,
		Resolve: func(query string) (fileRef, bool) {
			matches := hearth.Match(notes, identity, query)
			if len(matches) != 1 {
				return fileRef{}, false
			}
			fr, err := buildFileRef(matches[0], dir)
			if err != nil {
				return fileRef{}, false
			}
			return fr, true
		},
	}

	var ctxs []*EvalContext
	for _, n := range notes {
		fr, err := buildFileRef(n, dir)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ctxs = append(ctxs, &EvalContext{Note: fr.Note, File: fr.File, Vault: vault})
	}
	return ctxs, errs
}

// relPath is the vault-relative path a note is keyed by everywhere in
// sill — hearth.BuildGraph's node keys, fileRef.Path, and FileMeta.Path
// all have to agree, or Vault.LinksTo/Resolve silently never match.
func relPath(n *hearth.Note, vaultDir string) string {
	rel, err := filepath.Rel(vaultDir, n.Path)
	if err != nil {
		return n.Path
	}
	return rel
}

// noteIdentity is the string set a note can be addressed by: its
// vault-relative path, its absolute path, and its bare filename — the
// same identity hearth.BuildGraph uses to resolve [[wikilink]] targets,
// so file("Some Note") and a [[Some Note]] wikilink resolve consistently
// to the same note.
func noteIdentity(n *hearth.Note, vaultDir string) []string {
	basename := strings.TrimSuffix(filepath.Base(n.Path), filepath.Ext(n.Path))
	return []string{relPath(n, vaultDir), n.Path, basename}
}

func buildFileRef(n *hearth.Note, vaultDir string) (fileRef, error) {
	info, err := os.Stat(n.Path)
	if err != nil {
		return fileRef{}, err
	}
	rel := relPath(n, vaultDir)

	base := filepath.Base(n.Path)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	basename := strings.TrimSuffix(base, filepath.Ext(base))
	folder := filepath.Dir(rel)
	if folder == "." {
		folder = ""
	}

	frontmatter := n.Frontmatter
	if frontmatter == nil {
		frontmatter = map[string]any{}
	}
	links, embeds := linksAndEmbeds(n.Body)

	return fileRef{
		Path: rel,
		Note: frontmatter,
		File: FileMeta{
			Name:     base,
			Basename: basename,
			Path:     rel,
			Folder:   folder,
			Ext:      ext,
			Size:     info.Size(),
			CTime:    info.ModTime(), // ponytail: no portable creation time in os.FileInfo; mtime stands in.
			MTime:    info.ModTime(),
			Tags:     extractTags(frontmatter, n.Body),
			Links:    links,
			Embeds:   embeds,
		},
	}, nil
}

// linksAndEmbeds splits body's [[wikilink]] references into true links
// and ![[embeds]] — a distinction hearth.Extract doesn't make (it treats
// both as the same Ref, matching hearth's type-agnostic, link-syntax-
// agnostic design), but Obsidian's own file.links vs file.embeds does.
// hearth.BuildGraph's backlink graph still treats both as edges — an
// embed is still a real reference for link.linksTo()/backlinks purposes,
// Obsidian's own graph view does the same.
var embedRe = regexp.MustCompile(`!\[\[([^\]|]+)(?:\|[^\]]*)?\]\]`)

func linksAndEmbeds(body string) (links, embeds []string) {
	embedSet := make(map[string]bool)
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range embedRe.FindAllStringSubmatch(line, -1) {
			target := strings.TrimSpace(m[1])
			embeds = append(embeds, target)
			embedSet[target] = true
		}
	}
	for _, r := range hearth.Extract(body) {
		if !embedSet[r.Target] {
			links = append(links, r.Target)
		}
	}
	return links, embeds
}

var inlineTagRe = regexp.MustCompile(`#([A-Za-z0-9_/-]+)`)

// extractTags collects tags from the frontmatter tags: property (a list,
// or a comma-separated string) and from #inline tags in the body,
// skipping fenced code blocks the same way hearth.Extract does.
func extractTags(note map[string]any, body string) []string {
	var tags []string
	seen := map[string]bool{}
	add := func(t string) {
		t = strings.TrimSpace(strings.TrimPrefix(t, "#"))
		if t != "" && !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}

	switch v := note["tags"].(type) {
	case string:
		for _, t := range strings.Split(v, ",") {
			add(t)
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				add(s)
			}
		}
	}

	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range inlineTagRe.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
	}
	return tags
}

// Query evaluates base's global filters AND'd with the named view's own
// filters (view filters are concatenated with global filters via AND,
// per the Bases spec) against every note under vaultDir. An empty
// viewName selects the first view if the base has one; a base with no
// views and no viewName just applies the global filters.
//
// The returned []error are non-fatal warnings: notes that failed to
// parse (see LoadVault) or that errored while being filtered are simply
// absent from the results, not a reason to fail the whole query — a
// real .base file (kepano-obsidian's Books.base) filters on
// categories.contains(...) where most notes simply lack a categories
// property; one note's filter blowing up must not blank every other
// note's results, the same lesson as hearth's DEC-LSh8yPXvpn. The final
// error return is reserved for a genuinely fatal problem not tied to any
// one note (an unknown --view name).
func Query(base *Base, vaultDir, viewName string) ([]*EvalContext, []error, error) {
	notes, warnings := LoadVault(vaultDir)
	for _, ctx := range notes {
		ctx.Formulas = base.Formulas
	}

	view, err := selectView(base, viewName)
	if err != nil {
		return nil, warnings, err
	}

	var out []*EvalContext
	for _, ctx := range notes {
		ok, err := EvalFilter(base.Filters, ctx)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("%s: %w", ctx.File.Path, err))
			continue
		}
		if !ok {
			continue
		}
		if view != nil {
			ok, err = EvalFilter(view.Filters, ctx)
			if err != nil {
				warnings = append(warnings, fmt.Errorf("%s: %w", ctx.File.Path, err))
				continue
			}
			if !ok {
				continue
			}
		}
		out = append(out, ctx)
	}
	return out, warnings, nil
}

func selectView(base *Base, viewName string) (*View, error) {
	if viewName != "" {
		for i := range base.Views {
			if base.Views[i].Name == viewName {
				return &base.Views[i], nil
			}
		}
		return nil, fmt.Errorf("no view named %q", viewName)
	}
	if len(base.Views) > 0 {
		return &base.Views[0], nil
	}
	return nil, nil
}

// Lint evaluates base's sillAssert sidecar against every note under
// vaultDir. Returns an error if base has no sillAssert extension. The
// returned []error are non-fatal per-note load warnings, same contract
// as Query.
func Lint(base *Base, vaultDir string) (*AssertionResult, []error, error) {
	a, err := ParseAssertion(base)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, fmt.Errorf("base has no %s extension to lint", sillAssertKey)
	}
	notes, warnings := LoadVault(vaultDir)
	for _, ctx := range notes {
		ctx.Formulas = base.Formulas
	}
	res, evalWarnings := EvalAssertion(a, notes)
	return res, append(warnings, evalWarnings...), nil
}
