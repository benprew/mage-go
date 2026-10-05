package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolutionManaEye(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "unaffordable", "zero-pay", "zero-decline", "source-left", "source-return", "source-phase", "source-control", "source-abilities", "spell-left", "copy", "clone", "same-player", "noninstant", "mana-source", "uncounterable"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: scenario != "decline" && scenario != "zero-decline"}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			eye := repositoryCard(t, "LEG", "In the Eye of Chaos")
			if err := mage.ValidateCardDefinition(eye.Name(), eye); err != nil {
				t.Fatal(err)
			}
			eye.SetOwner(a.PlayerID())
			a.AddToHand(eye)
			if err := g.CastSpellByID(a.PlayerID(), eye.ID(), nil, 0); err == nil {
				t.Fatal("unpaid enchantment cast accepted")
			}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), eye.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(eye.ID()) != nil {
				t.Fatal("enchantment entered before resolution")
			}
			g.ResolveStack()
			if !eye.HasSuperType(core.SuperWorld) {
				t.Fatal("world metadata lost")
			}
			g.SetActivePlayerIndex(1)
			caster := b
			if scenario == "same-player" {
				caster = a
				g.SetActivePlayerIndex(0)
			}
			types := "Instant"
			if scenario == "noninstant" {
				types = "Sorcery"
			}
			x := 2
			mana := `[ManaCostX,ManaCostU]`
			if strings.HasPrefix(scenario, "zero-") {
				mana = `[ManaCostGeneric(0)]`
				x = 0
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Victim",Typeline:(Supertypes:[],Cardtypes:[`+types+`],Subtypes:[]),ManaCost:`+mana+`,Rules:[SpellActions(ActionList([PlayerAction(You,GainLife(Integer(1)))]))])`), "victim.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			victim := factory()
			if scenario == "uncounterable" {
				mage.WithUncounterable()(victim.(*mage.BaseCard))
			}
			victim.SetOwner(caster.PlayerID())
			caster.AddToHand(victim)
			caster.ManaPool().Add(core.Blue, 1)
			caster.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(caster.PlayerID(), victim.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			spell := g.StackPeek()
			caster.ManaPool().Clear()
			if scenario != "unaffordable" && scenario != "mana-source" {
				caster.ManaPool().Add(core.Colorless, 3)
			}
			if scenario == "mana-source" {
				for range 3 {
					g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), caster.PlayerID())
				}
			}
			g.PutTriggersOnStack()
			if scenario == "noninstant" {
				if g.StackSize() != 1 {
					t.Fatal("sorcery triggered the tax")
				}
				return
			}
			if g.StackSize() != 2 || a.offers+b.offers != 0 {
				t.Fatal("trigger or payment timing")
			}
			switch scenario {
			case "source-left", "source-return":
				g.BouncePermanentToHand(g.FindPermanent(eye.ID()))
				if scenario == "source-return" {
					a.RemoveFromHand(eye.ID())
					g.PutOnBattlefield(eye, a.PlayerID())
				}
			case "source-phase":
				g.PhaseOut(eye.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: eye.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), eye.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "spell-left":
				g.GetStack().RemoveByID(spell.ID)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("trigger copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveTopOfStack()
				if caster.ManaPool().TotalMana() != 3 || g.StackSize() != 2 {
					t.Fatal("clone changed pending payment")
				}
			}
			g.SetXValue(99)
			g.ResolveTopOfStack()
			paid := scenario != "decline" && scenario != "zero-decline" && scenario != "unaffordable"
			remaining := g.GetStack().FindByID(spell.ID) != nil
			if scenario == "spell-left" {
				if caster.offers != 1 || caster.ManaPool().TotalMana() != 0 {
					t.Fatal("absent spell lost its mana value or captured caster")
				}
				return
			}
			if remaining != (paid || scenario == "uncounterable") {
				t.Fatal("counter or paid branch failed")
			}
			if scenario == "copy" {
				g.ResolveTopOfStack()
				if g.GetStack().FindByID(spell.ID) != nil {
					t.Fatal("second payment inherited first outcome")
				}
			}
			if scenario != "mana-source" {
				want := 3
				if scenario == "unaffordable" {
					want = 0
				}
				if paid && !strings.HasPrefix(scenario, "zero-") {
					want = 0
				}
				if caster.ManaPool().TotalMana() != want {
					t.Fatalf("mana = %d, want %d", caster.ManaPool().TotalMana(), want)
				}
			}
			if scenario != "same-player" && a.offers != 0 {
				t.Fatal("tax controller paid instead of caster")
			}
		})
	}
}

