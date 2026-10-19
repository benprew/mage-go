package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const linkedExileCost = `Exile([AnyNumberOfPermanents(TheCreatedTokens)],[])`
const linkedExileResult = `NumPermanentsExiledThisWay`

func TestRONLinkedExilePaymentTetravus(t *testing.T) {
	for _, exileFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(exileFirst), func(t *testing.T) {
			factory, err := Compile(hostUntapRecord(t, "ATQ", "Tetravus"))
			if err != nil {
				t.Fatal(err)
			}
			a := &linkedExileOrderPayer{ronResolutionCounterPayer: &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: true}, exileFirst: exileFirst}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Colorless, 6)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p.Counters[core.P1P1] != 3 || !p.HasKeyword(core.Flying) {
				t.Fatal("entry characteristics")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("upkeep abilities were combined")
			}
			clone := g.Clone()
			g.ResolveStack()
			wantCounters, wantTokens := 3, 0
			if exileFirst {
				wantCounters, wantTokens = 1, 2
			}
			if g.FindPermanent(card.ID()).Counters[core.P1P1] != wantCounters || len(g.FilterBattlefield(mage.IsToken)) != wantTokens {
				t.Fatalf("first upkeep: counters=%d tokens=%d", g.FindPermanent(card.ID()).Counters[core.P1P1], len(g.FilterBattlefield(mage.IsToken)))
			}
			if len(clone.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("pending clone shared tokens")
			}
			a.accept = false
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if len(g.FilterBattlefield(mage.IsToken)) != wantTokens {
				t.Fatal("declined exile moved tokens")
			}
			a.accept = true
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(card.ID()).Counters[core.P1P1] != wantCounters || len(g.FilterBattlefield(mage.IsToken)) != wantTokens {
				t.Fatal("exile count or repeated creation")
			}
		})
	}
}

func TestRONLinkedExilePaymentInvalid(t *testing.T) {
	producer := `Activated(PayMana([]),ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Tetravite]),[])],[])]))`
	valid := `MayCost(` + linkedExileCost + `),If(CostWasPaid,[GainLife(` + linkedExileResult + `)])`
	for _, body := range []string{
		strings.Replace(valid, "AnyNumberOfPermanents(TheCreatedTokens)", "AnyNumberOfPermanents(TheCreatedTokens())", 1),
		strings.Replace(valid, "TheCreatedTokens", "AnyPermanent", 1),
		strings.Replace(valid, ",[])", ",[Unknown])", 1),
		strings.Replace(valid, "CostWasPaid", "CostWasntPaid", 1),
		strings.Replace(valid, linkedExileResult, linkedExileResult+"()", 1),
		strings.Replace(valid, "If(CostWasPaid", "If(CostWasPaid()", 1),
		`MayCost(` + linkedExileCost + `),If(CostWasPaid,[GainLife(Integer(1))]),GainLife(` + linkedExileResult + `)`,
		`GainLife(` + linkedExileResult + `),` + valid,
		`MustCost(` + linkedExileCost + `),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(` + linkedExileCost + `),If(CostWasPaid,[])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+producer+`,Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func linkedExileFactory(t *testing.T, body string) func() mage.Card {
	t.Helper()
	return permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateTokens([NumberTokens(Integer(3),TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Tetravite]),[]))],[])])),Activated(PayMana([]),ActionList([`+body+`]))`)
}

func TestRONLinkedExilePaymentGameplay(t *testing.T) {
	for _, state := range []string{"paid", "declined", "zero", "control", "phase", "phase-cycle", "orphan", "return", "copy", "delay"} {
		t.Run(state, func(t *testing.T) {
			a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: state != "declined"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			action := `GainLife(` + linkedExileResult + `)`
			if state == "delay" {
				action = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + action + `]))`
			}
			factory := linkedExileFactory(t, `MayCost(`+linkedExileCost+`),If(CostWasPaid,[`+action+`])`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(factory(), a.PlayerID())
			for _, p := range []*mage.Permanent{source, other} {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != 6 {
				t.Fatal("creation")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.FilterBattlefield(mage.IsToken)) != 6 {
				t.Fatal("paid before resolution")
			}
			switch state {
			case "zero":
				a.choice = 0
			case "control":
				for _, p := range tokens[:3] {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
			case "phase":
				g.PhaseOut(tokens[0].ID())
				g.PhaseOut(tokens[1].ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(tokens[0].ID()))
			case "orphan", "return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if state == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.SetPlayerAt(0, &ronResolutionCounterClonePayer{Player: clone.GetPlayer(a.PlayerID()), choice: 1})
			g.ResolveStack()
			wantLife, wantTokens := 22, 4
			switch state {
			case "declined", "zero":
				wantLife, wantTokens = 20, 6
			case "phase":
				wantLife, wantTokens = 21, 3
			case "copy":
				wantLife, wantTokens = 23, 3
			case "delay":
				wantLife = 20
			}
			if a.Life() != wantLife || len(g.FilterBattlefield(mage.IsToken)) != wantTokens {
				t.Fatalf("life=%d tokens=%d", a.Life(), len(g.FilterBattlefield(mage.IsToken)))
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared result")
			}
			clone.ResolveStack()
			wantCloneLife := 21
			if state == "copy" {
				wantCloneLife = 22
			}
			if state == "delay" {
				wantCloneLife = 20
			}
			if clone.GetPlayer(a.PlayerID()).Life() != wantCloneLife {
				t.Fatal("clone lost its separate payment choice")
			}
			if a.Life() != wantLife {
				t.Fatal("clone changed original")
			}
			if state == "delay" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
				if a.Life() != 22 {
					t.Fatal("delayed result lost")
				}
			}
			for _, p := range tokens[3:] {
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("other instance paid")
				}
			}
		})
	}
}

func TestRONLinkedExilePaymentScopes(t *testing.T) {
	body := `MayCost(` + linkedExileCost + `),If(CostWasPaid,[GainLife(` + linkedExileResult + `),MayCost(` + linkedExileCost + `),If(CostWasPaid,[GainLife(` + linkedExileResult + `)]),GainLife(` + linkedExileResult + `)])`
	factory := linkedExileFactory(t, body)
	a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	for _, index := range []int{0, 1} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), index, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if a.Life() != 25 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("nested results overwrote the outer payment")
	}
}

type linkedExileOrderPayer struct {
	*ronResolutionCounterPayer
	exileFirst bool
}

func (p *linkedExileOrderPayer) ChooseTriggerOrder([]mage.TriggerOrderInfo) []int {
	if p.exileFirst {
		return []int{0, 1}
	}
	return []int{1, 0}
}

func TestRONLinkedExilePaymentInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Tetravite]),[])],[])])),Activated(PayMana([]),ActionList([MayCost(` + linkedExileCost + `),If(CostWasPaid,[GainLife(` + linkedExileResult + `)])]))])`
	report, err := InspectFS(fstest.MapFS{"test.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"Exile": "resolution_cost", "AnyNumberOfPermanents": "exile_payment_selection", "TheCreatedTokens": "permanent_collection_reference", "NumPermanentsExiledThisWay": "value"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s as %s", name, role)
		}
	}
}
