package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronTurnSkipPlayer struct {
	*mage.BasePlayer
	accept bool
	asks   int
}

func (p *ronTurnSkipPlayer) ChooseMayAbility(string) bool { p.asks++; return p.accept }

func TestRONTurnBeginReplacement(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			card := repositoryCard(t, set, "Time Vault")
			a := &ronTurnSkipPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.asks != 0 {
				t.Fatal("failed casting changed state")
			}
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !p.Tapped {
				t.Fatal("entry not tapped")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), len(p.RuntimeAbilities)-1, nil); err == nil {
				t.Fatal("tapped source activated")
			}
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("ordinary untap restriction failed")
			}
			g.SetTurn(2)
			if beginRONReplacementTurn(g) || a.asks != 1 || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("replacement timing failed")
			}
			clone := g.Clone()
			g.AdvanceToNextTurn()
			if !beginRONReplacementTurn(g) || g.FindPermanent(p.ID()).Tapped {
				t.Fatal("next actual turn did not untap")
			}
			if !clone.FindPermanent(p.ID()).Tapped {
				t.Fatal("clone shared continuation")
			}
			clone.AdvanceToNextTurn()
			if !beginRONReplacementTurn(clone) || clone.FindPermanent(p.ID()).Tapped {
				t.Fatal("clone lost continuation")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), len(p.RuntimeAbilities)-1, nil); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(p.ID()).Tapped || g.HasExtraTurns() {
				t.Fatal("extra turn occurred at announcement")
			}
			g.ResolveStack()
			if !g.HasExtraTurns() {
				t.Fatal("extra turn not scheduled")
			}
			g.AdvanceToNextTurn()
			a.accept = false
			if g.ActivePlayerObj().PlayerID() != a.PlayerID() || !beginRONReplacementTurn(g) {
				t.Fatal("extra turn did not start")
			}
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("decline untapped source")
			}
		})
	}
}

func TestRONTurnBeginReplacementInvalid(t *testing.T) {
	valid := `ReplaceWouldBeginATurn(PlayerWouldBeginTheirTurn(You),[MayCost(SkipThisTurn),If(CostWasPaid,[UntapPermanent(ThisPermanent)])])`
	for _, bad := range []string{
		strings.Replace(valid, "PlayerWouldBeginTheirTurn", "PlayerWouldBeginTheirTurnWrong", 1),
		strings.Replace(valid, "PlayerWouldBeginTheirTurn(You)", "PlayerWouldBeginTheirTurn(You,You)", 1),
		strings.Replace(valid, "UntapPermanent(ThisPermanent)", "UntapPermanent(ThisPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "You", "Opponent", 1), strings.Replace(valid, "You", "You()", 1),
		strings.Replace(valid, "SkipThisTurn", "SkipThisTurn()", 1), strings.Replace(valid, "CostWasPaid", "CostWasPaid()", 1),
		strings.Replace(valid, "ThisPermanent", "HostPermanent", 1), strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "UntapPermanent(ThisPermanent)", "GainLife(Integer(1))", 1),
		strings.Replace(valid, "MayCost(SkipThisTurn),", "", 1), strings.Replace(valid, "If(CostWasPaid,[UntapPermanent(ThisPermanent)])", "", 1),
		strings.Replace(valid, "MayCost(SkipThisTurn)", "MayCost(SkipThisTurn,SkipThisTurn)", 1),
		strings.Replace(valid, "If(CostWasPaid", "Unless(CostWasPaid", 1),
	} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[If(PermanentPassesFilter(ThisPermanent,IsTapped),[`+bad+`])])`))
		if err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRONTurnBeginReplacementInventory(t *testing.T) {
	record := repositoryRecord(t, "LEB", "Time Vault", "LEB.ron")
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Time Vault", "card", "card", "card", true, entries)
	for name, role := range map[string]string{"PlayerWouldBeginTheirTurn": "turn_replacement_event", "SkipThisTurn": "turn_replacement_cost", "CostWasPaid": "turn_replacement_payment_result", "UntapPermanent": "turn_replacement_continuation"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %s in role %s", name, role)
		}
	}
}

func TestRONTurnBeginReplacementExtraTurns(t *testing.T) {
	a := &ronTurnSkipPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Time Vault"), a.PlayerID())
	g.UntapPermanent(p)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), len(p.RuntimeAbilities)-1, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	g.ResolveStack()
	g.AdvanceToNextTurn()
	if g.ActivePlayerObj().PlayerID() != a.PlayerID() || beginRONReplacementTurn(g) {
		t.Fatal("first extra turn was not skipped")
	}
	g.AdvanceToNextTurn()
	a.accept = false
	if g.ActivePlayerObj().PlayerID() != a.PlayerID() || !beginRONReplacementTurn(g) || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("second extra turn did not start with untap")
	}
	g.AdvanceToNextTurn()
	if g.ActivePlayerObj().PlayerID() != b.PlayerID() || !beginRONReplacementTurn(g) {
		t.Fatal("multiple activations created a skip debt")
	}
}

func beginRONReplacementTurn(g *mage.Game) bool {
	if !g.TryBeginTurn() {
		return false
	}
	g.RunStepWithPriority(core.Upkeep)
	return true
}
