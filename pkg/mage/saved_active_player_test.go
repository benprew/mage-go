package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedActivePlayerDelayedFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("active at creation")
	body := Sequence(GainLifeFor(FromPlayer(saved), Fixed(2)), GainLifeFor(ActivePlayer, Fixed(3)))
	declaration := (SavedPlayerInstruction{Subject: ActivePlayer, Binding: saved, Body: AtNext(EachEndStep, body)}).Effect()
	card := NewArtifact("Source", "", WithAbility(Activate(Costs(), declaration)))
	if err := ValidateCardDefinition("Source", card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	for _, active := range []int{1, 0} {
		g.SetActivePlayerIndex(active)
		if err := ApplyEffect(g, declaration, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: g}); ok {
			t.Fatal("snapshot escaped its resolution frame")
		}
	}
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	g.SetActivePlayerIndex(0)
	clone := g.Clone()
	clone.SetActivePlayerIndex(1)
	for _, game := range []*Game{g, clone} {
		for range 3 {
			if len(game.WaitingDelayedPermanentSources()) != 0 || game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("query executed a player read or inferred a source")
			}
		}
		game.RunStepWithPriority(core.EndStep)
		wantA, wantB := 28, 22
		if game == clone {
			wantA, wantB = 22, 28
		}
		if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB || len(game.DelayedTriggers()) != 0 {
			t.Fatal("independent saved frames or live turn reads changed")
		}
		if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: game}); ok {
			t.Fatal("delayed body did not restore the outer frame")
		}
	}
	if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, GainLifeFor(FromPlayer(BindPlayer("unbound")), Fixed(1))))))); err == nil {
		t.Fatal("unbound delayed read accepted")
	}
	if err := ApplyEffect(g, GainLifeFor(ActivePlayer, Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil || a.Life() != 29 {
		t.Fatal("live active-player read changed after delayed execution")
	}
}
