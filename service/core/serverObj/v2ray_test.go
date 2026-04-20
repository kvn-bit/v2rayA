package serverObj

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

func loadXHTTPFixture(t *testing.T, name string) map[string]interface{} {
	t.Helper()
	path := filepath.Join("..", "..", "..", "tmp", "test-configs", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	var m map[string]interface{}
	if err = jsoniter.Unmarshal(b, &m); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", path, err)
	}
	return m
}

func marshalJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := jsoniter.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return string(b)
}

func assertJSONEqual(t *testing.T, got string, want interface{}) {
	t.Helper()
	var gotValue interface{}
	if err := jsoniter.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("failed to unmarshal got json: %v", err)
	}
	wantJSON := marshalJSON(t, want)
	var wantValue interface{}
	if err := jsoniter.Unmarshal([]byte(wantJSON), &wantValue); err != nil {
		t.Fatalf("failed to unmarshal want json: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("unexpected json\nwant: %s\ngot:  %s", wantJSON, got)
	}
}

func passthroughJSONFromXHTTPFixture(t *testing.T, name string) string {
	t.Helper()
	m := loadXHTTPFixture(t, name)
	delete(m, "path")
	delete(m, "host")
	delete(m, "mode")
	return marshalJSON(t, m)
}

func rawJSONFromXHTTPFixture(t *testing.T, name string) string {
	t.Helper()
	return marshalJSON(t, loadXHTTPFixture(t, name))
}

func configuredXHTTPSettingsJSON(t *testing.T, v *V2Ray) string {
	t.Helper()
	cfg, err := v.Configuration(PriorInfo{Tag: "proxy"})
	if err != nil {
		t.Fatalf("failed to generate configuration: %v", err)
	}
	return marshalJSON(t, cfg.CoreOutbound.StreamSettings.XHTTPSettings)
}

func TestV2RayXHTTPConfigurationWithoutRawExtras(t *testing.T) {
	v := &V2Ray{
		Add:       "proxy.example.com",
		Port:      "443",
		ID:        "11111111-1111-1111-1111-111111111111",
		Net:       "xhttp",
		Path:      "/",
		Host:      "proxy.example.com",
		XHTTPMode: "auto",
		TLS:       "none",
		Protocol:  "vless",
	}

	got := configuredXHTTPSettingsJSON(t, v)
	want := map[string]interface{}{
		"path": "/",
		"host": "proxy.example.com",
		"mode": "auto",
	}
	assertJSONEqual(t, got, want)
}

func TestV2RayXHTTPConfigurationWithDownloadSettingsExtraPayload(t *testing.T) {
	v := &V2Ray{
		Add:          "proxy.example.com",
		Port:         "443",
		ID:           "11111111-1111-1111-1111-111111111111",
		Net:          "xhttp",
		Path:         "/",
		Host:         "proxy.example.com",
		XHTTPMode:    "packet-up",
		XHTTPRawJson: rawJSONFromXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
		TLS:          "reality",
		SNI:          "proxy.example.com",
		PublicKey:    "FAKE_PUBLIC_KEY",
		ShortId:      "0123456789ab",
		SpiderX:      "/",
		Fingerprint:  "chrome",
		Protocol:     "vless",
	}

	got := configuredXHTTPSettingsJSON(t, v)
	assertJSONEqual(t, got, map[string]interface{}{
		"path":  "/",
		"host":  "proxy.example.com",
		"mode":  "packet-up",
		"extra": loadXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
	})
}

func TestV2RayXHTTPRoundtripPreservesRawExtras(t *testing.T) {
	rawJSON := rawJSONFromXHTTPFixture(t, "xhttp-extra-with-downloads.json")
	original := &V2Ray{
		Ps:           "xhttp-node",
		Add:          "proxy.example.com",
		Port:         "443",
		ID:           "11111111-1111-1111-1111-111111111111",
		Net:          "xhttp",
		Path:         "/",
		Host:         "proxy.example.com",
		XHTTPMode:    "packet-up",
		XHTTPRawJson: rawJSON,
		TLS:          "reality",
		SNI:          "proxy.example.com",
		PublicKey:    "FAKE_PUBLIC_KEY",
		ShortId:      "0123456789ab",
		SpiderX:      "/",
		Fingerprint:  "chrome",
		Protocol:     "vless",
	}

	link := original.ExportToURL()
	parsedObj, err := ParseVlessURL(link)
	if err != nil {
		t.Fatalf("failed to parse exported url: %v", err)
	}
	if parsedObj.XHTTPRawJson != rawJSON {
		t.Fatalf("xhttpRawJson changed after roundtrip\nwant: %s\ngot:  %s", rawJSON, parsedObj.XHTTPRawJson)
	}

	got := configuredXHTTPSettingsJSON(t, parsedObj)
	assertJSONEqual(t, got, map[string]interface{}{
		"path":  "/",
		"host":  "proxy.example.com",
		"mode":  "packet-up",
		"extra": loadXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
	})
}

