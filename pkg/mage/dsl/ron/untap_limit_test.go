package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONUntapLimitsRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		artifact  bool
	}{{"LEA", "Smoke", false}, {"ATQ", "Damping Field", true}} {
		for _, scenario := range []string{"normal", "source-left", "source-phase", "source-abilities", "recipient-abilities", "clone", "control", "phase-in", "return"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				_, a, b := ronGame()
				g := mage.NewGame(&ronPaymentPlayer{BasePlayer: a, accept: true}, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(card, b.PlayerID())
				makeSubject := func(name string) mage.Card {
					if tc.artifact {
						return mage.NewArtifact(name, "")
					}
					return mage.NewCreature(name, "", 2, 2)
				}
				first := g.PutOnBattlefield(makeSubject("First"), a.PlayerID())
				second := g.PutOnBattlefield(makeSubject("Second"), a.PlayerID())
				other := g.PutOnBattlefield(makeSubject("Other"), b.PlayerID())
				for _, p := range []*mage.Permanent{first, second, other} {
					g.TapPermanent(p)
				}
				blocked := true
				switch scenario {
				case "source-left":
					g.DestroyPermanent(source)
					blocked = false
				case "source-phase":
					g.PhaseOut(source.ID())
					blocked = false
				case "source-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					blocked = false
				case "recipient-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), second.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phase-in":
					g.PhaseOut(second.ID())
				case "return":
					g.BouncePermanentToHand(source)
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				clone := g.Clone()
				if scenario == "clone" {
					clone.DestroyPermanent(clone.FindPermanent(source.ID()))
					clone.RunStepWithPriority(core.Untap)
					if clone.FindPermanent(second.ID()).Tapped {
						t.Fatal("clone kept departed rule")
					}
				}
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped != blocked || !g.FindPermanent(other.ID()).Tapped {
					t.Fatal("incorrect untap set")
				}
				if scenario != "clone" && !clone.FindPermanentIncludingPhased(first.ID()).Tapped {
					t.Fatal("clone changed")
				}
				if blocked && !g.UntapPermanent(g.FindPermanent(second.ID())) {
					t.Fatal("other untap effect prohibited")
				}
			})
		}
	}
}

func TestRONUntapLimitsInvalid(t *testing.T) {
	for _, rule := range []string{
		`EachPlayerEffect(AnyPlayer(),[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),IsCardtype(Creature))])`,
		`EachPlayerEffect(SinglePlayer(You),[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),IsCardtype(Creature))])`,
		`EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(ValueX,IsCardtype(Creature))])`,
		`EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(-1),IsCardtype(Creature))])`,
		`EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1,2),IsCardtype(Creature))])`,
		`EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),SinglePermanent(Ref_TargetPermanent))])`,
		`EachPlayerEffect(AnyPlayer,[])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONUntapLimitsCasting(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
		amount    int
	}{{"LEA", "Smoke", core.Red, 2}, {"ATQ", "Damping Field", core.White, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting did not roll back")
			}
			a.ManaPool().Add(tc.color, tc.amount)
			if tc.name == "Damping Field" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("casting payment or timing")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("enchantment did not resolve")
			}
		})
	}
}

func TestRONUntapLimitsInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),IsCardtype(Creature))])])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"AnyPlayer": "player_filter", "CantUntapMoreThanNumberPermanentsDuringTheirUntapStep": "player_rule_operation", "Integer": "value", "IsCardtype": "permanent_filter"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONUntapLimitsReprints(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Smoke" && name != "Damping Field" {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, factory()); err != nil {
				t.Fatal(err)
			}
			count++
		}
	}
	if count != 7 {
		t.Fatalf("records=%d", count)
	}
}
