package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestMatchingUntapDecision(t *testing.T) {
	for _, scenario := range []string{"boost", "reduce", "switch", "source-phase", "source-return", "source-removed", "source-abilities", "recipient-abilities", "control", "clone", "phase-in"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewArtifact("Rule", "", WithStaticAbility(PreventUntapDuringStepForMatching(And(IsCreature, HasPowerGTE(3)))))
			sourceCard.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(sourceCard, b.PlayerID())
			power := 2
			if scenario != "boost" && scenario != "switch" {
				power = 3
			}
			card := NewCreature("Subject", "", power, 4)
			subject := g.PutOnBattlefield(card, a.PlayerID())
			g.TapPermanent(subject)
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
			g.TapPermanent(other)
			blocked := true
			var mod ContinuousModification
			switch scenario {
			case "boost":
				mod = ModifyPT(This, Fixed(1), Fixed(0))
			case "reduce":
				mod = ModifyPT(This, Fixed(-1), Fixed(0))
				blocked = false
			case "switch":
				mod = SwitchPowerToughness(This)
			case "source-phase":
				g.PhaseOut(source.ID())
				blocked = false
			case "source-return":
				g.BouncePermanentToHand(source)
				b.RemoveFromHand(sourceCard.ID())
				g.PutOnBattlefield(sourceCard, b.PlayerID())
			case "source-removed":
				g.DestroyPermanent(source)
				blocked = false
			case "source-abilities":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				blocked = false
			case "recipient-abilities":
				mod = RemoveAbilities(This)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: subject.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
			case "phase-in":
				g.PhaseOut(subject.ID())
			}
			if mod != nil {
				if err := ApplyEffect(g, Until(core.EndOfTurn, mod), subject.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.DestroyPermanent(clone.FindPermanent(source.ID()))
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(subject.ID()).Tapped != blocked {
				t.Fatalf("restriction: tapped=%v power=%d toughness=%d blocked=%v", g.FindPermanent(subject.ID()).Tapped, g.FindPermanent(subject.ID()).CurrentPower(g), g.FindPermanent(subject.ID()).CurrentToughness(g), blocked)
			}
			if scenario != "control" && !g.FindPermanent(other.ID()).Tapped {
				t.Fatal("inactive player untapped")
			}
			if !clone.FindPermanentIncludingPhased(subject.ID()).Tapped {
				t.Fatal("clone changed with original")
			}
			if scenario == "clone" {
				clone.RunStepWithPriority(core.Untap)
				if clone.FindPermanent(subject.ID()).Tapped {
					t.Fatal("clone retained departed source restriction")
				}
			}
			if !g.UntapPermanent(g.FindPermanent(subject.ID())) && blocked {
				t.Fatal("restriction prohibited other untap effects")
			}
		})
	}
}

func TestMatchingUntapDecisionSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Rule", "")
	filter := NewPermanentFilter("creatures while source is tapped", func(p *Permanent, game *Game) bool {
		source := game.FindPermanent(card.ID())
		return p.HasType(core.TypeCreature) && source != nil && source.Tapped
	})
	card.AddAbility(StaticAbility(PreventUntapDuringStepForMatching(filter)))
	source := g.PutOnBattlefield(card, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	for _, p := range []*Permanent{source, first, second} {
		g.TapPermanent(p)
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("decision changed during untap operation")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
		t.Fatal("restriction membership did not refresh")
	}
}
