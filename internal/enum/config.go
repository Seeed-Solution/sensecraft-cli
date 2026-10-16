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
)
