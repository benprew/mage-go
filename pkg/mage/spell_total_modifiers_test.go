package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellTotalModifiersAdditionalAndX(t *testing.T) {
	for _, route := range []string{"hand", "alternate"} {
		for _, component := range []string{"additional", "action", "per target", "X"} {
			t.Run(route+component, func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				targets := DeclareAnyNumberDamageTargets("recipients")
				spell := NewInstant("Reduced", "{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(targets)))
				x := 0
				switch component {
				case "additional":
					WithAdditionalCost(GenericCost(2))(spell)
				case "action":
					spell = NewInstant("Reduced", "{R}", NewSpell(GainLifeFor(You, Fixed(1)), WithTarget(TargetUpToNCreaturesOrPlayers(2)), WithCost(GenericCost(2))))
				case "per target":
					WithAdditionalCost(ManaCostPerTarget("{1}"))(spell)
				case "X":
					spell = NewInstant("Reduced", "{X}{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(targets)))
					x = 2
				}
				spell.SetOwner(a.PlayerID())
				a.ManaPool().Add(Red, 1)
				g.effects.Rules.SpellCostReductions[Red] = 2
				ids := []uuid.UUID{a.PlayerID(), b.PlayerID()}
				var err error
				if route == "alternate" {
					a.AddToGraveyard(spell)
					err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneGraveyard, spell.ManaCost(), ids, x)
				} else {
					a.AddToHand(spell)
					err = g.CastSpellByName(a.PlayerID(), spell.Name(), ids, x)
				}
				if err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != x {
					t.Fatal("wrong payment or X")
				}
			})
		}
	}
}

func TestSpellTotalModifiersOrderAndLock(t *testing.T) {
	for _, kind := range []string{"color", "type", "conditional", "target conditional"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			target := DeclarePlayerTarget("player")
			spell := NewInstant("Order", "{X}{R}", Spell(GainLifeFor(You, Fixed(1)), WithTargets(target)), WithAdditionalCost(GenericCost(2)))
			g.effects.Rules.SpellCostIncreases[Red] = 1
			switch kind {
			case "color":
				g.effects.Rules.SpellCostReductions[Red] = 3
			case "type":
				g.AddSpellTypeCostReduction(TypeInstant, 3)
			case "conditional":
				WithSelfCostReduction(FixedAmount(3), nil)(spell)
			case "target conditional":
				WithTargetConditionalCostReduction(3, func(_ *Game, controller uuid.UUID, _ Card, targets []uuid.UUID) bool {
					return controller == a.PlayerID() && len(targets) == 1 && targets[0] == b.PlayerID()
				})(spell)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Colorless, 2)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 2); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong total after increases and reductions")
			}
			clone := g.Clone()
			copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			if copied.XValue != 2 || g.StackPeek().XValue != 2 {
				t.Fatal("cost changes altered announced X")
			}
			clone.effects.Rules.SpellCostReductions[Red] = 99
			if g.effects.Rules.SpellCostReductions[Red] == 99 {
				t.Fatal("clone shared cost changes")
			}
		})
	}
}

func TestSpellTotalModifiersPaymentAndChannel(t *testing.T) {
	for _, state := range []string{"payable", "underpaid", "channel", "channel unpayable"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			spell := NewInstant("Total", "{X}{R}", Spell(DealDamageEvenly(This, targets, ValueAtResolution(XValue())), WithTargets(targets)), WithAdditionalCost(ManaCostPerTarget("{1}")), WithAdditionalCost(LifePayCost(2)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			if state == "payable" {
				a.ManaPool().Add(Colorless, 3)
			}
			if state == "underpaid" {
				a.ManaPool().Add(Colorless, 2)
			}
			if state == "channel unpayable" {
				a.SetLife(4)
			}
			beforeLife, beforeMana := a.Life(), a.ManaPool().TotalMana()
			events := 0
			for _, event := range []EventType{EvtSpellCast, EvtLifeLost, EvtBecomesTarget, EvtTapped} {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
					if reader == g {
						events++
					}
					return false
				}))), a.PlayerID())
			}
			g.effects.Rules.SpellCostReductions[Red] = 2
			if state == "channel" || state == "channel unpayable" {
				g.effects.Rules.SetChannelActive(a.PlayerID())
			}
			err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{a.PlayerID(), b.PlayerID()}, 3)
			bad := state == "underpaid" || state == "channel unpayable"
			if (err != nil) != bad {
				t.Fatalf("cast=%v, bad=%v", err, bad)
			}
			if bad {
				if a.Life() != beforeLife || a.ManaPool().TotalMana() != beforeMana || g.StackSize() != 0 || len(a.Hand()) != 1 || events != 0 {
					t.Fatal("failed total cost changed resources or events")
				}
				return
			}
			wantLife := 18
			if state == "channel" {
				wantLife = 15
			}
			if a.Life() != wantLife || a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong locked life or mana payment")
			}
		})
	}
}

func TestSpellTotalModifiersLockedBeforeSacrifice(t *testing.T) {
	g, a, _ := newActivationDivisionGame()
	reducer := g.PutOnBattlefield(NewCreature("Reducer", "", 1, 1), a.PlayerID())
	spell := NewInstant("Sacrifice", "{R}", Spell(GainLifeFor(You, Fixed(1))),
		WithAdditionalCost(GenericCost(2)), WithAdditionalCost(SacrificeNMatchingCost(1, IsCreature, "Sacrifice a creature")),
		WithSelfCostReduction(FixedAmount(2), CondControlsMatching(IsCreature)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Red, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(reducer.ID()) != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("cost changed after its reducer left")
	}
}
