package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticGrantedActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	rule := GrantEachActivatedAbility(WithoutPermanent(MatchingPermanents(HasSubType("Zombie")), This), Costs(ManaPayment("{B}")), Regenerate(This))
	source := NewCreature("Grant source", "", 2, 3, WithSubTypes("Zombie"), WithAbility(Static(rule)))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	recipient := NewArtifact("Recipient", "", WithSubTypes("Zombie"))
	recipient.SetOwner(b.PlayerID())
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(source.ID()).RuntimeAbilities) != 1 {
		t.Fatal("source received excluded ability")
	}
	abilities := g.FindPermanent(recipient.ID()).RuntimeAbilities
	if len(abilities) != 1 {
		t.Fatalf("noncreature recipient has %d abilities", len(abilities))
	}
	ability := UnwrapAbility(abilities[0]).(ActivatedAbility)
	if ability.Source() != recipient.ID() || ability.Controller() != b.PlayerID() {
		t.Fatal("grant used donor context")
	}
	if !IsRegenerationEffect(ability.Effects()[0]) {
		t.Fatal("missing regeneration metadata")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 || len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("grant clone isolation failed")
	}
	g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("phased source retained grant")
	}
}

func TestStaticGrantedActivationValidation(t *testing.T) {
	saved := BindPermanent("outside")
	for _, body := range []ResolvingEffect{Regenerate(From(saved)), Regenerate(DeclarePermanentTarget("undeclared", IsCreature)), nil} {
		card := NewEnchantment("Invalid grant", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), body))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "abilities[0]") {
			t.Fatalf("missing grant diagnostic: %v", err)
		}
	}
	card := NewEnchantment("Valid grant", "", WithAbility(Static(GrantEachActivatedAbility(WithoutPermanent(MatchingPermanents(IsCreature), This), Costs(), Regenerate(This)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestStaticGrantedManaActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewEnchantment("Mana grant", "", WithAbility(Static(GrantEachActivatedAbility(ControlledPermanents(IsLand), Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1))))))
	g.PutOnBattlefield(source, a.PlayerID())
	first, second, opponent := NewLand("First"), NewLand("Second"), NewLand("Opponent")
	for _, card := range []Card{first, second} {
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.PutOnBattlefield(opponent, b.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(opponent.ID()).RuntimeAbilities) != 0 {
		t.Fatal("controlled set included opponent")
	}
	one := UnwrapAbility(g.FindPermanent(first.ID()).RuntimeAbilities[0]).(ActivatedAbility)
	two := UnwrapAbility(g.FindPermanent(second.ID()).RuntimeAbilities[0]).(ActivatedAbility)
	if one == two || one.AbilityID() == two.AbilityID() || one.Source() != first.ID() || two.Source() != second.ID() {
		t.Fatal("recipients share runtime ability state")
	}
	if !g.CanAfford(a.PlayerID(), ManaCost{Blue: 2}, nil) {
		t.Fatal("planner cannot use granted mana")
	}
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 2}); err != nil {
		t.Fatal(err)
	}
	if clone.GetStack().Size() != 0 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Blue) != 2 || a.ManaPool().Count(Blue) != 0 || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("granted mana timing or clone isolation failed")
	}
}

func TestWithoutPermanentAbsentAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 1, 1)
	g.PutOnBattlefield(card, a.PlayerID())
	saved := BindPermanent("excluded")
	set := WithoutPermanent(MatchingPermanents(IsCreature), From(saved))
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
	if len(set.resolvePermanents(ctx)) != 1 {
		t.Fatal("absent reference removed a member")
	}
	if err := Snapshot(This, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("saved reference did not exclude member")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	if len(set.resolvePermanents(ctx)) != 1 {
		t.Fatal("old reference excluded returned object")
	}
}
