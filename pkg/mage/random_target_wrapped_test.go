package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestWrappedRandomTargetDeclarationIsolation(t *testing.T) {
	for _, path := range []string{"activation", "spell", "trigger", "prompt"} {
		t.Run(path, func(t *testing.T) {
			g, a, b := newRandomTargetGame()
			declaration := &targetDeclaration{name: "random", spec: TargetRandom(TargetCreature())}
			first := addRandomTargetCreature(g, b, "First")
			second := addRandomTargetCreature(g, b, "Second")
			g.SetRandomResults([]int{0, 1})
			var source Card
			switch path {
			case "activation":
				source = NewArtifact("Source", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), GenericCost(0), WithTarget(declaration)))
			case "spell":
				source = NewInstant("Source", "", NewSpell(GainLifeFor(You, Fixed(1)), WithTarget(declaration)))
			case "trigger":
				source = NewArtifact("Source", "", WithAbility(NewTriggered(EvtUpkeep, false, GainLifeFor(You, Fixed(1))).AddTarget(declaration)))
			case "prompt":
				source = NewArtifact("Source", "")
			}
			source.SetOwner(a.PlayerID())
			if path != "spell" {
				g.PutOnBattlefield(source, a.PlayerID())
			}
			var objects []*StackObject
			for i, want := range []uuid.UUID{first.ID(), second.ID()} {
				switch path {
				case "activation":
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				case "spell":
					card := source.Copy()
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
						t.Fatal(err)
					}
				case "trigger":
					g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				case "prompt":
					got := g.promptTargetsForList(a.PlayerID(), source, []Target{declaration})
					if !slices.Equal(got, []uuid.UUID{want}) {
						t.Fatalf("prompt %d: %v", i, got)
					}
				}
				if path != "prompt" {
					obj := g.Stack().Peek()
					if obj == nil || !slices.Equal(obj.Targets, []uuid.UUID{want}) {
						t.Fatalf("announcement %d: wrong targets", i)
					}
					objects = append(objects, obj)
				}
				if declaration.IsChosen() || len(declaration.Chosen()) != 0 {
					t.Fatalf("announcement %d stored choices in the declaration: %v", i, declaration.Chosen())
				}
			}
			if len(objects) > 0 && !slices.Equal(objects[0].Targets, []uuid.UUID{first.ID()}) {
				t.Fatal("second announcement changed the first instance")
			}
		})
	}
}

func TestWrappedRandomTargetFailedPaymentKeepsDeclaration(t *testing.T) {
	g, a, b := newRandomTargetGame()
	declaration := &targetDeclaration{name: "random", spec: TargetRandom(TargetNCreatures(2))}
	addRandomTargetCreature(g, b, "First")
	addRandomTargetCreature(g, b, "Second")
	source := NewArtifact("Source", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), GenericCost(0), WithTarget(declaration), WithCost(Tap()), WithCost(LifePayCost(2)), WithCost(ManaCostPerTarget("{R}"))))
	source.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(source, a.PlayerID())
	a.ManaPool().Add(Red, 1)
	g.SetRandomResults([]int{0, 0, 7})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("payment should fail")
	}
	if g.RandIntn(10) != 7 {
		t.Fatal("payment failure occurred before target sampling")
	}
	if declaration.IsChosen() {
		t.Fatalf("failed announcement stored declaration choices: %v", declaration.Chosen())
	}
	if perm.Tapped || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.Stack().Peek() != nil {
		t.Fatal("failed payment changed live resources")
	}
}

