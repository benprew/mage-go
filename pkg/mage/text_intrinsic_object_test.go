package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextIntrinsicObjectAtomic(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	p := g.PutOnBattlefield(NewCreature("Unsupported", "", 2, 2, WithAbility(ProtectionFromColors(core.White).WithLayeredColorTextRevision()), WithAbility(NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1)))))), a.PlayerID())
	change, _ := ReplaceColorWord(core.White, core.Blue)
	before := len(g.effects.effects)
	if err := g.AddPrintedIntrinsicCreatureTextEffect(p, core.Indefinite, change); err == nil {
		t.Fatal("partial object accepted")
	}
	if len(g.effects.effects) != before || !g.FindPermanent(p.ID()).HasProtectionFrom(NewCreature("White", "{W}", 1, 1)) {
		t.Fatal("rejection changed object")
	}
}

func TestTextIntrinsicObjectBindingDeclarations(t *testing.T) {
	pair := BindColorWordPair("unbound")
	change := ColorWordSubstitution(FirstColorWord(pair), SecondColorWord(pair))
	for _, effect := range []ResolvingEffect{ChangePrintedIntrinsicCreatureText(This, core.Indefinite, change), ChangePrintedIntrinsicCreatureText(This, core.Indefinite), ChangePrintedIntrinsicCreatureText(This, core.EndOfCombat, change), ChangePrintedIntrinsicCreatureText(This, core.Indefinite, nil)} {
		card := NewCreature("Invalid", "", 2, 2, WithAbility(NewAction(ActionActivated, WithEffect(effect))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid text declaration accepted")
		}
	}
}

func intrinsicObjectTestCard(t *testing.T) Card {
	t.Helper()
	build := func(changes []TextChange) CardFactory {
		return func() Card {
			white, red := core.White, core.Red
			for _, change := range changes {
				white = change.ApplyColor(white)
				red = change.ApplyColor(red)
			}
			return NewCreature("Intrinsic", "{W}", 2, 2, WithIntrinsicCreatureSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(ProtectionFromColors(white).WithLayeredColorTextRevision())), WithDefinitionAbilityGroup(1, WithAbility(ProtectionFromColors(red).WithLayeredColorTextRevision())))
		}
	}
	plan, err := NewCardDefinitionPlan(build(nil))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) { return build(changes), nil })(card.(*BaseCard))
	return card
}

func TestTextIntrinsicObjectPreflight(t *testing.T) {
	for _, scenario := range []string{"invalid-word", "duration", "missing-builder", "missing-last-declaration", "face-down", "phased", "nil", "departed"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := newSpellCopyGame()
			card := intrinsicObjectTestCard(t)
			p := g.PutOnBattlefield(card, a.PlayerID())
			change, _ := ReplaceColorWord(core.White, core.Blue)
			duration := core.Indefinite
			switch scenario {
			case "invalid-word":
				change = TextChange{}
			case "duration":
				duration = core.EndOfCombat
			case "missing-builder":
				p.Card.(*BaseCard).textDefinition = nil
			case "missing-last-declaration":
				g.MutablePermanent(p.ID()).RuntimeAbilities = g.FindPermanent(p.ID()).RuntimeAbilities[:1]
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "nil":
				p = nil
			case "departed":
				g.BouncePermanentToHand(p)
			}
			count := len(g.effects.effects)
			if err := g.AddPrintedIntrinsicCreatureTextEffect(p, duration, change); err == nil {
				t.Fatal("invalid object accepted")
			}
			if len(g.effects.effects) != count {
				t.Fatal("preflight installed partial changes")
			}
		})
	}
}
