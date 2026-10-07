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

func TestRONActivationPropertyClauses(t *testing.T) {
	for _, name := range []string{"Tourach's Gate", "Triassic Egg"} {
		for _, change := range []string{"legal", "false", "phase", "phase-cycle", "unpaid", "control", "host-change", "copy", "departed", "returned"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				set := "FEM"
				if name == "Triassic Egg" {
					set = "LEG"
				}
				n := hostUntapRecord(t, set, name)
				rules := n.Fields["Rules"]
				last := rules.Args[len(rules.Args)-1]
				rule := `ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),ActionList([GainLife(Integer(1))]),` + renderTestNode(last.Args[2]) + `)`
				parsed, err := Parse(strings.NewReader(rule), "clause.ron")
				if err != nil {
					t.Fatal(err)
				}
				if name == "Tourach's Gate" {
					parsed = append([]Node{rules.Args[0]}, parsed...)
				}
				n.Fields["Rules"] = Node{Kind: List, Args: parsed}
				factory, err := Compile(n)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				c := factory()
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				host := g.PutOnBattlefield(mage.NewLand("Host"), a.PlayerID())
				other := g.PutOnBattlefield(mage.NewLand("Other"), a.PlayerID())
				if name == "Tourach's Gate" {
					g.Attach(p.ID(), host.ID())
				}
				g.MutablePermanent(p.ID()).Counters[core.Hatchling] = 2
				payer := a
				switch change {
				case "false":
					if name == "Tourach's Gate" {
						g.TapPermanent(host)
					} else {
						g.MutablePermanent(p.ID()).Counters[core.Hatchling] = 1
					}
				case "phase":
					if name == "Tourach's Gate" {
						g.PhaseOut(host.ID())
					} else {
						g.PhaseOut(p.ID())
					}
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					payer = b
				case "departed", "returned":
					if name == "Tourach's Gate" {
						g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
						if change == "returned" {
							g.PutOnBattlefield(host.Card, a.PlayerID())
						}
					} else {
						g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
						if change == "returned" {
							g.PutOnBattlefield(c, a.PlayerID())
						}
					}
				case "host-change":
					g.TapPermanent(host)
					if name == "Tourach's Gate" {
						g.Attach(p.ID(), other.ID())
					}
				}
				amount := 1
				if change == "unpaid" {
					amount = 0
				}
				payer.ManaPool().Add(core.Colorless, amount)
				before := payer.Life()
				err = g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), 0, nil)
				allowed := change != "false" && change != "phase" && change != "unpaid" && change != "departed" && change != "returned"
				if !allowed {
					if err == nil || payer.ManaPool().TotalMana() != amount || g.StackSize() != 0 {
						t.Fatal("invalid activation changed resources")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				g.MutablePermanent(p.ID()).Counters[core.Hatchling] = 0
				g.TapPermanent(other)
				g.TapPermanent(host)
				clone := g.Clone()
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), payer.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				g.DestroyPermanent(g.FindPermanent(p.ID()))
				g.ResolveStack()
				want := before + 1
				if change == "copy" {
					want++
				}
				if payer.Life() != want || clone.GetPlayer(payer.PlayerID()).Life() != before {
					t.Fatal("condition repeated at resolution or shared clone")
				}
				clone.ResolveStack()
				if clone.GetPlayer(payer.PlayerID()).Life() != before+1 {
					t.Fatal("clone lost condition body")
				}
			})
		}
	}
}

func renderTestNode(n Node) string {
	if n.Kind == Symbol {
		return n.Name
	}
	if n.Kind == IntegerNode {
		return fmt.Sprint(n.Integer)
	}
	parts := make([]string, len(n.Args))
	for i, arg := range n.Args {
		parts[i] = renderTestNode(arg)
	}
	if n.Kind == List {
		return "[" + strings.Join(parts, ",") + "]"
	}
	return n.Name + "(" + strings.Join(parts, ",") + ")"
}

