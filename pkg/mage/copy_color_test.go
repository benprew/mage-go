package mage

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCopyColorSnapshots(t *testing.T) {
	plan, err := NewCardDefinitionPlan(func() Card {
		c := NewCreature("Donor", "{G}", 2, 2, WithAbility(EntersWithNCounters(core.P1P1, 2)), WithAbility(StaticAbility(PTEqualsControlledCount(IsCreature))))
		c.SetColorOverride([]core.Color{core.Red})
		return c
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, colors := range [][]core.Color{nil, {}, {core.Colorless}, {core.Blue}, {core.Blue, core.White}, {core.Blue, core.Blue}} {
		want := make([]core.Color, 0, len(colors))
		for _, color := range colors {
			if color != core.Colorless && !slices.Contains(want, color) {
				want = append(want, color)
			}
		}
		input := slices.Clone(colors)
		mod := CopyRetainColors(input...)
		if len(input) > 0 {
			input[0] = core.Black
		}
		values, err := plan.CopiableValues().WithModifications(mod, CopyAddCardTypes(core.TypeEnchantment))
		if err != nil {
			t.Fatal(err)
		}
		first, err := values.Build()
		if err != nil {
			t.Fatal(err)
		}
		second, err := values.Build()
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(cardColors(first), want) || !slices.Equal(cardColors(second), want) || first.ManaCost().Green != 1 || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
			t.Fatal("saved colors, mana cost or independent rules")
		}
		original, err := plan.CopiableValues().Build()
		if err != nil || !slices.Equal(cardColors(original), []core.Color{core.Red}) {
			t.Fatal("changed donor", err)
		}
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(first, a.PlayerID())
		if !slices.Equal(p.Colors(), want) || p.Counters[core.P1P1] != 2 || p.CurrentPower(g) != 3 {
			t.Fatal("entry colors or rules")
		}
		g.ChangeObjectColor(p.ID(), core.Black)
		captured, err := g.CaptureCopiableValues(p)
		if err != nil {
			t.Fatal(err)
		}
		rebuilt, err := captured.Build()
		if err != nil || !slices.Equal(cardColors(rebuilt), want) {
			t.Fatal("ordinary layer entered snapshot", err)
		}
		if len(want) == 0 {
			p.Card.(*BaseCard).SetColorOverride(nil)
			if _, err := g.CaptureCopiableValues(p); err == nil {
				t.Fatal("absent override replaced an explicit colorless snapshot")
			}
		}
	}
	for _, color := range []core.Color{core.AnyColor, core.Color(-1)} {
		if _, err := plan.CopiableValues().WithModifications(CopyRetainColors(color)); err == nil {
			t.Fatal("invalid color accepted")
		}
	}
}

// The test declaration distinguishes color definitions from ordinary color effects.
type testCopyColorDefinition struct{ effectSource }

func (*testCopyColorDefinition) DefinesColor() bool              { return true }
func (*testCopyColorDefinition) GetLayer() core.Layer            { return core.LayerColor }
func (*testCopyColorDefinition) GetDuration() core.Duration      { return core.WhileOnBattlefield }
func (*testCopyColorDefinition) Text() string                    { return "color definition" }
func (*testCopyColorDefinition) Properties() EffectProperties    { return EffectProperties{} }
func (e *testCopyColorDefinition) IsActive(g *Game) bool         { return g.FindPermanent(e.SourceID()) != nil }
func (e *testCopyColorDefinition) cloneEffect() ContinuousEffect { c := *e; return &c }
func (e *testCopyColorDefinition) ApplyContinuous(ctx *EffectContext) error {
	if p := ctx.Game.MutablePermanent(e.SourceID()); p != nil {
		colors := []core.Color{core.Red}
		p.ColorOverride = &colors
	}
	return nil
}

func TestCopyColorDefinitionAndReplacementIdentity(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := &optionalEntryPlayer{BasePlayer: NewBasePlayer("B"), accept: false}
	g := NewGame(a, b)
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Cycle", "{U}", 2, 2,
			WithAbility(StaticAbility(&testCopyColorDefinition{})),
			WithAbility(MayCopyAsEntersKeepingColors(IsCreature)),
			WithAbility(EntersWithNCounters(core.P1P1, 2)),
			WithAbility(Static(While(Controls(IsCreature), SetColors(This, core.Green)))))
	})
	if err != nil {
		t.Fatal(err)
	}
	donor, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p == nil || a.offers != 1 || len(p.Card.Abilities()) != 3 || p.Counters[core.P1P1] != 2 || !slices.Equal(p.Colors(), []core.Color{core.Green}) {
		t.Fatal("color CDA removal, ordinary color ability, replacement repetition", a.offers)
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := values.Build()
	if err != nil || !slices.Equal(cardColors(rebuilt), []core.Color{core.Blue}) {
		t.Fatal("retained base color", err)
	}
	if len(source.Card.Abilities()) != 4 {
		t.Fatal("removed donor ability")
	}
}

func TestCopyColorBatchAndPriority(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithAbility(EntersWithNCounters(core.P1P1, 2))} })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, b.PlayerID())
	seen := 0
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
		p := action.(*battlefieldEntryAction).permanent
		if p.Name() != "Donor" || p.Counters[core.P1P1] != 0 {
			t.Fatal("copy did not precede later replacements")
		}
		if seen == 0 && !slices.Equal(p.Colors(), []core.Color{core.Blue}) || seen == 1 && len(p.Colors()) != 0 {
			t.Fatal("batch shared retained colors")
		}
		seen++
		return action
	}})
	cards := make([]Card, 0, 2)
	for _, cost := range []string{"{U}", ""} {
		plan, err := NewCardDefinitionPlan(func() Card {
			return NewCreature("Physical", cost, 1, 1, WithAbility(MayCopyAsEntersKeepingColors(IsCreature)))
		})
		if err != nil {
			t.Fatal(err)
		}
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, card)
	}
	entries := g.putTokensOnBattlefieldTogether(cards, a.PlayerID())
	if len(entries) != 2 || seen != 2 || a.offers != 2 || entries[0].Counters[core.P1P1] != 2 || entries[1].Counters[core.P1P1] != 2 {
		t.Fatal("independent entry batch")
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(MayCopyAsEntersKeepingColors(IsCreature)))); err == nil {
		t.Fatal("nonpermanent entry accepted")
	}
	if err := ValidateCardDefinition("Invalid", NewCreature("Invalid", "", 1, 1, WithAbility(MayCopyAsEntersKeepingColors(IsCreature, CopyRetainColors(core.AnyColor))))); err == nil {
		t.Fatal("invalid colors accepted")
	}
}
