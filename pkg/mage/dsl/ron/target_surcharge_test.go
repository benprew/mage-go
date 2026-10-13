package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronTargetSurcharge = `CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(1)])))`

func TestRONTargetSurchargeFireball(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "empty", "single", "zero", "zero-share", "partial", "all-illegal", "returned", "phase", "phase-cycle", "prevention", "copy"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := distributedDamageGame()
				card := repositoryCard(t, set, "Fireball")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 20), b.PlayerID())
				walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 20), b.PlayerID())
				ids := []uuid.UUID{first.ID(), walker.ID(), b.PlayerID()}
				x := 7
				switch state {
				case "empty":
					ids = nil
				case "single":
					ids = ids[:1]
				case "zero":
					x = 0
				case "zero-share":
					x = 1
				case "all-illegal":
					ids = ids[:2]
				}
				cost := x + max(0, len(ids)-1)
				a.ManaPool().Add(core.Red, 1)
				a.ManaPool().Add(core.Colorless, cost)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != x || a.prompts != 0 {
					t.Fatal("wrong payment, X or allocation choice")
				}
				share, firstShare, walkerShare, playerShare := 0, 0, 0, 0
				if len(ids) > 0 {
					share = x / len(ids)
					firstShare = share
				}
				if len(ids) > 1 {
					walkerShare = share
				}
				if len(ids) > 2 {
					playerShare = share
				}
				switch state {
				case "partial":
					first.GrantBaseAttr(core.Shroud)
					firstShare = 0
					walkerShare = x / 2
					playerShare = x / 2
				case "all-illegal":
					first.GrantBaseAttr(core.Shroud)
					walker.GrantBaseAttr(core.Shroud)
					firstShare = 0
					walkerShare = 0
				case "returned":
					g.BouncePermanentToHand(first)
					b.RemoveFromHand(first.Card.ID())
					first = g.PutOnBattlefield(first.Card, b.PlayerID())
					firstShare = 0
					walkerShare = x / 2
					playerShare = x / 2
				case "phase":
					g.PhaseOut(first.ID())
					firstShare = 0
					walkerShare = x / 2
					playerShare = x / 2
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(first.ID()))
				case "prevention":
					g.AddPreventionShield(first.ID(), 1)
					firstShare = max(0, firstShare-1)
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					firstShare *= 2
					walkerShare *= 2
					playerShare *= 2
				}
				g.SetXValue(99)
				clone := g.Clone()
				g.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.FindPermanent(walker.ID()).Counters[core.Loyalty] != 20 {
					t.Fatal("clone shared resolution")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if game.FindPermanentIncludingPhased(first.ID()).Damage != firstShare || game.FindPermanent(walker.ID()).Counters[core.Loyalty] != 20-walkerShare || game.GetPlayer(b.PlayerID()).Life() != 20-playerShare {
						t.Fatal("wrong damage or saved recipient")
					}
					if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 || game.StackSize() != 0 {
						t.Fatal("resolution changed locked payment")
					}
				}
			})
		}
	}
}

func TestRONTargetSurchargeRollback(t *testing.T) {
	for _, state := range []string{"underpaid", "duplicate", "illegal", "phased", "shroud", "timing"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := distributedDamageGame()
			card := repositoryCard(t, "LEA", "Fireball")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{b.PlayerID(), victim.ID()}
			mana := 4
			switch state {
			case "timing":
				g.SetStep(core.Upkeep)
			case "underpaid":
				mana = 3
			case "duplicate":
				ids[1] = ids[0]
			case "illegal":
				ids[1] = uuid.New()
			case "phased":
				g.PhaseOut(victim.ID())
			case "shroud":
				victim.GrantBaseAttr(core.Shroud)
			}
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, mana)
			events := 0
			for _, event := range []core.EventType{core.EvtSpellCast, core.EvtBecomesTarget, core.EvtTapped} {
				g.PutOnBattlefield(mage.NewEnchantment("Observer", "", mage.WithAbility(mage.NewTriggered(event, false, mage.GainLifeFor(mage.You, mage.Fixed(1))).SetCondition(func(_ *core.GameEvent, reader mage.GameReader, _, _ uuid.UUID) bool {
					if reader == g {
						events++
					}
					return false
				}))), a.PlayerID())
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 4); err == nil {
				t.Fatal("invalid cast succeeded")
			}
			if a.ManaPool().TotalMana() != mana+1 || len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() || g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 || events != 0 {
				t.Fatal("failed cast changed resources or events")
			}
		})
	}
}

