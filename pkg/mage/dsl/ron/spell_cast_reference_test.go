package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellCastReferenceRealCards(t *testing.T) {
	for _, name := range []string{"Presence of the Master", "Nether Void"} {
		for _, scenario := range []string{"counter", "pay", "decline", "short", "mana-source", "departed", "returned", "control", "spell-control", "spell-type", "spell-copy", "phased", "phased-before", "other-caster", "nonmatching", "uncounterable", "removed", "recast", "copy", "failed-cast"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: scenario != "decline" && scenario != "counter"}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				g.SetActivePlayerIndex(1)
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if scenario == "phased-before" {
					g.PhaseOut(source.ID())
				}
				caster := b.BasePlayer
				if scenario == "other-caster" {
					caster = a.BasePlayer
					g.SetActivePlayerIndex(0)
				}
				spellName := "Crusade"
				if scenario == "nonmatching" {
					spellName = "Sol Ring"
				}
				spell := repositoryCard(t, "LEB", spellName)
				spell.SetOwner(caster.PlayerID())
				caster.AddToHand(spell)
				if scenario != "failed-cast" {
					caster.ManaPool().Add(core.White, 2)
					caster.ManaPool().Add(core.Colorless, 1)
				}
				err := g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0)
				victim := g.StackPeek()
				g.PutTriggersOnStack()
				if scenario == "failed-cast" {
					if err == nil || g.StackSize() != 0 {
						t.Fatal("failed cast changed stack")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				matching := scenario != "phased-before" && (name != "Presence of the Master" || scenario != "nonmatching")
				want := 1
				if matching {
					want++
				}
				if g.StackSize() != want || a.offers != 0 || b.offers != 0 {
					t.Fatal("trigger timing or filter")
				}
				if !matching {
					return
				}
				trigger := g.StackPeek()
				if len(trigger.Targets) != 0 {
					t.Fatal("event reference became a target")
				}
				a.ManaPool().Clear()
				b.ManaPool().Clear()
				caster.ManaPool().Add(core.Colorless, 3)
				if scenario == "short" {
					caster.ManaPool().Clear()
					caster.ManaPool().Add(core.Colorless, 2)
				}
				if scenario == "mana-source" {
					caster.ManaPool().Clear()
					for range 3 {
						land := repositoryCard(t, "LEB", "Forest")
						land.SetOwner(caster.PlayerID())
						g.PutOnBattlefield(land, caster.PlayerID())
					}
				}
				var recast, spellCopy *mage.StackObject
				switch scenario {
				case "departed", "returned":
					g.DestroyPermanent(g.FindPermanent(source.ID()))
					if scenario == "returned" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "control":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "spell-type":
					victim.Card.(*mage.BaseCard).CloneFrom(mage.NewArtifact("Changed", ""))
				case "spell-copy":
					g.GetStack().RemoveByID(trigger.ID)
					spellCopy = g.CopyStackObjectDirect(victim, caster.PlayerID(), false)
					if spellCopy == nil {
						t.Fatal("spell copy failed")
					}
					g.PushStack(trigger)
					if g.StackSize() != 3 {
						t.Fatal("spell copy emitted a cast event")
					}
				case "spell-control":
					victim.Controller = a.PlayerID()
				case "phased":
					g.PhaseOut(source.ID())
				case "uncounterable":
					mage.WithUncounterable()(victim.Card.(*mage.BaseCard))
				case "removed", "recast":
					g.GetStack().RemoveByID(victim.ID)
					if scenario == "recast" {
						g.GetStack().RemoveByID(trigger.ID)
						recast = &mage.StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: caster.PlayerID()}
						g.PushStack(recast)
						g.PushStack(trigger)
					}
				case "copy":
					if g.CopyStackObjectDirect(trigger, a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				clone.SetPlayerAt(0, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: a.accept})
				clone.SetPlayerAt(1, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: b.accept})
				before := g.StackSize()
				clone.ResolveTopOfStack()
				if g.StackSize() != before || a.offers != 0 || b.offers != 0 {
					t.Fatal("clone changed live bindings")
				}
				g.ResolveTopOfStack()
				if (clone.GetStack().FindByID(victim.ID) != nil) != (g.GetStack().FindByID(victim.ID) != nil) {
					t.Fatal("clone lost captured spell or payer")
				}
				paid := name == "Nether Void" && scenario != "counter" && scenario != "decline" && scenario != "short"
				absent := scenario == "removed" || scenario == "recast"
				survives := paid || scenario == "uncounterable"
				if !absent && (g.GetStack().FindByID(victim.ID) != nil) != survives {
					t.Fatalf("wrong counter outcome: paid=%v stack=%d", paid, g.StackSize())
				}
				if spellCopy != nil && g.GetStack().FindByID(spellCopy.ID) == nil {
					t.Fatal("event reference followed spell copy")
				}
				if recast != nil && g.GetStack().FindByID(recast.ID) == nil {
					t.Fatal("event followed a new cast")
				}
				if !absent && !survives && !slices.ContainsFunc(g.GetPlayer(caster.PlayerID()).Graveyard(), func(c mage.Card) bool { return c.ID() == spell.ID() }) {
					t.Fatal("counter missed owner's graveyard")
				}
				if name == "Nether Void" && scenario != "short" {
					payer := b.offers
					other := a.offers
					if caster == a.BasePlayer {
						payer, other = other, payer
					}
					if payer != 1 || other != 0 {
						t.Fatalf("wrong captured payer: %d/%d", payer, other)
					}
				}
				if scenario == "short" && g.GetPlayer(caster.PlayerID()).ManaPool().TotalMana() != 2 {
					t.Fatal("failed payment consumed mana")
				}
				if scenario == "copy" {
					g.ResolveTopOfStack()
				}
			})
		}
	}
}

func TestRONSpellCastReferenceInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([CounterSpell(Trigger_ThatSpell)]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([PlayerAction(ControllerOfSpell(Trigger_ThatSpell),GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([PlayerAction(Trigger_ThatPlayer,LoseLife(Integer(1)))]))`,
		`Activated(PayMana([]), ActionList([PlayerMayCost(Trigger_ThatPlayer, PayMana([])), Unless(CostWasPaid,[SacrificePermanent(ThisPermanent)])]))`,
		`TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([CounterSpell(Trigger_ThatSpell)]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([CounterSpell(Trigger_ThatSpell())]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([CounterSpell(Trigger_ThatPlayer)]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([PlayerAction(Trigger_ThatSpell,DrawACard)]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([])),TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,DrawACard)]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("invalid scope accepted: %s", rule)
		}
	}
}

func TestRONSpellCastReferenceIndependentEvents(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([PlayerAction(Trigger_ThatPlayer,DrawACard),CounterSpell(Trigger_ThatSpell)]))`)
	g, a, b := ronGame()
	for range 2 {
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	fillRONLibrary(t, a, 3)
	fillRONLibrary(t, b, 3)
	first := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Sol Ring"))
	g.PutTriggersOnStack()
	second := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Dark Ritual"))
	g.PutTriggersOnStack()
	for range 2 {
		g.ResolveTopOfStack()
	}
	if len(b.Hand()) != 2 || len(a.Hand()) != 0 || g.GetStack().FindByID(second.ID) != nil || g.GetStack().FindByID(first.ID) == nil {
		t.Fatal("second event replaced first event bindings")
	}
	for range 2 {
		g.ResolveTopOfStack()
	}
	if len(a.Hand()) != 2 || len(b.Hand()) != 2 || g.StackSize() != 0 {
		t.Fatal("independent event bindings lost")
	}
}

func TestRONSpellCastReferenceDeclarationAndTarget(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsCardtype(Instant)),Targeted([TargetSpell(IsCardtype(Artifact))],ActionList([PlayerAction(Trigger_ThatPlayer,DrawACard),CounterSpell(Ref_TargetSpell),CounterSpell(Trigger_ThatSpell)])))`)
	for range 2 {
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		trigger := card.Abilities()[0].(*mage.GenericTriggered)
		unbound := mage.NewArtifact("Unbound", "", mage.WithAbility(mage.NewTriggered(core.EvtSpellCast, false, trigger.Effects()...)))
		if err := mage.ValidateCardDefinition(unbound.Name(), unbound); err == nil || !strings.Contains(err.Error(), "producer") {
			t.Fatal("event reads lack producer validation")
		}
	}
	g, a, b := ronGame()
	fillRONLibrary(t, b, 2)
	first := castRONSpell(t, g, a, repositoryCard(t, "LEB", "Sol Ring"))
	card := factory()
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	second := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Dark Ritual"))
	g.PutTriggersOnStack()
	if g.StackSize() != 3 || len(g.StackPeek().Targets) != 1 || g.StackPeek().Targets[0] != first.ID {
		t.Fatal("target and event declarations overlap")
	}
	g.ResolveTopOfStack()
	if g.StackSize() != 0 || len(b.Hand()) != 1 || len(a.Hand()) != 0 || g.GetStack().FindByID(second.ID) != nil {
		t.Fatal("event reads lost inside target scope")
	}
}

func TestRONSpellCastReferenceCasterAndCurrentController(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([PlayerAction(Trigger_ThatPlayer,DrawACard),PlayerAction(ControllerOfSpell(Trigger_ThatSpell),DiscardACard)]))`)
	g, a, b := ronGame()
	fillRONLibrary(t, b, 2)
	handCard := repositoryCard(t, "LEB", "Forest")
	handCard.SetOwner(a.PlayerID())
	a.AddToHand(handCard)
	card := factory()
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	victim := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Dark Ritual"))
	g.PutTriggersOnStack()
	victim.Controller = a.PlayerID()
	g.ResolveTopOfStack()
	if len(b.Hand()) != 1 || len(a.Hand()) != 0 || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != handCard.ID() {
		t.Fatal("caster snapshot replaced by current controller")
	}
	if g.GetStack().FindByID(victim.ID) == nil {
		t.Fatal("player action countered spell")
	}
}
