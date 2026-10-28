package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CompileAppMarkdown turns a Markdown document into the block array the app
// editor stores in deployment.blocks. It does not upload files or insert
// tool-blocks. Rules follow the editor paste parser and docs/md-to-app.md.
func CompileAppMarkdown(src string) (AppDocCompileResult, error) {
	src = strings.ReplaceAll(strings.TrimPrefix(src, "\uFEFF"), "\r\n", "\n")
	if strings.TrimSpace(src) == "" {
		return AppDocCompileResult{}, fmt.Errorf("markdown is empty")
	}
	c := &docCompiler{
		pend:     []AppDocPendingUpload{},
		warnings: []string{},
		ids:      map[string]int{},
	}
	c.parse(src)
	return AppDocCompileResult{
		Deployment:       AppDocDeployment{Blocks: c.blocks},
		SuggestedAppName: c.appName,
		SuggestedExcerpt: c.excerpt,
		PendingUploads:   c.pend,
		Warnings:         c.warnings,
	}, nil
}

// AppDocCompileResult is the stdout of `app doc compile`.
type AppDocCompileResult struct {
	Deployment       AppDocDeployment      `json:"deployment"`
	SuggestedAppName string                `json:"suggested_app_name,omitempty"`
	SuggestedExcerpt string                `json:"suggested_excerpt,omitempty"`
	PendingUploads   []AppDocPendingUpload `json:"pending_uploads"`
	Warnings         []string              `json:"warnings"`
}

// AppDocDeployment is the object merged into an app create/update payload.
type AppDocDeployment struct {
	Blocks []map[string]any `json:"blocks"`
}

// AppDocPendingUpload is a local path the compiler refused to embed.
// Upload it with `file upload`, then put the https file_url into the block.
type AppDocPendingUpload struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	BlockID string `json:"block_id,omitempty"`
	Alt     string `json:"alt,omitempty"`
}

type docCompiler struct {
	blocks   []map[string]any
	pend     []AppDocPendingUpload
	warnings []string
	ids      map[string]int
	appName  string
	excerpt  string
	// liftExcerpt is set after the opening H1 is taken as the app name.
	// The next paragraph becomes suggested_excerpt and is not a block.
	liftExcerpt bool
	emitted     bool
	tableWarned bool
}

func (c *docCompiler) parse(src string) {
	lines := strings.Split(src, "\n")
	var para []string
	var fence []string
	inFence := false
	fenceLang := ""
	pendingAlign := ""

	flushParagraph := func() {
		if len(para) == 0 {
			return
		}
		joined := strings.TrimSpace(strings.Join(para, " "))
		para = para[:0]
		if joined == "" {
			return
		}
		c.emitParagraph(joined, pendingAlign)
		pendingAlign = ""
	}

	for i := 0; i < len(lines); {
		raw := lines[i]
		trimmed := strings.TrimSpace(strings.TrimRight(raw, " \t"))

		if inFence {
			if strings.HasPrefix(trimmed, "```") {
				inFence = false
				c.emitCode(strings.Join(fence, "\n"), fenceLang, pendingAlign)
				pendingAlign = ""
				fence = fence[:0]
				fenceLang = ""
				i++
				continue
			}
			fence = append(fence, raw)
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			flushParagraph()
			inFence = true
			fenceLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			i++
			continue
		}

		if trimmed == "" {
			flushParagraph()
			i++
			continue
		}

		if align, ok := parseAlignComment(trimmed); ok {
			flushParagraph()
			pendingAlign = align
			i++
			continue
		}

		if isTableRow(trimmed) && !c.tableWarned {
			c.tableWarned = true
			c.warnings = append(c.warnings, "editor has no table block; a table row was left as a paragraph. Split it into bullets or a code block")
		}

		if level, text, ok := parseATX(trimmed); ok {
			flushParagraph()
			text, align := splitAlignSuffix(text)
			if align == "" {
				align = pendingAlign
			}
			pendingAlign = ""
			c.emitHeading(level, text, align)
			i++
			continue
		}

		if isDivider(trimmed) {
			flushParagraph()
			c.emitDivider(pendingAlign)
			pendingAlign = ""
			i++
			continue
		}

		if checked, body, ok := parseTodo(trimmed); ok {
			flushParagraph()
			c.emitTodo(checked, body, leadingIndent(raw), pendingAlign)
			pendingAlign = ""
			i++
			continue
		}

		if _, ok := parseBullet(trimmed); ok {
			flushParagraph()
			for i < len(lines) {
				ln := lines[i]
				tr := strings.TrimSpace(ln)
				if tr == "" {
					break
				}
				if _, _, isTodo := parseTodo(tr); isTodo {
					break
				}
				item, isBullet := parseBullet(tr)
				if !isBullet {
					break
				}
				c.emitList("bullet", "b", item, leadingIndent(ln), pendingAlign)
				pendingAlign = ""
				i++
			}
			continue
		}

		if _, ok := parseNumber(trimmed); ok {
			flushParagraph()
			for i < len(lines) {
				ln := lines[i]
				tr := strings.TrimSpace(ln)
				if tr == "" {
					break
				}
				item, isNum := parseNumber(tr)
				if !isNum {
					break
				}
				c.emitList("number", "n", item, leadingIndent(ln), pendingAlign)
				pendingAlign = ""
				i++
			}
			continue
		}

		if strings.HasPrefix(trimmed, ">") {
			flushParagraph()
			var q []string
			for i < len(lines) {
				tr := strings.TrimSpace(lines[i])
				if tr == "" || !strings.HasPrefix(tr, ">") {
					break
				}
				q = append(q, strings.TrimSpace(strings.TrimPrefix(tr, ">")))
				i++
			}
			c.emitQuote(strings.Join(q, " "), pendingAlign)
			pendingAlign = ""
			continue
		}

		para = append(para, raw)
		i++
	}

	if inFence {
		c.emitCode(strings.Join(fence, "\n"), fenceLang, pendingAlign)
		pendingAlign = ""
	}
	flushParagraph()
	if pendingAlign != "" {
		c.warnings = append(c.warnings, "align "+pendingAlign+" was not applied; no block followed it")
	}
	if c.blocks == nil {
		c.blocks = []map[string]any{}
	}
}

