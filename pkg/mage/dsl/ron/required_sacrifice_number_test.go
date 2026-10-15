package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type requiredNumberPlayer struct{ *optionalSacrificePlayer }

func (p *requiredNumberPlayer) ChoosePermanent(candidates []*mage.Permanent, _ string, _ mage.GameReader) *mage.Permanent {
	return candidates[len(candidates)-1]
}

func TestRONRequiredSacrificeNumberLich(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Lich")
			rules := record.Fields["Rules"].Args
			rule := rules[3]
			body := rule.Args[1].Args[0]
			branch := body.Args[1]
			branch.Args[1] = nameScopeNode(t, `[DrawACard]`)
			body.Args[1] = branch
			rule.Args[1].Args[0] = body
			factory := staticBaseSubrule(t, record, rules[0], rules[1], rules[2], rule)
			a := &requiredNumberPlayer{&optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: false}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			victims := make([]*mage.Permanent, 0, 2)
			for range 2 {
				c := mage.NewCreature("Payment", "", 1, 1)
				c.SetOwner(b.PlayerID())
				victims = append(victims, g.PutOnBattlefield(c, a.PlayerID()))
			}
			a.choice = victims[0].ID()
			g.DealDamageToPlayer(a, 2, source.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(b.Graveyard()) != 0 {
				t.Fatal("early payment or absent trigger")
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(b.Graveyard()) != 2 || g.FindPermanent(source.ID()) == nil || a.offers != 0 || a.Life() != -2 {
				t.Fatal("event amount, payer, or nontoken payment")
			}
			if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
				t.Fatal("original changed clone")
			}
			clone.ResolveStack()
			if len(clone.GetPlayer(b.PlayerID()).Graveyard())+len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 2 {
				t.Fatal("clone lost event amount")
			}
			if _, err := Compile(staticBaseRecord(t, set, "Lich")); err == nil || !strings.Contains(err.Error(), "unsupported graveyard event permanent filter (SinglePermanent)") {
				t.Fatalf("full record diagnostic: %v", err)
			}
		})
	}
}

func TestRONRequiredSacrificeNumberBranches(t *testing.T) {
	for _, condition := range []string{"If(CostWasPaid", "Unless(CostWasPaid", "If(CostWasntPaid", "Unless(CostWasntPaid"} {
		for _, n := range []string{"0", "1", "2", "-1"} {
			t.Run(condition+n, func(t *testing.T) {
				rule := `Activated(PayMana([]),ActionList([MustCost(SacrificeNumberPermanents(Integer(` + n + `),IsNonToken)),` + condition + `,[GainLife(Integer(3))])]))`
				factory := permanentScopeFactory(t, "Artifact", rule)
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: false}
				g := mage.NewGame(a, mage.NewBasePlayer("B"))
				g.SetStep(core.PrecombatMain)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				paid := n != "2"
				want := 20
				if (condition == "If(CostWasPaid" || condition == "Unless(CostWasntPaid") == paid {
					want = 23
				}
				if a.Life() != want || a.offers != 0 || (g.FindPermanent(source.ID()) == nil) != (n == "1") {
					t.Fatal("branch, zero amount, or source inclusion")
				}
			})
		}
	}
}

