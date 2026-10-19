package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTurnAttackRuleInstructionDeclaration(t *testing.T) {
	for _, require := range []bool{false, true} {
		instruction := TurnAttackRuleInstruction{Filter: IsCreature, Require: require}
		if require {
			instruction.Player = ActivePlayer
		}
		effect := instruction.Effect()
		got, ok := InspectTurnAttackRuleInstruction(effect)
		if !ok || got.Require != require || got.Player != instruction.Player || got.Filter.Text() != IsCreature.Text() {
			t.Fatal("lost declaration")
		}
		got.Require = !require
		again, _ := InspectTurnAttackRuleInstruction(effect)
		if again.Require != require {
			t.Fatal("inspection changed declaration")
		}
	}
	for _, instruction := range []TurnAttackRuleInstruction{
		{Filter: IsCreature, Require: true},
		{Filter: IsCreature, Player: You},
		{Filter: IsCreature, Require: true, Player: (*PlayerTargetBinding)(nil)},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, effect := range []Effect{nil, (*turnAttackRuleEffect)(nil), &turnAttackRuleEffect{require: true}, MustAttackThisTurn(This), destructionInspectionTrap{}} {
		if _, ok := InspectTurnAttackRuleInstruction(effect); ok {
			t.Fatal("accepted invalid effect")
		}
	}
}

func TestTurnAttackRuleInstructionQuery(t *testing.T) {
	for _, require := range []bool{false, true} {
		for _, state := range []string{"alone", "prefix", "suffix", "opaque filter", "nested opaque filter", "opaque player", "foreign player", "unknown suffix", "all illegal", "unknown all illegal", "clone", "copy"} {
			t.Run(state+map[bool]string{false: "/restriction", true: "/requirement"}[require], func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				instruction := TurnAttackRuleInstruction{Filter: IsCreature, Require: require}
				if require {
					instruction.Player = ActivePlayer
				}
				known, want := true, false
				trap := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") })
				switch state {
				case "opaque filter", "unknown all illegal":
					instruction.Filter = trap
					known = false
				case "nested opaque filter":
					instruction.Filter = And(IsCreature, trap)
					known = false
				case "opaque player":
					instruction.Require = true
					instruction.Player = FromPlayer(BindPlayer("unbound"))
					known = false
				case "foreign player":
					instruction.Require = true
					instruction.Player = DeclarePlayerTarget("foreign")
					known = false
				}
				effect := instruction.Effect()
				obj.Effects = []Effect{effect}
				switch state {
				case "prefix":
					obj.Effects = []Effect{Sequence(effect, Destroy(target))}
					known = false
				case "suffix", "clone", "copy":
					obj.Effects = []Effect{Sequence(Destroy(target), effect)}
					want = true
				case "unknown suffix":
					obj.Effects = []Effect{Sequence(Destroy(target), effect, destructionInspectionTrap{})}
					known = false
				case "all illegal", "unknown all illegal":
					p.GrantBaseAttr(core.Shroud)
				}
				watched := []PermanentVisit{{p.ID(), p.incarnationID}}
				for range 2 {
					certain, k := g.InspectSpellDestruction(obj, watched)
					if certain != want || k != known {
						t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
					}
				}
				if len(g.effects.effects) != 0 || g.FindPermanent(p.ID()) == nil {
					t.Fatal("inspection changed state")
				}
				if state == "clone" {
					clone := g.Clone()
					clone.PhaseOut(p.ID())
					if certain, k := clone.InspectSpellDestruction(clone.StackPeek(), watched); certain || !k {
						t.Fatal("clone ignored legality")
					}
				}
				if state == "copy" {
					copied := g.CopyStackObjectDirect(obj, obj.Controller, false)
					if copied == nil {
						t.Fatal("copy failed")
					}
					if certain, k := g.InspectSpellDestruction(copied, watched); !certain || !k {
						t.Fatal("copy lost declaration")
					}
				}
				if state == "suffix" || state == "clone" || state == "copy" {
					g.ResolveStack()
					if g.FindPermanent(p.ID()) != nil || len(g.effects.effects) == 0 {
						t.Fatal("native declaration changed ordered execution")
					}
				}
				if state == "all illegal" {
					g.ResolveStack()
					if len(g.effects.effects) != 0 {
						t.Fatal("illegal spell installed a rule")
					}
				}
			})
		}
	}
}

func TestTurnAttackRuleInstructionExecution(t *testing.T) {
	for _, require := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetTurn(1)
		instruction := TurnAttackRuleInstruction{Filter: IsCreature, Require: require}
		if require {
			instruction.Player = ActivePlayer
		}
		g.SetActivePlayerIndex(1)
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		effect := instruction.Effect()
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.RemoveFromBattlefield(source)
		card := NewCreature("Late", "", 2, 3)
		p := g.PutOnBattlefield(card, b.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		g.PhaseIn(g.PhaseOut(p.ID()))
		g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
		p = g.PutOnBattlefield(card, b.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		clone := g.Clone()
		g.ExecuteAttackers(b.PlayerID(), nil)
		if g.GetCombat().IsAttacking(p.ID()) != require {
			t.Fatal("rule did not use current members")
		}
		clone.RunStepWithPriority(core.Cleanup)
		if !clone.FindPermanent(p.ID()).CanDeclareAsAttacker(clone) || clone.GetCombat().IsAttacking(p.ID()) {
			t.Fatal("cleanup or clone isolation")
		}
	}
}
