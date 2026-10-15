package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryPresenceNativeDetection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := PermanentMatchesFilter(This, AnyPermanent)
	rule := WhenEvent(Enters(This), GainLifeFor(You, Fixed(2)), WithInterveningIf(condition)).(*GenericTriggered)
	card := NewArtifact("Presence", "", WithAbility(rule))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	event := &core.GameEvent{Type: core.EvtZoneChange, FromZone: core.ZoneHand, ToZone: core.ZoneBattlefield, SourceID: source.ID(), SourceIncarnation: source.incarnationID}
	check := func(want bool) {
		t.Helper()
		if rule.CheckTrigger(event, g) != want {
			t.Fatalf("entry detection want %v", want)
		}
	}
	check(true)
	phased := g.PhaseOut(source.ID())
	check(false)
	g.PhaseIn(phased)
	check(true)
	g.RemoveFromBattlefield(source)
	check(false)
}

func TestEntryPresenceNativeCopiedRecheck(t *testing.T) {
	for _, leave := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewArtifact("Presence", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, Fixed(2)), WithInterveningIf(PermanentMatchesFilter(This, AnyPermanent)))))
		source := g.PutOnBattlefield(card, a.PlayerID())
		g.PutTriggersOnStack()
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		clone := g.Clone()
		if leave {
			g.RemoveFromBattlefield(source)
		}
		g.ResolveStack()
		want := 24
		if leave {
			want = 20
		}
		if a.Life() != want {
			t.Fatalf("copied recheck life=%d want %d", a.Life(), want)
		}
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != 24 {
			t.Fatal("clone lost live source")
		}
	}
}
