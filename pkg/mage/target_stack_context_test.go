package mage

import (
	"slices"
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackManaValueTargetContext(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	ids := make([]uuid.UUID, 0, 3)
	for i, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID(), a.PlayerID()} {
		card := NewInstant("Target", "{3}")
		if i == 2 {
			card = NewCreature("Creature", "{3}", 1, 1)
		}
		g.pushStack(&StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: controller})
		ids = append(ids, card.ID())
	}
	source := NewInstant("Source", "{X}{U}")
	sourceSpell := &StackObject{ID: uuid.New(), Card: source, SourceID: source.ID(), Controller: a.PlayerID()}
	g.pushStack(sourceSpell)
	g.pushStack(&StackObject{ID: uuid.New(), Card: NewInstant("Ability", "{3}"), SourceID: uuid.New(), IsAbility: true})
	g.SetXValue(7)
	for _, tc := range []struct {
		name   string
		target Target
		want   []uuid.UUID
	}{
		{"any", TargetSpellOnStack(IsInstantCard, ManaValueIs(EqualTo(XValue()))), ids[:2]},
		{"own", TargetSpellOnStack(ObjectControlledBy(SelectController()), IsInstantCard, ManaValueIs(EqualTo(XValue()))), ids[:1]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !TargetsUseX([]Target{tc.target}) {
				t.Fatal("stack target lost its X restriction")
			}
			for _, ctx := range []TargetingContext{{}, {X: 3, XSpecified: true}, {X: 0, XSpecified: true}} {
				ctx.StackObjectID = sourceSpell.ID
				want := tc.want
				if ctx.XSpecified && ctx.X == 0 {
					want = nil
				}
				if got := PossibleTargets(tc.target, a.PlayerID(), source, g, ctx); !slices.Equal(got, want) {
					t.Fatalf("context %+v: targets = %v, want %v", ctx, got, want)
				}
				if g.XValue() != 7 {
					t.Fatal("target query changed resolution X")
				}
			}
		})
	}
}

func TestStackManaValueTargetXExpressions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value ValueSource
		want  []int
	}{
		{"X", XValue(), []int{4}},
		{"addition", Add(XValue(), Fixed(1)), []int{3}},
		{"multiplication", Mul(XValue(), Fixed(2)), []int{2}},
		{"half", HalfXRoundedDown(), []int{8, 9}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newPriorityTestGame()
			a, b := g.players[0], g.players[1]
			a.ManaPool().Add(Blue, 10)
			card := NewInstant("Target", "{X}{R}")
			g.pushStack(&StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), XValue: 3})
			g.PutOnBattlefield(NewArtifact("Unrelated", "{2}"), a.PlayerID())
			spell := NewInstant("Counter", "{X}{U}", NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(tc.value))), CounterSpell()))
			var got []int
			for _, option := range g.SpellTargetOptions(a.PlayerID(), spell, 0) {
				got = append(got, option.XValue)
				if len(option.Candidates) != 1 || !slices.Equal(option.Candidates[0], []uuid.UUID{card.ID()}) {
					t.Fatalf("unexpected candidates: %+v", option)
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("X values = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStackManaValueTargetRechecksAtResolution(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	a.ManaPool().Add(Blue, 4)
	b.ManaPool().Add(Red, 3)
	target := NewInstant("Target", "{3}", NewSpell(GainLife(1)))
	b.AddToHand(target)
	if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Counter", "{X}{U}", NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()))
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 3); err != nil {
		t.Fatal(err)
	}
	target.ApplyCopiableValues(NewInstant("Changed", "{2}", NewSpell(GainLife(1))).CopiableValues())
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatal("countered a target that no longer meets the mana value restriction")
	}
}

func TestStackManaValueTargetIncludesSpellCopies(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	a.ManaPool().Add(Blue, 6)
	b.ManaPool().Add(Red, 5)
	target := NewInstant("Target", "{X}{R}")
	b.AddToHand(target)
	if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 4); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(target.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("spell copy is missing")
	}
	g.SetXValue(0)
	spell := NewInstant("Counter", "{X}{U}", NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()))
	options := g.SpellTargetOptions(a.PlayerID(), spell)
	if len(options) != 1 || options[0].XValue != 5 || !slices.Equal(options[0].Candidates[0], []uuid.UUID{target.ID(), copied.SourceID}) {
		t.Fatalf("spell copy lost its mana value or identity: %+v", options)
	}
}

func TestCastTriggerCanTargetItsSourceSpell(t *testing.T) {
	g, a, _ := newModalGame()
	spell := NewInstant("Spell with cast trigger", "{0}", NewSpell(GainLife(1)),
		WithAbility(NewTriggered(EvtSpellCast, false, CounterSpell()).
			InZone(ZoneStack).
			SetConditionData(EventSourceIsSelf{}).
			AddTarget(TargetSpellOnStack(ObjectControlledBy(SelectController()), IsInstantCard))))
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	stack := g.StackObjects()
	if len(stack) != 2 || !stack[1].IsAbility || !slices.Equal(stack[1].Targets, []uuid.UUID{spell.ID()}) {
		t.Fatalf("cast trigger did not target its source spell: %v", stack)
	}
	target := TargetSpellOnStack(ObjectControlledBy(SelectController()))
	if got := PossibleTargets(target, a.PlayerID(), spell, g, TargetingContext{StackObjectID: stack[0].ID}); len(got) != 0 {
		t.Fatalf("a spell can target itself: %v", got)
	}
	if got := PossibleTargets(target, a.PlayerID(), spell, g, TargetingContext{StackObjectID: stack[1].ID}); !slices.Equal(got, []uuid.UUID{spell.ID()}) {
		t.Fatalf("the ability cannot target its source spell: %v", got)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("cast trigger did not counter its source spell")
	}
}

func TestStackTargetConstructorsAcceptMixedFilters(t *testing.T) {
	g, a, b := newModalGame()
	own := NewInstant("Own instant", "{U}")
	other := NewInstant("Other instant", "{U}")
	highCost := NewInstant("High cost instant", "{1}{U}")
	red := NewInstant("Red instant", "{R}")
	creature := NewCreature("Creature", "{U}", 1, 1)
	for _, obj := range []*StackObject{
		{ID: uuid.New(), Card: own, SourceID: own.ID(), Controller: a.PlayerID()},
		{ID: uuid.New(), Card: other, SourceID: other.ID(), Controller: b.PlayerID()},
		{ID: uuid.New(), Card: highCost, SourceID: highCost.ID(), Controller: a.PlayerID()},
		{ID: uuid.New(), Card: red, SourceID: red.ID(), Controller: b.PlayerID()},
		{ID: uuid.New(), Card: creature, SourceID: creature.ID(), Controller: a.PlayerID()},
	} {
		g.pushStack(obj)
	}
	filters := []ObjectFilter{IsInstantCard, ManaValueIs(EqualTo(Fixed(1))), ObjectHasColor(Blue)}
	if got := TargetSpellOnStack(filters...).Possible(a.PlayerID(), nil, g); !slices.Equal(got, []uuid.UUID{own.ID(), other.ID()}) {
		t.Fatalf("instant spell candidates = %v", got)
	}
	ownFilters := append([]ObjectFilter{ObjectControlledBy(SelectController())}, filters...)
	if got := TargetSpellOnStack(ownFilters...).Possible(a.PlayerID(), nil, g); !slices.Equal(got, []uuid.UUID{own.ID()}) {
		t.Fatalf("controlled instant spell candidates = %v", got)
	}
}
