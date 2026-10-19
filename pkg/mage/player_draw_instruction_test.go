package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerDrawInstructionDeclaration(t *testing.T) {
	subjects := OpponentsOf(You)
	amount := Fixed(2)
	effect := (PlayerDrawInstruction{Subjects: subjects, Amount: amount}).Effect()
	got, ok := InspectPlayerDrawInstruction(effect)
	if !ok || got.Subjects != subjects || got.Amount != amount {
		t.Fatal("lost declaration")
	}
	for _, bad := range []PlayerDrawInstruction{{}, {Subjects: EachPlayer}, {Amount: amount}} {
		if bad.Effect() != nil {
			t.Fatal("missing operands accepted")
		}
	}
	var absent *drawEachEffect
	for _, bad := range []Effect{nil, absent, DrawEach(nil, amount), DrawEach(EachPlayer, nil), DrawFor(You, amount), Sequence(effect), destructionInspectionTrap{}} {
		if _, ok := InspectPlayerDrawInstruction(bad); ok {
			t.Fatal("invalid native declaration")
		}
	}
	trap := (PlayerDrawInstruction{Subjects: collectionDamagePlayerTrap{}, Amount: damageInspectionValueTrap{}}).Effect()
	if _, ok := InspectPlayerDrawInstruction(trap); !ok {
		t.Fatal("declaration inspector ran or rejected callbacks")
	}
}

func TestPlayerDrawInstructionQuery(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "zero", "negative", "x", "empty", "opponents", "target", "foreign", "illegal player", "all illegal", "opaque amount", "opaque players", "opaque suffix", "opaque illegal", "foreign illegal", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			players := PlayerSet(EachPlayer)
			amount := ResolutionValue(Fixed(2))
			want, known := false, true
			player := DeclarePlayerTarget("player")
			switch state {
			case "zero":
				amount = Fixed(0)
			case "negative":
				amount = Fixed(-1)
				known = false
			case "x":
				amount = ValueAtResolution(XValue())
				known = false
			case "empty":
				players = noDamagePlayers{}
			case "opponents":
				players = OpponentsOf(ActivePlayer)
			case "target", "illegal player", "all illegal":
				players = OpponentsOf(player)
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				if state != "target" {
					obj.Targets[1] = uuid.New()
				}
			case "foreign", "foreign illegal":
				players = OpponentsOf(player)
				known = false
			case "opaque amount", "opaque illegal":
				amount = damageInspectionValueTrap{}
				known = false
			case "opaque players":
				players = collectionDamagePlayerTrap{}
				known = false
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
			}
			draw := (PlayerDrawInstruction{Subjects: players, Amount: amount}).Effect()
			obj.Effects = []Effect{draw}
			switch state {
			case "prefix":
				obj.Effects = []Effect{Sequence(draw, Destroy(target))}
				known = false
			case "zero", "empty", "illegal player":
				obj.Effects = []Effect{Sequence(draw, Destroy(target))}
				want = true
			case "suffix":
				obj.Effects = []Effect{Sequence(Destroy(target), draw)}
				want = true
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(target), draw, destructionInspectionTrap{})}
				known = false
			case "all illegal", "opaque illegal", "foreign illegal":
				p.GrantBaseAttr(core.Shroud)
			case "phase", "return":
				obj.Effects = []Effect{Sequence(Destroy(target), draw)}
			}
			before := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					certain, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
					if certain != want || k != known {
						t.Fatalf("query=%v/%v want=%v/%v", certain, k, want, known)
					}
				}
			}
			if !reflect.DeepEqual(before.random, g.random) || len(g.GetPlayer(obj.Controller).Hand()) != 0 || g.StackSize() != before.StackSize() || len(g.resolution.playerBindings) != 0 {
				t.Fatal("inspection changed game state")
			}
		})
	}
}
