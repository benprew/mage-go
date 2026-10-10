package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextModalSpellContinuationDeclaration(t *testing.T) {
	for _, scenario := range []string{"mode-count", "cost", "capture", "effects", "targets", "options", "activated"} {
		t.Run(scenario, func(t *testing.T) {
			modal := NewModalSpell([]Mode{{Label: "First", Effects: []Effect{GainLifeFor(You, Fixed(1))}}, {Label: "Second", Effects: []Effect{GainLifeFor(You, Fixed(2))}}}).WithTextContinuation()
			switch scenario {
			case "mode-count":
				modal.modes = modal.modes[:1]
			case "cost":
				modal.costs = []Cost{LifePayCost(1)}
			case "capture":
				modal.dslCaptures = []activationCapture{nil}
			case "effects":
				modal.effects = []Effect{GainLifeFor(You, Fixed(1))}
			case "targets":
				modal.targets = []Target{DeclarePermanentTarget("extra", IsCreature).targetSpec()}
			case "options":
				modal.dslOptions = []ModalSpellOption{nil}
			case "activated":
				modal.kind = ActionActivated
			}
			if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(modal))); err == nil || !strings.Contains(err.Error(), "modal spell text continuation") {
				t.Fatalf("invalid modal continuation diagnostic: %v", err)
			}
		})
	}
}

func TestTextModalSpellContinuationContextRejection(t *testing.T) {
	for _, scenario := range []string{"marker", "group", "builder", "mode-count", "unchosen-target", "target-name", "target-count", "target-controller", "timing", "effect-count", "mode-choice", "modal-targets", "casting-bindings", "empty-change", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(revised bool) CardFactory {
				return func() Card {
					name := "chosen"
					if revised && scenario == "target-name" {
						name = "other"
					}
					chosen := DeclarePermanentTarget(name, IsCreature)
					if revised && scenario == "target-controller" {
						chosen.spec.(*PermanentTarget).controllerOnly = true
					}
					unchosen := DeclarePlayerTarget("unchosen")
					modes := []Mode{{Label: "Chosen", Targets: []Target{chosen.targetSpec()}, Effects: []Effect{TapSubject(chosen)}}, {Label: "Unchosen", Targets: []Target{unchosen.targetSpec()}, Effects: []Effect{GainLifeFor(unchosen, Fixed(2))}}}
					if revised {
						switch scenario {
						case "mode-count":
							modes = append(modes, Mode{Label: "Third", Effects: []Effect{GainLifeFor(You, Fixed(3))}})
						case "unchosen-target":
							modes[1].Targets = []Target{DeclarePermanentTarget("unchosen", IsCreature).targetSpec()}
							modes[1].Effects = []Effect{GainLifeFor(You, Fixed(2))}
						case "target-count":
							modes[0].Targets = append(modes[0].Targets, DeclarePermanentTarget("second", IsCreature).targetSpec())
						case "effect-count":
							modes[1].Effects = append(modes[1].Effects, GainLifeFor(You, Fixed(3)))
						}
					}
					modal := NewModalSpell(modes)
					if !revised || scenario != "marker" {
						modal.WithTextContinuation()
					}
					if revised && scenario == "timing" {
						modal.timing = TimingSorcery
					}
					option := WithAbility(modal)
					if scenario != "group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					return NewInstant("Modal", "", option)
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
			switch scenario {
			case "mode-choice":
				before.ModeChoice = 2
			case "modal-targets":
				before.ModalTargets[1] = []uuid.UUID{b.PlayerID()}
			case "casting-bindings":
				before.capturedNumbers = map[*NumberBinding]int{BindNumber("paid"): 2}
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			changes := []TextChange{change}
			if scenario == "empty-change" {
				changes = nil
			}
			if scenario == "invalid-change" {
				changes = []TextChange{{}}
			}
			if err := g.RevisePrintedSpellText(before.ID, changes...); err == nil || g.StackPeek() != before {
				t.Fatal("incompatible modal continuation changed live state")
			}
			g.ResolveTopOfStack()
			if !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("rejection changed original binding")
			}
		})
	}
}

func TestTextModalSpellContinuationEmptyGroupAndCopy(t *testing.T) {
	build := func(changes []TextChange) CardFactory {
		return func() Card {
			color := core.Red
			for _, change := range changes {
				color = change.ApplyColor(color)
			}
			first := DeclarePermanentTargets("empty", 0, 1, HasColorFilter(color))
			second := DeclarePermanentTarget("chosen", HasColorFilter(color))
			modes := []Mode{
				{Label: "Chosen", Targets: []Target{first.targetSpec(), second.targetSpec()}, Effects: []Effect{DestroyEach(first), TapSubject(second)}},
				{Label: "Other", Effects: []Effect{GainLifeFor(You, Fixed(5))}},
			}
			return NewInstant("Grouped modal", "", WithDefinitionAbilityGroup(0, WithAbility(NewModalSpell(modes).WithTextContinuation())))
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
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{red.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	a.chooseQueue = [][]uuid.UUID{{green.ID()}}
	copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != green.ID() {
		t.Fatal("copy lost the nonempty second declaration")
	}
	g.ResolveTopOfStack()
	if p := g.FindPermanent(green.ID()); p == nil || !p.Tapped {
		t.Fatal("second group shifted into the empty first group")
	}
	g.ResolveTopOfStack()
	if p := g.FindPermanent(red.ID()); p == nil || p.Tapped {
		t.Fatal("original lost revised target legality")
	}
}
