package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEntryHistoryAnimateDead(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Animate Dead")
			if _, err := Compile(record); err != nil {
				t.Fatal("complete full record rejected", err)
			}
			grant := record.Fields["Rules"].Args[1].Args[2].Args[0].Args[0]
			filter := grant.Args[1].Args[1].Args[0].Args[0].Args[0]
			if filter.Name != "And" || filter.Args[0].Args[1].Name != "PutOntoBattlefieldByPermanent" {
				t.Fatal("real linked enchant filter changed")
			}
			probe := nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([TapEachPermanent(IsCardtype(Creature)),PutGraveyardCardOntoBattlefield(EnchantedGraveyardCard,[EntersUnderPlayersControl(You)]),TapEachPermanent(IsCardtype(Creature))]))])`)
			body := probe.Fields["Rules"].Args[1].Args[1].Args[0]
			body.Args[0].Args[0] = filter
			body.Args[2].Args[0] = filter
			record.Fields = maps.Clone(record.Fields)
			record.Fields["Rules"] = probe.Fields["Rules"]
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			guest := mage.NewCreature("Guest", "", 3, 4)
			guest.SetOwner(b.PlayerID())
			b.AddToGraveyard(guest)
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 2), b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			clone := g.Clone()
			if set == "LEA" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("trigger copy failed")
			}
			g.ResolveStack()
			if p := g.FindPermanent(guest.ID()); p == nil || !p.Tapped || p.ControllerID() != a.PlayerID() || g.FindPermanent(other.ID()).Tapped {
				t.Fatal("forward linked query did not select exactly the returned guest")
			}
			if clone.FindPermanent(guest.ID()) != nil {
				t.Fatal("pending clone shared entry")
			}
			clone.ResolveStack()
			if p := clone.FindPermanent(guest.ID()); p == nil || !p.Tapped {
				t.Fatal("pending clone lost entry history")
			}
		})
	}
}

func TestRONEntryHistoryInvalid(t *testing.T) {
	for _, predicate := range []string{`PutOntoBattlefieldByPermanent`, `PutOntoBattlefieldByPermanent()`, `PutOntoBattlefieldByPermanent(ThisPermanent())`, `PutOntoBattlefieldByPermanent(Ref_TargetPermanent)`, `PutOntoBattlefieldByPermanent(ThisPermanent,ThisPermanent)`} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),Activated(PayMana([]),ActionList([TapEachPermanent(` + predicate + `)]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("accepted %s", predicate)
		}
	}
}

func TestRONEntryHistoryContextsAndInventory(t *testing.T) {
	const predicate = `PutOntoBattlefieldByPermanent(ThisPermanent)`
	const prefix = `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),`
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[TapEachPermanent(` + predicate + `)])`,
		`Activated(TapEachPermanent(` + predicate + `),ActionList([]))`,
		`PermanentLayerEffect(` + predicate + `,[AdjustPT(1,1)])`,
	} {
		if _, err := Compile(nameScopeNode(t, prefix+rule+`])`)); err == nil {
			t.Fatalf("unsupported context accepted: %s", rule)
		}
	}
	node := nameScopeNode(t, prefix+`Activated(PayMana([]),ActionList([TapEachPermanent(`+predicate+`)]))])`)
	relation := node.Fields["Rules"].Args[1].Args[1].Args[0].Args[0].Args[0]
	for _, ctx := range []scope{{resolving: true}, {graveyardEnchant: true}, {graveyardEnchant: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileContextualCollection(relation, ctx); err == nil {
			t.Fatal("unsupported lexical context accepted")
		}
	}
	source := prefix + `Activated(PayMana([]),ActionList([TapEachPermanent(` + predicate + `)]))])`
	report, err := InspectFS(fstest.MapFS{"probe.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "ThisPermanent" && strings.Contains(sig.Context, "PutOntoBattlefieldByPermanent") {
			if sig.Role != "permanent_reference" {
				t.Fatalf("source role = %s", sig.Role)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing typed history source in inventory")
	}
}

func TestRONEntryHistoryDelayed(t *testing.T) {
	const source = `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantGraveyardCard(IsCardtype(Creature),AnyPlayer),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(PutOntoBattlefieldByPermanent(ThisPermanent))])),PutGraveyardCardOntoBattlefield(EnchantedGraveyardCard,[EntersUnderPlayersControl(You)])]))])`
	factory, err := Compile(nameScopeNode(t, source))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	guest := mage.NewCreature("Guest", "", 3, 4)
	guest.SetOwner(b.PlayerID())
	b.AddToGraveyard(guest)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || g.FindPermanent(guest.ID()) == nil {
		t.Fatal("projection did not return its host and lose its obsolete attachment")
	}
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || g.FindPermanent(guest.ID()).Tapped {
			t.Fatal("waiting query lost source or executed its body")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	g.RunStepWithPriority(core.EndStep)
	if !g.FindPermanent(guest.ID()).Tapped {
		t.Fatal("forward delayed query lost the departed source link")
	}
}
