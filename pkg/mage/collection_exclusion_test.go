package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCollectionExclusionDelayedSource(t *testing.T) {
	for _, change := range []string{"none", "phase", "cycle", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 3)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), b.PlayerID())
			excluded := WithoutPermanents(MatchingPermanents(AnyPermanent), SingletonPermanent(This))
			body := AtNext(EachEndStep, TapEach(IntersectPermanents(MatchingPermanents(IsCreature), excluded)))
			if err := ValidateCardDefinition("Delay", NewArtifact("Delay", "", WithAbility(Activate(Costs(), body)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			original := g.CaptureDamageSource(p.ID())
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for range 2 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(original) || g.FindPermanent(other.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("exclusion query lost source or executed body")
				}
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.FindPermanentIncludingPhased(p.ID()).Tapped != (change == "return") || !game.FindPermanent(other.ID()).Tapped {
					t.Fatal("delayed exclusion used the wrong visit")
				}
			}
		})
	}
}

func TestCollectionExclusionDeclarations(t *testing.T) {
	for _, excluded := range []PermanentSet{nil, SingletonPermanent(From(BindPermanent("missing")))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), TapEach(WithoutPermanents(MatchingPermanents(AnyPermanent), excluded)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid excluded set accepted")
		}
	}
}
