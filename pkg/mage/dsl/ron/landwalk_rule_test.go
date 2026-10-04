package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONLandwalkRuleCards(t *testing.T) {
	for _, tc := range []struct {
		name             string
		lands            []string
		power, toughness int
	}{
		{"Crevasse", []string{"Mountain"}, 0, 0},
		{"Deadfall", []string{"Forest"}, 0, 0},
		{"Gosta Dirk", []string{"Island"}, 4, 4},
		{"Great Wall", []string{"Plains"}, 0, 0},
		{"Lord Magnus", []string{"Plains", "Forest"}, 4, 3},
		{"Quagmire", []string{"Swamp"}, 0, 0},
		{"Undertow", []string{"Island"}, 0, 0},
		{"Ur-Drago", []string{"Swamp"}, 4, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", tc.name)
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			if tc.power != 0 && (source.CurrentPower(g) != tc.power || source.CurrentToughness(g) != tc.toughness || !source.HasKeyword(core.FirstStrike)) {
				t.Fatal("printed creature rules changed")
			}
			walkers := map[string]*mage.Permanent{}
			for _, land := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest"} {
				kw := core.LandwalkAttr(land)
				walkers[land] = g.PutOnBattlefield(mage.NewCreature("Walker", "", 2, 2, mage.WithKeyword(kw)), a.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", land), b.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", land), a.PlayerID())
			}
			check := func(game *mage.Game, active bool) {
				t.Helper()
				for land, walker := range walkers {
					suppressed := false
					for _, selected := range tc.lands {
						suppressed = suppressed || selected == land
					}
					p := game.FindPermanent(walker.ID())
					if !p.HasKeyword(core.LandwalkAttr(land)) {
						t.Fatal("blocking override removed landwalk")
					}
					for _, defender := range []mage.Player{a, b} {
						if mage.HasLandwalkEvasion(p, defender.PlayerID(), game) != (!active || !suppressed) {
							t.Fatalf("%s evasion does not match current rule", land)
						}
					}
				}
			}
			check(g, true)
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
			check(clone, false)
			check(g, true)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, true)
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			check(g, false)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, true)
			remover := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, false)
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, true)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			check(g, false)
			g.PutOnBattlefield(card, b.PlayerID())
			check(g, true)
		})
	}
}

func landwalkRuleSource(filter, operations string) string {
	return fmt.Sprintf(`Card(Name: "Rule", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentRuleEffect(%s, %s)])`, filter, operations)
}

func TestRONLandwalkRuleInstancesAndLegality(t *testing.T) {
	for _, land := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "Desert"} {
		t.Run(land, func(t *testing.T) {
			walk := fmt.Sprintf("Landwalk(IsLandType(%s))", land)
			src := landwalkRuleSource("And([IsCardtype(Creature), HasAbility("+walk+")])", "[CanBeBlockedAsThoughItDidntHave("+walk+")]")
			nodes, err := Parse(strings.NewReader(src), "rule.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first := g.PutOnBattlefield(factory(), a.PlayerID())
			second := g.PutOnBattlefield(factory(), b.PlayerID())
			kw := core.LandwalkAttr(land)
			walker := g.PutOnBattlefield(mage.NewCreature("Walker", "", 2, 2, mage.WithKeyword(kw)), a.PlayerID())
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
			g.PutOnBattlefield(mage.NewLand("Land", mage.WithSubTypes(land)), b.PlayerID())
			if mage.HasLandwalkEvasion(walker, b.PlayerID(), g) || !mage.CanBlock(blocker, walker, g) {
				t.Fatal("override did not permit an ordinary blocker")
			}
			late := g.PutOnBattlefield(mage.NewCreature("Late flyer", "", 2, 2, mage.WithKeyword(kw), mage.WithKeyword(core.Flying)), a.PlayerID())
			if mage.HasLandwalkEvasion(late, b.PlayerID(), g) || mage.CanBlock(blocker, late, g) {
				t.Fatal("override bypassed flying or ignored late entry")
			}
			otherLand := "Forest"
			if land == otherLand {
				otherLand = "Island"
			}
			dual := g.PutOnBattlefield(mage.NewCreature("Two walks", "", 2, 2, mage.WithKeyword(kw), mage.WithKeyword(core.LandwalkAttr(otherLand))), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", otherLand), b.PlayerID())
			if !mage.HasLandwalkEvasion(dual, b.PlayerID(), g) {
				t.Fatal("override suppressed another landwalk ability")
			}
			g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			if mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), b.PlayerID(), g) {
				t.Fatal("one departing instance erased the other rule")
			}
			g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
			if !mage.HasLandwalkEvasion(g.FindPermanent(walker.ID()), b.PlayerID(), g) {
				t.Fatal("departed rules remained active")
			}
		})
	}
}

