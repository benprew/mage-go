package mage

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_ability_activation_event_test.go ---
func TestAbilityActivatedCapturesActivator(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithActivator(actor), GainLifeFor(FromPlayer(actor), Fixed(2)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	one := NewArtifact("One", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	two := NewArtifact("Two", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(one, a.PlayerID())
	g.PutOnBattlefield(two, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), two.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack size = %d, want 4", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original players")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 23 {
		t.Fatalf("life totals = %d, %d, want 25 and 23", a.Life(), b.Life())
	}
}

func TestAbilityActivatedAvailabilityAndPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	costly := NewArtifact("Costly", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), ManaCostOf("{9}")))
	g.PutOnBattlefield(costly, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), costly.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed activation created a trigger")
	}
	mana := NewArtifact("Mana", "", WithAbility(NewManaAbility(Blue)))
	g.PutOnBattlefield(mana, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), mana.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("mana activation did not create one trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("pending trigger lost its controller")
	}
	g.PhaseOut(watcher.ID())
	otherMana := NewArtifact("Other mana", "", WithAbility(NewManaAbility(Blue)))
	g.PutOnBattlefield(otherMana, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), otherMana.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}
}

func TestAbilityActivatedDeclaration(t *testing.T) {
	actor := BindPlayer("actor")
	if got := InspectDefinition(ActivatedAbilityEvent().WithActivator(actor)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatal("activation pattern inspection lost its binding")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Good", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithActivator(actor), GainLifeFor(FromPlayer(actor), Fixed(1))))),
		},
		{
			Name:    "nil activator",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithActivator(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate activator",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithActivator(actor).WithActivator(actor), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound activator",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), GainLifeFor(FromPlayer(actor), Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "non-battlefield source",
			Card:    NewInstant("Bad spell", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "valid modal",
			Card: NewEnchantment("Modal", "", WithAbility(WhenEventModes(
				ActivatedAbilityEvent().WithActivator(actor),
				ModeOf("Gain", GainLifeFor(FromPlayer(actor), Fixed(1))),
				ModeOf("Draw", DrawFor(FromPlayer(actor), Fixed(1)))))),
		},
	})
}

// --- from composable_ability_source_event_test.go ---
func TestAbilityActivatedSourceCaptureAfterSacrificeAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("activated source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, b.PlayerID())
	artifact := NewArtifact("Source", "", WithKeyword(Lifelink), WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, a.PlayerID())
	old := g.FindPermanent(artifact.ID()).incarnationID
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("life = %d, want 18", game.GetPlayer(b.PlayerID()).Life())
		}
		if game.GetPlayer(a.PlayerID()).Life() != 23 {
			t.Fatalf("source controller life = %d, want 23", game.GetPlayer(a.PlayerID()).Life())
		}
		if game.FindPermanent(artifact.ID()).incarnationID == old {
			t.Fatal("source return kept old incarnation")
		}
	}
}

func TestAbilityActivatedSourceDeclaration(t *testing.T) {
	source := BindDamageSource("source")
	if got := InspectDefinition(ActivatedAbilityEvent().WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatal("source capture missing from inspection")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Good", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithSource(source), DealDamageFrom(FromDamageSource(source), You, Fixed(1))))),
		},
		{
			Name:    "nil source capture",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithSource(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate source capture",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithSource(source).WithSource(source), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound source",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), DealDamageFrom(FromDamageSource(source), You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestAbilityActivatedSourceSurvivesPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(watcher, b.PlayerID())
	artifact := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.PhaseOut(artifact.ID())
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatalf("life = %d, want 19", b.Life())
	}
}

// --- from composable_any_draw_card_event_test.go ---
func TestAnyDrawEventCardVisitsAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDrawsCard(drawer).WithCard(drawn),
		IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(FromPlayer(drawer), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	landA, landB := NewLand("A land"), NewLand("B land")
	a.AddToLibrary(landA)
	b.AddToLibrary(landB)
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("draw triggers = %d", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatalf("clone life = %d, %d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
	if _, ok := b.DiscardCard(landB.ID()); !ok {
		t.Fatal("drawn land absent")
	}
	b.AddToHand(landB)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("stale draw visit counted: %d, %d", a.Life(), b.Life())
	}
}

func TestOpponentDrawEventCardAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDrawsCard(drawer).WithCard(drawn),
		IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(You, Fixed(1))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	a.AddToLibrary(NewLand("Own"))
	b.AddToLibrary(NewLand("Other"))
	g.PlayerDrawCard(a)
	phase := g.PhaseOut(watcher.ID())
	g.PlayerDrawCard(b)
	g.PhaseIn(phase)
	b.AddToLibrary(NewLand("Later"))
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("opponent draw triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("queued draw trigger life = %d", a.Life())
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(watcher.ID(), b.PlayerID(), func(*Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 0 {
		t.Fatalf("replaced draw: replacement = %d, stack = %d", replaced, g.StackSize())
	}
}

func TestAnyDrawEventCardDeclaration(t *testing.T) {
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	if info := InspectDefinition(PlayerDrawsCard(drawer).WithCard(drawn)); info.Role != "event-pattern" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
	body := IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(You, Fixed(1)))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer).WithCard(drawn), body))),
		},
		{
			Name:          "nil card",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer).WithCard(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "unnamed card",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDrawsCard(drawer).WithCard(BindCard("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "repeated card",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer).WithCard(drawn).WithCard(drawn), body))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing card producer",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), body))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "scope escape",
			Card: NewEnchantment("Scope", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer).WithCard(drawn), body)),
				WithAbility(Activate(Costs(), body))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_any_draw_event_test.go ---
