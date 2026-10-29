package service

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
	"sensecraft-cloud-terminal/internal/ui"
)

// AppPublishDeclarationURL is the English Publish Declaration (Console warranty link).
const AppPublishDeclarationURL = "https://sensecraft.seeed.cc/legal/en/declaration"

// ErrConfirmRequired is returned for L2 write ops when --yes is missing
// and there was no interactive TTY confirm. Agents must ask in chat first
// (operation, environment, and object id); the user's first request is not confirmation.
var ErrConfirmRequired = errors.New("confirmation required: ask the user first (include the operation, environment, and object id), wait for them to confirm, then re-run with --yes. The first user request (for example \"delete 63760\") is intent, not confirmation")

// ErrDeclarationRequired matches Console publish dialog warrantyAccepted checkbox.
// Agreement is client-side only (API body stays app_id + privacy); see legal URL in the error.
var ErrDeclarationRequired = errors.New(
	"publish declaration agreement required: re-run with --agree-declaration after the user has read and agreed to " +
		AppPublishDeclarationURL +
		" (submitting for review locks editing until review finishes)",
)

// ErrAppLockedUnderReview matches Console isAppLockedForEditing (be_public=2).
var ErrAppLockedUnderReview = errors.New("app is under review (be_public=2); editing is locked — wait for review or run app withdraw")

// ErrPublishedAppNoDelete matches Console: delete only when be_public==0.
var ErrPublishedAppNoDelete = errors.New("published or under-review apps cannot be deleted; unpublish a public app, or withdraw an app under review, then delete")

// ErrForeignAppNoEdit matches Console isMyApp / list.vue user_id == user_id.
var ErrForeignAppNoEdit = errors.New("cannot edit another user's app")

// AppSketchSaveHint is returned when updating a published app (plaza copy unchanged).
const AppSketchSaveHint = "published app was saved as a draft (sketch); plaza listing is unchanged until: sensecraft-cli app publish --id <app_id> --agree-declaration --yes"

// AppDraftPublishHint is returned when privacy=2 is sent for an already-public app
// (Console publish-draft / publishDraft).
const AppDraftPublishHint = "submitted draft for review (same as Console publishDraft); plaza listing updates after approval"

func requireYes(yes bool) error {
	if yes {
		return nil
	}
	if promptL2Confirm() {
		return nil
	}
	return ErrConfirmRequired
}

func promptL2Confirm() bool {
	if os.Getenv("SENSECRAFT_NO_CONFIRM_PROMPT") != "" {
		return false
	}
	// go test registers -test.v; never block the suite on stdin.
	if flag.Lookup("test.v") != nil {
		return false
	}
	c := ui.DefaultConsole()
	if !c.IsInteractive() {
		return false
	}
	ok, err := c.Confirm("L2 write will take effect now (delete/publish/deploy/stop). Confirm execution", false)
	return err == nil && ok
}

func RequireYes(yes bool) error {
	return requireYes(yes)
}

func requireAgreeDeclaration(agree bool) error {
	if !agree {
		return ErrDeclarationRequired
	}
	return nil
}

type AppPrivacyParams struct {
	AppID            string
	Privacy          int // 2=submit review, 0=private; never 1
	Yes              bool
	AgreeDeclaration bool // required when Privacy=2 (Console warrantyAccepted)
}

