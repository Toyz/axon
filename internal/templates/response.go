package templates

import (
	"fmt"
	"slices"
	"strings"

	"github.com/toyz/axon/internal/models"
)

// ResponseHandlerData represents data needed for response handling template
type ResponseHandlerData struct {
	ReturnType       models.ReturnType
	HandlerCall      string
	ResponseHandling string
}

// GenerateResponseHandling generates response handling code based on handler return type
func GenerateResponseHandling(route models.RouteMetadata, controllerName string) (string, error) {
	handlerCall := generateHandlerCall(route, controllerName)

	// Check if err variable is already declared by parameter binding
	errAlreadyDeclared := hasPathParameters(route.Parameters)

	switch route.ReturnType.Type {
	case models.ReturnTypeDataError:
		return generateDataErrorResponse(handlerCall, errAlreadyDeclared), nil
	case models.ReturnTypeResponseError:
		return generateResponseErrorResponse(handlerCall, errAlreadyDeclared), nil
	case models.ReturnTypeError:
		return generateErrorResponse(handlerCall, errAlreadyDeclared), nil
	default:
		return "", fmt.Errorf("unsupported return type: %v", route.ReturnType.Type)
	}
}

// hasPathParameters checks if the route has path parameters that would declare err variable
func hasPathParameters(parameters []models.Parameter) bool {
	for _, param := range parameters {
		if param.Source == models.ParameterSourcePath {
			return true
		}
	}
	return false
}

// generateHandlerCall creates the handler method call with appropriate parameters
func generateHandlerCall(route models.RouteMetadata, controllerName string) string {
	// Create a slice to hold parameters in the correct order
	type paramWithPosition struct {
		name     string
		position int
		source   models.ParameterSource
	}

	var orderedParams []paramWithPosition

	// Add all parameters from the route based on method signature
	for _, param := range route.Parameters {
		switch param.Source {
		case models.ParameterSourceContext:
			// Always pass context if method expects it
			orderedParams = append(orderedParams, paramWithPosition{
				name:     "c", // Always use 'c' in the wrapper function
				position: param.Position,
				source:   param.Source,
			})
		case models.ParameterSourcePath:
			// For path parameters, only pass if method signature includes them
			// This is determined by the method signature analysis
			paramName := strings.TrimSuffix(param.Name, ":*") // Clean wildcard parameter names (remove :* suffix)
			orderedParams = append(orderedParams, paramWithPosition{
				name:     paramName,
				position: param.Position,
				source:   param.Source,
			})
		case models.ParameterSourceBody:
			orderedParams = append(orderedParams, paramWithPosition{
				name:     "body", // Always use 'body' for body parameters in wrapper
				position: param.Position,
				source:   param.Source,
			})
		}
	}

	// Assign positions to path and body parameters
	maxContextPosition := -1
	for _, param := range orderedParams {
		if param.source == models.ParameterSourceContext && param.position > maxContextPosition {
			maxContextPosition = param.position
		}
	}

	nextPosition := maxContextPosition + 1
	for i := range orderedParams {
		if orderedParams[i].position == -1 {
			orderedParams[i].position = nextPosition
			nextPosition++
		}
	}

	// Sort parameters by position
	for i := 0; i < len(orderedParams)-1; i++ {
		for j := i + 1; j < len(orderedParams); j++ {
			if orderedParams[i].position > orderedParams[j].position {
				orderedParams[i], orderedParams[j] = orderedParams[j], orderedParams[i]
			}
		}
	}

	// Extract parameter names in order
	var params []string
	for _, param := range orderedParams {
		params = append(params, param.name)
	}

	paramStr := strings.Join(params, ", ")

	// Extract method name from HandlerName (format: ControllerName.MethodName)
	parts := strings.Split(route.HandlerName, ".")
	methodName := route.HandlerName
	if len(parts) == 2 {
		methodName = parts[1]
	}

	return fmt.Sprintf("handler.%s(%s)", methodName, paramStr)
}

// generateDataErrorResponse generates response handling for (data, error) return type
func generateDataErrorResponse(handlerCall string, errAlreadyDeclared bool) string {
	if errAlreadyDeclared {
		return fmt.Sprintf(`		var data interface{}
		data, err = %s
		if err != nil {
			return handleError(c, err)
		}
		return c.JSON(http.StatusOK, data)`, handlerCall)
	} else {
		return fmt.Sprintf(`		data, err := %s
		if err != nil {
			return handleError(c, err)
		}
		return c.JSON(http.StatusOK, data)`, handlerCall)
	}
}

