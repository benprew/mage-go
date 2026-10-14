package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func onlyPlayerSyntheticFactory(t *testing.T, rules string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Synthetic permission",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rules+`])`), "synthetic.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("synthetic", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func onlyPlayerProbe(t *testing.T, set, name, cost, body string) mage.CardFactory {
	t.Helper()
	record := hostUntapRecord(t, set, name)
	nodes, err := Parse(strings.NewReader(`Card(Name:"Permission probe",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),ActivatedWithModifiers(`+cost+`,`+body+`,OtherPlayersMayActivate(AnyPlayer))])`), "probe.ron")
	if err != nil {
		t.Fatal(err)
	}
	probe := nodes[0]
	for _, rule := range record.Fields["Rules"].Args {
		if rule.Name == "ActivatedWithModifiers" {
			probe.Fields["Rules"].Args[1].Args[2] = rule.Args[2]
			factory, err := Compile(probe)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("probe", factory()); err != nil {
				t.Fatal(err)
			}
			return factory
		}
	}
	t.Fatal("missing real activation modifier")
	return nil
}

func TestRONOnlyPlayerActivationTimingTargetsAndInstances(t *testing.T) {
	for _, player := range []string{`OwnerOfPermanent(ThisPermanent)`, `ControllerOfPermanent(ThisPermanent)`, `OwnerOfPermanent(HostPermanent)`, `ControllerOfPermanent(HostPermanent)`} {
		factory := onlyPlayerSyntheticFactory(t, `EnchantPermanent(IsCardtype(Creature)),ActivatedWithModifiers(PayMana([ManaCostG]),Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,1)],UntilEndOfTurn)])),And([OnlyPlayerMayActivate(`+player+`),ActivateOnlyAsASorcery,ActivateNoMoreThanNumberTimesEachTurn(Integer(1))]))`)
		g, a, b := ronGame()
		hostCard := mage.NewCreature("Host", "", 2, 3)
		hostCard.SetOwner(b.PlayerID())
		host := g.PutOnBattlefield(hostCard, b.PlayerID())
		card := factory()
		card.SetOwner(b.PlayerID())
		source := g.PutOnBattlefield(card, b.PlayerID())
		g.Attach(source.ID(), host.ID())
		other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), a.PlayerID())
		b.ManaPool().Add(core.Green, 3)
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{host.ID()}); err == nil || b.ManaPool().TotalMana() != 3 {
			t.Fatal("permission bypassed sorcery timing")
		}
		g.SetActivePlayerIndex(1)
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{other.ID()}); err == nil || b.ManaPool().TotalMana() != 3 {
			t.Fatal("permission bypassed target legality")
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(host.ID()).CurrentPower(g) != 3 {
			t.Fatal("target binding lost")
		}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{host.ID()}); err == nil {
			t.Fatal("permission bypassed activation limit")
		}
		second := factory()
		second.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(second, b.PlayerID())
		g.Attach(p.ID(), host.ID())
		if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
			t.Fatal("factory instances shared state", err)
		}
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
		g.ResolveStack()
		if g.FindPermanent(host.ID()).CurrentPower(g) != 3 {
			t.Fatal("target control not rechecked at resolution")
		}
	}
}

func TestRONOnlyPlayerActivationRollbackAndModes(t *testing.T) {
	a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), invalid: true}
	b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	factory := onlyPlayerProbe(t, "LEA", "Personal Incarnation", `And([PayLife(Integer(2)),DiscardACard])`, `ActionList([GainLife(Integer(3))])`)
	c := factory()
	c.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(c, b.PlayerID())
	payment := mage.NewLand("Payment")
	payment.SetOwner(a.PlayerID())
	a.AddToHand(payment)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.Life() != 20 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment did not roll back")
	}
	a.invalid = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || len(a.Hand()) != 0 || b.Life() != 20 {
		t.Fatal("wrong player paid")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("wrong resolution controller")
	}
	modeA := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: 1}
	modeB := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: 0}
	g = mage.NewGame(modeA, modeB)
	g.SetStep(core.PrecombatMain)
	factory = onlyPlayerProbe(t, "LEA", "Personal Incarnation", `PayMana([])`, `Modal_ChooseOne([ActionList([GainLife(Integer(2))]),ActionList([GainLife(Integer(3))])])`)
	c = factory()
	c.SetOwner(modeA.PlayerID())
	source = g.PutOnBattlefield(c, modeB.PlayerID())
	if err := g.ActivateAbilityByIndex(modeB.PlayerID(), source.ID(), 0, nil); err == nil || modeB.prompts != 0 {
		t.Fatal("ineligible player chose mode")
	}
	if err := g.ActivateAbilityByIndex(modeA.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if modeA.prompts != 1 || g.StackPeek().ModeChoice != 1 {
		t.Fatal("eligible player did not choose mode")
	}
	g.ResolveStack()
	if modeA.Life() != 23 || modeB.Life() != 20 {
		t.Fatal("mode belonged to source controller")
	}
}

