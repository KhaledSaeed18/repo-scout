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
	ok := Settings{WorkerCount: 2, MaxFileSize: 1, DupMinSimilarity: 0.5, DupMinLines: 2}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid settings, got %v", err)
	}
}
