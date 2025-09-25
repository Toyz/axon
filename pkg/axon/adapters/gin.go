package adapters

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/toyz/axon/pkg/axon"
)

// GinFileHeader implements axon.FileHeader for Gin
type GinFileHeader struct {
	filename string
	header   map[string][]string
	size     int64
	file     multipart.File
}

func (gfh *GinFileHeader) Filename() string {
	return gfh.filename
}

func (gfh *GinFileHeader) Header() map[string][]string {
	return gfh.header
}

func (gfh *GinFileHeader) Size() int64 {
	return gfh.size
}

func (gfh *GinFileHeader) Open() (interface{}, error) {
	return gfh.file, nil
}

// GinMultipartForm implements axon.MultipartForm for Gin
type GinMultipartForm struct {
	values map[string][]string
	files  map[string][]axon.FileHeader
}

func (gmf *GinMultipartForm) Value() map[string][]string {
	return gmf.values
}

func (gmf *GinMultipartForm) File() map[string][]axon.FileHeader {
	return gmf.files
}

// GinAdapter implements axon.WebServerInterface for Gin framework
type GinAdapter struct {
	engine *gin.Engine
}

// NewGinAdapter creates a new Gin adapter
func NewGinAdapter(g *gin.Engine) *GinAdapter {
	return &GinAdapter{engine: g}
}

// NewDefaultGinAdapter creates a new Gin adapter with default Gin instance
func NewDefaultGinAdapter() *GinAdapter {
	return &GinAdapter{engine: gin.Default()}
}

// convertAxonPathToGin converts AxonPath to Gin path format
func (ga *GinAdapter) convertAxonPathToGin(path axon.AxonPath) string {
	parts := path.Parts()
	ginPath := ""
	for _, part := range parts {
		if part.Value == "" || part.Value == "/" {
			continue
		}

		switch part.Type {
		case axon.StaticPart:
			ginPath += part.Value
		case axon.ParameterPart:
			ginPath += ":" + part.Value
		case axon.WildcardPart:
			ginPath += "/*path"
		default:
			ginPath += part.Value
		}
	}
	return ginPath
}

// RegisterRoute registers a route with the Gin server
func (ga *GinAdapter) RegisterRoute(method string, path axon.AxonPath, handler axon.HandlerFunc, middlewares ...axon.MiddlewareFunc) {
	// Convert axon.HandlerFunc to gin.HandlerFunc
	ginHandler := ga.convertHandler(handler)

	// Convert middleware functions
	var ginMiddlewares []gin.HandlerFunc
	for _, middleware := range middlewares {
		ginMiddlewares = append(ginMiddlewares, ga.convertMiddleware(middleware))
	}

	// Convert Axon path to Gin path format
	ginPath := ga.convertAxonPathToGin(path)

	// Register the route with middlewares
	handlers := append(ginMiddlewares, ginHandler)
	ga.engine.Handle(method, ginPath, handlers...)
}

// RegisterGroup registers a route group with the Gin server
func (ga *GinAdapter) RegisterGroup(prefix string) axon.RouteGroup {
	ginGroup := ga.engine.Group(prefix)
	return &GinRouteGroup{group: ginGroup, adapter: ga}
}

// Use registers a global middleware with the Gin server
func (ga *GinAdapter) Use(middleware axon.MiddlewareFunc) {
	ginMiddleware := ga.convertMiddleware(middleware)
	ga.engine.Use(ginMiddleware)
}

// Start starts the Gin server
func (ga *GinAdapter) Start(addr string) error {
	return ga.engine.Run(addr)
}

// Stop stops the Gin server (Gin doesn't have built-in graceful shutdown)
func (ga *GinAdapter) Stop(ctx context.Context) error {
	// Gin doesn't have built-in server shutdown, so we'll implement it
	// This would typically be handled by the http.Server wrapping Gin
	return nil
}

// Name returns the adapter name
func (ga *GinAdapter) Name() string {
	return "Gin"
}

// GetEngine returns the underlying Gin engine
func (ga *GinAdapter) GetEngine() *gin.Engine {
	return ga.engine
}

