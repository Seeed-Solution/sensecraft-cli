package definition

type OpenApiResponse[T any] struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type OpenApiIntResponse[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type ListDeviceItem struct {
	DeviceEui  string `json:"device_eui"`
	DeviceName string `json:"device_name"`
}

type ListDeviceResponse []ListDeviceItem

type CreateDevelopmentKitResponse struct {
	EUI       string `json:"eui"`
	DeviceKey string `json:"device_key"`
}

type ViewNodeKeyResponse struct {
	EUI       string `json:"eui"`
	DeviceKey string `json:"device_key"`
	Token     string `json:"token"`
}

type ViewOrgInfoResponse struct {
	OrgID int64 `json:"orgId"`
}
