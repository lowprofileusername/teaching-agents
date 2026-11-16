package main

import (
	"testing"
	"time"
)

func TestTeacherRestDoesNotDependOnTaskCompletion(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		mode, state string
		elapsed     time.Duration
		want        string
	}{
		{"auto", "idle", 89 * time.Second, "active"},
		{"auto", "idle", 90 * time.Second, "sit"},
		{"auto", "thinking", 5 * time.Minute, "active"},
		{"sit", "answering", 0, "sit"},
		{"active", "done", 5 * time.Minute, "active"},
	} {
		if got := resolvedTeacherPosture(c.mode, c.state, now.Add(-c.elapsed), now); got != c.want {
			t.Errorf("%+v got %s", c, got)
		}
	}
}
