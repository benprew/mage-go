package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONDepartedAuraCard(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([ReturnDeadGuestGraveyardCardToHand]))])`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	hostCard := mage.NewCreature("Host", "", 2, 2)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DestroyPermanent(host)
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() || len(b.Hand()) != 0 {
		t.Fatal("did not return exact Aura to owner")
	}
}

func TestRONDepartedAuraCardInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([ReturnDeadGuestGraveyardCardToHand]))`,
		`TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([ReturnDeadGuestGraveyardCardToHand]))`,
		`EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([ReturnDeadGuestGraveyardCardToHand()]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "probe.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid guest return accepted")
		}
	}
}

func TestRONDepartedAuraCardRecordPrerequisites(t *testing.T) {
	for _, name := range []string{"Puppet Master", "Takklemaggot"} {
		t.Run(name, func(t *testing.T) {
			record := repositoryRecord(t, "LEG", name, "guest.ron")
			if _, err := Compile(record); err != nil {
				t.Fatalf("unexpected complete-record result: %v", err)
			}
			rules := record.Fields["Rules"]
			death := rules.Args[len(rules.Args)-1]
			if death.Args[0].Name != "WhenAPermanentDies" {
				t.Fatal("missing host death event")
			}
			probe, err := Parse(strings.NewReader(`Card(Name:"Body",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([ReturnDeadGuestGraveyardCardToHand]))])`), "body.ron")
			if err != nil {
				t.Fatal(err)
			}
			body := probe[0].Fields["Rules"].Args[0].Args[1]
			if name == "Puppet Master" {
				action := death.Args[1].Args[0].Args[1].Args[1].Args[1].Args[1].Args[0]
				if action.Name != "ReturnDeadGuestGraveyardCardToHand" {
					t.Fatalf("unexpected Puppet Master body: %s", action.Name)
				}
				body.Args = []Node{{Kind: List, Args: []Node{action}}}
			}
			death.Args[1] = body
			rules.Args = []Node{rules.Args[0], death}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEA", "Wall of Wood")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.DestroyPermanent(host)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() {
				t.Fatal("record event did not supply departed Aura")
			}
		})
	}
}
