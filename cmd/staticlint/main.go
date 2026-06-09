// Staticlint - это инструмент для статического анализа кода, объединяющий различные анализаторы.
//
// Механизм запуска:
//  1. Скомпилируйте анализатор:
//     go build -o staticlint ./cmd/staticlint/main.go
//  2. Запустите его для проверки проекта:
//     ./staticlint ./...
//
// Состав анализаторов:
// - Стандартные анализаторы пакета golang.org/x/tools/go/analysis/passes.
// - Все анализаторы класса SA пакета staticcheck.io.
// - Все анализаторы класса S1 пакета staticcheck.io.
// - Анализатор bodyclose для проверки закрытия тел HTTP-ответов.
// - Анализатор osexit, запрещающий прямой вызов os.Exit в функции main пакета main.
package main

import (
	"go/ast"
	"os"
	"strings"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
)

func main() {

	var mychecks []*analysis.Analyzer

	// Анализаторы SA из пакета staticcheck
	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "SA") {

			// исключаем анализатор os.Exit
			if v.Analyzer.Name == "SA3000" {
				continue
			}

			mychecks = append(mychecks, v.Analyzer)
		}
	}

	// Анализаторы из passes
	mychecks = append(mychecks, printf.Analyzer, shadow.Analyzer, structtag.Analyzer, loopclosure.Analyzer, copylock.Analyzer)

	//не менее одного анализатора остальных классов пакета staticcheck.io
	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "S1") {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	//двух или более любых публичных анализаторов на ваш выбор
	mychecks = append(mychecks, errcheck.Analyzer)
	// bodyclose
	mychecks = append(mychecks, bodyclose.Analyzer)

	// os.exit
	mychecks = append(mychecks, OsExitAnalyzer)

	multichecker.Main(mychecks...)

}

// OsExitAnalyzer проверяет наличие прямого вызова функции os.Exit в функции main пакета main.
var OsExitAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "checks for direct os.Exit calls in main function",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if !strings.HasSuffix(filename, ".go") {
			continue
		}

		if strings.Contains(filename, "go-build") {
			continue
		}

		if _, err := os.Stat(filename); err != nil {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				return true
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				_, ok = sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				obj := pass.TypesInfo.Uses[sel.Sel]
				if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "os" && obj.Name() == "Exit" {
					pass.Reportf(
						sel.Pos(),
						"direct call to os.Exit in main function is prohibited",
					)
				}

				if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "log" && obj.Name() == "Fatal" {
					pass.Reportf(
						sel.Pos(),
						"direct call to log.Fatal in main function is prohibited",
					)
				}
				return true
			})

			return true
		})
	}

	return nil, nil
}
