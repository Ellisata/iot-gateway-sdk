// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package driver

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDiscoveryTransportJSONRoundTrip Transport 必须原样穿过 describe 的 JSON，
// 零值则完全不出现在报文里 —— 旧网关/旧插件正是靠这一点保持兼容。
func TestDiscoveryTransportJSONRoundTrip(t *testing.T) {
	for _, want := range []ProbeTransport{ProbeTransportTCP, ProbeTransportUDP} {
		raw, err := json.Marshal(DiscoverySpec{Title: "扫描", Transport: want})
		if err != nil {
			t.Fatalf("marshal %q: %v", want, err)
		}
		if !strings.Contains(string(raw), `"transport":"`+string(want)+`"`) {
			t.Fatalf("transport %q 未出现在 JSON 里: %s", want, raw)
		}
		var out DiscoverySpec
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal %q: %v", want, err)
		}
		if out.Transport != want {
			t.Fatalf("Transport 往返后 = %q，期望 %q", out.Transport, want)
		}
	}
}

// TestDiscoveryTransportOmitted 未声明传输层时不得输出该键：旧插件不认识它，
// 而 omitempty 是"新增可选字段不递增 APIVersion"这条约定的一部分。
func TestDiscoveryTransportOmitted(t *testing.T) {
	raw, err := json.Marshal(DiscoverySpec{Title: "扫描"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "transport") {
		t.Fatalf("零值 Transport 必须省略，实际 %s", raw)
	}
}

// TestDiscoveryTransportOldPayload 旧插件（不带本字段）的 describe 报文必须
// 解析为零值，且网关据此跳过预筛。
func TestDiscoveryTransportOldPayload(t *testing.T) {
	var out DiscoverySpec
	if err := json.Unmarshal([]byte(`{"title":"扫描","defaultPort":502,"maxBudgetMs":6000}`), &out); err != nil {
		t.Fatalf("unmarshal old payload: %v", err)
	}
	if out.Transport != ProbeTransportUnknown {
		t.Fatalf("旧报文 Transport = %q，期望空串", out.Transport)
	}
}
