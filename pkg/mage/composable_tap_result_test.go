package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestTapResultTransitionAndBranch(t *testing.T) {
	for _, alreadyTapped := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		p := NewCreature("Subject", "", 2, 2)
		g.PutOnBattlefield(p, a.PlayerID())
		if alreadyTapped {
			g.TapPermanent(g.FindPermanent(p.ID()))
		}
		result := BindNumber("tap result")
		body := Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(3))))
		if err := body.Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := 23
		if alreadyTapped {
			want = 20
		}
		wantResult := 1
		if alreadyTapped {
			wantResult = 0
		}
		if a.Life() != want || g.resolution.numberBindings[result] != wantResult {
			t.Fatalf("already tapped %v: life %d, result %d", alreadyTapped, a.Life(), g.resolution.numberBindings[result])
		}
	}
}

func TestTapResultAbsentPhasedAndReturned(t *testing.T) {
	for _, state := range []string{"absent", "phased", "returned"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := NewCreature("Subject", "", 2, 2)
			g.PutOnBattlefield(p, a.PlayerID())
			result := BindNumber("tap result")
			g.resolution.numberBindings = map[*NumberBinding]int{result: 1}
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: g.FindPermanent(p.ID()).incarnationID}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(p, a.PlayerID())
			}
			body := Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(3))))
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || g.resolution.numberBindings[result] != 0 {
				t.Fatal("absent incarnation reused a tap result")
			}
		})
	}
}

func TestTapResultValidationAndClone(t *testing.T) {
	result := BindNumber("tap result")
	inner := BindNumber("inner")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2)))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"nil binding", TapSubject(This, TapAmountInto(nil)), "binding"},
		{"duplicate", TapSubject(This, TapAmountInto(result), TapAmountInto(result)), "once"},
		{"missing producer", IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1))), "producer"},
		{"nil value", IfPositive(nil, GainLifeFor(You, Fixed(1))), "non-nil"},
		{"nil body", IfPositive(Fixed(1), nil), "non-nil"},
		{"branch escape", Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), SnapshotNumber(Fixed(3), inner)), GainLifeFor(You, FromNumber(inner))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), tc.body)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation = %v", err)
			}
		})
	}

	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(valid, a.PlayerID())
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), valid.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 || g.FindPermanent(valid.ID()).Tapped {
		t.Fatal("clone shared tap state or result")
	}
}

func TestTapResultTargetedSpellAndStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	result := BindNumber("tap result")
	body := Sequence(TapSubject(target, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	ability := Spell(body, WithTargets(target))
	spell := NewInstant("Synthetic tap", "", WithAbility(ability))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if copied == nil {
		t.Fatal("stack copy failed")
	}
	if len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
		t.Fatalf("copy targets = %v", copied.Targets)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("copied or cloned result used the wrong controller: clone A=%d B=%d, original A=%d B=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life(), b.Life())
	}
	if !clone.FindPermanent(first.ID()).Tapped || !clone.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copied or cloned target changed the original game")
	}

	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("returned target used a stale tap result")
	}
}
