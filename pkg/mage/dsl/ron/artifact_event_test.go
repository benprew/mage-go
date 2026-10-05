package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONArtifactEvents(t *testing.T) {
	for _, name := range []string{"Haunting Wind", "Powerleech", "Artifact Possession"} {
		for _, scenario := range []string{"tap", "retap", "activate", "tap-cost", "mana", "failed", "sacrifice", "departed", "returned", "control", "phased", "copy", "source-left", "source-phased", "removed", "other", "own", "nonartifact", "twice", "entry-tapped", "mana-no-tap", "failed-tap", "prevention", "reattach", "phase-cycle", "source-control", "host-phased-before"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				if scenario == "nonartifact" && name == "Artifact Possession" {
					return
				}
				g, a, b := ronGame()
				rules := []mage.CardOption{mage.WithAbility(mage.Activate(mage.Costs(mage.ManaPayment("{1}")), mage.GainLifeFor(mage.You, mage.Fixed(0))))}
				if scenario == "sacrifice" {
					rules = []mage.CardOption{mage.WithAbility(mage.Activate(mage.Costs(mage.SacrificeSourceCost()), mage.GainLifeFor(mage.You, mage.Fixed(0))))}
				}
				if scenario == "tap-cost" {
					rules = []mage.CardOption{mage.WithAbility(mage.Activate(mage.Costs(mage.Tap()), mage.GainLifeFor(mage.You, mage.Fixed(0))))}
				}
				if scenario == "mana" {
					rules = []mage.CardOption{mage.WithAbility(mage.NewManaAbility(core.Blue))}
				}
				if scenario == "mana-no-tap" {
					rules = []mage.CardOption{mage.WithAbility(mage.Activate(mage.Costs(), mage.AddManaAmount(core.Blue, mage.Fixed(1))))}
				}
				if scenario == "failed-tap" {
					rules = []mage.CardOption{mage.WithAbility(mage.Activate(mage.Costs(mage.Tap(), mage.ManaPayment("{2}")), mage.GainLifeFor(mage.You, mage.Fixed(0))))}
				}
				hostCard := mage.NewArtifact("Host", "", rules...)
				if scenario == "nonartifact" {
					hostCard = mage.NewCreature("Host", "", 1, 3, rules...)
				}
				if scenario == "entry-tapped" {
					hostCard = mage.NewArtifact("Host", "", mage.WithKeyword(core.EntersTapped))
				}
				hostCard.SetOwner(b.PlayerID())
				controller := b.PlayerID()
				if scenario == "own" {
					controller = a.PlayerID()
				}
				host := g.PutOnBattlefield(hostCard, controller)
				card := repositoryCard(t, "ATQ", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if name == "Artifact Possession" {
					g.Attach(source.ID(), host.ID())
				}
				if scenario == "source-phased" {
					g.PhaseOut(source.ID())
				}
				if scenario == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				if scenario == "host-phased-before" {
					g.PhaseOut(host.ID())
				}
				target := host
				if scenario == "other" {
					target = g.PutOnBattlefield(mage.NewArtifact("Other", ""), b.PlayerID())
				}
				activation := scenario == "activate" || scenario == "tap-cost" || scenario == "mana" || scenario == "failed" || scenario == "sacrifice" || scenario == "mana-no-tap" || scenario == "failed-tap"
				if activation {
					if scenario != "failed" && scenario != "failed-tap" {
						g.GetPlayer(controller).ManaPool().Add(core.Colorless, 1)
					}
					err := g.ActivateAbilityByIndex(controller, host.ID(), 0, nil)
					if (err != nil) != (scenario == "failed" || scenario == "failed-tap") {
						t.Fatalf("activation: %v", err)
					}
				} else {
					g.TapPermanent(g.FindPermanent(target.ID()))
					if scenario == "twice" {
						g.UntapPermanent(g.FindPermanent(target.ID()))
						g.TapPermanent(g.FindPermanent(target.ID()))
					}
					if scenario == "retap" {
						g.TapPermanent(g.FindPermanent(target.ID()))
					}
				}
				want := 1
				if scenario == "twice" {
					want = 2
				}
				if scenario == "prevention" {
					g.AddDamagePreventionRule(mage.WithFrom(mage.IsID(source.ID())))
				}
				if scenario == "phase-cycle" {
					g.PhaseIn(g.PhaseOut(host.ID()))
				}
				if scenario == "reattach" && name == "Artifact Possession" {
					other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
					g.Attach(source.ID(), other.ID())
				}
				if scenario == "host-phased-before" || scenario == "failed" || scenario == "failed-tap" || scenario == "entry-tapped" || (scenario == "nonartifact" && name != "Artifact Possession") || scenario == "source-phased" || scenario == "removed" || (scenario == "own" && name == "Powerleech") || (scenario == "other" && name == "Artifact Possession") {
					want = 0
				}
				if scenario == "departed" || scenario == "returned" {
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					if scenario == "returned" {
						g.PutOnBattlefield(hostCard, a.PlayerID())
					}
				}
				if scenario == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "phased" {
					g.PhaseOut(host.ID())
				}
				if scenario == "source-control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "source-left" {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				}
				g.PutTriggersOnStack()
				base := 0
				if activation && scenario != "failed" && scenario != "failed-tap" && scenario != "mana" && scenario != "mana-no-tap" {
					base = 1
				}
				if g.StackSize() != want+base {
					t.Fatalf("stack=%d want=%d", g.StackSize(), want+base)
				}
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
					want++
				}
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone changed live life")
				}
				g.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					pa, pb := game.GetPlayer(a.PlayerID()), game.GetPlayer(b.PlayerID())
					wa, wb := 20, 20
					if name == "Powerleech" {
						wa += want
						if scenario == "copy" {
							wa--
							wb++
						}
					} else {
						damage := want
						if scenario == "prevention" {
							damage = 0
						}
						if name == "Artifact Possession" {
							damage *= 2
						}
						if scenario == "control" || scenario == "own" {
							wa -= damage
						} else {
							wb -= damage
						}
					}
					if pa.Life() != wa || pb.Life() != wb {
						t.Fatalf("life=(%d,%d) want=(%d,%d)", pa.Life(), pb.Life(), wa, wb)
					}
				}
			})
		}
	}
}

