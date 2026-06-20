package response

import "testing"

func TestFromHTTPStatusMapsConnectionLimit(t *testing.T) {
	if got := FromHTTPStatus(515); got != CodeConnectionLimit {
		t.Fatalf("FromHTTPStatus(515) = %d, want %d", got, CodeConnectionLimit)
	}
}

func TestNormalizeCodeMapsHTTPStatusValues(t *testing.T) {
	tests := []struct {
		name       string
		raw        interface{}
		httpStatus int
		want       int
	}{
		{name: "http 429", raw: 429, httpStatus: 200, want: CodeTooManyRequests},
		{name: "http 515", raw: 515, httpStatus: 200, want: CodeConnectionLimit},
		{name: "empty with http 515", raw: nil, httpStatus: 515, want: CodeConnectionLimit},
	}

	for _, tt := range tests {
		if got := NormalizeCode(tt.raw, tt.httpStatus); got != tt.want {
			t.Fatalf("%s: NormalizeCode(%v, %d) = %d, want %d", tt.name, tt.raw, tt.httpStatus, got, tt.want)
		}
	}
}
