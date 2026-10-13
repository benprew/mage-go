package ron

import (
	"maps"

	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEligibleAlternativePaymentWorms(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Worms of the Earth")
	rules, _ := list(record.Fields["Rules"])
	for _, state := range []string{"damage", "prevent", "sacrifice", "decline"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronAlternativePlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state != "decline"}, mode: 1}
			g := mage.NewGame(a, b)
			body := rules[2]
			if state == "sacrifice" {
				b.mode = 0
			}
			isolated := record
			isolated.Fields = map[string]Node{}
			maps.Copy(isolated.Fields, record.Fields)
			isolated.Fields["Rules"] = Node{Kind: List, Args: []Node{body}}
			factory, err := Compile(isolated)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for range 2 {
				g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			}
			if state == "prevent" {
				g.AddPreventionShield(b.PlayerID(), 5)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.ResolveStack()
			want := 20
			if state == "damage" {
				want = 15
			}
			if b.Life() != want || (g.FindPermanent(source.ID()) != nil) != (state == "decline") {
				t.Fatal("wrong accepted payment or destruction", b.Life())
			}
			if state == "sacrifice" && len(g.FilterBattlefield(mage.ControlledBy(b.PlayerID()))) != 0 {
				t.Fatal("lands not sacrificed")
			}
		})
	}
	if _, err := Compile(record); err != nil {
		t.Fatal(err)
	}
}

func TestRONEligibleAlternativePaymentFrames(t *testing.T) {
	for _, state := range []string{"live", "left", "return", "phase", "control", "copy", "clone", "delay"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			body := `AnyPlayerMayCost(Opponent,HavePermanentDealDamage(ThisPermanent,Integer(5),Player(ActionPlayer))),IfElse(CostWasPaid,[PlayerAction(ActionPlayer,GainLife(Integer(1)))],[GainLife(Integer(10))])`
			if state == "delay" {
				body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))`
			}
			factory := permanentScopeFactory(t, "Artifact", `Lifelink,Activated(PayMana([]),ActionList([`+body+`]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "delay" {
				g.ResolveStack()
			}
			switch state {
			case "left", "return", "delay":
				g.RemoveFromBattlefield(source)
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			clone := g.Clone()
			clone.SetPlayerAt(1, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: true})
			for _, game := range []*mage.Game{g, clone} {
				if state == "delay" {
					for range 3 {
						game.WaitingDelayedPermanentSources()
					}
					if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("waiting query made a payment")
					}
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				}
				game.ResolveStack()
				wantA, wantB := 25, 16
				if state == "control" {
					wantA, wantB = 20, 21
				}
				if state == "copy" {
					wantA, wantB = 30, 12
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life %d/%d want %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
			}
		})
	}
}

func TestRONEligibleAlternativePaymentInvalid(t *testing.T) {
	for _, cost := range []string{`Or([])`, `Or([PayMana([])])`, `Or([PayMana([]),PayLife(Integer(1))])`, `Or([PayMana([]),SacrificeNumberPermanents(Integer(0),IsCardtype(Land))])`, `SacrificeNumberPermanents(ValueX,IsCardtype(Land))`, `HavePermanentDealDamage(ThisPermanent,Integer(5),Player(Unbound))`, `HavePermanentDealDamage(ThisPermanent,Integer(5))`, `HavePermanentDealDamage(ThisPermanent,Integer(5),Player(ActionPlayer()))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([AnyPlayerMayCost(AnyPlayer,` + cost + `),If(CostWasPaid,[GainLife(Integer(1))])]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid payment accepted", cost)
		}
	}
	for _, continuation := range []string{`If(CostWasntPaid,[PlayerAction(ActionPlayer,GainLife(Integer(1)))])`, `If(CostWasPaid,[GainLife(Integer(1))]),PlayerAction(ActionPlayer,GainLife(Integer(1)))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([AnyPlayerMayCost(AnyPlayer,HavePermanentDealDamage(ThisPermanent,Integer(1),Player(ActionPlayer))),` + continuation + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("payer escaped", continuation)
		}
	}
	valid := `Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([AnyPlayerMayCost(AnyPlayer,Or([PayMana([]),Or([SacrificeAPermanent(IsCardtype(Land)),HavePermanentDealDamage(ThisPermanent,Integer(1),Player(ActionPlayer))])])),If(CostWasPaid,[GainLife(Integer(1))])]))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(valid)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["blocked.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Blocked",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[CardsCantEnterTheBattlefield(IsCardtype(Creature))])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}
