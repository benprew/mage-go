package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDeathHistoryCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		counter   core.CounterType
		power     int
	}{
		{"ARN", "Khabál Ghoul", core.P1P1, 1}, {"LEB", "Scavenging Ghoul", core.Corpse, 2},
	} {
		for _, change := range []string{"none", "return", "phase-before", "phase-after", "control", "copy"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				victim := g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", b.PlayerID()), b.PlayerID())
				g.DestroyPermanent(victim)
				c := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				if change == "phase-before" {
					g.PhaseOut(p.ID())
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				wantStack := 1
				if change == "phase-before" {
					wantStack = 0
				}
				if g.StackSize() != wantStack || p.Counters[tc.counter] != 0 {
					t.Fatal("wrong trigger timing")
				}
				switch change {
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
				case "phase-after":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				token := g.PutOnBattlefield(deathHistoryCard(t, "ATQ", "Ornithopter", a.PlayerID()), a.PlayerID())
				token.IsToken = true
				other := g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", b.PlayerID()), b.PlayerID())
				g.DestroyPermanents([]*mage.Permanent{token, other, other})
				if g.CreatureDeaths() != 3 {
					t.Fatalf("deaths = %d", g.CreatureDeaths())
				}
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanentIncludingPhased(c.ID()).Counters[tc.counter] != 0 {
					t.Fatal("clone changed live counters")
				}
				g.ResolveStack()
				want := 3
				if change == "return" || change == "phase-before" || change == "phase-after" {
					want = 0
				}
				if change == "copy" {
					want = 6
				}
				for _, game := range []*mage.Game{g, clone} {
					current := game.FindPermanentIncludingPhased(c.ID())
					if current.Counters[tc.counter] != want {
						t.Fatalf("counters = %d, want %d", current.Counters[tc.counter], want)
					}
					dp := 0
					if tc.counter == core.P1P1 {
						dp = want
					}
					if current.CurrentPower(game) != tc.power+dp {
						t.Fatal("wrong power")
					}
				}
				if change == "none" {
					clone.RunStepWithPriority(core.Cleanup)
					if clone.CreatureDeaths() != 0 || g.CreatureDeaths() != 3 {
						t.Fatal("cleanup history isolation")
					}
					clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					clone.PutTriggersOnStack()
					clone.ResolveStack()
					if clone.FindPermanent(c.ID()).Counters[tc.counter] != 3 {
						t.Fatal("old deaths counted in next turn")
					}
					if tc.counter == core.Corpse {
						if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err == nil {
							t.Fatal("opponent activation accepted")
						}
						if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
							t.Fatal(err)
						}
						if g.FindPermanent(p.ID()).Counters[core.Corpse] != 2 {
							t.Fatal("counter cost missing")
						}
						g.ResolveStack()
						g.DestroyPermanent(g.FindPermanent(p.ID()))
						if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).Tapped || g.CreatureDeaths() != 3 {
							t.Fatal("regeneration counted as death")
						}
					}
				}
			})
		}
	}
}

func TestRONDeathHistoryEvents(t *testing.T) {
	for _, action := range []string{"sacrifice", "exile", "bounce", "regenerate", "artifact", "phase", "animate"} {
		t.Run(action, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Khabál Ghoul"), a.PlayerID())
			c := repositoryCard(t, "LEB", "Llanowar Elves")
			if action == "artifact" || action == "animate" {
				c = repositoryCard(t, "LEB", "Sol Ring")
			}
			c.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(c, b.PlayerID())
			want := 0
			switch action {
			case "sacrifice":
				g.DoSacrifice(p)
				want = 1
			case "exile":
				g.ExilePermanent(p)
			case "bounce":
				g.BouncePermanentToHand(p)
			case "regenerate":
				g.AddRegenerationShield(p.ID())
				g.DestroyPermanent(p)
			case "artifact":
				g.DestroyPermanent(p)
			case "phase":
				g.PhaseOut(p.ID())
				g.DestroyPermanents([]*mage.Permanent{p})
			case "animate":
				factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(IsCardtype(Artifact),[AddCardtype(Creature),SetPT(PT(2,2))])`)
				g.PutOnBattlefield(factory(), a.PlayerID())
				g.ApplyContinuousEffects()
				g.DestroyPermanent(p)
				want = 1
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(source.ID()).Counters[core.P1P1] != want {
				t.Fatalf("counters = %d, want %d", g.FindPermanent(source.ID()).Counters[core.P1P1], want)
			}
		})
	}
}

func TestRONDeathHistoryInvalid(t *testing.T) {
	for _, value := range []string{
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn()`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature),IsCardtype(Creature))`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Planeswalker))`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Artifact))`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature,Artifact))`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature()))`,
		`NumCreaturesOrPlaneswalkersThatDiedThisTurn(ControlledByPlayer(You))`,
	} {
		t.Run(value, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([PutCounters([NumberCountersOfTypeOnPermanent(%s,ChargeCounter,ThisPermanent)])]))])`, value)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("invalid death filter accepted")
			}
		})
	}
}