func TestPlayerDrawsCardEitherPlayerAndCompletedDraws(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToLibrary(NewArtifact("A card", ""))
	for range 2 {
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(a)
	if g.StackSize() != 0 {
		t.Fatal("triggers entered stack before placement")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if len(obj.Targets) != 0 {
			t.Fatal("captured drawer became a target")
		}
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 24 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 0 {
		t.Fatalf("replacement = %d, stack = %d", replaced, g.StackSize())
	}
}

func TestPlayerDrawsCardSourceLifecycleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.PlayerDrawCard(a)
	g.PhaseIn(phased)
	g.PlayerDrawCard(a)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if obj.Controller == a.PlayerID() {
			g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			break
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerDrawsCardDeclaration(t *testing.T) {
	drawer := BindPlayer("drawer")
	if info := InspectDefinition(PlayerDrawsCard(drawer)); info.Role != "event-pattern" || info.Text != "a player draws a card" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
		},
		{
			Name:          "nil player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(nil), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty player name",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(BindPlayer("")), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(BindPlayer("other")), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "scope escape",
			Card:          NewEnchantment("Scope", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1)))), WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_any_life_gain_event_test.go ---
func TestPlayerGainsLifeCapturesActualEventAndEachPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("amount")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerGainsLife(player).WithAmount(amount),
		LoseLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 3})
	g.gainLifeWithEvent(a, 1)
	g.gainLifeWithEvent(b, 1)
	g.gainLifeWithEvent(a, 0)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerGainsLifeSourceAndFrameLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("amount")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerGainsLife(player).WithAmount(amount),
		LoseLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.gainLifeWithEvent(b, 1)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.gainLifeWithEvent(b, 2)
	g.gainLifeWithEvent(a, 4)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[1], b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 21 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestPlayerGainsLifeDeclaration(t *testing.T) {
	player, amount := BindPlayer("gainer"), BindNumber("amount")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(player).WithAmount(amount), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
		},
		{
			Name:          "nil player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(nil), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil amount",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(player).WithAmount(nil), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(BindPlayer("")), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(player).WithAmount(BindNumber("gainer")), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "repeated",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(player).WithAmount(amount).WithAmount(amount), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "one amount",
		},
		{
			Name:          "missing",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerGainsLife(player), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	if got := InspectDefinition(PlayerGainsLife(player).WithAmount(amount)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}

// --- from composable_any_life_loss_event_test.go ---
func TestPlayerLosesLifeCapturesEachCompletedLoss(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("loser"), BindNumber("lost")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerLosesLife(player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PlayerLoseLife(a, 2)
	g.PlayerLoseLife(b, 3)
	g.PlayerLoseLife(b, 0)
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack = %d, want 3", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerLosesLifeSimultaneousAndSourceLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("loser"), BindNumber("lost")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerLosesLife(player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, Fixed(1)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 21 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestPlayerLosesLifeDeclaration(t *testing.T) {
	player, amount := BindPlayer("loser"), BindNumber("lost")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(player).WithAmount(amount), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
		},
		{
			Name:          "nil player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(nil), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil amount",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(player).WithAmount(nil), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(BindPlayer("")), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate name",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(player).WithAmount(BindNumber("loser")), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "repeated",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(player).WithAmount(amount).WithAmount(amount), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "one amount",
		},
		{
			Name:          "missing",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerLosesLife(player), GainLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	if got := InspectDefinition(PlayerLosesLife(player).WithAmount(amount)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}

// --- from composable_artifact_activation_event_test.go ---
func TestArtifactAbilityWithoutTapCostEvent(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want int
	}{
		{"without tap", ManaCostOf("{0}"), 1},
		{"with tap", Tap(), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			artifact := BindPermanent("activated artifact")
			controller := BindPlayer("artifact controller")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				ArtifactAbilityActivatedWithoutTapCost().WithArtifact(artifact).WithController(controller),
				Sequence(AddCountersTo(From(artifact), Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			source := NewArtifact("Source", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), tc.cost))
			g.PutOnBattlefield(source, b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != 1+tc.want {
				t.Fatalf("stack=%d, want %d", got, 1+tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := g.FindPermanent(source.ID()).Counters[Charge]; got != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
			if b.Life() != 21+tc.want {
				t.Fatalf("life=%d, want %d", b.Life(), 21+tc.want)
			}
		})
	}
}

func TestArtifactAbilityWithoutTapCostFiltersAndLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("controller")
	watcher := NewAura("Watcher", "", WithCastTarget(TargetArtifact()), WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost().AttachedToSource().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	attached := NewArtifact("Attached", "", WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	attached.SetOwner(b.PlayerID())
	g.PutOnBattlefield(attached, b.PlayerID())
	g.Attach(watcher.ID(), attached.ID())
	other := NewArtifact("Other", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), ManaCostOf("{0}")))
	g.PutOnBattlefield(other, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), attached.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack=%d, want 3", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 24 {
		t.Fatalf("life=%d, want 24", b.Life())
	}
}

func TestArtifactAbilityWithoutTapCostDeclaration(t *testing.T) {
	artifact := BindPermanent("artifact")
	controller := BindPlayer("controller")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Valid", "", WithAbility(WhenEvent(
				ArtifactAbilityActivatedWithoutTapCost().WithArtifact(artifact).WithController(controller),
				GainLifeFor(FromPlayer(controller), Fixed(1))))),
		},
		{
			Name:    "nil artifact",
			Card:    NewEnchantment("Invalid", "", WithAbility(WhenEvent(ArtifactAbilityActivatedWithoutTapCost().WithArtifact(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil controller",
			Card:    NewEnchantment("Invalid", "", WithAbility(WhenEvent(ArtifactAbilityActivatedWithoutTapCost().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate artifact",
			Card:    NewEnchantment("Invalid", "", WithAbility(WhenEvent(ArtifactAbilityActivatedWithoutTapCost().WithArtifact(artifact).WithArtifact(artifact), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate controller",
			Card:    NewEnchantment("Invalid", "", WithAbility(WhenEvent(ArtifactAbilityActivatedWithoutTapCost().WithController(controller).WithController(controller), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "missing controller binding",
			Card: NewEnchantment("Missing", "", WithAbility(WhenEvent(
				ArtifactAbilityActivatedWithoutTapCost(), GainLifeFor(FromPlayer(controller), Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestArtifactAbilityWithoutTapCostExcludesManaTap(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Mana artifact", "", WithAbility(NewManaAbility(Blue)))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("tap-cost mana activation matched no-tap event")
	}
	planned := NewArtifact("Planned mana", "", WithActivatedAbility(AddMana(Green, 1), Tap()))
	g.PutOnBattlefield(planned, b.PlayerID())
	if err := g.AutoTapForCost(b.PlayerID(), ManaCost{Green: 1}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("planned tap-cost mana activation matched no-tap event")
	}
}

func TestArtifactAbilityWithoutTapCostFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactAbilityActivatedWithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Costly", "", WithActivatedAbility(GainLifeFor(You, Fixed(1)), ManaCostOf("{9}")))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("failed payment published an activation event")
	}
}

// --- from composable_artifact_tap_event_test.go ---
func TestArtifactBecomesTappedCapturesParticipants(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		artifact, opponent    bool
		phaseSource, phaseObj bool
		returnObj             bool
		wantTriggers          int
		wantCounters          int
	}{
		{"own artifact", true, false, false, false, false, 1, 1},
		{"opponent artifact", true, true, false, false, false, 1, 1},
		{"nonartifact", false, true, false, false, false, 0, 0},
		{"phased source", true, true, true, false, false, 0, 0},
		{"phased artifact", true, true, false, true, false, 0, 0},
		{"returned artifact", true, true, false, false, true, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			object := BindPermanent("tapped artifact")
			controller := BindPlayer("tap controller")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				ArtifactBecomesTapped().WithArtifact(object).WithController(controller),
				Sequence(AddCountersTo(From(object), Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			var tapped Card
			if tc.artifact {
				tapped = NewArtifact("Artifact", "")
			} else {
				tapped = NewLand("Land")
			}
			owner := a.PlayerID()
			if tc.opponent {
				owner = b.PlayerID()
			}
			g.PutOnBattlefield(tapped, owner)
			if tc.phaseSource {
				g.PhaseOut(watcher.ID())
			}
			if tc.phaseObj {
				g.PhaseOut(tapped.ID())
			}
			g.TapPermanent(g.FindPermanent(tapped.ID()))
			g.TapPermanent(g.FindPermanent(tapped.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.wantTriggers {
				t.Fatalf("stack=%d, want %d", got, tc.wantTriggers)
			}
			if tc.returnObj {
				g.RemoveFromBattlefield(g.FindPermanent(tapped.ID()))
				g.PutOnBattlefield(tapped, a.PlayerID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := g.FindPermanentIncludingPhased(tapped.ID()).Counters[Charge]; got != tc.wantCounters {
				t.Fatalf("counters=%d, want %d", got, tc.wantCounters)
			}
			wantA, wantB := 20, 20
			if tc.wantTriggers == 1 {
				if tc.opponent {
					wantB++
				} else {
					wantA++
				}
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life A=%d B=%d, want %d %d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestArtifactBecomesTappedPendingCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().WithController(controller), GainLifeFor(FromPlayer(controller), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "")
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.TapPermanent(g.FindPermanent(artifact.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("pending trigger or copy is absent")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestArtifactBecomesTappedDeclaration(t *testing.T) {
	artifact := BindPermanent("artifact")
	controller := BindPlayer("controller")
	if got := InspectDefinition(ArtifactBecomesTapped()); got.Role != "event-pattern" || got.Text != "artifact becomes tapped" {
		t.Fatalf("inspection=%+v", got)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(
				ArtifactBecomesTapped().WithArtifact(artifact).WithController(controller),
				GainLifeFor(FromPlayer(controller), Fixed(1))))),
		},
		{
			Name:    "nil artifact",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithArtifact(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty artifact name",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithArtifact(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate artifact",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithArtifact(artifact).WithArtifact(artifact), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil controller",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty controller name",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithController(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate controller",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithController(controller).WithController(controller), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "conflicting names",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(ArtifactBecomesTapped().WithArtifact(artifact).WithController(BindPlayer("artifact")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:          "unbound artifact producer",
			Card:          NewArtifact("Unbound", "", WithAbility(WhenEvent(ArtifactBecomesTapped(), AddCountersTo(From(artifact), Charge, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "unbound controller producer",
			Card:          NewArtifact("Unbound", "", WithAbility(WhenEvent(ArtifactBecomesTapped(), GainLifeFor(FromPlayer(controller), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "bad source",
			Card:          NewInstant("Bad source", "", WithAbility(WhenEvent(ArtifactBecomesTapped(), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
}

// --- from composable_attached_creature_death_test.go ---
func TestAttachedCreatureDiesCapturesEventState(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		controller := BindPlayer("former controller")
		toughness := BindNumber("former toughness")
		aura := NewAura("Watcher", "", WithAbility(WhenEvent(
			AttachedCreatureDies().WithController(controller).WithToughness(toughness),
			GainLifeFor(FromPlayer(controller), FromNumber(toughness)))))
		aura.SetOwner(a.PlayerID())
		host := NewCreature("Host", "", 2, 3)
		host.SetOwner(a.PlayerID())
		other := NewCreature("Other", "", 1, 4)
		other.SetOwner(a.PlayerID())
		g.PutOnBattlefield(host, b.PlayerID())
		g.PutOnBattlefield(other, a.PlayerID())
		g.PutOnBattlefield(aura, a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(AttachedPermanent, Fixed(0), Fixed(2))), aura.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if simultaneous {
			g.DestroyPermanents([]*Permanent{g.FindPermanent(host.ID()), g.FindPermanent(other.ID()), g.FindPermanent(aura.ID())})
		} else {
			g.DestroyPermanent(g.FindPermanent(other.ID()))
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatalf("simultaneous=%v stack=%d, want 1", simultaneous, g.StackSize())
		}
		clone := g.Clone()
		clone.ResolveStack()
		if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatal("clone did not keep event state separate")
		}
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.zones.ClearLKI()
		g.PutOnBattlefield(host, a.PlayerID())
		g.ResolveStack()
		if a.Life() != 20 || b.Life() != 30 {
			t.Fatalf("life=(%d,%d), want (20,30)", a.Life(), b.Life())
		}
	}
}

func TestAttachedCreatureDiesRejectsOtherEvents(t *testing.T) {
	for _, change := range []string{"unattached", "other", "phased", "exile", "land", "returned host"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		aura := NewAura("Watcher", "", WithAbility(WhenEvent(AttachedCreatureDies(), GainLifeFor(You, Fixed(1)))))
		host := NewCreature("Host", "", 2, 2)
		host.SetOwner(b.PlayerID())
		g.PutOnBattlefield(host, b.PlayerID())
		g.PutOnBattlefield(aura, a.PlayerID())
		if change != "unattached" {
			g.Attach(aura.ID(), host.ID())
		}
		switch change {
		case "other":
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(b.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			g.DestroyPermanent(g.FindPermanent(other.ID()))
		case "phased":
			g.PhaseOut(aura.ID())
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		case "exile":
			g.ExilePermanent(g.FindPermanent(host.ID()))
		case "land":
			land := NewLand("Land")
			land.SetOwner(b.PlayerID())
			g.PutOnBattlefield(land, b.PlayerID())
			g.Attach(aura.ID(), land.ID())
			g.DestroyPermanent(g.FindPermanent(land.ID()))
		case "returned host":
			g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
			g.PutOnBattlefield(host, b.PlayerID())
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		default:
			g.DestroyPermanent(g.FindPermanent(host.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatalf("%s: stack=%d, want 0", change, g.StackSize())
		}
	}
}

func TestAttachedCreatureDiesValidation(t *testing.T) {
	b := BindNumber("toughness")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewAura("Good", "", WithAbility(WhenEvent(AttachedCreatureDies().WithToughness(b), GainLifeFor(You, FromNumber(b))))),
		},
		{
			Name:          "nil controller binding",
			Card:          NewAura("Bad", "", WithAbility(WhenEvent(AttachedCreatureDies().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
	})
}

// --- from composable_controlled_death_test.go ---
func TestCreatureYouControlDies(t *testing.T) {
	for _, scenario := range []string{"own", "opponent", "stolen", "lost", "self", "simultaneous", "control after", "departed", "copy", "clone", "phased", "exile", "noncreature", "token"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureYouControlDies(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if scenario == "noncreature" {
				other = NewArtifact("Other", "")
			}
			if scenario == "token" {
				other = NewToken("Other", 1, 1, []CardType{TypeCreature}, nil)
			}
			owner := a.PlayerID()
			if scenario == "opponent" || scenario == "stolen" {
				owner = b.PlayerID()
			}
			other.SetOwner(owner)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, owner)
			if scenario == "stolen" || scenario == "lost" {
				controller := a.PlayerID()
				if scenario == "lost" {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: other.ID(), ControllerID: controller, Duration: Indefinite})
			}
			count := 1
			switch scenario {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
				count = 2
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
				count = 0
			default:
				if scenario == "phased" {
					g.PhaseOut(source.ID())
				}
				g.DestroyPermanent(g.FindPermanent(other.ID()))
				if scenario == "opponent" || scenario == "lost" || scenario == "phased" || scenario == "noncreature" {
					count = 0
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != count || a.Life() != 20 {
				t.Fatalf("stack=%d want=%d life=%d", g.StackSize(), count, a.Life())
			}
			switch scenario {
			case "control after":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "departed":
				g.ExilePermanent(g.FindPermanent(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
					t.Fatal("clone state is shared")
				}
			}
			g.ResolveStack()
			wantB := 20
			if scenario == "copy" {
				wantB++
			}
			if a.Life() != 20+count || b.Life() != wantB {
				t.Fatalf("life=%d/%d", a.Life(), b.Life())
			}
		})
	}
}

// --- from composable_controller_damage_event_test.go ---
func TestControllerIsDealtDamageActualAmountAndControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 5, attacker.ID())
	g.DealDamageToPlayer(b, 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
		t.Fatalf("pending stack = %d", g.StackSize())
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatalf("life after captured damage = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
		game.DealDamageToPlayer(a, 1, attacker.ID())
		game.DealDamageToPlayer(b, 2, attacker.ID())
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatalf("pending after control change = %d", game.StackSize())
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 19 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatalf("life after control change = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestControllerIsDealtDamageReplacementAbsenceAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 1, 4)
	g.PutOnBattlefield(victim, a.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: a.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayer(a, 2, victim.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("redirected damage matched player recipient")
	}
	g.effects.replacements = nil
	g.PlayerLoseLife(a, 1)
	g.DealDamageToPlayer(a, 0, victim.ID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 2, victim.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("life loss, zero, or prevented damage triggered")
	}
	permanent := g.FindPermanent(source.ID())
	permanent.PhasedOut = true
	g.DealDamageToPlayer(a, 1, victim.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	permanent.PhasedOut = false
	g.DealDamageToPlayer(a, 1, victim.ID())
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending source departure stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 18 {
		t.Fatalf("captured trigger controller life = %d", a.Life())
	}
}

func TestControllerIsDealtDamageRegistration(t *testing.T) {
	amount := BindNumber("amount")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewEnchantment("Valid", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount))))), WantErr: false},
		{Name: "nil", Card: NewEnchantment("Nil", "", WithAbility(WhenEvent(ControllerIsDealtDamage(nil), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "non-nil"},
		{Name: "empty", Card: NewEnchantment("Empty", "", WithAbility(WhenEvent(ControllerIsDealtDamage(BindNumber("")), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "name"},
		{Name: "unbound", Card: NewEnchantment("Unbound", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, FromNumber(amount))))), WantErr: true, WantErrSubstr: "producer"},
		{Name: "other ability", Card: NewEnchantment("Other", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, Fixed(1)))), WithAbility(WhenEvent(Enters(This), GainLifeFor(You, FromNumber(amount))))), WantErr: true, WantErrSubstr: "producer"},
		{Name: "bad zone", Card: NewInstant("Instant", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount))))), WantErr: true, WantErrSubstr: "battlefield"},
	})
	if got := InspectDefinition(ControllerIsDealtDamage(amount)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}

func TestControllerIsDealtDamageSimultaneousCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEventModes(
		ControllerIsDealtDamage(amount),
		ModeOf("Gain", GainLifeFor(You, FromNumber(amount))),
		ModeOf("Draw", DrawFor(You, Fixed(1))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 1, 4)
	second := NewCreature("Second", "", 1, 4)
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 2, g.CaptureDamageSource(first.ID())),
		PlayerDamageAssignment(a, 3, g.CaptureDamageSource(second.ID())),
	)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("damage events = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatalf("copied captured amounts = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

// --- from composable_creature_death_color_test.go ---
func TestCreatureDiesWithoutColorUsesDepartureState(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mana      string
		setColors []Color
		want      int
	}{
		{"red", "{R}", nil, 1},
		{"colorless", "", nil, 1},
		{"black", "{B}", nil, 0},
		{"black and red", "{B}{R}", nil, 0},
		{"became black", "{R}", []Color{Black}, 0},
		{"ceased to be black", "{B}", []Color{Blue}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithoutColor(Black), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			victim := NewCreature("Victim", tc.mana, 2, 2)
			victim.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.setColors != nil {
				if err := ApplyEffect(g, Until(EndOfTurn, SetColors(This, tc.setColors...)), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			g.zones.ClearLKI()
			g.PutOnBattlefield(victim, b.PlayerID())
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life=%d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestCreatureDiesWithoutColorSimultaneousAndSourceState(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		watcher := NewCreature("Watcher", "{B}", 1, 1, WithAbility(WhenEvent(
			AnotherCreatureDies().WithoutColor(Black).Nontoken(), GainLifeFor(You, Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		red := NewCreature("Red", "{R}", 1, 1)
		red.SetOwner(b.PlayerID())
		black := NewCreature("Black", "{B}", 1, 1)
		black.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(red, b.PlayerID())
		g.PutOnBattlefield(black, b.PlayerID())
		if phase {
			g.PhaseOut(watcher.ID())
			g.DestroyPermanent(g.FindPermanent(red.ID()))
		} else {
			g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(red.ID()), g.FindPermanent(black.ID())})
		}
		g.PutTriggersOnStack()
		want := 1
		if phase {
			want = 0
		}
		if got := g.StackSize(); got != want {
			t.Fatalf("phase=%t stack=%d, want %d", phase, got, want)
		}
	}
}

func TestCreatureDiesWithoutColorRejectsInvalidColor(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "Colorless",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithoutColor(Colorless), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
		{
			Name: "AnyColor",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithoutColor(AnyColor), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
		{
			Name: "Color(99)",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithoutColor(Color(99)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
		{
			Name: "duplicate filter",
			Card: NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithoutColor(Black).WithoutColor(Red), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
	})
}

func TestCreatureDiesWithoutColorIgnoresNonDeathsAndReturnedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().WithoutColor(Black), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "{R}", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.ExilePermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("exile matched death")
	}
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life=%d, want 21", a.Life())
	}
}

// --- from composable_creature_death_controller_test.go ---
func TestCreatureDiesCapturesFormerController(t *testing.T) {
	for _, tc := range []struct {
		name        string
		pattern     func(*PlayerBinding) EventPattern
		ownerB      bool
		controllerB bool
		want        int
	}{
		{"any creature", func(p *PlayerBinding) EventPattern { return CreatureDies().WithController(p) }, true, false, 1},
		{"controlled creature", func(p *PlayerBinding) EventPattern { return CreatureYouControlDies().WithController(p) }, true, false, 1},
		{"another controlled creature", func(p *PlayerBinding) EventPattern { return AnotherCreatureYouControlDies().WithController(p) }, true, false, 1},
		{"opponent creature", func(p *PlayerBinding) EventPattern { return CreatureYouControlDies().WithController(p) }, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			former := BindPlayer("former controller")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(tc.pattern(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1)
			owner := a.PlayerID()
			if tc.ownerB {
				owner = b.PlayerID()
			}
			victim.SetOwner(owner)
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			g.PutOnBattlefield(victim, owner)
			if tc.controllerB != tc.ownerB {
				controller := a.PlayerID()
				if tc.controllerB {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: victim.ID(), ControllerID: controller, Duration: Indefinite})
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), tc.want)
			}
			if tc.want == 1 {
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone changed original life")
				}
			}
			g.ResolveStack()
			wantA, wantB := 20, 20
			if tc.want == 1 {
				if tc.controllerB {
					wantB++
				} else {
					wantA++
				}
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=(%d,%d), want (%d,%d)", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestCreatureDiesControllerCopyAndSimultaneousDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureYouControlDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("copy changed event controller: life=(%d,%d)", a.Life(), b.Life())
	}
}

func TestCreatureDiesControllerSeparateSimultaneousEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("event controller leaked between triggers: life=(%d,%d)", a.Life(), b.Life())
	}
}

func TestCreatureDiesControllerPhasedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed a death")
	}
}

func TestCreatureDiesControllerValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Factory: func() Card {
				b := BindPlayer("controller")
				return NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(b), GainLifeFor(FromPlayer(b), Fixed(1)))))
			},
			WantErr: false,
		},
		{
			Name: "nil",
			Factory: func() Card {
				return NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(nil), GainLifeFor(You, Fixed(1)))))
			},
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name: "undeclared read",
			Factory: func() Card {
				b := BindPlayer("controller")
				return NewCreature("Read", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(FromPlayer(b), Fixed(1)))))
			},
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "duplicate name",
			Factory: func() Card {
				a := BindPlayer("same")
				b := BindPlayer("same")
				return NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithController(a), Sequence(SnapshotPlayer(You, b), GainLifeFor(FromPlayer(a), Fixed(1))))))
			},
			WantErr:       true,
			WantErrSubstr: "duplicate binding name",
		},
	})
}

// --- from composable_creature_death_owner_test.go ---
func TestCreatureDiesOwnerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	owner, controller := BindPlayer("owner"), BindPlayer("controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithOwner(owner).WithController(controller),
		Sequence(GainLifeFor(FromPlayer(owner), Fixed(2)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	first.SetOwner(b.PlayerID())
	second.SetOwner(a.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 24 {
		t.Fatalf("life=(%d,%d), want (25,24)", a.Life(), b.Life())
	}
}

func TestCreatureDiesOwnerCaptureSourceAndPhasing(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		owner := BindPlayer("owner")
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
			CreatureDies().WithOwner(owner), GainLifeFor(FromPlayer(owner), Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		victim := NewCreature("Victim", "", 1, 1)
		victim.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(victim, a.PlayerID())
		if phase {
			g.PhaseOut(watcher.ID())
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phased source observed death")
			}
			continue
		}
		g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatalf("stack=%d, want 2", g.StackSize())
		}
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != 21 {
			t.Fatalf("life=(%d,%d), want (21,21)", a.Life(), b.Life())
		}
	}
}

func TestCreatureDiesOwnerCaptureValidation(t *testing.T) {
	owner := BindPlayer("owner")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithOwner(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithOwner(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding name",
		},
		{
			Name:          "unbound",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(FromPlayer(owner), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithOwner(owner).WithController(owner), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "different bindings",
		},
		{
			Name: "valid",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureDies().Nontoken().WithOwner(owner), GainLifeFor(FromPlayer(owner), Fixed(1))))),
			WantErr: false,
		},
	})
}

// --- from composable_creature_death_power_test.go ---
func TestCreatureDeathPowerCapturedBeforeDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("former power")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(3), Fixed(0))), victim.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	clone := g.Clone()
	g.zones.ClearLKI()
	g.PutOnBattlefield(victim, b.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 25 || b.Life() != 25 || clone.GetPlayer(a.PlayerID()).Life() != 25 {
		t.Fatalf("life original=(%d,%d) clone=%d, want (25,25) and 25", a.Life(), b.Life(), clone.GetPlayer(a.PlayerID()).Life())
	}
}

func TestCreatureDeathPowerPhasedSourceDoesNotObserve(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("power")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 3, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatalf("stack=%d life=%d, want 0 and 20", g.StackSize(), a.Life())
	}
}

func TestCreatureDeathPowerSimultaneousAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power := BindNumber("power")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	watcher.SetOwner(a.PlayerID())
	one := NewCreature("One", "", 2, 2)
	two := NewCreature("Two", "", 4, 4)
	one.SetOwner(a.PlayerID())
	two.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(one, a.PlayerID())
	g.PutOnBattlefield(two, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(one.ID()), g.FindPermanent(two.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 20 {
		t.Fatalf("life=(%d,%d), want (26,20)", a.Life(), b.Life())
	}
}

func TestCreatureDeathPowerDeclaration(t *testing.T) {
	power := BindNumber("power")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(power), GainLifeFor(You, FromNumber(power))))), WantErr: false},
		{Name: "nil", Card: NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(nil), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "power binding"},
		{Name: "empty", Card: NewCreature("Empty", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(BindNumber("")), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "binding name"},
		{Name: "wrong event", Card: NewCreature("Wrong", "", 1, 1, WithAbility(WhenEvent(PermanentDies().WithPower(BindNumber("power")), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "creature"},
		{Name: "unbound", Card: NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, FromNumber(BindNumber("power")))))), WantErr: true, WantErrSubstr: "producer"},
		{Name: "duplicate", Card: NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithPower(BindNumber("a")).WithPower(BindNumber("b")), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "one power"},
	})
}

// --- from composable_creature_death_toughness_test.go ---
func TestCreatureDiesCapturesToughnessAtDetection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	toughness := BindNumber("former toughness")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(toughness), GainLifeFor(You, FromNumber(toughness)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.zones.ClearLKI()
	g.PutOnBattlefield(victim, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatalf("life=%d, want 23", a.Life())
	}
}

func TestCreatureDiesToughnessValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil",
			Card:          NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name: "missing producer",
			Factory: func() Card {
				b := BindNumber("toughness")
				return NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, FromNumber(b)))))
			},
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "duplicate name",
			Factory: func() Card {
				a, b := BindNumber("same"), BindNumber("same")
				return NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(a), Sequence(SnapshotNumber(Fixed(1), b), GainLifeFor(You, FromNumber(a))))))
			},
			WantErr:       true,
			WantErrSubstr: "duplicate binding name",
		},
		{
			Name: "non dominating",
			Factory: func() Card {
				b := BindNumber("toughness")
				return NewCreature("Non Dominating", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(b), GainLifeFor(You, FromNumber(b)))), WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(b)))))
			},
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "valid",
			Factory: func() Card {
				b := BindNumber("toughness")
				return NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(b), GainLifeFor(You, FromNumber(b)))))
			},
			WantErr: false,
		},
	})
}

func TestCreatureDiesCapturesOwnToughness(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("former toughness")
	creature := NewCreature("Self", "", 2, 6, WithAbility(WhenEvent(CreatureDies().WithToughness(n), GainLifeFor(You, FromNumber(n)))))
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	g.zones.ClearLKI()
	g.ResolveStack()
	if a.Life() != 26 {
		t.Fatalf("life=%d, want 26", a.Life())
	}
}

func TestCreatureDiesToughnessIgnoresNonDeaths(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("toughness")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithToughness(n), GainLifeFor(You, FromNumber(n)))))
	watcher.SetOwner(a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(land.ID()))
	g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtZoneChange, SourceID: land.ID(), FromZone: ZoneBattlefield, ToZone: ZoneGraveyard})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("stack=%d, want 0", g.StackSize())
	}
}

func TestCreatureDiesToughnessSimultaneousAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("former toughness")
	former := BindPlayer("former controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies().WithController(former).WithToughness(n),
		GainLifeFor(FromPlayer(former), FromNumber(n)))))
	watcher.SetOwner(a.PlayerID())
	first := NewCreature("First", "", 2, 2)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 3, 4)
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(3))), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 29 || a.Life() != 20 {
		t.Fatalf("clone life=%d, original life=%d; want 29 and 20", clone.GetPlayer(a.PlayerID()).Life(), a.Life())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.zones.ClearLKI()
	g.ResolveStack()
	if a.Life() != 34 || b.Life() != 20 {
		t.Fatalf("life=(%d,%d), want (34,20)", a.Life(), b.Life())
	}
}

