package ron

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func repositoryCard(t *testing.T, set, name string) mage.Card {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String == name {
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			return factory()
		}
	}
	t.Fatalf("missing card %s", name)
	return nil
}

func ronGame() (*mage.Game, *mage.BasePlayer, *mage.BasePlayer) {
	a, b := mage.NewBasePlayer("Alice"), mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	return g, a, b
}

func TestRONBasicLandTypesSupplyMana(t *testing.T) {
	g, player, _ := ronGame()
	for _, name := range []string{"Forest", "Tropical Island"} {
		card := repositoryCard(t, "LEB", name)
		card.SetOwner(player.PlayerID())
		g.PutOnBattlefield(card, player.PlayerID())
	}
	if !g.CanAfford(player.PlayerID(), core.ParseManaCost("{G}{U}"), nil) {
		t.Fatal("RON land types did not supply their intrinsic mana abilities")
	}
}

func TestRONDamageUsesPreventionAndTargetLegality(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "prevent", true: "illegal"}[remove], func(t *testing.T) {
			g, player, opponent := ronGame()
			creature := repositoryCard(t, "LEB", "Air Elemental")
			creature.SetOwner(opponent.PlayerID())
			perm := g.PutOnBattlefield(creature, opponent.PlayerID())
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			bolt.SetOwner(player.PlayerID())
			player.AddToHand(bolt)
			player.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(player.PlayerID(), bolt.ID(), []uuid.UUID{perm.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if remove {
				g.DestroyPermanent(perm)
			} else {
				g.AddPreventionShield(perm.ID(), 2)
			}
			g.ResolveStack()
			if !remove && g.FindPermanent(perm.ID()).Damage != 1 {
				t.Fatal("damage did not use prevention")
			}
			if remove && g.FindPermanent(perm.ID()) != nil {
				t.Fatal("removed target returned")
			}
			if opponent.Life() != 20 {
				t.Fatal("illegal permanent target dealt damage to a player")
			}
		})
	}
}

func TestRONAncestralRecallDrawsForTargetPlayer(t *testing.T) {
	g, player, opponent := ronGame()
	for range 5 {
		card := repositoryCard(t, "LEB", "Forest")
		card.SetOwner(opponent.PlayerID())
		opponent.AddToLibrary(card)
	}
	spell := repositoryCard(t, "LEB", "Ancestral Recall")
	spell.SetOwner(player.PlayerID())
	player.AddToHand(spell)
	player.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(player.PlayerID(), spell.ID(), []uuid.UUID{opponent.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(opponent.Hand()) != 3 || len(player.Hand()) != 0 {
		t.Fatal("draw actions used the wrong player")
	}
}

func TestRONRegenerationUsesItsDeclaredSubject(t *testing.T) {
	for _, targeted := range []bool{false, true} {
		t.Run(map[bool]string{false: "source", true: "target"}[targeted], func(t *testing.T) {
			g, player, _ := ronGame()
			card := repositoryCard(t, "LEB", "Drudge Skeletons")
			card.SetOwner(player.PlayerID())
			permanent := g.PutOnBattlefield(card, player.PlayerID())
			if targeted {
				spell := repositoryCard(t, "LEB", "Death Ward")
				spell.SetOwner(player.PlayerID())
				player.AddToHand(spell)
				player.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(player.PlayerID(), spell.ID(), []uuid.UUID{permanent.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				player.ManaPool().Add(core.Black, 1)
				if err := g.ActivateAbilityByIndex(player.PlayerID(), permanent.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if !g.HasRegenerationShield(permanent.ID()) {
				t.Fatal("regeneration did not use its declared subject")
			}
			g.DestroyPermanent(g.FindPermanent(permanent.ID()))
			if g.FindPermanent(permanent.ID()) == nil || g.HasRegenerationShield(permanent.ID()) {
				t.Fatal("destruction did not consume the regeneration shield")
			}
		})
	}
}

func TestRONActivatedManaPaysTapAndSacrificeCosts(t *testing.T) {
	g, player, _ := ronGame()
	lotus := repositoryCard(t, "LEB", "Black Lotus")
	lotus.SetOwner(player.PlayerID())
	perm := g.PutOnBattlefield(lotus, player.PlayerID())
	if err := g.ActivateAbilityByIndex(player.PlayerID(), perm.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(perm.ID()) != nil || len(player.Graveyard()) != 1 {
		t.Fatal("mana activation did not sacrifice its source")
	}
	if player.ManaPool().TotalMana() != 3 {
		t.Fatal("mana activation did not produce three mana")
	}
	if !g.GetStack().IsEmpty() {
		t.Fatal("mana ability used the stack")
	}
}

func TestRONStaticLayersTrackAuraAndGlobalBoosts(t *testing.T) {
	g, player, _ := ronGame()
	creature := repositoryCard(t, "LEB", "White Knight")
	creature.SetOwner(player.PlayerID())
	perm := g.PutOnBattlefield(creature, player.PlayerID())
	crusade := repositoryCard(t, "LEB", "Crusade")
	crusade.SetOwner(player.PlayerID())
	global := g.PutOnBattlefield(crusade, player.PlayerID())
	if g.FindPermanent(perm.ID()).CurrentPower(g) != 3 {
		t.Fatal("global boost did not apply")
	}
	g.DestroyPermanent(global)
	if g.FindPermanent(perm.ID()).CurrentPower(g) != 2 {
		t.Fatal("global boost survived its source")
	}
	aura := repositoryCard(t, "LEB", "Holy Strength")
	aura.SetOwner(player.PlayerID())
	attached := g.PutOnBattlefield(aura, player.PlayerID())
	g.Attach(attached.ID(), perm.ID())
	if g.FindPermanent(perm.ID()).CurrentPower(g) != 3 || g.FindPermanent(perm.ID()).CurrentToughness(g) != 4 {
		t.Fatal("Aura boost did not apply to its host")
	}
	g.DestroyPermanent(attached)
	if g.FindPermanent(perm.ID()).CurrentToughness(g) != 2 {
		t.Fatal("Aura boost survived removal")
	}
}

func TestRONColorIndicatorOverridesManaCostColors(t *testing.T) {
	g, player, _ := ronGame()
	card := repositoryCard(t, "LEG", "Crimson Kobolds")
	card.SetOwner(player.PlayerID())
	perm := g.PutOnBattlefield(card, player.PlayerID())
	colors := g.EffectiveColors(perm.ID())
	if len(colors) != 1 || colors[0] != core.Red {
		t.Fatalf("color indicator: %v", colors)
	}
}

func TestRONSelfEntryTriggerUsesEngineEvent(t *testing.T) {
	g, player, _ := ronGame()
	for range 2 {
		card := repositoryCard(t, "LEB", "Forest")
		card.SetOwner(player.PlayerID())
		player.AddToLibrary(card)
	}
	src := `Card(Name: "RON Draw On Entry", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: [Human]), CardPT: (Power: 1, Toughness: 1), Rules: [TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([DrawNumberCards(Integer(1))]))])`
	nodes, err := Parse(strings.NewReader(src), "trigger.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	card.SetOwner(player.PlayerID())
	g.PutOnBattlefield(card, player.PlayerID())
	g.ResolveStack()
	if len(player.Hand()) != 1 {
		t.Fatal("entry trigger did not draw one card")
	}
}
