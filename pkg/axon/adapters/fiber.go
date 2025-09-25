package adapters

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/toyz/axon/pkg/axon"
)

// FiberAdapter wraps a Fiber app to implement axon.WebServerInterface
type FiberAdapter struct {
	app *fiber.App
}

// NewFiberAdapter creates a new Fiber adapter instance
func NewFiberAdapter() *FiberAdapter {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Handle errors appropriately
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	return &FiberAdapter{app: app}
}

// NewDefaultFiberAdapter creates a new Fiber adapter with default middleware
func NewDefaultFiberAdapter() *FiberAdapter {
	adapter := NewFiberAdapter()

	// Add default middleware
	adapter.app.Use(logger.New())
	adapter.app.Use(recover.New())

	return adapter
}

// convertAxonPathToFiber converts AxonPath to Fiber path format
func (fa *FiberAdapter) convertAxonPathToFiber(path axon.AxonPath) string {
	parts := path.Parts()
	fiberPath := ""
	for _, part := range parts {
		if part.Value == "" || part.Value == "/" {
			continue
		}

		switch part.Type {
		case axon.StaticPart:
			fiberPath += part.Value
		case axon.ParameterPart:
			fiberPath += ":" + part.Value
		case axon.WildcardPart:
			fiberPath += "*"
		default:
			fiberPath += part.Value
		}
	}
	return fiberPath
}

// RegisterRoute registers a route with the Fiber app
func (fa *FiberAdapter) RegisterRoute(method string, path axon.AxonPath, handler axon.HandlerFunc, middlewares ...axon.MiddlewareFunc) {
	// Convert Axon path format to Fiber format
	fiberPath := fa.convertAxonPathToFiber(path)

	// Convert middlewares to Fiber handlers
	var fiberMiddlewares []fiber.Handler
	for _, mw := range middlewares {
		fiberMiddlewares = append(fiberMiddlewares, convertAxonMiddlewareToFiber(mw))
	}

	// Convert main handler
	fiberHandler := convertAxonHandlerToFiber(handler)

	// Combine middlewares and handler
	handlers := append(fiberMiddlewares, fiberHandler)

	// Register the route
	switch strings.ToUpper(method) {
	case "GET":
		fa.app.Get(fiberPath, handlers...)
	case "POST":
		fa.app.Post(fiberPath, handlers...)
	case "PUT":
		fa.app.Put(fiberPath, handlers...)
	case "DELETE":
		fa.app.Delete(fiberPath, handlers...)
	case "PATCH":
		fa.app.Patch(fiberPath, handlers...)
	case "OPTIONS":
		fa.app.Options(fiberPath, handlers...)
	case "HEAD":
		fa.app.Head(fiberPath, handlers...)
	}
}

// RegisterGroup creates a new route group with the given prefix
func (fa *FiberAdapter) RegisterGroup(prefix string) axon.RouteGroup {
	fiberGroup := fa.app.Group(prefix)
	return &FiberRouteGroup{group: fiberGroup, adapter: fa}
}

// Use adds middleware to the Fiber app
func (fa *FiberAdapter) Use(middleware axon.MiddlewareFunc) {
	fa.app.Use(convertAxonMiddlewareToFiber(middleware))
}

// Start starts the Fiber server
func (fa *FiberAdapter) Start(addr string) error {
	return fa.app.Listen(addr)
}

// Stop stops the Fiber server
func (fa *FiberAdapter) Stop(ctx context.Context) error {
	return fa.app.Shutdown()
}

// Name returns the adapter name
func (fa *FiberAdapter) Name() string {
	return "Fiber"
}

// FiberRouteGroup wraps a Fiber route group to implement axon.RouteGroup
type FiberRouteGroup struct {
	group   fiber.Router
	adapter *FiberAdapter
}

