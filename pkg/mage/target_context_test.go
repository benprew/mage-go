package mage

import (
	"fmt"
	"slices"
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaValueTargetContext(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	zero := g.PutOnBattlefield(NewArtifact("Zero", "{0}"), b.PlayerID())
	three := g.PutOnBattlefield(NewArtifact("Three", "{3}"), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Nonartifact", "{3}", 3, 3), b.PlayerID())
	target := TargetPermanent(ObjectAnd(IsCardType(TypeArtifact), ManaValueIs(EqualTo(XValue()))))
	source := NewSorcery("Test", "{X}{R}")
	g.SetXValue(7)
	for _, tc := range []struct {
		ctx  TargetingContext
		want []uuid.UUID
	}{
		{TargetingContext{}, []uuid.UUID{zero.ID(), three.ID()}},
		{TargetingContext{X: 0, XSpecified: true}, []uuid.UUID{zero.ID()}},
		{TargetingContext{X: 3, XSpecified: true}, []uuid.UUID{three.ID()}},
		{TargetingContext{X: 2, XSpecified: true}, nil},
	} {
		if got := PossibleTargets(target, a.PlayerID(), source, g, tc.ctx); !slices.Equal(got, tc.want) {
			t.Fatalf("context %+v: got %v, want %v", tc.ctx, got, tc.want)
		}
		if g.XValue() != 7 {
			t.Fatal("target query changed resolution X")
		}
	}
}

func TestSpellTargetOptions(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	zero := g.PutOnBattlefield(NewArtifact("Zero", "{0}"), b.PlayerID())
	three := g.PutOnBattlefield(NewArtifact("Three", "{3}"), b.PlayerID())
	g.PutOnBattlefield(NewArtifact("Too expensive", "{6}"), b.PlayerID())
	a.ManaPool().Add(Red, 5)
	spell := NewSorcery("Test", "{X}{R}", NewTargetedSpell(TargetPermanent(IsCardType(TypeArtifact), ManaValueIs(EqualTo(XValue()))), DestroyTarget()))
	options := g.SpellTargetOptions(a.PlayerID(), spell)
	if len(options) != 2 || options[0].XValue != 0 || options[1].XValue != 3 {
		t.Fatalf("unexpected options: %+v", options)
	}
	if !slices.Equal(options[0].Candidates[0], []uuid.UUID{zero.ID()}) || !slices.Equal(options[1].Candidates[0], []uuid.UUID{three.ID()}) {
		t.Fatalf("wrong target pairs: %+v", options)
	}
}

func TestManaValueTargetCastAndResolution(t *testing.T) {
	for _, x := range []int{0, 3} {
		t.Run(fmt.Sprint(x), func(t *testing.T) {
			g := newPriorityTestGame()
			g.SetStep(PrecombatMain)
			a, b := g.players[0], g.players[1]
			victim := g.PutOnBattlefield(NewArtifact("Victim", fmt.Sprintf("{%d}", x)), b.PlayerID())
			spell := NewSorcery("Test", "{X}{R}", NewTargetedSpell(TargetArtifactWithManaValueX(), DestroyTarget()))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 10)
			g.SetXValue(8)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{victim.ID()}, x+1); err == nil {
				t.Fatal("accepted wrong announced X")
			}
			if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 10 || !g.Stack().IsEmpty() || g.XValue() != 8 {
				t.Fatal("failed cast changed state")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{victim.ID()}, x); err != nil {
				t.Fatal(err)
			}
			g.SetXValue(9)
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("resolution did not use stack X")
			}
		})
	}
}

