package main

import (
	"strings"
	"testing"
)

type parentFetchDomain struct {
	UpstreamKey         string
	ParentFetchMode     string
	ParentL1UpstreamKey string
	ParentL2UpstreamKey string
	L1RespectL2         bool
	UseL2               bool
	L2UpstreamKey       string
}

type upstreamTarget struct {
	Addr   string
	NodeID int64
}

func filterOnlineTargets(targets []upstreamTarget, status map[int64]bool) []upstreamTarget {
	if len(targets) == 0 {
		return targets
	}
	hasStatus := false
	out := make([]upstreamTarget, 0, len(targets))
	for _, t := range targets {
		if t.NodeID == 0 {
			out = append(out, t)
			continue
		}
		online, ok := status[t.NodeID]
		if ok {
			hasStatus = true
			if online {
				out = append(out, t)
			}
			continue
		}
		out = append(out, t)
	}
	if hasStatus {
		return out
	}
	return targets
}

func hasOnlineTargets(key string, upstreams map[string][]upstreamTarget, status map[int64]bool) bool {
	targets, ok := upstreams[key]
	if !ok || len(targets) == 0 {
		return false
	}
	return len(filterOnlineTargets(targets, status)) > 0
}

func resolveL3Upstream(domain parentFetchDomain, upstreams map[string][]upstreamTarget, l1Status, l2Status map[int64]bool) (string, string) {
	mode := strings.ToLower(strings.TrimSpace(domain.ParentFetchMode))
	if mode == "" || mode == "origin" {
		return domain.UpstreamKey, "origin"
	}
	if mode == "l1" {
		if domain.ParentL1UpstreamKey != "" && hasOnlineTargets(domain.ParentL1UpstreamKey, upstreams, l1Status) {
			return domain.ParentL1UpstreamKey, "parent"
		}
		if domain.ParentL2UpstreamKey != "" && hasOnlineTargets(domain.ParentL2UpstreamKey, upstreams, l2Status) {
			return domain.ParentL2UpstreamKey, "parent"
		}
		return domain.UpstreamKey, "origin"
	}
	if mode == "l2" {
		if domain.ParentL2UpstreamKey != "" && hasOnlineTargets(domain.ParentL2UpstreamKey, upstreams, l2Status) {
			return domain.ParentL2UpstreamKey, "parent"
		}
		return domain.UpstreamKey, "origin"
	}
	return domain.UpstreamKey, "origin"
}

func resolveL1UseL2(domain parentFetchDomain, upstreams map[string][]upstreamTarget, l2Status map[int64]bool, skipL2 bool) bool {
	if skipL2 || !domain.UseL2 || domain.L2UpstreamKey == "" {
		return false
	}
	return hasOnlineTargets(domain.L2UpstreamKey, upstreams, l2Status)
}

func TestResolveL3UpstreamOriginDirect(t *testing.T) {
	domain := parentFetchDomain{
		UpstreamKey:     "upstream_1",
		ParentFetchMode: "origin",
	}
	key, layer := resolveL3Upstream(domain, nil, nil, nil)
	if key != "upstream_1" || layer != "origin" {
		t.Fatalf("got key=%s layer=%s", key, layer)
	}
}

func TestResolveL3UpstreamL1WithFailover(t *testing.T) {
	upstreams := map[string][]upstreamTarget{
		"l1_upstream_5": {{Addr: "1.1.1.1", NodeID: 41}},
		"l2_upstream_5": {{Addr: "2.2.2.2", NodeID: 56}},
		"upstream_1":  {{Addr: "9.9.9.9"}},
	}
	domain := parentFetchDomain{
		UpstreamKey:         "upstream_1",
		ParentFetchMode:     "l1",
		ParentL1UpstreamKey: "l1_upstream_5",
		ParentL2UpstreamKey: "l2_upstream_5",
	}
	key, layer := resolveL3Upstream(domain, upstreams, map[int64]bool{41: false}, map[int64]bool{56: true})
	if key != "l2_upstream_5" || layer != "parent" {
		t.Fatalf("expected l2 failover, got key=%s layer=%s", key, layer)
	}
	key, layer = resolveL3Upstream(domain, upstreams, map[int64]bool{41: false}, map[int64]bool{56: false})
	if key != "upstream_1" || layer != "origin" {
		t.Fatalf("expected origin fallback, got key=%s layer=%s", key, layer)
	}
}

func TestResolveL3UpstreamL2Offline(t *testing.T) {
	upstreams := map[string][]upstreamTarget{
		"l2_upstream_5": {{Addr: "2.2.2.2", NodeID: 56}},
		"upstream_1":  {{Addr: "9.9.9.9"}},
	}
	domain := parentFetchDomain{
		UpstreamKey:         "upstream_1",
		ParentFetchMode:     "l2",
		ParentL2UpstreamKey: "l2_upstream_5",
	}
	key, layer := resolveL3Upstream(domain, upstreams, nil, map[int64]bool{56: false})
	if key != "upstream_1" || layer != "origin" {
		t.Fatalf("expected origin fallback, got key=%s layer=%s", key, layer)
	}
}

