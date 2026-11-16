package image

import (
	"os"
	"strings"

	"sensecraft-cloud-terminal/internal/output"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

// NewImageCmd 把一段提示词交回调用它的 Agent，由 Agent 的模型写 SVG。
func NewImageCmd() *cobra.Command {
	var brief string
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Return a prompt so the agent can write an SVG",
		Long: `给调用方 Agent 返回一段提示词。

Agent 用这段提示词让自己的模型写 SVG，再在本机渲染成 PNG，交给用户过目。
本命令不画图，也不上传。
本机生图工具从环境变量 SENSECRAFT_IMAGE_CMD 读取；没有就走 SVG。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := service.BuildImagePrompt(brief, os.Getenv("SENSECRAFT_IMAGE_CMD"))
			prompt.LocalTool = strings.TrimSpace(prompt.LocalTool)
			return output.PrintJSON(prompt)
		},
	}
	cmd.Flags().StringVar(&brief, "brief", "", "画面文案；空着时提示词会要求 Agent 先向用户询问")
	return cmd
}