// --- from composable_creature_dies_test.go ---
func TestComposableCreatureDies(t *testing.T) {
	for _, scenario := range []string{"self", "other", "simultaneous", "noncreature", "exile", "phased", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1,
				WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if scenario == "noncreature" {
				other = NewArtifact("Other", "")
			}
			other.SetOwner(b.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			want := 21
			switch scenario {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.MutablePermanent(source.ID()).Damage = 1
				g.MutablePermanent(other.ID()).Damage = 1
				g.CheckStateBasedActions()
				want = 22
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
				want = 20
			default:
				if scenario == "phased" {
					g.PhaseOut(source.ID())
				}
				g.DestroyPermanent(g.FindPermanent(other.ID()))
				if scenario == "noncreature" || scenario == "phased" {
					want = 20
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != want-20 || a.Life() != 20 {
				t.Fatal("wrong pending death count or premature resolution")
			}
			if g.StackSize() > 0 && len(g.StackPeek().Targets) != 0 {
				t.Fatal("death participant became a target")
			}
			if scenario == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != want || a.Life() != 20 || g.StackSize() != 1 {
					t.Fatal("clone resolution changed the original")
				}
			}
			g.ResolveStack()
			if a.Life() != want {
				t.Fatalf("life = %d, want %d", a.Life(), want)
			}
		})
	}
}

// --- from composable_damage_event_source_test.go ---
func TestReceivesDamageSourceCaptureAfterDepartureAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("amount")
	source := BindDamageSource("source")
	victim := NewCreature("Victim", "", 2, 7, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, FromNumber(amount)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	card := NewCreature("Source", "", 2, 2, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	captured := g.CaptureDamageSource(card.ID())
	other := g.PutOnBattlefield(NewCreature("Other Source", "", 1, 2), b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealSimultaneousDamage(
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 2, captured),
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, g.CaptureDamageSource(other.ID())),
	)
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("saved source damage: A=%d B=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
		if game.FindPermanent(card.ID()).incarnationID != returned.incarnationID {
			t.Fatal("returned source changed")
		}
	}
}

func TestReceivesDamageSourceCaptureAfterSpellLeavesStack(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("spell source")
	victim := NewCreature("Victim", "", 1, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, BindNumber("amount")).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	spell := NewInstant("Spell", "", nil)
	spell.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: spell, Controller: a.PlayerID(), SourceID: spell.ID()})
	captured := g.CaptureDamageSource(spell.ID())
	end()
	if !captured.Valid() {
		t.Fatal("spell source was not captured")
	}
	g.DealDamageToPermanentFromSource(g.FindPermanent(victim.ID()), 1, captured)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatalf("spell source did not survive stack departure: life = %d", b.Life())
	}
}

func TestReceivesDamageSourceCapturePreventionPhasingAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	victim := NewCreature("Victim", "", 1, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, BindNumber("amount")).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage created a trigger")
	}
	g.PhaseOut(victim.ID())
	g.DealDamageToPermanent(g.FindPermanentIncludingPhased(victim.ID()), 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased recipient created a trigger")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, attacker.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.PhaseOut(attacker.ID())
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 18 {
		t.Fatalf("copied captures lost phased source: life = %d", b.Life())
	}
}

func TestReceivesDamageSourceCaptureAfterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	victim := NewCreature("Victim", "", 1, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, BindNumber("amount")).WithSource(source),
		DealDamageFrom(FromDamageSource(source), ActivePlayer, Fixed(1)))))
	g.PutOnBattlefield(victim, b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(attacker.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("redirected damage triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatalf("redirected source was not captured: A=%d B=%d", a.Life(), b.Life())
	}
}

func TestReceivesDamageSourceDeclaration(t *testing.T) {
	source := BindDamageSource("source")
	amount := BindNumber("amount")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 3, WithAbility(WhenEvent(
				ReceivesDamage(This, amount).WithSource(source),
				DealDamageFrom(FromDamageSource(source), You, FromNumber(amount))))),
			WantErr: false,
		},
		{
			Name: "nil source",
			Card: NewCreature("Bad", "", 1, 3, WithAbility(WhenEvent(
				ReceivesDamage(This, amount).WithSource(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty source",
			Card: NewCreature("Bad", "", 1, 3, WithAbility(WhenEvent(
				ReceivesDamage(This, amount).WithSource(BindDamageSource("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate source",
			Card: NewCreature("Bad", "", 1, 3, WithAbility(WhenEvent(
				ReceivesDamage(This, amount).WithSource(source).WithSource(source), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound source",
			Card: NewCreature("Bad", "", 1, 3, WithAbility(WhenEvent(
				ReceivesDamage(This, amount), DealDamageFrom(FromDamageSource(source), You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	if got := InspectDefinition(ReceivesDamage(This, amount).WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}

// --- from composable_damage_recipient_event_test.go ---
func TestDealsDamageCapturesPermanentRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	victim := NewCreature("Victim", "", 3, 3)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
	g.PutOnBattlefield(victim, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	if g.FindPermanent(victim.ID()).Counters[P1P1] != 0 || clone.FindPermanent(victim.ID()).Counters[P1P1] != 0 {
		t.Fatal("returned recipient used an old binding")
	}
}

func TestDealsDamagePermanentRecipientPresentAndAbsent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if got := g.FindPermanent(victim.ID()).Counters[P1P1]; got != 2 {
		t.Fatalf("counters = %d", got)
	}
	g.PhaseOut(source.ID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseOut(victim.ID())
	g.DealDamageToPermanent(g.FindPermanentIncludingPhased(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased recipient was damaged")
	}
}

func TestDealsDamagePermanentRecipientDeclaration(t *testing.T) {
	bound := BindPermanent("recipient")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, BindNumber("amount")).WithPermanent(bound),
				AddCountersTo(From(bound), P1P1, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "nil recipient",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, BindNumber("amount")).WithPermanent(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty recipient",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, BindNumber("amount")).WithPermanent(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate recipient",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, BindNumber("amount")).WithPermanent(bound).WithPermanent(bound), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "player damage with recipient",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToPlayer(This, BindPlayer("player")).WithPermanent(bound), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound recipient",
			Card: NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, BindNumber("amount")), AddCountersTo(From(bound), P1P1, Fixed(1))))),
			WantErr: true,
		},
	})
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(bound),
		AddCountersTo(From(bound), P1P1, Fixed(1)))))
	if meta := InspectDefinition(good.Abilities()[0]); !strings.Contains(fmt.Sprintf("%+v", meta), "event-permanent-capture") {
		t.Fatalf("missing capture: %+v", meta)
	}
}

func TestDealsDamagePermanentRecipientFollowsFinalDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("redirected damage stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if got := g.FindPermanent(victim.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("redirected recipient counters = %d", got)
	}
}

func TestDealsDamagePermanentRecipientsInOneBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := BindPermanent("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithPermanent(recipient),
		AddCountersTo(From(recipient), P1P1, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	first := NewCreature("First", "", 5, 5)
	second := NewCreature("Second", "", 5, 5)
	first.SetOwner(b.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	if err := ApplyEffect(g, DealDamageEach(This,
		DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(1)),
		source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.ResolveStack()
	for _, card := range []Card{first, second} {
		if got := g.FindPermanent(card.ID()).Counters[P1P1]; got != 1 {
			t.Fatalf("%s counters = %d", card.Name(), got)
		}
	}
}

// --- from composable_damaged_creature_death_test.go ---
func TestCreatureDiesDamagedByThisTurnIdentity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		damage        bool
		returnVictim  bool
		returnSource  bool
		phaseSource   bool
		phaseVictim   bool
		preventDamage bool
		want          int
	}{
		{name: "dealt damage", damage: true, want: 1},
		{name: "no damage"},
		{name: "returned victim", damage: true, returnVictim: true},
		{name: "returned source", damage: true, returnSource: true},
		{name: "phased source", damage: true, phaseSource: true},
		{name: "phased victim", damage: true, phaseVictim: true, want: 1},
		{name: "prevented", damage: true, preventDamage: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewCreature("Source", "", 1, 5, WithAbility(WhenEvent(
				CreatureDies().DamagedByThisTurn(), GainLifeFor(You, Fixed(1)))))
			sourceCard.SetOwner(a.PlayerID())
			victimCard := NewCreature("Victim", "", 1, 5)
			victimCard.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(sourceCard.Name(), sourceCard); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if tc.preventDamage {
				g.AddPreventionShield(victim.ID(), 1)
			}
			if tc.damage {
				g.DealDamageToPermanentFromSource(victim, 1, g.CaptureDamageSource(source.ID()))
			}
			if tc.returnVictim {
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			}
			if tc.returnSource {
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(sourceCard, a.PlayerID())
			}
			if tc.phaseSource {
				g.PhaseOut(source.ID())
			}
			if tc.phaseVictim {
				g.PhaseIn(g.PhaseOut(victim.ID()))
			}
			g.DestroyPermanent(victim)
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.want {
				t.Fatalf("life=%d, want %d", a.Life(), 20+tc.want)
			}
		})
	}
}

func TestCreatureDiesDamagedByThisTurnSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 5, WithAbility(WhenEvent(
		CreatureDies().DamagedByThisTurn(), GainLifeFor(You, Fixed(1)))))
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	victimCard := NewCreature("Victim", "", 1, 5)
	victimCard.SetOwner(b.PlayerID())
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	g.DealDamageToPermanentFromSource(victim, 1, g.CaptureDamageSource(source.ID()))
	g.DestroyPermanents([]*Permanent{source, victim})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
}

func TestCreatureDiesDamagedByThisTurnRejectsInvalidPattern(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "PermanentDies",
			Card:    NewArtifact("Invalid", "", WithAbility(WhenEvent(PermanentDies().DamagedByThisTurn(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "DamagedByThisTurn duplicate",
			Card:    NewArtifact("Invalid", "", WithAbility(WhenEvent(CreatureDies().DamagedByThisTurn().DamagedByThisTurn(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestCreatureDiesDamagedByThisTurnRequiresEventIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
	g.DealDamageToPermanentFromSource(victim, 1, g.CaptureDamageSource(source.ID()))
	cond := eventCreatureDamagedBySourceThisTurn{}
	if cond.CheckTriggerCond(&GameEvent{SourceID: victim.ID()}, g, source.ID(), a.PlayerID()) {
		t.Fatal("an event without incarnation matched damage history")
	}
}

// --- from composable_death_card_test.go ---
func TestDeathEventCardVisit(t *testing.T) {
	for _, leave := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		dead := BindCard("dead")
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
		watcher.SetOwner(a.PlayerID())
		victim := NewCreature("Victim", "", 2, 2)
		victim.SetOwner(b.PlayerID())
		if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(victim, b.PlayerID())
		g.DestroyPermanent(g.FindPermanent(victim.ID()))
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatalf("stack=%d", g.StackSize())
		}
		if leave {
			card, _, ok := g.MoveFromAnyGraveyard(victim.ID(), ZoneExile)
			if !ok {
				t.Fatal("first move failed")
			}
			b.AddToGraveyard(card)
		}
		clone := g.Clone()
		clone.ResolveStack()
		if len(b.Hand()) != 0 {
			t.Fatal("clone changed original hand")
		}
		g.ResolveStack()
		if got := len(b.Hand()); got != map[bool]int{false: 1, true: 0}[leave] {
			t.Fatalf("leave=%v, hand=%d", leave, got)
		}
	}
}

func TestDeathEventCardDeclaration(t *testing.T) {
	b := BindCard("dead")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithCard(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies().WithCard(BindCard("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding name",
		},
		{
			Name:          "missing",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), ReturnCardToHand(FromCard(b))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "wrong scope",
			Card: NewCreature("Wrong scope", "", 1, 1,
				WithAbility(WhenEvent(CreatureDies().WithCard(b), GainLifeFor(You, Fixed(1)))),
				WithAbility(Activate(Costs(), ReturnCardToHand(FromCard(b))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "duplicate",
			Card: NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithCard(BindCard("same")),
				Sequence(ReturnCardToHand(FromCard(BindCard("same"))), GainLifeFor(You, Fixed(1)))))),
			WantErr: true,
		},
	})
}

func TestDeathEventCardSimultaneousAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("dead")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	watcher.SetOwner(a.PlayerID())
	first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
	first.SetOwner(b.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 2 || len(b.Graveyard()) != 0 {
		t.Fatalf("hand=%d, graveyard=%d", len(b.Hand()), len(b.Graveyard()))
	}
}

func TestDeathEventCardPhasedSourceAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("dead")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	watcher.SetOwner(a.PlayerID())
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	token.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(token, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(token.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("token trigger stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if len(b.Hand()) != 0 {
		t.Fatal("token became a card in hand")
	}
	g.PhaseOut(watcher.ID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatalf("phased source trigger stack=%d", g.StackSize())
	}
}

func TestDeathEventCardSourceDiesSimultaneously(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("other dead creature")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if len(b.Hand()) != 1 || len(a.Graveyard()) != 1 {
		t.Fatalf("victim hand=%d, source graveyard=%d", len(b.Hand()), len(a.Graveyard()))
	}
}

// --- from composable_death_count_test.go ---
func TestCreatureDeathCountActions(t *testing.T) {
	for _, action := range []string{"destroy", "sacrifice", "graveyard", "token", "artifact", "exile", "return", "regenerate", "indestructible"} {
		t.Run(action, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 2)
			if action == "indestructible" {
				card = NewCreature("Subject", "", 2, 2, WithKeyword(Indestructible))
			}
			if action == "artifact" {
				card = NewArtifact("Subject", "")
			}
			card.SetOwner(b.PlayerID())
			g.PutOnBattlefield(card, b.PlayerID())
			p := g.MutablePermanent(card.ID())
			want := 1
			switch action {
			case "destroy", "artifact":
				g.DestroyPermanent(p)
				if action == "artifact" {
					want = 0
				}
			case "sacrifice":
				g.DoSacrifice(p)
			case "graveyard":
				g.PutPermanentIntoGraveyard(p)
			case "token":
				p.IsToken = true
				g.DestroyPermanent(p)
			case "exile":
				g.ExilePermanent(p)
				want = 0
			case "return":
				if err := ApplyEffect(g, ReturnToHand(This), card.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 0
			case "regenerate":
				g.AddRegenerationShield(card.ID())
				g.DestroyPermanent(p)
				want = 0
			case "indestructible":
				g.DestroyPermanent(p)
				want = 0
			}
			for _, player := range []Player{a, b} {
				ctx := &EffectContext{Game: g, Controller: player.PlayerID()}
				for range 3 {
					if got := CreaturesDiedThisTurn().resolveValue(ctx); got != want {
						t.Fatalf("count = %d, want %d", got, want)
					}
				}
			}
		})
	}
}

func TestCreatureDeathCountResolutionCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 2, WithAbility(At(EachEndStep, GainLifeFor(You, CreaturesDiedThisTurn()))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("end-step ability must use the stack")
	}
	original := g.StackObjects()[0]
	g.CopyStackObjectDirect(original, b.PlayerID(), false)
	g.ExilePermanent(g.FindPermanent(source.ID()))
	victims := make([]*Permanent, 0, 2)
	for _, player := range []Player{a, b} {
		c := NewCreature("Victim", "", 1, 1)
		c.SetOwner(player.PlayerID())
		g.PutOnBattlefield(c, player.PlayerID())
		victims = append(victims, g.FindPermanent(c.ID()))
	}
	g.DestroyPermanents(victims)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("copies must read resolution-time deaths in their own game")
	}
	clone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if CreaturesDiedThisTurn().resolveValue(&EffectContext{Game: clone}) != 0 || CreaturesDiedThisTurn().resolveValue(&EffectContext{Game: g}) != 2 {
		t.Fatal("turn reset must clear only the cloned count")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("source departure changed the turn count")
	}
}

func TestCreatureDeathCountSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Victim", "", 1, 1)
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	n := BindNumber("before")
	body := Sequence(SnapshotNumber(CreaturesDiedThisTurn(), n), Sacrifice(This),
		GainLifeFor(You, FromNumber(n)), GainLifeFor(You, CreaturesDiedThisTurn()))
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}

// --- from composable_death_not_sacrificed_test.go ---
func TestPermanentDiesNotSacrificedOutcomeAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		PermanentDies().OfType(TypeArtifact).YouControlled().NotSacrificed().WithController(former),
		GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(watcher.Abilities()[0]).Children[0].Text; !strings.Contains(got, "not sacrificed") {
		t.Fatalf("inspection = %q", got)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := func(name string) *Permanent {
		card := NewArtifact(name, "")
		card.SetOwner(a.PlayerID())
		return g.PutOnBattlefield(card, a.PlayerID())
	}
	g.DoSacrifice(artifact("Sacrificed"))
	ownedByB := NewArtifact("Controlled by A", "")
	ownedByB.SetOwner(b.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(ownedByB, a.PlayerID()))
	opponent := NewArtifact("Controlled by B", "")
	opponent.SetOwner(a.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(opponent, b.PlayerID()))
	g.DestroyPermanent(artifact("Destroyed"))
	replaced := artifact("Exiled")
	g.AddExileIfWouldGoToGraveyardThisTurn(replaced.ID(), watcher.ID())
	g.DestroyPermanent(replaced)
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("triggers = %d, want 2", got)
	}
	if len(g.StackPeek().Targets) != 0 {
		t.Fatal("event controller became a target")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatalf("life = (%d, %d), want (23, 20)", a.Life(), b.Life())
	}
}

func TestPermanentDiesNotSacrificedSourcePresence(t *testing.T) {
	for _, tc := range []struct {
		name string
		prep func(*Game, *Permanent)
		want int
	}{
		{"present", nil, 1},
		{"phased", func(g *Game, p *Permanent) { g.PhaseOut(p.ID()) }, 0},
		{"absent", func(g *Game, p *Permanent) { g.RemoveFromBattlefield(p) }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				PermanentDies().OfType(TypeArtifact).NotSacrificed(), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewArtifact("Victim", "")
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.prep != nil {
				tc.prep(g, g.FindPermanent(watcher.ID()))
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPermanentDiesNotSacrificedSimultaneousSourceDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		PermanentDies().OfType(TypeArtifact).YouControlled().NotSacrificed(),
		GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewArtifact("Victim", "")
	victim.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("triggers = %d, want 2", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("life = %d, want 22", got)
	}
}

func TestPermanentDiesNotSacrificedDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Good", "", WithAbility(WhenEvent(
				PermanentDies().NotSacrificed(), GainLifeFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "duplicate NotSacrificed",
			Card: NewArtifact("Bad", "", WithAbility(WhenEvent(
				PermanentDies().NotSacrificed().NotSacrificed(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate YouControlled",
			Card: NewArtifact("Bad", "", WithAbility(WhenEvent(
				PermanentDies().YouControlled().YouControlled(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

// --- from composable_death_observation_test.go ---
type deathCountConditionForTest int

func (deathCountConditionForTest) stateCondition() {}
func (n deathCountConditionForTest) checkState(g GameReader, _ StateContext) bool {
	return g.CreatureDeaths() >= int(n)
}
func (deathCountConditionForTest) inspectDefinition() definitionNode {
	return definitionNode{Role: "condition", Text: "a creature died this turn"}
}

func TestDeathObservationIndividual(t *testing.T) {
	for _, action := range []string{"destroy", "sacrifice", "graveyard", "token"} {
		t.Run(action, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			ability := func() Ability {
				return WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn()), WithInterveningIf(deathCountConditionForTest(1)))
			}
			g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(ability())), a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1, WithAbility(ability()))
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			p := g.MutablePermanent(victim.ID())
			switch action {
			case "destroy":
				g.DestroyPermanent(p)
			case "sacrifice":
				g.DoSacrifice(p)
			case "graveyard":
				g.PutPermanentIntoGraveyard(p)
			case "token":
				p.IsToken = true
				g.DestroyPermanent(p)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("death trigger resolved without the stack")
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 21 || g.CreatureDeaths() != 1 {
				t.Fatalf("life = %d/%d, deaths = %d; want 21/21 and 1", a.Life(), b.Life(), g.CreatureDeaths())
			}
		})
	}
}

func TestDeathObservationSacrificeEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	trigger := NewTriggered(EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool { return game.CreatureDeaths() == 1 })
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(trigger)), a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DoSacrifice(g.FindPermanent(victim.ID()))
	g.ResolveStack()
	if a.Life() != 21 || g.CreatureDeaths() != 1 {
		t.Fatalf("life = %d, deaths = %d; want 21 and 1", a.Life(), g.CreatureDeaths())
	}
}

func TestDeathObservationStateTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(WhenState(deathCountConditionForTest(1), Sacrifice(This)))), a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	if len(g.triggers.pending) != 1 {
		t.Fatalf("pending state triggers = %d, want 1 before another observation", len(g.triggers.pending))
	}
}

func TestDeathObservationSimultaneousAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ab := WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn()), WithInterveningIf(deathCountConditionForTest(2)))
	first := NewCreature("Observer", "", 1, 1, WithAbility(ab))
	second := NewCreature("Victim", "", 1, 1)
	protected := NewCreature("Survivor", "", 1, 1, WithKeyword(Indestructible))
	state := NewEnchantment("State observer", "", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool { return game.CreatureDeaths() == 1 })))
	g.PutOnBattlefield(state, a.PlayerID())
	for _, card := range []Card{first, second, protected} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	clone := g.Clone()
	clone.DestroyPermanents([]*Permanent{clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID()), clone.FindPermanent(protected.ID())})
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 24 {
		t.Fatalf("clone life = %d, want 24 from two triggers that each see two deaths", got)
	}
	if clone.CreatureDeaths() != 2 || g.CreatureDeaths() != 0 || a.Life() != 20 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("simultaneous deaths changed the original game or counted prevented destruction")
	}
	if clone.FindPermanent(state.ID()) == nil {
		t.Fatal("state trigger observed a partial simultaneous group")
	}
}

// --- from composable_death_permanent_test.go ---
func TestDeathPermanentBindingKeepsDepartedDamageSource(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		dead := BindPermanent("dead")
		watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
			AnotherCreatureDies().WithPermanent(dead),
			Sequence(AddCountersTo(From(dead), P1P1, Fixed(1)),
				DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(2))))))
		watcher.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
			t.Fatal(err)
		}
		victim := NewCreature("Victim", "", 2, 2, WithKeyword(Lifelink))
		victim.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(victim, b.PlayerID())
		if simultaneous {
			g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
		} else {
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatalf("simultaneous=%v: stack=%d", simultaneous, g.StackSize())
		}
		g.PutOnBattlefield(victim, a.PlayerID())
		clone := g.Clone()
		clone.ResolveStack()
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("clone changed original")
		}
		g.ResolveStack()
		if got := g.FindPermanent(victim.ID()).Counters[P1P1]; got != 0 {
			t.Fatalf("returned incarnation gained %d counters", got)
		}
		if a.Life() != 18 || b.Life() != 22 {
			t.Fatalf("simultaneous=%v: life A=%d B=%d", simultaneous, a.Life(), b.Life())
		}
		if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("clone lost death capture")
		}
	}
}