// GinRouteGroup implements axon.RouteGroup for Gin
type GinRouteGroup struct {
	group   *gin.RouterGroup
	adapter *GinAdapter
}

// RegisterRoute registers a route within the group
func (grg *GinRouteGroup) RegisterRoute(method string, path axon.AxonPath, handler axon.HandlerFunc, middlewares ...axon.MiddlewareFunc) {
	ginHandler := grg.adapter.convertHandler(handler)

	var ginMiddlewares []gin.HandlerFunc
	for _, middleware := range middlewares {
		ginMiddlewares = append(ginMiddlewares, grg.adapter.convertMiddleware(middleware))
	}

	ginPath := grg.adapter.convertAxonPathToGin(path)
	handlers := append(ginMiddlewares, ginHandler)
	grg.group.Handle(method, ginPath, handlers...)
}

// Use registers middleware with the group
func (grg *GinRouteGroup) Use(middleware axon.MiddlewareFunc) {
	ginMiddleware := grg.adapter.convertMiddleware(middleware)
	grg.group.Use(ginMiddleware)
}

// Group creates a sub-group
func (grg *GinRouteGroup) Group(prefix string) axon.RouteGroup {
	subGroup := grg.group.Group(prefix)
	return &GinRouteGroup{group: subGroup, adapter: grg.adapter}
}

// convertHandler converts axon.HandlerFunc to gin.HandlerFunc
func (ga *GinAdapter) convertHandler(handler axon.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestContext := &GinRequestContext{ctx: c}
		if err := handler(requestContext); err != nil {
			// Handle error - Gin expects errors to be handled differently
			if httpErr, ok := err.(*axon.HTTPError); ok {
				c.JSON(httpErr.Code, gin.H{"error": httpErr.Message})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
		}
	}
}

// convertMiddleware converts axon.MiddlewareFunc to gin.HandlerFunc
func (ga *GinAdapter) convertMiddleware(middleware axon.MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestContext := &GinRequestContext{ctx: c}

		// Create a "next" function that calls c.Next()
		next := func(rc axon.RequestContext) error {
			c.Next()
			return nil
		}

		wrappedHandler := middleware(next)
		if err := wrappedHandler(requestContext); err != nil {
			// Handle error properly - convert axon.HTTPError
			if httpErr, ok := err.(*axon.HTTPError); ok {
				c.AbortWithStatusJSON(httpErr.Code, gin.H{"error": httpErr.Message})
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
		}
	}
}

// GinRequestContext implements axon.RequestContext for Gin
type GinRequestContext struct {
	ctx *gin.Context
}

// Method returns the HTTP method
func (grc *GinRequestContext) Method() string {
	return grc.ctx.Request.Method
}

// Path returns the request path
func (grc *GinRequestContext) Path() string {
	return grc.ctx.Request.URL.Path
}

// Param returns a path parameter
func (grc *GinRequestContext) Param(name string) string {
	if name == "*" || name == "path" {
		// Handle wildcard parameter - Gin uses *path for catch-all
		return grc.ctx.Param("path")
	}
	return grc.ctx.Param(name)
}

// QueryParam returns a query parameter
func (grc *GinRequestContext) QueryParam(name string) string {
	return grc.ctx.Query(name)
}

// QueryParams returns all query parameters
func (grc *GinRequestContext) QueryParams() map[string][]string {
	return grc.ctx.Request.URL.Query()
}

// RealIP returns the real IP address
func (grc *GinRequestContext) RealIP() string {
	return grc.ctx.ClientIP()
}

// ParamNames returns parameter names
func (grc *GinRequestContext) ParamNames() []string {
	var names []string
	for _, param := range grc.ctx.Params {
		names = append(names, param.Key)
	}
	return names
}

// ParamValues returns parameter values
func (grc *GinRequestContext) ParamValues() []string {
	var values []string
	for _, param := range grc.ctx.Params {
		values = append(values, param.Value)
	}
	return values
}

// SetParam sets a parameter value
func (grc *GinRequestContext) SetParam(name, value string) {
	grc.ctx.Params = append(grc.ctx.Params, gin.Param{Key: name, Value: value})
}

