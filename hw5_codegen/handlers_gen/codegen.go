package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

var Empty struct{}

func ConcatValues(values []string) string {
	return strings.Join(values, ", ")
}

func Render(tplName string, data interface{}) string {
	out := bytes.NewBuffer(nil)
	validatorTmps.ExecuteTemplate(out, tplName, data)
	return out.String()
}

var (
	tmplFuncs = template.FuncMap{
		"Concat": ConcatValues,
		"Render": Render,
	}

	generalTpl = template.Must(template.New("generalTpl").Funcs(tmplFuncs).Parse(`
package {{.General.PackageName}}

import (
	"net/http"
	"encoding/json"
	"fmt"
	"strconv"
	"runtime/debug"
	"net/url"
)

type ApiErrorResponse struct {
	Error string {{ .General.JSONErrorTag }}
}

type ApiSuccessResponse struct {
	Error    string      {{ .General.JSONErrorTag }}
	Response interface{} {{ .General.JSONResponseTag }}
}

var Empty struct{}

func isAuthenticated(r *http.Request) bool {
	return r.Header.Get("X-Auth") == "100500"
}

func errorResponse(status int, message string, w http.ResponseWriter) {
	res, _ := json.Marshal(ApiErrorResponse{message})
	w.WriteHeader(status)
	w.Write(res)
}

func successResponse(status int, obj interface{}, w http.ResponseWriter) {
	res, _ := json.Marshal(ApiSuccessResponse{"", obj})
	w.WriteHeader(status)
	w.Write(res)	
}

func proccessError(err error, w http.ResponseWriter) {	
	switch err.(type) {
	case ApiError:
		errorResponse((err.(ApiError)).HTTPStatus, err.Error(), w)
	default:
		errorResponse(http.StatusInternalServerError, err.Error(), w)
	}
}

func getOrDefault(values url.Values, key string, defaultValue string) string {
	items, ok := values[key]
	if !ok {
		return defaultValue
	}
	if len(items) == 0 {
		return defaultValue
	}

	return items[0]
}

{{ range $key, $value := .ServeHTTP }}
func (h *{{ $key }}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			fmt.Printf("%#v\n", err)
			errorResponse(http.StatusInternalServerError, "Internal server error", w)
		}
	}()

	switch r.URL.Path {
	{{ range $value }}
	case "{{ .Path }}":
		h.wrapper{{ .Name }}(w, r)
	{{ end }}
	default:
		errorResponse(http.StatusNotFound, "unknown method", w)
	}
}
{{ end }}

{{ range .ServeHTTP }}
	{{ range . }}
func (api *{{.Receiver}}) wrapper{{.Name}}(w http.ResponseWriter, r *http.Request) {
	{{ if .IsValidateMethod }}
	if r.Method != "{{ .Method }}" {
		errorResponse(http.StatusNotAcceptable, "bad method", w)
		return
	}
	{{ end }}
	{{ if .IsAuth }}
	ok := isAuthenticated(r)
	if !ok {
		errorResponse(http.StatusForbidden, "unauthorized", w)
		return
	}
	{{ end }}

	r.ParseForm()
	params := new({{.In}})
	err := params.fillFromForm(r.Form)
	if err != nil {
		errorResponse(http.StatusBadRequest, err.Error(), w)
		return
	}

	res, err := api.{{ .Name }}(r.Context(), *params)
	if err != nil {
		proccessError(err, w)
		return
	}

	successResponse(http.StatusOK, res, w)
}
	{{ end }}
{{ end }}

{{ range .ParamsStructures }}
func (s *{{ .Name }}) fillFromForm(params url.Values) error {
	{{ range .Fields }}
		{{ if .IsString }}
	s.{{ .FieldName }} = getOrDefault(params, "{{ .ParamName }}", "{{ .Default }}")
		{{ end }}
		{{ if .IsInt }}
	{{ .FieldName }}, err := strconv.Atoi(getOrDefault(params, "{{ .ParamName }}", "{{ .Default }}"))
	if err != nil {
		return fmt.Errorf("{{ .ParamName }} must be int")
	}
	s.{{ .FieldName }} = {{ .FieldName }}
		{{ end }}

		{{ range .Validators }}
		{{ Render .Template . }}
		{{ end }}
	{{ end }}

	return nil
}
{{ end }}
`))

	validatorTmps = template.New("validatorTmpls")

	enumValidatorTpl = template.Must(validatorTmps.New("enumValidatorTpl").Funcs(tmplFuncs).Parse(`
	{{ .FieldName }}Values := map[string]struct{}{
		{{ range $key, $value := .Values }}
			"{{ $value }}"	: Empty,
		{{ end }}
	}
	if _, ok := {{ .FieldName }}Values[s.{{ .FieldName }}]; !ok {
		return fmt.Errorf("{{ .ParamName }} must be one of [{{ Concat .Values }}]")
	}`))

	requiredValidatorTpl = template.Must(validatorTmps.New("requiredValidatorTpl").Funcs(tmplFuncs).Parse(`
	if s.{{ .FieldName }} == "" {
		return fmt.Errorf("{{ .ParamName }} must me not empty")
	}
`))

	minValidatorTpl = template.Must(validatorTmps.New("minValidatorTpl").Funcs(tmplFuncs).Parse(`
	if {{if eq .ParamType "string"}}{{printf "len(s.%s)" .FieldName}}{{else}}{{printf "s.%s" .FieldName}}{{end}} < {{.Value}} {
		return fmt.Errorf("{{.ParamName}}{{if eq .ParamType "string"}}{{" len"}}{{else}}{{end}} must be >= {{.Value}}")
	}
`))

	maxValidatorTpl = template.Must(validatorTmps.New("maxValidatorTpl").Funcs(tmplFuncs).Parse(`
	if {{if eq .ParamType "string"}}{{printf "len(s.%s)" .FieldName}}{{else}}{{printf "s.%s" .FieldName}}{{end}} > {{.Value}} {
		return fmt.Errorf("{{.ParamName}}{{if eq .ParamType "string"}}{{" len"}}{{else}}{{end}} must be <= {{.Value}}")
	}
`))
)

