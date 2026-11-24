package enum

const (
	DomainNameAIServerGlobal = "https://api-gateway.seeed.cc"
	DomainNameAIServerChina  = "https://api-gateway.seeed.cc" // SenseCraft AI is single-domain; China uses CN account + same AI host
	// DomainNameAIConsole is the SenseCraft AI website. API calls use the gateway above.
	DomainNameAIConsole = "https://sensecraft.seeed.cc"
	// DomainNameAppHub is the CodeCraft Hub site printed in app links.
	DomainNameAppHub = "https://codecraft.seeed.cc/hub"

	AIApiPathAppList              = "/flow/app/v2/list_app"
	AIApiPathAppView              = "/flow/app/v3/view_app"
	AIApiPathAppPrivacy           = "/flow/app/v3/edit_privacy_app"
	AIApiPathAppDelete            = "/flow/app/v2/delete_app"
	AIApiPathAppCreate            = "/flow/app/v3/create_app"
	AIApiPathAppUpdate            = "/flow/app/v3/update_app"
	AIApiPathAppClone             = "/flow/app/v3/clone_app"
	AIApiPathAppCallbackSketch    = "/flow/app/v3/callback_sketch"
	AIApiPathModelList            = "/model/list_model"
	AIApiPathModelMine            = "/model/list_custom_model"
	AIApiPathModelView            = "/model/view_model"
	AIApiPathModelApply           = "/model/apply_model"
	AIApiPathModelPublish         = "/model/publish_model"
	AIApiPathModelDelete          = "/model/delete_model"
	AIApiPathModelCreate          = "/model/create_model"
	AIApiPathModelEdit            = "/model/edit_model"
	AIApiPathAcquireFile          = "/user/acquire_file_url"
	AIApiPathGetConstant          = "/user/get_constant"
	AIApiPathGetProductionOptions = "/user/get_production_options"
	AIApiPathDeviceListNode       = "/device/list_node"
	AIApiPathDeviceView           = "/device/view_device"
	AIApiPathDeviceUpdateStream   = "/device/update_stream"
	AIApiPathDeviceOrderRecord    = "/device/get_order_record"
	AIApiPathActivityList         = "/activity/list"
)

// ResolveAIServerHost returns the SenseCraft AI API origin (without /openapi) for remote_type.
func ResolveAIServerHost(remoteType string) string {
	switch remoteType {
	case RemoteTypeChina:
		return DomainNameAIServerChina
	default:
		return DomainNameAIServerGlobal
	}
}
