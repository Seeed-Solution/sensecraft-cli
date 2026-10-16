package enum

const (
	OpenApiUrlViewOrganization = "/view_organization"

	OpenApiUrlGroupCreate      = "/create_group"
	OpenApiUrlGroupUpdate      = "/update_group"
	OpenApiUrlGroupDelete      = "/delete_group"
	OpenApiUrlGroupList        = "/list_groups"
	OpenApiUrlGroupMoveDevices = "/move_devices_to_group"

	OpenApiUrlDeviceList                 = "/list_devices"
	OpenApiUrlDeviceView                 = "/view_devices"
	OpenApiUrlDeviceChannels             = "/list_device_channels"
	OpenApiUrlDeviceRunningStatus        = "/view_device_running_status"
	OpenApiUrlDeviceBind                 = "/bind_device"
	OpenApiUrlDeviceDelete               = "/delete_devices"
	OpenApiUrlDeviceSimcard              = "/view_simcard"
	OpenApiUrlDeviceCreateDevelopmentKit = "/device/create_development_kit"
	OpenApiUrlDeviceViewNodeKey          = "/device/view_node_key"

	OpenApiUrlDataViewLatestTelemetry = "/view_latest_telemetry_data"
	OpenApiUrlDataListTelemetry       = "/list_telemetry_data"
	OpenApiUrlDataAggregateChart      = "/aggregate_chart_points"
)
