package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type countSacrificeChooser struct {
	*BasePlayer
	invalidAt int
	calls     int
	invalid   *Permanent
	check     func([]*Permanent, GameReader)
}

func (p *countSacrificeChooser) ChoosePermanent(candidates []*Permanent, _ string, g GameReader) *Permanent {
	p.calls++
	if p.check != nil {
		p.check(candidates, g)
	}
	if p.calls == p.invalidAt {
		return p.invalid
	}
	return candidates[0]
}

func TestSacrificeCountIncludingSourceBatch(t *testing.T) {
	a := &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	observed := 0
	var ids []uuid.UUID
	watch := func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.FromZone != core.ZoneBattlefield {
			return false
		}
		observed++
		for _, id := range ids {
			if reader.FindPermanent(id) == nil {
				t.Fatal("trigger condition lost simultaneous battlefield")
			}
		}
		for _, id := range ids {
			if g.FindPermanent(id) != nil {
				t.Fatal("event saw partial sacrifice")
			}
		}
		return true
	}
	sourceCard := NewCreature("Source", "", 2, 2, WithAbility(Activate(Costs(SacrificeNMatchingIncludingSourceCost(2, IsCreature, "Sacrifice two")), GainLifeFor(You, Fixed(3)))), WithAbility(NewTriggered(core.EvtZoneChange, false, GainLifeFor(You, Fixed(1))).SetCondition(watch)), WithAbility(NewTriggered(core.EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(watch)))
	sourceCard.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	otherCard := NewCreature("Other", "", 1, 1, WithKeyword(core.Indestructible))
	otherCard.SetOwner(b.PlayerID())
	other := g.PutOnBattlefield(otherCard, a.PlayerID())
	ids = []uuid.UUID{source.ID(), other.ID()}
	observed = 0
	g.AddRegenerationShield(other.ID())
	a.check = func(_ []*Permanent, reader GameReader) {
		for _, id := range ids {
			if reader.FindPermanent(id) == nil {
				t.Fatal("choice removed a sacrifice before all choices")
			}
		}
	}
	clone := g.Clone()
	if err := SacrificeNMatchingIncludingSourceCost(2, IsCreature, "Sacrifice two").Pay(source.ID(), a.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	if a.calls != 2 || len(b.Graveyard()) != 2 || len(a.Graveyard()) != 0 || a.Life() != 20 || observed != 4 {
		t.Fatalf("choices=%d graves=%d/%d life=%d events=%d", a.calls, len(a.Graveyard()), len(b.Graveyard()), a.Life(), observed)
	}
	for _, id := range ids {
		if clone.FindPermanent(id) == nil {
			t.Fatal("payment changed clone")
		}
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("source ability or simultaneous triggers lost", a.Life())
	}
}

func TestSacrificeCountIncludingSourceRollback(t *testing.T) {
	for _, scenario := range []string{"missing", "phased", "opponent", "second-invalid", "returned", "control", "stale-phase"} {
		t.Run(scenario, func(t *testing.T) {
			a := &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
			otherCard := NewCreature("Other", "", 1, 1)
			other := g.PutOnBattlefield(otherCard, a.PlayerID())
			a.ManaPool().Add(core.Red, 1)
			switch scenario {
			case "missing":
				g.RemoveFromBattlefield(other)
			case "phased":
				g.PhaseOut(other.ID())
			case "opponent":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: other.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "second-invalid":
				a.invalidAt = 2
				a.invalid = source
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{ManaCostOf("{R}"), SacrificeNMatchingIncludingSourceCost(2, IsCreature, "Sacrifice two")}})
			if err == nil {
				switch scenario {
				case "returned":
					g.RemoveFromBattlefield(other)
					g.PutOnBattlefield(otherCard, a.PlayerID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: other.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "stale-phase":
					g.PhaseOut(other.ID())
				default:
					t.Fatal("invalid preparation accepted")
				}
				err = tx.Commit()
			}
			if err == nil || g.FindPermanent(source.ID()) == nil || a.ManaPool().TotalMana() != 1 || len(a.Graveyard()) != 0 {
				t.Fatal("failed cost changed resources", err)
			}
		})
	}
}

func TestSacrificeCountIncludingSourceInvalid(t *testing.T) {
	for _, n := range []int{-1, 0} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(SacrificeNMatchingIncludingSourceCost(n, IsCreature, "Sacrifice")), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid count accepted", n)
		}
	}
}

func TestSacrificeCountIncludingSourceActivation(t *testing.T) {
	a := &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := NewCreature("Source", "", 2, 2, WithAbility(Activate(Costs(ManaCostOf("{R}"), SacrificeNMatchingIncludingSourceCost(2, IsCreature, "Sacrifice two")), GainLifeFor(You, Fixed(3)))))
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	a.ManaPool().Add(core.Red, 1)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.calls != 2 || g.FindPermanent(source.ID()) != nil || a.Life() != 20 || g.StackSize() != 1 {
		t.Fatal("source payment timing")
	}
	clone := g.Clone()
	g.PutOnBattlefield(card, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("pending effect followed returned source")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("live effect changed clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone lost paid ability")
	}
}

func TestSacrificeCountIncludingSourceSingleStaleRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 1), a.PlayerID())
	a.ManaPool().Add(core.Red, 1)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{ManaCostOf("{R}"), SacrificeNMatchingIncludingSourceCost(1, IsCreature, "Sacrifice one")}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(victim.ID())
	if err := tx.Commit(); err == nil || a.ManaPool().TotalMana() != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("single-count stale payment spent resources")
	}
}