// QueryString returns the query string
func (grc *GinRequestContext) QueryString() string {
	return grc.ctx.Request.URL.RawQuery
}

// Request returns the request interface
func (grc *GinRequestContext) Request() axon.RequestInterface {
	return &GinRequestInterface{ctx: grc.ctx}
}

// Response returns the response interface
func (grc *GinRequestContext) Response() axon.ResponseInterface {
	return &GinResponseInterface{ctx: grc.ctx}
}

// Bind binds request body to a struct
func (grc *GinRequestContext) Bind(i interface{}) error {
	return grc.ctx.ShouldBindJSON(i)
}

// Validate validates a struct (placeholder implementation)
func (grc *GinRequestContext) Validate(i interface{}) error {
	// Gin doesn't have built-in validation, this would need to be implemented
	return nil
}

// Get returns a value from context
func (grc *GinRequestContext) Get(key string) interface{} {
	value, _ := grc.ctx.Get(key)
	return value
}

// Set sets a value in context
func (grc *GinRequestContext) Set(key string, val interface{}) {
	grc.ctx.Set(key, val)
}

// FormValue returns a form value
func (grc *GinRequestContext) FormValue(name string) string {
	return grc.ctx.PostForm(name)
}

// FormParams returns all form parameters
func (grc *GinRequestContext) FormParams() (map[string][]string, error) {
	err := grc.ctx.Request.ParseForm()
	if err != nil {
		return nil, err
	}
	return grc.ctx.Request.PostForm, nil
}

// FormFile returns a form file
func (grc *GinRequestContext) FormFile(name string) (axon.FileHeader, error) {
	file, header, err := grc.ctx.Request.FormFile(name)
	if err != nil {
		return nil, err
	}
	return &GinFileHeader{
		filename: header.Filename,
		size:     header.Size,
		header:   header.Header,
		file:     file,
	}, nil
}

// MultipartForm returns the multipart form
func (grc *GinRequestContext) MultipartForm() (axon.MultipartForm, error) {
	err := grc.ctx.Request.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		return nil, err
	}

	form := grc.ctx.Request.MultipartForm
	files := make(map[string][]axon.FileHeader)

	for key, fileHeaders := range form.File {
		var axonHeaders []axon.FileHeader
		for _, fh := range fileHeaders {
			file, err := fh.Open()
			if err != nil {
				continue
			}
			axonHeaders = append(axonHeaders, &GinFileHeader{
				filename: fh.Filename,
				size:     fh.Size,
				header:   fh.Header,
				file:     file,
			})
		}
		files[key] = axonHeaders
	}

	return &GinMultipartForm{
		values: form.Value,
		files:  files,
	}, nil
}

// GinRequestInterface implements axon.RequestInterface for Gin
type GinRequestInterface struct {
	ctx *gin.Context
}

// Header returns a request header
func (gri *GinRequestInterface) Header(key string) string {
	return gri.ctx.GetHeader(key)
}

// SetHeader sets a request header
func (gri *GinRequestInterface) SetHeader(key, value string) {
	gri.ctx.Request.Header.Set(key, value)
}

// Body returns the request body
func (gri *GinRequestInterface) Body() []byte {
	body, _ := io.ReadAll(gri.ctx.Request.Body)
	return body
}

// ContentLength returns the content length
func (gri *GinRequestInterface) ContentLength() int64 {
	return gri.ctx.Request.ContentLength
}

// ContentType returns the content type
func (gri *GinRequestInterface) ContentType() string {
	return gri.ctx.ContentType()
}

// Cookies returns the request cookies
func (gri *GinRequestInterface) Cookies() []axon.AxonCookie {
	var cookies []axon.AxonCookie
	for _, cookie := range gri.ctx.Request.Cookies() {
		cookies = append(cookies, axon.AxonCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			MaxAge:   cookie.MaxAge,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
			SameSite: convertGinSameSite(cookie.SameSite),
		})
	}
	return cookies
}

