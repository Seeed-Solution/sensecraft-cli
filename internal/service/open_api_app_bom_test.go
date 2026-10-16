package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFilterMaterialOptions(t *testing.T) {
	raw := []byte(`{
		"material_options": [
			{"sku":"113991254","name":"XIAO ESP32S3 Sense","classify":"XIAO","price":"15.99","pic_url":"https://example/x.png"},
			{"sku":"110061361","name":"reCamera 2002","classify":"reCamera","price":"99.00"}
		]
	}`)
	out, err := filterMaterialOptions(raw, "xiao")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if int(m["count"].(float64)) != 1 {
		t.Fatalf("count=%v", m["count"])
	}
	list := m["list"].([]any)
	item := list[0].(map[string]any)
	if item["sku"] != "113991254" {
		t.Fatalf("sku=%v", item["sku"])
	}
	if item["price"] != "15.99" || item["pic_url"] == "" {
		t.Fatalf("price/pic_url dropped: %v", item)
	}

	out, err = filterMaterialOptions(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if int(m["count"].(float64)) != 2 {
		t.Fatalf("empty keyword count=%v", m["count"])
	}
}

func TestSandboxBOMCatalogIncludesPicURL(t *testing.T) {
	raw := sandboxBOMCatalog("recamera")
	if !strings.Contains(string(raw), `"pic_url"`) {
		t.Fatalf("sandbox catalog missing pic_url: %s", raw)
	}
	if !strings.Contains(string(raw), `"price"`) {
		t.Fatalf("sandbox catalog missing price: %s", raw)
	}
}

func TestBOMSetRequiresYes(t *testing.T) {
	s := AppService{}
	_, err := s.SetBOM(AppBOMSetParams{
		AppID: "a1", FromJSON: `{"materials":[]}`, Yes: false,
	})
	if err == nil {
		t.Fatal("expected confirmation required")
	}
}
