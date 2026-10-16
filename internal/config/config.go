package config

const (
	// ProfileKindSenseCAP is SenseCAP OpenAPI Access Key auth.
	// The same key is used for SenseCraft AI /openapi.
	ProfileKindSenseCAP = "sensecap"
)

// AccessApiConfig is a SenseCAP OpenAPI Access Key profile (legacy shape preserved).
type AccessApiConfig struct {
	RemoteType string  `json:"remote_type,omitempty"`
	Host       *string `json:"host,omitempty"`
	OrgId      string  `json:"org_id,omitempty"`
	KeyId      string  `json:"key_id,omitempty"`
	AccessKey  string  `json:"access_key,omitempty"`
	CreateTime string  `json:"create_time,omitempty"`
}

type CliConfig struct {
	AppConfigList []AccessApiConfig `json:"app_config_list,omitempty"`
	UseApi        string            `json:"use_api,omitempty"`
}

var cfg *CliConfig

func LoadConfig() (*CliConfig, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg, err := loadCliConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
