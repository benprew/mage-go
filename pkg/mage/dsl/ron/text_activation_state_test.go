package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextActivationStateHistory(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Dragon Whelp"}, {"PROMOS", "Nalathni Dragon"}, {"FEM", "Farrelite Priest"}, {"FEM", "Initiates of the Ebon Hand"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			a.ManaPool().Add(core.Red, 8)
			a.ManaPool().Add(core.Colorless, 8)
			for range 3 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("lost active identity")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("revision reset activation history")
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("saved history did not cause sacrifice")
			}
		})
	}
}

func TestRONTextActivationStateQueuedAndRepeated(t *testing.T) {
	original := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)()
	g, a, b := ronGame()
	p := g.PutOnBattlefield(original, a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{red.ID()}); err == nil {
		t.Fatal("old filter remained")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal(err)
	}
	change, _ = mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedActivatedAbility(g.FindPermanent(p.ID()), id, change); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err == nil {
		t.Fatal("successive revision lost order")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{blue.ID()}); err != nil {
		t.Fatal(err)
	}
	cloned := g.Clone()
	g.ResolveStack()
	for _, target := range []*mage.Permanent{red, green, blue} {
		if g.FindPermanent(target.ID()) != nil || cloned.FindPermanent(target.ID()) == nil {
			t.Fatal("pending body or clone changed")
		}
	}
	cloned.ResolveStack()
	for _, target := range []*mage.Permanent{red, green, blue} {
		if cloned.FindPermanent(target.ID()) != nil {
			t.Fatal("clone lost saved body")
		}
	}
}

func TestRONTextActivationStateLimitsAndIdentity(t *testing.T) {
	for _, modifier := range []string{`ActivateOnlyOnceEachTurn`, `ActivateNoMoreThanNumberTimesEachTurn(Integer(2))`} {
		t.Run(modifier, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostR]),ActionList([GainLife(Integer(1))]),`+modifier+`)`)
			g, a, b := ronGame()
			card := factory()
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("unpaid activation succeeded")
			}
			if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Red, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal("failed payment spent limit", err)
			}
			g.ResolveStack()
			clone := g.Clone()
			if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			if modifier != `ActivateOnlyOnceEachTurn` {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
					t.Fatal(err)
				}
				if err := clone.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal("clone shared limit", err)
				}
				clone.ResolveStack()
			}
			before := a.ManaPool().TotalMana()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("revision reset limit")
			}
			if a.ManaPool().TotalMana() != before {
				t.Fatal("rejected activation spent mana")
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.ApplyContinuousEffects()
			b.ManaPool().Add(core.Red, 3)
			if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("control change reset limit")
			}
			phased := g.PhaseOut(p.ID())
			if err := g.RevisePrintedActivatedAbility(p, id, change); err == nil {
				t.Fatal("phased source revised")
			}
			g.PhaseIn(phased)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("phasing reset limit")
			}
			g.RunStepWithPriority(core.Cleanup)
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			b.ManaPool().Add(core.Red, 3)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal("cleanup did not reset revised limit", err)
			}
			g.ResolveStack()
			stale := g.FindPermanent(p.ID())
			g.BouncePermanentToHand(stale)
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.RevisePrintedActivatedAbility(stale, id, change); err == nil {
				t.Fatal("returned source inherited revision")
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), returned.ID(), 0, nil); err != nil {
				t.Fatal("returned source inherited limit", err)
			}
		})
	}
}

func TestRONTextActivationStateHistoryQueueAndClone(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Dragon Whelp")
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	a.ManaPool().Add(core.Red, 8)
	for range 3 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.CounterStackObject(g.StackPeek().ID)
	}
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("queued original or copy lost history")
	}
	if len(clone.DelayedTriggers()) != 0 {
		t.Fatal("clone shared result storage")
	}
	clone.ResolveStack()
	if len(clone.DelayedTriggers()) != 2 {
		t.Fatal("clone lost bound history")
	}
	g.RunStepWithPriority(core.EndStep)
	g.RunStepWithPriority(core.Cleanup)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	fresh := g.PutOnBattlefield(repositoryCard(t, "LEA", "Dragon Whelp"), a.PlayerID())
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), fresh.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("another instance inherited activation history")
	}
}

func TestRONTextActivationStateEligibility(t *testing.T) {
	g, a, _ := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([GainLife(Integer(1))]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	other := g.PutOnBattlefield(factory(), a.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	original := p.RuntimeAbilities[0]
	if err := g.RevisePrintedActivatedAbility(p, other.RuntimeAbilities[0].AbilityID(), change); err == nil {
		t.Fatal("other declaration matched")
	}
	if err := g.RevisePrintedActivatedAbility(p, original.AbilityID(), mage.TextChange{}); err == nil {
		t.Fatal("invalid change accepted")
	}
	if p.RuntimeAbilities[0] != original {
		t.Fatal("failed revision changed ability")
	}
	g.MutablePermanent(p.ID()).RuntimeAbilities = nil
	if err := g.RevisePrintedActivatedAbility(p, original.AbilityID(), change); err == nil {
		t.Fatal("removed ability revised")
	}
	grant := mage.NewAction(mage.ActionActivated, mage.WithTextRevisionActivationState(), mage.WithEffect(mage.GainLifeFor(mage.You, mage.Fixed(1))))
	g.MutablePermanent(p.ID()).RuntimeAbilities = append(g.FindPermanent(p.ID()).RuntimeAbilities, grant)
	if err := g.RevisePrintedActivatedAbility(p, grant.AbilityID(), change); err == nil {
		t.Fatal("unprinted grant revised")
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			if _, err := Compile(hostUntapRecord(t, set, name)); err == nil {
				t.Fatal("partial text card accepted", set, name)
			}
		}
	}
}

func TestRONTextActivationStateIndependentHistoriesAndCleanup(t *testing.T) {
	condition := `TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(2)))`
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([If(`+condition+`,[GainLife(Integer(1))])])),Activated(PayMana([]),ActionList([If(`+condition+`,[GainLife(Integer(2))])]))`)
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	for index := range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if a.Life() != 20 {
		t.Fatal("independent declarations shared history")
	}
	clone := g.Clone()
	change, _ := mage.ReplaceColorWord(core.White, core.Blue)
	for index := range 2 {
		id := source.RuntimeAbilities[index].AbilityID()
		if err := g.RevisePrintedActivatedAbility(source, id, change); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if a.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("history rebinding or clone isolation failed")
	}
	for index := range 2 {
		if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, nil); err != nil {
			t.Fatal(err)
		}
		clone.ResolveStack()
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("original clone history changed")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	for index := range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if a.Life() != 23 {
		t.Fatal("cleanup retained revised history")
	}
}

