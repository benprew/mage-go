package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRemoveAbilitiesTemporaryCapturesSetAndLayerOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 2, 2, WithKeyword(Flying))
	g.PutOnBattlefield(first, a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, GrantKeywordModification(This, CantRegenerate)), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilitiesEach(MatchingPermanents(IsCreature))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(first.ID()).HasKeyword(Flying) || !g.FindPermanent(first.ID()).HasKeyword(CantRegenerate) {
		t.Fatal("removal must remove abilities and preserve rules restrictions")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PhaseIn(g.PhaseOut(first.ID()))
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("source departure and phasing must preserve temporary removal")
	}
	second := NewCreature("Second", "", 2, 2, WithKeyword(Flying))
	g.PutOnBattlefield(second, b.PlayerID())
	if !g.FindPermanent(second.ID()).HasKeyword(Flying) {
		t.Fatal("the resolving set must not include a later permanent")
	}
	if err := ApplyEffect(g, Until(EndOfTurn, GrantKeywordModification(This, Vigilance)), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(first.ID()).HasKeyword(Vigilance) || g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("a later grant must restore only its own ability")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(first.ID()).HasKeyword(Flying) || g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("expiry must be isolated between clones")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("the old effect must not remove abilities from a returned permanent")
	}
}

func TestRemoveAbilitiesStaticSetUpdates(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "", WithAbility(Static(While(Controls(IsArtifact), RemoveAbilitiesEach(MatchingPermanents(IsCreature))))))
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 2, 2, WithKeyword(Flying))
	g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "", 2, 2, WithKeyword(Vigilance))
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).HasKeyword(Flying) || g.FindPermanent(second.ID()).HasKeyword(Vigilance) {
		t.Fatal("the static set must include later creatures")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ApplyContinuousEffects()
	if !g.FindPermanent(first.ID()).HasKeyword(Flying) || !g.FindPermanent(second.ID()).HasKeyword(Vigilance) {
		t.Fatal("source departure must end the static removal")
	}
}

func TestRemoveAbilitiesValidationAndMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, RemoveAbilities(target))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != 0 || !strings.Contains(info.Text, "remove all abilities") {
		t.Fatalf("removal metadata = %+v", info)
	}
	if err := ValidateCardDefinition("Unbound", NewInstant("Unbound", "", WithAbility(Spell(body)))); err == nil || !strings.Contains(err.Error(), "target binding requires a declaration") {
		t.Fatalf("undeclared target error = %v", err)
	}
	if err := ValidateCardDefinition("Nil", NewInstant("Nil", "", WithAbility(Spell(Until(EndOfTurn, RemoveAbilities(nil)))))); err == nil || !strings.Contains(err.Error(), "supply a non-nil expression") {
		t.Fatalf("nil target error = %v", err)
	}
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", WithAbility(Spell(body, WithTargets(target))))); err != nil {
		t.Fatal(err)
	}
}
