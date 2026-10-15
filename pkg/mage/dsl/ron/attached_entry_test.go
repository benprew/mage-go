package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttachedEntryGuest(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Takklemaggot", "guest.ron")
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
	rules := record.Fields["Rules"]
	death := rules.Args[2]
	entry := death.Args[1].Args[0].Args[1].Args[1].Args[0]
	if entry.Name != "ReturnDeadGuestGraveyardCardToBattlefield" {
		t.Fatalf("unexpected action %s", entry.Name)
	}
	probe := nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([ChooseAPermanent(IsCardtype(Creature))]))])`)
	body := probe.Fields["Rules"].Args[1].Args[1]
	body.Args[0].Args = append(body.Args[0].Args, entry)
	death.Args[1] = body
	rules.Args = []Node{rules.Args[0], death}
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"normal", "clone", "copy", "new-visit", "phased-host"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			firstCard := mage.NewCreature("First", "", 2, 2)
			firstCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, b.PlayerID())
			next := g.PutOnBattlefield(mage.NewCreature("Next", "", 2, 2), b.PlayerID())
			card := factory()
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(p.ID(), first.ID())
			g.DestroyPermanent(first)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			if state == "new-visit" {
				c, _ := b.RemoveFromGraveyard(card.ID())
				b.AddToGraveyard(c)
			}
			if state == "phased-host" {
				g.PhaseOut(next.ID())
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if state == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()) != nil {
					t.Fatal("clone changed original")
				}
			}
			g.ResolveStack()
			got := g.FindPermanent(card.ID())
			if state == "new-visit" || state == "phased-host" {
				if got != nil {
					t.Fatal("invalid visit returned")
				}
				return
			}
			if got == nil || got.AttachedTo != next.ID() || got.ControllerID() != a.PlayerID() || got.Card.Owner() != b.PlayerID() {
				t.Fatal("departed Aura entry failed")
			}
		})
	}
}

func TestRONAttachedEntryCoffin(t *testing.T) {
	record := repositoryRecord(t, "ATQ", "Tawnos's Coffin", "coffin.ron")
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
	var entry Node
	var walk func(Node)
	walk = func(n Node) {
		if n.Name == "PutEachExiledCardOntoTheBattlefield" {
			entry = n
		}
		for _, arg := range n.Args {
			walk(arg)
		}
	}
	walk(record.Fields["Rules"])
	if entry.Args[1].Args[1].Name != "EntersAttachedToPermanent" {
		t.Fatal("missing entry option")
	}
	probe := nameScopeNode(t, `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([ChooseAPermanent(IsCardtype(Creature)),Exile([Permanent(TheChosenPermanent),EachPermanent(And([IsEnchantmentType(Aura),AttachedToPermanent(TheChosenPermanent)]))],[]),PutGraveyardCardOntoBattlefield(Ref_TargetGraveyardCard,[EntersNormally])])))])`)
	rules := probe.Fields["Rules"]
	actions := rules.Args[0].Args[1].Args[1].Args[0]
	actions.Args = append(actions.Args, entry)
	rules.Args[0].Args[1].Args[1].Args[0] = actions
	probe.Fields["Rules"] = rules
	factory, err := Compile(probe)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	oldCard := mage.NewCreature("Old", "", 2, 2)
	oldCard.SetOwner(a.PlayerID())
	old := g.PutOnBattlefield(oldCard, a.PlayerID())
	nextCard := mage.NewCreature("Next", "", 2, 2)
	nextCard.SetOwner(b.PlayerID())
	b.AddToGraveyard(nextCard)
	card := mage.NewAura("Aura", "", mage.WithCastTarget(mage.TargetCreature()))
	card.SetOwner(b.PlayerID())
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), old.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{nextCard.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	got := g.FindPermanent(card.ID())
	if got == nil || got.AttachedTo != nextCard.ID() || got.ControllerID() != b.PlayerID() || g.FindExiledCard(old.ID()) == nil {
		t.Fatal("grouped exile attachment failed")
	}
}

func TestRONAttachedEntryInvalid(t *testing.T) {
	const valid = `ReturnDeadGuestGraveyardCardToBattlefield([EntersUnderPlayersControl(You),EntersAttachedToPermanent(TheChosenPermanent)])`
	for _, body := range []string{
		strings.Replace(valid, "TheChosenPermanent", "You", 1),
		strings.Replace(valid, "TheChosenPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "EntersAttachedToPermanent(TheChosenPermanent)", "EntersAttachedToPermanent", 1),
		strings.Replace(valid, "EntersAttachedToPermanent(TheChosenPermanent)", "EntersAttachedToPermanent(TheChosenPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "EntersUnderPlayersControl(You)", "EntersUnderPlayersControl(You),EntersUnderPlayersControl(You)", 1),
		strings.Replace(valid, "EntersAttachedToPermanent(TheChosenPermanent)", "EntersAsNonAuraEnchantment(ThisPermanent)", 1),
		`ReturnDeadGuestGraveyardCardToBattlefield`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([ChooseAPermanent(IsCardtype(Creature)),`+body+`]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("invalid entry accepted: %s", body)
		}
	}
	node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+valid+`]))])`)
	if _, err := Compile(node); err == nil {
		t.Fatal("wrong event accepted")
	}
}

func TestRONAttachedEntryLinkedInvalid(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([Exile([TheTopCardOfLibrary],[])]))`
	for _, option := range []string{
		`EntersAttachedToPermanent`, `EntersAttachedToPermanent()`,
		`EntersAttachedToPermanent(You)`, `EntersAttachedToPermanent(TheChosenPermanent)`,
		`EntersAttachedToPermanent(ThisPermanent()),EntersAttachedToPermanent(ThisPermanent)`,
		`EntersAttachedToPermanent(ThisPermanent,ThisPermanent)`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+producer+`,Activated(PayMana([]),ActionList([PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[EntersUnderOwnersControl,`+option+`])]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("invalid exile entry accepted %s", option)
		}
	}
}

func TestRONAttachedEntryInventory(t *testing.T) {
	src := `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([ChooseAPermanent(IsCardtype(Creature)),ReturnDeadGuestGraveyardCardToBattlefield([EntersUnderPlayersControl(You),EntersAttachedToPermanent(TheChosenPermanent)])]))])`
	report, err := InspectFS(fstest.MapFS{"entry.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory failed: %v", err)
	}
	for name, role := range map[string]string{"ReturnDeadGuestGraveyardCardToBattlefield": "action", "EntersAttachedToPermanent": "entry_operation", "TheChosenPermanent": "permanent_reference"} {
		found := false
		for _, sig := range report.Signatures {
			found = found || sig.Name == name && sig.Role == role
		}
		if !found {
			t.Fatalf("missing role %s %s", name, role)
		}
	}
}

func TestRONAttachedEntryLinked(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Enchantment))],ActionList([Exile([Permanent(Ref_TargetPermanent)],[])]))),Activated(PayMana([]),ActionList([ChooseAPermanent(IsCardtype(Creature)),PutEachExiledCardOntoTheBattlefield(WasExiledWithPermanent(ThisPermanent),[EntersUnderOwnersControl,EntersAttachedToPermanent(TheChosenPermanent)])]))`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 2, mage.WithKeyword(core.Shroud)), b.PlayerID())
	card := mage.NewAura("Aura", "", mage.WithCastTarget(mage.TargetCreature()))
	card.SetOwner(b.PlayerID())
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{aura.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindExiledCard(card.ID()) == nil {
		t.Fatal("producer did not exile Aura")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	got := g.FindPermanent(card.ID())
	if got == nil || got.AttachedTo != host.ID() || got.ControllerID() != b.PlayerID() {
		t.Fatal("linked return failed to attach without targeting")
	}
}
