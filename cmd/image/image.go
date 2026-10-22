package image

import (
	"os"
	"strings"

	"sensecraft-cloud-terminal/internal/output"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

// NewImageCmd returns a prompt for the calling agent to write an SVG.
func NewImageCmd() *cobra.Command {
	var brief string
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Return a prompt so the agent can write an SVG",
		Long: `Return a prompt to the calling agent.

The agent uses that prompt to write an SVG, render it to PNG locally, and show it to the user.
This command does not draw or upload.
The local image tool comes from SENSECRAFT_IMAGE_CMD; when that is empty, use SVG.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := service.BuildImagePrompt(brief, os.Getenv("SENSECRAFT_IMAGE_CMD"))
			prompt.LocalTool = strings.TrimSpace(prompt.LocalTool)
			return output.PrintJSON(prompt)
		},
	}
	cmd.Flags().StringVar(&brief, "brief", "", "Picture brief; when empty, the prompt tells the agent to ask the user first")
	return cmd
}
