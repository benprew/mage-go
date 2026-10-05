package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONModalActivationBalm(t *testing.T) {
	for _, scenario := range []string{"life", "player", "permanent", "phase-cycle", "unpaid", "tapped", "wrong-controller", "invalid-mode", "illegal-target", "shroud", "phase", "return", "copy", "cleanup"} {
		t.Run(scenario, func(t *testing.T) {
			mode := 1
			if scenario == "life" {
				mode = 0
			}
			if scenario == "invalid-mode" {
				mode = 9
			}
			g, a, b := ronModalGame(mode)
			card := repositoryCard(t, "FEM", "Balm of Restoration")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 2)
			g.SetActivePlayerIndex(0)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("artifact entered before resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			victimCard := mage.NewCreature("Recipient", "", 1, 20)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			a.targets = []uuid.UUID{b.PlayerID()}
			if scenario == "permanent" || scenario == "phase-cycle" || scenario == "shroud" || scenario == "phase" || scenario == "return" {
				a.targets = []uuid.UUID{victim.ID()}
			}
			if scenario == "illegal-target" {
				a.targets = []uuid.UUID{uuid.New()}
			}
			if scenario == "shroud" {
				victim.GrantBaseAttr(core.Shroud)
			}
			if scenario == "tapped" {
				g.TapPermanent(source)
			}
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			payer := a.PlayerID()
			if scenario == "wrong-controller" {
				payer = b.PlayerID()
			}
			before := a.ManaPool().TotalMana()
			err := g.ActivateAbilityByIndex(payer, source.ID(), 0, nil)
			invalid := scenario == "unpaid" || scenario == "tapped" || scenario == "wrong-controller" || scenario == "invalid-mode" || scenario == "illegal-target" || scenario == "shroud"
			if invalid {
				if err == nil {
					t.Fatal("illegal activation accepted")
				}
				if g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Tapped != (scenario == "tapped") || len(a.Graveyard()) != 0 || a.ManaPool().TotalMana() != before || g.StackSize() != 0 {
					t.Fatal("failed activation changed resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()) != nil || len(a.Graveyard()) != 1 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 || a.Life() != 20 {
				t.Fatal("cost or resolution timing")
			}
			obj := g.StackPeek()
			if obj.ModeChoice != mode || len(obj.ModalTargets) != 2 || len(obj.ModalTargets[1-mode]) != 0 || len(obj.Targets) != mode {
				t.Fatal("mode target selection")
			}
			clone := g.Clone()
			if scenario == "phase" {
				g.PhaseOut(victim.ID())
			}
			if scenario == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(victim.ID()))
			}
			if scenario == "return" {
				g.BouncePermanentToHand(g.FindPermanent(victim.ID()))
				b.RemoveFromHand(victimCard.ID())
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			}
			if scenario == "copy" {
				a.targets = []uuid.UUID{a.PlayerID()}
				if copied := g.CopyStackObjectDirect(obj, a.PlayerID(), true); copied == nil || copied.ModeChoice != 1 || len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID() {
					t.Fatal("copy mode or retarget")
				}
			}
			prompts := a.prompts
			g.DealDamageToPlayer(b, 1, uuid.New())
			if b.Life() != 19 {
				t.Fatal("prevention started before resolution")
			}
			g.ResolveStack()
			if a.prompts != prompts {
				t.Fatal("resolution chose mode again")
			}
			if scenario == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
			}
			if mode == 0 {
				if a.Life() != 22 {
					t.Fatal("life mode did not gain two")
				}
				g.DealDamageToPlayer(b, 3, uuid.New())
				if b.Life() != 16 {
					t.Fatal("unchosen mode installed a shield")
				}
			} else if scenario == "permanent" || scenario == "phase-cycle" || scenario == "return" {
				g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, uuid.New())
				want := 1
				if scenario == "return" {
					want = 3
				}
				if g.FindPermanent(victim.ID()).Damage != want {
					t.Fatal("permanent prevention or incarnation")
				}
			} else if scenario == "phase" {
				g.PhaseIn([]uuid.UUID{victim.ID()})
				g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, uuid.New())
				if g.FindPermanent(victim.ID()).Damage != 3 {
					t.Fatal("illegal phased target received shield")
				}
			} else {
				g.DealDamageToPlayer(b, 3, uuid.New())
				want := 18
				if scenario == "cleanup" {
					want = 16
				}
				if b.Life() != want {
					t.Fatal("shield amount or expiry")
				}
				g.DealDamageToPlayer(b, 1, uuid.New())
				if b.Life() != want-1 {
					t.Fatal("shield was not consumed")
				}
				if scenario == "copy" {
					g.DealDamageToPlayer(a, 3, uuid.New())
					if a.Life() != 19 {
						t.Fatal("copy changed original recipient")
					}
				}
			}
			if clone.StackSize() != 1 || clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.FindPermanent(victimCard.ID()).Damage != 0 {
				t.Fatal("live resolution changed pending clone")
			}
			clone.ResolveStack()
			if mode == 0 && clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("clone lost life mode")
			}
			if mode == 1 {
				clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 3, uuid.New())
				want := 19
				if scenario == "permanent" || scenario == "phase-cycle" || scenario == "phase" || scenario == "return" {
					want = 17
				}
				if clone.GetPlayer(b.PlayerID()).Life() != want {
					t.Fatal("clone lost recipient")
				}
			}
		})
	}
}

