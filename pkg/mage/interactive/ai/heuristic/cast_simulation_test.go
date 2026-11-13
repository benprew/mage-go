package heuristic

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestCastSimulationPlansCrusadeAndLions(t *testing.T) {
	g, alice, bob := makeGame()
	g.SetActivePlayerIndex(1)
	g.SetStep(core.PrecombatMain)
	bob.SetLife(14)
	addLands(g, bob, "Plains", 3)
	addLands(g, bob, "Forest", 1)
	for _, name := range []string{"Savannah Lions", "White Knight", "Llanowar Elves"} {
		perm := mage.NewPermanent(createCard(t, name, bob.PlayerID()), bob.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(perm)
	}
	mammoth := mage.NewPermanent(createCard(t, "War Mammoth", alice.PlayerID()), alice.PlayerID())
	mammoth.RevokeBaseAttr(core.AttrSummonSick)
	mammoth.Tapped = true
	aura := mage.NewPermanent(createCard(t, "Divine Transformation", alice.PlayerID()), alice.PlayerID())
	g.AddToBattlefield(mammoth, aura)
	g.Attach(aura.ID(), mammoth.ID())
	g.AddToBattlefield(makePerm("Savannah Lions", "{W}", 2, 1, alice.PlayerID()))
	for _, name := range []string{"Crusade", "Savannah Lions", "White Knight"} {
		bob.AddToHand(createCard(t, name, bob.PlayerID()))
	}
	s := New(ai.AggroWeighted)
	for i := range 2 {
		action := s.PriorityAction(bob, g, 1, true)
		if action.Type != interactive.ActionCastSpell || (action.CardName != "Crusade" && action.CardName != "Savannah Lions") {
			t.Fatalf("play %d: expected Crusade or Lions, got %+v", i, action)
		}
		t.Logf("%s casts %s", bob.Name(), action.CardName)
		if err := g.CastSpellByID(bob.PlayerID(), action.CardID, action.Targets, action.XValue); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	powers := 0
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == bob.PlayerID() && perm.HasType(core.TypeCreature) && perm.CurrentPower(g) == 3 {
			powers++
		}
	}
	if powers != 3 {
		t.Fatalf("expected three 3-power creatures, got %d", powers)
	}
}

func TestCastSimulationResolvesETBWithoutChangingLiveGame(t *testing.T) {
	g, p, _ := makeGame()
	g.SetStep(core.PostcombatMain)
	p.SetLife(5)
	addLands(g, p, "Forest", 2)
	plain := mage.NewCreature("Plain", "{1}{G}", 2, 2)
	healer := mage.NewCreature("Healer", "{1}{G}", 2, 2,
		mage.WithAbility(mage.EntersBattlefieldTrigger(mage.GainLife(3), false)))
	for _, card := range []*mage.Card{plain, healer} {
		card.SetOwner(p.PlayerID())
		p.AddToHand(card)
	}
	action := New(ai.MidrangeWeighted).PriorityAction(p, g, 1, true)
	if action.Type != interactive.ActionCastSpell || action.CardID != healer.ID() {
		t.Fatalf("expected healer, got %+v", action)
	}
	if p.Life() != 5 || len(p.Hand()) != 2 || len(g.AllBattlefield()) != 2 || len(g.StackObjects()) != 0 {
		t.Fatal("simulation changed live state")
	}
	for _, permanent := range g.AllBattlefield() {
		if permanent.Tapped {
			t.Fatal("simulation tapped a live mana source")
		}
	}
}

func TestCastSimulationResolvesAura(t *testing.T) {
	g, p, _ := makeGame()
	g.SetStep(core.PostcombatMain)
	addLands(g, p, "Plains", 1)
	creature := makePerm("Creature", "{W}", 2, 2, p.PlayerID())
	g.AddToBattlefield(creature)
	aura := createCard(t, "Holy Strength", p.PlayerID())
	p.AddToHand(aura)
	clone := simulatePermanentCast(g, p.PlayerID(), aura, []uuid.UUID{creature.ID()}, 0)
	if clone == nil {
		t.Fatal("could not simulate Aura")
	}
	boosted := clone.FindPermanent(creature.ID())
	if boosted.CurrentPower(clone) != 3 || boosted.CurrentToughness(clone) != 4 {
		t.Fatalf("unexpected Aura result: %d/%d", boosted.CurrentPower(clone), boosted.CurrentToughness(clone))
	}
	if creature.CurrentPower(g) != 2 || creature.CurrentToughness(g) != 2 || len(p.Hand()) != 1 {
		t.Fatal("Aura simulation changed live state")
	}
}

func TestCastSimulationKeepsContextualEvasion(t *testing.T) {
	var values [2]int
	for i, keyword := range []core.Attr{core.FirstStrike, core.Reach} {
		g, p, opponent := makeGame()
		g.SetStep(core.PostcombatMain)
		addLands(g, p, "Island", 2)
		g.AddToBattlefield(makePerm("Blocker", "{G}", 2, 3, opponent.PlayerID(), mage.WithKeyword(keyword)))
		flyer := mage.NewCreature("Flyer", "{1}{U}", 2, 2, mage.WithKeyword(core.Flying))
		flyer.SetOwner(p.PlayerID())
		p.AddToHand(flyer)
		var ok bool
		values[i], ok = New(ai.MidrangeWeighted).permanentCastValue(g, p.PlayerID(), flyer, nil, 0)
		if !ok {
			t.Fatal("could not simulate flyer")
		}
	}
	if values[0]-values[1] < 2 {
		t.Fatalf("expected unanswered flyer to be worth more: ground %d, reach %d", values[0], values[1])
	}
}

func TestCastSimulationKeepsSmallPositiveValue(t *testing.T) {
	g, p, _ := makeGame()
	g.SetStep(core.PostcombatMain)
	mox := mage.NewArtifact("Mana source", "{0}", mage.WithManaAbility(core.White))
	mox.SetOwner(p.PlayerID())
	p.AddToHand(mox)
	action := New(ai.BurnWeighted).PriorityAction(p, g, 1, true)
	if action.Type != interactive.ActionCastSpell || action.CardID != mox.ID() {
		t.Fatalf("expected a free mana source to be cast, got %+v", action)
	}
}

func TestRemovalHandValueAllowsWinningCombat(t *testing.T) {
	g, p, opponent := makeGame()
	g.SetStep(core.DeclareBlockers)
	addLands(g, p, "Plains", 1)
	addLands(g, opponent, "Forest", 2)
	bird := mage.NewPermanent(createCard(t, "Birds of Paradise", opponent.PlayerID()), opponent.PlayerID())
	g.AddToBattlefield(bird)
	attacker := makePerm("Flyer", "{U}", 2, 2, p.PlayerID(), mage.WithKeyword(core.Flying), mage.WithKeyword(core.Trample))
	g.AddToBattlefield(attacker)
	g.GetCombat().AddAttacker(attacker.ID(), opponent.PlayerID())
	g.GetCombat().AddBlocker(bird.ID(), attacker.ID())
	opponent.SetLife(2)
	removal := createCard(t, "Swords to Plowshares", p.PlayerID())
	p.AddToHand(removal)
	if value, ok := New(ai.AggroWeighted).removalCastValue(g, p.PlayerID(), removal, []uuid.UUID{bird.ID()}, 0); !ok || value <= 0 {
		t.Fatalf("expected winning combat to exceed hand value, got %d, %v", value, ok)
	}
}

func TestPreserveLastRemovalAgainstRedundantManaCreature(t *testing.T) {
	for _, step := range []core.PhaseStep{core.Upkeep, core.DeclareAttackers, core.DeclareBlockers, core.CombatDamage, core.PostcombatMain} {
		g, bob, alice := makeGame()
		g.SetStep(step)
		addLands(g, bob, "Plains", 1)
		addLands(g, alice, "Forest", 2)
		for range 2 {
			g.AddToBattlefield(mage.NewPermanent(createCard(t, "Birds of Paradise", alice.PlayerID()), alice.PlayerID()))
		}
		bob.AddToHand(createCard(t, "Swords to Plowshares", bob.PlayerID()))
		action := New(ai.AggroWeighted).PriorityAction(bob, g, 1, step.IsMainPhase())
		if action.Type != interactive.ActionPass {
			t.Fatalf("%s: expected to keep last removal, got %+v", step, action)
		}
	}
}
