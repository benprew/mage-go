package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryDeathExileDisintegrate(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 3), b.PlayerID())
			card := repositoryCard(t, set, "Disintegrate")
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 3); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.CheckStateBasedActions()
			if g.FindExiledCard(p.Card.ID()) == nil || g.CreatureDeaths() != 0 {
				t.Fatal("lethal damage did not exile")
			}
		})
	}
}

func TestRONTemporaryDeathExileInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateReplaceWouldPutIntoGraveyardUntil()`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[],UntilEndOfTurn)`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[ExileItInstead()],UntilEndOfTurn)`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[ExileItInstead],UntilEndOfCombat)`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(Ref_TargetPermanent)),[ExileItInstead],UntilEndOfTurn)`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(AnyPermanent),[ExileItInstead],UntilEndOfTurn)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONTemporaryDeathExileGameplay(t *testing.T) {
	for _, scenario := range []string{"zero-damage", "nonlethal", "prevented", "player", "walker", "illegal", "phase-before", "phase-cycle", "return-before", "return-after", "expiry", "sacrifice", "zero-toughness", "control", "regeneration", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 4), b.PlayerID())
			p.Card.SetOwner(b.PlayerID())
			target := p.ID()
			amount := 1
			if scenario == "zero-damage" {
				amount = 0
			}
			if scenario == "player" {
				target = b.PlayerID()
			}
			if scenario == "walker" {
				p = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID())
				p.Card.SetOwner(b.PlayerID())
				target = p.ID()
			}
			card := repositoryCard(t, "LEB", "Disintegrate")
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, amount); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Red, amount+1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, amount); err != nil {
				t.Fatal(err)
			}
			if p.Damage != 0 {
				t.Fatal("damage before resolution")
			}
			switch scenario {
			case "prevented":
				g.AddPreventionShield(p.ID(), 1)
			case "illegal":
				p.GrantBaseAttr(core.Shroud)
			case "phase-before":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return-before":
				g.BouncePermanentToHand(p)
				p = g.PutOnBattlefield(p.Card, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			g.ResolveStack()
			if scenario == "player" {
				if b.Life() != 19 {
					t.Fatal("player damage")
				}
				g.DestroyPermanent(p)
				if g.FindExiledCard(p.Card.ID()) != nil {
					t.Fatal("player projected to permanent")
				}
				return
			}
			if scenario == "phase-before" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				p = g.FindPermanent(p.ID())
			}
			switch scenario {
			case "return-after":
				g.BouncePermanentToHand(p)
				p = g.PutOnBattlefield(p.Card, b.PlayerID())
			case "expiry":
				g.RunStepWithPriority(core.Cleanup)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			}
			if scenario == "nonlethal" {
				if g.FindPermanent(p.ID()).Damage != 1 {
					t.Fatal("nonlethal damage")
				}
			}
			if scenario == "copy" && g.FindPermanent(p.ID()).Damage != 2 {
				t.Fatal("copy lost X or recipient")
			}
			if scenario == "zero-damage" || scenario == "prevented" {
				if g.FindPermanent(p.ID()).Damage != 0 {
					t.Fatal("zero or prevented damage")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				current := game.FindPermanent(p.ID())
				switch scenario {
				case "sacrifice":
					game.DoSacrifice(current)
				case "zero-toughness":
					game.AddCountersWithReplacement(current, core.M1M1, 4, uuid.Nil, false)
					game.CheckStateBasedActions()
				default:
					game.DestroyPermanent(current)
				}
				want := scenario != "walker" && scenario != "illegal" && scenario != "phase-before" && scenario != "return-before" && scenario != "return-after" && scenario != "expiry"
				if (game.FindExiledCard(p.Card.ID()) != nil) != want {
					t.Fatalf("exile=%t want=%t", game.FindExiledCard(p.Card.ID()) != nil, want)
				}
				if want && game.CreatureDeaths() != 0 {
					t.Fatal("replacement triggered death")
				}
			}
		})
	}
}

func TestRONTemporaryDeathExileInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 4), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 4), b.PlayerID())
	for i, player := range []*mage.BasePlayer{a, b} {
		card := repositoryCard(t, "LEB", "Disintegrate")
		player.AddToHand(card)
		player.ManaPool().Add(core.Red, 1)
		target := first.ID()
		if i == 1 {
			g.ResolveStack()
			g.SetActivePlayerIndex(1)
			target = second.ID()
		}
		if err := g.CastSpellByID(player.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	g.DestroyPermanents([]*mage.Permanent{first, second})
	if g.FindExiledCard(first.Card.ID()) == nil || g.FindExiledCard(second.Card.ID()) == nil || g.CreatureDeaths() != 0 {
		t.Fatal("independent spell targets")
	}
	if clone.FindPermanent(first.ID()) == nil || clone.FindPermanent(second.ID()) == nil {
		t.Fatal("clone shared departures")
	}
	clone.ResolveStack()
	clone.DestroyPermanents([]*mage.Permanent{clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID())})
	if clone.FindExiledCard(first.Card.ID()) == nil || clone.FindExiledCard(second.Card.ID()) == nil {
		t.Fatal("clone lost pending targets")
	}
}

func TestRONTemporaryDeathExileActionContext(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[ExileItInstead],UntilEndOfTurn)]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	p.Card.SetOwner(a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !p.Tapped {
		t.Fatal("activation payment")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	g.DoSacrifice(g.FindPermanent(p.ID()))
	if g.FindExiledCard(p.Card.ID()) == nil || len(b.Graveyard()) != 0 {
		t.Fatal("source context or phase cycle")
	}
}

func TestRONTemporaryDeathExileSourceLifetime(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(Ref_TargetPermanent)),[ExileItInstead],UntilEndOfTurn)])))`)
	for _, change := range []string{"departed", "returned", "phased", "removed", "retarget-copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 3), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 3), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "retarget-copy" {
				phased := g.PhaseOut(first.ID())
				copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
				if copied == nil {
					t.Fatal("ability copy")
				}
				if len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
					t.Fatal("copy retarget")
				}
				g.PhaseIn(phased)
			}
			g.ResolveStack()
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				if change == "returned" {
					g.PutOnBattlefield(source.Card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			g.DestroyPermanents([]*mage.Permanent{first, second})
			if g.FindExiledCard(first.Card.ID()) == nil || (g.FindExiledCard(second.Card.ID()) != nil) != (change == "retarget-copy") {
				t.Fatal("rule lifetime or independent copy target")
			}
		})
	}
}

func TestRONTemporaryDeathExileInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[ExileItInstead],UntilEndOfTurn)`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string]*InventorySignature)
	walkInventory(nodes[0], "Rule", "action", "action", "action", true, entries)
	for name, role := range map[string]string{"APermanentWouldDie": "temporary_death_replacement_event", "SinglePermanent": "permanent_collection", "ThisPermanent": "permanent_reference", "ExileItInstead": "replacement_operation", "UntilEndOfTurn": "duration"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name {
				found = true
				if entry.Role != role {
					t.Fatalf("%s role=%s want=%s", name, entry.Role, role)
				}
			}
		}
		if !found {
			t.Fatal("missing", name)
		}
	}
}

func TestRONTemporaryDeathExileDelayedContext(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(SinglePermanent(ThisPermanent)),[ExileItInstead],UntilEndOfTurn)]))]))`)
	for _, state := range []string{"present", "returned"} {
		t.Run(state, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			p.Card.SetOwner(a.PlayerID())
			g.ResolveStack()
			if state == "returned" {
				g.BouncePermanentToHand(p)
				p = g.PutOnBattlefield(p.Card, a.PlayerID())
			}
			g.RunStepWithPriority(core.EndStep)
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			if (g.FindExiledCard(p.Card.ID()) != nil) != (state == "present") {
				t.Fatal("triggered delay followed a returned source")
			}
		})
	}
}