func (s AppService) SetAppPrivacy(p AppPrivacyParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.AppID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	if p.Privacy != 0 && p.Privacy != 2 {
		return nil, fmt.Errorf("privacy must be 0 (unpublish) or 2 (submit review); never 1")
	}
	if p.Privacy == 2 {
		if err := requireAgreeDeclaration(p.AgreeDeclaration); err != nil {
			return nil, err
		}
	}
	view, err := s.ViewApp(AppViewParams{AppID: p.AppID, ViewSketch: true})
	if err != nil {
		return nil, err
	}
	existing, err := rawJSONObject(view)
	if err != nil {
		return nil, err
	}
	if err := s.rejectUnownedAppWrite(p.AppID, existing); err != nil {
		return nil, err
	}
	current := appMapBePublic(existing)
	if p.Privacy == 2 && current == 2 {
		return nil, ErrAppLockedUnderReview
	}
	body := map[string]any{
		"app_id":  p.AppID,
		"privacy": p.Privacy,
	}
	data, err := s.client.Post(enum.AIApiPathAppPrivacy, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = sandboxAppPrivacy(p.AppID, p.Privacy)
		} else {
			return nil, err
		}
	}
	if p.Privacy == 2 && current == 1 {
		data = attachAppJSONMeta(data, map[string]any{
			"draft_publish": true,
			"hint":          AppDraftPublishHint,
			"be_public":     2,
		})
	}
	name := stringFromMap(existing, "app_name")
	id := strings.TrimSpace(p.AppID)
	// 提交审核后只给详情页。审核中编辑页会报 The application must not allow editing。
	if p.Privacy == 2 {
		return s.stampReviewDetailURL(data, id, name), nil
	}
	return s.stampOwnerDetailURL(data, id, name), nil
}

type AppDeleteParams struct {
	AppID string
	Yes   bool
}

func (s AppService) DeleteApp(p AppDeleteParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.AppID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	view, err := s.ViewApp(AppViewParams{AppID: p.AppID, ViewSketch: true})
	if err != nil {
		return nil, err
	}
	existing, err := rawJSONObject(view)
	if err != nil {
		return nil, err
	}
	if err := s.rejectUnownedAppWrite(p.AppID, existing); err != nil {
		return nil, err
	}
	if appMapBePublic(existing) != 0 {
		return nil, ErrPublishedAppNoDelete
	}
	body := map[string]any{"app_id": p.AppID}
	data, err := s.client.Post(enum.AIApiPathAppDelete, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			return sandboxAppDelete(p.AppID), nil
		}
		return nil, err
	}
	return data, nil
}

type AppCreateParams struct {
	Name     string
	FromJSON string // path or "-"
	// Markdown is the document text, already read by the command.
	// MarkdownSet is true when --from-md was passed, including a blank file.
	Markdown    string
	MarkdownSet bool
}

func (s AppService) CreateApp(p AppCreateParams) (json.RawMessage, error) {
	name := strings.TrimSpace(p.Name)
	var extra map[string]any
	var notes *AppDocCompileResult

	if strings.TrimSpace(p.FromJSON) != "" {
		body, err := LoadJSONObject(p.FromJSON)
		if err != nil {
			return nil, err
		}
		if err := rejectForbiddenPublicFlags(body); err != nil {
			return nil, err
		}
		extra = body
	}
	if p.MarkdownSet {
		result, err := CompileAppMarkdown(p.Markdown)
		if err != nil {
			return nil, err
		}
		notes = &result
		if extra == nil {
			extra = map[string]any{}
		}
		applyCompiledMarkdown(extra, notes)
	}
	if extra != nil {
		if name == "" {
			name = stringFromMap(extra, "app_name")
		}
		// create_app only accepts app_name; remaining fields go to a follow-up update when present.
		delete(extra, "app_name")
		delete(extra, "app_id")
		if len(extra) == 0 {
			extra = nil
		}
	}
	if err := validateAppName(name); err != nil {
		return nil, err
	}
	if extra != nil {
		NormalizeDeploymentBlocks(extra)
		if err := validateAppInfoPatch(extra); err != nil {
			return nil, err
		}
		if err := ValidateDeploymentBlocks(extra); err != nil {
			return nil, err
		}
		if scene := stringFromMap(extra, "scene"); scene != "" {
			if err := s.validateAppSceneID(scene); err != nil {
				return nil, err
			}
		}
	}

	created, err := s.client.Post(enum.AIApiPathAppCreate, map[string]any{"app_name": name})
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			created = sandboxAppCreate(name)
		} else {
			return nil, err
		}
	}

	appID := extractAppID(created)
	if appID == "" {
		return created, nil
	}
	if extra == nil {
		// create_app often returns only app_id. Keep the name we sent so the
		// detail link still gets the title slug.
		return attachCompileNotes(s.withAppConsoleURL(ensureAppIdentity(created, appID, name)), notes), nil
	}
	// A just-created app has no sketch yet. view_sketch=1 returns 11306
	// ("Not Found the resources"); still write the rest of the payload.
	updated, err := s.applyAppPatch(appID, extra, true)
	if err != nil {
		return nil, fmt.Errorf("created app_id=%s but update failed: %w", appID, err)
	}
	// aiserver update_app often returns {} / null — never drop the known app_id
	// (Agents would retry create and spawn duplicates).
	return attachCompileNotes(s.withAppConsoleURL(finalizeAppCreateResult(created, updated, appID, name)), notes), nil
}

