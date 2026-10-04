package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSelectedDiscardBatchEvents(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToHand(NewArtifact("Card", ""))
		}
		if replacement {
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		}
		seen := 0
		watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			seen++
			if evt.PlayerID != a.PlayerID() || len(a.Hand()) != 1 {
				t.Error("event observed partial discard")
			}
			if replacement && len(a.Library()) != 2 || !replacement && len(a.Graveyard()) != 2 {
				t.Error("event observed partial destination")
			}
			return false
		})
		g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), b.PlayerID())
		if err := ApplyEffect(g, Sequence(Discard(You, Fixed(2)), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if seen != 2 || a.Life() != 21 {
			t.Fatal("discard lost events or continuation")
		}
	}
}

type selectedDiscardHandReplacement struct {
	replacementBase
	card uuid.UUID
}

func (r *selectedDiscardHandReplacement) Matches(action Action, game GameReader) bool {
	d, ok := action.(*DiscardAction)
	return ok && d.CausedByEffect() && d.CardID() == r.card && len(game.GetPlayer(d.PlayerID()).Hand()) == 2
}
func (*selectedDiscardHandReplacement) Replace(action Action, _ *Game) Action {
	copied := *action.(*DiscardAction)
	copied.destination = core.ZoneLibrary
	return &copied
}
func (*selectedDiscardHandReplacement) IsActive(GameReader) bool { return true }
func (r *selectedDiscardHandReplacement) Clone() ReplacementEffect {
	return &selectedDiscardHandReplacement{card: r.card}
}

func TestSelectedDiscardBatchReplacementPreparation(t *testing.T) {
	for _, recheck := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		first, second := NewLand("First"), NewLand("Second")
		a.AddToHand(first)
		a.AddToHand(second)
		g.effects.AddReplacement(&selectedDiscardHandReplacement{card: second.ID()})
		if recheck {
			g.effects.AddReplacement(&discardToHandReplacement{})
		}
		clone := g.Clone()
		if err := ApplyEffect(clone, Discard(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		p := clone.GetPlayer(a.PlayerID())
		wantHand, wantLibrary := 0, 1
		if recheck {
			wantHand, wantLibrary = 1, 0
		}
		if len(p.Hand()) != wantHand || len(p.Library()) != wantLibrary || len(p.Graveyard()) != 1 || len(a.Hand()) != 2 {
			t.Fatal("replacement preparation, recheck, or clone isolation failed")
		}
	}
}
