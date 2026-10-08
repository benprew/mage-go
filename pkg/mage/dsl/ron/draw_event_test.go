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

func TestRONDrawEventUnderworldDreams(t *testing.T) {
	for _, change := range []string{"none", "departed", "returned", "phased-after", "copy", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Underworld Dreams")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(core.Black, 3)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			for range 3 {
				b.AddToLibrary(repositoryCard(t, "LEB", "Forest"))
			}
			a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			g.PlayerDrawCard(b)
			if b.Life() != 20 || g.StackSize() != 0 {
				t.Fatal("damage occurred before resolution")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatalf("draw triggers = %d", g.StackSize())
			}
			for _, obj := range g.StackObjects() {
				if len(obj.Targets) != 0 {
					t.Fatal("drawer became a target")
				}
			}
			switch change {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				if change == "returned" {
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phased-after":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "prevention":
				g.AddPreventionShield(b.PlayerID(), 2)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("clone changed live game")
			}
			g.ResolveStack()
			want := 18
			if change == "copy" {
				want = 17
			}
			if change == "prevention" {
				want = 20
			}
			if b.Life() != want || a.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("draw binding or source damage lost")
			}
		})
	}
}

func TestRONDrawEventBindings(t *testing.T) {
	for _, filter := range []string{"AnyPlayer", "Opponent", "SinglePlayer(You)"} {
		t.Run(filter, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Enchantment", `TriggerA(WhenAPlayerDrawsACard(`+filter+`),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2)))]))`)
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			for range 5 {
				a.AddToLibrary(repositoryCard(t, "LEB", "Forest"))
				b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
			}
			phase := g.PhaseOut(p.ID())
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phased source triggered")
			}
			g.PhaseIn(phase)
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			g.PutTriggersOnStack()
			want := 2
			if filter == "AnyPlayer" {
				want = 4
			}
			if g.StackSize() != want {
				t.Fatalf("pending events = %d", g.StackSize())
			}
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.PutOnBattlefield(c, a.PlayerID())
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			life := 22
			if filter == "AnyPlayer" {
				life = 24
			}
			if a.Life() != life || b.Life() != life || clone.GetPlayer(a.PlayerID()).Life() != life || clone.GetPlayer(b.PlayerID()).Life() != life {
				t.Fatal("event player or control context overwritten")
			}
		})
	}
}

func TestRONDrawEventActualDraws(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEG", "Underworld Dreams")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	b.AddToLibrary(repositoryCard(t, "LEB", "Forest"))
	a.AddToLibrary(repositoryCard(t, "LEB", "Island"))
	g.RunStepWithPriority(core.Draw)
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("controller draw triggered")
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(p.ID(), b.PlayerID(), func(*mage.Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 1 {
		t.Fatal("replaced draw triggered")
	}
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("completed draw did not deal damage")
	}
}

func TestRONDrawEventDeclarations(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerDrawsACard(),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerDrawsACard(AnyPlayer()),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerDrawsACard(SinglePlayer(Opponent)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerDrawsACard(SinglePlayer(You())),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerDrawsACard(SinglePlayer(HostController)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerDrawsACard(AnyPlayer),ActionList([TapPermanent(Trigger_ThatPlayer)]))`,
		`TriggerA(WhenAPlayerDrawsACard(AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer(),GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPlayerDrawsACard(AnyPlayer),ActionList([GainLife(Integer(1))])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "invalid-draw.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONDrawEventFastingClause(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "DRK.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range nodes {
		if n.Fields["Name"].String != "Fasting" {
			continue
		}
		if _, err := Compile(n); err != nil {
			t.Fatal("full Fasting failed", err)
		}
		for _, rule := range n.Fields["Rules"].Args {
			if rule.Name != "TriggerA" || rule.Args[0].Name != "WhenAPlayerDrawsACard" {
				continue
			}
			found = true
			n.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			a.AddToLibrary(repositoryCard(t, "LEB", "Forest"))
			b.AddToLibrary(repositoryCard(t, "LEB", "Island"))
			g.PlayerDrawCard(b)
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent draw destroyed Fasting")
			}
			g.PlayerDrawCard(a)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.FindPermanent(p.ID()) == nil {
				t.Fatal("controller draw timing")
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("draw clause failed to destroy Fasting")
			}
		}
	}
	if !found {
		t.Fatal("missing real draw clause")
	}
}

func TestRONDrawEventInventory(t *testing.T) {
	source := `Card(Name:"Draw",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAPlayerDrawsACard(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`
	report, err := InspectFS(fstest.MapFS{"draw.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"WhenAPlayerDrawsACard": "event", "SinglePlayer": "player_filter", "You": "player_reference", "Trigger_ThatPlayer": "player_reference"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
				if s.Support != "unverified" {
					t.Fatal("inventory claimed gameplay")
				}
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONDrawEventSeparateInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `TriggerA(WhenAPlayerDrawsACard(Opponent),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2))),GainLife(Integer(1))]))`)
	g, a, b := ronGame()
	for _, player := range []*mage.BasePlayer{a, b} {
		c := factory()
		c.SetOwner(player.PlayerID())
		g.PutOnBattlefield(c, player.PlayerID())
		player.AddToLibrary(repositoryCard(t, "LEB", "Forest"))
	}
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("instance events lost")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone mutated live instances")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("factory instances shared bindings")
	}
}