func TestDeathPermanentBindingDeclaration(t *testing.T) {
	dead := BindPermanent("dead")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithPermanent(dead), DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "nil permanent",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithPermanent(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty permanent",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithPermanent(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate permanent",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().WithPermanent(dead).WithPermanent(dead), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound",
			Card: NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies(), DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1))))),
			WantErr: true,
		},
	})
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().WithPermanent(dead), DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1)))))
	if meta := InspectDefinition(good.Abilities()[0]); meta.Role != "event-trigger" || len(meta.Children[0].Children) == 0 {
		t.Fatalf("missing capture inspection: %+v", meta)
	}
}

func TestDeathPermanentBindingStackCopyAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindPermanent("dead")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().WithPermanent(dead),
		DealDamageFrom(DamageSourceOf(From(dead)), You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn([]uuid.UUID{watcher.ID()})
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("present source did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatalf("copy lost binding: life A=%d B=%d", a.Life(), b.Life())
	}
}

// --- from composable_death_pt_filter_test.go ---
func TestDeathPowerOrToughnessFilter(t *testing.T) {
	for _, tc := range []struct {
		name      string
		power     int
		toughness int
		change    bool
		opponent  bool
		phase     bool
		want      int
	}{
		{"low power", 1, 4, false, false, false, 1},
		{"low toughness", 4, 1, false, false, false, 1},
		{"both high", 2, 2, false, false, false, 0},
		{"changed before death", 3, 4, true, false, false, 1},
		{"opposing low creature", 1, 4, false, true, false, 0},
		{"phased source", 1, 4, false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
				CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewCreature("Victim", "", tc.power, tc.toughness)
			victim.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(victim, controller)
			if tc.change {
				if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(-2), Fixed(0))), victim.ID(), controller, nil); err != nil {
					t.Fatal(err)
				}
			}
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
			g.zones.ClearLKI()
			g.PutOnBattlefield(victim, controller)
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life = %d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestDeathPowerOrToughnessFilterBatchAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	low := NewCreature("Low", "", 1, 3)
	low.SetOwner(a.PlayerID())
	high := NewCreature("High", "", 3, 3)
	high.SetOwner(a.PlayerID())
	replaced := NewCreature("Replaced", "", 1, 3)
	replaced.SetOwner(a.PlayerID())
	for _, c := range []Card{watcher, low, high, replaced} {
		g.PutOnBattlefield(c, a.PlayerID())
	}
	g.AddExileIfWouldGoToGraveyardThisTurn(replaced.ID(), watcher.ID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(low.ID()), g.FindPermanent(high.ID()), g.FindPermanent(replaced.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("triggers = %d, want 1", got)
	}
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("life = (%d, %d), want (21, 21)", a.Life(), b.Life())
	}
}

func TestDeathPowerOrToughnessFilterDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 2, 2, WithAbility(WhenEvent(
				CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "duplicate PowerOrToughnessAtMost",
			Card: NewCreature("Bad", "", 2, 2, WithAbility(WhenEvent(
				CreatureDies().PowerOrToughnessAtMost(1).PowerOrToughnessAtMost(2), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "PermanentDies PowerOrToughnessAtMost",
			Card: NewCreature("Bad", "", 2, 2, WithAbility(WhenEvent(
				PermanentDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
	good := NewCreature("Good", "", 2, 2, WithAbility(WhenEvent(
		CreatureYouControlDies().PowerOrToughnessAtMost(1), GainLifeFor(You, Fixed(1)))))
	if text := InspectDefinition(good.Abilities()[0]).Children[0].Text; !strings.Contains(text, "power or toughness 1 or less") {
		t.Fatalf("inspection = %q", text)
	}
}

// --- from composable_death_subtype_test.go ---
func TestCreatureDeathSubtypeEventState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	goblin := NewCreature("Goblin", "", 2, 2, WithSubTypes("Goblin"))
	goblin.SetOwner(b.PlayerID())
	elf := NewCreature("Elf", "", 2, 2, WithSubTypes("Elf"))
	elf.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(goblin, b.PlayerID())
	g.PutOnBattlefield(elf, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(goblin.ID()), g.FindPermanent(elf.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want one Goblin death trigger", g.StackSize())
	}
	clone := g.Clone()
	g.zones.ClearLKI()
	g.PutOnBattlefield(goblin, b.PlayerID())
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 21 || b.Life() != 20 {
		t.Fatal("death trigger lost event subtype or clone isolation")
	}
}

func TestCreatureDeathSubtypePhasedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	goblin := NewCreature("Goblin", "", 2, 2, WithSubTypes("Goblin"))
	goblin.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(goblin, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.DestroyPermanent(g.FindPermanent(goblin.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased-out watcher observed a death")
	}
}

func TestCreatureDeathSubtypeUsesEffectiveType(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2, WithSubTypes("Elf"))
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, SetSubtypeFamily(This, SubtypeCreature, "Goblin")), victim.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want one trigger from effective subtype", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}

func TestCreatureDeathSubtypeDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "empty",
			Card:          NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().OfSubtype(""), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name:          "wrong family",
			Card:          NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().OfSubtype("Island"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "creature subtype",
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies().OfSubtype("Goblin").OfSubtype("Elf"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "one subtype",
		},
		{
			Name:          "permanent",
			Card:          NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(PermanentDies().OfType(TypeLand).OfSubtype("Goblin"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "creature",
		},
	})
}

// --- from composable_delayed_death_test.go ---
func TestWhenDiesThisTurnIncarnationAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), a.PlayerID())
	saved := BindPermanent("victim")
	effect := Sequence(Snapshot(This, saved), WhenDiesThisTurn(From(saved), GainLifeFor(You, Fixed(3))))
	if err := ApplyEffect(g, effect, victim.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.PutPermanentIntoGraveyard(clone.FindPermanent(victim.ID()))
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatal("clone lost the delayed trigger or changed the original")
	}
	g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("death did not trigger")
	}
	g.PutOnBattlefield(victim.Card, a.PlayerID())
	g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("trigger followed a returned card")
	}
}

func TestWhenDiesThisTurnValidation(t *testing.T) {
	saved := BindPermanent("saved")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Valid", "", 1, 1, WithAbility(Activate(Costs(), Sequence(Snapshot(This, saved), WhenDiesThisTurn(From(saved), GainLifeFor(You, Fixed(1))))))),
			WantErr: false,
		},
		{
			Name:          "unbound",
			Card:          NewInstant("Invalid", "", Spell(WhenDiesThisTurn(From(saved), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil target",
			Card:          NewInstant("Invalid", "", Spell(WhenDiesThisTurn(nil, GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "delayed",
		},
		{
			Name:          "nil effect",
			Card:          NewInstant("Invalid", "", Spell(WhenDiesThisTurn(This, nil))),
			WantErr:       true,
			WantErrSubstr: "delayed",
		},
	})
}

func TestWhenDiesThisTurnTargetSourceAndExpiry(t *testing.T) {
	for _, tc := range []struct {
		name string
		act  func(*Game, uuid.UUID, uuid.UUID)
		want int
	}{
		{"dies", func(g *Game, _, victim uuid.UUID) { g.PutPermanentIntoGraveyard(g.FindPermanent(victim)) }, 23},
		{"source leaves", func(g *Game, source, victim uuid.UUID) {
			g.PutPermanentIntoGraveyard(g.FindPermanent(source))
			g.PutPermanentIntoGraveyard(g.FindPermanent(victim))
		}, 23},
		{"returned before death", func(g *Game, _, victim uuid.UUID) {
			p := g.FindPermanent(victim)
			g.BouncePermanentToHand(p)
			g.PutOnBattlefield(p.Card, p.ControllerID())
			g.PutPermanentIntoGraveyard(g.FindPermanent(victim))
		}, 20},
		{"expires", func(g *Game, _, victim uuid.UUID) {
			g.triggers.ClearEndOfTurn()
			g.PutPermanentIntoGraveyard(g.FindPermanent(victim))
		}, 20},
		{"phased", func(g *Game, _, victim uuid.UUID) { g.PhaseOut(victim) }, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			target := DeclarePermanentTarget("victim", IsCreature)
			ability := NewInstant("Watch", "", Spell(WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), WithTargets(target)))
			if err := ValidateCardDefinition("Watch", ability); err != nil {
				t.Fatal(err)
			}
			g.resolution.targetSpecs = []Target{target.targetDeclaration}
			if err := ApplyEffect(g, WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			tc.act(g, source.ID(), victim.ID())
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != tc.want {
				t.Fatalf("life = %d, want %d", a.Life(), tc.want)
			}
		})
	}
}

func TestWhenDiesThisTurnSimultaneousAndReplacement(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
		target := DeclarePermanentTarget("victim", IsCreature)
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		if err := ApplyEffect(g, WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		if replaced {
			g.AddExileIfWouldGoToGraveyardThisTurn(victim.ID(), source.ID())
		}
		g.DestroyPermanents([]*Permanent{source, victim})
		g.PutTriggersOnStack()
		g.ResolveStack()
		want := 23
		if replaced {
			want = 20
		}
		if a.Life() != want {
			t.Fatalf("replaced=%t: life=%d, want %d", replaced, a.Life(), want)
		}
	}
}

func TestWhenDiesThisTurnAbsentTargetDoesNotSubscribe(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
	target := DeclarePermanentTarget("victim", IsCreature)
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.PhaseOut(victim.ID())
	if err := ApplyEffect(g, WhenDiesThisTurn(target, GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 0 {
		t.Fatal("absent target created a subscription")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	g.PutPermanentIntoGraveyard(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("a later death matched an absent target")
	}
}

// --- from composable_discard_card_event_test.go ---
func TestDiscardEventCardVisitAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := BindCard("discarded")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(BindPlayer("discarder")).WithCard(card),
		IfCardHasType(FromCard(card), TypeLand, GainLifeFor(You, Fixed(2))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land")
	b.AddToHand(land)
	g.PlayerDiscard(b, land.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatal("clone lost copied discard card")
	}
	b.RemoveFromGraveyard(land.ID())
	b.AddToHand(land)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("discard capture followed a later zone visit")
	}
}

func TestDiscardEventCardReplacementBatchAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := BindCard("discarded")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDiscardsCard(BindPlayer("player")).WithCard(card),
		IfCardHasType(FromCard(card), TypeLand, GainLifeFor(You, Fixed(1))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	b.AddToHand(NewLand("First"))
	b.AddToHand(NewLand("Second"))
	g.discardHandsTogether([]uuid.UUID{b.PlayerID()}, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("batch triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("batch life = %d", a.Life())
	}
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	b.AddToHand(NewLand("Replaced"))
	g.PlayerDiscardByEffect(b, b.Hand()[0].ID(), watcher.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("library replacement exposed a graveyard card")
	}
	phase := g.PhaseOut(watcher.ID())
	b.AddToHand(NewLand("Phased"))
	g.PlayerDiscard(b, b.Hand()[0].ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed discard")
	}
	g.PhaseIn(phase)
}

func TestDiscardEventCardDeclaration(t *testing.T) {
	card := BindCard("card")
	if info := InspectDefinition(OpponentDiscardsCard(BindPlayer("player")).WithCard(card)); info.Role != "event-pattern" || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")).WithCard(card), IfCardHasType(FromCard(card), TypeLand, GainLifeFor(You, Fixed(1)))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")).WithCard(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "unnamed",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")).WithCard(BindCard("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")).WithCard(card).WithCard(card), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")), IfCardHasType(FromCard(card), TypeLand, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "out of scope",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("player")).WithCard(card), GainLifeFor(You, Fixed(1)))),
				WithAbility(Activate(Costs(), IfCardHasType(FromCard(card), TypeLand, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_discard_event_test.go ---
func TestOpponentDiscardsCardPerCardAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(discarder), LoseLifeFor(FromPlayer(discarder), Fixed(2)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	own := NewArtifact("Own", "")
	a.AddToHand(own)
	g.PlayerDiscard(a, own.ID())
	for range 2 {
		card := NewArtifact("Other", "")
		b.AddToHand(card)
		g.PlayerDiscard(b, card.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 16 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	card := NewArtifact("Replaced", "")
	b.AddToHand(card)
	g.PutOnBattlefield(NewArtifact("Replacement", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	g.PlayerDiscardByEffect(b, card.ID(), source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("replaced discard triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 14 {
		t.Fatalf("opponent life = %d", b.Life())
	}
	if _, ok := g.PlayerDiscardByEffect(b, card.ID(), source.ID()); ok {
		t.Fatal("discarded card was removed twice")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed discard caused a trigger")
	}
}

func TestOpponentDiscardsCardIdentityAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewArtifact("Discard", "")
	b.AddToHand(card)
	phase := g.PhaseOut(source.ID())
	g.PlayerDiscard(b, card.ID())
	g.PhaseIn(phase)
	card = NewArtifact("Discard", "")
	b.AddToHand(card)
	g.PlayerDiscard(b, card.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	obj := g.StackObjects()[0]
	if len(obj.Targets) != 0 {
		t.Fatal("discarder became a target")
	}
	g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 24 {
			t.Fatalf("opponent life = %d", game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestOpponentDiscardsCardDeclaration(t *testing.T) {
	discarder := BindPlayer("discarder")
	if info := InspectDefinition(OpponentDiscardsCard(discarder)); info.Role != "event-pattern" || info.Text != "an opponent discards a card" {
		t.Fatalf("inspection = %+v", info)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDiscardsCard(nil), GainLifeFor(FromPlayer(discarder), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDiscardsCard(BindPlayer("")), GainLifeFor(FromPlayer(discarder), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound other",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDiscardsCard(BindPlayer("other")), GainLifeFor(FromPlayer(discarder), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "out of scope",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1)))),
				WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(discarder), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestOpponentDiscardsCardEventTimeControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	for _, p := range []Player{a, b} {
		card := NewArtifact("Discard", "")
		p.AddToHand(card)
		g.PlayerDiscard(p, card.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackObjects()[0].Controller != b.PlayerID() {
		t.Fatal("discard matched the wrong controller")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_draw_card_event_capture_test.go ---
func TestControllerDrawsCardIntoExactHandVisit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerDrawsCardInto(drawer, drawn),
		IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(FromPlayer(drawer), Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	land := NewLand("Land")
	a.AddToLibrary(land)
	a.AddToLibrary(NewArtifact("Artifact", ""))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("draw captures changed across clone")
	}
	if _, ok := a.DiscardCard(land.ID()); !ok {
		t.Fatal("drawn card absent from hand")
	}
	a.AddToHand(land)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("draw capture followed a new hand visit")
	}
}

func TestControllerDrawsCardIntoDeclaration(t *testing.T) {
	drawer, drawn := BindPlayer("drawer"), BindCard("drawn")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Valid", "", WithAbility(WhenEvent(ControllerDrawsCardInto(drawer, drawn), IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(You, Fixed(1)))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Nil", "", WithAbility(WhenEvent(ControllerDrawsCardInto(drawer, nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "missing",
			Card:          NewEnchantment("Missing", "", WithAbility(WhenEvent(ControllerDrawsCard(drawer), IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestControllerDrawsCardIntoCopyAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawn := BindCard("drawn")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerDrawsCardInto(BindPlayer("drawer"), drawn),
		IfCardHasType(FromCard(drawn), TypeLand, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToLibrary(NewLand("Land"))
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("draw did not create one trigger")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("copied trigger lost its exact drawn card")
	}
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("empty-library draw created a trigger")
	}
	phased := g.PhaseOut(source.ID())
	a.AddToLibrary(NewLand("Later"))
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed a draw")
	}
	g.PhaseIn(phased)
}

// --- from composable_draw_event_test.go ---
func TestControllerDrawEventActualDraws(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("drawer")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(player), GainLifeFor(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for range 2 {
		a.AddToLibrary(NewArtifact("Card", ""))
	}
	b.AddToLibrary(NewArtifact("Other", ""))
	g.PlayerDrawCard(b)
	if err := ApplyEffect(g, DrawFor(You, Fixed(3)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || !a.DrewFromEmpty() {
		t.Fatalf("stack = %d, empty = %v", g.StackSize(), a.DrewFromEmpty())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("incorrect draw recipients")
	}
	replacements := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replacements++ })
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if replacements != 1 || g.StackSize() != 0 {
		t.Fatal("replaced draw triggered")
	}
}

func TestControllerDrawEventLifecycle(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			for range 3 {
				a.AddToLibrary(NewArtifact("A", ""))
				b.AddToLibrary(NewArtifact("B", ""))
			}
			player := BindPlayer("drawer")
			body := Sequence(GainLifeFor(FromPlayer(player), Fixed(2)), GainLifeFor(You, Fixed(1)))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			pattern := ControllerDrawsCard(player)
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", GainLifeFor(You, Fixed(9))))
			}
			source := NewEnchantment("Watcher", "", WithAbility(trigger))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PlayerDrawCard(a)
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					if len(obj.Targets) != 0 {
						t.Fatal("event player became a target")
					}
					if obj.Controller == a.PlayerID() {
						game.CopyStackObjectDirect(obj, b.PlayerID(), false)
						break
					}
				}
				game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
				game.PutOnBattlefield(source, a.PlayerID())
				cloned := game.Clone()
				for _, resolved := range []*Game{game, cloned} {
					resolved.ResolveStack()
					if delayed {
						resolved.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
						resolved.PutTriggersOnStack()
						resolved.ResolveStack()
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 25 || resolved.GetPlayer(b.PlayerID()).Life() != 24 {
						t.Fatalf("life = %d, %d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
					}
				}
			}
		}
	}
}

func TestControllerDrawEventSourceIdentity(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToLibrary(NewArtifact("Card", ""))
		}
		source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(BindPlayer("drawer")), Destroy(This))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		phased := g.PhaseOut(source.ID())
		g.PlayerDrawCard(a)
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("phased source triggered")
		}
		g.PhaseIn(phased)
		g.PlayerDrawCard(a)
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
		} else {
			phased = g.PhaseOut(source.ID())
			g.PhaseIn(phased)
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("pending trigger lost")
		}
		g.ResolveStack()
		if (g.FindPermanent(source.ID()) != nil) != returned {
			t.Fatal("incorrect source incarnation")
		}
	}
}

func TestControllerDrawEventValidation(t *testing.T) {
	p := BindPlayer("drawer")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(p), GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(nil), GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(BindPlayer("")), GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound other",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(BindPlayer("other")), GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "out of scope",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(ControllerDrawsCard(p), GainLifeFor(FromPlayer(p), Fixed(1)))),
				WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_event_amount_test.go ---
func TestDamageEventAmountLifecycle(t *testing.T) {
	for _, modal := range []bool{false, true} {
		for _, delayed := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player, amount := BindPlayer("damaged"), BindNumber("damage amount")
			pattern := DealsDamageToPlayer(This, player).WithAmount(amount)
			body := GainLifeFor(FromPlayer(player), FromNumber(amount))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", DrawFor(You, Fixed(1))))
			}
			source := NewCreature("Source", "", 1, 1, WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.AddPreventionShield(b.PlayerID(), 2)
			g.DealDamageToPlayer(b, 5, source.ID())
			g.DealDamageToPlayer(a, 1, source.ID())
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(source, b.PlayerID())
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					game.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				stackClone := game.Clone()
				for _, resolved := range []*Game{game, stackClone} {
					resolved.ResolveStack()
					if delayed {
						delayedClone := resolved.Clone()
						for _, final := range []*Game{resolved, delayedClone} {
							final.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
							final.PutTriggersOnStack()
							final.ResolveStack()
							if final.GetPlayer(a.PlayerID()).Life() != 21 || final.GetPlayer(b.PlayerID()).Life() != 23 {
								t.Fatal("delayed event amount lost or shared")
							}
						}
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 23 {
						t.Fatal("event amount lost or shared")
					}
					if FromNumber(amount).resolveValue(&EffectContext{Game: resolved}) != 0 {
						t.Fatal("amount escaped resolution")
					}
				}
			}
		}
	}
}

func TestDamageEventAmountValidation(t *testing.T) {
	p, n := BindPlayer("player"), BindNumber("amount")
	read := GainLifeFor(You, FromNumber(n))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToOpponent(This, p).WithAmount(n), read))),
			WantErr: false,
		},
		{
			Name:          "unbound",
			Card:          NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "unrelated",
			Card:          NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(Enters(This), read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil",
			Card:          NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p).WithAmount(nil), read))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p).WithAmount(BindNumber("")), read))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p).WithAmount(BindNumber("player")), read))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name: "cross ability",
			Card: NewCreature("Cross ability", "", 1, 1,
				WithAbility(WhenEvent(DealsDamageToPlayer(This, p).WithAmount(n), read)), WithAbility(At(YourUpkeep, read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDamageEventAmountFrameIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("amount")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	capture := &eventAmountCapture{n, 3}
	if err := capture.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := (&eventAmountCapture{n, 9}).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromNumber(n).resolveValue(ctx) != 3 {
		t.Fatal("clone shared number map")
	}
	clone.resolution.Clear()
	if len(clone.resolution.numberBindings) != 0 {
		t.Fatal("clear retained number map")
	}
	if err := ApplyEffect(g, GainLifeFor(You, FromNumber(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || FromNumber(n).resolveValue(ctx) != 3 {
		t.Fatal("nested resolution did not isolate and restore number map")
	}
	g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	loop := ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), Sequence(
		GainLifeFor(You, FromNumber(n)), &eventAmountCapture{n, 9}))
	if err := loop.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 26 || FromNumber(n).resolveValue(ctx) != 3 {
		t.Fatal("loop did not isolate and restore number map")
	}
}

func TestDamageEventAmountAnyRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, amount), GainLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 1, 10)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: other.ID(), TargetID: a.PlayerID(), Amount: 9})
	g.FireEvent(GameEvent{Type: EvtLifeLost, SourceID: source.ID(), TargetID: a.PlayerID(), Amount: 9})
	g.DealDamageToPermanent(g.FindPermanent(other.ID()), 4, source.ID())
	g.DealDamageToPlayer(b, 2, source.ID())
	g.AddPreventionShield(b.PlayerID(), 3)
	g.DealDamageToPlayer(b, 3, source.ID())
	g.DealDamageToPlayer(b, 0, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 18 {
		t.Fatal("damage amounts or filtering incorrect")
	}
}

func TestDamageEventAmountCopyRetarget(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	amount, target := BindNumber("damage"), DeclarePlayerTarget("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, amount), GainLifeFor(target, FromNumber(amount)), WithTargets(target))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 3, source.ID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil {
		t.Fatal("missing trigger")
	}
	b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if copied == nil || copied.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("targets were not independent")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("retargeting changed the captured amount")
	}
}

// --- from composable_event_opponent_test.go ---
func TestDamageOpponentBindingValidation(t *testing.T) {
	p := BindPlayer("opponent")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToOpponent(This, p), DrawFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "nil",
			Card: NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToOpponent(This, nil), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name: "empty",
			Card: NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToOpponent(This, BindPlayer("")), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name: "scope error",
			Card: NewCreature("Scope", "", 1, 1,
				WithAbility(WhenEvent(DealsDamageToOpponent(This, p), DrawFor(FromPlayer(p), Fixed(1)))),
				WithAbility(WhenEvent(Enters(This), DrawFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDamageOpponentBindingFiltering(t *testing.T) {
	for _, combat := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged opponent")
		source := NewCreature("Source", "", 2, 2, WithAbility(WhenEvent(
			DealsDamageToOpponent(This, player), GainLifeFor(FromPlayer(player), Fixed(3)))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		g.damage.resolvingCombatDamage = combat
		g.DealDamageToPlayer(a, 1, source.ID())
		g.DealDamageToPlayer(b, 1, a.PlayerID())
		g.AddPreventionShield(b.PlayerID(), 2)
		g.DealDamageToPlayer(b, 2, source.ID())
		g.DealDamageToPlayer(b, 0, source.ID())
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: source.ID(), Amount: 1, Flag: combat})
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("unmatched damage triggered")
		}
		g.DealDamageToPlayer(b, 2, source.ID())
		g.PutTriggersOnStack()
		if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
			t.Fatal("opponent damage must produce one trigger without targets")
		}
		g.ResolveStack()
		if a.Life() != 19 || b.Life() != 20 {
			t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
		}
	}
}

func TestDamageOpponentBindingControlCopyClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged opponent")
	source := NewCreature("Source", "", 2, 2, WithAbility(WhenEvent(
		DealsDamageToOpponent(This, player), DiscardAtRandom(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for _, name := range []string{"First", "Second", "Third"} {
		b.AddToHand(NewSorcery(name, ""))
	}
	g.DealDamageToPlayer(b, 1, source.ID())
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.DealDamageToPlayer(b, 1, source.ID())
	g.DealDamageToPlayer(a, 1, source.ID())
	g.SetRandomResults([]int{1, 0, 7})
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatalf("control change produced %d triggers", game.StackSize())
		}
		for _, obj := range game.StackObjects() {
			if obj.Controller == a.PlayerID() {
				if game.CopyStackObjectDirect(obj, b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
				break
			}
		}
		game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
		game.PutOnBattlefield(source, a.PlayerID())
		game.ResolveStack()
		hand := game.GetPlayer(b.PlayerID()).Hand()
		if len(hand) != 1 || hand[0].Name() != "Third" || game.RandIntn(10) != 7 {
			t.Fatal("capture, random queue, or copy changed the discard recipient")
		}
		if game == clone && len(b.Hand()) != 3 {
			t.Fatal("clone changed the original hand")
		}
	}
}

// --- from composable_event_player_test.go ---
func TestDamagePlayerBinding(t *testing.T) {
	for _, combat := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged player")
		source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
			DealsDamageToPlayer(This, player), GainLifeFor(FromPlayer(player), Fixed(3)))))
		source.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(source, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: a.PlayerID(), Amount: 1, Flag: combat})
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 2, Flag: combat})
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		clone := g.Clone()
		for _, game := range []*Game{g, clone} {
			game.PutTriggersOnStack()
			if game.StackSize() != 2 {
				t.Fatalf("stack size = %d", game.StackSize())
			}
			for _, obj := range game.StackObjects() {
				if len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 {
					t.Fatal("event participant became a target")
				}
			}
			game.ResolveStack()
			if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 {
				t.Fatal("event participant lost or shared")
			}
		}
	}
}

func TestDamagePlayerBindingValidation(t *testing.T) {
	p := BindPlayer("player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Test", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), DrawFor(FromPlayer(p), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "missing",
			Card:          NewCreature("Test", "", 1, 1, WithAbility(WhenEvent(Enters(This), DrawFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil",
			Card:          NewCreature("Test", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, nil), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewCreature("Test", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, BindPlayer("")), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
	})
}

func TestDamagePlayerBindingCopyAndTargets(t *testing.T) {
	for _, modal := range []bool{false, true} {
		a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged")
		target := DeclarePlayerTarget("target")
		body := Sequence(GainLifeFor(FromPlayer(player), Fixed(3)), GainLifeFor(target, Fixed(1)))
		trigger := WhenEvent(DealsDamageToPlayer(This, player), body, WithTargets(target))
		if modal {
			trigger = WhenEventModes(DealsDamageToPlayer(This, player), ModeOf("Life", body, WithTargets(target)), ModeOf("Other", DrawFor(You, Fixed(1))))
		}
		source := NewCreature("Source", "", 1, 1, WithAbility(trigger))
		source.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		perm := g.PutOnBattlefield(source, a.PlayerID())
		g.DealDamageToPlayer(b, 1, source.ID())
		a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
		g.PutTriggersOnStack()
		original := g.StackPeek()
		if original == nil || len(original.Targets) != 1 {
			t.Fatal("missing declared target")
		}
		g.RemoveFromBattlefield(perm)
		g.PutOnBattlefield(source, b.PlayerID())
		b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
		cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
		if cp == nil || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
			t.Fatal("copy target isolation failed")
		}
		clone := g.Clone()
		clone.ResolveStack()
		if a.Life() != 20 || b.Life() != 19 {
			t.Fatal("clone changed original game")
		}
		g.ResolveStack()
		for _, game := range []*Game{g, clone} {
			if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 26 {
				t.Fatal("copy changed event participant")
			}
			if _, ok := FromPlayer(player).resolvePlayer(&EffectContext{Game: game}); ok {
				t.Fatal("event binding leaked")
			}
		}
	}
}

func TestDamagePlayerBindingEventFiltering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	source := NewCreature("Source", "", 1, 4, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), GainLifeFor(FromPlayer(p), Fixed(3)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 1, 4)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: other.ID(), TargetID: b.PlayerID(), Amount: 1})
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: other.ID(), Amount: 1})
	g.FireEvent(GameEvent{Type: EvtLifeLost, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 1})
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.DealDamageToPlayer(b, 0, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unmatched or prevented damage triggered")
	}
	g.DealDamageToPlayer(a, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("damage to controller did not trigger")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("wrong damage recipient")
	}
}

func TestDamagePlayerBindingScope(t *testing.T) {
	p := BindPlayer("damaged")
	read := GainLifeFor(FromPlayer(p), Fixed(1))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "cross ability",
			Card:          NewCreature("Cross ability", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), read)), WithAbility(At(YourUpkeep, read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDamagePlayerBindingDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), AtNext(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(3))))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 22 || game.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("delayed trigger lost player")
		}
	}
}

// --- from composable_global_prevention_test.go ---
func TestGlobalCombatPreventionLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Recipient", "", 4, 4)
	card.SetOwner(b.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, PreventAllCombatDamage(), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.effects.ClearReplacementsEndOfTurn()
	for _, game := range []*Game{g, clone} {
		player := game.GetPlayer(b.PlayerID())
		game.damage.SetResolvingCombatDamage(true)
		game.DealDamageToPlayer(player, 2, uuid.Nil)
		game.DealDamageToPermanent(game.FindPermanent(card.ID()), 2, uuid.Nil)
		game.damage.SetResolvingCombatDamage(false)
		wantLife, wantDamage := 20, 0
		if game == clone {
			wantLife, wantDamage = 18, 2
		}
		if player.Life() != wantLife || game.FindPermanent(card.ID()).Damage != wantDamage {
			t.Fatalf("clone %v: life %d, damage %d; want %d, %d", game == clone, player.Life(), game.FindPermanent(card.ID()).Damage, wantLife, wantDamage)
		}
		game.DealDamageToPlayer(player, 1, uuid.Nil)
		game.DealDamageToPermanent(game.FindPermanent(card.ID()), 1, uuid.Nil)
		if player.Life() != wantLife-1 || game.FindPermanent(card.ID()).Damage != wantDamage+1 {
			t.Fatal("combat prevention stopped noncombat damage")
		}
	}
}

