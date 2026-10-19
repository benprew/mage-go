package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRequiredTapInspectionEnergyTap(t *testing.T) {
	for _, state := range []string{"normal", "zero", "X", "tapped", "phased", "phase-cycle", "returned", "control", "shroud", "copy", "opponent copy", "clone", "countered"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			cost, want := "{3}{U}", 4
			if state == "zero" {
				cost, want = "", 0
			}
			if state == "X" {
				cost = "{X}{3}{U}"
			}
			creature := mage.NewCreature("Subject", cost, 2, 3)
			creature.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(creature, a.PlayerID())
			card := repositoryCard(t, "LEG", "Energy Tap")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "tapped":
				g.TapPermanent(p)
				want = 0
			case "phased":
				g.PhaseOut(p.ID())
				want = 0
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(creature.ID())
				g.PutOnBattlefield(creature, a.PlayerID())
				want = 0
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 0
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				want = 0
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "opponent copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				for range 2 {
					if certain, known := game.InspectSpellDestruction(game.StackPeek(), nil); certain || !known {
						t.Fatal("native required tap is unknown")
					}
				}
				if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || game.FindPermanentIncludingPhased(p.ID()).Tapped != (state == "tapped") {
					t.Fatal("inspection paid or produced mana")
				}
			}
			if state == "countered" {
				g.CounterStackObject(g.StackPeek().ID)
				want = 0
			}
			g.ResolveStack()
			if a.ManaPool().Count(core.Colorless) != want {
				t.Fatalf("mana = %d, want %d", a.ManaPool().Count(core.Colorless), want)
			}
			if b.ManaPool().TotalMana() != 0 {
				t.Fatal("copy ignored its controller")
			}
			if state != "countered" {
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).ManaPool().Count(core.Colorless) != want {
					t.Fatal("clone differs")
				}
			}
		})
	}
}

func TestRONRequiredTapInspectionInvalid(t *testing.T) {
	for _, body := range []string{
		`MustCost(TapPermanent(Ref_TargetPermanent())),If(CostWasPaid,[AddMana(ManaProduceC)])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid(),[AddMana(ManaProduceC)])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid,[AddManaRepeated(ManaProduceC,ManaValueOfPermanent(Ref_TargetPermanent2))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid,[AddManaRepeated(ManaProduceC,ManaValueOfPermanent(Ref_TargetPlayer))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid,[])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+body+`])))])`)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
