package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONReceivedDamageHistoryShark(t *testing.T) {
	for _, attacking := range []bool{false, true} {
		for _, change := range []string{"none", "undamaged", "prevented", "regenerated", "return", "phase-cycle", "late", "copy", "source-return", "removed", "opponent-return-after"} {
			t.Run(boolName(attacking)+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				card := repositoryCard(t, "DRK", "Giant Shark")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				other := mage.NewCreature("Other", "", 0, 12)
				q := g.PutOnBattlefield(other, b.PlayerID())
				if change == "prevented" {
					g.AddPreventionShield(q.ID(), 1)
				}
				if change != "undamaged" && change != "late" {
					g.DealDamageToPermanentFromSource(q, 1, g.CaptureDamageSource(p.ID()))
				}
				switch change {
				case "regenerated":
					g.AddRegenerationShield(q.ID())
					g.DestroyPermanent(q)
				case "return":
					g.RemoveFromBattlefield(q)
					q = g.PutOnBattlefield(other, b.PlayerID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(q.ID()))
				case "removed":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
					g.ApplyContinuousEffects()
				}
				p.RevokeBaseAttr(core.AttrSummonSick)
				q.RevokeBaseAttr(core.AttrSummonSick)
				if attacking {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
				} else {
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{q.ID()})
				}
				event := core.EvtDeclaredBlocker
				if attacking {
					event = core.EvtCreatureBlockedBy
				}
				g.FireEvent(core.GameEvent{Type: event, SourceID: p.ID(), TargetID: q.ID()})
				if change == "late" {
					g.DealDamageToPermanentFromSource(q, 1, g.CaptureDamageSource(p.ID()))
				}
				g.PutTriggersOnStack()
				want := 1
				if change == "undamaged" || change == "prevented" || change == "return" || change == "late" || change == "removed" {
					want = 0
				}
				if g.StackSize() != want {
					t.Fatalf("stack=%d want=%d", g.StackSize(), want)
				}
				if change == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					want++
				}
				if change == "source-return" {
					g.RemoveFromBattlefield(p)
					p = g.PutOnBattlefield(card, a.PlayerID())
					want = 0
				}
				if change == "opponent-return-after" {
					g.RemoveFromBattlefield(q)
					g.PutOnBattlefield(other, b.PlayerID())
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.FindPermanent(p.ID()).CurrentPower(clone) != 4 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					shark := game.FindPermanent(p.ID())
					if shark.CurrentPower(game) != 4+2*want || shark.CurrentToughness(game) != 4 || shark.HasKeyword(core.Trample) != (want > 0) {
						t.Fatal("wrong saved source, boost or trample")
					}
					game.GetEffects().RemoveEndOfTurn()
					game.ApplyContinuousEffects()
					if shark.CurrentPower(game) != 4 || shark.HasKeyword(core.Trample) {
						t.Fatal("boost did not expire")
					}
				}
			})
		}
	}
}

func TestRONReceivedDamageHistoryInvalid(t *testing.T) {
	for _, filter := range []string{
		`WasDealtDamageThisTurn()`, `WasDealtDamageThisTurn(ThisPermanent)`,
		`WasDealtDamageThisTurn(Integer(1))`, `And([IsCardtype(Creature),WasDealtDamageThisTurn,WasDealtDamageThisTurn])`,
		`And([WasDealtDamageThisTurn])`,
	} {
		n := hostUntapRecord(t, "DRK", "Giant Shark")
		parts, err := Parse(strings.NewReader(filter), "filter.ron")
		if err != nil {
			t.Fatal(err)
		}
		n.Fields["Rules"].Args[1].Args[0].Args[1] = parts[0]
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted", filter)
		}
	}
	for _, rule := range []string{
		`SpellActions(Targeted([TargetPermanent(And([IsCardtype(Creature),WasDealtDamageThisTurn]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`,
		`Activated(PayMana([]),ActionList([DestroyEachPermanent(And([IsCardtype(Creature),WasDealtDamageThisTurn]))]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted unsupported history context", rule)
		}
	}
}

func TestRONReceivedDamageHistoryBlocking(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
	p := g.PutOnBattlefield(repositoryCard(t, "DRK", "Giant Shark"), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 0, 12), b.PlayerID())
	q.RevokeBaseAttr(core.AttrSummonSick)
	g.DealDamageToPermanentFromSource(q, 1, g.CaptureDamageSource(p.ID()))
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{q.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: q.ID()}})
	if p.CurrentPower(g) != 6 || !p.HasKeyword(core.Trample) {
		t.Fatal("blocking a damaged attacker did not trigger")
	}
}

func TestRONReceivedDamageHistoryCastingAndCombat(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
	card := repositoryCard(t, "DRK", "Giant Shark")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid casting did not roll back")
	}
	a.ManaPool().Add(core.Colorless, 5)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || a.ManaPool().TotalMana() != 0 || !p.HasSubType("Shark") {
		t.Fatal("wrong cost, entry or subtype")
	}
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.ApplyContinuousEffects()
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("attacked a player without an Island")
	}
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
	g.ApplyContinuousEffects()
	if !p.CanDeclareAsAttacker(g) {
		t.Fatal("could not attack a player with an Island")
	}
	assignments := make([]mage.BlockAssignment, 0, 3)
	for i := range 3 {
		q := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 12), b.PlayerID())
		if i < 2 {
			g.DealDamageToPermanentFromSource(q, 1, g.CaptureDamageSource(p.ID()))
		}
		assignments = append(assignments, mage.BlockAssignment{BlockerID: q.ID(), AttackerID: p.ID()})
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	g.ExecuteBlockers(assignments)
	if p.CurrentPower(g) != 8 || p.CurrentToughness(g) != 4 || !p.HasKeyword(core.Trample) {
		t.Fatal("each damaged blocker must supply one boost")
	}
	g.ExecuteCombatDamage()
	if p.CurrentPower(g) != 8 {
		t.Fatal("later combat damage supplied another block event")
	}
}

func TestRONReceivedDamageHistoryInventory(t *testing.T) {
	n := hostUntapRecord(t, "DRK", "Giant Shark")
	entries := map[string]*InventorySignature{}
	walkInventory(n, "Giant Shark", "Card", "card", "collection", false, entries)
	count := 0
	for _, entry := range entries {
		if entry.Name == "WasDealtDamageThisTurn" {
			if entry.Role != "permanent_filter" || entry.Kind != "symbol" {
				t.Fatalf("wrong history role: %+v", entry)
			}
			count++
		}
	}
	if count != 2 {
		t.Fatalf("history event contexts=%d", count)
	}
}
