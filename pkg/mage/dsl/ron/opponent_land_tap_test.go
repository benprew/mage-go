package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var opponentLandTapCards = []struct{ set, name, land string }{
	{"LEA", "Lifetap", "Forest"}, {"LEB", "Lifetap", "Forest"},
	{"2ED", "Lifetap", "Forest"}, {"3ED", "Lifetap", "Forest"},
	{"CED", "Lifetap", "Forest"}, {"CEI", "Lifetap", "Forest"},
	{"LEG", "Lifeblood", "Mountain"},
}

func TestRONOpponentLandTapRealCards(t *testing.T) {
	for _, tc := range opponentLandTapCards {
		for _, scenario := range []string{"tap", "own", "wrong-subtype", "nonland", "dual", "retap", "untap-retap", "entry-tapped", "source-phased", "land-phased", "phase-cycle", "source-phased-after", "source-departed", "source-returned", "land-returned", "source-control-after", "land-control-after", "source-control-before", "ability-removed", "copy"} {
			t.Run(tc.set+"/"+tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				landCard := repositoryCard(t, "LEB", tc.land)
				if scenario == "wrong-subtype" {
					landCard = repositoryCard(t, "LEB", "Island")
				}
				if scenario == "dual" {
					name := "Tropical Island"
					if tc.land == "Mountain" {
						name = "Volcanic Island"
					}
					landCard = repositoryCard(t, "LEB", name)
				}
				if scenario == "nonland" {
					landCard = mage.NewCreature("Subtype fixture", "", 1, 1, mage.WithSubTypes(tc.land))
				}
				if scenario == "entry-tapped" {
					landCard = mage.NewLand("Tapped fixture", mage.WithSubTypes(tc.land), mage.WithKeyword(core.EntersTapped))
				}
				landCard.SetOwner(a.PlayerID())
				controller := b.PlayerID()
				if scenario == "own" {
					controller = a.PlayerID()
				}
				land := g.PutOnBattlefield(landCard, controller)
				if scenario == "source-phased" {
					g.PhaseOut(source.ID())
				}
				if scenario == "land-phased" {
					g.PhaseOut(land.ID())
				}
				if scenario == "phase-cycle" {
					g.PhaseIn(g.PhaseOut(land.ID()))
				}
				if scenario == "source-control-before" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "ability-removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				g.TapPermanent(g.FindPermanent(land.ID()))
				if scenario == "retap" {
					g.TapPermanent(g.FindPermanent(land.ID()))
				}
				if scenario == "untap-retap" {
					g.UntapPermanent(g.FindPermanent(land.ID()))
					g.TapPermanent(g.FindPermanent(land.ID()))
				}
				switch scenario {
				case "source-phased-after":
					g.PhaseOut(source.ID())
				case "source-departed", "source-returned":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if scenario == "source-returned" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "land-returned":
					g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
					g.PutOnBattlefield(landCard, a.PlayerID())
				case "source-control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "land-control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				g.PutTriggersOnStack()
				want := 1
				switch scenario {
				case "own", "wrong-subtype", "nonland", "entry-tapped", "source-phased", "land-phased", "source-control-before", "ability-removed":
					want = 0
				case "untap-retap":
					want = 2
				}
				if g.StackSize() != want || a.Life() != 20 || b.Life() != 20 {
					t.Fatalf("event timing: stack=%d want=%d life=%d/%d", g.StackSize(), want, a.Life(), b.Life())
				}
				cloned := g.Clone()
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("trigger copy failed")
					}
					g.ResolveTopOfStack()
					if b.Life() != 21 || a.Life() != 20 {
						t.Fatal("copy used original controller")
					}
				}
				g.ResolveStack()
				if a.Life() != 20+want {
					t.Fatalf("life=%d want=%d", a.Life(), 20+want)
				}
				if cloned.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone shared life")
				}
				cloned.ResolveStack()
				if cloned.GetPlayer(a.PlayerID()).Life() != 20+want || cloned.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("clone lost trigger context")
				}
			})
		}
	}
}

