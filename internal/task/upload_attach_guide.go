package task

import (
	"errors"
	"fmt"
	"strings"

	"sensecraft-cloud-terminal/internal/service"
	"sensecraft-cloud-terminal/internal/ui"
)

// ModelUploadAssoc is the resolved association for model upload.
type ModelUploadAssoc struct {
	ModelID  string
	Unlinked bool
	Yes      bool
}

// FileUploadAssoc is the resolved association for file upload (types 1–2).
type FileUploadAssoc struct {
	ModelID  string
	AppID    string
	ForApp   bool
	Unlinked bool
	Yes      bool
}

// ResolveModelUploadAssoc requires --id or --unlinked before any OSS upload.
// Interactive TTY: guide the user to pick attach vs new-model create.
// Agents (non-TTY): fail with ErrAttachRequired so they ask in chat first.
func ResolveModelUploadAssoc(console *ui.Console, modelID string, unlinked, yes bool) (ModelUploadAssoc, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID != "" && unlinked {
		return ModelUploadAssoc{}, fmt.Errorf("use only one of --id or --unlinked")
	}
	if modelID != "" || unlinked {
		return ModelUploadAssoc{ModelID: modelID, Unlinked: unlinked, Yes: yes}, nil
	}
	if console != nil && console.IsInteractive() {
		return promptModelUploadAssoc(console, yes)
	}
	return ModelUploadAssoc{}, service.ErrAttachRequired
}

func promptModelUploadAssoc(console *ui.Console, yes bool) (ModelUploadAssoc, error) {
	idx, err := console.Select(
		"This upload is not linked yet. How should the file be associated?",
		[]string{
			"Attach to an existing model (writes file_url and bumps version)",
			"Brand-new model (unlinked upload for later model create)",
			"Cancel",
		},
		0,
	)
	if err != nil {
		return ModelUploadAssoc{}, err
	}
	switch idx {
	case 0:
		id, err := console.Input("Model ID", "")
		if err != nil {
			return ModelUploadAssoc{}, err
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return ModelUploadAssoc{}, fmt.Errorf("model id is required")
		}
		if !yes {
			ok, err := console.Confirm("Attach this file to model "+id+" and bump patch version?", false)
			if err != nil {
				return ModelUploadAssoc{}, err
			}
			if !ok {
				return ModelUploadAssoc{}, errors.New("cancelled")
			}
			yes = true
		}
		return ModelUploadAssoc{ModelID: id, Yes: yes}, nil
	case 1:
		ok, err := console.Confirm("Upload without attaching (for model create only)?", false)
		if err != nil {
			return ModelUploadAssoc{}, err
		}
		if !ok {
			return ModelUploadAssoc{}, errors.New("cancelled")
		}
		return ModelUploadAssoc{Unlinked: true}, nil
	default:
		return ModelUploadAssoc{}, errors.New("cancelled")
	}
}

// ResolveFileUploadAssoc gates type-1/2 uploads that can attach to a model or app.
// Types 3–6 (app package / firmware / doc / archive) are never gated here.
func ResolveFileUploadAssoc(console *ui.Console, fileType int, modelID, appID string, forApp, unlinked, yes bool) (FileUploadAssoc, error) {
	modelID = strings.TrimSpace(modelID)
	appID = strings.TrimSpace(appID)
	if modelID != "" && appID != "" {
		return FileUploadAssoc{}, fmt.Errorf("use only one of --model-id or --app-id")
	}
	if unlinked && (modelID != "" || appID != "") {
		return FileUploadAssoc{}, fmt.Errorf("use only one of --unlinked or --model-id/--app-id")
	}
	if forApp && modelID != "" {
		return FileUploadAssoc{}, fmt.Errorf("--for-app is for type-1 app images; do not combine with --model-id")
	}

	// Types that have no model/app attach flags stay open.
	if fileType != 0 && fileType != service.FileTypeImage && fileType != service.FileTypeFile {
		return FileUploadAssoc{
			ModelID: modelID, AppID: appID, ForApp: forApp, Unlinked: true, Yes: yes,
		}, nil
	}

	if modelID != "" || appID != "" || forApp || unlinked {
		return FileUploadAssoc{
			ModelID: modelID, AppID: appID, ForApp: forApp, Unlinked: unlinked, Yes: yes,
		}, nil
	}

	if console != nil && console.IsInteractive() {
		return promptFileUploadAssoc(console, fileType, yes)
	}
	return FileUploadAssoc{}, service.ErrAttachRequired
}

func promptFileUploadAssoc(console *ui.Console, fileType int, yes bool) (FileUploadAssoc, error) {
	options := []string{
		"Attach to an existing model (--model-id)",
		"Attach as app cover image (--app-id)",
		"Unlinked upload for later JSON (model create / pic_url / images[])",
		"Cancel",
	}
	if fileType == service.FileTypeImage {
		options = []string{
			"Attach as app cover image (--app-id)",
			"Unlinked image for app images[] later (--for-app)",
			"Unlinked image for model pic_url later (--unlinked)",
			"Cancel",
		}
	}

	idx, err := console.Select(
		"This upload is not linked yet. How should the file be associated?",
		options,
		0,
	)
	if err != nil {
		return FileUploadAssoc{}, err
	}

	if fileType == service.FileTypeImage {
		switch idx {
		case 0:
			id, err := console.Input("App ID", "")
			if err != nil {
				return FileUploadAssoc{}, err
			}
			id = strings.TrimSpace(id)
			if id == "" {
				return FileUploadAssoc{}, fmt.Errorf("app id is required")
			}
			if !yes {
				ok, err := console.Confirm("Attach this image as the cover of app "+id+"?", false)
				if err != nil {
					return FileUploadAssoc{}, err
				}
				if !ok {
					return FileUploadAssoc{}, errors.New("cancelled")
				}
				yes = true
			}
			return FileUploadAssoc{AppID: id, Yes: yes}, nil
		case 1:
			return FileUploadAssoc{ForApp: true}, nil
		case 2:
			return FileUploadAssoc{Unlinked: true}, nil
		default:
			return FileUploadAssoc{}, errors.New("cancelled")
		}
	}

	switch idx {
	case 0:
		id, err := console.Input("Model ID", "")
		if err != nil {
			return FileUploadAssoc{}, err
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return FileUploadAssoc{}, fmt.Errorf("model id is required")
		}
		if !yes {
			ok, err := console.Confirm("Attach this file to model "+id+" and bump patch version?", false)
			if err != nil {
				return FileUploadAssoc{}, err
			}
			if !ok {
				return FileUploadAssoc{}, errors.New("cancelled")
			}
			yes = true
		}
		return FileUploadAssoc{ModelID: id, Yes: yes}, nil
	case 1:
		id, err := console.Input("App ID", "")
		if err != nil {
			return FileUploadAssoc{}, err
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return FileUploadAssoc{}, fmt.Errorf("app id is required")
		}
		if !yes {
			ok, err := console.Confirm("Attach this image as the cover of app "+id+"?", false)
			if err != nil {
				return FileUploadAssoc{}, err
			}
			if !ok {
				return FileUploadAssoc{}, errors.New("cancelled")
			}
			yes = true
		}
		return FileUploadAssoc{AppID: id, Yes: yes}, nil
	case 2:
		ok, err := console.Confirm("Upload without attaching (put file_url into create/update JSON later)?", false)
		if err != nil {
			return FileUploadAssoc{}, err
		}
		if !ok {
			return FileUploadAssoc{}, errors.New("cancelled")
		}
		return FileUploadAssoc{Unlinked: true}, nil
	default:
		return FileUploadAssoc{}, errors.New("cancelled")
	}
}
