package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func delusionTargetFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	n := hostUntapRecord(t, "LEG", "Glyph of Delusion")
	actions := &n.Fields["Rules"].Args[0].Args[0].Args[1].Args[0]
	actions.Args = actions.Args[:1]
	n.Fields["Name"] = Node{Kind: String, String: "Delusion target contract"}
	f, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Delusion target contract", f()); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRONTargetBlockHistoryGameplay(t *testing.T) {
	for _, change := range []string{"normal", "return-wall", "phase-wall", "cycle-wall", "wall-shroud", "return-attacker", "phase-attacker", "cycle-attacker", "attacker-shroud", "all-shroud", "control", "copy", "clone", "power"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			wallCard := repositoryCard(t, "LEB", "Wall of Wood")
			wallCard.SetOwner(b.PlayerID())
			wall := g.PutOnBattlefield(wallCard, b.PlayerID())
			attackerCard := mage.NewCreature("Attacker", "", 3, 5)
			attackerCard.SetOwner(a.PlayerID())
			attacker := g.PutOnBattlefield(attackerCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 5), a.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
			g.GetCombat().Reset()
			spell := delusionTargetFactory(t)()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			ids := []uuid.UUID{wall.ID(), attacker.ID()}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), ids, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID(), other.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("unrelated target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), ids, 0); err != nil {
				t.Fatal(err)
			}
			if attacker.Counters[core.Glyph] != 0 {
				t.Fatal("action ran before resolution")
			}
			switch change {
			case "return-wall":
				g.BouncePermanentToHand(wall)
				b.RemoveFromHand(wallCard.ID())
				g.PutOnBattlefield(wallCard, b.PlayerID())
			case "phase-wall":
				g.PhaseOut(wall.ID())
			case "cycle-wall":
				g.PhaseIn(g.PhaseOut(wall.ID()))
			case "wall-shroud":
				g.MutablePermanent(wall.ID()).GrantBaseAttr(core.Shroud)
			case "return-attacker":
				g.BouncePermanentToHand(attacker)
				a.RemoveFromHand(attackerCard.ID())
				g.PutOnBattlefield(attackerCard, a.PlayerID())
			case "phase-attacker":
				g.PhaseOut(attacker.ID())
			case "cycle-attacker":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "attacker-shroud":
				g.MutablePermanent(attacker.ID()).GrantBaseAttr(core.Shroud)
			case "all-shroud":
				g.MutablePermanent(attacker.ID()).GrantBaseAttr(core.Shroud)
				g.MutablePermanent(wall.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "power":
				g.MutablePermanent(attacker.ID()).AddCounter(core.P1P1, 2)
			}
			clone := g.Clone()
			if change == "clone" {
				clone.PhaseOut(attacker.ID())
			}
			g.ResolveStack()
			if clone.FindPermanentIncludingPhased(attacker.ID()).Counters[core.Glyph] != 0 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				want := 3
				switch change {
				case "return-attacker", "phase-attacker", "attacker-shroud", "all-shroud":
					want = 0
				case "copy":
					want = 6
				case "power":
					want = 5
				}
				if change == "clone" && game == clone {
					want = 0
				}
				if got := game.FindPermanentIncludingPhased(attacker.ID()).Counters[core.Glyph]; got != want {
					t.Fatalf("counters: got %d want %d", got, want)
				}
				if game.FindPermanent(other.ID()).Counters[core.Glyph] != 0 {
					t.Fatal("unrelated creature affected")
				}
			}
		})
	}
}

func TestRONTargetBlockHistoryInvalid(t *testing.T) {
	for _, expr := range []string{`WasBlockedByDefenderThisTurn`, `WasBlockedByDefenderThisTurn()`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent1,Ref_TargetPermanent1)`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent1())`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent)`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent2)`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent3)`, `WasBlockedByDefenderThisTurn(ThisPermanent)`, `WasBlockedByDefenderThisTurn(HostPermanent)`, `WasBlockedByDefenderThisTurn(You)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent),TargetPermanent(` + expr + `)],ActionList([TapPermanent(Ref_TargetPermanent2)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", expr)
		}
	}
	if _, err := Compile(hostUntapRecord(t, "LEG", "Glyph of Delusion")); err == nil || !strings.Contains(err.Error(), "supply one indefinite type modification") {
		t.Fatalf("full-card blocker: %v", err)
	}
}

func TestRONTargetBlockHistoryContexts(t *testing.T) {
	body := `Targeted([TargetPermanent(IsCreatureType(Wall)),TargetPermanent(And([IsCardtype(Creature),WasBlockedByDefenderThisTurn(Ref_TargetPermanent1)]))],ActionList([TapPermanent(Ref_TargetPermanent2)]))`
	for _, context := range []string{"activation", "modal", "entry", "opponent"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), b.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), a.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
			g.GetCombat().Reset()
			scopedBody := body
			if context == "opponent" {
				scopedBody = strings.Replace(scopedBody, `TargetPermanent(And([IsCardtype(Creature),WasBlockedByDefenderThisTurn(Ref_TargetPermanent1)]))`, `TargetPermanentOfAPlayersChoice(And([IsCardtype(Creature),WasBlockedByDefenderThisTurn(Ref_TargetPermanent1)]),Opponent)`, 1)
			}
			rules := `Activated(PayMana([]),` + scopedBody + `)`
			if context == "modal" {
				rules = `Activated(PayMana([]),Modal_ChooseOne([` + scopedBody + `,ActionList([GainLife(Integer(1))])]))`
			}
			if context == "entry" {
				rules = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + scopedBody + `)`
			}
			f := permanentScopeFactory(t, "Artifact", rules)
			a.queue = [][]uuid.UUID{{wall.ID()}, {attacker.ID()}}
			b.queue = [][]uuid.UUID{{attacker.ID()}}
			source := g.PutOnBattlefield(f(), a.PlayerID())
			if context == "entry" {
				g.PutTriggersOnStack()
			} else {
				var ids []uuid.UUID
				if context == "opponent" {
					ids = []uuid.UUID{wall.ID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if !g.FindPermanent(attacker.ID()).Tapped {
				t.Fatal("context lost block pair")
			}
		})
	}
}

func TestRONTargetBlockHistoryInventory(t *testing.T) {
	n := hostUntapRecord(t, "LEG", "Glyph of Delusion")
	entries := map[string]*InventorySignature{}
	walkInventory(n, "Glyph of Delusion", "Card", "card", "collection", false, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "Ref_TargetPermanent1" && strings.Contains(entry.Context, "WasBlockedByDefenderThisTurn") && entry.Role == "permanent_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("target history reference role missing")
	}
}