func TestRONActivationPropertyInvalid(t *testing.T) {
	for _, condition := range []string{
		`PermanentPassesFilter(Ref_TargetPermanent,IsUntapped)`, `PermanentPassesFilter(HostPermanent,IsUntapped)`,
		`PermanentPassesFilter(ThisPermanent(),IsUntapped)`, `PermanentPassesFilter(ThisPermanent,IsUntapped())`,
		`PermanentPassesFilter(ThisPermanent,And([]))`, `PermanentPassesFilter(ThisPermanent,HasACounterOfType(UnknownCounter))`,
		`PermanentPassesFilter(ThisPermanent,HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(-1)),HatchlingCounter))`,
		`PermanentPassesFilter(ThisPermanent,HasNumberCountersOfType(GreaterThanOrEqualTo(ValueX),HatchlingCounter))`,
		`PermanentPassesFilter(ThisPermanent,HasNumberCountersOfType(GreaterThan(Integer(1)),HatchlingCounter))`,
		`PermanentPassesFilter(ThisPermanent)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),ActivateOnlyIf(` + condition + `))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", condition)
		}
	}
}

type ronEggPlayer struct {
	*ronModePlayer
	accept bool
}

func (p *ronEggPlayer) ChooseMayAbility(string) bool { return p.accept }
func (p *ronEggPlayer) ChooseCardFromHand(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	if len(cards) > 0 {
		return cards[0]
	}
	return nil
}

func TestRONActivationPropertyTriassicEgg(t *testing.T) {
	for mode := range 2 {
		for _, change := range []string{"legal", "one-counter", "phase", "wrong-controller", "bad-target", "decline", "clone", "copy", "stale-target", "source-return"} {
			t.Run(fmt.Sprintf("%d/%s", mode, change), func(t *testing.T) {
				a := &ronEggPlayer{ronModePlayer: &ronModePlayer{BasePlayer: mage.NewBasePlayer("A"), mode: mode}, accept: change != "decline"}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				c := repositoryCard(t, "LEG", "Triassic Egg")
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				a.ManaPool().Add(core.Colorless, 4)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(c.ID())
				for range 2 {
					a.ManaPool().Add(core.Colorless, 3)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					g.UntapPermanent(g.FindPermanent(p.ID()))
				}
				if g.FindPermanent(p.ID()).Counters[core.Hatchling] != 2 {
					t.Fatal("counter activation")
				}
				v := mage.NewCreature("Creature", "", 2, 3)
				v.SetOwner(a.PlayerID())
				if mode == 0 {
					a.AddToHand(v)
				} else {
					a.AddToGraveyard(v)
					a.targets = []uuid.UUID{v.ID()}
				}
				payer := a.PlayerID()
				switch change {
				case "one-counter":
					g.MutablePermanent(p.ID()).Counters[core.Hatchling] = 1
				case "phase":
					g.PhaseOut(p.ID())
				case "wrong-controller":
					payer = b.PlayerID()
				case "bad-target":
					if mode == 1 {
						a.targets = []uuid.UUID{uuid.New()}
					}
				}
				err := g.ActivateAbilityByIndex(payer, p.ID(), 1, nil)
				invalid := change == "one-counter" || change == "phase" || change == "wrong-controller" || change == "bad-target" && mode == 1
				if invalid {
					if err == nil || g.FindPermanentIncludingPhased(p.ID()) == nil || g.StackSize() != 0 {
						t.Fatal("illegal activation consumed source")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(p.ID()) != nil || g.StackSize() != 1 || len(g.StackPeek().Targets) != mode {
					t.Fatal("sacrifice or mode targets")
				}
				clone := g.Clone()
				if change == "stale-target" && mode == 1 {
					a.RemoveFromGraveyard(v.ID())
					a.AddToGraveyard(v)
				}
				if change == "source-return" {
					a.RemoveFromGraveyard(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				g.ResolveStack()
				want := (mode != 0 || change != "decline") && (mode != 1 || change != "stale-target")
				if (g.FindPermanent(v.ID()) != nil) != want {
					t.Fatal("wrong mode entry or visit identity")
				}
				if clone.FindPermanent(v.ID()) != nil {
					t.Fatal("clone shared entry")
				}
				if change == "clone" {
					clone.SetPlayerAt(0, &ronEggPlayer{ronModePlayer: &ronModePlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, mode: mode}, accept: true})
					clone.ResolveStack()
					if clone.FindPermanent(v.ID()) == nil {
						t.Fatal("clone entry")
					}
				}
			})
		}
	}
}

func TestRONActivationPropertyThresholdsAndModifiers(t *testing.T) {
	for _, tc := range []struct {
		filter  string
		count   int
		allowed bool
	}{
		{`HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),HatchlingCounter)`, 0, false},
		{`HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),HatchlingCounter)`, 1, false},
		{`HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),HatchlingCounter)`, 2, true},
		{`HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(2)),HatchlingCounter)`, 3, true},
		{`HasNumberCountersOfType(GreaterThanOrEqualTo(Integer(256)),HatchlingCounter)`, 255, false},
		{`HasNoCountersOfType(HatchlingCounter)`, 0, true},
		{`HasACounterOfType(HatchlingCounter)`, 0, false},
		{`HasNumberCountersOfType(EqualTo(Integer(2)),HatchlingCounter)`, 3, false},
		{`And([IsUntapped,Or([HasNoCountersOfType(HatchlingCounter),HasACounterOfType(ChargeCounter)])])`, 3, true},
	} {
		t.Run(fmt.Sprintf("%s/%d", tc.filter, tc.count), func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),ActionList([GainLife(Integer(1))]),And([ActivateOnlyIf(PermanentPassesFilter(ThisPermanent,`+tc.filter+`)),ActivateOnlyIf(PlayerPassesFilter(You,IsTheirUpkeep)),ActivateOnlyAsASorcery]))`)
			g, a, _ := ronGame()
			g.SetStep(core.Upkeep)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Hatchling] = uint8(tc.count)
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 1
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("sorcery restriction overwritten")
			}
			if a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("timing rejection changed resources")
			}
			factory = permanentScopeFactory(t, "Artifact", `ActivatedWithModifiers(PayMana([ManaCostGeneric(1)]),ActionList([GainLife(Integer(1))]),And([ActivateOnlyIf(PermanentPassesFilter(ThisPermanent,`+tc.filter+`)),ActivateOnlyIf(PlayerPassesFilter(You,IsTheirUpkeep))]))`)
			c = factory()
			c.SetOwner(a.PlayerID())
			p = g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(p.ID()).Counters[core.Hatchling] = uint8(tc.count)
			g.MutablePermanent(p.ID()).Counters[core.Charge] = 1
			err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil)
			if (err == nil) != tc.allowed {
				t.Fatal("wrong live counter threshold", err)
			}
		})
	}
}

