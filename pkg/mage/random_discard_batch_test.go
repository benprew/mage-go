package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomDiscardBatchEventsAndReplacements(t *testing.T) {
	for _, replacement := range []string{"none", "library", "recheck", "prevent"} {
		t.Run(replacement, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			switch replacement {
			case "library", "recheck":
				g.effects.AddReplacement(&selectedDiscardHandReplacement{card: second.ID()})
				if replacement == "recheck" {
					g.effects.AddReplacement(&discardToHandReplacement{})
				}
			case "prevent":
				g.effects.AddReplacement(&preventRandomDiscardReplacement{card: first.ID()})
			}
			seen := 0
			watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				seen++
				wantHand := 0
				if replacement == "recheck" || replacement == "prevent" {
					wantHand = 1
				}
				if evt.PlayerID != a.PlayerID() || len(a.Hand()) != wantHand {
					t.Error("event saw a partial batch")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), b.PlayerID())
			g.SetRandomResults([]int{0, 0, 7})
			if err := ApplyEffect(g, Sequence(DiscardAtRandom(You, Fixed(2)), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantHand, wantLibrary, wantGrave, wantEvents := 0, 0, 2, 2
			switch replacement {
			case "library":
				wantLibrary, wantGrave = 1, 1
			case "recheck":
				wantHand, wantGrave, wantEvents = 1, 1, 1
			case "prevent":
				wantHand, wantGrave, wantEvents = 1, 1, 1
			}
			if len(a.Hand()) != wantHand || len(a.Library()) != wantLibrary || len(a.Graveyard()) != wantGrave || seen != wantEvents || a.Life() != 21 || g.RandIntn(10) != 7 {
				t.Fatalf("batch destinations/events: %d/%d/%d/%d", len(a.Hand()), len(a.Library()), len(a.Graveyard()), seen)
			}
		})
	}
}

type preventRandomDiscardReplacement struct {
	replacementBase
	card uuid.UUID
}

func (r *preventRandomDiscardReplacement) Matches(action Action, _ GameReader) bool {
	d, ok := action.(*DiscardAction)
	return ok && d.CausedByEffect() && d.CardID() == r.card
}
func (*preventRandomDiscardReplacement) Replace(Action, *Game) Action { return nil }
func (*preventRandomDiscardReplacement) IsActive(GameReader) bool     { return true }
func (r *preventRandomDiscardReplacement) Clone() ReplacementEffect {
	return &preventRandomDiscardReplacement{card: r.card}
}