// Cookie returns a specific cookie
func (gri *GinRequestInterface) Cookie(name string) (axon.AxonCookie, error) {
	cookie, err := gri.ctx.Cookie(name)
	if err != nil {
		return axon.AxonCookie{}, err
	}

	// Gin's Cookie method only returns the value, so we need to find the full cookie
	for _, c := range gri.ctx.Request.Cookies() {
		if c.Name == name {
			return axon.AxonCookie{
				Name:     c.Name,
				Value:    c.Value,
				Path:     c.Path,
				Domain:   c.Domain,
				MaxAge:   c.MaxAge,
				Secure:   c.Secure,
				HttpOnly: c.HttpOnly,
				SameSite: convertGinSameSite(c.SameSite),
			}, nil
		}
	}

	return axon.AxonCookie{Name: name, Value: cookie}, nil
}

// GinResponseInterface implements axon.ResponseInterface for Gin
type GinResponseInterface struct {
	ctx *gin.Context
}

// Status returns the response status code
func (gri *GinResponseInterface) Status() int {
	return gri.ctx.Writer.Status()
}

// SetStatus sets the response status code
func (gri *GinResponseInterface) SetStatus(code int) {
	gri.ctx.Status(code)
}

// Header returns a response header
func (gri *GinResponseInterface) Header(key string) string {
	return gri.ctx.Writer.Header().Get(key)
}

// SetHeader sets a response header
func (gri *GinResponseInterface) SetHeader(key, value string) {
	gri.ctx.Header(key, value)
}

// JSON writes a JSON response
func (gri *GinResponseInterface) JSON(code int, i interface{}) error {
	gri.ctx.JSON(code, i)
	return nil
}

// JSONPretty writes a pretty JSON response
func (gri *GinResponseInterface) JSONPretty(code int, i interface{}, indent string) error {
	gri.ctx.IndentedJSON(code, i)
	return nil
}

// String writes a string response
func (gri *GinResponseInterface) String(code int, s string) error {
	gri.ctx.String(code, s)
	return nil
}

// HTML writes an HTML response
func (gri *GinResponseInterface) HTML(code int, html string) error {
	gri.ctx.Data(code, "text/html; charset=utf-8", []byte(html))
	return nil
}

// Blob writes a blob response
func (gri *GinResponseInterface) Blob(code int, contentType string, b []byte) error {
	gri.ctx.Data(code, contentType, b)
	return nil
}

// Stream writes a streaming response
func (gri *GinResponseInterface) Stream(code int, contentType string, r interface{}) error {
	if reader, ok := r.(io.Reader); ok {
		gri.ctx.DataFromReader(code, -1, contentType, reader, nil)
		return nil
	}
	return axon.NewHTTPError(500, "Invalid stream reader")
}

// SetCookie sets a response cookie
func (gri *GinResponseInterface) SetCookie(cookie axon.AxonCookie) {
	maxAge := cookie.MaxAge
	if cookie.MaxAge == 0 && !cookie.Expires.IsZero() {
		maxAge = int(time.Until(cookie.Expires).Seconds())
	}

	gri.ctx.SetCookie(
		cookie.Name,
		cookie.Value,
		maxAge,
		cookie.Path,
		cookie.Domain,
		cookie.Secure,
		cookie.HttpOnly,
	)
}

// Size returns the response size
func (gri *GinResponseInterface) Size() int64 {
	return int64(gri.ctx.Writer.Size())
}

// Written returns whether the response has been written
func (gri *GinResponseInterface) Written() bool {
	return gri.ctx.Writer.Written()
}

// Writer returns the underlying response writer
func (gri *GinResponseInterface) Writer() interface{} {
	return gri.ctx.Writer
}

// convertGinSameSite converts http.SameSite to axon.SameSiteMode
func convertGinSameSite(sameSite http.SameSite) axon.SameSiteMode {
	switch sameSite {
	case http.SameSiteStrictMode:
		return axon.SameSiteStrictMode
	case http.SameSiteLaxMode:
		return axon.SameSiteLaxMode
	case http.SameSiteNoneMode:
		return axon.SameSiteNoneMode
	default:
		return axon.SameSiteDefaultMode
	}
}
