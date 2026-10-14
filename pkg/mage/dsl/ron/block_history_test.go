package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// The fixture tests a shared clause while the full card remains unsupported.
func glyphHistoryFactory(t *testing.T, name string) mage.CardFactory {
	t.Helper()
	n := hostUntapRecord(t, "LEG", name)
	var target, action Node
	for _, rule := range n.Fields["Rules"].Args {
		if rule.Name != "SpellActions" {
			continue
		}
		targeted := rule.Args[0]
		target = targeted.Args[0]
		parts := targeted.Args[1].Args[0].Args
		if name == "Glyph of Doom" {
			parts = parts[0].Args[1].Args[0].Args
		}
		action = parts[0]
	}
	n.Fields["Name"] = Node{Kind: String, String: "Blocking history contract"}
	n.Fields["Rules"] = Node{Kind: List, Args: []Node{{Kind: Tuple, Name: "SpellActions", Args: []Node{{Kind: Tuple, Name: "Targeted", Args: []Node{target, {Kind: Tuple, Name: "ActionList", Args: []Node{{Kind: List, Args: []Node{action}}}}}}}}}}
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(name, factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONBlockHistoryRealActions(t *testing.T) {
	for _, name := range []string{"Glyph of Doom", "Glyph of Reincarnation"} {
		for _, change := range []string{"none", "return-attacker", "phase-attacker", "return-wall", "phase-wall", "control", "clone", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				factory := glyphHistoryFactory(t, name)
				g, a, b := ronGame()
				wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), b.PlayerID())
				attackerCard := repositoryCard(t, "LEB", "Llanowar Elves")
				attackerCard.SetOwner(a.PlayerID())
				attacker := g.PutOnBattlefield(attackerCard, a.PlayerID())
				spared := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
				g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
				g.GetCombat().Reset()
				g.RunStepWithPriority(core.PrecombatMain)
				spell := factory()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(core.Black, 1)
				a.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				switch change {
				case "return-attacker":
					g.BouncePermanentToHand(g.FindPermanent(attacker.ID()))
					a.RemoveFromHand(attackerCard.ID())
					g.PutOnBattlefield(attackerCard, a.PlayerID())
				case "phase-attacker":
					g.PhaseOut(attacker.ID())
				case "return-wall":
					card := wall.Card
					g.BouncePermanentToHand(g.FindPermanent(wall.ID()))
					b.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "phase-wall":
					g.PhaseOut(wall.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.AddRegenerationShield(attacker.ID())
				g.ResolveStack()
				destroyed := change == "none" || change == "control" || change == "clone" || change == "copy"
				if name == "Glyph of Doom" && change != "copy" {
					destroyed = false
				}
				if (g.FindPermanentIncludingPhased(attacker.ID()) == nil) != destroyed || g.FindPermanent(spared.ID()) == nil {
					t.Fatal("wrong history destruction or regeneration")
				}
				if clone.FindPermanent(attacker.ID()) == nil {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				if clone.FindPermanent(attacker.ID()) != nil {
					t.Fatal("clone lost history or target")
				}
			})
		}
	}
}

func TestRONBlockHistoryInvalid(t *testing.T) {
	for _, filter := range []string{`WasBlockedByDefenderThisTurn`, `WasBlockedByDefenderThisTurn()`, `WasBlockedByDefenderThisTurn(ThisPermanent,ThisPermanent)`, `WasBlockedByDefenderThisTurn(Ref_TargetPermanent)`, `WasBlockedByDefenderThisTurn(HostPermanent)`, `WasBlockedByDefenderThisTurn(You)`, `WasBlockedByDefenderThisTurn(ThisPermanent())`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([DestroyEachPermanent(` + filter + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "bad-history.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", filter)
		}
	}
	for _, name := range []string{"Glyph of Reincarnation"} {
		if _, err := Compile(hostUntapRecord(t, "LEG", name)); err == nil {
			t.Fatal("prerequisite published incomplete card", name)
		}
	}
}

func TestRONBlockHistoryInventory(t *testing.T) {
	n := hostUntapRecord(t, "LEG", "Glyph of Doom")
	entries := map[string]*InventorySignature{}
	walkInventory(n, "Glyph of Doom", "Card", "card", "collection", false, entries)
	found := false
	for _, e := range entries {
		if e.Name == "Ref_TargetPermanent" && e.Role == "permanent_reference" && strings.Contains(e.Context, "WasBlockedByDefenderThisTurn") {
			found = true
		}
	}
	if !found {
		t.Fatal("history reference role missing under unsupported delay")
	}
}

func TestRONBlockHistoryDelusionSourceDuration(t *testing.T) {
	n := hostUntapRecord(t, "LEG", "Glyph of Delusion")
	targeted := n.Fields["Rules"].Args[0].Args[0]
	grant := targeted.Args[1].Args[0].Args[1]
	if grant.Name != "CreatePermanentLayerEffect" || len(grant.Args) != 2 {
		t.Fatal("Oracle grants have no end-of-turn duration")
	}
}
