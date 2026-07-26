package controllers

import (
	"cdn-api/models"
	"testing"
)

func TestComputeSiteCnameFields(t *testing.T) {
	pkg := models.UserPackage{
		CnameDomain:   "7plvip.com",
		CnameHostname: "package-entry",
	}

	prefix, root := computeSiteCnameFields(models.Site{
		CnameMode: "domain",
		Domains:   []string{"Api.Example.com"},
	}, pkg, nil, nil, nil)
	if prefix != "api.example.com" || root != "7plvip.com" {
		t.Fatalf("domain mode = (%q, %q)", prefix, root)
	}

	prefix, root = computeSiteCnameFields(models.Site{CnameMode: "package"}, pkg, nil, nil, nil)
	if prefix != "package-entry" || root != "7plvip.com" {
		t.Fatalf("package mode = (%q, %q)", prefix, root)
	}

	overrideRoot := "new-root.example"
	prefix, root = computeSiteCnameFields(models.Site{CnameMode: "package"}, pkg, nil, nil, &overrideRoot)
	if prefix != "package-entry" || root != "new-root.example" {
		t.Fatalf("package root override = (%q, %q)", prefix, root)
	}
}