// --- from composable_graveyard_leave_event_test.go ---
func TestGraveyardLeaveBatchEventCountAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("left")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCardsLeaveYourGraveyard().WithCount(count), GainLifeFor(You, FromNumber(count)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first, second, other := NewLand("First"), NewLand("Second"), NewLand("Other")
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	b.AddToGraveyard(other)
	g.MoveCardsFromGraveyard(a.PlayerID(), []uuid.UUID{first.ID(), second.ID(), uuid.New()}, ZoneExile)
	g.MoveFromGraveyard(b.PlayerID(), other.ID(), ZoneExile)
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("batch triggers = %d, want 1", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatalf("clone changed original or lost count: %d, %d", a.Life(), clone.GetPlayer(a.PlayerID()).Life())
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("resolved batch count = %d, want 24", a.Life())
	}
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	g.MoveFromGraveyard(a.PlayerID(), first.ID(), ZoneHand)
	g.MoveFromGraveyard(a.PlayerID(), second.ID(), ZoneHand)
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("separate moves = %d triggers, want 2", got)
	}
	g.ResolveStack()
	if a.Life() != 26 {
		t.Fatalf("separate count = %d, want 26", a.Life())
	}
}

func TestGraveyardLeaveEventSourceLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OneOrMoreCardsLeaveYourGraveyard(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewLand("Card")
	a.AddToGraveyard(card)
	phase := g.PhaseOut(watcher.ID())
	g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	a.AddToGraveyard(card)
	g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("active source did not trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("queued controller changed: %d, %d", a.Life(), b.Life())
	}
	a.AddToGraveyard(card)
	g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("returned source followed former controller")
	}
}

func TestGraveyardLeaveEventDeclaration(t *testing.T) {
	count := BindNumber("left")
	pattern := OneOrMoreCardsLeaveYourGraveyard().WithCount(count)
	if info := InspectDefinition(pattern); info.Role != "event-pattern" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(count))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Nil", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard().WithCount(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "count binding",
		},
		{
			Name:          "duplicate",
			Card:          NewEnchantment("Duplicate", "", WithAbility(WhenEvent(pattern.WithCount(count), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "count binding",
		},
		{
			Name:          "unnamed",
			Card:          NewEnchantment("Unnamed", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard().WithCount(BindNumber("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "missing",
			Card:          NewEnchantment("Missing", "", WithAbility(WhenEvent(OneOrMoreCardsLeaveYourGraveyard(), GainLifeFor(You, FromNumber(count))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "wrong scope",
			Card:          NewEnchantment("Scope", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))), WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(count))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "wrong zone",
			Card:          NewInstant("Instant", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
}

// --- from composable_land_entry_event_test.go ---
func TestLandEntryControllerEvent(t *testing.T) {
	for _, tc := range []struct {
		name        string
		land        bool
		phaseSource bool
		want        int
	}{
		{"own land", true, false, 1},
		{"opponent land", true, false, 1},
		{"nonland", false, false, 0},
		{"phased source", true, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			controller := BindPlayer("entry controller")
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				LandEnters().WithController(controller), GainLifeFor(FromPlayer(controller), Fixed(1)))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phaseSource {
				g.PhaseOut(watcher.ID())
			}
			var card Card = NewArtifact("Other", "")
			if tc.land {
				card = NewLand("Other")
			}
			controllerID := b.PlayerID()
			if tc.name == "own land" {
				controllerID = a.PlayerID()
			}
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, controllerID)
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantA, wantB := 20, 20
			if tc.want == 1 {
				if controllerID == a.PlayerID() {
					wantA++
				} else {
					wantB++
				}
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=%d,%d want=%d,%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestLandEntryControllerDeclaration(t *testing.T) {
	binding := BindPlayer("controller")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Good", "", WithAbility(WhenEvent(LandEnters().WithController(binding),
				GainLifeFor(FromPlayer(binding), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:    "nil binding",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty binding",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithController(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate binding",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithController(binding).WithController(binding), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound",
			Card:    NewArtifact("Unbound", "", WithAbility(WhenEvent(LandEnters(), GainLifeFor(FromPlayer(binding), Fixed(1))))),
			WantErr: true,
		},
	})
	good := NewArtifact("Good", "", WithAbility(WhenEvent(LandEnters().WithController(binding),
		GainLifeFor(FromPlayer(binding), Fixed(1)))))
	if got := InspectDefinition(good.Abilities()[0]); got.Role != "event-trigger" || len(got.Children) == 0 {
		t.Fatalf("inspection: %+v", got)
	}
}

func TestLandEntryControllerPendingCopyAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().WithController(controller), DealDamageFrom(This, FromPlayer(controller), Fixed(2)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(a.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("land did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 16 {
		t.Fatalf("entry controller capture lost: %d,%d", a.Life(), b.Life())
	}
}

// --- from composable_land_mana_event_test.go ---
func TestLandTappedForManaPlayerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("mana player")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player), DealDamageFrom(This, FromPlayer(player), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("clone changed original or lost event player")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 19 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestLandTappedForManaDeclaration(t *testing.T) {
	player := BindPlayer("mana player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil player",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithPlayer(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty player",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithPlayer(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate player",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithPlayer(player).WithPlayer(player), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound",
			Card: NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
				GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestLandTappedForManaFilteringAndSourceLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("player")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player), GainLifeFor(FromPlayer(player), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact mana", "", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))))
	g.PutOnBattlefield(artifact, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	zero := NewLand("Zero mana", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(0)))))
	g.PutOnBattlefield(zero, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), zero.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("nonland or zero production matched")
	}
	g.PhaseOut(watcher.ID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Green, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed mana event")
	}
	g.PhaseIn([]uuid.UUID{watcher.ID()})
	second := NewLand("Second", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Green, Fixed(1)))))
	g.PutOnBattlefield(second, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_land_tap_event_test.go ---
func TestOpponentLandBecomesTapped(t *testing.T) {
	for _, tc := range []struct {
		name, subtype                               string
		opp, phaseSource, phaseLand, retap, nonland bool
		want                                        int
	}{
		{"opposing Forest", "Forest", true, false, false, false, false, 1},
		{"own Forest", "Forest", false, false, false, false, false, 0},
		{"other subtype", "Mountain", true, false, false, false, false, 0},
		{"phased source", "Forest", true, true, false, false, false, 0},
		{"phased land", "Forest", true, false, true, false, false, 0},
		{"already tapped", "Forest", true, false, false, true, false, 1},
		{"nonland Forest", "Forest", true, false, false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1)))))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			land := NewLand("Land", WithSubTypes(tc.subtype))
			if tc.nonland {
				land = NewCreature("Creature", "", 1, 1, WithSubTypes(tc.subtype))
			}
			controller := a.PlayerID()
			if tc.opp {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(land, controller)
			if tc.phaseSource {
				g.PhaseOut(source.ID())
			}
			if tc.phaseLand {
				g.PhaseOut(land.ID())
			}
			g.TapPermanent(g.FindPermanent(land.ID()))
			if tc.retap {
				g.TapPermanent(g.FindPermanent(land.ID()))
			}
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life=%d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestOpponentLandBecomesTappedPendingIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	land := NewLand("Land", WithSubTypes("Forest"))
	g.PutOnBattlefield(land, b.PlayerID())
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestOpponentLandBecomesTappedDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Valid", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped(), GainLifeFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "empty subtype",
			Card: NewEnchantment("Invalid", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype(""), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name: "Goblin subtype",
			Card: NewEnchantment("Invalid", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name: "duplicate subtype",
			Card: NewEnchantment("Duplicate", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype("Forest").OfSubtype("Mountain"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name: "spell source",
			Card: NewInstant("Invalid source", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped(), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
	if got := InspectDefinition(OpponentLandBecomesTapped().OfSubtype("Forest")); got.Role != "event-pattern" || got.Text != "opponent Forest becomes tapped" {
		t.Fatalf("inspection = %+v", got)
	}
}

// --- from composable_life_event_test.go ---
func TestOpponentLifeEventFiltering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("loser"), BindNumber("lost")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(player).WithAmount(amount), GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PlayerLoseLife(a, 2)
	g.PlayerLoseLife(b, 0)
	g.PlayerLoseLife(b, -1)
	g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: b.PlayerID(), Amount: 0})
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unmatched event triggered")
	}
	g.PlayerLoseLife(b, 3)
	g.DealDamageToPlayer(b, 4, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if len(obj.Targets) != 0 {
			t.Fatal("event player became a target")
		}
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestOpponentLifeEventLifecycle(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player, amount := BindPlayer("loser"), BindNumber("lost")
			body := Sequence(GainLifeFor(FromPlayer(player), FromNumber(amount)), GainLifeFor(You, Fixed(1)))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			pattern := OpponentLosesLife(player).WithAmount(amount)
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", GainLifeFor(You, Fixed(1))))
			}
			source := NewEnchantment("Watcher", "", WithAbility(trigger))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.PlayerLoseLife(b, 3)
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.PlayerLoseLife(b, 2)
			g.PlayerLoseLife(a, 4)
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					if obj.Controller == a.PlayerID() {
						game.CopyStackObjectDirect(obj, b.PlayerID(), false)
						break
					}
				}
				game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
				game.PutOnBattlefield(source, a.PlayerID())
				stackClone := game.Clone()
				for _, resolved := range []*Game{game, stackClone} {
					resolved.ResolveStack()
					if delayed {
						resolved.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
						resolved.PutTriggersOnStack()
						resolved.ResolveStack()
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 23 {
						t.Fatalf("life = %d, %d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
					}
				}
			}
		}
	}
}

func TestOpponentLifeEventValidation(t *testing.T) {
	p, n := BindPlayer("player"), BindNumber("amount")
	read := GainLifeFor(FromPlayer(p), FromNumber(n))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(p).WithAmount(n), read))),
			WantErr: false,
		},
		{
			Name:          "nil player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(nil).WithAmount(n), read))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil amount",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(p).WithAmount(nil), read))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "unbound",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(p), read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "duplicate binding",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(p).WithAmount(BindNumber("player")), read))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "empty player name",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(BindPlayer("")).WithAmount(n), read))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name: "scope error",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(OpponentLosesLife(p).WithAmount(n), read)),
				WithAbility(At(YourUpkeep, read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestOpponentLifeEventPaymentAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("loser")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(player), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.PlayerLoseLife(b, 2)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	cost := LifePayCost(3)
	if !cost.CanPay(source.ID(), b.PlayerID(), g) {
		t.Fatal("cost unavailable")
	}
	if err := cost.Pay(source.ID(), b.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 15 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_life_gain_event_test.go ---
func TestControllerLifeGainEventFiltering(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		requested, replaced, want int
		replace, consume          bool
	}{
		{name: "positive", requested: 3, want: 3},
		{name: "zero"},
		{name: "negative", requested: -1},
		{name: "replacement", requested: 3, replace: true, replaced: 7, want: 7},
		{name: "replacement zero", requested: 3, replace: true},
		{name: "replacement negative", requested: 3, replace: true, replaced: -1},
		{name: "consumed", requested: 3, replace: true, consume: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player := BindPlayer("gainer")
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				ControllerGainsLife(player), LoseLifeFor(FromPlayer(player), Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.gainLifeWithEvent(b, 2)
			g.PlayerLoseLife(a, 1)
			for _, amount := range []int{0, -1} {
				g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: a.PlayerID(), Amount: amount})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unmatched event triggered")
			}
			if tc.replace {
				g.effects.AddReplacement(&lifeOutcomeReplacement{amount: tc.replaced, consume: tc.consume})
			}
			g.gainLifeWithEvent(a, tc.requested)
			g.PutTriggersOnStack()
			wantTriggers := 0
			if tc.want > 0 {
				wantTriggers = 1
			}
			if g.StackSize() != wantTriggers {
				t.Fatalf("stack = %d, want %d", g.StackSize(), wantTriggers)
			}
			g.ResolveStack()
			if a.Life() != 19+tc.want-wantTriggers || b.Life() != 22 {
				t.Fatalf("life = %d, %d", a.Life(), b.Life())
			}
		})
	}
}

func TestControllerLifeGainEventLifecycle(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player := BindPlayer("gainer")
			body := Sequence(LoseLifeFor(FromPlayer(player), Fixed(2)), LoseLifeFor(You, Fixed(1)))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			pattern := ControllerGainsLife(player)
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", LoseLifeFor(You, Fixed(9))))
			}
			source := NewEnchantment("Watcher", "", WithAbility(trigger))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.gainLifeWithEvent(a, 1)
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.gainLifeWithEvent(a, 1)
			g.gainLifeWithEvent(b, 1)
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					if len(obj.Targets) != 0 {
						t.Fatal("event player became a target")
					}
					if obj.Controller == a.PlayerID() {
						game.CopyStackObjectDirect(obj, b.PlayerID(), false)
						break
					}
				}
				game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
				game.PutOnBattlefield(source, a.PlayerID())
				cloned := game.Clone()
				for _, resolved := range []*Game{game, cloned} {
					resolved.ResolveStack()
					if delayed {
						resolved.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
						resolved.PutTriggersOnStack()
						resolved.ResolveStack()
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 17 || resolved.GetPlayer(b.PlayerID()).Life() != 17 {
						t.Fatalf("life = %d, %d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
					}
				}
			}
		}
	}
}

func TestControllerLifeGainEventSourceIdentity(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(BindPlayer("gainer")), Destroy(This))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		phased := g.PhaseOut(source.ID())
		g.gainLifeWithEvent(a, 1)
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("phased source triggered")
		}
		g.PhaseIn(phased)
		g.gainLifeWithEvent(a, 1)
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
		} else {
			phased = g.PhaseOut(source.ID())
			g.PhaseIn(phased)
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("pending trigger lost")
		}
		g.ResolveStack()
		if (g.FindPermanent(source.ID()) != nil) != returned {
			t.Fatal("incorrect source incarnation")
		}
	}
}

func TestControllerLifeGainEventValidation(t *testing.T) {
	p := BindPlayer("gainer")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(p), LoseLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(nil), LoseLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(BindPlayer("")), LoseLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound other",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(BindPlayer("other")), LoseLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "scope error",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(ControllerGainsLife(p), LoseLifeFor(FromPlayer(p), Fixed(1)))),
				WithAbility(At(YourUpkeep, LoseLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_modal_trigger_test.go ---
func TestTypedModalTriggerEventAndCopy(t *testing.T) {
	for _, dies := range []bool{false, true} {
		t.Run(map[bool]string{false: "enters", true: "dies"}[dies], func(t *testing.T) {
			a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
			b := newRecordingPlayer("B")
			g := NewGame(a, b)
			target := DeclarePlayerTarget("recipient")
			missing := DeclarePermanentTarget("creature", IsCreature)
			pattern := Enters(This)
			if dies {
				pattern = Dies(This)
			}
			trigger := WhenEventModes(pattern,
				ModeOf("Unavailable", Destroy(missing), WithTargets(missing)),
				ModeOf("Life", GainLifeFor(target, Fixed(3)), WithTargets(target)),
			)
			source := NewEnchantment("Modal source", "", WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
			perm := g.PutOnBattlefield(source, a.PlayerID())
			if dies {
				g.DestroyPermanent(perm)
			}
			g.PutTriggersOnStack()
			original := g.StackPeek()
			if original == nil || original.ModeChoice != 1 || len(a.menus) != 1 || len(a.menus[0]) != 1 || a.menus[0][0] != "Life" {
				t.Fatal("typed trigger lost its legal mode")
			}
			if a.Life() != 20 {
				t.Fatal("mode resolved before priority")
			}
			if !dies {
				g.RemoveFromBattlefield(perm)
			}
			b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
			cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
			if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
				t.Fatal("copy lost mode or independent targets")
			}
			g.ResolveStack()
			if a.Life() != 23 || b.Life() != 23 {
				t.Fatalf("life = %d, %d", a.Life(), b.Life())
			}
			if len(a.menus) != 1 {
				t.Fatal("copy chose a new mode")
			}
		})
	}
}

func TestTypedModalStepTrigger(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A"), choice: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	trigger := AtModes(YourUpkeep, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))
	source := NewEnchantment("Upkeep source", "", WithAbility(trigger))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || len(a.menus) != 0 {
		t.Fatal("trigger matched opponent upkeep")
	}
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("upkeep trigger did not use the stack")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestTypedModalTriggerValidation(t *testing.T) {
	first := DeclarePlayerTarget("first")
	second := DeclarePlayerTarget("second")
	modes := []ModeDefinition{ModeOf("First", GainLifeFor(first, Fixed(1)), WithTargets(first)), ModeOf("Second", GainLifeFor(first, Fixed(1)), WithTargets(second))}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "WhenEventModes scope",
			Card:          NewEnchantment("Scope", "", WithAbility(WhenEventModes(Enters(This), modes...))),
			WantErr:       true,
			WantErrSubstr: "modes[1]",
		},
		{
			Name:          "AtModes scope",
			Card:          NewEnchantment("Scope", "", WithAbility(AtModes(YourUpkeep, modes...))),
			WantErr:       true,
			WantErrSubstr: "modes[1]",
		},
		{
			Name: "WhenEventModes nil event",
			Factory: func() Card {
				return NewEnchantment("Panic", "", WithAbility(WhenEventModes(nil, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))))
			},
			WantErr: true,
		},
		{
			Name: "AtModes nil step",
			Factory: func() Card {
				return NewEnchantment("Panic", "", WithAbility(AtModes(nil, ModeOf("One", GainLifeFor(You, Fixed(1))), ModeOf("Two", GainLifeFor(You, Fixed(2))))))
			},
			WantErr: true,
		},
		{
			Name: "WhenEventModes no modes",
			Factory: func() Card {
				return NewEnchantment("Panic", "", WithAbility(WhenEventModes(Enters(This))))
			},
			WantErr: true,
		},
		{
			Name: "AtModes one mode",
			Factory: func() Card {
				return NewEnchantment("Panic", "", WithAbility(AtModes(YourUpkeep, ModeOf("One", GainLifeFor(You, Fixed(1))))))
			},
			WantErr: true,
		},
		{
			Name:    "nil mode body",
			Card:    NewEnchantment("Nil", "", WithAbility(WhenEventModes(Enters(This), ModeOf("Missing", nil), ModeOf("Life", GainLifeFor(You, Fixed(1)))))),
			WantErr: true,
		},
	})
}

