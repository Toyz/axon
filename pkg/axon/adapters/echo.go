package adapters

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/toyz/axon/pkg/axon"
)

// EchoAdapter implements axon.WebServerInterface for Echo v4
type EchoAdapter struct {
	engine *echo.Echo
}

// NewEchoAdapter creates a new Echo adapter
func NewEchoAdapter(e *echo.Echo) *EchoAdapter {
	return &EchoAdapter{engine: e}
}

// NewDefaultEchoAdapter creates a new Echo adapter with default Echo instance
func NewDefaultEchoAdapter() *EchoAdapter {
	return &EchoAdapter{engine: echo.New()}
}

// RegisterRoute registers a route with the Echo server
func (ea *EchoAdapter) RegisterRoute(method string, path axon.AxonPath, handler axon.HandlerFunc, middlewares ...axon.MiddlewareFunc) {
	// Convert Axon path format to Echo path format
	parts := path.Parts()
	echoPath := ""
	for _, part := range parts {
		switch part.Type {
		case axon.StaticPart:
			echoPath += part.Value
		case axon.ParameterPart:
			echoPath += ":" + part.Value
		case axon.WildcardPart:
			echoPath += "*"
		default:
			echoPath += part.Value
		}
	}
	// Convert axon.HandlerFunc to echo.HandlerFunc
	echoHandler := ea.convertHandler(handler)

	// Convert axon middlewares to echo middlewares
	echoMiddlewares := make([]echo.MiddlewareFunc, len(middlewares))
	for i, mw := range middlewares {
		echoMiddlewares[i] = ea.convertMiddleware(mw)
	}

	// Register with Echo
	ea.engine.Add(method, echoPath, echoHandler, echoMiddlewares...)
}

// RegisterGroup creates a new route group
func (ea *EchoAdapter) RegisterGroup(prefix string) axon.RouteGroup {
	echoGroup := ea.engine.Group(prefix)
	return &EchoGroupAdapter{group: echoGroup, adapter: ea}
}

// Use adds global middleware
func (ea *EchoAdapter) Use(middleware axon.MiddlewareFunc) {
	echoMiddleware := ea.convertMiddleware(middleware)
	ea.engine.Use(echoMiddleware)
}

// Start starts the server
func (ea *EchoAdapter) Start(addr string) error {
	return ea.engine.Start(addr)
}

// Stop stops the server
func (ea *EchoAdapter) Stop(ctx context.Context) error {
	return ea.engine.Shutdown(ctx)
}

// Name returns the adapter name
func (ea *EchoAdapter) Name() string {
	return "Echo"
}

// GetEngine returns the underlying Echo instance
func (ea *EchoAdapter) GetEngine() *echo.Echo {
	return ea.engine
}

// EchoGroupAdapter implements axon.RouteGroup for Echo groups
type EchoGroupAdapter struct {
	group   *echo.Group
	adapter *EchoAdapter
}

// RegisterRoute registers a route with the group
func (ega *EchoGroupAdapter) RegisterRoute(method string, path axon.AxonPath, handler axon.HandlerFunc, middlewares ...axon.MiddlewareFunc) {
	// Convert Axon path format to Echo path format
	parts := path.Parts()
	echoPath := ""
	for _, part := range parts {
		switch part.Type {
		case axon.StaticPart:
			echoPath += part.Value
		case axon.ParameterPart:
			echoPath += ":" + part.Value
		case axon.WildcardPart:
			echoPath += "*"
		default:
			echoPath += part.Value
		}
	}
	echoHandler := ega.adapter.convertHandler(handler)

	echoMiddlewares := make([]echo.MiddlewareFunc, len(middlewares))
	for i, mw := range middlewares {
		echoMiddlewares[i] = ega.adapter.convertMiddleware(mw)
	}

	ega.group.Add(method, echoPath, echoHandler, echoMiddlewares...)
}

// Use adds middleware to the group
func (ega *EchoGroupAdapter) Use(middleware axon.MiddlewareFunc) {
	echoMiddleware := ega.adapter.convertMiddleware(middleware)
	ega.group.Use(echoMiddleware)
}

// Group creates a sub-group
func (ega *EchoGroupAdapter) Group(prefix string) axon.RouteGroup {
	subGroup := ega.group.Group(prefix)
	return &EchoGroupAdapter{group: subGroup, adapter: ega.adapter}
}

// convertHandler converts axon.HandlerFunc to echo.HandlerFunc
func (ea *EchoAdapter) convertHandler(handler axon.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := &EchoRequestContext{context: c}
		return handler(ctx)
	}
}

