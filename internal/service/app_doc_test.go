package service

import (
	"strings"
	"testing"
)

const sampleMD = `# 罐盖计数

传送带上用 XIAO ESP32S3 Sense 数瓶盖。适合产线巡检。

## 介绍

把摄像头对准瓶盖，打开应用后即可看到计数。

## 步骤：烧录固件

用 USB 连接 XIAO ESP32S3 Sense，然后烧录。

- 固件：` + "`firmware.bin`" + `
- 烧录时不要断开 USB

## 步骤：看计数

烧录完成后重新上电，屏幕上会显示当前数量。

## 硬件

- Seeed Studio XIAO ESP32S3 Sense × 1
`

func TestCompileAppMarkdownSample(t *testing.T) {
	got, err := CompileAppMarkdown(sampleMD)
	if err != nil {
		t.Fatal(err)
	}
	if got.SuggestedAppName != "罐盖计数" {
		t.Fatalf("name %q", got.SuggestedAppName)
	}
	if got.SuggestedExcerpt != "传送带上用 XIAO ESP32S3 Sense 数瓶盖。适合产线巡检。" {
		t.Fatalf("excerpt %q", got.SuggestedExcerpt)
	}
	want := []struct {
		id, typ, content, title string
	}{
		{"p1", "paragraph", "把摄像头对准瓶盖，打开应用后即可看到计数。", ""},
		{"s1", "step", "烧录固件", "烧录固件"},
		{"p2", "paragraph", "用 USB 连接 XIAO ESP32S3 Sense，然后烧录。", ""},
		{"b1", "bullet", "固件：<code>firmware.bin</code>", ""},
		{"b2", "bullet", "烧录时不要断开 USB", ""},
		{"s2", "step", "看计数", "看计数"},
		{"p3", "paragraph", "烧录完成后重新上电，屏幕上会显示当前数量。", ""},
		{"h1", "h2", "硬件", ""},
		{"b3", "bullet", "Seeed Studio XIAO ESP32S3 Sense × 1", ""},
	}
	if len(got.Deployment.Blocks) != len(want) {
		t.Fatalf("blocks %d: %#v", len(got.Deployment.Blocks), got.Deployment.Blocks)
	}
	for i, w := range want {
		b := got.Deployment.Blocks[i]
		if b["id"] != w.id || b["type"] != w.typ || b["content"] != w.content {
			t.Fatalf("block %d got id=%v type=%v content=%q", i, b["id"], b["type"], b["content"])
		}
		if w.title != "" && b["title"] != w.title {
			t.Fatalf("block %d title %v", i, b["title"])
		}
	}
}

