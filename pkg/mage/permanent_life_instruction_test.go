package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentLifeInstructionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("life", AnyPermanent)
	for _, mana := range []bool{false, true} {
		instruction := PermanentLifeInstruction{Subject: target, ManaValue: mana}
		got, ok := InspectPermanentLifeInstruction(instruction.Effect())
		if !ok || got.Subject != target || got.ManaValue != mana {
			t.Fatal("lost information declaration")
		}
		got.Subject = nil
		again, _ := InspectPermanentLifeInstruction(instruction.Effect())
		if again.Subject != target {
			t.Fatal("inspection shares mutable storage")
		}
	}
	if (PermanentLifeInstruction{}).Effect() != nil {
		t.Fatal("nil target accepted")
	}
	var absent *PermanentTargetBinding
	var absentEffect *lifeAction
	for _, effect := range []Effect{
		nil, absentEffect, GainLifeFor(ControllerOfWithLKI(absent), PowerOfWithLKI(absent)),
		GainLifeFor(ControllerOfWithLKI(target), PowerOfWithLKI(DeclarePermanentTarget("foreign", AnyPermanent))),
		GainLifeFor(OwnerOfWithLKI(target), PowerOfWithLKI(target)),
		GainLifeFor(ControllerOfWithLKI(target), ToughnessOfWithLKI(target)),
		LoseLifeFor(ControllerOfWithLKI(target), PowerOfWithLKI(target)),
		GainLifeForWithResult(ControllerOfWithLKI(target), PowerOfWithLKI(target), LifeGainedInto(BindNumber("gain"))),
		GainLifeFor(ControllerOfWithLKI(This), PowerOfWithLKI(This)),
		GainLifeFor(lifeInspectionPlayerTrap{}, damageInspectionValueTrap{}),
		Sequence((PermanentLifeInstruction{Subject: target}).Effect(), destructionInspectionTrap{}),
	} {
		if _, ok := InspectPermanentLifeInstruction(effect); ok {
			t.Fatal("unsupported information accepted")
		}
	}
}

func TestPermanentLifeInstructionSpell(t *testing.T) {
	for _, mana := range []bool{false, true} {
		for _, state := range []string{"only", "prefix", "suffix", "indestructible", "regeneration", "phase", "phase cycle", "returned", "missing visit", "foreign", "opaque suffix", "opaque all illegal", "partial illegal", "duplicate", "foreign all illegal", "opaque amount", "mismatched amount", "partial illegal prefix"} {
			t.Run(map[bool]string{true: "mana", false: "power"}[mana]+"/"+state, func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				life := (PermanentLifeInstruction{Subject: target, ManaValue: mana}).Effect()
				obj.Effects = []Effect{life}
				want, known := false, true
				watched := []PermanentVisit{{p.ID(), p.incarnationID}}
				switch state {
				case "prefix":
					obj.Effects = []Effect{Sequence(life, Destroy(target))}
					known = false
				case "suffix":
					obj.Effects = []Effect{Sequence(Destroy(target), life)}
					want = true
				case "indestructible":
					p.GrantBaseAttr(core.Indestructible)
					obj.Effects = []Effect{Sequence(Destroy(target), life)}
				case "regeneration":
					g.AddRegenerationShield(p.ID())
					obj.Effects = []Effect{Sequence(DestroyWithoutRegeneration(target), life)}
					want = true
				case "phase":
					g.PhaseOut(p.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.BouncePermanentToHand(p)
					g.PutOnBattlefield(p.Card, obj.Controller)
				case "missing visit":
					obj.targetIncarnations = nil
					known = false
				case "partial illegal prefix":
					p.GrantBaseAttr(core.Shroud)
					q := g.PutOnBattlefield(NewLand("other"), obj.Controller)
					other := DeclarePermanentTarget("other", IsLand)
					obj.TargetSpecs = append(obj.TargetSpecs, other.targetSpec())
					obj.Targets = append(obj.Targets, q.ID())
					obj.targetIncarnations[q.ID()] = q.incarnationID
					obj.TargetZones[q.ID()] = core.ZoneBattlefield
					obj.Effects = []Effect{Sequence(life, Destroy(other))}
					watched = []PermanentVisit{{q.ID(), q.incarnationID}}
					want = true
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
					obj.Targets = append(obj.Targets, p.ID())
					known = false
				case "foreign all illegal":
					p.GrantBaseAttr(core.Shroud)
					obj.Effects = []Effect{(PermanentLifeInstruction{Subject: DeclarePermanentTarget("foreign", IsLand)}).Effect()}
					known = false
				case "opaque amount":
					obj.Effects = []Effect{Sequence(Destroy(target), GainLifeFor(ControllerOfWithLKI(target), damageInspectionValueTrap{}))}
					known = false
				case "mismatched amount":
					obj.Effects = []Effect{Sequence(Destroy(target), GainLifeFor(ControllerOfWithLKI(target), PowerOfWithLKI(DeclarePermanentTarget("foreign", IsLand))))}
					known = false
				case "foreign":
					obj.Effects = []Effect{(PermanentLifeInstruction{Subject: DeclarePermanentTarget("foreign", IsLand)}).Effect()}
					known = false
				case "opaque suffix":
					obj.Effects = []Effect{Sequence(Destroy(target), life, destructionInspectionTrap{})}
					known = false
				case "opaque all illegal":
					p.GrantBaseAttr(core.Shroud)
					obj.Effects = []Effect{Sequence(life, destructionInspectionTrap{})}
					known = false
				case "partial illegal":
					p.GrantBaseAttr(core.Shroud)
					other := DeclarePlayerTarget("other")
					obj.TargetSpecs = append(obj.TargetSpecs, other.targetSpec())
					obj.Targets = append(obj.Targets, obj.Controller)
				}
				for _, game := range []*Game{g, g.Clone()} {
					for range 2 {
						certain, k := game.InspectSpellDestruction(game.StackPeek(), watched)
						if certain != want || k != known {
							t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
						}
					}
				}
				if g.GetPlayer(obj.Controller).Life() != 20 {
					t.Fatal("inspection gained life")
				}
				if state == "regeneration" && !g.HasRegenerationShield(p.ID()) {
					t.Fatal("inspection used shield")
				}
			})
		}
	}
}

func TestPermanentLifeInstructionDeparture(t *testing.T) {
	for _, mana := range []bool{false, true} {
		for _, state := range []string{"departed", "returned", "phase", "control", "negative"} {
			t.Run(map[bool]string{true: "mana", false: "power"}[mana]+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("subject", "{3}{G}", 3, 8)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				p.AddCounter(core.P1P1, 2)
				target := DeclarePermanentTarget("subject", AnyPermanent)
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				receiver, want := b.PlayerID(), 5
				if mana {
					want = 4
				}
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						q := g.PutOnBattlefield(card, a.PlayerID())
						q.AddCounter(core.P1P1, 9)
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					receiver = a.PlayerID()
				case "negative":
					if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(-9), Fixed(0))), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if !mana {
						want = 0
					}
				}
				effect := (PermanentLifeInstruction{Subject: target, ManaValue: mana}).Effect()
				clone := g.Clone()
				local := *ctx
				local.Game = clone
				if err := effect.Apply(&local); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || b.Life() != 20 || clone.GetPlayer(receiver).Life() != 20+want {
					t.Fatal("clone or LKI contract failed")
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if g.GetPlayer(receiver).Life() != 20+want {
					t.Fatal("wrong information life gain")
				}
			})
		}
	}
}
