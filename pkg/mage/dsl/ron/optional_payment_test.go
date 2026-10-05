package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPaymentPlayer struct {
	*mage.BasePlayer
	accept bool
	offers int
}

func (p *ronPaymentPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONOptionalPaymentRealCards(t *testing.T) {
	for _, name := range []string{"Force Spike", "Vodalian Mage"} {
		for _, choice := range []string{"pay", "decline", "unaffordable", "mana-source", "departed", "phased", "illegal", "uncounterable", "controller"} {
			t.Run(name+"/"+choice, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "decline"}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				victim := repositoryCard(t, "LEB", "Sol Ring")
				target := castRONSpell(t, g, a.BasePlayer, victim)
				a.ManaPool().Clear()
				if choice != "unaffordable" && choice != "mana-source" {
					a.ManaPool().Add(core.Colorless, 1)
				}
				var landID uuid.UUID
				if choice == "mana-source" {
					land := repositoryCard(t, "LEB", "Forest")
					land.SetOwner(a.PlayerID())
					landID = g.PutOnBattlefield(land, a.PlayerID()).ID()
				}
				set := "LEG"
				if name == "Vodalian Mage" {
					set = "FEM"
				}
				counter := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, counter); err != nil {
					t.Fatal(err)
				}
				if name == "Force Spike" {
					counter.SetOwner(b.PlayerID())
					b.AddToHand(counter)
					if err := g.CastSpellByID(b.PlayerID(), counter.ID(), []uuid.UUID{target.ID}, 0); err == nil {
						t.Fatal("unpaid cast succeeded")
					}
					b.ManaPool().Add(core.Blue, 1)
					if err := g.CastSpellByID(b.PlayerID(), counter.ID(), []uuid.UUID{target.ID}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					counter.SetOwner(b.PlayerID())
					source := g.PutOnBattlefield(counter, b.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID}); err == nil {
						t.Fatal("unpaid activation succeeded")
					}
					if source.Tapped || g.StackSize() != 1 {
						t.Fatal("failed activation changed state")
					}
					b.ManaPool().Add(core.Blue, 1)
					if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(source.ID()).Tapped {
						t.Fatal("tap cost not paid")
					}
					if choice == "departed" {
						g.DestroyPermanent(g.FindPermanent(source.ID()))
					}
					if choice == "phased" {
						g.PhaseOut(source.ID())
					}
				}
				b.ManaPool().Add(core.Colorless, 3)
				if choice == "controller" {
					target.Controller = b.PlayerID()
				}
				if choice == "uncounterable" {
					mage.WithUncounterable()(target.Card.(*mage.BaseCard))
				}
				if choice == "illegal" {
					g.GetStack().RemoveByID(target.ID)
				}
				if a.offers != 0 || b.offers != 0 {
					t.Fatal("payment offered before resolution")
				}
				beforeA, beforeB := a.ManaPool().TotalMana(), b.ManaPool().TotalMana()
				g.ResolveTopOfStack()
				remaining := g.GetStack().FindByID(target.ID) != nil
				if choice == "decline" || choice == "unaffordable" {
					if remaining || !slices.ContainsFunc(g.GetPlayer(a.PlayerID()).Graveyard(), func(c mage.Card) bool { return c.ID() == victim.ID() }) {
						t.Fatal("unpaid spell not countered")
					}
					if g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != beforeA {
						t.Fatal("unpaid cost consumed mana")
					}
				} else if choice != "illegal" && !remaining {
					t.Fatal("paid spell countered")
				}
				expectedOffers := 1
				if choice == "illegal" || choice == "unaffordable" {
					expectedOffers = 0
				}
				if a.offers+b.offers != expectedOffers {
					t.Fatalf("offers A=%d B=%d", a.offers, b.offers)
				}
				if choice == "controller" {
					if a.offers != 0 || b.offers != 1 || g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != beforeB-1 {
						t.Fatal("wrong spell controller paid")
					}
				} else if choice != "illegal" && choice != "unaffordable" && choice != "decline" {
					if a.offers != 1 || b.offers != 0 {
						t.Fatal("owner or ability controller used as payer")
					}
					if choice == "mana-source" {
						if !g.FindPermanent(landID).Tapped {
							t.Fatal("available mana source not used")
						}
					} else if g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != beforeA-1 {
						t.Fatal("payment amount incorrect")
					}
				}
			})
		}
	}
}