func TestRONTargetSurchargeInstancesAndRetarget(t *testing.T) {
	g, a, b := distributedDamageGame()
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), b.PlayerID())
	for i, x := range []int{6, 3} {
		record := staticBaseRecord(t, "LEA", "Fireball")
		typeline := record.Fields["Typeline"]
		typeline.Fields["Cardtypes"] = Node{Kind: List, Args: []Node{{Kind: Symbol, Name: "Instant"}}}
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		ids := []uuid.UUID{victim.ID(), b.PlayerID()}
		if i == 1 {
			ids = ids[:1]
		}
		a.ManaPool().Add(core.Red, 1)
		a.ManaPool().Add(core.Colorless, x+max(0, len(ids)-1))
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
			t.Fatal(err)
		}
	}
	if g.StackSize() != 2 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong independent payment")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(victim.ID()).Damage != 3 {
		t.Fatal("second instance lost X")
	}
	a.targets = []uuid.UUID{a.PlayerID(), victim.ID()}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("retargeted copy failed")
	}
	g.ResolveTopOfStack()
	if a.Life() != 17 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 6 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("copy did not keep X and fresh targets without payment")
	}
	g.ResolveTopOfStack()
	if b.Life() != 17 || g.FindPermanent(victim.ID()).Damage != 9 {
		t.Fatal("original targets changed")
	}
}

func TestRONTargetSurchargeInvalid(t *testing.T) {
	for _, rule := range []string{
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst)`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst())`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(0)])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(-1)])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostR])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostX])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(1),ManaCostGeneric(1)])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(Integer(1))])))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayLife(Integer(1))))`,
		`CastEffect(AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(1)]),PayMana([])))`,
		ronTargetSurcharge + `,` + ronTargetSurcharge,
	} {
		t.Run(rule, func(t *testing.T) { targetSurchargeReject(t, "Sorcery", rule+`,SpellActions(`+evenUnionBody+`)`) })
	}
	targetSurchargeReject(t, "Artifact", ronTargetSurcharge)
	targetSurchargeReject(t, "Sorcery", ronTargetSurcharge+`,SpellActions(Modal_ChooseOne([ActionList([GainLife(Integer(1))])]))`)
	targetSurchargeReject(t, "Sorcery", ronTargetSurcharge+`,SpellActions(Targeted_Modal([Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget)]))]))`)
	targetSurchargeReject(t, "Sorcery", `SpellActions(ActionList([AdditionalCastingCostForEachTargetBeyondTheFirst(PayMana([ManaCostGeneric(1)]))]))`)
}

func targetSurchargeReject(t *testing.T, kind, rules string) {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:[]),ManaCost:[ManaCostX,ManaCostR],Rules:[`+rules+`])`), "target-surcharge-invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "target-surcharge-invalid.ron:") {
		t.Fatalf("expected positioned declaration error: %v", err)
	}
}

func TestRONTargetSurchargeInventory(t *testing.T) {
	record := staticBaseRecord(t, "LEA", "Fireball")
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Fireball", "Card", "card", "collection", false, entries)
	for name, role := range map[string]string{"AdditionalCastingCostForEachTargetBeyondTheFirst": "casting_restriction", "PayMana": "cost", "ManaCostGeneric": "metadata", "Ref_AnyTargets": "recipient_collection_reference"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lost %s role", name, role)
		}
	}
}

func TestRONTargetSurchargeOrdinaryShapes(t *testing.T) {
	for _, shape := range []string{"none", "single", "groups", "distributed"} {
		for _, route := range []string{"hand", "alternate"} {
			t.Run(shape+"/"+route, func(t *testing.T) {
				g, a, b := distributedDamageGame()
				body := `ActionList([GainLife(Integer(1))])`
				ids := []uuid.UUID(nil)
				count := 0
				switch shape {
				case "single":
					body = `Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget)]))`
					ids = []uuid.UUID{b.PlayerID()}
					count = 1
				case "groups":
					body = `Targeted([AnyTarget,AnyTarget],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Ref_AnyTarget1),DamageToRecipients(Integer(2),Ref_AnyTarget2)])]))`
					ids = []uuid.UUID{b.PlayerID(), b.PlayerID()}
					count = 2
				case "distributed":
					body = distributedDamageBody
					ids = []uuid.UUID{a.PlayerID(), b.PlayerID()}
					count = 2
					a.choose = func(chosen []uuid.UUID, total int, _ *mage.Game) map[uuid.UUID]int {
						if total != 4 || len(chosen) != 2 || a.ManaPool().TotalMana() != 6 {
							t.Error("allocation changed payment timing")
						}
						return map[uuid.UUID]int{a.PlayerID(): 1, b.PlayerID(): 3}
					}
				}
				factory := permanentScopeFactory(t, "Instant", strings.Replace(ronTargetSurcharge, "ManaCostGeneric(1)", "ManaCostGeneric(3)", 1)+`,SpellActions(`+body+`)`)
				card := factory()
				card.SetOwner(a.PlayerID())
				a.ManaPool().Add(core.Colorless, 3+3*max(0, count-1))
				var err error
				if route == "hand" {
					a.AddToHand(card)
					err = g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0)
				} else {
					a.AddToGraveyard(card)
					err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), core.ZoneGraveyard, card.ManaCost(), ids, 0)
				}
				if err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 3 {
					t.Fatal("surcharge did not count announced target occurrences")
				}
				g.ResolveStack()
				wantA, wantB := 20, 20
				switch shape {
				case "none":
					wantA = 21
				case "single":
					wantB = 19
				case "groups":
					wantB = 17
				case "distributed":
					wantA = 19
					wantB = 17
				}
				if a.Life() != wantA || b.Life() != wantB {
					t.Fatal("action binding changed")
				}
			})
		}
	}
}
