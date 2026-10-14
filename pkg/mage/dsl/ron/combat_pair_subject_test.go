package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatPairSubjectInfiniteAuthority(t *testing.T) {
	for _, attacking := range []bool{false, true} {
		for _, change := range []string{"none", "reattach", "aura-leaves", "host-return", "host-phase", "host-phase-cycle", "opponent-return", "opponent-phase", "regenerate", "indestructible", "toughness", "copy", "control", "doubled"} {
			t.Run(boolName(attacking)+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				aura := g.PutOnBattlefield(repositoryCard(t, "LEG", "Infinite Authority"), a.PlayerID())
				hostCard := mage.NewCreature("Host", "", 0, 8)
				host := g.PutOnBattlefield(hostCard, a.PlayerID())
				g.Attach(aura.ID(), host.ID())
				otherCard := mage.NewCreature("Opponent", "", 0, 3, mage.WithKeyword(core.Shroud))
				other := g.PutOnBattlefield(otherCard, b.PlayerID())
				host.RevokeBaseAttr(core.AttrSummonSick)
				other.RevokeBaseAttr(core.AttrSummonSick)
				if change == "copy" {
					g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: host.ID(), TargetID: other.ID()})
				} else if attacking {
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{host.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: host.ID()}})
				} else {
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{other.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: host.ID(), AttackerID: other.ID()}})
				}
				g.PutTriggersOnStack()
				if change == "copy" && g.StackSize() != 1 {
					t.Fatalf("stack=%d", g.StackSize())
				}
				switch change {
				case "reattach":
					g.Attach(aura.ID(), other.ID())
				case "aura-leaves":
					g.RemoveFromBattlefield(aura)
				case "host-return":
					g.RemoveFromBattlefield(host)
					g.PutOnBattlefield(hostCard, a.PlayerID())
				case "host-phase":
					g.PhaseOut(host.ID())
				case "host-phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "opponent-return":
					g.RemoveFromBattlefield(other)
					g.PutOnBattlefield(otherCard, b.PlayerID())
				case "opponent-phase":
					g.PhaseOut(other.ID())
				case "regenerate":
					g.AddRegenerationShield(other.ID())
				case "indestructible":
					g.MutablePermanent(other.ID()).GrantBaseAttr(core.Indestructible)
				case "toughness":
					if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.SetBasePowerToughness(mage.This, mage.Fixed(0), mage.Fixed(8))), other.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					g.ApplyContinuousEffects()
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "doubled":
					doubler := g.PutOnBattlefield(mage.NewArtifact("Doubler", ""), a.PlayerID())
					g.AddCounterDoubler(doubler.ID(), core.P1P1, mage.PermanentFilter{})
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: aura.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{clone, g} {
					game.ResolveStack()
					if change != "opponent-phase" && game.FindPermanent(other.ID()) == nil {
						t.Fatal("early destruction")
					}
					game.RunStepWithPriority(core.EndCombat)
					paid := change != "opponent-return" && change != "opponent-phase" && change != "regenerate" && change != "indestructible"
					if (game.FindPermanent(other.ID()) == nil) != (paid || change == "opponent-phase") {
						t.Fatal("wrong destruction result")
					}
					if change != "host-phase" && game.FindPermanent(host.ID()).Counters[core.P1P1] != 0 {
						t.Fatal("early counter")
					}
					game.RunStepWithPriority(core.EndStep)
					if change == "host-phase" {
						game.PhaseIn([]uuid.UUID{host.ID()})
					}
					want := 0
					if paid && change != "host-return" && change != "host-phase" {
						want = 1
					}
					if change == "doubled" {
						want *= 2
					}
					if game.FindPermanent(host.ID()).Counters[core.P1P1] != want {
						t.Fatalf("host counters=%d want=%d", game.FindPermanent(host.ID()).Counters[core.P1P1], want)
					}
				}
			})
		}
	}
}

