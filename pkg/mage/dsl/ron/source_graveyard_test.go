package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceGraveyardLich(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, state := range []string{"destroy", "bounce", "exile", "phase", "control", "return", "copy", "ability-lost", "replaced", "phase-cycle"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, "Lich")
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if a.Life() != 0 || !a.IsAlive() {
					t.Fatal("entry or exemption")
				}
				a.SetLife(20)
				other := mage.NewArtifact("Other", "")
				other.SetOwner(a.PlayerID())
				g.DestroyPermanent(g.PutOnBattlefield(other, a.PlayerID()))
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("unrelated death matched")
				}
				switch state {
				case "replaced":
					g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
					g.DestroyPermanent(p)
				case "phase-cycle":
					g.PhaseOut(p.ID())
					g.PhaseIn([]uuid.UUID{p.ID()})
					g.DestroyPermanent(p)
				case "bounce":
					g.BouncePermanentToHand(p)
				case "exile":
					g.ExilePermanent(p)
				case "phase":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					b.SetLife(20)
					g.DestroyPermanent(p)
				case "ability-lost":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
					g.DestroyPermanent(p)
				default:
					g.DestroyPermanent(p)
				}
				g.PutTriggersOnStack()
				triggered := state != "bounce" && state != "exile" && state != "phase" && state != "ability-lost" && state != "replaced"
				if g.StackSize() != map[bool]int{true: 1, false: 0}[triggered] || !a.IsAlive() || !b.IsAlive() {
					t.Fatal("departure timing")
				}
				if !triggered {
					return
				}
				if state == "return" {
					a.RemoveFromGraveyard(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					b.SetLife(20)
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				g.ResolveTopOfStack()
				losesB := state == "control" || state == "copy"
				if a.IsAlive() != losesB || b.IsAlive() == losesB {
					t.Fatal("wrong controller lost")
				}
				if !clone.GetPlayer(a.PlayerID()).IsAlive() || !clone.GetPlayer(b.PlayerID()).IsAlive() {
					t.Fatal("clone shared loss")
				}
				clone.ResolveTopOfStack()
				if clone.GetPlayer(a.PlayerID()).IsAlive() != losesB || clone.GetPlayer(b.PlayerID()).IsAlive() == losesB {
					t.Fatal("clone controller")
				}
			})
		}
	}
}

func TestRONSourceGraveyardInvalid(t *testing.T) {
	for _, filter := range []string{`SinglePermanent`, `SinglePermanent()`, `SinglePermanent(ThisPermanent())`, `SinglePermanent(HostPermanent)`, `SinglePermanent(Ref_TargetPermanent)`, `SinglePermanent(ThisPermanent,ThisPermanent)`} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(`+filter+`,AnyPlayer),ActionList([LoseTheGame]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONSourceGraveyardLichPayment(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			a := &requiredNumberPlayer{&optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, set, "Lich")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for range 3 {
				a.AddToLibrary(mage.NewCreature("Draw", "", 1, 1))
			}
			g.PlayerGainLife(a, 2)
			if a.Life() != 0 || len(a.Hand()) != 2 || len(a.Library()) != 1 {
				t.Fatal("life gain did not become draws")
			}
			a.SetLife(20)
			payment := mage.NewArtifact("Payment", "")
			payment.SetOwner(a.PlayerID())
			g.PutOnBattlefield(payment, a.PlayerID())
			g.DealDamageToPlayer(a, 2, source.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("damage trigger")
			}
			g.ResolveTopOfStack()
			if !a.IsAlive() || len(a.Graveyard()) != 2 || len(g.AllBattlefield()) != 0 {
				t.Fatal("simultaneous payment or early loss")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("source sacrifice did not queue one loss")
			}
			g.ResolveTopOfStack()
			if a.IsAlive() || !b.IsAlive() || a.Life() != 18 {
				t.Fatal("paid source departure did not cause explicit loss")
			}
		})
	}
}

func TestRONSourceGraveyardBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),PlayerAction(ControllerOfDeadPermanent,GainLife(Integer(3)))]))`)
	for range 2 {
		g, a, b := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, b.PlayerID())
		g.DestroyPermanent(p)
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.Life() != 22 || b.Life() != 23 {
			t.Fatal("owner and former controller bindings")
		}
	}
}

func TestRONSourceGraveyardIntervening(t *testing.T) {
	for _, sacrificed := range []bool{false, true} {
		factory := permanentScopeFactory(t, "Artifact", `TriggerI(WhenAPermanentIsPutIntoAPlayersGraveyard(SinglePermanent(ThisPermanent),AnyPlayer),PermanentPutInGraveyardPassesFilter(WasntSacrificed),ActionList([GainLife(Integer(2))]))`)
		g, a, _ := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if sacrificed {
			if err := mage.Sacrifice(mage.This).Apply(&mage.EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
		} else {
			g.DestroyPermanent(p)
		}
		g.PutTriggersOnStack()
		g.ResolveStack()
		want := 22
		if sacrificed {
			want = 20
		}
		if a.Life() != want {
			t.Fatal("death cause condition")
		}
	}
}
