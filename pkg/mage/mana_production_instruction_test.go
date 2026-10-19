package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaProductionInstructionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("recipient")
	amount := Fixed(3)
	effect := (ManaProductionInstruction{Subject: target, Color: core.Black, Amount: amount}).Effect()
	got, ok := InspectManaProductionInstructions(Sequence(AddMana(core.Red, 2), CompositeEffects("nested", effect, AddManaAmount(core.Colorless, ValueAtResolution(XValue())))))
	if !ok || len(got) != 3 || got[0].Subject != You || got[0].Color != core.Red || got[1].Subject != target || got[1].Amount != amount || got[2].Color != core.Colorless {
		t.Fatal("lost native operands or instruction order")
	}
	got[1].Subject = nil
	again, _ := InspectManaProductionInstructions(effect)
	if again[0].Subject != target {
		t.Fatal("inspection shares result storage")
	}
	var missing *PlayerTargetBinding
	for _, d := range []ManaProductionInstruction{
		{}, {Subject: missing, Color: core.Red, Amount: amount}, {Subject: You, Color: core.Red},
		{Subject: ActivePlayer, Color: core.AnyColor, Amount: amount}, {Subject: You, Color: core.Color(-1), Amount: amount}, {Subject: You, Color: core.Color(99), Amount: amount},
	} {
		if d.Effect() != nil {
			t.Fatal("invalid declaration built an effect")
		}
	}
	var absent *manaAmountEffect
	var fixed *addManaEffect
	var player *manaToPlayerEffect
	var sequence *sequenceEffect
	var composite *compositeEffect
	for _, tree := range []Effect{nil, absent, fixed, player, sequence, composite, AddMana(core.AnyColor, 1), AddManaAmount(core.Red, nil), AddManaToPlayer(missing, core.Red, amount), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectManaProductionInstructions(tree); ok {
			t.Fatal("invalid or unknown child accepted")
		}
	}
	for _, tree := range []Effect{Sequence(), CompositeEffects("empty"), AddMana(core.Colorless, 0), AddManaAmount(core.Red, damageInspectionValueTrap{})} {
		if _, ok := InspectManaProductionInstructions(tree); !ok {
			t.Fatal("native declaration rejected")
		}
	}
}

func TestManaProductionInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "zero prefix", "suffix", "active", "target", "illegal", "all illegal", "missing", "foreign", "duplicate", "opaque subject", "opaque amount", "opaque suffix", "opaque all illegal", "saved amount", "X", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			pool := g.GetPlayer(obj.Controller).ManaPool()
			pool.AddRestricted(core.Green, 3, core.ArtifactSpellsOnly{})
			pool.DrainGeneric(1)
			subject, amount := PlayerRef(You), ResolutionValue(Fixed(3))
			target := DeclarePlayerTarget("recipient")
			if state == "target" || state == "illegal" || state == "all illegal" || state == "missing" || state == "foreign" || state == "duplicate" {
				subject = target
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, g.GetOpponent(obj.Controller).PlayerID())
			}
			want, known := false, true
			switch state {
			case "active":
				subject = ActivePlayer
			case "zero prefix":
				amount = Fixed(0)
			case "illegal", "all illegal":
				obj.Targets[1] = uuid.New()
				if state == "all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "missing":
				obj.TargetSpecs, obj.Targets = obj.TargetSpecs[:1], obj.Targets[:1]
				known = false
			case "foreign":
				subject = DeclarePlayerTarget("recipient")
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, obj.Targets[1])
				known = false
			case "opaque subject":
				subject = lifeInspectionPlayerTrap{}
				known = false
			case "opaque amount", "opaque all illegal":
				amount = damageInspectionValueTrap{}
				known = false
				if state == "opaque all illegal" {
					p.GrantBaseAttr(core.Shroud)
				}
			case "saved amount":
				amount = FromNumber(BindNumber("unbound"))
				known = false
			case "X":
				amount = ValueAtResolution(XValue())
				obj.XValue = 4
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(p)
				g.GetPlayer(p.ControllerID()).RemoveFromHand(p.Card.ID())
				g.PutOnBattlefield(p.Card, p.ControllerID())
			}
			mana := (ManaProductionInstruction{Subject: subject, Color: core.Black, Amount: amount}).Effect()
			obj.Effects = []Effect{mana}
			switch state {
			case "prefix", "zero prefix":
				obj.Effects = []Effect{Sequence(mana, Destroy(land))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), mana)}
				want = true
			case "illegal":
				obj.Effects = []Effect{Sequence(mana, Destroy(land))}
				want = true
			case "all illegal", "opaque all illegal", "phase", "return":
				obj.Effects = []Effect{Sequence(Destroy(land), mana)}
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), mana, destructionInspectionTrap{})}
				known = false
			}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("proof=%v/%v want=%v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(g.GetPlayer(obj.Controller).ManaPool(), before.GetPlayer(obj.Controller).ManaPool()) || !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() {
				t.Fatal("inspection changed mana, RNG or stack")
			}
		})
	}
}

func TestManaProductionInstructionExecution(t *testing.T) {
	for _, color := range []core.Color{core.Colorless, core.White, core.Blue, core.Black, core.Red, core.Green} {
		for _, count := range []int{-2, 0, 3} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePlayerTarget("recipient")
			card := NewInstant("Mana", "", Spell((ManaProductionInstruction{Subject: target, Color: color, Amount: Fixed(count)}).Effect(), WithTargets(target)))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if b.ManaPool().TotalMana() != 0 {
				t.Fatal("clone changed original pool")
			}
			g.ResolveStack()
			want := max(count, 0)
			for _, game := range []*Game{g, clone} {
				pool := game.GetPlayer(b.PlayerID()).ManaPool()
				if pool.TotalMana() != want || len(pool.ProducedThisTurn) != want || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("wrong recipient, amount or history")
				}
				for _, produced := range pool.ProducedThisTurn {
					if produced.Color != color {
						t.Fatal("wrong mana type")
					}
				}
			}
		}
	}
}
