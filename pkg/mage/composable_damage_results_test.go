package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageResultsIterateFinalRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.AddPreventionShield(a.PlayerID(), 2)
	results, item := BindDamageResults("results"), BindDamageResult("item")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2), DamageResultsInto(results)),
		ForEachDamageResult(results, item, Sequence(
			GainLifeFor(You, DamageResultAmount(item)),
			TapSubject(DamageResultPermanent(item)),
		)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 24 || b.Life() != 20 || victim.Damage != 4 || !victim.Tapped {
		t.Fatalf("life = %d, %d; victim damage = %d, tapped = %t", a.Life(), b.Life(), victim.Damage, victim.Tapped)
	}
}

func TestDamageResultsClearAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	results, item := BindDamageResults("results"), BindDamageResult("item")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(0), DamageResultsInto(results)),
		ForEachDamageResult(results, item, DealDamageFrom(DamageResultSource(item), You, Fixed(3))),
	)
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatalf("original = %d, %d; clone = %d, %d", a.Life(), b.Life(), clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
}

func TestDamageResultsKeepDepartedSourceAndPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	results, item := BindDamageResults("results"), BindDamageResult("item")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(OpponentsOf(You), WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(1), DamageResultsInto(results)),
		Sacrifice(This),
		ForEachDamageResult(results, item, DealDamageFrom(DamageResultSource(item), DamageResultPlayer(item), DamageResultAmount(item))),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 18 || g.FindPermanent(source.ID()) != nil {
		t.Fatalf("life = %d, %d; source remains = %t", a.Life(), b.Life(), g.FindPermanent(source.ID()) != nil)
	}
}

func TestDamageResultsExcludeReturnedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Target", "", 2, 9)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("target", 1, 1, IsCreature)
	results := BindDamageResults("results")
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2), DamageResultsInto(results))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	outcomes := g.resolution.damageResultsBindings[results]
	if len(outcomes) != 2 || outcomes[0].playerID != a.PlayerID() || outcomes[1].playerID != b.PlayerID() || outcomes[0].permanent.id != uuid.Nil || returned.Damage != 0 {
		t.Fatalf("outcomes = %+v; returned damage = %d", outcomes, returned.Damage)
	}
}

func TestDamageResultsCloneActiveFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	results, item := BindDamageResults("results"), BindDamageResult("item")
	outcome := damageOutcome{amount: 1, playerID: b.PlayerID(), source: g.CaptureDamageSource(source.ID())}
	g.resolution.damageResultsBindings = map[*DamageResultsBinding][]damageOutcome{results: {outcome}}
	g.resolution.damageResultBindings = map[*DamageResultBinding]damageOutcome{item: outcome}
	clone := g.Clone()
	clone.resolution.damageResultsBindings[results][0].amount = 9
	copyItem := clone.resolution.damageResultBindings[item]
	copyItem.amount = 7
	clone.resolution.damageResultBindings[item] = copyItem
	if g.resolution.damageResultsBindings[results][0].amount != 1 || g.resolution.damageResultBindings[item].amount != 1 {
		t.Fatal("clone changed original damage result frame")
	}
}

func TestDamageResultsDeclaration(t *testing.T) {
	results, item := BindDamageResults("results"), BindDamageResult("item")
	valid := NewInstant("Valid", "", Spell(Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)),
		ForEachDamageResult(results, item, GainLifeFor(You, DamageResultAmount(item))),
	)))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	invalid := []ResolvingEffect{
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(nil)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results), DamageResultsInto(results)),
		ForEachDamageResult(results, item, GainLifeFor(You, DamageResultAmount(item))),
		Sequence(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)), GainLifeFor(You, DamageResultAmount(item))),
		Sequence(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)), ForEachDamageResult(results, nil, GainLifeFor(You, Fixed(1)))),
	}
	for _, effect := range invalid {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(effect))); err == nil || !strings.Contains(err.Error(), "binding") {
			t.Fatalf("invalid declaration accepted: %v", err)
		}
	}
}