// finalizeAppCreateResult prefers an update body that still carries identity;
// otherwise keeps create payload and forces app_id / app_name.
func finalizeAppCreateResult(created, updated json.RawMessage, appID, name string) json.RawMessage {
	if extractAppID(updated) != "" {
		return updated
	}
	return ensureAppIdentity(created, appID, name)
}

func ensureAppIdentity(raw json.RawMessage, appID, name string) json.RawMessage {
	m := map[string]any{}
	if len(raw) > 0 && string(raw) != "null" {
		if parsed, err := rawJSONObject(raw); err == nil {
			m = parsed
		} else {
			// bare string id from create_app
			var s string
			if err := json.Unmarshal(raw, &s); err == nil && strings.TrimSpace(s) != "" {
				m["app_id"] = strings.TrimSpace(s)
			}
		}
	}
	if appID != "" {
		m["app_id"] = appID
	}
	if name != "" {
		if stringFromMap(m, "app_name") == "" {
			m["app_name"] = name
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

func (s AppService) withAppConsoleURL(data json.RawMessage) json.RawMessage {
	id := extractAppID(data)
	name := extractAppString(data, "app_name")
	return s.stampOwnerDetailURL(data, id, name)
}

// stampOwnerDetailURL adds the edit link and the creator-center detail link.
// appName is used only to build the slug; it is not written into the JSON.
func (s AppService) stampOwnerDetailURL(data json.RawMessage, appID, appName string) json.RawMessage {
	return s.stampAppLinks(data, appID, appName, true)
}

// stampReviewDetailURL is the link set after submit-for-review. be_public=2
// locks the editor, so console_url (/application/edit) is omitted.
func (s AppService) stampReviewDetailURL(data json.RawMessage, appID, appName string) json.RawMessage {
	return s.stampAppLinks(data, appID, appName, false)
}

func (s AppService) stampAppLinks(data json.RawMessage, appID, appName string, includeEdit bool) json.RawMessage {
	if extractAppID(data) == "" && strings.TrimSpace(appID) != "" {
		data = ensureAppIdentity(data, appID, "")
	}
	id := extractAppID(data)
	if id == "" {
		id = strings.TrimSpace(appID)
	}
	name := extractAppString(data, "app_name")
	if name == "" {
		name = strings.TrimSpace(appName)
	}
	out := data
	if includeEdit {
		out = attachConsoleURL(data, "app_id", id, AppEditConsoleURL(s.remoteType, id))
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil || m == nil {
		return out
	}
	if !includeEdit {
		delete(m, "console_url")
	}
	detail := AppDetailConsoleURL(s.remoteType, id, name)
	if detail != "" {
		m["detail_url"] = detail
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return out
	}
	return encoded
}

type AppUpdateParams struct {
	AppID    string
	FromJSON string
	// Markdown is the document text, already read by the command.
	// MarkdownSet is true when --from-md was passed, including a blank file.
	Markdown    string
	MarkdownSet bool
	Yes         bool
	// ActivitySet is true when --activity-id was passed. An empty ActivityID
	// omits the field so the server leaves existing activity tags unchanged.
	ActivitySet bool
	ActivityID  string
	// CancelActivitySet is true when --cancel-activity-id was passed. An empty
	// value omits the field. A positive integer leaves that activity.
	CancelActivitySet bool
	CancelActivityID  string
}

// appInfoMergeKeys are Console infoForm fields. aiserver often no-ops sparse
// updates that only send app_name; CLI fills missing keys from view_sketch first
// (same idea as model update merge). Do not merge be_public — visibility is
// publish/unpublish only.
var appInfoMergeKeys = []string{
	"app_name", "excerpt", "scene", "allow_clone", "allow_commerce",
	"images", "license", "author",
}

func (s AppService) UpdateApp(p AppUpdateParams) (json.RawMessage, error) {
	if err := requireYes(p.Yes); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.AppID) == "" {
		return nil, fmt.Errorf("--id is required")
	}
	fromJSON := strings.TrimSpace(p.FromJSON)
	if fromJSON == "" && !p.MarkdownSet && !p.ActivitySet && !p.CancelActivitySet {
		return nil, fmt.Errorf("provide --from-json, --from-md, --activity-id, and/or --cancel-activity-id")
	}
	body := map[string]any{}
	if fromJSON != "" {
		loaded, err := LoadJSONObject(fromJSON)
		if err != nil {
			return nil, err
		}
		body = loaded
	}
	var notes *AppDocCompileResult
	if p.MarkdownSet {
		result, err := CompileAppMarkdown(p.Markdown)
		if err != nil {
			return nil, err
		}
		notes = &result
		applyCompiledMarkdown(body, notes)
	}
	if p.ActivitySet {
		body["activity_id"] = strings.TrimSpace(p.ActivityID)
	}
	if p.CancelActivitySet {
		body["cancel_activity_id"] = strings.TrimSpace(p.CancelActivityID)
	}
	updated, err := s.applyAppPatch(p.AppID, body, false)
	if err != nil {
		return nil, err
	}
	return attachCompileNotes(updated, notes), nil
}

func (s AppService) AttachCoverImage(appID, fileURL string, yes bool) (json.RawMessage, error) {
	if err := requireYes(yes); err != nil {
		return nil, fmt.Errorf("%w; --app-id attaches the image as the app cover", err)
	}
	appID = strings.TrimSpace(appID)
	fileURL = strings.TrimSpace(fileURL)
	if appID == "" {
		return nil, fmt.Errorf("--app-id is required")
	}
	if fileURL == "" {
		return nil, fmt.Errorf("file_url is required")
	}
	view, err := s.ViewApp(AppViewParams{AppID: appID, ViewSketch: true})
	if err != nil {
		return nil, fmt.Errorf("load existing app before attach: %w", err)
	}
	existing, err := rawJSONObject(view)
	if err != nil {
		return nil, fmt.Errorf("parse existing app before attach: %w", err)
	}
	images := []string{fileURL}
	for _, old := range normalizeAppImages(existing["images"]) {
		if old != fileURL {
			images = append(images, old)
		}
	}
	return s.applyAppPatch(appID, map[string]any{"images": images}, false)
}

// appResourceNotFound is the AI API code for view_app when the sketch or
// published snapshot does not exist yet (just-created apps).
const appResourceNotFound = 11306

func isAppResourceNotFound(err error) bool {
	var api client.AIAPIError
	return errors.As(err, &api) && api.Code == appResourceNotFound
}

// loadExistingApp prefers the sketch. When that snapshot is missing, it falls
// back to the published copy. Both missing returns the 11306 error.
func (s AppService) loadExistingApp(appID string) (map[string]any, error) {
	view, err := s.ViewApp(AppViewParams{AppID: appID, ViewSketch: true})
	if err != nil && isAppResourceNotFound(err) {
		published, pubErr := s.ViewApp(AppViewParams{AppID: appID, ViewSketch: false})
		if pubErr == nil {
			view, err = published, nil
		} else if !isAppResourceNotFound(pubErr) {
			err = pubErr
		}
	}
	if err != nil {
		return nil, err
	}
	return rawJSONObject(view)
}

// applyCompiledMarkdown copies a Markdown compile into a create/update body.
// app_name and excerpt are filled only when the body does not already have
// them. Compiled blocks come first; blocks already in the JSON are appended,
// so hand-written tool-blocks stay after the document.
func applyCompiledMarkdown(body map[string]any, notes *AppDocCompileResult) {
	if body == nil || notes == nil {
		return
	}
	if stringFromMap(body, "app_name") == "" && strings.TrimSpace(notes.SuggestedAppName) != "" {
		name := notes.SuggestedAppName
		if utf8.RuneCountInString(name) > maxAppNameRunes {
			name = string([]rune(name)[:maxAppNameRunes])
			notes.Warnings = append(notes.Warnings, fmt.Sprintf("suggested_app_name truncated to %d characters", maxAppNameRunes))
		}
		body["app_name"] = name
	}
	if stringFromMap(body, "excerpt") == "" && strings.TrimSpace(notes.SuggestedExcerpt) != "" {
		excerpt := notes.SuggestedExcerpt
		if utf8.RuneCountInString(excerpt) > maxAppExcerptRunes {
			excerpt = string([]rune(excerpt)[:maxAppExcerptRunes])
		}
		body["excerpt"] = excerpt
	}
	compiled := make([]any, 0, len(notes.Deployment.Blocks))
	for _, block := range notes.Deployment.Blocks {
		compiled = append(compiled, block)
	}
	if len(compiled) == 0 {
		return
	}
	raw, ok := body["deployment"]
	if !ok || raw == nil {
		body["deployment"] = map[string]any{"blocks": compiled}
		return
	}
	dep, ok := raw.(map[string]any)
	if !ok {
		return
	}
	existing, _ := dep["blocks"].([]any)
	merged := make([]any, 0, len(compiled)+len(existing))
	merged = append(merged, compiled...)
	merged = append(merged, existing...)
	dep["blocks"] = merged
}

func attachCompileNotes(data json.RawMessage, notes *AppDocCompileResult) json.RawMessage {
	if notes == nil {
		return data
	}
	m := map[string]any{}
	if len(data) > 0 && string(data) != "null" {
		if parsed, err := rawJSONObject(data); err == nil && parsed != nil {
			m = parsed
		}
	}
	pending := notes.PendingUploads
	if pending == nil {
		pending = []AppDocPendingUpload{}
	}
	warnings := notes.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	m["pending_uploads"] = pending
	m["warnings"] = warnings
	out, err := json.Marshal(m)
	if err != nil {
		return data
	}
	return out
}

func (s AppService) applyAppPatch(appID string, body map[string]any, allowMissingSnapshot bool) (json.RawMessage, error) {
	if err := rejectForbiddenPublicFlags(body); err != nil {
		return nil, err
	}
	NormalizeDeploymentBlocks(body)
	if err := validateAppInfoPatch(body); err != nil {
		return nil, err
	}
	if err := ValidateDeploymentBlocks(body); err != nil {
		return nil, err
	}
	if raw, ok := body["materials"]; ok {
		if err := validateAppMaterials(raw); err != nil {
			return nil, err
		}
	}
	tagsDropped, err := normalizeAppActivityID(body)
	if err != nil {
		return nil, err
	}
	if err := normalizeAppCancelActivityID(body); err != nil {
		return nil, err
	}
	if scene := stringFromMap(body, "scene"); scene != "" {
		if err := s.validateAppSceneID(scene); err != nil {
			return nil, err
		}
	}
	body["app_id"] = appID
	userSetAllowClone := false
	if _, ok := body["allow_clone"]; ok {
		userSetAllowClone = true
	}

	existing, err := s.loadExistingApp(appID)
	if err != nil {
		if !(allowMissingSnapshot && isAppResourceNotFound(err)) {
			return nil, fmt.Errorf("load existing app before update: %w", err)
		}
		existing = map[string]any{}
	} else if err := s.rejectUnownedAppWrite(appID, existing); err != nil {
		return nil, err
	}
	currentPublic := appMapBePublic(existing)
	if currentPublic == 2 {
		return nil, ErrAppLockedUnderReview
	}
	if n, ok := positiveInt64(body["activity_id"]); ok {
		if err := s.rejectInactiveActivity(n); err != nil {
			return nil, err
		}
		// Contest entry on the web also writes allow_clone=1 when it is not already 1.
		// An explicit allow_clone in this JSON is left alone.
		if !userSetAllowClone && !allowCloneIsOne(existing["allow_clone"]) {
			body["allow_clone"] = 1
		}
	}

	if patchTouchesAppInfo(body) {
		mergeAppInfoFromExisting(body, existing)
	}

	// Console info save: published apps keep be_public=1 while writing the sketch.
	if currentPublic == 1 && patchTouchesAppInfo(body) {
		body["be_public"] = 1
	}
	syncAppCoverFields(body)
	if _, ok := body["images"]; ok {
		if err := validateAppImageCount(normalizeAppImages(body["images"])); err != nil {
			return nil, err
		}
	}
	data, err := s.client.Post(enum.AIApiPathAppUpdate, body)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			data = sandboxAppUpdate(appID, body)
		} else {
			return nil, err
		}
	}
	meta := map[string]any{}
	if currentPublic == 1 {
		meta["be_public"] = 1
		meta["sketch_saved"] = true
		meta["hint"] = AppSketchSaveHint
	}
	if tagsDropped {
		meta["tags_ignored"] = true
		tagHint := "tags is ignored; pass activity_id to join an activity"
		if hint, _ := meta["hint"].(string); hint != "" {
			meta["hint"] = hint + " " + tagHint
		} else {
			meta["hint"] = tagHint
		}
	}
	if len(meta) > 0 {
		data = attachAppJSONMeta(data, meta)
	}
	return s.stampOwnerDetailURL(data, appID, stringFromMap(existing, "app_name")), nil
}

// normalizeAppActivityID prepares update_app activity participation.
// activity_id is optional. Missing, null, or blank leaves the key unset so the
// server keeps existing activity tags. A present value must be a positive
// integer and is sent as a JSON number. tags is never sent.
func normalizeAppActivityID(body map[string]any) (tagsDropped bool, err error) {
	if body == nil {
		return false, nil
	}
	_, tagsDropped = body["tags"]
	delete(body, "tags")
	raw, ok := body["activity_id"]
	if !ok || raw == nil {
		delete(body, "activity_id")
		return tagsDropped, nil
	}
	if s, isString := raw.(string); isString && strings.TrimSpace(s) == "" {
		delete(body, "activity_id")
		return tagsDropped, nil
	}
	n, ok := positiveInt64(raw)
	if !ok {
		return tagsDropped, fmt.Errorf("activity_id must be a positive integer")
	}
	body["activity_id"] = n
	return tagsDropped, nil
}

// normalizeAppCancelActivityID prepares update_app activity withdrawal.
// cancel_activity_id is optional. Missing, null, or blank leaves the key unset.
// A present value must be a positive integer and is sent as a JSON number.
// Unlike activity_id, it is not checked against the ongoing activity list and
// does not change allow_clone.
func normalizeAppCancelActivityID(body map[string]any) error {
	if body == nil {
		return nil
	}
	raw, ok := body["cancel_activity_id"]
	if !ok || raw == nil {
		delete(body, "cancel_activity_id")
		return nil
	}
	if s, isString := raw.(string); isString && strings.TrimSpace(s) == "" {
		delete(body, "cancel_activity_id")
		return nil
	}
	n, ok := positiveInt64(raw)
	if !ok {
		return fmt.Errorf("cancel_activity_id must be a positive integer")
	}
	if join, joinOK := positiveInt64(body["activity_id"]); joinOK && join == n {
		return fmt.Errorf("activity_id and cancel_activity_id must not be the same")
	}
	body["cancel_activity_id"] = n
	return nil
}

func positiveInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		if t > 0 {
			return int64(t), true
		}
	case int64:
		if t > 0 {
			return t, true
		}
	case float64:
		if t > 0 && t == float64(int64(t)) {
			return int64(t), true
		}
	case json.Number:
		n, err := t.Int64()
		if err == nil && n > 0 {
			return n, true
		}
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func appMapBePublic(m map[string]any) int {
	n, ok := asInt(m["be_public"])
	if !ok {
		return 0
	}
	return n
}

func appOwnerID(m map[string]any) string {
	if id := stringFromMap(m, "user_id"); id != "" {
		return id
	}
	return ""
}

func (s AppService) rejectUnownedAppWrite(appID string, existing map[string]any) error {
	if s.sandbox {
		if sandboxIsForeignPublicID(appID) {
			return fmt.Errorf("%w (%s)", ErrForeignAppNoEdit, appID)
		}
		return nil
	}
	if uid := strings.TrimSpace(s.userID); uid != "" {
		if owner := appOwnerID(existing); owner != "" && owner != uid {
			return fmt.Errorf("%w (%s)", ErrForeignAppNoEdit, appID)
		}
	}
	return nil
}

func attachAppJSONMeta(raw json.RawMessage, extra map[string]any) json.RawMessage {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		out := map[string]any{"raw": json.RawMessage(raw)}
		for k, v := range extra {
			out[k] = v
		}
		b, _ := json.Marshal(out)
		return b
	}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return b
}