func TestRONRequiredSacrificeNumberInvalid(t *testing.T) {
	for _, body := range []string{
		`MustCost(SacrificeNumberPermanents(Integer(1),AnyPermanent))`,
		`MustCost(SacrificeNumberPermanents(Integer(1))),If(CostWasPaid,[DrawACard])`,
		`MustCost(SacrificeNumberPermanents(Integer(1),And([]))),If(CostWasPaid,[DrawACard])`,
		`MustCost(SacrificeNumberPermanents(Integer(1),Other(ThisPermanent))),If(CostWasPaid,[DrawACard])`,
		`MustCost(SacrificeNumberPermanents(Trigger_AmountOfDamageDealt,AnyPermanent)),If(CostWasPaid,[DrawACard])`,
		`MustCost(SacrificeNumberPermanents(Integer(1),AnyPermanent)),If(CostWasPaid(),[DrawACard])`,
		`MustCost(SacrificeNumberPermanents(Integer(1),AnyPermanent)),If(CostWasPaid,[])`,
		`MustCost(SacrificeNumberPermanents(Integer(1),AnyPermanent)),DrawACard,If(CostWasPaid,[DrawACard])`,
		`MustCost(SacrificeNumberPermanents(Integer(1),AnyPermanent)),If(CostWasPaid,[DrawACard]),If(CostWasPaid,[DrawACard])`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONRequiredSacrificeNumberContexts(t *testing.T) {
	for _, context := range []string{"activated", "upkeep", "delayed", "spell"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "control", "copy"} {
			t.Run(context+"/"+state, func(t *testing.T) {
				body := `MustCost(SacrificeNumberPermanents(Integer(2),IsCardtype(Land))),If(CostWasPaid,[GainLife(Integer(3))])`
				rule := `Activated(PayMana([]),ActionList([` + body + `]))`
				if context == "upkeep" {
					rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([` + body + `]))`
				}
				if context == "delayed" {
					rule = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))]))`
				}
				cardtype := "Artifact"
				if context == "spell" {
					cardtype = "Instant"
					rule = `SpellActions(ActionList([` + body + `]))`
				}
				factory := permanentScopeFactory(t, cardtype, rule)
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: false}
				b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: false}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				for range 2 {
					g.PutOnBattlefield(mage.NewLand("A payment"), a.PlayerID())
					g.PutOnBattlefield(mage.NewLand("B payment"), b.PlayerID())
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if context == "spell" {
					castRONSpell(t, g, a.BasePlayer, card)
				} else {
					source = g.PutOnBattlefield(card, a.PlayerID())
					if context == "upkeep" {
						queueRONUpkeep(g, a)
					} else if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if context == "delayed" {
						g.ResolveStack()
						g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						g.PutTriggersOnStack()
					}
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(source)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							g.PutOnBattlefield(card, b.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				if a.Life() != 20 || len(a.Graveyard()) != 0 {
					t.Fatal("payment occurred before resolution")
				}
				clone := g.Clone()
				g.ResolveStack()
				wantB := 20
				if state == "copy" {
					wantB = 23
				}
				if a.Life() != 23 || b.Life() != wantB || a.offers+b.offers != 0 {
					t.Fatalf("payer, branch or offer: %d/%d", a.Life(), b.Life())
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("original changed clone")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("clone lost payment")
				}
			})
		}
	}
}

func TestRONRequiredSacrificeNumberLocalResults(t *testing.T) {
	rule := `Activated(PayMana([]),ActionList([MustCost(SacrificeNumberPermanents(Integer(1),IsCardtype(Land))),If(CostWasPaid,[MustCost(SacrificeNumberPermanents(Integer(1),IsCardtype(Land))),Unless(CostWasPaid,[GainLife(Integer(2))]),GainLife(Integer(3))]),MustCost(SacrificeNumberPermanents(Integer(1),IsCardtype(Land))),Unless(CostWasPaid,[GainLife(Integer(7))])]))`
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	g.PutOnBattlefield(mage.NewLand("Payment"), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 32 {
		t.Fatal("nested or adjacent payment result escaped")
	}
}

func TestRONRequiredSacrificeNumberWaitingAndLKI(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned"} {
		t.Run(state, func(t *testing.T) {
			rule := `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MustCost(SacrificeNumberPermanents(PowerOfPermanent(ThisPermanent),IsCardtype(Land))),If(CostWasPaid,[GainLife(Integer(3))])]))]))`
			n := nameScopeNode(t, `Card(Name:"Dynamic",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[`+rule+`])`)
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for range 2 {
				g.PutOnBattlefield(mage.NewLand("Payment"), a.PlayerID())
			}
			saved := g.CaptureDamageSource(source.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if state != "live" {
				g.BouncePermanentToHand(source)
			}
			if state == "returned" {
				a.RemoveFromHand(card.ID())
				fresh := g.PutOnBattlefield(card, b.PlayerID())
				fresh.AddCounter(core.P1P1, 7)
			}
			clone := g.Clone()
			for range 3 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) || a.Life() != 20 || len(a.Graveyard()) != 0 {
					t.Fatal("waiting query changed state or lost exact source")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 23 || len(a.Graveyard()) != 2 {
				t.Fatal("source LKI or dynamic payment amount")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("waiting clone changed")
			}
		})
	}
}

func TestRONRequiredSacrificeNumberNontokenAndInventory(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Lich")
	rule := record.Fields["Rules"].Args[3]
	body := rule.Args[1].Args[0]
	branch := body.Args[1]
	branch.Args[1] = nameScopeNode(t, `[DrawACard]`)
	body.Args[1] = branch
	rule.Args[1].Args[0] = body
	factory := staticBaseSubrule(t, record, rule)
	g, a, b := ronGame()
	a.AddToLibrary(mage.NewLand("Failure result"))
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1), a.PlayerID())
	token := g.PutOnBattlefield(mage.NewCreature("Token", "", 1, 1), a.PlayerID())
	token.IsToken = true
	phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 1), a.PlayerID())
	g.PhaseOut(phased.ID())
	g.PutOnBattlefield(mage.NewCreature("Opponent", "", 1, 1), b.PlayerID())
	g.DealDamageToPlayer(a, 3, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 1 || g.FindPermanent(victim.ID()) == nil || g.FindPermanent(source.ID()) == nil || g.FindPermanent(token.ID()) == nil || len(a.Graveyard()) != 0 {
		t.Fatal("unaffordable nontoken payment changed resources or lost unpaid branch")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEA.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Lich"`) {
			text = line
			break
		}
	}
	report, err := InspectFS(fstest.MapFS{"lich.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"SacrificeNumberPermanents": "resolution_cost", "Trigger_AmountOfDamageDealt": "value", "IsNonToken": "permanent_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inventory role %s/%s", name, role)
		}
	}
}