func (c *docCompiler) emitParagraph(text, align string) {
	if html, ok := editorVideoHTML(text); ok {
		if c.liftExcerpt && !c.emitted {
			c.liftExcerpt = false
			c.warnings = append(c.warnings, "opening paragraph is a video; kept as a block instead of suggested_excerpt")
		}
		c.addText("paragraph", "p", html, 0, align, nil)
		return
	}
	if c.liftExcerpt && !c.emitted {
		html := renderInline(text, "", &c.pend)
		c.excerpt = strings.TrimSpace(htmlToPlain(html))
		c.liftExcerpt = false
		if utf8.RuneCountInString(c.excerpt) > maxAppExcerptRunes {
			c.warnings = append(c.warnings, fmt.Sprintf("suggested_excerpt is longer than %d characters; app create --from-md and app update --from-md keep the first %d", maxAppExcerptRunes, maxAppExcerptRunes))
		}
		return
	}
	id := c.nextID("p")
	html := renderInline(text, id, &c.pend)
	if strings.TrimSpace(stripTags(html)) == "" && !strings.Contains(html, "<img") && !strings.Contains(html, "<video") && !strings.Contains(html, "<iframe") && !strings.Contains(html, "<a ") {
		clearPendingBlock(c.pend, id)
		return
	}
	c.addTextWithID(id, "paragraph", html, 0, align, nil)
}

func (c *docCompiler) emitHeading(level int, text, align string) {
	plain := strings.TrimSpace(htmlToPlain(renderInline(text, "", nil)))
	if title, ok := stepTitle(plain); ok {
		c.liftExcerpt = false
		if title == "" {
			c.warnings = append(c.warnings, "skipped an empty step heading")
			return
		}
		c.addStep(title)
		return
	}
	if isIntroHeading(plain) {
		c.liftExcerpt = false
		return
	}
	if level == 1 && !c.emitted && c.appName == "" {
		c.appName = plain
		c.liftExcerpt = true
		return
	}
	c.liftExcerpt = false
	id := c.nextID("h")
	c.addTextWithID(id, "h"+fmt.Sprint(level), renderInline(text, id, &c.pend), 0, align, nil)
}

func (c *docCompiler) emitList(typ, prefix, item string, indent int, align string) {
	c.liftExcerpt = false
	id := c.nextID(prefix)
	c.addTextWithID(id, typ, renderInline(item, id, &c.pend), indent, align, nil)
}

func (c *docCompiler) emitTodo(checked bool, body string, indent int, align string) {
	c.liftExcerpt = false
	id := c.nextID("td")
	inner := renderInline(body, id, &c.pend)
	if checked {
		inner = "[x] " + inner
	}
	c.addTextWithID(id, "todo", inner, indent, align, nil)
}

func (c *docCompiler) emitQuote(text, align string) {
	c.liftExcerpt = false
	id := c.nextID("q")
	c.addTextWithID(id, "quote", renderInline(text, id, &c.pend), 0, align, nil)
}

func (c *docCompiler) emitCode(content, lang, align string) {
	c.liftExcerpt = false
	if strings.TrimSpace(content) == "" && lang == "" {
		return
	}
	var cfg map[string]any
	if lang = sanitizeLang(lang); lang != "" {
		cfg = map[string]any{"lang": lang}
	}
	c.addText("code", "c", content, 0, align, cfg)
}

func (c *docCompiler) emitDivider(align string) {
	c.liftExcerpt = false
	c.addText("divider", "dv", "", 0, align, nil)
}

