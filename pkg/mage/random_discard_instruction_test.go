package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomDiscardInstructionDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("discard")
	effect := (RandomDiscardInstruction{Subject: target, Amount: Fixed(3)}).Effect()
	got, ok := InspectRandomDiscardInstructions(Sequence(effect, CompositeEffects("nested discard", effect)))
	if !ok || len(got) != 2 || got[0].Subject != target {
		t.Fatal("lost discard declaration")
	}
	got[0].Subject = nil
	again, _ := InspectRandomDiscardInstructions(effect)
	if again[0].Subject != target {
		t.Fatal("inspection shares declaration storage")
	}
	var absent *PlayerTargetBinding
	for _, instruction := range []RandomDiscardInstruction{{}, {Subject: absent, Amount: Fixed(1)}, {Subject: You}} {
		if instruction.Effect() != nil {
			t.Fatal("incomplete declaration built an effect")
		}
	}
	var absentEffect *randomDiscardSubjectEffect
	for _, effect := range []Effect{
		nil, absentEffect, DiscardAtRandom(nil, Fixed(1)), DiscardAtRandom(You, nil),
		Sequence(effect, destructionInspectionTrap{}),
	} {
		if _, ok := InspectRandomDiscardInstructions(effect); ok {
			t.Fatal("incomplete discard tree accepted")
		}
	}
}

func TestRandomDiscardInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "zero", "negative", "x", "active", "controller", "missing", "foreign", "duplicate", "illegal", "all illegal", "opaque amount", "opaque subject", "opaque child", "opaque suffix", "saved", "opaque all illegal", "missing all illegal", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, land := targetQueryFixture(t)
			player := DeclarePlayerTarget("discard")
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
				subject = DeclarePlayerTarget("discard")
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
			discard := (RandomDiscardInstruction{Subject: subject, Amount: amount}).Effect()
			obj.Effects = []Effect{discard}
			switch state {
			case "prefix", "zero", "negative":
				obj.Effects = []Effect{Sequence(discard, Destroy(land))}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), discard)}
				want = true
			case "illegal":
				obj.Effects = []Effect{Sequence(discard, Destroy(land))}
				want = true
			case "all illegal", "opaque all illegal", "missing all illegal":
				p.GrantBaseAttr(core.Shroud)
				if len(obj.Targets) > 1 {
					obj.Targets[1] = uuid.New()
				}
			case "opaque child":
				obj.Effects = []Effect{Sequence(discard, destructionInspectionTrap{})}
				known = false
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), DiscardAtRandom(You, damageInspectionValueTrap{}))}
				known = false
			case "phase", "return":
				obj.Effects = []Effect{Sequence(Destroy(land), discard)}
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

func TestRandomDiscardInstructionBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{NewArtifact("First", ""), NewArtifact("Second", ""), NewArtifact("Third", "")}
	for _, card := range cards {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
	}
	events := 0
	watcher := NewTriggered(core.EvtDiscard, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if len(a.Hand()) != 0 || len(a.Graveyard()) != 3 {
			t.Error("discard event preceded complete batch")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), b.PlayerID())
	g.SetRandomResults([]int{2, 0, 0, 7, 9})
	effect := (RandomDiscardInstruction{Subject: You, Amount: Fixed(5)}).Effect()
	instructions, ok := InspectRandomDiscardInstructions(effect)
	if !ok || len(instructions) != 1 {
		t.Fatal("missing native instruction")
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if events != 3 || len(a.Graveyard()) != 3 || a.Graveyard()[0].ID() != cards[2].ID() || g.RandIntn(10) != 7 {
		t.Fatal("batch lost selection, events or random state")
	}
	baseline := g.Clone()
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if events != 3 || g.RandIntn(10) != baseline.RandIntn(10) {
		t.Fatal("empty hand changed discard batch")
	}
}
