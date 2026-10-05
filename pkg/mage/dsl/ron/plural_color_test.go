package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPluralColorRealCards(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Dwarven Song", core.Red}, {"Heaven's Gate", core.White}, {"Sea Kings' Blessing", core.Blue}, {"Sylvan Paradise", core.Green}, {"Touch of Darkness", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", tc.name)
			if err := mage.ValidateCardDefinition(tc.name, c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			p := g.PutOnBattlefield(mage.NewCreature("First", "{W}{U}", 2, 3), a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 4), b.PlayerID())
			r := g.PutOnBattlefield(mage.NewCreature("Unselected", "{W}{U}", 1, 1), b.PlayerID())
			ids := []uuid.UUID{p.ID(), q.ID()}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), ids, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(tc.color, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), ids, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !slices.Equal(p.Colors(), []core.Color{core.White, core.Blue}) {
				t.Fatal("payment or resolution timing")
			}
			clone := g.Clone()
			g.ResolveStack()
			for _, id := range ids {
				if !slices.Equal(g.FindPermanent(id).Colors(), []core.Color{tc.color}) {
					t.Fatal("did not replace colors")
				}
			}
			if !slices.Equal(r.Colors(), []core.Color{core.White, core.Blue}) || len(clone.FindPermanent(q.ID()).Colors()) != 0 {
				t.Fatal("membership or clone isolation")
			}
			clone.ResolveStack()
			if !slices.Equal(clone.FindPermanent(q.ID()).Colors(), []core.Color{tc.color}) {
				t.Fatal("pending clone lost targets")
			}
			g.RunStepWithPriority(core.Cleanup)
			if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.White, core.Blue}) || len(g.FindPermanent(q.ID()).Colors()) != 0 {
				t.Fatal("color duration did not expire")
			}
			if !slices.Equal(clone.FindPermanent(p.ID()).Colors(), []core.Color{tc.color}) {
				t.Fatal("cleanup changed clone")
			}
		})
	}
}

func TestRONPluralColorInvalid(t *testing.T) {
	target := `OneOrMoreTargetPermanents(IsCardtype(Creature))`
	action := `CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,[SetColor(SimpleColorList([Red]))],UntilEndOfTurn)`
	for _, rule := range []string{
		`SpellActions(Targeted([` + target + `],ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([` + action + `]))])))`,
		`SpellActions(Modal_ChooseOne([Targeted([` + target + `],ActionList([` + action + `])),ActionList([` + action + `])]))`,
		`SpellActions(Targeted([` + target + `],ActionList([CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,[SetColor(SimpleColorList([Red])),SetColor(SimpleColorList([Blue]))],UntilEndOfTurn)])))`,

		`SpellActions(Targeted([OneOrMoreTargetPermanents],ActionList([` + action + `])))`,
		`SpellActions(Targeted([OneOrMoreTargetPermanents()],ActionList([` + action + `])))`,
		`SpellActions(Targeted([` + target + `],ActionList([` + strings.Replace(action, "Ref_TargetPermanents", "Ref_TargetPermanent", 1) + `])))`,
		`SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`,
		`SpellActions(ActionList([` + action + `]))`,
		`SpellActions(Targeted([` + target + `],ActionList([` + strings.Replace(action, "[Red]", "[Colorless]", 1) + `])))`,
		`SpellActions(Targeted([` + target + `],ActionList([` + strings.Replace(action, "[Red]", "[Red()]", 1) + `])))`,
		`SpellActions(Targeted([` + target + `],ActionList([` + strings.Replace(action, "[SetColor(SimpleColorList([Red]))]", "[]", 1) + `])))`,
		`SpellActions(Targeted([` + target + `],ActionList([` + strings.Replace(action, "UntilEndOfTurn", "UntilEndOfTurn()", 1) + `])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONPluralColorLegalityAndIdentity(t *testing.T) {
	for _, change := range []string{"shroud", "return", "phase", "phase-cycle", "control", "all-illegal", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Dwarven Song")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			victim := mage.NewCreature("Victim", "{U}", 2, 3)
			p := g.PutOnBattlefield(victim, b.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Other", "{U}", 2, 3), a.PlayerID())
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID(), q.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(victim, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "all-illegal":
				p.GrantBaseAttr(core.Shroud)
				q.GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				clone.PhaseIn([]uuid.UUID{p.ID()})
			}
			for _, game := range []*mage.Game{g, clone} {
				want := core.Red
				if change == "shroud" || change == "return" || change == "phase" || change == "all-illegal" {
					want = core.Blue
				}
				if !slices.Equal(game.FindPermanent(p.ID()).Colors(), []core.Color{want}) {
					t.Fatalf("wrong victim colors in %s", change)
				}
				want = core.Red
				if change == "all-illegal" {
					want = core.Blue
				}
				if !slices.Equal(game.FindPermanent(q.ID()).Colors(), []core.Color{want}) {
					t.Fatal("wrong other colors")
				}
				if change != "all-illegal" {
					game.PhaseIn(game.PhaseOut(q.ID()))
					if !slices.Equal(game.FindPermanent(q.ID()).Colors(), []core.Color{core.Red}) {
						t.Fatal("phase cycle ended captured effect")
					}
				}
			}
		})
	}
}

func TestRONPluralColorAnnouncement(t *testing.T) {
	for _, bad := range []string{"empty", "duplicate", "land", "shroud", "phased"} {
		t.Run(bad, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Dwarven Song")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			p := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 3), b.PlayerID())
			ids := []uuid.UUID{p.ID()}
			switch bad {
			case "empty":
				ids = nil
			case "duplicate":
				ids = []uuid.UUID{p.ID(), p.ID()}
			case "land":
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				ids = []uuid.UUID{land.ID()}
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
			case "phased":
				g.PhaseOut(p.ID())
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), ids, 0); err == nil {
				t.Fatal("illegal target declaration accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 {
				t.Fatal("target failure spent resources")
			}
		})
	}
}

func TestRONPluralColorInstancesAndLayerOrder(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Colors",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([OneOrMoreTargetPermanents(IsCardtype(Creature))],ActionList([CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,[SetColor(SimpleColorList([Green,Red]))],UntilEndOfTurn)])))])`), "colors.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "{U}", 2, 2), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "{U}", 2, 2), b.PlayerID())
	for _, id := range []uuid.UUID{p.ID(), q.ID()} {
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{id}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	for _, id := range []uuid.UUID{p.ID(), q.ID()} {
		if !slices.Equal(g.FindPermanent(id).Colors(), []core.Color{core.Green, core.Red}) {
			t.Fatal("factory bindings crossed")
		}
	}
	c := repositoryCard(t, "LEG", "Touch of Darkness")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Black}) || !slices.Equal(g.FindPermanent(q.ID()).Colors(), []core.Color{core.Green, core.Red}) {
		t.Fatal("later color effect or independent membership")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.PutOnBattlefield(p.Card, a.PlayerID())
	g.ApplyContinuousEffects()
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Blue}) {
		t.Fatal("color effect followed returned incarnation")
	}
}

func TestRONPluralColorInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([OneOrMoreTargetPermanents(IsCardtype(Creature))],ActionList([CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,[SetColor(SimpleColorList([Red]))],UntilEndOfTurn),UnknownAction])))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"IsCardtype": "permanent_filter", "Ref_TargetPermanents": "permanent_collection_reference", "SimpleColorList": "color_list", "Red": "color"}
	for _, entry := range report.Signatures {
		if role, ok := want[entry.Name]; ok {
			if entry.Role != role {
				t.Fatalf("%s role=%s want=%s", entry.Name, entry.Role, role)
			}
			delete(want, entry.Name)
		}
	}
	if len(want) != 0 || report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatal("inventory lost rejected descendants")
	}
}

func TestRONPluralColorContexts(t *testing.T) {
	body := `Targeted([OneOrMoreTargetPermanents(IsCardtype(Creature))],ActionList([CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,[SetColor(SimpleColorList([]))],UntilEndOfTurn)]))`
	for _, context := range []string{"activation", "trigger", "mode"} {
		t.Run(context, func(t *testing.T) {
			rule := `Activated(PayMana([]),` + body + `)`
			kind := "Artifact"
			if context == "trigger" {
				rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + body + `)`
			}
			if context == "mode" {
				kind = "Instant"
				rule = `SpellActions(Modal_ChooseOne([` + body + `,ActionList([GainLife(Integer(1))])]))`
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Context",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:[]),Rules:[`+rule+`])`), "context.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			q := g.PutOnBattlefield(mage.NewCreature("Target", "{R}{G}", 2, 3), b.PlayerID())
			switch context {
			case "mode":
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{q.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "activation":
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				g.RemoveFromBattlefield(p)
			case "trigger":
				g.PutOnBattlefield(card, a.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
			}
			if g.StackSize() != 1 {
				t.Fatal("missing pending action")
			}
			g.ResolveStack()
			if len(g.FindPermanent(q.ID()).Colors()) != 0 {
				t.Fatal("empty list did not make target colorless")
			}
		})
	}
}

type pluralColorPlayer struct {
	*mage.BasePlayer
	targets []uuid.UUID
}

func (p *pluralColorPlayer) ChooseTargets(_ []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	return slices.Clone(p.targets)
}

func TestRONPluralColorRetargetCopy(t *testing.T) {
	a := &pluralColorPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	p := g.PutOnBattlefield(mage.NewCreature("Original", "{U}", 2, 2), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Copy target", "{U}", 2, 2), b.PlayerID())
	c := repositoryCard(t, "LEG", "Dwarven Song")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{q.ID()}
	cp := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if cp == nil || !slices.Equal(cp.Targets, []uuid.UUID{q.ID()}) {
		t.Fatal("copy did not use its own targets")
	}
	g.ResolveStackObject(cp)
	if !slices.Equal(g.FindPermanent(q.ID()).Colors(), []core.Color{core.Red}) || !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Blue}) {
		t.Fatal("copy bindings changed original")
	}
	g.ResolveStack()
	if !slices.Equal(g.FindPermanent(p.ID()).Colors(), []core.Color{core.Red}) {
		t.Fatal("original targets lost after copy")
	}
}
