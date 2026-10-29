package authhint

import "fmt"

// NeedLogin is the next command when no Access Key is saved.
func NeedLogin() string {
	return "No Access Key configured. Run: sensecraft-cli auth init --site Global --no-wait. Show verification_url to the user unchanged and end this turn. After they authorize, run the --device-code command from that command's hint."
}

// HaveKey is the next step when a key is already configured but this call cannot use it.
func HaveKey(problem string) string {
	return problem + " Run: sensecraft-cli auth info. Ask the user which access_id to use."
}

// DeviceCodeIssued tells the agent how to hand the browser step back to the user.
func DeviceCodeIssued(deviceCode string) string {
	return fmt.Sprintf("Show verification_url to the user unchanged and end this turn. Do not change the URL. Do not run --device-code in this turn. Do not run --no-wait again; that invalidates this code. After the user authorizes, run: sensecraft-cli auth init --device-code %s", deviceCode)
}

// ChooseAccessKey tells the agent to let the user select an existing key or name a new one.
func ChooseAccessKey(deviceCode string) string {
	return fmt.Sprintf("Ask the user to select one existing Access Key, or create a new one by providing the Access Key name. Do not pick a key yourself. To select: sensecraft-cli auth init --device-code %s --key-id <access_id>. To create: sensecraft-cli auth init --device-code %s --label <name> --access-level read-write", deviceCode, deviceCode)
}

// DeviceCodeExpired is returned when the authorization code can no longer be polled.
func DeviceCodeExpired() string {
	return "authorization expired. Run: sensecraft-cli auth init --site Global --no-wait. Do not reuse the previous device code."
}
