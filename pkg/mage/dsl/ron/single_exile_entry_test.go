package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSingleExileEntryCoffin(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ATQ", "Tawnos's Coffin", "coffin.ron"))
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"untap", "leave", "stale", "token", "protection", "clone", "copy", "early-leave", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			coffin := factory()
			coffin.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(coffin, a.PlayerID())
			card := mage.NewCreature("Subject", "", 2, 4)
			if state == "token" {
				card = mage.NewToken("Token", 2, 4, []core.CardType{core.TypeCreature}, nil)
			}

			card.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(card, a.PlayerID())
			host.AddCounter(core.P1P1, 3)
			aura := mage.NewAura("Guest", "{B}", mage.WithCastTarget(mage.TargetCreature()))
			aura.SetOwner(a.PlayerID())
			guest := g.PutOnBattlefield(aura, a.PlayerID())
			g.Attach(guest.ID(), host.ID())
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{host.ID()}); err != nil {
				t.Fatal(err)
			}
			if state == "early-leave" {
				g.DestroyPermanent(source)
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) != nil || g.FindPermanent(aura.ID()) != nil {
				t.Fatal("exile failed")
			}
			if state == "protection" {
				g.FindExiledCard(card.ID()).Card.(*mage.BaseCard).AddAbility(mage.ProtectionFromColor(core.Black))
			}
			if state == "stale" {
				saved, _ := g.RemoveFromExile(card.ID())
				g.ExileCard(saved, source.ID())
			}
			if state == "early-leave" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("past departure replayed")
				}
				return
			}
			if state == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			if state == "leave" {
				g.DestroyPermanent(source)
			} else {
				g.UntapPermanent(source)
			}
			g.PutTriggersOnStack()
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			games := []*mage.Game{g}
			if state == "clone" {
				games = append(games, g.Clone())
			}
			for _, game := range games {
				game.ResolveStack()
				got := game.FindPermanent(card.ID())
				attached := game.FindPermanent(aura.ID())
				if state == "token" || state == "stale" {
					if got != nil || attached != nil || game.FindExiledCard(aura.ID()) == nil {
						t.Fatal("failed return continued")
					}
					continue
				}
				if got == nil || !got.Tapped || got.ControllerID() != b.PlayerID() || got.Counters[core.P1P1] != 3 {
					t.Fatal("entrant lost owner, tapped status or counters")
				}
				if state == "protection" {
					if attached != nil || game.FindExiledCard(aura.ID()) == nil {
						t.Fatal("illegal Aura returned")
					}
				} else if attached == nil || attached.AttachedTo != got.ID() {
					t.Fatal("Aura did not follow exact entrant")
				}
			}
		})
	}
}

const singleExileGroup = `Exile([Permanent(ThisPermanent),EachPermanent(IsCardtype(Creature))],[])`
const singleExileMove = `MustCost(PutExiledCardOntoBattlefield(TheSinglePermanentExiledThisWay,[EntersUnderOwnersControl]))`

func TestRONSingleExileEntryInvalid(t *testing.T) {
	for _, body := range []string{
		singleExileMove + `,If(CostWasPaid,[GainLife(Integer(1))])`,
		singleExileGroup + `,` + singleExileMove,
		singleExileGroup + `,` + singleExileMove + `,If(CostWasPaid(),[GainLife(Integer(1))])`,
		singleExileGroup + `,` + singleExileMove + `,If(CostWasPaid,[])`,
		singleExileGroup + `,` + singleExileMove + `,If(CostWasntPaid,[TapPermanent(ThePermanentPutOnTheBattlefieldThisWay)])`,
		singleExileGroup + `,` + singleExileMove + `,If(CostWasPaid,[GainLife(Integer(1))]),TapPermanent(ThePermanentPutOnTheBattlefieldThisWay)`,
		singleExileGroup + `,MustCost(PutExiledCardOntoBattlefield(TheOtherPermanentsExiledThisWay,[EntersUnderOwnersControl])),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, options := range []string{`[]`, `[EntersUnderOwnersControl()]`, `[EntersUnderOwnersControl,EntersTapped,EntersTapped]`, `[EntersUnderOwnersControl,EntersWithNotedCounters]`, `[EntersUnderOwnersControl,EntersTapped()]`, `[EntersUnderOwnersControl,EntersNormally]`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+singleExileGroup+`,MustCost(PutExiledCardOntoBattlefield(TheSinglePermanentExiledThisWay,`+options+`)),If(CostWasPaid,[GainLife(Integer(1))])]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", options)
		}
	}
}