// generateResponseErrorResponse generates response handling for (*Response, error) return type
func generateResponseErrorResponse(handlerCall string, errAlreadyDeclared bool) string {
	responseHandling := `
		return handleAxonResponse(c, response)`

	if errAlreadyDeclared {
		return fmt.Sprintf(`		var response *axon.Response
		response, err = %s
		if err != nil {
			return handleError(c, err)
		}
		if response == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "handler returned nil response")
		}%s`, handlerCall, responseHandling)
	} else {
		return fmt.Sprintf(`		response, err := %s
		if err != nil {
			return handleError(c, err)
		}
		if response == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "handler returned nil response")
		}%s`, handlerCall, responseHandling)
	}
}

// generateErrorResponse generates response handling for error return type
func generateErrorResponse(handlerCall string, errAlreadyDeclared bool) string {
	var assignment string
	if errAlreadyDeclared {
		assignment = "err ="
	} else {
		assignment = "err :="
	}

	return fmt.Sprintf(`		%s %s
		if err != nil {
			return err
		}
		return nil`, assignment, handlerCall)
}

// hasPassContextFlag checks if the route has the PassContext flag
func hasPassContextFlag(flags []string) bool {
	return slices.Contains(flags, "-PassContext")
}

// GenerateRouteWrapper generates a complete route wrapper function
func GenerateRouteWrapper(route models.RouteMetadata, controllerName string, parserRegistry ParserRegistryInterface) (string, error) {
	wrapperName := fmt.Sprintf("wrap%s%s", controllerName, route.HandlerName)

	// Generate parameter binding code
	paramBindingCode, err := GenerateParameterBindingCode(route.Parameters, parserRegistry)
	if err != nil {
		return "", fmt.Errorf("failed to generate parameter binding: %w", err)
	}

	// Generate body binding code if needed
	bodyBindingCode := generateBodyBindingCode(route.Parameters, route.Method)

	// Generate response handling code
	responseHandlingCode, err := GenerateResponseHandling(route, controllerName)
	if err != nil {
		return "", fmt.Errorf("failed to generate response handling: %w", err)
	}

	// Generate middleware application code
	middlewareCode := generateMiddlewareApplication(route.Middlewares)

	var template string
	if len(route.Middlewares) > 0 {
		// Template with middleware application
		template = `func %s(handler *%s, %s) echo.HandlerFunc {
	baseHandler := func(c echo.Context) error {
%s%s
%s
	}
%s
	return finalHandler
}`
	} else {
		// Template without middleware
		template = `func %s(handler *%s) echo.HandlerFunc {
	return func(c echo.Context) error {
%s%s
%s
	}
}`
	}

	if len(route.Middlewares) > 0 {
		middlewareParams := generateMiddlewareParameters(route.Middlewares)
		return fmt.Sprintf(template,
			wrapperName,
			controllerName,
			middlewareParams,
			paramBindingCode,
			bodyBindingCode,
			responseHandlingCode,
			middlewareCode), nil
	} else {
		return fmt.Sprintf(template,
			wrapperName,
			controllerName,
			paramBindingCode,
			bodyBindingCode,
			responseHandlingCode), nil
	}
}

// generateBodyBindingCode generates body parameter binding code
func generateBodyBindingCode(parameters []models.Parameter, method string) string {
	// Don't generate body binding for GET requests
	if method == "GET" {
		return ""
	}

	for _, param := range parameters {
		if param.Source == models.ParameterSourceBody {
			return fmt.Sprintf(`		var body %s
		if err := c.Bind(&body); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
`, param.Type)
		}
	}
	return ""
}

// generateMiddlewareParameters generates the parameter list for middleware dependencies
func generateMiddlewareParameters(middlewares []string) string {
	var params []string
	for _, middleware := range middlewares {
		params = append(params, fmt.Sprintf("%s *%s", strings.ToLower(middleware), middleware))
	}
	return strings.Join(params, ", ")
}

// generateMiddlewareApplication generates code to apply middlewares in the specified order
func generateMiddlewareApplication(middlewares []string) string {
	if len(middlewares) == 0 {
		return ""
	}

	var code strings.Builder
	code.WriteString("\n	// Apply middlewares in order\n")
	code.WriteString("	finalHandler := baseHandler\n")

	// Apply middlewares in reverse order so they execute in the correct order
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		middlewareVar := strings.ToLower(middleware)
		code.WriteString(fmt.Sprintf("	finalHandler = %s.Handle(finalHandler)\n", middlewareVar))
	}

	return code.String()
}

