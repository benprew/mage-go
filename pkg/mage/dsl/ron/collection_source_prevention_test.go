package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const heroismBody = `ActionForEachPermanent(And([IsAttacking,IsColor(Red),IsCardtype(Creature)]),[PlayerMayCost(ControllerOfPermanent(EachablePermanent),PayMana([ManaCostGeneric(2),ManaCostR])),Unless(CostWasPaid,[CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(EachablePermanent)),[PreventThatDamage],UntilEndOfTurn)])])`

func TestRONCollectionSourcePreventionHeroism(t *testing.T) {
	for _, state := range []string{"decline", "pay-one", "insufficient", "source-leaves", "late-attacker", "stops-attacking", "phase-before", "return-before", "phase-cycle", "return-after", "copy", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state == "pay-one" || state == "insufficient"}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "FEM", "Heroism")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed hand")
			}
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("missing sacrifice accepted")
			}
			foodCard := mage.NewCreature("White cost", "{W}", 1, 10)
			foodCard.SetOwner(a.PlayerID())
			food := g.PutOnBattlefield(foodCard, a.PlayerID())
			firstCard := mage.NewCreature("First red", "{R}", 2, 10, mage.WithKeyword(core.Shroud))
			firstCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second red", "{R}", 2, 10), b.PlayerID())
			blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 2, 10), b.PlayerID())
			idle := g.PutOnBattlefield(mage.NewCreature("Idle red", "{R}", 2, 10), b.PlayerID())
			for _, p := range []*mage.Permanent{first, second, blue, idle} {
				p.RevokeBaseAttr(core.AttrSummonSick)
			}
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{first.ID(), second.ID(), blue.ID()})
			b.ManaPool().Add(core.Red, 1)
			b.ManaPool().Add(core.Colorless, 2)
			if state == "insufficient" {
				b.ManaPool().Clear()
				b.ManaPool().Add(core.Red, 1)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(food.ID()) != nil || b.offers != 0 || a.offers != 0 {
				t.Fatal("sacrifice or payment timing failed")
			}
			switch state {
			case "source-leaves":
				g.RemoveFromBattlefield(source)
			case "late-attacker":
				g.GetCombat().AddAttacker(idle.ID(), a.PlayerID())
			case "stops-attacking":
				g.RemoveFromCombat(first.ID())
			case "phase-before":
				g.PhaseOut(first.ID())
			case "return-before":
				g.RemoveFromBattlefield(first)
				g.PutOnBattlefield(firstCard, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			pending := g.Clone()
			pending.SetPlayerAt(0, &ronRepeatedPaymentClonePlayer{Player: pending.GetPlayer(a.PlayerID())})
			pending.SetPlayerAt(1, &ronRepeatedPaymentClonePlayer{Player: pending.GetPlayer(b.PlayerID())})
			g.ResolveStack()
			expectedFirst := state != "stops-attacking" && state != "phase-before" && state != "return-before"
			if state == "pay-one" {
				expectedFirst = false
			}
			if state == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(first.ID()))
			}
			if state == "return-after" {
				g.RemoveFromBattlefield(first)
				g.PutOnBattlefield(firstCard, b.PlayerID())
				expectedFirst = false
			}
			if state == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
				expectedFirst = false
			}
			if state != "phase-before" && sourceCombatPrevented(g, first.ID(), a.PlayerID()) != expectedFirst {
				t.Fatal("first exact visit prevention failed")
			}
			if sourceCombatPrevented(g, second.ID(), a.PlayerID()) != (state != "cleanup") {
				t.Fatal("second payment inherited first result")
			}
			if sourceCombatPrevented(g, blue.ID(), a.PlayerID()) {
				t.Fatal("blue attacker selected")
			}
			if sourceCombatPrevented(g, idle.ID(), a.PlayerID()) != (state == "late-attacker") {
				t.Fatal("selection did not occur at resolution")
			}
			if a.offers != 0 {
				t.Fatal("wrong controller paid")
			}
			if state == "pay-one" && b.ManaPool().TotalMana() != 0 {
				t.Fatal("payment was not consumed")
			}
			if state == "insufficient" && b.ManaPool().TotalMana() != 1 {
				t.Fatal("failed payment lost mana")
			}
			if state == "decline" {
				if b.offers != 2 {
					t.Fatal("payments were not separate", b.offers)
				}
				pending.ResolveStack()
				if !sourceCombatPrevented(pending, second.ID(), a.PlayerID()) || !sourceCombatPrevented(g, first.ID(), a.PlayerID()) {
					t.Fatal("clone changed bindings")
				}
				before := b.offers
				g.DealDamageToPlayer(a, 1, first.ID())
				if b.offers != before || a.Life() != 19 {
					t.Fatal("damage offered payment or prevented noncombat damage")
				}
			}
		})
	}
}

func TestRONCollectionSourcePreventionShape(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+heroismBody+`]))`)
	if err := mage.ValidateCardDefinition("loop prevention", factory()); err != nil {
		t.Fatal(err)
	}
}

func TestRONCollectionSourcePreventionInvalid(t *testing.T) {
	for _, body := range []string{
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(EachablePermanent)),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(And([])),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(And([IsAttacking()])),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(ThisPermanent())),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(ThisPermanent)),[PreventThatDamage()],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(ThisPermanent)),[PreventThatDamage],UntilEndOfCombat)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid declaration", body)
		}
	}
}
