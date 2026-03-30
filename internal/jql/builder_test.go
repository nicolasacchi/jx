package jql

import "testing"

func TestBasicProject(t *testing.T) {
	got := New().Project("MLF").Build()
	want := "project = MLF"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProjectAndStatus(t *testing.T) {
	got := New().Project("MLF").Status("In Progress").Build()
	want := `project = MLF AND status = "In Progress"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFullQuery(t *testing.T) {
	got := New().
		Project("MLF").
		Status("In Progress").
		Type("Story").
		UpdatedSince("-7d").
		OrderBy("updated", "DESC").
		Build()
	want := `project = MLF AND status = "In Progress" AND issuetype = "Story" AND updated >= "-7d" ORDER BY updated DESC`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSprintCurrent(t *testing.T) {
	got := New().Project("MLF").Sprint("current").Build()
	want := "project = MLF AND sprint in openSprints()"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssigneeCurrentUser(t *testing.T) {
	got := New().Assignee("me").Build()
	want := "assignee = currentUser()"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssigneeUnassigned(t *testing.T) {
	got := New().Assignee("unassigned").Build()
	want := "assignee is EMPTY"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLabels(t *testing.T) {
	got := New().Project("MLF").Labels([]string{"bug", "urgent"}).Build()
	want := `project = MLF AND labels = "bug" AND labels = "urgent"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolutionUnresolved(t *testing.T) {
	got := New().Resolution("unresolved").Build()
	want := "resolution is EMPTY"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTextSearch(t *testing.T) {
	got := New().Text("login error").Build()
	want := `text ~ "login error"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEmpty(t *testing.T) {
	b := New()
	if !b.IsEmpty() {
		t.Error("expected empty builder")
	}
	if got := b.Build(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestOrderByOnly(t *testing.T) {
	got := New().OrderBy("created", "ASC").Build()
	want := "ORDER BY created ASC"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParent(t *testing.T) {
	got := New().Parent("MLF-5147").Build()
	want := "parent = MLF-5147"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestComponent(t *testing.T) {
	got := New().Project("MLF").Component("Backend").Build()
	want := `project = MLF AND component = "Backend"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFixVersion(t *testing.T) {
	got := New().Project("MLF").FixVersion("v2.1").Build()
	want := `project = MLF AND fixVersion = "v2.1"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDueBefore(t *testing.T) {
	got := New().DueBefore("2026-04-01").Build()
	want := `duedate <= "2026-04-01"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDueAfter(t *testing.T) {
	got := New().DueAfter("2026-03-01").Build()
	want := `duedate >= "2026-03-01"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
