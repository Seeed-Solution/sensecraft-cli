package app

import (
	"encoding/json"
	"testing"

	"sensecraft-cloud-terminal/internal/cmdtest"

	"github.com/spf13/cobra"
)

func TestAppListOutputCount(t *testing.T) {
	cmdtest.SetupConfig(t, "https://unused.invalid")
	for _, args := range [][]string{
		{"app", "list", "--mine", "--sandbox"},
		{"app", "activity", "list", "--sandbox"},
		{"app", "bom", "search", "--name", "ESP32-S3 Sense", "--sandbox"},
	} {
		t.Run(args[1], func(t *testing.T) {
			root := &cobra.Command{Use: "test"}
			root.AddCommand(NewAppCmd())
			stdout, err := cmdtest.Execute(t, root, args)
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Count *int             `json:"count"`
				List  []map[string]any `json:"list"`
				Page  json.RawMessage  `json:"page"`
			}
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatal(err)
			}
			if result.Count == nil || *result.Count != len(result.List) {
				t.Fatal("list count is missing or incorrect")
			}
			if args[1] == "list" && len(result.Page) == 0 {
				t.Fatal("app pagination metadata was dropped")
			}
			if args[1] == "bom" && len(result.List) != 1 {
				t.Fatal("normalized BOM query did not match the sandbox product")
			}
		})
	}
}
