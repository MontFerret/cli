package module

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"path"
	"strconv"
	"strings"
	"unicode"
)

func rewriteComposition(target *composition, id, packagePath string, historicalPackages map[string]struct{}) (*compositionRewrite, error) {
	aliases, paths, err := compositionImports(target.File)
	if err != nil {
		return nil, err
	}

	registered := registeredModulePackages(target.Call, target.CoreAlias, aliases)
	if _, exists := registered[packagePath]; exists {
		return &compositionRewrite{Source: target.Source, Registered: true}, nil
	}

	for registeredPath := range registered {
		if registeredPath == packagePath {
			continue
		}
		if _, historical := historicalPackages[registeredPath]; historical {
			return nil, fmt.Errorf(
				"module %s is registered from historical package path %s, but the selected release uses %s; migrate the existing registration manually",
				id,
				registeredPath,
				packagePath,
			)
		}
	}

	moduleAlias := paths[packagePath]
	if moduleAlias == "." || moduleAlias == "_" {
		return nil, fmt.Errorf("module package %s is imported as %q in %s; use a normal import before installing", packagePath, moduleAlias, target.Filename)
	}

	if moduleAlias == "" {
		moduleAlias, err = deterministicModuleAlias(id, packagePath, target.File, aliases)
		if err != nil {
			return nil, err
		}

		addModuleImport(target.File, moduleAlias, packagePath)
	}

	moduleOption := &ast.CallExpr{
		Fun: &ast.SelectorExpr{X: ast.NewIdent(target.CoreAlias), Sel: ast.NewIdent("WithModules")},
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{X: ast.NewIdent(moduleAlias), Sel: ast.NewIdent("New")},
		}},
	}

	if target.Call.Ellipsis.IsValid() {
		if len(target.Call.Args) != 1 {
			return nil, fmt.Errorf("unsupported variadic ferret.New call in %s", target.Filename)
		}

		sliceCopy := &ast.CallExpr{
			Fun: ast.NewIdent("append"),
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun:  &ast.ArrayType{Elt: &ast.SelectorExpr{X: ast.NewIdent(target.CoreAlias), Sel: ast.NewIdent("Option")}},
					Args: []ast.Expr{ast.NewIdent("nil")},
				},
				target.Call.Args[0],
			},
			Ellipsis: token.Pos(1),
		}
		target.Call.Args = []ast.Expr{&ast.CallExpr{
			Fun:  ast.NewIdent("append"),
			Args: []ast.Expr{sliceCopy, moduleOption},
		}}
		target.Call.Ellipsis = token.Pos(1)
	} else {
		target.Call.Args = append(target.Call.Args, moduleOption)
	}

	var output bytes.Buffer
	if err := format.Node(&output, target.FileSet, target.File); err != nil {
		return nil, fmt.Errorf("format updated composition %s: %w", target.Filename, err)
	}

	return &compositionRewrite{Source: output.Bytes(), Changed: !bytes.Equal(output.Bytes(), target.Source)}, nil
}

func compositionImports(file *ast.File) (map[string]string, map[string]string, error) {
	aliases := make(map[string]string, len(file.Imports))
	paths := make(map[string]string, len(file.Imports))

	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("decode import %s: %w", spec.Path.Value, err)
		}

		alias := path.Base(importPath)
		if spec.Name != nil {
			alias = spec.Name.Name
		}

		aliases[alias] = importPath
		paths[importPath] = alias
	}

	return aliases, paths, nil
}

func registeredModulePackages(call *ast.CallExpr, coreAlias string, aliases map[string]string) map[string]struct{} {
	registered := make(map[string]struct{})

	ast.Inspect(call, func(node ast.Node) bool {
		option, ok := node.(*ast.CallExpr)
		if !ok || !isSelectorCall(option, coreAlias, "WithModules") {
			return true
		}

		for _, argument := range option.Args {
			constructor, ok := argument.(*ast.CallExpr)
			if !ok || len(constructor.Args) != 0 || constructor.Ellipsis.IsValid() {
				continue
			}

			selector, ok := constructor.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "New" {
				continue
			}

			qualifier, ok := selector.X.(*ast.Ident)
			if !ok {
				continue
			}

			if packagePath := aliases[qualifier.Name]; packagePath != "" {
				registered[packagePath] = struct{}{}
			}
		}

		return true
	})

	return registered
}

func deterministicModuleAlias(id, packagePath string, file *ast.File, aliases map[string]string) (string, error) {
	var normalized strings.Builder
	for _, value := range id {
		if unicode.IsLetter(value) || unicode.IsDigit(value) {
			normalized.WriteRune(value)
		} else {
			normalized.WriteByte('_')
		}
	}

	digest := sha256.Sum256([]byte(id))
	alias := fmt.Sprintf("ferretmod_%s_%x", normalized.String(), digest[:6])
	if !token.IsIdentifier(alias) {
		return "", fmt.Errorf("cannot derive a valid Go import alias for registry module %q", id)
	}

	if existing := aliases[alias]; existing != "" && existing != packagePath {
		return "", fmt.Errorf("generated import alias %q for %s collides with existing import %s", alias, id, existing)
	}
	if object := file.Scope.Lookup(alias); object != nil {
		return "", fmt.Errorf("generated import alias %q for %s collides with a declaration in the composition file", alias, id)
	}

	return alias, nil
}

func addModuleImport(file *ast.File, alias, packagePath string) {
	spec := &ast.ImportSpec{
		Name: ast.NewIdent(alias),
		Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(packagePath)},
	}

	for _, declaration := range file.Decls {
		imports, ok := declaration.(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			continue
		}

		imports.Specs = append(imports.Specs, spec)
		if len(imports.Specs) > 1 && !imports.Lparen.IsValid() {
			imports.Lparen = imports.Pos()
			imports.Rparen = imports.End()
		}
		file.Imports = append(file.Imports, spec)

		return
	}

	file.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}}, file.Decls...)
	file.Imports = append(file.Imports, spec)
}
