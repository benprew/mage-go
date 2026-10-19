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

func TestRONActivationPermissionEfreet(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "ARN", "Ifh-Bíff Efreet")
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.Green, 2)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	if p == nil || !p.HasKeyword(core.Flying) || p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 || !p.HasSubType("Efreet") || a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong complete-record characteristics or cost")
	}
	ground := g.PutOnBattlefield(mage.NewCreature("Ground", "", 1, 10), a.PlayerID())
	flying := g.PutOnBattlefield(mage.NewCreature("Flying", "", 1, 10, mage.WithKeyword(core.Flying), mage.WithKeyword(core.Shroud)), b.PlayerID())
	phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 10, mage.WithKeyword(core.Flying)), b.PlayerID())
	g.PhaseOut(phased.ID())
	a.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("opponent spent source controller's mana")
	}
	b.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackPeek().Controller != b.PlayerID() || b.ManaPool().TotalMana() != 0 || a.ManaPool().TotalMana() != 1 || a.Life() != 20 || p.Damage != 0 || p.Tapped {
		t.Fatal("wrong activator, payment or timing")
	}
	callbacks := 0
	g.SetOnDamageDealt(func(source, _ string, amount int, combat bool) {
		callbacks++
		if source != c.Name() || amount != 1 || combat || a.Life() != 19 || b.Life() != 19 || g.FindPermanent(p.ID()).Damage != 1 || g.FindPermanent(flying.ID()).Damage != 1 {
			t.Error("incomplete simultaneous damage batch")
		}
	})
	g.ResolveStack()
	if callbacks != 4 || ground.Damage != 0 || g.FindPermanentIncludingPhased(phased.ID()).Damage != 0 {
		t.Fatal("wrong recipient collection")
	}
	g.SetOnDamageDealt(nil)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal("source controller lost permission", err)
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 18 || g.FindPermanent(p.ID()).Damage != 2 {
		t.Fatal("controller activation failed")
	}
}

func TestRONActivationPermissionIdentity(t *testing.T) {
	for _, change := range []string{"none", "control", "leave", "return", "phase", "phase-cycle", "copy", "abilities"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "ARN", "Ifh-Bíff Efreet")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			p.GrantBaseAttr(core.Lifelink)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10, mage.WithKeyword(core.Flying)), b.PlayerID())
			b.ManaPool().Add(core.Green, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: victim.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(victim.ID()).Damage != 0 {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				packets, recipients := 1, 4
				if change == "copy" {
					packets = 2
				}
				if change == "leave" || change == "phase" || change == "abilities" {
					recipients = 3
				}
				wantA, wantB := 20-packets, 20-packets
				if change == "control" {
					wantB += recipients * packets
				} else if change != "abilities" {
					wantA += recipients * packets
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB || game.FindPermanent(victim.ID()).Damage != packets {
					t.Fatalf("wrong source LKI or controller: A=%d B=%d damage=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), game.FindPermanent(victim.ID()).Damage)
				}
			}
		})
	}
}

func TestRONActivationPermissionActorsAndTiming(t *testing.T) {
	for _, modal := range []bool{false, true} {
		body := `ActionList([GainLife(Integer(2))])`
		if modal {
			body = `Modal_ChooseOne([ActionList([GainLife(Integer(2))]),ActionList([GainLife(Integer(3))])])`
		}
		factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostG]),`+body+`,And([OtherPlayersMayActivate(AnyPlayer),ActivateOnlyAsASorcery]))`)
		g, a, b := ronGame()
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		b.ManaPool().Add(core.Green, 1)
		if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
			t.Fatal("permission removed sorcery timing")
		}
		g.SetActivePlayerIndex(1)
		if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 20 || b.Life() != 22 || b.ManaPool().TotalMana() != 0 {
			t.Fatal("You did not read the activator")
		}
	}
}

func TestRONActivationPermissionModalChoices(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: 0}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: 1}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostG]),Modal_ChooseOne([ActionList([GainLife(Integer(2))]),ActionList([GainLife(Integer(3))])]),OtherPlayersMayActivate(AnyPlayer))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	a.ManaPool().Add(core.Green, 1)
	b.ManaPool().Add(core.Green, 1)
	for _, actor := range []*ronModePlayer{a, b} {
		if err := g.ActivateAbilityByIndex(actor.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.StackPeek().ModeChoice != actor.mode || g.StackPeek().Controller != actor.PlayerID() || actor.prompts != 1 {
			t.Fatal("mode or stack controller belongs to another player")
		}
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("independent activations shared mode or clone state")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatal("cloned modes lost their activators")
	}
}

func TestRONActivationPermissionIndependentInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(2))]),And([OtherPlayersMayActivate(AnyPlayer),ActivateNoMoreThanNumberTimesEachTurn(Integer(1))]))`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err == nil || g.StackSize() != 1 {
		t.Fatal("changing activator bypassed source ability limit")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal("factory instances shared activation history", err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 24 {
		t.Fatal("activation and copy controllers shared resolving context")
	}
}

func TestRONActivationPermissionCostRollback(t *testing.T) {
	a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B"), invalid: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(And([PayLife(Integer(2)),DiscardACard]),ActionList([GainLife(Integer(3))]),OtherPlayersMayActivate(AnyPlayer))`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	for _, actor := range []*ronChosenCostPlayer{a, b} {
		card := mage.NewLand("Payment")
		card.SetOwner(actor.PlayerID())
		actor.AddToHand(card)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil || a.Life() != 20 || b.Life() != 20 || len(a.Hand()) != 1 || len(b.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("invalid opponent discard did not roll back life and cards")
	}
	b.invalid = false
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 || a.Life() != 20 || len(b.Hand()) != 0 || len(a.Hand()) != 1 || len(b.Graveyard()) != 1 {
		t.Fatal("payment used another player's resources")
	}
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatal("resolution used the source controller")
	}
}

