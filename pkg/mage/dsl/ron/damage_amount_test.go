package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONDamageAmountCards(t *testing.T) {
	for _, name := range []string{"El-Hajjâj", "Spirit Link"} {
		for _, scenario := range []string{"player", "permanent", "prevented", "partial", "twice", "other", "departure", "return", "phase-after", "phase-before", "control-after", "control-before", "reattach", "copy", "removed"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				set := "ARN"
				if name == "Spirit Link" {
					set = "LEG"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				host := source
				if name == "Spirit Link" {
					c := mage.NewCreature("Host", "", 3, 9)
					c.SetOwner(b.PlayerID())
					host = g.PutOnBattlefield(c, b.PlayerID())
					g.Attach(source.ID(), host.ID())
				}
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 9), b.PlayerID())
				if scenario == "phase-before" {
					g.PhaseOut(source.ID())
				}
				if scenario == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				if scenario == "control-before" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "prevented" {
					g.AddPreventionShield(b.PlayerID(), 3)
				}
				if scenario == "partial" {
					g.AddPreventionShield(b.PlayerID(), 2)
				}
				damageSource := host.ID()
				if scenario == "other" {
					damageSource = other.ID()
				}
				if scenario == "permanent" {
					g.DealDamageToPermanent(other, 3, damageSource)
				} else {
					g.DealDamageToPlayer(b, 3, damageSource)
				}
				want := 3
				if scenario == "partial" {
					want = 1
				}
				if scenario == "prevented" || scenario == "phase-before" || scenario == "removed" || scenario == "other" {
					want = 0
				}
				if scenario == "twice" {
					g.DealDamageToPlayer(b, 2, damageSource)
					want = 5
				}
				if scenario == "departure" || scenario == "return" {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if scenario == "return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				}
				if scenario == "phase-after" {
					g.PhaseOut(source.ID())
				}
				if scenario == "control-after" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "reattach" && name == "Spirit Link" {
					g.Attach(source.ID(), other.ID())
				}
				if a.Life() != 20 {
					t.Fatal("trigger resolved before priority")
				}
				g.PutTriggersOnStack()
				count := 0
				if want > 0 {
					count = 1
				}
				if scenario == "twice" {
					count = 2
				}
				if g.StackSize() != count {
					t.Fatalf("stack %d, want %d", g.StackSize(), count)
				}
				clone := g.Clone()
				if scenario == "copy" {
					g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				}
				g.ResolveStack()
				gainA, gainB := want, 0
				if scenario == "control-before" {
					gainA, gainB = 0, want
				}
				if scenario == "copy" {
					gainB = want
				}
				lostB := 3
				if scenario == "permanent" {
					lostB = 0
				}
				if scenario == "partial" {
					lostB = 1
				}
				if scenario == "prevented" {
					lostB = 0
				}
				if scenario == "twice" {
					lostB = 5
				}
				if a.Life() != 20+gainA || b.Life() != 20-lostB+gainB {
					t.Fatalf("life %d/%d, want %d/%d", a.Life(), b.Life(), 20+gainA, 20-lostB+gainB)
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				cloneGainB := 0
				if scenario == "control-before" {
					cloneGainB = want
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20+gainA || clone.GetPlayer(b.PlayerID()).Life() != 20-lostB+cloneGainB {
					t.Fatal("clone lost event amount")
				}
			})
		}
	}
}