type TemplateVariables struct {
	General          map[string]string
	ServeHTTP        map[string][]HttpFunction
	ParamsStructures []ParamsStructure
}

type Instructions struct {
	Url    string
	Auth   bool
	Method string
}

type HttpFunction struct {
	Receiver string
	In       string
	Path     string
	Auth     bool
	Method   string
	Name     string
}

func (f *HttpFunction) IsAuth() bool {
	return f.Auth
}

func (f *HttpFunction) IsValidateMethod() bool {
	return f.Method != ""
}

type Validator interface{}

type RequiredValidator struct {
	FieldName string
	ParamName string
	Template  string
}

type MinValidator struct {
	FieldName string
	Value     string
	ParamName string
	ParamType string
	Template  string
}

type MaxValidator struct {
	FieldName string
	Value     string
	ParamName string
	ParamType string
	Template  string
}

type EnumValidator struct {
	FieldName string
	Values    []string
	ParamName string
	Template  string
}

type Field struct {
	FieldName  string
	ParamName  string
	Default    string
	ParamType  string
	Validators []Validator
}

var apiValidatorPattern = regexp.MustCompile(`".*"`)

func (f *Field) ParseApiValidator(schema string) {
	schema = strings.ReplaceAll(apiValidatorPattern.FindString(schema), "\"", "")
	rules := strings.Split(schema, ",")
	validators := make([]Validator, 0)
	for _, rule := range rules {
		switch {
		case rule == "required":
			validators = append(validators, RequiredValidator{
				FieldName: f.FieldName,
				ParamName: f.ParamName,
				Template:  "requiredValidatorTpl",
			})
		case rule[:3] == "min":
			validators = append(validators, MinValidator{
				Value:     rule[4:],
				FieldName: f.FieldName,
				ParamName: f.ParamName,
				ParamType: f.ParamType,
				Template:  "minValidatorTpl",
			})
		case rule[:3] == "max":
			validators = append(validators, MaxValidator{
				Value:     rule[4:],
				FieldName: f.FieldName,
				ParamName: f.ParamName,
				ParamType: f.ParamType,
				Template:  "maxValidatorTpl",
			})
		case rule[:4] == "enum":
			validators = append(validators, EnumValidator{
				Values:    strings.Split(rule[5:], "|"),
				FieldName: f.FieldName,
				ParamName: f.ParamName,
				Template:  "enumValidatorTpl",
			})
		case rule[:7] == "default":
			f.Default = rule[8:]
		case rule[:9] == "paramname":
			f.ParamName = rule[10:]
		}
	}
	f.Validators = validators
}

