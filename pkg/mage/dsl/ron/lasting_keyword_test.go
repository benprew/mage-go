package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONLastingKeywordGravitySphere(t *testing.T) {
	for _, change := range []string{"none", "source-leaves", "source-phase", "source-abilities", "clone", "control", "late-entry", "later-grant"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Gravity Sphere")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, 2)
			q := g.PutOnBattlefield(mage.NewCreature("Flyer", "", 2, 3, mage.WithKeyword(core.Flying), mage.WithKeyword(core.Vigilance)), b.PlayerID())
			noncreature := g.PutOnBattlefield(mage.NewArtifact("Flying artifact", "", mage.WithKeyword(core.Flying)), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || !source.Card.HasSuperType(core.SuperWorld) || a.ManaPool().TotalMana() != 0 || g.FindPermanent(q.ID()).HasKeyword(core.Flying) || !g.FindPermanent(noncreature.ID()).HasKeyword(core.Flying) {
				t.Fatal("wrong casting or static selection")
			}
			clone := g.Clone()
			wantFlying := false
			switch change {
			case "source-leaves":
				g.RemoveFromBattlefield(source)
				wantFlying = true
			case "source-phase":
				g.PhaseOut(source.ID())
				wantFlying = true
			case "source-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantFlying = true
			case "clone":
				clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
				clone.ApplyContinuousEffects()
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "late-entry":
				later := g.PutOnBattlefield(mage.NewCreature("Later flyer", "", 1, 1, mage.WithKeyword(core.Flying)), a.PlayerID())
				if g.FindPermanent(later.ID()).HasKeyword(core.Flying) {
					t.Fatal("static selection omitted later entry")
				}
			case "later-grant":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Flying)), q.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantFlying = true
			}
			g.ApplyContinuousEffects()
			if g.FindPermanent(q.ID()).HasKeyword(core.Flying) != wantFlying || !g.FindPermanent(q.ID()).HasKeyword(core.Vigilance) {
				t.Fatal("wrong static lifetime or timestamp")
			}
			if clone.FindPermanent(q.ID()).HasKeyword(core.Flying) != (change == "clone") {
				t.Fatal("clone layer state crossed games")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if change == "later-grant" && g.FindPermanent(q.ID()).HasKeyword(core.Flying) {
				t.Fatal("expired grant overrode static removal")
			}
			if change == "source-phase" {
				g.PhaseIn([]uuid.UUID{source.ID()})
				g.ApplyContinuousEffects()
				if g.FindPermanent(q.ID()).HasKeyword(core.Flying) {
					t.Fatal("phase-in did not restore static rule")
				}
			}
		})
	}
}

func TestRONLastingKeywordWurmNative(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "LEG", "Elder Land Wurm")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.White, 3)
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	if p.CanDeclareAsAttacker(g) {
		t.Fatal("defender could attack before blocking")
	}
	if p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 5 || !p.HasKeyword(core.Defender) || !p.HasKeyword(core.Trample) {
		t.Fatal("wrong Wurm metadata")
	}
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.SetActivePlayerIndex(1)
	b.SetAttackers(1, []string{"Attacker"})
	a.SetBlockers(1, map[string]string{"Elder Land Wurm": "Attacker"})
	g.RunStepWithPriority(core.DeclareAttackers)
	g.RunStepWithPriority(core.DeclareBlockers)
	if !g.GetCombat().IsBlocking(p.ID()) || g.FindPermanent(p.ID()).HasKeyword(core.Defender) || !g.FindPermanent(p.ID()).HasKeyword(core.Trample) {
		t.Fatal("native block did not remove only defender")
	}
	g.RunStepWithPriority(core.EndCombat)
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(p.ID()).HasKeyword(core.Defender) || !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("indefinite removal expired or defender still restricts attacks")
	}
}

func TestRONLastingKeywordWurmIdentity(t *testing.T) {
	for _, change := range []string{"none", "return-before", "phase-before", "phase-cycle", "control", "copy", "return-after", "phase-after", "abilities-after"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Elder Land Wurm")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: p.ID(), PlayerID: a.PlayerID(), Flag: true})
			if !g.FindPermanent(p.ID()).HasKeyword(core.Defender) {
				t.Fatal("removal ran at detection")
			}
			clone := g.Clone()
			switch change {
			case "return-before":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase-before":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("trigger copy failed")
			}
			g.ResolveStack()
			if change == "phase-before" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			wantDefender := change == "return-before" || change == "phase-before"
			if g.FindPermanent(p.ID()).HasKeyword(core.Defender) != wantDefender {
				t.Fatal("pending trigger lost incarnation or presence")
			}
			switch change {
			case "return-after":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
				wantDefender = true
			case "phase-after":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "abilities-after":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(p.ID()).HasKeyword(core.Defender) != wantDefender {
				t.Fatal("wrong indefinite lifetime")
			}
			clone.ResolveStack()
			if clone.FindPermanent(p.ID()).HasKeyword(core.Defender) {
				t.Fatal("clone lost its pending removal")
			}
		})
	}
}

func TestRONLastingKeywordInvalid(t *testing.T) {
	for _, rule := range []string{
		`EachPermanentLayerEffect(IsCardtype(Creature),[LosesAbility(Flying),AddAbility([Vigilance])])`,
		`EachPermanentLayerEffect(IsCardtype(Land),[LosesAbility(Flying)])`,
		`EachPermanentLayerEffect(And([IsCardtype(Creature),HasAbility(Flying)]),[LosesAbility(Flying)])`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Flying),LosesAbility(Defender)])]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Flying())])]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[LosesAbility()])]))`,

		`EachPermanentLayerEffect(IsCardtype(Creature),[LosesAbility()])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[LosesAbility(Flying())])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[LosesAbility(TheChosenAbility)])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[LosesAbility(AnyBandsWithOthers)])`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[LosesAbility(Flying)])]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Enchant(IsCardtype(Creature)))])]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(Trigger_ThatCreature,[LosesAbility(Defender)])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONLastingKeywordTargetInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[LosesAbility(Flying)])])))`)
	g, a, b := ronGame()
	firstCard, secondCard := factory(), factory()
	if firstCard.Abilities()[0] == secondCard.Abilities()[0] {
		t.Fatal("factory shared ability declarations")
	}
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(secondCard, a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("First flyer", "", 1, 2, mage.WithKeyword(core.Flying)), b.PlayerID())
	r := g.PutOnBattlefield(mage.NewCreature("Second flyer", "", 2, 3, mage.WithKeyword(core.Flying)), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{first.ID()}); err == nil || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("illegal target spent cost")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{r.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(first)
	g.RemoveFromBattlefield(second)
	g.ResolveStack()
	if g.FindPermanent(q.ID()).HasKeyword(core.Flying) || g.FindPermanent(r.ID()).HasKeyword(core.Flying) {
		t.Fatal("instance or copied bindings crossed targets")
	}
	if !clone.FindPermanent(q.ID()).HasKeyword(core.Flying) {
		t.Fatal("live resolution changed clone")
	}
	clone.ResolveStack()
	if clone.FindPermanent(q.ID()).HasKeyword(core.Flying) || clone.FindPermanent(r.ID()).HasKeyword(core.Flying) {
		t.Fatal("clone lost independent target bindings")
	}
}
