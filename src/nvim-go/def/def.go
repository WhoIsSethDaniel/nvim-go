package def

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"nvim-go/buffer"

	"github.com/garyburd/neovim-go/vim"
	"github.com/garyburd/neovim-go/vim/plugin"
	"github.com/rogpeppe/godef/go/ast"
	"github.com/rogpeppe/godef/go/parser"
	"github.com/rogpeppe/godef/go/printer"
	"github.com/rogpeppe/godef/go/token"
	"github.com/rogpeppe/godef/go/types"
)

func init() {
	plugin.HandleCommand("Godef", &plugin.CommandOptions{}, def)
}

func def(v *vim.Vim) error {
	// take GOPATH, set types.GoPath to it if it's not empty.
	p := os.Getenv("GOPATH")
	// gb := goPath(p)
	gopath := strings.Split(p, ":")
	for i, d := range gopath {
		gopath[i] = filepath.Join(d, "src")
	}
	r := runtime.GOROOT()
	if r != "" {
		gopath = append(gopath, r+"/src")
	}
	types.GoPath = gopath

	buf, err := v.CurrentBuffer()
	if err != nil {
		return v.WriteErr("cannot get current buffer")
	}

	filename, err := v.BufferName(buf)
	if err != nil {
		return v.WriteErr("cannot get current buffer name")
	}

	searchpos, err := buffer.ByteOffset(v)
	if err != nil {
		return v.WriteErr("cannot get current buffer byte offset")
	}

	// TODO(zchee): Convert to go code
	var src interface{}
	v.Eval("getbufline(bufnr('%'), 1, '$')", src)

	pkgScope := ast.NewScope(parser.Universe)
	f, err := parser.ParseFile(types.FileSet, filename, src, 0, pkgScope)
	if f == nil {
		fail("cannot parse %s: %v", filename, err)
	}

	var o ast.Node
	switch {
	case searchpos >= 0:
		o = findIdentifier(f, searchpos)

	default:
		fmt.Fprintf(os.Stderr, "no expression or offset specified\n")
		flag.Usage()
		os.Exit(2)
	}
	switch e := o.(type) {
	case *ast.ImportSpec:
		path := importPath(e)
		pkg, err := build.Default.Import(path, "", build.FindOnly)
		if err != nil {
			fail("error finding import path for %s: %s", path, err)
		}
		fmt.Println(pkg.Dir)
	case ast.Expr:
		// add declarations from other files in the local package and try again
		pkg, err := parseLocalPackage(filename, f, pkgScope)
		if pkg == nil {
			fmt.Printf("parseLocalPackage error: %v\n", err)
		}
		// if flag.NArg() > 0 {
		// 	// Reading declarations in other files might have
		// 	// resolved the original expression.
		// 	e = parseExpr(f.Scope, flag.Arg(0)).(ast.Expr)
		// }
		if obj, _ := types.ExprType(e, types.DefaultImporter); obj != nil {
			out := types.FileSet.Position(types.DeclPos(obj))
			// done(obj, typ)
			fmt.Println(out)
			v.Command("lgetexpr '" + fmt.Sprintf("%v", out) + "'")
			v.Command("sil ll 1")
			v.Feedkeys("zz", "normal", false)
		}
		// fail("no declaration found for %v", pretty{e})
	}

	return nil
}

func findGbProject(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("project root is blank")
	}
	start := path
	for path != filepath.Dir(path) {
		root := filepath.Join(path, "src")
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				path = filepath.Dir(path)
				continue
			}
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf(`could not find project root in "%s" or its parents`, start)
}

// goPath returns a GOPATH for path p.
func goPath(p string) string {
	goPath := os.Getenv("GOPATH")
	p = filepath.Clean(p)
	for _, root := range filepath.SplitList(goPath) {
		if strings.HasPrefix(p, filepath.Join(root, "src")+string(filepath.Separator)) {
			return goPath
		}
	}

	project, err := findGbProject(p)
	if err == nil {
		return project + string(filepath.ListSeparator) +
			filepath.Join(project, "vendor") + string(filepath.ListSeparator) +
			goPath
	}

	return goPath
}

func fail(s string, a ...interface{}) {
	fmt.Fprint(os.Stderr, "godef: "+fmt.Sprintf(s, a...)+"\n")
	os.Exit(2)
}

func importPath(n *ast.ImportSpec) string {
	p, err := strconv.Unquote(n.Path.Value)
	if err != nil {
		fail("invalid string literal %q in ast.ImportSpec", n.Path.Value)
	}
	return p
}

