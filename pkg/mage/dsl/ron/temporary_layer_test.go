package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryLayersDeclareTypedReads(t *testing.T) {
	for _, modification := range []string{`AdjustPT(3, 3)`, `AddAbility([Flying])`, `AdjustPT(-1, 0), AddAbility([Flying, FirstStrike])`} {
		t.Run(modification, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [%s], UntilEndOfTurn)])))`, modification))
			first, second := factory(), factory()
			action := first.Abilities()[0].(*mage.ActionDefinition)
			if action.Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declaration")
			}
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			reads, installations := 0, 0
			var walk func(mage.DefinitionMetadata)
			walk = func(n mage.DefinitionMetadata) {
				if n.Role == "until" {
					installations++
				}
				if n.Role == "permanent-ref" && n.Text == "Ref_TargetPermanent" {
					reads++
				}
				for _, child := range n.Children {
					walk(child)
				}
			}
			walk(mage.InspectDefinition(action))
			if installations != 1 {
				t.Fatal("modifications did not share one installation")
			}
			if reads == 0 {
				t.Fatal("temporary layer has no typed subject reads")
			}
			invalid := mage.NewInstant("Missing declaration", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(action.Effects()...))))
			if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
				t.Fatal("missing declaration accepted")
			}
		})
	}
}

func TestRONTemporaryLayersRejectInvalidDeclarations(t *testing.T) {
	for _, body := range []string{
		`ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [], UntilEndOfTurn)])`,
		`ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(1, 1)], UntilEndOfTurn)])`,
		`Targeted([TargetPlayer(AnyPlayer)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(1, 1)], UntilEndOfTurn)]))`,
		`Targeted([TargetPermanent(AnyPermanent)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent(), [AdjustPT(1, 1)], UntilEndOfTurn)]))`,
		`ActionList([CreatePermanentLayerEffectUntil(HostPermanent, [AdjustPT(1, 1)], UntilEndOfTurn)])`,
		`ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatPermanent, [AdjustPT(1, 1)], UntilEndOfTurn)])`,
		`ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [AdjustPT(Integer(1), 1)], UntilEndOfTurn)])`,
		`ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [AddAbility([Flying()])], UntilEndOfTurn)])`,
		`ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [AdjustPT(1, 1)], UntilEndOfCombat)])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), %s)])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid-layer.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid declaration accepted: %s", body)
		}
	}
}

func TestRONTemporaryLayerSpellCopiesAndLegality(t *testing.T) {
	for _, name := range []string{"Giant Growth", "Jump"} {
		for _, change := range []string{"none", "returned", "phased", "phase-in"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEB", "Llanowar Elves")
				card.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(card, b.PlayerID())
				otherCard := repositoryCard(t, "LEB", "Llanowar Elves")
				otherCard.SetOwner(a.PlayerID())
				other := g.PutOnBattlefield(otherCard, a.PlayerID())
				spell := repositoryCard(t, "LEB", name)
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(core.Green, 1)
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if target.CurrentPower(g) != 1 || target.HasKeyword(core.Flying) {
					t.Fatal("layer started at announcement")
				}
				clone := g.Clone()
				phased := g.PhaseOut(target.ID())
				copied := g.CopySpellOnStack(spell.ID(), b.PlayerID(), true)
				if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != other.ID() {
					t.Fatal("copy target frame")
				}
				g.PhaseIn(phased)
				current := target.ID()
				switch change {
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					b.RemoveFromHand(card.ID())
					current = g.PutOnBattlefield(card, b.PlayerID()).ID()
				case "phased":
					g.PhaseOut(target.ID())
				case "phase-in":
					g.PhaseIn(g.PhaseOut(target.ID()))
				}
				g.ResolveStack()
				check := func(game *mage.Game, id uuid.UUID, boosted bool) {
					t.Helper()
					p := game.FindPermanentIncludingPhased(id)
					power := 1
					if boosted && name == "Giant Growth" {
						power = 4
					}
					if p == nil || p.CurrentPower(game) != power || p.HasKeyword(core.Flying) != (boosted && name == "Jump") {
						t.Fatal("wrong subject or layer")
					}
				}
				check(g, other.ID(), true)
				check(g, current, change == "none" || change == "phase-in")
				check(clone, target.ID(), false)
				clone.ResolveStack()
				check(clone, target.ID(), true)
				check(clone, other.ID(), false)
				g.PhaseIn([]uuid.UUID{current})
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				check(g, other.ID(), false)
				check(g, current, false)
				check(clone, target.ID(), true)
			})
		}
	}
}

