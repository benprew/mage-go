package ron

import (
	"maps"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const entryPresenceRule = `TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),EnteringPermanentPassesFilter(OnTheBattlefield),ActionList([GainLife(Integer(2))]))`

func entryPresenceCard(t *testing.T, rule string) mage.Card {
	t.Helper()
	factory, err := Compile(nameScopeNode(t, `Card(Name:"Presence",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`))
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	return card
}

func TestRONEntryPresenceTiming(t *testing.T) {
	for _, change := range []string{"present", "left", "returned", "phased", "phase-cycle", "control", "lost-ability", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := entryPresenceCard(t, entryPresenceRule)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("entry did not queue one trigger")
			}
			clone := g.Clone()
			want := 22
			switch change {
			case "left", "returned":
				g.BouncePermanentToHand(source)
				want = 20
				if change == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					g.PutTriggersOnStack()
				}
			case "phased":
				g.PhaseOut(source.ID())
				want = 20
			case "phase-cycle":
				phased := g.PhaseOut(source.ID())
				g.PhaseIn(phased)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "lost-ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 24
			}
			g.ResolveStack()
			if a.Life() != want {
				t.Fatalf("life=%d want %d", a.Life(), want)
			}
			wantB := 20
			if change == "returned" {
				wantB = 22
			}
			if b.Life() != wantB {
				t.Fatalf("other life=%d want %d", b.Life(), wantB)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("original changed clone")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("pending clone lost entry condition")
			}
		})
	}
}

func TestRONEntryPresenceInvalid(t *testing.T) {
	for _, pair := range [][2]string{
		{"OnTheBattlefield", "OnTheBattlefield()"},
		{"OnTheBattlefield", "HasAbility(Flying)"},
		{"EnteringPermanentPassesFilter(OnTheBattlefield)", "EnteringPermanentPassesFilter"},
		{"EnteringPermanentPassesFilter(OnTheBattlefield)", "EnteringPermanentPassesFilter()"},
		{"EnteringPermanentPassesFilter(OnTheBattlefield)", "EnteringPermanentPassesFilter(OnTheBattlefield,OnTheBattlefield)"},
		{"SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent())"},
		{"SinglePermanent(ThisPermanent)", "SinglePermanent(HostPermanent)"},
		{"WhenAPermanentEntersTheBattlefield", "WhenAPermanentDies"},
		{"WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent))", "AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))"},
	} {
		bad := strings.Replace(entryPresenceRule, pair[0], pair[1], 1)
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+bad+`])`)); err == nil {
			t.Fatalf("invalid accepted: %s", bad)
		}
	}
}

func TestRONEntryPresenceAnimateDead(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Animate Dead")
			if _, err := Compile(record); err != nil {
				t.Fatal("complete full record rejected", err)
			}
			rules := record.Fields["Rules"]
			trigger := rules.Args[1]
			trigger.Args = append([]Node(nil), trigger.Args...)
			body := trigger.Args[2]
			body.Args = append([]Node(nil), body.Args...)
			body.Args[0].Args = append([]Node(nil), body.Args[0].Args[:2]...)
			trigger.Args[2] = body
			record.Fields = maps.Clone(record.Fields)
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{rules.Args[0], trigger}}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			for _, leave := range []bool{false, true} {
				g, a, b := ronGame()
				guest := mage.NewCreature("Guest", "", 3, 4)
				guest.SetOwner(b.PlayerID())
				b.AddToGraveyard(guest)
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("real entry condition did not queue")
				}
				if leave {
					g.BouncePermanentToHand(source)
				}
				g.ResolveStack()
				if (g.FindPermanent(guest.ID()) != nil) == leave {
					t.Fatal("source presence did not guard real grant and return")
				}
				if leave && len(b.Graveyard()) != 1 {
					t.Fatal("guard moved the graveyard card")
				}
			}
		})
	}
}

func TestRONEntryPresenceInstances(t *testing.T) {
	g, a, _ := ronGame()
	first := g.PutOnBattlefield(entryPresenceCard(t, entryPresenceRule), a.PlayerID())
	g.PutOnBattlefield(entryPresenceCard(t, entryPresenceRule), a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent entries did not queue")
	}
	g.RemoveFromBattlefield(first)
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("one source supplied another entry condition")
	}
}