// findIdentifier looks for an identifier at byte-offset searchpos
// inside the parsed source represented by node.
// If it is part of a selector expression, it returns
// that expression rather than the identifier itself.
//
// As a special case, if it finds an import
// spec, it returns ImportSpec.
//
func findIdentifier(f *ast.File, searchpos int) ast.Node {
	ec := make(chan ast.Node)
	found := func(startPos, endPos token.Pos) bool {
		start := types.FileSet.Position(startPos).Offset
		end := start + int(endPos-startPos)
		return start <= searchpos && searchpos <= end
	}
	go func() {
		var visit func(ast.Node) bool
		visit = func(n ast.Node) bool {
			var startPos token.Pos
			switch n := n.(type) {
			default:
				return true
			case *ast.Ident:
				startPos = n.NamePos
			case *ast.SelectorExpr:
				startPos = n.Sel.NamePos
			case *ast.ImportSpec:
				startPos = n.Pos()
			case *ast.StructType:
				// If we find an anonymous bare field in a
				// struct type, its definition points to itself,
				// but we actually want to go elsewhere,
				// so assume (dubiously) that the expression
				// works globally and return a new node for it.
				for _, field := range n.Fields.List {
					if field.Names != nil {
						continue
					}
					t := field.Type
					if pt, ok := field.Type.(*ast.StarExpr); ok {
						t = pt.X
					}
					if id, ok := t.(*ast.Ident); ok {
						if found(id.NamePos, id.End()) {
							ec <- parseExpr(f.Scope, id.Name)
							runtime.Goexit()
						}
					}
				}
				return true
			}
			if found(startPos, n.End()) {
				ec <- n
				runtime.Goexit()
			}
			return true
		}
		ast.Walk(FVisitor(visit), f)
		ec <- nil
	}()
	ev := <-ec
	if ev == nil {
		fmt.Fprint(os.Stderr, "godef: nil\n")
		// fail("no identifier found")
	}
	return ev
}

type orderedObjects []*ast.Object

func (o orderedObjects) Less(i, j int) bool { return o[i].Name < o[j].Name }
func (o orderedObjects) Len() int           { return len(o) }
func (o orderedObjects) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }

// func done(v *vim.Vim, obj *ast.Object, typ types.Type) token.Position {
// defer os.Exit(0)
// pos := types.FileSet.Position(types.DeclPos(obj))
// fmt.Printf("%v\n", pos)
// if typ.Kind == ast.Bad {
// 	return []
// }
// fmt.Printf("%s\n", strings.Replace(typeStr(obj, typ), "\n", "\n\t", -1))
// return pos
// }

func typeStr(obj *ast.Object, typ types.Type) string {
	switch obj.Kind {
	case ast.Fun, ast.Var:
		return fmt.Sprintf("%s %v", obj.Name, prettyType{typ})
	case ast.Pkg:
		return fmt.Sprintf("import (%s %s)", obj.Name, typ.Node.(*ast.ImportSpec).Path.Value)
	case ast.Con:
		if decl, ok := obj.Decl.(*ast.ValueSpec); ok {
			return fmt.Sprintf("const %s %v = %s", obj.Name, prettyType{typ}, pretty{decl.Values[0]})
		}
		return fmt.Sprintf("const %s %v", obj.Name, prettyType{typ})
	case ast.Lbl:
		return fmt.Sprintf("label %s", obj.Name)
	case ast.Typ:
		typ = typ.Underlying(false, types.DefaultImporter)
		return fmt.Sprintf("type %s %v", obj.Name, prettyType{typ})
	}
	return fmt.Sprintf("unknown %s %v", obj.Name, typ.Kind)
}

func parseExpr(s *ast.Scope, expr string) ast.Expr {
	n, err := parser.ParseExpr(types.FileSet, "<arg>", expr, s)
	if err != nil {
		fail("cannot parse expression: %v", err)
	}
	switch n := n.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return n
	}
	fail("no identifier found in expression")
	return nil
}

type FVisitor func(n ast.Node) bool

func (f FVisitor) Visit(n ast.Node) ast.Visitor {
	if f(n) {
		return f
	}
	return nil
}

var errNoPkgFiles = errors.New("no more package files found")

// parseLocalPackage reads and parses all go files from the
// current directory that implement the same package name
// the principal source file, except the original source file
// itself, which will already have been parsed.
//
func parseLocalPackage(filename string, src *ast.File, pkgScope *ast.Scope) (*ast.Package, error) {
	pkg := &ast.Package{src.Name.Name, pkgScope, nil, map[string]*ast.File{filename: src}}
	d, f := filepath.Split(filename)
	if d == "" {
		d = "./"
	}
	fd, err := os.Open(d)
	if err != nil {
		return nil, errNoPkgFiles
	}
	defer fd.Close()

	list, err := fd.Readdirnames(-1)
	if err != nil {
		return nil, errNoPkgFiles
	}

	for _, pf := range list {
		file := filepath.Join(d, pf)
		if !strings.HasSuffix(pf, ".go") ||
			pf == f ||
			pkgName(file) != pkg.Name {
			continue
		}
		src, err := parser.ParseFile(types.FileSet, file, nil, 0, pkg.Scope)
		if err == nil {
			pkg.Files[file] = src
		}
	}
	if len(pkg.Files) == 1 {
		return nil, errNoPkgFiles
	}
	return pkg, nil
}

// pkgName returns the package name implemented by the
// go source filename.
//
func pkgName(filename string) string {
	prog, _ := parser.ParseFile(types.FileSet, filename, nil, parser.PackageClauseOnly, nil)
	if prog != nil {
		return prog.Name.Name
	}
	return ""
}

func hasSuffix(s, suff string) bool {
	return len(s) >= len(suff) && s[len(s)-len(suff):] == suff
}

type pretty struct {
	n interface{}
}

func (p pretty) String() string {
	var b bytes.Buffer
	printer.Fprint(&b, types.FileSet, p.n)
	return b.String()
}

type prettyType struct {
	n types.Type
}

func (p prettyType) String() string {
	// TODO print path package when appropriate.
	// Current issues with using p.n.Pkg:
	//	- we should actually print the local package identifier
	//	rather than the package path when possible.
	//	- p.n.Pkg is non-empty even when
	//	the type is not relative to the package.
	return pretty{p.n.Node}.String()
}
