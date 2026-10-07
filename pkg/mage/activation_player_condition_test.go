package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationPlayerConditionIdentity(t *testing.T) {
	for _, attached := range []bool{false, true} {
		for _, owner := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 3)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			source := p
			var ref PermanentRef = This
			if attached {
				source = g.PutOnBattlefield(NewAura("Source", ""), b.PlayerID())
				g.Attach(source.ID(), p.ID())
				ref = AttachedPermanent
			}
			condition := PlayerIsPermanentController(ref)
			if owner {
				condition = PlayerIsPermanentOwner(ref)
			}
			ctx := StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: a.PlayerID()}
			check := func(game *Game, want bool) {
				t.Helper()
				if condition.checkState(game, ctx) != want {
					t.Fatalf("attached=%v owner=%v want=%v", attached, owner, want)
				}
			}
			check(g, true)
			clone := g.Clone()
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, owner)
			check(clone, true)
			ctx.ControllerID = b.PlayerID()
			check(g, !owner)
			ctx.ControllerID = a.PlayerID()
			phased := g.PhaseOut(p.ID())
			check(g, false)
			g.PhaseIn(phased)
			check(g, owner)
			if attached {
				g.MutablePermanent(source.ID()).attachedIncarnation = uuid.New()
				check(g, false)
				g.Attach(source.ID(), p.ID())
				phased = g.PhaseOut(source.ID())
				check(g, false)
				g.PhaseIn(phased)
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			check(g, false)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, a.PlayerID())
			check(g, false)
			ctx.ControllerID = uuid.Nil
			check(clone, false)
			ctx.ControllerID = uuid.New()
			check(clone, false)
			ctx.ControllerID = a.PlayerID()
			ctx.SourceIncarnation = uuid.New()
			check(clone, false)
		}
	}
}

func TestActivationPlayerConditionDeclarations(t *testing.T) {
	for _, build := range []func(PermanentRef) StateCondition{PlayerIsPermanentOwner, PlayerIsPermanentController} {
		for _, ref := range []PermanentRef{nil, DeclarePermanentTarget("target", IsCreature), From(BindPermanent("saved"))} {
			c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(build(ref)))))
			if err := ValidateCardDefinition(c.Name(), c); err == nil {
				t.Fatal("invalid reference accepted")
			}
		}
		for _, ref := range []PermanentRef{This, AttachedPermanent} {
			c := NewAura("Valid", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), AnyPlayerMayActivate(), ActivationIf(build(ref)))))
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
		}
	}
}
