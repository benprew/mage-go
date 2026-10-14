package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type variantChooser struct {
	*BasePlayer
	answer, choices int
	inspect         func()
}

func (p *variantChooser) ChooseMode(_ []string, _ string) int {
	p.choices++
	if p.inspect != nil {
		p.inspect()
	}
	return p.answer
}

func variantPlan(t *testing.T) *CardDefinitionPlan {
	t.Helper()
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Variant", "", 0, 0, WithCardType(core.TypeArtifact), WithSubTypes("Shapeshifter"), WithAbility(ChooseCopiableVariantAsEnters(
			[]CopyModification{CopySetPowerToughness(3, 3)},
			[]CopyModification{CopySetPowerToughness(2, 2), CopyAddKeywords(core.Flying)},
			[]CopyModification{CopySetPowerToughness(1, 6), CopyAddSubTypes("Wall"), CopyAddKeywords(core.Defender)},
		)))
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestEntryVariantTimingIdentity(t *testing.T) {
	a := &variantChooser{BasePlayer: NewBasePlayer("A"), answer: 2}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	plan := variantPlan(t)
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(b.PlayerID())
	a.inspect = func() {
		if len(g.AllBattlefield()) != 0 {
			t.Fatal("choice after entry")
		}
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.inspect = nil
	if p == nil || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 6 || !p.HasSubType("Wall") || !p.HasSubType("Shapeshifter") || !p.HasKeyword(core.Defender) || a.choices != 1 {
		t.Fatal("wrong variant or repeated choice")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	if copyCard.Power() != 1 || copyCard.Toughness() != 6 || !cardHasKeyword(copyCard, core.Defender) {
		t.Fatal("variant not copiable")
	}
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 2)
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(4), Fixed(5))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	values, err = g.CaptureCopiableValues(g.FindPermanent(p.ID()))
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err = values.Build()
	if err != nil || copyCard.Power() != 1 || copyCard.Toughness() != 6 {
		t.Fatal("ordinary changes copied", err)
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	if a.choices != 1 {
		t.Fatal("phase cycle repeated choice")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if b.Hand()[0].Power() != 0 || b.Hand()[0].Toughness() != 0 || cardHasKeyword(b.Hand()[0], core.Defender) {
		t.Fatal("departure retained variant")
	}
	b.RemoveFromHand(card.ID())
	a.answer = 1
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.CurrentPower(g) != 2 || returned.CurrentToughness(g) != 2 || !returned.HasKeyword(core.Flying) || returned.HasKeyword(core.Defender) || a.choices != 2 {
		t.Fatal("return did not choose fresh variant")
	}
	if clone.FindPermanent(p.ID()).Card.Power() != 1 || !clone.FindPermanent(p.ID()).HasKeyword(core.Defender) {
		t.Fatal("return changed clone")
	}
}

func TestEntryVariantInvalid(t *testing.T) {
	for _, ability := range []Ability{
		ChooseCopiableVariantAsEnters(), ChooseCopiableVariantAsEnters(nil), ChooseCopiableVariantAsEnters([]CopyModification{nil}),
		ChooseCopiableVariantAsEnters([]CopyModification{CopyAddKeywords()}), ChooseCopiableVariantAsEnters([]CopyModification{CopyAddKeywords(core.Attr(-1))}),
		ChooseCopiableVariantAsEnters([]CopyModification{CopyAddSubTypes()}), ChooseCopiableVariantAsEnters([]CopyModification{CopyAddSubTypes("")}),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(ability))); err == nil {
			t.Fatal("invalid variant accepted")
		}
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(ChooseCopiableVariantAsEnters([]CopyModification{CopySetPowerToughness(1, 1)})))); err == nil {
		t.Fatal("instant entry variant accepted")
	}
}

func TestEntryVariantPriorityAndCopy(t *testing.T) {
	a := &variantChooser{BasePlayer: NewBasePlayer("A"), answer: 1}
	b := &optionalEntryPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
	g := NewGame(a, b)
	card, err := variantPlan(t).Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	seen := 0
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action { seen++; return action }})
	copyPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(copyCard, b.PlayerID())
	if p == nil || p.Name() != source.Name() || seen != 1 || !p.HasKeyword(core.Flying) {
		t.Fatal("copy did not acquire entry variant or lost prior keyword")
	}
	if a.choices != 1 {
		t.Fatal("copy used original controller")
	}
}

func TestEntryVariantInstancesAndBatch(t *testing.T) {
	a := &variantChooser{BasePlayer: NewBasePlayer("A"), answer: -1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	plan := variantPlan(t)
	cards := make([]Card, 0, 2)
	for range 2 {
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, card)
	}
	a.inspect = func() {
		if len(g.AllBattlefield()) != 0 {
			t.Fatal("batch registered before all choices")
		}
	}
	entrants := g.putTokensOnBattlefieldTogether(cards, a.PlayerID())
	a.inspect = nil
	if len(entrants) != 2 || a.choices != 2 {
		t.Fatal("batch did not choose independently")
	}
	for _, p := range entrants {
		if p.Card.Power() != 3 || p.Card.Toughness() != 3 {
			t.Fatal("invalid answer did not use first variant")
		}
	}
	first := entrants[0]
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if a.choices != 2 || g.FindPermanent(first.ID()).Card.Power() != 3 {
		t.Fatal("control repeated choice")
	}
	otherPlan, err := NewCardDefinitionPlan(func() Card {
		kinds := []CopyModification{CopySetPowerToughness(5, 4)}
		ability := ChooseCopiableVariantAsEnters(kinds)
		kinds[0] = CopySetPowerToughness(1, 1)
		return NewCreature("Snapshot", "", 0, 0, WithAbility(ability))
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := otherPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p == nil || p.Card.Power() != 5 || first.Card.Power() != 3 {
		t.Fatal("caller mutation changed variant or another instance")
	}
}
