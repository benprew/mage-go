package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetSurchargeCountAndPayment(t *testing.T) {
	for _, route := range []string{"hand", "alternate"} {
		for count := 0; count <= 3; count++ {
			t.Run(route+string(rune('0'+count)), func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				recipients := DeclareAnyNumberDamageTargets("recipients")
				spell := NewInstant("Surcharge", "{X}{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(recipients), AdditionalManaPerTargetBeyondFirst(2)))
				if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
					t.Fatal(err)
				}
				spell.SetOwner(a.PlayerID())
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
				ids := []uuid.UUID{a.PlayerID(), b.PlayerID(), victim.ID()}[:count]
				want := 3 + 2*max(0, count-1)
				a.ManaPool().Add(Red, 1)
				a.ManaPool().Add(Colorless, want)
				var err error
				if route == "hand" {
					a.AddToHand(spell)
					err = g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 3)
				} else {
					a.AddToGraveyard(spell)
					err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneGraveyard, spell.ManaCost(), ids, 3)
				}
				if err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != 3 {
					t.Fatal("wrong payment or announced X")
				}
			})
		}
	}
}

func TestTargetSurchargeGroupsAndReductions(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	first, second := DeclarePlayerTarget("first"), DeclarePlayerTarget("second")
	spell := NewInstant("Groups", "{X}{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargetGroups(first, second), AdditionalManaPerTargetBeyondFirst(2)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	g.effects.Rules.SpellCostIncreases[Red] = 1
	g.effects.Rules.SpellCostReductions[Red] = 3
	a.ManaPool().Add(Red, 1)
	a.ManaPool().Add(Colorless, 4)
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{b.PlayerID()}, {b.PlayerID()}}, 4); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("cost did not count both target occurrences before reduction")
	}
}

func TestTargetSurchargeRollback(t *testing.T) {
	for _, state := range []string{"underpaid", "illegal", "phased"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			spell := NewInstant("Rollback", "{X}{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(targets), AdditionalManaPerTargetBeyondFirst(1)), WithAdditionalCost(LifePayCost(2)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Colorless, 2)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{b.PlayerID(), victim.ID()}
			if state == "illegal" {
				ids[1] = uuid.New()
			}
			if state == "phased" {
				g.PhaseOut(victim.ID())
			}
			events := 0
			for _, event := range []EventType{EvtSpellCast, EvtLifeLost, EvtBecomesTarget, EvtTapped} {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
					if reader == g {
						events++
					}
					return false
				}))), a.PlayerID())
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 2); err == nil {
				t.Fatal("invalid cast succeeded")
			}
			if a.Life() != 20 || a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 || events != 0 {
				t.Fatal("failed cast changed live resources or events")
			}
		})
	}
}

func TestTargetSurchargeCopiesAndLegality(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			spell := NewInstant("Copies", "{X}{R}", Spell(DealDamageEvenly(This, targets, ValueAtResolution(XValue())), WithTargets(targets), AdditionalManaPerTargetBeyondFirst(1)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Colorless, 5)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), victim.ID()}, 4); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			p := clone.FindPermanent(victim.ID())
			if state == "phased" {
				clone.PhaseOut(p.ID())
			} else {
				clone.RemoveFromBattlefield(p)
				if state == "returned" {
					clone.PutOnBattlefield(p.Card, b.PlayerID())
				}
			}
			clone.ResolveStackObject(copyObj)
			if clone.GetPlayer(b.PlayerID()).Life() != 16 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("copy did not divide again without payment")
			}
			if b.Life() != 20 || victim.Damage != 0 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if b.Life() != 18 || g.FindPermanent(victim.ID()).Damage != 2 || a.ManaPool().TotalMana() != 0 {
				t.Fatalf("original life=%d damage=%d mana=%d", b.Life(), g.FindPermanent(victim.ID()).Damage, a.ManaPool().TotalMana())
			}
		})
	}
}

func TestTargetSurchargeDeclarations(t *testing.T) {
	for _, amount := range []int{-1, 0} {
		card := NewInstant("Invalid", "", Spell(GainLifeFor(You, Fixed(1)), AdditionalManaPerTargetBeyondFirst(amount)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "positive") {
			t.Fatalf("invalid amount: %v", err)
		}
	}
	option := AdditionalManaPerTargetBeyondFirst(1)
	card := NewInstant("Valid", "", Spell(GainLifeFor(You, Fixed(1)), option))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	metadata := InspectDefinition(card.Abilities()[0])
	found := false
	for _, node := range metadata.Children {
		if node.Role == "additional-mana-per-target-beyond-first" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing cost inspection")
	}
}

func TestTargetSurchargeReusableDeclarationAndLock(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	option := AdditionalManaPerTargetBeyondFirst(1)
	targets := DeclareAnyNumberDamageTargets("recipients")
	reducer := g.PutOnBattlefield(NewCreature("Reducer", "", 1, 1), a.PlayerID())
	spell := NewInstant("Locked", "{X}{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(targets), option),
		WithAdditionalCost(SacrificeNMatchingCost(1, IsCreature, "Sacrifice a creature")),
		WithSelfCostReduction(FixedAmount(1), CondControlsMatching(IsCreature)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Red, 1)
	a.ManaPool().Add(Colorless, 2)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{a.PlayerID(), b.PlayerID()}, 2); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(reducer.ID()) != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("cost was not locked before sacrifice")
	}
	other := NewInstant("Other", "{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(targets), option), WithAdditionalCost(LifePayCost(2)))
	other.SetOwner(a.PlayerID())
	a.AddToHand(other)
	a.ManaPool().Add(Red, 1)
	if err := g.CastSpellByName(a.PlayerID(), other.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 2 {
		t.Fatal("second instance reused the first target count")
	}
}
