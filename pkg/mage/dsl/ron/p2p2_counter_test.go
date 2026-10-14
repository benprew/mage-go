package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONP2P2CounterSoulExchange(t *testing.T) {
	for _, state := range []string{"thrull", "other", "clone", "doubled", "illegal", "payment-return", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(hostUntapRecord(t, "FEM", "Soul Exchange"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			set, name := "FEM", "Basal Thrull"
			if state == "other" {
				set, name = "LEB", "Grizzly Bears"
			}
			paid := repositoryCard(t, set, name)
			g.PutOnBattlefield(paid, a.PlayerID())
			guest := repositoryCard(t, "LEB", "Grizzly Bears")
			guest.SetOwner(a.PlayerID())
			a.AddToGraveyard(guest)
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Black, 2)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(paid.ID()) != nil || g.FindPermanent(guest.ID()) != nil || len(g.GetExile()) != 1 {
				t.Fatal("incorrect cost or entry timing")
			}
			if state == "payment-return" {
				if _, ok := g.RemoveFromExile(paid.ID()); !ok {
					t.Fatal("missing payment")
				}
				g.PutOnBattlefield(paid, a.PlayerID())
			}
			if state == "illegal" {
				a.RemoveFromGraveyard(guest.ID())
				g.PutOnBattlefield(guest, a.PlayerID())
			}
			if state == "doubled" {
				doubler := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
				g.AddCounterDoubler(doubler.ID(), core.P2P2, mage.PermanentFilter{})
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 1
			if state == "other" || state == "illegal" {
				want = 0
			}
			if state == "doubled" {
				want = 2
			}
			p := g.FindPermanent(guest.ID())
			if p == nil || p.Counters[core.P2P2] != want || p.CurrentPower(g) != 2+2*want || p.CurrentToughness(g) != 2+2*want {
				t.Fatal("wrong returned permanent counters", want)
			}
			if state == "clone" {
				if clone.FindPermanent(guest.ID()) != nil {
					t.Fatal("live resolution changed pending clone")
				}
				clone.ResolveStack()
				if cp := clone.FindPermanent(guest.ID()); cp == nil || cp.Counters[core.P2P2] != 1 {
					t.Fatal("clone lost payment or return bindings")
				}
			}
		})
	}
}

func TestRONP2P2CounterInvalid(t *testing.T) {
	for _, counter := range []string{"PTCounter", "PTCounter()", "PTCounter(2)", "PTCounter(2,2,2)", "PTCounter(Integer(2),2)", "PTCounter(2,3)"} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([PutCounters([ACounterOfTypeOnPermanent(` + counter + `,ThisPermanent)])]))])`
		records, err := Parse(strings.NewReader(source), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(records[0]); err == nil {
			t.Fatal("invalid counter accepted", counter)
		}
	}
}

func TestRONP2P2CounterSavedVisit(t *testing.T) {
	for _, mode := range []string{"present", "phase", "cycle", "return"} {
		t.Run(mode, func(t *testing.T) {
			body := returnedGuest + `,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(2,2),ThePermanentPutOnTheBattlefieldThisWay)])]))`
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([`+body+`])))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			guest := repositoryCard(t, "LEB", "Grizzly Bears")
			guest.SetOwner(a.PlayerID())
			a.AddToGraveyard(guest)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{guest.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(guest.ID())
			g.BouncePermanentToHand(source)
			switch mode {
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(guest.ID())
				g.PutOnBattlefield(guest, a.PlayerID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if mode == "phase" {
				g.PhaseIn([]uuid.UUID{guest.ID()})
			}
			want := 1
			if mode == "phase" || mode == "return" {
				want = 0
			}
			if g.FindPermanent(guest.ID()).Counters[core.P2P2] != want {
				t.Fatal("saved visit did not control placement")
			}
		})
	}
}

func TestRONP2P2CounterSoulExchangeRollback(t *testing.T) {
	g, a, _ := ronGame()
	paid := repositoryCard(t, "FEM", "Basal Thrull")
	g.PutOnBattlefield(paid, a.PlayerID())
	guest := repositoryCard(t, "LEB", "Grizzly Bears")
	guest.SetOwner(a.PlayerID())
	a.AddToGraveyard(guest)
	spell := repositoryCard(t, "FEM", "Soul Exchange")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{guest.ID()}, 0); err == nil {
		t.Fatal("incomplete mana payment succeeded")
	}
	if g.FindPermanent(paid.ID()) == nil || a.ManaPool().TotalMana() != 1 || len(g.GetExile()) != 0 || g.StackSize() != 0 || len(a.Graveyard()) != 1 {
		t.Fatal("failed casting changed payment resources")
	}
}

func TestRONP2P2CounterReplacementOrder(t *testing.T) {
	for _, tc := range []struct {
		name         string
		choice, want int
		prevent      bool
	}{
		{"add-then-double", 0, 4, false}, {"double-then-add", 1, 3, false}, {"prevent", 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("Alice"), index: 1 - tc.choice}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("Bob"), index: tc.choice}
			g := mage.NewGame(a, b)
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(2, 2), Ref_TargetPermanent)]), GainLife(Integer(1))])))`)()
			source := g.PutOnBattlefield(card, a.PlayerID())
			target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
			g.AddReplacementEffect(&counterPlacementReplacement{source: source.ID(), prevent: tc.prevent, counter: core.P2P2})
			g.AddCounterDoubler(source.ID(), core.P2P2, mage.AnyPermanent)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(target.ID()).Counters[core.P2P2] != tc.want {
				t.Fatal("placement ignored affected player's replacement order")
			}
			if a.choices != 0 || b.choices != 1 || a.Life() != 21 {
				t.Fatal("wrong replacement chooser or prevented placement stopped next action")
			}
		})
	}
}
