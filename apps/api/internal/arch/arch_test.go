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
//	"legacy/<package>"        the flat layer-first packages being retired
type zone string

// kind is the zone's architectural role: domain, platform, transport, app,
// infra or legacy.
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

	switch from.kind() {
	case "transport", "app":
		return to.kind() == "app" && to.module() == from.module()
	case "infra":
		// Its own use cases, to implement their ports.
		return to.kind() == "app" && to.module() == from.module()
	case "legacy":
		return contains(legacyAllowed[from], to)
	}
	return false
}

// legacyAllowed keeps the dependency direction of the flat packages until each
// capability moves into a module.
var legacyAllowed = map[zone][]zone{
	"legacy/handler":    {"legacy/service", "legacy/export", "legacy/middleware", "legacy/config"},
	"legacy/service":    {"legacy/export", "legacy/config"},
	"legacy/repository": {"legacy/config"},
	"legacy/adapter":    {"legacy/service", "legacy/config"},
	"legacy/export":     {"legacy/config"},
	"legacy/middleware": {"legacy/config"},
	"legacy/config":     {},
}

// externalConfinement lists third-party prefixes and the zones that may import
// them, per layering-rules.md section 2.
var externalConfinement = []struct {
	prefix string
	allow  func(z zone) bool
}{
	{"github.com/gin-gonic/gin", func(z zone) bool {
		return z.kind() == "transport" || z == "legacy/handler" || z == "legacy/middleware" ||
			z == "platform/httpx" || z == "platform/auth" || z == "platform/logging"
	}},
	{"github.com/jackc/pgx", func(z zone) bool {
		return z.kind() == "infra" || z == "legacy/repository" || z == "platform/pgxdb"
	}},
	{"github.com/xuri/excelize", func(z zone) bool {
		return (z.kind() == "infra" && z.module() == "reporting") || z == "legacy/export"
	}},
	{"google.golang.org/api", func(z zone) bool {
		return (z.kind() == "infra" && z.module() == "formsync") || z == "legacy/adapter"
	}},
}

// domainAllowedExternal are the only non-stdlib imports internal/domain may use.
// Both are value types rather than infrastructure clients.
var domainAllowedExternal = []string{"golang.org/x/text", "github.com/google/uuid"}

// baseline freezes the violations that existed when these rules were
// introduced. Keys are "<path under internal/>|<violated import>"; values name
// the phase that retires them. The map may only shrink: an entry that no longer
// occurs fails the test, so the fixing commit also deletes its entry.
var baseline = map[string]string{
	// Handlers reaching past the application boundary, retired with the
	// module that owns each capability.

	// Use cases holding a concrete repository instead of a port. Each is
	// retired when its capability gains ports.go.
	"service/attendance_service.go|legacy/repository":   "module migration",
	"service/driver_service.go|legacy/repository":       "module migration",
	"service/site_service.go|legacy/repository":         "module migration",
	"service/vehicle_service.go|legacy/repository":      "module migration",
	"service/audit_service.go|legacy/repository":        "module migration",
	"service/dashboard_service.go|legacy/repository":    "module migration",
	"service/form_service.go|legacy/repository":         "module migration",
	"service/fuel_service.go|legacy/repository":         "module migration",
	"service/holiday_service.go|legacy/repository":      "module migration",
	"service/import_service.go|legacy/repository":       "module migration",
	"service/maintenance_service.go|legacy/repository":  "module migration",
	"service/master_service.go|legacy/repository":       "module migration",
	"service/notification_service.go|legacy/repository": "module migration",
	"service/precheck_service.go|legacy/repository":     "module migration",
	"service/region_service.go|legacy/repository":       "module migration",
	"service/report_service.go|legacy/repository":       "module migration",
	"service/ride_service.go|legacy/repository":         "module migration",
	"service/task_service.go|legacy/repository":         "module migration",
	"service/form_service.go|legacy/adapter":            "phase 5 (formsync)",

	// Excel rendering outside the reporting boundary.
	"service/import_service.go|github.com/xuri/excelize": "phase 4",
}

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
		if z.kind() == "transport" || z == "legacy/handler" {
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
		if z.kind() != "transport" && z != "legacy/handler" {
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
	return name == "repository" || strings.HasSuffix(name, "infra")
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
		return zone("legacy/" + parts[0])
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

func contains(zs []zone, z zone) bool {
	for _, v := range zs {
		if v == z {
			return true
		}
	}
	return false
}
