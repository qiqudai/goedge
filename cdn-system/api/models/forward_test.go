package models

import "testing"

func TestDecodeStringList(t *testing.T) {
	got := decodeStringList(`["88","99/udp","8080/tcp"]`)
	want := []string{"88", "99/udp", "8080/tcp"}
	if len(got) != len(want) {
		t.Fatalf("decode json list = %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d = %q want %q", i, got[i], want[i])
		}
	}

	legacy := decodeStringList("88 99/udp,8080")
	if len(legacy) != 3 {
		t.Fatalf("legacy decode = %#v", legacy)
	}
}

func TestDecodeOrigins(t *testing.T) {
	jsonOrigins := decodeOrigins(`[{"address":"1.1.1.1:8080","weight":2,"enable":true}]`)
	if len(jsonOrigins) != 1 || jsonOrigins[0].Address != "1.1.1.1:8080" || jsonOrigins[0].Weight != 2 {
		t.Fatalf("json origins = %#v", jsonOrigins)
	}

	legacy := decodeOrigins("1.1.1.1:8080 8.8.8.8:53")
	if len(legacy) != 2 || legacy[0].Weight != 1 || !legacy[0].Enable {
		t.Fatalf("legacy origins = %#v", legacy)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	f := &Forward{
		ListenPorts: []string{"88", "99/udp"},
		Origins: []ForwardOrigin{
			{Address: "1.1.1.1:8080", Weight: 1, Enable: true},
		},
		Settings: map[string]interface{}{"remark": "test"},
		Remark:   "test",
	}
	if err := f.BeforeSave(nil); err != nil {
		t.Fatalf("BeforeSave: %v", err)
	}
	if f.ListenPortsRaw == "" || f.OriginsRaw == "" {
		t.Fatal("raw fields should be populated")
	}

	out := &Forward{
		ListenPortsRaw: f.ListenPortsRaw,
		OriginsRaw:     f.OriginsRaw,
		SettingsRaw:    f.SettingsRaw,
	}
	if err := out.AfterFind(nil); err != nil {
		t.Fatalf("AfterFind: %v", err)
	}
	if len(out.ListenPorts) != 2 || out.Remark != "test" {
		t.Fatalf("round trip = %#v", out)
	}
}
