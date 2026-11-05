package app

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestListOutputCount(t *testing.T) {
	for _, tt := range []struct {
		name string
		data string
		want string
	}{
		{"page preserved", `{"list":[{"app_id":"1"}],"page":2}`, `{"list":[{"app_id":"1"}],"page":2,"count":1}`},
		{"empty", `{"list":[]}`, `{"list":[],"count":0}`},
		{"null list", `{"list":null}`, `{"list":null,"count":0}`},
		{"existing total", `{"list":[1],"count":42}`, `{"list":[1],"count":42}`},
		{"scalar list", `{"list":"value"}`, `{"list":"value"}`},
		{"array", `[1,2]`, `[1,2]`},
		{"large ID", `{"list":[9007199254740993]}`, `{"list":[9007199254740993],"count":1}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, ai := range []bool{false, true} {
				var out bytes.Buffer
				var err error
				if ai {
					err = (AICmdContext{stdout: &out}).PrintJsonData(json.RawMessage(tt.data))
				} else {
					err = (CmdContext{stdout: &out}).PrintJsonData(json.RawMessage(tt.data))
				}
				if err != nil {
					t.Fatal(err)
				}
				var got, want any
				gotDecoder := json.NewDecoder(&out)
				gotDecoder.UseNumber()
				if err := gotDecoder.Decode(&got); err != nil {
					t.Fatal(err)
				}
				wantDecoder := json.NewDecoder(strings.NewReader(tt.want))
				wantDecoder.UseNumber()
				if err := wantDecoder.Decode(&want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}
			}
		})
	}
}
