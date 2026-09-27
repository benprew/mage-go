package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestReverseTurnDamageSetRecipientDeathAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipientCard := NewCreature("Recipient", "", 1, 6)
	recipient := g.PutOnBattlefield(recipientCard, a.PlayerID())
	firstCard := NewCreature("First", "", 1, 6)
	first := g.PutOnBattlefield(firstCard, b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 6), b.PlayerID())
	set := PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))
	ctx := &EffectContext{Game: g, SourceID: recipient.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: recipient.incarnationID}
	check := func(want ...*Permanent) {
		t.Helper()
		got := set.resolvePermanents(ctx)
		if len(got) != len(want) {
			t.Fatalf("selected %v, want %v", got, want)
		}
		for i := range got {
			if got[i].ID() != want[i].ID() || got[i].incarnationID != want[i].incarnationID {
				t.Fatalf("selected %v, want %v", got, want)
			}
		}
	}
	check()
	g.AddPreventionShield(recipient.ID(), 1)
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(first.ID()))
	check()
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(first.ID()))
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(second.ID()))
	check(first, second)
	g.PhaseOut(first.ID())
	check(second)
	g.PhaseIn([]uuid.UUID{first.ID()})
	check(first, second)
	g.RemoveFromBattlefield(recipient)
	g.zones.ClearLKI()
	check(first, second)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(first.ID()))
	if got := set.resolvePermanents(&EffectContext{Game: clone, SourceID: recipient.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: recipient.incarnationID}); len(got) != 1 || got[0].ID() != second.ID() {
		t.Fatalf("clone selection = %v", got)
	}
	check(first, second)
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(firstCard, b.PlayerID())
	check(second)
	newRecipient := g.PutOnBattlefield(recipientCard, a.PlayerID())
	ctx.sourceIncarnation = newRecipient.incarnationID
	check()
	g.damage.ClearEndOfTurn()
	ctx.sourceIncarnation = recipient.incarnationID
	check()
}

func TestReverseTurnDamageSetDefinition(t *testing.T) {
	set := PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))
	good := NewCreature("Good", "", 1, 5, WithAbility(WhenEvent(Dies(This), TapEach(set))))
	if err := ValidateCardDefinition("Good", good); err != nil {
		t.Fatal(err)
	}
	if InspectDefinition(set).Role != "permanent-set" {
		t.Fatal("missing permanent-set inspection role")
	}
	for _, badSet := range []PermanentSet{
		PermanentsThatDealtDamageToThisTurn(nil, MatchingPermanents(IsCreature)),
		PermanentsThatDealtDamageToThisTurn(This, nil),
		PermanentsThatDealtDamageToThisTurn(DeclarePermanentTarget("recipient", IsCreature), MatchingPermanents(IsCreature)),
	} {
		bad := NewInstant("Bad", "", Spell(TapEach(badSet)))
		if err := ValidateCardDefinition("Bad", bad); err == nil {
			t.Fatal("accepted invalid reverse history set")
		}
	}
}

func TestReverseTurnDamageSetDeathTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 1, 4, WithAbility(WhenEvent(Dies(This),
		TapEach(PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))))))
	card.SetOwner(a.PlayerID())
	recipient := g.PutOnBattlefield(card, a.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 4), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 4), b.PlayerID())
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(attacker.ID()))
	g.DestroyPermanent(recipient)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending death triggers = %d", g.StackSize())
	}
	g.PutOnBattlefield(card, a.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if !clone.FindPermanent(attacker.ID()).Tapped || g.FindPermanent(attacker.ID()).Tapped {
		t.Fatal("copy did not keep the death recipient or changed the original")
	}
	g.ResolveStack()
	if !g.FindPermanent(attacker.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("death trigger selected the wrong damage sources")
	}
}