func TestRONActivationPermissionTargetsAndPresence(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostG]),Targeted([TargetPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,1)],UntilEndOfTurn)])),OtherPlayersMayActivate(AnyPlayer))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("A's creature", "", 2, 2), a.PlayerID())
	r := g.PutOnBattlefield(mage.NewCreature("B's creature", "", 2, 2), b.PlayerID())
	b.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("target legality used source controller or paid an invalid cost")
	}
	g.PhaseOut(p.ID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{r.ID()}); err == nil || b.ManaPool().TotalMana() != 1 {
		t.Fatal("phased source could activate")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{r.ID()}); err != nil {
		t.Fatal(err)
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: r.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	g.ResolveStack()
	if g.FindPermanent(r.ID()).CurrentPower(g) != 2 {
		t.Fatal("resolution did not recheck target control against activator")
	}
}

func TestRONActivationPermissionEfreetPreventionAndMembership(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "ARN", "Ifh-Bíff Efreet"), a.PlayerID())
	protected := g.PutOnBattlefield(mage.NewCreature("Protected", "", 1, 10, mage.WithKeyword(core.Flying), mage.WithAbility(mage.ProtectionFromColor(core.Green))), b.PlayerID())
	lost := g.PutOnBattlefield(mage.NewCreature("Lost flying", "", 1, 10, mage.WithKeyword(core.Flying)), b.PlayerID())
	gained := g.PutOnBattlefield(mage.NewCreature("Gained flying", "", 1, 1), b.PlayerID())
	b.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(lost.ID()).RevokeBaseAttr(core.Flying)
	g.MutablePermanent(gained.ID()).GrantBaseAttr(core.Flying)
	g.AddPreventionShield(a.PlayerID(), 1)
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 19 || g.FindPermanent(p.ID()).Damage != 1 || g.FindPermanent(protected.ID()).Damage != 0 || g.FindPermanent(lost.ID()).Damage != 0 || g.FindPermanent(gained.ID()) != nil {
		t.Fatal("source properties, resolution membership, prevention or lethal damage failed")
	}
}

func TestRONActivationPermissionInvalid(t *testing.T) {
	for _, modifier := range []string{
		`OtherPlayersMayActivate`, `OtherPlayersMayActivate()`, `OtherPlayersMayActivate(AnyPlayer,AnyPlayer)`,
		`OtherPlayersMayActivate(AnyPlayer())`, `OtherPlayersMayActivate(Opponent)`, `OtherPlayersMayActivate(You)`,
		`OtherPlayersMayActivate(Ref_TargetPlayer)`, `OtherPlayersMayActivate(SinglePlayer(You))`,
		`And([OtherPlayersMayActivate(AnyPlayer),OtherPlayersMayActivate(AnyPlayer)])`,
		`And([OtherPlayersMayActivate(AnyPlayer),And([OtherPlayersMayActivate(AnyPlayer)])])`,
		`OnlyPlayerMayActivate(You)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),` + modifier + `)])`
		nodes, err := Parse(strings.NewReader(src), "invalid-permission.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid permission accepted", modifier)
		}
	}
	for _, rule := range []string{
		`OtherPlayersMayActivate(AnyPlayer)`,
		`Activated(OtherPlayersMayActivate(AnyPlayer),ActionList([GainLife(Integer(1))]))`,
		`Activated(PayMana([]),ActionList([OtherPlayersMayActivate(AnyPlayer)]))`,
		`ActivatedWithModifiers(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,ControlsA(IsCardtype(Land))),[AddMana(ManaProduceG)],[AddMana(ManaProduceG)])]),OtherPlayersMayActivate(AnyPlayer))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("permission accepted outside its supported context", rule)
		}
	}
}

func TestRONActivationPermissionStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Ifh-Bíff Efreet"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	files := fstest.MapFS{"a.ron": {Data: []byte(record)}, "reprint.ron": {Data: []byte(record)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Ifh-Bíff Efreet")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Ifh-Bíff Efreet")
	if err != nil || first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory instances", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(record, "OtherPlayersMayActivate(AnyPlayer)", "OtherPlayersMayActivate(Opponent)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Ifh-Bíff Efreet") {
		t.Fatal("unsupported collection changed installed registry")
	}
	landEdge := hostUntapRecord(t, "LEG", "Land's Edge")
	if _, err := Compile(landEdge); err != nil {
		t.Fatal("Land's Edge must compile with its discarded-card result", err)
	}
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory compilation", err)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "AnyPlayer" && signature.Role == "player_filter" && strings.Contains(signature.Examples[0].Path, "args[2]") {
			return
		}
	}
	t.Fatal("permission player filter lacks semantic role")
}
