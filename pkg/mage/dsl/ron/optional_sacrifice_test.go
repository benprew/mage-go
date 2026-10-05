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

type optionalSacrificePlayer struct {
	*ronSacrificeChooser
	accept bool
	offers int
}

func (p *optionalSacrificePlayer) ChooseMayAbility(string) bool {
	p.offers++
	return p.accept
}

var optionalSacrificeCards = []struct {
	set, name, payment string
	damage             int
}{
	{"LEG", "Elder Spawn", "Island", 6},
	{"ATQ", "Yawgmoth Demon", "Sol Ring", 2},
}

func TestRONOptionalSacrificeCards(t *testing.T) {
	for _, tc := range optionalSacrificeCards {
		for _, scenario := range []string{"pay", "decline", "missing", "opponent", "wrong-type", "payment-phased", "invalid-choice", "stale-choice", "departed", "returned", "source-phased", "control", "prevent", "source-disabled", "unpaid-departed", "unpaid-returned", "unpaid-phased", "unpaid-control"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: scenario != "decline" && !strings.HasPrefix(scenario, "unpaid-")}
				b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
				g := mage.NewGame(a, b)
				g.SetTurn(2)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				payment := repositoryCard(t, "LEB", tc.payment)
				if scenario == "wrong-type" {
					payment = repositoryCard(t, "LEB", "Forest")
				}
				payment.SetOwner(b.PlayerID())
				controller := a.PlayerID()
				if scenario == "opponent" {
					controller = b.PlayerID()
				}
				victim := g.PutOnBattlefield(payment, controller)
				a.choice = victim.ID()
				g.AddRegenerationShield(victim.ID())
				if scenario == "missing" {
					g.RemoveFromBattlefield(victim)
				}
				if scenario == "payment-phased" {
					g.PhaseOut(victim.ID())
				}
				if scenario == "invalid-choice" {
					a.invalid = source
				}
				if scenario == "stale-choice" {
					a.invalid = victim
					g.RemoveFromBattlefield(victim)
					g.PutOnBattlefield(payment, a.PlayerID())
				}
				if scenario == "source-disabled" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				queueRONUpkeep(g, a)
				if scenario == "source-disabled" {
					if g.StackSize() != 0 {
						t.Fatal("removed ability triggered")
					}
					return
				}
				if g.StackSize() != 1 || a.offers != 0 || g.FindPermanent(victim.ID()) == nil && scenario != "missing" && scenario != "payment-phased" {
					t.Fatal("payment occurred before resolution")
				}
				paid := scenario == "pay" || scenario == "departed" || scenario == "returned" || scenario == "source-phased" || scenario == "control"
				switch scenario {
				case "departed", "returned", "unpaid-departed", "unpaid-returned":
					g.BouncePermanentToHand(source)
					if scenario == "returned" || scenario == "unpaid-returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "source-phased", "unpaid-phased":
					g.PhaseOut(source.ID())
				case "control", "unpaid-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "prevent":
					a.accept = false
					g.AddPreventionShield(a.PlayerID(), 1)
				}
				g.ResolveTopOfStack()
				wantLife := 20
				if !paid {
					wantLife -= tc.damage
				}
				if scenario == "prevent" {
					wantLife++
				}
				if a.Life() != wantLife || b.Life() != 20 {
					t.Fatalf("life %d/%d, want %d/20", a.Life(), b.Life(), wantLife)
				}
				if paid {
					if g.FindPermanent(victim.ID()) != nil || len(b.Graveyard()) != 1 {
						t.Fatal("sacrifice did not go to owner or regeneration replaced it")
					}
				} else if scenario != "missing" && scenario != "payment-phased" && g.FindPermanent(victim.ID()) == nil {
					t.Fatal("failed payment consumed a permanent")
				}
				present := tc.name == "Yawgmoth Demon" || paid || scenario == "unpaid-returned" || scenario == "unpaid-control"
				if scenario == "departed" || scenario == "source-phased" || scenario == "unpaid-departed" || scenario == "unpaid-phased" {
					present = false
				}
				if (g.FindPermanent(source.ID()) != nil) != present {
					t.Fatal("wrong source survival")
				}
				if tc.name == "Yawgmoth Demon" && !paid && present && scenario != "unpaid-returned" && !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("unpaid branch did not tap source")
				}
				if scenario == "unpaid-returned" && g.FindPermanent(source.ID()).Tapped {
					t.Fatal("old trigger tapped the returned source")
				}
				wantOffers := 1
				if scenario == "missing" || scenario == "opponent" || scenario == "wrong-type" || scenario == "payment-phased" {
					wantOffers = 0
				}
				if a.offers != wantOffers || b.offers != 0 {
					t.Fatalf("offers %d/%d, want %d/0", a.offers, b.offers, wantOffers)
				}
			})
		}
	}
}

