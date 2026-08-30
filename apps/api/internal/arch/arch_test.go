// Package arch enforces the rules in
// .agents/skills/backend-architecture/references/layering-rules.md.
// It holds tests only; no production code imports it.
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "ltc-system/apps/api"

// internalRoot is apps/api/internal, relative to this package directory.
const internalRoot = ".."

// zone is the architectural position of a file, one of:
//
//	"domain"                  shared business kernel
//	"platform/<name>"         shared technical kernel
//	"mod/<module>/<segment>"  transport, app or infra of one capability module
type zone string

// kind is the zone's architectural role: domain, platform, transport, app or
// infra.
func (z zone) kind() string {
	parts := strings.Split(string(z), "/")
	if parts[0] == "mod" {
		return parts[2]
	}
	return parts[0]
}

func (z zone) module() string {
	parts := strings.Split(string(z), "/")
	if parts[0] == "mod" {
		return parts[1]
	}
	return ""
}

// allowedInternal reports whether a file in `from` may import `to`,
// per layering-rules.md section 2.
func allowedInternal(from, to zone) bool {
	if from == to {
		return true
	}

	switch to.kind() {
	case "domain":
		return true
	case "platform":
		return from.kind() != "domain"
	}

	// A module segment may only reach its own use cases; cross-module traffic
	// goes through a port that cmd/server injects.
	switch from.kind() {
	case "transport", "app", "infra":
		return to.kind() == "app" && to.module() == from.module()
	}
	return false
}

// externalConfinement lists third-party prefixes and the zones that may import
// them, per layering-rules.md section 2.
var externalConfinement = []struct {
	prefix string
	allow  func(z zone) bool
}{
	{"github.com/gin-gonic/gin", func(z zone) bool {
		return z.kind() == "transport" ||
			z == "platform/httpx" || z == "platform/auth" || z == "platform/logging"
	}},
	{"github.com/jackc/pgx", func(z zone) bool {
		return z.kind() == "infra" || z == "platform/pgxdb"
	}},
	{"github.com/xuri/excelize", func(z zone) bool {
		// The spreadsheet SDK stays in the infra boundary that renders or decodes files.
		return z.kind() == "infra" && excelModules[z.module()]
	}},
}

// excelModules own a capability whose deliverable is a spreadsheet file.
var excelModules = map[string]bool{
	"reporting":    true,
	"caseimport":   true,
	"ops":          true,
	"casemgmt":     true,
	"caregiver":    true,
	"driverreport": true,
}

// domainAllowedExternal are the only non-stdlib imports internal/domain may use.
// Both are value types rather than infrastructure clients.
var domainAllowedExternal = []string{"golang.org/x/text", "github.com/google/uuid"}

// baseline froze the violations that existed when these rules were introduced.
// The migration into internal/modules/ retired every one of them, so it is now
// empty and must stay empty: a new violation is a defect to fix, not an entry to
// add. Keys are "<path under internal/>|<violated import>".
var baseline = map[string]string{}

func TestImportMatrix(t *testing.T) {
	seen := map[string]bool{}

	forEachSourceFile(t, func(rel string, z zone, f *ast.File) {
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			violation := ""

			switch {
			case strings.HasPrefix(p, modulePath+"/internal/"):
				target := zoneOf(strings.TrimPrefix(p, modulePath+"/internal/"))
				if target == "" || allowedInternal(z, target) {
					continue
				}
				violation = string(target)

			case isStdlib(p):
				continue

			default:
				if z.kind() == "domain" && !hasAnyPrefix(p, domainAllowedExternal) {
					violation = p
					break
				}
				for _, c := range externalConfinement {
					if strings.HasPrefix(p, c.prefix) && !c.allow(z) {
						violation = c.prefix
					}
				}
			}

			if violation == "" {
				continue
			}
			key := rel + "|" + violation
			seen[key] = true
			if _, frozen := baseline[key]; frozen {
				continue
			}
			t.Errorf("%s (zone %q) must not import %q -- see layering-rules.md section 2", rel, z, p)
		}
	})

	reportRetired(t, "baseline", baseline, seen)
}

// bindingTagBaseline freezes gin validator tags that still sit outside the
// transport boundary.
var bindingTagBaseline = map[string]string{}

