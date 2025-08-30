package templates

// TemplateRegistry provides a centralized way to access all templates
type TemplateRegistry struct {
	templates map[string]string
}

// NewTemplateRegistry creates a new template registry with all templates
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]string),
	}
	
	registry.registerProviderTemplates()
	registry.registerResponseTemplates()
	registry.registerInterfaceTemplates()
	registry.registerMiddlewareTemplates()
	registry.registerRouteTemplates()
	
	return registry
}

// Get retrieves a template by name
func (tr *TemplateRegistry) Get(name string) (string, bool) {
	template, exists := tr.templates[name]
	return template, exists
}

// MustGet retrieves a template by name, panics if not found
func (tr *TemplateRegistry) MustGet(name string) string {
	template, exists := tr.templates[name]
	if !exists {
		panic("template not found: " + name)
	}
	return template
}

// registerProviderTemplates registers all provider-related templates
func (tr *TemplateRegistry) registerProviderTemplates() {
	// Basic provider without lifecycle
	tr.templates["provider"] = `func New{{.StructName}}({{range $i, $dep := .InjectedDeps}}{{if $i}}, {{end}}{{$dep.Name}} {{$dep.Type}}{{end}}) *{{.StructName}} {
	return &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: {{generateInitCode .Type}},
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}{{if not .Dependencies}}
{{end}}	}
}`
	
	// Provider with lifecycle
	tr.templates["lifecycle-provider"] = `func New{{.StructName}}(lc fx.Lifecycle{{range .Dependencies}}{{if not .IsInit}}, {{.Name}} {{.Type}}{{end}}{{end}}) *{{.StructName}} {
	service := &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: {{generateInitCode .Type}},
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},{{if .HasStop}}
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},{{end}}
	})
	
	return service
}`
	
	// Simple provider without dependencies
	tr.templates["simple-provider"] = `func New{{.StructName}}() *{{.StructName}} {
	return &{{.StructName}}{
		
	}
}`
	
	// Transient factory provider
	tr.templates["transient-provider"] = `// New{{.StructName}}Factory creates a factory function for {{.StructName}} (Transient mode)
func New{{.StructName}}Factory({{range $i, $dep := .InjectedDeps}}{{if $i}}, {{end}}{{$dep.Name}} {{$dep.Type}}{{end}}) func() *{{.StructName}} {
	return func() *{{.StructName}} {
		return &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}			{{.FieldName}}: {{generateInitCode .Type}},
{{else}}			{{.FieldName}}: {{.Name}},
{{end}}{{end}}{{if not .Dependencies}}
{{end}}		}
	}
}`

	// Logger provider with immediate initialization
	tr.templates["logger-provider"] = `func New{{.StructName}}(lc fx.Lifecycle{{range .Dependencies}}{{if not .IsInit}}, {{.Name}} {{.Type}}{{end}}{{end}}) *{{.StructName}} {
	// Initialize logger immediately for fx.WithLogger to work
	var handler slog.Handler
	if {{.ConfigParam}}.LogLevel == "debug" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	
	service := &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: slog.New(handler),
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
	
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},{{if .HasStop}}
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},{{end}}
	})
	
	return service
}`

	// Simple logger provider without lifecycle
	tr.templates["simple-logger-provider"] = `func New{{.StructName}}({{range $i, $dep := .InjectedDeps}}{{if $i}}, {{end}}{{$dep.Name}} {{$dep.Type}}{{end}}) *{{.StructName}} {
	// Initialize logger immediately
	var handler slog.Handler
	if {{.ConfigParam}}.LogLevel == "debug" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	
	return &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.FieldName}}: slog.New(handler),
{{else}}		{{.FieldName}}: {{.Name}},
{{end}}{{end}}	}
}`

	// Init invoke template for lifecycle management
	tr.templates["init-invoke"] = `func init{{.StructName}}Lifecycle(lc fx.Lifecycle, service *{{.StructName}}) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
{{if eq .StartMode "Background"}}			go func() {
				if err := service.Start(ctx); err != nil {
					log.Printf("background start error in %s: %v", "{{.StructName}}", err)
				}
			}()
			return nil
{{else}}			return service.Start(ctx)
{{end}}		},{{if .HasStop}}
		OnStop: func(ctx context.Context) error {
			return service.Stop(ctx)
		},{{end}}
	})
}`
}

