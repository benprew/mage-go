package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTappedUntapRuleLifetime(t *testing.T) {
	for _, change := range []string{"none", "untap", "retap", "stun", "phase", "phase-cycle", "source-return", "target-return", "target-phase-cycle", "control", "remove-abilities", "simultaneous"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("subject", AnyPermanent)
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), Sequence(TapSubject(target), PreventUntapWhileSourceTapped(target)), WithTargets(target))))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			subjectCard := NewArtifact("Subject", "")
			subjectCard.SetOwner(b.PlayerID())
			subject := g.PutOnBattlefield(subjectCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{subject.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			clone := g.Clone()
			blocked := true
			switch change {
			case "untap", "retap":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				blocked = false
				if change == "retap" {
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			case "stun":
				g.MutablePermanent(source.ID()).Counters[core.Stun] = 1
				if g.UntapPermanent(g.FindPermanent(source.ID())) {
					t.Fatal("stun untapped")
				}
			case "phase", "phase-cycle":
				ids := g.PhaseOut(source.ID())
				blocked = false
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				blocked = false
			case "target-return":
				g.BouncePermanentToHand(g.FindPermanent(subject.ID()))
				b.RemoveFromHand(subjectCard.ID())
				subject = g.PutOnBattlefield(subjectCard, b.PlayerID())
				g.TapPermanent(subject)
				blocked = false
			case "target-phase-cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "remove-abilities":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
				g.AddContinuousEffect(RemoveAllAbilities(subject.ID(), core.Indefinite))
			case "simultaneous":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(subject.ID()).Tapped != blocked {
				t.Fatal("untap decision")
			}
			if blocked && !g.UntapPermanent(g.FindPermanent(subject.ID())) {
				t.Fatal("other untap blocked")
			}
			if blocked {
				g.TapPermanent(g.FindPermanent(subject.ID()))
			}
			if change == "control" || change == "simultaneous" {
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(subject.ID()).Tapped {
					t.Fatal("next step retained expired rule")
				}
			}
			clone.SetActivePlayerIndex(1)
			clone.RunStepWithPriority(core.Untap)
			if !clone.FindPermanent(subject.ID()).Tapped {
				t.Fatal("clone lifetime changed")
			}
		})
	}
}

func TestTappedUntapRuleDefinition(t *testing.T) {
	result := BindPermanent("unbound")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "unbound", Card: NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PreventUntapWhileSourceTapped(From(result))))), WantErr: true},
		{Name: "nil subject", Card: NewArtifact("Invalid", "", WithAbility(Activate(Costs(), PreventUntapWhileSourceTapped(nil)))), WantErr: true},
		{Name: "spell source", Card: NewInstant("Invalid", "", WithEffect(PreventUntapWhileSourceTapped(This))), WantErr: true},
	})
}

func TestTappedUntapRuleIndependentSources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("subject", AnyPermanent)
	factory := func() Card {
		return NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), Sequence(TapSubject(target), PreventUntapWhileSourceTapped(target)), WithTargets(target))))
	}
	first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), a.PlayerID())
	subject := g.PutOnBattlefield(NewArtifact("Subject", ""), b.PlayerID())
	for _, source := range []*Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{subject.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	clone := g.Clone()
	g.UntapPermanent(g.FindPermanent(first.ID()))
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(subject.ID()).Tapped {
		t.Fatal("one source ended another duration")
	}
	g.UntapPermanent(g.FindPermanent(second.ID()))
	g.TapPermanent(g.FindPermanent(second.ID()))
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(subject.ID()).Tapped {
		t.Fatal("retap restarted duration")
	}
	clone.SetActivePlayerIndex(1)
	clone.RunStepWithPriority(core.Untap)
	if !clone.FindPermanent(subject.ID()).Tapped {
		t.Fatal("clone shared expiration state")
	}
}

func TestTappedUntapRuleWaitingSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	g.TapPermanent(source)
	if err := ApplyEffect(g, AtNext(EachEndStep, PreventUntapWhileSourceTapped(This)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		sources := game.WaitingDelayedPermanentSources()
		if len(sources) != 1 {
			t.Fatalf("waiting source count %d", len(sources))
		}
		if len(game.effects.effects) != 0 {
			t.Fatal("query installed restriction")
		}
	}
}
