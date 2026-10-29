// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSpecI18nRoundTrip 扫描能力的可选译文必须原样穿过 describe 的 JSON。
// 缺省字段保持为 nil，旧插件/旧网关的兼容性依赖这一点。
func TestSpecI18nRoundTrip(t *testing.T) {
	in := DiscoverySpec{
		Title:           "扫描",
		Description:     "说明",
		TitleI18n:       map[string]string{"en": "Scan"},
		DescriptionI18n: map[string]string{"en": "Description"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out DiscoverySpec
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.TitleI18n["en"] != "Scan" || out.DescriptionI18n["en"] != "Description" {
		t.Fatalf("i18n maps lost in round trip: %+v", out)
	}

	raw, err = json.Marshal(DiscoverySpec{Title: "扫描"})
	if err != nil {
		t.Fatalf("marshal without i18n: %v", err)
	}
	if strings.Contains(string(raw), "titleI18n") || strings.Contains(string(raw), "descriptionI18n") {
		t.Fatalf("nil i18n maps must be omitted, got %s", raw)
	}
}

// TestPointDiscoverySpecI18nRoundTrip 点位扫描规格同理。
func TestPointDiscoverySpecI18nRoundTrip(t *testing.T) {
	in := PointDiscoverySpec{
		Title:           "枚举",
		Description:     "说明",
		TitleI18n:       map[string]string{"en": "Enumerate"},
		DescriptionI18n: map[string]string{"en": "Description"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out PointDiscoverySpec
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.TitleI18n["en"] != "Enumerate" || out.DescriptionI18n["en"] != "Description" {
		t.Fatalf("i18n maps lost in round trip: %+v", out)
	}
}
