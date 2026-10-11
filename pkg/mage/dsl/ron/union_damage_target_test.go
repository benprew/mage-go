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

func TestRONUnionDamageTargetWitches(t *testing.T) {
	for _, scenario := range []string{"players", "creatures", "walker", "same", "self", "first-illegal", "second-illegal", "all-illegal", "return", "phase", "phase-cycle", "source-leaves", "source-return", "source-phase", "source-control", "prevent", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ARN", "Cuombajj Witches"), a.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 5), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 5), a.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			hidden := g.PutOnBattlefield(mage.NewCreature("Hidden", "", 2, 5), b.PlayerID())
			g.PhaseOut(hidden.ID())
			protected := g.PutOnBattlefield(mage.NewCreature("Hexproof", "", 2, 5), b.PlayerID())
			protected.GrantBaseAttr(core.Hexproof)
			left, right := first.ID(), second.ID()
			if scenario == "players" {
				left, right = b.PlayerID(), a.PlayerID()
			}
			if scenario == "same" {
				right = left
			}
			if scenario == "self" {
				right = source.ID()
			}
			if scenario == "walker" {
				second = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID())
				right = second.ID()
			}
			b.requested = []uuid.UUID{right}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{left, land.ID()}); err == nil {
				t.Fatal("summoning sickness ignored")
			}
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID(), right}); err == nil {
				t.Fatal("land target accepted")
			}
			if source.Tapped || g.StackSize() != 0 {
				t.Fatal("failed announcement changed state")
			}
			b.candidates = nil
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{left, land.ID()}); err != nil {
				t.Fatal(err)
			}
			if !source.Tapped || g.StackPeek().Targets[1] != right || len(b.candidates) != 1 || len(a.candidates) != 0 {
				t.Fatal("target chooser or tap cost")
			}
			for _, id := range b.candidates[0] {
				if id == land.ID() || id == hidden.ID() || id == protected.ID() {
					t.Fatal("opponent choice changed targeting permissions")
				}
			}
			if first.Damage != 0 || second.Damage != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("damage before resolution")
			}
			switch scenario {
			case "first-illegal":
				first.GrantBaseAttr(core.Shroud)
			case "second-illegal":
				second.GrantBaseAttr(core.Shroud)
			case "all-illegal":
				first.GrantBaseAttr(core.Shroud)
				second.GrantBaseAttr(core.Shroud)
			case "return":
				g.BouncePermanentToHand(first)
				g.PutOnBattlefield(first.Card, b.PlayerID())
			case "phase":
				g.PhaseOut(first.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "source-leaves":
				g.BouncePermanentToHand(source)
			case "source-return":
				g.BouncePermanentToHand(source)
				g.PutOnBattlefield(source.Card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: second.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "prevent":
				g.AddPreventionShield(first.ID(), 1)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.FindPermanent(first.ID()) != nil && clone.FindPermanent(first.ID()).Damage != 0 {
				t.Fatal("clone damage shared")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if scenario == "players" {
					if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 19 {
						t.Fatal("player groups mixed")
					}
					continue
				}
				want := 1
				if scenario == "copy" {
					want = 2
				}
				if scenario == "same" {
					want = 2
				}
				if scenario == "first-illegal" || scenario == "all-illegal" || scenario == "return" || scenario == "phase" || scenario == "prevent" {
					want = 0
				}
				if game.FindPermanentIncludingPhased(first.ID()).Damage != want {
					t.Fatalf("first damage=%d want=%d", game.FindPermanentIncludingPhased(first.ID()).Damage, want)
				}
				if scenario == "same" {
					continue
				}
				if scenario == "self" {
					if game.FindPermanent(source.ID()).Damage != 1 {
						t.Fatal("self recipient")
					}
					continue
				}
				if scenario == "walker" {
					if game.FindPermanent(second.ID()).Counters[core.Loyalty] != 4 {
						t.Fatal("planeswalker recipient")
					}
					continue
				}
				want = 1
				if scenario == "copy" {
					want = 2
				}
				if scenario == "second-illegal" || scenario == "all-illegal" {
					want = 0
				}
				if game.FindPermanent(second.ID()).Damage != want {
					t.Fatalf("second damage=%d want=%d", game.FindPermanent(second.ID()).Damage, want)
				}
			}
		})
	}
}

