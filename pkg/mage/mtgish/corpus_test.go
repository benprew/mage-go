package mtgish_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/mtgish"
)

type corpusReport struct {
	JSONLSHA256  string   `json:"mtgish_jsonl_sha256"`
	OracleSHA256 string   `json:"oracle_json_sha256"`
	Records      int      `json:"records"`
	Compiled     int      `json:"compiled"`
	Tested       int      `json:"tested"`
	Passed       int      `json:"passed"`
	Cards        []string `json:"cards"`
	FailedCards  []string `json:"failed_cards"`
}

type oracleFace struct {
	Name           string
	Type           string
	ManaCost       string
	Text           string
	Power          string
	Toughness      string
	ColorIndicator string
}

type oracleRule struct {
	kind    core.AbilityType
	keyword core.Attr
	mana    []core.Color
	damage  int
}

func TestMtgishCorpus(t *testing.T) {
	dataDir := os.Getenv("MTGISH_AUDIT_DATA")
	if dataDir == "" {
		t.Skip("set MTGISH_AUDIT_DATA to a directory with mtgish.lines.json and oracle.json")
	}
	definitions, report := compileCorpus(t, filepath.Join(dataDir, "mtgish.lines.json"))
	oracle, oracleHash := readOracleCorpus(t, filepath.Join(dataDir, "oracle.json"))
	report.OracleSHA256 = oracleHash
	registry := mage.NewRegistry()
	if err := registry.RegisterDefinitions(definitions...); err != nil {
		t.Fatal(err)
	}

	report.Cards = make([]string, 0, len(definitions))
	report.FailedCards = []string{}
	for _, definition := range definitions {
		name := definition.Name()
		report.Cards = append(report.Cards, name)
		ran := false
		passed := t.Run(name, func(t *testing.T) {
			ran = true
			faces := oracle[name]
			if len(faces) != 1 {
				t.Fatalf("got %d single-face Oracle records for %q, want 1", len(faces), name)
			}
			checkCorpusCard(t, registry, faces[0])
		})
		if !ran {
			continue
		}
		report.Tested++
		if passed {
			report.Passed++
		} else {
			report.FailedCards = append(report.FailedCards, name)
		}
	}
	t.Logf("compiled %d of %d records, %d tested, %d passed, %d failed", report.Compiled, report.Records, report.Tested, report.Passed, len(report.FailedCards))
	if path := os.Getenv("MTGISH_AUDIT_REPORT"); path != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func compileCorpus(t *testing.T, path string) ([]*mage.CardDefinition, corpusReport) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	hash := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hash))
	scanner.Buffer(nil, 1<<20)
	var definitions []*mage.CardDefinition
	var records int
	for scanner.Scan() {
		records++
		source, err := mtgish.ParseMtgishJSON(scanner.Bytes())
		if err != nil {
			continue
		}
		if definition, err := mtgish.Compile(source); err == nil {
			definitions = append(definitions, definition)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(definitions) == 0 {
		t.Fatal("no card in the input compiled")
	}
	slices.SortFunc(definitions, func(a, b *mage.CardDefinition) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return definitions, corpusReport{
		JSONLSHA256: fmt.Sprintf("%x", hash.Sum(nil)),
		Records:     records,
		Compiled:    len(definitions),
	}
}

func readOracleCorpus(t *testing.T, path string) (map[string][]oracleFace, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var records []struct {
		Layout string
		Cards  []oracleFace
	}
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	oracle := make(map[string][]oracleFace, len(records))
	for _, record := range records {
		if record.Layout == "single" && len(record.Cards) == 1 {
			face := record.Cards[0]
			oracle[face.Name] = append(oracle[face.Name], face)
		}
	}
	return oracle, fmt.Sprintf("%x", sha256.Sum256(data))
}

func checkCorpusCard(t *testing.T, registry *mage.Registry, oracle oracleFace) {
	t.Helper()
	caster, opponent := mage.NewBasePlayer("Caster"), mage.NewBasePlayer("Opponent")
	g := mage.NewGame(caster, opponent)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	card, err := registry.CreateInstance(oracle.Name, caster.PlayerID())
	if err != nil {
		t.Fatal(err)
	}
	another, err := registry.CreateCard(oracle.Name)
	if err != nil {
		t.Fatal(err)
	}
	if card.ID() == another.ID() || card.Definition() != another.Definition() || card.Owner() != caster.PlayerID() {
		t.Fatal("registry did not create separate instances from the shared definition")
	}

	cost := checkOracleCharacteristics(t, card, oracle)
	rules := readOracleRules(t, oracle)
	abilities := card.Abilities()
	if len(abilities) != len(rules) {
		t.Fatalf("got %d abilities, want %d from Oracle and basic land types", len(abilities), len(rules))
	}

	caster.AddToHand(card)
	fundCorpusSpell(caster, cost)
	isSpell := card.HasType(core.TypeInstant) || card.HasType(core.TypeSorcery)
	var permanent *mage.Permanent
	if !isSpell {
		if card.HasType(core.TypeLand) {
			if err := g.PlayLand(caster.PlayerID(), card.ID()); err != nil {
				t.Fatal(err)
			}
		} else {
			castCorpusSpell(t, g, card, nil)
			g.ResolveStack()
		}
		permanent = g.FindPermanent(card.ID())
		if card.HasType(core.TypeCreature) && card.Toughness() <= 0 {
			if permanent != nil || !slices.Contains(caster.Graveyard(), card) {
				t.Fatal("a creature with zero or less toughness did not go to the graveyard")
			}
		} else if permanent == nil || permanent.ControllerID() != caster.PlayerID() {
			t.Fatal("the card did not enter the battlefield under its caster's control")
		} else if card.HasType(core.TypeCreature) {
			if power, toughness := permanent.CurrentPower(g), permanent.CurrentToughness(g); power != card.Power() || toughness != card.Toughness() {
				t.Fatalf("creature entered as %d/%d, want %d/%d", power, toughness, card.Power(), card.Toughness())
			}
		}
	}

	for i, rule := range rules {
		if got := abilities[i].Type(); got != rule.kind {
			t.Fatalf("ability %d has type %v, want %v", i, got, rule.kind)
		}
		if rule.kind == core.AbilityStatic {
			if mage.AbilityAttribute(abilities[i]) != rule.keyword || !card.HasKeyword(rule.keyword) {
				t.Fatalf("ability %d lacks printed keyword %v", i, rule.keyword)
			}
			if permanent != nil && !permanent.HasKeyword(rule.keyword) {
				t.Fatalf("permanent lacks printed keyword %v", rule.keyword)
			}
		} else {
			checkCorpusAction(t, g, card, opponent.PlayerID(), i, rule)
		}
	}
}

var corpusManaCost = regexp.MustCompile(`^(?:\{(?:[0-9]+|[WUBRGCX]|[WUBRG]/[WUBRG])\})*$`)

func checkOracleCharacteristics(t *testing.T, card *mage.Card, oracle oracleFace) core.ManaCost {
	t.Helper()
	line := make([]string, 0, len(card.SuperTypes())+len(card.Types()))
	for _, kind := range card.SuperTypes() {
		line = append(line, kind.String())
	}
	for _, kind := range card.Types() {
		line = append(line, kind.String())
	}
	typeLine := strings.Join(line, " ")
	if subtypes := card.SubTypes(); len(subtypes) != 0 {
		typeLine += " — " + strings.Join(subtypes, " ")
	}
	if typeLine != oracle.Type {
		t.Errorf("type line = %q, want %q", typeLine, oracle.Type)
	}
	if !corpusManaCost.MatchString(oracle.ManaCost) {
		t.Fatalf("unsupported Oracle mana cost %q", oracle.ManaCost)
	}
	cost := core.ParseManaCost(oracle.ManaCost)
	cost.Colorless = strings.Count(oracle.ManaCost, "{C}")
	if got := card.ManaCost(); !reflect.DeepEqual(got, cost) {
		t.Errorf("mana cost = %#v, want %#v from %q", got, cost, oracle.ManaCost)
	}
	wantValue := cost.Generic + cost.Colorless + cost.White + cost.Blue + cost.Black + cost.Red + cost.Green + len(cost.Hybrid)
	if got := card.ManaCost().CMC(); got != wantValue {
		t.Errorf("mana value = %d, want %d", got, wantValue)
	}

	colorSymbols := oracle.ManaCost
	if oracle.ColorIndicator != "" {
		colorSymbols = oracle.ColorIndicator
	}
	var colors []core.Color
	for _, color := range core.Colors() {
		if strings.ContainsRune(colorSymbols, rune("CWUBRG"[color])) {
			colors = append(colors, color)
		}
	}
	actualColors := card.Colors()
	slices.Sort(actualColors)
	if !slices.Equal(actualColors, colors) {
		t.Errorf("colors = %v, want %v", actualColors, colors)
	}
	for _, stat := range []struct {
		name    string
		printed string
		actual  int
	}{
		{"power", oracle.Power, card.Power()},
		{"toughness", oracle.Toughness, card.Toughness()},
	} {
		printed := stat.printed
		if printed == "" {
			printed = "0"
		}
		if got := strconv.Itoa(stat.actual); got != printed {
			t.Errorf("%s = %s, want %s", stat.name, got, printed)
		}
	}
	return cost
}

var (
	oracleReminder = regexp.MustCompile(`\([^()]*\)`)
	oracleAddMana  = regexp.MustCompile(`^Add ((?:\{[WUBRGC]\})+)\.`)
	oracleDamage   = regexp.MustCompile(`^(.+) deals ([0-9]+) damage to any target\.$`)
	oracleKeywords = map[string]core.Attr{
		"flying": core.Flying, "first strike": core.FirstStrike, "double strike": core.DoubleStrike,
		"deathtouch": core.Deathtouch, "defender": core.Defender, "flash": core.Flash,
		"haste": core.Haste, "hexproof": core.Hexproof, "indestructible": core.Indestructible,
		"lifelink": core.Lifelink, "menace": core.Menace, "reach": core.Reach, "shroud": core.Shroud,
		"trample": core.Trample, "vigilance": core.Vigilance, "banding": core.Banding, "fear": core.Fear,
	}
	oracleBasicTypes = map[string]core.Color{
		"Plains": core.White, "Island": core.Blue, "Swamp": core.Black, "Mountain": core.Red, "Forest": core.Green,
	}
)

func readOracleRules(t *testing.T, oracle oracleFace) []oracleRule {
	t.Helper()
	var rules []oracleRule
	text := oracle.Text
	for oracleReminder.MatchString(text) {
		text = oracleReminder.ReplaceAllString(text, "")
	}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		effect, tap := strings.CutPrefix(line, "{T}: ")
		if mana := oracleAddMana.FindStringSubmatch(effect); mana != nil {
			rule := oracleRule{kind: core.AbilitySpell}
			if tap {
				rule.kind = core.AbilityMana
			}
			for i := 1; i < len(mana[1]); i += 3 {
				color := strings.IndexByte("CWUBRG", mana[1][i])
				rule.mana = append(rule.mana, core.Color(color))
			}
			rules = append(rules, rule)
		} else if damage := oracleDamage.FindStringSubmatch(effect); damage != nil {
			amount, err := strconv.Atoi(damage[2])
			if err != nil || amount < 1 {
				t.Fatalf("invalid Oracle damage amount in %q", line)
			}
			rule := oracleRule{kind: core.AbilitySpell, damage: amount}
			source := damage[1]
			if tap {
				rule.kind = core.AbilityActivated
				if source != "This creature" && source != oracle.Name && !strings.HasPrefix(oracle.Name, source+",") {
					t.Fatalf("unsupported Oracle damage source %q", source)
				}
			} else if source != oracle.Name {
				t.Fatalf("unsupported Oracle damage source %q", source)
			}
			rules = append(rules, rule)
		} else {
			for _, token := range strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ';' }) {
				keyword, ok := oracleKeywords[strings.ToLower(strings.TrimSpace(token))]
				if !ok {
					t.Fatalf("unsupported Oracle rule %q", line)
				}
				rules = append(rules, oracleRule{kind: core.AbilityStatic, keyword: keyword})
			}
		}
	}
	types, subtypes, _ := strings.Cut(oracle.Type, " — ")
	if slices.Contains(strings.Fields(types), "Land") {
		for subtype := range strings.FieldsSeq(subtypes) {
			if color, ok := oracleBasicTypes[subtype]; ok {
				rules = append(rules, oracleRule{kind: core.AbilityMana, mana: []core.Color{color}})
			}
		}
	}
	return rules
}

