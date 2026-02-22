package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

type Card struct {
	Name       string   `json:"name"`
	ManaCost   string   `json:"mana_cost"`
	CMC        float64  `json:"cmc"`
	TypeLine   string   `json:"type_line"`
	OracleText string   `json:"oracle_text"`
	Colors     []string `json:"colors"`
	Keywords   []string `json:"keywords"`
	Rarity     string   `json:"rarity"`
	Power      string   `json:"power,omitempty"`
	Toughness  string   `json:"toughness,omitempty"`
}

var keywordMap = map[string]string{
	"Flying":       "Flying",
	"Trample":      "Trample",
	"First strike": "FirstStrike",
	"Defender":     "Defender",
	"Haste":        "Haste",
	"Vigilance":    "Vigilance",
	"Banding":      "Banding",
	"Reach":        "Reach",
	"Deathtouch":   "Deathtouch",
	"Swampwalk":    "Swampwalk",
	"Islandwalk":   "Islandwalk",
	"Forestwalk":   "Forestwalk",
	"Mountainwalk": "Mountainwalk",
	"Plainswalk":   "Plainswalk",
}

var basicLandMana = map[string]string{
	"Plains":   "White",
	"Island":   "Blue",
	"Swamp":    "Black",
	"Mountain": "Red",
	"Forest":   "Green",
}

const (
	typeCreature = "creature"
	typeLand     = "land"
	typeArtifact = "artifact"
)

func main() {
	input := flag.String("input", "", "JSON card data file")
	dir := flag.String("dir", "", "output directory for generated Go files")
	pkg := flag.String("pkg", "", "Go package name (default: directory basename)")
	flag.Parse()

	if *input == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "Usage: cardgen -input <file.json> -dir <outdir> [-pkg <name>]")
		os.Exit(1)
	}
	if *pkg == "" {
		*pkg = filepath.Base(*dir)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fatal("reading input: %v", err)
	}
	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		fatal("parsing JSON: %v", err)
	}

	groups := groupCards(cards)
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fatal("creating directory: %v", err)
	}

	type spec struct {
		file, funcName, category string
	}
	specs := []spec{
		{"creatures.go", "registerCreatures", typeCreature},
		{"spells.go", "registerSpells", "spell"},
		{"enchantments.go", "registerEnchantments", "enchantment"},
		{"artifacts.go", "registerArtifacts", typeArtifact},
		{"lands.go", "registerLands", typeLand},
	}

	var funcNames []string
	total := 0
	for _, s := range specs {
		cc := groups[s.category]
		if len(cc) == 0 {
			continue
		}
		path := filepath.Join(*dir, s.file)
		f, err := os.Create(path)
		if err != nil {
			fatal("creating %s: %v", path, err)
		}
		err = fileTmpl.Execute(f, map[string]any{
			"Package":   *pkg,
			"FuncName":  s.funcName,
			"Body":      renderCards(s.category, cc),
			"NeedsCore": needsCoreImport(s.category, cc),
		})
		f.Close()
		if err != nil {
			fatal("writing %s: %v", path, err)
		}
		if err := exec.Command("gofmt", "-w", path).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "gofmt %s: %v\n", path, err)
		}
		funcNames = append(funcNames, s.funcName)
		fmt.Printf("  %s: %d cards\n", s.file, len(cc))
		total += len(cc)
	}

	if len(funcNames) > 0 {
		path := filepath.Join(*dir, "test.go")
		f, err := os.Create(path)
		if err != nil {
			fatal("creating %s: %v", path, err)
		}
		if err := testTmpl.Execute(f, map[string]any{
			"Package":   *pkg,
			"FuncNames": funcNames,
		}); err != nil {
			fatal("writing %s: %v", path, err)
		}
		f.Close()
		if err := exec.Command("gofmt", "-w", path).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "gofmt %s: %v\n", path, err)
		}
	}

	fmt.Printf("Done! %d cards in %s\n", total, *dir)
}

// --- Grouping ---

func groupCards(cards []Card) map[string][]Card {
	groups := map[string][]Card{}
	for _, c := range cards {
		groups[cardCategory(c)] = append(groups[cardCategory(c)], c)
	}
	for _, cc := range groups {
		sort.Slice(cc, func(i, j int) bool { return cc[i].Name < cc[j].Name })
	}
	return groups
}

func cardCategory(c Card) string {
	tl := c.TypeLine
	switch {
	case strings.HasPrefix(tl, "Basic Land"), strings.HasPrefix(tl, "Land"):
		return typeLand
	case strings.HasPrefix(tl, "Artifact Creature"), strings.HasPrefix(tl, "Creature"):
		return typeCreature
	case strings.HasPrefix(tl, "Instant"), strings.HasPrefix(tl, "Sorcery"):
		return "spell"
	case strings.HasPrefix(tl, "Enchantment"):
		return "enchantment"
	case strings.HasPrefix(tl, "Artifact"):
		return typeArtifact
	default:
		return typeArtifact
	}
}

func needsCoreImport(category string, cards []Card) bool {
	for _, c := range cards {
		if len(mapKeywords(c.Keywords)) > 0 {
			return true
		}
		if category == typeCreature && strings.HasPrefix(c.TypeLine, "Artifact Creature") {
			return true
		}
		if category == typeLand && strings.HasPrefix(c.TypeLine, "Basic Land") {
			return true
		}
	}
	return false
}

// --- Template helpers ---

