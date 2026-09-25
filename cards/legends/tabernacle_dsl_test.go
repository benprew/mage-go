package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestTabernacleGrantedSource(t *testing.T) {
	for _, change := range []string{"none", "return before placement", "return on stack", "control", "land leaves", "regenerate", "indestructible", "phased"} {
		t.Run(change, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			land, err := mage.CreateCard("The Tabernacle at Pendrell Vale")
			if err != nil {
				t.Fatal(err)
			}
			creature := mage.NewCreature("Subject", "", 2, 2)
			other := mage.NewCreature("Other", "", 2, 2)
			land.SetOwner(a.PlayerID())
			creature.SetOwner(b.PlayerID())
			other.SetOwner(a.PlayerID())
			g.PutOnBattlefield(land, a.PlayerID())
			g.PutOnBattlefield(creature, b.PlayerID())
			g.PutOnBattlefield(other, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			if change == "return before placement" {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.PutOnBattlefield(creature, b.PlayerID())
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.StackPeek().SourceID != creature.ID() || g.StackPeek().Controller != b.PlayerID() || len(g.StackPeek().Targets) != 0 {
				t.Fatal("expected one nontargeted trigger controlled by the creature's controller")
			}
			info := mage.InspectDefinition(g.StackPeek().Effects[0])
			if info.Role != "unless-paid" || info.Properties.Outcome != mage.OutcomeDetriment {
				t.Fatal("expected typed payment and destruction metadata")
			}
			switch change {
			case "return on stack":
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.PutOnBattlefield(creature, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: land.ID(), TargetID: creature.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "land leaves":
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
			case "regenerate":
				g.AddRegenerationShield(creature.ID())
			case "indestructible":
				g.MutablePermanent(creature.ID()).GrantBaseAttr(core.Indestructible)
			case "phased":
				g.PhaseOut(creature.ID())
			}
			g.ResolveStack()
			survives := change == "return before placement" || change == "return on stack" || change == "regenerate" || change == "indestructible" || change == "phased"
			if (g.FindPermanentIncludingPhased(creature.ID()) != nil) != survives {
				t.Fatal("wrong source destruction outcome")
			}
			if g.FindPermanent(other.ID()) == nil {
				t.Fatal("destroyed another player's creature")
			}
		})
	}
}

func TestTabernacleGrantsToItselfWhenCreature(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	land, err := mage.CreateCard("The Tabernacle at Pendrell Vale")
	if err != nil {
		t.Fatal(err)
	}
	land.AddType(core.TypeCreature)
	land.SetOwner(a.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().SourceID != land.ID() {
		t.Fatal("all creatures must include the grant source")
	}
}

func TestTabernaclePaymentAfterControlChange(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(map[bool]string{false: "decline", true: "pay"}[accept], func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			land := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Tabernacle at Pendrell Vale")
			creature := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			b.ManaPool().Add(core.Green, 1)
			b.QueueMayAbilityChoices(accept)
			a.QueueMayAbilityChoices(!accept)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: land, TargetID: creature, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			g.ResolveStack()
			if (g.FindPermanent(creature) != nil) != accept {
				t.Fatal("payment did not use the triggering controller")
			}
			wantMana := 1
			if accept {
				wantMana = 0
			}
			if b.ManaPool().TotalMana() != wantMana {
				t.Fatal("incorrect payment amount")
			}
		})
	}
}
