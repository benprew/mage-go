package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func activePlayerControlRecord(t *testing.T, body string) Node {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Active control",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostU],Rules:[SpellActions(`+body+`)])`), "active.ron")
	if err != nil {
		t.Fatal(err)
	}
	return nodes[0]
}

func TestRONActivePlayerControlTargets(t *testing.T) {
	for _, change := range []string{"normal", "active", "control", "phase", "cycle", "return", "copy", "payment"} {
		t.Run(change, func(t *testing.T) {
			n := activePlayerControlRecord(t, `Targeted([TargetPermanent(And([IsNonCreatureType(Wall),IsCardtype(Creature),ControlledByPlayer(TheActivePlayer)]))],ActionList([TapPermanent(Ref_TargetPermanent)]))`)
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("target declarations shared by factories")
			}
			g, a, b := ronGame()
			g.SetActivePlayerIndex(1)
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
			victimCard := mage.NewCreature("Victim", "", 2, 3)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if change != "payment" {
				a.ManaPool().Add(core.Blue, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{own.ID()}, 0); err == nil {
				t.Fatal("nonactive player's creature accepted")
			}
			err = g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{victim.ID()}, 0)
			if change == "payment" {
				if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || victim.Tapped {
					t.Fatal("failed payment changed game")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			legal := true
			switch change {
			case "active":
				g.SetActivePlayerIndex(0)
				legal = false
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: own.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				legal = false
			case "phase":
				g.PhaseOut(victim.ID())
				legal = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				legal = false
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(victim.ID()).Tapped {
				t.Fatal("live resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanentIncludingPhased(victim.ID()).Tapped != legal || game.FindPermanent(own.ID()).Tapped {
					t.Fatal("active target legality or clone isolation")
				}
			}
		})
	}
}

func TestRONActivePlayerControlCollections(t *testing.T) {
	for _, change := range []string{"normal", "active", "control", "phase", "cycle", "return", "copy"} {
		t.Run(change, func(t *testing.T) {
			n := activePlayerControlRecord(t, `ActionList([TapEachPermanent(And([IsCardtype(Creature),ControlledByPlayer(TheActivePlayer)]))])`)
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetActivePlayerIndex(1)
			own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
			victimCard := mage.NewCreature("Victim", "", 2, 3)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			wantOwn, wantVictim := false, true
			switch change {
			case "active":
				g.SetActivePlayerIndex(0)
				wantOwn, wantVictim = true, false
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: own.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				wantVictim = false
			case "phase":
				g.PhaseOut(victim.ID())
				wantVictim = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			late := g.PutOnBattlefield(mage.NewCreature("Late", "", 2, 3), b.PlayerID())
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanent(own.ID()).Tapped != wantOwn || game.FindPermanentIncludingPhased(victim.ID()).Tapped != wantVictim || game.FindPermanent(late.ID()).Tapped == wantOwn {
					t.Fatal("collection did not read current active player and members")
				}
			}
		})
	}
}

func TestRONActivePlayerControlInvalid(t *testing.T) {
	for _, filter := range []string{
		`ControlledByPlayer()`, `ControlledByPlayer(TheActivePlayer,You)`, `ControlledByPlayer(TheActivePlayer())`,
		`ControlledByPlayer(Ref_TargetPlayer)`, `ControlledByPlayer(Ref_TargetPermanent)`,
	} {
		for _, body := range []string{
			`Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)]))`,
			`ActionList([TapEachPermanent(` + filter + `)])`,
		} {
			if _, err := Compile(activePlayerControlRecord(t, body)); err == nil {
				t.Fatalf("accepted invalid declaration: %s", body)
			}
		}
	}
}

func TestRONActivePlayerControlNative(t *testing.T) {
	const valid = `Card(Name:"Active control",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([TapEachPermanent(ControlledByPlayer(TheActivePlayer))]))])`
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"active.ron": {Data: []byte(valid)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.CreateCard("Active control"); err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(strings.Replace(valid, "TheActivePlayer", "TheActivePlayer()", 1), "Active control", "Invalid control", 1)
	if err := LoadInto(registry, fstest.MapFS{"active.ron": {Data: []byte(valid + invalid)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONActivePlayerControlRepositoryClauses(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Nettling Imp", "Siren's Call"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				record := hostUntapRecord(t, set, name)
				if _, err := Compile(record); (err == nil) != (name == "Siren's Call") {
					t.Fatalf("complete record support: %v", err)
				}
				var predicate Node
				var find func(Node)
				find = func(n Node) {
					if n.Name == "ControlledByPlayer" && len(n.Args) == 1 && n.Args[0].Name == "TheActivePlayer" {
						predicate = n
					}
					for _, child := range n.Args {
						find(child)
					}
				}
				find(record.Fields["Rules"])
				if len(predicate.Args) != 1 || predicate.Args[0].Kind != Symbol || predicate.Args[0].Name != "TheActivePlayer" {
					t.Fatal("missing repository active-player clause")
				}
				n := activePlayerControlRecord(t, `ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(AnyPermanent)]))])`)
				delay := n.Fields["Rules"].Args[0].Args[0].Args[0].Args[0]
				tap := delay.Args[1].Args[0].Args[0]
				tap.Args[0] = predicate
				if name == "Nettling Imp" {
					n = activePlayerControlRecord(t, `Targeted([TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent)]))`)
					targeted := n.Fields["Rules"].Args[0].Args[0]
					targeted.Args[0].Args[0].Args[0] = predicate
				}
				factory, err := Compile(n)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
				c := factory()
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				a.ManaPool().Add(core.Blue, 1)
				var targets []uuid.UUID
				if name == "Nettling Imp" {
					g.SetActivePlayerIndex(1)
					targets = []uuid.UUID{other.ID()}
				}
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if name == "Nettling Imp" {
					if g.FindPermanent(own.ID()).Tapped || !g.FindPermanent(other.ID()).Tapped {
						t.Fatal("repository target predicate did not select the active player")
					}
					return
				}
				g.SetActivePlayerIndex(1)
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					game.PutTriggersOnStack()
					if game.StackSize() != 1 {
						t.Fatal("delayed clause not stacked")
					}
					game.ResolveStack()
					if game.FindPermanent(own.ID()).Tapped || !game.FindPermanent(other.ID()).Tapped {
						t.Fatal("delayed collection captured the earlier active player")
					}
				}
			})
		}
	}
}

func TestRONActivePlayerControlInventory(t *testing.T) {
	n := activePlayerControlRecord(t, `ActionList([TapEachPermanent(ControlledByPlayer(TheActivePlayer))])`)
	entries := map[string]*InventorySignature{}
	walkInventory(n, "Active control", "Card", "card", "active-control", false, entries)
	found := false
	for _, entry := range entries {
		if entry.Role == "unknown" {
			t.Fatalf("unknown inventory role: %s", entry.Name)
		}
		if entry.Name == "TheActivePlayer" {
			found = true
			if entry.Role != "player_reference" {
				t.Fatalf("active player role: %s", entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("active player reference missing from inventory")
	}
}