func (f *Field) IsString() bool {
	return f.ParamType == "string"
}

func (f *Field) IsInt() bool {
	return f.ParamType == "int"
}

type ParamsStructure struct {
	Name   string
	Fields []Field
}

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	// Preprocess file
	structures := map[string]ast.StructType{}
	functions := []HttpFunction{}
	for _, f := range file.Decls {
		// processing structures
		if g, ok := f.(*ast.GenDecl); ok {
			for _, spec := range g.Specs {
				if currType, ok := spec.(*ast.TypeSpec); ok {
					if currStruct, ok := currType.Type.(*ast.StructType); ok {
						structures[currType.Name.Name] = *currStruct
					}
				}
			}
		}

		// processing functions
		if fun, ok := f.(*ast.FuncDecl); ok {
			if fun.Doc == nil || len(fun.Doc.List) == 0 {
				continue
			}
			funcComment := fun.Doc.List[0].Text
			if strings.HasPrefix(funcComment, "// apigen:api ") && fun.Recv != nil {
				expr, _ := fun.Recv.List[0].Type.(*ast.StarExpr)
				identReceiver, _ := expr.X.(*ast.Ident)
				instructions := new(Instructions)
				json.Unmarshal([]byte(strings.Replace(funcComment, "// apigen:api ", "", 1)), &instructions)
				// fmt.Println(instructions)

				identIn, _ := fun.Type.Params.List[1].Type.(*ast.Ident)
				functions = append(functions, HttpFunction{
					Receiver: identReceiver.Name,
					In:       identIn.Name,
					Path:     instructions.Url,
					Auth:     instructions.Auth,
					Method:   instructions.Method,
					Name:     fun.Name.Name,
				})
			}
		}
	}

	// Calculation
	serveHttp := map[string][]HttpFunction{}
	for _, fun := range functions {
		if _, ok := serveHttp[fun.Receiver]; !ok {
			serveHttp[fun.Receiver] = make([]HttpFunction, 0)
		}
		serveHttp[fun.Receiver] = append(serveHttp[fun.Receiver], fun)
	}
	processedParams := make(map[string]struct{}, 0)
	paramsStructures := make([]ParamsStructure, 0)
	for _, fun := range functions {
		if _, ok := processedParams[fun.In]; ok {
			continue
		}
		rawParam := structures[fun.In]
		fields := make([]Field, 0)
		for _, rawField := range rawParam.Fields.List {
			t, _ := rawField.Type.(*ast.Ident)
			field := Field{
				FieldName: rawField.Names[0].Name,
				ParamType: t.Name,
				ParamName: strings.ToLower(rawField.Names[0].Name),
			}
			field.ParseApiValidator(rawField.Tag.Value)
			fields = append(fields, field)
		}

		paramsStructures = append(paramsStructures, ParamsStructure{
			Name:   fun.In,
			Fields: fields,
		})
	}

	// fmt.Printf("type: %T data: %+v\n", paramsStructures, paramsStructures)

	// Generate
	generalTpl.Execute(out, TemplateVariables{
		General: map[string]string{
			"JSONErrorTag":    "`json:\"error\"`",
			"JSONResponseTag": "`json:\"response\"`",
			"PackageName":     file.Name.Name,
		},
		ServeHTTP:        serveHttp,
		ParamsStructures: paramsStructures,
	})

	fmt.Println("Completed")
}
