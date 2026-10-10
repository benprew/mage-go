package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNoLandTypeDjinnUpkeep(t *testing.T) {
	for _, scenario := range []string{"island", "swamp", "dual", "token", "exile", "prevent", "departed", "returned", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.Upkeep)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			card := repositoryCard(t, "ARN", "Serendib Djinn")
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if !source.HasKeyword(core.Flying) || source.CurrentPower(g) != 5 || source.CurrentToughness(g) != 6 {
				t.Fatal("printed characteristics")
			}
			name := "Island"
			if scenario == "swamp" {
				name = "Swamp"
			}
			if scenario == "dual" {
				name = "Tropical Island"
			}
			land := repositoryCard(t, "LEB", name)
			land.SetOwner(b.PlayerID())
			if scenario == "token" {
				land = mage.NewToken("Island token", 0, 0, []core.CardType{core.TypeLand}, []string{"Island"})
				land.SetOwner(b.PlayerID())
			}
			victim := g.PutOnBattlefield(land, a.PlayerID())
			a.choice = victim.ID()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.FindPermanent(victim.ID()) == nil || a.Life() != 20 {
				t.Fatal("upkeep announcement timing")
			}
			if scenario == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(victim.ID(), source.ID())
			}
			if scenario == "prevent" {
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			if scenario == "departed" || scenario == "returned" {
				g.BouncePermanentToHand(source)
				if scenario == "returned" {
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.PhaseOut(victim.ID())
			}
			g.ResolveStack()
			want := 17
			if scenario == "swamp" {
				want = 20
			}
			if scenario == "prevent" {
				want = 19
			}
			if a.Life() != want || b.Life() != 20 || g.FindPermanent(victim.ID()) != nil {
				t.Fatalf("sacrifice or damage: A=%d B=%d", a.Life(), b.Life())
			}
			if scenario == "exile" && g.FindExiledCard(victim.ID()) == nil {
				t.Fatal("replacement destination")
			}
			if scenario == "clone" {
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanentIncludingPhased(victim.ID()) == nil {
					t.Fatal("cloned sacrifice facts")
				}
			}
		})
	}
}

func TestRONNoLandTypeDjinnState(t *testing.T) {
	for _, scenario := range []string{"acquired", "counter", "control", "phase", "returned", "clone", "copy", "two", "removed"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
			g.PutOnBattlefield(mage.NewArtifact("Other permanent", ""), a.PlayerID())
			card := repositoryCard(t, "ARN", "Serendib Djinn")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("non-Island land did not prevent trigger")
			}
			g.BouncePermanentToHand(land)
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("no-land condition did not trigger")
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("pending state trigger repeated")
			}
			switch scenario {
			case "acquired":
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("countered trigger did not rearm")
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: land.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(source.ID())
				g.ResolveStack()
				if g.FindPermanentIncludingPhased(source.ID()) == nil {
					t.Fatal("sacrificed phased source")
				}
				g.PhaseIn([]uuid.UUID{source.ID()})
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("phase-in did not rearm")
				}
			case "returned":
				g.BouncePermanentToHand(source)
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(source.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("clone state isolation")
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(source.ID()) == nil {
					t.Fatal("copy sacrificed foreign permanent")
				}
			case "two":
				other := g.PutOnBattlefield(repositoryCard(t, "ARN", "Serendib Djinn"), a.PlayerID())
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("instance identities collided")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(other.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("second instance used first source")
				}
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			survives := scenario == "control" || scenario == "returned"
			if (g.FindPermanent(source.ID()) != nil) != survives {
				t.Fatal("pending source identity or controller")
			}
		})
	}
}

func TestRONNoLandTypeDjinnInactiveAndLastLand(t *testing.T) {
	for _, mode := range []string{"source-phase", "land-phase", "ability-removal", "last-upkeep"} {
		t.Run(mode, func(t *testing.T) {
			g, a, _ := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Serendib Djinn"), a.PlayerID())
			switch mode {
			case "source-phase":
				g.PhaseOut(source.ID())
				g.BouncePermanentToHand(land)
			case "land-phase":
				g.PhaseOut(land.ID())
			case "ability-removal":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
				g.BouncePermanentToHand(land)
			case "last-upkeep":
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveTopOfStack()
				if a.Life() != 17 || g.FindPermanent(land.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("last-land upkeep sequence")
				}
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			want := 1
			if mode == "source-phase" || mode == "ability-removal" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatal("state detection", g.StackSize(), want)
			}
			g.ResolveStack()
			if want == 1 && g.FindPermanent(source.ID()) != nil {
				t.Fatal("state trigger did not sacrifice source")
			}
		})
	}
}

func TestRONNoLandTypeInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(You),IsCardtype(Land)),ActionList([SacrificePermanent(ThisPermanent)]))`
	for _, filter := range []string{"IsCardtype", "IsCardtype()", "IsCardtype(Land,Creature)", "IsCardtype(Land())", "IsCardtype(Unknown)", "IsCardtype(Creature)", "And([IsCardtype(Land)])"} {
		assertConditionalRejected(t, strings.Replace(valid, "IsCardtype(Land)", filter, 1))
	}
	for _, player := range []string{"AnyPlayer", "SinglePlayer(Opponent)", "SinglePlayer(TheChosenPlayer)", "SinglePlayer(Trigger_ThatPlayer)", "SinglePlayer(You())"} {
		assertConditionalRejected(t, strings.Replace(valid, "SinglePlayer(You)", player, 1))
	}
}

func TestRONNoLandTypeDjinnCasting(t *testing.T) {
	g, a, _ := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	card := repositoryCard(t, "ARN", "Serendib Djinn")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed cast spent resources")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("casting payment or timing")
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || !p.HasKeyword(core.Flying) {
		t.Fatal("RON permanent spell entry")
	}
}
