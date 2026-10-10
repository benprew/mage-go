package ron

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func combatLayerCreature(t *testing.T, rules string) mage.Card {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Layer subject",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:8),Rules:[`+rules+`])`), "combat-layer.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory()
}

func TestRONTextCombatPairLayerOrderingExpiry(t *testing.T) {
	for _, kind := range []core.EventType{core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy} {
		t.Run(kind.String(), func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Aisling Leprechaun")
			p := g.PutOnBattlefield(card, a.PlayerID())
			idx := 0
			if kind == core.EvtCreatureBlockedBy {
				idx = 1
			}
			id := p.RuntimeAbilities[idx].AbilityID()
			first := combatTextChange(t, core.Green, core.Red)
			second := combatTextChange(t, core.Red, core.Blue)
			old := g.PutOnBattlefield(mage.NewCreature("Old", "", 0, 8), b.PlayerID())
			fresh := g.PutOnBattlefield(mage.NewCreature("Fresh", "", 0, 8), b.PlayerID())
			fireTextPair(g, p, old, kind)
			if err := g.AddPrintedCombatPairTextEffect(p, id, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedCombatPairTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, second); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				g.ApplyContinuousEffects()
			}
			if err := g.RevisePrintedCombatPairTrigger(g.FindPermanent(p.ID()), id, first); err == nil {
				t.Fatal("direct revision changed a layered result")
			}
			fireTextPair(g, g.FindPermanent(p.ID()), fresh, kind)
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("pending triggers changed during installation")
			}
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if !slices.Contains(clone.FindPermanent(old.ID()).Colors(), core.Green) || !slices.Contains(clone.FindPermanent(fresh.ID()).Colors(), core.Blue) {
				t.Fatal("expiry changed pending bodies")
			}
			fireTextPair(clone, clone.FindPermanent(p.ID()), clone.FindPermanent(fresh.ID()), kind)
			clone.ResolveStack()
			if !slices.Contains(clone.FindPermanent(fresh.ID()).Colors(), core.Green) {
				t.Fatal("expiry did not recompute the surviving change from baseline")
			}
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(fresh.ID()).Colors(), core.Blue) {
				t.Fatal("clone expiry changed original rules")
			}
			siblingKind := core.EvtCreatureBlockedBy
			if kind == siblingKind {
				siblingKind = core.EvtDeclaredBlocker
			}
			fireTextPair(g, g.FindPermanent(p.ID()), old, siblingKind)
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(old.ID()).Colors(), core.Green) {
				t.Fatal("text changes reached a sibling declaration")
			}
			phased := g.PhaseOut(p.ID())
			g.RunStepWithPriority(core.Cleanup)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			fireTextPair(g, g.FindPermanent(p.ID()), fresh, kind)
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(fresh.ID()).Colors(), core.Green) {
				t.Fatal("expiry while phased lost the baseline")
			}
			if err := g.AddPrintedCombatPairTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
			removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			g.ApplyContinuousEffects()
			fireTextPair(g, g.FindPermanent(p.ID()), fresh, kind)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("removed declaration detected combat")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			g.ApplyContinuousEffects()
			fireTextPair(g, g.FindPermanent(p.ID()), fresh, kind)
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(fresh.ID()).Colors(), core.Green) {
				t.Fatal("restoration retained expired text")
			}
			if err := g.AddPrintedCombatPairTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, first); err != nil {
				t.Fatal(err)
			}
			stale := g.FindPermanent(p.ID())
			g.BouncePermanentToHand(stale)
			if g.LKIAbilities(p.ID())[idx] != stale.RuntimeAbilities[idx] {
				t.Fatal("departure lost the final rule")
			}
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.AddPrintedCombatPairTextEffect(stale, id, core.EndOfTurn, first); err == nil {
				t.Fatal("returned source accepted the former visit")
			}
			fireTextPair(g, returned, fresh, kind)
			g.ResolveStack()
			if !slices.Contains(g.FindPermanent(fresh.ID()).Colors(), core.Green) {
				t.Fatal("returned source lost printed rules")
			}
		})
	}
}

func TestRONTextCombatPairLayerBaselineController(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(combatLayerCreature(t, `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),GainLife(Integer(1)))]))`), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	first := combatTextChange(t, core.Green, core.Red)
	if err := g.RevisePrintedCombatPairTrigger(p, id, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedCombatPairTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, combatTextChange(t, core.Red, core.Blue)); err != nil {
		t.Fatal(err)
	}
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
	fireTextPair(g, g.FindPermanent(p.ID()), other, core.EvtDeclaredBlocker)
	clone := g.Clone()
	clone.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	clone.ApplyContinuousEffects()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	fireTextPair(clone, clone.FindPermanent(p.ID()), clone.FindPermanent(other.ID()), core.EvtDeclaredBlocker)
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("current source controller reads changed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("clone controller changed the original declaration")
	}
}

func TestRONTextCombatPairLayerDirectBaseline(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEG", "Aisling Leprechaun"), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	if err := g.RevisePrintedCombatPairTrigger(p, id, combatTextChange(t, core.Green, core.Red)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedCombatPairTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, combatTextChange(t, core.Red, core.Blue)); err != nil {
		t.Fatal(err)
	}
	q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
	g.RunStepWithPriority(core.Cleanup)
	fireTextPair(g, g.FindPermanent(p.ID()), q, core.EvtDeclaredBlocker)
	g.ResolveStack()
	if !slices.Contains(g.FindPermanent(q.ID()).Colors(), core.Red) {
		t.Fatal("cleanup lost the prior direct baseline")
	}
}

