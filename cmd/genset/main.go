// genset generates stub Go files for a card set from its JSON data.
//
// Usage:
//
//	go run ./cmd/genset <set-code> <json-file> <output-dir>
//
// Example:
//
//	go run ./cmd/genset legends cards/legends.json cards/legends/
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all card sets
)

type Card struct {
	Name       string   `json:"name"`
	ManaCost   string   `json:"mana_cost"`
	CMC        float64  `json:"cmc"`
	TypeLine   string   `json:"type_line"`
	OracleText string   `json:"oracle_text"`
	Power      string   `json:"power"`
	Toughness  string   `json:"toughness"`
	Colors     []string `json:"colors"`
	Keywords   []string `json:"keywords"`
	Rarity     string   `json:"rarity"`
	Set        string   `json:"set"`
}

type cardCategory int

const (
	catCreature cardCategory = iota
	catArtifact
	catEnchantment
	catSpell
	catLand
)

func categorize(c Card) cardCategory {
	tl := strings.ToLower(c.TypeLine)
	if strings.Contains(tl, "creature") {
		return catCreature
	}
	if strings.Contains(tl, "land") {
		return catLand
	}
	if strings.Contains(tl, "artifact") {
		return catArtifact
	}
	if strings.Contains(tl, "enchantment") {
		return catEnchantment
	}
	return catSpell
}

func isLegendary(c Card) bool {
	return strings.Contains(c.TypeLine, "Legendary")
}

func isWorld(c Card) bool {
	return strings.Contains(c.TypeLine, "World")
}

func isArtifactCreature(c Card) bool {
	tl := strings.ToLower(c.TypeLine)
	return strings.Contains(tl, "artifact") && strings.Contains(tl, "creature")
}

func isAura(c Card) bool {
	return strings.Contains(c.TypeLine, "Aura") ||
		strings.Contains(strings.ToLower(c.OracleText), "enchant ")
}

func isInstant(c Card) bool {
	return strings.Contains(c.TypeLine, "Instant")
}

func subtypes(c Card) []string {
	parts := strings.SplitN(c.TypeLine, "—", 2)
	if len(parts) < 2 {
		return nil
	}
	raw := strings.TrimSpace(parts[1])
	subs := strings.Fields(raw)
	return subs
}

// oracleComment formats oracle text as a Go comment block.
func oracleComment(c Card) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("// %s %s", c.Name, c.ManaCost))
	lines = append(lines, fmt.Sprintf("// %s", c.TypeLine))
	if c.Power != "" && c.Toughness != "" {
		lines = append(lines, fmt.Sprintf("// %s/%s", c.Power, c.Toughness))
	}
	if c.OracleText != "" {
		for _, line := range strings.Split(c.OracleText, "\n") {
			lines = append(lines, fmt.Sprintf("// %s", line))
		}
	}
	lines = append(lines, "// TODO: implement")
	return strings.Join(lines, "\n")
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "usage: genset <set-name> <json-file> <output-dir>\n")
		os.Exit(1)
	}
	setName := os.Args[1]
	jsonFile := os.Args[2]
	outDir := os.Args[3]

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", jsonFile, err)
		os.Exit(1)
	}

	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		fmt.Fprintf(os.Stderr, "parse %s: %v\n", jsonFile, err)
		os.Exit(1)
	}

	// Deduplicate by name (Scryfall can return multiple printings)
	seen := map[string]bool{}
	var unique []Card
	for _, c := range cards {
		if seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		unique = append(unique, c)
	}
	cards = unique

	// Check the registry for already-registered names (reprints from earlier sets)
	var reprints []Card
	var fresh []Card
	for _, c := range cards {
		if mage.CardRegistered(c.Name) {
			reprints = append(reprints, c)
		} else {
			fresh = append(fresh, c)
		}
	}
	if len(reprints) > 0 {
		fmt.Printf("skipping %d reprints (already registered):\n", len(reprints))
		for _, c := range reprints {
			fmt.Printf("  - %s\n", c.Name)
		}
		fmt.Println()
	}
	cards = fresh

	// Sort by name within each category
	sort.Slice(cards, func(i, j int) bool { return cards[i].Name < cards[j].Name })

	// Group by color for creatures, then by category for others
	creatures := groupCreaturesByColor(filter(cards, catCreature))
	artifacts := filter(cards, catArtifact)
	enchantments := filter(cards, catEnchantment)
	spells := filter(cards, catSpell)
	lands := filter(cards, catLand)

	// Derive package name from output directory base name
	pkgName := filepath.Base(outDir)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", outDir, err)
		os.Exit(1)
	}

	// Derive set code from JSON filename (e.g. "data/DRK.json" → "DRK").
	setCode := strings.TrimSuffix(filepath.Base(jsonFile), filepath.Ext(jsonFile))
	// Copy the catalog JSON into the output dir for go:embed.
	catalogJSON := filepath.Join(filepath.Dir(jsonFile), "catalog", setCode+".json")
	if catalogData, readErr := os.ReadFile(catalogJSON); readErr == nil {
		catalogDst := filepath.Join(outDir, setCode+".json")
		if writeErr := os.WriteFile(catalogDst, catalogData, 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "write catalog json: %v\n", writeErr)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "warning: catalog JSON not found at %s (run fetchcatalog first)\n", catalogJSON)
	}
	writeSet(outDir, pkgName, setName, setCode, catalogJSON)
	writeTest(outDir, pkgName)
	writeCreatures(outDir, pkgName, creatures)
	writeArtifacts(outDir, pkgName, artifacts)
	writeEnchantments(outDir, pkgName, enchantments)
	writeSpells(outDir, pkgName, spells)
	writeLands(outDir, pkgName, lands)

	fmt.Printf("generated %s/ (%d creatures, %d artifacts, %d enchantments, %d spells, %d lands)\n",
		outDir, countCreatures(creatures), len(artifacts), len(enchantments), len(spells), len(lands))
}

