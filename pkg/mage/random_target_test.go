package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type randomTargetPlayer struct {
	*BasePlayer
	numbers []int
}

func (p *randomTargetPlayer) ChooseNumber(minimum, maximum int, _ string) int {
	if len(p.numbers) == 0 {
		return maximum
	}
	n := p.numbers[0]
	p.numbers = p.numbers[1:]
	return min(max(n, minimum), maximum)
}

func newRandomTargetGame() (*Game, *randomTargetPlayer, *randomTargetPlayer) {
	a := &randomTargetPlayer{BasePlayer: NewBasePlayer("A")}
	b := &randomTargetPlayer{BasePlayer: NewBasePlayer("B")}
	return NewGame(a, b), a, b
}

func addRandomTargetCreature(g *Game, owner Player, name string, opts ...CardOption) *Permanent {
	card := NewCreature(name, "{1}", 1, 1, opts...)
	card.SetOwner(owner.PlayerID())
	return g.PutOnBattlefield(card, owner.PlayerID())
}

func TestTargetRandomActivatedAbilityChoosesDistinctLegalTargetsBeforeStack(t *testing.T) {
	g, a, b := newRandomTargetGame()
	source := NewArtifact("Random Source", "{R}", WithActivatedAbility(
		FuncEffect("record random targets", EffectProperties{}, func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil }),
		GenericCost(0),
		WithTarget(TargetRandom(TargetNCreatures(2))),
	))
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())
	first := addRandomTargetCreature(g, b, "First")
	second := addRandomTargetCreature(g, b, "Second")
	third := addRandomTargetCreature(g, b, "Third")
	addRandomTargetCreature(g, b, "Protected", WithAbility(ProtectionFromColor(Red)))
	g.SetRandomResults([]int{2, 0})

	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, nil); err != nil {
		t.Fatalf("activate random-target ability: %v", err)
	}
	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("expected ability on stack")
	}
	if got, want := obj.Targets, []uuid.UUID{third.ID(), first.ID()}; !slices.Equal(got, want) {
		t.Fatalf("random targets = %v, want %v (second candidate was %v)", got, want, second.ID())
	}
	if len(obj.TargetSpecs) != 2 {
		t.Fatalf("stack target specs = %d, want 2", len(obj.TargetSpecs))
	}
}

func TestTargetRandomVariableCountAsksOnlyForCount(t *testing.T) {
	g, a, b := newRandomTargetGame()
	a.numbers = []int{2}
	source := NewArtifact("Random Any Number", "{0}", WithActivatedAbility(
		GainLife(1), GenericCost(0),
		WithTarget(TargetRandom(TargetUpToNCreatures(4))),
	))
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())
	first := addRandomTargetCreature(g, b, "First")
	addRandomTargetCreature(g, b, "Second")
	third := addRandomTargetCreature(g, b, "Third")
	g.SetRandomResults([]int{2, 0})

	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, nil); err != nil {
		t.Fatalf("activate random-target ability: %v", err)
	}
	if got, want := g.Stack().Peek().Targets, []uuid.UUID{third.ID(), first.ID()}; !slices.Equal(got, want) {
		t.Fatalf("random targets = %v, want %v", got, want)
	}
}

func TestTargetRandomCountUsesOneToXBoundsForSpell(t *testing.T) {
	for _, tc := range []struct {
		name        string
		x           int
		random      []int
		wantTargets int
	}{
		{name: "X is zero", x: 0},
		{name: "X is positive", x: 3, random: []int{1, 2, 0}, wantTargets: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newRandomTargetGame()
			spell := NewInstant("Random X Targets", "{X}", NewTargetedSpell(
				TargetRandomCount(TargetOneToXCreatures()), GainLife(1),
			))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Colorless, tc.x)
			first := addRandomTargetCreature(g, b, "First")
			addRandomTargetCreature(g, b, "Second")
			third := addRandomTargetCreature(g, b, "Third")
			g.SetRandomResults(tc.random)

			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, tc.x); err != nil {
				t.Fatalf("cast random X-target spell: %v", err)
			}
			if got := len(g.Stack().Peek().Targets); got != tc.wantTargets {
				t.Fatalf("random target count = %d, want %d", got, tc.wantTargets)
			}
			if tc.wantTargets > 0 {
				if got, want := g.Stack().Peek().Targets, []uuid.UUID{third.ID(), first.ID()}; !slices.Equal(got, want) {
					t.Fatalf("random targets = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestTargetRandomTriggeredAbilityChoosesBeforeStack(t *testing.T) {
	g, a, b := newRandomTargetGame()
	source := NewArtifact("Random Trigger", "{0}", WithAbility(
		NewTriggered(EvtUpkeep, false, GainLife(1)).AddTarget(TargetRandom(TargetCreature())),
	))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	first := addRandomTargetCreature(g, b, "First")
	second := addRandomTargetCreature(g, b, "Second")
	g.SetRandomResults([]int{1})

	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	obj := g.Stack().Peek()
	if obj == nil {
		t.Fatal("expected trigger on stack")
	}
	if got, want := obj.Targets, []uuid.UUID{second.ID()}; !slices.Equal(got, want) {
		t.Fatalf("random trigger target = %v, want %v (first was %v)", got, want, first.ID())
	}
	if len(obj.TargetSpecs) != 1 {
		t.Fatalf("stack target specs = %d, want 1", len(obj.TargetSpecs))
	}
}

func TestRandomTargetUsesNormalFizzleRules(t *testing.T) {
	g, a, b := newRandomTargetGame()
	source := NewArtifact("Random Fizzle", "{0}", WithActivatedAbility(
		GainLife(3), GenericCost(0), WithTarget(TargetRandom(TargetCreature())),
	))
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())
	target := addRandomTargetCreature(g, b, "Only Target")
	g.SetRandomResults([]int{0})

	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, nil); err != nil {
		t.Fatalf("activate random-target ability: %v", err)
	}
	g.DestroyPermanent(target)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("life after all targets became illegal = %d, want 20", got)
	}
}

