package enum

const (
	DeviceTypeGtw  = 1
	DeviceTypeNode = 2
	DeviceTypeKit  = 5
)

const (
	DevKitSKUBlankDevice   = "blank_device"
	DevKitSKUXiao          = "xiao"
	DevKitSKUGroveVisionAI = "grove_vision_ai"
)

var DevKitSKUList = []string{
	DevKitSKUBlankDevice,
	DevKitSKUXiao,
	DevKitSKUGroveVisionAI,
}
