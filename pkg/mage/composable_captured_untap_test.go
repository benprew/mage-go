package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCapturedUntapControlAndExpiry(t *testing.T) {
	for _, change := range []string{"none", "control", "phase", "return", "untapped", "other restriction"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			id := p.ID()
			p.Tapped = true
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: id}
			for range 2 {
				if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phase":
				g.PhaseOut(id)
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID()).Tapped = true
			case "untapped":
				g.UntapPermanent(p)
			case "other restriction":
				g.MutablePermanent(id).GrantBaseAttr(AttrDoesNotUntap)
			}
			g.turns.SetActivePlayerIndex(1)
			g.doUntap()
			if change == "control" && g.FindPermanent(id).Tapped {
				t.Fatal("captured A restriction blocked B's untap")
			}
			g.turns.SetActivePlayerIndex(0)
			g.doUntap()
			if change == "none" && !g.FindPermanent(id).Tapped {
				t.Fatal("A's next untap was not blocked")
			}
			if change == "return" && g.FindPermanent(id).Tapped {
				t.Fatal("restriction followed returned card")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{id})
			}
			if change == "other restriction" {
				g.MutablePermanent(id).RevokeBaseAttr(AttrDoesNotUntap)
			}
			g.MutablePermanent(id).Tapped = true
			if change == "control" {
				g.turns.SetActivePlayerIndex(1)
			}
			g.doUntap()
			if g.FindPermanent(id).Tapped {
				t.Fatal("restriction did not expire during A's step")
			}
		})
	}
}

func TestCapturedUntapCloneAndIndependentPlayers(t *testing.T) {
	g, a, b, p := controllerTestGame(t)
	p.Tapped = true
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
	if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.Controller = b.PlayerID()
	if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.doUntap()
	clone.doUntap()
	if clone.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone restriction did not expire")
	}
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone consumed original restriction")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.turns.SetActivePlayerIndex(1)
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("A's step consumed B's restriction")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("B's restriction did not expire")
	}
}

func TestCapturedUntapAbsentAndValidation(t *testing.T) {
	g, a, _, p := controllerTestGame(t)
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: uuid.New()}
	if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.Controller = a.PlayerID()
	saved := BindPermanent("missing")
	if err := SkipNextUntapOf(From(saved), You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	p.Tapped = true
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("absent reference installed restriction")
	}
	player := BindPlayer("player")
	for _, body := range []ResolvingEffect{
		SkipNextUntapOf(nil, You), SkipNextUntapOf(This, nil),
		SkipNextUntapOf(This, FromPlayer(player)),
		SkipNextUntapOf(DeclarePermanentTarget("target", IsCreature), You),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid definition")
		}
	}
}

func TestCapturedUntapTargetSourceExitAndExistingRestriction(t *testing.T) {
	g, a, b, p := controllerTestGame(t)
	p.Tapped = true
	target := DeclarePermanentTarget("creature", IsCreature)
	body := SkipNextUntapOf(target, You)
	ability := Activate(Costs(), body, WithTargets(target))
	card := NewArtifact("Source", "", WithAbility(ability))
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Card: card, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(source)
	g.ResolveStack()
	g.SkipNextUntap(p.ID())
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("source departure removed restriction")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.turns.SetActivePlayerIndex(1)
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("overlapping restrictions did not both expire")
	}
	if body.Properties().Outcome != OutcomeDetriment {
		t.Fatal("missing detriment metadata")
	}
}

func TestCapturedUntapSkippedStep(t *testing.T) {
	g, a, _, p := controllerTestGame(t)
	p.Tapped = true
	if err := SkipNextUntapOf(This, You).Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}); err != nil {
		t.Fatal(err)
	}
	g.turns.BuildNextTurn()
	g.SkipNextOccurrenceOfStep(Untap)
	step, ok := g.turns.PopNextStep()
	if !ok || step != Upkeep {
		t.Fatalf("next step = %v, %v", step, ok)
	}
	g.doUpkeepActions()
	g.turns.BuildNextTurn()
	step, ok = g.turns.PopNextStep()
	if !ok || step != Untap {
		t.Fatalf("next step = %v, %v", step, ok)
	}
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("skipped step consumed restriction")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("restriction survived actual untap step")
	}
}
