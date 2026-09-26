package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDealDamageFromCapturesFinalPermanentRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 8), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	result := BindPermanent("damaged permanent")
	amount := BindNumber("damage dealt")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(2), DamageAmountInto(amount), DamagePermanentInto(result)),
		AddCountersTo(From(result), core.P1P1, FromNumber(amount)),
	)
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if victim.Counters[core.P1P1] != 0 {
		t.Fatal("clone changed original recipient")
	}
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	gotVictim := g.FindPermanent(victim.ID())
	if gotVictim.Damage != 2 || gotVictim.Counters[core.P1P1] != 2 || b.Life() != 20 {
		t.Fatalf("damage=%d counters=%d life=%d", gotVictim.Damage, gotVictim.Counters[core.P1P1], b.Life())
	}
}

func TestDealDamageFromPermanentResultClearsAndKeepsIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	result := BindPermanent("damaged permanent")
	for _, amount := range []int{0, 2} {
		if amount == 2 {
			if err := ApplyEffect(g, PreventNextDamage(This, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		effect := Sequence(Snapshot(This, result), DealDamageFrom(This, This, Fixed(amount), DamagePermanentInto(result)), AddCountersTo(From(result), core.P1P1, Fixed(1)))
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(source.ID()).Counters[core.P1P1] != 0 {
			t.Fatalf("amount %d retained stale recipient", amount)
		}
	}
	victimCard := NewCreature("Victim", "", 2, 8)
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	target := DeclareDamageTarget("victim")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	g.RemoveFromBattlefield(victim)
	returned := g.PutOnBattlefield(victimCard, b.PlayerID())
	effect := Sequence(Snapshot(This, result), DealDamageFrom(This, target, Fixed(2), DamagePermanentInto(result)), AddCountersTo(From(result), core.P1P1, Fixed(1)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Damage != 0 || g.FindPermanent(source.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("old recipient binding followed returned card")
	}
}

func TestDealDamageFromPermanentResultAbsentAfterPhaseOrPlayerRedirect(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
		target := DeclareDamageTarget("victim")
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
		if phase {
			g.PhaseOut(victim.ID())
		} else {
			g.effects.AddReplacement(&creatureDamageRedirectReplacement{creatureID: victim.ID(), targetPlayerID: b.PlayerID()})
		}
		result := BindPermanent("damaged permanent")
		effect := Sequence(Snapshot(This, result), DealDamageFrom(This, target, Fixed(2), DamagePermanentInto(result)), AddCountersTo(From(result), core.P1P1, Fixed(1)))
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(source.ID()).Counters[core.P1P1] != 0 {
			t.Fatalf("phase=%t retained stale permanent result", phase)
		}
		if !phase && b.Life() != 18 {
			t.Fatal("redirection to player failed")
		}
	}
}

func TestDealDamageFromPermanentResultDeclaration(t *testing.T) {
	result := BindPermanent("damaged permanent")
	valid := NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(result)), AddCountersTo(From(result), core.P1P1, Fixed(1)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(nil)),
		DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(BindPermanent(""))),
		DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(result), DamagePermanentInto(result)),
	} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(effect))); err == nil {
			t.Fatal("invalid permanent result passed registration")
		}
	}
	missing := NewInstant("Missing", "", Spell(AddCountersTo(From(result), core.P1P1, Fixed(1))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
}
