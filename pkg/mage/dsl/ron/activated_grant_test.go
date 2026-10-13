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

func TestRONActivatedGrantZombieMaster(t *testing.T) {
	for _, change := range []string{"none", "donor-left", "donor-phase", "recipient-phase", "recipient-return", "control", "remove", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Zombie Master"), a.PlayerID())
			victim := mage.NewCreature("Zombie", "", 2, 3, mage.WithSubTypes("Zombie"))
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(donor.ID()).RuntimeAbilities) != 2 || len(g.FindPermanent(other.ID()).RuntimeAbilities) != 0 || !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
				t.Fatal("wrong grant membership")
			}
			ability := g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID()
			g.ApplyContinuousEffects()
			if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != ability {
				t.Fatal("layer pass changed ability identity")
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation accepted")
			}
			b.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().TotalMana() != 0 || g.HasRegenerationShield(p.ID()) {
				t.Fatal("payment or resolution timing")
			}
			switch change {
			case "donor-left":
				g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			case "donor-phase":
				g.PhaseOut(donor.ID())
			case "recipient-phase":
				g.PhaseOut(p.ID())
			case "recipient-return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(victim, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "remove":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.HasRegenerationShield(p.ID()) {
				t.Fatal("clone changed live shield")
			}
			g.ResolveStack()
			want := change != "recipient-return" && change != "recipient-phase"
			if g.HasRegenerationShield(p.ID()) != want || clone.HasRegenerationShield(p.ID()) != want || g.HasRegenerationShield(donor.ID()) {
				t.Fatal("grant body used wrong incarnation or source")
			}
			if want {
				g.DestroyPermanent(g.FindPermanent(p.ID()))
				if g.FindPermanent(p.ID()) == nil || !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("regeneration failed")
				}
			}
			g.ApplyContinuousEffects()
			if change == "donor-left" || change == "donor-phase" {
				if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 || g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
					t.Fatal("inactive donor retained grant")
				}
			}
		})
	}
}

func TestRONActivatedGrantLifeMatrix(t *testing.T) {
	for _, change := range []string{"none", "source-left", "target-phase", "target-return", "shroud", "phase-cycle", "control", "copy", "two"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.Upkeep)
			card := repositoryCard(t, "LEG", "Life Matrix")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := mage.NewCreature("Victim", "", 2, 4)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("unpaid activation did not roll back")
			}
			a.ManaPool().Add(core.Colorless, 8)
			g.SetStep(core.PrecombatMain)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err == nil || a.ManaPool().TotalMana() != 8 {
				t.Fatal("upkeep restriction missing")
			}
			g.SetStep(core.Upkeep)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err == nil || a.ManaPool().TotalMana() != 8 || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("invalid target spent costs")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 4 || !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(p.ID()).Counters[core.Matrix] != 0 {
				t.Fatal("activation timing")
			}
			switch change {
			case "source-left":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "target-phase":
				g.PhaseOut(p.ID())
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(victim, b.PlayerID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "two":
				g.MutablePermanent(source.ID()).Tapped = false
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{p.ID()})
			want := 1
			if change == "copy" || change == "two" {
				want = 2
			}
			if change == "target-phase" || change == "target-return" || change == "shroud" {
				want = 0
			}
			q := g.FindPermanent(p.ID())
			if q.Counters[core.Matrix] != want || len(q.RuntimeAbilities) != want {
				t.Fatalf("counter/grant count %d/%d want %d", q.Counters[core.Matrix], len(q.RuntimeAbilities), want)
			}
			if want == 0 {
				return
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if change == "source-left" && len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
				t.Fatal("donor departure ended captured grant")
			}
			actor := g.FindPermanent(p.ID()).ControllerID()
			if err := g.ActivateAbilityByIndex(actor, p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(p.ID()).Counters[core.Matrix] != want-1 || g.HasRegenerationShield(p.ID()) {
				t.Fatal("counter payment timing")
			}
			if clone.FindPermanent(p.ID()).Counters[core.Matrix] != want {
				t.Fatal("clone shares payment")
			}
			g.ResolveStack()
			if !g.HasRegenerationShield(p.ID()) {
				t.Fatal("recipient did not regenerate")
			}
			for i := 1; i < want; i++ {
				if err := g.ActivateAbilityByIndex(actor, p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			}
			if err := g.ActivateAbilityByIndex(actor, p.ID(), 0, nil); err == nil || g.FindPermanent(p.ID()).Counters[core.Matrix] != 0 {
				t.Fatal("missing counter cost accepted")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(p.ID()).RuntimeAbilities) != want {
				t.Fatal("phasing lost grant")
			}
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.PutOnBattlefield(victim, b.PlayerID())
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 {
				t.Fatal("grant followed returned object")
			}
		})
	}
}

