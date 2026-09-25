package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatedTokensAreVisibleTogetherToEntryTriggers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	var observed []int
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(
		NewTriggered(EvtZoneChange, false).SetCondition(func(evt *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
			if evt.ToZone == ZoneBattlefield && evt.SourceID != uuid.Nil {
				if p := reader.FindPermanent(evt.SourceID); p != nil && p.IsToken {
					observed = append(observed, len(reader.FilterBattlefield(IsToken)))
				}
			}
			return false
		}),
	))
	g.PutOnBattlefield(watcher, a.PlayerID())
	if err := ApplyEffect(g, CreateTokens(2, "Soldier", 1, 1, []CardType{TypeCreature}, []string{"Soldier"}), watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(observed) != 2 || observed[0] != 2 || observed[1] != 2 {
		t.Fatalf("entry trigger observed token counts %v, want [2 2]", observed)
	}
}
