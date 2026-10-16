package service

import (
	"errors"
	"strings"
	"testing"
)

func TestCloudDeployRequiresYesAndFlags(t *testing.T) {
	s := CloudService{}
	_, err := s.DeployModel(CloudDeployParams{EUI: "e", ModelID: "m", Yes: false})
	if !errors.Is(err, ErrConfirmRequired) {
		t.Fatalf("want ErrConfirmRequired, got %v", err)
	}
	_, err = s.DeployModel(CloudDeployParams{ModelID: "m", Yes: true})
	if err == nil || !strings.Contains(err.Error(), "--eui") {
		t.Fatalf("want --eui required, got %v", err)
	}
	_, err = s.DeployModel(CloudDeployParams{EUI: "e", Yes: true})
	if err == nil || !strings.Contains(err.Error(), "--model-id") {
		t.Fatalf("want --model-id required, got %v", err)
	}
}

func TestCloudOrderRequiresRequestID(t *testing.T) {
	s := CloudService{}
	_, err := s.GetOrderRecord(CloudOrderParams{})
	if err == nil || !strings.Contains(err.Error(), "--request-id") {
		t.Fatalf("want --request-id required, got %v", err)
	}
}