func TestRONTemporaryLayerActivationDepartureAndRollback(t *testing.T) {
	for _, name := range []string{"Elven Lyre", "Flying Carpet"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			fixture := repositoryCard(t, "LEB", "Llanowar Elves")
			fixture.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(fixture, b.PlayerID())
			set := "ARN"
			cost := 2
			if name == "Elven Lyre" {
				set = "FEM"
				cost = 1
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("unpaid ability succeeded")
			}
			if g.FindPermanent(source.ID()).Tapped || len(a.Graveyard()) != 0 || !g.GetStack().IsEmpty() {
				t.Fatal("payment did not roll back")
			}
			a.ManaPool().Add(core.Colorless, cost)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if name == "Elven Lyre" && g.FindPermanent(source.ID()) != nil {
				t.Fatal("sacrifice cost not paid")
			}
			if name == "Flying Carpet" {
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			}
			clone := g.Clone()
			clone.ResolveStack()
			if target.CurrentPower(g) != 1 || target.HasKeyword(core.Flying) {
				t.Fatal("clone changed live game")
			}
			g.ResolveStack()
			wantPower := 1
			if name == "Elven Lyre" {
				wantPower = 3
			}
			if g.FindPermanent(target.ID()).CurrentPower(g) != wantPower || g.FindPermanent(target.ID()).HasKeyword(core.Flying) != (name == "Flying Carpet") {
				t.Fatal("departed source lost layer")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(target.ID()).CurrentPower(g) != 1 || g.FindPermanent(target.ID()).HasKeyword(core.Flying) {
				t.Fatal("layer survived cleanup")
			}
		})
	}
}

func TestRONTemporaryLayerSourceIdentity(t *testing.T) {
	for _, change := range []string{"none", "returned", "phased", "phase-in", "installed-return"} {
		t.Run(change, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "LEB", "Shivan Dragon")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Red, 2)
			for range 2 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			current := source.ID()
			switch change {
			case "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				current = g.PutOnBattlefield(card, a.PlayerID()).ID()
			case "phased":
				g.PhaseOut(source.ID())
			case "phase-in":
				g.PhaseIn(g.PhaseOut(source.ID()))
			}
			g.ResolveStack()
			if change == "installed-return" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				current = g.PutOnBattlefield(card, a.PlayerID()).ID()
			}
			g.PhaseIn([]uuid.UUID{current})
			want := 5
			if change == "none" || change == "phase-in" {
				want = 7
			}
			if g.FindPermanent(current).CurrentPower(g) != want {
				t.Fatal("source reference followed another incarnation or absent subject")
			}
			if change == "none" {
				g.PhaseIn(g.PhaseOut(current))
				if g.FindPermanent(current).CurrentPower(g) != 7 {
					t.Fatal("phasing erased captured layer")
				}
			}
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(current).CurrentPower(clone) != 5 || g.FindPermanent(current).CurrentPower(g) != want {
				t.Fatal("cleanup shared layer state")
			}
		})
	}
}