// oracle returns the card's oracle text formatted as Go comment lines.
// Returns "" for vanilla/keyword-only cards.
func oracle(c Card) string {
	if c.OracleText == "" || isKeywordOnly(c) {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(c.OracleText, "\n") {
		fmt.Fprintf(&b, "\t// %s\n", line)
	}
	return b.String()
}

func isKeywordOnly(c Card) bool {
	text := strings.TrimSpace(c.OracleText)
	if text == "" {
		return true
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, part := range strings.Split(line, ", ") {
			part = strings.TrimSpace(part)
			if _, ok := keywordMap[part]; ok {
				continue
			}
			if strings.HasPrefix(part, "Enchant ") {
				continue
			}
			return false
		}
	}
	return true
}

func parseSubTypes(typeLine string) []string {
	parts := strings.SplitN(typeLine, "\u2014", 2)
	if len(parts) < 2 {
		return nil
	}
	if raw := strings.TrimSpace(parts[1]); raw != "" {
		return strings.Fields(raw)
	}
	return nil
}

func parsePT(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func mapKeywords(keywords []string) []string {
	var out []string
	for _, kw := range keywords {
		if g, ok := keywordMap[kw]; ok {
			out = append(out, g)
		}
	}
	return out
}

func joinQuoted(ss []string) string {
	q := make([]string, len(ss))
	for i, s := range ss {
		q[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(q, ", ")
}

// formatOpts builds the trailing ",\n\t\t\tOpt1,\n\t\t" portion
// of a constructor call. Returns "" when opts is empty.
func formatOpts(opts []string) string {
	if len(opts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, opt := range opts {
		fmt.Fprintf(&b, "\n\t\t\t%s,", opt)
	}
	b.WriteString("\n\t\t")
	return "," + b.String()
}

func creatureOpts(c Card) string {
	var opts []string
	if subs := parseSubTypes(c.TypeLine); len(subs) > 0 {
		opts = append(opts, fmt.Sprintf("WithSubTypes(%s)", joinQuoted(subs)))
	}
	if strings.HasPrefix(c.TypeLine, "Artifact Creature") {
		opts = append(opts, "WithCardType(TypeArtifact)")
	}
	for _, kw := range mapKeywords(c.Keywords) {
		opts = append(opts, fmt.Sprintf("WithKeyword(%s)", kw))
	}
	return formatOpts(opts)
}

func landOpts(c Card) string {
	subtypes := parseSubTypes(c.TypeLine)
	var opts []string
	if len(subtypes) > 0 {
		opts = append(opts, fmt.Sprintf("WithSubTypes(%s)", joinQuoted(subtypes)))
	}
	for _, st := range subtypes {
		if color, ok := basicLandMana[st]; ok {
			opts = append(opts, fmt.Sprintf("WithManaAbility(%s)", color))
		}
	}
	return formatOpts(opts)
}

func renderCards(category string, cards []Card) string {
	var b strings.Builder
	for i, c := range cards {
		if i > 0 {
			b.WriteString("\n")
		}
		cardTmpls.ExecuteTemplate(&b, category, c)
	}
	return b.String()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// --- Templates ---

var funcMap = template.FuncMap{
	"q":               func(s string) string { return fmt.Sprintf("%q", s) },
	"oracle":          oracle,
	"power":           func(c Card) int { return parsePT(c.Power) },
	"toughness":       func(c Card) int { return parsePT(c.Toughness) },
	"creatureOpts":    creatureOpts,
	"spellCtor":       func(c Card) string { if strings.HasPrefix(c.TypeLine, "Instant") { return "NewInstant" }; return "NewSorcery" },
	"enchantmentCtor": func(c Card) string { if strings.Contains(c.TypeLine, "Aura") { return "NewAura" }; return "NewEnchantment" },
	"landOpts":        landOpts,
}

var cardTmpls = template.Must(template.New("cards").Funcs(funcMap).Parse(
	"{{define \"creature\"}}" +
		"{{oracle .}}" +
		"\tRegister({{q .Name}}, func() Card {\n" +
		"\t\treturn NewCreature({{q .Name}}, {{q .ManaCost}}, {{power .}}, {{toughness .}}{{creatureOpts .}})\n" +
		"\t})\n" +
		"{{end}}" +

		"{{define \"spell\"}}" +
		"{{oracle .}}" +
		"\tRegister({{q .Name}}, func() Card {\n" +
		"\t\treturn {{spellCtor .}}({{q .Name}}, {{q .ManaCost}}, nil)\n" +
		"\t})\n" +
		"{{end}}" +

		"{{define \"enchantment\"}}" +
		"{{oracle .}}" +
		"\tRegister({{q .Name}}, func() Card {\n" +
		"\t\treturn {{enchantmentCtor .}}({{q .Name}}, {{q .ManaCost}})\n" +
		"\t})\n" +
		"{{end}}" +

		"{{define \"artifact\"}}" +
		"{{oracle .}}" +
		"\tRegister({{q .Name}}, func() Card {\n" +
		"\t\treturn NewArtifact({{q .Name}}, {{q .ManaCost}})\n" +
		"\t})\n" +
		"{{end}}" +

		"{{define \"land\"}}" +
		"{{oracle .}}" +
		"\tRegister({{q .Name}}, func() Card {\n" +
		"\t\treturn NewLand({{q .Name}}{{landOpts .}})\n" +
		"\t})\n" +
		"{{end}}"))

var fileTmpl = template.Must(template.New("file").Parse(`package {{.Package}}

import (
	. "github.com/mage/mage/pkg/mage"
{{- if .NeedsCore}}
	. "github.com/mage/mage/pkg/mage/core"
{{- end}}
)

func init() {
	{{.FuncName}}()
}

func {{.FuncName}}() {
{{.Body}}}
`))

var testTmpl = template.Must(template.New("test").Parse(`package {{.Package}}

// Ensure all registration functions are referenced.
{{range .FuncNames}}var _ = {{.}}
{{end}}`))
