package output

import (
	"encoding/json"
	"fmt"
	"os"
)

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func PrintRaw(raw json.RawMessage) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		_, err = fmt.Fprintln(os.Stdout, string(raw))
		return err
	}
	return PrintJSON(v)
}
