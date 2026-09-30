package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func modalSurchargeCard(option ModalSpellOption) Card {
	targets := DeclareAnyNumberDamageTargets("recipients")
	return NewInstant("Modal surcharge", "{X}{R}", WithAbility(SpellModesWithOptions(
		[]ModalSpellOption{option},
		ModeOf("Damage", DealDamageEvenly(This, targets, ValueAtResolution(XValue())), WithTargets(targets)),
		ModeOf("Life", GainLifeFor(You, Fixed(3))))))
}

func TestModalTargetSurchargeSelectedCount(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		for mode := range 2 {
			for count := 0; count <= 3; count++ {
				if mode == 1 && count != 0 {
					continue
				}
				t.Run(fmtSurchargeCase(alternate, mode, count), func(t *testing.T) {
					a := &allocatedModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}, mode: mode}
					b := newRecordingPlayer("B")
					g := NewGame(a, b)
					card := modalSurchargeCard(AdditionalManaPerTargetBeyondFirst(2))
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
					ids := []uuid.UUID{a.PlayerID(), b.PlayerID(), victim.ID()}[:count]
					a.chooseQueue = [][]uuid.UUID{ids}
					a.ManaPool().Add(Red, 1)
					a.ManaPool().Add(Colorless, 4+2*max(0, count-1))
					var err error
					if alternate {
						a.AddToGraveyard(card)
						err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), ZoneGraveyard, card.ManaCost(), ids, 4)
					} else {
						a.AddToHand(card)
						err = g.CastSpellByName(a.PlayerID(), card.Name(), ids, 4)
					}
					if err != nil {
						t.Fatal(err)
					}
					if a.ManaPool().TotalMana() != 0 || g.StackPeek().ModeChoice != mode {
						t.Fatal("wrong mode or selected-target payment")
					}
					g.ResolveStack()
					if mode == 1 && (a.Life() != 23 || b.Life() != 20) {
						t.Fatal("inactive damage mode resolved")
					}
				})
			}
		}
	}
}

func fmtSurchargeCase(alternate bool, mode, count int) string {
	route := "hand"
	if alternate {
		route = "alternate"
	}
	return route + string(rune('0'+mode)) + string(rune('0'+count))
}

func TestModalTargetSurchargeDeclarations(t *testing.T) {
	for _, amount := range []int{-1, 0, 1} {
		card := modalSurchargeCard(AdditionalManaPerTargetBeyondFirst(amount))
		err := ValidateCardDefinition(card.Name(), card)
		if amount <= 0 && (err == nil || !strings.Contains(err.Error(), "positive")) {
			t.Fatalf("invalid amount: %v", err)
		}
		if amount > 0 && err != nil {
			t.Fatal(err)
		}
		found := false
		for _, node := range InspectDefinition(card.Abilities()[0]).Children {
			found = found || node.Role == "additional-mana-per-target-beyond-first"
		}
		if !found {
			t.Fatal("missing modal surcharge inspection")
		}
	}
}

func TestModalTargetSurchargeRollback(t *testing.T) {
	for _, state := range []string{"underpaid", "illegal", "phased", "duplicate", "unpaid-life"} {
		t.Run(state, func(t *testing.T) {
			a := &allocatedModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}}
			b := newRecordingPlayer("B")
			g := NewGame(a, b)
			card := modalSurchargeCard(AdditionalManaPerTargetBeyondFirst(1))
			WithAdditionalCost(LifePayCost(2))(card.(*BaseCard))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{b.PlayerID(), victim.ID()}
			a.ManaPool().Add(Red, 1)
			generic := 5
			if state == "underpaid" {
				generic = 4
			}
			a.ManaPool().Add(Colorless, generic)
			switch state {
			case "illegal":
				ids[1] = uuid.New()
			case "phased":
				g.PhaseOut(victim.ID())
			case "duplicate":
				ids[1] = ids[0]
			case "unpaid-life":
				a.SetLife(1)
			}
			life := a.Life()
			events := 0
			for _, event := range []EventType{EvtSpellCast, EvtLifeLost, EvtBecomesTarget, EvtTapped} {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(event, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
					if reader == g {
						events++
					}
					return false
				}))), a.PlayerID())
			}
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), ids, 4); err == nil {
				t.Fatal("invalid modal cast succeeded")
			}
			if a.Life() != life || a.ManaPool().TotalMana() != generic+1 || len(a.Hand()) != 1 || g.StackSize() != 0 || events != 0 {
				t.Fatal("failed modal cast changed resources or published events")
			}
		})
	}
}

func TestModalTargetSurchargeCopiesAndLock(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			a := &allocatedModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}}
			b := newRecordingPlayer("B")
			g := NewGame(a, b)
			option := AdditionalManaPerTargetBeyondFirst(2)
			card := modalSurchargeCard(option)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			g.effects.Rules.SpellCostIncreases[Red] = 1
			g.effects.Rules.SpellCostReductions[Red] = 2
			a.ManaPool().Add(Red, 1)
			a.ManaPool().Add(Colorless, 5)
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{b.PlayerID(), victim.ID()}, 4); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("modal cost modifiers did not apply to combined total")
			}
			clone := g.Clone()
			copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			p := clone.FindPermanent(victim.ID())
			if state == "phased" {
				clone.PhaseOut(p.ID())
			} else {
				clone.RemoveFromBattlefield(p)
				if state == "returned" {
					clone.PutOnBattlefield(p.Card, b.PlayerID())
				}
			}
			clone.ResolveStackObject(copied)
			if copied.ModeChoice != 0 || clone.GetPlayer(b.PlayerID()).Life() != 16 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("copy mode, legality, or unpaid copy contract failed")
			}
			if b.Life() != 20 || victim.Damage != 0 {
				t.Fatal("clone changed original")
			}
			a.mode = 1
			other := modalSurchargeCard(option)
			other.SetOwner(a.PlayerID())
			a.AddToHand(other)
			a.ManaPool().Add(Red, 1)
			if err := g.CastSpellByName(a.PlayerID(), other.Name(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 2 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("second pending mode reused target count")
			}
			g.ResolveStack()
			g.ResolveStack()
			if a.Life() != 23 || b.Life() != 18 || g.FindPermanent(victim.ID()).Damage != 2 {
				t.Fatalf("original life A=%d B=%d damage=%d", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
			}
		})
	}
}

func TestModalTargetSurchargeNilOption(t *testing.T) {
	card := modalSurchargeCard(nil)
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil modal option accepted")
	}
}

func TestModalTargetSurchargeOptionIsolation(t *testing.T) {
	a := &allocatedModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}}
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	targets := DeclareAnyNumberDamageTargets("recipients")
	options := []ModalSpellOption{AdditionalManaPerTargetBeyondFirst(1), AdditionalManaPerTargetBeyondFirst(2)}
	ability := SpellModesWithOptions(options,
		ModeOf("Damage", DealDamageEvenly(This, targets, Fixed(4)), WithTargets(targets)),
		ModeOf("Life", GainLifeFor(You, Fixed(3))))
	options[0] = nil
	card := NewInstant("Immutable modal options", "{R}", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Red, 1)
	a.ManaPool().Add(Colorless, 3)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{a.PlayerID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("options did not add independent surcharges")
	}
}
