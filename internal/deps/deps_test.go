package deps

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantLen int
		wantMgr string
	}{
		{"package.json", `{"name":"x","dependencies":{"lodash":"^4.0.0"},"devDependencies":{"jest":"29"}}`, 2, ManagerNPM},
		{"composer.json", `{"require":{"monolog/monolog":"^2.0"},"require-dev":{"phpunit/phpunit":"^9.0"}}`, 2, ManagerComposer},
		{"go.mod", "module example.com/x\n\ngo 1.22\n\nrequire (\n\tgithub.com/foo/bar v1.2.3\n\tgolang.org/x/sys v0.0.0\n)\n\nrequire github.com/baz v2.0.0\n", 3, ManagerGo},
		{"Cargo.toml", "[dependencies]\nserde = \"1.0\"\ntokio = { version = \"1.35\", features = [\"full\"] }\n\n[dev-dependencies]\ncriterion = \"0.5\"\n", 3, ManagerCargo},
		{"pom.xml", `<project><dependencies><dependency><groupId>org.slf4j</groupId><artifactId>slf4j-api</artifactId><version>2.0.9</version></dependency></dependencies></project>`, 1, ManagerMaven},
		{"requirements.txt", "flask==2.3.0\nrequests>=2.31\n# comment\n", 2, ManagerPip},
		{"other.txt", "anything", 0, ""},
	}
	for _, c := range cases {
		mgr, entries, err := Parse(c.name, c.content)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if mgr != c.wantMgr {
			t.Errorf("%s: manager got %q want %q", c.name, mgr, c.wantMgr)
		}
		if len(entries) != c.wantLen {
			t.Errorf("%s: got %d entries, want %d (%+v)", c.name, len(entries), c.wantLen, entries)
		}
	}
}

func TestParseErrors(t *testing.T) {
	if _, _, err := Parse("package.json", "{not json"); err == nil {
		t.Fatalf("expected error for malformed json")
	}
	if _, _, err := Parse("pom.xml", "<not xml"); err == nil {
		t.Fatalf("expected error for malformed xml")
	}
}

func TestParseEntryFields(t *testing.T) {
	_, entries, err := Parse("package.json", `{"dependencies":{"lodash":"^4.0.0"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "lodash" || entries[0].Version != "^4.0.0" || entries[0].Scope != "production" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