func TestRONLandwalkRuleInvalid(t *testing.T) {
	walk := "Landwalk(IsLandType(Island))"
	validFilter := "And([IsCardtype(Creature), HasAbility(" + walk + ")])"
	validOp := "[CanBeBlockedAsThoughItDidntHave(" + walk + ")]"
	for _, tc := range []struct{ filter, operations string }{
		{"AnyPermanent", validOp},
		{"And([])", validOp},
		{"And([IsCardtype(Land), HasAbility(" + walk + ")])", validOp},
		{"And([IsCardtype(Creature), HasAbility(Landwalk(IsLandType(Forest)))])", validOp},
		{"And([IsCardtype(Creature), HasAbility(" + walk + "), IsColor(Blue)])", validOp},
		{"And([IsCardtype(Creature), HasAbility(" + walk + "), ControlledByAPlayer(SinglePlayer(You))])", validOp},
		{"And([IsCardtype(Creature), HasAbility(" + walk + "), Other(ThisPermanent)])", validOp},
		{"Or([IsCardtype(Creature), HasAbility(" + walk + ")])", validOp},
		{"And([IsCardtype(Creature), HasAbility(Landwalk(IsSupertype(Legendary)))])", validOp},
		{validFilter, "[]"},
		{validFilter, "[CanBeBlockedAsThoughItDidntHave]"},
		{validFilter, "[CanBeBlockedAsThoughItDidntHave(Flying)]"},
		{validFilter, "[CanBeBlockedAsThoughItDidntHave(Landwalk(IsLandType(Island())))]"},
		{validFilter, "[CanBeBlockedAsThoughItDidntHave(" + walk + ", " + walk + ")]"},
		{validFilter, "[CanBeBlockedAsThoughItDidntHave(" + walk + "), CantAttack]"},
	} {
		src := landwalkRuleSource(tc.filter, tc.operations)
		nodes, err := Parse(strings.NewReader(src), "invalid-rule.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted unsupported declaration: %s", src)
		}
	}
	for _, rule := range []string{
		"EachPermanentRuleEffect()",
		"EachPermanentRuleEffect(" + validFilter + ")",
		"PermanentRuleEffect(ThisPermanent, " + validOp + ")",
		"SpellActions(ActionList([EachPermanentRuleEffect(" + validFilter + ", " + validOp + ")]))",
	} {
		for _, cardtype := range []string{"Instant", "Enchantment"} {
			src := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [` + cardtype + `], Subtypes: []), Rules: [` + rule + `])`
			nodes, err := Parse(strings.NewReader(src), "invalid-context.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatalf("accepted invalid context: %s", rule)
			}
		}
	}
}

func TestRONLandwalkRuleInventory(t *testing.T) {
	src := landwalkRuleSource("And([IsCardtype(Creature), HasAbility(Landwalk(IsLandType(Island)))])", "[CanBeBlockedAsThoughItDidntHave(Landwalk(IsLandType(Island)))]")
	report, err := InspectFS(fstest.MapFS{"rule.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"And": "permanent_filter", "HasAbility": "permanent_filter", "CanBeBlockedAsThoughItDidntHave": "permanent_rule_operation", "Landwalk": "keyword", "IsLandType": "landwalk_filter", "Island": "land_subtype"}
	for _, entry := range report.Signatures {
		if want, ok := roles[entry.Name]; ok && entry.Role != want {
			t.Errorf("%s role = %s, want %s", entry.Name, entry.Role, want)
		}
	}
	if report.Compiled != 1 {
		t.Fatal("supported rule did not compile")
	}
}

func TestRONLandwalkRuleNativeCombat(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprint(present), func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			walker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
			walker.RevokeBaseAttr(core.AttrSummonSick)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "DRK", "Hidden Path"), a.PlayerID())
			var source *mage.Permanent
			if present {
				source = g.PutOnBattlefield(repositoryCard(t, "LEG", "Deadfall"), b.PlayerID())
			}
			a.SetAttackers(1, []string{"Llanowar Elves"})
			b.SetBlockers(1, map[string]string{"Grizzly Bears": "Llanowar Elves"})
			g.RunStepWithPriority(core.DeclareAttackers)
			g.RunStepWithPriority(core.DeclareBlockers)
			if !g.FindPermanent(walker.ID()).HasKeyword(core.Forestwalk) {
				t.Fatal("override removed the granted ability")
			}
			if present {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.RunStepWithPriority(core.CombatDamage)
			wantLife := 19
			if present {
				wantLife = 20
			}
			if b.Life() != wantLife {
				t.Fatalf("defender life = %d, want %d", b.Life(), wantLife)
			}
			if (g.FindPermanent(walker.ID()) == nil) != present {
				t.Fatal("native blocking did not follow the override")
			}
		})
	}
}
