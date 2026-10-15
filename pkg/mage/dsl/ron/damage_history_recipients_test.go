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

const ronHistoryPlayers = `EachPlayer(And([Opponent,WasDealtDamageByPermanentThisGame(ThisPermanent)]))`
const ronHistoryWalkers = `EachPermanent(And([IsCardtype(Planeswalker),WasDealtDamageByPermanentThisGame(ThisPermanent)]))`

func TestRONDamageHistoryFallen(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "The Fallen")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("cast payment or timing")
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID())
	untouched := g.PutOnBattlefield(mage.NewPlaneswalker("Untouched", "", 10), b.PlayerID())
	for range 2 {
		g.DealDamageToPlayer(b, 1, p.ID())
		g.DealDamageToPermanent(walker, 1, p.ID())
	}
	g.DealDamageToPlayer(a, 1, p.ID())
	g.RunStepWithPriority(core.Cleanup)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if b.Life() != 18 || walker.Counters[core.Loyalty] != 8 {
		t.Fatal("opponent upkeep triggered")
	}
	callbacks := 0
	g.SetOnDamageDealt(func(source, target string, amount int, combat bool) {
		callbacks++
		if source != "The Fallen" || amount != 1 || combat || b.Life() != 17 || g.FindPermanent(walker.ID()).Counters[core.Loyalty] != 7 {
			t.Error("damage batch was not simultaneous")
		}
	})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	if b.Life() != 18 || walker.Counters[core.Loyalty] != 8 {
		t.Fatal("trigger resolved before the stack")
	}
	g.ResolveStack()
	if callbacks != 2 || a.Life() != 19 || untouched.Counters[core.Loyalty] != 10 {
		t.Fatal("history, opponents, or noncumulative damage")
	}
}

func TestRONDamageHistoryLifetime(t *testing.T) {
	for _, state := range []string{"present", "departed", "returned", "phased", "phase-cycle", "control", "recipient-return", "recipient-phased"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "The Fallen")
			p := g.PutOnBattlefield(card, a.PlayerID())
			walkerCard := mage.NewPlaneswalker("Walker", "", 10)
			walker := g.PutOnBattlefield(walkerCard, b.PlayerID())
			g.DealDamageToPlayer(b, 1, p.ID())
			g.DealDamageToPermanent(walker, 1, p.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			switch state {
			case "departed":
				g.RemoveFromBattlefield(p)
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				ids := g.PhaseOut(p.ID())
				g.PhaseIn(ids)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "recipient-return":
				g.RemoveFromBattlefield(walker)
				walker = g.PutOnBattlefield(walkerCard, b.PlayerID())
			case "recipient-phased":
				g.PhaseOut(walker.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			if b.Life() != 18 {
				t.Fatal("pending ability lost its source history or controller")
			}
			want := 8
			if state == "recipient-return" {
				want = 10
			}
			if state == "recipient-phased" {
				want = 9
			}
			if g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != want {
				t.Fatal("recipient identity or phasing")
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 19 {
				t.Fatal("clone changed before resolution")
			}
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 18 {
				t.Fatal("clone lost saved history")
			}
			if state == "returned" {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.ResolveStack()
				if b.Life() != 18 {
					t.Fatal("returned source inherited history")
				}
			}
		})
	}
}

func TestRONDamageHistoryPreventionAndInstances(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "DRK", "The Fallen"), a.PlayerID())
	other := g.PutOnBattlefield(repositoryCard(t, "DRK", "The Fallen"), a.PlayerID())
	walker := g.PutOnBattlefield(mage.NewPlaneswalker("Protected", "", 10), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, p.ID())
	g.DealDamageToPermanent(walker, 0, p.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if b.Life() != 20 || walker.Counters[core.Loyalty] != 10 {
		t.Fatal("prevented or zero damage entered history")
	}
	g.DealDamageToPlayer(b, 1, p.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.Stack().FindBySourceID(p.ID()), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatal("independent instances or copy lost history")
	}
	g.RemoveFromBattlefield(other)
	ids := g.PhaseOut(p.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(ids)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if b.Life() != 16 {
		t.Fatal("phase cycle lost history")
	}
}

func TestRONDamageHistoryInvalid(t *testing.T) {
	for _, players := range []string{
		`WasDealtDamageByPermanentThisGame`,
		`WasDealtDamageByPermanentThisGame()`,
		`WasDealtDamageByPermanentThisGame(ThisPermanent())`,
		`WasDealtDamageByPermanentThisGame(Ref_TargetPermanent)`,
		`WasDealtDamageByPermanentThisGame(EachablePermanent)`,
		`WasDealtDamageByPermanentThisGame(ThisSpell)`,
		`And([Opponent,WasDealtDamageByPermanentThisGame(ThisPermanent),Opponent])`,
		`And([WasDealtDamageByPermanentThisGame(ThisPermanent),WasDealtDamageByPermanentThisGame(ThisPermanent)])`,
		`Or([Opponent,WasDealtDamageByPermanentThisGame(ThisPermanent)])`,
	} {
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),MultipleRecipients([EachPlayer(` + players + `),` + ronHistoryWalkers + `]))]))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("accepted", players)
		}
	}
	for _, permanents := range []string{
		`And([])`,
		`And([IsCardtype(Planeswalker),WasDealtDamageByPermanentThisGame(ThisPermanent),WasDealtDamageByPermanentThisGame(ThisPermanent)])`,
		`And([IsCardtype(Planeswalker),WasDealtDamageByPermanentThisGame(Ref_TargetPermanent)])`,
		`Or([IsCardtype(Planeswalker),WasDealtDamageByPermanentThisGame(ThisPermanent)])`,
	} {
		rule := `Activated(PayMana([]),ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),MultipleRecipients([` + ronHistoryPlayers + `,EachPermanent(` + permanents + `)]))]))`
		source := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("accepted", permanents)
		}
	}
}