func TestRONModalActivationDeclarations(t *testing.T) {
	good := `Targeted([TargetPermanent(AnyPermanent)],ActionList([TapPermanent(Ref_TargetPermanent)]))`
	for _, body := range []string{`Modal_ChooseOne`, `Modal_ChooseOne()`, `Modal_ChooseOne([])`, `Modal_ChooseOne([` + good + `])`, `Modal_ChooseOne([` + good + `,ActionList([TapPermanent(Ref_TargetPermanent)])])`, `Modal_ChooseOne([` + good + `,Targeted([TargetPlayer(AnyPlayer)],ActionList([TapPermanent(Ref_TargetPlayer)]))])`, `Modal_ChooseOne([` + good + `,Unknown([])])`, `Modal_ChooseOne([` + good + `,Modal_ChooseOne([` + good + `,` + good + `])])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),`+body+`)])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, name := range []string{"Activated", "ActivatedWithModifiers"} {
		t.Run(name, func(t *testing.T) {
			modifier := ""
			if name == "ActivatedWithModifiers" {
				modifier = ",ActivateOnlyOnceEachTurn"
			}
			factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`%s(PayMana([]),Modal_ChooseOne([%s,%s])%s)`, name, good, good, modifier))
			first, second := factory(), factory()
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
			}
			modes := first.Abilities()[0].(*mage.ActionDefinition).Modes()
			other := second.Abilities()[0].(*mage.ActionDefinition).Modes()
			if modes[0].Targets[0] == modes[1].Targets[0] || modes[0].Targets[0] == other[0].Targets[0] {
				t.Fatal("shared declarations")
			}
			for _, mode := range modes {
				if strings.Contains(mode.Label, "Ref_") || strings.Contains(mode.Label, "mode1/") {
					t.Fatal("binding name in label")
				}
			}
			g, a, b := ronModalGame(1)
			p := g.PutOnBattlefield(first, a.PlayerID())
			q := g.PutOnBattlefield(second, a.PlayerID())
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 3), b.PlayerID())
			a.targets = []uuid.UUID{target.ID()}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !g.FindPermanent(target.ID()).Tapped {
				t.Fatal("second mode lost binding")
			}
			if name == "ActivatedWithModifiers" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
					t.Fatal("modal limit ignored")
				}
			}
		})
	}
}

func TestRONModalActivationPaymentBindings(t *testing.T) {
	for _, cost := range []string{`PayManaX([ManaCostX],ValueX)`, `SacrificeAPermanent(IsCardtype(Creature))`, `And([])`} {
		t.Run(cost, func(t *testing.T) {
			g, a, b := ronModalGame(1)
			value := `ValueX`
			if strings.HasPrefix(cost, "Sacrifice") {
				value = `ToughnessOfTheSacrificedCreature`
			}
			if cost == `And([])` {
				value = `Integer(3)`
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(`+cost+`,Modal_ChooseOne([ActionList([GainLife(Integer(1))]),ActionList([GainLife(`+value+`)])]))`)
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			paymentCard := mage.NewCreature("Payment", "", 2, 3)
			paymentCard.SetOwner(a.PlayerID())
			g.PutOnBattlefield(paymentCard, a.PlayerID())
			g.SetXValue(3)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(cost, "Sacrifice") {
				if len(a.Graveyard()) != 1 {
					t.Fatal("creature was not sacrificed")
				}
				a.RemoveFromGraveyard(paymentCard.ID())
				returned := g.PutOnBattlefield(paymentCard, b.PlayerID())
				returned.AddCounter(core.P1P1, 8)
			}
			g.SetXValue(99)
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if a.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 23 || b.Life() != 20 {
				t.Fatal("mode lost captured payment or controller")
			}
		})
	}
}

