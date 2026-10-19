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

const saprolingRON = `TokenWithPT(PT(1, 1), Colors([Green]), [], [Creature], SubtypesList([Saproling]), [])`

func TestRONTokenCreationRealCards(t *testing.T) {
	for _, name := range []string{"Thallid", "Thallid Devourer", "Elvish Farmer"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", name)
			if err := mage.ValidateCardDefinition(name, c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if p.Counters[core.Spore] != 0 {
				t.Fatal("counter arrived before resolution")
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Spore] != 1 {
				t.Fatal("missing upkeep counter")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err == nil {
				t.Fatal("unpaid activation accepted")
			}
			g.MutablePermanent(p.ID()).AddCounter(core.Spore, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.FilterBattlefield(mage.IsToken)) != 0 || g.FindPermanent(p.ID()).Counters[core.Spore] != 0 {
				t.Fatal("creation did not wait for resolution")
			}
			g.ResolveStack()
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != 1 {
				t.Fatal("wrong token count")
			}
			token := tokens[0]
			if token.Name() != "Saproling Token" || !token.HasSubType("Saproling") || token.CurrentPower(g) != 1 || token.CurrentToughness(g) != 1 || token.ControllerID() != a.PlayerID() || token.Card.Owner() != a.PlayerID() || !slices.Contains(token.Colors(), core.Green) {
				t.Fatal("wrong token characteristics")
			}
			if name != "Thallid" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
					t.Fatal(err)
				}
				if len(g.FilterBattlefield(mage.IsToken)) != 0 {
					t.Fatal("sacrifice not paid")
				}
				g.ResolveStack()
				if name == "Elvish Farmer" && a.Life() != 22 {
					t.Fatal("missing life")
				}
				if name == "Thallid Devourer" && (g.FindPermanent(p.ID()).CurrentPower(g) != 3 || g.FindPermanent(p.ID()).CurrentToughness(g) != 4) {
					t.Fatal("missing bonus")
				}
				g.RunStepWithPriority(core.Cleanup)
				if name == "Thallid Devourer" && g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
					t.Fatal("bonus did not expire")
				}
			}
		})
	}
	for _, tc := range []struct {
		set, name, token string
		colors           []core.Color
		flying           bool
	}{
		{"LEB", "The Hive", "Wasp", nil, true}, {"LEG", "Boris Devilboon", "Minor Demon", []core.Color{core.Black, core.Red}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			c := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.RunStepWithPriority(core.Untap)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.Colorless, 5)
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(p.ID()).Tapped || len(g.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("tap cost or timing")
			}
			g.ResolveStack()
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != 1 || tokens[0].Name() != tc.token || tokens[0].HasKeyword(core.Flying) != tc.flying || len(tokens[0].Colors()) != len(tc.colors) {
				t.Fatal("wrong named token")
			}
			for _, color := range tc.colors {
				if !slices.Contains(tokens[0].Colors(), color) {
					t.Fatal("missing color")
				}
			}
			if tc.flying && !tokens[0].HasType(core.TypeArtifact) {
				t.Fatal("missing artifact type")
			}
		})
	}
}

func TestRONTokenCreationTownBatch(t *testing.T) {
	g, a, _ := ronGame()
	var seen []int
	watcher := mage.NewCreature("Watcher", "", 1, 1, mage.WithAbility(mage.NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, r mage.GameReader, _, _ uuid.UUID) bool {
		if evt.ToZone == core.ZoneBattlefield {
			if p := r.FindPermanent(evt.SourceID); p != nil && p.IsToken {
				seen = append(seen, len(r.FilterBattlefield(mage.IsToken)))
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	c := repositoryCard(t, "FEM", "Icatian Town")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.White, 1)
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(seen) != 4 {
		t.Fatal(seen)
	}
	for _, n := range seen {
		if n != 4 {
			t.Fatal("tokens did not enter together", seen)
		}
	}
	for _, p := range g.FilterBattlefield(mage.IsToken) {
		if p.Name() != "Citizen Token" || !slices.Contains(p.Colors(), core.White) || p.Card.ManaCost().CMC() != 0 {
			t.Fatal("wrong Citizen")
		}
	}
	if len(a.Graveyard()) != 1 {
		t.Fatal("spell did not finish")
	}
}

func TestRONTokenCreationCopiesAndSource(t *testing.T) {
	for _, change := range []string{"leave", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Thallid")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Spore, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "leave":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(c, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(clone.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				tokens := game.FilterBattlefield(mage.IsToken)
				if len(tokens) != 2 {
					t.Fatal("wrong count")
				}
				owners := map[uuid.UUID]int{}
				for _, token := range tokens {
					owners[token.Card.Owner()]++
					if token.ControllerID() != token.Card.Owner() {
						t.Fatal("wrong controller")
					}
				}
				if owners[a.PlayerID()] != 1 || owners[b.PlayerID()] != 1 {
					t.Fatal("lost resolving controller")
				}
			}
		})
	}
}

