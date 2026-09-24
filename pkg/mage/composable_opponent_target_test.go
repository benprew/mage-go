package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedOpponentTargetCandidatesAndActivation(t *testing.T) {
	for _, self := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
		source := NewArtifact("Drain", "", WithAbility(Activate(Costs(), Sequence(LoseLifeFor(target, Fixed(1)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		candidates := target.Possible(a.PlayerID(), source, g)
		if len(candidates) != 1 || candidates[0] != b.PlayerID() {
			t.Fatal("incorrect opponent candidates")
		}
		id := b.PlayerID()
		if self {
			id = a.PlayerID()
		}
		err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
		if self {
			if err == nil || g.StackSize() != 0 {
				t.Fatal("accepted controller target")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != 19 {
			t.Fatal("incorrect drain")
		}
	}
}

func TestTypedOpponentTargetCopy(t *testing.T) {
	for _, choice := range []string{"keep", "retarget", "invalid"} {
		t.Run(choice, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
			ability := Activate(Costs(), Sequence(LoseLifeFor(target, Fixed(1)), GainLifeFor(You, Fixed(1))), WithTargets(target))
			source := NewCreature("Drain", "", 1, 1, WithAbility(ability))
			original := &StackObject{ID: uuid.New(), SourceID: source.ID(), TargetSource: source, Controller: a.PlayerID(), IsAbility: true, Effects: ability.Effects(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets()}
			g.pushStack(original)
			id := a.PlayerID()
			if choice == "invalid" {
				id = b.PlayerID()
			}
			b.chooseQueue = [][]uuid.UUID{{id}}
			cp := g.CopyStackObjectDirect(original, b.PlayerID(), choice != "keep")
			want := b.PlayerID()
			if choice == "retarget" {
				want = a.PlayerID()
			}
			if cp.Targets[0] != want || original.Targets[0] != b.PlayerID() {
				t.Fatal("incorrect copy targets")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			wantA, wantB := 20, 20
			if choice == "retarget" {
				wantA, wantB = 19, 21
			}
			if clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("copy did not recheck opponent restriction")
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveTopOfStack()
			g.ResolveTopOfStack()
			if a.Life() != wantA+1 || b.Life() != wantB-1 {
				t.Fatal("original lost its target or controller")
			}
		})
	}
}

func TestTypedOpponentTargetTriggerController(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), Sequence(LoseLifeFor(target, Fixed(1)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.AddControlEffect(ControlEffectSpec{SourceID: victim.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() || g.StackPeek().Targets[0] != b.PlayerID() {
		t.Fatal("source control changed trigger controller or target")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatal("trigger used source controller at resolution")
	}
}
