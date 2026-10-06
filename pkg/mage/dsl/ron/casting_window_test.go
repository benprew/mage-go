package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCastingWindows(t *testing.T) {
	for _, name := range []string{"Reset", "Teleport"} {
		for step := core.Untap; step <= core.Cleanup; step++ {
			for _, index := range []int{0, 1} {
				t.Run(fmt.Sprintf("%s/%s/player%d", name, step, index), func(t *testing.T) {
					g, a, b := ronGame()
					if name == "Reset" && step >= core.Draw {
						g.SetStep(core.Upkeep)
					}
					g.SetStep(step)
					caster := a
					if index == 1 {
						caster = b
					}
					card := repositoryCard(t, "LEG", name)
					card.SetOwner(caster.PlayerID())
					caster.AddToHand(card)
					caster.ManaPool().Add(core.Blue, 3)
					target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
					targets := []uuid.UUID{target.ID()}
					want := step == core.DeclareAttackers
					if name == "Reset" {
						targets = nil
						want = index == 1 && step >= core.Draw
					}
					if got := len(g.GetCastableSpells(caster.PlayerID())) == 1; got != want {
						t.Fatalf("cast query = %v, want %v", got, want)
					}
					err := g.CastSpellByID(caster.PlayerID(), card.ID(), targets, 0)
					if (err == nil) != want {
						t.Fatalf("cast = %v, allowed = %v", err, want)
					}
					if !want && (len(caster.Hand()) != 1 || caster.ManaPool().TotalMana() != 3 || !g.GetStack().IsEmpty()) {
						t.Fatal("timing rejection changed resources")
					}
				})
			}
		}
	}
}

func TestRONCastingWindowGameplay(t *testing.T) {
	t.Run("Reset", func(t *testing.T) {
		g, a, b := ronGame()
		g.SetStep(core.Upkeep)
		lands := make([]uuid.UUID, 0, 2)
		for _, player := range []mage.Player{a, b} {
			card := repositoryCard(t, "LEB", "Island")
			card.SetOwner(player.PlayerID())
			p := g.PutOnBattlefield(card, player.PlayerID())
			g.TapPermanent(p)
			lands = append(lands, p.ID())
		}
		creature := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
		g.TapPermanent(creature)
		phased := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
		g.TapPermanent(phased)
		g.PhaseOut(phased.ID())
		card := repositoryCard(t, "LEG", "Reset")
		card.SetOwner(b.PlayerID())
		b.AddToHand(card)
		b.ManaPool().Add(core.Blue, 2)
		if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("upkeep cast allowed")
		}
		g.SetStep(core.Draw)
		if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		late := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
		g.TapPermanent(late)
		g.SetStep(core.Upkeep)
		clone := g.Clone()
		clone.ResolveStack()
		if !g.FindPermanent(lands[1]).Tapped || clone.FindPermanent(lands[1]).Tapped {
			t.Fatal("clone isolation")
		}
		g.ResolveStack()
		if !g.FindPermanent(lands[0]).Tapped || g.FindPermanent(lands[1]).Tapped || g.FindPermanent(late.ID()).Tapped || !g.FindPermanent(creature.ID()).Tapped || !g.FindPermanentIncludingPhased(phased.ID()).Tapped {
			t.Fatal("Reset selection or timing")
		}
		if b.ManaPool().TotalMana() != 0 || len(b.Graveyard()) != 1 {
			t.Fatal("Reset payment or destination")
		}
	})
	t.Run("Teleport", func(t *testing.T) {
		g, a, b := ronGame()
		g.SetStep(core.DeclareAttackers)
		card := repositoryCard(t, "LEG", "Teleport")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
			t.Fatal("unpaid Teleport allowed")
		}
		if len(a.Hand()) != 1 || !g.GetStack().IsEmpty() {
			t.Fatal("payment rollback")
		}
		blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
		a.ManaPool().Add(core.Blue, 3)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.SetStep(core.DeclareBlockers)
		g.ResolveStack()
		if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) {
			t.Fatal("Teleport restriction did not resolve")
		}
		g.GetEffects().RemoveEndOfTurn()
		g.ApplyContinuousEffects()
		if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(target.ID()), g) {
			t.Fatal("Teleport survived cleanup")
		}
	})
}

