package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestFilterMaterialOptionsTokens(t *testing.T) {
	raw := []byte(`{"material_options":[
		{"sku":"100069905","name":"XIAO 1.47\" Touch Display (ESP32-S3)","block_support":"\u662f","price":"29.90"},
		{"sku":"other","name":"Touch Display","block_support":"\u5426","note":"ESP32S3","classify":"XIAO"}
	]}`)
	for _, tt := range []struct {
		query string
		want  []string
	}{
		{"XIAO ESP32-S3 1.47", []string{"100069905"}},
		{"1.47 Touch", []string{"100069905"}},
		{"Touch Display ESP32-S3", []string{"100069905", "other"}},
		{"esp32s3", []string{"100069905", "other"}},
		{"(ESP32-S3) 1.47\"", []string{"100069905"}},
		{"100069905", []string{"100069905"}},
		{"XIAO other", []string{"other"}},
		{"1.47 other", []string{}},
		{"Touch missing", []string{}},
		{"Touch", []string{"100069905", "other"}},
		{"   ", []string{"100069905", "other"}},
	} {
		t.Run(tt.query, func(t *testing.T) {
			out, err := filterMaterialOptions(raw, tt.query)
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Count int              `json:"count"`
				List  []map[string]any `json:"list"`
			}
			if err := json.Unmarshal(out, &result); err != nil {
				t.Fatal(err)
			}
			skus := make([]string, 0)
			for _, item := range result.List {
				skus = append(skus, item["sku"].(string))
				if item["block_supported"] != (item["sku"] == "100069905") {
					t.Fatalf("unexpected support flag: %#v", item)
				}
				if item["sku"] == "100069905" && (item["block_support"] != "\u662f" || item["price"] != "29.90") {
					t.Fatal("catalog fields were changed")
				}
			}
			if !reflect.DeepEqual(skus, tt.want) || result.Count != len(tt.want) {
				t.Fatalf("got %v count=%d, want %v", skus, result.Count, tt.want)
			}
		})
	}
}

func TestFilterMaterialOptions(t *testing.T) {
	raw := []byte(`{
		"material_options": [
			{"sku":"113991115","name":"XIAO ESP32S3 Sense","classify":"XIAO","price":"15.99","pic_url":"https://example/x.png"},
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
	if item["sku"] != "113991115" {
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

func TestBOMQuantityRange(t *testing.T) {
	if err := validateAppMaterials([]any{map[string]any{"quantity": "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := validateAppMaterials([]any{map[string]any{"quantity": "999"}}); err != nil {
		t.Fatal(err)
	}
	for _, quantity := range []any{"1000", "0", "01", 1} {
		if err := validateAppMaterials([]any{map[string]any{"quantity": quantity}}); err == nil {
			t.Fatalf("accepted quantity %#v", quantity)
		}
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
