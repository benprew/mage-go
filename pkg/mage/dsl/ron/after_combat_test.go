package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAfterCombatTiming(t *testing.T) {
	for _, playerIndex := range []int{0, 1} {
		for _, completed := range []bool{false, true} {
			for _, step := range []core.PhaseStep{core.PrecombatMain, core.BeginCombat, core.DeclareAttackers, core.DeclareBlockers, core.CombatDamage, core.EndCombat, core.PostcombatMain, core.EndStep} {
				g, a, b := ronGame()
				if completed {
					g.RunStepWithPriority(core.EndCombat)
				}
				g.SetStep(step)
				player := a
				if playerIndex == 1 {
					player = b
				}
				card := repositoryCard(t, "LEG", "Glyph of Reincarnation")
				card.SetOwner(player.PlayerID())
				player.AddToHand(card)
				player.ManaPool().Add(core.Green, 1)
				wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), a.PlayerID())
				if got := len(g.GetCastableSpells(player.PlayerID())) == 1; got != completed {
					t.Fatalf("query at %s = %v, want %v", step, got, completed)
				}
				err := g.CastSpellByID(player.PlayerID(), card.ID(), []uuid.UUID{wall.ID()}, 0)
				if (err == nil) != completed {
					t.Fatalf("cast at %s: %v, want %v", step, err, completed)
				}
				if !completed && (len(player.Hand()) != 1 || player.ManaPool().TotalMana() != 1 || !g.GetStack().IsEmpty()) {
					t.Fatal("rejected cast changed resources")
				}
				if completed {
					g.SetTurn(g.CurrentTurn() + 1)
					g.SetStep(core.Upkeep)
					if g.CopySpellOnStack(card.ID(), player.PlayerID(), false) == nil {
						t.Fatal("copy repeated timing check")
					}
					clone := g.Clone()
					clone.ResolveStack()
					g.ResolveStack()
					if len(player.Graveyard()) != 1 {
						t.Fatal("announced spell did not resolve after timing changed")
					}
				}
			}
		}
	}
}

func TestRONAfterCombatInvalid(t *testing.T) {
	for _, expression := range []string{"IsAfterCombat()", "IsAfterCombat(You)", "And([IsAfterCombat,IsDuringCombat])", "And([IsDuringCombat,And([IsAfterCombat])])"} {
		record := hostUntapRecord(t, "LEG", "Glyph of Reincarnation")
		n, err := Parse(strings.NewReader("CastEffect(CantBeCastUnless("+expression+"))"), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		rules := record.Fields["Rules"]
		rules.Args[0] = n[0]
		record.Fields["Rules"] = rules
		if _, err := Compile(record); err == nil {
			t.Fatal("accepted", expression)
		}
	}
}

func TestRONAfterCombatScope(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `CastEffect(CantBeCastUnless(And([IsAfterCombat,And([IsPlayersTurn(You)])]))),SpellActions(ActionList([GainLife(Integer(1))]))`)
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	g.SetStep(core.BeginCombat)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("cast during first combat")
	}
	g.RunStepWithPriority(core.EndCombat)
	g.SetStep(core.PostcombatMain)
	g.SetActivePlayerIndex(1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("cast on the wrong player turn")
	}
	g.SetActivePlayerIndex(0)
	g.SetStep(core.BeginCombat)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("nested conjunction body did not execute")
	}
	for _, source := range []string{
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(IsAfterCombat))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(IsAfterCombat)),SpellActions(Modal_ChooseOne([ActionList([GainLife(Integer(1))]),ActionList([GainLife(Integer(2))])]))])`,
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([IsAfterCombat]))])`,
	} {
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid context", source)
		}
	}
}
