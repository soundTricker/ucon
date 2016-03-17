package swagger

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/favclip/ucon"
	"golang.org/x/net/context"
)

// interface compatibility check
var _ ucon.HandlersScannerPlugin = &Plugin{}
var _ ucon.Context = &HandlerInfo{}
var _ ucon.HandlerContainer = &HandlerInfo{}

var swaggerOperationKey = &struct{ temp string }{}

var httpReqType = reflect.TypeOf(&http.Request{})
var httpRespType = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
var netContextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()
var uconHTTPErrorType = reflect.TypeOf((*ucon.HTTPErrorResponse)(nil)).Elem()

// DefaultTypeSchemaMapper is used for mapping from go-type to swagger-schema.
var DefaultTypeSchemaMapper = map[reflect.Type]*TypeSchema{
	reflect.TypeOf(time.Time{}): &TypeSchema{
		RefName: "",
		Schema: &Schema{
			Type:   "string",
			Format: "date-time", // RFC3339
		},
		AllowRef: false,
	},
}

// TypeSchema is a container of swagger schema and its attributes.
// RefName must be given if AllowRef is true.
type TypeSchema struct {
	RefName  string
	Schema   *Schema
	AllowRef bool
}

// Plugin is a holder for all of plugin settings.
type Plugin struct {
	options             *Options
	swagger             *Object
	typeSchemaMapper    map[reflect.Type]*TypeSchema
	typeParameterMapper map[reflect.Type]map[string]*parameterWrapper
}

type parameterWrapper struct {
	StructField reflect.StructField
}

// HandlerInfo is a container of the handler function and the operation with the context.
// HandlerInfo implements interfaces of ucon.HandlerContainer and ucon.Context.
type HandlerInfo struct {
	HandlerFunc interface{}
	Operation
	Context ucon.Context
}

// Options is a container of optional settings to configure a plugin.
type Options struct {
	Object                 *Object
	DefinitionNameModifier func(refT reflect.Type, defName string) string
}

// NewPlugin returns new swagger plugin configured with the options.
func NewPlugin(opts *Options) *Plugin {
	if opts == nil {
		opts = &Options{}
	}
	plugin := &Plugin{
		options:          opts,
		swagger:          opts.Object,
		typeSchemaMapper: make(map[reflect.Type]*TypeSchema),
	}
	for k, v := range DefaultTypeSchemaMapper {
		plugin.typeSchemaMapper[k] = v
	}
	var _ ucon.HandlersScannerPlugin = plugin

	return plugin
}

// HandlersScannerProcess executes scanning all registered handlers to serve swagger.json.
func (p *Plugin) HandlersScannerProcess(m *ucon.ServeMux, rds []*ucon.RouteDefinition) error {
	// construct swagger.json
	for _, rd := range rds {
		err := p.processHandler(rd)
		if err != nil {
			return err
		}
	}

	err := p.swagger.finish()
	if err != nil {
		return err
	}

	// supply swagger.json endpoint
	m.HandleFunc("GET", "/api/swagger.json", func(w http.ResponseWriter, r *http.Request) *Object {
		return p.swagger
	})

	return nil
}

func (p *Plugin) processHandler(rd *ucon.RouteDefinition) error {
	if p.swagger == nil {
		p.swagger = &Object{}
	}
	if p.swagger.Paths == nil {
		p.swagger.Paths = make(Paths, 0)
	}
	if p.swagger.Definitions == nil {
		p.swagger.Definitions = make(Definitions, 0)
	}

	item := p.swagger.Paths[rd.PathTemplate.PathTemplate]
	if item == nil {
		item = &PathItem{}
	}

	var putOperation func(op *Operation)
	switch rd.Method {
	case "GET":
		putOperation = func(op *Operation) {
			item.Get = op
		}
	case "PUT":
		putOperation = func(op *Operation) {
			item.Put = op
		}
	case "POST":
		putOperation = func(op *Operation) {
			item.Post = op
		}
	case "DELETE":
		putOperation = func(op *Operation) {
			item.Delete = op
		}
	case "OPTIONS":
		putOperation = func(op *Operation) {
			item.Options = op
		}
	case "HEAD":
		putOperation = func(op *Operation) {
			item.Head = op
		}
	case "PATCH":
		putOperation = func(op *Operation) {
			item.Patch = op
		}
	default:
		return fmt.Errorf("unknown method: %s", rd.Method)
	}

	op, err := p.extractHandlerInfo(rd)
	if err != nil {
		return err
	}
	if op != nil {
		p.swagger.Paths[rd.PathTemplate.PathTemplate] = item

		putOperation(op)

		for _, tsc := range p.typeSchemaMapper {
			if !tsc.AllowRef {
				continue
			}
			if tsc.RefName == "" {
				return errors.New("Name is required")
			}

			_, ok := p.swagger.Definitions[tsc.RefName]
			if !ok {
				p.swagger.Definitions[tsc.RefName] = tsc.Schema
			}
		}
	}

	return nil
}

