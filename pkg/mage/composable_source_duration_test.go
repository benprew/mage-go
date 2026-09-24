package mage

import (
	"slices"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableSourceDurationLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	host := NewCreature("Host", "{R}", 2, 2)
	source.SetOwner(a.PlayerID())
	host.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	g.MutablePermanent(source.ID()).AttachedTo = host.ID()
	effect := UntilSourceLeaves(This,
		ModifyPT(AttachedPermanent, Fixed(2), Fixed(3)),
		SetColors(AttachedPermanent, Blue),
		GrantKeywordModification(AttachedPermanent, Flying),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check := func(game *Game, active bool) {
		t.Helper()
		p := game.FindPermanent(host.ID())
		wantPower, wantToughness := 2, 2
		if active {
			wantPower, wantToughness = 4, 5
		}
		if p.CurrentPower(game) != wantPower || p.CurrentToughness(game) != wantToughness || p.HasAttr(Flying) != active || slices.Contains(game.EffectiveColors(host.ID()), Blue) != active {
			t.Fatalf("active %v: power %d, toughness %d, flying %v, blue %v", active, p.CurrentPower(game), p.CurrentToughness(game), p.HasAttr(Flying), slices.Contains(game.EffectiveColors(host.ID()), Blue))
		}
	}
	check(g, true)
	phased := g.PhaseOut(source.ID())
	g.effects.RemoveEndOfCombat()
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(g, true)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, true)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	clone.PutOnBattlefield(source, a.PlayerID())
	clone.ApplyContinuousEffects()
	check(clone, false)
	check(g, true)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ApplyContinuousEffects()
	check(g, false)
}

func TestComposableSourceDurationTriggerContext(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		body := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(2), Fixed(0)))
		source := NewCreature("Source", "", 1, 1, WithAbility(At(YourUpkeep, body)))
		host := NewCreature("Host", "", 2, 2)
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutOnBattlefield(host, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.stack.Size() != 1 {
			t.Fatal("missing source trigger")
		}
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(source, b.PlayerID())
		} else {
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.PhaseOut(source.ID())
		}
		g.ResolveStack()
		want := 4
		if returned {
			want = 2
		}
		if got := g.FindPermanent(host.ID()).CurrentPower(g); got != want {
			t.Fatalf("returned %v: power = %d, want %d", returned, got, want)
		}
	}
}

func TestComposableSourceDurationDirectPhasedContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	g.PhaseOut(source.ID())
	effect := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(0)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(host.ID()).CurrentPower(g); got != 3 {
		t.Fatalf("power = %d, want 3", got)
	}
}

func TestComposableSourceDurationBeforeResolution(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", 1, 1)
			host := NewCreature("Host", "", 2, 2)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(host, a.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: g.FindPermanent(source.ID()).incarnationID}
			if state == "phased" {
				g.PhaseOut(source.ID())
			} else {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					g.PutOnBattlefield(source, a.PlayerID())
				}
			}
			effect := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(3), Fixed(0)))
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 2
			if state == "phased" {
				want = 5
			}
			if got := g.FindPermanent(host.ID()).CurrentPower(g); got != want {
				t.Fatalf("power = %d, want %d", got, want)
			}
		})
	}
}

func TestComposableSourceDurationCapturedMembers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	effect := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), ValueAtResolution(CountBattlefield(SelectController(), IsCreature)), Fixed(0)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	later := NewCreature("Later", "", 2, 2)
	g.PutOnBattlefield(later, a.PlayerID())
	phased := g.PhaseOut(host.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CurrentPower(g) != 4 || g.FindPermanent(later.ID()).CurrentPower(g) != 2 {
		t.Fatal("membership, amount, or phasing did not retain the captured value")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CurrentPower(g) != 2 {
		t.Fatal("modification followed a returned member")
	}
}

func TestComposableSourceDurationValidation(t *testing.T) {
	saved := BindPermanent("missing")
	for _, effect := range []ResolvingEffect{
		UntilSourceLeaves(This),
		UntilSourceLeaves(This, nil),
		UntilSourceLeaves(This, ModifyPT(From(saved), Fixed(1), Fixed(1))),
	} {
		card := NewCreature("Invalid duration", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition("Invalid duration", card); err == nil || !strings.Contains(err.Error(), "abilities[0].effects[0]") {
			t.Fatalf("validation = %v", err)
		}
	}
	spell := NewSorcery("Invalid source", "", WithAbility(Spell(UntilSourceLeaves(This, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(1))))))
	if err := ValidateCardDefinition("Invalid source", spell); err == nil {
		t.Fatal("spell-only source accepted")
	}
}
