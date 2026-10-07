package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryCollectionRealCards(t *testing.T) {
	cases := []struct {
		set, name, selection string
		power, toughness     int
	}{
		{"ARN", "Army of Allah", "attack", 2, 0},
		{"ARN", "Piety", "block", 0, 3},
		{"DRK", "Bone Flute", "all", -1, 0},
		{"DRK", "Holy Light", "nonwhite", -1, -1},
		{"DRK", "Marsh Gas", "all", -2, 0},
		{"DRK", "Morale", "attack", 1, 1},
		{"LEG", "Hell Swarm", "all", -1, 0},
		{"LEG", "Shield Wall", "you", 0, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			add := func(name, cost string, owner uuid.UUID) *mage.Permanent {
				c := mage.NewCreature(name, cost, 4, 4)
				c.SetOwner(owner)
				return g.PutOnBattlefield(c, owner)
			}
			own := add("Own", "{U}", a.PlayerID())
			enemy := add("Enemy", "{U}", b.PlayerID())
			white := add("White", "{W}", a.PlayerID())
			phased := add("Phased", "{U}", a.PlayerID())
			g.PhaseOut(phased.ID())
			landCard := repositoryCard(t, "LEB", "Forest")
			landCard.SetOwner(a.PlayerID())
			land := g.PutOnBattlefield(landCard, a.PlayerID())
			g.GetCombat().AddAttacker(own.ID(), b.PlayerID())
			g.GetCombat().AddBlocker(enemy.ID(), own.ID())
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
				a.ManaPool().Add(color, 8)
			}
			var source *mage.Permanent
			if tc.name == "Bone Flute" {
				source = g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("tap cost was not paid")
				}
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			} else {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			if own.CurrentPower(g) != 4 || enemy.CurrentToughness(g) != 4 {
				t.Fatal("effect started before resolution")
			}
			early := add("Before resolution", "{U}", a.PlayerID())
			pending := g.Clone()
			g.ResolveStack()
			selected := func(kind string) bool {
				switch tc.selection {
				case "all":
					return true
				case "nonwhite":
					return kind != "white"
				case "you":
					return kind != "enemy"
				case "attack":
					return kind == "own"
				case "block":
					return kind == "enemy"
				}
				return false
			}
			check := func(game *mage.Game, id uuid.UUID, affected bool) {
				t.Helper()
				p := game.FindPermanentIncludingPhased(id)
				power, toughness := 4, 4
				if affected {
					power += tc.power
					toughness += tc.toughness
				}
				if p == nil || p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
					t.Fatalf("%s: expected %d/%d", id, power, toughness)
				}
			}
			for kind, p := range map[string]*mage.Permanent{"own": own, "enemy": enemy, "white": white, "early": early} {
				check(g, p.ID(), selected(kind))
				check(pending, p.ID(), false)
			}
			check(g, phased.ID(), false)
			g.PhaseIn([]uuid.UUID{phased.ID()})
			check(g, phased.ID(), false)
			late := add("After resolution", "{U}", a.PlayerID())
			check(g, late.ID(), false)
			g.RemoveFromCombat(own.ID())
			g.RemoveFromCombat(enemy.ID())
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, own.ID(), selected("own"))
			check(g, enemy.ID(), selected("enemy"))
			g.PhaseIn(g.PhaseOut(own.ID()))
			check(g, own.ID(), selected("own"))
			pending.ResolveStack()
			check(pending, own.ID(), selected("own"))
			g.BouncePermanentToHand(g.FindPermanent(own.ID()))
			a.RemoveFromHand(own.Card.ID())
			returned := g.PutOnBattlefield(own.Card, a.PlayerID())
			check(g, returned.ID(), false)
			if g.FindPermanent(land.ID()).HasType(core.TypeCreature) {
				t.Fatal("land was changed")
			}
			if tc.name != "Bone Flute" && len(a.Graveyard()) != 1 {
				t.Fatal("spell did not go to graveyard")
			}
			expired := g.Clone()
			expired.GetEffects().RemoveEndOfTurn()
			expired.ApplyContinuousEffects()
			check(expired, enemy.ID(), false)
			check(g, enemy.ID(), selected("enemy"))
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			check(g, enemy.ID(), false)
			check(g, white.ID(), false)
			check(g, early.ID(), false)
		})
	}
}

func TestRONTemporaryCollectionInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [AdjustPT(Integer(1), 2)], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [AdjustPT(1)], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [AddAbility([])], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [SetPT(PT(2, 2))], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [AdjustPT(1, 2)], UntilEndOfCombat)`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [AdjustPT(1, 2)], UntilEndOfTurn())`,
		`CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents, [AdjustPT(1, 2)], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(And([]), [AdjustPT(1, 2)], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)), [AdjustPT(1, 2)], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(Or([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You))]), [AdjustPT(1, 2)], UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffectUntil(Other(ThisPermanent), [AdjustPT(1, 2)], UntilEndOfTurn)`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([%s]))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid-collection.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONTemporaryCollectionCopyController(t *testing.T) {
	g, a, b := ronGame()
	add := func(owner uuid.UUID) *mage.Permanent {
		card := mage.NewCreature("Subject", "", 3, 3)
		card.SetOwner(owner)
		return g.PutOnBattlefield(card, owner)
	}
	own, enemy := add(a.PlayerID()), add(b.PlayerID())
	spell := repositoryCard(t, "LEG", "Shield Wall")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.White, 2)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(spell.ID(), b.PlayerID(), false)
	if copied == nil || len(copied.Targets) != 0 {
		t.Fatal("collection spell copy has targets")
	}
	g.PhaseOut(own.ID())
	g.ResolveStack()
	if g.FindPermanent(enemy.ID()).CurrentToughness(g) != 5 {
		t.Fatal("copy did not use its controller")
	}
	g.PhaseIn([]uuid.UUID{own.ID()})
	if g.FindPermanent(own.ID()).CurrentToughness(g) != 3 {
		t.Fatal("original captured a phased creature")
	}
	if len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("copy became a graveyard card")
	}
}