func (p *Plugin) extractHandlerInfo(rd *ucon.RouteDefinition) (*Operation, error) {
	var op *Operation
	op, ok := rd.HandlerContainer.Value(swaggerOperationKey).(*Operation)
	if !ok || op == nil {
		op = &Operation{
			Description: fmt.Sprintf("%s %s", rd.Method, rd.PathTemplate.PathTemplate),
		}
	}
	if len(op.Responses) == 0 {
		op.Responses = make(Responses, 0)
		op.Responses["200"] = &Response{
			Description: fmt.Sprintf("response of %s %s", rd.Method, rd.PathTemplate.PathTemplate),
		}
	}

	var reqType, respType, errType reflect.Type
	handlerT := reflect.TypeOf(rd.HandlerContainer.Handler())
	for i, numIn := 0, handlerT.NumIn(); i < numIn; i++ {
		arg := handlerT.In(i)
		if arg == httpReqType {
			continue
		} else if arg == httpRespType {
			continue
		} else if arg == netContextType {
			continue
		}
		reqType = arg
		break
	}
	for i, numOut := 0, handlerT.NumOut(); i < numOut; i++ {
		ret := handlerT.Out(i)
		if ret.AssignableTo(errorType) {
			errType = ret
			continue
		}
		respType = ret
	}
	if respType == nil && errType == nil {
		// static file handler...?
		return nil, nil
	}

	// parameter
	var bodyParameter *Parameter
	if reqType != nil {
		paramMap, err := p.reflectTypeToParameterMapper(reqType)
		if err != nil {
			return nil, err
		}

		needBody := false
	outer:
		for paramName, pw := range paramMap {
			// in path
			if pw.InPath() {
				op.Parameters = append(op.Parameters, &Parameter{
					Name:     paramName,
					In:       "path",
					Required: true,
					Type:     pw.ParameterType(),
					Format:   pw.ParameterFormat(),
				})

				continue
			} else {
				for _, pathParam := range rd.PathTemplate.PathParameters {
					if paramName != pathParam {
						continue
					}
					op.Parameters = append(op.Parameters, &Parameter{
						Name:     paramName,
						In:       "path",
						Required: true,
						Type:     pw.ParameterType(),
						Format:   pw.ParameterFormat(),
					})
					continue outer
				}
			}

			// in query
			if pw.InQuery() {
				param := &Parameter{
					Name:     pw.Name(),
					In:       "query",
					Required: pw.Required(),
					Type:     pw.ParameterType(),
					Format:   pw.ParameterFormat(),
				}
				if param.Type == "array" {
					param.Items = &Items{}
					tsc, err := p.reflectTypeToTypeSchemaContainer(pw.StructField.Type)
					if err != nil {
						return nil, err
					}
					// NOTE(laco) Parameter.Items doesn't allow `$ref`.
					// Parameter.Items.Type is required.
					if tsc.Schema == nil || tsc.Schema.Items == nil || tsc.Schema.Items.Type == "" {
						return nil, errors.New("Items is required")
					}
					param.Items.Type = tsc.Schema.Items.Type
					param.Items.Format = tsc.Schema.Items.Format
				}

				op.Parameters = append(op.Parameters, param)

				continue
			}

			if pw.Private() {
				continue
			}

			needBody = true
		}

		// in body
		if needBody {
			bodyParameter = &Parameter{
				Name:     "body",
				In:       "body",
				Required: true,
				Schema:   nil,
			}
			op.Parameters = append(op.Parameters, bodyParameter)
		}
	}

	if reqType != nil && bodyParameter != nil {
		tsc, err := p.reflectTypeToTypeSchemaContainer(reqType)
		if err != nil {
			return nil, err
		}
		if bodyParameter != nil {
			if tsc.AllowRef && tsc.RefName != "" {
				bodyParameter.Schema = &Schema{
					Ref: fmt.Sprintf("#/definitions/%s", tsc.RefName),
				}
			} else if tsc.AllowRef {
				return nil, errors.New("Name is required")
			} else {
				bodyParameter.Schema = tsc.Schema
			}
		}
	}

	if respType != nil {
		tsc, err := p.reflectTypeToTypeSchemaContainer(respType)
		if err != nil {
			return nil, err
		}

		for _, resp := range op.Responses {
			if tsc.AllowRef && tsc.RefName != "" {
				resp.Schema = &Schema{
					Ref: fmt.Sprintf("#/definitions/%s", tsc.RefName),
				}
			} else if tsc.AllowRef {
				return nil, errors.New("Name is required")
			} else {
				resp.Schema = tsc.Schema
			}
		}
	}

	if errType != nil {
		if errType == errorType {
			// pass
		} else if errType == uconHTTPErrorType {
			// pass
		} else {
			tsc, err := p.reflectTypeToTypeSchemaContainer(errType)
			if err != nil {
				return nil, err
			}

			if op.Responses["default"] == nil {
				resp := &Response{
					Description: "???", // TODO
				}
				op.Responses["default"] = resp
				if tsc.AllowRef && tsc.RefName != "" {
					resp.Schema = &Schema{
						Ref: fmt.Sprintf("#/definitions/%s", tsc.RefName),
					}
				} else if tsc.AllowRef {
					return nil, errors.New("Name is required")
				} else {
					resp.Schema = tsc.Schema
				}

			}
		}
	}

	return op, nil
}

