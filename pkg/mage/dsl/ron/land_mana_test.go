package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLandManaRealCards(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Manabarbs", "Wild Growth"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			t.Run(name+"/"+set, func(t *testing.T) {
				g, a, b := ronGame()
				land := repositoryCard(t, "LEB", "Island")
				land.SetOwner(b.PlayerID())
				host := g.PutOnBattlefield(land, b.PlayerID())
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				var targets []uuid.UUID
				if name == "Wild Growth" {
					targets = []uuid.UUID{host.ID()}
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil {
					t.Fatal("unpaid cast succeeded")
				}
				a.ManaPool().Add(core.Red, 4)
				a.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if err := g.TapForMana(b.PlayerID(), host.ID()); err != nil {
					t.Fatal(err)
				}
				wantBlue, wantGreen := 1, 0
				if name == "Mana Flare" {
					wantBlue = 2
				}
				if name == "Wild Growth" {
					wantGreen = 1
				}
				if b.ManaPool().Count(core.Blue) != wantBlue || b.ManaPool().Count(core.Green) != wantGreen || b.Life() != 20 || g.StackSize() != 0 {
					t.Fatal("wrong recipient, production, or timing")
				}
				g.PutTriggersOnStack()
				wantStack := 0
				if name == "Manabarbs" {
					wantStack = 1
				}
				if g.StackSize() != wantStack {
					t.Fatalf("stack = %d, want %d", g.StackSize(), wantStack)
				}
				g.ResolveStack()
				wantLife := 20
				if name == "Manabarbs" {
					wantLife = 19
				}
				if b.Life() != wantLife || a.Life() != 20 {
					t.Fatal("damage used wrong player")
				}
			})
		}
	}
}

func TestRONLandManaLifecycle(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Manabarbs", "Wild Growth"} {
		for _, state := range []string{"live", "source-phased", "removed", "source-left", "host-returned", "host-control", "reattach", "clone", "tap-only", "nonland", "zero"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewLand("Host", mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.AddManaAmount(core.Blue, mage.Fixed(1)))))
				if state == "nonland" {
					card = mage.NewArtifact("Host", "", mage.WithManaAbility(core.Blue))
				}
				if state == "zero" {
					card = mage.NewLand("Host", mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.AddManaAmount(core.Blue, mage.Fixed(0)))))
				}
				host := g.PutOnBattlefield(card, b.PlayerID())
				source := g.PutOnBattlefield(repositoryCard(t, "LEB", name), a.PlayerID())
				if name == "Wild Growth" && state != "nonland" {
					g.Attach(source.ID(), host.ID())
				}
				wantBonus := 1
				actor := b.PlayerID()
				switch state {
				case "source-phased":
					g.PhaseOut(source.ID())
					wantBonus = 0
				case "removed":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
					wantBonus = 0
				case "source-left":
					g.RemoveFromBattlefield(source)
					wantBonus = 0
				case "host-returned":
					g.RemoveFromBattlefield(host)
					host = g.PutOnBattlefield(card, b.PlayerID())
					if name == "Wild Growth" {
						wantBonus = 0
					}
				case "host-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					actor = a.PlayerID()
				case "reattach":
					if name == "Wild Growth" {
						other := g.PutOnBattlefield(mage.NewLand("Other"), b.PlayerID())
						g.Attach(source.ID(), other.ID())
						wantBonus = 0
					}
				case "tap-only", "nonland", "zero":
					wantBonus = 0
				case "clone":
					clone := g.Clone()
					if err := clone.ActivateAbilityByIndex(actor, host.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					clone.PutTriggersOnStack()
					clone.ResolveStack()
					if g.FindPermanent(host.ID()).Tapped || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || b.Life() != 20 {
						t.Fatal("clone shared mana, life, or tap state")
					}
				}
				if state == "tap-only" {
					g.TapPermanent(host)
				} else if err := g.ActivateAbilityByIndex(actor, host.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
				player := g.GetPlayer(actor)
				wantBlue, wantGreen, wantLife := 1, 0, 20
				if state == "tap-only" || state == "zero" {
					wantBlue = 0
				}
				if name == "Mana Flare" {
					wantBlue += wantBonus
				}
				if name == "Wild Growth" {
					wantGreen = wantBonus
				}
				if name == "Manabarbs" {
					wantLife -= wantBonus
				}
				if player.ManaPool().Count(core.Blue) != wantBlue || player.ManaPool().Count(core.Green) != wantGreen || player.Life() != wantLife {
					t.Fatalf("blue/green/life = %d/%d/%d; want %d/%d/%d", player.ManaPool().Count(core.Blue), player.ManaPool().Count(core.Green), player.Life(), wantBlue, wantGreen, wantLife)
				}
			})
		}
	}
}

type ronProducedTypePlayer struct {
	*mage.BasePlayer
	choice core.Color
	calls  int
}

func (p *ronProducedTypePlayer) ChooseManaColor(string) core.Color { p.calls++; return p.choice }