func (c *docCompiler) addStep(title string) {
	id := c.nextID("s")
	c.blocks = append(c.blocks, map[string]any{
		"id":       id,
		"type":     "step",
		"title":    title,
		"content":  title,
		"expanded": true,
		"indent":   0,
	})
	c.emitted = true
}

func (c *docCompiler) addText(typ, prefix, content string, indent int, align string, cfg map[string]any) {
	c.addTextWithID(c.nextID(prefix), typ, content, indent, align, cfg)
}

func (c *docCompiler) addTextWithID(id, typ, content string, indent int, align string, cfg map[string]any) {
	if !alignApplies(typ) {
		if align != "" && align != "left" {
			c.warnings = append(c.warnings, "align "+align+" ignored on "+typ+" "+id+"; alignment applies to paragraph and h1-h6")
		}
		align = ""
	}
	block := map[string]any{
		"id":      id,
		"type":    typ,
		"content": content,
		"indent":  indent,
	}
	if align == "center" || align == "right" {
		block["align"] = align
	}
	if cfg != nil {
		block["config"] = cfg
	}
	c.blocks = append(c.blocks, block)
	c.emitted = true
}

func (c *docCompiler) nextID(prefix string) string {
	c.ids[prefix]++
	return fmt.Sprintf("%s%d", prefix, c.ids[prefix])
}

func clearPendingBlock(pend []AppDocPendingUpload, id string) {
	for i := range pend {
		if pend[i].BlockID == id {
			pend[i].BlockID = ""
		}
	}
}

