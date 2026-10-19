package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLifeInstructionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("life")
	for _, gain := range []bool{false, true} {
		instruction := LifeInstruction{Subject: target, Amount: Fixed(3), Gain: gain}
		effect := instruction.Effect()
		got, ok := InspectLifeInstructions(Sequence(effect, CompositeEffects("nested life", effect)))
		if !ok || len(got) != 2 || got[0].Subject != target || got[0].Gain != gain {
			t.Fatal("lost life declaration")
		}
		got[0].Subject = nil
		again, _ := InspectLifeInstructions(effect)
		if again[0].Subject != target {
			t.Fatal("inspection shares declaration storage")
		}
	}
	var absent *PlayerTargetBinding
	for _, instruction := range []LifeInstruction{{}, {Subject: absent, Amount: Fixed(1)}, {Subject: You}} {
		if instruction.Effect() != nil {
			t.Fatal("incomplete declaration built an effect")
		}
	}
	var absentEffect *lifeAction
	for _, effect := range []Effect{nil, absentEffect, GainLifeFor(nil, Fixed(1)), LoseLifeFor(You, nil), GainLifeForWithResult(You, Fixed(1), LifeGainedInto(BindNumber("gain"))), LoseLifeForWithResult(You, Fixed(1), LifeLostInto(BindNumber("loss"))), LoseLifeForWithResult(You, Fixed(1), nil), Sequence(GainLifeFor(You, Fixed(1)), destructionInspectionTrap{})} {
		if _, ok := InspectLifeInstructions(effect); ok {
			t.Fatal("incomplete or result-bearing life tree accepted")
		}
	}
}

func TestLifeInstructionSpell(t *testing.T) {
	for _, gain := range []bool{false, true} {
		for _, state := range []string{"only", "prefix", "suffix", "zero", "negative", "x", "active", "missing", "foreign", "duplicate", "illegal", "all illegal", "opaque amount", "opaque subject", "opaque child", "opaque suffix", "controller", "history", "saved", "opaque all illegal", "missing all illegal"} {
			t.Run(state+map[bool]string{true: " gain", false: " loss"}[gain], func(t *testing.T) {
				g, p, obj, land := targetQueryFixture(t)
				player := DeclarePlayerTarget("life")
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				subject := PlayerRef(player)
				amount := ResolutionValue(Fixed(3))
				want, known := false, true
				switch state {
				case "zero":
					amount = Fixed(0)
				case "negative":
					amount = Fixed(-1)
				case "x":
					amount = ValueAtResolution(XValue())
				case "controller":
					subject = You
				case "history":
					subject = DeclarePlayerTarget("history", TargetPlayerAttackedThisTurn())
					known = false
				case "saved":
					subject = FromPlayer(BindPlayer("saved"))
					known = false
				case "opaque all illegal":
					amount = damageInspectionValueTrap{}
					p.GrantBaseAttr(core.Shroud)
					obj.Targets[1] = uuid.New()
					known = false
				case "missing all illegal":
					obj.TargetSpecs = obj.TargetSpecs[:1]
					obj.Targets = obj.Targets[:1]
					p.GrantBaseAttr(core.Shroud)
					known = false
				case "active":
					subject = ActivePlayer
				case "missing":
					obj.TargetSpecs = obj.TargetSpecs[:1]
					obj.Targets = obj.Targets[:1]
					known = false
				case "foreign":
					subject = DeclarePlayerTarget("life")
					known = false
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
					obj.Targets = append(obj.Targets, obj.Controller)
					known = false
				case "illegal", "all illegal":
					obj.Targets[1] = uuid.New()
				case "opaque amount":
					amount = damageInspectionValueTrap{}
					known = false
				case "opaque subject":
					subject = lifeInspectionPlayerTrap{}
					known = false
				}
				life := (LifeInstruction{Subject: subject, Amount: amount, Gain: gain}).Effect()
				obj.Effects = []Effect{life}
				switch state {
				case "prefix", "zero", "negative":
					obj.Effects = []Effect{Sequence(life, Destroy(land))}
					known = false
				case "suffix":
					obj.Effects = []Effect{Sequence(Destroy(land), life)}
					want = true
				case "illegal":
					obj.Effects = []Effect{Sequence(life, Destroy(land))}
					want = true
				case "all illegal":
					p.GrantBaseAttr(core.Shroud)
				case "opaque child":
					obj.Effects = []Effect{Sequence(life, destructionInspectionTrap{})}
					known = false
				case "opaque suffix":
					obj.Effects = []Effect{Sequence(Destroy(land), GainLifeFor(You, damageInspectionValueTrap{}))}
					known = false
				}
				for _, game := range []*Game{g, g.Clone()} {
					for range 2 {
						certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
						if certain != want || k != known {
							t.Fatalf("proof=%v/%v want=%v/%v", certain, k, want, known)
						}
					}
				}
				if g.GetPlayer(obj.Controller).Life() != 20 || g.FindPermanent(p.ID()) == nil {
					t.Fatal("query changed state")
				}
			})
		}
	}
}

type lifeInspectionPlayerTrap struct{}

func (lifeInspectionPlayerTrap) playerRef()       {}
func (lifeInspectionPlayerTrap) damageRecipient() {}
func (lifeInspectionPlayerTrap) resolvePlayer(*EffectContext) (uuid.UUID, bool) {
	panic("player callback")
}
func (lifeInspectionPlayerTrap) resolveRecipient(*EffectContext) (uuid.UUID, bool) {
	panic("recipient callback")
}

func TestLifeInstructionExecution(t *testing.T) {
	for _, state := range []string{"gain", "loss", "replacement", "prohibited gain", "prohibited loss", "zero", "negative", "x"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			gain := state != "loss" && state != "prohibited loss"
			amount := ResolutionValue(Fixed(3))
			want := 23
			switch state {
			case "loss":
				want = 17
			case "replacement":
				g.AddReplacementEffect(&lifeDrawRewrite{replace: func(action *LifeGainAction, _ *Game) Action {
					return NewLifeGainAction(action.ActionSource(), action.PlayerID(), action.Amount()*2)
				}})
				want = 26
			case "prohibited gain":
				g.effects.Rules.AddCantGainLife(a.PlayerID())
				want = 20
			case "prohibited loss":
				g.effects.Rules.AddCantLoseLife(a.PlayerID())
				want = 20
			case "zero":
				amount = Fixed(0)
				want = 20
			case "negative":
				amount = Fixed(-3)
				want = 20
			case "x":
				amount = ValueAtResolution(XValue())
				g.SetXValue(4)
				want = 24
			}
			g.AddPreventionShield(a.PlayerID(), 10)
			effect := (LifeInstruction{Subject: You, Amount: amount, Gain: gain}).Effect()
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
}
