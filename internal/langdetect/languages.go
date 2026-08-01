package langdetect

// language table keyed by file extension (without dot) and by exact filename.
var (
	byExtension = map[string]*Lang{}
	byFilename  = map[string]*Lang{}
)

var languages = []*Lang{
	{Name: "Go", Extensions: []string{"go"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "Rust", Extensions: []string{"rs"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "Python", Extensions: []string{"py", "pyw"}, LineComments: []string{"#"}},
	{Name: "Java", Extensions: []string{"java"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "Kotlin", Extensions: []string{"kt", "kts"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "TypeScript", Extensions: []string{"ts", "tsx", "mts", "cts"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "JavaScript", Extensions: []string{"js", "jsx", "mjs", "cjs"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "PHP", Extensions: []string{"php"}, LineComments: []string{"//", "#"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "C#", Extensions: []string{"cs"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "C++", Extensions: []string{"cpp", "cc", "cxx", "c++", "hpp", "hh", "hxx"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "C", Extensions: []string{"c", "h"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "Swift", Extensions: []string{"swift"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},

	// Additional text formats for useful LOC metrics beyond the required set.
	{Name: "JSON", Extensions: []string{"json", "jsonc", "json5"}},
	{Name: "YAML", Extensions: []string{"yml", "yaml"}, LineComments: []string{"#"}},
	{Name: "TOML", Extensions: []string{"toml"}, LineComments: []string{"#"}},
	{Name: "Markdown", Extensions: []string{"md", "markdown"}},
	{Name: "HTML", Extensions: []string{"html", "htm", "vue", "svelte", "xml", "svg", "xhtml"}, BlockComments: [][2]string{{"<!--", "-->"}}},
	{Name: "CSS", Extensions: []string{"css"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "SCSS", Extensions: []string{"scss", "sass"}, LineComments: []string{"//"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "Shell", Extensions: []string{"sh", "bash", "zsh"}, LineComments: []string{"#"}},
	{Name: "Ruby", Extensions: []string{"rb", "rake", "gemspec"}, LineComments: []string{"#"}},
	{Name: "SQL", Extensions: []string{"sql"}, LineComments: []string{"--"}, BlockComments: [][2]string{{"/*", "*/"}}},
	{Name: "Lua", Extensions: []string{"lua"}, LineComments: []string{"--"}},
	{Name: "R", Extensions: []string{"r", "R"}, LineComments: []string{"#"}},
	{Name: "Docker", Filenames: []string{"Dockerfile", "dockerfile"}, LineComments: []string{"#"}},
	{Name: "Make", Filenames: []string{"Makefile", "makefile", "GNUmakefile"}, LineComments: []string{"#"}},
	{Name: "Plain Text", Extensions: []string{"txt", "text", "csv", "log"}},
}

func init() {
	for _, l := range languages {
		for _, e := range l.Extensions {
			byExtension[e] = l
		}
		for _, f := range l.Filenames {
			byFilename[f] = l
		}
	}
}