func TestTypedModalTriggerClone(t *testing.T) {
	a := &modalTriggerPlayer{recordingPlayer: newRecordingPlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	trigger := WhenEventModes(Enters(This), ModeOf("Life", GainLifeFor(target, Fixed(3)), WithTargets(target)), ModeOf("Other", GainLifeFor(You, Fixed(1))))
	source := NewEnchantment("Clone source", "", WithAbility(trigger))
	source.SetOwner(a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 || g.StackSize() != 1 {
		t.Fatal("clone changed the original trigger or player")
	}
	g.ResolveStack()
	if a.Life() != 23 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("original lost its target binding")
	}
}

// --- from composable_nontoken_death_test.go ---
func TestAnotherNontokenCreatureDies(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"other controller", 1}, {"own controller", 1}, {"token", 0},
		{"self", 0}, {"simultaneous", 1}, {"phased source", 0}, {"absent source", 0},
		{"exile", 0}, {"returned source", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Observer", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureDies().Nontoken(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if tc.name == "token" {
				other = NewToken("Other", 1, 1, []CardType{TypeCreature}, nil)
			}
			other.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			controller := b.PlayerID()
			if tc.name == "own controller" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(other, controller)
			if tc.name == "phased source" {
				g.PhaseOut(source.ID())
			}
			if tc.name == "absent source" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			switch tc.name {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
			default:
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			if tc.name == "returned source" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, a.PlayerID())
			}
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack = %d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := a.Life(); got != 20+tc.want {
				t.Fatalf("life = %d, want %d", got, 20+tc.want)
			}
			if got := clone.GetPlayer(a.PlayerID()).Life(); got != 20+tc.want {
				t.Fatalf("clone life = %d, want %d", got, 20+tc.want)
			}
		})
	}
}

func TestAnotherNontokenCreatureDiesInspection(t *testing.T) {
	a := NewBasePlayer("A")
	source := NewCreature("Observer", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().Nontoken(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(source.Abilities()[0])
	if got := meta.Children[0].Text; got != "another nontoken creature dies" {
		t.Fatalf("pattern = %q", got)
	}
}

func TestAnotherNontokenCreatureDiesStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Observer", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureDies().Nontoken(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	if copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false); copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("life = %d/%d, want 21/21", a.Life(), b.Life())
	}
}

// --- from composable_opponent_creature_death_test.go ---
func TestOpponentCreatureDiesFormerController(t *testing.T) {
	for _, tc := range []struct {
		name                string
		ownerB, controllerB bool
		want                int
	}{
		{"opponent", true, true, 1},
		{"stolen from opponent", true, false, 0},
		{"opponent controls your creature", false, true, 1},
		{"your creature", false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			former := BindPlayer("former")
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentCreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1)
			owner := a.PlayerID()
			if tc.ownerB {
				owner = b.PlayerID()
			}
			victim.SetOwner(owner)
			g.PutOnBattlefield(victim, owner)
			if tc.controllerB != tc.ownerB {
				controller := a.PlayerID()
				if tc.controllerB {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: victim.ID(), ControllerID: controller, Duration: Indefinite})
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), tc.want)
			}
			if tc.want == 1 {
				clone := g.Clone()
				clone.ResolveStack()
				if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
					t.Fatal("clone shared trigger state")
				}
			}
			g.ResolveStack()
			if b.Life() != 20+tc.want {
				t.Fatalf("opponent life = %d", b.Life())
			}
		})
	}
}

func TestOpponentCreatureDiesLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*Game, *Permanent, *Permanent)
		want  int
	}{
		{"ordinary death", nil, 1},
		{"phased source", func(g *Game, s, _ *Permanent) { g.PhaseOut(s.ID()) }, 0},
		{"exiled creature", func(g *Game, _, v *Permanent) { g.ExilePermanent(v) }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(OpponentCreatureDies(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1)
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.setup != nil {
				tc.setup(g, g.FindPermanent(source.ID()), g.FindPermanent(victim.ID()))
			}
			if tc.name != "exiled creature" {
				g.DestroyPermanent(g.FindPermanent(victim.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), tc.want)
			}
		})
	}
}

func TestOpponentCreatureDiesDeclaration(t *testing.T) {
	former := BindPlayer("former")
	pattern := OpponentCreatureDies().WithController(former)
	if info := InspectDefinition(pattern); info.Role != "event-pattern" || info.Text != "a creature an opponent controls dies" {
		t.Fatalf("inspection = %+v", info)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(pattern, GainLifeFor(FromPlayer(former), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentCreatureDies().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentCreatureDies().WithController(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentCreatureDies(), GainLifeFor(FromPlayer(former), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestOpponentCreatureDiesSimultaneousAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(OpponentCreatureDies(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	token.SetOwner(b.PlayerID())
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	for _, card := range []Card{source, victim, token, artifact} {
		g.PutOnBattlefield(card, card.Owner())
	}
	g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(victim.ID()), g.FindPermanent(token.ID()), g.FindPermanent(artifact.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestOpponentCreatureDiesPendingCopyAndReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(OpponentCreatureDies().WithController(former), GainLifeFor(FromPlayer(former), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	obj := g.StackPeek()
	if len(obj.Targets) != 0 {
		t.Fatal("event controller became a target")
	}
	if g.CopyStackObjectDirect(obj, a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("copy or clone lost event-time controller")
	}
	g.ResolveStack()
	if b.Life() != 22 {
		t.Fatalf("life = %d", b.Life())
	}
}

// --- from composable_opponent_draw_event_test.go ---
func TestOpponentDrawEventDamageAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Draw watcher", "", WithAbility(WhenEvent(
		OpponentDrawsCard(drawer), DealDamageFrom(This, FromPlayer(drawer), Fixed(1)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToLibrary(NewArtifact("A card", ""))
	for range 3 {
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	g.PlayerDrawCard(a)
	for range 2 {
		g.PlayerDrawCard(b)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("draw triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 18 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), b.PlayerID(), func(*Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 1 {
		t.Fatalf("replacement = %d, triggers = %d", replaced, g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatalf("opponent life = %d", b.Life())
	}
}

func TestOpponentDrawEventControlAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 4 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Draw watcher", "", WithAbility(WhenEvent(
		OpponentDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phase := g.PhaseOut(source.ID())
	g.PlayerDrawCard(b)
	g.PhaseIn(phase)
	g.PlayerDrawCard(b)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("draw triggers = %d", g.StackSize())
	}
	var first *StackObject
	for _, obj := range g.StackObjects() {
		if len(obj.Targets) != 0 {
			t.Fatal("event drawer became a target")
		}
		if obj.Controller == a.PlayerID() {
			first = obj
		}
	}
	if first == nil {
		t.Fatal("missing first controller")
	}
	g.CopyStackObjectDirect(first, b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 24 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestOpponentDrawEventValidation(t *testing.T) {
	drawer := BindPlayer("drawer")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDrawsCard(nil), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDrawsCard(BindPlayer("")), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound different",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentDrawsCard(BindPlayer("different")), GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "scope error",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(OpponentDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1)))),
				WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(drawer), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_other_creature_death_test.go ---
func TestAnotherCreatureYouControlDies(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"controlled", 1}, {"opponent", 0}, {"stolen", 1}, {"lost", 0},
		{"self", 0}, {"simultaneous", 1}, {"noncreature", 0},
		{"exile", 0}, {"token", 1}, {"phased", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlDies(), Sequence(GainLifeFor(You, Fixed(1)), AddCountersTo(This, P1P1, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			if tc.name == "noncreature" {
				other = NewArtifact("Other", "")
			}
			if tc.name == "token" {
				other = NewToken("Other", 1, 1, []CardType{TypeCreature}, nil)
			}
			owner := a.PlayerID()
			if tc.name == "opponent" || tc.name == "stolen" {
				owner = b.PlayerID()
			}
			other.SetOwner(owner)
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, owner)
			if tc.name == "stolen" || tc.name == "lost" {
				controller := a.PlayerID()
				if tc.name == "lost" {
					controller = b.PlayerID()
				}
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: other.ID(), ControllerID: controller, Duration: Indefinite})
			}
			switch tc.name {
			case "self":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "simultaneous":
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
			case "exile":
				g.ExilePermanent(g.FindPermanent(other.ID()))
			default:
				if tc.name == "phased" {
					g.PhaseOut(source.ID())
				}
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != tc.want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || g.StackSize() != tc.want {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.want || clone.GetPlayer(a.PlayerID()).Life() != 20+tc.want {
				t.Fatal("wrong life result")
			}
			if perm := g.FindPermanent(source.ID()); perm != nil && tc.want == 1 && perm.Counters[P1P1] != 1 {
				t.Fatal("missing source counter")
			}
		})
	}
}

func TestAnotherCreatureYouControlDiesSourceIncarnation(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		name := "after trigger"
		if simultaneous {
			name = "simultaneous"
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlDies(), Sequence(GainLifeFor(You, Fixed(1)), AddCountersTo(This, P1P1, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, a.PlayerID())
			if simultaneous {
				g.DestroyPermanents([]*Permanent{g.FindPermanent(source.ID()), g.FindPermanent(other.ID())})
			} else {
				g.DestroyPermanent(g.FindPermanent(other.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing trigger")
			}
			if !simultaneous {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.PutOnBattlefield(source, b.PlayerID())
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("trigger controller changed")
			}
			if g.FindPermanent(source.ID()).Counters[P1P1] != 0 {
				t.Fatal("new incarnation received counter")
			}
		})
	}
}

func TestAnotherCreatureYouControlDiesCopyAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlDies(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(source.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Text != "another creature you control dies" {
		t.Fatal("death filter missing from inspection")
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || len(original.TargetSpecs) != 0 {
		t.Fatal("death event made a target")
	}
	if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("copy did not use its own controller")
	}
}

// --- from composable_permanent_death_test.go ---
func TestPermanentDiesTypeAndFormerController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		PermanentDies().OfType(TypeLand).WithController(former),
		GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewLand("First")
	first.SetOwner(a.PlayerID())
	second := NewLand("Second")
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(creature.ID()))
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("land triggers = %d, want 2", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("life = (%d, %d), want (21, 21)", a.Life(), b.Life())
	}
}

func TestPermanentDiesTypeExcludesOtherEvents(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(*Game, *Permanent)
		want    int
	}{
		{"artifact death", nil, 1},
		{"phased source", func(g *Game, source *Permanent) { g.PhaseOut(source.ID()) }, 0},
		{"absent source", func(g *Game, source *Permanent) { g.RemoveFromBattlefield(source) }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				PermanentDies().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewArtifact("Victim", "")
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.prepare != nil {
				tc.prepare(g, g.FindPermanent(watcher.ID()))
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPermanentDiesSimultaneousSourceAndReplacement(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		former := BindPlayer("former")
		watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
			PermanentDies().OfType(TypeLand).WithController(former),
			GainLifeFor(FromPlayer(former), Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		land := NewLand("Land")
		land.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(land, b.PlayerID())
		if replaced {
			g.AddExileIfWouldGoToGraveyardThisTurn(land.ID(), watcher.ID())
		}
		g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(land.ID())})
		g.PutTriggersOnStack()
		want := 1
		if replaced {
			want = 0
		}
		if got := g.StackSize(); got != want {
			t.Fatalf("replaced=%v: triggers=%d, want %d", replaced, got, want)
		}
		if want == 1 {
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
		}
		g.ResolveStack()
		if got := b.Life(); got != 20+2*want {
			t.Fatalf("replaced=%v: former controller life=%d", replaced, got)
		}
	}
}

func TestPermanentDiesAllTypesAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		PermanentDies(), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(watcher.Abilities()[0]).Children[0].Text; got != "a permanent dies" {
		t.Fatalf("inspection=%q", got)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(land.ID()), g.FindPermanent(artifact.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("triggers=%d, want 2", got)
	}
}

func TestPermanentDiesTypeRegistration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "TypeInstant",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				PermanentDies().OfType(TypeInstant), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "TypeSorcery",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				PermanentDies().OfType(TypeSorcery), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "CardType(-1)",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				PermanentDies().OfType(CardType(-1)), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "repeated OfType",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				PermanentDies().OfType(TypeLand).OfType(TypeArtifact), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "CreatureDies OfType Land",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				CreatureDies().OfType(TypeLand), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

// --- from composable_permanent_tap_event_test.go ---
func TestPermanentBecomesTappedCapturesTransition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	object := BindPermanent("tapped permanent")
	controller := BindPlayer("tap controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PermanentBecomesTapped().OfType(TypeCreature).ByOpponent().WithPermanent(object).WithController(controller),
		Sequence(AddCountersTo(From(object), Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	creature := NewCreature("Creature", "", 2, 2)
	g.PutOnBattlefield(creature, b.PlayerID())
	g.TapPermanent(g.FindPermanent(creature.ID()))
	g.TapPermanent(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 || g.FindPermanent(creature.ID()).Counters[Charge] != 0 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if b.Life() != 21 || g.FindPermanent(creature.ID()).Counters[Charge] != 1 {
		t.Fatal("tap participants were not saved")
	}
}

func TestPermanentBecomesTappedIdentityAndSourceLifecycle(t *testing.T) {
	for _, state := range []string{"returned", "phased subject", "phased source", "source leaves", "present"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			object := BindPermanent("object")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				PermanentBecomesTapped().WithPermanent(object), AddCountersTo(From(object), Charge, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			card := NewArtifact("Object", "")
			g.PutOnBattlefield(card, b.PlayerID())
			if state == "phased source" {
				g.PhaseOut(watcher.ID())
			}
			if state == "phased subject" {
				g.PhaseOut(card.ID())
			}
			g.TapPermanent(g.FindPermanent(card.ID()))
			if state == "source leaves" {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			}
			g.PutTriggersOnStack()
			want := 1
			if state == "phased source" || state == "phased subject" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d, want %d", g.StackSize(), want)
			}
			if state == "returned" {
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.ResolveStack()
			wantCounters := int(1)
			if state == "returned" || want == 0 {
				wantCounters = 0
			}
			if got := g.FindPermanentIncludingPhased(card.ID()).Counters[Charge]; got != wantCounters {
				t.Fatalf("counters=%d, want %d", got, wantCounters)
			}
		})
	}
}

func TestPermanentBecomesTappedDeclaration(t *testing.T) {
	object := BindPermanent("object")
	controller := BindPlayer("controller")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewArtifact("Valid", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithPermanent(object).WithController(controller), GainLifeFor(FromPlayer(controller), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:    "nil permanent",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithPermanent(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty permanent",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithPermanent(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate permanent",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithPermanent(object).WithPermanent(object), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil controller",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate controller",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().WithController(controller).WithController(controller), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate ByOpponent",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().ByOpponent().ByOpponent(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate OfType",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().OfType(TypeCreature).OfType(TypeArtifact), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "instant OfType",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(PermanentBecomesTapped().OfType(TypeInstant), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound",
			Card:    NewArtifact("Unbound", "", WithAbility(WhenEvent(PermanentBecomesTapped(), AddCountersTo(From(object), Charge, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestPermanentBecomesTappedFilterAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("tap controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PermanentBecomesTapped().OfType(TypeCreature).ByOpponent().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Own", "", 1, 1), a.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Other type", ""), b.PlayerID())
	g.TapPermanent(artifact)
	if err := TapEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !first.Tapped || !second.Tapped {
		t.Fatal("batch did not tap both creatures")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.ResolveStack()
	if b.Life() != 23 || a.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

// --- from composable_player_discard_event_test.go ---
func TestPlayerDiscardsCardBothPlayersAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for _, player := range []Player{a, b} {
		card := NewArtifact("Discard", "")
		player.AddToHand(card)
		g.PlayerDiscard(player, card.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerDiscardsCardDeclaration(t *testing.T) {
	player := BindPlayer("discarder")
	if info := InspectDefinition(PlayerDiscardsCard(player)); info.Role != "event-pattern" || info.Text != "a player discards a card" {
		t.Fatalf("inspection = %+v", info)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(player), GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(nil), GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("")), GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "unbound other",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("other")), GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "scope error",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(PlayerDiscardsCard(player), GainLifeFor(FromPlayer(player), Fixed(1)))),
				WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestPlayerDiscardsCardReplacementPhasingAndDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewArtifact("Ignored", "")
	a.AddToHand(card)
	phase := g.PhaseOut(source.ID())
	g.PlayerDiscard(a, card.ID())
	g.PhaseIn(phase)
	card = NewArtifact("Replaced", "")
	b.AddToHand(card)
	g.PutOnBattlefield(NewArtifact("Replacement", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	if _, ok := g.PlayerDiscardByEffect(b, card.ID(), source.ID()); !ok {
		t.Fatal("replacement prevented the discard")
	}
	if _, ok := g.PlayerDiscardByEffect(b, card.ID(), source.ID()); ok {
		t.Fatal("one card was discarded twice")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	stackObject := g.StackObjects()[0]
	if len(stackObject.Targets) != 0 {
		t.Fatal("saved player became a target")
	}
	g.CopyStackObjectDirect(stackObject, a.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	found := false
	for _, candidate := range b.Library() {
		found = found || candidate.ID() == card.ID()
	}
	if !found || len(b.Graveyard()) != 0 {
		t.Fatal("discard replacement did not move the card to the library")
	}
}

// --- from composable_prevention_amount_test.go ---
func TestTypedPreventionSavedAmount(t *testing.T) {
	for _, power := range []int{-2, 0, 3} {
		t.Run(Fixed(power).Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", power, 4)
			g.PutOnBattlefield(source, a.PlayerID())
			saved := BindPermanent("amount source")
			body := Sequence(Snapshot(This, saved), PreventNextDamage(You, PowerOf(From(saved))))
			card := NewCreature("Shield", "", 2, 3, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			clone := g.Clone()
			clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 5, uuid.Nil)
			want := 15 + max(power, 0)
			if got := clone.GetPlayer(a.PlayerID()).Life(); got != want {
				t.Fatalf("life = %d, want %d", got, want)
			}
			if a.Life() != 20 {
				t.Fatal("clone consumed original shield")
			}
			g.DealDamageToPlayer(a, 2, uuid.Nil)
			g.DealDamageToPlayer(a, 3, uuid.Nil)
			if a.Life() != want {
				t.Fatalf("life = %d, want %d", a.Life(), want)
			}
		})
	}
}

func TestTypedPreventionTargetAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject := NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(subject, a.PlayerID())
	recipient := DeclarePlayerTarget("recipient")
	amount := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(PreventNextDamage(recipient, PowerOf(amount)), WithTargets(recipient, amount))
	card := NewInstant("Shield", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), subject.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
	g.DealDamageToPlayer(b, 5, uuid.Nil)
	if b.Life() != 18 || a.Life() != 20 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

func TestTypedPreventionAmountValidation(t *testing.T) {
	saved := BindPermanent("missing")
	target := DeclarePermanentTarget("undeclared", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil amount",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(PreventNextDamage(You, nil)))),
			WantErr: true,
		},
		{
			Name:    "nil PowerOf",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(PreventNextDamage(You, PowerOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "PowerOf unbound",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(PreventNextDamage(You, PowerOf(From(saved)))))),
			WantErr: true,
		},
		{
			Name:    "PowerOf undeclared target",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(PreventNextDamage(You, PowerOf(target))))),
			WantErr: true,
		},
		{
			Name:    "nil ValueAtResolution",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(PreventNextDamage(You, ValueAtResolution(nil))))),
			WantErr: true,
		},
	})
}

func TestTypedPreventionUnusedCapacityExpires(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(5)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	g.effects.RemoveEndOfTurn()
	g.effects.ClearReplacementsEndOfTurn()
	g.DealDamageToPlayer(a, 2, uuid.Nil)
	if a.Life() != 18 {
		t.Fatalf("life = %d, want 18", a.Life())
	}
}

func TestTypedPreventionSavedRecipientLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Recipient", "", 3, 6)
			g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("recipient")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
				g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			}
			if err := PreventNextDamage(From(saved), PowerOf(From(saved))).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{c.ID()})
			}
			p := g.FindPermanent(c.ID())
			g.DealDamageToPermanent(p, 4, uuid.Nil)
			want := 1
			if change == "return" || change == "phase" {
				want = 4
			}
			if p.Damage != want {
				t.Fatalf("damage = %d, want %d", p.Damage, want)
			}
		})
	}
}

// --- from composable_prevention_test.go ---
func TestPreventionShieldNewIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "", 3, 3)
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.AddPreventionShield(card.ID(), 2)
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(card.ID()), 1, uuid.Nil)
	if g.FindPermanent(card.ID()).Damage != 1 {
		t.Fatal("shield followed a returned card")
	}
}

func TestTypedPreventionConsumptionAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	ca := clone.GetPlayer(a.PlayerID())
	clone.DealDamageToPlayer(ca, 3, uuid.Nil)
	if ca.Life() != 19 || a.Life() != 20 {
		t.Fatal("cloned shield changed original")
	}
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	g.DealDamageToPlayer(a, 2, uuid.Nil)
	if a.Life() != 19 {
		t.Fatal("shield capacity was not consumed")
	}
	g.effects.RemoveEndOfTurn()
	g.effects.ClearReplacementsEndOfTurn()
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	if a.Life() != 18 {
		t.Fatal("expired shield prevented damage")
	}
}

// --- from composable_sacrifice_event_test.go ---
func TestCreatureYouSacrificeEventFiltering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	own := NewCreature("Own", "", 1, 1)
	g.PutOnBattlefield(own, a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	g.PutOnBattlefield(other, b.PlayerID())
	land := NewLand("Land")
	g.PutOnBattlefield(land, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(other.ID()))
	g.DoSacrifice(g.FindPermanent(land.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("opponent creature or own land matched")
	}
	g.DoSacrifice(g.FindPermanent(own.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
		t.Fatal("own creature must create one untargeted trigger")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
	g.DoSacrifice(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("source sacrifice did not trigger from LKI")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want 22", a.Life())
	}
}

func TestCreatureYouSacrificeEventIdentityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	g.PutOnBattlefield(first, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(first.ID()))
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatal("pending trigger lost in clone")
		}
		if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), true) == nil {
			t.Fatal("trigger copy failed")
		}
		game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
		game.PutOnBattlefield(source, b.PlayerID())
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatal("return changed pending trigger or copied controller")
		}
		if game == clone && a.Life() != 20 {
			t.Fatal("clone changed original game")
		}
	}
}

func TestCreatureYouSacrificeEventPhasedSourceAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PhaseOut(source.ID())
	creature := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed sacrifice")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	creature = NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phased-in source missed sacrifice")
	}
	g.ResolveStack()
	g.ExilePermanent(g.FindPermanent(source.ID()))
	creature = NewCreature("Third", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source observed sacrifice")
	}
}

func TestCreatureYouSacrificeEventControllerAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	stolen := NewCreature("Stolen", "", 1, 1)
	stolen.SetOwner(b.PlayerID())
	g.PutOnBattlefield(stolen, b.PlayerID())
	g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: stolen.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
	g.DoSacrifice(g.FindPermanent(stolen.ID()))
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	g.PutOnBattlefield(token, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(token.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("trigger count = %d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want 22", a.Life())
	}
}

// --- from composable_source_leaves_test.go ---
func TestSourceLeavesBattlefieldDestinationsAndIdentity(t *testing.T) {
	for _, destination := range []string{"graveyard", "exile", "hand"} {
		t.Run(destination, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Leaving watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			phased := g.PhaseOut(source.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("phasing caused a leave trigger")
			}
			g.PhaseIn(phased)
			switch destination {
			case "graveyard":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "exile":
				g.ExilePermanent(g.FindPermanent(source.ID()))
			case "hand":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack=%d, want 1", g.StackSize())
			}
			old := g.StackPeek()
			if old == nil || len(old.TargetSpecs) != 0 {
				t.Fatal("leave trigger has targets")
			}
			g.PutOnBattlefield(source, b.PlayerID())
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("leave trigger changed controller after return")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution state")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("cloned trigger did not resolve")
			}
		})
	}
}

func TestSourceLeavesBattlefieldSimultaneousAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(1)))))
	second := NewCreature("Second watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(1)))))
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want 2", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatal("simultaneous leave or copied controller was lost")
	}
}

