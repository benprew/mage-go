package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticDefendingCountGaea(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(repositoryRecord(t, set, "Gaea's Liege", set+".ron"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			for range 2 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			}
			for range 4 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			a.RemoveFromHand(card.ID())
			castRONSpell(t, g, a, card)
			pending := g.Clone()
			pending.ResolveStack()
			checkConditionalPT(t, pending, card.ID(), 2, 2)
			g.ResolveStack()
			checkConditionalPT(t, g, card.ID(), 2, 2)
			p := g.FindPermanent(card.ID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, []uuid.UUID{land.ID()}); err != nil {
				t.Fatal(err)
			}
			if land.HasSubType("Forest") {
				t.Fatal("land changed before resolution")
			}
			g.ResolveStack()
			if !g.FindPermanent(land.ID()).HasSubType("Forest") {
				t.Fatal("activation did not change land")
			}
			g.UntapPermanent(g.FindPermanent(p.ID()))
			g.SetStep(core.DeclareAttackers)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			if !g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("attack not declared")
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 5, 5)
			clone := g.Clone()
			ids := g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 4, 4)
			checkConditionalPT(t, clone, p.ID(), 5, 5)
			g.PhaseIn(ids)
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 5, 5)
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(4)), mage.SwitchPowerToughness(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 10, 8)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 6, 6)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 1, 1)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 6, 6)
			g.PhaseIn(g.PhaseOut(p.ID()))
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, p.ID(), 3, 3)
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			if g.FindPermanent(land.ID()).HasSubType("Forest") {
				t.Fatal("source departure did not end activation")
			}
			g.PutOnBattlefield(card, a.PlayerID())
			checkConditionalPT(t, g, card.ID(), 2, 2)
		})
	}
}

func TestRONStaticDefendingCountInvalid(t *testing.T) {
	const valid = `Card(Name:"Count",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(TheNumberOfPermanentsOnTheBattlefield(ControlledByPlayer(PlayerOrControllerOfPlaneswalkerCreatureIsAttacking(ThisPermanent))))])])`
	for _, source := range []string{
		strings.ReplaceAll(valid, "PlayerOrControllerOfPlaneswalkerCreatureIsAttacking(ThisPermanent)", "PlayerOrControllerOfPlaneswalkerCreatureIsAttacking"),
		strings.ReplaceAll(valid, "PlayerOrControllerOfPlaneswalkerCreatureIsAttacking(ThisPermanent)", "PlayerOrControllerOfPlaneswalkerCreatureIsAttacking(ThisPermanent,ThisPermanent)"),
		strings.ReplaceAll(valid, "IsAttacking(ThisPermanent)", "IsAttacking(ThisPermanent())"),
		strings.ReplaceAll(valid, "IsAttacking(ThisPermanent)", "IsAttacking(HostPermanent)"),
		strings.ReplaceAll(valid, "IsAttacking(ThisPermanent)", "IsAttacking(Ref_TargetPermanent)"),
		strings.ReplaceAll(valid, "IsAttacking(ThisPermanent)", "IsAttacking(You)"),
	} {
		nodes, err := Parse(strings.NewReader(source), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", source)
		}
	}
	nodes, err := Parse(strings.NewReader(`ControlledByPlayer(PlayerOrControllerOfPlaneswalkerCreatureIsAttacking(ThisPermanent))`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {permanent: true, resolving: true}, {permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
		if _, err := compileStaticAttackerController(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted", ctx)
		}
	}
}

func TestRONStaticDefendingCountDeclaration(t *testing.T) {
	for _, ownForests := range []int{1, 3} {
		g, a, b := ronGame()
		for range ownForests {
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
		}
		for range 4 {
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
		}
		card := repositoryCard(t, "LEB", "Gaea's Liege")
		p := g.PutOnBattlefield(card, a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		g.PutOnBattlefield(mage.NewEnchantment("Restriction", "", mage.WithAbility(mage.Static(mage.CantAttackEach(mage.MatchingPermanents(mage.And(mage.IsCreature, mage.HasPowerGTE(3))))))), b.PlayerID())
		g.ApplyContinuousEffects()
		g.SetStep(core.DeclareAttackers)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		wantAttack := ownForests == 1
		if g.GetCombat().IsAttacking(p.ID()) != wantAttack {
			t.Fatal("attack restriction did not use nonattacking power")
		}
		g.ApplyContinuousEffects()
		if wantAttack {
			checkConditionalPT(t, g, p.ID(), 4, 4)
		} else {
			checkConditionalPT(t, g, p.ID(), 3, 3)
		}
	}
}

func TestRONStaticDefendingCountLiveControlAndWalker(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
	for range 3 {
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	}
	factory, err := Compile(repositoryRecord(t, "LEB", "Gaea's Liege", "LEB.ron"))
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	other := g.PutOnBattlefield(factory(), b.PlayerID())
	checkConditionalPT(t, g, other.ID(), 3, 3)
	walker := g.PutOnBattlefield(mage.NewArtifact("Walker", "", mage.WithCardType(core.TypePlaneswalker)), b.PlayerID())
	g.SetStep(core.DeclareAttackers)
	p := g.PutOnBattlefieldAttacking(card, a.PlayerID(), walker.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, p.ID(), 3, 3)
	g.RemoveFromBattlefield(walker)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, p.ID(), 3, 3)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	checkConditionalPT(t, g, p.ID(), 4, 4)
	checkConditionalPT(t, g, other.ID(), 4, 4)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("control change kept attacker in combat")
	}
	checkConditionalPT(t, g, p.ID(), 4, 4)
}
