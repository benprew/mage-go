package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControlledStaticCards(t *testing.T) {
	for _, tc := range []struct {
		set, name, creature string
		power, toughness    int
		keyword             core.Attr
	}{
		{"LEB", "Castle", "Llanowar Elves", 1, 3, 0},
		{"LEB", "Orcish Oriflamme", "Llanowar Elves", 2, 1, 0},
		{"LEG", "Fortified Area", "Wall of Wood", 1, 3, core.Banding},
		{"FEM", "Goblin War Drums", "Llanowar Elves", 1, 1, core.Menace},
		{"LEG", "Jacques le Vert", "Llanowar Elves", 1, 3, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			put := func(set, name string, controller mage.Player) *mage.Permanent {
				card := repositoryCard(t, set, name)
				card.SetOwner(b.PlayerID())
				return g.PutOnBattlefield(card, controller.PlayerID())
			}
			own := put("LEB", tc.creature, a)
			opposing := put("LEB", tc.creature, b)
			if tc.name == "Orcish Oriflamme" {
				g.GetCombat().AddAttacker(own.ID(), b.PlayerID())
				g.GetCombat().AddAttacker(opposing.ID(), a.PlayerID())
			}
			source := put(tc.set, tc.name, a)
			check := func(game *mage.Game, active bool) {
				t.Helper()
				target := game.FindPermanent(own.ID())
				baseP, baseT := 1, 1
				if tc.creature == "Wall of Wood" {
					baseP, baseT = 0, 3
				}
				wantP, wantT := baseP, baseT
				if active {
					wantP, wantT = tc.power, tc.toughness
				}
				if target.CurrentPower(game) != wantP || target.CurrentToughness(game) != wantT {
					t.Fatalf("own creature P/T = %d/%d, want %d/%d", target.CurrentPower(game), target.CurrentToughness(game), wantP, wantT)
				}
				if tc.keyword != 0 && target.HasKeyword(tc.keyword) != active {
					t.Fatal("keyword membership mismatch")
				}
			}
			check(g, true)
			if g.FindPermanent(opposing.ID()).CurrentPower(g) != opposing.Card.Power() || g.FindPermanent(opposing.ID()).CurrentToughness(g) != opposing.Card.Toughness() {
				t.Fatal("opposing creature received boost")
			}
			if tc.keyword != 0 && g.FindPermanent(opposing.ID()).HasKeyword(tc.keyword) {
				t.Fatal("opposing creature received keyword")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
			check(clone, false)
			check(g, true)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, false)
			if tc.keyword != 0 && !g.FindPermanent(opposing.ID()).HasKeyword(tc.keyword) {
				t.Fatal("grant did not follow source controller")
			}
			if tc.name == "Jacques le Vert" && g.FindPermanent(source.ID()).CurrentToughness(g) != 4 {
				t.Fatal("source excluded from its own green creature boost")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if tc.name == "Orcish Oriflamme" {
				g.GetCombat().AddAttacker(own.ID(), a.PlayerID())
				g.ApplyContinuousEffects()
			}
			check(g, true)
			phased := g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			check(g, false)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, true)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			check(g, false)
			returned := g.PutOnBattlefield(source.Card, b.PlayerID())
			check(g, true)
			g.RemoveFromBattlefield(returned)
			check(g, false)
		})
	}
}

func TestRONControlledStaticRejectsInvalidFilters(t *testing.T) {
	for _, filter := range []string{
		`ControlledByAPlayer(SinglePlayer(You()))`,
		`ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))`,
		`ControlledByAPlayer(SinglePlayer(ActionPlayer))`,
		`ControlledByAPlayer(SinglePlayer(HostController))`,
		`ControlledByAPlayer(You)`,
		`ControlledByAPlayer(SinglePlayer(You), SinglePlayer(You))`,
		`Or([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You))])`,
		`And([])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid static", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(And([IsCardtype(Creature), %s]), [AdjustPT(1, 1)])])`, filter)
		nodes, err := Parse(strings.NewReader(src), "static-invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid static filter: %s", filter)
		}
	}
}