func (p *Plugin) reflectTypeToTypeSchemaContainer(refT reflect.Type) (*TypeSchema, error) {
	if refT.Kind() == reflect.Ptr {
		refT = refT.Elem()
	}

	if v, ok := p.typeSchemaMapper[refT]; ok {
		return v, nil
	}

	defName := refT.Name()
	if p.options.DefinitionNameModifier != nil {
		defName = p.options.DefinitionNameModifier(refT, defName)
	}

	schema := &Schema{}

	allowRef := false
	if defName != "" && refT.PkgPath() != "" {
		// reject builtin-type, aka int, bool, string
		allowRef = true
	}
	ts := &TypeSchema{
		RefName:  defName,
		Schema:   schema,
		AllowRef: allowRef,
	}
	p.typeSchemaMapper[refT] = ts

	schema.Type = reflectTypeToSwaggerTypeString(refT)
	if schema.Type == "" {
		return nil, fmt.Errorf("unknown schema type: %s", refT.Kind().String())
	} else if schema.Type == "object" || schema.Type == "array" {
		switch refT.Kind() {
		case reflect.Struct:
			var process func(refT reflect.Type) error
			process = func(refT reflect.Type) error {
				if refT.Kind() == reflect.Ptr {
					refT = refT.Elem()
				}
				if refT.Kind() != reflect.Struct {
					return nil
				}
				for i, numField := 0, refT.NumField(); i < numField; i++ {
					sf := refT.Field(i)

					tagJSON := ucon.NewTagJSON(sf.Tag)
					if tagJSON.Ignored() {
						continue
					}

					if sf.Anonymous {
						err := process(sf.Type) // it just means same struct.
						if err != nil {
							return err
						}
						continue
					}

					name := tagJSON.Name()
					if name == "" {
						name = sf.Name
					}

					tsc, err := p.reflectTypeToTypeSchemaContainer(sf.Type)
					if err != nil {
						return err
					}

					if schema.Properties == nil {
						schema.Properties = make(map[string]*Schema, 0)
					}
					if tsc.AllowRef && tsc.RefName != "" {
						schema.Properties[name] = &Schema{
							Ref: fmt.Sprintf("#/definitions/%s", tsc.RefName),
						}
					} else if tsc.AllowRef {
						return errors.New("Name is required")
					} else {
						schema.Properties[name] = tsc.Schema
					}
				}
				return nil
			}
			err := process(refT)
			if err != nil {
				return nil, err
			}
		case reflect.Slice, reflect.Array:
			tsc, err := p.reflectTypeToTypeSchemaContainer(refT.Elem())
			if err != nil {
				return nil, err
			}
			if tsc.AllowRef && tsc.RefName != "" {
				schema.Items = &Schema{
					Ref: fmt.Sprintf("#/definitions/%s", tsc.RefName),
				}
			} else if tsc.AllowRef {
				return nil, errors.New("Name is required")
			} else {
				schema.Items = tsc.Schema
			}
		}
	}

	return p.reflectTypeToTypeSchemaContainer(refT)
}

