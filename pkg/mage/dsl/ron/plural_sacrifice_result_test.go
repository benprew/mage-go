package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPluralSacrificeResultSword(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		for _, kind := range []string{"player", "creature", "planeswalker"} {
			t.Run(kind+string(rune('0'+count)), func(t *testing.T) {
				a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: count}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "LEG", "Sword of the Ages")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Red, 6)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || !source.Tapped {
					t.Fatal("cast or tapped entry failed")
				}
				g.UntapPermanent(source)
				for i := range 2 {
					c := mage.NewCreature("Paid", "", i+2, 5)
					c.SetOwner(b.PlayerID())
					g.PutOnBattlefield(c, a.PlayerID())
				}
				target := b.PlayerID()
				if kind == "creature" {
					target = g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 10), b.PlayerID()).ID()
				}
				if kind == "planeswalker" {
					target = g.PutOnBattlefield(mage.NewPlaneswalker("Target", "", 10), b.PlayerID()).ID()
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
					t.Fatal(err)
				}
				if len(g.GetExile()) != 0 || len(a.Graveyard()) != 1 || len(b.Graveyard()) != count {
					t.Fatal("payment or exile timing")
				}
				clone := g.Clone()
				g.ResolveStack()
				amount := []int{0, 2, 5}[count]
				if kind == "player" && b.Life() != 20-amount {
					t.Fatal("damage", b.Life(), amount)
				}
				if kind == "creature" && g.FindPermanent(target).Damage != amount {
					t.Fatal("creature damage")
				}
				if kind == "planeswalker" && g.FindPermanent(target).Counters[core.Loyalty] != 10-amount {
					t.Fatal("loyalty damage")
				}
				if len(g.GetExile()) != count+1 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
					t.Fatal("source or creature cards not exiled")
				}
				if len(clone.GetExile()) != 0 {
					t.Fatal("clone changed")
				}
				clone.ResolveStack()
				if len(clone.GetExile()) != count+1 {
					t.Fatal("clone lost results")
				}
			})
		}
	}
}

func TestRONPluralSacrificeResultInvalid(t *testing.T) {
	cost := `SacrificeAnyNumberOfPermanents(And([ControlledByAPlayer(SinglePlayer(You)),IsCardtype(Creature)]))`
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([GainLife(TotalPowerOfPermanents(ThePermanentsSacrificedThisWay))]))`,
		`Activated(` + cost + `,ActionList([GainLife(TotalPowerOfPermanents(ThePermanentsSacrificedThisWay()))]))`,
		`Activated(` + cost + `,ActionList([GainLife(TotalPowerOfPermanents(ThisPermanent))]))`,
		`Activated(` + cost + `,ActionList([GainLife(TotalPowerOfPermanents())]))`,
		`Activated(PayMana([]),ActionList([Exile([EachCardInGraveyards(ThePermanentsSacrificedThisWay)],[])]))`,
		`Activated(` + cost + `,ActionList([Exile([CardInGraveyards(ThisPermanent),EachCardInGraveyards(ThePermanentsSacrificedThisWay)],[])]))`,
		`Activated(` + cost + `,ActionList([Exile([EachCardInGraveyards(ThePermanentsSacrificedThisWay())],[])]))`,
		`Activated(` + cost + `,ActionList([Exile([EachCardInGraveyards(ThePermanentsSacrificedThisWay)],[ExilesFaceDown])]))`,
		`AsPermanentEnters(ThisPermanent,[GainLife(TotalPowerOfPermanents(ThePermanentsSacrificedThisWay))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid-result.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONPluralSacrificeResultIdentityAndLegality(t *testing.T) {
	for _, change := range []string{"source-return", "creature-return", "target-return", "target-phase", "target-shroud", "phase-cycle", "counter", "prevent", "copy", "token", "negative"} {
		t.Run(change, func(t *testing.T) {
			a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 1}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEG", "Sword of the Ages")
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.UntapPermanent(source)
			power := 4
			if change == "negative" {
				power = -2
			}
			victimCard := mage.NewCreature("Paid", "", power, 5)
			victim := g.PutOnBattlefield(victimCard, a.PlayerID())
			if change == "token" {
				g.MutablePermanent(victim.ID()).IsToken = true
			}
			targetCard := mage.NewCreature("Target", "", 1, 20)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source-return":
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "creature-return":
				a.RemoveFromGraveyard(victimCard.ID())
				g.PutOnBattlefield(victimCard, a.PlayerID())
				g.MutablePermanent(victimCard.ID()).AddCounter(core.P1P1, 8)
			case "target-return":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase", "phase-cycle":
				g.PhaseOut(target.ID())
				if change == "phase-cycle" {
					g.PhaseIn([]uuid.UUID{target.ID()})
				}
			case "target-shroud":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
			case "prevent":
				if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.This, mage.AnyPermanent), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			damage, exiled := 4, 2
			switch change {
			case "source-return", "creature-return", "token":
				exiled = 1
			case "target-return", "target-phase", "target-shroud", "counter":
				damage, exiled = 0, 0
			case "prevent", "negative":
				damage = 0
			case "copy":
				damage = 8
			}
			actual := g.FindPermanentIncludingPhased(target.ID())
			if actual == nil || actual.Damage != damage || len(g.GetExile()) != exiled {
				t.Fatal("result", actual, len(g.GetExile()), damage, exiled)
			}
			if change == "source-return" && g.FindPermanent(card.ID()) == nil {
				t.Fatal("followed source return")
			}
			if change == "creature-return" && g.FindPermanent(victimCard.ID()) == nil {
				t.Fatal("followed creature return")
			}
		})
	}
}

func TestRONPluralSacrificeResultFrames(t *testing.T) {
	cost := `SacrificeAnyNumberOfPermanents(And([ControlledByAPlayer(SinglePlayer(You)),IsCardtype(Creature)]))`
	body := `GainLife(TotalPowerOfPermanents(ThePermanentsSacrificedThisWay)),Exile([EachCardInGraveyards(ThePermanentsSacrificedThisWay)],[])`
	for _, delay := range []bool{false, true} {
		actions := `ActionList([` + body + `])`
		if delay {
			actions = `ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,` + actions + `)])`
		}
		factory := permanentScopeFactory(t, "Artifact", `Activated(`+cost+`,`+actions+`)`)
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 1}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		first, second := factory(), factory()
		g.PutOnBattlefield(first, a.PlayerID())
		g.PutOnBattlefield(second, a.PlayerID())
		g.PutOnBattlefield(mage.NewCreature("Paid", "", 3, 3), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		a.answer = 0
		for _, card := range []mage.Card{first, second} {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
		}
		clone := g.Clone()
		for _, game := range []*mage.Game{g, clone} {
			game.ResolveStack()
			if delay {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
			}
			if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 || len(game.GetExile()) != 1 {
				t.Fatal("payment frame overwritten")
			}
		}
	}
}