func TestRONTextActivationStateCopiedDefinitionAndResults(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DrawACard,RevealTheCardDrawnThisWay,DiscardTheCardDrawnThisWay,DestroyPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedActivatedAbility(p, id, change); err != nil {
		t.Fatal(err)
	}
	values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
	if err != nil {
		t.Fatal(err)
	}
	physical := repositoryCard(t, "LEA", "Wall of Wood")
	copiedCard, err := mage.BuildEntryCopy(physical, values)
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(copiedCard, a.PlayerID())
	copiedID := copied.RuntimeAbilities[0].AbilityID()
	change, _ = mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedActivatedAbility(copied, copiedID, change); err != nil {
		t.Fatal(err)
	}
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	first, second := repositoryCard(t, "LEA", "Forest"), repositoryCard(t, "LEA", "Island")
	a.SetLibrary([]mage.Card{first, second})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), copied.ID(), 0, []uuid.UUID{green.ID()}); err == nil {
		t.Fatal("copy inherited a text-layer value")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), copied.ID(), 0, []uuid.UUID{blue.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 2 || len(g.PublicReveals()) != 2 {
		t.Fatal("revised actions shared draw/reveal results")
	}
	if g.FindPermanent(red.ID()) == nil || g.FindPermanent(green.ID()) != nil || g.FindPermanent(blue.ID()) != nil {
		t.Fatal("copied definitions lost independent text")
	}
	g.BouncePermanentToHand(g.FindPermanent(copied.ID()))
	if len(a.Hand()) != 1 || a.Hand()[0].Name() != "Wall of Wood" || len(a.Hand()[0].Abilities()) != 0 {
		t.Fatal("departure retained copied runtime revision")
	}
}

func TestRONTextActivationStateModal(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Modal_ChooseOne([Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])),ActionList([GainLife(Integer(2))])]))`)
	g, a, b := ronModalGame(0)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	a.targets = []uuid.UUID{red.ID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	id := source.RuntimeAbilities[0].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedActivatedAbility(source, id, change); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("revised mode retained old filter")
	}
	a.targets = []uuid.UUID{green.ID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.mode = 1
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(red.ID()) != nil || g.FindPermanent(green.ID()) != nil {
		t.Fatal("queued modes lost their bodies or choices")
	}
}