func (p *Plugin) reflectTypeToParameterMapper(refT reflect.Type) (map[string]*parameterWrapper, error) {
	if m, ok := p.typeParameterMapper[refT]; ok {
		return m, nil
	}

	parameterMap := make(map[string]*parameterWrapper, 0)

	var process func(refT reflect.Type) error
	process = func(refT reflect.Type) error {
		if refT.Kind() == reflect.Ptr {
			refT = refT.Elem()
		}

		for i, numField := 0, refT.NumField(); i < numField; i++ {
			sf := refT.Field(i)
			pw := &parameterWrapper{
				StructField: sf,
			}

			if pw.Private() {
				continue
			}

			if sf.Anonymous {
				err := process(sf.Type)
				if err != nil {
					return err
				}
				continue
			}

			name := pw.Name()
			if name == "" {
				name = sf.Name
			}

			parameterMap[name] = pw
		}
		return nil
	}

	err := process(refT)
	if err != nil {
		return nil, err
	}

	return parameterMap, nil
}

// AddTag adds the tag to top-level tags definition.
func (p *Plugin) AddTag(tag *Tag) *Tag {
	if p.options.Object == nil {
		p.options.Object = &Object{}
	}
	p.options.Object.Tags = append(p.options.Object.Tags, tag)

	return tag
}

func reflectTypeToSwaggerTypeString(refT reflect.Type) string {
	if refT.Kind() == reflect.Ptr {
		refT = refT.Elem()
	}

	switch refT.Kind() {
	case reflect.Struct:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "string"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	default:
		return ""
	}
}

func (pw *parameterWrapper) ParameterType() string {
	return reflectTypeToSwaggerTypeString(pw.StructField.Type)
}

func (pw *parameterWrapper) ParameterFormat() string {
	refT := pw.StructField.Type

	if refT.Kind() == reflect.Ptr {
		refT = refT.Elem()
	}

	switch refT.Kind() {
	case reflect.Int64:
		return "int64"
	default:
		return ""
	}
}

func (pw *parameterWrapper) InPath() bool {
	swaggerTag := NewTagSwagger(pw.StructField.Tag)
	return swaggerTag.In() == "path"
}

func (pw *parameterWrapper) InQuery() bool {
	swaggerTag := NewTagSwagger(pw.StructField.Tag)
	return swaggerTag.In() == "query"
}

func (pw *parameterWrapper) Name() string {
	swaggerTag := NewTagSwagger(pw.StructField.Tag)
	name := swaggerTag.Name()
	if name != "" {
		return name
	}

	jsonTag := ucon.NewTagJSON(pw.StructField.Tag)
	name = jsonTag.Name()
	if name != "" {
		return name
	}

	return pw.StructField.Name
}

func (pw *parameterWrapper) Required() bool {
	return NewTagSwagger(pw.StructField.Tag).Required()
}

func (pw *parameterWrapper) Private() bool {
	swaggerTag := NewTagSwagger(pw.StructField.Tag)
	if swaggerTag.Private() {
		return true
	}

	jsonTag := ucon.NewTagJSON(pw.StructField.Tag)
	if jsonTag.Ignored() {
		return true
	}

	return false
}

// NewHandlerInfo returns new HandlerInfo containing given handler function.
func NewHandlerInfo(handler interface{}) *HandlerInfo {
	ucon.CheckFunction(handler)
	return &HandlerInfo{
		HandlerFunc: handler,
	}
}

// Handler returns contained handler function.
func (wr *HandlerInfo) Handler() interface{} {
	return wr.HandlerFunc
}

// Value returns the value contained with the key.
func (wr *HandlerInfo) Value(key interface{}) interface{} {
	if key == swaggerOperationKey {
		return &wr.Operation
	}
	if wr.Context != nil {
		return wr.Context.Value(key)
	}
	return nil
}