func TestRONOptionalPaymentInvalid(t *testing.T) {
	pair := `PlayerMayCost(ControllerOfSpell(Ref_TargetSpell), PayMana([ManaCostGeneric(1)])), Unless(CostWasPaid, [CounterSpell(Ref_TargetSpell)])`
	for _, actions := range []string{
		`Unless(CostWasPaid, [CounterSpell(Ref_TargetSpell)])`,
		`PlayerMayCost(ControllerOfTargetSpell, PayMana([]))`,
		strings.Replace(pair, "CostWasPaid", "CostWasPaid()", 1),
		strings.Replace(pair, "PayMana([ManaCostGeneric(1)])", "PayMana([ManaCostX])", 1),
		strings.Replace(pair, "PayMana([ManaCostGeneric(1)])", "TapPermanent(ThisPermanent)", 1),
		strings.Replace(pair, "Ref_TargetSpell)", "Ref_TargetPermanent)", 1),
		strings.Replace(pair, "Ref_TargetSpell)", "Ref_TargetSpell1)", 1),
		strings.Replace(pair, "Unless(CostWasPaid,", "GainLife(Integer(1)), Unless(CostWasPaid,", 1),
		pair + `, Unless(CostWasPaid, [CounterSpell(Ref_TargetSpell)])`,
		strings.Replace(pair, "[CounterSpell(Ref_TargetSpell)]", "[]", 1),
		strings.Replace(pair, "[CounterSpell(Ref_TargetSpell)]", "[Unless(CostWasPaid, [CounterSpell(Ref_TargetSpell)])]", 1),
	} {
		source := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetSpell(AnySpell)], ActionList([%s])))])`, actions)
		nodes, err := Parse(strings.NewReader(source), "invalid-payment.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
	for _, ref := range []string{"ControllerOfTargetSpell", "ControllerOfSpell(Ref_TargetSpell)", "ControllerOfSpell(Ref_TargetPermanent)", "ControllerOfTargetSpell()"} {
		source := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([PlayerMayCost(%s, PayMana([])), Unless(CostWasPaid, [GainLife(Integer(1))])]))])`, ref)
		nodes, err := Parse(strings.NewReader(source), "invalid-scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted unbound %s", ref)
		}
	}
}