func TestRONOnlyPlayerActivationInventory(t *testing.T) {
	for _, player := range []string{`OwnerOfPermanent(ThisPermanent)`, `ControllerOfPermanent(HostPermanent)`} {
		src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),OnlyPlayerMayActivate(` + player + `))])`
		report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
		if err != nil || report.Compiled != 1 {
			t.Fatal("inventory compilation", err)
		}
		wanted := map[string]string{"OnlyPlayerMayActivate": "activation_modifier", strings.Split(player, "(")[0]: "player_reference", strings.TrimRight(strings.Split(player, "(")[1], ")"): "permanent_reference"}
		for _, signature := range report.Signatures {
			if role, ok := wanted[signature.Name]; ok && signature.Role == role {
				delete(wanted, signature.Name)
			}
		}
		if len(wanted) != 0 {
			t.Fatal("missing semantic roles", wanted)
		}
	}
}

func TestRONOnlyPlayerActivationRealModifierProbes(t *testing.T) {
	for _, owner := range []bool{false, true} {
		set, name := "FEM", "Merseine"
		if owner {
			set, name = "LEA", "Personal Incarnation"
		}
		factory := onlyPlayerProbe(t, set, name, `PayMana([ManaCostG])`, `ActionList([GainLife(Integer(2))])`)
		g, a, b := ronGame()
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), a.PlayerID())
		c := factory()
		c.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(c, b.PlayerID())
		g.Attach(source.ID(), host.ID())
		a.ManaPool().Add(core.Green, 1)
		b.ManaPool().Add(core.Green, 1)
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
			t.Fatal("wrong player paid or activated")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.StackPeek().Controller != a.PlayerID() || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong payer or stack controller")
		}
		g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
		clone := g.Clone()
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.BouncePermanentToHand(g.FindPermanent(source.ID()))
		g.ResolveStack()
		if a.Life() != 22 || b.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("permission was rechecked on resolution or clone changed")
		}
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != 22 {
			t.Fatal("clone lost activating player")
		}
		payer := a
		if !owner {
			payer = b
		}
		clone.GetPlayer(payer.PlayerID()).ManaPool().Add(core.Green, 1)
		if err := clone.ActivateAbilityByIndex(payer.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal("permission did not follow live owner/controller", err)
		}
	}
}

func TestRONOnlyPlayerActivationPresenceAndReattachment(t *testing.T) {
	factory := onlyPlayerProbe(t, "FEM", "Merseine", `PayMana([])`, `ActionList([GainLife(Integer(1))])`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 3), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unattached source granted permission")
	}
	g.Attach(source.ID(), first.ID())
	for _, phaseHost := range []bool{false, true} {
		phaseID := source.ID()
		if phaseHost {
			phaseID = first.ID()
		}
		phased := g.PhaseOut(phaseID)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
			t.Fatal("phased object granted permission")
		}
		g.PhaseIn(phased)
	}
	g.Attach(source.ID(), second.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("old host controller retained permission")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatal("reattached ability used wrong player")
	}
	c := second.Card
	c.SetOwner(b.PlayerID())
	g.BouncePermanentToHand(g.FindPermanent(second.ID()))
	b.RemoveFromHand(c.ID())
	g.PutOnBattlefield(c, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("returned host restored stale attachment")
	}
}

func TestRONOnlyPlayerActivationInvalidAndBoundaries(t *testing.T) {
	valid := `OnlyPlayerMayActivate(OwnerOfPermanent(ThisPermanent))`
	for _, modifier := range []string{
		`OnlyPlayerMayActivate`, `OnlyPlayerMayActivate()`, `OnlyPlayerMayActivate(You)`,
		`OnlyPlayerMayActivate(OwnerOfPermanent)`, `OnlyPlayerMayActivate(ControllerOfPermanent())`,
		`OnlyPlayerMayActivate(ControllerOfPermanent(ThisPermanent,ThisPermanent))`,
		`OnlyPlayerMayActivate(OwnerOfPermanent(HostPermanent))`,
		`OnlyPlayerMayActivate(OwnerOfPermanent(ThisPermanent),You)`,
		`OnlyPlayerMayActivate(OwnerOfPermanent())`, `OnlyPlayerMayActivate(OwnerOfPermanent(ThisPermanent()))`,
		`OnlyPlayerMayActivate(ControllerOfPermanent(Ref_TargetPermanent))`,
		`OnlyPlayerMayActivate(ControllerOfPermanent(Trigger_ThatPermanent))`,
		`And([` + valid + `,` + valid + `])`, `And([OtherPlayersMayActivate(AnyPlayer),` + valid + `])`,
		`And([` + valid + `,And([OtherPlayersMayActivate(AnyPlayer)])])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),` + modifier + `)])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid permission accepted", modifier)
		}
	}
	for _, rule := range []string{
		valid,
		`Activated(` + valid + `,ActionList([GainLife(Integer(1))]))`,
		`Activated(PayMana([]),ActionList([` + valid + `]))`,
		`SpellActions(ActionList([` + valid + `]))`,
		`ActivatedWithModifiers(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,ControlsA(IsCardtype(Land))),[AddMana(ManaProduceG)],[AddMana(ManaProduceG)])]),` + valid + `)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("permission accepted outside supported context", rule)
		}
	}
	for _, tc := range []struct{ set, name, gap string }{{"LEA", "Personal Incarnation", "unsupported selected-source redirection event"}} {
		if _, err := Compile(hostUntapRecord(t, tc.set, tc.name)); err == nil || !strings.Contains(err.Error(), tc.gap) {
			t.Fatalf("%s lost independent gap: %v", tc.name, err)
		}
	}
	src := `Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),` + valid + `)])`
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}}
	r := mage.NewRegistry()
	if err := LoadInto(r, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, `Name:"Valid"`, `Name:"Bad"`, 1))}
	files["bad.ron"].Data = []byte(strings.Replace(string(files["bad.ron"].Data), valid, `OnlyPlayerMayActivate(You)`, 1))
	if err := LoadInto(r, files, "*.ron"); err == nil || !r.CardRegistered("Valid") || r.CardRegistered("Bad") {
		t.Fatal("invalid collection changed registry")
	}
}
