package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONConditionalTargetQueryGameplay(t *testing.T) {
	for _, name := range []string{"Stone Rain", "Ice Storm", "Sinkhole", "Shatter"} {
		for _, state := range []string{"quiet", "attack", "block", "not-declared-blocker", "phase", "phase-cycle", "return", "removed", "control", "copy"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "DRK", "Lurker")
				victim := g.PutOnBattlefield(card, b.PlayerID())
				kind := core.TypeLand
				if name == "Shatter" {
					kind = core.TypeArtifact
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, kind)), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfCombat, mage.RemoveAbilities(mage.This)), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want := false
				switch state {
				case "attack", "phase-cycle":
					g.FindPermanent(victim.ID()).RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{victim.ID()})
					want = true
				case "block":
					attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 4), a.PlayerID())
					attacker.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: attacker.ID()}})
					want = true
				case "not-declared-blocker":
					g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: victim.ID(), PlayerID: b.PlayerID()})
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, "LEA", name), victim.ID())
				g.GetEffects().RemoveEndOfCombat()
				g.ApplyContinuousEffects()
				switch state {
				case "removed":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), victim.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = true
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: obj.ID, TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
					if obj == nil {
						t.Fatal("copy failed")
					}
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
					g.PutOnBattlefield(card, b.PlayerID())
				}
				if state == "phase" || state == "phase-cycle" {
					g.PhaseOut(victim.ID())
					if state == "phase-cycle" {
						g.PhaseIn([]uuid.UUID{victim.ID()})
					}
				}
				instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
				if !ok || len(instructions) != 1 {
					t.Fatal("destruction declaration lost")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("target declaration lost")
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					got, known := plan.Select(game, game.StackPeek())
					if !known || (len(got) == 1) != want {
						t.Fatalf("got %v known %v; want legal %v", got, known, want)
					}
					game.ResolveStack()
					if (game.FindPermanentIncludingPhased(victim.ID()) == nil) != want {
						t.Fatal("native resolution differs")
					}
				}
				if state == "quiet" {
					spell := repositoryCard(t, "LEA", name)
					spell.SetOwner(a.PlayerID())
					a.AddToHand(spell)
					mana := a.ManaPool().TotalMana()
					if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{victim.ID()}, 0); err == nil || a.ManaPool().TotalMana() != mana || g.StackSize() != 0 {
						t.Fatal("illegal conditional target paid or entered the stack")
					}
				}
			})
		}
	}
}

func TestRONConditionalTargetQueryInvalid(t *testing.T) {
	valid := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Unless(PermanentPassesFilter(ThisPermanent,Or([AttackedThisTurn,BlockedThisTurn])),[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfSpells(AnySpell)])])])`
	for _, invalid := range []string{
		strings.Replace(valid, "AttackedThisTurn", "AttackedThisTurn()", 1),
		strings.Replace(valid, "BlockedThisTurn", "BlockedThisTurn()", 1),
		strings.Replace(valid, "Or([AttackedThisTurn,BlockedThisTurn])", "Or([])", 1),
		strings.Replace(valid, "Or([AttackedThisTurn,BlockedThisTurn])", "Or([AttackedThisTurn])", 1),
		strings.Replace(valid, "ThisPermanent,Or", "Ref_TargetPermanent,Or", 1),
		strings.Replace(valid, "AnySpell", "IsColor(Unknown)", 1),
	} {
		if _, err := Compile(nameScopeNode(t, invalid)); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
	}
}
