package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableSkipNextUntapSavedLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			id := p.ID()
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: id, Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(id)
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(id))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := SkipNextUntap(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := change == "none" || change == "phase cycle" || change == "control"
			if clone.turns.HasSkipNextUntap(id) != want {
				t.Fatalf("skip = %v, want %v", clone.turns.HasSkipNextUntap(id), want)
			}
			if g.turns.HasSkipNextUntap(id) {
				t.Fatal("clone changed the original")
			}
		})
	}
}

func TestComposableSkipNextUntapTargetAndSourceExit(t *testing.T) {
	g, a, _, p := controllerTestGame(t)
	p.Tapped = true
	target := DeclarePermanentTarget("creature", IsCreature)
	body := SkipNextUntap(target)
	ability := Activate(Costs(), body, WithTargets(target))
	card := NewArtifact("Skip source", "", WithAbility(ability))
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Card: card, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(source)
	g.ResolveStack()
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("source departure removed the restriction")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("restriction did not expire")
	}
	if body.Properties().Outcome != OutcomeDetriment {
		t.Fatal("missing detriment metadata")
	}
}

func TestComposableSkipNextUntapValidation(t *testing.T) {
	saved := BindPermanent("saved")
	for _, body := range []ResolvingEffect{
		SkipNextUntap(nil),
		SkipNextUntap(From(saved)),
		SkipNextUntap(DeclarePermanentTarget("undeclared", IsCreature)),
		Sequence(SkipNextUntap(From(saved)), Snapshot(This, saved)),
	} {
		card := NewCreature("Invalid skip", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid subject")
		}
	}
}