func TestWrappedRandomTargetCloneCopyAndIncarnation(t *testing.T) {
	g, a, b := newRandomTargetGame()
	declaration := &targetDeclaration{name: "random", spec: TargetRandom(TargetCreature())}
	source := NewArtifact("Source", "", WithActivatedAbility(GainLifeFor(You, Fixed(2)), GenericCost(0), WithTarget(declaration)))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	first := addRandomTargetCreature(g, b, "First")
	second := addRandomTargetCreature(g, b, "Second")
	g.SetRandomResults([]int{0, 1})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	original := g.Stack().Peek()
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(clone.Stack().Peek().Targets, []uuid.UUID{second.ID()}) || !slices.Equal(original.Targets, []uuid.UUID{first.ID()}) {
		t.Fatal("clone announcement changed original targets")
	}
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), false)
	if !slices.Equal(copied.Targets, original.Targets) {
		t.Fatal("copy changed random targets")
	}
	if got := g.RandIntn(2); got != 1 {
		t.Fatalf("copy or clone consumed original randomness: %d", got)
	}
	if declaration.IsChosen() {
		t.Fatal("clone wrote shared declaration choices")
	}
	g.DestroyPermanent(first)
	g.PutOnBattlefield(first.Card, b.PlayerID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("returned target made an old instance legal")
	}
	clone.ResolveTopOfStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("clone resolution was not isolated")
	}
}

func TestWrappedRandomTargetCountAndOrdinaryTargets(t *testing.T) {
	for _, tc := range []struct {
		name      string
		spec      Target
		x         int
		numbers   []int
		random    []int
		wantCount int
	}{
		{name: "fixed", spec: TargetRandom(TargetNCreatures(2)), random: []int{1, 0}, wantCount: 2},
		{name: "chosen count", spec: TargetRandom(TargetUpToNCreatures(4)), numbers: []int{2}, random: []int{1, 0}, wantCount: 2},
		{name: "chosen zero", spec: TargetRandom(TargetUpToNCreatures(4)), numbers: []int{0}},
		{name: "random count", spec: TargetRandomCount(TargetUpToNCreatures(4)), random: []int{2, 1, 0}, wantCount: 2},
		{name: "X", spec: TargetRandomCount(TargetOneToXCreatures()), x: 2, random: []int{1, 1, 0}, wantCount: 2},
		{name: "zero X", spec: TargetRandomCount(TargetOneToXCreatures())},
		{name: "insufficient", spec: TargetRandom(TargetNCreatures(3))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newRandomTargetGame()
			a.numbers = tc.numbers
			source := NewArtifact("Source", "{R}")
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			first := addRandomTargetCreature(g, b, "First")
			second := addRandomTargetCreature(g, b, "Second")
			phased := addRandomTargetCreature(g, b, "Phased")
			phased.PhasedOut = true
			addRandomTargetCreature(g, b, "Protected", WithAbility(ProtectionFromColor(Red)))
			declaration := &targetDeclaration{name: "random", spec: tc.spec}
			ordinary := DeclarePlayerTarget("player")
			g.SetRandomResults(append(slices.Clone(tc.random), 7))
			got := g.acquireRandomTargets(a.PlayerID(), source, []Target{ordinary.targetSpec(), declaration, TargetPlayer()}, []uuid.UUID{b.PlayerID(), a.PlayerID()}, tc.x)
			want := []uuid.UUID{b.PlayerID()}
			if tc.wantCount == 2 {
				want = append(want, second.ID(), first.ID())
			}
			want = append(want, a.PlayerID())
			if !slices.Equal(got, want) || g.RandIntn(10) != 7 || declaration.IsChosen() {
				t.Fatalf("announcement = %v, want %v with no extra random consumption or stored choices", got, want)
			}
		})
	}
}

func TestWrappedRandomTargetTypedResolution(t *testing.T) {
	g, a, b := newRandomTargetGame()
	binding := &PermanentTargetBinding{&targetDeclaration{name: "random", spec: TargetRandom(TargetCreature())}}
	source := NewArtifact("Typed source", "", WithAbility(Activate(Costs(), TapSubject(binding), WithTargets(binding))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	first := addRandomTargetCreature(g, b, "First")
	second := addRandomTargetCreature(g, b, "Second")
	g.SetRandomResults([]int{1})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if first.Tapped || !second.Tapped {
		t.Fatal("typed effect did not read the sampled instance target")
	}
}