func TestRONActivatedGrantInvalid(t *testing.T) {
	for _, ability := range []string{
		`Activated()`, `Activated(PayMana([]),ActionList([RegeneratePermanent(HostPermanent)]))`,
		`Activated(PayMana([]),ActionList([RegeneratePermanent(Ref_TargetPermanent)]))`,
		`Activated(PayManaX([ManaCostX],ValueX),ActionList([GainLife(ValueX)]))`,
		`Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([RegeneratePermanent(Ref_TargetPermanent)])))`,
		`ActivatedWithModifiers(PayMana([]),ActionList([RegeneratePermanent(ThisPermanent)]),ActivateOnlyOnceEachTurn)`,
		`Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatCreature)]))`,
	} {
		for _, layer := range []string{`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([%s])])`, `Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[AddAbility([%s])])]))`} {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+layer+`])`, ability)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s", src)
			}
		}
	}
}

func TestRONActivatedGrantCasting(t *testing.T) {
	for _, tc := range []struct {
		set, name        string
		mana             core.Color
		count            int
		power, toughness int
	}{
		{"LEA", "Zombie Master", core.Black, 3, 2, 3}, {"LEG", "Life Matrix", core.Colorless, 4, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			if card.Power() != tc.power || card.Toughness() != tc.toughness {
				t.Fatal("wrong printed P/T")
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(tc.mana, tc.count)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("casting cost or entry timing")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("permanent did not enter")
			}
		})
	}
}

func TestRONActivatedGrantRecipientScopeAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Activated(PayMana([]),ActionList([PlayerAction(You,GainLife(Integer(2))),PutCounters([ACounterOfTypeOnPermanent(ChargeCounter,ThisPermanent)])]))])])`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 1), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 1), b.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 2 || len(g.FindPermanent(q.ID()).RuntimeAbilities) != 2 {
		t.Fatal("independent donors did not grant separate abilities")
	}
	for _, pair := range []struct{ actor, id uuid.UUID }{{a.PlayerID(), p.ID()}, {b.PlayerID(), q.ID()}} {
		if err := g.ActivateAbilityByIndex(pair.actor, pair.id, 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 || g.FindPermanent(p.ID()).Counters[core.Charge] != 1 || g.FindPermanent(q.ID()).Counters[core.Charge] != 1 {
		t.Fatal("body used donor controller or another recipient")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost body bindings")
	}
}

func TestRONActivatedGrantNoncreatureZombieAndMembership(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Zombie Master"), a.PlayerID())
	p := g.PutOnBattlefield(mage.NewArtifact("Zombie artifact", "", mage.WithSubTypes("Zombie")), b.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 || g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("creature filter leaked into regeneration or swampwalk")
	}
	b.ManaPool().Add(core.Black, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.HasRegenerationShield(p.ID()) {
		t.Fatal("noncreature Zombie did not regenerate")
	}
	g.PhaseIn(g.PhaseOut(source.ID()))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
		t.Fatal("phase cycle lost live grant")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 {
		t.Fatal("donor ability removal retained grant")
	}
}

func TestRONActivatedGrantDeclarationsAndStrictLoading(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.SplitSeq(string(data), "\n")
		for line := range lines {
			if strings.Contains(line, `Name: "Zombie Master"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Life Matrix"`) {
			files["LEG.ron"] = &fstest.MapFile{Data: []byte(line)}
		}
	}
	for file, data := range files {
		nodes, err := Parse(strings.NewReader(string(data.Data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("complete records failed strict loading: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 7 || report.UniqueCards != 2 {
		t.Fatalf("reprint counts: %+v %v", report, err)
	}
	roles := map[string]bool{}
	for _, entry := range report.Signatures {
		roles[entry.Name+"/"+entry.Role] = true
	}
	for _, key := range []string{"Activated/keyword", "RemoveCounters/cost", "RegeneratePermanent/action", "ThisPermanent/permanent_reference"} {
		if !roles[key] {
			t.Fatal("missing nested role", key)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[EachPermanentLayerEffect(AnyPermanent,[AddAbility([Activated(PayMana([]),ActionList([RegeneratePermanent(HostPermanent)]))])])])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid grant published partial registry")
	}
}

func TestRONActivatedGrantStaticHostAndSource(t *testing.T) {
	for _, host := range []bool{false, true} {
		for _, change := range []string{"none", "move", "donor-phase", "donor-left", "control"} {
			t.Run(fmt.Sprintf("%t/%s", host, change), func(t *testing.T) {
				rule := `PermanentLayerEffect(ThisPermanent,[AddAbility([Activated(PayMana([]),ActionList([PlayerAction(You,GainLife(Integer(1)))]))])])`
				kind := "Artifact"
				if host {
					kind = "Enchantment"
					rule = `EnchantPermanent(IsCardtype(Creature)),PermanentLayerEffect(HostPermanent,[AddAbility([Activated(PayMana([]),ActionList([PlayerAction(You,GainLife(Integer(1)))]))])])`
				}
				var card mage.Card
				if host {
					nodes, err := Parse(strings.NewReader(`Card(Name:"Host grant",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "host.ron")
					if err != nil {
						t.Fatal(err)
					}
					factory, err := Compile(nodes[0])
					if err != nil {
						t.Fatal(err)
					}
					card = factory()
				} else {
					card = permanentScopeFactory(t, kind, rule)()
				}
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				donor := g.PutOnBattlefield(card, a.PlayerID())
				p := donor
				if host {
					p = g.PutOnBattlefield(mage.NewCreature("Host", "", 1, 2), b.PlayerID())
					g.Attach(donor.ID(), p.ID())
				}
				g.ApplyContinuousEffects()
				actor := g.FindPermanent(p.ID()).ControllerID()
				index := 0
				if !host {
					index = 1
				}
				if err := g.ActivateAbilityByIndex(actor, p.ID(), index, nil); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "move":
					if host {
						other := g.PutOnBattlefield(mage.NewCreature("New host", "", 1, 2), a.PlayerID())
						g.Attach(donor.ID(), other.ID())
					}
				case "donor-phase":
					g.PhaseOut(donor.ID())
				case "donor-left":
					g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				g.ResolveStack()
				if g.GetPlayer(actor).Life() != 21 {
					t.Fatal("pending grant changed actor after donor change")
				}
				if host && (change == "move" || change == "donor-phase" || change == "donor-left") {
					g.ApplyContinuousEffects()
					if len(g.FindPermanent(p.ID()).RuntimeAbilities) != 0 {
						t.Fatal("old host retained live grant")
					}
				}
			})
		}
	}
}

func TestRONActivatedGrantCopyRetarget(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.Upkeep)
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Life Matrix"), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 2), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 2), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(first.ID())
	g.PhaseOut(source.ID())
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
		t.Fatal("copy did not keep separate target")
	}
	g.PhaseIn(phased)
	g.ResolveStack()
	for _, p := range []*mage.Permanent{first, second} {
		if g.FindPermanent(p.ID()).Counters[core.Matrix] != 1 || len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
			t.Fatal("copy and original crossed bindings")
		}
	}
}

func TestRONActivatedGrantSwampwalk(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Zombie Master"), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Zombie", "", 2, 3, mage.WithSubTypes("Zombie")), a.PlayerID())
	g.ApplyContinuousEffects()
	if mage.HasLandwalkEvasion(g.FindPermanent(attacker.ID()), b.PlayerID(), g) {
		t.Fatal("swampwalk applied without Swamp")
	}
	swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
	if !mage.HasLandwalkEvasion(g.FindPermanent(attacker.ID()), b.PlayerID(), g) || mage.HasLandwalkEvasion(g.FindPermanent(donor.ID()), b.PlayerID(), g) {
		t.Fatal("wrong swampwalk or source exclusion")
	}
	g.PhaseOut(swamp.ID())
	if mage.HasLandwalkEvasion(g.FindPermanent(attacker.ID()), b.PlayerID(), g) {
		t.Fatal("phased Swamp enabled swampwalk")
	}
}

func TestRONActivatedGrantCounterReplacement(t *testing.T) {
	for _, prevent := range []bool{false, true} {
		g, a, b := ronGame()
		g.SetStep(core.Upkeep)
		donor := g.PutOnBattlefield(repositoryCard(t, "LEG", "Life Matrix"), a.PlayerID())
		p := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 3), b.PlayerID())
		if prevent {
			g.AddReplacementEffect(&matrixCounterPrevention{counterPlacementReplacement{source: donor.ID(), prevent: true}})
		} else {
			g.AddCounterDoubler(donor.ID(), core.Matrix, mage.AnyPermanent)
		}
		a.ManaPool().Add(core.Colorless, 4)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), donor.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := int(2)
		if prevent {
			want = 0
		}
		if g.FindPermanent(p.ID()).Counters[core.Matrix] != want || len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
			t.Fatal("counter replacement changed the separate ability grant")
		}
	}
}

type matrixCounterPrevention struct{ counterPlacementReplacement }

func (*matrixCounterPrevention) Matches(action mage.Action, _ mage.GameReader) bool {
	placement, ok := action.(*mage.AddCountersAction)
	return ok && placement.CounterType() == core.Matrix && placement.Amount() > 0 && !placement.OnEntry()
}

func (r *matrixCounterPrevention) Clone() mage.ReplacementEffect { cloned := *r; return &cloned }