// convertMiddleware converts axon.MiddlewareFunc to echo.MiddlewareFunc
func (ea *EchoAdapter) convertMiddleware(middleware axon.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Convert echo.HandlerFunc to axon.HandlerFunc
			axonNext := func(ctx axon.RequestContext) error {
				return next(c)
			}

			// Call axon middleware
			axonHandler := middleware(axonNext)

			// Convert back to echo context and call
			ctx := &EchoRequestContext{context: c}
			return axonHandler(ctx)
		}
	}
}

// EchoRequestContext implements axon.RequestContext for Echo
type EchoRequestContext struct {
	context echo.Context
}

// Method returns the HTTP method
func (erc *EchoRequestContext) Method() string {
	return erc.context.Request().Method
}

// Path returns the request path
func (erc *EchoRequestContext) Path() string {
	return erc.context.Request().URL.Path
}

// RealIP returns the real IP address
func (erc *EchoRequestContext) RealIP() string {
	return erc.context.RealIP()
}

// Param returns path parameter by name
func (erc *EchoRequestContext) Param(key string) string {
	return erc.context.Param(key)
}

// ParamNames returns path parameter names
func (erc *EchoRequestContext) ParamNames() []string {
	return erc.context.ParamNames()
}

// ParamValues returns path parameter values
func (erc *EchoRequestContext) ParamValues() []string {
	return erc.context.ParamValues()
}

// SetParam sets path parameter
func (erc *EchoRequestContext) SetParam(name, value string) {
	names := append(erc.context.ParamNames(), name)
	values := append(erc.context.ParamValues(), value)
	erc.context.SetParamNames(names...)
	erc.context.SetParamValues(values...)
}

// QueryParam returns query parameter by name
func (erc *EchoRequestContext) QueryParam(key string) string {
	return erc.context.QueryParam(key)
}

// QueryParams returns all query parameters
func (erc *EchoRequestContext) QueryParams() map[string][]string {
	return erc.context.QueryParams()
}

// QueryString returns the query string
func (erc *EchoRequestContext) QueryString() string {
	return erc.context.QueryString()
}

// Request returns the request interface
func (erc *EchoRequestContext) Request() axon.RequestInterface {
	return &EchoRequestInterface{request: erc.context.Request()}
}

// Response returns the response interface
func (erc *EchoRequestContext) Response() axon.ResponseInterface {
	return &EchoResponseInterface{response: erc.context.Response(), context: erc.context}
}

// Bind binds request body to provided struct
func (erc *EchoRequestContext) Bind(i interface{}) error {
	return erc.context.Bind(i)
}

// Validate validates the provided struct
func (erc *EchoRequestContext) Validate(i interface{}) error {
	return erc.context.Validate(i)
}

// Get retrieves data from context
func (erc *EchoRequestContext) Get(key string) interface{} {
	return erc.context.Get(key)
}

// Set stores data in context
func (erc *EchoRequestContext) Set(key string, val interface{}) {
	erc.context.Set(key, val)
}

// FormValue returns form value by name
func (erc *EchoRequestContext) FormValue(name string) string {
	return erc.context.FormValue(name)
}

// FormParams returns form parameters
func (erc *EchoRequestContext) FormParams() (map[string][]string, error) {
	return erc.context.FormParams()
}

// FormFile returns uploaded file by name
func (erc *EchoRequestContext) FormFile(name string) (axon.FileHeader, error) {
	file, err := erc.context.FormFile(name)
	if err != nil {
		return nil, err
	}
	return &EchoFileHeader{header: file}, nil
}

// MultipartForm returns multipart form
func (erc *EchoRequestContext) MultipartForm() (axon.MultipartForm, error) {
	form, err := erc.context.MultipartForm()
	if err != nil {
		return nil, err
	}
	return &EchoMultipartForm{form: form}, nil
}

// EchoRequestInterface implements axon.RequestInterface for Echo requests
type EchoRequestInterface struct {
	request *http.Request
}

// Header returns request header value
func (eri *EchoRequestInterface) Header(key string) string {
	return eri.request.Header.Get(key)
}

// SetHeader sets request header
func (eri *EchoRequestInterface) SetHeader(key, value string) {
	eri.request.Header.Set(key, value)
}

// Body returns request body
func (eri *EchoRequestInterface) Body() []byte {
	// Note: This is a simplified implementation
	// In production, you'd want to handle this more carefully
	return nil
}

// ContentLength returns content length
func (eri *EchoRequestInterface) ContentLength() int64 {
	return eri.request.ContentLength
}

// ContentType returns content type
func (eri *EchoRequestInterface) ContentType() string {
	return eri.request.Header.Get("Content-Type")
}

