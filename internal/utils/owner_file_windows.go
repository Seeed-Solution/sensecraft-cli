//go:build windows

package utils

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

// restrictToCurrentUser 把文件的访问控制改成只允许当前用户，并断开从目录继承来的权限。
func restrictToCurrentUser(path string) error {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return fmt.Errorf("open process token: %w", err)
	}
	defer token.Close()

	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("read process user: %w", err)
	}
	sid, err := tokenUser.User.Sid.Copy()
	if err != nil {
		return fmt.Errorf("copy user sid: %w", err)
	}
	runtime.KeepAlive(tokenUser)

	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}}, nil)
	if err != nil {
		return fmt.Errorf("build file acl: %w", err)
	}
	runtime.KeepAlive(sid)

	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	); err != nil {
		return fmt.Errorf("set file acl: %w", err)
	}
	return nil
}