func TestRONArtifactEventInvalid(t *testing.T) {
	event := `WhenAPlayerActivatesAnAbility(AnyPlayer,And([AbilityOfAPermanent(IsCardtype(Artifact)),DoesntHaveTapSelfInCost]))`
	for _, bad := range []string{
		strings.Replace(event, "AnyPlayer", "You", 1), strings.Replace(event, "DoesntHaveTapSelfInCost", "DoesntHaveTapSelfInCost()", 1),
		strings.Replace(event, "Artifact", "Creature", 1), strings.Replace(event, "DoesntHaveTapSelfInCost", "HasTapSelfInCost", 1),
		strings.Replace(event, "IsCardtype(Artifact)", "IsCardtype(Artifact,Creature)", 1),
		strings.Replace(event, "DoesntHaveTapSelfInCost", "DoesntHaveTapSelfInCost,DoesntHaveTapSelfInCost", 1),
		strings.Replace(event, "AbilityOfAPermanent(IsCardtype(Artifact))", "AbilityOfPermanent(HostPermanent)", 1),
	} {
		source := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(%s,ActionList([GainLife(Integer(1))]))])`, bad)
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRONArtifactEventNativeLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	names := []string{"Haunting Wind", "Powerleech", "Artifact Possession"}
	for _, name := range names {
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+name+`"`) {
				files[name+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 3 {
		t.Fatal("missing card")
	}
	for _, name := range names {
		g, a, b := ronGame()
		host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		var targets []uuid.UUID
		if name == "Artifact Possession" {
			targets = []uuid.UUID{host.ID()}
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil {
			t.Fatal("unpaid casting accepted")
		}
		a.ManaPool().Add(core.Colorless, 4)
		a.ManaPool().Add(core.Black, 1)
		a.ManaPool().Add(core.Green, 2)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(card.ID()) == nil {
			t.Fatal("casting failed")
		}
		g.TapPermanent(g.FindPermanent(host.ID()))
		g.PutTriggersOnStack()
		g.ResolveStack()
		wa, wb := 20, 19
		if name == "Powerleech" {
			wa, wb = 21, 20
		}
		if name == "Artifact Possession" {
			wb = 18
		}
		if a.Life() != wa || b.Life() != wb {
			t.Fatal("loaded card failed")
		}
	}
}

func TestRONArtifactEventInstancesAndBatch(t *testing.T) {
	g, a, b := ronGame()
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "ATQ", "Haunting Wind"), a.PlayerID())
	}
	first := g.PutOnBattlefield(mage.NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewArtifact("Second", ""), b.PlayerID())
	action := mage.TapEach(mage.MatchingPermanents(mage.IsArtifact))
	if err := action.Apply(&mage.EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("batch incomplete")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d", g.StackSize())
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatal("event or factory binding shared")
		}
	}
}

func TestRONArtifactEventScopeAndTapDeclarations(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPermanentBecomesTapped(IsCardtype(Artifact)),ActionList([TapPermanent(Trigger_ThatPlayer)]))`,
		`TriggerA(WhenAPermanentBecomesTapped(IsCardtype(Artifact)),ActionList([TapPermanent(Trigger_ThatPermanent())]))`,
		`TriggerA(WhenAPermanentBecomesTapped(IsCardtype(Artifact)),ActionList([GainLife(Integer(1))])),Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatPermanent)]))`,
		`TriggerA(WhenAPermanentBecomesTapped(And([IsCardtype(Artifact),ControlledByAPlayer(You)])),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentBecomesTapped(And([IsCardtype(Artifact),IsCardtype(Artifact)])),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPermanentBecomesTapped(IsCardtype(Artifact())),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerActivatesAnAbility(AnyPlayer,And([AbilityOfAPermanent(IsCardtype(Artifact)),DoesntHaveTapSelfInCost])),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	rule := `TriggerA(WhenAPermanentBecomesTapped(And([ControlledByAPlayer(Opponent),IsCardtype(Artifact)])),ActionList([GainLife(Integer(1))]))`
	factory := permanentScopeFactory(t, "Enchantment", rule)
	g, a, b := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
	g.TapPermanent(host)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("reversed conjunction failed")
	}
}