func TestRONUnionDamageTargetProjection(t *testing.T) {
	body := `SpellActions(Targeted([AnyTarget],ActionList([If(PermanentPassesFilter(AnyTargetAsAPermanent,IsCardtype(Creature)),[PutPermanentIntoItsOwnersHand(AnyTargetAsAPermanent),PlayerAction(ControllerOfPermanent(AnyTargetAsAPermanent),GainLife(Integer(3)))])])))`
	factory := permanentScopeFactory(t, "Instant", body)
	if factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0] == factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared declaration")
	}
	for _, kind := range []string{"creature", "player", "walker", "phase", "return", "control"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			creature := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 5), b.PlayerID())
			target := creature.ID()
			if kind == "player" {
				target = b.PlayerID()
			}
			if kind == "walker" {
				target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID()).ID()
			}
			card := factory()
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			if kind == "phase" {
				g.PhaseOut(creature.ID())
			}
			if kind == "return" {
				g.BouncePermanentToHand(creature)
				g.PutOnBattlefield(creature.Card, b.PlayerID())
			}
			if kind == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: creature.ID(), TargetID: creature.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				wantA, wantB := 20, 20
				if kind == "creature" {
					wantB = 23
				}
				if kind == "control" {
					wantA = 23
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("projection category, identity or LKI")
				}
			}
		})
	}
}

func TestRONUnionDamageTargetInvalid(t *testing.T) {
	for _, body := range []string{
		`Targeted([AnyTarget()],ActionList([]))`,
		`Targeted([AnyTargetOfAPlayersChoice],ActionList([]))`,
		`Targeted([AnyTargetOfAPlayersChoice(You)],ActionList([]))`,
		`Targeted([AnyTargetOfAPlayersChoice(Opponent())],ActionList([]))`,
		`Targeted([AnyTargetOfAPlayersChoice(Opponent,Opponent)],ActionList([]))`,
		`ActionList([TapPermanent(AnyTargetAsAPermanent)])`,
		`Targeted([TargetPlayer(AnyPlayer)],ActionList([TapPermanent(AnyTargetAsAPermanent)]))`,
		`Targeted([TargetPlayerOrPermanent(AnyPlayer,IsCardtype(Planeswalker))],ActionList([TapPermanent(AnyTargetAsAPermanent)]))`,
		`Targeted([AnyTarget],ActionList([TapPermanent(AnyTargetAsAPermanent())]))`,
		`Targeted([AnyTarget],ActionList([TapPermanent(Ref_AnyTarget)]))`,
		`Targeted([AnyTarget],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Ref_AnyTarget1)])]))`,
		`Targeted([AnyTarget,AnyTarget],ActionList([TapPermanent(AnyTargetAsAPermanent)]))`,
		`Targeted([AnyTarget,AnyTarget],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Ref_AnyTarget)])]))`,
		`Targeted([AnyTarget,AnyTarget],ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Ref_AnyTarget3)])]))`,
		`Targeted([AnyTarget,TargetPermanent(AnyPermanent)],ActionList([]))`,
		`Modal_ChooseOne([Targeted([AnyTarget],ActionList([])),ActionList([TapPermanent(AnyTargetAsAPermanent)])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(`+body+`)])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONUnionDamageTargetRemainingRecords(t *testing.T) {
	for _, tc := range []struct{ set, name, diagnostic string }{
		{"LEA", "Disintegrate", ""}, {"LEB", "Disintegrate", ""}, {"2ED", "Disintegrate", ""}, {"3ED", "Disintegrate", ""}, {"CED", "Disintegrate", ""}, {"CEI", "Disintegrate", ""}, {"LEG", "Pyrotechnics", "TargetedDistributed"},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			f, err := os.Open(fmt.Sprintf("../../../../cards/mtgish/%s.lines.ron", tc.set))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			nodes, err := Parse(f, tc.set)
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range nodes {
				if n.Fields["Name"].String == tc.name {
					_, err := Compile(n)
					if tc.name == "Disintegrate" {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
						t.Fatalf("remaining blocker: %v", err)
					}
					return
				}
			}
			t.Fatal("record missing")
		})
	}
}

func TestRONUnionDamageTargetProjectionRule(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget],ActionList([If(PermanentPassesFilter(AnyTargetAsAPermanent,IsCardtype(Creature)),[CreatePermanentRuleEffectUntil(AnyTargetAsAPermanent,[CantBeRegenerated],UntilEndOfTurn)])])))`)
	for _, kind := range []string{"creature", "player", "walker", "return", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 5), b.PlayerID())
			target := p.ID()
			if kind == "player" {
				target = b.PlayerID()
			}
			if kind == "walker" {
				target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID()).ID()
			}
			card := factory()
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			if kind == "return" {
				g.BouncePermanentToHand(p)
				g.PutOnBattlefield(p.Card, b.PlayerID())
			}
			g.ResolveStack()
			if kind == "cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			want := kind == "creature" || kind == "cycle"
			if g.FindPermanent(p.ID()).HasKeyword(core.CantRegenerate) != want {
				t.Fatal("projected rule identity")
			}
			g.RunStepWithPriority(core.Cleanup)
			if g.FindPermanent(p.ID()).HasKeyword(core.CantRegenerate) {
				t.Fatal("projected rule did not expire")
			}
		})
	}
}

