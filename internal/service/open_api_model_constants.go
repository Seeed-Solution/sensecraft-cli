package service

import (
	"encoding/json"
	"errors"

	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

func (s ModelService) Constants() (json.RawMessage, error) {
	data, err := s.client.Get(enum.AIApiPathGetConstant, nil)
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			raw, _ := json.Marshal(map[string]any{
				"model_task_array": []map[string]any{
					{"id": "1", "name": "Detection"},
					{"id": "2", "name": "Classification"},
					{"id": "4", "name": "Pose"},
					{"id": "6", "name": "Audio Classification"},
					{"id": "7", "name": "Tracking"},
				},
				"model_format_array": []map[string]any{
					{"id": "1", "name": "onnx"},
					{"id": "2", "name": "tflite"},
					{"id": "3", "name": "engine"},
					{"id": "4", "name": "pt"},
					{"id": "5", "name": "cvimodel"},
					{"id": "6", "name": "rknn"},
					{"id": "7", "name": "alpkg"},
					{"id": "8", "name": "zip"},
				},
				"ai_framework_array": []map[string]any{
					{"id": "1", "name": "YOLOV5"},
					{"id": "2", "name": "YOLOV8"},
					{"id": "8", "name": "YOLOV5U"},
					{"id": "9", "name": "YOLOV10"},
					{"id": "10", "name": "YOLO11"},
					{"id": "11", "name": "YOLO12"},
					{"id": "12", "name": "YOLO26"},
					{"id": "13", "name": "NanoDet"},
					{"id": "14", "name": "YOLOX"},
					{"id": "15", "name": "DETR"},
				},
				"app_scenarios_array": []map[string]any{
					{"id": "1", "name": "Smart Camera"},
					{"id": "5", "name": "Other"},
				},
				"scenarios_array": []map[string]any{
					{"id": "1", "name": "General"},
				},
				"model_precision_array": []map[string]any{
					{"id": "1", "name": "INT8"},
					{"id": "2", "name": "FLOAT 16"},
					{"id": "3", "name": "FLOAT 32"},
				},
				"device_type_array": []map[string]any{
					{"id": "sandbox-xiao", "name": "XIAO", "uniform_type": 32},
					{"id": "sandbox-vision", "name": "Grove Vision", "uniform_type": 36},
					{"id": "sandbox-jetson", "name": "Jetson", "uniform_type": 26},
					{"id": "sandbox-recamera", "name": "reCamera", "uniform_type": 40},
					{"id": "sandbox-pro", "name": "reCamera Pro", "uniform_type": 53},
				},
			})
			return raw, nil
		}
		return nil, err
	}
	return data, nil
}