func TestRONTemporaryCollectionBoneFluteRollback(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Bone Flute")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := repositoryCard(t, "LEB", "Air Elemental")
	victim.SetOwner(b.PlayerID())
	subject := g.PutOnBattlefield(victim, b.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	if g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 || !g.GetStack().IsEmpty() {
		t.Fatal("failed cost changed live state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("mana cost was not paid")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("tapped source activated again")
	}
	g.ResolveStack()
	if g.FindPermanent(subject.ID()).CurrentPower(g) != 3 {
		t.Fatal("collection reduction was not applied")
	}
}

func TestRONTemporaryCollectionSharedContexts(t *testing.T) {
	action := `CreateEachPermanentLayerEffectUntil(And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(You))]), [AdjustPT(1, 2), AdjustPT(2, -1)], UntilEndOfTurn)`
	for _, rule := range []string{
		`Activated(PayMana([]), ActionList([%s]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([%s]))`,
		`Activated(PayMana([]), ActionList([MayCost(PayMana([])), If(CostWasPaid, [%s])]))`,
	} {
		t.Run(rule, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			other := mage.NewCreature("Other", "", 3, 3)
			other.SetOwner(a.PlayerID())
			subject := g.PutOnBattlefield(other, a.PlayerID())
			enemyCard := mage.NewCreature("Enemy", "", 3, 3)
			enemyCard.SetOwner(b.PlayerID())
			enemy := g.PutOnBattlefield(enemyCard, b.PlayerID())
			factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(rule, action))
			first, second := factory(), factory()
			if first.Abilities()[0] == second.Abilities()[0] {
				t.Fatal("factories share ability state")
			}
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			first.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(first, a.PlayerID())
			if strings.HasPrefix(rule, "Activated") {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.ResolveStack()
			if g.FindPermanent(subject.ID()).CurrentPower(g) != 6 || g.FindPermanent(subject.ID()).CurrentToughness(g) != 4 || g.FindPermanent(enemy.ID()).CurrentPower(g) != 3 {
				t.Fatal("collection did not use resolving controller or combine adjustments")
			}
		})
	}
	src := `Card(Name: "Inventory collection", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([CreateEachPermanentLayerEffectUntil(IsCardtype(Creature), [AdjustPT(1, 2)], UntilEndOfTurn)]))])`
	nodes, err := Parse(strings.NewReader(src), "collection.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory collection", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"IsCardtype": "permanent_filter", "AdjustPT": "temporary_modification", "UntilEndOfTurn": "duration"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lacks %s role", name, role)
		}
	}
}

func TestRONTemporaryCollectionOpponentCombat(t *testing.T) {
	for _, name := range []string{"Army of Allah", "Morale"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetActivePlayerIndex(1)
			g.SetStep(core.DeclareBlockers)
			card := repositoryCard(t, "LEB", "Air Elemental")
			card.SetOwner(b.PlayerID())
			attacker := g.PutOnBattlefield(card, b.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
			set, power, toughness := "ARN", 6, 4
			if name == "Morale" {
				set, power, toughness = "DRK", 5, 5
			}
			spell := repositoryCard(t, set, name)
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 3)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.RemoveFromCombat(attacker.ID())
			if g.FindPermanent(attacker.ID()).CurrentPower(g) != power || g.FindPermanent(attacker.ID()).CurrentToughness(g) != toughness {
				t.Fatal("opponent attacker did not retain the bonus")
			}
		})
	}
}

func TestRONTemporaryCollectionHolyLightDeaths(t *testing.T) {
	g, a, b := ronGame()
	for _, owner := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		c := repositoryCard(t, "LEB", "Llanowar Elves")
		c.SetOwner(owner)
		g.PutOnBattlefield(c, owner)
	}
	white := repositoryCard(t, "LEB", "Savannah Lions")
	white.SetOwner(b.PlayerID())
	survivor := g.PutOnBattlefield(white, b.PlayerID())
	spell := repositoryCard(t, "DRK", "Holy Light")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(survivor.ID()) == nil || survivor.CurrentPower(g) != 2 || survivor.CurrentToughness(g) != 1 {
		t.Fatal("white creature was reduced")
	}
	if len(a.Graveyard()) != 2 || len(b.Graveyard()) != 1 {
		t.Fatal("nonwhite creatures did not die through state-based actions")
	}
}
