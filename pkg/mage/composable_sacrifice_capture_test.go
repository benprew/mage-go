package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureYouSacrificePermanentCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("sacrificed creature")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouSacrifice().WithPermanent(paid),
		Sequence(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)),
			AddCountersTo(From(paid), core.P1P1, Fixed(1))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 3, 3)
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("sacrifice did not create one trigger")
	}
	g.PutOnBattlefield(victim, b.PlayerID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatal("captured source did not survive return, copy, or clone")
		}
		if game.FindPermanent(victim.ID()).CurrentPower(game) != 3 {
			t.Fatal("old permanent reference modified the returned incarnation")
		}
	}
}

func TestCreatureYouSacrificePermanentCaptureSelfAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	source := NewCreature("Self", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouSacrifice().WithPermanent(paid),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	g.PutOnBattlefield(other, a.PlayerID())
	g.PhaseOut(source.ID())
	g.DoSacrifice(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source captured a sacrifice")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.DoSacrifice(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("self sacrifice missed its event")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatal("self sacrifice lost departure source")
	}
}

func TestCreatureYouSacrificePermanentCaptureValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func() EventPattern
		want string
	}{
		{"valid", func() EventPattern { return CreatureYouSacrifice().WithPermanent(BindPermanent("paid")) }, ""},
		{"nil", func() EventPattern { return CreatureYouSacrifice().WithPermanent(nil) }, "non-nil"},
		{"unnamed", func() EventPattern { return CreatureYouSacrifice().WithPermanent(BindPermanent("")) }, "name"},
		{"duplicate", func() EventPattern {
			return CreatureYouSacrifice().WithPermanent(BindPermanent("one")).WithPermanent(BindPermanent("two"))
		}, "once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Synthetic", "", 1, 1, WithAbility(WhenEvent(tc.make(), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	missing := BindPermanent("missing")
	card := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(),
		DealDamageFrom(DamageSourceOf(From(missing)), You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound event participant passed registration")
	}
}