// RegisterRoute registers a route with this group
func (frg *FiberRouteGroup) RegisterRoute(method string, path axon.AxonPath, handler axon.HandlerFunc, middlewares ...axon.MiddlewareFunc) {
	// Convert Axon path format to Fiber format
	fiberPath := frg.adapter.convertAxonPathToFiber(path)

	// Convert middlewares to Fiber handlers
	var fiberMiddlewares []fiber.Handler
	for _, mw := range middlewares {
		fiberMiddlewares = append(fiberMiddlewares, convertAxonMiddlewareToFiber(mw))
	}

	// Convert main handler
	fiberHandler := convertAxonHandlerToFiber(handler)

	// Combine middlewares and handler
	handlers := append(fiberMiddlewares, fiberHandler)

	// Register the route
	switch strings.ToUpper(method) {
	case "GET":
		frg.group.Get(fiberPath, handlers...)
	case "POST":
		frg.group.Post(fiberPath, handlers...)
	case "PUT":
		frg.group.Put(fiberPath, handlers...)
	case "DELETE":
		frg.group.Delete(fiberPath, handlers...)
	case "PATCH":
		frg.group.Patch(fiberPath, handlers...)
	case "OPTIONS":
		frg.group.Options(fiberPath, handlers...)
	case "HEAD":
		frg.group.Head(fiberPath, handlers...)
	}
}

// Use adds middleware to this route group
func (frg *FiberRouteGroup) Use(middleware axon.MiddlewareFunc) {
	frg.group.Use(convertAxonMiddlewareToFiber(middleware))
}

// Group creates a sub-group with the given prefix
func (frg *FiberRouteGroup) Group(prefix string) axon.RouteGroup {
	subGroup := frg.group.Group(prefix)
	return &FiberRouteGroup{group: subGroup, adapter: frg.adapter}
}

// convertAxonHandlerToFiber converts an Axon handler to a Fiber handler
func convertAxonHandlerToFiber(handler axon.HandlerFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create Axon request context wrapper
		axonCtx := &FiberRequestContext{ctx: c}

		// Call the Axon handler
		err := handler(axonCtx)
		if err != nil {
			// Handle error - convert to appropriate Fiber response
			if httpErr, ok := err.(*axon.HTTPError); ok {
				return c.Status(httpErr.Code).JSON(httpErr)
			} else {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}
		return nil
	}
}

// convertAxonMiddlewareToFiber converts an Axon middleware to a Fiber middleware
func convertAxonMiddlewareToFiber(middleware axon.MiddlewareFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create Axon request context wrapper
		axonCtx := &FiberRequestContext{ctx: c}

		// Call the Axon middleware
		err := middleware(func(ctx axon.RequestContext) error {
			// Continue to next middleware/handler
			return c.Next()
		})(axonCtx)

		if err != nil {
			// Handle error - convert to appropriate Fiber response
			if httpErr, ok := err.(*axon.HTTPError); ok {
				return c.Status(httpErr.Code).JSON(httpErr)
			} else {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}
		return nil
	}
}

// FiberRequestContext wraps fiber.Ctx to implement axon.RequestContext
type FiberRequestContext struct {
	ctx *fiber.Ctx
}

// Request data methods
func (frc *FiberRequestContext) Method() string {
	return frc.ctx.Method()
}

func (frc *FiberRequestContext) Path() string {
	return frc.ctx.Path()
}

func (frc *FiberRequestContext) RealIP() string {
	return frc.ctx.IP()
}

// Parameter methods
func (frc *FiberRequestContext) Param(name string) string {
	// Handle wildcard parameter
	if name == "*" {
		return frc.ctx.Params("*")
	}
	return frc.ctx.Params(name)
}

func (frc *FiberRequestContext) ParamNames() []string {
	// Fiber doesn't provide direct access to param names, but we can extract from route
	return []string{} // Simplified implementation
}

func (frc *FiberRequestContext) ParamValues() []string {
	// Fiber doesn't provide direct access to param values array
	return []string{} // Simplified implementation
}

func (frc *FiberRequestContext) SetParam(name, value string) {
	// Fiber doesn't support setting params after route matching
	// This is a no-op for Fiber
}

// Query parameter methods
func (frc *FiberRequestContext) QueryParam(key string) string {
	return frc.ctx.Query(key)
}

