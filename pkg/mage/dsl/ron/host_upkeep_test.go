package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var hostUpkeepCards = map[string]core.CardType{"Cursed Land": core.TypeLand, "Feedback": core.TypeEnchantment, "Wanderlust": core.TypeCreature, "Warp Artifact": core.TypeArtifact, "Unstable Mutation": core.TypeCreature}

func upkeepHost(name string, kind core.CardType) mage.Card {
	switch kind {
	case core.TypeLand:
		return mage.NewLand(name)
	case core.TypeEnchantment:
		return mage.NewEnchantment(name, "")
	case core.TypeArtifact:
		return mage.NewArtifact(name, "")
	default:
		return mage.NewCreature(name, "", 4, 8)
	}
}

func TestRONHostUpkeepReprints(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	collection := fstest.MapFS{}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if _, ok := hostUpkeepCards[name]; !ok {
				continue
			}
			count++
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, factory()); err != nil {
				t.Fatal(err)
			}
		}
		var selected []string
		for line := range strings.SplitSeq(string(data), "\n") {
			for name := range hostUpkeepCards {
				if strings.Contains(line, `Name: "`+name+`"`) {
					selected = append(selected, line)
				}
			}
		}
		if len(selected) > 0 {
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(selected, "\n"))}
		}
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if count != 26 || len(registry.RegisteredCardNames()) != 5 {
		t.Fatalf("records=%d names=%d", count, len(registry.RegisteredCardNames()))
	}
}

func TestRONHostUpkeepGameplay(t *testing.T) {
	for name, kind := range hostUpkeepCards {
		for _, scenario := range []string{"normal", "move", "control-before", "control-after", "source-control", "source-left", "source-return", "host-return", "phase-before", "phase-after", "phase-cycle", "removed", "removed-after", "unattached", "copy"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				hc := upkeepHost("Host", kind)
				hc.SetOwner(b.PlayerID())
				host := g.PutOnBattlefield(hc, b.PlayerID())
				otherCard := upkeepHost("Other", kind)
				otherCard.SetOwner(a.PlayerID())
				other := g.PutOnBattlefield(otherCard, a.PlayerID())
				set := "LEB"
				if name == "Unstable Mutation" {
					set = "ARN"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if scenario != "unattached" {
					g.Attach(source.ID(), host.ID())
				}
				if scenario == "control-before" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				if scenario == "phase-before" {
					g.PhaseOut(host.ID())
				}
				if scenario == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				player := b.PlayerID()
				wrong := a.PlayerID()
				if scenario == "control-before" {
					player, wrong = wrong, player
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: wrong})
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: player})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("wrong step or player triggered")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player})
				switch scenario {
				case "removed-after":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				case "move":
					g.Attach(source.ID(), other.ID())
				case "control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-left", "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if scenario == "source-return" {
						g.PutOnBattlefield(card, a.PlayerID())
						g.Attach(card.ID(), other.ID())
					}
				case "host-return":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					g.PutOnBattlefield(hc, b.PlayerID())
				case "phase-after":
					g.PhaseOut(host.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				}
				want := 1
				if scenario == "phase-before" || scenario == "removed" || scenario == "unattached" {
					want = 0
				}
				pending := g.Clone()
				for _, game := range []*mage.Game{pending, g} {
					game.PutTriggersOnStack()
					if game.StackSize() != want {
						t.Fatalf("stack=%d want=%d", game.StackSize(), want)
					}
					amount := want
					if scenario == "copy" {
						if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
						amount++
					}
					game.ResolveStack()
					if game == pending && (a.Life() != 20 || b.Life() != 20 || g.FindPermanentIncludingPhased(host.ID()).Counters[core.M1M1] != 0) {
						t.Fatal("clone changed original")
					}
					if name == "Unstable Mutation" {
						target := host
						if scenario == "move" {
							target = other
						}
						if scenario == "host-return" || scenario == "phase-after" {
							amount = 0
						}
						p := game.FindPermanentIncludingPhased(target.ID())
						if p == nil || int(p.Counters[core.M1M1]) != amount {
							t.Fatalf("counter target=%v want=%d", p, amount)
						}
						if scenario == "normal" && p.CurrentPower(game) != 7-amount {
							t.Fatal("static bonus or counter missing")
						}
					} else {
						if game.GetPlayer(player).Life() != 20-amount || game.GetPlayer(wrong).Life() != 20 {
							t.Fatal("event player changed")
						}
					}
				}
			})
		}
	}
}

