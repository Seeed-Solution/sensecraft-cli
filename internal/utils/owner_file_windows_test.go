//go:build windows

package utils

import (
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWriteJSONRestrictsACLToCurrentUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sensecraft.json")
	if err := WriteJSON(path, map[string]string{"access_key": "secret"}, true); err != nil {
		t.Fatal(err)
	}

	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}

	descriptor, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil {
		t.Fatal("file has no dacl")
	}
	if dacl.AceCount != 1 {
		t.Fatalf("ace count = %d, want 1", dacl.AceCount)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !sid.Equals(tokenUser.User.Sid) {
		t.Fatal("file acl is not limited to the current user")
	}
}
