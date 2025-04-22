package mcmpapi

// McmpApiRequestParams defines the structure for parameters needed in an API call.
type McmpApiRequestParams struct {
	PathParams  map[string]string `json:"pathParams"`  // Parameters to replace in the resource path (e.g., {userId})
	QueryParams map[string]string `json:"queryParams"` // Parameters to append as query string (?key=value)
	Body        interface{}       `json:"body"`        // Request body (accept any JSON structure) - Changed from json.RawMessage for swag compatibility
}

// McmpApiCallRequest defines the structure for the API call request body.
type McmpApiCallRequest struct {
	ServiceName   string               `json:"serviceName" validate:"required"` // Target service name
	ActionName    string               `json:"actionName" validate:"required"`  // Target action name (operationId)
	RequestParams McmpApiRequestParams `json:"requestParams"`                   // Parameters for the external API call
}

// --- Removed Structs for Generic API Call ---