func TestManaValueTargetRechecksCharacteristics(t *testing.T) {
	g := newPriorityTestGame()
	g.SetStep(PrecombatMain)
	a, b := g.players[0], g.players[1]
	victim := g.PutOnBattlefield(NewArtifact("Victim", "{3}"), b.PlayerID())
	spell := NewSorcery("Test", "{X}{R}", NewTargetedSpell(TargetArtifactWithManaValueX(), DestroyTarget()))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Red, 4)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{victim.ID()}, 3); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(victim.ID()).Card.ApplyCopiableValues(NewArtifact("Changed", "{2}").CopiableValues())
	g.ResolveStack()
	if g.FindPermanent(victim.ID()) == nil {
		t.Fatal("destroyed a target that no longer meets the restriction")
	}
}

func TestManaValueFilterStructure(t *testing.T) {
	filter := ObjectAnd(IsCardType(TypeArtifact), ManaValueIs(EqualTo(XValue())))
	if filter.Kind() != "And" || !filter.UsesX() {
		t.Fatal("lost filter structure")
	}
	children := filter.Children()
	comparison, ok := children[1].ManaValueComparison()
	if !ok || comparison.Operator != "EqualTo" {
		t.Fatal("lost comparison structure")
	}
	if _, ok := comparison.Value.(xValue); !ok {
		t.Fatal("lost X value source")
	}
	children[0] = ObjectFilter{}
	if typ, ok := filter.Children()[0].CardType(); !ok || typ != TypeArtifact {
		t.Fatal("inspection changed filter")
	}
}

func TestManaValueComparisonComposition(t *testing.T) {
	g := newPriorityTestGame()
	a, b := g.players[0], g.players[1]
	zero := g.PutOnBattlefield(NewArtifact("Zero", "{0}"), b.PlayerID())
	three := g.PutOnBattlefield(NewArtifact("Three", "{3}"), b.PlayerID())
	source := NewSorcery("Test", "{X}{R}")
	for _, tc := range []struct {
		name   string
		filter ObjectFilter
		ctx    TargetingContext
		want   []uuid.UUID
	}{
		{"fixed", ManaValueIs(EqualTo(Fixed(3))), TargetingContext{}, []uuid.UUID{three.ID()}},
		{"negated unspecified", ObjectNot(ManaValueIs(EqualTo(XValue()))), TargetingContext{}, []uuid.UUID{zero.ID(), three.ID()}},
		{"negated zero", ObjectNot(ManaValueIs(EqualTo(XValue()))), TargetingContext{X: 0, XSpecified: true}, []uuid.UUID{three.ID()}},
		{"union", ObjectOr(ManaValueIs(EqualTo(XValue())), ManaValueIs(EqualTo(Fixed(3)))), TargetingContext{X: 0, XSpecified: true}, []uuid.UUID{zero.ID(), three.ID()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := TargetPermanent(IsCardType(TypeArtifact), tc.filter)
			if got := PossibleTargets(target, a.PlayerID(), source, g, tc.ctx); !slices.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestManaValueCopyRetainsAnnouncedX(t *testing.T) {
	g := newPriorityTestGame()
	g.SetStep(PrecombatMain)
	a, b := g.players[0], g.players[1]
	g.PutOnBattlefield(NewArtifact("Zero", "{0}"), b.PlayerID())
	victim := g.PutOnBattlefield(NewArtifact("Three", "{3}"), b.PlayerID())
	spell := NewSorcery("Test", "{X}{R}", NewTargetedSpell(TargetArtifactWithManaValueX(), DestroyTarget()))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Red, 4)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{victim.ID()}, 3); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(0)
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if copied == nil || copied.XValue != 3 || !slices.Equal(copied.Targets, []uuid.UUID{victim.ID()}) {
		t.Fatalf("wrong copy targets: %+v", copied)
	}
	if g.XValue() != 0 {
		t.Fatal("copy targeting changed resolution state")
	}
}

func TestManaValueTargetXExpressions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value ValueSource
		wantX []int
	}{
		{"X", XValue(), []int{4}},
		{"addition", Add(Fixed(1), XValue()), []int{3}},
		{"multiplication", Mul(XValue(), Fixed(2)), []int{2}},
		{"half down", HalfXRoundedDown(), []int{8, 9}},
		{"half up", HalfXRoundedUp(), []int{7, 8}},
		{"nested half", HalfRoundUp(Add(XValue(), Fixed(1))), []int{6, 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newPriorityTestGame()
			a, b := g.players[0], g.players[1]
			a.ManaPool().Add(Red, 10)
			victim := g.PutOnBattlefield(NewArtifact("Four", "{4}"), b.PlayerID())
			filter := ManaValueIs(EqualTo(tc.value))
			target := TargetPermanent(IsCardType(TypeArtifact), filter)
			spell := NewSorcery("Removal", "{X}{R}", NewTargetedSpell(target, DestroyTarget()))
			if !TargetsUseX(spell.CastTargets()) {
				t.Error("X expression was not detected")
			}
			if got := PossibleTargets(target, a.PlayerID(), spell, g, TargetingContext{}); !slices.Equal(got, []uuid.UUID{victim.ID()}) {
				t.Errorf("unspecified X excluded a candidate: %v", got)
			}
			var gotX []int
			for _, option := range g.SpellTargetOptions(a.PlayerID(), spell, 0) {
				gotX = append(gotX, option.XValue)
				if !slices.Equal(option.Candidates[0], []uuid.UUID{victim.ID()}) {
					t.Fatalf("wrong candidates: %+v", option)
				}
			}
			if !slices.Equal(gotX, tc.wantX) {
				t.Fatalf("X values = %v, want %v", gotX, tc.wantX)
			}
		})
	}
}