func fundCorpusSpell(player mage.Player, cost core.ManaCost) {
	for color, amount := range [...]int{cost.Generic + cost.Colorless, cost.White, cost.Blue, cost.Black, cost.Red, cost.Green} {
		player.ManaPool().Add(core.Color(color), amount)
	}
	for _, symbol := range cost.Hybrid {
		player.ManaPool().Add(symbol.A, 1)
	}
}

func castCorpusSpell(t *testing.T, g *mage.Game, card *mage.Card, targets []uuid.UUID) {
	t.Helper()
	if err := g.CastSpellByID(card.Owner(), card.ID(), targets, 0); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 {
		t.Fatalf("cast did not put %s on the stack", card.Name())
	}
	checkCorpusMana(t, g.GetPlayer(card.Owner()), nil)
}

func checkCorpusAction(t *testing.T, startingGame *mage.Game, card *mage.Card, opponentID uuid.UUID, index int, rule oracleRule) {
	t.Helper()
	if rule.damage == 0 {
		playCorpusAction(t, startingGame.Clone(), card, index, rule, nil)
		return
	}
	for _, targetKind := range []string{"player", "creature", "planeswalker"} {
		g := startingGame.Clone()
		targetID := opponentID
		threshold := rule.damage + 1
		switch targetKind {
		case "creature":
			targetID = g.PutOnBattlefield(mage.NewCreature("Target Creature", "{0}", 1, threshold), opponentID).ID()
		case "planeswalker":
			targetID = g.PutOnBattlefield(mage.NewPlaneswalker("Target Planeswalker", "{0}", threshold), opponentID).ID()
		}
		playCorpusAction(t, g, card, index, rule, []uuid.UUID{targetID})
		switch targetKind {
		case "player":
			if life := g.GetPlayer(targetID).Life(); life != 20-rule.damage {
				t.Errorf("target's life = %d, want %d", life, 20-rule.damage)
			}
		case "creature":
			permanent := g.FindPermanent(targetID)
			if permanent == nil || permanent.Damage != rule.damage {
				t.Errorf("creature did not take %d damage", rule.damage)
			}
		case "planeswalker":
			permanent := g.FindPermanent(targetID)
			if permanent == nil || int(permanent.Counters[core.Loyalty]) != 1 {
				t.Errorf("planeswalker did not lose %d loyalty", rule.damage)
			}
		}
	}
}