func TestRONSingleExileEntryBranches(t *testing.T) {
	for _, branch := range []string{
		`If(CostWasPaid,[TapPermanent(ThePermanentPutOnTheBattlefieldThisWay),GainLife(Integer(2))])`,
		`Unless(CostWasntPaid,[TapPermanent(ThePermanentPutOnTheBattlefieldThisWay),GainLife(Integer(2))])`,
		`IfElse(CostWasntPaid,[GainLife(Integer(5))],[TapPermanent(ThePermanentPutOnTheBattlefieldThisWay),GainLife(Integer(2))])`,
	} {
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+singleExileGroup+`,`+singleExileMove+`,`+branch+`]))`)
		g, a, _ := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		got := g.FindPermanent(card.ID())
		if got == nil || !got.Tapped || a.Life() != 22 {
			t.Fatal("branch polarity or entrant result")
		}
	}
}

func TestRONSingleExileEntryWaitingInstances(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ATQ", "Tawnos's Coffin", "coffin.ron"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	sources := make([]uuid.UUID, 0, 2)
	subjects := make([]uuid.UUID, 0, 2)
	for i, player := range []mage.Player{a, b} {
		card := factory()
		card.SetOwner(player.PlayerID())
		source := g.PutOnBattlefield(card, player.PlayerID())
		subject := mage.NewCreature("Subject", "", 2, 4)
		subject.SetOwner(player.PlayerID())
		p := g.PutOnBattlefield(subject, player.PlayerID())
		p.AddCounter(core.Charge, i+2)
		player.ManaPool().Add(core.Colorless, 3)
		if err := g.ActivateAbilityByIndex(player.PlayerID(), source.ID(), 1, []uuid.UUID{p.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		sources = append(sources, source.ID())
		subjects = append(subjects, subject.ID())
	}
	for range 2 {
		if len(g.WaitingDelayedExileSources()) != 2 || len(g.GetExile()) != 2 {
			t.Fatal("waiting query changed saved visits")
		}
	}
	clone := g.Clone()
	clone.UntapPermanent(clone.FindPermanent(sources[0]))
	clone.ResolveStack()
	if clone.FindPermanent(subjects[0]) == nil || g.FindPermanent(subjects[0]) != nil || clone.FindPermanent(subjects[1]) != nil {
		t.Fatal("clone or installation frames overlap")
	}
	g.UntapPermanent(g.FindPermanent(sources[1]))
	g.ResolveStack()
	got := g.FindPermanent(subjects[1])
	if got == nil || got.Counters[core.Charge] != 3 || g.FindPermanent(subjects[0]) != nil {
		t.Fatal("independent factory results overlap")
	}
}

func TestRONSingleExileEntryContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`PutExiledCardOntoBattlefield(TheSinglePermanentExiledThisWay,[EntersUnderOwnersControl]) If(CostWasPaid,[GainLife(Integer(1))])`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileSingleExileEntry(nodes[0], nodes[1], ctx); err == nil {
			t.Fatal("accepted invalid context")
		}
	}
}

func TestRONSingleExileEntryInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + singleExileGroup + `,` + singleExileMove + `,If(CostWasPaid,[GainLife(Integer(1))])]))])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"TheSinglePermanentExiledThisWay": "card_reference", "EntersUnderOwnersControl": "entry_operation"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONSingleExileEntryUntapChoice(t *testing.T) {
	a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card := repositoryCard(t, "ATQ", "Tawnos's Coffin")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	subjectCard := mage.NewCreature("Subject", "", 2, 4)
	subjectCard.SetOwner(a.PlayerID())
	subject := g.PutOnBattlefield(subjectCard, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{subject.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(subject.ID()) != nil || a.untapCalls != 1 {
		t.Fatal("declined untap returned creature")
	}
	a.untap = true
	g.RunStepWithPriority(core.Untap)
	g.RunStepWithPriority(core.Upkeep)
	g.ResolveStack()
	if g.FindPermanent(subject.ID()) == nil || a.untapCalls != 2 {
		t.Fatalf("accepted untap: creature=%v calls=%d tapped=%v delayed=%d stack=%d", g.FindPermanent(subject.ID()) != nil, a.untapCalls, g.FindPermanent(source.ID()).Tapped, len(g.DelayedTriggers()), g.StackSize())
	}
}

func TestRONSingleExileEntryAnnouncement(t *testing.T) {
	for _, state := range []string{"mana", "illegal-target"} {
		g, a, b := ronGame()
		card := repositoryCard(t, "ATQ", "Tawnos's Coffin")
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		subjectCard := mage.NewCreature("Subject", "", 2, 4)
		subjectCard.SetOwner(b.PlayerID())
		subject := g.PutOnBattlefield(subjectCard, b.PlayerID())
		if state == "illegal-target" {
			subject.GrantBaseAttr(core.Shroud)
			a.ManaPool().Add(core.Colorless, 3)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{subject.ID()}); err == nil {
			t.Fatal("illegal activation succeeded")
		}
		wantMana := 0
		if state == "illegal-target" {
			wantMana = 3
		}
		if source.Tapped || a.ManaPool().TotalMana() != wantMana || g.FindPermanent(subject.ID()) == nil || len(g.GetExile()) != 0 || len(g.DelayedTriggers()) != 0 {
			t.Fatal("failed announcement changed game")
		}
	}
}
