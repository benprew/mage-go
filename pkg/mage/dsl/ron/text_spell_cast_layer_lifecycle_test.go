package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextSpellCastLayerRecords(t *testing.T) {
	records := 0
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, tc := range []struct {
			name  string
			color core.Color
		}{
			{"Crystal Rod", core.Blue}, {"Iron Star", core.Red}, {"Ivory Cup", core.White}, {"Throne of Bone", core.Black}, {"Wooden Sphere", core.Green},
		} {
			records++
			t.Run(set+"/"+tc.name, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, set, tc.name)
				p := g.PutOnBattlefield(card, a.PlayerID())
				id := p.RuntimeAbilities[0].AbilityID()
				next := core.Red
				if tc.color == next {
					next = core.Blue
				}
				last := core.Green
				if tc.color == last {
					last = core.White
				}
				first, _ := mage.ReplaceColorWord(tc.color, next)
				second, _ := mage.ReplaceColorWord(next, last)
				if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, first); err != nil {
					t.Fatal(err)
				}
				if err := g.AddPrintedSpellCastTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, second); err != nil {
					t.Fatal(err)
				}
				for range 3 {
					g.ApplyContinuousEffects()
				}
				if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
					t.Fatal("identity changed")
				}
				if err := g.RevisePrintedSpellCastTrigger(g.FindPermanent(p.ID()), id, first); err == nil {
					t.Fatal("direct revision changed a layer result")
				}
				castRONSpell(t, g, b, textTriggerSpell(last))
				g.PutTriggersOnStack()
				if g.StackSize() != 2 || a.offers != 0 {
					t.Fatal("ordered filter or choice timing failed")
				}
				a.ManaPool().Add(core.Colorless, 1)
				g.ResolveStack()
				if a.Life() != 21 || a.offers != 1 {
					t.Fatal("payment body failed")
				}
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				castRONSpell(t, clone, clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, textTriggerSpell(tc.color))
				clone.PutTriggersOnStack()
				if clone.StackSize() != 2 {
					t.Fatal("clone expiry failed to restore baseline")
				}
				g.ApplyContinuousEffects()
				castRONSpell(t, g, b, textTriggerSpell(tc.color))
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("clone expiry changed original")
				}
				g.ResolveStack()
				phased := g.PhaseOut(p.ID())
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
				castRONSpell(t, g, b, textTriggerSpell(tc.color))
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("phased expiry lost baseline")
				}
				a.accept = false
				a.ManaPool().Add(core.Colorless, 1)
				g.ResolveStack()
				if a.Life() != 21 || a.ManaPool().TotalMana() != 1 {
					t.Fatal("declined payment spent resources")
				}
				if err := g.AddPrintedSpellCastTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, first); err != nil {
					t.Fatal(err)
				}
				remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
				remove := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
				remove.SetSourceID(remover.ID())
				g.AddContinuousEffect(remove)
				g.ApplyContinuousEffects()
				castRONSpell(t, g, b, textTriggerSpell(next))
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("removed trigger detected cast")
				}
				g.ResolveStack()
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
				g.ApplyContinuousEffects()
				castRONSpell(t, g, b, textTriggerSpell(tc.color))
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("restoration retained expired text")
				}
				g.ResolveStack()
				if err := g.AddPrintedSpellCastTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, first); err != nil {
					t.Fatal(err)
				}
				stale := g.FindPermanent(p.ID())
				g.BouncePermanentToHand(stale)
				lki := g.LKIAbilities(p.ID())
				if len(lki) != 1 || lki[0] != stale.RuntimeAbilities[0] {
					t.Fatal("LKI lost final rule")
				}
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
				if err := g.AddPrintedSpellCastTextEffect(stale, id, core.EndOfTurn, first); err == nil {
					t.Fatal("returned source accepted old visit")
				}
				castRONSpell(t, g, b, textTriggerSpell(next))
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("return retained prior text")
				}
				g.ResolveStack()
				castRONSpell(t, g, b, textTriggerSpell(tc.color))
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("return lost printed text")
				}
			})
		}
	}
	if records != 30 {
		t.Fatalf("tested %d records", records)
	}
}

func TestRONTextSpellCastLayerBaselineController(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(Opponent,IsColor(Red)),ActionList([GainLife(Integer(1))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	baseline, _ := mage.ReplaceColorWord(core.Red, core.Green)
	layer, _ := mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedSpellCastTrigger(p, id, baseline); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedSpellCastTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, layer); err != nil {
		t.Fatal(err)
	}
	castRONSpell(t, g, b, textTriggerSpell(core.Blue))
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.ApplyContinuousEffects()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	ca := clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer
	cb := clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer
	castRONSpell(t, clone, ca, textTriggerSpell(core.Green))
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("pending or current controller changed")
	}
	castRONSpell(t, clone, cb, textTriggerSpell(core.Green))
	clone.PutTriggersOnStack()
	if clone.StackSize() != 1 {
		t.Fatal("opponent condition lost current controller")
	}
	clone.ResolveStack()
	g.ResolveStack()
	castRONSpell(t, g, b, textTriggerSpell(core.Blue))
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("clone control or expiry changed original")
	}
	g.RunStepWithPriority(core.Cleanup)
	castRONSpell(t, g, b, textTriggerSpell(core.Green))
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("cleanup lost prior direct baseline")
	}
}

func TestRONTextSpellCastLayerBasicLandTargets(t *testing.T) {
	g, a, b := ronModalGame(0)
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),Targeted([TargetPermanent(IsLandType(Forest))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	castRONSpell(t, g, b, textTriggerSpell(core.Red))
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedSpellCastTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{forest.ID()}
	g.PutTriggersOnStack()
	castRONSpell(t, g, b, textTriggerSpell(core.Red))
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	a.targets = []uuid.UUID{island.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("pending basic land targets lost text")
	}
	g.ResolveStack()
	if g.FindPermanent(forest.ID()) != nil || g.FindPermanent(island.ID()) != nil {
		t.Fatal("basic land text failed at resolution")
	}
}
