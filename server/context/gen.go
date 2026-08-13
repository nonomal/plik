//go:build ignore

// gen.go generates getter/setter methods for the Context struct.
//
// Usage:
//   go generate ./server/context/
//
// Or directly:
//   go run gen.go > context.go

package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"unicode"
)

type field struct {
	Name     string
	Type     string
	Panic    bool
	Internal bool
}

func (f field) UCName() string {
	r := []rune(f.Name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (f field) IsBool() bool {
	return f.Type == "bool"
}

func (f field) SetterSuffix() string {
	uc := f.UCName()
	if f.IsBool() {
		return strings.TrimPrefix(uc, "Is")
	}
	return uc
}

var fields = []field{
	{"config", "*common.Configuration", true, false},
	{"logger", "*logger.Logger", true, false},
	{"metadataBackend", "*metadata.Backend", true, false},
	{"dataBackend", "data.Backend", true, false},
	{"streamBackend", "data.Backend", true, false},
	{"authenticator", "*common.SessionAuthenticator", true, false},
	{"metrics", "*common.PlikMetrics", true, false},
	{"pagingQuery", "*common.PagingQuery", true, false},
	{"sourceIP", "net.IP", false, false},
	{"upload", "*common.Upload", false, false},
	{"file", "*common.File", false, false},
	{"user", "*common.User", false, false},
	{"originalUser", "*common.User", false, true},
	{"token", "*common.Token", false, false},
	{"isWhitelisted", "*bool", false, true},
	{"isRedirectOnFailure", "bool", false, false},
	{"isQuick", "bool", false, false},
	{"req", "*http.Request", false, false},
	{"resp", "http.ResponseWriter", false, false},
	{"mu", "sync.RWMutex", false, true},
}

// maxNameLen returns the longest field name length for struct alignment.
func maxNameLen() int {
	max := 0
	for _, f := range fields {
		if len(f.Name) > max {
			max = len(f.Name)
		}
	}
	return max
}

// alignedFields returns fields with a Pad string for struct column alignment.
type alignedField struct {
	field
	Pad string
}

func getAlignedFields() []alignedField {
	max := maxNameLen()
	result := make([]alignedField, len(fields))
	for i, f := range fields {
		pad := strings.Repeat(" ", max-len(f.Name))
		result[i] = alignedField{f, pad}
	}
	return result
}

const tmpl = `package context

import (
	"net"
	"net/http"
	"sync"

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/plik/server/metadata"
)

// Context to be propagated throughout the middleware chain
type Context struct {
{{- range .Fields}}
	{{.Name}}{{.Pad}} {{.Type}}
{{- end}}
}
{{range .Fields}}{{if not .Internal}}
{{- if .IsBool}}
// {{.UCName}} get {{.Name}} from the context.
func (ctx *Context) {{.UCName}}() {{.Type}} {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.{{.Name}}
}
{{- else}}
// Get{{.UCName}} get {{.Name}} from the context.
func (ctx *Context) Get{{.UCName}}() {{.Type}} {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
{{if .Panic}}
	if ctx.{{.Name}} == nil {
		panic("missing {{.Name}} from context")
	}
{{end}}
	return ctx.{{.Name}}
}
{{- end}}

// Set{{.SetterSuffix}} set {{.Name}} in the context
func (ctx *Context) Set{{.SetterSuffix}}({{.Name}} {{.Type}}) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.{{.Name}} = {{.Name}}
}
{{end}}{{end}}
`

func main() {
	data := struct {
		Fields []alignedField
	}{
		Fields: getAlignedFields(),
	}

	t := template.Must(template.New("context").Parse(tmpl))
	if err := t.Execute(os.Stdout, data); err != nil {
		fmt.Fprintf(os.Stderr, "template error: %s\n", err)
		os.Exit(1)
	}
}
