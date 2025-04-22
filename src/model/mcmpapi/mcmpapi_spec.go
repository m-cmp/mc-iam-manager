package mcmpapi // Change package name

// McmpApiAuthInfo holds authentication details for a service. (Renamed)
type McmpApiAuthInfo struct {
	Type     string `yaml:"type"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// McmpApiServiceDefinition defines a single mcmp service. (Renamed)
type McmpApiServiceDefinition struct {
	Version string          `yaml:"version"`
	BaseURL string          `yaml:"baseurl"`
	Auth    McmpApiAuthInfo `yaml:"auth"` // Use renamed AuthInfo
}

// McmpApiServiceAction defines a single API action for a service. (Renamed)
type McmpApiServiceAction struct {
	Method       string `yaml:"method"`
	ResourcePath string `yaml:"resourcePath"`
	Description  string `yaml:"description"`
}

// McmpApiDefinitions holds all parsed service and action definitions. (Renamed)
type McmpApiDefinitions struct {
	Services       map[string]McmpApiServiceDefinition        `yaml:"services"`       // Use renamed ServiceDefinition
	ServiceActions map[string]map[string]McmpApiServiceAction `yaml:"serviceActions"` // Use renamed ServiceAction
}
