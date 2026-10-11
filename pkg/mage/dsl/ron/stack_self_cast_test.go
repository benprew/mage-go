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

func TestRONStackSelfCastManaVortex(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "missing", "failed", "copy-spell", "departed", "controller", "copy-trigger"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: scenario != "decline"}
			b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "DRK", "Mana Vortex")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			var land *mage.Permanent
			if scenario != "missing" {
				land = g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
				a.choice = land.ID()
			}
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
			if scenario != "failed" {
				a.ManaPool().Add(core.Blue, 2)
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
			if scenario == "failed" {
				g.PutTriggersOnStack()
				if err == nil || g.StackSize() != 0 || a.offers != 0 || len(a.Hand()) != 1 {
					t.Fatal("failed cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			spell := g.StackPeek()
			if scenario == "copy-spell" {
				g.CopyStackObjectDirect(spell, b.PlayerID(), false)
			}
			if scenario == "departed" {
				g.CounterStackObject(spell.ID)
			}
			if scenario == "controller" {
				spell.Controller = b.PlayerID()
			}

			g.PutTriggersOnStack()
			want := 2
			if scenario == "copy-spell" {
				want = 3
			}
			if scenario == "departed" {
				want = 1
			}
			if g.StackSize() != want || !g.StackPeek().IsAbility {
				t.Fatalf("stack = %d, want %d", g.StackSize(), want)
			}
			if scenario == "copy-trigger" {
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
				g.ResolveTopOfStack()
			}
			g.ResolveTopOfStack()
			paid := scenario != "decline" && scenario != "missing"
			if land != nil && (g.FindPermanent(land.ID()) == nil) != paid {
				t.Fatal("incorrect payment")
			}
			if scenario == "copy-trigger" {
				if g.StackSize() != 0 {
					t.Fatal("independent second payment did not counter spell")
				}
				return
			}
			if !paid {
				if g.StackSize() != 0 {
					t.Fatal("unpaid spell survived")
				}
				return
			}
			if scenario == "departed" {
				if g.StackSize() != 0 || a.offers != 1 {
					t.Fatal("pending trigger did not survive departure")
				}
				return
			}
			for g.StackSize() > 0 {
				g.ResolveTopOfStack()
			}
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("paid spell failed to enter")
			}
		})
	}
}

func TestRONStackSelfCastInvalid(t *testing.T) {
	for _, rule := range []string{
		`FromStack()`, `FromStack(TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([])))`,
		`FromStack(TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You),SingleSpell(ThisSpell())),ActionList([])))`,
		`FromStack(TriggerA(WhenAPlayerCastsASpell(SinglePlayer(Opponent),SingleSpell(ThisSpell)),ActionList([])))`,
		`FromStack(TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([])))`,
		`TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You),SingleSpell(ThisSpell)),ActionList([]))`,
		`FromStack(TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You),SingleSpell(ThisSpell)),ActionList([SacrificePermanent(ThisPermanent)])))`,
	} {
		n := nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONStackSelfCastManaVortexBattlefield(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
		b := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}
		g := mage.NewGame(a, b)
		g.SetTurn(2)
		source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Mana Vortex"), a.PlayerID())
		la := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
		lb := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
		g.PutTriggersOnStack()
		for g.StackSize() > 0 {
			g.CounterStackObject(g.StackPeek().ID)
		}
		if phase {
			g.PhaseOut(source.ID())
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		if phase {
			if g.StackSize() != 0 {
				t.Fatal("phased source triggered")
			}
			g.PhaseIn([]uuid.UUID{source.ID()})
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
		}
		g.ResolveTopOfStack()
		if g.FindPermanent(lb.ID()) != nil || g.FindPermanent(la.ID()) == nil {
			t.Fatal("wrong upkeep player sacrificed")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveTopOfStack()
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("global absence did not trigger")
		}
		g.ResolveTopOfStack()
		if g.FindPermanent(source.ID()) != nil {
			t.Fatal("no-land source survived")
		}
	}
}

func TestRONStackSelfCastCloneAndRecast(t *testing.T) {
	a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: false}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	card := repositoryCard(t, "DRK", "Mana Vortex")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 4)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	old := g.StackPeek()
	g.CounterStackObject(old.ID)
	g.PutTriggersOnStack()
	oldTrigger := g.StackPeek()
	fresh := &mage.StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID()}
	g.PushStack(fresh)
	clone := g.Clone()
	clone.ResolveStackObject(oldTrigger)
	if clone.StackSize() != 2 || clone.StackPeek().ID != fresh.ID {
		t.Fatal("old trigger affected new stack visit")
	}
	if a.offers != 0 {
		t.Fatal("clone used original player")
	}
	g.ResolveStackObject(oldTrigger)
	if g.StackSize() != 2 || g.StackPeek().ID != fresh.ID || a.offers != 1 {
		t.Fatal("old capture followed returned card")
	}
}

func TestRONStackSelfCastRecordsAndInventory(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Mana Vortex")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared trigger declaration")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(fstest.MapFS{"DRK.ron": &fstest.MapFile{Data: data}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range report.Issues {
		if issue.Card == "Mana Vortex" {
			t.Fatal(issue)
		}
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "ThisSpell" && signature.Role == "spell_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing self spell reference role")
	}
	line := ""
	for candidate := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(candidate, `Name: "Mana Vortex"`) {
			line = candidate
		}
	}
	files := fstest.MapFS{"vortex.ron": &fstest.MapFile{Data: []byte(line)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[FromStack(TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([])))])`)}
	registry := mage.NewRegistry()
	registry.Register("Sentinel", func() mage.Card { return mage.NewArtifact("Sentinel", "") })
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Sentinel"); err != nil {
		t.Fatal("invalid collection erased registry")
	}
	if _, err := registry.CreateCard("Mana Vortex"); err == nil {
		t.Fatal("invalid collection published partial registry")
	}
}