func alignApplies(typ string) bool {
	switch typ {
	case "paragraph", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func isIntroHeading(plain string) bool {
	switch strings.ToLower(strings.TrimSpace(plain)) {
	case "介绍", "introduction":
		return true
	default:
		return false
	}
}

var (
	stepCN  = regexp.MustCompile(`^步骤\s*\d*\s*[:：]\s*(.*)$`)
	stepEN  = regexp.MustCompile(`(?i)^step\s*(?:\d+)?\s*[:：.]\s*(.*)$`)
	stepNth = regexp.MustCompile(`^第\s*\d+\s*步\s*[:：.]?\s*(.*)$`)
	stepNum = regexp.MustCompile(`^\d+\.\s+(.+)$`)
)

func stepTitle(plain string) (string, bool) {
	plain = strings.TrimSpace(plain)
	for _, re := range []*regexp.Regexp{stepCN, stepEN, stepNth} {
		if m := re.FindStringSubmatch(plain); m != nil {
			return strings.TrimSpace(m[1]), true
		}
	}
	if m := stepNum.FindStringSubmatch(plain); m != nil {
		return strings.TrimSpace(m[1]), true
	}
	return "", false
}

func parseATX(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(line) || line[i] != ' ' {
		return 0, "", false
	}
	return i, strings.TrimSpace(line[i+1:]), true
}

func parseAlignComment(line string) (string, bool) {
	re := regexp.MustCompile(`(?i)^<!--\s*align:\s*(left|center|right)\s*-->$`)
	m := re.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return strings.ToLower(m[1]), true
}

var alignSuffix = regexp.MustCompile(`(?i)\s*\{align=(left|center|right)\}\s*$`)

func splitAlignSuffix(text string) (string, string) {
	m := alignSuffix.FindStringSubmatch(text)
	if m == nil {
		return text, ""
	}
	return strings.TrimSpace(alignSuffix.ReplaceAllString(text, "")), strings.ToLower(m[1])
}

func isDivider(line string) bool {
	return regexp.MustCompile(`^(\*{3,}|-{3,}|_{3,})$`).MatchString(line)
}

func isTableRow(line string) bool {
	return strings.HasPrefix(line, "|") && strings.Contains(line[1:], "|")
}

func parseTodo(line string) (checked bool, body string, ok bool) {
	re := regexp.MustCompile(`^[-*+]\s+\[([ xX])\]\s+(.*)$`)
	m := re.FindStringSubmatch(line)
	if m == nil {
		return false, "", false
	}
	return strings.EqualFold(m[1], "x"), strings.TrimSpace(m[2]), true
}

func parseBullet(line string) (string, bool) {
	re := regexp.MustCompile(`^[-*+]\s+(.*)$`)
	m := re.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

func parseNumber(line string) (string, bool) {
	re := regexp.MustCompile(`^\d+\.\s+(.*)$`)
	m := re.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

func leadingIndent(raw string) int {
	n := 0
	for _, r := range raw {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 2
			continue
		}
		break
	}
	indent := n / 2
	// Editor toolbar stops at 2 (MarkdownBlockEditor MAX_BLOCK_INDENT_LEVEL).
	if indent > 2 {
		return 2
	}
	return indent
}

func sanitizeLang(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" || strings.ContainsAny(lang, " \t") {
		return ""
	}
	for _, r := range lang {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("+#.-_", r) {
			return ""
		}
	}
	return lang
}

const (
	editorImgStyle    = "max-width: 100%; width: 100%; height: auto; display: block; margin: 8px 0; border-radius: 4px;"
	editorVidStyle    = "max-width: 100%; height: auto; display: block; border-radius: 4px;"
	editorLinkStyle   = "color: #409eff; text-decoration: none; border-bottom: 1px solid #409eff;"
	editorIframeAllow = "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
)

var (
	videoTagRe    = regexp.MustCompile(`(?i)^<video\s+([^>]*)>\s*</video>`)
	directVideoRe = regexp.MustCompile(`(?i)\.(mp4|webm|ogg|mov|avi|wmv|flv|m3u8)(\?.*)?$`)
	fileLinkExtRe = regexp.MustCompile(`(?i)\.(pdf|doc|docx|xls|xlsx|ppt|pptx|zip|rar|7z|bin|hex|elf|img|tflite|onnx|engine|pt|cvimodel|json|csv|txt)(?:[?#].*)?$`)
	ytShortsRe    = regexp.MustCompile(`(?i)youtube\.com/shorts/([a-zA-Z0-9_-]{11})`)
	ytWatchRe     = regexp.MustCompile(`(?i)(?:youtube\.com/(?:[^/]+/.+/|(?:v|e(?:mbed)?)/|.*[?&]v=)|youtu\.be/)([^"&?/\s]{11})`)
	ytEmbedRe     = regexp.MustCompile(`(?i)youtube\.com/embed/([^"&?/\s]{11})`)
	tiktokUserRe  = regexp.MustCompile(`(?i)tiktok\.com/@[^/]+/video/(\d+)`)
	tiktokShortRe = regexp.MustCompile(`(?i)tiktok\.com/v/(\d+)`)
	douyinModalRe = regexp.MustCompile(`(?i)[?&]modal_id=(\d+)`)
	douyinVideoRe = regexp.MustCompile(`(?i)douyin\.com/video/([^"&?/\s]+)`)
	douyinShareRe = regexp.MustCompile(`(?i)iesdouyin\.com/share/video/([^"&?/\s]+)`)
	biliBVRe      = regexp.MustCompile(`(?i)bilibili\.com/video/(BV[^"&?/\s]+)`)
	biliAVRe      = regexp.MustCompile(`(?i)bilibili\.com/video/(av[^"&?/\s]+)`)
	b23Re         = regexp.MustCompile(`(?i)b23\.tv/([^"&?/\s]+)`)
	vimeoRe       = regexp.MustCompile(`(?i)vimeo\.com/(\d+)`)
	vimeoEmbedRe  = regexp.MustCompile(`(?i)vimeo\.com/embed/(\d+)`)
	youkuRe       = regexp.MustCompile(`(?i)youku\.com/v_show/id_([^"&?/\s=]+)`)
	iqiyiRe       = regexp.MustCompile(`(?i)iqiyi\.com/v_([^"&?/\s]+)`)
)

// editorVideoHTML matches the app editor Insert Video button.
// A paragraph that is only a remote video URL, or a <video src> tag, becomes
// the HTML sanitizeInlineHtml keeps: a styled <video> for a file, or a
// video-embed-wrapper iframe for YouTube, TikTok, Douyin, Bilibili, Vimeo,
// Youku, and iQiyi. Local files return false so the caller can queue an upload.
func editorVideoHTML(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if src, n, ok := matchVideoTag(text); ok && n == len(text) {
		if !isRemoteURL(src) {
			return "", false
		}
		return buildEditorVideoHTML(src), true
	}
	if isRemoteURL(text) && isEditorVideoURL(text) {
		return buildEditorVideoHTML(text), true
	}
	return "", false
}

func matchVideoTag(src string) (string, int, bool) {
	m := videoTagRe.FindStringSubmatch(src)
	if m == nil {
		return "", 0, false
	}
	return attrValue(m[1], "src"), len(m[0]), true
}

func isEditorVideoURL(raw string) bool {
	raw = normalizeVideoURL(raw)
	if directVideoRe.MatchString(raw) {
		return true
	}
	_, _, ok := platformEmbed(raw)
	return ok
}

func buildEditorVideoHTML(raw string) string {
	src := normalizeVideoURL(raw)
	if embed, douyin, ok := platformEmbed(src); ok {
		return editorIframe(embed, douyin)
	}
	return `<video src="` + escapeAttr(src) + `" controls style="` + editorVidStyle + `"></video>`
}

func normalizeVideoURL(raw string) string {
	raw = strings.TrimSpace(raw)
	for strings.Contains(raw, "&amp;") {
		raw = strings.ReplaceAll(raw, "&amp;", "&")
	}
	return raw
}

func platformEmbed(raw string) (embed string, douyin bool, ok bool) {
	if directVideoRe.MatchString(raw) {
		return "", false, false
	}
	if id := firstSubmatch(raw, ytShortsRe, ytWatchRe, ytEmbedRe); id != "" {
		return "https://www.youtube.com/embed/" + id, false, true
	}
	if id := firstSubmatch(raw, tiktokUserRe, tiktokShortRe); id != "" {
		return "https://www.tiktok.com/embed/v2/" + id, false, true
	}
	if id := firstSubmatch(raw, douyinModalRe, douyinVideoRe, douyinShareRe); id != "" {
		return "https://open.douyin.com/player/video?vid=" + url.QueryEscape(id) + "&autoplay=0", true, true
	}
	if m := biliBVRe.FindStringSubmatch(raw); m != nil {
		return "https://player.bilibili.com/player.html?bvid=" + m[1] + "&page=1", false, true
	}
	if m := biliAVRe.FindStringSubmatch(raw); m != nil {
		aid := m[1]
		if len(aid) >= 2 && strings.EqualFold(aid[:2], "av") {
			aid = aid[2:]
		}
		return "https://player.bilibili.com/player.html?aid=" + aid + "&page=1", false, true
	}
	if m := b23Re.FindStringSubmatch(raw); m != nil {
		return "https://www.bilibili.com/video/" + m[1], false, true
	}
	if id := firstSubmatch(raw, vimeoRe, vimeoEmbedRe); id != "" {
		return "https://player.vimeo.com/video/" + id, false, true
	}
	if m := youkuRe.FindStringSubmatch(raw); m != nil {
		return "https://player.youku.com/embed/" + m[1], false, true
	}
	if m := iqiyiRe.FindStringSubmatch(raw); m != nil {
		return "https://open.iqiyi.com/developer/player_js/coopPlayerIndex.html?vid=" + m[1], false, true
	}
	return "", false, false
}

func firstSubmatch(s string, res ...*regexp.Regexp) string {
	for _, re := range res {
		if m := re.FindStringSubmatch(s); m != nil && m[1] != "" {
			return m[1]
		}
	}
	return ""
}

func editorIframe(src string, douyin bool) string {
	ref := ""
	if douyin {
		ref = ` referrerpolicy="unsafe-url"`
	}
	iframe := `<iframe src="` + escapeAttr(src) + `" width="100%" height="100%" frameborder="0" allow="` + editorIframeAllow + `" allowfullscreen` + ref + ` style="position: absolute; top: 0; left: 0; width: 100%; height: 100%; border: none; border-radius: 4px;"></iframe>`
	return `<div class="video-embed-wrapper" style="position: relative; width: 100%; padding-bottom: 56.25%; height: 0; overflow: hidden; border-radius: 4px; margin: 8px 0; background: #000;">` + iframe + `</div>`
}

func editorLinkHTML(href, inner string) string {
	if isEditorFileLink(href) {
		return `<a href="` + escapeAttr(href) + `" target="_blank" rel="noopener noreferrer" download class="doc-file-download-link" data-file-download="1">` + inner + `</a>`
	}
	return `<a href="` + escapeAttr(href) + `" target="_blank" rel="noopener noreferrer" style="` + editorLinkStyle + `">` + inner + `</a>`
}

func isEditorFileLink(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	path := raw
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		path = u.EscapedPath()
		if u.RawQuery != "" {
			path += "?" + u.RawQuery
		}
		if u.Fragment != "" {
			path += "#" + u.Fragment
		}
	}
	return fileLinkExtRe.MatchString(path)
}

func editorImgHTML(src, alt string) string {
	return `<img src="` + escapeAttr(src) + `" alt="` + escapeAttr(alt) + `" draggable="false" style="` + editorImgStyle + `" />`
}

var (
	fontColorPresets = stringSet(
		"#ffffff", "#ef4444", "#f97316", "#facc15",
		"#10b981", "#3b82f6", "#8b5cf6", "#ec4899",
	)
	bgColorPresets = stringSet(
		"#f3f4f6", "#fee2e2", "#fed7aa", "#fef3c7", "#d1fae5", "#dbeafe", "#ede9fe", "#fce7f3",
		"#e5e7eb", "#9ca3af", "#f87171", "#fb923c", "#fbbf24", "#34d399", "#60a5fa", "#a78bfa", "#ec4899",
	)
)

func stringSet(vals ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		out[v] = struct{}{}
	}
	return out
}