func TestCopySpellRetainsXTargetCount(t *testing.T) {
	g, a, b := newModalGame()
	g.SetStep(PrecombatMain)
	a.ManaPool().Add(Red, 3)
	first := g.PutOnBattlefield(NewArtifact("First", "{2}"), b.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", "{2}"), b.PlayerID())
	spell := NewSorcery("Removal", "{X}{R}", NewTargetedSpell(TargetXPermanents(), DestroyTarget()))
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(0)
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{first.ID(), second.ID()}) {
		t.Fatalf("copy lost required targets: %+v", copied)
	}
}

func TestOpponentChoosesXTargets(t *testing.T) {
	g, a, b := newModalGame()
	g.SetStep(PrecombatMain)
	a.ManaPool().Add(Red, 3)
	first := g.PutOnBattlefield(NewArtifact("First", "{2}"), b.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", "{2}"), b.PlayerID())
	spell := NewSorcery("Removal", "{X}{R}", NewTargetedSpell(TargetOpponentChoice(TargetXPermanents()), DestroyTarget()))
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	obj := g.StackObjects()[0]
	if !slices.Equal(obj.Targets, []uuid.UUID{first.ID(), second.ID()}) {
		t.Fatalf("wrong opponent target count: %v", obj.Targets)
	}
}

func TestFreeModalCastPromptsWithAnnouncedX(t *testing.T) {
	g, a, b := newModalGame()
	zero := g.PutOnBattlefield(NewArtifact("Zero", "{0}"), b.PlayerID())
	g.PutOnBattlefield(NewArtifact("Three", "{3}"), b.PlayerID())
	spell := NewSorcery("Removal", "{X}{R}", NewModalSpell([]Mode{
		{Label: "Destroy", Targets: []Target{TargetArtifactWithManaValueX()}, Effects: []Effect{DestroyTarget()}},
		{Label: "Gain life", Effects: []Effect{GainLife(1)}},
	}))
	a.AddToGraveyard(spell)
	g.SetXValue(3)
	if err := g.CastCardFromZoneWithoutPaying(a.PlayerID(), spell.ID(), ZoneGraveyard, nil, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackObjects()[0]
	if obj.XValue != 0 || !slices.Equal(obj.Targets, []uuid.UUID{zero.ID()}) {
		t.Fatalf("wrong free cast announcement: %+v", obj)
	}
	if g.XValue() != 3 {
		t.Fatal("free cast changed resolution X")
	}
}