func TestRONDeathHistoryInventory(t *testing.T) {
	src := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([PutCounters([NumberCountersOfTypeOnPermanent(NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature)),ChargeCounter,ThisPermanent)])]))])`
	report, err := InspectFS(fstest.MapFS{"history.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range report.Signatures {
		if s.Name == "IsCardtype" && s.Role == "permanent_filter" {
			return
		}
	}
	t.Fatal("missing historical permanent filter role")
}

func deathHistoryCard(t *testing.T, set, name string, owner uuid.UUID) mage.Card {
	t.Helper()
	c := repositoryCard(t, set, name)
	c.SetOwner(owner)
	return c
}

func TestRONDeathHistoryCastingAndInstances(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		counter   core.CounterType
	}{
		{"ARN", "Khabál Ghoul", core.P1P1},
		{"LEA", "Scavenging Ghoul", core.Corpse}, {"LEB", "Scavenging Ghoul", core.Corpse},
		{"2ED", "Scavenging Ghoul", core.Corpse}, {"3ED", "Scavenging Ghoul", core.Corpse},
		{"CED", "Scavenging Ghoul", core.Corpse}, {"CEI", "Scavenging Ghoul", core.Corpse},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 || g.FindPermanent(c.ID()) != nil {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.Black, 4)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			second := g.PutOnBattlefield(deathHistoryCard(t, tc.set, tc.name, b.PlayerID()), b.PlayerID())
			if tc.counter == core.Corpse {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 1, nil); err == nil {
					t.Fatal("unpaid counter cost accepted")
				}
				if g.StackSize() != 0 {
					t.Fatal("failed activation created stack object")
				}
			}
			g.DoSacrifice(g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", a.PlayerID()), a.PlayerID()))
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("independent source triggers missing")
			}
			clone := g.Clone()
			clone.DoSacrifice(clone.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", b.PlayerID()), b.PlayerID()))
			clone.ResolveStack()
			g.ResolveStack()
			for _, id := range []uuid.UUID{c.ID(), second.ID()} {
				if g.FindPermanent(id).Counters[tc.counter] != 1 || clone.FindPermanent(id).Counters[tc.counter] != 2 {
					t.Fatal("resolution value did not use its own game")
				}
			}
		})
	}
}

func TestRONDeathHistoryActionContexts(t *testing.T) {
	for _, context := range []string{"spell", "activation"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			amount := `NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature))`
			action := fmt.Sprintf(`ActionList([PutCounters([NumberCountersOfTypeOnPermanent(%s,ChargeCounter,ThisPermanent)])])`, amount)
			types, rules := "Artifact", `Activated(TapPermanent(ThisPermanent),`+action+`)`
			if context == "spell" {
				types = "Instant"
				action = fmt.Sprintf(`ActionList([PutCounters([NumberCountersOfTypeOnPermanent(%s,ChargeCounter,Ref_TargetPermanent)])])`, amount)
				rules = `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],` + action + `))`
			}
			c := permanentScopeFactory(t, types, rules)()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			var subject uuid.UUID
			if context == "spell" {
				target := g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", b.PlayerID()), b.PlayerID())
				subject = target.ID()
				castRONSpell(t, g, a, c, subject)
			} else {
				subject = g.PutOnBattlefield(c, a.PlayerID()).ID()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), subject, 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.DoSacrifice(g.PutOnBattlefield(deathHistoryCard(t, "LEB", "Llanowar Elves", a.PlayerID()), a.PlayerID()))
			g.ResolveStack()
			if g.FindPermanent(subject).Counters[core.Charge] != 1 {
				t.Fatal("history amount failed in action context")
			}
		})
	}
}