func TestRONLandManaProducedTypesAndInstances(t *testing.T) {
	for _, color := range []core.Color{core.Red, core.Blue, core.Green, core.Colorless} {
		t.Run(fmt.Sprint(color), func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &ronProducedTypePlayer{BasePlayer: mage.NewBasePlayer("B"), choice: color}
			g := mage.NewGame(a, b)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Mana Flare"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Mana Flare"), b.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Mixed", mage.WithAbility(mage.Activate(mage.Costs(mage.TapCost(mage.This)), mage.Sequence(mage.AddManaAmount(core.Blue, mage.Fixed(2)), mage.AddManaAmount(core.Red, mage.Fixed(1)))))), b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			wantBlue, wantRed := 4, 1
			if color == core.Red {
				wantBlue, wantRed = 2, 3
			}
			if b.ManaPool().Count(core.Blue) != wantBlue || b.ManaPool().Count(core.Red) != wantRed || b.calls != 2 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
				t.Fatal("type choice, fallback, amount, instance, or recipient mismatch")
			}
		})
	}
}

func TestRONLandManaPendingCopiesAndLKI(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Manabarbs"), a.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(land)
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(source.Card, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("pending trigger or copy lost")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone lost captured actor or shared life")
	}
	g.ResolveStack()
	if b.Life() != 18 || a.Life() != 20 {
		t.Fatal("copy changed actor after source and land departure")
	}
}

func TestRONLandManaRollbackAndLegality(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Wild Growth"} {
		g, a, b := ronGame()
		land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
		source := g.PutOnBattlefield(repositoryCard(t, "LEB", name), b.PlayerID())
		if name == "Wild Growth" {
			g.Attach(source.ID(), land.ID())
		}
		spell := mage.NewSorcery("Payment", "{3}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
			t.Fatal("unaffordable payment succeeded")
		}
		g.PutTriggersOnStack()
		if g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
			t.Fatal("failed payment leaked state")
		}
	}
	for _, state := range []string{"nonland", "shroud", "phased"} {
		g, a, b := ronGame()
		host := g.PutOnBattlefield(mage.NewLand("Host"), b.PlayerID())
		if state == "nonland" {
			host = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 2), b.PlayerID())
		}
		if state == "shroud" {
			host = g.PutOnBattlefield(mage.NewLand("Host", mage.WithKeyword(core.Shroud)), b.PlayerID())
		}
		if state == "phased" {
			g.PhaseOut(host.ID())
		}
		aura := repositoryCard(t, "LEB", "Wild Growth")
		aura.SetOwner(a.PlayerID())
		a.AddToHand(aura)
		a.ManaPool().Add(core.Green, 1)
		if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
			t.Fatalf("accepted %s", state)
		}
		if a.ManaPool().Count(core.Green) != 1 || g.StackSize() != 0 || len(a.Hand()) != 1 {
			t.Fatal("illegal Aura consumed payment")
		}
	}
}

func TestRONLandManaInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),ActionList([PlayerAction(Trigger_ThatPlayer,AddMana(Trigger_AnyManaTypeProduced))]))`
	for _, rule := range []string{
		strings.Replace(valid, "AnyPlayer,IsCardtype(Land)", "AnyPlayer", 1),
		strings.Replace(valid, "AnyPlayer,IsCardtype(Land)", "AnyPlayer,IsCardtype(Land),AnyPermanent", 1),
		strings.Replace(valid, "IsCardtype(Land)", "IsCardtype(Land())", 1),
		strings.Replace(valid, "AnyPlayer,", "AnyPlayer(),", 1),
		strings.Replace(valid, "AnyPlayer,", "Opponent,", 1),
		strings.Replace(valid, "IsCardtype(Land)", "IsCardtype(Creature)", 1),
		strings.Replace(valid, "IsCardtype(Land)", "IsLandType(Mountain)", 1),
		strings.Replace(valid, "Trigger_AnyManaTypeProduced", "Trigger_AnyManaTypeProduced()", 1),
		strings.Replace(valid, "Trigger_ThatPlayer", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "Trigger_ThatPlayer", "Ref_TargetPlayer2", 1),
		strings.Replace(valid, "Trigger_ThatPlayer", "Trigger_ThatPermanent", 1),
		strings.Replace(valid, "Trigger_AnyManaTypeProduced", "AnyManaColor", 1),
		strings.Replace(valid, "Trigger_AnyManaTypeProduced", "ManaProduceG", 1),
		strings.Replace(valid, "Trigger_ThatPlayer", "You", 1),
		valid + `,Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,AddMana(ManaProduceG))]))`,
		valid + `,Activated(PayMana([]),ActionList([AddMana(Trigger_AnyManaTypeProduced)]))`,
		`TriggerA(WhenAPermanentIsTappedForMana(SinglePermanent(ThisPermanent)),ActionList([AddMana(ManaProduceG)]))`,
		`TriggerA(WhenAPermanentIsTappedForMana(SinglePermanent(HostPermanent)),ActionList([AddMana(ManaProduceG)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONLandManaTypedEvents(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1))),PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),GainLife(Integer(2)))]))`)
	for _, state := range []string{"control", "return", "phased", "phase-cycle", "departed"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(first, a.PlayerID())
			g.PutOnBattlefield(second, b.PlayerID())
			card := repositoryCard(t, "LEB", "Island")
			land := g.PutOnBattlefield(card, b.PlayerID())
			if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: land.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "return":
				g.RemoveFromBattlefield(land)
				g.PutOnBattlefield(card, a.PlayerID())
			case "departed":
				g.RemoveFromBattlefield(land)
			case "phased":
				g.PhaseOut(land.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(land.ID()))
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			wantA, wantB := 20, 26
			if state == "control" {
				wantA, wantB = 24, 22
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life = %d/%d, want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONLandManaInventory(t *testing.T) {
	src := `Card(Name:"Mana",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,IsCardtype(Land)),ActionList([PlayerAction(Trigger_ThatPlayer,AddMana(Trigger_AnyManaTypeProduced))]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"AnyPlayer": "player_filter", "IsCardtype": "permanent_filter", "Trigger_ThatPlayer": "player_reference", "AddMana": "player_action", "Trigger_AnyManaTypeProduced": "mana_production"}
	for _, entry := range report.Signatures {
		if role, ok := want[entry.Name]; ok {
			if entry.Role != role {
				t.Fatalf("%s role = %s, want %s", entry.Name, entry.Role, role)
			}
			delete(want, entry.Name)
		}
	}
	if len(want) != 0 || report.Compiled != 1 {
		t.Fatalf("missing roles %v; compiled %d", want, report.Compiled)
	}
}

