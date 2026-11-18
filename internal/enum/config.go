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
	// Empty means https://api-gateway.seeed.cc for every site.
	// This does not change page links. App pages use EnvNameAppSite.
	// Model pages use EnvNameConsoleHost.
	EnvNameAIHost = "SENSECRAFT_AI_HOST"
	// EnvNameAppSite overrides the CodeCraft Hub base printed in app links.
	// Empty means https://codecraft.seeed.cc/hub. A trailing slash is removed.
	// The value may include a path such as /hub. Query strings are rejected.
	EnvNameAppSite = "SENSECRAFT_APP_SITE"
	// EnvNameConsoleHost overrides the SenseCraft AI website origin used in model links.
	// Empty means https://sensecraft.seeed.cc. A trailing slash, /openapi, or /authapi is removed.
	EnvNameConsoleHost = "SENSECRAFT_CONSOLE_HOST"
	// EnvNameAuthHost overrides the account-center origin used by auth init
	// (device code, token, and the user APIs that exchange an Access Key).
	// Empty means account.seeed.cc or account.seeed.cn from the selected site.
	// A trailing /authapi is removed. The browser page is the verification URI
	// returned by that host, not a URL built by the CLI.
	EnvNameAuthHost = "SENSECRAFT_AUTH_HOST"
)