func TestRONUnionDamageTargetNumberedConsumers(t *testing.T) {
	for _, operation := range []string{
		`SpellDealsDamage(ThisSpell,Integer(2),Ref_AnyTarget1)`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(2),Ref_AnyTarget1)])`,
	} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget,AnyTargetOfAPlayersChoice(Opponent)],ActionList([CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(Integer(1),Ref_AnyTarget1),[PreventThatDamage]),`+operation+`])))`)
		g, a, b := opponentTargetGame()
		b.requested = []uuid.UUID{a.PlayerID()}
		card := factory()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID(), b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if b.Life() != 19 || a.Life() != 20 {
			t.Fatal("numbered consumer binding")
		}
	}
}

func TestRONUnionDamageTargetPaymentAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent)]),Targeted([AnyTarget,AnyTargetOfAPlayersChoice(Opponent)],ActionList([PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Ref_AnyTarget1),DamageToRecipients(Integer(2),Ref_AnyTarget2)])])))`)
	firstCard, secondCard := factory(), factory()
	for _, card := range []mage.Card{firstCard, secondCard} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	left, right := firstCard.Abilities()[0].(*mage.ActionDefinition), secondCard.Abilities()[0].(*mage.ActionDefinition)
	for i := range 2 {
		if left.Targets()[i] == right.Targets()[i] {
			t.Fatal("factory target shared")
		}
	}
	missing := mage.NewArtifact("Missing", "", mage.WithAbility(mage.NewAction(mage.ActionActivated, mage.WithEffects(left.Effects()...))))
	if mage.ValidateCardDefinition(missing.Name(), missing) == nil {
		t.Fatal("undeclared numbered bindings accepted")
	}
	g, a, b := opponentTargetGame()
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	second := g.PutOnBattlefield(secondCard, a.PlayerID())
	b.requested = []uuid.UUID{a.PlayerID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{b.PlayerID(), b.PlayerID()}); err == nil {
		t.Fatal("unpaid activation")
	}
	if g.FindPermanent(first.ID()).Tapped || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Blue, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{b.PlayerID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	b.requested = []uuid.UUID{b.PlayerID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{a.PlayerID(), a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if a.Life() != 19 || b.Life() != 18 {
		t.Fatal("later instance binding")
	}
	g.ResolveTopOfStack()
	if a.Life() != 17 || b.Life() != 17 {
		t.Fatal("earlier instance binding overwritten")
	}
}

func TestRONUnionDamageTargetWitchesCastingAndRetarget(t *testing.T) {
	g, a, b := opponentTargetGame()
	card := repositoryCard(t, "ARN", "Cuombajj Witches")
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid creature cast")
	}
	a.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	if source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 3 {
		t.Fatal("creature metadata")
	}
	a.requested = []uuid.UUID{b.PlayerID()}
	b.requested = []uuid.UUID{a.PlayerID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID(), a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	b.candidates = nil
	a.candidates = nil
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || len(a.candidates) != 0 || len(b.candidates) != 2 || copied.Targets[0] != a.PlayerID() || copied.Targets[1] != a.PlayerID() {
		t.Fatalf("copy controller target choices: calls=%d/%d", len(a.candidates), len(b.candidates))
	}
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 20 {
		t.Fatal("retargeted copy lost group identities")
	}
}

func TestRONUnionDamageTargetSingleChoiceAndInventory(t *testing.T) {
	g, a, b := opponentTargetGame()
	b.requested = []uuid.UUID{a.PlayerID()}
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTargetOfAPlayersChoice(Opponent)],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Ref_AnyTarget)])))`)
	card := factory()
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("singular opponent choice")
	}
	report, err := InspectFS(fstest.MapFS{"inventory.ron": &fstest.MapFile{Data: []byte(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([AnyTargetOfAPlayersChoice(Opponent)],ActionList([TapPermanent(AnyTargetAsAPermanent)])))])`)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "Opponent" && signature.Role == "player_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("union chooser has no semantic role")
	}
}
