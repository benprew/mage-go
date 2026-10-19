package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type partitionPlayer struct {
	Player
	answers []int
	reasons []string
	during  func()
}

func (p *partitionPlayer) ChooseMode(_ []string, reason string) int {
	p.reasons = append(p.reasons, reason)
	if p.during != nil {
		f := p.during
		p.during = nil
		f()
	}
	if len(p.answers) == 0 {
		return 0
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer
}

func TestNamedPermanentPartition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &partitionPlayer{Player: b, answers: []int{0, 1, 1}}
	g := NewGame(a, chooser)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 4), b.PlayerID())
	third := g.PutOnBattlefield(NewCreature("Third", "", 1, 4), b.PlayerID())
	left, right := BindPermanentSet("left"), BindPermanentSet("right")
	effect := SeparatePermanentsIntoNamedPiles(You, UnionPermanents(MatchingPermanents(IsCreature), MatchingPermanents(IsCreature)), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(chooser.reasons) != 3 {
		t.Fatal("each exact visit must occur once", chooser.reasons)
	}
	if !strings.Contains(chooser.reasons[1], "Left") || !strings.Contains(chooser.reasons[1], "First") {
		t.Fatal("prior public choices missing", chooser.reasons)
	}
	gotLeft, gotRight := FromPermanentSet(left).resolvePermanents(ctx), FromPermanentSet(right).resolvePermanents(ctx)
	if len(gotLeft) != 1 || gotLeft[0].ID() != first.ID() || len(gotRight) != 2 {
		t.Fatal("incorrect partition", gotLeft, gotRight)
	}
	clone := g.Clone()
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 4), b.PlayerID())
	g.PhaseOut(first.ID())
	if len(FromPermanentSet(left).resolvePermanents(ctx)) != 0 {
		t.Fatal("phased member selected")
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	if len(FromPermanentSet(left).resolvePermanents(ctx)) != 1 {
		t.Fatal("phase cycle lost membership")
	}
	g.RemoveFromBattlefield(second)
	returned := g.PutOnBattlefield(second.Card, b.PlayerID())
	if err := TapEach(FromPermanentSet(right)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(third.ID()).Tapped || returned.Tapped || late.Tapped {
		t.Fatal("membership followed a new visit")
	}
	empty := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsLand), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
	if err := empty.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPermanentSet(left).resolvePermanents(ctx))+len(FromPermanentSet(right).resolvePermanents(ctx)) != 0 {
		t.Fatal("empty result retained old members")
	}
	if len(FromPermanentSet(right).resolvePermanents(&EffectContext{Game: clone})) != 2 {
		t.Fatal("clone shared result writes")
	}
}

func TestNamedPermanentPartitionSnapshotAndInvalidChoice(t *testing.T) {
	for _, answer := range []int{-1, 9} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		chooser := &partitionPlayer{Player: a, answers: []int{answer}}
		g := NewGame(chooser, b)
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 4), a.PlayerID())
		left, right := BindPermanentSet("left"), BindPermanentSet("right")
		chooser.during = func() {
			g.RemoveFromBattlefield(second)
			g.PutOnBattlefield(NewCreature("Late", "", 1, 4), a.PlayerID())
		}
		effect := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		members := FromPermanentSet(left).resolvePermanents(ctx)
		if len(chooser.reasons) != 1 || len(members) != 1 || members[0].ID() != first.ID() || len(FromPermanentSet(right).resolvePermanents(ctx)) != 0 {
			t.Fatalf("snapshot or required fallback failed: choices=%v members=%v right=%v", chooser.reasons, members, FromPermanentSet(right).resolvePermanents(ctx))
		}
	}
}

func TestNamedPermanentPartitionDefinition(t *testing.T) {
	left, right := BindPermanentSet("left"), BindPermanentSet("right")
	for _, piles := range [][]NamedPermanentPile{
		nil, {{"Left", left}}, {{"", left}, {"Right", right}}, {{"Left", left}, {"Left", right}}, {{"Left", left}, {"Right", left}}, {{"Left", nil}, {"Right", right}},
	} {
		effect := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), piles...)
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted", piles)
		}
	}
	effect := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right})
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(effect, TapEach(FromPermanentSet(left)), TapEach(FromPermanentSet(right))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestNamedPermanentPartitionRepartition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &partitionPlayer{Player: a, answers: []int{0, 0, 1, 1}}
	g := NewGame(chooser, b)
	g.PutOnBattlefield(NewCreature("First", "", 1, 4), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 1, 4), a.PlayerID())
	left, right := BindPermanentSet("left"), BindPermanentSet("right")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	piles := []NamedPermanentPile{{"Left", left}, {"Right", right}}
	if err := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), piles...).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := SeparatePermanentsIntoNamedPiles(You, FromPermanentSet(left), piles...).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPermanentSet(left).resolvePermanents(ctx)) != 0 || len(FromPermanentSet(right).resolvePermanents(ctx)) != 2 {
		t.Fatal("outputs cleared before inputs were read")
	}
}

func TestNamedPermanentPartitionResolutionFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &partitionPlayer{Player: a, answers: []int{0, 1, 1, 0}}
	g := NewGame(chooser, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 4), a.PlayerID())
	left, right := BindPermanentSet("left"), BindPermanentSet("right")
	body := Sequence(SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), NamedPermanentPile{"Left", left}, NamedPermanentPile{"Right", right}), AtNext(EachEndStep, TapEach(FromPermanentSet(left))))
	card := NewArtifact("Partition source", "", WithAbility(Activate(Costs(), body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if len(chooser.reasons) != 4 {
		t.Fatal("wrong resolution choice count")
	}
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
			t.Fatal("separate resolution frames lost their own piles")
		}
	}
}
