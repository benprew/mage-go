package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFuturePreventionInstructionDeclaration(t *testing.T) {
	target := DeclareDamageTarget("shield")
	effect := (FuturePreventionInstruction{Recipient: target, Amount: Fixed(3)}).Effect()
	got, ok := InspectFuturePreventionInstructions(Sequence(effect, CompositeEffects("nested", effect)))
	if !ok || len(got) != 2 || got[0].Recipient != target {
		t.Fatal("lost native declaration")
	}
	got[0].Recipient = nil
	again, _ := InspectFuturePreventionInstructions(effect)
	if again[0].Recipient != target {
		t.Fatal("inspection shares storage")
	}
	var absent *DamageTargetBinding
	for _, instruction := range []FuturePreventionInstruction{{}, {Recipient: absent, Amount: Fixed(1)}, {Recipient: You}} {
		if instruction.Effect() != nil {
			t.Fatal("incomplete declaration built an effect")
		}
	}
	var absentEffect *preventionInstallation
	var absentSequence *sequenceEffect
	var absentComposite *compositeEffect
	for _, effect := range []Effect{nil, absentEffect, absentSequence, absentComposite, PreventNextDamage(nil, Fixed(1)), PreventNextDamage(You, nil), Sequence(effect, destructionInspectionTrap{}), PreventCombatDamageToAndBy(This)} {
		if _, ok := InspectFuturePreventionInstructions(effect); ok {
			t.Fatal("unknown tree accepted")
		}
	}
}

func TestFuturePreventionInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "zero", "negative", "x", "controller", "active", "player target", "missing", "foreign", "duplicate", "illegal", "all illegal", "shroud", "phase", "return", "opaque amount", "opaque recipient", "saved", "opaque suffix", "opaque all illegal", "missing all illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			target := DeclareDamageTarget("shield")
			victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), obj.Controller)
			obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
			obj.Targets = append(obj.Targets, victim.ID())
			g.stack.Pop()
			g.PushStack(obj)
			recipient := DamageRecipientRef(target)
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
				recipient = You
			case "active":
				recipient = ActivePlayer
			case "player target":
				player := DeclarePlayerTarget("player")
				recipient = player
				obj.TargetSpecs[1] = player.targetSpec()
				obj.Targets[1] = obj.Controller
			case "missing", "missing all illegal":
				obj.TargetSpecs = obj.TargetSpecs[:1]
				obj.Targets = obj.Targets[:1]
				known = false
				if state == "missing all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "foreign":
				recipient = DeclareDamageTarget("shield")
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, victim.ID())
				known = false
			case "illegal", "all illegal":
				g.RemoveFromBattlefield(victim)
				if state == "all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "shroud":
				victim.GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(victim.ID())
			case "return":
				g.BouncePermanentToHand(victim)
				g.GetPlayer(obj.Controller).RemoveFromHand(victim.Card.ID())
				g.PutOnBattlefield(victim.Card, obj.Controller)
			case "opaque amount", "opaque all illegal":
				amount = damageInspectionValueTrap{}
				known = false
				if state == "opaque all illegal" {
					p.GrantBaseAttr(core.Shroud)
					obj.Targets[1] = uuid.New()
				}
			case "opaque recipient":
				recipient = lifeInspectionPlayerTrap{}
				known = false
			case "saved":
				recipient = FromPlayer(BindPlayer("saved"))
				known = false
			}
			shield := (FuturePreventionInstruction{Recipient: recipient, Amount: amount}).Effect()
			obj.Effects = []Effect{shield}
			switch state {
			case "prefix", "zero", "negative":
				obj.Effects = []Effect{Sequence(shield, Destroy(land))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), shield)}
				want = true
			case "illegal", "shroud", "phase", "return":
				obj.Effects = []Effect{Sequence(shield, Destroy(land))}
				want = true
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), shield, destructionInspectionTrap{})}
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
			g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, uuid.New())
			if current := g.FindPermanent(victim.ID()); state != "phase" && current != nil && current.Damage != 1 {
				t.Fatal("inspection installed a shield")
			}
			if g.FindPermanent(p.ID()) == nil {
				t.Fatal("inspection destroyed a permanent")
			}
		})
	}
}

func TestFuturePreventionInstructionExecution(t *testing.T) {
	for _, state := range []string{"fixed", "x", "zero", "negative"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			amount, budget := ResolutionValue(Fixed(3)), 3
			switch state {
			case "x":
				amount = ValueAtResolution(XValue())
				g.SetXValue(4)
				budget = 4
			case "zero":
				amount = Fixed(0)
				budget = 0
			case "negative":
				amount = Fixed(-2)
				budget = 0
			}
			effect := (FuturePreventionInstruction{Recipient: You, Amount: amount}).Effect()
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			baseline := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), budget+1, uuid.New())
				if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("wrong native shield")
				}
				game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, uuid.New())
				if game.GetPlayer(a.PlayerID()).Life() != 18 {
					t.Fatal("shield reused its budget")
				}
			}
			if baseline.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone state crossed")
			}
		})
	}
}