func TestRONControlledStaticMembership(t *testing.T) {
	for _, name := range []string{"Castle", "Jacques le Vert", "Orcish Oriflamme", "Fortified Area", "Goblin War Drums"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEG"
			if name == "Castle" || name == "Orcish Oriflamme" {
				set = "LEB"
			}
			if name == "Goblin War Drums" {
				set = "FEM"
			}
			sourceCard := repositoryCard(t, set, name)
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			card := repositoryCard(t, "LEB", "Llanowar Elves")
			if name == "Fortified Area" {
				card = repositoryCard(t, "LEB", "Wall of Wood")
			}
			card.SetOwner(a.PlayerID())
			target := g.PutOnBattlefield(card, a.PlayerID())
			switch name {
			case "Castle":
				if target.CurrentToughness(g) != 3 {
					t.Fatal("late creature did not receive boost")
				}
				g.TapPermanent(target)
				g.ApplyContinuousEffects()
				if g.FindPermanent(target.ID()).CurrentToughness(g) != 1 {
					t.Fatal("tapped creature retained boost")
				}
				g.UntapPermanent(g.FindPermanent(target.ID()))
				g.ApplyContinuousEffects()
				if g.FindPermanent(target.ID()).CurrentToughness(g) != 3 {
					t.Fatal("untapped creature did not regain boost")
				}
			case "Jacques le Vert":
				if target.CurrentToughness(g) != 3 {
					t.Fatal("late green creature did not receive boost")
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue)), target.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(target.ID()).CurrentToughness(g) != 1 {
					t.Fatal("color change did not change membership")
				}
			case "Orcish Oriflamme":
				if target.CurrentPower(g) != 1 {
					t.Fatal("nonattacker received boost")
				}
				g.GetCombat().AddAttacker(target.ID(), b.PlayerID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(target.ID()).CurrentPower(g) != 2 {
					t.Fatal("attacker did not receive boost")
				}
				g.RemoveFromCombat(target.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(target.ID()).CurrentPower(g) != 1 {
					t.Fatal("combat removal retained boost")
				}
			case "Fortified Area", "Goblin War Drums":
				kw := core.Banding
				if name == "Goblin War Drums" {
					kw = core.Menace
				}
				if !target.HasKeyword(kw) {
					t.Fatal("late creature did not receive keyword")
				}
				phased := g.PhaseOut(target.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(target.ID()) != nil {
					t.Fatal("phased recipient remained visible")
				}
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				if !g.FindPermanent(target.ID()).HasKeyword(kw) {
					t.Fatal("phased-in recipient lost keyword")
				}
				nonWallCard := repositoryCard(t, "LEB", "Llanowar Elves")
				nonWall := g.PutOnBattlefield(nonWallCard, a.PlayerID())
				if name == "Fortified Area" && (nonWall.HasKeyword(kw) || nonWall.CurrentPower(g) != 1) {
					t.Fatal("non-Wall received Wall bonus")
				}
			}
			if name != "Orcish Oriflamme" {
				removalCard := repositoryCard(t, "LEB", "Forest")
				removal := g.PutOnBattlefield(removalCard, a.PlayerID())
				effect := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
				effect.SetSourceID(removal.ID())
				g.AddContinuousEffect(effect)
				if name == "Fortified Area" || name == "Goblin War Drums" {
					kw := core.Banding
					if name == "Goblin War Drums" {
						kw = core.Menace
					}
					if g.FindPermanent(target.ID()).HasKeyword(kw) {
						t.Fatal("removed static grant still applied")
					}
				} else if g.FindPermanent(target.ID()).CurrentToughness(g) != 1 {
					t.Fatal("removed static ability still applied")
				}
				g.RemoveFromBattlefield(g.FindPermanent(removal.ID()))
				if name == "Castle" && g.FindPermanent(target.ID()).CurrentToughness(g) != 3 {
					t.Fatal("restored static ability did not apply")
				}
			}
		})
	}
}

func TestRONControlledStaticNestedConjunctionAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(And([IsCardtype(Creature), And([ControlledByAPlayer(SinglePlayer(You)), Or([IsColor(Green), IsColor(White)])])]), [AdjustPT(1, 2), AddAbility([Flying])])`)
	g, a, b := ronGame()
	first, second := factory(), factory()
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	g.PutOnBattlefield(first, a.PlayerID())
	source := g.PutOnBattlefield(second, b.PlayerID())
	targetCard := repositoryCard(t, "LEB", "Llanowar Elves")
	target := g.PutOnBattlefield(targetCard, a.PlayerID())
	if target.CurrentPower(g) != 2 || target.CurrentToughness(g) != 3 || !target.HasKeyword(core.Flying) {
		t.Fatal("nested mixed layer failed")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(target.ID()).CurrentPower(g) != 3 {
		t.Fatal("separate instances did not accumulate")
	}
}

func TestRONControlledStaticInventoryRoles(t *testing.T) {
	src := `Card(Name: "Inventory static", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [EachPermanentLayerEffect(And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You))]), [AdjustPT(1, 1)])])`
	report, err := InspectFS(fstest.MapFS{"static.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, entry := range report.Signatures {
		if entry.Name == "You" {
			seen = true
			if entry.Role != "player_reference" {
				t.Fatalf("controller operand role = %s", entry.Role)
			}
		}
	}
	if !seen {
		t.Fatal("controller reference missing from inventory")
	}
}
