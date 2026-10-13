package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRoundedDamageCards(t *testing.T) {
	for _, name := range []string{"Banshee", "Eternal Flame"} {
		for _, amount := range []int{0, 1, 2, 3, 5, 6} {
			for _, kind := range []string{"opponent", "own-walker", "other-walker"} {
				t.Run(fmt.Sprintf("%s/%d/%s", name, amount, kind), func(t *testing.T) {
					g, a, b := ronGame()
					card := repositoryCard(t, "DRK", name)
					if err := mage.ValidateCardDefinition(name, card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					var source *mage.Permanent
					if name == "Banshee" {
						source = g.PutOnBattlefield(card, a.PlayerID())
						g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
					} else {
						a.AddToHand(card)
						for range amount {
							g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
						}
						g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
					}
					target := b.PlayerID()
					if kind != "opponent" {
						controller := a.PlayerID()
						if kind == "other-walker" {
							controller = b.PlayerID()
						}
						target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 20), controller).ID()
					}
					if name == "Banshee" {
						g.SetXValue(amount)
						a.ManaPool().Add(core.Colorless, amount)
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
							t.Fatal(err)
						}
						g.SetXValue(99)
					} else {
						a.ManaPool().Add(core.Red, 4)
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
							t.Fatal(err)
						}
					}
					if a.Life() != 20 || b.Life() != 20 {
						t.Fatal("early damage")
					}
					wantTarget, wantSelf := amount/2, (amount+1)/2
					if name == "Eternal Flame" {
						wantTarget = amount
					}
					callbacks := 0
					g.SetOnDamageDealt(func(_, _ string, _ int, combat bool) {
						callbacks++
						if combat || a.Life() != 20-wantSelf {
							t.Error("batch not simultaneous")
						}
					})
					clone := g.Clone()
					clone.ResolveStack()
					if a.Life() != 20 || b.Life() != 20 {
						t.Fatal("clone changed original")
					}
					callbacks = 0
					g.ResolveStack()
					if a.Life() != 20-wantSelf {
						t.Fatalf("self life %d", a.Life())
					}
					if kind == "opponent" {
						if b.Life() != 20-wantTarget {
							t.Fatalf("target life %d", b.Life())
						}
					} else if g.FindPermanent(target).Counters[core.Loyalty] != 20-wantTarget {
						t.Fatal("wrong walker damage")
					}
					wantCallbacks := 0
					if wantTarget > 0 {
						wantCallbacks++
					}
					if wantSelf > 0 {
						wantCallbacks++
					}
					if callbacks != wantCallbacks {
						t.Fatalf("events %d, want %d", callbacks, wantCallbacks)
					}
					if a.ManaPool().TotalMana() != 0 {
						t.Fatal("incorrect payment")
					}
				})
			}
		}
	}
}

func TestRONRoundedDamageInvalid(t *testing.T) {
	for _, value := range []string{"HalfRoundedDown", "HalfRoundedUp()", "HalfRoundedDown(Integer(1),Integer(2))", "HalfRoundedDown(Unknown)", "HalfRoundedUp(ValueX)", "HalfRoundedDown(TheNumberOfCountersRemovedThisWay)", "HalfRoundedUp(PowerOfPermanent(Ref_TargetPermanent))"} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(` + value + `,Player(You))])]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-rounded.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}

