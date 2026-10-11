package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronControlExchange = `SpellActions(Targeted([TargetPermanent(AnyPermanent),TargetPermanent(AnyPermanent)],ActionList([ExchangeControl(Ref_TargetPermanent1,Ref_TargetPermanent2),PlayerAction(You,GainLife(Integer(1)))])))`

func TestRONControlExchangeGameplay(t *testing.T) {
	for _, change := range []string{"none", "same", "blocked", "phase", "phase-cycle", "return", "control", "shroud"} {
		t.Run(change, func(t *testing.T) {
			f := permanentScopeFactory(t, "Instant", ronControlExchange)
			g, a, b := ronGame()
			firstCard := mage.NewCreature("First", "", 2, 3)
			firstCard.SetOwner(a.PlayerID())
			secondCard := mage.NewArtifact("Second", "")
			secondCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(firstCard, a.PlayerID())
			second := g.PutOnBattlefield(secondCard, b.PlayerID())
			spell := f()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "same":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "blocked":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.AttrCantChangeControl)
			case "phase":
				g.PhaseOut(second.ID())
			case "phase-cycle":
				g.PhaseOut(second.ID())
				g.PhaseIn([]uuid.UUID{second.ID()})
			case "return":
				g.BouncePermanentToHand(second)
				b.RemoveFromHand(secondCard.ID())
				second = g.PutOnBattlefield(secondCard, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "shroud":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
			}
			beforeFirst, beforeSecond := g.FindPermanentIncludingPhased(first.ID()).ControllerID(), g.FindPermanentIncludingPhased(second.ID()).ControllerID()
			g.ResolveStack()
			success := change == "none" || change == "phase-cycle"
			wantFirst, wantSecond := beforeFirst, beforeSecond
			if success {
				wantFirst, wantSecond = beforeSecond, beforeFirst
			}
			if g.FindPermanentIncludingPhased(first.ID()).ControllerID() != wantFirst || g.FindPermanentIncludingPhased(second.ID()).ControllerID() != wantSecond {
				t.Fatal("incorrect exchange")
			}
			if a.Life() != 21 {
				t.Fatal("failed exchange stopped continuation")
			}
			clone.ResolveStack()
			if clone.FindPermanent(first.ID()).ControllerID() != b.PlayerID() || clone.FindPermanent(second.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone lost target bindings")
			}
		})
	}
}

func TestRONControlExchangeInvalid(t *testing.T) {
	for _, body := range []string{`ExchangeControl`, `ExchangeControl(ThisPermanent)`, `ExchangeControl(ThisPermanent,ThisPermanent,ThisPermanent)`, `ExchangeControl(ThisPermanent(),ThisPermanent)`, `ExchangeControl(You,ThisPermanent)`, `ExchangeControl(TheChosenPermanent,ThisPermanent)`, `ExchangeControl(Ref_TargetPermanent1,Ref_TargetPermanent2)`, `ExchangeControl(ThePermanentChosenByPlayerThisWay(You),ThisPermanent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
}

func TestRONControlExchangeGauntletsProjection(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Gauntlets of Chaos")
	action := record.Fields["Rules"].Args[0].Args[1].Args[1].Args[0].Args[0]
	if action.Name != "ExchangeControl" || len(action.Args) != 2 {
		t.Fatal("source exchange changed")
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Projection",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+ronControlExchange+`])`), "projection.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes[0].Fields["Rules"].Args[0].Args[0].Args[1].Args[0].Args[0] = action
	f, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first, second := f(), f()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared declaration")
	}
	if err = mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewArtifact("First", ""), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 1), b.PlayerID())
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	if err = g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{p.ID(), q.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if p.ControllerID() != b.PlayerID() || q.ControllerID() != a.PlayerID() {
		t.Fatal("real RON exchange failed")
	}
	if _, err = Compile(record); err == nil {
		t.Fatal("incomplete card accepted")
	}
}

func TestRONControlExchangeActivationAndCopies(t *testing.T) {
	f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPermanent(AnyPermanent)],ActionList([ExchangeControl(ThisPermanent,Ref_TargetPermanent)])))`)
	for _, change := range []string{"none", "depart", "return", "phase", "phase-cycle", "removed"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := f()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			target := g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("unpaid activation succeeded")
			}
			if g.StackSize() != 0 {
				t.Fatal("failed cost left a pending body")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "none" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				g.ResolveTopOfStack()
				if source.ControllerID() != b.PlayerID() || target.ControllerID() != a.PlayerID() {
					t.Fatal("copy did not exchange")
				}
			}
			switch change {
			case "depart":
				g.BouncePermanentToHand(source)
			case "return":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(card.ID())
				source = g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseOut(source.ID())
				g.PhaseIn([]uuid.UUID{source.ID()})
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			success := change == "phase-cycle" || change == "removed"
			wantSource, wantTarget := a.PlayerID(), b.PlayerID()
			if success {
				wantSource, wantTarget = wantTarget, wantSource
			}
			if p := g.FindPermanentIncludingPhased(source.ID()); p != nil && p.ControllerID() != wantSource {
				t.Fatal("source identity or copy changed exchange")
			}
			if target.ControllerID() != wantTarget {
				t.Fatal("partial exchange after source change")
			}
		})
	}
}

