package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGraveyardEnchantClause(t *testing.T) {
	record := chosenDiscardRecord(t, "LEA", "Animate Dead")
	record.Fields = maps.Clone(record.Fields)
	rules := record.Fields["Rules"]
	rules.Args = rules.Args[:1]
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := mage.NewCreature("Host", "", 2, 2, mage.WithKeyword(core.Shroud))
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.CheckStateBasedActions()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("real enchant clause did not retain attachment")
	}
}

func TestRONGraveyardEnchantRead(t *testing.T) {
	node := nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([PutGraveyardCardIntoHand(EnchantedGraveyardCard)]))])`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := mage.NewCreature("Host", "", 2, 2)
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	card := factory()
	card.SetOwner(a.PlayerID())
	if g.PutOnBattlefield(card, a.PlayerID()) == nil {
		t.Fatal("entry failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 1 || b.Hand()[0].ID() != host.ID() {
		t.Fatal("host was not returned to owner")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("Aura retained absent graveyard host")
	}
}

func TestRONGraveyardEnchantInvalid(t *testing.T) {
	for _, rule := range []string{
		`EnchantGraveyardCard`, `EnchantGraveyardCard()`, `EnchantGraveyardCard(IsCardtype(Creature))`,
		`EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer())`, `EnchantGraveyardCard(IsCardtype(Land),AnyPlayer)`,
		`EnchantGraveyardCard(IsCardtype(Creature),You)`, `EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),EnchantPermanent(IsCardtype(Creature))`,
	} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(source), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONGraveyardEnchantReprints(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Animate Dead")
			if _, err := Compile(record); err != nil {
				t.Fatal("complete Animate Dead record rejected", err)
			}
			record.Fields = maps.Clone(record.Fields)
			rules := record.Fields["Rules"]
			rules.Args = []Node{rules.Args[0]}
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Animate Dead enchant clause", factory()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRONGraveyardEnchantBattlefieldRead(t *testing.T) {
	node := nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([PutGraveyardCardOntoBattlefield(EnchantedGraveyardCard,[EntersUnderPlayersControl(You)])]))])`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := mage.NewCreature("Host", "", 2, 2)
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	aura := factory()
	aura.SetOwner(a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	pending := g.Clone()
	g.ResolveStack()
	returned := g.FindPermanent(host.ID())
	if returned == nil || returned.ControllerID() != a.PlayerID() || returned.Card.Owner() != b.PlayerID() || g.FindPermanent(aura.ID()) != nil {
		t.Fatal("graveyard host entry or unattached Aura state")
	}
	pending.ResolveStack()
	if pending.FindPermanent(host.ID()) == nil {
		t.Fatal("pending clone lost graveyard attachment")
	}
}

func TestRONGraveyardEnchantReferenceScope(t *testing.T) {
	for _, source := range []string{
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PutGraveyardCardIntoHand(EnchantedGraveyardCard)]))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([PutGraveyardCardIntoHand(EnchantedGraveyardCard)]))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),Activated(PayMana([]),ActionList([PutGraveyardCardIntoHand(EnchantedGraveyardCard())]))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer)])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer)])`,
	} {
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("accepted invalid scope: %s", source)
		}
	}
}

func TestRONGraveyardEnchantRemovalBoundary(t *testing.T) {
	source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Enchant(IsCardtype(Creature)))])]))])`
	if _, err := Compile(nameScopeNode(t, source)); err == nil {
		t.Fatal("accepted graveyard enchant removal through permanent-only adapter")
	}
}

func TestRONGraveyardEnchantInventory(t *testing.T) {
	source := `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer)])`
	report, err := InspectFS(fstest.MapFS{"probe.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, entry := range report.Signatures {
		found[entry.Role+"/"+entry.Name] = true
	}
	for _, key := range []string{"rule/EnchantGraveyardCard", "card_filter/IsCardtype", "player_filter/AnyPlayer"} {
		if !found[key] {
			t.Fatalf("missing syntax role %s", key)
		}
	}
}
