package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageHistoryRecipientSetsAndSimultaneousAction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 3), a.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
	untouched := g.PutOnBattlefield(NewPlaneswalker("Untouched", "", 5), b.PlayerID())
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(source.ID()))
	g.DealDamageToPermanentFromSource(walker, 1, g.CaptureDamageSource(source.ID()))
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	players := PlayersDealtDamageBy(This, OpponentsOf(You))
	permanents := PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker))
	if got := players.resolvePlayers(ctx); len(got) != 1 || got[0] != b.PlayerID() {
		t.Fatalf("player history = %v", got)
	}
	if got := permanents.resolvePermanents(ctx); len(got) != 1 || got[0] != walker {
		t.Fatalf("permanent history = %v", got)
	}
	if err := DealDamageEach(This, DamageRecipients(players, permanents), Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 || walker.Counters[Loyalty] != 3 || untouched.Counters[Loyalty] != 5 {
		t.Fatal("damage history selection changed recipients or damage outcome")
	}
}

func TestDamageHistoryRecipientSetsIdentityPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 2, 3)
	oldSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	walkerCard := NewPlaneswalker("Walker", "", 5)
	oldWalker := g.PutOnBattlefield(walkerCard, b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: oldSource.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: oldSource.incarnationID}
	players := PlayersDealtDamageBy(This, OpponentsOf(You))
	permanents := PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker))
	g.AddPreventionShield(oldWalker.ID(), 1)
	g.DealDamageToPermanentFromSource(oldWalker, 1, g.CaptureDamageSource(oldSource.ID()))
	if len(permanents.resolvePermanents(ctx)) != 0 {
		t.Fatal("prevented damage entered the selected set")
	}
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(oldSource.ID()))
	g.DealDamageToPermanentFromSource(oldWalker, 1, g.CaptureDamageSource(oldSource.ID()))
	g.PhaseOut(oldWalker.ID())
	if len(permanents.resolvePermanents(ctx)) != 0 || len(players.resolvePlayers(ctx)) != 1 {
		t.Fatal("phasing changed player history or selected a phased permanent")
	}
	g.PhaseIn([]uuid.UUID{oldWalker.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(oldWalker.ID()))
	if len(permanents.resolvePermanents(&EffectContext{Game: clone, SourceID: oldSource.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: oldSource.incarnationID})) != 0 || len(permanents.resolvePermanents(ctx)) != 1 {
		t.Fatal("clone and original share a recipient selection")
	}
	g.RemoveFromBattlefield(oldWalker)
	g.PutOnBattlefield(walkerCard, b.PlayerID())
	if len(permanents.resolvePermanents(ctx)) != 0 {
		t.Fatal("returned recipient inherited history")
	}
	g.RemoveFromBattlefield(oldSource)
	newSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if len(players.resolvePlayers(ctx)) != 1 {
		t.Fatal("departed source lost its recorded incarnation")
	}
	g.zones.ClearLKI()
	if len(players.resolvePlayers(ctx)) != 1 {
		t.Fatal("cleanup erased durable source history")
	}
	ctx.sourceIncarnation = newSource.incarnationID
	if len(players.resolvePlayers(ctx)) != 0 {
		t.Fatal("returned source inherited history")
	}
}

func TestDamageHistoryRecipientSetsValidation(t *testing.T) {
	set := PlayersDealtDamageBy(This, OpponentsOf(You))
	permanents := PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker))
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(Enters(This), DealDamageEach(This, DamageRecipients(set, permanents), Fixed(1)))))
	if err := ValidateCardDefinition("Good", good); err != nil {
		t.Fatal(err)
	}
	if InspectDefinition(set).Role != "player-set" || InspectDefinition(permanents).Role != "permanent-set" {
		t.Fatal("history sets lack inspection roles")
	}
	for _, tc := range []struct {
		name string
		set  DamageRecipientSet
	}{
		{"nil source", DamageRecipients(PlayersDealtDamageBy(nil, EachPlayer), permanents)},
		{"nil players", DamageRecipients(PlayersDealtDamageBy(This, nil), permanents)},
		{"nil permanents", DamageRecipients(set, PermanentsDealtDamageBy(This, nil))},
		{"undeclared target", DamageRecipients(PlayersDealtDamageBy(DamageSourceOf(DeclarePermanentTarget("source", IsCreature)), EachPlayer), permanents)},
	} {
		bad := NewInstant(tc.name, "", Spell(DealDamageEach(This, tc.set, Fixed(1))))
		if err := ValidateCardDefinition(tc.name, bad); err == nil {
			t.Fatalf("accepted %s: %v", tc.name, err)
		}
	}
}
