package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTextSpellActionDeclaration(t *testing.T) {
	for _, scenario := range []string{"nil-subject", "empty", "nil-change", "unbound"} {
		t.Run(scenario, func(t *testing.T) {
			target := DeclareSpellTarget("subject")
			pair := BindColorWordPair("words")
			var subject StackObjectRef = target
			changes := []TextChangeRef{ColorWordSubstitution(FirstColorWord(pair), SecondColorWord(pair))}
			switch scenario {
			case "nil-subject":
				subject = nil
			case "empty":
				changes = nil
			case "nil-change":
				changes[0] = nil
			}
			effect := ChangePrintedSpellText(subject, changes...)
			card := NewInstant("Invalid", "", Spell(effect, WithTargets(target)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
}

func TestTextSpellActionAtomic(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	card := NewInstant("Unmarked", "", Spell(GainLifeFor(You, Fixed(1))))
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	binding := BindSpell("subject")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: obj.ID}
	pair := BindColorWordPair("words")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := ChangePrintedSpellText(FromSpell(binding), ColorWordSubstitution(FirstColorWord(pair), SecondColorWord(pair)))
	if err := effect.Apply(ctx); err == nil || g.StackPeek() != obj {
		t.Fatal("missing saved words changed the stack")
	}
	chooser := &textWordTester{Player: a, answers: []string{"Red", "Blue"}}
	g.SetPlayerAt(0, chooser)
	if err := ChooseColorWordPair(You, pair).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := effect.Apply(ctx); err == nil || g.StackPeek() != obj || chooser.calls != 2 {
		t.Fatal("unmarked spell changed or action repeated choices")
	}
	g.CounterStackObject(obj.ID)
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	returned := g.StackPeek()
	if returned.ID == obj.ID {
		t.Fatal("cast reused stack identity")
	}
	if err := effect.Apply(ctx); err != nil || g.StackPeek() != returned {
		t.Fatal("saved reference followed another cast", err)
	}
}

func TestTextSpellActionSavedList(t *testing.T) {
	build := func() Card {
		return NewInstant("Subject", "", WithDefinitionAbilityGroup(0,
			WithAbility(NewSpell(GainLifeFor(You, Fixed(1)), WithSpellTextContinuation()))))
	}
	plan, err := NewCardDefinitionPlan(build)
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build, nil })(card.(*BaseCard))
	g, a, _ := newSpellCopyGame()
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	binding := BindSpell("subject")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: obj.ID}
	pair := BindColorWordPair("words")
	missing := BindColorWordPair("missing")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	chooser := &textWordTester{Player: a, answers: []string{"Red", "Blue"}}
	g.SetPlayerAt(0, chooser)
	if err := ChooseColorWordPair(You, pair).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	valid := ColorWordSubstitution(FirstColorWord(pair), SecondColorWord(pair))
	invalid := ColorWordSubstitution(FirstColorWord(missing), SecondColorWord(missing))
	if err := ChangePrintedSpellText(FromSpell(binding), valid, invalid).Apply(ctx); err == nil || g.StackPeek() != obj {
		t.Fatal("partial list changed the spell")
	}
	refs := []TextChangeRef{valid}
	effect := ChangePrintedSpellText(FromSpell(binding), refs...)
	refs[0] = invalid
	if err := effect.Apply(ctx); err != nil || g.StackPeek() == obj || chooser.calls != 2 {
		t.Fatal("valid no-match revision failed or caller changed saved list", err)
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("no-match revision damaged the body")
	}
}
