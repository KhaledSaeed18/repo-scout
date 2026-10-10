package architecture

import "testing"

// TestImportResolutionPerLanguage builds a tiny repository per language and
// checks the import edges each one produces.
func TestImportResolutionPerLanguage(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  [][2]string // from, to
	}{
		{
			name: "python modules, packages and src layout",
			files: map[string]string{
				"app/main.py":            "import app.util\nfrom app.models import User\nimport os\n",
				"app/util.py":            "def f(): pass\n",
				"app/models/__init__.py": "class User: pass\n",
				"tools/run.py":           "from lib.core import x\n",
				"src/lib/core.py":        "x = 1\n",
			},
			want: [][2]string{
				{"app/main.py", "app/util.py"},
				{"app/main.py", "app/models/__init__.py"},
				{"tools/run.py", "src/lib/core.py"},
			},
		},
		{
			name: "rust crate, super and self paths",
			files: map[string]string{
				"src/main.rs":            "use crate::config::Settings;\nuse std::io;\n",
				"src/config.rs":          "pub struct Settings;\n",
				"src/net/mod.rs":         "use self::client::Client;\n",
				"src/net/client.rs":      "use super::mod_helpers;\npub struct Client;\n",
				"src/net/mod_helpers.rs": "\n",
			},
			want: [][2]string{
				{"src/main.rs", "src/config.rs"},
				{"src/net/mod.rs", "src/net/client.rs"},
				{"src/net/client.rs", "src/net/mod_helpers.rs"},
			},
		},
		{
			name: "java and kotlin under source roots",
			files: map[string]string{
				"src/main/java/com/acme/App.java":          "package com.acme;\nimport com.acme.util.Strings;\nimport java.util.List;\n",
				"src/main/java/com/acme/util/Strings.java": "package com.acme.util;\n",
				"src/main/kotlin/com/acme/Main.kt":         "package com.acme\nimport com.acme.util.Text\n",
				"src/main/kotlin/com/acme/util/Text.kt":    "package com.acme.util\n",
			},
			want: [][2]string{
				{"src/main/java/com/acme/App.java", "src/main/java/com/acme/util/Strings.java"},
				{"src/main/kotlin/com/acme/Main.kt", "src/main/kotlin/com/acme/util/Text.kt"},
			},
		},
		{
			name: "c and c++ quoted includes",
			files: map[string]string{
				"src/main.c":    "#include \"util.h\"\n#include <stdio.h>\n#include \"include/api.h\"\n",
				"src/util.h":    "int f(void);\n",
				"include/api.h": "int g(void);\n",
				"main.cpp":      "#include \"src/util.h\"\n",
			},
			want: [][2]string{
				{"src/main.c", "src/util.h"},
				{"src/main.c", "include/api.h"},
				{"main.cpp", "src/util.h"},
			},
		},
		{
			name: "php namespaces",
			files: map[string]string{
				"index.php":           "<?php\nuse App\\Http\\Router;\nuse \\Vendor\\Lib;\n",
				"App/Http/Router.php": "<?php\nclass Router {}\n",
			},
			want: [][2]string{{"index.php", "App/Http/Router.php"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range c.files {
				write(t, root, rel, content)
			}
			rep, err := Build(root, fileList(root), readAll(root))
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			got := map[[2]string]bool{}
			for _, e := range rep.Edges {
				got[[2]string{e.From, e.To}] = true
			}
			for _, w := range c.want {
				if !got[w] {
					t.Errorf("missing edge %s -> %s; got %v", w[0], w[1], rep.Edges)
				}
			}
			if len(rep.Edges) != len(c.want) {
				t.Errorf("expected %d edges, got %d: %v", len(c.want), len(rep.Edges), rep.Edges)
			}
		})
	}
}
