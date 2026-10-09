package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRandomHandDiscardRealCards(t *testing.T) {
	for _, name := range []string{"Nebuchadnezzar", "Rag Man"} {
		for _, state := range []string{"present", "departed", "returned", "phased", "control", "zero", "short", "unpaid", "opponent-turn", "self-target", "sick"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, base, b := ronGame()
				a := &ronCardNameTester{Player: base, name: "Island"}
				g.SetPlayerAt(0, a)
				set := "LEG"
				if name == "Rag Man" {
					set = "DRK"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.SetTurn(2)
				if state != "sick" {
					g.RunStepWithPriority(core.Untap)
					p = g.FindPermanent(card.ID())
				}
				g.SetStep(core.PrecombatMain)
				held := []mage.Card{mage.NewLand("Island"), mage.NewCreature("Creature", "", 1, 1), mage.NewLand("Island"), mage.NewCreature("Other creature", "", 1, 1)}
				if state == "short" {
					held = held[:1]
				}
				for _, c := range held {
					c.SetOwner(b.PlayerID())
					b.AddToHand(c)
				}
				a.AddToHand(mage.NewLand("Island"))
				x := 3
				if state == "zero" {
					x = 0
				}
				g.SetXValue(x)
				g.SetRandomResults([]int{2, 0, 0, 9})
				if name == "Rag Man" {
					a.ManaPool().Add(core.Black, 3)
				} else {
					a.ManaPool().Add(core.Colorless, x)
				}
				if state == "unpaid" {
					a.ManaPool().Clear()
				}
				if state == "opponent-turn" {
					g.SetActivePlayerIndex(1)
				}
				if state == "sick" {
					g.SetTurn(1)
				}
				targets := []uuid.UUID{b.PlayerID()}
				if state == "self-target" {
					targets[0] = a.PlayerID()
				}
				mana := a.ManaPool().TotalMana()
				err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets)
				bad := state == "unpaid" || state == "opponent-turn" || state == "self-target" || state == "sick"
				if bad {
					if err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != mana || p.Tapped || a.calls != 0 || len(g.PublicReveals()) != 0 || len(b.Hand()) != len(held) {
						t.Fatal("illegal activation changed state")
					}
					if g.RandIntn(10) != 2 {
						t.Fatal("failed payment consumed randomness")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if a.calls != 0 || len(g.PublicReveals()) != 0 || len(b.Hand()) != len(held) {
					t.Fatal("action ran at announcement")
				}
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(p.ID())
						g.PutOnBattlefield(p.Card, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				clone := g.Clone()
				ca := &ronCardNameTester{Player: clone.GetPlayer(a.PlayerID()), name: "Mountain"}
				clone.SetPlayerAt(0, ca)
				g.SetXValue(99)
				g.ResolveStack()
				clone.ResolveStack()
				reveals, discarded := 3, 2
				if name == "Rag Man" {
					reveals, discarded = len(held), 1
					if state == "short" {
						discarded = 0
					}
				} else if state == "zero" {
					reveals, discarded = 0, 0
				} else if state == "short" {
					reveals, discarded = 1, 1
				}
				if len(g.PublicReveals()) != reveals || len(b.Graveyard()) != discarded || len(b.Hand()) != len(held)-discarded {
					t.Fatalf("reveal/discard mismatch: %d/%d", len(g.PublicReveals()), len(b.Graveyard()))
				}
				calls := 0
				if name == "Nebuchadnezzar" {
					calls = 1
				}
				if a.calls != calls || ca.calls != calls {
					t.Fatal("name choice timing failed")
				}
				cloneDiscards := discarded
				if name == "Nebuchadnezzar" {
					cloneDiscards = 0
				}
				if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != cloneDiscards {
					t.Fatal("clone shared choice or hand")
				}
			})
		}
	}
}

func TestRONRandomHandDiscardInvalid(t *testing.T) {
	for _, action := range []string{
		`RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType`,
		`RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1))`,
		`RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(-1),IsNamed(TheChosenName))`,
		`RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(ValueX,IsNamed(TheChosenName))`,
		`RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsNamed(TheChosenName()))`,
		`RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsCardtype(Creature))`,
		`RevealHandAndDiscardACardOfTypeAtRandom`,
		`RevealHandAndDiscardACardOfTypeAtRandom(IsNamed(TheChosenName))`,
		`RevealHandAndDiscardACardOfTypeAtRandom(IsCardtype(You))`,
		`RevealHandAndDiscardACardOfTypeAtRandom(IsCardtype(Creature),IsCardtype(Creature))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ChooseACardName(AnyOracleCard),PlayerAction(You,`+action+`)]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rule := range []string{
		`SpellActions(ActionList([PlayerAction(You,RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsNamed(TheChosenName))),ChooseACardName(AnyOracleCard)]))`,
		`SpellActions(ActionList([ChooseACardName(AnyOracleCard),PlayerAction(ActionPlayer,RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsNamed(TheChosenName)))]))`,
		`Activated(RevealHandAndDiscardACardOfTypeAtRandom(IsCardtype(Creature)),ActionList([GainLife(Integer(1))]))`,
		`PermanentLayerEffect(ThisPermanent,[RevealHandAndDiscardACardOfTypeAtRandom(IsCardtype(Creature))])`,
		`AsPermanentEnters(ThisPermanent,[RevealHandAndDiscardACardOfTypeAtRandom(IsCardtype(Creature))])`,
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

func TestRONRandomHandDiscardContexts(t *testing.T) {
	for _, kind := range []string{"spell", "trigger", "delayed", "nested"} {
		t.Run(kind, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronCardNameTester{Player: base, name: "Island"}
			g.SetPlayerAt(0, a)
			card := mage.NewLand("Island")
			card.SetOwner(b.PlayerID())
			b.AddToHand(card)
			actions := `ChooseACardName(AnyOracleCard),PlayerAction(Ref_TargetPlayer,RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsNamed(TheChosenName))),GainLife(Integer(2))`
			typ, rule := "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			if kind == "trigger" {
				typ, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			}
			if kind == "delayed" {
				typ, rule = "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([ChooseACardName(AnyOracleCard),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(Ref_TargetPlayer,RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsNamed(TheChosenName))),GainLife(Integer(2))]))])))`
			}
			if kind == "nested" {
				rule = `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([PlayerActions(You,[ChooseACardName(AnyOracleCard),PlayerActions(Ref_TargetPlayer,[RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType(Integer(1),IsNamed(TheChosenName)),RevealHandAndDiscardACardOfTypeAtRandom(IsCardtype(Creature))])]),GainLife(Integer(2))])))`
			}
			c := permanentScopeFactory(t, typ, rule)()
			c.SetOwner(a.PlayerID())
			switch kind {
			case "spell", "nested":
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				g.PutOnBattlefield(c, a.PlayerID())
				g.PutTriggersOnStack()
			case "delayed":
				p := g.PutOnBattlefield(c, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if len(g.PublicReveals()) != 0 || a.calls != 0 {
				t.Fatal("body ran at announcement")
			}
			g.ResolveStack()
			if kind == "delayed" {
				if len(g.PublicReveals()) != 0 || len(b.Hand()) != 1 {
					t.Fatal("delay ran early")
				}
				b.RemoveFromHand(card.ID())
				b.AddToHand(card)
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
				if len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 || len(clone.PublicReveals()) != 0 {
					t.Fatal("clone shared delayed movement")
				}
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
					t.Fatal("clone lost delayed choice")
				}
			}
			if len(b.Graveyard()) != 1 || len(b.Hand()) != 0 || len(g.PublicReveals()) != 1 || a.Life() != 22 || a.calls != 1 {
				t.Fatal("context lost saved name or action ordering")
			}
		})
	}
}

func TestRONRandomHandDiscardCast(t *testing.T) {
	for _, tc := range []struct {
		set, name            string
		generic, black, blue int
	}{{"LEG", "Nebuchadnezzar", 3, 1, 1}, {"DRK", "Rag Man", 2, 2, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid creature cast")
			}
			a.ManaPool().Add(core.Colorless, tc.generic)
			a.ManaPool().Add(core.Black, tc.black)
			a.ManaPool().Add(core.Blue, tc.blue)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("creature entered before resolution")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || !p.HasType(core.TypeCreature) || a.ManaPool().TotalMana() != 0 || len(p.RuntimeAbilities) != 1 {
				t.Fatal("native creature casting lost metadata or ability")
			}
		})
	}
}
