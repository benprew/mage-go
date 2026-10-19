package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHistoryPlayerTargetQueryRealCard(t *testing.T) {
	for _, state := range []string{"opponent", "self", "phase", "return", "cleanup", "clone", "copy", "prevention"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			player := b
			if state == "self" {
				player = a
			}
			p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 4), player.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(player.PlayerID(), []uuid.UUID{p.ID()})
			obj := castRONSpell(t, g, a, repositoryCard(t, "DRK", "Fire and Brimstone"), player.PlayerID())
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "prevention":
				g.AddPreventionShield(b.PlayerID(), 2)
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("complete history packet tree is unknown or proves destruction")
				}
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("query dealt damage")
			}
			if state == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
				if g.PlayerAttackedThisTurn(player.PlayerID()) || g.StackSize() != 0 || a.Life() != 16 || b.Life() != 16 {
					t.Fatal("cleanup priority lost history before resolution or retained it after cleanup")
				}
				return
			}
			if state == "clone" {
				clone := g.Clone()
				if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), nil); certain || !known {
					t.Fatal("clone lost query policy")
				}
				clone.ResolveTopOfStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 16 || clone.GetPlayer(b.PlayerID()).Life() != 16 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone crossed damage state")
				}
			}
			g.ResolveTopOfStack()
			wantA, wantB := 16, 16
			switch state {
			case "self":
				wantA, wantB = 12, 20
			case "copy":
				wantA, wantB = 20, 12
			case "prevention":
				wantB = 18
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=%d/%d want=%d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONHistoryPlayerTargetQueryBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AttackedThisTurn)],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(4),Player(Ref_TargetPlayer))])])))`)
	g, a, b := ronGame()
	for _, player := range []*mage.BasePlayer{a, b} {
		p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 4), player.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(player.PlayerID(), []uuid.UUID{p.ID()})
	}
	first := castRONSpell(t, g, a, factory(), a.PlayerID())
	second := castRONSpell(t, g, a, factory(), b.PlayerID())
	for _, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("per-instance history binding lost")
		}
	}
	first.TargetSpecs[0] = second.TargetSpecs[0]
	if _, known := g.InspectSpellDestruction(first, nil); known {
		t.Fatal("foreign factory declaration accepted")
	}
	g.ResolveTopOfStack()
	if b.Life() != 16 || a.Life() != 20 {
		t.Fatal("second target used first binding")
	}
}

func TestRONHistoryPlayerTargetQueryInvalid(t *testing.T) {
	valid := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AttackedThisTurn)],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(4),Player(Ref_TargetPlayer))])])))])`
	for _, source := range []string{
		strings.Replace(valid, "AttackedThisTurn", "AttackedThisTurn()", 1),
		strings.Replace(valid, "AttackedThisTurn", "And([AttackedThisTurn,Opponent])", 1),
		strings.Replace(valid, "AttackedThisTurn", "DidntAttackThisTurn", 1),
		strings.Replace(valid, "Ref_TargetPlayer", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "Ref_TargetPlayer", "Ref_TargetPlayer2", 1),
		strings.Replace(valid, "TargetPlayer(AttackedThisTurn)", "", 1),
	} {
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("invalid history declaration accepted: %s", source)
		}
	}
}
