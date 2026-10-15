package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type discardLibraryChooser struct {
	*BasePlayer
	prompts    []string
	orderCalls int
	invalid    bool
}

func (p *discardLibraryChooser) ChooseMayAbility(reason string) bool {
	p.prompts = append(p.prompts, reason)
	return true
}

func (p *discardLibraryChooser) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	p.orderCalls++
	if p.invalid {
		return NewLand("Invalid")
	}
	return cards[0]
}

func TestDiscardLibraryOrder(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "chosen", true: "invalid-fallback"}[invalid], func(t *testing.T) {
			a := &discardLibraryChooser{BasePlayer: NewBasePlayer("A"), invalid: invalid}
			g := NewGame(a, NewBasePlayer("B"))
			g.PutOnBattlefield(NewArtifact("Rule", "", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
			first, second, old := NewLand("First"), NewLand("Second"), NewLand("Old")
			a.AddToHand(first)
			a.AddToHand(second)
			a.AddToLibrary(old)
			seen := 0
			watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				seen++
				if len(a.Hand()) != 0 || len(a.Library()) != 3 {
					t.Error("event observed partial movement")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
			if got := g.discardActionsTogether([]*DiscardAction{NewDiscardAction(uuid.Nil, a.PlayerID(), first.ID()), NewDiscardAction(uuid.Nil, a.PlayerID(), second.ID())}); got != 2 {
				t.Fatal(got)
			}
			if a.orderCalls != 1 || a.Library()[0].ID() != first.ID() || a.Library()[1].ID() != second.ID() || a.Library()[2].ID() != old.ID() || seen != 2 {
				t.Error("missing top-first choice, fallback, or discard events")
			}
			if len(a.prompts) != 2 || !strings.Contains(a.prompts[0], first.Name()) || !strings.Contains(a.prompts[1], second.Name()) {
				t.Fatal("discard choice did not identify the private card", a.prompts)
			}
		})
	}
}
