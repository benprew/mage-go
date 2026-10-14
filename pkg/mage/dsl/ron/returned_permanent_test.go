package ron

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const returnedGuest = `PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally])`
const returnedGuestCounter = `PutCounters([ACounterOfTypeOnPermanent(PTCounter(+1,+1),ThePermanentPutOnTheBattlefieldThisWay)])`

func TestRONReturnedPermanentGameplay(t *testing.T) {
	for _, mode := range []string{"normal", "explicit-controller", "copy", "stale", "latest-failure", "delayed", "branch"} {
		t.Run(mode, func(t *testing.T) {
			body := returnedGuest + `,` + returnedGuestCounter
			switch mode {
			case "explicit-controller":
				body = `PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersUnderPlayersControl(You)]),` + returnedGuestCounter
			case "latest-failure":
				body = returnedGuest + `,` + returnedGuest + `,` + returnedGuestCounter
			case "delayed":
				body = returnedGuest + `,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + returnedGuestCounter + `]))`
			case "branch":
				body = returnedGuest + `,If(PermanentPassesFilter(ThePermanentPutOnTheBattlefieldThisWay,IsCardtype(Creature)),[` + returnedGuestCounter + `])`
			}
			factory := permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([`+body+`])))`)
			g, a, b := ronGame()
			guest := repositoryCard(t, "LEB", "Grizzly Bears")
			guest.SetOwner(a.PlayerID())
			a.AddToGraveyard(guest)
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if mode == "stale" {
				moved, ok := g.MoveFromGraveyard(a.PlayerID(), guest.ID(), core.ZoneHand)
				if !ok {
					t.Fatal("could not move target")
				}
				a.AddToGraveyard(moved)
			}
			if mode == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				if mode == "stale" {
					if game.FindPermanent(guest.ID()) != nil {
						t.Fatal("return followed a new graveyard visit")
					}
					continue
				}
				p := game.FindPermanent(guest.ID())
				if p == nil {
					t.Fatal("return failed")
				}
				if mode == "delayed" {
					if p.CurrentPower(game) != 2 {
						t.Fatal("delayed body ran early")
					}
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
				}
				want := 3
				if mode == "latest-failure" {
					want = 2
				}
				if game.FindPermanent(guest.ID()).CurrentPower(game) != want {
					t.Fatal("counter used the wrong entry result")
				}
				if mode == "copy" && p.ControllerID() != b.PlayerID() {
					t.Fatal("copy lost its controller")
				}
			}
		})
	}
}

func TestRONReturnedPermanentInvalid(t *testing.T) {
	for _, body := range []string{
		returnedGuestCounter,
		returnedGuestCounter + `,` + returnedGuest,
		`If(PermanentPassesFilter(ThisPermanent,IsCardtype(Artifact)),[` + returnedGuest + `]),` + returnedGuestCounter,
		returnedGuest + `,TapPermanent(ThePermanentPutOnTheBattlefieldThisWay())`,
		`PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[]),` + returnedGuestCounter,
		`PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersTapped]),` + returnedGuestCounter,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + returnedGuest + `])),` + returnedGuestCounter,
	} {
		assertGraveyardInvalid(t, `Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([`+body+`])))`)
	}
}

func TestRONReturnedPermanentInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([`+returnedGuest+`,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([`+returnedGuestCounter+`]))])))`)
	g, a, b := ronGame()
	guests := make([]uuid.UUID, 0, 2)
	for range 2 {
		guest := repositoryCard(t, "LEB", "Grizzly Bears")
		guest.SetOwner(a.PlayerID())
		a.AddToGraveyard(guest)
		guests = append(guests, guest.ID())
		spell := factory()
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{guest.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	for _, id := range guests {
		if g.FindPermanent(id).CurrentPower(g) != 3 {
			t.Fatal("factory instances shared an entry result")
		}
	}
}

func TestRONReturnedPermanentSourceClause(t *testing.T) {
	record := hostUntapRecord(t, "FEM", "Soul Exchange")
	rules := record.Fields["Rules"]
	spellRule := rules.Args[1]
	targeted := spellRule.Args[0]
	actions := targeted.Args[1]
	producer := actions.Args[0].Args[0]
	if producer.Name != "PutGraveyardCardOntoBattlefield" {
		t.Fatal("source return producer changed")
	}
	consumer := nameScopeNode(t, `TapPermanent(ThePermanentPutOnTheBattlefieldThisWay)`)
	actions.Args = []Node{{Kind: List, Args: []Node{producer, consumer}}}
	targeted.Args = []Node{targeted.Args[0], actions}
	spellRule.Args = []Node{targeted}
	rules.Args = []Node{spellRule}
	record.Fields = maps.Clone(record.Fields)
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
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
	g.ResolveStack()
	if p := g.FindPermanent(guest.ID()); p == nil || !p.Tapped {
		t.Fatal("source return clause did not supply its entrant")
	}
	if _, err := Compile(hostUntapRecord(t, "FEM", "Soul Exchange")); err != nil {
		t.Fatal("complete Soul Exchange must compile", err)
	}
}

func TestRONReturnedPermanentSavedFrames(t *testing.T) {
	for _, mode := range []string{"present", "phase", "cycle", "return", "historical", "shadow"} {
		t.Run(mode, func(t *testing.T) {
			body := returnedGuest + `,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + returnedGuestCounter + `]))`
			if mode == "historical" {
				body += `,` + returnedGuest + `,` + returnedGuestCounter
			}
			if mode == "shadow" {
				body = returnedGuest + `,If(PermanentPassesFilter(ThePermanentPutOnTheBattlefieldThisWay,IsCardtype(Creature)),[` + returnedGuest + `,` + returnedGuestCounter + `]),` + returnedGuestCounter
			}
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
			if mode == "shadow" {
				if p.CurrentPower(g) != 3 {
					t.Fatal("branch producer replaced the outer result")
				}
				return
			}
			if p.CurrentPower(g) != 2 {
				t.Fatal("latest failed producer reused an earlier result")
			}
			g.BouncePermanentToHand(source)
			clone := g.Clone()
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
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
			}
			if mode == "phase" {
				g.PhaseIn([]uuid.UUID{guest.ID()})
			}
			want := 3
			if mode == "phase" || mode == "return" {
				want = 2
			}
			if g.FindPermanent(guest.ID()).CurrentPower(g) != want || clone.FindPermanent(guest.ID()).CurrentPower(clone) != 3 {
				t.Fatal("delayed entry result lost exact identity or clone isolation")
			}
		})
	}
}