func TestRONTemporaryLayerHostResolutionAndCapture(t *testing.T) {
	for _, change := range []string{"reattach", "departed-source", "returned-source", "returned-host", "phased-host"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEB", "Llanowar Elves")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			otherCard := repositoryCard(t, "LEB", "Llanowar Elves")
			otherCard.SetOwner(b.PlayerID())
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			auraCard := repositoryCard(t, "LEB", "Blessing")
			auraCard.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			a.ManaPool().Add(core.White, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			current := host.ID()
			switch change {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "departed-source":
				g.DestroyPermanent(g.FindPermanent(aura.ID()))
			case "returned-source":
				g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
				a.RemoveFromHand(auraCard.ID())
				returned := g.PutOnBattlefield(auraCard, a.PlayerID())
				g.Attach(returned.ID(), other.ID())
			case "returned-host":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				b.RemoveFromHand(hostCard.ID())
				current = g.PutOnBattlefield(hostCard, b.PlayerID()).ID()
			case "phased-host":
				g.PhaseOut(host.ID())
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{current})
			wantHost, wantOther := 1, 1
			switch change {
			case "reattach":
				wantOther = 2
			case "departed-source", "returned-source":
				wantHost = 2
			}
			if g.FindPermanent(current).CurrentPower(g) != wantHost || g.FindPermanent(other.ID()).CurrentPower(g) != wantOther {
				t.Fatal("host resolution did not use original source attachment")
			}
			if change == "reattach" {
				g.Attach(aura.ID(), host.ID())
				if g.FindPermanent(other.ID()).CurrentPower(g) != 2 || g.FindPermanent(host.ID()).CurrentPower(g) != 1 {
					t.Fatal("installed layer followed later attachment")
				}
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanent(current).CurrentPower(g) != 1 || g.FindPermanent(other.ID()).CurrentPower(g) != 1 {
				t.Fatal("host layer survived cleanup")
			}
		})
	}
}

func TestRONTemporaryLayerTriggerAndMixedSubjects(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEB", "Llanowar Elves")
	card.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(card, b.PlayerID())
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(1, 2), AddAbility([Flying])], UntilEndOfTurn)])))`)
	sourceCard := factory()
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if g.FindPermanent(target.ID()).CurrentPower(g) != 1 {
		t.Fatal("trigger layer started before resolution")
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.ResolveStack()
	p := g.FindPermanent(target.ID())
	if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 || !p.HasKeyword(core.Flying) {
		t.Fatal("mixed trigger layers lost target")
	}
	action := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [AddAbility([FirstStrike])], UntilEndOfTurn), CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AddAbility([Trample])], UntilEndOfTurn)])))`)()
	action.SetOwner(a.PlayerID())
	actor := g.PutOnBattlefield(action, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), actor.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(actor.ID()).HasKeyword(core.FirstStrike) || g.FindPermanent(actor.ID()).HasKeyword(core.Trample) || !g.FindPermanent(target.ID()).HasKeyword(core.Trample) {
		t.Fatal("source and target reads were conflated")
	}
}

func TestRONTemporaryLayerInventoryRoles(t *testing.T) {
	src := `Card(Name: "Layer", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [AdjustPT(1, 2), AddAbility([Flying])], UntilEndOfTurn)]))])`
	nodes, err := Parse(strings.NewReader(src), "layer.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Layer", "Card", "card", "collection", true, entries)
	want := map[string]string{"ThisPermanent": "permanent_reference", "AdjustPT": "temporary_modification", "AddAbility": "temporary_modification", "Flying": "keyword", "UntilEndOfTurn": "duration"}
	for name, role := range want {
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

func TestRONTemporaryLayerCrossRuleAndSpellSourceRejected(t *testing.T) {
	for _, declaration := range []string{
		`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [AdjustPT(1, 1)], UntilEndOfTurn)]))])`,
		`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([TapPermanent(Ref_TargetPermanent)]))), Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [AdjustPT(1, 1)], UntilEndOfTurn)]))])`,
	} {
		nodes, err := Parse(strings.NewReader(declaration), "invalid-layer.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid source or cross-rule read accepted")
		}
	}
}