func TestCompileInlineMarks(t *testing.T) {
	src := "# 标题\n\n段落\n\n" +
		"这是 **粗体** 和 *斜体* 和 ~~删除~~ 和 <u>下划线</u>。\n\n" +
		"看 [这里](https://example.com) 和 ![接线](https://cdn.example.com/a.png)。\n\n" +
		`<span style="color:#ef4444;background-color:#fee2e2">红</span> 和 <span style="color:#000000">丢色</span>` + "\n\n" +
		"<!-- align: right -->\n\n右对齐\n\n" +
		"## 居中 {align=center}\n\n" +
		"看 ![接线](./images/a.png) 和 [固件](./firmware.bin)。\n\n" +
		"a_b_c 保持下划线。\n\n" +
		"- [ ] 未做\n" +
		"- [x] 做好\n\n" +
		"> 引用一\n" +
		"> 引用二\n\n" +
		"```cpp\n" +
		"void setup() { if (a < b) {} }\n" +
		"```\n\n" +
		"---\n\n" +
		"### 1. 部署模型\n\n" +
		"## Step 2. Connect\n\n" +
		"## Step: Flash\n"
	got, err := CompileAppMarkdown(src)
	if err != nil {
		t.Fatal(err)
	}
	find := func(typ, contains string) map[string]any {
		t.Helper()
		for _, b := range got.Deployment.Blocks {
			if b["type"] == typ && strings.Contains(asString(b["content"]), contains) {
				return b
			}
		}
		t.Fatalf("missing %s containing %q in %#v", typ, contains, got.Deployment.Blocks)
		return nil
	}
	p := find("paragraph", "<strong>粗体</strong>")
	if !strings.Contains(asString(p["content"]), "<em>斜体</em>") ||
		!strings.Contains(asString(p["content"]), "<s>删除</s>") ||
		!strings.Contains(asString(p["content"]), "<u>下划线</u>") {
		t.Fatalf("inline %q", p["content"])
	}
	link := find("paragraph", `href="https://example.com"`)
	if !strings.Contains(asString(link["content"]), `target="_blank"`) ||
		!strings.Contains(asString(link["content"]), `rel="noopener noreferrer"`) ||
		!strings.Contains(asString(link["content"]), ">这里</a>") ||
		strings.Contains(asString(link["content"]), "doc-file-download-link") {
		t.Fatalf("link %q", link["content"])
	}
	if !strings.Contains(asString(link["content"]), `src="https://cdn.example.com/a.png"`) ||
		!strings.Contains(asString(link["content"]), `alt="接线"`) ||
		!strings.Contains(asString(link["content"]), `draggable="false"`) {
		t.Fatalf("image %q", link["content"])
	}
	color := find("paragraph", `color: #ef4444`)
	if !strings.Contains(asString(color["content"]), `background-color: #fee2e2`) ||
		strings.Contains(asString(color["content"]), "#000000") ||
		!strings.Contains(asString(color["content"]), "丢色") {
		t.Fatalf("color %q", color["content"])
	}
	right := find("paragraph", "右对齐")
	if right["align"] != "right" {
		t.Fatalf("align %v", right["align"])
	}
	center := find("h2", "居中")
	if center["align"] != "center" {
		t.Fatalf("heading align %v", center)
	}
	local := find("paragraph", "看 接线")
	if strings.Contains(asString(local["content"]), "./") {
		t.Fatalf("local path embedded %q", local["content"])
	}
	if !strings.Contains(asString(local["content"]), "固件") {
		t.Fatalf("file label dropped %q", local["content"])
	}
	if len(got.PendingUploads) != 2 || got.PendingUploads[0].Kind != "image" || got.PendingUploads[1].Kind != "file" {
		t.Fatalf("pending %#v", got.PendingUploads)
	}
	plain := find("paragraph", "a_b_c")
	if strings.Contains(asString(plain["content"]), "<em>") {
		t.Fatalf("snake case became italic %q", plain["content"])
	}
	todo := find("todo", "未做")
	if strings.HasPrefix(asString(todo["content"]), "[x]") || strings.Contains(asString(todo["content"]), "[ ]") {
		t.Fatalf("unchecked todo %q", todo["content"])
	}
	done := find("todo", "[x] 做好")
	if done == nil {
		t.Fatal("checked todo")
	}
	quote := find("quote", "引用一 引用二")
	if quote == nil {
		t.Fatal("quote")
	}
	code := find("code", "a < b")
	cfg, _ := code["config"].(map[string]any)
	if cfg["lang"] != "cpp" {
		t.Fatalf("lang %#v", code["config"])
	}
	div := find("divider", "")
	if div["content"] != "" {
		t.Fatalf("divider %#v", div)
	}
	for _, title := range []string{"部署模型", "Connect", "Flash"} {
		ok := false
		for _, b := range got.Deployment.Blocks {
			if b["type"] == "step" && b["title"] == title && b["content"] == title && b["expanded"] == true {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("missing step %s in %#v", title, got.Deployment.Blocks)
		}
	}
}

func TestCompileJoinedLinesAndVideo(t *testing.T) {
	got, err := CompileAppMarkdown("第一行\n第二行\n\n<video src=\"https://cdn.example.com/a.mp4\" controls></video>\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Deployment.Blocks) != 2 {
		t.Fatalf("%#v", got.Deployment.Blocks)
	}
	if got.Deployment.Blocks[0]["content"] != "第一行 第二行" {
		t.Fatalf("joined %q", got.Deployment.Blocks[0]["content"])
	}
	video := asString(got.Deployment.Blocks[1]["content"])
	if !strings.Contains(video, `src="https://cdn.example.com/a.mp4"`) || !strings.Contains(video, "controls") || strings.Contains(video, "iframe") {
		t.Fatalf("video %q", video)
	}
}

func TestCompileEditorVideoFileAndIndent(t *testing.T) {
	src := "# 标题\n\n简介\n\n" +
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ\n\n" +
		"https://youtu.be/dQw4w9WgXcQ\n\n" +
		"https://www.bilibili.com/video/BV1xx411c7mD\n\n" +
		"https://www.bilibili.com/video/av170001\n\n" +
		"https://www.douyin.com/jingxuan?modal_id=7123456789012345678\n\n" +
		"https://www.tiktok.com/@user/video/1234567890123456789\n\n" +
		"https://vimeo.com/123456789\n\n" +
		"https://v.youku.com/v_show/id_XMzI2.html\n\n" +
		"https://www.iqiyi.com/v_19rrabc.html\n\n" +
		"https://cdn.example.com/demo.mp4\n\n" +
		"<video src=\"./clip.mp4\" controls></video>\n\n" +
		"[固件](https://cdn.example.com/firmware.bin?dl=1)\n\n" +
		"[说明](https://cdn.example.com/note.pdf)\n\n" +
		"#### 四级\n\n" +
		"##### 五级\n\n" +
		"###### 六级\n\n" +
		"  - 缩进一项\n" +
		"        - 更深一项\n" +
		"![图](./cover.svg)\n"
	got, err := CompileAppMarkdown(src)
	if err != nil {
		t.Fatal(err)
	}
	find := func(contains string) string {
		t.Helper()
		for _, b := range got.Deployment.Blocks {
			content := asString(b["content"])
			if strings.Contains(content, contains) || asString(b["title"]) == contains {
				return content + "|" + asString(b["type"])
			}
		}
		t.Fatalf("missing %q in %#v", contains, got.Deployment.Blocks)
		return ""
	}
	yt := find("youtube.com/embed/dQw4w9WgXcQ")
	if !strings.Contains(yt, "video-embed-wrapper") || !strings.Contains(yt, "<iframe") {
		t.Fatalf("youtube %s", yt)
	}
	if !strings.Contains(find("player.bilibili.com/player.html?bvid=BV1xx411c7mD"), "iframe") {
		t.Fatal("bilibili bv")
	}
	if !strings.Contains(find("player.bilibili.com/player.html?aid=170001"), "iframe") {
		t.Fatal("bilibili av")
	}
	dy := find("open.douyin.com/player/video?vid=7123456789012345678")
	if !strings.Contains(dy, `referrerpolicy="unsafe-url"`) {
		t.Fatalf("douyin %s", dy)
	}
	if !strings.Contains(find("tiktok.com/embed/v2/1234567890123456789"), "iframe") {
		t.Fatal("tiktok")
	}
	if !strings.Contains(find("player.vimeo.com/video/123456789"), "iframe") {
		t.Fatal("vimeo")
	}
	if !strings.Contains(find("player.youku.com/embed/XMzI2.html"), "iframe") {
		t.Fatal("youku")
	}
	if !strings.Contains(find("open.iqiyi.com/developer/player_js/coopPlayerIndex.html?vid=19rrabc.html"), "iframe") {
		t.Fatal("iqiyi")
	}
	mp4 := find("cdn.example.com/demo.mp4")
	if !strings.Contains(mp4, "<video") || !strings.Contains(mp4, "controls") || strings.Contains(mp4, "iframe") {
		t.Fatalf("mp4 %s", mp4)
	}
	file := find("firmware.bin")
	if !strings.Contains(file, `class="doc-file-download-link"`) || !strings.Contains(file, `data-file-download="1"`) || !strings.Contains(file, " download") {
		t.Fatalf("file %s", file)
	}
	if !strings.Contains(find("note.pdf"), "doc-file-download-link") {
		t.Fatal("pdf")
	}
	for _, level := range []string{"h4", "h5", "h6"} {
		ok := false
		for _, b := range got.Deployment.Blocks {
			if b["type"] == level {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("missing %s", level)
		}
	}
	indented := false
	for _, b := range got.Deployment.Blocks {
		if b["type"] == "bullet" && strings.Contains(asString(b["content"]), "缩进一项") {
			if b["indent"] != 1 {
				t.Fatalf("indent %v", b["indent"])
			}
			indented = true
		}
	}
	if !indented {
		t.Fatal("indent bullet")
	}
	capped := false
	for _, b := range got.Deployment.Blocks {
		if b["type"] == "bullet" && strings.Contains(asString(b["content"]), "更深一项") {
			if b["indent"] != 2 {
				t.Fatalf("deep indent %v", b["indent"])
			}
			capped = true
		}
	}
	if !capped {
		t.Fatal("deep indent bullet")
	}
	if len(got.PendingUploads) != 2 {
		t.Fatalf("pending %#v", got.PendingUploads)
	}
	sawVideo, sawSVG := false, false
	for _, p := range got.PendingUploads {
		if p.Kind == "video" && p.Path == "./clip.mp4" {
			sawVideo = true
		}
		if p.Kind == "file" && p.Path == "./cover.svg" {
			sawSVG = true
		}
	}
	if !sawVideo || !sawSVG {
		t.Fatalf("pending %#v", got.PendingUploads)
	}
	for _, b := range got.Deployment.Blocks {
		if strings.Contains(asString(b["content"]), "clip.mp4") {
			t.Fatalf("local video embedded %q", b["content"])
		}
	}
}

func TestValidateDeploymentBlocksRejectsMarkdown(t *testing.T) {
	err := ValidateDeploymentBlocks(map[string]any{
		"deployment": map[string]any{
			"blocks": []any{
				map[string]any{"type": "paragraph", "content": "这是 **粗体**"},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "app doc compile") {
		t.Fatalf("got %v", err)
	}
	if err := ValidateDeploymentBlocks(map[string]any{
		"deployment": map[string]any{
			"blocks": []any{
				map[string]any{"type": "paragraph", "content": "这是 <strong>粗体</strong>"},
				map[string]any{"type": "code", "content": "let x = **y**\n# include"},
				map[string]any{"type": "h2", "content": "# 标题"},
			},
		},
	}); err == nil || !strings.Contains(err.Error(), "h2") {
		t.Fatalf("heading markdown should fail, got %v", err)
	}
	if err := ValidateDeploymentBlocks(map[string]any{
		"deployment": map[string]any{
			"blocks": []any{
				map[string]any{"type": "paragraph", "content": "这是 <strong>粗体</strong>"},
				map[string]any{"type": "code", "content": "**not markdown**\n# include <stdio.h>"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDeploymentBlocks(map[string]any{"app_name": "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeDeploymentBlocksFillsEditorFormat(t *testing.T) {
	body := map[string]any{
		"deployment": map[string]any{
			"blocks": []any{
				map[string]any{"type": "paragraph", "content": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"},
				map[string]any{"type": "paragraph", "content": `<a href="https://cdn.example.com/note.pdf">说明文档</a>`},
				map[string]any{"type": "paragraph", "content": `<img src="https://cdn.example.com/a.png" alt="接线">`},
				map[string]any{"type": "paragraph", "content": "看这个 https://www.youtube.com/watch?v=dQw4w9WgXcQ 先不要变播放器"},
				map[string]any{"type": "paragraph", "content": `<video src="https://cdn.example.com/demo.mp4" controls></video>`},
				map[string]any{"type": "code", "content": "**keep**"},
				map[string]any{"type": "paragraph", "content": `<a href="https://seeed.cc/x" target="_blank" rel="noopener noreferrer" style="` + editorLinkStyle + `">Seeed</a>`},
			},
		},
	}
	NormalizeDeploymentBlocks(body)
	blocks, _, err := deploymentBlockList(body)
	if err != nil {
		t.Fatal(err)
	}
	got := func(i int) string { return asString(blocks[i].(map[string]any)["content"]) }
	if !strings.Contains(got(0), "youtube.com/embed/dQw4w9WgXcQ") || !strings.Contains(got(0), "video-embed-wrapper") {
		t.Fatalf("video paragraph: %s", got(0))
	}
	if !strings.Contains(got(1), "doc-file-download-link") || !strings.Contains(got(1), " download") {
		t.Fatalf("file link: %s", got(1))
	}
	if !strings.Contains(got(2), `draggable="false"`) || !strings.Contains(got(2), "接线") {
		t.Fatalf("image: %s", got(2))
	}
	if strings.Contains(got(3), "video-embed-wrapper") || !strings.Contains(got(3), "watch?v=dQw4w9WgXcQ") {
		t.Fatalf("in-sentence url changed: %s", got(3))
	}
	if !strings.Contains(got(4), "demo.mp4") || !strings.Contains(got(4), "controls") || strings.Contains(got(4), "iframe") {
		t.Fatalf("direct video: %s", got(4))
	}
	if got(5) != "**keep**" {
		t.Fatalf("code block rewritten: %s", got(5))
	}
	kept := `<a href="https://seeed.cc/x" target="_blank" rel="noopener noreferrer" style="` + editorLinkStyle + `">Seeed</a>`
	if got(6) != kept {
		t.Fatalf("complete link rewritten: %s", got(6))
	}
	if err := ValidateDeploymentBlocks(body); err != nil {
		t.Fatal(err)
	}
	again := []string{got(0), got(1), got(2), got(3), got(4), got(5), got(6)}
	NormalizeDeploymentBlocks(body)
	for i, want := range again {
		if got(i) != want {
			t.Fatalf("second pass changed block %d\n%s\n%s", i, want, got(i))
		}
	}

	raw := map[string]any{
		"deployment": map[string]any{
			"blocks": []any{map[string]any{"type": "paragraph", "content": "这是 **粗体**"}},
		},
	}
	NormalizeDeploymentBlocks(raw)
	if err := ValidateDeploymentBlocks(raw); err == nil || !strings.Contains(err.Error(), "app doc compile") {
		t.Fatalf("markdown should still be rejected, got %v", err)
	}
}

func TestApplyCompiledMarkdownFillsNameAndAppendsToolBlock(t *testing.T) {
	notes, err := CompileAppMarkdown("# QA Name\n\n这是摘要\n\n**粗体**\n\n![图](./a.png)\n")
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"scene": "1",
		"deployment": map[string]any{
			"blocks": []any{map[string]any{"id": "t1", "type": "flash-firmware"}},
		},
	}
	applyCompiledMarkdown(body, &notes)
	if body["app_name"] != "QA Name" || body["excerpt"] != "这是摘要" {
		t.Fatalf("name/excerpt %#v %#v", body["app_name"], body["excerpt"])
	}
	blocks, _, err := deploymentBlockList(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) < 2 {
		t.Fatalf("blocks %#v", blocks)
	}
	if !strings.Contains(asString(blocks[0].(map[string]any)["content"]), "<strong>粗体</strong>") {
		t.Fatalf("compiled block %#v", blocks[0])
	}
	if blocks[len(blocks)-1].(map[string]any)["type"] != "flash-firmware" {
		t.Fatalf("tool block not appended: %#v", blocks)
	}
	if len(notes.PendingUploads) != 1 || notes.PendingUploads[0].Path != "./a.png" {
		t.Fatalf("pending %#v", notes.PendingUploads)
	}
	long, err := CompileAppMarkdown("# " + strings.Repeat("名", 80) + "\n\n" + strings.Repeat("摘", 400) + "\n")
	if err != nil {
		t.Fatal(err)
	}
	trimmed := map[string]any{}
	applyCompiledMarkdown(trimmed, &long)
	if utf8len(asString(trimmed["app_name"])) != maxAppNameRunes || utf8len(asString(trimmed["excerpt"])) != maxAppExcerptRunes {
		t.Fatalf("truncation %#v", trimmed)
	}
	joined := strings.Join(long.Warnings, "\n")
	if !strings.Contains(joined, "suggested_app_name truncated") || !strings.Contains(joined, "suggested_excerpt is longer") {
		t.Fatalf("warnings %#v", long.Warnings)
	}
}

func utf8len(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