func TestRONModalActivationBalmSimultaneous(t *testing.T) {
	g, a, b := ronModalGame(1)
	card := repositoryCard(t, "FEM", "Balm of Restoration")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 20), b.PlayerID())
	a.targets = []uuid.UUID{victim.ID()}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	g.PhaseIn(g.PhaseOut(victim.ID()))
	clone := g.Clone()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 3, 10), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 10), b.PlayerID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(victim.ID()), 3, g.CaptureDamageSource(first.ID())), mage.PermanentDamageAssignment(g.FindPermanent(victim.ID()), 3, g.CaptureDamageSource(second.ID())))
	if g.FindPermanent(victim.ID()).Damage != 4 || clone.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("simultaneous shield amount, phase identity or clone isolation")
	}
	clone.DealDamageToPermanent(clone.FindPermanent(victim.ID()), 3, uuid.New())
	if clone.FindPermanent(victim.ID()).Damage != 1 {
		t.Fatal("live shield consumption changed clone")
	}
}

func TestRONModalActivationStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "FEM.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		if node.Fields["Name"].String == "Balm of Restoration" {
			files["balm.ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
		}
	}
	if len(files) != 1 {
		t.Fatal("missing complete Balm record")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatalf("inventory: %v", err)
	}
	seen := false
	for _, entry := range report.Signatures {
		if entry.Name == "Modal_ChooseOne" {
			seen = true
			if entry.Role != "actions" || entry.CompiledOccurrences != 1 {
				t.Fatal("modal role")
			}
		}
	}
	if !seen {
		t.Fatal("missing modal inventory")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Modal_ChooseOne([ActionList([GainLife(Integer(1))]),ActionList([DestroyPermanent(Ref_TargetPermanent)])]))])`)}
	if registry, err = LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry accepted")
	}
}

type ronModalCounterPlayer struct {
	*ronModePlayer
	count int
}

func (p *ronModalCounterPlayer) ChooseNumber(_, _ int, _ string) int { return p.count }

func TestRONModalActivationCounterBindings(t *testing.T) {
	a := &ronModalCounterPlayer{ronModePlayer: &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: 1}, count: 2}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	factory := permanentScopeFactory(t, "Artifact", `Activated(`+anyChargeCost+`,Modal_ChooseOne([ActionList([GainLife(Integer(1))]),ActionList([GainLife(TheNumberOfCountersRemovedThisWay)])]))`)
	first, second := factory(), factory()
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, a.PlayerID())
	p.AddCounter(core.Charge, 2)
	q.AddCounter(core.Charge, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.count = 3
	if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).Counters[core.Charge] != 0 || g.FindPermanent(q.ID()).Counters[core.Charge] != 0 {
		t.Fatal("counter costs not paid")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 25 || clone.GetPlayer(a.PlayerID()).Life() != 25 {
		t.Fatal("pending modes shared payment values")
	}
}
