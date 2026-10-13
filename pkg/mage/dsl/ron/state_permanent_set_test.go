package ron

import (
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStatePermanentSetCity(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "ARN", "City in a Bottle"))
	if err != nil {
		t.Fatal(err)
	}
	for _, active := range []int{0, 1} {
		g, a, b := ronGame()
		g.SetActivePlayerIndex(active)
		card := mage.NewArtifact("Aladdin's Lamp", "")
		card.SetOwner(b.PlayerID())
		first := g.PutOnBattlefield(card, a.PlayerID())
		second := g.PutOnBattlefield(mage.NewLand("Desert"), b.PlayerID())
		token := g.PutOnBattlefield(mage.NewToken("Aladdin's Lamp", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
		mountain := g.PutOnBattlefield(mage.NewLand("Mountain"), a.PlayerID())
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		spell := mage.NewInstant("Aladdin's Lamp", "")
		a.AddToHand(spell)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
			t.Fatal("casting rule missing")
		}
		land := mage.NewLand("Desert")
		b.AddToHand(land)
		if err := g.PlayLand(b.PlayerID(), land.ID()); err == nil {
			t.Fatal("land rule missing")
		}
		late := g.PutOnBattlefield(mage.NewArtifact("Aladdin's Ring", ""), b.PlayerID())
		g.CheckStateTriggers()
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("state trigger repeated", g.StackSize())
		}
		g.ResolveTopOfStack()
		if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
			t.Fatal("captured controllers did not sacrifice")
		}
		if len(b.Graveyard()) != 2 {
			t.Fatal("owner destination lost")
		}
		for _, p := range []*mage.Permanent{source, token, mountain, late} {
			if g.FindPermanent(p.ID()) == nil {
				t.Fatal("uncaptured subject moved", p.Name())
			}
		}
		g.CheckStateTriggers()
		g.PutTriggersOnStack()
		g.ResolveTopOfStack()
		if g.FindPermanent(late.ID()) != nil {
			t.Fatal("later detection did not capture late entry")
		}
	}
}

func TestRONStatePermanentSetInvalid(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(AnyPermanent),ActionList([])),Activated(PayMana([]),ActionList([ControllersSacrificeEachPermanent(Trigger_ThosePermanents)]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(AnyPermanent),ActionList([ControllersSacrificeEachPermanent(Trigger_ThosePermanents())]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(ControlledByPlayer(ActionPlayer)),ActionList([]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(Other(ThisPermanent())),ActionList([]))`,
		`FromGraveyard(TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(AnyPermanent),ActionList([])))`,

		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield,ActionList([]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(),ActionList([]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(And([])),ActionList([]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(Other(Ref_TargetPermanent)),ActionList([]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(Trigger_ThosePermanents),ActionList([]))`,
		`TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(IsCardtype(Land)),ActionList([SacrificePermanent(Trigger_ThosePermanents)]))`,
		`Activated(PayMana([]),ActionList([ControllersSacrificeEachPermanent(Trigger_ThosePermanents)]))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		if registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid declaration published", rule)
		}
	}
}

func TestRONStatePermanentSetPending(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "ARN", "City in a Bottle"))
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"returned", "phased", "cycle", "control", "source-left", "source-phased", "clone", "copy", "counter"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := mage.NewArtifact("Aladdin's Lamp", "")
			card.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(card, b.PlayerID())
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			late := g.PutOnBattlefield(mage.NewArtifact("Aladdin's Ring", ""), b.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("pending capture repeated")
			}
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(first.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-left":
				g.BouncePermanentToHand(source)
			case "source-phased":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.PhaseOut(first.ID())
			}
			g.ResolveTopOfStack()
			remains := scenario == "returned" || scenario == "phased"
			if (g.FindPermanentIncludingPhased(first.ID()) != nil) != remains {
				t.Fatal("exact captured visit lost")
			}
			if (g.FindPermanent(late.ID()) != nil) != (scenario != "counter") {
				t.Fatal("late entry changed saved set")
			}
			clone.ResolveTopOfStack()
			if (clone.FindPermanentIncludingPhased(first.ID()) != nil) != (remains || scenario == "clone") {
				t.Fatal("clone lost capture")
			}
		})
	}
}

func TestRONStatePermanentSetInstancesAndInventory(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "City in a Bottle")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, active := range []int{0, 1} {
		g, a, b := ronGame()
		g.SetActivePlayerIndex(active)
		first := g.PutOnBattlefield(factory(), a.PlayerID())
		victim := g.PutOnBattlefield(mage.NewArtifact("Aladdin's Lamp", ""), b.PlayerID())
		second := g.PutOnBattlefield(factory(), b.PlayerID())
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatal("instances shared state suppression")
		}
		g.ResolveStack()
		if g.FindPermanent(first.ID()) != nil || (g.FindPermanent(second.ID()) != nil) != (active == 0) || g.FindPermanent(victim.ID()) != nil {
			t.Fatal("instances shared source exclusion or captured members", active, g.FindPermanent(first.ID()) != nil, g.FindPermanent(second.ID()) != nil, g.FindPermanent(victim.ID()) != nil)
		}
	}
	entries := map[string]*InventorySignature{}
	walkInventory(record, "City in a Bottle", "Card", "card", "collection", true, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "ThisPermanent" && entry.Role == "permanent_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("state source exclusion lost reference role")
	}
}

func TestRONStatePermanentSetDelay(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAnyNumberOfPermanentsAreOnTheBattlefield(IsCardtype(Land)),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ControllersSacrificeEachPermanent(Trigger_ThosePermanents)]))]))`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewLand("First"), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveTopOfStack()
	g.BouncePermanentToHand(source)
	late := g.PutOnBattlefield(mage.NewLand("Late"), b.PlayerID())
	for range 3 {
		g.WaitingDelayedPermanentSources()
		if g.FindPermanent(first.ID()) == nil {
			t.Fatal("waiting query executed body")
		}
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(late.ID()) == nil {
		t.Fatal("delay did not retain detected visits")
	}
	if clone.FindPermanent(first.ID()) == nil {
		t.Fatal("delay shared clone state")
	}
}
