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

var entryLands = []struct {
	name  string
	color core.Color
}{
	{"Dwarven Ruins", core.Red}, {"Ebon Stronghold", core.Black},
	{"Havenwood Battleground", core.Green}, {"Ruins of Trokair", core.White},
	{"Svyelunite Temple", core.Blue},
}

func TestRONEntryTappedLands(t *testing.T) {
	for _, tc := range entryLands {
		for _, ability := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, ability), func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, "FEM", tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				if err := g.PlayLand(a.PlayerID(), c.ID()); err != nil {
					t.Fatal(err)
				}
				p := g.FindPermanent(c.ID())
				if p == nil || !p.Tapped || g.StackSize() != 0 {
					t.Fatal("land did not enter tapped without a stack action")
				}
				if g.CanAfford(a.PlayerID(), core.ParseManaCost("{1}"), nil) {
					t.Fatal("tapped land supplied solver mana")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), ability, nil); err == nil {
					t.Fatal("tapped source paid tap cost")
				}
				if a.ManaPool().TotalMana() != 0 || g.FindPermanent(p.ID()) == nil {
					t.Fatal("failed payment changed state")
				}
				g.UntapPermanent(g.FindPermanent(p.ID()))
				if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), ability, nil); err == nil {
					t.Fatal("owner activated another player's land")
				}
				before := g.Clone()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), ability, nil); err != nil {
					t.Fatal(err)
				}
				if g.StackSize() != 0 || a.ManaPool().Count(tc.color) != ability+1 || b.ManaPool().TotalMana() != 0 {
					t.Fatal("mana amount, color, controller or timing")
				}
				if before.FindPermanent(p.ID()) == nil || before.FindPermanent(p.ID()).Tapped || before.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("payment changed prior clone")
				}
				if ability == 1 {
					if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 {
						t.Fatal("sacrifice did not go to owner")
					}
				} else if !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("tap cost was not paid")
				}
			})
		}
	}
}

func TestRONEntryTappedLifecycle(t *testing.T) {
	for _, tc := range append(append([]struct {
		name  string
		color core.Color
	}{}, entryLands...), struct {
		name  string
		color core.Color
	}{"Nevinyrral's Disk", core.Colorless}) {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "FEM"
			if tc.name == "Nevinyrral's Disk" {
				set = "LEB"
			}
			c := repositoryCard(t, set, tc.name)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, b.PlayerID())
			if !p.Tapped {
				t.Fatal("direct entry was untapped")
			}
			g.UntapPermanent(p)
			clone := g.Clone()
			g.PhaseIn(g.PhaseOut(p.ID()))
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("phasing repeated entry replacement")
			}
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(c.ID())
			returned := g.PutOnBattlefield(c, a.PlayerID())
			if !returned.Tapped {
				t.Fatal("return did not apply entry replacement")
			}
			if clone.FindPermanent(p.ID()) == nil || clone.FindPermanent(p.ID()).Tapped {
				t.Fatal("entry changed clone")
			}
			fresh := repositoryCard(t, set, tc.name)
			fresh.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(fresh, b.PlayerID())
			g.UntapPermanent(returned)
			if !g.FindPermanent(second.ID()).Tapped {
				t.Fatal("independent card instances shared status")
			}
		})
	}
}