// GenerateRouteRegistration generates the Echo route registration line with registry integration
func GenerateRouteRegistration(route models.RouteMetadata, controllerVar string, middlewares []string) (string, error) {
	// Build the handler call
	handlerCall := fmt.Sprintf("wrap%s%s(%s", controllerVar, route.HandlerName, controllerVar)

	// Add middleware parameters
	for _, middleware := range middlewares {
		handlerCall += fmt.Sprintf(", %s", strings.ToLower(middleware))
	}
	handlerCall += ")"

	// Convert Axon route syntax to Echo syntax
	echoPath := convertAxonPathToEcho(route.Path)

	// Generate handler variable name
	handlerVarName := fmt.Sprintf("handler_%s%s",
		strings.ToLower(controllerVar),
		strings.ToLower(route.HandlerName))

	// Build parameter types map
	paramTypesMap := "map[string]string{"
	if len(route.Parameters) > 0 {
		var paramPairs []string
		for _, param := range route.Parameters {
			if param.Source == models.ParameterSourcePath {
				// Clean parameter name by removing :* suffix for wildcard parameters
				cleanParamName := strings.TrimSuffix(param.Name, ":*")
				paramPairs = append(paramPairs, fmt.Sprintf(`"%s": "%s"`, cleanParamName, param.Type))
			}
		}
		if len(paramPairs) > 0 {
			paramTypesMap += strings.Join(paramPairs, ", ")
		}
	}
	paramTypesMap += "}"

	// Build middlewares array
	middlewaresArray := "[]string{"
	if len(middlewares) > 0 {
		var middlewareNames []string
		for _, middleware := range middlewares {
			middlewareNames = append(middlewareNames, fmt.Sprintf(`"%s"`, middleware))
		}
		middlewaresArray += strings.Join(middlewareNames, ", ")
	}
	middlewaresArray += "}"

	// Build middleware instances array
	middlewareInstancesArray := "[]axon.MiddlewareInstance{"
	if len(middlewares) > 0 {
		var instanceEntries []string
		for _, middleware := range middlewares {
			middlewareVar := strings.ToLower(middleware)
			instanceEntry := fmt.Sprintf(`{
			Name:     "%s",
			Handler:  %s.Handle,
			Instance: %s,
		}`, middleware, middlewareVar, middlewareVar)
			instanceEntries = append(instanceEntries, instanceEntry)
		}
		middlewareInstancesArray += strings.Join(instanceEntries, ", ")
	}
	middlewareInstancesArray += "}"

	// Generate the complete registration code
	template := `%s := %s
	e.%s("%s", %s)
	axon.DefaultRouteRegistry.RegisterRoute(axon.RouteInfo{
		Method:              "%s",
		Path:                "%s",
		EchoPath:            "%s",
		HandlerName:         "%s",
		ControllerName:      "%s",
		PackageName:         "PACKAGE_NAME",
		Middlewares:         %s,
		MiddlewareInstances: %s,
		ParameterTypes:      %s,
		Handler:             %s,
	})`

	return fmt.Sprintf(template,
		handlerVarName, handlerCall,
		route.Method, echoPath, handlerVarName,
		route.Method, route.Path, echoPath, route.HandlerName, controllerVar,
		middlewaresArray, middlewareInstancesArray, paramTypesMap, handlerVarName), nil
}

// convertAxonPathToEcho converts Axon route syntax to Echo route syntax
// Axon: /users/{id:int} -> Echo: /users/:id
// Axon: /files/{*} -> Echo: /files/*
func convertAxonPathToEcho(axonPath string) string {
	result := axonPath

	// Handle wildcard first: /files/{*} -> /files/*
	if strings.Contains(result, "{*}") {
		result = strings.ReplaceAll(result, "{*}", "*")
	}

	// Replace Axon parameter syntax {param:type} with Echo syntax :param
	for {
		start := strings.Index(result, "{")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start

		// Extract parameter definition: {id:int} -> id:int
		paramDef := result[start+1 : end]

		// Split by colon to get parameter name
		parts := strings.Split(paramDef, ":")
		if len(parts) > 0 {
			paramName := strings.TrimSpace(parts[0])
			// Replace {param:type} with :param
			result = result[:start] + ":" + paramName + result[end+1:]
		} else {
			// Invalid format, skip
			break
		}
	}

	return result
}