func TestRONOptionalPaymentInventoryRoles(t *testing.T) {
	source := `Card(Name: "Payment", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetSpell(AnySpell)], ActionList([PlayerMayCost(ControllerOfSpell(Ref_TargetSpell), PayMana([ManaCostGeneric(1)])), Unless(CostWasPaid, [CounterSpell(Ref_TargetSpell)])])))])`
	report, err := InspectFS(fstest.MapFS{"payment.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{
		"ControllerOfSpell": "player_reference",
		"Ref_TargetSpell":   "spell_reference",
		"PayMana":           "resolution_cost",
		"CostWasPaid":       "condition",
		"CounterSpell":      "action",
	} {
		if !slices.ContainsFunc(report.Signatures, func(entry InventorySignature) bool { return entry.Name == name && entry.Role == role }) {
			t.Fatalf("missing %s role for %s", role, name)
		}
	}
}

func TestRONOptionalPaymentCopiesClonesAndIdentity(t *testing.T) {
	for _, change := range []string{"copy", "recast", "uncounterable"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			firstCard, secondCard := repositoryCard(t, "LEB", "Lightning Bolt"), repositoryCard(t, "LEB", "Lightning Bolt")
			first := castRONSpell(t, g, a, firstCard, b.PlayerID())
			second := castRONSpell(t, g, a, secondCard, b.PlayerID())
			a.ManaPool().Clear()
			counter := castRONSpell(t, g, b, repositoryCard(t, "LEG", "Force Spike"), first.ID)
			clone := g.Clone()
			switch change {
			case "copy":
				copyObj := g.CopyStackObjectDirect(counter, b.PlayerID(), false)
				if copyObj == nil {
					t.Fatal("copy failed")
				}
				copyObj.Targets[0] = second.ID
				copyObj.TargetZones[second.ID] = core.ZoneStack
				g.ResolveTopOfStack()
				if g.GetStack().FindByID(first.ID) == nil || g.GetStack().FindByID(second.ID) != nil || counter.Targets[0] != first.ID {
					t.Fatal("copy changed original binding")
				}
			case "recast":
				g.GetStack().RemoveByID(first.ID)
				replacement := &mage.StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: a.PlayerID()}
				top := g.StackPeek()
				g.GetStack().RemoveByID(top.ID)
				g.PushStack(replacement)
				g.PushStack(top)
			default:
				mage.WithUncounterable()(first.Card.(*mage.BaseCard))
			}
			g.ResolveTopOfStack()
			if change == "uncounterable" {
				if g.GetStack().FindByID(first.ID) == nil {
					t.Fatal("uncounterable spell countered on refusal")
				}
			} else if g.GetStack().FindByID(first.ID) != nil {
				t.Fatal("unpaid original not countered")
			}
			if change == "recast" && !slices.ContainsFunc(g.GetStack().Objects(), func(o *mage.StackObject) bool { return o.SourceID == firstCard.ID() }) {
				t.Fatal("old target followed new cast")
			}
			if change != "copy" && g.GetStack().FindByID(second.ID) == nil {
				t.Fatal("unrelated target changed")
			}
			if change == "uncounterable" {
				return
			}
			clone.ResolveTopOfStack()
			if clone.GetStack().FindByID(first.ID) != nil || clone.GetStack().FindByID(second.ID) == nil {
				t.Fatal("clone shared target state")
			}
		})
	}
}

func TestRONOptionalPaymentIndependentContinuations(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			actions := `PlayerMayCost(You, PayMana([ManaCostGeneric(1)])), Unless(CostWasPaid, [GainLife(Integer(3))]), PlayerMayCost(You, PayMana([ManaCostGeneric(1)])), Unless(CostWasPaid, [GainLife(Integer(2))])`
			if nested {
				actions = `PlayerMayCost(You, PayMana([ManaCostW])), Unless(CostWasPaid, [PlayerMayCost(You, PayMana([ManaCostGeneric(1)])), Unless(CostWasPaid, [GainLife(Integer(3))])]), PlayerMayCost(You, PayMana([ManaCostGeneric(1)])), Unless(CostWasPaid, [GainLife(Integer(2))])`
			}
			factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([`+actions+`]))`)
			first, second := factory(), factory()
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
			}
			if first.Abilities()[0].(*mage.ActionDefinition).Effects()[0] == second.Abilities()[0].(*mage.ActionDefinition).Effects()[0] {
				t.Fatal("shared payment effect")
			}
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			a.ManaPool().Add(core.Colorless, 1)
			first.SetOwner(a.PlayerID())
			a.AddToHand(first)
			if err := g.CastSpellByID(a.PlayerID(), first.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			if a.offers != 1 || g.GetPlayer(a.PlayerID()).Life() != 22 || g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("independent or nested payment result leaked")
			}
		})
	}
}

func TestRONOptionalPaymentZeroCostAndRollback(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		g.SetStep(core.PrecombatMain)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([PlayerMayCost(You, PayMana([])), Unless(CostWasPaid, [GainLife(Integer(2))])]))`)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveTopOfStack()
		expected := 22
		if accept {
			expected = 20
		}
		if a.offers != 1 || g.GetPlayer(a.PlayerID()).Life() != expected {
			t.Fatal("zero cost was paid without a choice")
		}
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	g.SetStep(core.PrecombatMain)
	forest := repositoryCard(t, "LEB", "Forest")
	forest.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(forest, a.PlayerID())
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([PlayerMayCost(You, PayMana([ManaCostG, ManaCostU])), Unless(CostWasPaid, [GainLife(Integer(2))])]))`)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if a.offers != 0 || g.FindPermanent(source.ID()).Tapped || g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || g.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("unaffordable payment consumed a partial mana source")
	}
}
