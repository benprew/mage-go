package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAddCountersEachReplacementAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
	g.PhaseOut(phased.ID())
	g.AddCounterDoubler(first.ID(), core.P1P1, IsCreature)
	clone := g.Clone()
	if err := AddCountersEach(ControlledPermanents(IsCreature), core.P1P1, Fixed(2)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID(), SourceID: first.ID()}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(first.ID()).Counters[core.P1P1] != 4 || clone.FindPermanent(second.ID()).Counters[core.P1P1] != 4 || phased.Counters[core.P1P1] != 0 {
		t.Fatal("batch replacement or phasing failed")
	}
	if first.Counters[core.P1P1] != 0 || second.Counters[core.P1P1] != 0 {
		t.Fatalf("clone changed original counters: %d, %d", first.Counters[core.P1P1], second.Counters[core.P1P1])
	}
}

func TestAddCountersEachDeclaration(t *testing.T) {
	good := NewSorcery("Good", "", WithAbility(Spell(AddCountersEach(ControlledPermanents(IsCreature), core.P1P1, Fixed(2)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if len(InspectDefinition(good.Abilities()[0]).Children) == 0 {
		t.Fatal("batch missing from inspection")
	}
	for _, effect := range []ResolvingEffect{
		AddCountersEach(nil, core.P1P1, Fixed(1)),
		AddCountersEach(ControlledPermanents(IsCreature), core.P1P1, nil),
		AddCountersEach(DeclarePermanentTargets("missing", 1, 2, IsCreature), core.P1P1, Fixed(1)),
		AddCountersEach(ControlledPermanents(IsCreature), core.NumCounters, Fixed(1)),
	} {
		bad := NewSorcery("Bad", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid counter batch accepted")
		}
	}
}

func TestAddCountersEachPreparesBeforePlacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 3), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 3), a.PlayerID())
	source := g.PutOnBattlefield(NewArtifact("Doubler", ""), a.PlayerID())
	g.AddCounterDoubler(source.ID(), core.P1P1, NewPermanentFilter("second while first has no counters", func(p *Permanent, game *Game) bool {
		return p.ID() == second.ID() && game.FindPermanent(first.ID()).Counters[core.P1P1] == 0
	}))
	if err := AddCountersEach(fixedTapSet{[]*Permanent{first, second}}, core.P1P1, Fixed(1)).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if first.Counters[core.P1P1] != 1 || second.Counters[core.P1P1] != 2 {
		t.Fatalf("placements observed partial state: %d, %d", first.Counters[core.P1P1], second.Counters[core.P1P1])
	}
}

func TestAddCountersEachStaleDuplicateAndTargetGroup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 2)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 2), b.PlayerID())
	if err := AddCountersEach(fixedTapSet{[]*Permanent{old, current, current, other}}, core.P1P1, Fixed(1)).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if current.Counters[core.P1P1] != 1 || other.Counters[core.P1P1] != 1 {
		t.Fatal("stale or duplicate member changed result")
	}
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	if err := AddCountersEach(targets, core.P1P1, Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if current.Counters[core.P1P1] != 1 {
		t.Fatal("returned target inherited old announcement")
	}
}

func TestAddCountersEachAmountReadOnce(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 3), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 3), a.PlayerID())
	calls := 0
	effect := AddCountersEach(fixedTapSet{[]*Permanent{first, second}}, core.P1P1, MinValue(Fixed(2), countedLifeAmount{&calls}))
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || first.Counters[core.P1P1] != 2 || second.Counters[core.P1P1] != 2 {
		t.Fatal("batch amount was not shared")
	}
	if err := AddCountersEach(fixedTapSet{}, core.P1P1, countedLifeAmount{&calls}).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("empty set evaluated amount")
	}
}
