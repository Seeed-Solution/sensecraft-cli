package service

import (
	"fmt"
	"strings"
)

// ImagePrompt is the prompt handed back to the calling agent. This command does not draw.
type ImagePrompt struct {
	Prompt    string `json:"prompt"`
	Brief     string `json:"brief,omitempty"`
	LocalTool string `json:"local_tool,omitempty"`
}

// BuildImagePrompt builds the agent prompt from the brief and the local image tool.
func BuildImagePrompt(brief, localTool string) ImagePrompt {
	brief = strings.TrimSpace(brief)
	localTool = strings.TrimSpace(localTool)
	var b strings.Builder
	b.WriteString("You are preparing an image for SenseCraft. Follow the steps in order. Do not hand in a card that is only text.\n\n")
	if brief == "" {
		b.WriteString("The user has not given a brief. Ask for the brief, then run this command again with --brief. Do not invent the picture.\n")
	} else {
		fmt.Fprintf(&b, "Brief: %s\n", brief)
	}
	b.WriteString("\n")
	if localTool != "" {
		fmt.Fprintf(&b, "Local image tool: %s\nUse it first to produce a PNG from the brief. If it fails, use the code path below.\n", localTool)
	} else {
		b.WriteString("No local image tool is configured (SENSECRAFT_IMAGE_CMD is empty). Use code.\n")
	}
	b.WriteString(`
The code is an SVG, not pixels, and not a reflow of the brief.
Draw the things in the brief with rectangles, circles, ellipses, paths, and <text>. Paint the background first, then each object, and mark what is in front.
The picture must contain the objects described in the brief, not only a title.
Any sentence printed on the image must match the brief exactly and go in <text>.
Aspect ratio 4:3. Pixel size 1200×900. Do not exceed 1200×900.
Write the SVG only to sensecraft-image/cover.svg.
Do not call a cloud image model.

After the SVG is written, convert it on this machine to PNG, or to JPG, GIF, or WebP. Write sensecraft-image/cover.png when the result is PNG. The user reviews that image; keep the SVG for the next edit. Uploads accept JPG, PNG, GIF, and WebP, not SVG.

Show the image to the user and ask whether it meets the requirements (4:3, 1200×900, and the brief). Do not upload before the user chooses:
1. Upload this image
2. Generate again
3. Use the user's own image file
`)
	return ImagePrompt{Prompt: b.String(), Brief: brief, LocalTool: localTool}
}
