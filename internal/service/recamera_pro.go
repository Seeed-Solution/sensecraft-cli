package service

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	RecameraProUniformType = 53
	RecameraProDeviceType  = 53

	RecameraProTaskDetection      = 1
	RecameraProTaskClassification = 2
	RecameraProTaskSegmentation   = 3
	RecameraProTaskPose           = 4
	RecameraProTaskGeneration     = 5
	RecameraProTaskAudio          = 6
	RecameraProTaskTracking       = 7

	RecameraProModelFormatRKNN  = 6
	RecameraProModelFormatALPKG = 7
	RecameraProModelFormatZIP   = 8
)

var recameraProDetectionAlgorithms = map[int]string{
	1:  "yolov5",
	2:  "yolov8",
	8:  "yolov5u",
	9:  "yolov10",
	10: "yolo11",
	11: "yolo12",
	12: "yolo26",
	13: "nanodet",
	14: "yolox",
	15: "detr",
}

type RecameraProAlgorithmResult struct {
	UniformType int    `json:"uniform_type"`
	Task        int    `json:"task"`
	AIFramework string `json:"ai_framework,omitempty"`
	Algorithm   string `json:"algorithm,omitempty"`
	Supported   bool   `json:"supported"`
	ModelFormat string `json:"model_format"`
	FileAccept  string `json:"file_accept"`
	DeployMode  string `json:"deploy_mode"`
	Reason      string `json:"reason,omitempty"`
}

// ResolveRecameraProAlgorithm mirrors Console recameraProAlgorithm.js.
// reCamera Pro deployment is local device IP/USB, not aiserver update_stream.
func ResolveRecameraProAlgorithm(task int, aiFramework string) RecameraProAlgorithmResult {
	result := RecameraProAlgorithmResult{
		UniformType: RecameraProUniformType,
		Task:        task,
		AIFramework: strings.TrimSpace(aiFramework),
		Supported:   true,
		ModelFormat: "rknn (model_format=6)",
		FileAccept:  ".rknn",
		DeployMode:  "local-ip-or-usb",
	}
	switch task {
	case RecameraProTaskDetection:
		id, ok := parseRecameraProFramework(aiFramework)
		if !ok {
			return unsupportedRecameraPro(result, "detection requires a supported ai_framework id")
		}
		algorithm, ok := recameraProDetectionAlgorithms[id]
		if !ok {
			return unsupportedRecameraPro(result, "unsupported detection ai_framework")
		}
		result.Algorithm = algorithm
	case RecameraProTaskClassification:
		result.AIFramework = ""
		result.Algorithm = "classify"
	case RecameraProTaskPose:
		id, ok := parseRecameraProFramework(aiFramework)
		if !ok || id != 2 {
			return unsupportedRecameraPro(result, "pose only supports YOLOV8 (ai_framework=2)")
		}
		result.Algorithm = "yolov8_pose"
	case RecameraProTaskAudio:
		result.AIFramework = ""
		result.ModelFormat = "alpkg (7) or zip (8)"
		result.FileAccept = ".alpkg or .zip"
	case RecameraProTaskTracking:
		result.AIFramework = ""
		result.Algorithm = "bytetrack"
	case RecameraProTaskSegmentation, RecameraProTaskGeneration:
		return unsupportedRecameraPro(result, "task is hidden for reCamera Pro upload/deploy")
	default:
		return unsupportedRecameraPro(result, "unknown model task")
	}
	return result
}

func parseRecameraProFramework(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	id, err := strconv.Atoi(value)
	return id, err == nil
}

func unsupportedRecameraPro(result RecameraProAlgorithmResult, reason string) RecameraProAlgorithmResult {
	result.Supported = false
	result.Reason = reason
	result.Algorithm = ""
	return result
}

func ValidateRecameraProAlgorithm(task int, aiFramework string) (RecameraProAlgorithmResult, error) {
	result := ResolveRecameraProAlgorithm(task, aiFramework)
	if !result.Supported {
		return result, fmt.Errorf("unsupported reCamera Pro model: %s", result.Reason)
	}
	return result, nil
}

// NormalizeRecameraProModelPayload validates and normalizes a create/update body for Pro.
// It strips empty ai_framework and deletes the key for classification/audio/tracking.
func NormalizeRecameraProModelPayload(body map[string]any) error {
	if body == nil {
		return fmt.Errorf("model JSON is empty")
	}
	task, ok := asInt(body["task"])
	if !ok {
		return fmt.Errorf("reCamera Pro requires task (model_task_array id)")
	}
	fw := stringFromMap(body, "ai_framework")
	algo, err := ValidateRecameraProAlgorithm(task, fw)
	if err != nil {
		return err
	}

	format, hasFormat := asInt(body["model_format"])
	if !hasFormat {
		return fmt.Errorf("reCamera Pro requires model_format")
	}
	switch task {
	case RecameraProTaskAudio:
		if format != RecameraProModelFormatALPKG && format != RecameraProModelFormatZIP {
			return fmt.Errorf("reCamera Pro audio requires model_format 7 (alpkg) or 8 (zip)")
		}
	default:
		if format != RecameraProModelFormatRKNN {
			return fmt.Errorf("reCamera Pro non-audio tasks require model_format 6 (rknn)")
		}
	}

	if fileURL := stringFromMap(body, "file_url"); fileURL != "" {
		ext := strings.ToLower(filepath.Ext(strings.Split(fileURL, "?")[0]))
		switch task {
		case RecameraProTaskAudio:
			if format == RecameraProModelFormatALPKG && ext != ".alpkg" {
				return fmt.Errorf("model_format 7 requires .alpkg file_url")
			}
			if format == RecameraProModelFormatZIP && ext != ".zip" {
				return fmt.Errorf("model_format 8 requires .zip file_url")
			}
		default:
			if ext != ".rknn" {
				return fmt.Errorf("reCamera Pro vision tasks require .rknn file_url")
			}
		}
	}

	if prec := stringFromMap(body, "precision"); prec != "" {
		switch task {
		case RecameraProTaskAudio:
			if prec != "3" {
				return fmt.Errorf("reCamera Pro audio precision should be 3")
			}
		default:
			if prec != "1" {
				return fmt.Errorf("reCamera Pro vision precision should be 1")
			}
		}
	}

	if err := validateModelLabels(body["labels"]); err != nil {
		return err
	}

	// Console deletes ai_framework for Classification/Audio/Tracking (and empty values).
	switch task {
	case RecameraProTaskClassification, RecameraProTaskAudio, RecameraProTaskTracking:
		delete(body, "ai_framework")
	default:
		if algo.AIFramework == "" {
			delete(body, "ai_framework")
		} else {
			body["ai_framework"] = algo.AIFramework
		}
	}
	return nil
}

func IsRecameraProTarget(target string) bool {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "recamera-pro", "recamerapro", "pro", "53":
		return true
	default:
		return false
	}
}