func renderInline(src, blockID string, pend *[]AppDocPendingUpload) string {
	var b strings.Builder
	prev := rune(0)
	for len(src) > 0 {
		if html, n, ok := tryMarkup(src, prev, blockID, pend); ok {
			b.WriteString(html)
			prev = lastRune(src[:n])
			src = src[n:]
			continue
		}
		r, size := utf8.DecodeRuneInString(src)
		writeEscaped(&b, r)
		prev = r
		src = src[size:]
	}
	return b.String()
}

func tryMarkup(src string, prev rune, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	if html, n, ok := tryCodeSpan(src); ok {
		return html, n, true
	}
	if html, n, ok := tryImage(src, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryLink(src, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryVideo(src, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := trySpan(src, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryUnderline(src, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryBreak(src); ok {
		return html, n, true
	}
	if html, n, ok := tryWrap(src, "**", "strong", prev, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryWrap(src, "__", "strong", prev, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryWrap(src, "~~", "s", prev, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryWrap(src, "*", "em", prev, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryWrap(src, "_", "em", prev, blockID, pend); ok {
		return html, n, true
	}
	if html, n, ok := tryAutolink(src); ok {
		return html, n, true
	}
	return "", 0, false
}

func tryCodeSpan(src string) (string, int, bool) {
	if !strings.HasPrefix(src, "`") || strings.HasPrefix(src, "``") {
		return "", 0, false
	}
	rest := src[1:]
	idx := strings.IndexByte(rest, '`')
	if idx <= 0 || strings.Contains(rest[:idx], "\n") {
		return "", 0, false
	}
	return "<code>" + escapeText(rest[:idx]) + "</code>", idx + 2, true
}

func tryImage(src, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	if !strings.HasPrefix(src, "![") {
		return "", 0, false
	}
	alt, url, n, ok := parseMDLink(src[1:])
	if !ok {
		return "", 0, false
	}
	n++ // the leading '!'
	if isRemoteURL(url) {
		return editorImgHTML(url, alt), n, true
	}
	if pend != nil {
		*pend = append(*pend, AppDocPendingUpload{
			Path: url, Kind: uploadKind(url), BlockID: blockID, Alt: alt,
		})
	}
	if alt != "" {
		return escapeText(alt), n, true
	}
	return "", n, true
}

func tryLink(src, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	if !strings.HasPrefix(src, "[") {
		return "", 0, false
	}
	text, url, n, ok := parseMDLink(src)
	if !ok {
		return "", 0, false
	}
	inner := renderInline(text, blockID, pend)
	if !isRemoteURL(url) {
		if pend != nil {
			*pend = append(*pend, AppDocPendingUpload{
				Path: url, Kind: uploadKind(url), BlockID: blockID, Alt: htmlToPlain(inner),
			})
		}
		return inner, n, true
	}
	return editorLinkHTML(url, inner), n, true
}

func parseMDLink(src string) (text, url string, n int, ok bool) {
	if !strings.HasPrefix(src, "[") {
		return "", "", 0, false
	}
	close := strings.Index(src, "](")
	if close < 1 {
		return "", "", 0, false
	}
	end := strings.IndexByte(src[close+2:], ')')
	if end < 0 {
		return "", "", 0, false
	}
	text = src[1:close]
	url = strings.TrimSpace(src[close+2 : close+2+end])
	if i := strings.Index(url, ` "`); i > 0 {
		url = strings.TrimSpace(url[:i])
	}
	return text, url, close + 2 + end + 1, true
}

func tryVideo(src, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	srcURL, n, ok := matchVideoTag(src)
	if !ok {
		return "", 0, false
	}
	if isRemoteURL(srcURL) {
		return buildEditorVideoHTML(srcURL), n, true
	}
	if srcURL != "" && pend != nil {
		*pend = append(*pend, AppDocPendingUpload{Path: srcURL, Kind: "video", BlockID: blockID})
	}
	return "", n, true
}

func trySpan(src, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	re := regexp.MustCompile(`(?i)^<span\s+style="([^"]*)"\s*>`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		return "", 0, false
	}
	rest := src[len(m[0]):]
	end := strings.Index(strings.ToLower(rest), "</span>")
	if end < 0 {
		return "", 0, false
	}
	style := safeColorStyle(m[1])
	inner := renderInline(rest[:end], blockID, pend)
	n := len(m[0]) + end + len("</span>")
	if style == "" {
		return inner, n, true
	}
	return `<span style="` + style + `">` + inner + `</span>`, n, true
}

func safeColorStyle(style string) string {
	var out []string
	for _, decl := range strings.Split(style, ";") {
		idx := strings.Index(decl, ":")
		if idx < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(decl[:idx]))
		val := strings.ToLower(strings.TrimSpace(decl[idx+1:]))
		switch prop {
		case "color":
			if _, ok := fontColorPresets[val]; ok {
				out = append(out, "color: "+val)
			}
		case "background-color":
			if _, ok := bgColorPresets[val]; ok {
				out = append(out, "background-color: "+val)
			}
		}
	}
	return strings.Join(out, "; ")
}

func tryUnderline(src, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	if !strings.HasPrefix(strings.ToLower(src), "<u>") {
		return "", 0, false
	}
	rest := src[3:]
	end := strings.Index(strings.ToLower(rest), "</u>")
	if end < 0 {
		return "", 0, false
	}
	inner := renderInline(rest[:end], blockID, pend)
	return "<u>" + inner + "</u>", 3 + end + 4, true
}

func tryBreak(src string) (string, int, bool) {
	re := regexp.MustCompile(`(?i)^<br\s*/?>`)
	m := re.FindString(src)
	if m == "" {
		return "", 0, false
	}
	return "<br>", len(m), true
}

func tryWrap(src, delim, tag string, prev rune, blockID string, pend *[]AppDocPendingUpload) (string, int, bool) {
	if !strings.HasPrefix(src, delim) {
		return "", 0, false
	}
	if len(delim) == 1 && (delim == "*" && strings.HasPrefix(src, "**") || delim == "_" && strings.HasPrefix(src, "__")) {
		return "", 0, false
	}
	rest := src[len(delim):]
	idx := strings.Index(rest, delim)
	if idx <= 0 || strings.Contains(rest[:idx], "\n") {
		return "", 0, false
	}
	inner := rest[:idx]
	if strings.HasPrefix(inner, " ") || strings.HasSuffix(inner, " ") {
		return "", 0, false
	}
	n := len(delim) + idx + len(delim)
	var next rune
	if n < len(src) {
		next, _ = utf8.DecodeRuneInString(src[n:])
	}
	if len(delim) == 1 && (isWord(prev) || isWord(next)) {
		return "", 0, false
	}
	return "<" + tag + ">" + renderInline(inner, blockID, pend) + "</" + tag + ">", n, true
}

func tryAutolink(src string) (string, int, bool) {
	if !strings.HasPrefix(src, "<http://") && !strings.HasPrefix(src, "<https://") {
		return "", 0, false
	}
	end := strings.IndexByte(src, '>')
	if end < 0 {
		return "", 0, false
	}
	rawURL := src[1:end]
	return editorLinkHTML(rawURL, escapeText(rawURL)), end + 1, true
}

func isRemoteURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.HasPrefix(strings.ToLower(raw), "https://") || strings.HasPrefix(strings.ToLower(raw), "http://")
}

func uploadKind(path string) string {
	low := strings.ToLower(path)
	switch {
	// SVG is not an app image. Console upload rejects image/svg+xml.
	case hasExt(low, ".png", ".jpg", ".jpeg", ".gif", ".webp"):
		return "image"
	case hasExt(low, ".mp4", ".webm", ".mov"):
		return "video"
	default:
		return "file"
	}
}

func hasExt(path string, exts ...string) bool {
	for _, ext := range exts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func attrValue(attrs, name string) string {
	re := regexp.MustCompile(`(?i)(?:^|\s)` + regexp.QuoteMeta(name) + `\s*=\s*"([^"]*)"`)
	m := re.FindStringSubmatch(attrs)
	if m == nil {
		return ""
	}
	return m[1]
}

func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}

func writeEscaped(b *strings.Builder, r rune) {
	switch r {
	case '&':
		b.WriteString("&amp;")
	case '<':
		b.WriteString("&lt;")
	case '>':
		b.WriteString("&gt;")
	default:
		b.WriteRune(r)
	}
}

func escapeText(s string) string {
	var b strings.Builder
	for _, r := range s {
		writeEscaped(&b, r)
	}
	return b.String()
}

func escapeAttr(s string) string {
	return strings.ReplaceAll(escapeText(s), `"`, "&quot;")
}

func stripTags(s string) string {
	return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
}

func htmlToPlain(s string) string {
	s = stripTags(s)
	s = strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
		"&quot;", `"`,
	).Replace(s)
	return strings.TrimSpace(s)
}

var rawMarkdownPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"**bold**", regexp.MustCompile(`\*\*[^*\n]+\*\*`)},
	{"__bold__", regexp.MustCompile(`__[^_\n]+__`)},
	{"~~strike~~", regexp.MustCompile(`~~[^~\n]+~~`)},
	{"`code`", regexp.MustCompile("`[^`\n]+`")},
	{"# heading", regexp.MustCompile(`(?m)^#{1,6}[ \t]`)},
	{"list marker", regexp.MustCompile(`(?m)^[ \t]*[-*+][ \t]`)},
	{"ordered list", regexp.MustCompile(`(?m)^[ \t]*\d+\.[ \t]`)},
}

// ValidateDeploymentBlocks checks the modern tool config envelope and rejects
// prose that is still Markdown. Code blocks remain plain source.
func ValidateDeploymentBlocks(body map[string]any) error {
	blocks, ok, err := deploymentBlockList(body)
	if err != nil || !ok {
		return err
	}
	for i, item := range blocks {
		block, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("deployment.blocks[%d] must be an object", i)
		}
		typ := strings.ToLower(strings.TrimSpace(stringFromMap(block, "type")))
		if err := validateAppToolBlock(block, typ, i); err != nil {
			return err
		}
		if !proseBlockType(typ) {
			continue
		}
		raw, exists := block["content"]
		if !exists || raw == nil {
			continue
		}
		content, ok := raw.(string)
		if !ok {
			return fmt.Errorf("deployment.blocks[%d] content must be a string", i)
		}
		for _, p := range rawMarkdownPatterns {
			if loc := p.re.FindStringIndex(content); loc != nil {
				sample := content[loc[0]:loc[1]]
				return fmt.Errorf("deployment.blocks[%d] type %s still contains Markdown %s (%q). The editor shows those characters as text. Run: sensecraft-cli app doc compile --from-md <file>, or pass that file as app create --from-md / app update --from-md", i, typ, p.name, sample)
			}
		}
	}
	return nil
}

func proseBlockType(typ string) bool {
	switch typ {
	case "paragraph", "h1", "h2", "h3", "h4", "h5", "h6", "bullet", "number", "todo", "quote", "step":
		return true
	default:
		return false
	}
}

// NormalizeDeploymentBlocks fills editor attributes on text blocks before a
// create or update is sent. A paragraph that is only a remote video URL, or
// only a <video> tag, becomes the Insert Video HTML. Remote <a> and <img>
// tags gain the editor's target, download, and image attributes when those
// are missing. A video URL inside a sentence stays text. Markdown markers are
// not rewritten. code and step blocks are left alone.
func NormalizeDeploymentBlocks(body map[string]any) {
	blocks, ok, err := deploymentBlockList(body)
	if err != nil || !ok {
		return
	}
	for _, item := range blocks {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(stringFromMap(block, "type")))
		if typ == "step" || !proseBlockType(typ) {
			continue
		}
		content, ok := block["content"].(string)
		if !ok {
			continue
		}
		block["content"] = normalizeEditorContent(typ, content)
	}
}

func normalizeEditorContent(typ, content string) string {
	if typ == "paragraph" {
		if html, ok := editorVideoHTML(content); ok {
			return html
		}
	}
	return rewriteEditorTags(content)
}

func rewriteEditorTags(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if html, n, ok := consumeEditorAnchor(s[i:]); ok {
			b.WriteString(html)
			i += n
			continue
		}
		if html, n, ok := consumeEditorImg(s[i:]); ok {
			b.WriteString(html)
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func consumeEditorAnchor(s string) (string, int, bool) {
	if !tagAt(s, "a") {
		return "", 0, false
	}
	gt := strings.IndexByte(s, '>')
	if gt < 0 {
		return "", 0, false
	}
	endRel := strings.Index(strings.ToLower(s[gt+1:]), "</a>")
	if endRel < 0 {
		return "", 0, false
	}
	end := gt + 1 + endRel
	n := end + len("</a>")
	attrs := s[len("a")+1 : gt]
	href := attrValue(attrs, "href")
	inner := s[gt+1 : end]
	if !isRemoteURL(href) || anchorAlreadyEditor(attrs, href) {
		return s[:n], n, true
	}
	return editorLinkHTML(href, inner), n, true
}

func anchorAlreadyEditor(attrs, href string) bool {
	if !strings.EqualFold(attrValue(attrs, "target"), "_blank") {
		return false
	}
	if !strings.Contains(strings.ToLower(attrValue(attrs, "rel")), "noopener") {
		return false
	}
	if isEditorFileLink(href) {
		if !strings.Contains(attrValue(attrs, "class"), "doc-file-download-link") {
			return false
		}
		if attrValue(attrs, "data-file-download") != "1" {
			return false
		}
		return hasAttr(attrs, "download")
	}
	return strings.Contains(attrValue(attrs, "style"), editorLinkStyle)
}

func consumeEditorImg(s string) (string, int, bool) {
	if !tagAt(s, "img") {
		return "", 0, false
	}
	gt := strings.IndexByte(s, '>')
	if gt < 0 {
		return "", 0, false
	}
	n := gt + 1
	attrs := s[len("img")+1 : gt]
	src := attrValue(attrs, "src")
	if !isRemoteURL(src) || imgAlreadyEditor(attrs) {
		return s[:n], n, true
	}
	return editorImgHTML(src, attrValue(attrs, "alt")), n, true
}

func imgAlreadyEditor(attrs string) bool {
	if !strings.EqualFold(attrValue(attrs, "draggable"), "false") {
		return false
	}
	return strings.Contains(attrValue(attrs, "style"), editorImgStyle)
}

func tagAt(s, name string) bool {
	prefix := "<" + name
	if len(s) < len(prefix)+1 || !strings.EqualFold(s[:len(prefix)], prefix) {
		return false
	}
	next := s[len(prefix)]
	return next == '>' || next == '/' || unicode.IsSpace(rune(next))
}

func hasAttr(attrs, name string) bool {
	re := regexp.MustCompile(`(?i)(?:^|\s)` + regexp.QuoteMeta(name) + `(?:\s*=|\s|$)`)
	return re.MatchString(attrs)
}

func deploymentBlockList(body map[string]any) ([]any, bool, error) {
	if body == nil {
		return nil, false, nil
	}
	raw, ok := body["deployment"]
	if !ok || raw == nil {
		return nil, false, nil
	}
	dep, ok := raw.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("deployment must be an object with a blocks array")
	}
	blocks, ok := dep["blocks"]
	if !ok || blocks == nil {
		return nil, false, nil
	}
	list, ok := blocks.([]any)
	if !ok {
		return nil, false, fmt.Errorf("deployment.blocks must be an array")
	}
	return list, true, nil
}