func TestBindingTagsStayInTransport(t *testing.T) {
	seen := map[string]bool{}

	forEachSourceFile(t, func(rel string, z zone, f *ast.File) {
		if z.kind() == "transport" {
			return
		}
		ast.Inspect(f, func(n ast.Node) bool {
			field, ok := n.(*ast.Field)
			if !ok || field.Tag == nil || !strings.Contains(field.Tag.Value, `binding:"`) {
				return true
			}
			if seen[rel] {
				return true
			}
			seen[rel] = true
			if _, frozen := bindingTagBaseline[rel]; frozen {
				return true
			}
			t.Errorf("%s carries a binding tag outside the transport boundary -- see layering-rules.md section 3", rel)
			return true
		})
	})

	reportRetired(t, "bindingTagBaseline", bindingTagBaseline, seen)
}

// persistenceBindBaseline freezes handlers that still bind a request body
// straight into a persistence row struct.
var persistenceBindBaseline = map[string]string{}

func TestRequestBodiesUseTransportDTOs(t *testing.T) {
	seen := map[string]bool{}

	forEachSourceFile(t, func(rel string, z zone, f *ast.File) {
		if z.kind() != "transport" {
			return
		}

		persistenceVars := map[string]string{}
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok || spec.Type == nil {
				return true
			}
			sel, ok := spec.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || !isPersistencePackage(pkg.Name) {
				return true
			}
			for _, name := range spec.Names {
				persistenceVars[name.Name] = pkg.Name + "." + sel.Sel.Name
			}
			return true
		})
		if len(persistenceVars) == 0 {
			return
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(sel.Sel.Name, "Bind") && !strings.HasPrefix(sel.Sel.Name, "ShouldBind") {
				return true
			}
			unary, ok := call.Args[0].(*ast.UnaryExpr)
			if !ok || unary.Op != token.AND {
				return true
			}
			ident, ok := unary.X.(*ast.Ident)
			if !ok {
				return true
			}
			typeName, ok := persistenceVars[ident.Name]
			if !ok {
				return true
			}
			if seen[rel] {
				return true
			}
			seen[rel] = true
			if _, frozen := persistenceBindBaseline[rel]; frozen {
				return true
			}
			t.Errorf("%s binds a request body into %s -- see layering-rules.md section 3", rel, typeName)
			return true
		})
	})

	reportRetired(t, "persistenceBindBaseline", persistenceBindBaseline, seen)
}

// reportRetired fails when a frozen entry no longer occurs, so that fixing a
// violation and shrinking the baseline happen in the same commit.
func reportRetired(t *testing.T, name string, frozen map[string]string, seen map[string]bool) {
	t.Helper()
	for key, phase := range frozen {
		if !seen[key] {
			t.Errorf("%s entry %q (%s) no longer occurs -- delete it from arch_test.go", name, key, phase)
		}
	}
}

func isPersistencePackage(name string) bool {
	return strings.HasSuffix(name, "infra")
}

// zoneOf maps a path under internal/ to its zone, or "" for paths the rules do
// not cover.
func zoneOf(relPath string) zone {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	switch parts[0] {
	case "arch":
		return ""
	case "domain":
		return "domain"
	case "platform":
		if len(parts) < 2 {
			return ""
		}
		return zone("platform/" + parts[1])
	case "modules":
		if len(parts) < 3 {
			return ""
		}
		return zone("mod/" + parts[1] + "/" + parts[2])
	default:
		return ""
	}
}

func forEachSourceFile(t *testing.T, fn func(rel string, z zone, f *ast.File)) {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(internalRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", internalRoot, err)
	}
	sort.Strings(paths)

	fset := token.NewFileSet()
	for _, path := range paths {
		rel, err := filepath.Rel(internalRoot, path)
		if err != nil {
			t.Fatalf("rel %s: %v", path, err)
		}
		rel = filepath.ToSlash(rel)
		z := zoneOf(rel)
		if z == "" {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}
		fn(rel, z, f)
	}
}

func isStdlib(importPath string) bool {
	return !strings.Contains(strings.SplitN(importPath, "/", 2)[0], ".")
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
