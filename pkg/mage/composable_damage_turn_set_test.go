package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTurnDamagePermanentSetIdentityAndOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 5)
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	recipientCard := NewCreature("Recipient", "", 1, 5)
	recipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsDealtDamageByThisTurn(This, MatchingPermanents(IsCreature))
	check := func(want *Permanent) {
		t.Helper()
		got := set.resolvePermanents(ctx)
		if want == nil && len(got) != 0 || want != nil && (len(got) != 1 || got[0].ID() != want.ID() || got[0].incarnationID != want.incarnationID) {
			t.Fatalf("selected permanents = %v, want %v", got, want)
		}
	}
	check(nil)
	g.AddPreventionShield(recipient.ID(), 1)
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(source.ID()))
	check(nil)
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(source.ID()))
	check(recipient)
	g.PhaseOut(recipient.ID())
	check(nil)
	g.PhaseIn([]uuid.UUID{recipient.ID()})
	check(recipient)
	clone := g.Clone()
	clone.damage.ClearEndOfTurn()
	if got := set.resolvePermanents(&EffectContext{Game: clone, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}); len(got) != 0 {
		t.Fatal("clone retained cleared turn history")
	}
	check(recipient)
	g.RemoveFromBattlefield(recipient)
	g.PutOnBattlefield(recipientCard, b.PlayerID())
	check(nil)
	g.RemoveFromBattlefield(source)
	newSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.DealDamageToPermanentFromSource(other, 1, g.CaptureDamageSource(newSource.ID()))
	check(nil)
	ctx.sourceIncarnation = newSource.incarnationID
	got := set.resolvePermanents(ctx)
	if len(got) != 1 || got[0].ID() != other.ID() || got[0].incarnationID != other.incarnationID {
		t.Fatalf("new source history = %v", got)
	}
	g.damage.ClearEndOfTurn()
	check(nil)
}

func TestTurnDamagePermanentSetDefinition(t *testing.T) {
	set := PermanentsDealtDamageByThisTurn(This, MatchingPermanents(IsCreature))
	good := NewCreature("Good", "", 1, 5, WithAbility(WhenEvent(Enters(This), TapEach(set))))
	if err := ValidateCardDefinition("Good", good); err != nil {
		t.Fatal(err)
	}
	if InspectDefinition(set).Role != "permanent-set" {
		t.Fatal("missing permanent-set inspection role")
	}
	for _, badSet := range []PermanentSet{
		PermanentsDealtDamageByThisTurn(nil, MatchingPermanents(IsCreature)),
		PermanentsDealtDamageByThisTurn(This, nil),
		PermanentsDealtDamageByThisTurn(DamageSourceOf(DeclarePermanentTarget("source", IsCreature)), MatchingPermanents(IsCreature)),
	} {
		bad := NewInstant("Bad", "", Spell(TapEach(badSet)))
		if err := ValidateCardDefinition("Bad", bad); err == nil {
			t.Fatal("accepted invalid damage history set")
		}
	}
}
