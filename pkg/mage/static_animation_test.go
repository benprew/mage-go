package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStaticAnimationLayerContinuation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subjects := MatchingPermanents(And(IsLand, Not(IsCreature)))
	sourceCard := NewEnchantment("Animation", "", WithAbility(Static(AnimateEach(subjects, 2, 3, core.Black))))
	if err := ValidateCardDefinition("Animation", sourceCard); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land", WithSubTypes("Forest")), b.PlayerID())
	check := func(game *Game, creature bool) {
		t.Helper()
		p := game.FindPermanent(land.ID())
		if p.HasType(core.TypeCreature) != creature || !p.HasType(core.TypeLand) || !p.HasSubType("Forest") {
			t.Fatal("animation changed the wrong types")
		}
		if creature && (p.CurrentPower(game) != 2 || p.CurrentToughness(game) != 3 || len(p.Colors()) != 1 || p.Colors()[0] != core.Black) {
			t.Fatal("later layers lost the noncreature selection")
		}
	}
	check(g, true)
	removal := RemoveAllAbilities(source.ID(), core.EndOfTurn)
	g.AddContinuousEffect(removal)
	check(g, true)
	g.ApplyContinuousEffects()
	check(g, true)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	check(clone, false)
	check(g, true)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	check(g, false)
}

func TestStaticAnimationLayerOrder(t *testing.T) {
	for _, animationFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "set first", true: "animate first"}[animationFirst], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("Land", WithSubTypes("Forest")), a.PlayerID())
			animate := func() {
				g.PutOnBattlefield(NewEnchantment("Animation", "", WithAbility(Static(AnimateEach(MatchingPermanents(IsLand), 1, 1)))), b.PlayerID())
			}
			set := func() {
				if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(4), Fixed(5))), land.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if animationFirst {
				animate()
				set()
			} else {
				set()
				animate()
			}
			wantP, wantT := 1, 1
			if animationFirst {
				wantP, wantT = 4, 5
			}
			if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(2), Fixed(3))), land.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p := g.FindPermanent(land.ID())
			if p.CurrentPower(g) != wantP+2 || p.CurrentToughness(g) != wantT+3 {
				t.Fatalf("layer 7b timestamp or 7c order: %d/%d", p.CurrentPower(g), p.CurrentToughness(g))
			}
			g.MutablePermanent(land.ID()).AddCounter(core.P1P1, 1)
			if err := ApplyEffect(g, Until(core.EndOfTurn, SwitchPowerToughness(This)), land.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p = g.FindPermanent(land.ID())
			if p.CurrentPower(g) != wantT+4 || p.CurrentToughness(g) != wantP+3 {
				t.Fatal("animation ignored counters or layer 7d switch")
			}

		})
	}
}

func TestStaticAnimationTypeDependency(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewEnchantment("Animation", "", WithAbility(Static(AnimateEach(MatchingPermanents(HasSubType("Forest")), 1, 1)))), a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Mountain", WithSubTypes("Mountain")), b.PlayerID())
	change := BecomesBasicLandTargetEffect(land.ID(), core.EndOfTurn, "Forest")
	change.SetSourceID(source.ID())
	g.AddContinuousEffect(change)
	p := g.FindPermanent(land.ID())
	if !p.HasType(core.TypeCreature) || p.CurrentPower(g) != 1 || !p.HasSubType("Forest") {
		t.Fatal("animation ran before its land-type dependency")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(land.ID()).HasType(core.TypeCreature) || !g.FindPermanent(land.ID()).HasType(core.TypeCreature) {
		t.Fatal("dependency expiry changed original or retained selection")
	}
}

func TestStaticAnimationMetadataAndInvalidSet(t *testing.T) {
	card := NewEnchantment("Invalid", "", WithAbility(Static(AnimateEach(nil, 1, 1))))
	if err := ValidateCardDefinition("Invalid", card); err == nil {
		t.Fatal("nil animation set accepted")
	}
	rule := AnimateEach(MatchingPermanents(IsLand), 1, 1, core.Black).(*staticAnimationRule)
	for _, effect := range rule.staticEffects() {
		if !effect.Properties().Mass || effect.GetDuration() != core.WhileOnBattlefield {
			t.Fatal("missing static animation metadata")
		}
		if effect.GetLayer() == core.LayerPT && getEffectSublayer(effect) != core.SublayerPT_Set {
			t.Fatal("wrong base P/T sublayer")
		}
	}
}

func TestStaticAnimationCombatAndMana(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	land := g.PutOnBattlefield(NewLand("Defender land", WithSubTypes("Forest"), WithKeyword(core.Defender)), a.PlayerID())
	g.MutablePermanent(land.ID()).RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(land.ID()).Tapped = false
	source := g.PutOnBattlefield(NewEnchantment("Animation", "", WithAbility(Static(AnimateEach(MatchingPermanents(IsLand), 1, 1)))), b.PlayerID())
	p := g.FindPermanent(land.ID())
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("animation bypassed defender")
	}
	if !p.CanDeclareAsBlocker(g) {
		t.Fatal("animated land cannot block")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal("animation removed the land mana ability", err)
	}
	g.MutablePermanent(land.ID()).Tapped = false
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveKeyword(This, core.Defender)), land.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(land.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("untapped old animated land cannot attack")
	}
	g.AddContinuousEffect(PreventAttackingUntilEndOfTurn(land.ID()))
	if g.FindPermanent(land.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("animation bypassed attack restriction")
	}
	late := g.PutOnBattlefield(NewLand("Late land", WithSubTypes("Forest")), a.PlayerID())
	if late.CanDeclareAsAttacker(g) || late.CanTapForEffect(g) || !late.CanDeclareAsBlocker(g) {
		t.Fatal("animated land has wrong summoning sickness rules")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if !g.FindPermanent(late.ID()).CanTapForEffect(g) {
		t.Fatal("unanimated land cannot use mana")
	}
}

func TestStaticAnimationEntryAndIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("entry power")
	g.PutOnBattlefield(NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(AnotherCreatureEnters().WithPower(power), GainLifeFor(You, FromNumber(power))))), a.PlayerID())
	source := g.PutOnBattlefield(NewEnchantment("Animation", "", WithAbility(Static(AnimateEach(MatchingPermanents(IsLand), 1, 1)))), b.PlayerID())
	card := NewLand("Land", WithSubTypes("Forest"))
	card.SetOwner(b.PlayerID())
	land := g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	if g.Stack().Size() != 1 {
		t.Fatal("animated entry did not produce a creature entry event")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.incarnationID == land.incarnationID || returned.HasType(core.TypeCreature) {
		t.Fatal("animation followed a returned land")
	}
	g.ResolveTopOfStack()
	if a.Life() != 21 {
		t.Fatal("entry event lost the captured animated power")
	}
	g.PutOnBattlefield(source.Card, b.PlayerID())
	if !g.FindPermanent(card.ID()).HasType(core.TypeCreature) {
		t.Fatal("returned source did not select current land")
	}
}
