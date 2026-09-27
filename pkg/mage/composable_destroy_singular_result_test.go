package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestroySingularResult(t *testing.T) {
	for _, state := range []string{"destroyed", "regenerated", "indestructible", "phased", "absent", "returned"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewCreature("Subject", "", 2, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			result := BindNumber("destroyed")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			g.resolution.numberBindings = map[*NumberBinding]int{result: 1}
			switch state {
			case "regenerated":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
			case "phased":
				g.PhaseOut(p.ID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			}
			body := Sequence(DestroyWithResult(This, DestroyedThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(3))))
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "destroyed" {
				want = 1
			}
			if got := g.resolution.numberBindings[result]; got != want || a.Life() != 20+3*want {
				t.Fatalf("result = %d, life = %d; want %d, %d", got, a.Life(), want, 20+3*want)
			}
		})
	}
}

func TestDestroySingularResultDefinition(t *testing.T) {
	result := BindNumber("destroyed")
	valid := NewCreature("Valid", "", 2, 2, WithAbility(At(YourUpkeep, Sequence(DestroyWithResult(This, DestroyedThisWayInto(result)), IfPositive(FromNumber(result), DealDamageFrom(This, You, Fixed(7)))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{DestroyWithResult(This, DestroyedThisWayInto(nil)), "binding"},
		{DestroyWithResult(This, DestroyedThisWayInto(BindNumber(""))), "name"},
		{newDestroySubjectEffect(This, false, []DestroyResultOption{DestroyedThisWayInto(result), DestroyedThisWayInto(result)}), "once"},
		{IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1))), "producer"},
		{Sequence(IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1))), DestroyWithResult(This, DestroyedThisWayInto(result))), "producer"},
	} {
		card := NewCreature("Invalid", "", 2, 2, WithAbility(At(YourUpkeep, tc.body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("validation = %v; want %q", err, tc.want)
		}
	}
}

func TestDestroySingularResultWithoutRegenerationAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Subject", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.AddRegenerationShield(p.ID())
	result := BindNumber("destroyed")
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		body := Sequence(DestroyWithoutRegenerationWithResult(This, DestroyedThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
		if err := body.Apply(&EffectContext{Game: game, SourceID: p.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if game.resolution.numberBindings[result] != 1 || game.GetPlayer(a.PlayerID()).Life() != 22 || game.FindPermanent(p.ID()) != nil {
			t.Fatal("action did not destroy through the regeneration shield")
		}
		if game == clone && (g.FindPermanent(p.ID()) == nil || a.Life() != 20) {
			t.Fatal("clone changed the original game")
		}
	}
}

func TestDestroySingularResultTargetLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("destroyed")
	target := DeclarePermanentTarget("subject", IsCreature)
	spell := NewInstant("Destroy Test", "", Spell(Sequence(
		DestroyWithResult(target, DestroyedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	a.AddToHand(spell)
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.PutOnBattlefield(victim, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || g.FindPermanent(p.ID()) == nil {
		t.Fatal("an illegal target caused destruction or a stale result")
	}
}

func TestDestroySingularResultSourceDamageAfterDeparture(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Source", "", 7, 7)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindNumber("destroyed")
	body := Sequence(
		DestroyWithResult(This, DestroyedThisWayInto(result)),
		IfPositive(FromNumber(result), DealDamageFrom(This, You, Fixed(7))),
	)
	if err := body.Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 13 || g.FindPermanent(p.ID()) != nil {
		t.Fatalf("life = %d; source still present = %t", a.Life(), g.FindPermanent(p.ID()) != nil)
	}
}
