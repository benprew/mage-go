package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHostTapRealCards(t *testing.T) {
	for _, name := range []string{"Blight", "Spirit Shackle"} {
		for _, scenario := range []string{"tap", "retap", "twice", "other", "reattach", "source-left", "source-returned", "host-returned", "host-phased", "phase-cycle", "source-phased", "control", "removed", "shroud", "copy", "regenerate", "entry-tapped", "unattached", "source-control-after"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				hostCard := mage.NewCreature("Host", "", 4, 8)
				otherCard := mage.NewCreature("Other", "", 4, 8)
				if name == "Blight" {
					hostCard = mage.NewLand("Host")
					otherCard = mage.NewLand("Other")
				}
				if scenario == "shroud" {
					hostCard = mage.NewCreature("Host", "", 4, 8, mage.WithKeyword(core.Shroud))
					if name == "Blight" {
						hostCard = mage.NewLand("Host", mage.WithKeyword(core.Shroud))
					}
				}
				if scenario == "entry-tapped" {
					hostCard = mage.NewCreature("Host", "", 4, 8, mage.WithKeyword(core.EntersTapped))
					if name == "Blight" {
						hostCard = mage.NewLand("Host", mage.WithKeyword(core.EntersTapped))
					}
				}
				hostCard.SetOwner(b.PlayerID())
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				otherCard.SetOwner(b.PlayerID())
				other := g.PutOnBattlefield(otherCard, b.PlayerID())
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if scenario != "unattached" {
					g.Attach(source.ID(), host.ID())
				}
				if scenario == "source-phased" {
					g.PhaseOut(source.ID())
				}
				if scenario == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				target := host
				if scenario == "other" {
					target = other
				}
				g.TapPermanent(g.FindPermanent(target.ID()))
				want := 1
				if scenario == "retap" {
					g.TapPermanent(g.FindPermanent(host.ID()))
				}
				if scenario == "twice" {
					g.UntapPermanent(g.FindPermanent(host.ID()))
					g.TapPermanent(g.FindPermanent(host.ID()))
					want = 2
				}
				if scenario == "other" || scenario == "source-phased" || scenario == "removed" || scenario == "entry-tapped" || scenario == "unattached" {
					want = 0
				}
				switch scenario {
				case "source-control-after":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "reattach":
					g.Attach(source.ID(), other.ID())
				case "source-left", "source-returned":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if scenario == "source-returned" {
						returned := g.PutOnBattlefield(card, a.PlayerID())
						g.Attach(returned.ID(), other.ID())
					}
				case "host-returned":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					host = g.PutOnBattlefield(hostCard, b.PlayerID())
				case "host-phased":
					g.PhaseOut(host.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "regenerate":
					g.AddRegenerationShield(host.ID())
				}
				g.PutTriggersOnStack()
				if g.StackSize() != want {
					t.Fatalf("stack=%d want=%d", g.StackSize(), want)
				}
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					want++
				}
				clone := g.Clone()
				clone.ResolveStack()
				if p := g.FindPermanent(host.ID()); p != nil && p.Counters[core.M0M2] != 0 {
					t.Fatal("clone shared counters")
				}
				g.ResolveStack()
				affected := want > 0 && scenario != "host-returned" && scenario != "host-phased"
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanentIncludingPhased(host.ID())
					if name == "Blight" {
						destroyed := affected && scenario != "regenerate"
						if (p == nil) != destroyed {
							t.Fatalf("destroyed=%t want=%t", p == nil, destroyed)
						}
						if scenario == "regenerate" && game.HasRegenerationShield(host.ID()) {
							t.Fatal("regeneration was bypassed")
						}
					} else {
						count := 0
						if affected {
							count = want
						}
						if p == nil || p.Counters[core.M0M2] != count {
							t.Fatalf("counters=%v want=%d", p, count)
						}
						if p.CurrentToughness(game) != 8-2*count {
							t.Fatal("counter did not change toughness")
						}
					}
					if game.FindPermanent(other.ID()) == nil || game.FindPermanent(other.ID()).Counters[core.M0M2] != 0 {
						t.Fatal("event followed another attachment")
					}
				}
			})
		}
	}
}

func TestRONHostTapInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)), ActionList([DestroyPermanent(Trigger_ThatPermanent)]))`
	for _, rule := range []string{
		strings.Replace(valid, "SinglePermanent(HostPermanent)", "SinglePermanent(HostPermanent())", 1),
		strings.Replace(valid, "SinglePermanent(HostPermanent)", "SinglePermanent(HostPermanent, ThisPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(HostPermanent)", "SinglePermanent(Ref_TargetPermanent)", 1),
		strings.Replace(valid, "DestroyPermanent(Trigger_ThatPermanent)", "DestroyPermanent(Trigger_ThatPermanent())", 1),
		strings.Replace(valid, "DestroyPermanent(Trigger_ThatPermanent)", "PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))", 1),
		valid + `, Activated(PayMana([]), ActionList([DestroyPermanent(Trigger_ThatPermanent)]))`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: [Aura]), Rules: [EnchantPermanent(AnyPermanent), %s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: []), Rules: [`+valid+`])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("non-Aura accepted")
	}
}

func TestRONHostTapNativeCastingAndPayment(t *testing.T) {
	for _, name := range []string{"Blight", "Spirit Shackle"} {
		g, a, b := ronGame()
		hostCard := repositoryCard(t, "LEB", "Swamp")
		if name == "Spirit Shackle" {
			hostCard = mage.NewCreature("Host", "", 4, 8)
		}
		hostCard.SetOwner(b.PlayerID())
		host := g.PutOnBattlefield(hostCard, b.PlayerID())
		aura := repositoryCard(t, "LEG", name)
		aura.SetOwner(a.PlayerID())
		a.AddToHand(aura)
		a.ManaPool().Add(core.Black, 2)
		if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(aura.ID()).AttachedTo != host.ID() {
			t.Fatal("Aura did not attach")
		}
		if name == "Blight" {
			g.SetActivePlayerIndex(1)
			spell := mage.NewSorcery("Payment", "{2}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0); err == nil {
				t.Fatal("unpaid spell cast")
			}
			g.PutTriggersOnStack()
			if g.FindPermanent(host.ID()).Tapped || g.StackSize() != 0 || b.ManaPool().TotalMana() != 0 || len(b.Hand()) != 1 {
				t.Fatal("failed payment leaked tap or trigger")
			}
			if err := g.TapForMana(b.PlayerID(), host.ID()); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || g.FindPermanent(host.ID()) == nil {
				t.Fatal("tap trigger resolved as mana ability")
			}
		} else {
			g.TapPermanent(host)
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
			t.Fatal("tap trigger is missing or targeted")
		}
		g.ResolveStack()
		if name == "Blight" {
			if g.FindPermanent(host.ID()) != nil || g.FindPermanent(aura.ID()) != nil {
				t.Fatal("land or unattached Aura survived")
			}
		} else if g.FindPermanent(host.ID()).Counters[core.M0M2] != 1 {
			t.Fatal("counter missing")
		}
	}
}

func TestRONHostTapPendingBindingsAndBatch(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	var factory mage.CardFactory
	for _, node := range nodes {
		if node.Fields["Name"].String == "Spirit Shackle" {
			factory, err = Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if factory == nil {
		t.Fatal("missing Spirit Shackle")
	}
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("First", "", 4, 8)
			hostCard.SetOwner(b.PlayerID())
			otherCard := mage.NewCreature("Second", "", 4, 8)
			otherCard.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(hostCard, b.PlayerID())
			second := g.PutOnBattlefield(otherCard, b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), first.ID())
			if batch {
				card2 := factory()
				card2.SetOwner(a.PlayerID())
				source2 := g.PutOnBattlefield(card2, a.PlayerID())
				g.Attach(source2.ID(), second.ID())
				if err := mage.ApplyEffect(g, mage.TapEach(mage.MatchingPermanents(mage.IsCreature)), a.PlayerID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			} else {
				g.TapPermanent(first)
				g.Attach(source.ID(), second.ID())
				g.TapPermanent(second)
			}
			pending := g.Clone()
			for _, game := range []*mage.Game{g, pending} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatal("events lost")
				}
				game.ResolveStack()
				if game.FindPermanent(first.ID()).Counters[core.M0M2] != 1 || game.FindPermanent(second.ID()).Counters[core.M0M2] != 1 {
					t.Fatal("event bindings collided")
				}
			}
		})
	}
}
