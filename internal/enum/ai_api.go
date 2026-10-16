package enum

const (
	DomainNameAIServerGlobal = "https://sensecraft.seeed.cc"
	DomainNameAIServerChina  = "https://sensecraft.seeed.cc" // SenseCraft AI is single-domain; China uses CN account + same AI host

	AIApiPathAppList              = "/flow/app/v2/list_app"
	AIApiPathAppView              = "/flow/app/v3/view_app"
	AIApiPathAppClone             = "/flow/app/v3/clone_app"
	AIApiPathAppPrivacy           = "/flow/app/v3/edit_privacy_app"
	AIApiPathAppDelete            = "/flow/app/v2/delete_app"
	AIApiPathAppCreate            = "/flow/app/v3/create_app"
	AIApiPathAppUpdate            = "/flow/app/v3/update_app"
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
)

// ResolveAIServerHost returns the SenseCraft AI site origin (without /openapi) for remote_type.
func ResolveAIServerHost(remoteType string) string {
	switch remoteType {
	case RemoteTypeChina:
		return DomainNameAIServerChina
	default:
		return DomainNameAIServerGlobal
	}
}
