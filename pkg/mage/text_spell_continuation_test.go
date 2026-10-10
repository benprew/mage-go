package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextSpellContinuationDeclaration(t *testing.T) {
	for _, action := range []*ActionDefinition{
		NewActivated(TapCost(This), GainLifeFor(You, Fixed(1)), WithSpellTextContinuation()),
		NewSpell(GainLifeFor(You, Fixed(1)), WithCost(LifePayCost(1)), WithSpellTextContinuation()),
	} {
		card := NewInstant("Invalid", "", WithAbility(action))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid spell continuation marker accepted")
		}
	}
}

func TestTextSpellContinuationRejects(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	spell := NewInstant("Unmarked", "", Spell(GainLifeFor(You, Fixed(1))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	change, _ := ReplaceColorWord(core.Black, core.Red)
	for _, id := range []uuid.UUID{uuid.New(), spell.ID(), obj.ID} {
		if err := g.RevisePrintedSpellText(id, change); err == nil || g.StackPeek() != obj {
			t.Fatal("unchecked declaration changed live state")
		}
	}
}

func TestTextSpellContinuationContextRejection(t *testing.T) {
	for _, scenario := range []string{"marker", "group", "builder", "target-kind", "target-name", "target-count", "target-controller", "timing", "effect-count", "invalid-change", "empty-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(revised bool) CardFactory {
				return func() Card {
					name := "target"
					if revised && scenario == "target-name" {
						name = "other target"
					}
					target := DeclarePermanentTarget(name, IsCreature)
					if revised && scenario == "target-controller" {
						target.spec.(*PermanentTarget).controllerOnly = true
					}
					options := make([]ActionOption, 0, 6)
					options = append(options, WithEffects(TapSubject(target)), WithTarget(target.targetSpec()))
					if !revised || scenario != "marker" {
						options = append(options, WithSpellTextContinuation())
					}
					if revised {
						switch scenario {
						case "target-kind":
							player := DeclarePlayerTarget(name)
							options = append(options[:0], WithEffects(GainLifeFor(player, Fixed(1))), WithTarget(player.targetSpec()), WithSpellTextContinuation())
						case "target-count":
							options = append(options, WithTarget(DeclarePermanentTarget("second", IsCreature).targetSpec()))
						case "timing":
							options = append(options, WithTiming(TimingSorcery))
						case "effect-count":
							options = append(options, WithEffects(GainLifeFor(You, Fixed(1))))
						}
					}
					option := WithAbility(NewAction(ActionSpell, options...))
					if scenario != "group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					return NewInstant("Declaration", "", option)
				}
			}
			plan, err := NewCardDefinitionPlan(build(false))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			}
			g, a, b := newSpellCopyGame()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			p := g.PutOnBattlefield(NewCreature("Target", "", 2, 2), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			change, _ := ReplaceColorWord(core.Red, core.Green)
			changes := []TextChange{change}
			if scenario == "invalid-change" {
				changes = []TextChange{{}}
			}
			if scenario == "empty-change" {
				changes = nil
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil || g.StackPeek() != before {
				t.Fatal("incompatible continuation changed live state")
			}
			g.ResolveStack()
			if p := g.FindPermanent(p.ID()); p == nil || !p.Tapped {
				t.Fatal("rejection damaged the original binding")
			}
		})
	}
}

func TestTextSpellContinuationEmptyGroupAndCopy(t *testing.T) {
	build := func(changes []TextChange) CardFactory {
		return func() Card {
			color := core.Red
			for _, change := range changes {
				color = change.ApplyColor(color)
			}
			first := DeclarePermanentTargets("first", 0, 1, HasColorFilter(color))
			second := DeclarePermanentTargets("second", 0, 1, HasColorFilter(color))
			member := BindPermanent("member")
			action := Spell(
				Sequence(DestroyEach(first), ForEach(second, member, TapSubject(From(member)))),
				WithTargetGroups(first, second))
			WithSpellTextContinuation()(action)
			return NewInstant("Grouped", "", WithDefinitionAbilityGroup(0, WithAbility(action)))
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
	g, a, b := newSpellCopyGame()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	red := g.PutOnBattlefield(NewCreature("Red", "{R}", 2, 2), b.PlayerID())
	green := g.PutOnBattlefield(NewCreature("Green", "{G}", 2, 2), b.PlayerID())
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), card.Name(), [][]uuid.UUID{{}, {red.ID()}}); err != nil {
		t.Fatal(err)
	}
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	a.chooseQueue = [][]uuid.UUID{{green.ID()}}
	copyObj := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if copyObj == nil || len(copyObj.Targets) != 1 || copyObj.Targets[0] != green.ID() {
		t.Fatal("copy used printed target restrictions")
	}
	g.ResolveStack()
	if p := g.FindPermanent(green.ID()); p == nil || !p.Tapped {
		t.Fatal("copy lost the nonempty second group")
	}
	if p := g.FindPermanent(red.ID()); p == nil || p.Tapped {
		t.Fatal("original spell lost revised target legality")
	}
}
