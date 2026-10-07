package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolutionCounterLivingArtifact(t *testing.T) {
	for _, state := range []string{"pay", "decline", "empty", "return", "phase", "control", "copy"} {
		t.Run(state, func(t *testing.T) {
			a := &counterPaymentClonePayer{Player: mage.NewBasePlayer("A"), accept: state != "decline"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "LEA", "Living Artifact")
			card.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(p.ID(), host.ID())
			g.DealDamageToPlayer(a, 2, b.PlayerID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.Vitality] != 2 {
				t.Fatal("damage did not add counters")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			if state == "empty" {
				g.MutablePermanent(p.ID()).RemoveCounter(core.Vitality, 2)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.Life() != 18 {
				t.Fatal("upkeep timing")
			}
			switch state {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, a.PlayerID())
				g.Attach(p.ID(), host.ID())
				g.MutablePermanent(p.ID()).AddCounter(core.Vitality, 2)
			case "phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 19
			if state == "decline" || state == "empty" || state == "return" || state == "phase" {
				want = 18
			}
			if state == "copy" {
				want = 20
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 18 {
				t.Fatal("clone shared result")
			}
		})
	}
}

func TestRONResolutionCounterChosenBranches(t *testing.T) {
	for _, count := range []int{0, 2} {
		for _, accept := range []bool{false, true} {
			a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: count}, accept: accept}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(`+anyChargeCost+`),If(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay)]),MayCost(`+anyChargeCost+`),Unless(CostWasntPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])]))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			q := g.PutOnBattlefield(factory(), a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 5)
			g.MutablePermanent(q.ID()).AddCounter(core.Charge, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 0
			if accept {
				want = 2 * count
			}
			if a.Life() != 20+want || int(g.FindPermanent(p.ID()).Counters[core.Charge]) != 5-want || g.FindPermanent(q.ID()).Counters[core.Charge] != 5 {
				t.Fatal("payment binding or instance isolation")
			}
			if clone.FindPermanent(p.ID()).Counters[core.Charge] != 5 {
				t.Fatal("clone counters changed")
			}
			if !accept && a.calls != 0 {
				t.Fatal("declined payment chose a count")
			}
		}
	}
}

type ronResolutionCounterPayer struct {
	*ronCounterPayer
	accept bool
}

func (p *ronResolutionCounterPayer) ChooseMayAbility(string) bool { return p.accept }

func TestRONResolutionCounterInvalid(t *testing.T) {
	for _, body := range []string{
		`MayCost(RemoveCounters([])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(RemoveCounters([ACounterOfTypeFromPermanent(ChargeCounter,HostPermanent)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(RemoveCounters([ACounterOfTypeFromPermanent(UnknownCounter,ThisPermanent)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(-1),ChargeCounter,ThisPermanent)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(` + anyChargeCost + `),If(CostWasntPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])`,
		`MayCost(` + anyChargeCost + `),Unless(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])`,
		`MayCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(Integer(1))]),GainLife(TheNumberOfCountersRemovedThisWay)`,
		`GainLife(TheNumberOfCountersRemovedThisWay),MayCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(` + anyChargeCost + `),If(CostWasPaid,[])`,
		`MayCost(` + anyChargeCost + `),If(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay())])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONResolutionCounterTokenCount(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `AsPermanentEnters(ThisPermanent,[EntersWithNumberCounters(Integer(3),PTCounter(+1,+1))]),TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayCost(RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(PTCounter(+1,+1),ThisPermanent)])),If(CostWasPaid,[CreateTokens([NumberTokens(TheNumberOfCountersRemovedThisWay,TokenWithPT(PT(1,1),Colorless,[],[Artifact,Creature],SubtypesList([Tetravite]),[Flying]))],[])])]))`)

	a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.P1P1] != 1 {
		t.Fatal("chosen removal did not pay")
	}
	count := 0
	for _, q := range g.FilterBattlefield(mage.AnyPermanent) {
		if q.HasSubType("Tetravite") {
			count++
			if !q.HasKeyword(core.Flying) || q.CurrentPower(g) != 1 || q.CurrentToughness(g) != 1 {
				t.Fatal("token metadata")
			}
		}
	}
	if count != 2 {
		t.Fatal("paid count did not create tokens")
	}
}

func TestRONResolutionCounterFullRecords(t *testing.T) {
	if _, err := Compile(hostUntapRecord(t, "LEG", "Cocoon")); err != nil {
		t.Fatal(err)
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		card := repositoryCard(t, set, "Living Artifact")
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ set, name, blocker string }{{"ATQ", "Tetravus", "unsupported token keyword (PermanentRuleEffect)"}} {
		n := hostUntapRecord(t, tc.set, tc.name)
		if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), tc.blocker) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestRONResolutionCounterLivingArtifactCasting(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Living Artifact")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 2), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
		t.Fatal("unpaid spell accepted")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{creature.ID()}, 0); err == nil {
		t.Fatal("nonartifact target accepted")
	}
	if a.ManaPool().Count(core.Green) != 1 || len(a.Hand()) != 1 {
		t.Fatal("illegal casting spent resources")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.AttachedTo != host.ID() || a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting or attachment")
	}
	g.DealDamageToPlayer(b, 3, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("opponent damage triggered")
	}
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealDamageToPlayer(a, 3, host.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(p.ID()).Counters[core.Vitality] != 2 {
		t.Fatal("counters used proposed damage")
	}
}

func TestRONResolutionCounterMandatory(t *testing.T) {
	for _, counters := range []int{0, 1, 2} {
		a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}, accept: false}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MustCost(RemoveCounters([ACounterOfTypeFromPermanent(PupaCounter,ThisPermanent)])),Unless(CostWasPaid,[GainLife(Integer(3))])]))`)
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		g.MutablePermanent(p.ID()).AddCounter(core.Pupa, counters)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 20
		if counters == 0 {
			want = 23
		}
		if a.Life() != want || int(g.FindPermanent(p.ID()).Counters[core.Pupa]) != max(0, counters-1) {
			t.Fatal("mandatory payment or unpaid branch")
		}
	}
}

func TestRONResolutionCounterChosenCopiesAndClones(t *testing.T) {
	a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(`+anyChargeCost+`),If(CostWasPaid,[GainLife(TheNumberOfCountersRemovedThisWay)])]))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.MutablePermanent(p.ID()).AddCounter(core.Charge, 6)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.SetPlayerAt(0, &ronResolutionCounterClonePayer{Player: clone.GetPlayer(a.PlayerID()), choice: 1})
	clone.SetPlayerAt(1, &ronResolutionCounterClonePayer{Player: clone.GetPlayer(b.PlayerID()), choice: 3})
	g.SetPlayerAt(1, &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: b, choice: 2}, accept: true})
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 || g.FindPermanent(p.ID()).Counters[core.Charge] != 2 || clone.FindPermanent(p.ID()).Counters[core.Charge] != 6 {
		t.Fatal("copy or clone identity")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 23 || clone.FindPermanent(p.ID()).Counters[core.Charge] != 2 || a.Life() != 22 {
		t.Fatal("clone shared choices or branch count")
	}
}

type ronResolutionCounterClonePayer struct {
	mage.Player
	choice int
}

func (*ronResolutionCounterClonePayer) ChooseMayAbility(string) bool        { return true }
func (p *ronResolutionCounterClonePayer) ChooseNumber(int, int, string) int { return p.choice }
