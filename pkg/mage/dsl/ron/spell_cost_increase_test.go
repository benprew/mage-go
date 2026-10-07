package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellCostIncreaseDerelor(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		for _, black := range []bool{false, true} {
			g, a, b := ronGame()
			derelor := repositoryCard(t, "FEM", "Derelor")
			if err := mage.ValidateCardDefinition(derelor.Name(), derelor); err != nil {
				t.Fatal(err)
			}
			derelor.SetOwner(a.PlayerID())
			a.AddToHand(derelor)
			a.ManaPool().Add(core.Colorless, 3)
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), derelor.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(derelor.ID()) != nil {
				t.Fatal("entered before resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(derelor.ID())
			if source.CurrentPower(g) != 4 || source.CurrentToughness(g) != 4 || !source.HasSubType("Thrull") {
				t.Fatal("Derelor metadata")
			}
			caster := a
			if opponent {
				caster = b
			}
			name := "Healing Salve"
			if black {
				name = "Dark Ritual"
			}
			spell := repositoryCard(t, "LEA", name)
			spell.SetOwner(caster.PlayerID())
			caster.AddToHand(spell)
			color := core.White
			if black {
				color = core.Black
			}
			caster.ManaPool().Add(color, 1)
			targets := []uuid.UUID(nil)
			if !black {
				targets = []uuid.UUID{caster.PlayerID()}
			}
			if black && !opponent {
				if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), targets, 0); err == nil {
					t.Fatal("unpaid black increase accepted")
				}
				if caster.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || len(caster.Hand()) != 1 {
					t.Fatal("cost rollback")
				}
				caster.ManaPool().Add(core.Black, 1)
			}
			if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			if caster.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect payment")
			}
			g.ResolveStack()
		}
	}
}

func TestRONSpellCostIncreaseInvalid(t *testing.T) {
	for _, rule := range []string{
		`PlayerEffect(You,[IncreaseSpellCost(IsColor(Black),PayMana([ManaCostX]))])`,
		`PlayerEffect(Opponent,[IncreaseSpellCost(IsColor(Black),PayMana([ManaCostB]))])`,
		`EachPlayerEffect(SinglePlayer(You),[IncreaseSpellCost(IsColor(White),PayMana([ManaCostGeneric(3)]))])`,
		`PlayerEffect(You,[IncreaseSpellCost(IsColor(Black),PayMana([ManaCostGeneric(-1)]))])`,
		`PlayerEffect(You,[IncreaseSpellCost(IsColor(Black),PayMana([ManaCostB]),UntilEndOfTurn)])`,
		`PlayerEffect(You,[IncreaseSpellCost(IsColor(Black),PayMana(ManaCostB))])`,
		`PlayerEffect(You,[IncreaseSpellCost(IsColor(Black),PayMana([ManaCostB])),IncreaseSpellCost(AnySpell,PayMana([]))])`,
		`PlayerEffect(You,[IncreaseSpellCost(And([]),PayMana([ManaCostB]))])`,
		`PlayerEffect(You,[IncreaseSpellCost(IsColor(Black()),PayMana([ManaCostB]))])`,
	} {
		t.Run(rule, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("accepted invalid tax")
			}
		})
	}
}

func TestRONSpellCostIncreaseLifecycle(t *testing.T) {
	g, a, b := ronGame()
	factory := spellTaxRepositoryFactory(t, "FEM", "Derelor")
	first, second := factory(), factory()
	if first.ID() == second.ID() {
		t.Fatal("factory identity shared")
	}
	source := g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	castProbe := func(game *mage.Game, caster mage.Player, black int, want bool) {
		t.Helper()
		spell := repositoryCard(t, "LEA", "Dark Ritual")
		spell.SetOwner(caster.PlayerID())
		caster.AddToHand(spell)
		caster.ManaPool().Clear()
		caster.ManaPool().Add(core.Black, black)
		before := game.StackSize()
		err := game.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0)
		if (err == nil) != want {
			t.Fatalf("cast with %d B = %v, want success %v", black, err, want)
		}
		if !want {
			if caster.ManaPool().TotalMana() != black || game.StackSize() != before {
				t.Fatal("failed payment changed state")
			}
			caster.RemoveFromHand(spell.ID())
		} else {
			if caster.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect black payment")
			}
			game.ResolveStack()
		}
	}
	castProbe(g, a, 2, false)
	castProbe(g, a, 3, true)
	castProbe(g, b, 1, true)
	clone := g.Clone()
	g.PhaseOut(source.ID())
	castProbe(g, a, 2, true)
	castProbe(clone, clone.GetPlayer(a.PlayerID()), 2, false)
	g.PhaseIn([]uuid.UUID{source.ID()})
	castProbe(g, a, 2, false)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	castProbe(g, a, 2, true)
	castProbe(g, b, 2, true)
	g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	castProbe(g, b, 1, true)
	g.RunStepWithPriority(core.Cleanup)
	castProbe(g, b, 1, false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	castProbe(g, b, 1, true)
	returned := g.PutOnBattlefield(first, b.PlayerID())
	castProbe(g, b, 2, true)
	returned.RevokeBaseAttr(core.AttrSummonSick)
	g.SetActivePlayerIndex(1)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{returned.ID()})
	g.ExecuteCombatDamage()
	if a.Life() != 16 {
		t.Fatal("Derelor combat damage", a.Life())
	}
}

