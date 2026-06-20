package services

import (
	"cdn-api/models"
	"reflect"
	"testing"
)

func TestBuildIssueCAFallbackList_OnlyUsesAutomaticFallbacksThatCanRegisterWithoutEAB(t *testing.T) {
	cases := []struct {
		name    string
		primary string
		want    []string
	}{
		{name: "letsencrypt", primary: "letsencrypt", want: []string{"letsencrypt"}},
		{name: "zerossl", primary: "zerossl", want: []string{"zerossl", "letsencrypt"}},
		{name: "google", primary: "google", want: []string{"google", "letsencrypt"}},
		{name: "buypass legacy", primary: "buypass", want: []string{"buypass", "letsencrypt"}},
		{name: "unknown", primary: "unknown", want: []string{"letsencrypt"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildIssueCAFallbackList(tc.primary); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("fallback list = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestCertDomainMatchesSiteDomain(t *testing.T) {
	cases := []struct {
		name       string
		certDomain string
		siteDomain string
		want       bool
	}{
		{name: "exact", certDomain: "www.example.com", siteDomain: "www.example.com", want: true},
		{name: "wildcard site matches subdomain", certDomain: "www.example.com", siteDomain: "*.example.com", want: true},
		{name: "wildcard site does not match apex", certDomain: "example.com", siteDomain: "*.example.com", want: false},
		{name: "wildcard site requires label boundary", certDomain: "badexample.com", siteDomain: "*.example.com", want: false},
		{name: "wildcard cert matches wildcard site", certDomain: "*.example.com", siteDomain: "*.example.com", want: true},
		{name: "wildcard cert does not match exact site", certDomain: "*.example.com", siteDomain: "www.example.com", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := certDomainMatchesSiteDomain(tc.certDomain, tc.siteDomain); got != tc.want {
				t.Fatalf("certDomainMatchesSiteDomain(%q, %q) = %v, want %v", tc.certDomain, tc.siteDomain, got, tc.want)
			}
		})
	}
}

func TestSiteMatchesCertDomain(t *testing.T) {
	site := models.Site{Domains: []string{"api.example.com", "*.boisconfort235.com"}}
	if !siteMatchesCertDomain(site, "www.boisconfort235.com") {
		t.Fatalf("expected wildcard site domain to match cert domain")
	}
	if siteMatchesCertDomain(site, "boisconfort235.com") {
		t.Fatalf("did not expect wildcard site domain to match apex")
	}
}

func TestIntersectInt64s(t *testing.T) {
	got := intersectInt64s([]int64{3, 1, 2, 3}, []int64{2, 3, 4})
	want := []int64{3, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("intersectInt64s = %#v, want %#v", got, want)
	}
}

func TestIssueResolvedIPsAllowed(t *testing.T) {
	if !issueResolvedIPsAllowed([]string{"144.172.98.93"}, []string{"144.172.98.93", "144.172.98.94"}) {
		t.Fatalf("expected resolved CDN IP to be allowed")
	}
	if issueResolvedIPsAllowed([]string{"144.172.98.93", "61.4.122.239"}, []string{"144.172.98.93"}) {
		t.Fatalf("expected stale non-package IP to be rejected")
	}
	if issueResolvedIPsAllowed(nil, []string{"144.172.98.93"}) {
		t.Fatalf("empty resolved IP list must be rejected")
	}
}

func TestNormalizeIssueIP(t *testing.T) {
	cases := map[string]string{
		"144.172.98.93":        "144.172.98.93",
		"144.172.98.93:80":     "144.172.98.93",
		"2001:db8::1":          "2001:db8::1",
		"not-an-ip":            "",
		"http://144.172.98.93": "",
	}
	for raw, want := range cases {
		if got := normalizeIssueIP(raw); got != want {
			t.Fatalf("normalizeIssueIP(%q) = %q, want %q", raw, got, want)
		}
	}
}
