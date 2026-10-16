package service

import "testing"

func TestResolveRecameraProAlgorithm(t *testing.T) {
	tests := []struct {
		name        string
		task        int
		framework   string
		supported   bool
		algorithm   string
		modelFormat string
	}{
		{"detection", 1, "2", true, "yolov8", "rknn (model_format=6)"},
		{"classification", 2, "", true, "classify", "rknn (model_format=6)"},
		{"pose", 4, "2", true, "yolov8_pose", "rknn (model_format=6)"},
		{"pose rejects yolov5", 4, "1", false, "", "rknn (model_format=6)"},
		{"audio", 6, "", true, "", "alpkg (7) or zip (8)"},
		{"tracking", 7, "", true, "bytetrack", "rknn (model_format=6)"},
		{"segmentation hidden", 3, "", false, "", "rknn (model_format=6)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRecameraProAlgorithm(tt.task, tt.framework)
			if got.Supported != tt.supported || got.Algorithm != tt.algorithm || got.ModelFormat != tt.modelFormat {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestRecameraProDetectionFrameworkMatrix(t *testing.T) {
	want := map[string]string{
		"1": "yolov5", "2": "yolov8", "8": "yolov5u", "9": "yolov10",
		"10": "yolo11", "11": "yolo12", "12": "yolo26", "13": "nanodet",
		"14": "yolox", "15": "detr",
	}
	for framework, algorithm := range want {
		got := ResolveRecameraProAlgorithm(RecameraProTaskDetection, framework)
		if !got.Supported || got.Algorithm != algorithm {
			t.Fatalf("framework=%s got %#v", framework, got)
		}
	}
}

func TestNormalizeRecameraProModelPayload(t *testing.T) {
	body := map[string]any{
		"task":         float64(2),
		"model_format": float64(6),
		"ai_framework": "2",
		"file_url":     "https://example.com/a.rknn",
		"precision":    "1",
		"labels": []any{
			map[string]any{"object_id": float64(0), "object_name": "cat"},
		},
	}
	if err := NormalizeRecameraProModelPayload(body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["ai_framework"]; ok {
		t.Fatal("classification must strip ai_framework")
	}

	bad := map[string]any{
		"task": float64(4), "model_format": float64(6), "ai_framework": "1",
		"labels": []any{map[string]any{"object_id": float64(0), "object_name": "a"}},
	}
	if err := NormalizeRecameraProModelPayload(bad); err == nil {
		t.Fatal("pose+yolov5 should fail")
	}
}