// Cookies returns all cookies
func (eri *EchoRequestInterface) Cookies() []axon.AxonCookie {
	cookies := eri.request.Cookies()
	result := make([]axon.AxonCookie, len(cookies))
	for i, c := range cookies {
		result[i] = axon.AxonCookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			Expires:  c.Expires,
			MaxAge:   c.MaxAge,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
			SameSite: axon.SameSiteMode(c.SameSite),
		}
	}
	return result
}

// Cookie returns specific cookie
func (eri *EchoRequestInterface) Cookie(name string) (axon.AxonCookie, error) {
	c, err := eri.request.Cookie(name)
	if err != nil {
		return axon.AxonCookie{}, err
	}
	return axon.AxonCookie{
		Name:     c.Name,
		Value:    c.Value,
		Path:     c.Path,
		Domain:   c.Domain,
		Expires:  c.Expires,
		MaxAge:   c.MaxAge,
		Secure:   c.Secure,
		HttpOnly: c.HttpOnly,
		SameSite: axon.SameSiteMode(c.SameSite),
	}, nil
}

// EchoResponseInterface implements axon.ResponseInterface for Echo responses
type EchoResponseInterface struct {
	response *echo.Response
	context  echo.Context
}

// Status returns response status code
func (eri *EchoResponseInterface) Status() int {
	return eri.response.Status
}

// SetStatus sets response status code
func (eri *EchoResponseInterface) SetStatus(code int) {
	eri.response.Status = code
}

// Header returns response header value
func (eri *EchoResponseInterface) Header(key string) string {
	return eri.response.Header().Get(key)
}

// SetHeader sets response header
func (eri *EchoResponseInterface) SetHeader(key, value string) {
	eri.response.Header().Set(key, value)
}

// JSON writes JSON response
func (eri *EchoResponseInterface) JSON(code int, i interface{}) error {
	return eri.context.JSON(code, i)
}

// JSONPretty writes pretty JSON response
func (eri *EchoResponseInterface) JSONPretty(code int, i interface{}, indent string) error {
	return eri.context.JSONPretty(code, i, indent)
}

// String writes string response
func (eri *EchoResponseInterface) String(code int, s string) error {
	return eri.context.String(code, s)
}

// HTML writes HTML response
func (eri *EchoResponseInterface) HTML(code int, html string) error {
	return eri.context.HTML(code, html)
}

// Blob writes blob response
func (eri *EchoResponseInterface) Blob(code int, contentType string, b []byte) error {
	return eri.context.Blob(code, contentType, b)
}

// Stream writes streaming response
func (eri *EchoResponseInterface) Stream(code int, contentType string, r interface{}) error {
	// Type assertion for io.Reader - this is framework-specific
	if reader, ok := r.(interface{ Read([]byte) (int, error) }); ok {
		return eri.context.Stream(code, contentType, reader)
	}
	return axon.NewHTTPError(500, "Invalid stream reader")
}

// SetCookie sets a cookie
func (eri *EchoResponseInterface) SetCookie(cookie axon.AxonCookie) {
	httpCookie := &http.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Expires:  cookie.Expires,
		MaxAge:   cookie.MaxAge,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HttpOnly,
		SameSite: http.SameSite(cookie.SameSite),
	}
	eri.context.SetCookie(httpCookie)
}

// Size returns response size
func (eri *EchoResponseInterface) Size() int64 {
	return eri.response.Size
}

// Written returns whether response has been written
func (eri *EchoResponseInterface) Written() bool {
	return eri.response.Committed
}

// Writer returns the underlying writer
func (eri *EchoResponseInterface) Writer() interface{} {
	return eri.response.Writer
}

// EchoFileHeader implements axon.FileHeader for Echo file uploads
type EchoFileHeader struct {
	header *multipart.FileHeader
}

// Filename returns the uploaded file name
func (efh *EchoFileHeader) Filename() string {
	return efh.header.Filename
}

// Header returns file headers
func (efh *EchoFileHeader) Header() map[string][]string {
	return efh.header.Header
}

// Size returns file size
func (efh *EchoFileHeader) Size() int64 {
	return efh.header.Size
}

// Open opens the uploaded file
func (efh *EchoFileHeader) Open() (interface{}, error) {
	return efh.header.Open()
}

// EchoMultipartForm implements axon.MultipartForm for Echo
type EchoMultipartForm struct {
	form *multipart.Form
}

// Value returns form values
func (emf *EchoMultipartForm) Value() map[string][]string {
	return emf.form.Value
}

// File returns form files
func (emf *EchoMultipartForm) File() map[string][]axon.FileHeader {
	result := make(map[string][]axon.FileHeader)
	for key, files := range emf.form.File {
		fileHeaders := make([]axon.FileHeader, len(files))
		for i, file := range files {
			fileHeaders[i] = &EchoFileHeader{header: file}
		}
		result[key] = fileHeaders
	}
	return result
}