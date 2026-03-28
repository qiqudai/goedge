package models

import (
    "encoding/json"
    "testing"
)

func TestRealtimeJSON(t *testing.T) {
    b, err := json.Marshal(EdgeDomain{})
    if err != nil {
        t.Fatal(err)
    }
    t.Log(string(b))
}