func filter(cards []Card, cat cardCategory) []Card {
	var out []Card
	for _, c := range cards {
		if categorize(c) == cat {
			out = append(out, c)
		}
	}
	return out
}

type colorGroup struct {
	Label string
	Cards []Card
}

func colorLabel(colors []string) string {
	if len(colors) == 0 {
		return "COLORLESS"
	}
	if len(colors) > 1 {
		return "MULTICOLOR"
	}
	switch colors[0] {
	case "W":
		return "WHITE"
	case "U":
		return "BLUE"
	case "B":
		return "BLACK"
	case "R":
		return "RED"
	case "G":
		return "GREEN"
	default:
		return "COLORLESS"
	}
}

func groupCreaturesByColor(cards []Card) []colorGroup {
	order := []string{"WHITE", "BLUE", "BLACK", "RED", "GREEN", "MULTICOLOR", "COLORLESS"}
	groups := map[string][]Card{}
	for _, c := range cards {
		label := colorLabel(c.Colors)
		groups[label] = append(groups[label], c)
	}
	var out []colorGroup
	for _, label := range order {
		if cs, ok := groups[label]; ok {
			out = append(out, colorGroup{Label: label, Cards: cs})
		}
	}
	return out
}

func countCreatures(groups []colorGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Cards)
	}
	return n
}

func writeSet(dir, pkg, setName, setCode, jsonFile string) {
	path := filepath.Join(dir, "set.go")
	f, _ := os.Create(path)
	defer f.Close()
	jsonBase := filepath.Base(jsonFile)
	fmt.Fprintf(f, `package %s

import (
	_ "embed"

	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
)

//go:embed %s
var catalogData []byte

func init() {
	catalog.RegisterSet(%q, %q, catalogData)
}
`, pkg, jsonBase, setCode, setName)
}

func writeTest(dir, pkg string) {
	path := filepath.Join(dir, "test.go")
	f, _ := os.Create(path)
	defer f.Close()
	fmt.Fprintf(f, `package %s

var _ = registerCreatures
var _ = registerArtifacts
var _ = registerEnchantments
var _ = registerLands
var _ = registerSpells
`, pkg)
}

// ptValue converts a power/toughness string to a Go integer literal.
// "*" becomes 0 (placeholder for variable P/T).
func ptValue(s string) string {
	if s == "*" || s == "" {
		return "0"
	}
	// Handle cases like "1+*" or similar
	if strings.Contains(s, "*") {
		return "0"
	}
	return s
}

func creatureStub(c Card) string {
	var b strings.Builder
	b.WriteString(oracleComment(c))
	b.WriteString("\n")

	// Build options
	var opts []string
	subs := subtypes(c)
	if len(subs) > 0 {
		quoted := make([]string, len(subs))
		for i, s := range subs {
			quoted[i] = fmt.Sprintf("%q", s)
		}
		opts = append(opts, fmt.Sprintf("WithSubTypes(%s)", strings.Join(quoted, ", ")))
	}
	if isLegendary(c) {
		opts = append(opts, "WithSuperTypes(SuperLegendary)")
	}
	if isArtifactCreature(c) {
		opts = append(opts, "WithCardType(TypeArtifact)")
	}

	optStr := ""
	if len(opts) > 0 {
		optStr = "\n"
		for _, o := range opts {
			optStr += "\t\t\t" + o + ",\n"
		}
		optStr += "\t\t"
	}

	fmt.Fprintf(&b, "\tRegister(%q, func() Card {\n", c.Name)
	fmt.Fprintf(&b, "\t\treturn NewCreature(%q, %q, %s, %s,%s)\n",
		c.Name, c.ManaCost, ptValue(c.Power), ptValue(c.Toughness), optStr)
	b.WriteString("\t})\n")
	return b.String()
}

func artifactStub(c Card) string {
	var b strings.Builder
	b.WriteString(oracleComment(c))
	b.WriteString("\n")
	fmt.Fprintf(&b, "\tRegister(%q, func() Card {\n", c.Name)
	fmt.Fprintf(&b, "\t\treturn NewArtifact(%q, %q)\n", c.Name, c.ManaCost)
	b.WriteString("\t})\n")
	return b.String()
}