func TestRONDamageHistoryController(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "DRK", "The Fallen"), a.PlayerID())
	g.DealDamageToPlayer(a, 1, p.ID())
	g.DealDamageToPlayer(b, 1, p.ID())
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("former controller upkeep triggered")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 19 {
		t.Fatal("new controller was damaged or history was cleared")
	}
}

func TestRONDamageHistoryTargets(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g, a, b := ronGame()
		player := `EachPlayer(WasDealtDamageByPermanentThisGame(Ref_TargetPermanent))`
		permanents := `EachPermanent(And([WasDealtDamageByPermanentThisGame(Ref_TargetPermanent),IsCardtype(Planeswalker)]))`
		recipients := player + `,` + permanents
		if reverse {
			recipients = permanents + `,` + player
		}
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),MultipleRecipients([`+recipients+`]))])))`)
		source := g.PutOnBattlefield(mage.NewCreature("History source", "", 1, 3), a.PlayerID())
		empty := g.PutOnBattlefield(mage.NewCreature("Empty source", "", 1, 3), a.PlayerID())
		walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID())
		g.DealDamageToPlayer(a, 1, source.ID())
		g.DealDamageToPlayer(b, 1, source.ID())
		g.DealDamageToPermanent(walker, 1, source.ID())
		for _, p := range []*mage.Permanent{source, empty} {
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
		}
		clone := g.Clone()
		g.ResolveStack()
		if a.Life() != 18 || b.Life() != 18 || g.FindPermanent(walker.ID()).Counters[core.Loyalty] != 8 {
			t.Fatalf("target or factory history binding changed: A=%d B=%d loyalty=%d", a.Life(), b.Life(), walker.Counters[core.Loyalty])
		}
		if clone.GetPlayer(a.PlayerID()).Life() != 19 {
			t.Fatal("pending spell clone changed")
		}
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != 18 {
			t.Fatal("pending spell clone lost binding")
		}
	}
}

func TestRONDamageHistoryContextsAndInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`MultipleRecipients([`+ronHistoryPlayers+`,`+ronHistoryWalkers+`])`), "history.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{permanent: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
		if _, err := compileMixedDamageRecipients(nodes[0], ctx); err == nil {
			t.Fatal("accepted replacement or nonresolving history")
		}
	}
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "The Fallen",`) {
			record = line
		}
	}
	if record == "" || !strings.Contains(record, `EachPlayer(And([Opponent, WasDealtDamageByPermanentThisGame(ThisPermanent)]))`) {
		t.Fatal("missing Oracle source correction")
	}
	files := fstest.MapFS{"DRK.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	one, err := registry.CreateCard("The Fallen")
	if err != nil {
		t.Fatal(err)
	}
	two, err := registry.CreateCard("The Fallen")
	if err != nil {
		t.Fatal(err)
	}
	if one.ID() == two.ID() {
		t.Fatal("factory reused an instance")
	}
	inventory, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Compiled != 1 || len(inventory.Issues) != 0 {
		t.Fatal("inventory did not compile the full record")
	}
	references := 0
	for _, signature := range inventory.Signatures {
		if signature.Role == "permanent_reference" && signature.Name == "ThisPermanent" {
			references += signature.Occurrences
		}
	}
	if references != 3 {
		t.Fatal("inventory lost the history source reference roles")
	}
}
