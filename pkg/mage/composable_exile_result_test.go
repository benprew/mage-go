package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileResultIncarnationAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Subject", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindNumber("exiled")
	body := Sequence(ExileWithResult(This, ExiledThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	clone := g.Clone()
	for _, tc := range []struct {
		name string
		game *Game
		move func(*Game)
		want int
	}{
		{"present", clone, nil, 1},
		{"returned", g, func(game *Game) {
			game.RemoveFromBattlefield(game.FindPermanent(p.ID()))
			game.PutOnBattlefield(card, a.PlayerID())
		}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.move != nil {
				tc.move(tc.game)
			}
			if err := body.Apply(&EffectContext{Game: tc.game, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
				t.Fatal(err)
			}
			if got := tc.game.resolution.numberBindings[result]; got != tc.want {
				t.Fatalf("result = %d, want %d", got, tc.want)
			}
			if got := tc.game.GetPlayer(a.PlayerID()).Life(); got != 20+2*tc.want {
				t.Fatalf("life = %d", got)
			}
		})
	}
}

func TestExileResultDeclaration(t *testing.T) {
	b := BindNumber("exiled")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(At(YourUpkeep, Sequence(ExileWithResult(This, ExiledThisWayInto(b)), GainLifeFor(You, FromNumber(b))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		effect ResolvingEffect
		want   string
	}{
		{ExileWithResult(This, ExiledThisWayInto(nil)), "binding"},
		{ExileWithResult(This, ExiledThisWayInto(BindNumber(""))), "name"},
		{ExileWithResult(This, ExiledThisWayInto(b), ExiledThisWayInto(b)), "once"},
		{Sequence(GainLifeFor(You, FromNumber(b)), ExileWithResult(This, ExiledThisWayInto(b))), "producer"},
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, tc.effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestExileResultAbsentPhasedAndTarget(t *testing.T) {
	for _, state := range []string{"absent", "phased", "returned", "present"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := NewCreature("Victim", "", 2, 2)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			result := BindNumber("exiled")
			target := DeclarePermanentTarget("target", IsCreature)
			spell := NewInstant("Exile Test", "", Spell(Sequence(ExileWithResult(target, ExiledThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2)))), WithTargets(target)))
			if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
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
				g.PutOnBattlefield(victim, b.PlayerID())
			}
			g.ResolveStack()
			want := 0
			if state == "present" {
				want = 1
			}
			if a.Life() != 20+2*want {
				t.Fatalf("life = %d, want %d", a.Life(), 20+2*want)
			}
			if state == "present" && g.FindPermanent(p.ID()) != nil {
				t.Fatal("subject remains on battlefield")
			}
		})
	}
}

func TestExileResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	result := BindNumber("exiled")
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Sequence(
		ExileWithResult(target, ExiledThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target))
	spell := NewInstant("Exile Copy", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopySpellOnStack(spell.ID(), a.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("spell copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(p.ID()) != nil {
		t.Fatalf("copy and original did not keep independent outcomes: life = %d", a.Life())
	}
}
