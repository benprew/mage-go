package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeMixedDelayedBerserk(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "clone", "copy", "returned", "phased", "shroud", "no attack", "changed type"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "", 3, 9)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				spell := repositoryCard(t, set, "Berserk")
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("unpaid cast changed state")
				}
				a.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				switch state {
				case "returned":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(card, a.PlayerID())
				case "phased":
					g.PhaseOut(p.ID())
				case "shroud":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete Berserk tree unknown or direct destruction claimed")
					}
				}
				if len(g.DelayedTriggers()) != 0 || g.FindPermanentIncludingPhased(p.ID()).CurrentPower(g) != 3 {
					t.Fatal("inspection executed instructions")
				}
				g.ResolveStack()
				affected := state != "returned" && state != "phased" && state != "shroud"
				wantPower := 3
				if affected {
					wantPower = 6
					if state == "copy" {
						wantPower = 12
					}
				}
				current := g.FindPermanentIncludingPhased(p.ID())
				if current.CurrentPower(g) != wantPower || current.HasKeyword(core.Trample) != affected {
					t.Fatal("incorrect mixed layer")
				}
				if !affected {
					if len(g.DelayedTriggers()) != 0 {
						t.Fatal("illegal spell scheduled a trigger")
					}
					return
				}
				if state != "no attack" {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
					g.GetCombat().RemoveFromCombat(p.ID())
				}
				if state == "changed type" {
					g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrIsCreature)
					if g.FindPermanent(p.ID()).HasType(core.TypeCreature) {
						t.Fatal("subject stayed a creature")
					}
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() == 0 {
					t.Fatal("end step did not trigger")
				}
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if (game.FindPermanent(p.ID()) == nil) != (state != "no attack") {
						t.Fatal("wrong delayed destruction")
					}
					game.GetEffects().RemoveEndOfTurn()
					game.ApplyContinuousEffects()
					if current := game.FindPermanent(p.ID()); current != nil && (current.CurrentPower(game) != 3 || current.HasKeyword(core.Trample)) {
						t.Fatal("mixed layer survived cleanup")
					}
				}
			})
		}
	}
}

func TestRONNativeMixedDelayedRejectsDeclarations(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([Trample]),AdjustPTX(PlusX,Zero,ValueX)],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AddAbility([Trample()]),AdjustPT(1,0)],UntilEndOfTurn)`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep(),ActionList([DestroyPermanent(Ref_TargetPermanent)]))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([]))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(PermanentPassesFilter(You,AttackedThisTurn),[DestroyPermanent(Ref_TargetPermanent)])]))`,
	} {
		source := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}