func TestRONTokenCreationInvalid(t *testing.T) {
	valid := `CreateTokens([` + saprolingRON + `], [])`
	for _, bad := range []string{
		`CreateTokens([], [])`, `CreateTokens([` + saprolingRON + `,` + saprolingRON + `], [])`, `CreateTokens([` + saprolingRON + `], [Tapped])`,
		strings.Replace(valid, `PT(1, 1)`, `PT(ValueX, 1)`, 1), strings.Replace(valid, `Colors([Green])`, `Colors([Colorless])`, 1),
		strings.Replace(valid, `[], [Creature]`, `[Legendary, Legendary], [Creature]`, 1), strings.Replace(valid, `[Creature]`, `[Sorcery]`, 1),
		strings.Replace(valid, `[Creature]`, `[Creature, Creature]`, 1), strings.Replace(valid, `Saproling`, `Island`, 1),
		strings.Replace(valid, `SubtypesList([Saproling]), []`, `SubtypesList([Saproling]), [GainLife(Integer(1))]`, 1),
		`CreateTokens([NumberTokens(ValueX, ` + saprolingRON + `)], [])`,
		`CreateTokens([NamedTokenWithPT("", PT(1,1), Colorless, [], [Creature], SubtypesList([Saproling]), [])], [])`,
	} {
		t.Run(bad, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+bad+`]))])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("invalid token declaration accepted")
			}
		})
	}
}

func TestRONTokenCreationInventory(t *testing.T) {
	src := `Card(Name:"Tokens",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([NumberTokens(Integer(4), ` + saprolingRON + `)], [])]))])`
	report, err := InspectFS(fstest.MapFS{"tokens.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"NumberTokens": "token_instruction", "TokenWithPT": "token_definition", "PT": "token_pt", "Colors": "token_colors", "SubtypesList": "token_subtypes", "Green": "token_color", "Saproling": "token_subtype", "Creature": "token_card_type"}
	for name, role := range expected {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
				if entry.Support != "unverified" {
					t.Fatal("inventory claims gameplay")
				}
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONTokenCreationCountsAndRollback(t *testing.T) {
	for _, amount := range []string{"Integer(0)", "Integer(-2)", "Integer(2)", "ValueX"} {
		t.Run(amount, func(t *testing.T) {
			src := `Card(Name:"Count",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(ActionList([CreateTokens([NumberTokens(` + amount + `, ` + saprolingRON + `)], [])]))])`
			nodes, err := Parse(strings.NewReader(src), "count.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			c := factory()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 3); err == nil {
				t.Fatal("unpayable cast accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 3); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			want := 0
			if amount == "Integer(2)" {
				want = 4
			}
			if amount == "ValueX" {
				want = 6
			}
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != want {
				t.Fatalf("got %d want %d", len(tokens), want)
			}
			for _, p := range tokens {
				if p.Card.ManaCost().CMC() != 0 || len(p.Card.Abilities()) != 0 {
					t.Fatal("creation spell leaked characteristics")
				}
			}
			other := factory()
			if other.ID() == c.ID() {
				t.Fatal("factory reused card")
			}
		})
	}
	g, a, _ := ronGame()
	c := repositoryCard(t, "LEB", "The Hive")
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpaid Hive accepted")
	}
	if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 {
		t.Fatal("failed cost was not atomic")
	}
}

func TestRONTokenCreationTriggerAndLegality(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([CreateTokens([`+saprolingRON+`], [])]))`)
	c := factory()
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(c, b.PlayerID())
	g.ResolveStack()
	tokens := g.FilterBattlefield(mage.IsToken)
	if len(tokens) != 1 || tokens[0].ControllerID() != a.PlayerID() {
		t.Fatal("pending trigger followed returned source")
	}
	factory = permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreateTokens([`+saprolingRON+`], [])])))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(target.ID())
	g.ResolveStack()
	if len(g.FilterBattlefield(mage.IsToken)) != 1 {
		t.Fatal("all-illegal ability created tokens")
	}
	g.PhaseOut(tokens[0].ID())
	g.PhaseIn([]uuid.UUID{tokens[0].ID()})
	if !g.FindPermanent(tokens[0].ID()).IsToken {
		t.Fatal("phase cycle lost token")
	}
	g.DestroyPermanent(g.FindPermanent(tokens[0].ID()))
	g.CheckStateBasedActions()
	if len(g.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("token did not leave")
	}
}

func TestRONTokenCreationDeclarationBoundaries(t *testing.T) {
	action := `CreateTokens([` + saprolingRON + `], [])`
	for _, rule := range []string{
		`Activated(` + action + `, ActionList([GainLife(Integer(1))]))`,
		`PermanentLayerEffect(ThisPermanent, [` + action + `])`,
		`Activated(PayMana([]), ActionList([CreateTokens([TokenCopyOfPermanent(ThisPermanent, CopyEffects([]))], [])]))`,
		`Activated(PayMana([]), ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([]),[])], [])]))`,
		`Activated(PayMana([]), ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Saproling]),[Flying(1)])], [])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Boundary",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "boundary.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported declaration accepted", rule)
		}
	}
}
