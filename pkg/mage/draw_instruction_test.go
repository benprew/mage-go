package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDrawInstructionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("draw")
	effect := (DrawInstruction{Subject: target, Amount: Fixed(3)}).Effect()
	got, ok := InspectDrawInstructions(Sequence(effect, CompositeEffects("nested draw", effect)))
	if !ok || len(got) != 2 || got[0].Subject != target {
		t.Fatal("lost draw declaration")
	}
	got[0].Subject = nil
	again, _ := InspectDrawInstructions(effect)
	if again[0].Subject != target {
		t.Fatal("inspection shares declaration storage")
	}
	var absent *PlayerTargetBinding
	for _, instruction := range []DrawInstruction{{}, {Subject: absent, Amount: Fixed(1)}, {Subject: You}} {
		if instruction.Effect() != nil {
			t.Fatal("incomplete declaration built an effect")
		}
	}
	var absentEffect *drawSubjectEffect
	for _, effect := range []Effect{
		nil, absentEffect, DrawFor(nil, Fixed(1)), DrawFor(You, nil),
		DrawFor(You, Fixed(1), DrawAmountInto(BindNumber("amount"))),
		DrawFor(You, Fixed(1), DrawCardInto(BindCard("card"))),
		DrawFor(You, Fixed(1), DrawCardsInto(BindCards("cards"))),
		DrawFor(You, Fixed(1), nil),
		DrawFor(You, Fixed(1), DrawAmountInto(nil)),
		DrawFor(You, Fixed(1), DrawCardInto(nil)),
		DrawFor(You, Fixed(1), DrawCardsInto(nil)),
		DrawFor(You, Fixed(1), DrawAmountInto(BindNumber("a")), DrawAmountInto(BindNumber("b"))),
		Sequence(effect, destructionInspectionTrap{}),
	} {
		if _, ok := InspectDrawInstructions(effect); ok {
			t.Fatal("incomplete or result-bearing draw tree accepted")
		}
	}
}

func TestDrawInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "zero", "negative", "x", "active", "controller", "missing", "foreign", "duplicate", "illegal", "all illegal", "opaque amount", "opaque subject", "opaque child", "opaque suffix", "saved", "opaque all illegal", "missing all illegal", "result suffix", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			player := DeclarePlayerTarget("draw")
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
			case "active":
				subject = ActivePlayer
			case "saved":
				subject = FromPlayer(BindPlayer("saved"))
				known = false
			case "missing", "missing all illegal":
				obj.TargetSpecs = obj.TargetSpecs[:1]
				obj.Targets = obj.Targets[:1]
				known = false
			case "foreign":
				subject = DeclarePlayerTarget("draw")
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				known = false
			case "illegal", "all illegal":
				obj.Targets[1] = uuid.New()
			case "opaque amount", "opaque all illegal":
				amount = damageInspectionValueTrap{}
				known = false
			case "opaque subject":
				subject = lifeInspectionPlayerTrap{}
				known = false
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
			}
			draw := (DrawInstruction{Subject: subject, Amount: amount}).Effect()
			obj.Effects = []Effect{draw}
			switch state {
			case "prefix", "zero", "negative":
				obj.Effects = []Effect{Sequence(draw, Destroy(land))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), draw)}
				want = true
			case "illegal":
				obj.Effects = []Effect{Sequence(draw, Destroy(land))}
				want = true
			case "all illegal", "opaque all illegal", "missing all illegal":
				p.GrantBaseAttr(core.Shroud)
				if len(obj.Targets) > 1 {
					obj.Targets[1] = uuid.New()
				}
			case "opaque child":
				obj.Effects = []Effect{Sequence(draw, destructionInspectionTrap{})}
				known = false
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), DrawFor(You, damageInspectionValueTrap{}))}
				known = false
			case "result suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), DrawFor(You, Fixed(1), DrawCardInto(BindCard("drawn"))))}
				known = false
			case "phase", "return":
				obj.Effects = []Effect{Sequence(Destroy(land), draw)}
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
			if !reflect.DeepEqual(g.random, before.random) || g.StackSize() != before.StackSize() || len(g.GetPlayer(obj.Controller).Hand()) != 0 || g.GetPlayer(obj.Controller).Life() != 20 {
				t.Fatal("inspection changed game state")
			}
		})
	}
}

func TestDrawInstructionExecution(t *testing.T) {
	for _, state := range []string{"fixed", "zero", "negative", "x", "replacement", "absent"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			for range 5 {
				card := NewLand("Library")
				card.SetOwner(a.PlayerID())
				a.AddToLibrary(card)
			}
			amount, subject := ResolutionValue(Fixed(3)), PlayerRef(You)
			want, life := 3, 20
			switch state {
			case "zero":
				amount, want = Fixed(0), 0
			case "negative":
				amount, want = Fixed(-2), 0
			case "x":
				amount, want = ValueAtResolution(XValue()), 4
				g.SetXValue(4)
			case "absent":
				subject, want = FromPlayer(BindPlayer("absent")), 0
			case "replacement":
				if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want, life = 2, 22
			}
			effect := (DrawInstruction{Subject: subject, Amount: amount}).Effect()
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != want || a.Life() != life || len(b.Hand()) != 0 {
				t.Fatalf("draw=%d life=%d want=%d/%d", len(a.Hand()), a.Life(), want, life)
			}
		})
	}
}