func TestRONEntryTappedDisk(t *testing.T) {
	for _, departure := range []bool{false, true} {
		t.Run(fmt.Sprint(departure), func(t *testing.T) {
			g, a, b := ronGame()
			disk := repositoryCard(t, "LEB", "Nevinyrral's Disk")
			if err := mage.ValidateCardDefinition(disk.Name(), disk); err != nil {
				t.Fatal(err)
			}
			disk.SetOwner(a.PlayerID())
			a.AddToHand(disk)
			if err := g.CastSpellByID(a.PlayerID(), disk.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast did not roll back")
			}
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.CastSpellByID(a.PlayerID(), disk.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(disk.ID()) != nil {
				t.Fatal("entry happened before spell resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(disk.ID())
			if source == nil || !source.Tapped {
				t.Fatal("cast Disk did not enter tapped")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("tapped Disk activated")
			}
			if a.ManaPool().TotalMana() != 1 {
				t.Fatal("illegal activation consumed mana")
			}
			g.UntapPermanent(source)
			a.ManaPool().Clear()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("unpaid Disk activated")
			}
			if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("failed activation paid tap cost")
			}
			add := func(c mage.Card) *mage.Permanent {
				c.SetOwner(b.PlayerID())
				return g.PutOnBattlefield(c, b.PlayerID())
			}
			creature := add(mage.NewCreature("Creature", "", 2, 2))
			both := add(mage.NewCreature("Both", "", 2, 2, mage.WithCardType(core.TypeArtifact)))
			enchantment := add(mage.NewEnchantment("Enchantment", ""))
			observer := mage.WhenEvent(mage.CreatureDies(), mage.GainLifeFor(mage.You, mage.Fixed(1))).(*mage.GenericTriggered)
			matchesDeath := observer.Condition
			observer.SetCondition(func(event *core.GameEvent, game mage.GameReader, sourceID, controllerID uuid.UUID) bool {
				return matchesDeath(event, game, sourceID, controllerID) && game.FindPermanent(creature.ID()) != nil && game.FindPermanent(enchantment.ID()) != nil && (game.FindPermanent(source.ID()) != nil) != departure
			})
			land := add(mage.NewLand("Land", mage.WithAbility(observer)))
			phased := add(mage.NewArtifact("Phased", ""))
			g.PhaseOut(phased.ID())
			protected := add(mage.NewArtifact("Protected", "", mage.WithKeyword(core.Indestructible)))
			g.AddRegenerationShield(both.ID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 1 || g.FindPermanent(creature.ID()) == nil {
				t.Fatal("activation timing or costs")
			}
			if departure {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanent(creature.ID()) == nil {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(b.PlayerID()).Life() != 21 {
					t.Fatalf("death trigger saw an intermediate destruction state: life %d", game.GetPlayer(b.PlayerID()).Life())
				}
				if game.FindPermanent(creature.ID()) != nil || game.FindPermanent(enchantment.ID()) != nil || game.FindPermanent(source.ID()) != nil {
					t.Fatal("destruction omitted a type or source")
				}
				if game.FindPermanent(both.ID()) == nil || game.HasRegenerationShield(both.ID()) {
					t.Fatal("union destroyed artifact creature more than once or bypassed regeneration")
				}
				if game.FindPermanent(land.ID()) == nil || game.FindPermanent(protected.ID()) == nil || game.FindPermanentIncludingPhased(phased.ID()) == nil {
					t.Fatal("destruction included land, phased or indestructible permanent")
				}
			}
		})
	}
}

func TestRONEntryTappedInvalid(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent)`, `AsPermanentEnters(ThisPermanent, EntersTapped)`,
		`AsPermanentEnters(ThisPermanent, [])`, `AsPermanentEnters(ThisPermanent, [EntersTapped()])`,
		`AsPermanentEnters(ThisPermanent(), [EntersTapped])`, `AsPermanentEnters(HostPermanent, [EntersTapped])`,
		`AsPermanentEnters(Ref_TargetPermanent, [EntersTapped])`, `AsPermanentEnters(ThisPermanent, [EntersTapped, DrawNumberCards(Integer(1))])`,
		`AsPermanentEnters(ThisPermanent, [ChooseAPlayer(Opponent())])`, `EntersTapped`,
		`Activated(TapPermanent(ThisPermanent), ActionList([EntersTapped]))`,
	} {
		t.Run(rule, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)
			nodes, err := Parse(strings.NewReader(src), "invalid-entry.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid entry declaration accepted")
			}
		})
	}
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[EntersTapped])])`
	nodes, err := Parse(strings.NewReader(src), "instant-entry.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("spell entry declaration accepted")
	}
}

func TestRONEntryTappedInventory(t *testing.T) {
	src := `Card(Name:"Entry",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[EntersTapped])])`
	report, err := InspectFS(fstest.MapFS{"entry.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ThisPermanent": "permanent_reference", "EntersTapped": "entry_operation"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
				if s.Support != "unverified" {
					t.Fatal("syntax claimed gameplay support")
				}
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONEntryTappedEventTiming(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersTapped]),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(2))]))`)
	g, a, _ := ronGame()
	c := factory()
	c.SetOwner(a.PlayerID())
	c.Abilities()[0].(*mage.GenericTriggered).AndConditionData(mage.SourceIsTapped{})
	c.AddAbility(mage.WhenEvent(mage.BecomesTapped(mage.This), mage.GainLifeFor(mage.You, mage.Fixed(5))))
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.PutTriggersOnStack()
	if !p.Tapped || g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("entry trigger did not see tapped state or entry emitted a tap event")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("entry trigger failed tapped condition")
	}
	g.UntapPermanent(g.FindPermanent(p.ID()))
	g.TapPermanent(g.FindPermanent(p.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 27 {
		t.Fatal("later tap transition failed to trigger")
	}
}

func TestRONEntryTappedNativeUntapAndOwner(t *testing.T) {
	for _, tc := range entryLands {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", tc.name)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, b.PlayerID())
			g.SetActivePlayerIndex(0)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("another player's untap step untapped source")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("entry attribute prevented normal untap")
			}
			g.SetStep(core.PrecombatMain)
			g.AddRegenerationShield(p.ID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 || b.ManaPool().Count(tc.color) != 2 {
				t.Fatal("sacrifice used regeneration, controller graveyard or wrong mana recipient")
			}
		})
	}
}

func TestRONEntryTappedCardCopy(t *testing.T) {
	g, a, _ := ronGame()
	c := repositoryCard(t, "LEB", "Nevinyrral's Disk")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.UntapPermanent(p)
	copied := c.Copy()
	q := g.PutOnBattlefield(copied, a.PlayerID())
	if !q.Tapped || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("card copy lost entry rule or copied battlefield status")
	}
}