func TestRONCastingWindowInvalid(t *testing.T) {
	for _, declaration := range []string{
		`CastEffect()`, `CastEffect(CantBeCastUnless())`, `CastEffect(CantBeCastUnless(IsDuringUpkeep()))`,
		`CastEffect(CantBeCastUnless(And([])))`, `CastEffect(CantBeCastUnless(And([IsDuringUpkeep, Unknown])))`,
		`CastEffect(CantBeCastUnless(IsAPlayersTurn(Ref_TargetPlayer)))`, `CastEffect(CantBeCastUnless(IsPlayersTurn(You())))`,
		`CastEffect(CantBeCastUnless(IsAfterCombat))`, `CastEffect(CantBeCastUnless(IsBeforeCombatDamageStep()))`,
		`CastEffect(CantBeCastUnless(IsDuringUpkeep)), CastEffect(CantBeCastUnless(IsDuringUpkeep))`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [%s, SpellActions(ActionList([GainLife(Integer(1))]))])`, declaration)
		nodes, err := Parse(strings.NewReader(src), "invalid-casting.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid %s", declaration)
		}
	}
}

func TestRONCastingWindowTeleportIdentity(t *testing.T) {
	for _, change := range []string{"none", "shroud", "returned", "phased", "phase-cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.DeclareAttackers)
			fixture := repositoryCard(t, "LEB", "Grizzly Bears")
			fixture.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(fixture, b.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			spell := repositoryCard(t, "LEG", "Teleport")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Blue, 3)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			current := target.ID()
			switch change {
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "returned":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(fixture.ID())
				current = g.PutOnBattlefield(fixture, b.PlayerID()).ID()
			case "phased":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			g.SetStep(core.EndStep)
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{current})
			want := change == "none" || change == "phase-cycle" || change == "control"
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			if got := !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(current), g); got != want {
				t.Fatalf("blocking restriction = %v, want %v", got, want)
			}
			if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(other.ID()), g) {
				t.Fatal("wrong permanent changed")
			}
			clone.ResolveStack()
			cloneBlocker := clone.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			if mage.CanBlock(cloneBlocker, clone.FindPermanent(target.ID()), clone) {
				t.Fatal("clone lost target")
			}
		})
	}
}

func TestRONCastingWindowOrderScopeAndInstances(t *testing.T) {
	for _, condition := range []string{`IsDuringUpkeep`, `IsDuringDeclareBlockersStep`, `And([IsPlayersTurn(You), IsDuringDeclareAttackersStep])`, `And([IsAPlayersTurn(Opponent), And([IsDuringUpkeep])])`} {
		for _, castFirst := range []bool{true, false} {
			restriction := `CastEffect(CantBeCastUnless(` + condition + `))`
			body := `SpellActions(ActionList([GainLife(Integer(1))]))`
			if castFirst {
				body = restriction + `,` + body
			} else {
				body += `,` + restriction
			}
			factory := permanentScopeFactory(t, "Instant", body)
			first, second := factory(), factory()
			if first.Abilities()[0] == second.Abilities()[0] {
				t.Fatal("shared spell instance")
			}
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			for step := core.Untap; step <= core.Cleanup; step++ {
				g.SetStep(step)
				for _, player := range []mage.Player{a, b} {
					card := factory()
					card.SetOwner(player.PlayerID())
					player.AddToHand(card)
					expected := step == core.Upkeep
					switch condition {
					case `IsDuringDeclareBlockersStep`:
						expected = step == core.DeclareBlockers
					case `And([IsPlayersTurn(You), IsDuringDeclareAttackersStep])`:
						expected = player == a && step == core.DeclareAttackers
					case `And([IsAPlayersTurn(Opponent), And([IsDuringUpkeep])])`:
						expected = player == b && step == core.Upkeep
					}
					err := g.CastSpellByID(player.PlayerID(), card.ID(), nil, 0)
					if (err == nil) != expected {
						t.Fatalf("%s/%s: %v", condition, step, err)
					}
					g.ResolveStack()
					player.RemoveFromHand(card.ID())
				}
			}
		}
	}
	for _, src := range []string{
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(IsDuringUpkeep))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(IsDuringUpkeep)),SpellActions(Modal_ChooseOne([ActionList([GainLife(Integer(1))]),ActionList([GainLife(Integer(2))])]))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CastEffect(CantBeCastUnless(IsDuringUpkeep))]))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([UntapEachPermanent(TheChosenPermanents)]))])`,
		`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([UntapEachPermanent(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))]))])`,
	} {
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONCastingWindowInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(And([IsAPlayersTurn(Opponent),IsDuringUpkeep]))),SpellActions(ActionList([UntapEachPermanent(IsCardtype(Land))]))])`
	nodes, err := Parse(strings.NewReader(src), "casting.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"CantBeCastUnless": "casting_restriction", "And": "casting_condition", "IsAPlayersTurn": "casting_condition", "Opponent": "player_reference", "IsDuringUpkeep": "casting_condition", "IsCardtype": "permanent_filter"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing role %s", name, role)
		}
	}
}

func TestRONCastingWindowStackCopies(t *testing.T) {
	for _, name := range []string{"Reset", "Teleport"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			if name == "Reset" {
				g.SetStep(core.Upkeep)
			}
			g.SetStep(core.DeclareAttackers)
			caster := a
			if name == "Reset" {
				caster = b
			}
			card := repositoryCard(t, "LEG", name)
			card.SetOwner(caster.PlayerID())
			caster.AddToHand(card)
			caster.ManaPool().Add(core.Blue, 3)
			first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), b.PlayerID())
			landA := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
			landB := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
			g.TapPermanent(landA)
			g.TapPermanent(landB)
			targets := []uuid.UUID{first.ID()}
			if name == "Reset" {
				targets = nil
			}
			if err := g.CastSpellByID(caster.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.SetStep(core.Upkeep)
			if name == "Reset" {
				if g.CopySpellOnStack(card.ID(), a.PlayerID(), false) == nil {
					t.Fatal("copy blocked by turn condition")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(landA.ID()).Tapped || !g.FindPermanent(landB.ID()).Tapped {
					t.Fatal("copied Reset did not use copy controller")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(landB.ID()).Tapped {
					t.Fatal("original Reset lost controller")
				}
			} else {
				g.PhaseOut(first.ID())
				copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), true)
				if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
					t.Fatal("copy target declaration")
				}
				g.PhaseIn([]uuid.UUID{first.ID()})
				g.ResolveTopOfStack()
				blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
				if mage.CanBlock(blocker, g.FindPermanent(second.ID()), g) || !mage.CanBlock(blocker, g.FindPermanent(first.ID()), g) {
					t.Fatal("copy target frame")
				}
				g.ResolveTopOfStack()
				if mage.CanBlock(blocker, g.FindPermanent(first.ID()), g) {
					t.Fatal("original target frame")
				}
			}
		})
	}
}

func TestRONCastingWindowResetExtraUpkeep(t *testing.T) {
	g, _, b := ronGame()
	g.SetStep(core.Upkeep)
	card := repositoryCard(t, "LEG", "Reset")
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	b.ManaPool().Add(core.Blue, 2)
	if len(g.GetCastableSpells(b.PlayerID())) != 0 {
		t.Fatal("Reset allowed in first upkeep")
	}
	clone := g.Clone()
	g.RunStepWithPriority(core.Upkeep)
	g.SetStep(core.Upkeep)
	b.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatalf("Reset rejected in second upkeep: %v", err)
	}
	if len(clone.GetCastableSpells(b.PlayerID())) != 0 {
		t.Fatal("clone inherited ended upkeep")
	}
}