func TestRONTextCombatPairLayerBasicLandTargets(t *testing.T) {
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(combatLayerCreature(t, `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),Targeted([TargetPermanent(IsLandType(Forest))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	fireTextPair(g, p, q, core.EvtDeclaredBlocker)
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedCombatPairTextEffect(p, p.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{forest.ID()}
	g.PutTriggersOnStack()
	fireTextPair(g, g.FindPermanent(p.ID()), q, core.EvtDeclaredBlocker)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	a.targets = []uuid.UUID{island.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("pending target declarations changed during expiry")
	}
	g.ResolveStack()
	if g.FindPermanent(forest.ID()) != nil || g.FindPermanent(island.ID()) != nil {
		t.Fatal("basic land targets lost their saved text")
	}
}

func TestRONTextCombatPairLayerWallDirections(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEG", "Infernal Medusa"}, {"ATQ", "Battering Ram"}, {"DRK", "Venom"},
		{"LEA", "Cockatrice"}, {"LEB", "Cockatrice"}, {"2ED", "Cockatrice"}, {"3ED", "Cockatrice"}, {"CED", "Cockatrice"}, {"CEI", "Cockatrice"},
		{"LEA", "Thicket Basilisk"}, {"LEB", "Thicket Basilisk"}, {"2ED", "Thicket Basilisk"}, {"3ED", "Thicket Basilisk"}, {"CED", "Thicket Basilisk"}, {"CEI", "Thicket Basilisk"},
	} {
		for _, kind := range []core.EventType{core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy} {
			t.Run(tc.set+"/"+tc.name+"/"+kind.String(), func(t *testing.T) {
				g, a, b := ronGame()
				p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				subject := p
				if tc.name == "Venom" {
					subject = g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), a.PlayerID())
					g.Attach(p.ID(), subject.ID())
				}
				for i, ability := range p.RuntimeAbilities {
					if _, ok := ability.(*mage.GenericTriggered); !ok || (tc.name == "Battering Ram" && i == 0) {
						continue
					}
					if err := g.AddPrintedCombatPairTextEffect(g.FindPermanent(p.ID()), ability.AbilityID(), core.EndOfTurn, combatTextChange(t, core.Green, core.Red)); err != nil {
						t.Fatal(err)
					}
				}
				if tc.name == "Cockatrice" && !g.FindPermanent(p.ID()).HasAttr(core.Flying) {
					t.Fatal("text layer changed the sibling flying rule")
				}
				if tc.name == "Battering Ram" {
					g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
					g.ResolveStack()
					if !g.FindPermanent(p.ID()).HasAttr(core.Banding) {
						t.Fatal("text layer changed the sibling combat rule")
					}
				}
				wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 8, mage.WithSubTypes("Wall")), b.PlayerID())
				if tc.name == "Venom" {
					fireTextPair(g, subject, wall, core.EvtCombatPair)
				} else {
					fireTextPair(g, subject, wall, kind)
				}
				g.ResolveStack()
				g.RunStepWithPriority(core.EndCombat)
				destroyed := (tc.name == "Infernal Medusa" && kind == core.EvtDeclaredBlocker) || (tc.name == "Battering Ram" && kind == core.EvtCreatureBlockedBy)
				if (g.FindPermanent(wall.ID()) == nil) != destroyed {
					t.Fatal("text layer changed the Wall restriction or event direction")
				}
				if tc.name == "Battering Ram" && g.FindPermanent(p.ID()).HasAttr(core.Banding) {
					t.Fatal("banding continued after combat")
				}
			})
		}
	}
}

func TestRONTextCombatPairLayerAuraReattachment(t *testing.T) {
	g, a, b := ronGame()
	aura := g.PutOnBattlefield(repositoryCard(t, "DRK", "Venom"), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First host", "", 0, 8), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second host", "", 0, 8), a.PlayerID())
	g.Attach(aura.ID(), first.ID())
	if err := g.AddPrintedCombatPairTextEffect(aura, aura.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, combatTextChange(t, core.Green, core.Red)); err != nil {
		t.Fatal(err)
	}
	old := g.PutOnBattlefield(mage.NewCreature("Old opponent", "", 0, 8), b.PlayerID())
	fresh := g.PutOnBattlefield(mage.NewCreature("New opponent", "", 0, 8), b.PlayerID())
	fireTextPair(g, first, old, core.EvtCombatPair)
	g.Attach(aura.ID(), second.ID())
	fireTextPair(g, first, fresh, core.EvtCombatPair)
	fireTextPair(g, second, fresh, core.EvtCombatPair)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("attachment change lost a pending capture or retained the former host")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.ResolveStack()
	g.RunStepWithPriority(core.EndCombat)
	if g.FindPermanent(old.ID()) != nil || g.FindPermanent(fresh.ID()) != nil {
		t.Fatal("expiry or departure lost the saved opponents")
	}
}