func TestRONSpellCostIncreaseFilters(t *testing.T) {
	for _, scope := range []string{"PlayerEffect(You", "EachPlayerEffect(AnyPlayer"} {
		for _, colors := range []string{"Black", "White", "Blue", "Red", "Green"} {
			g, a, b := ronGame()
			rule := scope + `,[IncreaseSpellCost(Or([IsColor(` + colors + `),IsCardtype(Artifact)]),PayMana([ManaCostGeneric(3)]))])`
			factory := permanentScopeFactory(t, "Enchantment", rule)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			spell := mage.NewInstant("Indicator", "{U}", mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))))
			color := map[string]core.Color{"Black": core.Black, "White": core.White, "Blue": core.Blue, "Red": core.Red, "Green": core.Green}[colors]
			spell.SetColorOverride([]core.Color{color})
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			taxed := strings.HasPrefix(scope, "Each")
			b.ManaPool().Add(core.Blue, 1)
			if taxed {
				if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0); err == nil {
					t.Fatal("color indicator ignored or opponent exempted")
				}
				b.ManaPool().Add(core.Colorless, 3)
			}
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().TotalMana() != 0 {
				t.Fatal("taxed wrong amount")
			}
			g.RemoveFromBattlefield(source)
			g.ResolveStack()
		}
	}
}

func spellTaxRepositoryFactory(t *testing.T, set, name string) mage.CardFactory {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String == name {
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			return factory
		}
	}
	t.Fatalf("missing %s", name)
	return nil
}

func TestRONSpellCostIncreaseRecordsAndInventory(t *testing.T) {
	entries, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	derelors, glooms := 0, 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		var records []string
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name == "Derelor" {
				derelors++
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				for range 2 {
					if err := mage.ValidateCardDefinition(name, factory()); err != nil {
						t.Fatal(err)
					}
				}
			}
			if name == "Gloom" {
				glooms++
				if _, err := Compile(node); err == nil || !strings.Contains(err.Error(), "ActivatedAbilityEffect") {
					t.Fatal("Gloom must retain its activation-tax gap", err)
				}
				rules := node.Fields["Rules"].Args
				if _, err := compileRule(rules[0], scope{permanent: true}); err != nil {
					t.Fatal("Gloom spell-tax prerequisite", err)
				}
			}
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Derelor"`) {
				records = append(records, line)
			}
		}
		if len(records) > 0 {
			files[entry.Name()] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
		}
	}
	if derelors != 1 || glooms != 6 {
		t.Fatalf("records: Derelor=%d Gloom=%d", derelors, glooms)
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("strict Derelor loading", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("inventory", err)
	}
	for _, sig := range report.Signatures {
		if sig.Role == "unknown" {
			t.Fatal("unclassified tax descendant", sig)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PlayerEffect(You,[IncreaseSpellCost(AnySpell,PayMana([ManaCostX]))])])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry")
	}
}

func TestRONSpellCostIncreasePermanentSpellCopy(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Derelor")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 3)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("original missing")
	}
	for _, caster := range []mage.Player{a, b} {
		spell := repositoryCard(t, "LEA", "Dark Ritual")
		spell.SetOwner(caster.PlayerID())
		caster.AddToHand(spell)
		caster.ManaPool().Clear()
		caster.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0); err == nil {
			t.Fatal("copied static effect lost its own controller", caster.Name())
		}
		caster.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if caster.ManaPool().TotalMana() != 0 {
			t.Fatal("copies shared static source identity")
		}
		g.ResolveStack()
	}
}