func TestRONCombatPairSubjectInvalid(t *testing.T) {
	for _, ref := range []string{"Trigger_ThatCreature()", "Trigger_ThatPermanent()", "Trigger_ThatPlayer", "Trigger_ThatCreature"} {
		rule := `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),ActionList([TapPermanent(` + ref + `)]))`
		if ref == "Trigger_ThatCreature" {
			rule = `Activated(PayMana([]),ActionList([TapPermanent(` + ref + `)]))`
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONCombatPairSubjectNestedSubscription(t *testing.T) {
	for _, ref := range []string{"Trigger_ThatCreature", "Trigger_ThatPermanent"} {
		pair := `Or([WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),WhenACreatureBecomesBlockedByACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature))])`
		rule := `TriggerA(` + pair + `,ActionList([TapPermanent(` + ref + `),CreateTriggerUntil(` + pair + `,ActionList([TapPermanent(` + ref + `)]),UntilEndOfCombat)]))`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "unbound or incompatible permanent reference") {
			t.Fatalf("nested subscription boundary %s: %v", ref, err)
		}
	}
}

func TestRONCombatPairSubjectNestedLocalEvent(t *testing.T) {
	for _, death := range []bool{false, true} {
		t.Run(boolName(death), func(t *testing.T) {
			event, ref := "WhenACreatureAttacksAndIsntBlocked", "Trigger_ThatCreature"
			if death {
				event, ref = "WhenAPermanentDies", "Trigger_ThatPermanent"
			}
			rule := `TriggerA(WhenACreatureBlocksACreature(SinglePermanent(ThisPermanent),IsCardtype(Creature)),ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,1),Trigger_ThatCreature)]),CreateTriggerUntil(` + event + `(SinglePermanent(Trigger_ThatOtherCreature)),ActionList([TapPermanent(` + ref + `),GainLife(Integer(1))]),UntilEndOfTurn)]))`
			g, a, b := ronGame()
			p := g.PutOnBattlefield(characteristicFactory(t, false, rule)(), a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: p.ID(), TargetID: q.ID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if death {
				if err := mage.ApplyEffect(g, mage.Destroy(mage.This), q.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			} else {
				g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: q.ID()})
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Tapped || a.Life() != 21 || !death && !g.FindPermanent(q.ID()).Tapped {
				t.Fatal("local event reused enclosing subject")
			}
		})
	}
}

func TestRONCombatPairSubjectShapes(t *testing.T) {
	for _, host := range []bool{false, true} {
		for _, direction := range []string{"blocks", "blocked", "union"} {
			for _, ref := range []string{"Trigger_ThatCreature", "Trigger_ThatPermanent"} {
				t.Run(boolName(host)+"/"+direction+"/"+ref, func(t *testing.T) {
					g, a, b := ronGame()
					subject := "ThisPermanent"
					if host {
						subject = "HostPermanent"
					}
					blocks := `WhenACreatureBlocksACreature(SinglePermanent(` + subject + `),IsCardtype(Creature))`
					blocked := strings.Replace(blocks, "WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature", 1)
					event, kind := blocks, core.EvtDeclaredBlocker
					if direction == "blocked" {
						event, kind = blocked, core.EvtCreatureBlockedBy
					}
					if direction == "union" {
						event, kind = `Or([`+blocked+`,`+blocks+`])`, core.EvtCombatPair
					}
					card := characteristicFactory(t, host, `TriggerA(`+event+`,ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1,1),`+ref+`)])]))]))`)()
					if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(card, a.PlayerID())
					watched := p
					if host {
						watched = g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), b.PlayerID())
						g.Attach(p.ID(), watched.ID())
					}
					q := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 8), b.PlayerID())
					g.FireEvent(core.GameEvent{Type: kind, SourceID: watched.ID(), TargetID: q.ID()})
					if host {
						g.Attach(p.ID(), q.ID())
					}
					g.PutTriggersOnStack()
					if got := g.StackBoundPermanentSources(); len(got) != 2 {
						t.Fatalf("saved subject and opponent sources=%d", len(got))
					}
					g.ResolveStack()
					g.RunStepWithPriority(core.EndStep)
					if g.FindPermanent(watched.ID()).Counters[core.P1P1] != 1 || g.FindPermanent(q.ID()).Counters[core.P1P1] != 0 {
						t.Fatal("alias did not save watched subject")
					}
				})
			}
		}
	}
}

func TestRONCombatPairSubjectCasting(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Infinite Authority")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid casting changed resources")
	}
	a.ManaPool().Add(core.White, 3)
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 3 {
		t.Fatal("illegal enchant target spent mana")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()).AttachedTo != host.ID() || a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting or attachment failed")
	}
}

func TestRONCombatPairSubjectIndependentEvents(t *testing.T) {
	g, a, b := ronGame()
	factory, err := Compile(hostUntapRecord(t, "LEG", "Infinite Authority"))
	if err != nil {
		t.Fatal(err)
	}
	hosts := make([]*mage.Permanent, 0, 4)
	for i := range 2 {
		aura := g.PutOnBattlefield(factory(), a.PlayerID())
		for j := range 2 {
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 0, 8), a.PlayerID())
			hosts = append(hosts, host)
			g.Attach(aura.ID(), host.ID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 0, 3), b.PlayerID())
			if i == 0 && j == 0 {
				g.AddRegenerationShield(other.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: host.ID(), TargetID: other.ID()})
		}
		g.RemoveFromBattlefield(aura)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.ResolveStack()
		game.RunStepWithPriority(core.EndCombat)
		game.RunStepWithPriority(core.EndStep)
		for i, host := range hosts {
			want := 1
			if i == 0 {
				want = 0
			}
			if game.FindPermanent(host.ID()).Counters[core.P1P1] != want {
				t.Fatal("event subject or destruction result was overwritten")
			}
		}
		if game == clone && g.StackSize() != 4 {
			t.Fatal("clone changed original pending events")
		}
	}
}

func TestRONCombatPairSubjectAtomicLoad(t *testing.T) {
	record := hostUntapRecord(t, "LEG", "Infinite Authority")
	raw, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	data := strings.Split(string(raw), "\n")[record.Position.Line-1]
	registry := mage.NewRegistry()
	files := fstest.MapFS{"valid.ron": &fstest.MapFile{Data: []byte(data)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4),Rules:[Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatCreature)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid event binding changed registry")
	}
}