func patchTouchesAppInfo(body map[string]any) bool {
	for _, k := range appInfoMergeKeys {
		if _, ok := body[k]; ok {
			return true
		}
	}
	return false
}

func mergeAppInfoFromExisting(body, existing map[string]any) {
	for _, k := range appInfoMergeKeys {
		if _, ok := body[k]; ok {
			continue
		}
		if v, ok := existing[k]; ok && v != nil {
			body[k] = v
		}
	}
	if _, ok := body["scene"]; ok {
		body["scene"] = stringifyAppScene(body["scene"])
	}
	if _, ok := body["images"]; ok {
		body["images"] = normalizeAppImages(body["images"])
	}
}

// syncAppCoverFields keeps images[] and pic_url in sync.
// Console list/detail (view_sketch=0) show pic_url; infoForm cover is images[0].
// Restore/create JSON from list_app often has pic_url only — without this, update_app
// writes neither a published cover nor a usable images list.
func syncAppCoverFields(body map[string]any) {
	if body == nil {
		return
	}
	imgs := normalizeAppImages(body["images"])
	cover := strings.TrimSpace(stringFromMap(body, "pic_url"))
	if cover != "" {
		imgs = prependUniqueURL(imgs, cover)
	}
	if len(imgs) == 0 {
		return
	}
	body["images"] = imgs
	body["pic_url"] = imgs[0]
}

