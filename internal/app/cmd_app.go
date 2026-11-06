package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"sensecraft-cloud-terminal/internal/config"
)

type CmdContext struct {
	Ctx     context.Context
	Config  config.AccessApiConfig
	SandBox bool
	stdout  io.Writer
	stderr  io.Writer
}

// AICmdContext carries the Access Key used for SenseCraft AI /openapi.
type AICmdContext struct {
	Ctx     context.Context
	Access  config.AccessApiConfig
	SandBox bool
	stdout  io.Writer
	stderr  io.Writer
}

func GetCmdContext() (CmdContext, error) {
	cliConfig, err := config.LoadConfig()
	if err != nil {
		return CmdContext{}, err
	}

	apiCfg, err := config.ResolveActiveAPIConfig(cliConfig)
	if err != nil {
		return CmdContext{}, err
	}

	return CmdContext{
		Ctx:    context.Background(),
		Config: apiCfg,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}, nil
}

func GetAICmdContext() (AICmdContext, error) {
	cliConfig, err := config.LoadConfig()
	if err != nil {
		return AICmdContext{}, err
	}
	access, err := config.ResolveActiveAPIConfig(cliConfig)
	if err != nil {
		return AICmdContext{}, err
	}
	return AICmdContext{
		Ctx:    context.Background(),
		Access: access,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}, nil
}

// RemoteType is the active site for AI host resolution.
func (ctx AICmdContext) RemoteType() string {
	return ctx.Access.RemoteType
}

func (ctx CmdContext) LogInfo(fields map[string]string) {
	fmt.Fprintln(ctx.stderr, fields)
	_ = fields
}

func (ctx CmdContext) LogError(err error, fields map[string]string) {
	fmt.Fprintln(ctx.stderr, err)
	_ = fields
}

func (ctx CmdContext) PrintJsonData(data interface{}) error {
	return printJSONData(ctx.stdout, data)
}

func (ctx CmdContext) PrintRawData(raw json.RawMessage) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		_, err = fmt.Fprintln(ctx.stdout, string(raw))
		return err
	}
	return ctx.PrintJsonData(v)
}

func (ctx AICmdContext) LogError(err error, fields map[string]string) {
	fmt.Fprintln(ctx.stderr, err)
	_ = fields
}

func (ctx AICmdContext) PrintJsonData(data interface{}) error {
	return printJSONData(ctx.stdout, data)
}

// printJSONData adds the current list length when the response has no count.
func printJSONData(out io.Writer, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) == nil && root != nil {
		if list, exists := root["list"]; exists {
			var rows []json.RawMessage
			if _, counted := root["count"]; !counted && json.Unmarshal(list, &rows) == nil {
				root["count"], _ = json.Marshal(len(rows))
				raw, err = json.Marshal(root)
				if err != nil {
					return err
				}
			}
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(json.RawMessage(raw))
}

func (ctx AICmdContext) PrintRawData(raw json.RawMessage) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		_, err = fmt.Fprintln(ctx.stdout, string(raw))
		return err
	}
	return ctx.PrintJsonData(v)
}