func TestRONRoundedDamageNestedValues(t *testing.T) {
	for _, tc := range []struct {
		value  string
		damage int
	}{
		{`HalfRoundedDown(Integer(5))`, 2}, {`HalfRoundedUp(Integer(5))`, 3},
		{`HalfRoundedUp(HalfRoundedDown(Integer(5)))`, 1},
		{`Twice(HalfRoundedDown(Integer(5)))`, 4},
		{`HalfRoundedDown(Minus(Integer(0),Integer(5)))`, 0},
	} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(`+tc.value+`,Player(You))])]))`)
		g, a, _ := ronGame()
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 20-tc.damage {
			t.Fatalf("%s: life %d", tc.value, a.Life())
		}
	}
}

func TestRONRoundedDamageEternalFlameResolutionCounts(t *testing.T) {
	for _, change := range []string{"entry", "phase", "control", "return", "batch-callback"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			lands := make([]*mage.Permanent, 0, 3)
			for range 3 {
				lands = append(lands, g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID()))
			}
			card := repositoryCard(t, "DRK", "Eternal Flame")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			amount := 3
			switch change {
			case "entry":
				g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
				amount = 4
			case "phase":
				g.PhaseOut(lands[0].ID())
				amount = 2
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: lands[0].ID(), TargetID: lands[0].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				amount = 2
			case "return":
				land := lands[0].Card
				g.BouncePermanentToHand(g.FindPermanent(lands[0].ID()))
				a.RemoveFromHand(land.ID())
				g.PutOnBattlefield(land, a.PlayerID())
			case "batch-callback":
				g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) { g.PhaseOut(lands[0].ID()) })
			}
			g.ResolveStack()
			if b.Life() != 20-amount || a.Life() != 20-(amount+1)/2 {
				t.Fatalf("resolution count: %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONRoundedDamageTargetIdentity(t *testing.T) {
	for _, name := range []string{"Banshee", "Eternal Flame"} {
		for _, change := range []string{"shroud", "phased", "phase-cycle", "returned", "face-down", "control"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				walkerCard := mage.NewPlaneswalker("Walker", "", 10)
				walkerCard.SetOwner(b.PlayerID())
				walker := g.PutOnBattlefield(walkerCard, b.PlayerID())
				card := repositoryCard(t, "DRK", name)
				card.SetOwner(a.PlayerID())
				if name == "Banshee" {
					source := g.PutOnBattlefield(card, a.PlayerID())
					g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
					g.SetXValue(5)
					a.ManaPool().Add(core.Colorless, 5)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{walker.ID()}); err != nil {
						t.Fatal(err)
					}
				} else {
					for range 5 {
						g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
					}
					a.AddToHand(card)
					a.ManaPool().Add(core.Red, 4)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{walker.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				}
				switch change {
				case "shroud":
					g.MutablePermanent(walker.ID()).GrantBaseAttr(core.Shroud)
				case "phased":
					g.PhaseOut(walker.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(walker.ID()))
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(walker.ID()))
					b.RemoveFromHand(walkerCard.ID())
					walker = g.PutOnBattlefield(walkerCard, b.PlayerID())
				case "face-down":
					g.MutablePermanent(walker.ID()).FaceDown = true
					g.MutablePermanent(walker.ID()).BasePTOverride = &[2]int{2, 10}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: walker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				g.ResolveStack()
				legal := change == "phase-cycle" || change == "control" || (change == "face-down" && name == "Banshee")
				wantSelf, wantTarget := 0, 0
				if legal {
					wantSelf, wantTarget = 3, 2
					if name == "Eternal Flame" {
						wantTarget = 5
					}
				}
				if a.Life() != 20-wantSelf {
					t.Fatal("illegal sole target did not suppress self damage")
				}
				current := g.FindPermanentIncludingPhased(walker.ID())
				if change == "face-down" {
					if current.Damage != wantTarget {
						t.Fatal("wrong face-down damage")
					}
				} else if current.Counters[core.Loyalty] != 10-wantTarget {
					t.Fatal("wrong target identity")
				}
			})
		}
	}
}

func TestRONRoundedDamageBansheeSourceAndCopies(t *testing.T) {
	for _, change := range []string{"departed", "returned", "phased", "control", "copy", "self-target", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Banshee")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.SetXValue(5)
			a.ManaPool().Add(core.Colorless, 5)
			target := b.PlayerID()
			if change == "self-target" {
				target = a.PlayerID()
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
				t.Fatal(err)
			}
			g.SetXValue(1)
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "prevention":
				g.AddPreventionShield(b.PlayerID(), 1)
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed live game")
			}
			g.ResolveStack()
			wantA, wantB := 17, 18
			switch change {
			case "copy":
				wantB = 13
			case "self-target":
				wantA, wantB = 15, 20
			case "prevention":
				wantA, wantB = 19, 19
			}
			if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatalf("damage/controller/X = %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONRoundedDamageEternalFlameCopies(t *testing.T) {
	g, a, b := ronGame()
	for range 5 {
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
	}
	for range 3 {
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
	}
	card := repositoryCard(t, "DRK", "Eternal Flame")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID() {
		t.Fatal("wrong copy target")
	}
	g.ResolveTopOfStack()
	if a.Life() != 17 || b.Life() != 18 {
		t.Fatal("copy used original controller's count or rounding")
	}
	g.ResolveTopOfStack()
	if a.Life() != 14 || b.Life() != 13 {
		t.Fatal("original lost its count or target")
	}
}

func TestRONRoundedDamageRollback(t *testing.T) {
	for _, name := range []string{"Banshee", "Eternal Flame"} {
		for _, failure := range []string{"unpaid", "wrong-target", "phased-target", "wrong-controller", "tapped", "sick"} {
			if name == "Eternal Flame" && (failure == "tapped" || failure == "sick") {
				continue
			}
			t.Run(name+"/"+failure, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "DRK", name)
				card.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if name == "Banshee" {
					source = g.PutOnBattlefield(card, a.PlayerID())
					if failure != "sick" {
						g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
					}
					if failure == "tapped" {
						g.TapPermanent(g.FindPermanent(source.ID()))
					}
				} else {
					a.AddToHand(card)
				}
				mana := 4
				if name == "Banshee" {
					mana = 5
				}
				if failure == "unpaid" {
					mana--
				}
				a.ManaPool().Add(core.Red, mana)
				g.SetXValue(5)
				target := b.PlayerID()
				if failure == "wrong-target" {
					target = g.PutOnBattlefield(mage.NewArtifact("Invalid", ""), b.PlayerID()).ID()
					if name == "Eternal Flame" {
						target = a.PlayerID()
					}
				}
				if failure == "phased-target" {
					walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID())
					g.PhaseOut(walker.ID())
					target = walker.ID()
				}
				payer := a.PlayerID()
				if failure == "wrong-controller" {
					payer = b.PlayerID()
				}
				var err error
				if source != nil {
					err = g.ActivateAbilityByIndex(payer, source.ID(), 0, []uuid.UUID{target})
				} else {
					err = g.CastSpellByID(payer, card.ID(), []uuid.UUID{target}, 0)
				}
				if err == nil || a.ManaPool().TotalMana() != mana || g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("failed announcement changed resources")
				}
				if source != nil && g.FindPermanent(source.ID()).Tapped != (failure == "tapped") {
					t.Fatal("failed payment tapped source")
				}
			})
		}
	}
}

func TestRONRoundedDamageBansheeLKI(t *testing.T) {
	for _, change := range []string{"departed", "returned", "phased"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Banshee")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.MutablePermanent(source.ID()).GrantBaseAttr(core.Lifelink)
			g.SetXValue(5)
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if change == "phased" {
				g.PhaseOut(source.ID())
			} else {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 18 {
				t.Fatalf("lost source lifelink/controller: %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONRoundedDamageIndependentInstances(t *testing.T) {
	g, a, b := ronGame()
	first := repositoryCard(t, "DRK", "Banshee")
	second := repositoryCard(t, "DRK", "Banshee")
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared ability declaration")
	}
	for i, card := range []mage.Card{first, second} {
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
		x := 3 + 2*i
		g.SetXValue(x)
		a.ManaPool().Add(core.Colorless, x)
		target := b.PlayerID()
		if i == 1 {
			target = a.PlayerID()
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveTopOfStack()
	if a.Life() != 15 || b.Life() != 20 {
		t.Fatal("second instance lost X/target")
	}
	g.ResolveTopOfStack()
	if a.Life() != 13 || b.Life() != 19 {
		t.Fatal("first instance reused second instance")
	}
}

func TestRONRoundedDamageBansheeCasting(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Banshee")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("creature entered before resolution")
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.CurrentPower(g) != 0 || source.CurrentToughness(g) != 1 || !source.HasSubType("Spirit") || a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong printed characteristics or payment")
	}
	g.SetXValue(0)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || g.FindPermanent(source.ID()).Tapped {
		t.Fatal("newly cast creature ignored summoning sickness")
	}
}

func TestRONRoundedDamageInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(ActionList([Unknown(HalfRoundedUp(HalfRoundedDown(ValueX))) ]))])`), "rounded.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("unsupported parent accepted")
	}
	for _, name := range []string{"HalfRoundedUp", "HalfRoundedDown", "ValueX"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("nested %s hidden", name)
		}
	}
	nodes, err = Parse(strings.NewReader(`Card(Name:"Inventory",Rules:[SpellActions(ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(HalfRoundedUp(HalfRoundedDown(Integer(5))),Player(You))])]))])`), "rounded.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries = map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", false, entries)
	for _, name := range []string{"HalfRoundedUp", "HalfRoundedDown", "Integer"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == "value" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lost value role", name)
		}
	}
}
