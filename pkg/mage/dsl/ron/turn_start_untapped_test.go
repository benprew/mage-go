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

func TestRONTurnStartUntappedPowerSurge(t *testing.T) {
	for _, change := range []string{"normal", "depart", "control", "copy", "clone", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Power Surge")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Red, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			first := g.PutOnBattlefield(mage.NewLand("First"), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewLand("Second"), a.PlayerID())
			g.TapPermanent(second)
			g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
			g.RunStepWithPriority(core.Untap)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.Life() != 20 {
				t.Fatal("detection or early damage")
			}
			pending := g.Clone()
			switch change {
			case "depart":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "phase":
				g.PhaseOut(first.ID())
			case "return":
				g.RemoveFromBattlefield(first)
				g.PutOnBattlefield(first.Card, b.PlayerID())
			}
			g.TapPermanent(second)
			g.ResolveStack()
			want := 19
			if change == "copy" {
				want = 18
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatal("saved count or event player", a.Life(), b.Life())
			}
			if pending.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			pending.ResolveStack()
			if pending.GetPlayer(a.PlayerID()).Life() != 19 {
				t.Fatal("pending clone history")
			}
			g.AdvanceToNextTurn()
			g.BeginActiveTurn()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			expected := 19
			if change == "control" || change == "return" {
				expected = 18
			}
			if change == "depart" {
				expected = 20
			}
			if b.Life() != expected {
				t.Fatal("new turn snapshot", b.Life(), expected)
			}
		})
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		repositoryCard(t, set, "Power Surge")
	}
}

func TestRONTurnStartUntappedIntervening(t *testing.T) {
	for _, change := range []string{"untapped", "tapped-start", "tap-after", "return", "phase", "phase-cycle", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			n := hostUntapRecord(t, "LEG", "Rasputin Dreamweaver")
			rules := n.Fields["Rules"]
			// The counter cap is a separate unsupported declaration.
			rules.Args = rules.Args[:len(rules.Args)-1]
			n.Fields["Rules"] = rules
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if p.Counters[core.Dream] != 7 {
				t.Fatal("entry counters")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if p.Counters[core.Dream] != 6 || a.ManaPool().TotalMana() != 1 {
				t.Fatal("counter mana")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.DealDamageToPermanent(g.FindPermanent(p.ID()), 1, uuid.New())
			if g.FindPermanent(p.ID()).Damage != 0 || g.FindPermanent(p.ID()).Counters[core.Dream] != 5 {
				t.Fatal("counter prevention")
			}
			if change == "tapped-start" {
				g.TapPermanent(p)
			}
			g.RunStepWithPriority(core.Untap)
			if change == "tap-after" {
				g.TapPermanent(p)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if change == "tapped-start" {
				if g.StackSize() != 0 {
					t.Fatal("live untap supplied history")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("turn-start trigger")
			}
			pending := g.Clone()
			want := 6
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				want = 7
			case "phase":
				g.PhaseOut(p.ID())
				want = 5
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				want = 7
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.Dream] != want {
				t.Fatal("history recheck", change)
			}
			pending.ResolveStack()
			if pending.FindPermanent(p.ID()).Counters[core.Dream] != 6 {
				t.Fatal("clone isolation")
			}
		})
	}
}

func TestRONTurnStartUntappedInvalid(t *testing.T) {
	valid := `Card(Name:"History",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefieldAtBeginningOfTurn(And([IsUntapped,IsCardtype(Land),ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))])),Player(Trigger_ThatPlayer))]))])`
	for _, replacement := range []string{`And([])`, `IsUntapped`, `And([IsTapped,IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))])`, `And([IsUntapped(),IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))])`, `And([IsUntapped,IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))])`, `And([IsUntapped,IsCardtype(Land),ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer))])`} {
		src := strings.Replace(valid, `And([IsUntapped,IsCardtype(Land),ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer))])`, replacement, 1)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid history accepted", replacement)
		}
	}
	condition := `TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),PermanentPassesFilter(ThisPermanent,StartedThisTurnUntapped),ActionList([GainLife(Integer(1))]))`
	for _, invalid := range []string{strings.Replace(condition, "StartedThisTurnUntapped", "StartedThisTurnUntapped()", 1), strings.Replace(condition, "StartedThisTurnUntapped", "StartedThisTurnUntapped(ThisPermanent)", 1), strings.Replace(condition, "PermanentPassesFilter(ThisPermanent", "PermanentPassesFilter(HostPermanent", 1)} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+invalid+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid history condition accepted")
		}
	}

	n := hostUntapRecord(t, "LEG", "Rasputin Dreamweaver")
	if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "unsupported self untap operations") {
		t.Fatal("independent cap must reject", err)
	}
}

func TestRONTurnStartUntappedInventoryAndLoad(t *testing.T) {
	n := hostUntapRecord(t, "LEB", "Power Surge")
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	src := strings.Split(string(data), "\n")[n.Position.Line-1]
	files := fstest.MapFS{"surge.ron": {Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"TheNumberOfPermanentsOnTheBattlefieldAtBeginningOfTurn": "value", "And": "permanent_filter", "ControlledByAPlayer": "permanent_filter", "SinglePlayer": "player_filter", "Trigger_ThatPlayer": "player_reference"}
	for name, role := range roles {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic role", name, role)
		}
	}
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	one, two := factory(), factory()
	if one.ID() == two.ID() {
		t.Fatal("shared factory identity")
	}
	if err := mage.ValidateCardDefinition(one.Name(), one); err != nil {
		t.Fatal(err)
	}
}

func TestRONTurnStartUntappedAtomicLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	n := hostUntapRecord(t, "LEB", "Power Surge")
	src := strings.Split(string(data), "\n")[n.Position.Line-1]
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}, "b.ron": {Data: []byte(src)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.CreateCard("Power Surge"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(strings.Replace(src, "Power Surge", "Bad Surge", 1), "IsUntapped", "IsTapped", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial collection loaded")
	}
	if _, err := registry.CreateCard("Bad Surge"); err == nil {
		t.Fatal("bad record published")
	}
	if _, err := registry.CreateCard("Power Surge"); err != nil {
		t.Fatal("failed collection changed registry")
	}
}