func TestManaCostPerTargetAutotapsCombinedCost(t *testing.T) {
	g, a, b := newRandomTargetGame()
	a.numbers = []int{2}
	source := NewArtifact("Per Target Cost", "{0}", WithActivatedAbility(
		GainLife(1), GenericCost(2),
		WithCost(Tap()),
		WithCost(ManaCostPerTarget("{R}")),
		WithTarget(TargetRandom(TargetUpToNCreatures(4))),
	))
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())
	for i, color := range []Color{Blue, Blue, Red, Red} {
		land := NewLand(string(rune('A'+i)), WithManaAbility(color))
		land.SetOwner(a.PlayerID())
		g.PutOnBattlefield(land, a.PlayerID())
	}
	addRandomTargetCreature(g, b, "First")
	addRandomTargetCreature(g, b, "Second")
	g.SetRandomResults([]int{0, 0})

	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, nil); err != nil {
		t.Fatalf("activate per-target-cost ability: %v", err)
	}
	if !g.FindPermanent(sourcePerm.ID()).Tapped {
		t.Fatal("source tap cost was not paid")
	}
	for _, perm := range g.AllBattlefield() {
		if perm.HasType(TypeLand) && !perm.Tapped {
			t.Fatalf("mana source %s was not tapped", perm.Name())
		}
	}
	if got := len(g.Stack().Peek().Targets); got != 2 {
		t.Fatalf("target count = %d, want 2", got)
	}
}

func TestManaCostPerTargetFailureIsAtomic(t *testing.T) {
	g, a, b := newRandomTargetGame()
	a.numbers = []int{2}
	source := NewArtifact("Per Target Cost", "{0}", WithActivatedAbility(
		GainLife(1), GenericCost(2),
		WithCost(Tap()),
		WithCost(ManaCostPerTarget("{R}")),
		WithTarget(TargetRandom(TargetUpToNCreatures(4))),
	))
	source.SetOwner(a.PlayerID())
	sourcePerm := g.PutOnBattlefield(source, a.PlayerID())
	lands := make([]*Permanent, 0, 3)
	for i, color := range []Color{Blue, Blue, Red} {
		land := NewLand(string(rune('A'+i)), WithManaAbility(color))
		land.SetOwner(a.PlayerID())
		lands = append(lands, g.PutOnBattlefield(land, a.PlayerID()))
	}
	addRandomTargetCreature(g, b, "First")
	addRandomTargetCreature(g, b, "Second")
	g.SetRandomResults([]int{0, 0})

	if err := g.ActivateAbilityByIndex(a.PlayerID(), sourcePerm.ID(), 0, nil); err == nil {
		t.Fatal("activation succeeded without enough mana")
	}
	if g.FindPermanent(sourcePerm.ID()).Tapped {
		t.Fatal("failed activation paid source tap cost")
	}
	for _, land := range lands {
		if g.FindPermanent(land.ID()).Tapped {
			t.Fatalf("failed activation tapped mana source %s", land.Name())
		}
	}
	if g.Stack().Peek() != nil {
		t.Fatal("failed activation created a stack object")
	}
}

func TestManaCostPerTargetSpellUsesRandomTargetsBeforePayment(t *testing.T) {
	for _, tc := range []struct {
		name    string
		redMana int
		wantErr bool
	}{
		{name: "payable", redMana: 2},
		{name: "atomic failure", redMana: 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newRandomTargetGame()
			spell := NewInstant("Per Target Spell", "{0}", NewSpell(
				GainLife(1),
				WithTarget(TargetRandom(TargetNCreatures(2))),
				WithCost(ManaCostPerTarget("{R}")),
			))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, tc.redMana)
			addRandomTargetCreature(g, b, "First")
			addRandomTargetCreature(g, b, "Second")
			g.SetRandomResults([]int{0, 0})

			err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("cast succeeded without enough per-target mana")
				}
				if got := a.ManaPool().Count(Red); got != tc.redMana {
					t.Fatalf("red mana after failed cast = %d, want %d", got, tc.redMana)
				}
				if len(a.Hand()) != 1 || g.Stack().Peek() != nil {
					t.Fatal("failed cast moved the card or created a stack object")
				}
				return
			}
			if err != nil {
				t.Fatalf("cast per-target-cost spell: %v", err)
			}
			if got := a.ManaPool().Count(Red); got != 0 {
				t.Fatalf("red mana after cast = %d, want 0", got)
			}
			if got := len(g.Stack().Peek().Targets); got != 2 {
				t.Fatalf("target count = %d, want 2", got)
			}
		})
	}
}
