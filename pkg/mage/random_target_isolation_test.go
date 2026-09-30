package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomTargetDeclarationIsolation(t *testing.T) {
	for _, path := range []string{"activation", "spell", "trigger", "prompt"} {
		t.Run(path, func(t *testing.T) {
			g, a, b := newRandomTargetGame()
			declaration := TargetRandom(TargetCreature())
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

func TestRandomTargetFailedPaymentKeepsDeclaration(t *testing.T) {
	g, a, b := newRandomTargetGame()
	declaration := TargetRandom(TargetNCreatures(2))
	addRandomTargetCreature(g, b, "First")
	addRandomTargetCreature(g, b, "Second")
	source := NewArtifact("Source", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), GenericCost(0), WithTarget(declaration), WithCost(Tap()), WithCost(LifePayCost(2)), WithCost(ManaCostPerTarget("{R}"))))
	source.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(source, a.PlayerID())
	a.ManaPool().Add(Red, 1)
	g.SetRandomResults([]int{0, 0})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("payment should fail")
	}
	if declaration.IsChosen() {
		t.Fatalf("failed announcement stored declaration choices: %v", declaration.Chosen())
	}
	if perm.Tapped || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.Stack().Peek() != nil {
		t.Fatal("failed payment changed live resources")
	}
}

func TestRandomTargetCloneCopyAndIncarnation(t *testing.T) {
	g, a, b := newRandomTargetGame()
	declaration := TargetRandom(TargetCreature())
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

func TestRandomTargetEmptyAndPhasedCandidatesKeepDeclaration(t *testing.T) {
	for _, randomCount := range []bool{false, true} {
		g, a, b := newRandomTargetGame()
		inner := TargetUpToNCreatures(3)
		declaration := TargetRandom(inner)
		if randomCount {
			declaration = TargetRandomCount(inner)
		}
		source := NewArtifact("Source", "{R}")
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		phased := addRandomTargetCreature(g, b, "Phased")
		phased.PhasedOut = true
		protected := addRandomTargetCreature(g, b, "Protected", WithAbility(ProtectionFromColor(Red)))
		prior := []uuid.UUID{protected.ID()}
		if err := declaration.Choose(a.PlayerID(), source, g, prior); err != nil {
			t.Fatal(err)
		}
		g.SetRandomResults([]int{1})
		chosen := g.chooseRandomTargets(a.PlayerID(), source, declaration.(*randomTarget), 0)
		if len(chosen) != 0 || g.RandIntn(2) != 1 {
			t.Fatal("empty legal pool selected a target or consumed randomness")
		}
		if !slices.Equal(declaration.Chosen(), prior) {
			t.Fatal("empty announcement cleared legacy choices")
		}
	}
}

func TestRandomTargetSamplingKeepsDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		spec      func() Target
		numbers   []int
		random    []int
		wantCount int
	}{
		{name: "fixed", spec: func() Target { return TargetRandom(TargetNCreatures(2)) }, random: []int{1, 0}, wantCount: 2},
		{name: "chosen count", spec: func() Target { return TargetRandom(TargetUpToNCreatures(5)) }, numbers: []int{2}, random: []int{1, 0}, wantCount: 2},
		{name: "chosen zero", spec: func() Target { return TargetRandom(TargetUpToNCreatures(5)) }, numbers: []int{0}},
		{name: "random count", spec: func() Target { return TargetRandomCount(TargetUpToNCreatures(5)) }, random: []int{2, 1, 0}, wantCount: 2},
		{name: "insufficient candidates", spec: func() Target { return TargetRandom(TargetNCreatures(3)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newRandomTargetGame()
			a.numbers = tc.numbers
			source := NewArtifact("Source", "")
			first := addRandomTargetCreature(g, b, "First")
			second := addRandomTargetCreature(g, b, "Second")
			declaration := tc.spec().(*randomTarget)
			g.SetRandomResults(append(slices.Clone(tc.random), 7))
			chosen := g.chooseRandomTargets(a.PlayerID(), source, declaration, 0)
			if len(chosen) != tc.wantCount || declaration.IsChosen() {
				t.Fatalf("choices = %v, declaration = %v", chosen, declaration.Chosen())
			}
			if tc.wantCount == 2 && !slices.Equal(chosen, []uuid.UUID{second.ID(), first.ID()}) {
				t.Fatal("sampling lost order or selected an identity twice")
			}
			if g.RandIntn(10) != 7 {
				t.Fatal("sampling consumed an unexpected random result")
			}
		})
	}
}
