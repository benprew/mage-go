package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExtraTurnTimeWalk(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		factory, err := Compile(staticBaseRecord(t, set, "Time Walk"))
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.HasExtraTurns() || len(a.Hand()) != 1 {
			t.Fatal("failed casting changed turn schedule")
		}
		a.ManaPool().Add(core.Blue, 1)
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if g.HasExtraTurns() {
			t.Fatal("extra turn ran at announcement")
		}
		clone := g.Clone()
		for _, game := range []*mage.Game{g, clone} {
			game.ResolveStack()
			if len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
				t.Fatal("Time Walk did not finish resolution")
			}
			game.AdvanceToNextTurn()
			if game.ActivePlayerObj().PlayerID() != a.PlayerID() {
				t.Fatal("Time Walk did not grant its controller an extra turn")
			}
			game.AdvanceToNextTurn()
			if game.ActivePlayerObj().PlayerID() != b.PlayerID() {
				t.Fatal("Time Walk changed normal turn order")
			}
		}
	}
}

func TestRONExtraTurnContexts(t *testing.T) {
	for _, state := range []string{"present", "departed", "returned", "phased"} {
		g, a, b := ronGame()
		card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerActions(Ref_TargetPlayer,[TakeAnExtraTurn]),TakeAnExtraTurn])))`)()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if g.HasExtraTurns() {
			t.Fatal("activation executed before resolution")
		}
		switch state {
		case "departed", "returned":
			g.DestroyPermanent(p)
			if state == "returned" {
				g.PutOnBattlefield(p.Card, a.PlayerID())
			}
		case "phased":
			g.PhaseOut(p.ID())
		}
		g.ResolveStack()
		for _, want := range []uuid.UUID{a.PlayerID(), b.PlayerID(), b.PlayerID()} {
			g.AdvanceToNextTurn()
			if g.ActivePlayerObj().PlayerID() != want {
				t.Fatalf("%s: lost explicit player, source independence or normal order", state)
			}
		}
	}
}

func TestRONExtraTurnInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([TakeAnExtraTurn()]))`,
		`SpellActions(ActionList([TakeAnExtraTurn(You)]))`,
		`SpellActions(ActionList([PlayerAction(Ref_TargetPlayer,TakeAnExtraTurn)]))`,
		`Activated(TakeAnExtraTurn,ActionList([GainLife(Integer(1))]))`,
		`PermanentLayerEffect(ThisPermanent,[TakeAnExtraTurn])`,
		`AsPermanentEnters(ThisPermanent,[TakeAnExtraTurn])`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONExtraTurnTimeWalkCopyAndCounter(t *testing.T) {
	for _, countered := range []bool{false, true} {
		g, a, b := ronGame()
		card := repositoryCard(t, "LEA", "Time Walk")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 1)
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if countered {
			g.CounterSpellOnStack(card.ID())
			g.ResolveStack()
			if g.HasExtraTurns() {
				t.Fatal("countered Time Walk granted a turn")
			}
			continue
		}
		if g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
			t.Fatal("Time Walk copy failed")
		}
		g.ResolveStack()
		for _, want := range []uuid.UUID{a.PlayerID(), b.PlayerID(), b.PlayerID()} {
			g.AdvanceToNextTurn()
			if g.ActivePlayerObj().PlayerID() != want {
				t.Fatal("copy lost its controller or newest-first order")
			}
		}
	}
}

func TestRONExtraTurnTriggeredAndDelayed(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		g, a, b := ronGame()
		action := `PlayerAction(Trigger_ThatPlayer,TakeAnExtraTurn)`
		if delayed {
			action = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))`
		}
		card := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([`+action+`]))`)()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		g.SetActivePlayerIndex(1)
		g.RunStepWithPriority(core.Upkeep)
		if delayed {
			if g.HasExtraTurns() {
				t.Fatal("delayed turn ran during upkeep")
			}
			g.DestroyPermanent(p)
			g.RunStepWithPriority(core.EndStep)
		}
		if id, ok := g.PopExtraTurn(); !ok || id != b.PlayerID() {
			t.Fatal("trigger lost its event player")
		}
	}
}

func TestRONExtraTurnTimeVaultActivationPrerequisite(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		record := staticBaseRecord(t, set, "Time Vault")
		if _, err := Compile(record); err != nil {
			t.Fatal(err)
		}
		var activation Node
		for _, rule := range record.Fields["Rules"].Args {
			if rule.Name == "Activated" {
				activation = rule
			}
		}
		if activation.Name == "" {
			t.Fatal("record has no activation")
		}
		g, a, _ := ronGame()
		node := nameScopeNode(t, `Card(Name:"Activation fixture",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[])`)
		node.Fields["Rules"] = Node{Kind: List, Args: []Node{activation}}
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		g.SetActivePlayerIndex(1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(p.ID()).Tapped || g.HasExtraTurns() {
			t.Fatal("tap payment or resolution timing failed")
		}
		g.ResolveStack()
		g.AdvanceToNextTurn()
		g.AdvanceToNextTurn()
		if g.ActivePlayerObj().PlayerID() != a.PlayerID() {
			t.Fatal("opponent-turn activation skipped the next normal turn")
		}
	}
}