func playCorpusAction(t *testing.T, g *mage.Game, card *mage.Card, index int, rule oracleRule, targets []uuid.UUID) {
	t.Helper()
	caster := g.GetPlayer(card.Owner())
	if rule.kind == core.AbilitySpell {
		castCorpusSpell(t, g, card, targets)
	} else {
		source := g.FindPermanent(card.ID())
		if source == nil {
			t.Fatal("activated ability has no source on the battlefield")
		}
		if source.HasType(core.TypeCreature) && !source.HasKeyword(core.Haste) {
			if err := g.ActivateAbilityByIndex(card.Owner(), card.ID(), index, targets); err == nil {
				t.Fatal("summoning sickness did not prevent activation")
			}
			source.RevokeBaseAttr(core.AttrSummonSick)
		}
		if err := g.ActivateAbilityByIndex(card.Owner(), card.ID(), index, targets); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(card.ID()).Tapped {
			t.Fatal("activation did not pay the tap cost")
		}
		if err := g.ActivateAbilityByIndex(card.Owner(), card.ID(), index, targets); err == nil {
			t.Fatal("activated twice without untapping")
		}
		var wantStack int
		if rule.kind == core.AbilityActivated {
			wantStack = 1
		}
		if got := g.StackSize(); got != wantStack {
			t.Fatalf("stack size after activation = %d, want %d", got, wantStack)
		}
		if rule.kind == core.AbilityMana {
			checkCorpusMana(t, caster, rule.mana)
			return
		}
	}
	g.ResolveStack()
	checkCorpusMana(t, caster, rule.mana)
	if rule.kind == core.AbilitySpell && !slices.ContainsFunc(caster.Graveyard(), func(c *mage.Card) bool { return c.ID() == card.ID() }) {
		t.Fatal("resolved spell is missing from the caster's graveyard")
	}
}

func checkCorpusMana(t *testing.T, player mage.Player, expected []core.Color) {
	t.Helper()
	var counts [core.AnyColor]int
	for _, color := range expected {
		counts[color]++
	}
	for i, count := range counts {
		color := core.Color(i)
		if got := player.ManaPool().Count(color); got != count {
			t.Errorf("mana pool has %d %s mana, want %d", got, color, count)
		}
	}
}