func TestRONHostUpkeepInvalid(t *testing.T) {
	valid := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(HostController)), ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))]))`
	for _, rule := range []string{
		strings.Replace(valid, "HostController", "HostController()", 1),
		strings.Replace(valid, "HostController", "HostController, You", 1),
		strings.Replace(valid, "SinglePlayer(HostController)", "HostController", 1),
		strings.Replace(valid, "AtTheBeginningOfAPlayersUpkeep", "AtTheBeginningOfAPlayersEndStep", 1),
		strings.Replace(valid, "Trigger_ThatPlayer", "Trigger_ThatPermanent", 1),
		valid + `, Activated(PayMana([]), ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+valid+`])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("non-Aura accepted")
	}
}

func TestRONHostUpkeepNativeCast(t *testing.T) {
	g, a, b := ronGame()
	host := mage.NewCreature("Host", "", 4, 8)
	host.SetOwner(b.PlayerID())
	g.PutOnBattlefield(host, b.PlayerID())
	card := repositoryCard(t, "LEB", "Wanderlust")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Green, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
		t.Fatal("invalid Aura target accepted")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != 3 {
		t.Fatal("invalid target changed state")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()).AttachedTo != host.ID() {
		t.Fatal("Aura did not attach")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || b.Life() != 20 {
		t.Fatal("trigger timing or targets")
	}
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("damage missing")
	}
}

func TestRONHostUpkeepIndependentBindings(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.ron")
	if err != nil {
		t.Fatal(err)
	}
	var factory mage.CardFactory
	for _, node := range nodes {
		if node.Fields["Name"].String == "Wanderlust" {
			factory, err = Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if factory == nil {
		t.Fatal("missing record")
	}
	g, a, b := ronGame()
	for _, player := range []*mage.BasePlayer{a, b} {
		host := mage.NewCreature("Host", "", 4, 8)
		host.SetOwner(player.PlayerID())
		g.PutOnBattlefield(host, player.PlayerID())
		card := factory()
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
		g.Attach(card.ID(), host.ID())
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	pending := g.Clone()
	for _, game := range []*mage.Game{pending, g} {
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatal("pending events lost")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatal("bindings collided")
		}
		if game == pending && (a.Life() != 20 || b.Life() != 20) {
			t.Fatal("clone changed original")
		}
	}
}

func TestRONHostUpkeepDamagePrevention(t *testing.T) {
	g, a, b := ronGame()
	host := mage.NewCreature("Host", "", 4, 8)
	host.SetOwner(b.PlayerID())
	g.PutOnBattlefield(host, b.PlayerID())
	card := repositoryCard(t, "LEB", "Wanderlust")
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(card.ID(), host.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.AddPreventionShield(b.PlayerID(), 1)
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("damage bypassed prevention")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("shield did not expire after use")
	}
}

func TestRONHostUpkeepNativeStep(t *testing.T) {
	for name, kind := range hostUpkeepCards {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			host := upkeepHost("Host", kind)
			host.SetOwner(b.PlayerID())
			g.PutOnBattlefield(host, b.PlayerID())
			set := "LEB"
			if name == "Unstable Mutation" {
				set = "ARN"
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(card.ID(), host.ID())
			g.RunStepWithPriority(core.Upkeep)
			if b.Life() != 20 || g.FindPermanent(host.ID()).Counters[core.M1M1] != 0 {
				t.Fatal("Aura controller upkeep triggered")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Upkeep)
			if name == "Unstable Mutation" {
				p := g.FindPermanent(host.ID())
				if p.Counters[core.M1M1] != 1 || p.CurrentPower(g) != 6 || p.CurrentToughness(g) != 10 {
					t.Fatal("native upkeep bonus or counter missing")
				}
			} else if b.Life() != 19 || a.Life() != 20 {
				t.Fatal("native upkeep damage missing")
			}
			if g.StackSize() != 0 {
				t.Fatal("upkeep trigger did not resolve")
			}
		})
	}
}