func (frc *FiberRequestContext) QueryParams() map[string][]string {
	result := make(map[string][]string)
	frc.ctx.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)
		result[keyStr] = append(result[keyStr], valueStr)
	})
	return result
}

func (frc *FiberRequestContext) QueryString() string {
	return string(frc.ctx.Request().URI().QueryString())
}

// Request and Response interfaces
func (frc *FiberRequestContext) Request() axon.RequestInterface {
	return &FiberRequest{ctx: frc.ctx}
}

func (frc *FiberRequestContext) Response() axon.ResponseInterface {
	return &FiberResponse{ctx: frc.ctx}
}

// Body handling
func (frc *FiberRequestContext) Bind(obj interface{}) error {
	return frc.ctx.BodyParser(obj)
}

func (frc *FiberRequestContext) Validate(obj interface{}) error {
	// Fiber doesn't have built-in validation, return nil for now
	return nil
}

// Context data
func (frc *FiberRequestContext) Get(key string) interface{} {
	return frc.ctx.Locals(key)
}

func (frc *FiberRequestContext) Set(key string, val interface{}) {
	frc.ctx.Locals(key, val)
}

// Form handling
func (frc *FiberRequestContext) FormValue(name string) string {
	return frc.ctx.FormValue(name)
}

func (frc *FiberRequestContext) FormParams() (map[string][]string, error) {
	// Simplified implementation
	return make(map[string][]string), nil
}

func (frc *FiberRequestContext) FormFile(name string) (axon.FileHeader, error) {
	// Get the multipart form first to access the underlying multipart.FileHeader
	form, err := frc.ctx.MultipartForm()
	if err != nil {
		return nil, err
	}

	files, exists := form.File[name]
	if !exists || len(files) == 0 {
		return nil, fiber.ErrBadRequest
	}

	// Convert Fiber's multipart file to standard multipart.FileHeader
	fiberFile := files[0]
	header := &multipart.FileHeader{
		Filename: fiberFile.Filename,
		Size:     fiberFile.Size,
		Header:   make(map[string][]string),
	}

	// Copy headers from Fiber file
	for key, values := range fiberFile.Header {
		header.Header[key] = values
	}

	return &FiberFileHeader{header: header}, nil
}

func (frc *FiberRequestContext) MultipartForm() (axon.MultipartForm, error) {
	// Get Fiber's multipart form
	fiberForm, err := frc.ctx.MultipartForm()
	if err != nil {
		return nil, err
	}

	// Convert to standard multipart.Form
	form := &multipart.Form{
		Value: fiberForm.Value,
		File:  make(map[string][]*multipart.FileHeader),
	}

	// Convert Fiber files to standard multipart.FileHeaders
	for key, fiberFiles := range fiberForm.File {
		for _, fiberFile := range fiberFiles {
			header := &multipart.FileHeader{
				Filename: fiberFile.Filename,
				Size:     fiberFile.Size,
				Header:   make(map[string][]string),
			}

			// Copy headers from Fiber file
			for key, values := range fiberFile.Header {
				header.Header[key] = values
			}

			form.File[key] = append(form.File[key], header)
		}
	}

	return &FiberMultipartForm{form: form}, nil
}

// FiberFileHeader wraps multipart.FileHeader to implement axon.FileHeader
type FiberFileHeader struct {
	header *multipart.FileHeader
}

func (ffh *FiberFileHeader) Filename() string {
	return ffh.header.Filename
}

func (ffh *FiberFileHeader) Size() int64 {
	return ffh.header.Size
}

func (ffh *FiberFileHeader) Header() map[string][]string {
	return ffh.header.Header
}

func (ffh *FiberFileHeader) Open() (interface{}, error) {
	return ffh.header.Open()
}

// FiberMultipartForm wraps multipart.Form to implement axon.MultipartForm
type FiberMultipartForm struct {
	form *multipart.Form
}

func (fmf *FiberMultipartForm) Value() map[string][]string {
	return fmf.form.Value
}

func (fmf *FiberMultipartForm) File() map[string][]axon.FileHeader {
	result := make(map[string][]axon.FileHeader)
	for key, files := range fmf.form.File {
		for _, file := range files {
			result[key] = append(result[key], &FiberFileHeader{header: file})
		}
	}
	return result
}

