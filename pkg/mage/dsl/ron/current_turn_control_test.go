package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCurrentTurnControlTargets(t *testing.T) {
	for _, change := range []string{"normal", "regain", "phase", "cycle", "return", "copy", "payment", "clone-turn"} {
		t.Run(change, func(t *testing.T) {
			n := activePlayerControlRecord(t, `Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByPlayer(TheActivePlayer),ControlledSinceBeginningOfTurn]))],ActionList([TapPermanent(Ref_TargetPermanent)]))`)
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declaration")
			}
			g, a, b := ronGame()
			card := mage.NewCreature("Old", "", 2, 3)
			old := g.PutOnBattlefield(card, b.PlayerID())
			g.SetActivePlayerIndex(1)
			g.SetTurn(2)
			fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 2, 3), b.PlayerID())
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if change != "payment" {
				a.ManaPool().Add(core.Blue, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{fresh.ID()}, 0); err == nil {
				t.Fatal("new entry accepted")
			}
			err = g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{old.ID()}, 0)
			if change == "payment" {
				if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || old.Tapped {
					t.Fatal("payment rollback")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			legal := true
			switch change {
			case "regain":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: old.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: old.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				legal = false
			case "phase":
				g.PhaseOut(old.ID())
				legal = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(old.ID()))
			case "return":
				g.RemoveFromBattlefield(old)
				g.PutOnBattlefield(card, b.PlayerID())
				legal = false
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if change == "clone-turn" {
				clone.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: old.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: old.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(old.ID()).Tapped {
				t.Fatal("live resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				want := legal && (change != "clone-turn" || game != clone)
				if game.FindPermanentIncludingPhased(old.ID()).Tapped != want || game.FindPermanent(fresh.ID()).Tapped {
					t.Fatal("history target recheck")
				}
			}
		})
	}
}

func TestRONCurrentTurnControlInvalid(t *testing.T) {
	for _, filter := range []string{`ControlledSinceBeginningOfTurn()`, `ControlledSinceBeginningOfTurn(You)`, `NotControlledSinceBeginningOfTurn()`, `ExceptFor()`, `ExceptFor(NotControlledSinceBeginningOfTurn())`, `ExceptFor(NotControlledSinceBeginningOfTurn,IsTapped)`, `ExceptFor(ControlledSinceBeginningOfTurn)`, `ExceptFor(IsTapped)`} {
		for _, body := range []string{`Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)]))`, `ActionList([TapEachPermanent(` + filter + `)])`} {
			if _, err := Compile(activePlayerControlRecord(t, body)); err == nil {
				t.Fatalf("accepted %s", body)
			}
		}
	}
}

func TestRONCurrentTurnControlNative(t *testing.T) {
	const valid = `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([TapEachPermanent(ExceptFor(NotControlledSinceBeginningOfTurn))]))])`
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"history.ron": {Data: []byte(valid)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(strings.Replace(valid, "History", "Invalid", 1), "NotControlledSinceBeginningOfTurn", "NotControlledSinceBeginningOfTurn()", 1)
	if err := LoadInto(registry, fstest.MapFS{"history.ron": {Data: []byte(valid + invalid)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONCurrentTurnControlSiren(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, change := range []string{"normal", "regain", "phase", "cycle", "return", "regenerate", "copy", "own-turn", "late-cast", "payment"} {
			t.Run(set+"/"+change, func(t *testing.T) {
				c := repositoryCard(t, set, "Siren's Call")
				g, a, b := ronGame()
				oldCard := mage.NewCreature("Old", "", 2, 3)
				oldCard.SetOwner(b.PlayerID())
				old := g.PutOnBattlefield(oldCard, b.PlayerID())
				old.Tapped = true
				peer := g.PutOnBattlefield(mage.NewCreature("Peer", "", 2, 3), b.PlayerID())
				peer.Tapped = true
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 3), b.PlayerID())
				wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 3, mage.WithSubTypes("Wall"), mage.WithKeyword(core.Defender)), b.PlayerID())
				own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 3), a.PlayerID())
				g.SetActivePlayerIndex(1)
				g.SetTurn(2)
				g.SetStep(core.PrecombatMain)
				fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 2, 3), b.PlayerID())
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				if change != "payment" {
					a.ManaPool().Add(core.Blue, 1)
				}
				if change == "own-turn" {
					g.SetActivePlayerIndex(0)
				}
				if change == "late-cast" {
					g.SetStep(core.PostcombatMain)
				}
				err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0)
				if change == "own-turn" || change == "late-cast" || change == "payment" {
					if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || (change != "payment" && a.ManaPool().TotalMana() != 1) {
						t.Fatal("casting or payment rollback")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveStack()
				g.RunStepWithPriority(core.DeclareAttackers)
				if !g.GetCombat().IsAttacking(attacker.ID()) {
					t.Fatal("complete spell did not require able creature to attack")
				}
				dies := true
				switch change {
				case "regain":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: old.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: old.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					dies = false
				case "phase":
					g.PhaseOut(old.ID())
					dies = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(old.ID()))
				case "return":
					g.RemoveFromBattlefield(old)
					old = g.PutOnBattlefield(oldCard, b.PlayerID())
					dies = false
				case "regenerate":
					g.AddRegenerationShield(old.ID())
					dies = false
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					game.PutTriggersOnStack()
					expected := 1
					if change == "copy" {
						expected = 2
					}
					if game.StackSize() != expected {
						t.Fatalf("delays: %d", game.StackSize())
					}
					game.ResolveStack()
					if (game.FindPermanentIncludingPhased(old.ID()) == nil) != dies {
						t.Fatal("destruction history or replacement")
					}
					if game.FindPermanent(peer.ID()) != nil {
						t.Fatal("second eligible batch member survived")
					}
					for _, id := range []uuid.UUID{attacker.ID(), wall.ID(), fresh.ID(), own.ID()} {
						if game.FindPermanent(id) == nil {
							t.Fatal("excluded creature destroyed")
						}
					}
				}
			})
		}
	}
}