func TestRONOptionalSacrificeBranchesAndScope(t *testing.T) {
	for _, condition := range []string{"If(CostWasPaid", "Unless(CostWasPaid", "If(CostWasntPaid", "Unless(CostWasntPaid"} {
		for _, accept := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", condition, accept), func(t *testing.T) {
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), ActionList([MayCost(SacrificeAPermanent(IsCardtype(Artifact))), `+condition+`, [GainLife(Integer(3))])]))`)
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: accept}
				g := mage.NewGame(a, mage.NewBasePlayer("B"))
				g.SetStep(core.PrecombatMain)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				if err := mage.ValidateCardDefinition(source.Name(), source.Card); err != nil {
					t.Fatal(err)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				want := 20
				positive := condition == "If(CostWasPaid" || condition == "Unless(CostWasntPaid"
				if positive == accept {
					want = 23
				}
				if a.Life() != want || (g.FindPermanent(source.ID()) == nil) != accept {
					t.Fatal("branch polarity or self payment")
				}
			})
		}
	}
	for _, actions := range []string{
		`MayCost(SacrificeAPermanent)`,
		`MayCost(SacrificeAPermanent()), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(IsCardtype(Artifact), IsTapped)), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(And([]))), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(SinglePermanent(Ref_TargetPermanent))), Unless(CostWasPaid, [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(IsCardtype(Artifact))), If(CostWasntPaid(), [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(IsCardtype(Artifact))), If(Unknown, [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(IsCardtype(Artifact))), GainLife(Integer(1)), If(CostWasntPaid, [GainLife(Integer(1))])`,
		`MayCost(SacrificeAPermanent(IsCardtype(Artifact))), If(CostWasntPaid, [GainLife(Integer(1))]), If(CostWasntPaid, [GainLife(Integer(1))])`,
		`If(CostWasntPaid, [GainLife(Integer(1))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + actions + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONOptionalSacrificeNativeLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range optionalSacrificeCards {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				files[tc.name+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("missing complete card")
	}
	for _, tc := range optionalSacrificeCards {
		g, a, _ := ronGame()
		card, err := registry.CreateCard(tc.name)
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("unpaid cast accepted")
		}
		a.RemoveFromHand(card.ID())
		castRONSpell(t, g, a, card)
		g.ResolveStack()
		if g.FindPermanent(card.ID()) == nil {
			t.Fatal("paid cast failed")
		}
		queueRONUpkeep(g, a)
		g.ResolveStack()
		if a.Life() != 20-tc.damage {
			t.Fatal("loaded card did not execute")
		}
	}
}

func TestRONOptionalSacrificeInstancesCopiesAndClones(t *testing.T) {
	for _, tc := range optionalSacrificeCards {
		t.Run(tc.name, func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
			b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: false}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			first := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			victim := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.payment), a.PlayerID())
			a.choice = victim.ID()
			queueRONUpkeep(g, a)
			if g.StackSize() != 2 {
				t.Fatal("independent sources did not trigger")
			}
			original := g.StackPeek()
			copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
			if copied == nil {
				t.Fatal("trigger copy failed")
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			if b.Life() != 20-tc.damage || a.Life() != 20 || g.FindPermanent(victim.ID()) == nil {
				t.Fatal("copy changed payer or original cost")
			}
			g.ResolveTopOfStack()
			if a.Life() != 20 || g.FindPermanent(victim.ID()) != nil {
				t.Fatal("first payment did not succeed")
			}
			g.ResolveTopOfStack()
			if a.Life() != 20-tc.damage || a.offers != 1 || a.choices != 1 {
				t.Fatal("second payment shared first result")
			}
			for _, p := range []*mage.Permanent{first, second} {
				current := g.FindPermanent(p.ID())
				if tc.name == "Elder Spawn" {
					if (current != nil) != (p.ID() == original.SourceID) {
						t.Fatal("copy sacrificed a source it did not control or original used wrong source")
					}
				} else if current == nil || !current.Tapped {
					t.Fatal("unpaid branches did not tap their respective source")
				}
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(victim.ID()) == nil {
				t.Fatal("original changed clone")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20-2*tc.damage || clone.GetPlayer(b.PlayerID()).Life() != 20-tc.damage || clone.FindPermanent(victim.ID()) == nil {
				t.Fatal("clone lost separate payment continuations")
			}
		})
	}
}

func TestRONOptionalSacrificeExplicitPlayerAndNestedResults(t *testing.T) {
	a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
	b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerMayCost(Ref_TargetPlayer, SacrificeAPermanent(IsCardtype(Artifact))), If(CostWasPaid, [MayCost(SacrificeAPermanent(IsCardtype(Land))), If(CostWasntPaid, [GainLife(Integer(3))])]), MayCost(SacrificeAPermanent(IsCardtype(Artifact))), Unless(CostWasntPaid, [GainLife(Integer(2))])])))`)()
	artifact := g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), b.PlayerID())
	castRONSpell(t, g, a.BasePlayer, card, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(artifact.ID()) != nil || a.Life() != 23 || b.Life() != 20 || a.offers != 0 || b.offers != 1 {
		t.Fatal("nested or adjacent results escaped their payment or used wrong payer")
	}
}

func TestRONOptionalSacrificeOtherCardRules(t *testing.T) {
	for _, tc := range optionalSacrificeCards {
		g, a, b := ronGame()
		source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		red := g.PutOnBattlefield(mage.NewCreature("Red blocker", "{R}", 7, 6, mage.WithKeyword(core.Flying)), b.PlayerID())
		blue := g.PutOnBattlefield(mage.NewCreature("Blue blocker", "{U}", 7, 6), b.PlayerID())
		if tc.name == "Elder Spawn" {
			if mage.CanBlock(red, source, g) || !mage.CanBlock(blue, source, g) {
				t.Fatal("red blocking restriction")
			}
			clone := g.Clone()
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			if !mage.CanBlock(g.FindPermanent(red.ID()), g.FindPermanent(source.ID()), g) || mage.CanBlock(clone.FindPermanent(red.ID()), clone.FindPermanent(source.ID()), clone) {
				t.Fatal("restriction lifecycle or clone isolation")
			}
		} else {
			if !source.HasKeyword(core.Flying) || !source.HasKeyword(core.FirstStrike) || mage.CanBlock(blue, source, g) || !mage.CanBlock(red, source, g) {
				t.Fatal("flying or first strike missing")
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: red.ID(), AttackerID: source.ID()}})
			if !g.GetCombat().IsBlocking(red.ID()) {
				t.Fatal("flying blocker was not declared")
			}
			g.ExecuteCombatDamage()
			if g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Damage != 0 || g.FindPermanent(red.ID()) != nil {
				t.Fatal("first strike did not kill blocker before normal damage")
			}
		}
	}
}

func TestRONOptionalSacrificeInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(SacrificeAPermanent(IsCardtype(Artifact))),If(CostWasntPaid,[GainLife(Integer(1))])]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for name, role := range map[string]string{"SacrificeAPermanent": "resolution_cost", "IsCardtype": "permanent_filter", "CostWasntPaid": "condition"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONOptionalSacrificeDetectionAndCurrentFilters(t *testing.T) {
	for _, tc := range optionalSacrificeCards {
		t.Run(tc.name, func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
			b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			phased := g.PhaseOut(source.ID())
			queueRONUpkeep(g, a)
			if g.StackSize() != 0 {
				t.Fatal("phased source triggered")
			}
			g.PhaseIn(phased)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			queueRONUpkeep(g, a)
			if g.StackSize() != 0 {
				t.Fatal("former controller upkeep triggered")
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 1 {
				t.Fatal("new controller upkeep did not trigger")
			}
			payment := g.PutOnBattlefield(repositoryCard(t, "LEB", tc.payment), b.PlayerID())
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			g.ResolveStack()
			if g.FindPermanent(payment.ID()) != nil || b.offers != 1 || a.offers != 0 || b.Life() != 20 {
				t.Fatal("pending trigger lost payment or controller after ability removal")
			}
		})
	}
	a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.SetStep(core.PrecombatMain)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(SacrificeAPermanent(And([IsCardtype(Land),Or([IsLandType(Island),IsLandType(Swamp)]),IsTapped]))),If(CostWasntPaid,[GainLife(Integer(2))])]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.TapPermanent(island)
	g.ResolveStack()
	if g.FindPermanent(island.ID()) != nil || a.offers != 1 || a.Life() != 20 {
		t.Fatal("payment did not read current recursive filter at resolution")
	}
}

func TestRONOptionalSacrificePolarityRetainsManaPayments(t *testing.T) {
	for _, branch := range []string{"If", "Unless"} {
		for _, condition := range []string{"CostWasPaid", "CostWasntPaid"} {
			for _, accept := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%t", branch, condition, accept), func(t *testing.T) {
					a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
					g := mage.NewGame(a, mage.NewBasePlayer("B"))
					g.SetStep(core.PrecombatMain)
					card := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([MayCost(PayMana([])), `+branch+`(`+condition+`,[GainLife(Integer(3))])]))`)()
					castRONSpell(t, g, a.BasePlayer, card)
					g.ResolveStack()
					want := 20
					if ((branch == "If") == (condition == "CostWasPaid")) == accept {
						want = 23
					}
					if a.Life() != want || a.offers != 1 {
						t.Fatal("fixed mana branch polarity changed")
					}
				})
			}
		}
	}
}
