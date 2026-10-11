package ron

import (
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpecificTapPaymentEnergyTap(t *testing.T) {
	for _, state := range []string{"normal", "tapped", "phased", "phase-cycle", "returned", "control", "shroud", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			card := repositoryCard(t, "LEG", "Energy Tap")
			g, a, b := ronGame()
			victim := mage.NewCreature("Subject", "{3}{U}", 2, 3)
			victim.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(victim, a.PlayerID())
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("tapped at announcement")
			}
			switch state {
			case "tapped":
				g.TapPermanent(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(victim.ID())
				g.PutOnBattlefield(victim, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 0
			if state == "normal" || state == "phase-cycle" || state == "copy" || state == "clone" {
				want = 4
			}
			if a.ManaPool().TotalMana() != want {
				t.Fatalf("mana %d want %d", a.ManaPool().TotalMana(), want)
			}
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != want {
				t.Fatal("clone resolution differs")
			}
			if state == "returned" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("returned subject paid")
			}
		})
	}
}

func TestRONSpecificTapPaymentTourachsGate(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Tourach's Gate")
	card.SetOwner(a.PlayerID())
	host := g.PutOnBattlefield(mage.NewLand("Host"), a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.AttachedTo != host.ID() {
		t.Fatal("Aura did not enter attached")
	}
	thrullCard := mage.NewCreature("Thrull", "", 1, 1, mage.WithSubTypes("Thrull"))
	thrullCard.SetOwner(a.PlayerID())
	thrull := g.PutOnBattlefield(thrullCard, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(thrull.ID()) != nil || g.FindPermanent(source.ID()).Counters[core.Time] != 3 {
		t.Fatal("counter payment failed")
	}
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 4), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 4), b.PlayerID())
	g.MutablePermanent(attacker.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.SetStep(core.DeclareAttackers)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	if !g.IsAttackingInCombat(attacker.ID()) {
		t.Fatal("attack failed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).Tapped {
		t.Fatal("host did not pay")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, nil); err == nil {
		t.Fatal("tapped host paid twice")
	}
	g.ResolveStack()
	if p := g.FindPermanent(attacker.ID()); p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 3 {
		t.Fatal("attacker boost failed")
	}
	if p := g.FindPermanent(other.ID()); p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 4 {
		t.Fatal("wrong collection member")
	}
	g.RunStepWithPriority(core.Cleanup)
	if p := g.FindPermanent(attacker.ID()); p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 4 {
		t.Fatal("boost survived cleanup")
	}
	for i := 2; i >= 0; i-- {
		g.RunStepWithPriority(core.Upkeep)
		g.ResolveStack()
		source = g.FindPermanent(card.ID())
		if i == 0 && source != nil {
			t.Fatal("Aura survived zero counters")
		}
		if i > 0 && (source == nil || int(source.Counters[core.Time]) != i) {
			t.Fatal("upkeep counter failed")
		}
	}
}

func TestRONSpecificTapPaymentPolarityAndLocalResults(t *testing.T) {
	for _, clause := range []string{"If(CostWasPaid", "Unless(CostWasntPaid", "If(CostWasntPaid", "Unless(CostWasPaid"} {
		t.Run(clause, func(t *testing.T) {
			card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MustCost(TapPermanent(Ref_TargetPermanent)),`+clause+`,[MustCost(TapPermanent(Ref_TargetPermanent)),Unless(CostWasPaid,[GainLife(Integer(3))]),GainLife(Integer(2))]) ])))`)()
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 1), b.PlayerID())
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 20
			if clause == "If(CostWasPaid" || clause == "Unless(CostWasntPaid" {
				want = 25
			}
			if a.Life() != want || !g.FindPermanent(p.ID()).Tapped {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
		})
	}
}

func TestRONSpecificTapPaymentInvalid(t *testing.T) {
	for _, body := range []string{
		`MustCost(TapPermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(HostPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPlayer)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Trigger_ThatPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent())),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent()),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent,Ref_TargetPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent))`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid,[])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`,
		`MayCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MustCost(TapPermanent(Ref_TargetPermanent)),If(CostWasPaid,[GainLife(Integer(1))]),If(CostWasPaid,[GainLife(Integer(2))])`,
		`AddManaRepeated(AnyManaColor,ManaValueOfPermanent(Ref_TargetPermanent))`,
		`AddManaRepeated(And([ManaProduceC,ManaProduceU]),ManaValueOfPermanent(Ref_TargetPermanent))`,
		`AddManaRepeated(ManaProduceC,ManaValueOfPermanent(Ref_TargetPlayer))`,
		`AddManaRepeated(ManaProduceC,ManaValueOfPermanent(Ref_TargetPermanent()))`,
	} {
		node := nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+body+`])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, rule := range []string{
		`Activated(TapPermanent(HostPermanent),ActionList([GainLife(Integer(1))]))`,
		`Activated(TapPermanent(HostPermanent()),ActionList([GainLife(Integer(1))]))`,
		`Activated(TapPermanent(Ref_TargetPermanent),ActionList([GainLife(Integer(1))]))`,
		`Activated(TapPermanent(HostPermanent,HostPermanent),ActionList([GainLife(Integer(1))]))`,
	} {
		node := nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONSpecificTapPaymentHostLifetime(t *testing.T) {
	for _, state := range []string{"normal", "tapped", "phased", "phase-cycle", "reattached", "departed", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Tourach's Gate")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 2, 4)
			hostCard.AddType(core.TypeLand)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewLand("Other"), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 5), a.PlayerID())
			enemy := g.PutOnBattlefield(mage.NewCreature("Enemy", "", 3, 5), b.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
			g.GetCombat().AddAttacker(enemy.ID(), a.PlayerID())
			switch state {
			case "tapped":
				g.TapPermanent(host)
			case "phased":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "reattached":
				g.TapPermanent(host)
				g.Attach(source.ID(), other.ID())
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, nil)
			if state == "tapped" || state == "phased" {
				if err == nil || g.StackSize() != 0 {
					t.Fatal("unavailable host paid")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			paidHost := host.ID()
			if state == "reattached" {
				paidHost = other.ID()
			}
			if !g.FindPermanent(paidHost).Tapped || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("wrong tap payment")
			}
			if state == "departed" {
				g.BouncePermanentToHand(source)
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			if p := g.FindPermanent(attacker.ID()); p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 4 {
				t.Fatal("lost saved controller")
			}
			want := 3
			if state == "copy" {
				want = 5
			}
			if g.FindPermanent(enemy.ID()).CurrentPower(g) != want {
				t.Fatal("copy controller selection failed")
			}
			if clone.FindPermanent(attacker.ID()).CurrentPower(clone) != 3 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.FindPermanent(attacker.ID()).CurrentPower(clone) != 5 {
				t.Fatal("clone lost payment body")
			}
		})
	}
}

func TestRONSpecificTapPaymentNativeAndInventory(t *testing.T) {
	files := fstest.MapFS{
		"energy.ron":  {Data: damageReceivedRecord(t, "LEG", "Energy Tap")},
		"gate.ron":    {Data: damageReceivedRecord(t, "FEM", "Tourach's Gate")},
		"reprint.ron": {Data: damageReceivedRecord(t, "LEG", "Energy Tap")},
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 3 || report.UniqueCards != 2 || len(report.Issues) != 0 {
		t.Fatal("wrong inventory", report.Issues)
	}
	roles := map[string]bool{}
	for _, signature := range report.Signatures {
		if signature.Name == "TapPermanent" {
			roles[signature.Role] = true
		}
		if signature.Role == "unknown" {
			t.Fatal("unknown nested role", signature.Name)
		}
	}
	if !roles["cost"] {
		t.Fatal("tap payment not classified as a cost")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Energy Tap")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Energy Tap")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("factories share declarations")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	games := make([]*mage.Game, 2)
	players := make([]*mage.BasePlayer, 2)
	for i, card := range []mage.Card{first, second} {
		g, a, _ := ronGame()
		games[i], players[i] = g, a
		victim := g.PutOnBattlefield(mage.NewCreature("Subject", "{2}", 1, 1), a.PlayerID())
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	for i, g := range games {
		g.ResolveStack()
		if players[i].ManaPool().TotalMana() != 2 {
			t.Fatal("instances share target state")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(HostPermanent),ActionList([GainLife(Integer(1))]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("invalid collection replaced registry")
	}
}

func TestRONSpecificTapPaymentContexts(t *testing.T) {
	for _, state := range []string{"live", "returned", "phased", "phase-cycle", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MustCost(TapPermanent(ThisPermanent)),If(CostWasPaid,[GainLife(Integer(2))]),GainLife(Integer(1))]))]))`)()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("missing waiting reference")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			want := 21
			if state == "live" || state == "phase-cycle" {
				want = 23
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if state == "copy" && b.Life() != 23 {
				t.Fatal("copy did not use its controller")
			}
		})
	}
}