func TestRONOpponentLandTapPaymentAndBatch(t *testing.T) {
	for _, tc := range []struct{ set, name, land string }{{"LEB", "Lifetap", "Forest"}, {"LEG", "Lifeblood", "Mountain"}} {
		for _, scenario := range []string{"mana", "paid-cast", "failed-cast", "batch", "instances"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
				want := 1
				switch scenario {
				case "mana":
					if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
						t.Fatal(err)
					}
					if b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || a.Life() != 20 {
						t.Fatal("mana trigger resolved immediately")
					}
				case "paid-cast", "failed-cast":
					g.SetActivePlayerIndex(1)
					cost := "{1}"
					if scenario == "failed-cast" {
						cost = "{2}"
						want = 0
					}
					spell := mage.NewSorcery("Payment fixture", cost, mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
					spell.SetOwner(b.PlayerID())
					b.AddToHand(spell)
					err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0)
					if scenario == "failed-cast" {
						if err == nil || g.FindPermanent(land.ID()).Tapped || len(b.Hand()) != 1 || b.ManaPool().TotalMana() != 0 {
							t.Fatal("failed payment did not roll back")
						}
					} else {
						if err != nil {
							t.Fatal(err)
						}
						if !g.FindPermanent(land.ID()).Tapped || g.StackSize() != 1 {
							t.Fatal("cast did not pay with land")
						}
					}
				case "batch":
					g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
					g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), a.PlayerID())
					if err := mage.ApplyEffect(g, mage.TapEach(mage.MatchingPermanents(mage.IsLand)), a.PlayerID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = 2
				case "instances":
					g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
					g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), b.PlayerID())
					g.TapPermanent(land)
					want = 2
				}
				g.PutTriggersOnStack()
				stack := want
				if scenario == "paid-cast" {
					stack++
				}
				if g.StackSize() != stack || a.Life() != 20 {
					t.Fatalf("stack=%d want=%d", g.StackSize(), stack)
				}
				for range want {
					g.ResolveTopOfStack()
				}
				if a.Life() != 20+want || b.Life() != 20 {
					t.Fatal("wrong trigger recipient or count")
				}
			})
		}
	}
}

func TestRONOpponentLandTapInvalidAndInventory(t *testing.T) {
	valid := `WhenAPermanentBecomesTapped(And([IsLandType(Forest), ControlledByAPlayer(Opponent)]))`
	for _, event := range []string{
		`WhenAPermanentBecomesTapped()`, `WhenAPermanentBecomesTapped(And([]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Forest)]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Goblin), ControlledByAPlayer(Opponent)]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(UnknownLand), ControlledByAPlayer(Opponent)]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType("Forest"), ControlledByAPlayer(Opponent)]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Forest()), ControlledByAPlayer(Opponent)]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Forest), ControlledByAPlayer(Opponent())]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Forest), ControlledByAPlayer(SinglePlayer(You))]))`,
		`WhenAPermanentBecomesTapped(Or([IsLandType(Forest), ControlledByAPlayer(Opponent)]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Forest), ControlledByAPlayer(Opponent), IsTapped]))`,
		`WhenAPermanentBecomesTapped(And([IsLandType(Forest), IsLandType(Mountain)]))`,
		`WhenAPermanentBecomesTapped(And([ControlledByAPlayer(Opponent), ControlledByAPlayer(Opponent)]))`,
		`WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent))`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [TriggerA(%s, ActionList([GainLife(Integer(1))]))])`, event)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", event)
		}
	}
	for _, action := range []string{`DestroyPermanent(Trigger_ThatPermanent)`, `PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))`} {
		src := fmt.Sprintf(`Card(Name: "Unbound", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [TriggerA(%s, ActionList([%s]))])`, valid, action)
		nodes, err := Parse(strings.NewReader(src), "unbound.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted unbound %s", action)
		}
	}
	src := fmt.Sprintf(`Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [TriggerA(%s, ActionList([GainLife(Integer(1))]))])`, valid)
	report, err := InspectFS(fstest.MapFS{"tap.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"And": "permanent_filter", "IsLandType": "permanent_filter", "Opponent": "player_filter"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONOpponentLandTapSubtypeAndOrder(t *testing.T) {
	for _, subtype := range []string{"Plains", "Island", "Swamp", "Mountain", "Forest", "Desert", "Gate", "Urzas", "PowerPlant"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", subtype, reverse), func(t *testing.T) {
				parts := "IsLandType(" + subtype + "), ControlledByAPlayer(Opponent)"
				if reverse {
					parts = "ControlledByAPlayer(Opponent), IsLandType(" + subtype + ")"
				}
				factory := permanentScopeFactory(t, "Enchantment", `TriggerA(WhenAPermanentBecomesTapped(And([`+parts+`])), ActionList([GainLife(Integer(2))]))`)
				g, a, b := ronGame()
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				g.PutOnBattlefield(card, a.PlayerID())
				land := g.PutOnBattlefield(mage.NewLand("Subtype fixture", mage.WithSubTypes(subtypeName(subtype))), b.PlayerID())
				g.TapPermanent(land)
				g.PutTriggersOnStack()
				g.ResolveStack()
				if a.Life() != 22 || b.Life() != 20 {
					t.Fatal("subtype or action amount changed")
				}
			})
		}
	}
}
