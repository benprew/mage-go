package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControlExchangeResultGauntlets(t *testing.T) {
	for _, change := range []string{"none", "blocked", "phase", "phase-cycle", "return", "shroud", "copy", "unpaid"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Gauntlets of Chaos"), a.PlayerID())
			left := g.PutOnBattlefield(mage.NewCreature("Left", "", 2, 20), a.PlayerID())
			rightCard := mage.NewCreature("Right", "", 2, 20)
			rightCard.SetOwner(b.PlayerID())
			right := g.PutOnBattlefield(rightCard, b.PlayerID())
			aura1 := g.PutOnBattlefield(mage.NewAura("Aura 1", "", mage.WithCastTarget(mage.TargetCreature())), a.PlayerID())
			aura2 := g.PutOnBattlefield(mage.NewAura("Aura 2", "", mage.WithCastTarget(mage.TargetCreature())), b.PlayerID())
			equipment := g.PutOnBattlefield(mage.NewEquipment("Equipment", ""), a.PlayerID())
			g.Attach(aura1.ID(), left.ID())
			g.Attach(aura2.ID(), right.ID())
			g.Attach(equipment.ID(), left.ID())
			if change != "unpaid" {
				a.ManaPool().Add(core.Colorless, 5)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{left.ID(), right.ID()})
			if change == "unpaid" {
				if err == nil || g.FindPermanent(source.ID()) == nil || g.StackSize() != 0 {
					t.Fatal("unpaid activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("cost payment failed")
			}
			if g.FindPermanent(aura1.ID()) == nil || left.ControllerID() != a.PlayerID() {
				t.Fatal("actions ran before resolution")
			}
			clone := g.Clone()
			switch change {
			case "blocked":
				g.MutablePermanent(left.ID()).GrantBaseAttr(core.AttrCantChangeControl)
			case "phase":
				g.PhaseOut(right.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(right.ID()))
			case "return":
				g.BouncePermanentToHand(right)
				b.RemoveFromHand(rightCard.ID())
				g.PutOnBattlefield(rightCard, b.PlayerID())
			case "shroud":
				g.MutablePermanent(right.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			success := change == "none" || change == "phase-cycle" || change == "copy"
			if (g.FindPermanent(aura1.ID()) == nil) != success {
				t.Fatal("Aura destruction did not use exchange success")
			}
			if g.FindPermanent(equipment.ID()) == nil {
				t.Fatal("destroyed Equipment")
			}
			if success && (g.FindPermanent(left.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(right.ID()).ControllerID() != a.PlayerID()) {
				t.Fatal("exchange failed")
			}
			clone.ResolveStack()
			if clone.FindPermanent(aura1.ID()) != nil || clone.FindPermanent(aura2.ID()) != nil || clone.FindPermanent(equipment.ID()) == nil {
				t.Fatal("clone lost exchange or Aura filter")
			}
		})
	}
}

func TestRONControlExchangeResultInvalid(t *testing.T) {
	for _, body := range []string{
		`If(PermanentsChangedControlThisWay,[DrawACard])`,
		`If(PermanentsChangedControlThisWay,[DrawACard]),ExchangeControl(ThisPermanent,ThisPermanent)`,
		`ExchangeControl(ThisPermanent,ThisPermanent),If(PermanentsChangedControlThisWay(),[DrawACard])`,
		`ExchangeControl(ThisPermanent,ThisPermanent),If(PermanentsChangedControlThisWay,[])`,
		`If(PermanentPassesFilter(ThisPermanent,IsCardtype(Artifact)),[ExchangeControl(ThisPermanent,ThisPermanent)]),If(PermanentsChangedControlThisWay,[DrawACard])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid-exchange-result.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid result accepted", body)
		}
	}
}

func TestRONControlExchangeResultContexts(t *testing.T) {
	for _, tc := range []struct {
		name, body         string
		immediate, delayed int
	}{
		{"latest-failure", `ExchangeControl(ThisPermanent,TheChosenPermanent),ExchangeControl(ThisPermanent,ThisPermanent),If(PermanentsChangedControlThisWay,[PlayerAction(You,GainLife(Integer(2)))])`, 20, 20},
		{"latest-success", `ExchangeControl(ThisPermanent,ThisPermanent),ExchangeControl(ThisPermanent,TheChosenPermanent),If(PermanentsChangedControlThisWay,[PlayerAction(You,GainLife(Integer(2)))])`, 22, 22},
		{"historical", `ExchangeControl(ThisPermanent,TheChosenPermanent),If(PermanentsChangedControlThisWay,[ExchangeControl(ThisPermanent,TheChosenPermanent)]),If(PermanentsChangedControlThisWay,[PlayerAction(You,GainLife(Integer(2)))])`, 22, 22},
		{"delayed", `ExchangeControl(ThisPermanent,TheChosenPermanent),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(PermanentsChangedControlThisWay,[PlayerAction(You,GainLife(Integer(2)))])])),ExchangeControl(ThisPermanent,ThisPermanent)`, 20, 22},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ChooseAPermanent(IsCardtype(Creature)),`+tc.body+`]))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(f(), a.PlayerID())
			g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 20), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != tc.immediate {
				t.Fatal("wrong local exchange result", a.Life())
			}
			clone := g.Clone()
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != tc.delayed {
					t.Fatal("delayed outcome lost its frame")
				}
			}
		})
	}
}

func TestRONControlExchangeResultCopiesAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([ExchangeControl(Ref_TargetPermanent1,Ref_TargetPermanent2),If(PermanentsChangedControlThisWay,[PlayerAction(You,GainLife(Integer(2)))])])))`)
	g, a, b := ronGame()
	left := g.PutOnBattlefield(mage.NewArtifact("Left", ""), a.PlayerID())
	right := g.PutOnBattlefield(mage.NewArtifact("Right", ""), b.PlayerID())
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{left.ID(), right.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopySpellOnStack(spell.ID(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("copied result or controller crossed frames")
	}
	clone.MutablePermanent(left.ID()).GrantBaseAttr(core.AttrCantChangeControl)
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("clone inherited successful outcomes")
	}
	other := factory()
	other.SetOwner(a.PlayerID())
	a.AddToHand(other)
	g.MutablePermanent(right.ID()).GrantBaseAttr(core.AttrCantChangeControl)
	if err := g.CastSpellByID(a.PlayerID(), other.ID(), []uuid.UUID{left.ID(), right.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("factory instance inherited earlier outcomes")
	}
}
