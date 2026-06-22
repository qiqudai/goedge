package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeoCountryFromIP2RegionSuffixCodes(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	luaDir := filepath.Join(root, "assets", "lua")
	xdb := filepath.Join(root, "assets", "data", "ip2region.xdb")
	if _, err := os.Stat(xdb); err != nil {
		t.Skip("ip2region.xdb missing")
	}

	script := `
package.path = "/lua/?.lua;/lua/?/init.lua;;"
local geo = require "geo_country"
local ip2region = require "resty.ip2region.ip2region"
local searcher = ip2region.new({file="/data/ip2region.xdb"})
local cases = {
  {ip="183.182.115.65", want="LA"},
  {ip="202.155.141.95", want="ID"},
  {ip="1.1.1.1", want="AU"},
}
for _, c in ipairs(cases) do
  local raw = searcher:search(c.ip)
  local got = geo.from_ip2region(tostring(raw or ""))
  if got ~= c.want then
    io.write(string.format("FAIL ip=%s raw=%s got=%s want=%s\n", c.ip, tostring(raw), got, c.want))
    os.exit(1)
  end
end
print("ok")
`
	cmd := exec.Command("docker", "run", "--rm",
		"-v", luaDir+":/lua:ro",
		"-v", xdb+":/data/ip2region.xdb:ro",
		"openresty/openresty:1.25.3.1-0-alpine-fat",
		"resty", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			t.Skip("docker unavailable")
		}
		t.Fatalf("geo_country lua test failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("unexpected output: %s", out)
	}
}