func TestParseVlessURLXHTTPExtraAliasHydratesRawJSON(t *testing.T) {
	rawJSON := rawJSONFromXHTTPFixture(t, "xhttp-extra-with-downloads.json")
	query := url.Values{
		"type":      []string{"xhttp"},
		"security":  []string{"reality"},
		"path":      []string{"/"},
		"host":      []string{"proxy.example.com"},
		"sni":       []string{"proxy.example.com"},
		"fp":        []string{"chrome"},
		"pbk":       []string{"FAKE_PUBLIC_KEY"},
		"sid":       []string{"0123456789ab"},
		"spx":       []string{"/"},
		"xhttpMode": []string{"packet-up"},
		"extra":     []string{rawJSON},
	}
	link := (&url.URL{
		Scheme:   "vless",
		User:     url.User("11111111-1111-1111-1111-111111111111"),
		Host:     "proxy.example.com:443",
		RawQuery: query.Encode(),
		Fragment: "xhttp-node",
	}).String()

	parsedObj, err := ParseVlessURL(link)
	if err != nil {
		t.Fatalf("failed to parse vless url with extra alias: %v", err)
	}
	if parsedObj.XHTTPRawJson != rawJSON {
		t.Fatalf("extra alias did not hydrate xhttpRawJson\nwant: %s\ngot:  %s", rawJSON, parsedObj.XHTTPRawJson)
	}
	if parsedObj.XHTTPMode != "packet-up" {
		t.Fatalf("nested mode did not resolve outer xhttp mode\nwant: %s\ngot:  %s", "packet-up", parsedObj.XHTTPMode)
	}

	got := configuredXHTTPSettingsJSON(t, parsedObj)
	assertJSONEqual(t, got, map[string]interface{}{
		"path":  "/",
		"host":  "proxy.example.com",
		"mode":  "packet-up",
		"extra": loadXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
	})
}

func TestV2RayXHTTPConfigurationRetainsFullObjectRawJSON(t *testing.T) {
	v := &V2Ray{
		Add:          "proxy.example.com",
		Port:         "443",
		ID:           "11111111-1111-1111-1111-111111111111",
		Net:          "xhttp",
		Path:         "/",
		Host:         "proxy.example.com",
		XHTTPMode:    "packet-up",
		XHTTPRawJson: rawJSONFromXHTTPFixture(t, "xhttp-settings-with-downloads.json"),
		TLS:          "reality",
		SNI:          "proxy.example.com",
		PublicKey:    "FAKE_PUBLIC_KEY",
		ShortId:      "0123456789ab",
		SpiderX:      "/",
		Fingerprint:  "chrome",
		Protocol:     "vless",
	}

	got := configuredXHTTPSettingsJSON(t, v)
	assertJSONEqual(t, got, loadXHTTPFixture(t, "xhttp-settings-with-downloads.json"))
}

func TestV2RayXHTTPConfigurationDerivesOuterModeFromNestedDownloadSettings(t *testing.T) {
	v := &V2Ray{
		Add:          "proxy.example.com",
		Port:         "443",
		ID:           "11111111-1111-1111-1111-111111111111",
		Net:          "xhttp",
		Path:         "/",
		Host:         "proxy.example.com",
		XHTTPMode:    "auto",
		XHTTPRawJson: rawJSONFromXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
		TLS:          "reality",
		SNI:          "proxy.example.com",
		PublicKey:    "FAKE_PUBLIC_KEY",
		ShortId:      "0123456789ab",
		SpiderX:      "/",
		Fingerprint:  "chrome",
		Protocol:     "vless",
	}

	got := configuredXHTTPSettingsJSON(t, v)
	assertJSONEqual(t, got, map[string]interface{}{
		"path":  "/",
		"host":  "proxy.example.com",
		"mode":  "packet-up",
		"extra": loadXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
	})
}

func TestV2RayXHTTPConfigurationKeepsExplicitModeOverNestedMode(t *testing.T) {
	v := &V2Ray{
		Add:          "proxy.example.com",
		Port:         "443",
		ID:           "11111111-1111-1111-1111-111111111111",
		Net:          "xhttp",
		Path:         "/",
		Host:         "proxy.example.com",
		XHTTPMode:    "stream-up",
		XHTTPRawJson: rawJSONFromXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
		TLS:          "reality",
		SNI:          "proxy.example.com",
		PublicKey:    "FAKE_PUBLIC_KEY",
		ShortId:      "0123456789ab",
		SpiderX:      "/",
		Fingerprint:  "chrome",
		Protocol:     "vless",
	}

	got := configuredXHTTPSettingsJSON(t, v)
	assertJSONEqual(t, got, map[string]interface{}{
		"path":  "/",
		"host":  "proxy.example.com",
		"mode":  "stream-up",
		"extra": loadXHTTPFixture(t, "xhttp-extra-with-downloads.json"),
	})
}

func TestV2RayXHTTPConfigurationKeepsAutoWhenNestedModeMissing(t *testing.T) {
	v := &V2Ray{
		Add:          "proxy.example.com",
		Port:         "443",
		ID:           "11111111-1111-1111-1111-111111111111",
		Net:          "xhttp",
		Path:         "/",
		Host:         "proxy.example.com",
		XHTTPMode:    "auto",
		XHTTPRawJson: `{"downloadSettings":{"address":"download.example.com","network":"xhttp","port":443}}`,
		TLS:          "reality",
		SNI:          "proxy.example.com",
		PublicKey:    "FAKE_PUBLIC_KEY",
		ShortId:      "0123456789ab",
		SpiderX:      "/",
		Fingerprint:  "chrome",
		Protocol:     "vless",
	}

	got := configuredXHTTPSettingsJSON(t, v)
	assertJSONEqual(t, got, map[string]interface{}{
		"path": "/",
		"host": "proxy.example.com",
		"mode": "auto",
		"extra": map[string]interface{}{
			"downloadSettings": map[string]interface{}{
				"address": "download.example.com",
				"network": "xhttp",
				"port":    float64(443),
			},
		},
	})
}