func TestRONActivationPropertyManaPlanning(t *testing.T) {
	factory := permanentScopeFactory(t, "Land", `ActivatedWithModifiers(TapPermanent(ThisPermanent),ActionList([AddMana(ManaProduceU)]),ActivateOnlyIf(PermanentPassesFilter(ThisPermanent,HasACounterOfType(ChargeCounter))))`)
	g, a, _ := ronGame()
	c := factory()
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	for _, count := range []uint8{0, 1, 0, 2} {
		g.MutablePermanent(p.ID()).Counters[core.Charge] = count
		if g.CanAfford(a.PlayerID(), core.ParseManaCost("{U}"), nil) != (count > 0) {
			t.Fatal("solver ignored condition")
		}
		if g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
			t.Fatal("query changed resources")
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).Tapped || a.ManaPool().Count(core.Blue) != 1 || g.StackSize() != 0 {
		t.Fatal("mana activation did not resolve immediately")
	}
}

func TestRONActivationPropertyInventoryAndStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	n := hostUntapRecord(t, "LEG", "Triassic Egg")
	record := strings.Split(string(data), "\n")[n.Position.Line-1]
	files := fstest.MapFS{"egg.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("corrected record failed validation", err)
	}
	for name, role := range map[string]string{"PermanentPassesFilter": "activation_condition", "ThisPermanent": "permanent_reference", "HasNumberCountersOfType": "permanent_filter", "HatchlingCounter": "permanent_counter_type"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Rules:[Unsupported])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection replaced registry")
	}
	if result, err := LoadFS(files, "*.ron"); err == nil || result != nil {
		t.Fatal("published partial registry")
	}
}
