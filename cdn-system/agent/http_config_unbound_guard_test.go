package main

import (
	"strings"
	"testing"
)

func TestWriteDefaultServer_InjectsUnboundGuardFor418(t *testing.T) {
	var b strings.Builder
	writeDefaultServer(&b, "80", false, errorPageContext{}, 418, false)
	out := b.String()
	if !strings.Contains(out, "content_by_lua_block {") || !strings.Contains(out, "guard.enforce(418)") {
		t.Fatalf("expected unbound guard lua hook for 418 default server, got: %s", out)
	}
}

func TestWriteDefaultServer_DoesNotInjectUnboundGuardFor404(t *testing.T) {
	var b strings.Builder
	writeDefaultServer(&b, "80", false, errorPageContext{}, 404, false)
	out := b.String()
	if strings.Contains(out, "guard.enforce(418)") || strings.Contains(out, "content_by_lua_block {") {
		t.Fatalf("did not expect unbound guard lua hook for non-418 default server, got: %s", out)
	}
}