// registerResponseTemplates registers all response handling templates
func (tr *TemplateRegistry) registerResponseTemplates() {
	tr.templates["route-wrapper"] = `func {{.WrapperName}}(handler *{{.ControllerName}}) echo.HandlerFunc {
	return func(c echo.Context) error {
{{.ParameterBindingCode}}{{.BodyBindingCode}}
{{.ResponseHandlingCode}}
	}
}`
	
	tr.templates["data-error-response"] = `		{{if .ErrAlreadyDeclared}}var data interface{}
		data, err = {{.HandlerCall}}{{else}}data, err := {{.HandlerCall}}{{end}}
		if err != nil {
			return handleError(c, err)
		}
		return c.JSON(http.StatusOK, data)`
		
	tr.templates["response-error-response"] = `		{{if .ErrAlreadyDeclared}}var response *axon.Response
		response, err = {{.HandlerCall}}{{else}}response, err := {{.HandlerCall}}{{end}}
		if err != nil {
			return handleError(c, err)
		}
		if response == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "handler returned nil response")
		}
		return handleAxonResponse(c, response)`
		
	tr.templates["error-response"] = `		{{if .ErrAlreadyDeclared}}err = {{.HandlerCall}}{{else}}err := {{.HandlerCall}}{{end}}
		if err != nil {
			return err
		}
		return nil`
	
	tr.templates["body-binding"] = `		var body {{.BodyType}}
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
`
}

// registerInterfaceTemplates registers all interface-related templates
func (tr *TemplateRegistry) registerInterfaceTemplates() {
	tr.templates["interface"] = `// {{.Name}} is the interface for {{.StructName}}
type {{.Name}} interface {
{{range .Methods}}	{{.Name}}({{range $i, $param := .Parameters}}{{if $i}}, {{end}}{{if $param.Name}}{{$param.Name}} {{end}}{{$param.Type}}{{end}}){{if .Returns}} ({{range $i, $ret := .Returns}}{{if $i}}, {{end}}{{$ret}}{{end}}){{end}}
{{end}}}`

	tr.templates["interface-provider"] = `func New{{.Name}}(impl *{{.StructName}}) {{.Name}} {
	return impl
}`
}

// registerMiddlewareTemplates registers all middleware-related templates
func (tr *TemplateRegistry) registerMiddlewareTemplates() {
	tr.templates["middleware-provider"] = `func New{{.StructName}}({{range $i, $dep := .InjectedDeps}}{{if $i}}, {{end}}{{$dep.Name}} {{$dep.Type}}{{end}}) *{{.StructName}} {
	return &{{.StructName}}{
{{range .Dependencies}}{{if .IsInit}}		{{.Name}}: {{generateInitCode .Type}},
{{else}}		{{.Name}}: {{.Name}},
{{end}}{{end}}	}
}`

	tr.templates["global-middleware-registration"] = `// RegisterGlobalMiddleware registers all global middleware with Echo
func RegisterGlobalMiddleware(e *echo.Echo{{range .GlobalMiddlewares}}, {{toCamelCase .Name}} *{{.StructName}}{{end}}) {
{{range .GlobalMiddlewares}}	e.Use({{toCamelCase .Name}}.Handle)
{{end}}}`

	tr.templates["middleware-registry"] = `// RegisterMiddlewares registers all middleware with the axon middleware registry
func RegisterMiddlewares({{range $i, $mw := .Middlewares}}{{if $i}}, {{end}}{{toCamelCase $mw.Name}} *{{$mw.StructName}}{{end}}) {
{{range .Middlewares}}	axon.RegisterMiddlewareHandler("{{.Name}}", {{toCamelCase .Name}})
{{end}}}`
}

// registerRouteTemplates registers all route-related templates
func (tr *TemplateRegistry) registerRouteTemplates() {
	tr.templates["route-registration-function"] = `// RegisterRoutes registers all HTTP routes with the Echo instance
func RegisterRoutes(e *echo.Echo{{range .Controllers}}, {{.VarName}} *{{.StructName}}{{end}}{{range .MiddlewareDeps}}, {{.VarName}} *{{.PackageName}}.{{.Name}}{{end}}) {
{{range .Controllers}}{{if .Prefix}}	{{.VarName}}Group := e.Group("{{.EchoPrefix}}")
{{end}}{{range .Routes}}{{template "RouteRegistration" .}}{{end}}{{end}}}`

	tr.templates["route-registration"] = `	{{.HandlerVar}} := {{.WrapperFunc}}({{.ControllerVar}})
{{if .HasMiddleware}}	{{.GroupVar}}.{{.Method}}("{{.EchoPath}}", {{.HandlerVar}}, {{.MiddlewareList}})
{{else}}	{{.GroupVar}}.{{.Method}}("{{.EchoPath}}", {{.HandlerVar}})
{{end}}	axon.DefaultRouteRegistry.RegisterRoute(axon.RouteInfo{
		Method:              "{{.Method}}",
		Path:                "{{.Path}}",
		EchoPath:            "{{.EchoPath}}",
		HandlerName:         "{{.HandlerName}}",
		ControllerName:      "{{.ControllerName}}",
		PackageName:         "{{.PackageName}}",
		Middlewares:         {{.MiddlewaresArray}},
		MiddlewareInstances: {{.MiddlewareInstancesArray}},
		ParameterInstances:  {{.ParameterInstancesArray}},
		Handler:             {{.HandlerVar}},
	})
`

	tr.templates["middleware-instance"] = `{
		Name:     "{{.Name}}",
		Handler:  {{.VarName}}.Handle,
		Instance: {{.VarName}},
	}`
}

// Global template registry instance
var DefaultTemplateRegistry = NewTemplateRegistry()