func TestResolveL1UseL2Failover(t *testing.T) {
	upstreams := map[string][]upstreamTarget{
		"l2_upstream_5": {{Addr: "2.2.2.2", NodeID: 56}},
	}
	d := parentFetchDomain{UseL2: true, L2UpstreamKey: "l2_upstream_5"}
	if !resolveL1UseL2(d, upstreams, map[int64]bool{56: true}, false) {
		t.Fatal("expected use l2 when healthy")
	}
	if resolveL1UseL2(d, upstreams, map[int64]bool{56: false}, false) {
		t.Fatal("expected skip l2 when offline")
	}
	if resolveL1UseL2(d, upstreams, map[int64]bool{56: true}, true) {
		t.Fatal("expected skip l2 when header set")
	}
}

func TestParentFetchScenarioMatrix(t *testing.T) {
	upstreams := map[string][]upstreamTarget{
		"upstream_1":    {{Addr: "9.9.9.9"}},
		"l1_upstream_5": {{Addr: "1.1.1.1", NodeID: 41}},
		"l2_upstream_5": {{Addr: "2.2.2.2", NodeID: 56}},
		"l2_upstream_3": {{Addr: "3.3.3.3", NodeID: 70}},
	}
	base := parentFetchDomain{
		UpstreamKey:         "upstream_1",
		ParentFetchMode:     "l1",
		ParentL1UpstreamKey: "l1_upstream_5",
		ParentL2UpstreamKey: "l2_upstream_5",
		UseL2:               true,
		L2UpstreamKey:       "l2_upstream_5",
	}

	type caseDef struct {
		name      string
		domain    parentFetchDomain
		l1Status  map[int64]bool
		l2Status  map[int64]bool
		wantKey   string
		wantLayer string
	}
	cases := []caseDef{
		{"L3_origin", parentFetchDomain{UpstreamKey: "upstream_1", ParentFetchMode: "origin"}, nil, nil, "upstream_1", "origin"},
		{"L3_l2", parentFetchDomain{UpstreamKey: "upstream_1", ParentFetchMode: "l2", ParentL2UpstreamKey: "l2_upstream_5"}, nil, map[int64]bool{56: true}, "l2_upstream_5", "parent"},
		{"L3_l1", base, map[int64]bool{41: true}, map[int64]bool{56: true}, "l1_upstream_5", "parent"},
		{"L3_l1_failover_l2", base, map[int64]bool{41: false}, map[int64]bool{56: true}, "l2_upstream_5", "parent"},
		{"L3_l1_both_down_origin", base, map[int64]bool{41: false}, map[int64]bool{56: false}, "upstream_1", "origin"},
		{"L3_l2_down_origin", parentFetchDomain{UpstreamKey: "upstream_1", ParentFetchMode: "l2", ParentL2UpstreamKey: "l2_upstream_5"}, nil, map[int64]bool{56: false}, "upstream_1", "origin"},
		{"L3_l1_recovery", base, map[int64]bool{41: true}, map[int64]bool{56: true}, "l1_upstream_5", "parent"},
	}
	for _, tc := range cases {
		key, layer := resolveL3Upstream(tc.domain, upstreams, tc.l1Status, tc.l2Status)
		if key != tc.wantKey || layer != tc.wantLayer {
			t.Fatalf("%s: got key=%s layer=%s want key=%s layer=%s", tc.name, key, layer, tc.wantKey, tc.wantLayer)
		}
	}
	if !resolveL1UseL2(base, upstreams, map[int64]bool{56: true}, false) {
		t.Fatal("L1_use_l2_healthy")
	}
	if resolveL1UseL2(base, upstreams, map[int64]bool{56: true}, true) {
		t.Fatal("L1_skip_l2_header")
	}
}

func TestSelectStreamTargetsParentL1Failover(t *testing.T) {
	stream := edgeStream{
		UseListenPort:   true,
		ParentFetchMode: "l1",
		Targets: []edgeStreamTarget{
			{Addr: "10.0.0.1", NodeID: 41, Enable: true},
			{Addr: "10.0.0.2", NodeID: 56, Enable: true, Backup: true},
			{Addr: "1.1.1.1:8080", Enable: true, Backup: true},
		},
	}
	status := streamStatusSnapshot{
		ParentL1: map[int64]bool{41: false},
		ParentL2: map[int64]bool{56: true},
	}
	targets := selectStreamTargets(stream, status)
	if len(targets) == 0 {
		t.Fatal("expected failover targets")
	}
	if targets[0].NodeID != 56 {
		t.Fatalf("expected L2 backup as primary after L1 offline, got %#v", targets[0])
	}
}