// FiberRequest wraps fiber.Ctx to implement axon.RequestInterface
type FiberRequest struct {
	ctx *fiber.Ctx
}

func (fr *FiberRequest) Header(key string) string {
	return fr.ctx.Get(key)
}

func (fr *FiberRequest) SetHeader(key, value string) {
	// Fiber doesn't allow modifying request headers after creation
	// This is a no-op
}

func (fr *FiberRequest) Body() []byte {
	return fr.ctx.Body()
}

func (fr *FiberRequest) ContentLength() int64 {
	return int64(len(fr.ctx.Body()))
}

func (fr *FiberRequest) ContentType() string {
	return fr.ctx.Get(fiber.HeaderContentType)
}

func (fr *FiberRequest) Cookies() []axon.AxonCookie {
	// Simplified implementation
	return []axon.AxonCookie{}
}

func (fr *FiberRequest) Cookie(name string) (axon.AxonCookie, error) {
	value := fr.ctx.Cookies(name)
	if value == "" {
		return axon.AxonCookie{}, fiber.ErrBadRequest
	}
	return axon.AxonCookie{
		Name:  name,
		Value: value,
	}, nil
}

// FiberResponse wraps fiber.Ctx to implement axon.ResponseInterface
type FiberResponse struct {
	ctx *fiber.Ctx
}

// Status methods
func (fr *FiberResponse) Status() int {
	return fr.ctx.Response().StatusCode()
}

func (fr *FiberResponse) SetStatus(code int) {
	fr.ctx.Status(code)
}

// Header methods
func (fr *FiberResponse) Header(key string) string {
	return string(fr.ctx.Response().Header.Peek(key))
}

func (fr *FiberResponse) SetHeader(name, value string) {
	fr.ctx.Set(name, value)
}

// Content methods
func (fr *FiberResponse) JSON(code int, data interface{}) error {
	return fr.ctx.Status(code).JSON(data)
}

func (fr *FiberResponse) JSONPretty(code int, data interface{}, indent string) error {
	return fr.ctx.Status(code).JSON(data) // Fiber doesn't support pretty JSON directly
}

func (fr *FiberResponse) String(code int, s string) error {
	return fr.ctx.Status(code).SendString(s)
}

func (fr *FiberResponse) HTML(code int, html string) error {
	fr.ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return fr.ctx.Status(code).SendString(html)
}

func (fr *FiberResponse) Blob(code int, contentType string, data []byte) error {
	fr.ctx.Set(fiber.HeaderContentType, contentType)
	return fr.ctx.Status(code).Send(data)
}

func (fr *FiberResponse) Stream(code int, contentType string, r interface{}) error {
	fr.ctx.Set(fiber.HeaderContentType, contentType)
	fr.ctx.Status(code)
	if reader, ok := r.(io.Reader); ok {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		return fr.ctx.Send(data)
	}
	return fr.ctx.SendString("unsupported stream type")
}

// Cookie methods
func (fr *FiberResponse) SetCookie(cookie axon.AxonCookie) {
	fiberCookie := &fiber.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		MaxAge:   cookie.MaxAge,
		Secure:   cookie.Secure,
		HTTPOnly: cookie.HttpOnly,
	}

	// Handle SameSite attribute
	switch cookie.SameSite {
	case axon.SameSiteStrictMode:
		fiberCookie.SameSite = "Strict"
	case axon.SameSiteLaxMode:
		fiberCookie.SameSite = "Lax"
	case axon.SameSiteNoneMode:
		fiberCookie.SameSite = "None"
	default:
		fiberCookie.SameSite = "Lax" // Default
	}

	fr.ctx.Cookie(fiberCookie)
}

// Response data methods
func (fr *FiberResponse) Size() int64 {
	return int64(len(fr.ctx.Response().Body()))
}

func (fr *FiberResponse) Written() bool {
	return fr.ctx.Response().StatusCode() != 0
}

func (fr *FiberResponse) Writer() interface{} {
	return fr.ctx.Response().BodyWriter()
}