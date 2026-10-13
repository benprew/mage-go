package ron

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func entryXFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Entry X",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[ManaCostX],Rules:[AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(ValueX,PTCounter(1,1))])],CardPT:(Power:0,Toughness:1))`), "entry-x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONEntryAnnouncedX(t *testing.T) {
	for _, x := range []int{0, 3, 300} {
		t.Run(fmt.Sprint(x), func(t *testing.T) {
			factory := entryXFactory(t)
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("entered before resolution")
			}
			cp := g.CopySpellOnStack(c.ID(), b.PlayerID(), false)
			if cp == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.SetXValue(77)
			g.ResolveStack()
			for _, id := range []mage.Card{c, cp.Card} {
				p := g.FindPermanent(id.ID())
				if p == nil || p.Counters[core.P1P1] != x || p.CurrentPower(g) != x {
					t.Fatal("entry did not use its own X")
				}
			}
			direct := g.PutOnBattlefield(factory(), b.PlayerID())
			if direct.Counters[core.P1P1] != 0 {
				t.Fatal("factory inherited X")
			}
			clone.ResolveStack()
			if clone.FindPermanent(c.ID()).Counters[core.P1P1] != x || clone.FindPermanent(direct.ID()) != nil {
				t.Fatal("clone shared entry state")
			}
		})
	}
}

func TestRONEntryAnnouncedXDeclarations(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(ValueX(),ChargeCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(ValueX(1),ChargeCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Trigger_AmountOfDamageDealt,ChargeCounter)])`,
		`AsPermanentEnters(HostPermanent,[EntersWithNumberCounters(ValueX,ChargeCounter)])`,
		`AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(ValueX,PoisonCounter)])`,
		`SpellActions(ActionList([EntersWithNumberCounters(ValueX,ChargeCounter)]))`,
		`Activated(PayMana([]),ActionList([EntersWithNumberCounters(ValueX,ChargeCounter)]))`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid-entry-x.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", rule)
		}
	}
}

func TestRONEntryAnnouncedXHydraRecord(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		n := repositoryRecord(t, set, "Rock Hydra", "hydra.ron")
		entryRecord := n
		entryRecord.Fields = maps.Clone(n.Fields)
		rules := n.Fields["Rules"]
		rules.Args = rules.Args[:1:1]
		entryRecord.Fields["Rules"] = rules
		factory, err := Compile(entryRecord)
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Red, 2)
		a.ManaPool().Add(core.Colorless, 4)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 4); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if p := g.FindPermanent(c.ID()); p == nil || p.Counters[core.P1P1] != 4 || p.CurrentToughness(g) != 4 {
			t.Fatal("Hydra entry contract failed")
		}
		if _, err := Compile(n); err == nil {
			t.Fatal("unsupported complete Hydra accepted")
		}
	}
}

func TestRONEntryAnnouncedXReplacementOrder(t *testing.T) {
	for _, x := range []int{0, 3} {
		for _, choice := range []int{0, 1} {
			t.Run(fmt.Sprintf("%d/%d", x, choice), func(t *testing.T) {
				a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: 1 - choice}
				b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: choice}
				g := mage.NewGame(a, b)
				donor := g.PutOnBattlefield(mage.NewArtifact("Donor", ""), a.PlayerID())
				g.AddETBAdditionalCounters(donor.ID(), core.P1P1, 1, mage.IsCreature, false)
				g.AddCounterDoubler(donor.ID(), core.P1P1, mage.IsCreature)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				c := entryXFactory(t)()
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
				b.ManaPool().Add(core.Colorless, x)
				g.SetActivePlayerIndex(1)
				if err := g.CastSpellByID(b.PlayerID(), c.ID(), nil, x); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				want := (x + 1) * 2
				if choice == 1 && x > 0 {
					want = x*2 + 1
				}
				if p := g.FindPermanent(c.ID()); p == nil || p.Counters[core.P1P1] != want {
					t.Fatalf("entry replacement order: %v, want %d", p, want)
				}
				if a.choices != 0 {
					t.Fatal("donor controller chose entry replacements")
				}
				if x > 0 && b.choices != 1 {
					t.Fatal("entering controller did not choose order")
				}
			})
		}
	}
}

func TestRONEntryAnnouncedXObserver(t *testing.T) {
	factory := entryXFactory(t)
	c := factory()
	sourceAbility := c.Abilities()[0]
	if factory().Abilities()[0].AbilityID() == sourceAbility.AbilityID() {
		t.Fatal("factory shared entry ability identity")
	}
	g, a, _ := ronGame()
	seen := 0
	observer := mage.NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader mage.GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == c.ID() && event.ToZone == core.ZoneBattlefield {
			p := reader.FindPermanent(c.ID())
			if p == nil || p.Counters[core.P1P1] != 3 {
				t.Fatal("entry observer did not see counters")
			}
			seen++
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(observer)), a.PlayerID())
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 3); err != nil {
		t.Fatal(err)
	}
	if seen != 0 {
		t.Fatal("entry event before resolution")
	}
	g.ResolveStack()
	if seen != 1 {
		t.Fatal("entry event missing or repeated")
	}
}
