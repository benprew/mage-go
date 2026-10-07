package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestUnlessPermanentCameUnderControlThisTurn(t *testing.T) {
	for _, change := range []string{"new", "old", "haste", "control", "control-back", "leave-new", "leave-old", "return-old", "phase-new", "phase-old", "unknown", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(2)
			card := NewCreature("Subject", "", 2, 3)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true}
			want := 20
			switch change {
			case "old", "leave-old", "return-old", "phase-old", "clone":
				g.SetTurn(3)
				want = 22
			case "haste":
				p.GrantBaseAttr(core.Haste)
				p.RevokeBaseAttr(core.AttrSummonSick)
			case "control", "control-back":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 20
				if change == "control-back" {
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					want = 20
				}
			case "unknown":
				ctx.sourceIncarnation = uuid.New()
			}
			switch change {
			case "leave-new", "leave-old", "return-old":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return-old" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase-new", "phase-old":
				g.PhaseOut(p.ID())
			}
			effect := UnlessPermanentCameUnderControlThisTurn(This, You, GainLifeFor(You, Fixed(2)))
			clone := g.Clone()
			if change == "clone" {
				clone.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				clone.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			copyCtx := *ctx
			copyCtx.Game = clone
			if err := effect.Apply(&copyCtx); err != nil {
				t.Fatal(err)
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			cloneWant := want
			if change == "clone" {
				cloneWant = 20
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
				t.Fatal("shared control history")
			}
		})
	}
}

func TestUnlessPermanentCameUnderControlDefinitions(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		UnlessPermanentCameUnderControlThisTurn(nil, You, GainLifeFor(You, Fixed(1))),
		UnlessPermanentCameUnderControlThisTurn(This, nil, GainLifeFor(You, Fixed(1))),
		UnlessPermanentCameUnderControlThisTurn(This, FromPlayer(BindPlayer("missing")), GainLifeFor(You, Fixed(1))),
		UnlessPermanentCameUnderControlThisTurn(This, You, nil),
	} {
		card := NewCreature("Invalid", "", 2, 3, WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
