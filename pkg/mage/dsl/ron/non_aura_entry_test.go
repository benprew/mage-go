package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func nonAuraEntryRecord(t *testing.T, delayed bool) Node {
	t.Helper()
	record := repositoryRecord(t, "LEG", "Takklemaggot", "non-aura.ron")
	rules := record.Fields["Rules"]
	death := rules.Args[2]
	branches := death.Args[1].Args[0].Args[1]
	fallback := branches.Args[2]
	if fallback.Args[0].Args[0].Args[1].Name != "EntersAsNonAuraEnchantment" {
		t.Fatal("source entry changed")
	}
	consumer := nameScopeNode(t, `PutCounters([ACounterOfTypeOnPermanent(PTCounter(+1,+1),ThePermanentPutOnTheBattlefieldThisWay)])`)
	if delayed {
		consumer = nameScopeNode(t, `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(+1,+1),ThePermanentPutOnTheBattlefieldThisWay)])]))`)
	}
	fallback.Args = []Node{fallback.Args[0], consumer}
	branches.Args[2] = fallback
	death.Args[1].Args[0].Args[1] = branches
	rules.Args = []Node{rules.Args[0], death}
	record.Fields["Rules"] = rules
	return record
}

func TestRONNonAuraEntry(t *testing.T) {
	for _, state := range []string{"normal", "stale", "copy", "clone", "phase", "return"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(nonAuraEntryRecord(t, true))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := factory()
			card.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.DestroyPermanent(host)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			if state == "stale" {
				c, _ := b.RemoveFromGraveyard(card.ID())
				b.AddToGraveyard(c)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if state == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()) != nil {
					t.Fatal("clone isolation failed")
				}
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if state == "stale" {
				if p != nil {
					t.Fatal("stale Aura returned")
				}
				return
			}
			if p == nil || p.HasSubType("Aura") || p.ControllerID() != a.PlayerID() || p.Card.Owner() != b.PlayerID() || p.AttachedTo != host.AttachedTo {
				t.Fatal("non-Aura return failed")
			}
			g.CheckStateBasedActions()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("non-Aura failed state actions")
			}
			if state == "phase" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if state == "return" {
				g.BouncePermanentToHand(p)
				c, _ := b.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(c, a.PlayerID())
				newHost := g.PutOnBattlefield(mage.NewCreature("Next", "", 2, 2), b.PlayerID())
				g.Attach(returned.ID(), newHost.ID())
			}
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				found := false
				for _, ref := range refs {
					found = found || ref.ID() == card.ID()
				}
				if !found {
					t.Fatalf("waiting query lost entrant: %v", refs)
				}
				if g.FindPermanent(card.ID()).Counters[core.P1P1] != 0 {
					t.Fatal("waiting query ran body")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			want := 1
			if state == "return" {
				want = 0
			}
			if g.FindPermanent(card.ID()).Counters[core.P1P1] != want {
				t.Fatal("saved entrant followed wrong visit")
			}
		})
	}
}

func TestRONNonAuraEntryInvalid(t *testing.T) {
	const entry = `ReturnDeadGuestGraveyardCardToBattlefield([EntersUnderPlayersControl(You),EntersAsNonAuraEnchantment])`
	for _, body := range []string{
		strings.Replace(entry, "EntersAsNonAuraEnchantment", "EntersAsNonAuraEnchantment()", 1),
		strings.Replace(entry, "EntersAsNonAuraEnchantment", "EntersAsNonAuraEnchantment(ThisPermanent)", 1),
		strings.Replace(entry, "EntersAsNonAuraEnchantment", "EntersAsNonAuraEnchantment,EntersAsNonAuraEnchantment", 1),
		`TapPermanent(ThePermanentPutOnTheBattlefieldThisWay),` + entry,
		`If(PermanentPassesFilter(ThisPermanent,IsCardtype(Enchantment)),[` + entry + `]),TapPermanent(ThePermanentPutOnTheBattlefieldThisWay)`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([`+body+`]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("invalid entry accepted: %s", body)
		}
	}
	node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+entry+`]))])`)
	if _, err := Compile(node); err == nil {
		t.Fatal("entry without Aura death accepted")
	}
}

func TestRONNonAuraEntryClearsResult(t *testing.T) {
	node := nonAuraEntryRecord(t, false)
	rules := node.Fields["Rules"]
	fallback := rules.Args[1].Args[1].Args[0].Args[1].Args[2]
	second := fallback.Args[0]
	second.Position.Column++
	fallback.Args = []Node{fallback.Args[0], second, fallback.Args[1]}
	rules.Args[1].Args[1].Args[0].Args[1].Args[2] = fallback
	node.Fields["Rules"] = rules
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	hostCard := mage.NewCreature("Host", "", 2, 2)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(p.ID(), host.ID())
	g.DestroyPermanent(host)
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	g.ResolveStack()
	p = g.FindPermanent(card.ID())
	if p == nil || p.HasSubType("Aura") || p.Counters[core.P1P1] != 0 {
		t.Fatal("failed second return retained the first result")
	}
}

func TestRONNonAuraEntryCopiedPhysicalCard(t *testing.T) {
	factory, err := Compile(nonAuraEntryRecord(t, false))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.SetPlayerAt(0, &ronEntryCopyPlayer{Player: a, accept: true})
	hostCard := mage.NewCreature("Host", "", 2, 2)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	modelCard := factory()
	modelCard.SetOwner(b.PlayerID())
	model := g.PutOnBattlefield(modelCard, a.PlayerID())
	g.Attach(model.ID(), host.ID())
	plan, err := mage.NewCardDefinitionPlan(func() mage.Card {
		return mage.NewArtifact("Physical", "", mage.WithAbility(mage.MayCopyAsEnters(mage.IsEnchantment)))
	})
	if err != nil {
		t.Fatal(err)
	}
	physical, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	physical.SetOwner(b.PlayerID())
	copied := g.PutOnBattlefield(physical, a.PlayerID())
	if copied.Name() != "Takklemaggot" {
		t.Fatal("entry did not copy Aura")
	}
	g.Attach(copied.ID(), host.ID())
	g.BouncePermanentToHand(model)
	g.DestroyPermanent(host)
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	g.ResolveStack()
	p := g.FindPermanent(physical.ID())
	if p == nil || p.Name() != "Physical" || !p.HasType(core.TypeEnchantment) || p.HasType(core.TypeArtifact) || p.HasSubType("Aura") || p.Counters[core.P1P1] != 1 || p.ControllerID() != a.PlayerID() || p.Card.Owner() != b.PlayerID() {
		t.Fatal("copied Aura ability did not return physical card as a non-Aura enchantment")
	}
}