func TestRONResolutionManaInvalid(t *testing.T) {
	for _, cost := range []string{`PayManaX`, `PayManaX([ManaCostX])`, `PayManaX([],Integer(2))`, `PayManaX([ManaCostGeneric(2)],Integer(2))`, `PayManaX([ManaCostX],Integer(-1))`, `PayManaX([ManaCostX],ValueX)`, `PayManaX([ManaCostX],ManaValueOfSpell(Trigger_ThatSpell))`, `PayManaX([ManaCostX],ManaValueOfSpell(Ref_TargetPermanent))`, `PayManaX([ManaCostX],ManaValueOfSpell(Ref_TargetSpell()))`, `PayManaX([ManaCostX],ManaValueOfSpell(Ref_TargetSpell,Ref_TargetSpell))`} {
		rule := fmt.Sprintf(`SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([PlayerMayCost(ControllerOfSpell(Ref_TargetSpell),%s),Unless(CostWasPaid,[CounterSpell(Ref_TargetSpell)])])))`, cost)
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
}

func TestRONResolutionManaContextsAndInstances(t *testing.T) {
	for _, context := range []string{"activation", "spell", "trigger", "delayed"} {
		for _, paidBody := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", context, paidBody), func(t *testing.T) {
				branch := "Unless(CostWasPaid"
				if paidBody {
					branch = "Unless(CostWasntPaid"
				}
				body := `ActionList([MayCost(PayManaX([ManaCostX,ManaCostX,ManaCostU,ManaCostGeneric(1)],Integer(2))),` + branch + `,[GainLife(Integer(4))])])`
				types, rule := "Artifact", `Activated(PayMana([]),`+body+`)`
				if context == "spell" {
					types = "Instant"
					rule = `SpellActions(` + body + `)`
				}
				if context == "trigger" {
					rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + body + `)`
				}
				if context == "delayed" {
					rule = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,` + body + `)]))`
				}
				factory := permanentScopeFactory(t, types, rule)
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				for _, player := range []*ronPaymentPlayer{a, b} {
					c := factory()
					if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
						t.Fatal(err)
					}
					c.SetOwner(player.PlayerID())
					if context == "spell" {
						player.AddToHand(c)
						if err := g.CastSpellByID(player.PlayerID(), c.ID(), nil, 0); err != nil {
							t.Fatal(err)
						}
					} else {
						p := g.PutOnBattlefield(c, player.PlayerID())
						if context == "trigger" {
							g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player.PlayerID()})
							g.PutTriggersOnStack()
						} else if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, nil); err != nil {
							t.Fatal(err)
						}
					}
					player.ManaPool().Add(core.Blue, 1)
					player.ManaPool().Add(core.Colorless, 5)
				}
				if a.offers+b.offers != 0 {
					t.Fatal("payment ran before resolution")
				}
				if context == "delayed" {
					g.ResolveStack()
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				}
				clone := g.Clone()
				clone.ResolveStack()
				if a.ManaPool().TotalMana() != 6 || b.ManaPool().TotalMana() != 6 {
					t.Fatal("clone shared payment mana")
				}
				g.ResolveStack()
				wantA, wantB := 20, 24
				if paidBody {
					wantA, wantB = 24, 20
				}
				if a.Life() != wantA || b.Life() != wantB || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 6 {
					t.Fatal("context lost player, amount or branch identity")
				}
			})
		}
	}
}

func TestRONResolutionManaAnnouncedX(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"X",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(ActionList([MayCost(PayManaX([ManaCostX],ValueX)),If(CostWasPaid,[GainLife(Integer(4))])]))])`), "x.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	c := factory()
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(99)
	g.ResolveStack()
	if a.Life() != 24 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("payment lost announced spell X")
	}
}

func TestRONResolutionManaInventory(t *testing.T) {
	files := fstest.MapFS{"a.ron": {Data: []byte(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([PlayerMayCost(Trigger_ThatPlayer,PayManaX([ManaCostX],ManaValueOfSpell(Trigger_ThatSpell))),Unless(CostWasPaid,[CounterSpell(Trigger_ThatSpell)])]))])`)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"PayManaX": "resolution_cost", "ManaValueOfSpell": "value", "Trigger_ThatSpell": "spell_reference"} {
		if !slices.ContainsFunc(report.Signatures, func(entry InventorySignature) bool { return entry.Name == name && entry.Role == role }) {
			t.Fatalf("missing role %s for %s", role, name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection published a partial registry")
	}
}

func TestRONResolutionManaSpellTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([PlayerMayCost(ControllerOfSpell(Ref_TargetSpell),PayManaX([ManaCostX],ManaValueOfSpell(Ref_TargetSpell))),Unless(CostWasPaid,[CounterSpell(Ref_TargetSpell)])])))`)
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	first := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEB", "Lightning Bolt"), b.PlayerID())
	second := castRONSpell(t, g, a.BasePlayer, mage.NewInstant("Other spell", "{5}", mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
	for _, obj := range []*mage.StackObject{first, second} {
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		castRONSpell(t, g, b.BasePlayer, card, obj.ID)
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 3)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if g.GetStack().FindByID(second.ID) == nil || a.ManaPool().TotalMana() != 3 {
		t.Fatal("clone changed the original target")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(second.ID) != nil || g.GetStack().FindByID(first.ID) == nil || a.ManaPool().TotalMana() != 3 {
		t.Fatal("unaffordable target tax changed another spell")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(first.ID) == nil || a.ManaPool().TotalMana() != 2 || a.offers != 1 || b.offers != 0 {
		t.Fatal("target instances shared amounts or players")
	}
}