func TestRONControlExchangeChoiceAndRoles(t *testing.T) {
	f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([ChooseAPermanent(IsCardtype(Creature)),ExchangeControl(ThisPermanent,TheChosenPermanent)]))`)
	g, a, b := ronGame()
	card := f()
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	target := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 1, mage.WithKeyword(core.Shroud)), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if source.ControllerID() != b.PlayerID() || target.ControllerID() != a.PlayerID() {
		t.Fatal("non-target choice failed")
	}
	for _, rule := range []string{`Activated(ExchangeControl(ThisPermanent,ThisPermanent),ActionList([DrawACard]))`, `AsPermanentEnters(ThisPermanent,[ExchangeControl(ThisPermanent,ThisPermanent)])`, `PermanentLayerEffect(ThisPermanent,[ExchangeControl(ThisPermanent,ThisPermanent)])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "roles.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("wrong context accepted")
		}
	}
}

func TestRONControlExchangeTriggerAndDelay(t *testing.T) {
	for _, context := range []string{"trigger", "delayed"} {
		for _, change := range []string{"none", "return"} {
			t.Run(context+"/"+change, func(t *testing.T) {
				body := `ChooseAPermanent(IsCardtype(Creature)),ExchangeControl(ThisPermanent,TheChosenPermanent)`
				rule := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([` + body + `]))`
				if context == "delayed" {
					rule = `Activated(PayMana([]),ActionList([ChooseAPermanent(IsCardtype(Creature)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ExchangeControl(ThisPermanent,TheChosenPermanent)]))]))`
				}
				f := permanentScopeFactory(t, "Artifact", rule)
				g, a, b := ronGame()
				card := f()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				targetCard := mage.NewCreature("Other", "", 1, 1)
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				if context == "trigger" {
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				} else {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					if target.ControllerID() != b.PlayerID() {
						t.Fatal("delay executed early")
					}
					if change == "return" {
						g.BouncePermanentToHand(target)
						b.RemoveFromHand(targetCard.ID())
						target = g.PutOnBattlefield(targetCard, b.PlayerID())
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				}
				if change == "return" {
					g.BouncePermanentToHand(source)
					a.RemoveFromHand(card.ID())
					source = g.PutOnBattlefield(card, a.PlayerID())
				}
				g.ResolveStack()
				wantSource, wantTarget := b.PlayerID(), a.PlayerID()
				if change == "return" {
					wantSource, wantTarget = wantTarget, wantSource
				}
				if source.ControllerID() != wantSource || target.ControllerID() != wantTarget {
					t.Fatal("trigger or delayed visit mismatch")
				}
			})
		}
	}
	entries := map[string]*InventorySignature{}
	record := staticBaseRecord(t, "LEG", "Gauntlets of Chaos")
	walkInventory(record, "Gauntlets of Chaos", "Card", "card", "collection", true, entries)
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name, "Ref_TargetPermanent") && entry.Role == "permanent_reference" {
			count++
		}
	}
	if count < 2 {
		t.Fatal("exchange operand roles missing")
	}
}

func TestRONControlExchangeDuration(t *testing.T) {
	f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([ExchangeControl(ThisPermanent,Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	card := f()
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	targetCard := mage.NewArtifact("Other", "")
	targetCard.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(targetCard, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.PhaseOut(target.ID())
	g.PhaseIn([]uuid.UUID{target.ID()})
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.ApplyContinuousEffects()
	if target.ControllerID() != a.PlayerID() || returned.ControllerID() != a.PlayerID() {
		t.Fatal("source departure ended exchange or returned source inherited it")
	}
	g.BouncePermanentToHand(target)
	b.RemoveFromHand(targetCard.ID())
	target = g.PutOnBattlefield(targetCard, b.PlayerID())
	g.ApplyContinuousEffects()
	if target.ControllerID() != b.PlayerID() {
		t.Fatal("returned target inherited old exchange")
	}
}
