package enum

const (
	RemoteTypeGlobal  = "Global"
	RemoteTypeChina   = "China"
	RemoteTypePrivate = "Private"

	DomainNameOpenApiGlobal = "https://sensecap.seeed.cc"
	DomainNameOpenApiChina  = "https://sensecap.seeed.cn"
)

const (
	ApiKeyStatusActive  = "active"
	ApiKeyStatusDisable = "disable"
)

const (
	ApiAccessLevelOR = "read"
	ApiAccessLevelRW = "write"
)

const (
	EnvNameConfigFile = "SENSECRAFT_CONFIG_FILE"
	EnvNameConfigPath = "SENSECRAFT_CLI_PATH"

	// EnvNameDataHost overrides the SenseCAP OpenAPI origin (no /openapi suffix).
	// Empty means the built-in Global/China/Private host.
	EnvNameDataHost = "SENSECRAFT_DATA_HOST"
	// EnvNameAIHost overrides the SenseCraft AI origin (no /openapi suffix).
	// Empty means https://api-gateway.seeed.cc. Console pages stay on sensecraft.seeed.cc.
	EnvNameAIHost = "SENSECRAFT_AI_HOST"
	// EnvNameAuthHost overrides the account-center origin used by auth init
	// (device code, token, and the user APIs that exchange an Access Key).
	// Empty means account.seeed.cc or account.seeed.cn from the selected site.
	// A trailing /authapi is removed. The browser page is the verification URI
	// returned by that host, not a URL built by the CLI.
	EnvNameAuthHost = "SENSECRAFT_AUTH_HOST"
)
