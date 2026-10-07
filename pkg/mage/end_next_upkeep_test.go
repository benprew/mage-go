package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntilEndOfNextUpkeepBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 4, WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(Controls(HasKeywordFilter(Flying)))))), a.PlayerID())
	if err := ApplyEffect(g, UntilEndOfNextUpkeep(You, GrantKeywordModification(This, Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.RunStepWithPriority(Upkeep)
	if a.Life() != 21 || g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("effect must apply during upkeep and expire at its end")
	}
	if !clone.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("expiry changed clone")
	}
	clone.RunStepWithPriority(Upkeep)
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("clone lost duration")
	}
}

func TestUntilEndOfNextUpkeepNewEffect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 4, WithAbility(At(YourUpkeep, UntilEndOfNextUpkeep(You, GrantKeywordModification(This, Flying))))), a.PlayerID())
	g.RunStepWithPriority(Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("new upkeep effect expired in same upkeep")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	clone := g.Clone()
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(Upkeep)
	if len(g.effects.effects) == 0 {
		t.Fatal("opponent upkeep expired duration")
	}
	g.SetActivePlayerIndex(0)
	g.RunStepWithPriority(Upkeep)
	if len(g.effects.effects) != 0 || len(clone.effects.effects) == 0 {
		t.Fatal("source departure or clone broke duration")
	}
}

func TestUntilEndOfNextUpkeepDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{UntilEndOfNextUpkeep(nil, GrantKeywordModification(This, Flying)), UntilEndOfNextUpkeep(You), UntilEndOfNextUpkeep(You, nil), UntilEndOfNextUpkeep(You, ChangeController(This, You)), UntilEndOfNextUpkeep(DeclarePlayerTarget("missing"), GrantKeywordModification(This, Flying))} {
		card := NewCreature("Invalid", "", 2, 4, WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid end-upkeep declaration accepted")
		}
	}
}

func TestUntilEndOfNextUpkeepSkippedControlAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 4), a.PlayerID())
	if err := ApplyEffect(g, UntilEndOfNextUpkeep(You, GrantKeywordModification(This, Flying)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.turns.Schedule().Remaining = []PhaseStep{Upkeep, PrecombatMain}
	g.SkipNextOccurrenceOfStep(Upkeep)
	step, ok := g.turns.PopNextStep()
	if !ok || step != PrecombatMain {
		t.Fatal("skip failed")
	}
	g.RunStepWithPriority(step)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("skipped upkeep expired effect")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(Upkeep)
	if !g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("source control redirected duration")
	}
	phased := g.PhaseOut(p.ID())
	g.SetActivePlayerIndex(0)
	g.AppendExtraStep(Upkeep)
	step, ok = g.turns.PopNextStep()
	if !ok || step != Upkeep {
		t.Fatal("extra upkeep missing")
	}
	g.RunStepWithPriority(step)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).HasKeyword(Flying) {
		t.Fatal("phasing postponed expiry")
	}
}