func TestRONLandManaInvalidBonusShapes(t *testing.T) {
	valid := `TriggerA(WhenAPermanentIsTappedForMana(SinglePermanent(HostPermanent)),ActionList([PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),AddMana(ManaProduceG))]))`
	for _, rule := range []string{
		strings.Replace(valid, "ManaProduceG", "And([ManaProduceG,ManaProduceU])", 1),
		strings.Replace(valid, "ManaProduceG", "And([])", 1),
		strings.Replace(valid, "ManaProduceG", "AnyManaColor", 1),
		strings.Replace(valid, "ManaProduceG", "Trigger_AnyManaTypeProduced", 1),
		strings.Replace(valid, "ControllerOfPermanent(Trigger_ThatPermanent)", "You", 1),
		strings.Replace(valid, "Trigger_ThatPermanent", "HostPermanent", 1),
		strings.Replace(valid, "AddMana(ManaProduceG)", "GainLife(Integer(1))", 1),
		strings.Replace(valid, "AddMana(ManaProduceG)", "AddManaRepeated(ManaProduceG,Integer(2))", 1),
		strings.Replace(valid, "SinglePermanent(HostPermanent)", "SinglePermanent(HostPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(HostPermanent)", "SinglePermanent(HostPermanent())", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid bonus",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Land)),`+rule+`])`), "invalid-bonus.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted unsupported bonus %s", rule)
		}
	}
}

func TestRONLandManaAutomaticPayment(t *testing.T) {
	for _, name := range []string{"Mana Flare", "Wild Growth"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", name), b.PlayerID())
			if name == "Wild Growth" {
				g.Attach(source.ID(), land.ID())
			}
			spell := mage.NewSorcery("Payment", "{2}", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if !g.CanAfford(a.PlayerID(), core.ParseManaCost("{2}"), nil) {
				t.Fatal("solver missed triggered mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(land.ID()).Tapped {
				t.Fatal("payment did not use exactly two mana")
			}
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("paid spell did not resolve")
			}
		})
	}
}

func TestRONLandManaBonusContexts(t *testing.T) {
	for _, tc := range []struct {
		production string
		color      core.Color
	}{
		{"ManaProduceW", core.White}, {"ManaProduceU", core.Blue}, {"ManaProduceB", core.Black}, {"ManaProduceR", core.Red}, {"ManaProduceG", core.Green}, {"ManaProduceC", core.Colorless}, {"And([ManaProduceG])", core.Green},
	} {
		t.Run(tc.production, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Bonus",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Land)),TriggerA(WhenAPermanentIsTappedForMana(SinglePermanent(HostPermanent)),ActionList([PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),AddMana(`+tc.production+`))]))])`), "bonus.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			land := g.PutOnBattlefield(mage.NewLand("Land", mage.WithManaAbility(core.Colorless)), b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), land.ID())
			if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			want := 1
			if tc.color == core.Colorless {
				want = 2
			}
			if b.ManaPool().Count(tc.color) != want || b.ManaPool().TotalMana() != 2 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("fixed bonus used wrong color or player")
			}
		})
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Mana Flare"), a.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Colorless", mage.WithManaAbility(core.Colorless)), b.PlayerID())
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(core.Colorless) != 2 {
		t.Fatal("colorless type was lost")
	}
}

func TestRONLandManaIndependentEventsAndDamage(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Manabarbs"), a.PlayerID())
	source.GrantBaseAttr(core.Lifelink)
	for _, player := range []*mage.BasePlayer{a, b} {
		land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), player.PlayerID())
		if err := g.TapForMana(player.PlayerID(), land.ID()); err != nil {
			t.Fatal(err)
		}
	}
	g.AddPreventionShield(b.PlayerID(), 1)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent events were lost")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("native prevention, lifelink, or actor identity failed")
	}
}
