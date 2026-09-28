package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntapResultTransitionAndStun(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Subject", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindNumber("untapped")
	body := Sequence(UntapPermanentWithResult(This, UntappedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	for _, tc := range []struct {
		tapped, stun bool
		want         int
	}{
		{true, false, 1}, {false, false, 0}, {true, true, 0}, {true, false, 1},
	} {
		p.Tapped = tc.tapped
		if tc.stun {
			p.AddCounter(core.Stun, 1)
		}
		before := a.Life()
		if err := body.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if got := g.resolution.numberBindings[result]; got != tc.want || a.Life() != before+2*tc.want {
			t.Fatalf("result=%d life=%d, want %d", got, a.Life(), tc.want)
		}
		if tc.stun && (p.Counters[core.Stun] != 0 || !p.Tapped) {
			t.Fatal("stun did not replace untap")
		}
	}
}

func TestUntapResultTargetAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	p.Tapped = true
	result := BindNumber("untapped")
	target := DeclarePermanentTarget("target", IsCreature)
	spell := NewInstant("Untap test", "", Spell(Sequence(
		UntapPermanentWithResult(target, UntappedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target)))
	a.AddToHand(spell)
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).AddCounter(core.Stun, 1)
	if g.FindPermanent(p.ID()).Counters[core.Stun] != 0 {
		t.Fatal("clone counter reached original")
	}
	clone.ResolveStack()
	g.ResolveStack()
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 || g.FindPermanent(p.ID()).Tapped || !clone.FindPermanent(p.ID()).Tapped {
		t.Fatalf("target untap outcome: original life=%d clone life=%d original tapped=%v clone tapped=%v", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), g.FindPermanent(p.ID()).Tapped, clone.FindPermanent(p.ID()).Tapped)
	}
}

func TestUntapResultAbsentPhasedAndReturnedTarget(t *testing.T) {
	for _, state := range []string{"absent", "phased", "returned", "present"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := NewCreature("Victim", "", 2, 2)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			p.Tapped = true
			result := BindNumber("untapped")
			target := DeclarePermanentTarget("target", IsCreature)
			spell := NewInstant("Untap", "", Spell(Sequence(
				UntapPermanentWithResult(target, UntappedThisWayInto(result)),
				IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
			), WithTargets(target)))
			a.AddToHand(spell)
			g.SetStep(core.PrecombatMain)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(victim, b.PlayerID()).Tapped = true
			}
			g.ResolveStack()
			want := 20
			if state == "present" {
				want = 22
			}
			if a.Life() != want {
				t.Fatalf("life=%d, want %d", a.Life(), want)
			}
		})
	}
}

func TestUntapResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	result := BindNumber("untapped")
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Sequence(
		UntapPermanentWithResult(target, UntappedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target))
	spell := NewInstant("Untap copy", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	p.Tapped = true
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(p.ID()).Tapped {
		t.Fatalf("life=%d tapped=%v", a.Life(), g.FindPermanent(p.ID()).Tapped)
	}
}

func TestUntapResultDeclaration(t *testing.T) {
	b := BindNumber("untapped")
	good := NewArtifact("Good", "", WithAbility(Activate(Costs(), Sequence(UntapPermanentWithResult(This, UntappedThisWayInto(b)), GainLifeFor(You, FromNumber(b))))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{UntapPermanentWithResult(This), "binding"},
		{UntapPermanentWithResult(nil, UntappedThisWayInto(b)), "subject"},
		{UntapPermanentWithResult(This, UntappedThisWayInto(nil)), "binding"},
		{UntapPermanentWithResult(This, UntappedThisWayInto(BindNumber(""))), "name"},
		{UntapPermanentWithResult(This, UntappedThisWayInto(b), UntappedThisWayInto(b)), "once"},
		{Sequence(GainLifeFor(You, FromNumber(b)), UntapPermanentWithResult(This, UntappedThisWayInto(b))), "producer"},
	} {
		bad := NewArtifact("Bad", "", WithAbility(Activate(Costs(), tc.effect)))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("validation=%v, want %q", err, tc.want)
		}
	}
}
