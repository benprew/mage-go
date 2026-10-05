package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestArtifactActivationCaptureSacrificedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindPermanent("artifact")
	controller := BindPlayer("controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ArtifactAbilityActivatedWithoutTapCost().WithArtifact(binding).WithController(controller), Sequence(LoseLifeFor(ControllerOfWithLKI(From(binding)), Fixed(2)), LoseLifeFor(FromPlayer(controller), Fixed(1))))))
	g.PutOnBattlefield(watcher, b.PlayerID())
	card := NewArtifact("Paid source", "", WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(0)))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{}); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 20 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

func TestArtifactActivationCaptureUsesEventIncarnation(t *testing.T) {
	for _, artifact := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		binding := BindPermanent("source")
		controller := BindPlayer("controller")
		g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(WhenEvent(ArtifactAbilityActivatedWithoutTapCost().WithArtifact(binding).WithController(controller), LoseLifeFor(FromPlayer(controller), Fixed(1))))), b.PlayerID())
		var card Card
		if artifact {
			card = NewArtifact("Original", "")
		} else {
			card = NewCreature("Original", "", 1, 1)
		}
		card.SetOwner(a.PlayerID())
		old := g.PutOnBattlefield(card, a.PlayerID())
		id, incarnation := old.ID(), old.incarnationID
		g.RemoveFromBattlefield(old)
		returned := g.PutOnBattlefield(card, b.PlayerID())
		if artifact {
			returned.Card.(*BaseCard).types = []core.CardType{core.TypeCreature}
		} else {
			returned.Card.(*BaseCard).types = []core.CardType{core.TypeArtifact}
		}
		g.FireEvent(core.GameEvent{Type: core.EvtAbilityActivated, SourceID: id, SourceIncarnation: incarnation, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		want := 0
		if artifact {
			want = 1
		}
		if g.StackSize() != want {
			t.Fatalf("stack=%d want=%d", g.StackSize(), want)
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		for _, game := range []*Game{g, clone} {
			if game.GetPlayer(a.PlayerID()).Life() != 20-want || game.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("event used returned controller")
			}
		}
	}
}