func TestRONDamageAmountInvalid(t *testing.T) {
	for _, trigger := range []string{`WhenAPermanentDealsDamage`, `WhenAPermanentDealsDamage()`, `WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent),AnyPlayer)`, `WhenAPermanentDealsDamage(ThisPermanent)`, `WhenAPermanentDealsDamage(SinglePermanent(Ref_TargetPermanent))`, `WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent()))`, `WhenAPermanentDealsDamage(SinglePermanent(HostPermanent))`} {
		source := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [TriggerA(` + trigger + `, ActionList([GainLife(Trigger_AmountOfDamageDealt)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", trigger)
		}
	}
	for _, rule := range []string{`Activated(TapPermanent(ThisPermanent),ActionList([GainLife(Trigger_AmountOfDamageDealt)]))`, `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Trigger_AmountOfDamageDealt)]))`, `TriggerA(WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent)),ActionList([GainLife(Trigger_AmountOfDamageDealt())]))`, `TriggerA(WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Trigger_AmountOfDamageDealt))]))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONDamageAmountInventory(t *testing.T) {
	source := `Card(Name: "Amount", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), CardPT: (Power: 1, Toughness: 1), Rules: [TriggerA(WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent)), ActionList([GainLife(Trigger_AmountOfDamageDealt)]))])`
	report, err := InspectFS(fstest.MapFS{"amount.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("compiled %d", report.Compiled)
	}
	for _, s := range report.Signatures {
		if s.Name == "ThisPermanent" && s.Role != "permanent_reference" {
			t.Fatal("wrong subject role")
		}
		if s.Name == "Trigger_AmountOfDamageDealt" && s.Role != "value" {
			t.Fatal("wrong amount role")
		}
	}
}

func TestRONDamageAmountCasting(t *testing.T) {
	for _, name := range []string{"El-Hajjâj", "Spirit Link"} {
		for _, paid := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", name, paid), func(t *testing.T) {
				g, a, b := ronGame()
				set := "ARN"
				if name == "Spirit Link" {
					set = "LEG"
				}
				c := repositoryCard(t, set, name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				var targets []uuid.UUID
				if name == "Spirit Link" {
					p := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
					targets = []uuid.UUID{p.ID()}
				}
				if paid {
					if name == "Spirit Link" {
						a.ManaPool().Add(core.White, 1)
					} else {
						a.ManaPool().Add(core.Black, 3)
					}
				}
				err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0)
				if !paid {
					if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
						t.Fatal("failed payment changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(c.ID())
				if p == nil {
					t.Fatal("card did not enter")
				}
				damageSource := p.ID()
				if name == "Spirit Link" {
					damageSource = targets[0]
				}
				g.DealDamageToPlayer(b, 2, damageSource)
				g.PutTriggersOnStack()
				g.ResolveStack()
				if a.Life() != 22 || b.Life() != 18 {
					t.Fatal("cast card did not execute damage trigger")
				}
			})
		}
	}
}

func TestRONDamageAmountBatchAndInstances(t *testing.T) {
	g, a, b := ronGame()
	firstCard, secondCard := repositoryCard(t, "ARN", "El-Hajjâj"), repositoryCard(t, "3ED", "El-Hajjâj")
	firstCard.SetOwner(a.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first, second := g.PutOnBattlefield(firstCard, a.PlayerID()), g.PutOnBattlefield(secondCard, b.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 9), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	firstSource, secondSource := g.CaptureDamageSource(first.ID()), g.CaptureDamageSource(second.ID())
	g.DealSimultaneousDamage(mage.PlayerDamageAssignment(b, 3, firstSource), mage.PermanentDamageAssignment(victim, 2, firstSource), mage.PlayerDamageAssignment(a, 4, secondSource))
	pending := g.Clone()
	for _, game := range []*mage.Game{g, pending} {
		game.PutTriggersOnStack()
		if game.StackSize() != 3 {
			t.Fatalf("stack %d", game.StackSize())
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("instances or batch amounts shared")
		}
	}
}

func TestRONDamageAmountCombat(t *testing.T) {
	for _, name := range []string{"El-Hajjâj", "Spirit Link"} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", name, blocked), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				set := "ARN"
				if name == "Spirit Link" {
					set = "LEG"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				attacker := source
				if name == "Spirit Link" {
					c := mage.NewCreature("Attacker", "", 2, 4)
					c.SetOwner(a.PlayerID())
					attacker = g.PutOnBattlefield(c, a.PlayerID())
					g.Attach(source.ID(), attacker.ID())
				}
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 4), b.PlayerID())
				a.SetAttackers(1, []string{attacker.Card.Name()})
				if blocked {
					b.SetBlockers(1, map[string]string{"Blocker": attacker.Card.Name()})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				g.RunStepWithPriority(core.DeclareBlockers)
				g.RunStepWithPriority(core.CombatDamage)
				amount := card.Power()
				if name == "Spirit Link" {
					amount = 2
				}
				wantB := 20 - amount
				if blocked {
					wantB = 20
				}
				if a.Life() != 20+amount || b.Life() != wantB {
					t.Fatalf("combat life %d/%d", a.Life(), b.Life())
				}
				if blocked && g.FindPermanent(blocker.ID()).Damage != amount {
					t.Fatal("wrong marked combat damage")
				}
			})
		}
	}
}

func TestRONDamageAmountAuraTargetLegality(t *testing.T) {
	for _, state := range []string{"land", "shroud", "phase", "return", "legal"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Spirit Link")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 1)
			hostCard := mage.NewCreature("Host", "", 2, 8)
			if state == "land" {
				hostCard = mage.NewLand("Land")
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			if state == "shroud" {
				host.GrantBaseAttr(core.Shroud)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
			if state == "land" || state == "shroud" {
				if err == nil || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("illegal target changed resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if state == "phase" {
				g.PhaseOut(host.ID())
			}
			if state == "return" {
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, b.PlayerID())
			}
			g.ResolveStack()
			if state != "legal" {
				if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
					t.Fatal("aura entered on illegal target")
				}
				return
			}
			if g.FindPermanent(card.ID()).AttachedTo != host.ID() {
				t.Fatal("aura not attached")
			}
		})
	}
}
