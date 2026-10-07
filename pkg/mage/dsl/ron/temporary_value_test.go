package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryValueCards(t *testing.T) {
	for _, name := range []string{"Howl from Beyond", "Great Defender"} {
		for _, change := range []string{"none", "returned", "phased", "phase-in"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEB", "Llanowar Elves")
				card.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(card, b.PlayerID())
				otherCard := repositoryCard(t, "LEB", "Shivan Dragon")
				otherCard.SetOwner(a.PlayerID())
				other := g.PutOnBattlefield(otherCard, a.PlayerID())
				set, x := "LEG", 0
				if name == "Howl from Beyond" {
					set, x = "LEB", 4
				}
				spell := repositoryCard(t, set, name)
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, x); err == nil {
					t.Fatal("unpaid cast succeeded")
				}
				if !g.GetStack().IsEmpty() || len(a.Hand()) != 1 {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.White, 1)
				a.ManaPool().Add(core.Colorless, 4)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, x); err != nil {
					t.Fatal(err)
				}
				if target.CurrentPower(g) != 1 || target.CurrentToughness(g) != 1 {
					t.Fatal("bonus started at announcement")
				}
				clone := g.Clone()
				phased := g.PhaseOut(target.ID())
				copySpell := g.CopySpellOnStack(spell.ID(), b.PlayerID(), true)
				if copySpell == nil || len(copySpell.Targets) != 1 || copySpell.Targets[0] != other.ID() {
					t.Fatal("copy did not use independent target")
				}
				g.PhaseIn(phased)
				current := target.ID()
				switch change {
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(current))
					b.RemoveFromHand(card.ID())
					current = g.PutOnBattlefield(card, b.PlayerID()).ID()
				case "phased":
					g.PhaseOut(current)
				case "phase-in":
					g.PhaseIn(g.PhaseOut(current))
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{current})
				p := g.FindPermanent(current)
				wantP, wantT := 1, 1
				if change == "none" || change == "phase-in" {
					if x > 0 {
						wantP += x
					} else {
						wantT++
					}
				}
				if p.CurrentPower(g) != wantP || p.CurrentToughness(g) != wantT {
					t.Fatalf("target got %d/%d, want %d/%d", p.CurrentPower(g), p.CurrentToughness(g), wantP, wantT)
				}
				cp, ct := 5, 5
				if x > 0 {
					cp += x
				} else {
					ct += 6
				}
				if g.FindPermanent(other.ID()).CurrentPower(g) != cp || g.FindPermanent(other.ID()).CurrentToughness(g) != ct {
					t.Fatal("copy lost X or read original target mana value")
				}
				clone.ResolveStack()
				if clone.FindPermanent(target.ID()).CurrentPower(clone) != 1+x {
					t.Fatal("clone lost announced X")
				}
				g.PhaseIn(g.PhaseOut(current))
				if g.FindPermanent(current).CurrentToughness(g) != wantT {
					t.Fatal("phasing erased installed bonus")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(current).CurrentPower(g) != 1 || g.FindPermanent(current).CurrentToughness(g) != 1 || g.FindPermanent(other.ID()).CurrentToughness(g) != 5 {
					t.Fatal("bonus did not expire")
				}
				if clone.FindPermanent(target.ID()).CurrentPower(clone) != 1+x {
					t.Fatal("cleanup mutated clone")
				}
			})
		}
	}
}

func TestRONTemporaryValueCaptureAndLocalScope(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEB", "Shivan Dragon")
	c.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(c, b.PlayerID())
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(0, 10), AdjustPTX(Integer(2), MinusX, ManaValueOfPermanent(ActionPermanent)), AddAbility([Flying])], UntilEndOfTurn)])))`)
	actor := factory()
	actor.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(actor, a.PlayerID())
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shares targets")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.FindPermanent(target.ID()).Card.(*mage.BaseCard).CloneFrom(repositoryCard(t, "LEB", "Llanowar Elves"))
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.ResolveStack()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 3 || g.FindPermanent(target.ID()).CurrentToughness(g) != 10 {
		t.Fatal("local mana value or signed coordinate is wrong")
	}
	g.FindPermanent(target.ID()).Card.(*mage.BaseCard).CloneFrom(repositoryCard(t, "LEB", "Shivan Dragon"))
	g.ApplyContinuousEffects()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 7 || g.FindPermanent(target.ID()).CurrentToughness(g) != 14 {
		t.Fatal("numeric bonus was not captured at resolution")
	}
	g.BouncePermanentToHand(g.FindPermanent(target.ID()))
	b.RemoveFromHand(c.ID())
	returned := g.PutOnBattlefield(c, b.PlayerID())
	if returned.CurrentPower(g) != 5 || returned.CurrentToughness(g) != 5 {
		t.Fatal("installed bonus followed a returned permanent")
	}
}

func TestRONTemporaryValueRejectsInvalidDeclarations(t *testing.T) {
	for _, part := range []string{`AdjustPTX(PlusX, Zero)`, `AdjustPTX(PlusX(), Zero, Integer(2))`, `AdjustPTX(You, Zero, Integer(2))`, `AdjustPTX(PlusX, Zero, ValueX)`, `AdjustPTX(PlusX, Zero, ManaValueOfPermanent(Ref_TargetPlayer))`, `AdjustPTX(PlusX, Zero, ManaValueOfPermanent(Ref_TargetPermanent()))`, `AdjustPTX(PlusX, Zero, ManaValueOfPermanent(ThisPermanent))`, `AdjustPTX(PlusX, Zero, PowerOfPermanent(ActionPermanent()))`} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [%s], UntilEndOfTurn)])))])`, part)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", part)
		}
	}
}

func TestRONTemporaryValueInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name: "Layer", Rules: [SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPTX(Zero, PlusX, ManaValueOfPermanent(ActionPermanent))], UntilEndOfTurn)])))])`), "layer.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Layer", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"AdjustPTX": "temporary_modification", "Zero": "pt_coordinate", "PlusX": "pt_coordinate", "ManaValueOfPermanent": "value", "ActionPermanent": "permanent_reference"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing role %s", name, role)
		}
	}
}

func TestRONTemporaryValueZeroXAndAnnouncementLegality(t *testing.T) {
	g, a, b := ronGame()
	land := repositoryCard(t, "LEB", "Forest")
	land.SetOwner(b.PlayerID())
	invalid := g.PutOnBattlefield(land, b.PlayerID())
	card := repositoryCard(t, "LEB", "Llanowar Elves")
	card.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(card, b.PlayerID())
	spell := repositoryCard(t, "LEB", "Howl from Beyond")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{invalid.ID()}, 0); err == nil {
		t.Fatal("noncreature target accepted")
	}
	if a.ManaPool().TotalMana() != 1 || !g.GetStack().IsEmpty() {
		t.Fatal("illegal cast changed payment")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 1 {
		t.Fatal("zero X changed power")
	}
}

func TestRONTemporaryValueSourceHostAndTriggerContexts(t *testing.T) {
	for _, context := range []string{"source", "host", "trigger"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEB", "Llanowar Elves")
			c.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(c, b.PlayerID())
			subject, typeline, subtypes := "ThisPermanent", "Artifact", ""
			if context == "host" {
				subject, typeline, subtypes = "HostPermanent", "Enchantment", "Aura"
			}
			effect := fmt.Sprintf(`ActionList([CreatePermanentLayerEffectUntil(%s, [AdjustPTX(PlusX, Integer(-1), Integer(2))], UntilEndOfTurn)])`, subject)
			rule := `Activated(PayMana([]), ` + effect + `)`
			if context == "host" {
				rule = `EnchantPermanent(IsCardtype(Creature)), ` + rule
			}
			if context == "trigger" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPTX(PlusX, PlusX, ManaValueOfPermanent(ActionPermanent))], UntilEndOfTurn)])))`
			}
			src := fmt.Sprintf(`Card(Name: "Numeric context", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: [%s]), Rules: [%s])`, typeline, subtypes, rule)
			nodes, err := Parse(strings.NewReader(src), "numeric.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			actor := factory()
			actor.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(actor, a.PlayerID())
			if context == "host" {
				g.Attach(source.ID(), target.ID())
			}
			if context != "trigger" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if context == "source" {
				if g.FindPermanent(target.ID()).CurrentPower(g) != 1 || g.FindPermanent(source.ID()).CurrentPower(g) != 2 || g.FindPermanent(source.ID()).CurrentToughness(g) != -1 {
					t.Fatal("source modification changed wrong permanent or amount")
				}
			} else {
				wantP, wantT := 3, 0
				if context == "trigger" {
					wantP, wantT = 2, 2
				}
				p := g.FindPermanent(target.ID())
				if wantT == 0 {
					if p != nil {
						t.Fatal("negative literal toughness did not reach state-based actions")
					}
				} else if p == nil || p.CurrentPower(g) != wantP || p.CurrentToughness(g) != wantT {
					t.Fatal("numeric trigger read wrong subject")
				}
			}
		})
	}
}