func TestRONCurrentTurnControlImpClauses(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := hostUntapRecord(t, set, "Nettling Imp")
			if _, err := Compile(record); err != nil {
				t.Fatalf("complete Imp record: %v", err)
			}
			filter := record.Fields["Rules"].Args[0].Args[1].Args[0].Args[0].Args[0]
			n := activePlayerControlRecord(t, `Targeted([TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent)]))`)
			n.Fields["Rules"].Args[0].Args[0].Args[0].Args[0].Args[0] = filter
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			old := g.PutOnBattlefield(mage.NewCreature("Old", "", 2, 3), b.PlayerID())
			g.SetActivePlayerIndex(1)
			g.SetTurn(2)
			fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 2, 3), b.PlayerID())
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{fresh.ID()}, 0); err == nil {
				t.Fatal("exact target clause accepted new creature")
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{old.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !g.FindPermanent(old.ID()).Tapped || g.FindPermanent(fresh.ID()).Tapped {
				t.Fatal("exact repository target clause")
			}
		})
	}
}

func TestRONCurrentTurnControlCollections(t *testing.T) {
	for _, filter := range []string{`ControlledSinceBeginningOfTurn`, `NotControlledSinceBeginningOfTurn`, `ExceptFor(NotControlledSinceBeginningOfTurn)`} {
		t.Run(filter, func(t *testing.T) {
			factory, err := Compile(activePlayerControlRecord(t, `ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([TapEachPermanent(And([IsCardtype(Creature),`+filter+`]))]))])`))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			old := g.PutOnBattlefield(mage.NewCreature("Old", "", 2, 3), a.PlayerID())
			g.SetActivePlayerIndex(1)
			g.SetTurn(2)
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 2, 3), b.PlayerID())
			phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 2, 3), a.PlayerID())
			g.PhaseOut(phased.ID())
			clone := g.Clone()
			clone.SetTurn(3)
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				positive := filter != `NotControlledSinceBeginningOfTurn`
				if game.FindPermanent(old.ID()).Tapped != positive || game.FindPermanent(fresh.ID()).Tapped != (positive == (game == clone)) || game.FindPermanentIncludingPhased(phased.ID()).Tapped {
					t.Fatal("delayed body did not read current history")
				}
			}
		})
	}
}

func TestRONCurrentTurnControlInventory(t *testing.T) {
	n := activePlayerControlRecord(t, `ActionList([TapEachPermanent(ExceptFor(NotControlledSinceBeginningOfTurn))])`)
	entries := map[string]*InventorySignature{}
	walkInventory(n, "History", "Card", "card", "history", false, entries)
	found := false
	for _, entry := range entries {
		if entry.Role == "unknown" {
			t.Fatalf("unknown role for %s", entry.Name)
		}
		if entry.Name == "NotControlledSinceBeginningOfTurn" {
			found = true
			if entry.Role != "permanent_filter" {
				t.Fatalf("history role: %s", entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing history predicate")
	}
}