func TestSourceLeavesBattlefieldInspection(t *testing.T) {
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(LeavesBattlefield(This), DrawFor(You, Fixed(1)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    source,
			WantErr: false,
		},
		{
			Name:    "spell source",
			Card:    NewSorcery("Invalid leave watcher", "", WithAbility(WhenEvent(LeavesBattlefield(This), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
	meta := InspectDefinition(source.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Text != "source leaves battlefield" {
		t.Fatal("leave pattern missing from inspection")
	}
}

// --- from composable_source_tap_event_test.go ---
func TestSourceBecomesTappedTransitionAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesTapped(This),
		Sequence(GainLifeFor(You, Fixed(1)), Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2)))))))
	watcher.SetOwner(a.PlayerID())
	other := NewLand("Other")
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.TapPermanent(g.FindPermanent(other.ID()))
	g.TapPermanent(g.FindPermanent(watcher.ID()))
	g.TapPermanent(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("tap transition queued %d triggers, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original life totals")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("pending trigger changed controller after source return")
	}
	if g.FindPermanent(watcher.ID()).CurrentPower(g) != 1 {
		t.Fatal("old tap trigger modified returned incarnation")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("clone lost pending tap trigger")
	}
}

func TestSourceBecomesTappedAbsentPhasedAndCopy(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("recipient")
	watcher := NewLand("Watcher", WithAbility(WhenEvent(BecomesTapped(This),
		GainLifeFor(target, Fixed(2)), WithTargets(target))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	phase := g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: watcher.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.TapPermanent(g.FindPermanent(watcher.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 1 {
		t.Fatal("tap trigger did not announce its target")
	}
	stackCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if stackCopy == nil || len(stackCopy.Targets) != 1 || stackCopy.Targets[0] != a.PlayerID() {
		t.Fatal("tap trigger copy lost its target")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("tap trigger copy changed target")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: watcher.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source triggered")
	}
}

func TestSourceBecomesTappedDeclaration(t *testing.T) {
	card := NewLand("Valid", WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    card,
			WantErr: false,
		},
		{
			Name:          "spell source",
			Card:          NewInstant("Invalid", "", WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
	meta := InspectDefinition(card.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Role != "event-pattern" {
		t.Fatal("source tap pattern missing from inspection")
	}
}

func TestSourceBecomesTappedDoesNotMatchEntryTapped(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Entry watcher", WithKeyword(EntersTapped),
		WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("entering tapped caused a source tap trigger")
	}
	g.UntapPermanent(g.FindPermanent(land.ID()))
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("later tap transition did not trigger")
	}
}

// --- from composable_step_player_test.go ---
func TestTurnStepPlayerCapture(t *testing.T) {
	for _, modal := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("step player")
		pattern := TurnStepBegins(EachUpkeep, player)
		body := Sequence(GainLifeFor(FromPlayer(player), Fixed(3)), GainLifeFor(You, Fixed(1)))
		ability := WhenEvent(pattern, body)
		if modal {
			ability = WhenEventModes(pattern, ModeOf("Life", body), ModeOf("Other", DrawFor(You, Fixed(1))))
		}
		source := NewEnchantment("Watcher", "", WithAbility(ability))
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		perm := g.PutOnBattlefield(source, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID(), TargetID: a.PlayerID()})
		g.RemoveFromBattlefield(perm)
		g.PutOnBattlefield(source, b.PlayerID())
		pendingClone := g.Clone()
		for _, game := range []*Game{g, pendingClone} {
			game.PutTriggersOnStack()
			original := game.StackPeek()
			if original == nil || len(original.Targets) != 0 {
				t.Fatal("missing nontargeted trigger")
			}
			if game.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
				t.Fatal("missing copy")
			}
			clone := game.Clone()
			clone.ResolveStack()
			if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone changed original")
			}
			game.ResolveStack()
			for _, resolved := range []*Game{game, clone} {
				if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 27 {
					t.Fatal("event player or controller changed")
				}
				if _, ok := FromPlayer(player).resolvePlayer(&EffectContext{Game: resolved}); ok {
					t.Fatal("binding leaked")
				}
			}
		}
	}
}

func TestTurnStepPlayerFiltering(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pattern TurnStepPattern
		event   EventType
		first   bool
		own     bool
	}{
		{"each upkeep", EachUpkeep, EvtUpkeep, false, false},
		{"own upkeep", YourUpkeep, EvtUpkeep, false, true},
		{"end step", EachEndStep, EvtEndStep, false, false},
		{"own end step", YourEndStep, EvtEndStep, false, true},
		{"combat end", EachEndOfCombat, EvtEndOfCombat, false, false},
		{"draw step", EachDrawStep, EvtDrawStep, false, false},
		{"first main", YourFirstMainPhase, EvtMainPhase, true, true},
		{"postcombat main", YourPostcombatMainPhase, EvtMainPhase, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := BindPlayer("player")
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(tt.pattern, p), GainLifeFor(FromPlayer(p), Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: a.PlayerID()})
			if tt.event == EvtMainPhase {
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: a.PlayerID(), Flag: !tt.first})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unmatched event triggered")
			}
			for _, player := range []Player{a, b} {
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: player.PlayerID(), Flag: tt.first})
			}
			g.PutTriggersOnStack()
			want := 2
			if tt.own {
				want = 1
			}
			if g.StackSize() != want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			wantB := 21
			if tt.own {
				wantB = 20
			}
			if a.Life() != 21 || b.Life() != wantB {
				t.Fatalf("life = %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestTurnStepPlayerDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("step player")
	body := Sequence(AtNext(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(3))), SnapshotPlayer(You, p))
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), body)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	perm := g.PutOnBattlefield(source, a.PlayerID())
	for _, player := range []Player{a, b} {
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: player.PlayerID()})
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.RemoveFromBattlefield(perm)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatal("delayed captures shared")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("delayed player changed")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 0 {
			t.Fatal("delayed trigger repeated")
		}
	}
}

func TestTurnStepPlayerValidation(t *testing.T) {
	p := BindPlayer("player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), GainLifeFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name:          "nil player",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty player name",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name: "scope error",
			Card: NewEnchantment("Scope", "",
				WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(1)))),
				WithAbility(At(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "nil pattern panic",
			Factory: func() Card {
				TurnStepBegins(nil, p)
				return nil
			},
			WantErr: true,
		},
	})
}

func TestTurnStepPlayerUpkeepAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("step player")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(3)))))
	g.PutOnBattlefield(source, b.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.doUpkeepActions()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.doUpkeepActions()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("upkeep trigger did not use the stack")
	}
	g.PhaseOut(source.ID())
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("upkeep event lost its player after source phased out")
	}
}

// --- from composable_targeted_event_test.go ---
func TestBecomesTargetedCapturesEventController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargeted(This).ByOpponent().WithController(actor),
		GainLifeFor(FromPlayer(actor), Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 2, 2)
	g.PutOnBattlefield(other, a.PlayerID())
	for _, event := range []GameEvent{
		{Type: EvtBecomesTarget, TargetID: other.ID(), PlayerID: b.PlayerID()},
		{Type: EvtBecomesTarget, TargetID: source.ID(), PlayerID: a.PlayerID()},
	} {
		g.FireEvent(event)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unrelated or own targeting matched")
	}
	for _, ability := range []bool{false, true} {
		g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: source.ID(), SourceID: uuid.New(), PlayerID: b.PlayerID(), Flag: ability})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("targeting triggers = %d, want 2", g.StackSize())
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 22 || g.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("event controller changed after source return")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone lost targeting capture")
	}
}

func TestBecomesTargetedUnavailableSource(t *testing.T) {
	for _, change := range []string{"absent", "phased"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(BecomesTargeted(This), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			if change == "absent" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			} else {
				g.PhaseOut(source.ID())
			}
			g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: source.ID(), PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unavailable source triggered")
			}
		})
	}
}

func TestBecomesTargetedByActivatedAbility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargeted(This).ByOpponent().WithController(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("target", IsCreature)
	activator := NewArtifact("Activator", "", WithAbility(Activate(Costs(), TapSubject(target), WithTargets(target))))
	g.PutOnBattlefield(activator, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), activator.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size = %d, want 2", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[1], a.PlayerID(), false)
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatal("activation or copied trigger lost its event controller")
	}
}

func TestBecomesTargetedDeclaration(t *testing.T) {
	actor := BindPlayer("actor")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		BecomesTargeted(This).ByOpponent().WithController(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    valid,
			WantErr: false,
		},
		{
			Name:          "nil controller",
			Card:          NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(BecomesTargeted(This).WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "controller binding",
		},
		{
			Name:          "duplicate controller",
			Card:          NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(BecomesTargeted(This).WithController(actor).WithController(actor), GainLifeFor(FromPlayer(actor), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "one controller",
		},
		{
			Name:          "unbound",
			Card:          NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(BecomesTargeted(This), GainLifeFor(FromPlayer(actor), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "spell source",
			Card:          NewInstant("Spell", "", WithAbility(WhenEvent(BecomesTargeted(This), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
	if info := InspectDefinition(BecomesTargeted(This)); info.Role != "event-pattern" {
		t.Fatalf("inspection role = %q", info.Role)
	}
}

// --- from composable_trigger_test.go ---
func TestComposableInterveningIf(t *testing.T) {
	for _, present := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		artifact := NewArtifact("Artifact", "")
		artifact.SetOwner(a.PlayerID())
		g.PutOnBattlefield(artifact, a.PlayerID())
		source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(Enters(This), DrawFor(You, Fixed(1)), WithInterveningIf(Controls(IsArtifact)))))
		source.SetOwner(a.PlayerID())
		a.AddToLibrary(NewCreature("Drawn", "", 1, 1))
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutTriggersOnStack()
		if !present {
			g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
		}
		g.ResolveStack()
		if (len(a.Hand()) == 1) != present {
			t.Fatalf("intervening condition ignored: present=%v hand=%d", present, len(a.Hand()))
		}
	}
}

func TestComposableTriggerTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	trigger := WhenEvent(Enters(This), DrawFor(target, Fixed(1)), WithTargets(target))
	card := NewCreature("Source", "", 1, 1, WithAbility(trigger))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Source", card); err != nil {
		t.Fatal(err)
	}
	b.AddToLibrary(NewCreature("Drawn", "", 1, 1))
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Controller: a.PlayerID(), IsAbility: true, TargetSource: card, Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: trigger.Targets(), Effects: trigger.Effects()}
	g.pushStack(obj)
	g.ResolveStackObject(obj)
	if len(b.Hand()) != 1 {
		t.Fatal("trigger lost player binding")
	}
}

// --- from composable_trigger_untargeted_test.go ---
func TestComposableEntryTriggerHasNoImplicitTarget(t *testing.T) {
	for _, scenario := range []string{"present", "departed", "phased", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			a.AddToLibrary(NewCreature("A draw", "", 1, 1))
			b.AddToLibrary(NewCreature("B draw", "", 1, 1))
			source := NewCreature("Entry source", "", 1, 1, WithAbility(WhenEvent(Enters(This), DrawFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutTriggersOnStack()
			obj := g.StackPeek()
			if obj == nil {
				t.Fatal("missing entry trigger")
			}
			if len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 {
				t.Fatal("untargeted entry trigger has an implicit target")
			}
			switch scenario {
			case "departed":
				g.ExilePermanent(g.FindPermanent(source.ID()))
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
				if cp == nil || len(cp.Targets) != 0 {
					t.Fatal("copy acquired a target")
				}
				g.ExilePermanent(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			if len(a.Hand()) != 1 {
				t.Fatal("source absence stopped the untargeted trigger")
			}
			want := 0
			if scenario == "copy" {
				want = 1
			}
			if len(b.Hand()) != want {
				t.Fatal("copy used the wrong controller")
			}
		})
	}
}

func TestComposableDeathTriggerHasNoImplicitTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToLibrary(NewCreature("Drawn", "", 1, 1))
	source := NewCreature("Death source", "", 1, 1, WithAbility(WhenEvent(Dies(This), DrawFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	obj := g.StackPeek()
	if obj == nil || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 {
		t.Fatal("death event participants became targets")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 {
		t.Fatal("death trigger did not draw")
	}
}
