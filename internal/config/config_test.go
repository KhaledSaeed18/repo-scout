package config

import "testing"

func TestDefaultsWithDefaults(t *testing.T) {
	s := Settings{}
	if s.WorkerCount != 0 {
		t.Fatalf("expected zero worker count, got %d", s.WorkerCount)
	}
	s = s.WithDefaults()
	if s.WorkerCount != Defaults().WorkerCount {
		t.Fatalf("expected %d workers, got %d", Defaults().WorkerCount, s.WorkerCount)
	}
}

func TestValidate(t *testing.T) {
	cases := []Settings{
		{WorkerCount: 0, MaxFileSize: 1, DupMinSimilarity: 0.5, DupMinLines: 2},
		{WorkerCount: 65, MaxFileSize: 1, DupMinSimilarity: 0.5, DupMinLines: 2},
		{WorkerCount: 2, MaxFileSize: 1, DupMinSimilarity: 1.5, DupMinLines: 2},
		{WorkerCount: 2, MaxFileSize: 1, DupMinSimilarity: 0.5, DupMinLines: 0},
	}
	for i, c := range cases {
		if err := c.Validate(); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
	ok := Settings{WorkerCount: 2, MaxFileSize: 1, DupMinSimilarity: 0.5, DupMinLines: 2, Theme: "light"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid settings, got %v", err)
	}
}

func TestValidateTheme(t *testing.T) {
	for _, theme := range []string{"system", "light", "dark"} {
		s := Defaults()
		s.Theme = theme
		if err := s.Validate(); err != nil {
			t.Fatalf("theme %q should be valid: %v", theme, err)
		}
	}
	s := Defaults()
	s.Theme = "solarized"
	if err := s.Validate(); err == nil {
		t.Fatal("expected unknown theme to be rejected")
	}
	if Defaults().Theme != "system" {
		t.Fatalf("expected system theme by default, got %q", Defaults().Theme)
	}
}

func TestFromEnvListensOnLoopbackByDefault(t *testing.T) {
	t.Setenv("REPO_SCOUT_ADDR", "")
	if got := FromEnv().Addr; got != "127.0.0.1:8080" {
		t.Fatalf("the API exposes local files; default must be loopback only, got %q", got)
	}
	t.Setenv("REPO_SCOUT_ADDR", "0.0.0.0:9000")
	if got := FromEnv().Addr; got != "0.0.0.0:9000" {
		t.Fatalf("REPO_SCOUT_ADDR must override the default, got %q", got)
	}
}
