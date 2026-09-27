package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedLandTappedForManaEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	other := g.PutOnBattlefield(NewLand("Other", WithManaAbility(Blue)), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Watcher", "", WithCastTarget(TargetLand()), WithAbility(WhenEvent(
		LandTappedForMana().OfAttachedLand().OfManaType(Blue), GainLifeFor(You, Fixed(1))))), b.PlayerID())
	g.Attach(aura.ID(), land.ID())

	if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 {
		t.Fatal("unattached land matched")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("trigger count = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 21 || b.Life() != 20 {
		t.Fatal("clone did not keep its own trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.Attach(aura.ID(), other.ID())
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 21 {
		t.Fatalf("life = %d/%d, want 21/21", a.Life(), b.Life())
	}

	oldIncarnation := land.incarnationID
	g.RemoveFromBattlefield(land)
	returned := g.PutOnBattlefield(land.Card, a.PlayerID())
	g.Attach(aura.ID(), returned.ID())
	g.FireEvent(GameEvent{Type: EvtTappedForMana, SourceID: returned.ID(), SourceIncarnation: oldIncarnation, PlayerID: a.PlayerID(), Amount: 1, ManaTypes: []Color{Blue}})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("old land incarnation matched a new attachment")
	}

	g.MutablePermanent(aura.ID()).PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtTappedForMana, SourceID: returned.ID(), SourceIncarnation: returned.incarnationID, PlayerID: a.PlayerID(), Amount: 1, ManaTypes: []Color{Blue}})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased aura matched")
	}
}

func TestAttachedLandTappedForManaDeclaration(t *testing.T) {
	pattern := LandTappedForMana().OfAttachedLand()
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || got.Text == "" {
		t.Fatalf("inspection = %+v", got)
	}
	valid := NewAura("Valid", "", WithCastTarget(TargetLand()), WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	duplicate := NewAura("Duplicate", "", WithCastTarget(TargetLand()), WithAbility(WhenEvent(pattern.OfAttachedLand(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("accepted duplicate attached-land filter")
	}
}
