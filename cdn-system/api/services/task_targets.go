package services

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"time"
)

const (
	TargetStateWaiting     = "waiting"
	TargetStateRunning     = "running"
	TargetStateSuccess     = "success"
	TargetStateFailedFinal = "failed_final"
)

type TaskTarget struct {
	State   string `json:"state"`
	Tries   int    `json:"tries"`
	RetryAt int64  `json:"retry_at"`
	Ret     string `json:"ret,omitempty"`
	LastAt  int64  `json:"last_at,omitempty"`
}

type TaskTargets struct {
	Nodes   map[string]*TaskTarget `json:"nodes"`
	Total   int                    `json:"total"`
	Success int                    `json:"success"`
	Fail    int                    `json:"fail"`
	Pending int                    `json:"pending"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewTaskTargets(nodeIDs []int64) *TaskTargets {
	targets := &TaskTargets{
		Nodes: make(map[string]*TaskTarget, len(nodeIDs)),
	}
	for _, id := range nodeIDs {
		key := strconv.FormatInt(id, 10)
		targets.Nodes[key] = &TaskTarget{State: TargetStateWaiting}
	}
	targets.EnsureCounts()
	return targets
}

func ParseTaskTargets(raw string) *TaskTargets {
	targets := &TaskTargets{Nodes: map[string]*TaskTarget{}}
	if raw == "" {
		return targets
	}
	if err := json.Unmarshal([]byte(raw), targets); err != nil {
		return targets
	}
	if targets.Nodes == nil {
		targets.Nodes = map[string]*TaskTarget{}
	}
	targets.EnsureCounts()
	return targets
}

func (t *TaskTargets) EnsureCounts() {
	total := len(t.Nodes)
	success := 0
	fail := 0
	pending := 0
	for _, target := range t.Nodes {
		switch target.State {
		case TargetStateSuccess:
			success++
		case TargetStateFailedFinal:
			fail++
		default:
			pending++
		}
	}
	t.Total = total
	t.Success = success
	t.Fail = fail
	t.Pending = pending
}

func (t *TaskTargets) Marshal() string {
	if t == nil {
		return ""
	}
	t.EnsureCounts()
	raw, _ := json.Marshal(t)
	return string(raw)
}

func (t *TaskTargets) ShouldDispatch(nodeID string, now time.Time) bool {
	if t == nil || nodeID == "" {
		return false
	}
	target := t.Nodes[nodeID]
	if target == nil {
		return false
	}
	switch target.State {
	case TargetStateSuccess, TargetStateFailedFinal, TargetStateRunning:
		return false
	}
	if target.RetryAt > 0 && time.Unix(target.RetryAt, 0).After(now) {
		return false
	}
	return true
}

func (t *TaskTargets) MarkRunning(nodeID string, now time.Time) int {
	target := t.ensureNode(nodeID)
	target.State = TargetStateRunning
	target.Tries++
	target.LastAt = now.Unix()
	target.RetryAt = 0
	t.EnsureCounts()
	return target.Tries
}

func (t *TaskTargets) MarkSuccess(nodeID string, ret string, now time.Time) int {
	target := t.ensureNode(nodeID)
	target.State = TargetStateSuccess
	target.Ret = ret
	target.LastAt = now.Unix()
	target.RetryAt = 0
	t.EnsureCounts()
	return target.Tries
}

func (t *TaskTargets) MarkFailure(nodeID string, ret string, now time.Time, maxRetries int) (bool, time.Time, int) {
	target := t.ensureNode(nodeID)
	if maxRetries <= 0 {
		maxRetries = 3
	}
	target.Ret = ret
	target.LastAt = now.Unix()
	if target.Tries >= maxRetries {
		target.State = TargetStateFailedFinal
		target.RetryAt = 0
		t.EnsureCounts()
		return true, time.Time{}, target.Tries
	}
	delay := jitterRetryDelay()
	retryAt := now.Add(delay)
	target.State = TargetStateWaiting
	target.RetryAt = retryAt.Unix()
	t.EnsureCounts()
	return false, retryAt, target.Tries
}

func (t *TaskTargets) ensureNode(nodeID string) *TaskTarget {
	if t.Nodes == nil {
		t.Nodes = map[string]*TaskTarget{}
	}
	target := t.Nodes[nodeID]
	if target == nil {
		target = &TaskTarget{State: TargetStateWaiting}
		t.Nodes[nodeID] = target
	}
	return target
}

func jitterRetryDelay() time.Duration {
	// 10-30 seconds
	return time.Duration(10+rand.Intn(21)) * time.Second
}