func prependUniqueURL(urls []string, first string) []string {
	first = strings.TrimSpace(first)
	if first == "" {
		return urls
	}
	out := make([]string, 0, len(urls)+1)
	out = append(out, first)
	for _, u := range urls {
		if u != first {
			out = append(out, u)
		}
	}
	return out
}

func stringifyAppScene(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func normalizeAppImages(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		if ss, ok := v.([]string); ok {
			out := make([]string, 0, len(ss))
			for _, s := range ss {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		switch t := item.(type) {
		case string:
			s := strings.TrimSpace(t)
			if s != "" {
				out = append(out, s)
			}
		case map[string]any:
			s := stringFromMap(t, "url")
			if s == "" {
				s = stringFromMap(t, "file_url")
			}
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func rawJSONObject(raw json.RawMessage) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("expected JSON object")
	}
	return m, nil
}

func extractAppID(raw json.RawMessage) string {
	if id := extractAppString(raw, "app_id"); id != "" {
		return id
	}
	if id := extractAppString(raw, "id"); id != "" {
		return id
	}
	// Some APIs return bare string id.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

func sandboxAppCreate(name string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"app_id":   "sandbox-created-app",
		"app_name": name,
	})
	return raw
}

func sandboxAppCreateUpdate(appID, name string, extra map[string]any) json.RawMessage {
	out := map[string]any{
		"app_id":   appID,
		"app_name": name,
		"updated":  true,
	}
	for k, v := range extra {
		if k == "app_id" {
			continue
		}
		out[k] = v
	}
	raw, _ := json.Marshal(out)
	return raw
}

func sandboxAppUpdate(appID string, body map[string]any) json.RawMessage {
	out := map[string]any{"app_id": appID, "updated": true}
	for k, v := range body {
		if k == "app_id" {
			continue
		}
		out[k] = v
	}
	raw, _ := json.Marshal(out)
	return raw
}

func extractAppString(raw json.RawMessage, key string) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return stringFromMap(m, key)
}

func sandboxAppPrivacy(appID string, privacy int) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"app_id":  appID,
		"privacy": privacy,
		"ok":      true,
	})
	return raw
}

func sandboxAppDelete(appID string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"app_id":  appID,
		"deleted": true,
	})
	return raw
}