func enchantmentStub(c Card) string {
	var b strings.Builder
	b.WriteString(oracleComment(c))
	b.WriteString("\n")
	fmt.Fprintf(&b, "\tRegister(%q, func() Card {\n", c.Name)
	if isAura(c) {
		fmt.Fprintf(&b, "\t\treturn NewAura(%q, %q)\n", c.Name, c.ManaCost)
	} else if isWorld(c) {
		fmt.Fprintf(&b, "\t\treturn NewEnchantment(%q, %q,\n", c.Name, c.ManaCost)
		fmt.Fprintf(&b, "\t\t\tWithSuperTypes(SuperWorld),\n")
		fmt.Fprintf(&b, "\t\t)\n")
	} else {
		fmt.Fprintf(&b, "\t\treturn NewEnchantment(%q, %q)\n", c.Name, c.ManaCost)
	}
	b.WriteString("\t})\n")
	return b.String()
}

func spellStub(c Card) string {
	var b strings.Builder
	b.WriteString(oracleComment(c))
	b.WriteString("\n")
	fmt.Fprintf(&b, "\tRegister(%q, func() Card {\n", c.Name)
	if isInstant(c) {
		fmt.Fprintf(&b, "\t\treturn NewInstant(%q, %q,\n", c.Name, c.ManaCost)
	} else {
		fmt.Fprintf(&b, "\t\treturn NewSorcery(%q, %q,\n", c.Name, c.ManaCost)
	}
	fmt.Fprintf(&b, "\t\t\tNewSpellAbility(),\n")
	fmt.Fprintf(&b, "\t\t)\n")
	b.WriteString("\t})\n")
	return b.String()
}

func landStub(c Card) string {
	var b strings.Builder
	b.WriteString(oracleComment(c))
	b.WriteString("\n")
	fmt.Fprintf(&b, "\tRegister(%q, func() Card {\n", c.Name)
	if isLegendary(c) {
		fmt.Fprintf(&b, "\t\treturn NewLand(%q,\n", c.Name)
		fmt.Fprintf(&b, "\t\t\tWithSuperTypes(SuperLegendary),\n")
		fmt.Fprintf(&b, "\t\t)\n")
	} else {
		fmt.Fprintf(&b, "\t\treturn NewLand(%q)\n", c.Name)
	}
	b.WriteString("\t})\n")
	return b.String()
}

var creaturesFileTmpl = template.Must(template.New("creatures").Parse(`package {{.Pkg}}

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerCreatures()
}

func registerCreatures() {
{{- range .Groups}}

	// ===== {{.Label}} CREATURES =====
{{range .Cards}}
{{.Stub}}
{{- end}}
{{- end}}
}
`))

func writeCreatures(dir, pkg string, groups []colorGroup) {
	type groupData struct {
		Label string
		Cards []struct{ Stub string }
	}
	var gd []groupData
	for _, g := range groups {
		var cards []struct{ Stub string }
		for _, c := range g.Cards {
			cards = append(cards, struct{ Stub string }{creatureStub(c)})
		}
		gd = append(gd, groupData{Label: g.Label, Cards: cards})
	}
	path := filepath.Join(dir, "creatures.go")
	f, _ := os.Create(path)
	defer f.Close()
	creaturesFileTmpl.Execute(f, struct {
		Pkg    string
		Groups []groupData
	}{pkg, gd})
}

// needsCoreImport checks if any card in the list uses core types (SuperTypes, CardTypes, etc.)
func needsCoreImport(cards []Card) bool {
	for _, c := range cards {
		if isLegendary(c) || isWorld(c) || isArtifactCreature(c) {
			return true
		}
	}
	return false
}

func writeFileWithCards(dir, pkg, filename, funcName string, cards []Card, stubFn func(Card) string) {
	path := filepath.Join(dir, filename)
	f, _ := os.Create(path)
	defer f.Close()

	if needsCoreImport(cards) {
		fmt.Fprintf(f, `package %s

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	%s()
}

func %s() {
`, pkg, funcName, funcName)
	} else {
		fmt.Fprintf(f, `package %s

import . "git.sr.ht/~cdcarter/mage-go/pkg/mage"

func init() {
	%s()
}

func %s() {
`, pkg, funcName, funcName)
	}

	for _, c := range cards {
		fmt.Fprintf(f, "\n%s\n", stubFn(c))
	}

	fmt.Fprintf(f, "}\n")
}

func writeArtifacts(dir, pkg string, cards []Card) {
	writeFileWithCards(dir, pkg, "artifacts.go", "registerArtifacts", cards, artifactStub)
}

func writeEnchantments(dir, pkg string, cards []Card) {
	writeFileWithCards(dir, pkg, "enchantments.go", "registerEnchantments", cards, enchantmentStub)
}

func writeSpells(dir, pkg string, cards []Card) {
	writeFileWithCards(dir, pkg, "spells.go", "registerSpells", cards, spellStub)
}

func writeLands(dir, pkg string, cards []Card) {
	writeFileWithCards(dir, pkg, "lands.go", "registerLands", cards, landStub)
}

