package mage

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_any_damage_source_test.go ---
func TestDealsDamageAnyRecipientCapturesSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
		DealsDamage(This, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, FromNumber(amount)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.DealDamageToPlayer(b, 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("player damage triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 19 {
			t.Fatalf("departed source damage = %d", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 18 {
			t.Fatalf("copied trigger damage = %d", got)
		}
	}
}

func TestDealsDamageAnyRecipientSourceDeclaration(t *testing.T) {
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, amount).WithSource(saved),
				DealDamageFrom(FromDamageSource(saved), You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "nil source",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, amount).WithSource(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty source",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, amount).WithSource(BindDamageSource("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate source",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, amount).WithSource(saved).WithSource(saved), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound",
			Card: NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
				DealsDamage(This, amount), DealDamageFrom(FromDamageSource(saved), You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	if got := InspectDefinition(DealsDamage(This, amount).WithSource(saved)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}

func TestDealsDamageAnyRecipientSourceBatchAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
		DealsDamage(This, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, FromNumber(amount)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	g.PutOnBattlefield(victim, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	if err := ApplyEffect(g, DealDamageEach(This,
		DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(1)),
		watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("batch triggers = %d", g.StackSize())
	}
	g.PhaseOut(watcher.ID())
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 16 {
		t.Fatalf("life after three captured sources = %d", got)
	}
}

func TestAttachedPermanentAnyDamageSourceCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, saved := BindNumber("amount"), BindDamageSource("source")
	aura := NewAura("Link", "", WithAbility(WhenEvent(
		DealsDamage(AttachedPermanent, amount).WithSource(saved),
		DealDamageFrom(FromDamageSource(saved), You, FromNumber(amount)))))
	host := NewCreature("Host", "", 2, 8)
	victim := NewCreature("Victim", "", 2, 8)
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("attached source triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.PhaseOut(aura.ID())
	g.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 18 {
		t.Fatalf("saved host damage = %d", got)
	}
}

// --- from composable_artifact_tap_opponent_test.go ---
func TestArtifactBecomesTappedByOpponent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := BindPermanent("artifact")
	controller := BindPlayer("controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().ByOpponent().WithArtifact(artifact).WithController(controller),
		Sequence(AddCountersTo(From(artifact), Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	own := NewArtifact("Own", "")
	own.SetOwner(b.PlayerID())
	g.PutOnBattlefield(own, a.PlayerID())
	g.TapPermanent(g.FindPermanent(own.ID()))
	opponent := NewArtifact("Opponent", "")
	opponent.SetOwner(a.PlayerID())
	g.PutOnBattlefield(opponent, b.PlayerID())
	g.TapPermanent(g.FindPermanent(opponent.ID()))
	g.TapPermanent(g.FindPermanent(opponent.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(opponent.ID()))
	g.PutOnBattlefield(opponent, a.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("could not copy trigger")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
	if got := g.FindPermanent(opponent.ID()).Counters[Charge]; got != 0 {
		t.Fatalf("returned artifact has %d counters", got)
	}
}

func TestArtifactBecomesTappedByOpponentPhasingAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().ByOpponent(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "")
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.TapPermanent(g.FindPermanent(artifact.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased watcher triggered")
	}
	if err := ValidateCardDefinition("Valid", watcher); err != nil {
		t.Fatal(err)
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().ByOpponent().ByOpponent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "filter") {
		t.Fatalf("duplicate filter validation=%v", err)
	}
}

// --- from composable_attached_damage_player_test.go ---
func TestAttachedDamageToPlayerCapturesParticipantAndAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	amount := BindNumber("amount")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, player).WithAmount(amount),
		GainLifeFor(FromPlayer(player), FromNumber(amount)),
	)))
	host := NewCreature("Host", "", 2, 8)
	other := NewCreature("Other", "", 2, 8)
	for _, card := range []Card{aura, host, other} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPermanent(g.FindPermanent(other.ID()), 2, host.ID())
	g.DealDamageToPlayer(b, 1, other.ID())
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, host.ID())
	g.DealDamageToPlayer(b, 3, host.ID())
	g.DealDamageToPlayer(a, 1, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d, want 2", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, b.PlayerID())
	g.Attach(aura.ID(), other.ID())
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 19 {
			t.Fatalf("life = %d/%d, want 20/19", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestAttachedDamageToPlayerPhasingAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("player")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, p), GainLifeFor(FromPlayer(p), Fixed(1)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("recipient life = %d, want 20", game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestAttachedDamageToPlayerRegistration(t *testing.T) {
	p := BindPlayer("player")
	n := BindNumber("amount")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "wrong source",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(DealsDamageToPlayer(From(BindPermanent("other")), p), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil player",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty player",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, p).WithAmount(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty amount",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, p).WithAmount(BindNumber("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "valid",
			Card:    NewAura("Valid", "", WithAbility(WhenEvent(DealsDamageToPlayer(AttachedPermanent, p).WithAmount(n), GainLifeFor(FromPlayer(p), FromNumber(n))))),
			WantErr: false,
		},
	})
}

// --- from composable_attached_damage_test.go ---
func TestAttachedDamageEventFinalAmountsAndLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("dealt")
	aura := NewAura("Link", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, amount), GainLifeFor(You, FromNumber(amount)))))
	aura.SetOwner(a.PlayerID())
	host := NewCreature("Host", "", 2, 8)
	host.SetOwner(b.PlayerID())
	victim := NewCreature("Victim", "", 2, 12)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(b, 2, victim.ID())
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, host.ID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, host.ID())
	g.DealDamageToPlayer(b, 4, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d, want 2", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, b.PlayerID())
	g.Attach(aura.ID(), victim.ID())
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 27 {
			t.Fatalf("old controller life = %d, want 27", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 14 {
			t.Fatalf("recipient life = %d, want 14", got)
		}
	}
}

func TestAttachedDamageEventTracksAttachmentAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("damage")
	aura := NewAura("Link", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, n), GainLifeFor(You, FromNumber(n)))))
	host := NewCreature("Host", "", 2, 8)
	other := NewCreature("Other", "", 2, 8)
	for _, card := range []Card{aura, host, other} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.Attach(aura.ID(), other.ID())
	g.DealDamageToPlayer(b, 2, host.ID())
	g.DealDamageToPlayer(b, 3, other.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 14 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

func TestAttachedDamageEventDeclaration(t *testing.T) {
	n := BindNumber("amount")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewAura("Good", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, n), GainLifeFor(You, FromNumber(n))))),
			WantErr: false,
		},
		{
			Name:    "nil amount",
			Card:    NewAura("Nil", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unnamed amount",
			Card:    NewAura("Unnamed", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, BindNumber("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "wrong source",
			Card:    NewAura("Wrong source", "", WithAbility(WhenEvent(DealsDamage(From(BindPermanent("other")), n), GainLifeFor(You, FromNumber(n))))),
			WantErr: true,
		},
		{
			Name:    "spell",
			Card:    NewInstant("Spell", "", Spell(GainLifeFor(You, FromNumber(n)))),
			WantErr: true,
		},
	})
}

func TestAttachedDamageEventStackCopyKeepsAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("damage")
	aura := NewAura("Link", "", WithAbility(WhenEvent(DealsDamage(AttachedPermanent, n), GainLifeFor(You, FromNumber(n)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPlayer(b, 2, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatalf("copy life = %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

// --- from composable_attached_land_mana_test.go ---
func TestAttachedLandTappedForManaEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(Blue)), a.PlayerID())
	other := g.PutOnBattlefield(NewLand("Other", WithManaAbility(Blue)), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Watcher", "", WithCastTarget(TargetLand()), WithAbility(WhenEvent(
		LandTappedForMana().OfAttachedLand().OfManaType(Blue), GainLifeFor(You, Fixed(1))))), b.PlayerID())
	g.Attach(aura.ID(), land.ID())

	if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 {
		t.Fatal("unattached land matched")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("trigger count = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 21 || b.Life() != 20 {
		t.Fatal("clone did not keep its own trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.Attach(aura.ID(), other.ID())
	g.ResolveStack()
	if b.Life() != 21 || a.Life() != 21 {
		t.Fatalf("life = %d/%d, want 21/21", a.Life(), b.Life())
	}

	oldIncarnation := land.incarnationID
	g.RemoveFromBattlefield(land)
	returned := g.PutOnBattlefield(land.Card, a.PlayerID())
	g.Attach(aura.ID(), returned.ID())
	g.FireEvent(GameEvent{Type: EvtTappedForMana, SourceID: returned.ID(), SourceIncarnation: oldIncarnation, PlayerID: a.PlayerID(), Amount: 1, ManaTypes: []Color{Blue}})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("old land incarnation matched a new attachment")
	}

	g.MutablePermanent(aura.ID()).PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtTappedForMana, SourceID: returned.ID(), SourceIncarnation: returned.incarnationID, PlayerID: a.PlayerID(), Amount: 1, ManaTypes: []Color{Blue}})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased aura matched")
	}
}

func TestAttachedLandTappedForManaDeclaration(t *testing.T) {
	pattern := LandTappedForMana().OfAttachedLand()
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewAura("Valid", "", WithCastTarget(TargetLand()), WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name:    "duplicate filter",
			Card:    NewAura("Duplicate", "", WithCastTarget(TargetLand()), WithAbility(WhenEvent(pattern.OfAttachedLand(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || got.Text == "" {
		t.Fatalf("inspection = %+v", got)
	}
}

// --- from composable_attached_step_test.go ---
func TestAttachedControllerUpkeepCapture(t *testing.T) {
	for _, change := range []string{"move", "source returns", "host returns", "phase host", "phase in", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host, player := BindPermanent("host"), BindPlayer("player")
			body := Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(FromPlayer(player), Fixed(3)), GainLifeFor(You, Fixed(1)))
			source := NewAura("Watcher", "", WithAbility(WhenEvent(AttachedControllerUpkeep(host, player), body)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			g.PutOnBattlefield(first, b.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.Attach(source.ID(), first.ID())
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong step or controller triggered")
			}
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
			want := uint8(2)
			switch change {
			case "move":
				g.Attach(source.ID(), second.ID())
			case "source returns":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, b.PlayerID())
				g.Attach(source.ID(), second.ID())
			case "host returns":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
				want = 0
			case "phase host":
				g.PhaseOut(first.ID())
				want = 0
			case "phase in":
				phased := g.PhaseOut(first.ID())
				g.PhaseIn(phased)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 1 || len(game.StackPeek().Targets) != 0 {
					t.Fatal("missing nontargeted trigger")
				}
				if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("missing copy")
				}
				clone := game.Clone()
				clone.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone leaked")
				}
				game.ResolveStack()
				for _, resolved := range []*Game{game, clone} {
					if resolved.FindPermanentIncludingPhased(first.ID()).Counters[Charge] != want || resolved.FindPermanent(second.ID()).Counters[Charge] != 0 {
						t.Fatal("captured host changed")
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 27 {
						t.Fatal("captured player or copy controller changed")
					}
				}
			}
		})
	}
}

func TestAttachedControllerUpkeepDelayedModes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host, player := BindPermanent("host"), BindPlayer("player")
	body := Sequence(AtNext(EachEndStep, Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(FromPlayer(player), Fixed(2)))), Snapshot(This, host), SnapshotPlayer(You, player))
	source := NewAura("Watcher", "", WithAbility(WhenEventModes(AttachedControllerUpkeep(host, player), ModeOf("Delay", body), ModeOf("Other", DrawFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unattached source triggered")
	}
	g.Attach(source.ID(), first.ID())
	phased := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.Attach(source.ID(), second.ID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("events lost separate captures")
	}
	g.ResolveStack()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.FindPermanent(first.ID()).Counters[Charge] != 1 || game.FindPermanent(second.ID()).Counters[Charge] != 1 {
			t.Fatal("delayed host overwritten")
		}
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("delayed player overwritten")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 0 {
			t.Fatal("delayed trigger repeated")
		}
	}
}

func TestAttachedControllerUpkeepValidation(t *testing.T) {
	host, player := BindPermanent("host"), BindPlayer("player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil host",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(AttachedControllerUpkeep(nil, BindPlayer("player")), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil player",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(AttachedControllerUpkeep(BindPermanent("host"), nil), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty host",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(AttachedControllerUpkeep(BindPermanent(""), BindPlayer("player")), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "same binding name",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(AttachedControllerUpkeep(BindPermanent("same"), BindPlayer("same")), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "scope error",
			Card: NewAura("Scope", "",
				WithAbility(WhenEvent(AttachedControllerUpkeep(host, player), DrawFor(You, Fixed(1)))),
				WithAbility(At(EachEndStep, TapSubject(From(host))))),
			WantErr: true,
		},
	})
}

// --- from composable_attached_tap_test.go ---
func TestAttachedBecomesTappedCapture(t *testing.T) {
	for _, change := range []string{"none", "move", "source leaves", "source returns", "host returns", "phase host", "phase in", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := BindPermanent("host")
			body := Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(You, Fixed(1)))
			source := NewAura("Watcher", "", WithAbility(WhenEvent(AttachedBecomesTapped(host), body)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			g.PutOnBattlefield(first, b.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.Attach(source.ID(), first.ID())
			g.TapPermanent(g.FindPermanent(second.ID()))
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong step or controller triggered")
			}
			g.TapPermanent(g.FindPermanent(first.ID()))
			want := uint8(2)
			switch change {
			case "move":
				g.Attach(source.ID(), second.ID())
			case "source leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source returns":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, b.PlayerID())
				g.Attach(source.ID(), second.ID())
			case "host returns":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
				want = 0
			case "phase host":
				g.PhaseOut(first.ID())
				want = 0
			case "phase in":
				phased := g.PhaseOut(first.ID())
				g.PhaseIn(phased)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 1 || len(game.StackPeek().Targets) != 0 {
					t.Fatal("missing nontargeted trigger")
				}
				if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("missing copy")
				}
				clone := game.Clone()
				clone.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone leaked")
				}
				game.ResolveStack()
				for _, resolved := range []*Game{game, clone} {
					if resolved.FindPermanentIncludingPhased(first.ID()).Counters[Charge] != want || resolved.FindPermanent(second.ID()).Counters[Charge] != 0 {
						t.Fatal("captured host changed")
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 21 {
						t.Fatal("captured player or copy controller changed")
					}
				}
			}
		})
	}
}

func TestAttachedBecomesTappedDelayedModes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := BindPermanent("host")
	body := Sequence(AtNext(EachEndStep, Sequence(AddCountersTo(From(host), Charge, Fixed(1)))), Snapshot(This, host))
	source := NewAura("Watcher", "", WithAbility(WhenEventModes(AttachedBecomesTapped(host), ModeOf("Delay", body), ModeOf("Other", DrawFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unattached source triggered")
	}
	g.Attach(source.ID(), first.ID())
	phased := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.Attach(source.ID(), second.ID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: second.ID(), PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("events lost separate captures")
	}
	g.ResolveStack()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.FindPermanent(first.ID()).Counters[Charge] != 1 || game.FindPermanent(second.ID()).Counters[Charge] != 1 {
			t.Fatal("delayed host overwritten")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 0 {
			t.Fatal("delayed trigger repeated")
		}
	}
}

func TestAttachedBecomesTappedValidation(t *testing.T) {
	host := BindPermanent("host")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil host",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(AttachedBecomesTapped(nil), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty host",
			Card:    NewAura("Invalid", "", WithAbility(WhenEvent(AttachedBecomesTapped(BindPermanent("")), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "scope error",
			Card: NewAura("Scope", "",
				WithAbility(WhenEvent(AttachedBecomesTapped(host), DrawFor(You, Fixed(1)))),
				WithAbility(At(EachEndStep, TapSubject(From(host))))),
			WantErr: true,
		},
	})
}

// --- from composable_attachment_lki_test.go ---
func TestAttachedPermanentResolutionLifetime(t *testing.T) {
	for _, change := range []string{"move", "move then leave", "host returns before source", "leave", "return", "return twice", "host returns", "turn cleanup", "phase source", "phase host", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			aura := NewAura("Attachment", "", WithAbility(Activate(Costs(), Sequence(AddCountersTo(AttachedPermanent, Charge, Fixed(1)), GainLifeFor(You, Fixed(2))))))
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, card := range []Card{first, second, aura} {
				card.SetOwner(a.PlayerID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			g.Attach(aura.ID(), first.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			wantFirst, wantSecond := 1, 0
			switch change {
			case "move", "move then leave":
				g.Attach(aura.ID(), second.ID())
				wantFirst, wantSecond = 0, 1
				if change == "move then leave" {
					g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				}
			case "leave", "return", "return twice", "host returns", "turn cleanup", "clone":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				if change == "return" || change == "return twice" {
					g.PutOnBattlefield(aura, a.PlayerID())
					g.Attach(aura.ID(), second.ID())
					if change == "return twice" {
						g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
					}
				}
				if change == "host returns" {
					g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
					g.PutOnBattlefield(first, a.PlayerID())
					wantFirst = 0
				}
				if change == "turn cleanup" {
					g.zones.ClearEndOfTurn()
				}
				if change == "clone" {
					original := g
					g = g.Clone()
					g.PutOnBattlefield(aura, a.PlayerID())
					g.Attach(aura.ID(), second.ID())
					g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
					original.ResolveStack()
					if original.FindPermanent(first.ID()).Counters[Charge] != 1 {
						t.Fatal("clone changed original attachment history")
					}
				}
			case "host returns before source":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, a.PlayerID())
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				wantFirst = 0
			case "phase source":
				g.PhaseOut(aura.ID())
			case "phase host":
				g.PhaseOut(first.ID())
				wantFirst = 0
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(first.ID()).Counters[Charge] != uint8(2*wantFirst) || g.FindPermanent(second.ID()).Counters[Charge] != uint8(2*wantSecond) {
				t.Fatal("attachment did not retain the required incarnation")
			}
			if g.GetPlayer(a.PlayerID()).Life() != 22 || g.GetPlayer(b.PlayerID()).Life() != 22 {
				t.Fatal("copy lost its controller or absent attachment stopped the sequence")
			}
		})
	}
}

// --- from composable_card_loop_test.go ---
func TestForEachSavedCardDrawOrderVisitsAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	cards, member := BindCards("drawn"), BindCard("member")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(2), DrawCardsInto(cards)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	var visited []uuid.UUID
	body := FuncEffect("inspect card", EffectProperties{}, func(game *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		card, ok := FromCard(member).resolveCard(&EffectContext{Game: game, Controller: controller})
		if ok {
			visited = append(visited, card.ID())
		}
		return nil
	})
	if _, ok := a.DiscardCard(second.ID()); !ok {
		t.Fatal("discard failed")
	}
	a.AddToHand(second)
	if err := ForEachSavedCard(cards, member, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 || visited[0] != first.ID() {
		t.Fatalf("visited after return = %v", visited)
	}
	visited = nil
	if err := ForEachSavedCard(cards, member, body).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 2 || visited[0] != first.ID() || visited[1] != second.ID() {
		t.Fatalf("clone visits = %v", visited)
	}
	if len(g.resolution.cardBindings) != 0 || len(clone.resolution.cardBindings) != 0 {
		t.Fatal("loop member escaped")
	}
}

func TestForEachSavedCardScopeAndFailureRestoresFrame(t *testing.T) {
	cards, member := BindCards("drawn"), BindCard("member")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)),
				ForEachSavedCard(cards, member, ReturnCardToHand(FromCard(member)))))),
			WantErr: false,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(ForEachSavedCard(cards, member, ReturnCardToHand(FromCard(member))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "member out of scope",
			Card:          NewInstant("Invalid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)), ForEachSavedCard(cards, member, ReturnCardToHand(FromCard(member))), ReturnCardToHand(FromCard(member))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil member",
			Card:          NewInstant("Invalid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)), ForEachSavedCard(cards, nil, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty member name",
			Card:          NewInstant("Invalid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardsInto(cards)), ForEachSavedCard(cards, BindCard(""), GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
	})
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToLibrary(NewArtifact("One", ""))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardsInto(cards)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	outer := BindCard("outer")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{outer: {id: uuid.New(), visit: uuid.New()}}
	want := errors.New("body failed")
	fail := FuncEffect("fail", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		delete(game.resolution.cardBindings, outer)
		return want
	})
	if err := ForEachSavedCard(cards, member, fail).Apply(ctx); !errors.Is(err, want) || len(g.resolution.cardBindings) != 1 || g.resolution.cardBindings[outer].id == uuid.Nil {
		t.Fatal("loop did not restore the outer card frame")
	}
}

func TestForEachSavedCardStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	for range 4 {
		b.AddToLibrary(NewArtifact("Library card", ""))
	}
	cards, member := BindCards("drawn"), BindCard("member")
	spell := NewInstant("Synthetic loop", "", Spell(Sequence(
		DrawFor(You, Fixed(2), DrawCardsInto(cards)),
		ForEachSavedCard(cards, member, GainLifeFor(You, Fixed(1))),
	)))
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), b.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 22 {
		t.Fatalf("copy life = %d, want 22", got)
	}
	if b.Life() != 20 || len(b.Hand()) != 0 {
		t.Fatal("copy changed the original game")
	}
	g.ResolveStack()
	if b.Life() != 22 || len(b.Hand()) != 2 || a.Life() != 20 {
		t.Fatal("original loop or draw failed")
	}
}

// --- from composable_cast_any_test.go ---
func TestAnyCastConditionOnceForEitherOrBothFacts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		revealed   bool
		controlled bool
		want       int
	}{
		{"neither", false, false, 20},
		{"reveal", true, false, 23},
		{"control", false, true, 23},
		{"both", true, true, 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			spell := NewInstant("Union", "", Spell(IfCast(AnyCast(
				RevealedSubtypeAtCast("Dragon"), ControlledCreatureSubtypeAtCast("Dragon")),
				GainLifeFor(You, Fixed(3)))))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			stack := g.StackObjects()[0]
			stack.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] = tc.controlled
			if tc.revealed {
				stack.CastContext.RevealedAtCast = append(stack.CastContext.RevealedAtCast,
					NewCreature("Dragon", "", 1, 1, WithSubTypes("Dragon")))
			}
			spellCopy := g.CopyStackObjectDirect(stack, b.PlayerID(), false)
			clone := g.Clone()
			spellCopy.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] = false
			spellCopy.CastContext.RevealedAtCast = nil
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != tc.want || clone.GetPlayer(a.PlayerID()).Life() != tc.want {
				t.Fatalf("life=%d clone=%d want=%d", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), tc.want)
			}
		})
	}
}

func TestAnyCastConditionDeclaration(t *testing.T) {
	good := AnyCast(RevealedSubtypeAtCast("Dragon"), ControlledCreatureSubtypeAtCast("Dragon"))
	if got := InspectDefinition(good); got.Role != "cast-condition" || len(got.Children) != 2 {
		t.Fatalf("inspection=%+v", got)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "empty",
			Card:          NewInstant("Invalid", "", Spell(IfCast(AnyCast(), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "at least",
		},
		{
			Name:          "nil",
			Card:          NewInstant("Invalid", "", Spell(IfCast(AnyCast(nil, RevealedSubtypeAtCast("Dragon")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "nil",
		},
		{
			Name:          "invalid child",
			Card:          NewInstant("Invalid", "", Spell(IfCast(AnyCast(RevealedSubtypeAtCast("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name:          "wrong lifecycle",
			Card:          NewArtifact("Wrong lifecycle", "", WithAbility(Activate(Costs(), IfCast(good, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "spell ability",
		},
	})
}

func TestAnyCastCopiesInputConditions(t *testing.T) {
	conditions := []CastCondition{RevealedSubtypeAtCast("Dragon")}
	union := AnyCast(conditions...)
	conditions[0] = ControlledCreatureSubtypeAtCast("Elf")
	ctx := &CastContext{RevealedAtCast: []Card{NewCreature("Dragon", "", 1, 1, WithSubTypes("Dragon"))}}
	if !union.checkCast(ctx) || union.checkCast(nil) {
		t.Fatal("cast condition changed with caller slice or matched without a cast")
	}
}

// --- from composable_cleanup_history_test.go ---
func TestCleanupHistoryAcrossPriorityRounds(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(victim.ID()))
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())
	g.trackers.Turn.RecordAttacked(attacker.ID())
	register := func() {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, CreaturesDiedThisTurn())}})
	}
	register()
	added := false
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		if game.CreatureDeaths() == 0 {
			t.Fatal("cleanup cleared deaths before priority")
		}
		if !added {
			added = true
			next := NewCreature("Cleanup victim", "", 1, 1)
			next.SetOwner(b.PlayerID())
			game.PutOnBattlefield(next, b.PlayerID())
			game.DoSacrifice(game.FindPermanent(next.ID()))
			register()
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Cleanup)
	if a.Life() != 24 || g.GetCleanupPriorityRounds() != 2 {
		t.Fatalf("life = %d, priority rounds = %d; want 24 and 2", a.Life(), g.GetCleanupPriorityRounds())
	}
	if g.CreatureDeaths() != 0 || !g.trackers.Turn.AttackedLastTurn(a.PlayerID())[attacker.ID()] {
		t.Fatal("final cleanup did not reset deaths and preserve attacker history")
	}
	if g.trackers.Turn.PlayerLifeGained(a.PlayerID()) != 0 {
		t.Fatal("life history survived final cleanup")
	}
}

func TestCleanupHistoryStateBasedActions(t *testing.T) {
	for _, trigger := range []bool{false, true} {
		t.Run(map[bool]string{false: "without trigger", true: "with trigger"}[trigger], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := NewCreature("Temporary survivor", "", 1, 0)
			victim.SetOwner(a.PlayerID())
			g.PutOnBattlefield(victim, a.PlayerID())
			if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(1))), victim.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if trigger {
				g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn())))), a.PlayerID())
			}
			seen := false
			g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
				seen = true
				if game.CreatureDeaths() != 1 {
					t.Fatalf("cleanup priority deaths = %d, want 1", game.CreatureDeaths())
				}
				return PriorityAction{Type: PriorityPass}
			})
			g.RunStepWithPriority(Cleanup)
			wantLife := 20
			if trigger {
				wantLife++
			}
			if !seen || g.GetCleanupPriorityRounds() != 1 || a.Life() != wantLife || g.CreatureDeaths() != 0 {
				t.Fatalf("priority = %v, rounds = %d, life = %d, deaths = %d", seen, g.GetCleanupPriorityRounds(), a.Life(), g.CreatureDeaths())
			}
		})
	}
}

func TestCleanupHistoryCloneAndActivationLimits(t *testing.T) {
	for _, granted := range []bool{false, true} {
		t.Run(map[bool]string{false: "intrinsic", true: "granted"}[granted], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))))
			if granted {
				source = NewArtifact("Source", "")
				g.PutOnBattlefield(NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))))), a.PlayerID())
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.trackers.Turn.RecordCreatureDeath()
			g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, CreaturesDiedThisTurn())}})
			if !g.doCleanupActions() {
				t.Fatal("cleanup did not request priority")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if clone.doCleanupActions() {
				t.Fatal("final cleanup requested priority")
			}
			if clone.CreatureDeaths() != 0 || g.CreatureDeaths() != 1 {
				t.Fatal("clone reset changed original history")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
				t.Fatal("cleanup priority reset activation limit")
			}
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal("final cleanup retained activation limit", err)
			}
			if g.GetPlayer(a.PlayerID()).Life() != 21 {
				t.Fatal("clone changed original life")
			}
		})
	}
}

func TestCleanupHistoryPhasedActivationReset(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.MutablePermanent(source.ID()).PhasedOut = true
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	clone.MutablePermanentIncludingPhased(source.ID()).PhasedOut = false
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal("phased permanent retained prior turn limit", err)
	}
	g.MutablePermanentIncludingPhased(source.ID()).PhasedOut = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("clone reset changed original limit")
	}
}

// --- from composable_combined_timing_test.go ---
func TestCombinedActivationTiming(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, granted := range []bool{false, true} {
			for _, mana := range []bool{false, true} {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				var first, second ActivationTimingOption = OnlyDuring(Upkeep), OnlyOnYourTurn()
				if reverse {
					first, second = second, first
				}
				var body = GainLifeFor(You, Fixed(1))
				if mana {
					body = AddMana(Blue, 1)
				}
				source := NewArtifact("Source", "")
				if granted {
					donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(TapCost(This)), body, first, second))))
					g.PutOnBattlefield(donor, a.PlayerID())
				} else {
					source = NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), body, first, second)))
				}
				g.PutOnBattlefield(source, b.PlayerID())
				g.ApplyContinuousEffects()
				for _, state := range []struct {
					active  int
					step    PhaseStep
					allowed bool
				}{{0, Upkeep, false}, {1, PrecombatMain, false}, {1, Upkeep, true}} {
					g.SetActivePlayerIndex(state.active)
					g.SetStep(state.step)
					_, planned := manaSourceAbilityForPlanning(g.FindPermanent(source.ID()).RuntimeAbilities[0], source.ID(), 0, g)
					if mana && planned != state.allowed {
						t.Fatal("mana planning ignored combined timing")
					}
					err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil)
					if !state.allowed {
						if err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
							t.Fatal("invalid activation paid costs")
						}
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					g.SetActivePlayerIndex(0)
					g.SetStep(EndStep)
					g.ResolveStack()
					if mana {
						if b.ManaPool().TotalMana() != 1 {
							t.Fatal("mana ability failed")
						}
					} else if b.Life() != 21 {
						t.Fatal("timing rechecked at resolution")
					}
				}
			}
		}
	}
}

// --- from composable_control_test.go ---
func TestChangeControllerLifetimeAndIdentity(t *testing.T) {
	for _, change := range []string{"none", "source leaves", "target phases", "target returns", "cannot change"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, target := controllerTestGame(t)
			sourceCard := NewCreature("Source", "", 1, 1)
			sourceCard.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(sourceCard, b.PlayerID())
			if change == "cannot change" {
				target.GrantBaseAttr(AttrCantChangeControl)
			}
			ref := DeclarePermanentTarget("target", IsCreature)
			ability := Spell(Until(EndOfTurn, ChangeController(ref, You)), WithTargets(ref))
			card := NewInstant("Control", "", WithAbility(ability))
			card.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition("Control", card); err != nil {
				t.Fatal(err)
			}
			obj := &StackObject{ID: uuid.New(), SourceID: source.ID(), Card: card, Controller: b.PlayerID(), Targets: []uuid.UUID{target.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			g.ResolveStackObject(obj)
			if change == "source leaves" {
				g.RemoveFromBattlefield(source)
			}
			if change == "target phases" {
				g.PhaseIn(g.PhaseOut(target.ID()))
			}
			if change == "target returns" {
				g.RemoveFromBattlefield(target)
				target = g.PutOnBattlefield(target.Card, a.PlayerID())
			}
			g.ApplyContinuousEffects()
			want := b.PlayerID()
			if change == "target returns" || change == "cannot change" {
				want = a.PlayerID()
			}
			if got := target.ControllerID(); got != want {
				t.Fatalf("controller = %s, want %s", got, want)
			}
			g.effects.RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if got := target.ControllerID(); got != a.PlayerID() {
				t.Fatalf("controller after expiry = %s, want %s", got, a.PlayerID())
			}
		})
	}
}

func TestChangeControllerTimestampAndClone(t *testing.T) {
	g, a, b, target := controllerTestGame(t)
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	effect := Until(EndOfTurn, ChangeController(This, You))
	if err := ApplyEffect(g, effect, target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("later controller = %s", got)
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if got := clone.FindPermanent(target.ID()).ControllerID(); got != b.PlayerID() {
		t.Fatalf("clone controller after expiry = %s", got)
	}
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("clone changed original controller to %s", got)
	}
	if effect.Properties().Outcome != OutcomeDetriment {
		t.Fatal("control change lost detrimental metadata")
	}
}

func TestChangeControllerCopiedSpellUsesCopyController(t *testing.T) {
	g, a, b, target := controllerTestGame(t)
	ref := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Until(EndOfTurn, ChangeController(ref, You)), WithTargets(ref))
	card := NewInstant("Copy Control", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	g.ResolveStackObject(copyObj)
	if got := target.ControllerID(); got != b.PlayerID() {
		t.Fatalf("copy controller = %s, want %s", got, b.PlayerID())
	}
	g.ResolveStackObject(obj)
	if got := target.ControllerID(); got != a.PlayerID() {
		t.Fatalf("original controller = %s, want %s", got, a.PlayerID())
	}
}

func TestChangeControllerValidation(t *testing.T) {
	target := DeclarePermanentTarget("target", IsCreature)
	player := DeclarePlayerTarget("player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil subject",
			Card:          NewInstant("Invalid Control", "", WithAbility(Spell(Until(EndOfTurn, ChangeController(nil, You))))),
			WantErr:       true,
			WantErrSubstr: ".subject",
		},
		{
			Name:          "nil controller",
			Card:          NewInstant("Invalid Control", "", WithAbility(Spell(Until(EndOfTurn, ChangeController(target, nil)), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: ".controller",
		},
		{
			Name:          "undeclared controller",
			Card:          NewInstant("Invalid Control", "", WithAbility(Spell(Until(EndOfTurn, ChangeController(target, player)), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: ".controller",
		},
		{
			Name:          "wrong duration",
			Card:          NewInstant("Invalid Control", "", WithAbility(Spell(Until(EndOfCombat, ChangeController(target, You)), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: "requires Until(EndOfTurn)",
		},
		{
			Name:          "source duration",
			Card:          NewInstant("Invalid Control", "", WithAbility(Spell(UntilSourceLeaves(This, ChangeController(target, You)), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: "requires Until(EndOfTurn)",
		},
		{
			Name:          "static duration",
			Card:          NewCreature("Invalid Static", "", 1, 1, WithAbility(Static(While(Controls(IsCreature), ChangeController(This, You))))),
			WantErr:       true,
			WantErrSubstr: "requires Until(EndOfTurn)",
		},
	})
}

// --- from composable_controlled_creature_entry_test.go ---
func TestControlledCreatureEntryIncludesSourceAndOtherCreatures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		flying   bool
		other    bool
		opponent bool
		phased   bool
		want     int
	}{
		{"source", true, false, false, false, 3},
		{"source without keyword", false, false, false, false, 0},
		{"other controlled", true, true, false, false, 2},
		{"opposing", true, true, true, false, 0},
		{"phased source", true, true, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			power := BindNumber("entry power")
			watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
				CreatureYouControlEnters().HasKeyword(Flying).WithPower(power),
				GainLifeFor(You, FromNumber(power)))))
			if tc.flying {
				watcher = NewCreature("Watcher", "", 3, 3, WithKeyword(Flying), WithAbility(WhenEvent(
					CreatureYouControlEnters().HasKeyword(Flying).WithPower(power),
					GainLifeFor(You, FromNumber(power)))))
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.other {
				g.PutTriggersOnStack()
				g.ResolveStack()
				if tc.phased {
					g.PhaseOut(watcher.ID())
				}
				controller := a.PlayerID()
				if tc.opponent {
					controller = b.PlayerID()
				}
				g.PutOnBattlefield(NewCreature("Other", "", 2, 2, WithKeyword(Flying)), controller)
			}
			baseline := a.Life()
			g.PutTriggersOnStack()
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != baseline || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := a.Life() - baseline; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestControlledCreatureEntryDeclaration(t *testing.T) {
	power := BindNumber("power")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouControlEnters().WithPower(power), GainLifeFor(You, FromNumber(power)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    valid,
			WantErr: false,
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(valid.Abilities()[0]); got.Role != "event-trigger" {
					t.Fatalf("inspection: %+v", got)
				}
			},
		},
		{
			Name: "missing capture",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				CreatureYouControlEnters().WithPower(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestControlledCreatureEntryKeepsQueuedSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouControlEnters(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("queued trigger lost original controller: %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_controller_damage_source_test.go ---
func TestControllerDamageSourceCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, source := BindNumber("amount"), BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, FromNumber(amount)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	dealer := NewCreature("Dealer", "", 2, 2, WithKeyword(Lifelink))
	g.PutOnBattlefield(dealer, b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealDamageToPlayer(a, 1, dealer.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	g.DealDamageToPlayer(a, 2, dealer.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(dealer.ID()))
	g.PutOnBattlefield(dealer, a.PlayerID())
	g.PhaseOut(dealer.ID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 16 {
			t.Fatalf("damaged player life = %d", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 24 {
			t.Fatalf("old source controller life = %d", got)
		}
	}
}

func TestControllerDamageSourceDeclaration(t *testing.T) {
	amount, source := BindNumber("amount"), BindDamageSource("source")
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(
		ControllerIsDealtDamage(amount).WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    good,
			WantErr: false,
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(ControllerIsDealtDamage(amount).WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 2 {
					t.Fatalf("inspection = %#v", got)
				}
			},
		},
		{
			Name: "nil source",
			Card: NewEnchantment("Bad", "", WithAbility(WhenEvent(
				ControllerIsDealtDamage(amount).WithSource(nil), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty name source",
			Card: NewEnchantment("Bad", "", WithAbility(WhenEvent(
				ControllerIsDealtDamage(amount).WithSource(BindDamageSource("")), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate source",
			Card: NewEnchantment("Bad", "", WithAbility(WhenEvent(
				ControllerIsDealtDamage(amount).WithSource(source).WithSource(source), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound source",
			Card: NewEnchantment("Bad", "", WithAbility(WhenEvent(
				ControllerIsDealtDamage(amount), DealDamageFrom(FromDamageSource(source), You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_creature_entry_any_test.go ---
func TestAnotherCreatureEntersAnyController(t *testing.T) {
	for _, tc := range []struct {
		name       string
		controller int
		phase      bool
		want       int
	}{
		{"same controller", 0, false, 3},
		{"other controller", 1, false, 3},
		{"phased watcher", 1, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			power := BindNumber("entered power")
			watcher := NewCreature("Watcher", "", 1, 4, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithPower(power), AddCountersTo(This, P1P1, FromNumber(power)))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			entered := NewCreature("Entered", "", 3, 3)
			entered.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if tc.controller == 1 {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entered, controller)
			g.PutTriggersOnStack()
			g.FindPermanent(entered.ID()).AddCounter(P1P1, 2)
			clone := g.Clone()
			clone.ResolveStack()
			if got := g.FindPermanentIncludingPhased(watcher.ID()).Counters[P1P1]; got != 0 {
				t.Fatalf("clone changed original counters=%d", got)
			}
			g.ResolveStack()
			if got := g.FindPermanentIncludingPhased(watcher.ID()).Counters[P1P1]; int(got) != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestAnotherCreatureEntersDeclaration(t *testing.T) {
	power := BindNumber("power")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithPower(power), GainLifeFor(You, FromNumber(power))))),
			WantErr: false,
		},
		{
			Name: "duplicate capture",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithPower(power).WithPower(power), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestAnotherCreatureEntersExcludesSelfAndNoncreature(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters(), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 0 {
		t.Fatalf("unexpected triggers=%d", got)
	}
}

// --- from composable_creature_entry_controller_test.go ---
func TestCreatureEntryControllerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("entrant controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(2)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entrant := NewCreature("Entrant", "", 1, 1)
	entrant.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entrant, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(entrant.ID()))
	g.PutOnBattlefield(entrant, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 24 {
		t.Fatalf("life A=%d B=%d, want 20 and 24", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 24 {
		t.Fatal("clone lost captured controller")
	}
}

func TestCreatureEntryControllerCapturePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("entrant controller")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureEnters().WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PhaseOut(watcher.ID())
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn([]uuid.UUID{watcher.ID()})
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutTriggersOnStack()
	g.PhaseOut(watcher.ID())
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatalf("life=%d, want 21", b.Life())
	}
}

func TestCreatureEntryControllerCaptureValidation(t *testing.T) {
	missing := BindPlayer("missing")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "nil controller",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty controller binding",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithController(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate controller",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureEnters().WithController(BindPlayer("one")).WithController(BindPlayer("two")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "undominated binding",
			Card: NewCreature("Missing", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureEnters(), GainLifeFor(FromPlayer(missing), Fixed(1))))),
			WantErr: true,
		},
	})
}

// --- from composable_creature_entry_power_test.go ---
func TestCreatureEntryPowerCapture(t *testing.T) {
	for _, tc := range []struct {
		name              string
		opponent, phased  bool
		depart, copyStack bool
		want              int
	}{
		{"controlled", false, false, false, false, 3},
		{"opponent", true, false, false, false, 0},
		{"phased source", false, true, false, false, 0},
		{"departed and returned", false, false, true, false, 3},
		{"stack copy", false, false, false, true, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			power := BindNumber("entry power")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPower(power), GainLifeFor(You, FromNumber(power)))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			creature := NewCreature("Entered", "", 3, 3)
			creature.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(creature, controller)
			g.PutTriggersOnStack()
			if !tc.opponent && !tc.phased {
				g.FindPermanent(creature.ID()).AddCounter(P1P1, 2)
			}
			if tc.depart {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.PutOnBattlefield(creature, b.PlayerID())
			}
			if tc.copyStack && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("stack copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if got := a.Life() + b.Life() - 40; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestCreatureEntryPowerDeclaration(t *testing.T) {
	power := BindNumber("power")
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().WithPower(power), AddCountersTo(This, P1P1, FromNumber(power)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    valid,
			WantErr: false,
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(valid.Abilities()[0]); got.Role != "event-trigger" || len(got.Children) == 0 {
					t.Fatalf("inspection: %+v", got)
				}
			},
		},
		{
			Name: "nil power",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPower(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "empty name power",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPower(BindNumber("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "duplicate power",
			Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPower(power).WithPower(power), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound power",
			Card: NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters(), GainLifeFor(You, FromNumber(power))))),
			WantErr: true,
		},
	})
}

// --- from composable_creature_entry_subtype_test.go ---
func TestCreatureEntrySubtypeFilter(t *testing.T) {
	for _, tc := range []struct {
		name, subtype                string
		token, opponent, phaseSource bool
		want                         int
	}{
		{"dragon", "Dragon", false, false, false, 1},
		{"other subtype", "Goblin", false, false, false, 0},
		{"token dragon", "Dragon", true, false, false, 0},
		{"opposing dragon", "Dragon", false, true, false, 0},
		{"phased source", "Dragon", false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().Nontoken().OfSubtype("Dragon"), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			var other *BaseCard
			if tc.token {
				other = NewToken("Other", 1, 1, []CardType{TypeCreature}, []string{tc.subtype})
			} else {
				other = NewCreature("Other", "", 1, 1, WithSubTypes(tc.subtype))
			}
			other.SetOwner(a.PlayerID())
			if tc.phaseSource {
				g.PhaseOut(source.ID())
			}
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(other, controller)
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

func TestCreatureEntrySubtypeDeclaration(t *testing.T) {
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().OfSubtype("Dragon"), GainLifeFor(You, Fixed(1)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    good,
			WantErr: false,
		},
		{
			Name: "empty subtype",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().OfSubtype(""), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "non-creature subtype Forest",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().OfSubtype("Forest"), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

// --- from composable_creature_entry_test.go ---
func TestAnotherNontokenControlledCreatureEnters(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"controlled", 1}, {"opponent", 0}, {"stolen", 1}, {"lost", 0},
		{"self", 0}, {"token", 0}, {"noncreature", 0}, {"phased source", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().Nontoken(), GainLifeFor(You, Fixed(1)))))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			if tc.name == "self" {
				g.PutOnBattlefield(source, a.PlayerID())
			} else {
				g.PutOnBattlefield(source, a.PlayerID())
				other := NewCreature("Other", "", 1, 1)
				switch tc.name {
				case "token":
					other = NewToken("Other", 1, 1, []CardType{TypeCreature}, nil)
				case "noncreature":
					other = NewArtifact("Other", "")
				}
				owner, controller := a.PlayerID(), a.PlayerID()
				if tc.name == "opponent" || tc.name == "stolen" {
					owner = b.PlayerID()
				}
				if tc.name == "opponent" || tc.name == "lost" {
					controller = b.PlayerID()
				}
				other.SetOwner(owner)
				if tc.name == "phased source" {
					g.PhaseOut(source.ID())
				}
				g.PutOnBattlefield(other, controller)
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
			if a.Life() != 20+tc.want {
				t.Fatalf("life=%d, want %d", a.Life(), 20+tc.want)
			}
		})
	}
}

func TestAnotherNontokenControlledCreatureEntersInspection(t *testing.T) {
	card := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().Nontoken(), GainLifeFor(You, Fixed(1)))))
	meta := InspectDefinition(card.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Text != "another nontoken creature you control enters" {
		t.Fatalf("wrong inspection: %+v", meta)
	}
}

func TestAnotherNontokenControlledCreatureEntersSourceIdentityAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().Nontoken(), Sequence(
			GainLifeFor(You, Fixed(1)), AddCountersTo(This, P1P1, Fixed(1))))))
	source.SetOwner(a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("entry did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("stack copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("copy did not retain its own controller")
	}
	if got := g.FindPermanent(source.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("returned source got %d counters", got)
	}
}

// --- from composable_damage_cleanup_test.go ---
func TestDamageCleanupStateBasedDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Observer", "", 2, 2, WithAbility(
		CreatureDealtDamageBySourceDiesTrigger(GainLifeFor(You, Fixed(1)), false)))
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Temporary survivor", "", 1, 0)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(3))), victim.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.RunStepWithPriority(Cleanup)
	if g.FindPermanent(victim.ID()) != nil || a.Life() != 21 {
		t.Fatalf("cleanup death must trigger once; life = %d, want 21", a.Life())
	}
	if g.GetDamageSources(victim.ID())[source.ID()] {
		t.Fatal("final cleanup retained damage history")
	}
}

func TestDamageCleanupPriorityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 2)
	victim := NewCreature("Victim", "", 2, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.SetDamageReflection(b.PlayerID(), victim.ID(), source.ID())
	register := func() {
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	}
	register()
	for round := range 2 {
		if !g.doCleanupActions() {
			t.Fatal("cleanup did not request priority")
		}
		if !g.GetDamageSources(victim.ID())[source.ID()] {
			t.Fatal("cleanup priority lost damage history")
		}
		if g.FindPermanent(victim.ID()).Damage != 0 {
			t.Fatal("cleanup retained marked damage")
		}
		if _, ok := g.damage.GetDamageReflection(b.PlayerID()); ok {
			t.Fatal("cleanup retained the expired reflection effect")
		}
		g.ResolveStack()
		if round == 0 {
			g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, victim.ID())
			register()
		}
	}
	if !g.GetDamageSources(victim.ID())[victim.ID()] {
		t.Fatal("second cleanup lost damage dealt during priority")
	}
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	if len(clone.GetDamageSources(victim.ID())) != 0 || len(g.GetDamageSources(victim.ID())) != 2 {
		t.Fatal("clone cleanup did not reset history independently")
	}
	if !clone.HasDealtDamageToPlayer(source.ID(), b.PlayerID()) || !clone.HasDealtDamageToPermanent(source.ID(), victim.ID()) {
		t.Fatal("final cleanup lost game damage history")
	}
	g.RunStepWithPriority(Cleanup)
	if len(g.GetDamageSources(victim.ID())) != 0 {
		t.Fatal("final cleanup retained damage history")
	}
}

// --- from composable_damage_each_result_test.go ---
func TestDealDamageEachTotalRecordsActualDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	total := BindNumber("total damage")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 || victim.Damage != 0 {
		t.Fatal("clone changed original game")
	}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 || b.Life() != 19 || g.FindPermanent(victim.ID()).Damage != 2 {
		t.Fatalf("life = %d, %d; damage = %d; want 23, 19, 2", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
	}
}

func TestDealDamageEachTotalClearsOnAbsentSourceAndZeroDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	total := BindNumber("total damage")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(total)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(0), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 18 {
		t.Fatal("zero damage retained an earlier result")
	}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 16 || b.Life() != 16 {
		t.Fatal("a new resolution retained an earlier result")
	}
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 16 || b.Life() != 16 {
		t.Fatal("absent source retained an earlier result")
	}
}

func TestDealDamageEachTotalFollowsReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	total := BindNumber("total")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 24 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 4 {
		t.Fatalf("life = %d, %d; victim damage = %d; want 24, 20, 4", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
	}
}

func TestDealDamageEachTotalExcludesReturnedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Target", "", 2, 9)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("target", 1, 1, IsCreature)
	g.resolution.targetSpecs = []Target{targets.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	total := BindNumber("total")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2), DamageTotalInto(total)),
		GainLifeFor(You, FromNumber(total)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{old.ID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 18 || returned.Damage != 0 {
		t.Fatalf("life = %d, %d; returned damage = %d; want 22, 18, 0", a.Life(), b.Life(), returned.Damage)
	}
}

func TestDealDamageEachTotalDeclaration(t *testing.T) {
	total := BindNumber("total")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(Sequence(
				DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(total)),
				GainLifeFor(You, FromNumber(total)),
			))),
			WantErr: false,
		},
		{
			Name:    "nil total binding",
			Card:    NewInstant("Invalid", "", Spell(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(nil)))),
			WantErr: true,
		},
		{
			Name:    "nil option",
			Card:    NewInstant("Invalid", "", Spell(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), nil))),
			WantErr: true,
		},
		{
			Name:    "duplicate total binding",
			Card:    NewInstant("Invalid", "", Spell(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2), DamageTotalInto(total), DamageTotalInto(total)))),
			WantErr: true,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Missing", "", Spell(GainLifeFor(You, FromNumber(total)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
	})
}

// --- from composable_damage_each_test.go ---
func TestDealDamageEachMixedRecipientsObserveOneBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Recipient", "", 4, 4), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if a.Life() != 18 || b.Life() != 19 || creature.Damage != 2 {
			t.Errorf("callback saw partial batch: A=%d B=%d creature=%d", a.Life(), b.Life(), creature.Damage)
		}
	})
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observed != 3 {
		t.Fatalf("damage callbacks = %d, want 3", observed)
	}
}

func TestDealDamageEachFiltersAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Recipient", "", 4, 4), b.PlayerID())
	g.PhaseOut(creature.ID())
	clone := g.Clone()
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2))
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 || creature.Damage != 0 {
		t.Fatal("batch changed original game")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone did not apply two damage assignments with captured lifelink")
	}
}

func TestDealDamageEachDeclarationValidation(t *testing.T) {
	targets := DeclarePermanentTargets("creature", 1, 1, IsCreature)
	multiple := DeclarePermanentTargets("multiple", 1, 2, IsCreature)
	other := DeclarePermanentTargets("other", 0, 2, IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewSorcery("Valid", "", WithAbility(Spell(DealDamageEach(This,
				DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2))))),
			WantErr: false,
		},
		{
			Name: "invalid recipient",
			Card: NewSorcery("Invalid", "", WithAbility(Spell(DealDamageEach(This,
				DamageRecipients(nil, MatchingPermanents(IsCreature)), Fixed(2))))),
			WantErr:       true,
			WantErrSubstr: "player",
		},
		{
			Name: "missing target declaration",
			Card: NewSorcery("Missing", "", WithAbility(Spell(DealDamageEach(This,
				DamageRecipients(EachPlayer, targets), Fixed(2))))),
			WantErr:       true,
			WantErrSubstr: "target binding requires",
		},
		{
			Name: "declared target",
			Card: NewSorcery("Declared", "", WithAbility(Spell(DealDamageEach(This,
				DamageRecipients(EachPlayer, targets), Fixed(2)), WithTargets(targets)))),
			WantErr: false,
		},
		{
			Name: "one variable target group supported",
			Card: NewSorcery("Supported", "", WithAbility(Spell(DealDamageEach(This,
				DamageRecipients(EachPlayer, multiple), Fixed(2)), WithTargets(multiple)))),
			WantErr: false,
		},
		{
			Name: "two variable target groups ambiguous",
			Card: NewSorcery("Ambiguous", "", WithAbility(Spell(DealDamageEach(This,
				DamageRecipients(EachPlayer, multiple), Fixed(2)), WithTargets(multiple, other)))),
			WantErr:       true,
			WantErrSubstr: "recorded selection boundaries",
		},
	})
}

func TestDealDamageEachMultiplePermanentTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 3, 4), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 3, 4), b.PlayerID())
	third := g.PutOnBattlefield(NewCreature("Third", "", 3, 4), b.PlayerID())
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ability := Spell(DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2)), WithTargets(targets))
	card := NewSorcery("Batch", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Batch", card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: expandTargetSpecs(ability.Targets(), []uuid.UUID{first.ID(), second.ID()}, 0), Effects: ability.Effects()}
	g.pushStack(obj)
	if obj.targetIncarnations[first.ID()] != first.incarnationID || obj.targetIncarnations[second.ID()] != second.incarnationID {
		t.Fatal("target incarnations were not captured")
	}
	if !g.isTargetStillLegal(first.ID(), card, a.PlayerID(), obj.TargetZones[first.ID()], obj.TargetSpecs[0]) {
		t.Fatal("selected creature became illegal before resolution")
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	if copyObj == nil || len(copyObj.TargetSpecs) != 2 {
		t.Fatal("copy lost the selected group")
	}
	cloneSecond := clone.FindPermanent(second.ID())
	clone.RemoveFromBattlefield(cloneSecond)
	clone.ResolveStackObject(copyObj)
	if clone.FindPermanent(first.ID()).Damage != 2 || clone.FindPermanent(third.ID()).Damage != 0 {
		t.Fatal("copy damaged an unselected or illegal permanent")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("copy lost the player recipients")
	}
	if first.Damage != 0 || second.Damage != 0 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed the original game")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()).Damage != 2 || g.FindPermanent(second.ID()).Damage != 2 || g.FindPermanent(third.ID()).Damage != 0 || a.Life() != 18 || b.Life() != 18 {
		t.Fatal("batch did not damage exactly the selected recipients")
	}
}

func TestDealDamageEachTargetIncarnationAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Target", "", 2, 3)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("creature", 1, 1, IsCreature)
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2))
	g.resolution.targetSpecs = []Target{targets.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{old.ID()}); err != nil {
		t.Fatal(err)
	}
	if current.Damage != 0 || a.Life() != 18 || b.Life() != 18 {
		t.Fatal("returned target received damage or player set was lost")
	}
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), []uuid.UUID{old.ID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 18 {
		t.Fatal("absent source dealt damage")
	}
}

// --- from composable_damage_from_test.go ---
func TestDealDamageFromSourceAndRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 2, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	shield := PreventNextDamage(You, Fixed(1))
	if err := ApplyEffect(g, shield, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.resolution.sourceCaptured = true
	g.resolution.sourceIncarnation = old.incarnationID
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(card, b.PlayerID())
	g.zones.ClearLKI()
	if got := g.CaptureDamageSource(card.ID()); !got.Valid() || got.Incarnation() != old.incarnationID {
		t.Fatalf("source snapshot = %+v; captured = %t; incarnation = %s", got, g.resolution.sourceCaptured, g.resolution.sourceIncarnation)
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		if err := ApplyEffect(game, DealDamageFrom(This, You, Fixed(3)), card.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("life totals = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
	if err := ApplyEffect(clone, DealDamageFrom(This, You, Fixed(1)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("clone damage affected original")
	}
}

func TestDealDamageFromPermanentTargetAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
	target := DeclareDamageTarget("recipient")
	effect := DealDamageFrom(This, target, Fixed(2))
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 2 {
		t.Fatalf("damage = %d", victim.Damage)
	}
	g.PhaseOut(victim.ID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 2 {
		t.Fatal("phased recipient received damage")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 {
		t.Fatal("player recipient did not receive damage")
	}
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(2)), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("absent source dealt damage")
	}
}

func TestDealDamageFromDefinitionValidation(t *testing.T) {
	target := DeclareDamageTarget("recipient")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(DealDamageFrom(This, target, Fixed(1)), WithTargets(target))),
			WantErr: false,
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(DealDamageFrom(This, You, Fixed(1))); got.Role != "resolving-effect" || len(got.Children) != 3 {
					t.Fatalf("inspection = %+v", got)
				}
			},
		},
		{
			Name:          "missing target declaration",
			Card:          NewInstant("Invalid", "", Spell(DealDamageFrom(This, target, Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "target binding requires",
		},
		{
			Name:          "nil source declaration",
			Card:          NewInstant("Missing", "", Spell(DealDamageFrom(nil, You, Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "damage source",
		},
	})
}

func TestDealDamageFromSpellSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Borrowed Bolt", "", nil)
	card.SetOwner(a.PlayerID())
	end := g.resolution.Begin(&StackObject{Card: card, Controller: b.PlayerID(), SourceID: card.ID()})
	defer end()
	if err := ApplyEffect(g, DealDamageFrom(This, You, Fixed(2)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 || !g.HasDealtDamageToPlayer(card.ID(), b.PlayerID()) {
		t.Fatal("resolving spell did not deal damage as source")
	}
}

func TestDealDamageFromDoesNotFollowReturnedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Victim", "", 2, 5)
	old := g.PutOnBattlefield(card, b.PlayerID())
	target := DeclareDamageTarget("recipient")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(2)), source.ID(), a.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Damage != 0 {
		t.Fatal("damage followed a returned target")
	}
}

// --- from composable_damage_history_set_test.go ---
func TestDamageHistoryRecipientSetsAndSimultaneousAction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 3), a.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
	untouched := g.PutOnBattlefield(NewPlaneswalker("Untouched", "", 5), b.PlayerID())
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(source.ID()))
	g.DealDamageToPermanentFromSource(walker, 1, g.CaptureDamageSource(source.ID()))
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	players := PlayersDealtDamageBy(This, OpponentsOf(You))
	permanents := PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker))
	if got := players.resolvePlayers(ctx); len(got) != 1 || got[0] != b.PlayerID() {
		t.Fatalf("player history = %v", got)
	}
	if got := permanents.resolvePermanents(ctx); len(got) != 1 || got[0] != walker {
		t.Fatalf("permanent history = %v", got)
	}
	if err := DealDamageEach(This, DamageRecipients(players, permanents), Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 18 || walker.Counters[Loyalty] != 3 || untouched.Counters[Loyalty] != 5 {
		t.Fatal("damage history selection changed recipients or damage outcome")
	}
}

func TestDamageHistoryRecipientSetsIdentityPhasingAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 2, 3)
	oldSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	walkerCard := NewPlaneswalker("Walker", "", 5)
	oldWalker := g.PutOnBattlefield(walkerCard, b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: oldSource.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: oldSource.incarnationID}
	players := PlayersDealtDamageBy(This, OpponentsOf(You))
	permanents := PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker))
	g.AddPreventionShield(oldWalker.ID(), 1)
	g.DealDamageToPermanentFromSource(oldWalker, 1, g.CaptureDamageSource(oldSource.ID()))
	if len(permanents.resolvePermanents(ctx)) != 0 {
		t.Fatal("prevented damage entered the selected set")
	}
	g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(oldSource.ID()))
	g.DealDamageToPermanentFromSource(oldWalker, 1, g.CaptureDamageSource(oldSource.ID()))
	g.PhaseOut(oldWalker.ID())
	if len(permanents.resolvePermanents(ctx)) != 0 || len(players.resolvePlayers(ctx)) != 1 {
		t.Fatal("phasing changed player history or selected a phased permanent")
	}
	g.PhaseIn([]uuid.UUID{oldWalker.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(oldWalker.ID()))
	if len(permanents.resolvePermanents(&EffectContext{Game: clone, SourceID: oldSource.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: oldSource.incarnationID})) != 0 || len(permanents.resolvePermanents(ctx)) != 1 {
		t.Fatal("clone and original share a recipient selection")
	}
	g.RemoveFromBattlefield(oldWalker)
	g.PutOnBattlefield(walkerCard, b.PlayerID())
	if len(permanents.resolvePermanents(ctx)) != 0 {
		t.Fatal("returned recipient inherited history")
	}
	g.RemoveFromBattlefield(oldSource)
	newSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if len(players.resolvePlayers(ctx)) != 1 {
		t.Fatal("departed source lost its recorded incarnation")
	}
	g.zones.ClearLKI()
	if len(players.resolvePlayers(ctx)) != 1 {
		t.Fatal("cleanup erased durable source history")
	}
	ctx.sourceIncarnation = newSource.incarnationID
	if len(players.resolvePlayers(ctx)) != 0 {
		t.Fatal("returned source inherited history")
	}
}

func TestDamageHistoryRecipientSetsValidation(t *testing.T) {
	set := PlayersDealtDamageBy(This, OpponentsOf(You))
	permanents := PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(Enters(This), DealDamageEach(This, DamageRecipients(set, permanents), Fixed(1))))),
			WantErr: false,
			Inspect: func(t *testing.T) {
				if InspectDefinition(set).Role != "player-set" || InspectDefinition(permanents).Role != "permanent-set" {
					t.Fatal("history sets lack inspection roles")
				}
			},
		},
		{
			Name:    "nil source",
			Card:    NewInstant("nil source", "", Spell(DealDamageEach(This, DamageRecipients(PlayersDealtDamageBy(nil, EachPlayer), permanents), Fixed(1)))),
			WantErr: true,
		},
		{
			Name:    "nil players",
			Card:    NewInstant("nil players", "", Spell(DealDamageEach(This, DamageRecipients(PlayersDealtDamageBy(This, nil), permanents), Fixed(1)))),
			WantErr: true,
		},
		{
			Name:    "nil permanents",
			Card:    NewInstant("nil permanents", "", Spell(DealDamageEach(This, DamageRecipients(set, PermanentsDealtDamageBy(This, nil)), Fixed(1)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("undeclared target", "", Spell(DealDamageEach(This, DamageRecipients(PlayersDealtDamageBy(DamageSourceOf(DeclarePermanentTarget("source", IsCreature)), EachPlayer), permanents), Fixed(1)))),
			WantErr: true,
		},
	})
}

// --- from composable_damage_permanent_result_test.go ---
func TestDealDamageFromCapturesFinalPermanentRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 8), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	result := BindPermanent("damaged permanent")
	amount := BindNumber("damage dealt")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(2), DamageAmountInto(amount), DamagePermanentInto(result)),
		AddCountersTo(From(result), P1P1, FromNumber(amount)),
	)
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if victim.Counters[P1P1] != 0 {
		t.Fatal("clone changed original recipient")
	}
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	gotVictim := g.FindPermanent(victim.ID())
	if gotVictim.Damage != 2 || gotVictim.Counters[P1P1] != 2 || b.Life() != 20 {
		t.Fatalf("damage=%d counters=%d life=%d", gotVictim.Damage, gotVictim.Counters[P1P1], b.Life())
	}
}

func TestDealDamageFromPermanentResultClearsAndKeepsIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	result := BindPermanent("damaged permanent")
	for _, amount := range []int{0, 2} {
		if amount == 2 {
			if err := ApplyEffect(g, PreventNextDamage(This, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		effect := Sequence(Snapshot(This, result), DealDamageFrom(This, This, Fixed(amount), DamagePermanentInto(result)), AddCountersTo(From(result), P1P1, Fixed(1)))
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(source.ID()).Counters[P1P1] != 0 {
			t.Fatalf("amount %d retained stale recipient", amount)
		}
	}
	victimCard := NewCreature("Victim", "", 2, 8)
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	target := DeclareDamageTarget("victim")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	g.RemoveFromBattlefield(victim)
	returned := g.PutOnBattlefield(victimCard, b.PlayerID())
	effect := Sequence(Snapshot(This, result), DealDamageFrom(This, target, Fixed(2), DamagePermanentInto(result)), AddCountersTo(From(result), P1P1, Fixed(1)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Damage != 0 || g.FindPermanent(source.ID()).Counters[P1P1] != 0 {
		t.Fatal("old recipient binding followed returned card")
	}
}

func TestDealDamageFromPermanentResultAbsentAfterPhaseOrPlayerRedirect(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
		target := DeclareDamageTarget("victim")
		g.resolution.targetSpecs = []Target{target.targetDeclaration}
		g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
		if phase {
			g.PhaseOut(victim.ID())
		} else {
			g.effects.AddReplacement(&creatureDamageRedirectReplacement{creatureID: victim.ID(), targetPlayerID: b.PlayerID()})
		}
		result := BindPermanent("damaged permanent")
		effect := Sequence(Snapshot(This, result), DealDamageFrom(This, target, Fixed(2), DamagePermanentInto(result)), AddCountersTo(From(result), P1P1, Fixed(1)))
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(source.ID()).Counters[P1P1] != 0 {
			t.Fatalf("phase=%t retained stale permanent result", phase)
		}
		if !phase && b.Life() != 18 {
			t.Fatal("redirection to player failed")
		}
	}
}

func TestDealDamageFromPermanentResultDeclaration(t *testing.T) {
	result := BindPermanent("damaged permanent")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(result)), AddCountersTo(From(result), P1P1, Fixed(1))))),
			WantErr: false,
		},
		{
			Name:    "nil permanent binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(nil)))),
			WantErr: true,
		},
		{
			Name:    "empty name permanent binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(BindPermanent(""))))),
			WantErr: true,
		},
		{
			Name:    "duplicate permanent binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamagePermanentInto(result), DamagePermanentInto(result)))),
			WantErr: true,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Missing", "", Spell(AddCountersTo(From(result), P1P1, Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_damage_permanent_source_test.go ---
func TestDamageSourceOfTargetKeepsDepartedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	oldCard := NewCreature("Old source", "", 2, 2, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(oldCard, a.PlayerID())
	target := DeclarePermanentTarget("source", IsCreature)
	recipient := BindPlayer("recipient")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{recipient: b.PlayerID()}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		target.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(oldCard, b.PlayerID())
	g.zones.ClearLKI()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		ctx.Game = game
		if err := DealDamageFrom(DamageSourceOf(target), FromPlayer(recipient), Fixed(2)).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 22 {
			t.Fatalf("old source lifelink and controller were lost: %d", game.GetPlayer(a.PlayerID()).Life())
		}
	}
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone changed the original result")
	}
}

func TestDamageSourceOfAbsentPhasedAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	ref := DeclarePermanentTarget("source", IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		ref.targetDeclaration: {{id: p.ID(), incarnation: p.incarnationID}},
	}}
	g.PhaseOut(p.ID())
	if err := DealDamageFrom(DamageSourceOf(ref), You, Fixed(2)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("phased source dealt damage")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if err := DealDamageFrom(DamageSourceOf(ref), You, Fixed(2)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 {
		t.Fatal("active source did not deal damage")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(DealDamageFrom(DamageSourceOf(ref), You, Fixed(1)), WithTargets(ref))),
			WantErr: false,
		},
		{
			Name:          "missing target declaration",
			Card:          NewInstant("Missing", "", Spell(DealDamageFrom(DamageSourceOf(ref), You, Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "target binding requires",
		},
		{
			Name:    "nil damage source permanent",
			Card:    NewInstant("Nil", "", Spell(DealDamageFrom(DamageSourceOf(nil), You, Fixed(1)))),
			WantErr: true,
		},
	})
}

func TestDamageSourceOfSpellTargetAndSavedSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Borrowed source", "", 2, 3, WithKeyword(Lifelink)), b.PlayerID())
	target := DeclarePermanentTarget("source", IsCreature)
	spell := NewInstant("Borrowed damage", "", Spell(DealDamageFrom(DamageSourceOf(target), You, Fixed(2)), WithTargets(target)))
	spell.SetOwner(a.PlayerID())
	ability := spell.Abilities()[0].(*ActionDefinition)
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	g.ResolveStackObject(obj)
	if a.Life() != 18 || b.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("spell used its own controller instead of the referenced source")
	}
	clone.ResolveStackObject(clone.StackPeek())
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost target source identity")
	}

	saved := BindPermanent("saved source")
	value := capturedPermanent{id: source.ID(), incarnation: source.incarnationID}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: value}
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(source.Card, a.PlayerID())
	if err := DealDamageFrom(DamageSourceOf(From(saved)), You, Fixed(1)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 17 || b.Life() != 23 || returned.ControllerID() != a.PlayerID() {
		t.Fatal("saved damage source followed returned card")
	}
}

// --- from composable_damage_player_recipient_test.go ---
func TestDealsDamageCapturesPlayerRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(player),
		GainLifeFor(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("capture or clone changed recipient")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatalf("life = %d", b.Life())
	}
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("permanent recipient supplied a player")
	}
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(b, 1, g.CaptureDamageSource(source.ID())),
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, g.CaptureDamageSource(source.ID())),
	)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("batch triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("batch lost player recipient or used permanent as player")
	}
	g.PhaseOut(source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
}

func TestDealsDamagePlayerRecipientDeclaration(t *testing.T) {
	bound := BindPlayer("recipient")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil recipient player",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty name recipient player",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate recipient player",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(bound).WithRecipientPlayer(bound), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "incompatible with DealsDamageToPlayer",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, bound).WithRecipientPlayer(bound), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:          "unbound recipient player",
			Card:          NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(DealsDamage(This, BindNumber("amount")), GainLifeFor(FromPlayer(bound), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDealsDamagePlayerRecipientFollowsFinalAction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("recipient")
	source := NewCreature("Source", "", 2, 2, WithAbility(WhenEvent(
		DealsDamage(This, BindNumber("amount")).WithRecipientPlayer(player),
		GainLifeFor(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("redirected permanent damage retained a player")
	}
}

// --- from composable_damage_player_result_test.go ---
func TestDealDamageFromCapturesActualPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	player := BindPlayer("damaged player")
	amount := BindNumber("damage dealt")
	effect := Sequence(DealDamageFrom(This, You, Fixed(3), DamageAmountInto(amount), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), FromNumber(amount)))
	for _, game := range []*Game{g, g.Clone()} {
		if err := ApplyEffect(game, effect, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
	}
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
}

func TestDealDamageFromPlayerResultAbsentAfterRedirectOrPrevention(t *testing.T) {
	for _, redirected := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
		if redirected {
			victim := g.PutOnBattlefield(NewCreature("Redirect", "", 2, 5), b.PlayerID())
			g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
		} else if err := ApplyEffect(g, PreventNextDamage(You, Fixed(3)), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		player := BindPlayer("damaged player")
		effect := Sequence(SnapshotPlayer(You, player), DealDamageFrom(This, You, Fixed(3), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), Fixed(4)))
		if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if b.Life() != 20 {
			t.Fatalf("redirected=%t: stale player result changed life to %d", redirected, b.Life())
		}
	}
}

func TestDealDamageFromPlayerResultRedirectedFromPermanent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
	g.effects.AddReplacement(&creatureDamageRedirectReplacement{creatureID: victim.ID(), targetPlayerID: b.PlayerID()})
	player := BindPlayer("damaged player")
	amount := BindNumber("damage dealt")
	target := DeclareDamageTarget("victim")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	effect := Sequence(DealDamageFrom(This, target, Fixed(3), DamageAmountInto(amount), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), FromNumber(amount)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 0 || b.Life() != 20 {
		t.Fatalf("victim damage = %d, player life = %d", victim.Damage, b.Life())
	}
}

func TestDealDamageFromPlayerResultDeclaration(t *testing.T) {
	player := BindPlayer("damaged player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(player)), GainLifeFor(FromPlayer(player), Fixed(1))))),
			WantErr: false,
		},
		{
			Name:    "nil player binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(nil)))),
			WantErr: true,
		},
		{
			Name:    "empty name player binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(BindPlayer(""))))),
			WantErr: true,
		},
		{
			Name:    "duplicate player binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamagePlayerInto(player), DamagePlayerInto(player)))),
			WantErr: true,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Missing", "", Spell(GainLifeFor(FromPlayer(player), Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_damage_result_test.go ---
func TestDealDamageFromCapturesActualAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(2)), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	dealt := BindNumber("damage dealt")
	effect := Sequence(DealDamageFrom(This, You, Fixed(3), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))
	for _, game := range []*Game{g, g.Clone()} {
		if err := ApplyEffect(game, effect, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
	}
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
}

func TestDealDamageFromResultDeclaration(t *testing.T) {
	dealt := BindNumber("dealt")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(2), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt))))),
			WantErr: false,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(GainLifeFor(You, FromNumber(dealt)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:    "nil amount binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamageAmountInto(nil)))),
			WantErr: true,
		},
		{
			Name:    "nil option",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), nil))),
			WantErr: true,
		},
		{
			Name:    "duplicate amount binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamageAmountInto(dealt), DamageAmountInto(dealt)))),
			WantErr: true,
		},
	})
}

func TestDealDamageFromResultZeroForZeroDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	dealt := BindNumber("dealt")
	effect := Sequence(
		DealDamageFrom(This, ActivePlayer, Fixed(2), DamageAmountInto(dealt)),
		DealDamageFrom(This, ActivePlayer, Fixed(0), DamageAmountInto(dealt)),
		GainLifeFor(You, FromNumber(dealt)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("zero damage kept a prior result")
	}
}

func TestDealDamageFromResultAbsentTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victimCard := NewCreature("Victim", "", 2, 5)
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	target := DeclareDamageTarget("recipient")
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	g.RemoveFromBattlefield(victim)
	returned := g.PutOnBattlefield(victimCard, b.PlayerID())
	dealt := BindNumber("dealt")
	effect := Sequence(DealDamageFrom(This, target, Fixed(2), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Damage != 0 || a.Life() != 20 {
		t.Fatal("returned target received damage or stale damage result")
	}
}

func TestDealDamageFromResultAfterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	redirect := g.PutOnBattlefield(NewCreature("Redirect", "", 2, 5), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: redirect.ID()})
	dealt := BindNumber("dealt")
	effect := Sequence(DealDamageFrom(This, You, Fixed(2), DamageAmountInto(dealt)), GainLifeFor(You, FromNumber(dealt)))
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if redirect.Damage != 2 || b.Life() != 22 || a.Life() != 22 {
		t.Fatalf("redirect damage = %d; life = %d, %d", redirect.Damage, a.Life(), b.Life())
	}
}

// --- from composable_damage_results_test.go ---
func TestDamageResultsIterateFinalRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.AddPreventionShield(a.PlayerID(), 2)
	results, item := BindDamageResults("results"), BindDamageResult("item")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2), DamageResultsInto(results)),
		ForEachDamageResult(results, item, Sequence(
			GainLifeFor(You, DamageResultAmount(item)),
			TapSubject(DamageResultPermanent(item)),
		)),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 24 || b.Life() != 20 || victim.Damage != 4 || !victim.Tapped {
		t.Fatalf("life = %d, %d; victim damage = %d, tapped = %t", a.Life(), b.Life(), victim.Damage, victim.Tapped)
	}
}

func TestDamageResultsClearAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	results, item := BindDamageResults("results"), BindDamageResult("item")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(0), DamageResultsInto(results)),
		ForEachDamageResult(results, item, DealDamageFrom(DamageResultSource(item), You, Fixed(3))),
	)
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatalf("original = %d, %d; clone = %d, %d", a.Life(), b.Life(), clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
}

func TestDamageResultsKeepDepartedSourceAndPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	results, item := BindDamageResults("results"), BindDamageResult("item")
	effect := Sequence(
		DealDamageEach(This, DamageRecipients(OpponentsOf(You), WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(1), DamageResultsInto(results)),
		Sacrifice(This),
		ForEachDamageResult(results, item, DealDamageFrom(DamageResultSource(item), DamageResultPlayer(item), DamageResultAmount(item))),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || b.Life() != 18 || g.FindPermanent(source.ID()) != nil {
		t.Fatalf("life = %d, %d; source remains = %t", a.Life(), b.Life(), g.FindPermanent(source.ID()) != nil)
	}
}

func TestDamageResultsExcludeReturnedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Target", "", 2, 9)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("target", 1, 1, IsCreature)
	results := BindDamageResults("results")
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2), DamageResultsInto(results))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	outcomes := g.resolution.damageResultsBindings[results]
	if len(outcomes) != 2 || outcomes[0].playerID != a.PlayerID() || outcomes[1].playerID != b.PlayerID() || outcomes[0].permanent.id != uuid.Nil || returned.Damage != 0 {
		t.Fatalf("outcomes = %+v; returned damage = %d", outcomes, returned.Damage)
	}
}

func TestDamageResultsCloneActiveFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	results, item := BindDamageResults("results"), BindDamageResult("item")
	outcome := damageOutcome{amount: 1, playerID: b.PlayerID(), source: g.CaptureDamageSource(source.ID())}
	g.resolution.damageResultsBindings = map[*DamageResultsBinding][]damageOutcome{results: {outcome}}
	g.resolution.damageResultBindings = map[*DamageResultBinding]damageOutcome{item: outcome}
	clone := g.Clone()
	clone.resolution.damageResultsBindings[results][0].amount = 9
	copyItem := clone.resolution.damageResultBindings[item]
	copyItem.amount = 7
	clone.resolution.damageResultBindings[item] = copyItem
	if g.resolution.damageResultsBindings[results][0].amount != 1 || g.resolution.damageResultBindings[item].amount != 1 {
		t.Fatal("clone changed original damage result frame")
	}
}

func TestDamageResultsDeclaration(t *testing.T) {
	results, item := BindDamageResults("results"), BindDamageResult("item")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(Sequence(
				DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)),
				ForEachDamageResult(results, item, GainLifeFor(You, DamageResultAmount(item))),
			))),
			WantErr: false,
		},
		{
			Name:          "nil damage results binding",
			Card:          NewInstant("Invalid", "", Spell(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(nil)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "duplicate damage results binding",
			Card:          NewInstant("Invalid", "", Spell(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results), DamageResultsInto(results)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "missing producer for results",
			Card:          NewInstant("Invalid", "", Spell(ForEachDamageResult(results, item, GainLifeFor(You, DamageResultAmount(item))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "item used outside loop",
			Card:          NewInstant("Invalid", "", Spell(Sequence(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)), GainLifeFor(You, DamageResultAmount(item))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "nil item binding",
			Card:          NewInstant("Invalid", "", Spell(Sequence(DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(1), DamageResultsInto(results)), ForEachDamageResult(results, nil, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
	})
}

// --- from composable_damage_source_controller_test.go ---
func TestDamageSourceControllerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	controller := BindPlayer("source controller")
	amount := BindNumber("amount")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, player).ToController().WithSourceController(controller).WithAmount(amount),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(a, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 2, host.ID())
	g.DealDamageToPlayer(a, 3, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 16 || game.GetPlayer(b.PlayerID()).Life() != 26 {
			t.Fatalf("life = %d/%d, want 16/26", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestDamageSourceControllerThisAndValidation(t *testing.T) {
	player := BindPlayer("damaged")
	controller := BindPlayer("source controller")
	valid := NewCreature("Valid", "", 1, 4, WithAbility(WhenEvent(
		DealsDamageToPlayer(This, player).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	valid.SetOwner(a.PlayerID())
	g.PutOnBattlefield(valid, a.PlayerID())
	g.DealDamageToPlayer(b, 1, valid.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatalf("life = %d/%d, want 21/19", a.Life(), b.Life())
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    valid,
			WantErr: false,
		},
		{
			Name:          "nil controller binding",
			Card:          NewCreature("Nil", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(nil), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "controller binding",
		},
		{
			Name:          "unnamed controller binding",
			Card:          NewCreature("Unnamed", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(BindPlayer("")), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate controller binding",
			Card:          NewCreature("Duplicate", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(controller).WithSourceController(controller), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "controller binding",
		},
		{
			Name:          "same binding",
			Card:          NewCreature("Same", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(player), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "wrong pattern",
			Card:          NewCreature("Wrong", "", 1, 1, WithAbility(WhenEvent(DealsDamageToOpponent(This, player).WithSourceController(controller), DrawFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "DealsDamageToPlayer",
		},
		{
			Name:          "unbound",
			Card:          NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player), GainLifeFor(FromPlayer(controller), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "cross ability",
			Card: NewCreature("Cross", "", 1, 1,
				WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSourceController(controller), DrawFor(You, Fixed(1)))),
				WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(controller), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_damage_source_result_test.go ---
func TestDamageSourceResultKeepsDepartedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Old Source", "", 1, 2, WithKeyword(Lifelink))
	old := g.PutOnBattlefield(card, a.PlayerID())
	source := BindDamageSource("damage source")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(source)),
		DealDamageFrom(FromDamageSource(source), You, Fixed(2)),
	)
	g.resolution.sourceCaptured = true
	g.resolution.sourceIncarnation = old.incarnationID
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	for _, game := range []*Game{g, g.Clone()} {
		if err := ApplyEffect(game, effect, card.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 17 {
			t.Fatalf("old source controller or lifelink lost: A=%d B=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
	if got := g.FindPermanent(card.ID()); got == nil || got.incarnationID != returned.incarnationID {
		t.Fatal("saved source changed returned incarnation")
	}
}

func TestDamageSourceResultClearsOnPreventedDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	binding := BindDamageSource("source")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)),
		PreventNextDamage(You, Fixed(1)),
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)),
		DealDamageFrom(FromDamageSource(binding), You, Fixed(2)),
	)
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 19 {
		t.Fatal("prevented damage retained a source")
	}
}

func TestDamageSourceResultCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	binding := BindDamageSource("source")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: damageSourceFromPermanent(p)}
	clone := g.Clone()
	saved := clone.resolution.damageSourceBindings[binding]
	saved.colors = append(saved.colors, Red)
	saved.keywords[Lifelink] = false
	clone.resolution.damageSourceBindings[binding] = saved
	if !g.resolution.damageSourceBindings[binding].HasKeyword(Lifelink) || g.resolution.damageSourceBindings[binding].HasColor(Red) {
		t.Fatal("clone changed original saved source")
	}
}

func TestDamageSourceResultSpellAfterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Two Bolts", "", nil)
	card.SetOwner(a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 5), b.PlayerID())
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	end := g.resolution.Begin(&StackObject{Card: card, Controller: a.PlayerID(), SourceID: card.ID()})
	defer end()
	binding := BindDamageSource("spell")
	effect := Sequence(
		DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)),
		DealDamageFrom(FromDamageSource(binding), You, Fixed(1)),
	)
	if err := ApplyEffect(g, effect, card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if victim.Damage != 2 || b.Life() != 20 || !g.HasDealtDamageToPermanent(card.ID(), victim.ID()) {
		t.Fatal("redirected spell damage lost source identity")
	}
}

func TestDamageSourceResultDeclaration(t *testing.T) {
	binding := BindDamageSource("source")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding)), DealDamageFrom(FromDamageSource(binding), You, Fixed(1))))),
			WantErr: false,
		},
		{
			Name:    "nil damage source binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamageSourceInto(nil)))),
			WantErr: true,
		},
		{
			Name:    "empty name damage source binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamageSourceInto(BindDamageSource(""))))),
			WantErr: true,
		},
		{
			Name:    "duplicate damage source binding",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, You, Fixed(1), DamageSourceInto(binding), DamageSourceInto(binding)))),
			WantErr: true,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Missing", "", Spell(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_damage_to_controller_test.go ---
func TestDamageToControllerEventFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p, n := BindPlayer("damaged"), BindNumber("amount")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, p).ToController().WithAmount(n),
		GainLifeFor(FromPlayer(p), FromNumber(n)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
		t.Fatal(err)
	}
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPlayer(b, 2, host.ID())
	g.AddPreventionShield(a.PlayerID(), 2)
	g.DealDamageToPlayer(a, 2, host.ID())
	g.DealDamageToPlayer(a, 3, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, b.PlayerID())
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatalf("life = %d/%d, want 20/18", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestDamageToControllerEventCurrentControllerAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	aura := NewAura("Watcher", "", WithAbility(WhenEvent(
		DealsDamageToPlayer(AttachedPermanent, p).ToController(),
		GainLifeFor(FromPlayer(p), Fixed(1)))))
	host := NewCreature("Host", "", 2, 8)
	for _, card := range []Card{aura, host} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.Attach(aura.ID(), host.ID())
	g.PhaseOut(aura.ID())
	g.DealDamageToPlayer(a, 1, host.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.DealDamageToPlayer(a, 1, host.ID())
	g.DealDamageToPlayer(b, 1, host.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	for _, game := range []*Game{g, g.Clone()} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatalf("life = %d/%d, want 18/21", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestDamageToControllerEventDeclaration(t *testing.T) {
	p := BindPlayer("damaged")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewAura("Valid", "", WithAbility(WhenEvent(
				DealsDamageToPlayer(AttachedPermanent, p).ToController(), DrawFor(FromPlayer(p), Fixed(1))))),
			WantErr: false,
		},
		{
			Name: "invalid source",
			Card: NewAura("Invalid", "", WithAbility(WhenEvent(
				DealsDamageToPlayer(From(BindPermanent("other")), p).ToController(), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "incompatible opponent filter",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(DealsDamageToOpponent(This, p).ToController(), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "incompatible combat damage filter",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(DealsCombatDamageToPlayer(This, p).ToController(), DrawFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestDamageToControllerEventSourceThis(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	source := NewCreature("Watcher", "", 1, 4, WithAbility(WhenEvent(
		DealsDamageToPlayer(This, p).ToController(), GainLifeFor(FromPlayer(p), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(b, 1, source.ID())
	g.DealDamageToPlayer(a, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatalf("life = %d/%d, want 21/19", a.Life(), b.Life())
	}
}

// --- from composable_damage_turn_set_test.go ---
func TestTurnDamagePermanentSetIdentityAndOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 5)
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	recipientCard := NewCreature("Recipient", "", 1, 5)
	recipient := g.PutOnBattlefield(recipientCard, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsDealtDamageByThisTurn(This, MatchingPermanents(IsCreature))
	check := func(want *Permanent) {
		t.Helper()
		got := set.resolvePermanents(ctx)
		if want == nil && len(got) != 0 || want != nil && (len(got) != 1 || got[0].ID() != want.ID() || got[0].incarnationID != want.incarnationID) {
			t.Fatalf("selected permanents = %v, want %v", got, want)
		}
	}
	check(nil)
	g.AddPreventionShield(recipient.ID(), 1)
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(source.ID()))
	check(nil)
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(source.ID()))
	check(recipient)
	g.PhaseOut(recipient.ID())
	check(nil)
	g.PhaseIn([]uuid.UUID{recipient.ID()})
	check(recipient)
	clone := g.Clone()
	clone.damage.ClearEndOfTurn()
	if got := set.resolvePermanents(&EffectContext{Game: clone, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}); len(got) != 0 {
		t.Fatal("clone retained cleared turn history")
	}
	check(recipient)
	g.RemoveFromBattlefield(recipient)
	g.PutOnBattlefield(recipientCard, b.PlayerID())
	check(nil)
	g.RemoveFromBattlefield(source)
	newSource := g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.DealDamageToPermanentFromSource(other, 1, g.CaptureDamageSource(newSource.ID()))
	check(nil)
	ctx.sourceIncarnation = newSource.incarnationID
	got := set.resolvePermanents(ctx)
	if len(got) != 1 || got[0].ID() != other.ID() || got[0].incarnationID != other.incarnationID {
		t.Fatalf("new source history = %v", got)
	}
	g.damage.ClearEndOfTurn()
	check(nil)
}

func TestTurnDamagePermanentSetDefinition(t *testing.T) {
	set := PermanentsDealtDamageByThisTurn(This, MatchingPermanents(IsCreature))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Good", "", 1, 5, WithAbility(WhenEvent(Enters(This), TapEach(set)))),
			WantErr: false,
			Inspect: func(t *testing.T) {
				if InspectDefinition(set).Role != "permanent-set" {
					t.Fatal("missing permanent-set inspection role")
				}
			},
		},
		{
			Name:    "nil source",
			Card:    NewInstant("Bad", "", Spell(TapEach(PermanentsDealtDamageByThisTurn(nil, MatchingPermanents(IsCreature))))),
			WantErr: true,
		},
		{
			Name:    "nil target filter",
			Card:    NewInstant("Bad", "", Spell(TapEach(PermanentsDealtDamageByThisTurn(This, nil)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("Bad", "", Spell(TapEach(PermanentsDealtDamageByThisTurn(DamageSourceOf(DeclarePermanentTarget("source", IsCreature)), MatchingPermanents(IsCreature))))),
			WantErr: true,
		},
	})
}

// --- from composable_destroy_graveyard_result_test.go ---
func TestDestroyEachGraveyardResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("mountains put into graveyard")
	put := func(name string) *Permanent {
		card := NewLand(name)
		card.SetOwner(b.PlayerID())
		return g.PutOnBattlefield(card, b.PlayerID())
	}
	first := put("First")
	second := put("Second")
	regenerated := put("Regenerated")
	g.AddRegenerationShield(regenerated.ID())
	indestructible := put("Indestructible")
	indestructible.GrantBaseAttr(Indestructible)
	phased := put("Phased")
	phased.PhasedOut = true
	set := MatchingPermanents(IsLand)
	body := Sequence(DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count)), GainLifeFor(You, FromNumber(count)))
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 2 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone did not count exactly two graveyard moves")
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || a.Life() != 20 {
		t.Fatal("clone changed the original game")
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 2 || a.Life() != 22 {
		t.Fatal("result did not count exactly two graveyard moves")
	}
	for _, p := range []*Permanent{regenerated, indestructible, phased} {
		if g.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatalf("%s did not survive", p.Name())
		}
	}
	if err := DestroyEach(MatchingPermanents(IsCreature), PermanentsPutIntoGraveyardThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 0 {
		t.Fatal("empty instruction retained a stale result")
	}
}

func TestDestroyEachGraveyardResultDefinition(t *testing.T) {
	count := BindNumber("graveyard count")
	set := MatchingPermanents(IsLand)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewSorcery("Valid", "", WithAbility(Spell(Sequence(
				DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count)),
				DealDamageFrom(This, You, FromNumber(count)),
			)))),
			WantErr: false,
		},
		{
			Name:          "nil graveyard result binding",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name graveyard result binding",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate graveyard result binding",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count), PermanentsPutIntoGraveyardThisWayInto(count))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(Sequence(DealDamageFrom(This, You, FromNumber(count)), DestroyEach(set, PermanentsPutIntoGraveyardThisWayInto(count)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDestroyEachGraveyardResultXTargetsAndReplacement(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXPermanentTargets("mountains", And(IsLand, HasSubType("Mountain")))
	count := BindNumber("mountains put into graveyard")
	spell := NewSorcery("Synthetic eruption", "{X}{U}{U}{U}", WithAbility(Spell(Sequence(
		DestroyEach(targets, PermanentsPutIntoGraveyardThisWayInto(count)),
		DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), FromNumber(count)),
	), WithTargets(targets))))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewLand("First", WithSubTypes("Mountain")), b.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second", WithSubTypes("Mountain")), b.PlayerID())
	plain := g.PutOnBattlefield(NewLand("Plain"), b.PlayerID())
	g.effects.AddReplacement(&exileInsteadOfGraveyardReplacement{cardID: second.ID()})
	g.SetStep(PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), plain.ID()}, 2); err == nil {
		t.Fatal("accepted a non-Mountain target")
	}
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("invalid targets changed game state")
	}
	a.ManaPool().Add(Blue, 3)
	a.ManaPool().Add(Colorless, 2)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatalf("life = %d/%d; want 19/19", a.Life(), b.Life())
	}
}

func TestDestroyEachGraveyardResultWithoutRegenerationAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Shielded"), b.PlayerID())
	g.AddRegenerationShield(land.ID())
	token := g.PutOnBattlefield(NewToken("Token", 1, 1, []CardType{TypeCreature}, nil), b.PlayerID())
	count := BindNumber("graveyard moves")
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(Or(IsLand, IsCreature)), PermanentsPutIntoGraveyardThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 2 || g.FindPermanent(land.ID()) != nil || g.FindPermanent(token.ID()) != nil {
		t.Fatal("result did not count the shielded land and token")
	}
}

// --- from composable_destroy_result_test.go ---
func TestDestroyEachCreatureDeathsResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("creatures died this way")
	makeCreature := func(name string) *Permanent {
		card := NewCreature(name, "{1}", 2, 2)
		card.SetOwner(b.PlayerID())
		return g.PutOnBattlefield(card, b.PlayerID())
	}
	first := makeCreature("First")
	second := makeCreature("Second")
	regenerated := makeCreature("Regenerated")
	g.AddRegenerationShield(regenerated.ID())
	indestructible := makeCreature("Indestructible")
	indestructible.GrantBaseAttr(Indestructible)
	phased := makeCreature("Phased")
	phased.PhasedOut = true
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	body := Sequence(
		DestroyEach(MatchingPermanents(Or(IsCreature, IsArtifact)), CreaturesDiedThisWayInto(count)),
		GainLifeFor(You, FromNumber(count)),
	)
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 2 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatalf("clone result = %d, life = %d", clone.resolution.numberBindings[count], clone.GetPlayer(a.PlayerID()).Life())
	}
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || a.Life() != 20 {
		t.Fatal("clone changed the original game")
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 2 || a.Life() != 22 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("result did not count the two destroyed creatures")
	}
	for _, p := range []*Permanent{regenerated, indestructible, phased} {
		if g.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatalf("%s should remain", p.Name())
		}
	}
	if err := DestroyEach(MatchingPermanents(IsLand), CreaturesDiedThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 0 || a.Life() != 22 {
		t.Fatal("second instruction retained a stale result")
	}
}

func TestDestroyEachCreatureDeathsResultDefinition(t *testing.T) {
	count := BindNumber("creatures died")
	set := MatchingPermanents(IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewSorcery("Valid", "", WithAbility(Spell(Sequence(
				DestroyEach(set, CreaturesDiedThisWayInto(count)),
				GainLifeFor(You, FromNumber(count)),
			)))),
			WantErr: false,
		},
		{
			Name:          "nil",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DestroyEach(set, CreaturesDiedThisWayInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DestroyEach(set, CreaturesDiedThisWayInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DestroyEach(set, CreaturesDiedThisWayInto(count), CreaturesDiedThisWayInto(count))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(GainLifeFor(You, FromNumber(count))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "late",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(Sequence(GainLifeFor(You, FromNumber(count)), DestroyEach(set, CreaturesDiedThisWayInto(count)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDestroyEachCreatureDeathsResultWithoutRegeneration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("deaths")
	card := NewCreature("Shielded", "", 2, 2)
	card.SetOwner(b.PlayerID())
	shielded := g.PutOnBattlefield(card, b.PlayerID())
	g.AddRegenerationShield(shielded.ID())
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature), CreaturesDiedThisWayInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if got := g.resolution.numberBindings[count]; got != 1 || g.FindPermanent(shielded.ID()) != nil {
		t.Fatalf("result = %d; shielded permanent still present = %t", got, g.FindPermanent(shielded.ID()) != nil)
	}
}

func TestDestroyEachCreatureDeathsResultSpellDamage(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("nonblack creatures died")
	spell := NewSorcery("Death Count", "", WithAbility(Spell(Sequence(
		DestroyEach(MatchingPermanents(And(IsCreature, Not(HasColorFilter(Black)))), CreaturesDiedThisWayInto(count)),
		DealDamageFrom(This, You, FromNumber(count)),
	))))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"White", "Black"} {
		cost := "{W}"
		if name == "Black" {
			cost = "{B}"
		}
		card := NewCreature(name, cost, 2, 2)
		card.SetOwner(b.PlayerID())
		g.PutOnBattlefield(card, b.PlayerID())
	}
	a.AddToHand(spell)
	g.SetStep(PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatalf("life totals = %d/%d; want 19/20", a.Life(), b.Life())
	}
	if len(g.FilterBattlefield(HasColorFilter(Black))) != 1 {
		t.Fatal("black creature should remain")
	}
}

// --- from composable_destroy_singular_result_test.go ---
func TestDestroySingularResult(t *testing.T) {
	for _, state := range []string{"destroyed", "regenerated", "indestructible", "phased", "absent", "returned"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewCreature("Subject", "", 2, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			result := BindNumber("destroyed")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			g.resolution.numberBindings = map[*NumberBinding]int{result: 1}
			switch state {
			case "regenerated":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				p.GrantBaseAttr(Indestructible)
			case "phased":
				g.PhaseOut(p.ID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			}
			body := Sequence(DestroyWithResult(This, DestroyedThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(3))))
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "destroyed" {
				want = 1
			}
			if got := g.resolution.numberBindings[result]; got != want || a.Life() != 20+3*want {
				t.Fatalf("result = %d, life = %d; want %d, %d", got, a.Life(), want, 20+3*want)
			}
		})
	}
}

func TestDestroySingularResultDefinition(t *testing.T) {
	result := BindNumber("destroyed")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewCreature("Valid", "", 2, 2, WithAbility(At(YourUpkeep, Sequence(DestroyWithResult(This, DestroyedThisWayInto(result)), IfPositive(FromNumber(result), DealDamageFrom(This, You, Fixed(7))))))),
			WantErr: false,
		},
		{
			Name:          "nil result binding",
			Card:          NewCreature("Invalid", "", 2, 2, WithAbility(At(YourUpkeep, DestroyWithResult(This, DestroyedThisWayInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name result binding",
			Card:          NewCreature("Invalid", "", 2, 2, WithAbility(At(YourUpkeep, DestroyWithResult(This, DestroyedThisWayInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate result binding",
			Card:          NewCreature("Invalid", "", 2, 2, WithAbility(At(YourUpkeep, newDestroySubjectEffect(This, false, []DestroyResultOption{DestroyedThisWayInto(result), DestroyedThisWayInto(result)})))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewCreature("Invalid", "", 2, 2, WithAbility(At(YourUpkeep, IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "late producer",
			Card:          NewCreature("Invalid", "", 2, 2, WithAbility(At(YourUpkeep, Sequence(IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1))), DestroyWithResult(This, DestroyedThisWayInto(result)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDestroySingularResultWithoutRegenerationAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Subject", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.AddRegenerationShield(p.ID())
	result := BindNumber("destroyed")
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		body := Sequence(DestroyWithoutRegenerationWithResult(This, DestroyedThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
		if err := body.Apply(&EffectContext{Game: game, SourceID: p.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if game.resolution.numberBindings[result] != 1 || game.GetPlayer(a.PlayerID()).Life() != 22 || game.FindPermanent(p.ID()) != nil {
			t.Fatal("action did not destroy through the regeneration shield")
		}
		if game == clone && (g.FindPermanent(p.ID()) == nil || a.Life() != 20) {
			t.Fatal("clone changed the original game")
		}
	}
}

func TestDestroySingularResultTargetLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("destroyed")
	target := DeclarePermanentTarget("subject", IsCreature)
	spell := NewInstant("Destroy Test", "", Spell(Sequence(
		DestroyWithResult(target, DestroyedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	a.AddToHand(spell)
	g.SetStep(PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.PutOnBattlefield(victim, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || g.FindPermanent(p.ID()) == nil {
		t.Fatal("an illegal target caused destruction or a stale result")
	}
}

func TestDestroySingularResultSourceDamageAfterDeparture(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Source", "", 7, 7)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindNumber("destroyed")
	body := Sequence(
		DestroyWithResult(This, DestroyedThisWayInto(result)),
		IfPositive(FromNumber(result), DealDamageFrom(This, You, Fixed(7))),
	)
	if err := body.Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 13 || g.FindPermanent(p.ID()) != nil {
		t.Fatalf("life = %d; source still present = %t", a.Life(), g.FindPermanent(p.ID()) != nil)
	}
}

// --- from composable_discard_amount_test.go ---
func TestTypedDiscardAmountContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToHand(NewArtifact("A card", ""))
		b.AddToHand(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "{2}", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	saved := BindPermanent("amount")
	body := Sequence(Snapshot(This, saved), Discard(You, PowerOf(From(saved))))
	card := NewCreature("Discard", "", 1, 1, WithAbility(Activate(Costs(), body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 || len(a.Hand()) != 3 {
		t.Fatal("saved amount, recipient, or clone isolation changed")
	}
	if err := ApplyEffect(g, Discard(You, ValueAtResolution(TargetPermanentManaValue())), uuid.Nil, b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 1 || len(a.Hand()) != 3 {
		t.Fatal("amount lost outer targets")
	}
}

type discardChoicePlayer struct {
	*BasePlayer
	requests []int
}

func (p *discardChoicePlayer) ChooseMayAbility(string) bool { return true }
func (p *discardChoicePlayer) ChooseCardsFromHand(n int, _ string, _ GameReader) []Card {
	p.requests = append(p.requests, n)
	hand := p.Hand()
	n = min(n, len(hand))
	return append([]Card(nil), hand[len(hand)-n:]...)
}

func TestTypedDiscardAmountChoiceAndReplacement(t *testing.T) {
	a := NewBasePlayer("A")
	b := &discardChoicePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	for range 3 {
		b.AddToHand(NewArtifact("Card", ""))
	}
	chosen := b.Hand()[2].ID()
	events := 0
	watcher := NewTriggered(EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() || evt.SourceID != chosen {
			t.Error("wrong discard event")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	if err := ApplyEffect(g, Discard(ActivePlayer, Fixed(0)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SetActivePlayerIndex(1)
	for _, n := range []int{-2, 0} {
		if err := ApplyEffect(g, Discard(ActivePlayer, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(b.requests) != 0 {
		t.Fatal("nonpositive amount requested a choice")
	}
	calls := 0
	amount := MinValue(Fixed(1), countedLifeAmount{&calls})
	if err := ApplyEffect(g, Discard(ControllerOf(This), amount), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("absent recipient evaluated amount")
	}
	if err := ApplyEffect(g, Discard(ActivePlayer, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(b.requests) != 1 || b.requests[0] != 1 || len(b.Hand()) != 2 || len(b.Library()) != 1 || b.Library()[0].ID() != chosen || len(b.Graveyard()) != 0 || events != 1 {
		t.Fatal("amount capture, affected-player choice, replacement, or event changed")
	}
}

func TestTypedDiscardAmountShortHandAndContinuation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Only card", ""))
	for range 2 {
		body := Sequence(Discard(You, SubtractValues(Fixed(5), Fixed(1))), GainLifeFor(You, Fixed(1)))
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 1 || a.Life() != 22 {
		t.Fatal("short or empty hand stopped resolution")
	}
}

func TestTypedDiscardAmountValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil amount",
			Card:    NewSorcery("Invalid discard", "", WithAbility(Spell(Discard(You, nil)))),
			WantErr: true,
		},
		{
			Name:    "nil subject in PowerOf",
			Card:    NewSorcery("Invalid discard", "", WithAbility(Spell(Discard(You, PowerOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "unbound permanent in PowerOf",
			Card:    NewSorcery("Invalid discard", "", WithAbility(Spell(Discard(You, PowerOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target in PowerOf",
			Card:    NewSorcery("Invalid discard", "", WithAbility(Spell(Discard(You, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))))),
			WantErr: true,
		},
		{
			Name:    "nil ValueAtResolution",
			Card:    NewSorcery("Invalid discard", "", WithAbility(Spell(Discard(You, ValueAtResolution(nil))))),
			WantErr: true,
		},
	})
	calls := 0
	info := InspectDefinition(Discard(You, countedLifeAmount{&calls}))
	if info.Properties.Outcome != OutcomeDetriment || calls != 0 {
		t.Fatal("inspection changed metadata or evaluated amount")
	}
}

func TestTypedDiscardAmountCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToHand(NewArtifact("A card", ""))
		b.AddToHand(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(Discard(You, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Discard", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if len(a.Hand()) != 3 || len(b.Hand()) != 1 {
		t.Fatal("copy lost controller or amount binding")
	}
	g.ResolveStackObject(original)
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("original lost its amount")
	}
}

// --- from composable_discard_card_result_test.go ---
func TestDiscardCardResultIdentityAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "")
	second := NewArtifact("Second", "")
	b.AddToHand(first)
	b.AddToHand(second)
	binding := BindCard("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(ActivePlayer, Fixed(1), DiscardCardInto(binding))
	g.SetActivePlayerIndex(1)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	got, ok := FromCard(binding).resolveCard(ctx)
	if !ok || got.ID() != first.ID() {
		t.Fatal("wrong discarded card")
	}
	clone := g.Clone()
	b.RemoveFromGraveyard(first.ID())
	b.AddToHand(first)
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("binding followed returned card")
	}
	if _, ok := FromCard(binding).resolveCard(&EffectContext{Game: clone}); !ok {
		t.Fatal("clone lost discard result")
	}
	if err := Discard(FromPlayer(BindPlayer("absent")), Fixed(1), DiscardCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("absent player retained result")
	}
}

func TestDiscardCardResultReplacementAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToHand(NewArtifact("Library card", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	binding := BindCard("discarded")
	target := DeclarePlayerTarget("player")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(target, Fixed(1), DiscardCardInto(binding))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 1 {
		t.Fatal("absent target changed hand")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Library()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("replacement did not move card")
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("library replacement produced a graveyard card")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("empty hand reused card")
	}
}

func TestDiscardCardResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := NewArtifact("First", "")
	second := NewArtifact("Second", "")
	b.AddToHand(first)
	b.AddToHand(second)
	binding := BindCard("discarded")
	target := DeclarePlayerTarget("player")
	spell := NewInstant("Discard", "", WithAbility(Spell(Sequence(Discard(target, Fixed(1), DiscardCardInto(binding)), ReturnCardToHand(FromCard(binding))), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 || len(b.Hand()) != 2 {
		t.Fatal("copy result or clone isolation failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 2 {
		t.Fatal("original result did not resolve")
	}
}

func TestDiscardCardResultDeclaration(t *testing.T) {
	b := BindCard("discarded")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(Discard(You, Fixed(1), DiscardCardInto(b)), ReturnCardToHand(FromCard(b))))),
			WantErr: false,
		},
		{
			Name:          "nil card binding",
			Card:          NewInstant("Invalid", "", Spell(Discard(You, Fixed(1), DiscardCardInto(nil)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "one card limit",
			Card:          NewInstant("Invalid", "", Spell(Discard(You, Fixed(2), DiscardCardInto(b)))),
			WantErr:       true,
			WantErrSubstr: "one card",
		},
		{
			Name:          "duplicate card binding",
			Card:          NewInstant("Invalid", "", Spell(Discard(You, Fixed(1), DiscardCardInto(b), DiscardCardInto(b)))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(ReturnCardToHand(FromCard(b)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_discard_card_test.go ---
func TestDiscardCardSavedDrawAndReplacement(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	drawn, other := NewArtifact("Drawn", ""), NewArtifact("Other", "")
	a.AddToLibrary(drawn)
	a.AddToHand(other)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), DiscardCard(FromCard(binding))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != other.ID() || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != drawn.ID() {
		t.Fatalf("saved draw: hand=%s graveyard=%s want hand=%s graveyard=%s", a.Hand()[0].Name(), a.Graveyard()[0].Name(), other.Name(), drawn.Name())
	}
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 1 {
		t.Fatal("departed card was discarded again")
	}

	replaced := NewArtifact("Replaced", "")
	a.AddToLibrary(replaced)
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 || a.Library()[0].ID() != replaced.ID() || len(a.Hand()) != 1 || len(a.Graveyard()) != 1 {
		t.Fatal("saved discard bypassed replacement or event service")
	}
}

func TestDiscardCardVisitAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Drawn", "")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("missing card")
	}
	a.AddToHand(card)
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("old binding followed a new hand visit")
	}
	if err := DiscardCard(FromCard(binding)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("clone did not retain an independent binding")
	}
}

func TestDiscardCardOwnerAndZone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("B's card", "")
	b.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	target := DeclarePlayerTarget("player")
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := DrawFor(target, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != card.ID() {
		t.Fatal("discard used the ability controller instead of the card owner")
	}
	ctx.Game.resolution.cardBindings[binding] = capturedCard{id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: ZoneGraveyard}
	if err := DiscardCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Graveyard()) != 1 {
		t.Fatal("graveyard reference was treated as a hand card")
	}
}

func TestDiscardCardDefinition(t *testing.T) {
	binding := BindCard("drawn")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), DiscardCard(FromCard(binding))))),
			WantErr: false,
		},
		{
			Name:          "nil card",
			Card:          NewInstant("Invalid", "", Spell(DiscardCard(nil))),
			WantErr:       true,
			WantErrSubstr: "card",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(DiscardCard(FromCard(binding)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "conditional producer",
			Card:          NewInstant("Invalid", "", Spell(Sequence(IfState(Controls(IsCreature), DrawFor(You, Fixed(1), DrawCardInto(binding))), DiscardCard(FromCard(binding))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	info := InspectDefinition(DiscardCard(FromCard(binding)))
	if info.Role != "resolving-effect" || info.Properties.Outcome != OutcomeDetriment {
		t.Fatalf("inspection = %+v", info)
	}
}

// --- from composable_discard_cards_result_test.go ---
func TestDiscardCardsResultVisitsAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	b.AddToHand(first)
	b.AddToHand(second)
	result := BindCards("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(ActivePlayer, Fixed(3), DiscardCardsInto(result))
	g.SetActivePlayerIndex(1)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 2 {
		t.Fatalf("discarded visits = %d", got)
	}
	clone := g.Clone()
	b.RemoveFromGraveyard(first.ID())
	b.AddToHand(first)
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 1 {
		t.Fatalf("returned card counted: %d", got)
	}
	b.RemoveFromHand(first.ID())
	b.AddToGraveyard(first)
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 1 {
		t.Fatalf("new graveyard visit counted: %d", got)
	}
	if got := CountCards(FromCards(result)).resolveValue(&EffectContext{Game: clone}); got != 2 {
		t.Fatalf("clone visits = %d", got)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 0 {
		t.Fatalf("empty hand kept result: %d", got)
	}
}

func TestDiscardCardsResultReplacementAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToHand(NewArtifact("First", ""))
	b.AddToHand(NewArtifact("Second", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	result := BindCards("discarded")
	target := DeclarePlayerTarget("player")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := Discard(target, Fixed(2), DiscardCardsInto(result))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 2 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("absent target changed state")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Library()) != 2 || len(b.Graveyard()) != 0 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("redirected discards entered result")
	}
}

func TestDiscardCardsResultPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	b.AddToHand(NewArtifact("Held", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	g.SetActivePlayerIndex(1)
	result := BindCards("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := Discard(ActivePlayer, Fixed(2), DiscardCardsInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 1 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("prevented discard changed result")
	}
}

func TestDiscardCardsResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	for range 4 {
		b.AddToHand(NewArtifact("Held", ""))
	}
	result := BindCards("discarded")
	target := DeclarePlayerTarget("player")
	spell := NewInstant("Discard", "", WithAbility(Spell(Sequence(
		Discard(target, Fixed(2), DiscardCardsInto(result)),
		GainLifeFor(You, CountCards(FromCards(result))),
	), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 {
		t.Fatal("copy result or clone isolation failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || len(b.Hand()) != 2 {
		t.Fatal("original result changed")
	}
}

func TestDiscardCardsResultDeclaration(t *testing.T) {
	b := BindCards("discarded")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(Discard(You, Fixed(2), DiscardCardsInto(b)), GainLifeFor(You, CountCards(FromCards(b)))))),
			WantErr: false,
		},
		{
			Name:          "nil cards binding",
			Card:          NewInstant("Invalid", "", Spell(Discard(You, Fixed(2), DiscardCardsInto(nil)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name cards binding",
			Card:          NewInstant("Invalid", "", Spell(Discard(You, Fixed(2), DiscardCardsInto(BindCards(""))))),
			WantErr:       true,
			WantErrSubstr: "binding name",
		},
		{
			Name:          "duplicate cards binding",
			Card:          NewInstant("Invalid", "", Spell(Discard(You, Fixed(2), DiscardCardsInto(b), DiscardCardsInto(b)))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(GainLifeFor(You, CountCards(FromCards(b))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "conditional producer",
			Card:          NewInstant("Invalid", "", Spell(Sequence(IfState(Controls(IsCreature), Discard(You, Fixed(2), DiscardCardsInto(b))), GainLifeFor(You, CountCards(FromCards(b)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_discard_each_result_test.go ---
func TestDiscardEachTotalActualAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A1", ""))
	a.AddToHand(NewArtifact("A2", ""))
	b.AddToHand(NewArtifact("B1", ""))
	count := BindNumber("discarded")
	body := Sequence(DiscardEach(EachPlayer, DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 23 {
		t.Fatalf("life = %d", got)
	}
	if len(a.Hand()) != 2 || len(b.Hand()) != 1 || a.Life() != 20 {
		t.Fatal("clone changed original")
	}
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatalf("life = %d", a.Life())
	}
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 {
		t.Fatal("empty hands retained stale count")
	}
}

func TestDiscardEachTotalReplacementAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A", ""))
	b.AddToHand(NewArtifact("B", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	count := BindNumber("discarded")
	body := Sequence(DiscardEach(EachPlayer, DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatalf("redirected discard count: life = %d", a.Life())
	}
	a.AddToHand(NewArtifact("A2", ""))
	b.AddToHand(NewArtifact("B2", ""))
	g.effects.AddReplacement(&discardToHandReplacement{})
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("prevented discard counted")
	}
	absent := Sequence(DiscardEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))
	if err := ApplyEffect(g, absent, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("absent set retained stale count")
	}
}

func TestDiscardEachTotalDeclaration(t *testing.T) {
	count := BindNumber("discarded")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewSorcery("Valid", "", WithAbility(Spell(Sequence(DiscardEach(EachPlayer, DiscardTotalInto(count)), GainLifeFor(You, FromNumber(count)))))),
			WantErr: false,
		},
		{
			Name:    "nil total binding",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEach(EachPlayer, DiscardTotalInto(nil))))),
			WantErr: true,
		},
		{
			Name:    "empty name total binding",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEach(EachPlayer, DiscardTotalInto(BindNumber("")))))),
			WantErr: true,
		},
		{
			Name:    "duplicate total binding",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEach(EachPlayer, DiscardTotalInto(count), DiscardTotalInto(count))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(Sequence(GainLifeFor(You, FromNumber(count)), DiscardEach(EachPlayer, DiscardTotalInto(count)))))),
			WantErr: true,
		},
	})
}

// --- from composable_discard_each_test.go ---
func TestDiscardEachSimultaneousAndContinuation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	seen := 0
	watcher := NewTriggered(EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		seen++
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
			t.Errorf("discard event observed partial hands: %d/%d", len(a.Hand()), len(b.Hand()))
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, Sequence(DiscardEach(EachPlayer), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if seen != 2 || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 || a.Life() != 21 {
		t.Fatalf("events %d, graves %d/%d, life %d", seen, len(a.Graveyard()), len(b.Graveyard()), a.Life())
	}
}

func TestDiscardEachDeclarationAndClone(t *testing.T) {
	valid := NewSorcery("Valid", "", WithAbility(Spell(Sequence(DiscardEach(EachPlayer), DrawEach(EachPlayer, Fixed(7))))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    valid,
			WantErr: false,
		},
		{
			Name:    "nil player set",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEach(nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound player set",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEach(OpponentsOf(FromPlayer(BindPlayer("missing"))))))),
			WantErr: true,
		},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	clone := g.Clone()
	if err := ApplyEffect(clone, DiscardEach(OpponentsOf(You)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("wrong selected player or clone changed original")
	}
}

func TestDiscardEachReplacementPreparedBeforeMoves(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	seen := 0
	watcher := NewTriggered(EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		seen++
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 || len(a.Library()) != 1 || len(b.Library()) != 1 {
			t.Error("event saw partial replacement outcomes")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, DiscardEach(EachPlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if seen != 2 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
		t.Fatal("replacement or discard events lost")
	}
}

func TestDiscardEachAbsentSetAndPreventedActions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("A card", ""))
	b.AddToHand(NewArtifact("B card", ""))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	if err := ApplyEffect(g, DiscardEach(EachPlayer), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 || g.PlayerDiscardCountThisTurn(b.PlayerID()) != 0 {
		t.Fatal("prevented discard moved a card or sent an event")
	}
	if err := ApplyEffect(g, DiscardEach(OpponentsOf(FromPlayer(BindPlayer("absent")))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("absent player set changed a hand")
	}
}

// --- from composable_discard_hand_test.go ---
func TestTypedDiscardEntireHandReplacementAndContinuation(t *testing.T) {
	a := NewBasePlayer("A")
	b := &discardChoicePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Keep", ""))
	for range 3 {
		b.AddToHand(NewArtifact("Discard", ""))
	}
	events := 0
	watcher := NewTriggered(EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() {
			t.Error("wrong discard recipient")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	g.SetActivePlayerIndex(1)
	body := Sequence(DiscardEntireHand(ActivePlayer), GainLifeFor(You, Fixed(1)))
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(b.Hand()) != 0 || len(b.Library()) != 3 || len(b.Graveyard()) != 0 || events != 3 || len(a.Hand()) != 1 || a.Life() != 22 || len(b.requests) != 0 {
		t.Fatal("hand discard lost replacement, events, recipient, or continuation")
	}
}

func TestTypedDiscardEntireHandEventCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToOpponent(This, player), DiscardEntireHand(FromPlayer(player)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || len(original.Targets) != 0 {
		t.Fatal("missing untargeted trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("missing copy")
	}
	a.AddToHand(NewArtifact("Keep", ""))
	b.AddToHand(NewArtifact("Late card", ""))
	clone := g.Clone()
	clone.ResolveStack()
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("clone or captured recipient changed")
	}
	g.ResolveStackObject(copied)
	if len(b.Hand()) != 0 || len(a.Hand()) != 1 {
		t.Fatal("copy lost event recipient")
	}
	b.AddToHand(NewArtifact("Later card", ""))
	g.ResolveStackObject(original)
	if len(b.Hand()) != 0 || len(b.Graveyard()) != 2 {
		t.Fatal("hand was captured before resolution")
	}
}

func TestTypedDiscardEntireHandValidationAndAbsence(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound subject",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(FromPlayer(BindPlayer("missing")))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(DeclarePlayerTarget("undeclared"))))),
			WantErr: true,
		},
	})
	info := InspectDefinition(DiscardEntireHand(You))
	if info.Properties.Outcome != OutcomeDetriment {
		t.Fatal("missing detriment metadata")
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Keep", ""))
	if err := ApplyEffect(g, Sequence(DiscardEntireHand(ControllerOf(This)), GainLifeFor(You, Fixed(1))), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Life() != 21 {
		t.Fatal("absent subject changed hand or stopped sequence")
	}
}

// --- from composable_discard_result_test.go ---
func TestDiscardEntireHandActualCount(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToHand(NewLand("Card"))
		}
		if replaced {
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		}
		count := BindNumber("discarded")
		body := Sequence(DiscardEntireHand(You, DiscardAmountInto(count)), GainLifeFor(You, FromNumber(count)))
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 23 || len(a.Hand()) != 0 || len(a.Library()) != map[bool]int{false: 0, true: 3}[replaced] {
			t.Fatalf("replacement %v: count or destination is wrong", replaced)
		}
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 23 {
			t.Fatal("empty hand kept an old count")
		}
	}
}

func TestDiscardEntireHandCountPreventionAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewLand("Card"))
	g.effects.AddReplacement(&discardToHandReplacement{})
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	count := BindNumber("discarded")
	g.resolution.numberBindings = map[*NumberBinding]int{count: 9}
	clone := g.Clone()
	if err := DiscardEntireHand(You, DiscardAmountInto(count)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 0 || g.resolution.numberBindings[count] != 9 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("prevented discard or clone frame is wrong")
	}
	if err := DiscardEntireHand(FromPlayer(BindPlayer("absent")), DiscardAmountInto(count)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 0 {
		t.Fatal("absent recipient kept an old count")
	}
}

func TestDiscardEntireHandCountDeclaration(t *testing.T) {
	count := BindNumber("discarded")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewSorcery("Valid", "", WithAbility(Spell(Sequence(DiscardEntireHand(You, DiscardAmountInto(count)), DrawFor(You, FromNumber(count)))))),
			WantErr: false,
		},
		{
			Name:          "nil discard amount binding",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(You, DiscardAmountInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name discard amount binding",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(You, DiscardAmountInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate discard amount binding",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(You, DiscardAmountInto(count), DiscardAmountInto(count))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(DrawFor(You, FromNumber(count))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "late producer",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(Sequence(DrawFor(You, FromNumber(count)), DiscardEntireHand(You, DiscardAmountInto(count)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDiscardEntireHandCountStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("discarded")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(Dies(This), Sequence(
		DiscardEntireHand(You, DiscardAmountInto(count)),
		GainLifeFor(You, FromNumber(count)),
	))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToHand(NewLand("First"))
	a.AddToHand(NewLand("Second"))
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil {
		t.Fatal("missing trigger")
	}
	stackCopy := g.CopyStackObjectDirect(original, a.PlayerID(), false)
	if stackCopy == nil {
		t.Fatal("missing copy")
	}
	g.ResolveStackObject(stackCopy)
	if a.Life() != 22 {
		t.Fatal("copy did not use its discard count")
	}
	a.AddToHand(NewLand("Third"))
	g.ResolveStackObject(original)
	if a.Life() != 23 || len(a.Hand()) != 0 {
		t.Fatal("original reused the copy's count")
	}
}

// --- from composable_divide_test.go ---
func TestDivideValuesRoundingAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, tc := range []struct {
		n, d, down, up int
	}{
		{5, 2, 2, 3}, {6, 2, 3, 3}, {0, 2, 0, 0},
		{-5, 2, -3, -2}, {-6, 2, -3, -3}, {5, 0, 0, 0},
		{5, -2, -3, -2}, {-5, -2, 2, 3},
	} {
		if got := DivideRoundDown(Fixed(tc.n), Fixed(tc.d)).resolveValue(ctx); got != tc.down {
			t.Fatalf("down(%d, %d) = %d, want %d", tc.n, tc.d, got, tc.down)
		}
		if got := DivideRoundUp(Fixed(tc.n), Fixed(tc.d)).resolveValue(ctx); got != tc.up {
			t.Fatalf("up(%d, %d) = %d, want %d", tc.n, tc.d, got, tc.up)
		}
	}
	if got := DivideRoundUp(HandSizeOf(You), Fixed(2)).resolveValue(ctx); got != 0 {
		t.Fatalf("empty hand result = %d", got)
	}
	if got := DivideRoundUp(Fixed(5), HandSizeOf(You)).resolveValue(ctx); got != 0 {
		t.Fatalf("dynamic zero divisor result = %d", got)
	}
}

func TestDivideValuesBindingAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("numerator")
	d := BindNumber("denominator")
	body := Sequence(SnapshotNumber(Fixed(5), n), SnapshotNumber(Fixed(2), d),
		GainLifeFor(You, DivideRoundUp(FromNumber(n), FromNumber(d))))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 23 {
		t.Fatal("division did not use the clone's resolution bindings")
	}
}

func TestDivideValuesDeclaration(t *testing.T) {
	n := BindNumber("n")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(Sequence(SnapshotNumber(Fixed(5), n),
				GainLifeFor(You, DivideRoundDown(FromNumber(n), Fixed(2)))))),
			WantErr: false,
		},
		{
			Name:    "nil numerator",
			Card:    NewInstant("Invalid", "", Spell(GainLifeFor(You, DivideRoundDown(nil, Fixed(2))))),
			WantErr: true,
		},
		{
			Name:    "nil denominator",
			Card:    NewInstant("Invalid", "", Spell(GainLifeFor(You, DivideRoundUp(Fixed(2), nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound numerator",
			Card:    NewInstant("Invalid", "", Spell(GainLifeFor(You, DivideRoundDown(FromNumber(n), Fixed(2))))),
			WantErr: true,
		},
		{
			Name:    "fixed zero denominator",
			Card:    NewInstant("Invalid", "", Spell(GainLifeFor(You, DivideRoundUp(Fixed(2), Fixed(0))))),
			WantErr: true,
		},
	})
}

// --- from composable_draw_amount_test.go ---
func TestTypedDrawAmountContext(t *testing.T) {
	for _, each := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToLibrary(NewArtifact("A card", ""))
			b.AddToLibrary(NewArtifact("B card", ""))
		}
		source := NewCreature("Amount", "{2}", 2, 3)
		g.PutOnBattlefield(source, a.PlayerID())
		saved := BindPermanent("amount")
		var draw = DrawFor(You, PowerOf(From(saved)))
		if each {
			draw = DrawEach(EachPlayer, PowerOf(From(saved)))
		}
		body := Sequence(Snapshot(This, saved), draw)
		card := NewCreature("Draw", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		wantB := 0
		if each {
			wantB = 2
		}
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 2 || len(clone.GetPlayer(b.PlayerID()).Hand()) != wantB {
			t.Fatal("incorrect saved amount or recipient")
		}
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
			t.Fatal("draw changed original game")
		}
		if err := ApplyEffect(g, DrawFor(You, ValueAtResolution(TargetPermanentManaValue())), uuid.Nil, b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
			t.Fatal(err)
		}
		if len(b.Hand()) != 2 {
			t.Fatal("amount lost the outer target context")
		}
	}
}

func TestTypedDrawAmountCaptureAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	amount := countedLifeAmount{&calls}
	for _, body := range []ResolvingEffect{DrawFor(ControllerOf(This), amount), DrawEach(OpponentsOf(ControllerOf(This)), amount)} {
		if err := ApplyEffect(g, body, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("absent subjects evaluated amount")
	}
	if err := ApplyEffect(g, DrawEach(EachPlayer, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !a.DrewFromEmpty() || !b.DrewFromEmpty() {
		t.Fatal("collection did not capture once or retain empty-library behavior")
	}
}

func TestTypedDrawAmountValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil amount DrawFor",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawFor(You, nil)))),
			WantErr: true,
		},
		{
			Name:    "nil amount DrawEach",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawEach(EachPlayer, nil)))),
			WantErr: true,
		},
		{
			Name:    "nil subject PowerOf DrawFor",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawFor(You, PowerOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "nil subject PowerOf DrawEach",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawEach(EachPlayer, PowerOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "unbound permanent PowerOf DrawFor",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawFor(You, PowerOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
		{
			Name:    "unbound permanent PowerOf DrawEach",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawEach(EachPlayer, PowerOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target PowerOf DrawFor",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawFor(You, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target PowerOf DrawEach",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawEach(EachPlayer, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))))),
			WantErr: true,
		},
		{
			Name:    "nil ValueAtResolution DrawFor",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawFor(You, ValueAtResolution(nil))))),
			WantErr: true,
		},
		{
			Name:    "nil ValueAtResolution DrawEach",
			Card:    NewSorcery("Invalid draw", "", WithAbility(Spell(DrawEach(EachPlayer, ValueAtResolution(nil))))),
			WantErr: true,
		},
	})
	for _, each := range []bool{false, true} {
		var body = DrawFor(You, Fixed(2))
		if each {
			body = DrawEach(EachPlayer, Fixed(2))
		}
		info := InspectDefinition(body)
		if info.Properties.DrawCount != 2 || info.Properties.Outcome != OutcomeBenefit || info.Properties.Mass != each {
			t.Fatalf("metadata = %+v", info)
		}
	}
}

func TestTypedDrawAmountOrderAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetActivePlayerIndex(1)
	for range 2 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	var order []uuid.UUID
	watcher := NewTriggered(EvtCardDrawn, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		order = append(order, evt.PlayerID)
		return false
	})
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, DrawEach(EachPlayer, SubtractValues(Fixed(3), Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 || order[0] != b.PlayerID() || order[1] != b.PlayerID() || order[2] != a.PlayerID() || order[3] != a.PlayerID() {
		t.Fatalf("draw order = %v", order)
	}
	replacements := 0
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replacements++ })
	if err := ApplyEffect(g, DrawFor(You, SubtractValues(Fixed(3), Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if replacements != 2 || a.DrewFromEmpty() || len(order) != 4 {
		t.Fatal("individual replacements or draw events changed")
	}
	for _, n := range []int{-1, 0} {
		if err := ApplyEffect(g, DrawEach(EachPlayer, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if replacements != 2 || b.DrewFromEmpty() {
		t.Fatal("nonpositive amount drew cards")
	}
}

func TestTypedDrawAmountCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(DrawFor(You, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Draw", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if len(a.Hand()) != 0 || len(b.Hand()) != 2 {
		t.Fatal("copy lost controller or target amount")
	}
	g.ResolveStackObject(original)
	if len(a.Hand()) != 2 || len(b.Hand()) != 2 {
		t.Fatal("original lost its amount")
	}
}

// --- from composable_draw_card_result_test.go ---
func TestDrawCardResultVisitAndAbsence(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Drawn", "")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	effect := DrawFor(You, Fixed(1), DrawCardInto(binding))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got, ok := FromCard(binding).resolveCard(ctx); !ok || got.ID() != card.ID() {
		t.Fatal("drawn card missing")
	}
	clone := g.Clone()
	if _, ok := a.DiscardCard(card.ID()); !ok {
		t.Fatal("discard failed")
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("old hand visit survived departure")
	}
	a.AddToHand(card)
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("old hand visit followed return")
	}
	if _, ok := FromCard(binding).resolveCard(&EffectContext{Game: clone}); !ok {
		t.Fatal("clone lost saved visit")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("empty draw reused result")
	}
}

func TestDrawCardResultDeclaration(t *testing.T) {
	b := BindCard("drawn")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(DrawFor(You, Fixed(1), DrawCardInto(b)))),
			WantErr: false,
		},
		{
			Name:          "nil card binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardInto(nil)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "one card limit",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(2), DrawCardInto(b)))),
			WantErr:       true,
			WantErrSubstr: "one card",
		},
		{
			Name:          "duplicate card binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardInto(b), DrawCardInto(b)))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "one kind of option",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardInto(b), DrawAmountInto(BindNumber("count"))))),
			WantErr:       true,
			WantErrSubstr: "one kind",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(ReturnCardToHand(FromCard(b)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDrawCardResultReplacementAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) {})
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("replaced draw saved a card")
	}
	b.AddToLibrary(NewArtifact("Other", ""))
	target := DeclarePlayerTarget("player")
	if err := DrawFor(target, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("absent target saved a card")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}
	if err := DrawFor(target, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if card, ok := FromCard(binding).resolveCard(ctx); !ok || card.Name() != "Other" {
		t.Fatal("legal target draw missing")
	}
}

type observeDrawCardEffect struct {
	binding *CardBinding
	seen    map[uuid.UUID]uuid.UUID
}

func (*observeDrawCardEffect) Text() string                 { return "observe drawn card" }
func (*observeDrawCardEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *observeDrawCardEffect) Apply(ctx *EffectContext) error {
	if card, ok := FromCard(e.binding).resolveCard(ctx); ok {
		e.seen[ctx.Controller] = card.ID()
	}
	return nil
}
func (e *observeDrawCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", bindingRead: e.binding, children: []definitionEdge{{"binding", e.binding}}}
}

func TestDrawCardResultStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aCard, bCard := NewArtifact("A card", ""), NewArtifact("B card", "")
	a.AddToLibrary(aCard)
	b.AddToLibrary(bCard)
	binding := BindCard("drawn")
	seen := make(map[uuid.UUID]uuid.UUID)
	ability := Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), &observeDrawCardEffect{binding, seen}))
	card := NewInstant("Synthetic draw", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Effects: ability.Effects()}
	g.pushStack(obj)
	if g.CopyStackObjectDirect(obj, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if seen[a.PlayerID()] != aCard.ID() || seen[b.PlayerID()] != bCard.ID() {
		t.Fatalf("copy results = %v", seen)
	}
	if len(g.resolution.cardBindings) != 0 {
		t.Fatal("card result escaped resolution")
	}
}

// --- from composable_draw_cards_result_test.go ---
func TestDrawCardsResultVisitsAndReplacement(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	result := BindCards("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := DrawFor(You, Fixed(3), DrawCardsInto(result))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 2 {
		t.Fatalf("drawn cards = %d, want 2", got)
	}
	cards := FromCards(result).resolveCards(ctx)
	if len(cards) != 2 || cards[0].ID() != first.ID() || cards[1].ID() != second.ID() {
		t.Fatal("draw result lost the cards or their draw order")
	}
	clone := g.Clone()
	if _, ok := a.DiscardCard(first.ID()); !ok {
		t.Fatal("discard failed")
	}
	a.AddToHand(first)
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 1 {
		t.Fatalf("returned card counted: %d", got)
	}
	if got := CountCards(FromCards(result)).resolveValue(&EffectContext{Game: clone}); got != 2 {
		t.Fatalf("clone cards = %d, want 2", got)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 0 {
		t.Fatalf("empty draw retained cards: %d", got)
	}
}

func TestDrawCardsResultReplacedDraw(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replaced++ })
	result := BindCards("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(2), DrawCardsInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if replaced != 2 || CountCards(FromCards(result)).resolveValue(ctx) != 0 {
		t.Fatal("replaced draws saved cards")
	}
}

func TestDrawCardsResultTargetAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	for range 4 {
		b.AddToLibrary(NewArtifact("Library card", ""))
	}
	result := BindCards("drawn")
	target := DeclarePlayerTarget("player")
	body := Sequence(DrawFor(target, Fixed(2), DrawCardsInto(result)), GainLifeFor(You, CountCards(FromCards(result))))
	spell := NewInstant("Synthetic draw", "", WithAbility(Spell(body, WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("copy life = %d, want 22", got)
	}
	if len(b.Hand()) != 0 {
		t.Fatal("copy changed original hand")
	}
	g.ResolveStack()
	if a.Life() != 22 || len(b.Hand()) != 2 {
		t.Fatal("original result or draw changed")
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(target, Fixed(1), DrawCardsInto(result)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := CountCards(FromCards(result)).resolveValue(ctx); got != 0 {
		t.Fatalf("absent target retained cards: %d", got)
	}
}

func TestDrawCardsResultDeclaration(t *testing.T) {
	result := BindCards("drawn")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(2), DrawCardsInto(result)), GainLifeFor(You, CountCards(FromCards(result)))))),
			WantErr: false,
		},
		{
			Name:          "nil cards binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardsInto(nil)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name cards binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardsInto(BindCards(""))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate cards binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardsInto(result), DrawCardsInto(result)))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "one kind of option",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawCardsInto(result), DrawCardInto(BindCard("one"))))),
			WantErr:       true,
			WantErrSubstr: "one kind",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(GainLifeFor(You, CountCards(FromCards(result))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "conditional producer",
			Card:          NewInstant("Invalid", "", Spell(Sequence(IfState(Controls(IsCreature), DrawFor(You, Fixed(1), DrawCardsInto(result))), GainLifeFor(You, CountCards(FromCards(result)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_draw_result_test.go ---
func TestDrawResultActualCountAndReset(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		a.AddToLibrary(NewArtifact("Card", ""))
	}
	n := BindNumber("drawn")
	body := Sequence(DrawFor(You, Fixed(3), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 2 || a.Life() != 22 || g.resolution.numberBindings[n] != 2 {
		t.Fatalf("draw count: hand=%d life=%d result=%d", len(a.Hand()), a.Life(), g.resolution.numberBindings[n])
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || g.resolution.numberBindings[n] != 0 {
		t.Fatal("empty draw reused an old result")
	}
	if err := DrawFor(ControllerOf(This), Fixed(1), DrawAmountInto(n)).Apply(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[n] != 0 {
		t.Fatal("absent recipient reused an old result")
	}
}

func TestDrawResultReplacementAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToLibrary(NewArtifact("Card", ""))
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replaced++ })
	n := BindNumber("drawn")
	body := Sequence(DrawFor(You, Fixed(2), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	replaced = 0
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 || len(a.Hand()) != 0 {
		t.Fatal("clone shared draw or result state")
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || replaced != 1 || a.DrewFromEmpty() || g.resolution.numberBindings[n] != 1 {
		t.Fatalf("replacement: life=%d replaced=%d empty=%v result=%d", a.Life(), replaced, a.DrewFromEmpty(), g.resolution.numberBindings[n])
	}
}

func TestDrawResultDefinition(t *testing.T) {
	n := BindNumber("drawn")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "valid",
			Card:    NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n))))),
			WantErr: false,
		},
		{
			Name:          "nil draw amount binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawAmountInto(nil)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "duplicate draw amount binding",
			Card:          NewInstant("Invalid", "", Spell(DrawFor(You, Fixed(1), DrawAmountInto(n), DrawAmountInto(n)))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(GainLifeFor(You, FromNumber(n)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "late producer",
			Card:          NewInstant("Invalid", "", Spell(Sequence(GainLifeFor(You, FromNumber(n)), DrawFor(You, Fixed(1), DrawAmountInto(n))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestDrawResultTargetAndStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	target := DeclarePlayerTarget("player")
	n := BindNumber("drawn")
	body := Sequence(DrawFor(target, Fixed(2), DrawAmountInto(n)), GainLifeFor(You, FromNumber(n)))
	ability := Spell(body, WithTargets(target))
	card := NewInstant("Synthetic draw", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{a.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copied := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if len(a.Hand()) != 2 || a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("copied draw: hand=%d A=%d B=%d", len(a.Hand()), a.Life(), b.Life())
	}
	if err := DrawFor(target, Fixed(1), DrawAmountInto(n)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[n] != 0 {
		t.Fatal("missing target reused a draw result")
	}
}

// --- from composable_draw_step_replacement_test.go ---
func TestStaticDrawStepReplacementLifetime(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Static skip", "", 1, 4, WithAbility(Static(SkipDrawStepForLife(3))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	assertSkip := func(game *Game, want bool) {
		t.Helper()
		got := game.effects.ApplyReplacements(&beginDrawStepAction{playerID: a.PlayerID()}, game) == nil
		if got != want {
			t.Fatalf("skip = %v, want %v", got, want)
		}
	}
	g.PutOnBattlefield(card, a.PlayerID())
	assertSkip(g, true)
	if len(g.triggers.Pending()) != 0 || g.stack.Size() != 0 || len(g.effects.replacements) != 0 {
		t.Fatal("static rule used entry registration")
	}
	clone := g.Clone()
	clone.AddContinuousEffect(RemoveAllAbilities(card.ID(), EndOfTurn))
	assertSkip(clone, false)
	assertSkip(g, true)
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	assertSkip(clone, true)
	clone.PhaseOut(card.ID())
	assertSkip(clone, false)
	clone.PhaseIn([]uuid.UUID{card.ID()})
	assertSkip(clone, true)
	clone.RemoveFromBattlefield(clone.FindPermanent(card.ID()))
	assertSkip(clone, false)
	clone.PutOnBattlefield(card, a.PlayerID())
	assertSkip(clone, true)
	if len(clone.turns.pendingStepLifeGains) != 4 {
		t.Fatalf("pending gains = %d", len(clone.turns.pendingStepLifeGains))
	}
	clone.RunStepWithPriority(PrecombatMain)
	if clone.GetPlayer(a.PlayerID()).Life() != 32 || a.Life() != 20 {
		t.Fatal("gain amount or clone isolation failed")
	}
}

func TestStaticDrawStepReplacementControl(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	b := &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	card := NewCreature("Static skip", "", 1, 4, WithAbility(Static(SkipDrawStepForLife(2))))
	g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.effects.ApplyReplacements(&beginDrawStepAction{playerID: a.PlayerID()}, g) == nil {
		t.Fatal("old controller skipped")
	}
	if g.effects.ApplyReplacements(&beginDrawStepAction{playerID: b.PlayerID()}, g) != nil {
		t.Fatal("new controller did not skip")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RunStepWithPriority(PrecombatMain)
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatal("pending gain changed recipient")
	}
}

func TestStaticDrawStepReplacementDecline(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		g.PutOnBattlefield(NewEnchantment("Static skip", "", WithAbility(Static(SkipDrawStepForLife(2)))), a.PlayerID())
	}
	action := &beginDrawStepAction{playerID: a.PlayerID()}
	if g.effects.ApplyReplacements(action, g) != action || len(g.turns.pendingStepLifeGains) != 0 {
		t.Fatal("declined replacement skipped the step")
	}
}

func TestStaticDrawStepReplacementAmountAndMultiplicity(t *testing.T) {
	for _, amount := range []int{0, 2, 5} {
		a := &acceptingMayPlayer{NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		for range 2 {
			g.PutOnBattlefield(NewEnchantment("Static skip", "", WithAbility(Static(SkipDrawStepForLife(amount)))), a.PlayerID())
		}
		action := &beginDrawStepAction{playerID: a.PlayerID()}
		if g.effects.ApplyReplacements(action, g) != nil {
			t.Fatal("accepted replacement did not skip")
		}
		if len(g.turns.pendingStepLifeGains) != 1 {
			t.Fatal("one skipped step produced multiple gains")
		}
		g.RunStepWithPriority(PrecombatMain)
		if a.Life() != 20+amount {
			t.Fatalf("life = %d, want %d", a.Life(), 20+amount)
		}
	}
}

func TestStaticDrawStepReplacementNegativeAmount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("negative amount accepted")
		}
	}()
	SkipDrawStepForLife(-1)
}

// --- from composable_dsl_test.go ---
func TestComposableStateSacrifice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "", 1, 1, WithSubTypes("Dwarf"), WithAbility(WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.CheckStateBasedActions()
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("state trigger bypassed stack")
	}
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.stack.Size() != 1 {
		t.Fatalf("stack size = %d", g.stack.Size())
	}
	g.ResolveStackObject(g.stack.Pop())
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
}

func TestComposableSourceReentry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), Sacrifice(This))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	old := g.stack.Pop()
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	g.PutOnBattlefield(c, a.PlayerID())
	g.ResolveStackObject(old)
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("old trigger sacrificed new incarnation")
	}
}

func TestComposableConditions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Dwarf", "", 1, 1, WithSubTypes("Dwarf"))
	g.PutOnBattlefield(c, a.PlayerID())
	conditions := []StateCondition{Controls(HasSubType("Dwarf")), ControlsNone(IsLand), BattlefieldHasNone(IsLand), AllConditions(Controls(IsCreature), NotCondition(Controls(IsLand))), AnyCondition(Controls(IsLand), Controls(IsCreature))}
	for _, condition := range conditions {
		ability := WhenState(condition, Sacrifice(This))
		ability.SetSource(c.ID())
		ability.SetController(a.PlayerID())
		for range 3 {
			if !ability.CheckTrigger(nil, g) {
				t.Fatal("condition false")
			}
		}
		if !ability.CheckTrigger(nil, g.Clone()) {
			t.Fatal("condition differs in clone")
		}
	}
}

func TestComposableDefinitionValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "empty sequence",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), Sequence()))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0]",
		},
		{
			Name:          "nil subject",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), Sacrifice(nil)))),
			WantErr:       true,
			WantErrSubstr: "effects[0].subject",
		},
		{
			Name:          "nil condition",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(nil, Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "condition",
		},
		{
			Name:          "empty conditions",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(AllConditions(), Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "condition",
		},
		{
			Name:          "nil child",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(AnyCondition(nil), Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "conditions[0]",
		},
		{
			Name:          "nil callback",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(StateConditionFunc("callback", []StateDependency{StateBattlefield}, nil), Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "callback",
		},
		{
			Name:          "missing dependencies",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(StateConditionFunc("callback", nil, func(StateReader, StateContext) bool { return true }), Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "dependencies",
		},
		{
			Name:    "battlefield action from spell",
			Card:    NewInstant("Invalid spell", "", WithEffect(Sacrifice(This))),
			WantErr: true,
		},
	})
}

func TestComposableMetadata(t *testing.T) {
	effect := Sequence(GainLifeFor(You, Fixed(2)), IfState(ControlsNone(IsLand), Sacrifice(This)))
	metadata := InspectDefinition(effect)
	if metadata.Role != "sequence" || len(metadata.Children) != 2 || metadata.Children[1].Role != "resolution-branch" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if metadata.Properties.LifeGain != 2 || effect.Text() == "" {
		t.Fatal("lost effect metadata")
	}
}

func TestComposableSourceLifetimes(t *testing.T) {
	for _, kind := range []string{"event", "activated", "delayed"} {
		for _, roundtrip := range []string{"reentry", "phasing", "clone"} {
			t.Run(kind+"/"+roundtrip, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				c := NewCreature("Source", "", 1, 1)
				switch kind {
				case "event":
					c.AddAbility(NewTriggered(EvtTapped, false, Sacrifice(This)).SetConditionData(EventSourceIsSelf{}))
				case "activated":
					WithActivatedAbility(Sacrifice(This), ManaCostOf("{0}"))(c)
				}
				g.PutOnBattlefield(c, a.PlayerID())
				switch kind {
				case "event":
					g.FireEvent(GameEvent{Type: EvtTapped, SourceID: c.ID()})
					g.PutTriggersOnStack()
				case "activated":
					if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				case "delayed":
					g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtTapped, SourceID: c.ID(), Controller: a.PlayerID(), Effects: []Effect{Sacrifice(This)}})
				}
				switch roundtrip {
				case "reentry":
					g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
					g.PutOnBattlefield(c, a.PlayerID())
				case "phasing":
					ids := g.PhaseOut(c.ID())
					g.PhaseIn(ids)
				case "clone":
					g = g.Clone()
				}
				if kind == "delayed" {
					g.FireEvent(GameEvent{Type: EvtTapped, SourceID: c.ID()})
					g.PutTriggersOnStack()
				}
				if g.stack.Size() != 1 {
					t.Fatalf("stack size = %d", g.stack.Size())
				}
				g.ResolveStackObject(g.stack.Pop())
				exists := g.FindPermanent(c.ID()) != nil
				if exists != (roundtrip == "reentry") {
					t.Fatalf("source exists = %v", exists)
				}
			})
		}
	}
}

func TestComposableSequenceSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	effect := Sequence(FuncEffect("return source", EffectProperties{}, func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
		g.PutOnBattlefield(c, controller)
		return nil
	}), Sacrifice(This))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("sequence followed source through zone change")
	}
}

func TestComposableStateReaderSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Green", "{G}", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	reader := snapshotState(g)
	p, ok := reader.Permanent(c.ID())
	if !ok || !p.HasColor(Green) {
		t.Fatal("missing snapshot")
	}
	first := reader.Battlefield()
	first[0] = StatePermanent{}
	clone := g.Clone()
	clone.ChangeObjectColor(c.ID(), Blue)
	after, _ := reader.Permanent(c.ID())
	if !after.HasColor(Green) {
		t.Fatal("snapshot changed")
	}
	original, _ := snapshotState(g).Permanent(c.ID())
	if !original.HasColor(Green) {
		t.Fatal("clone changed original")
	}
}

func TestComposableRegistrationBoundary(t *testing.T) {
	r := NewRegistry()
	defer func() {
		p := recover()
		if p == nil || !strings.Contains(fmt.Sprint(p), "Broken: abilities[0].condition") {
			t.Fatalf("panic = %v", p)
		}
	}()
	r.Register("Broken", func() Card { return NewCreature("Broken", "", 1, 1, WithAbility(WhenState(nil, Sacrifice(This)))) })
}

func TestComposableCloneOutstandingState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.CheckStateTriggers()
	if len(clone.triggers.Pending()) != 0 {
		t.Fatal("clone duplicated outstanding state trigger")
	}
	clone.ResolveStackObject(clone.stack.Pop())
	if g.stack.Size() != 1 || a.Life() != 20 {
		t.Fatal("clone changed original resolution")
	}
	if len(clone.triggers.Pending())+clone.stack.Size() != 1 {
		t.Fatal("clone did not rearm after resolution")
	}
}

func TestComposableConditionTruthTable(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("Opponent Dwarf", "", 1, 1, WithSubTypes("Dwarf")), b.PlayerID())
	for _, test := range []struct {
		condition StateCondition
		want      bool
	}{
		{Controls(HasSubType("Dwarf")), false}, {ControlsNone(HasSubType("Dwarf")), true},
		{BattlefieldHasNone(IsCreature), false}, {NotCondition(ControlsNone(IsLand)), false},
		{AllConditions(ControlsNone(IsLand), Controls(IsCreature)), false},
		{AnyCondition(Controls(IsLand), Controls(IsCreature)), false},
	} {
		if got := test.condition.checkState(g, StateContext{ControllerID: a.PlayerID()}); got != test.want {
			t.Fatalf("%s = %v", test.condition.inspectDefinition().Text, got)
		}
	}
}

func TestComposableTransientCharacteristics(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "{G}", 1, 1, WithAbility(WhenState(Controls(HasColorFilter(Blue)), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.ChangeObjectColor(c.ID(), Blue)
	g.ChangeObjectColor(c.ID(), Green)
	g.CheckStateTriggers()
	if len(g.triggers.Pending()) != 1 {
		t.Fatal("missed transient derived characteristic")
	}
}

func TestComposableDefinitionCyclesAndTypedNil(t *testing.T) {
	cyclic := &statePredicate{kind: "not"}
	cyclic.children = []StateCondition{cyclic}
	var nilEffect *sacrificeEffect
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "cyclic",
			Card:          NewCreature("Cycle", "", 1, 1, WithAbility(WhenState(cyclic, Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "cyclic",
		},
		{
			Name:          "typed nil effect",
			Card:          NewCreature("Nil", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), nilEffect))),
			WantErr:       true,
			WantErrSubstr: "effects[0]",
		},
	})
}

// --- from composable_enchantment_entry_subtype_test.go ---
func TestEnchantmentEntrySubtypeUsesEntryCharacteristics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		subtype    string
		controller bool
		phased     bool
		want       int
	}{
		{"aura", "Aura", false, false, 1},
		{"other enchantment", "Saga", false, false, 0},
		{"opposing aura", "Aura", true, false, 0},
		{"phased watcher", "Aura", false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters().OfSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			entry := NewEnchantment("Entry", "")
			entry.AddSubType(tc.subtype)
			entry.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.controller {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entry, controller)
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEnchantmentEntrySubtypeCaptureKeepsIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered aura")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().OfSubtype("Aura").WithPermanent(entered),
		AddCountersTo(From(entered), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.AddSubType("Aura")
	entry.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d, want 1", g.StackSize())
	}
	stackCopy := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if stackCopy == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.FindPermanent(entry.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("clone changed original: %d", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(entry, a.PlayerID())
	g.ResolveStack()
	if got := g.FindPermanent(entry.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("returned entry received %d counters", got)
	}
}

func TestEnchantmentEntrySubtypeSeesContinuousEffect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().OfSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "", WithAbility(Static(While(
		Controls(AnyPermanent), SetSubtypeFamily(This, SubtypeEnchantment, "Aura")))))
	entry.SetOwner(a.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("triggers = %d, want 1", got)
	}
}

func TestEnchantmentEntrySubtypeDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "empty subtype",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(EnchantmentYouControlEnters().OfSubtype(""), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "invalid subtype",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(EnchantmentYouControlEnters().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "multiple subtypes",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(EnchantmentYouControlEnters().OfSubtype("Aura").OfSubtype("Saga"), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "valid Aura",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters().OfSubtype("Aura"), GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(EnchantmentYouControlEnters().OfSubtype("Aura")); got.Role != "event-pattern" || got.Text == "" {
					t.Fatalf("inspection = %+v", got)
				}
			},
		},
	})
}

// --- from composable_enchantment_entry_test.go ---
func TestEnchantmentYouControlEntersCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered enchantment")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		AddCountersTo(From(entered), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	enchantment := NewEnchantment("Entry", "")
	enchantment.SetOwner(b.PlayerID())
	g.PutOnBattlefield(enchantment, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.FindPermanent(enchantment.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("clone changed original: %d counters", got)
	}
	g.ResolveStack()
	if got := g.FindPermanent(enchantment.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("entry added %d counters", got)
	}
}

func TestEnchantmentYouControlEntersFilters(t *testing.T) {
	for _, tc := range []struct {
		name     string
		opponent bool
		artifact bool
		phased   bool
		want     int
	}{
		{"controlled", false, false, false, 1},
		{"opponent controlled", true, false, false, 0},
		{"artifact", false, true, false, 0},
		{"phased source", false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters(), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			entry := NewEnchantment("Entry", "")
			if tc.artifact {
				entry = NewArtifact("Entry", "")
			}
			entry.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entry, controller)
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEnchantmentYouControlEntersDeclaration(t *testing.T) {
	entered := BindPermanent("entered")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil permanent binding",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(EnchantmentYouControlEnters().WithPermanent(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty permanent binding name",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(EnchantmentYouControlEnters().WithPermanent(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate permanent binding",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(EnchantmentYouControlEnters().WithPermanent(entered).WithPermanent(entered), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound read",
			Card: NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(
				EnchantmentYouControlEnters(), AddCountersTo(From(entered), P1P1, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestEnchantmentYouControlEntersReturnedAndCopied(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		entered := BindPermanent("entered")
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
			EnchantmentYouControlEnters().WithPermanent(entered),
			AddCountersTo(From(entered), P1P1, Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		entry := NewEnchantment("Entry", "")
		entry.SetOwner(b.PlayerID())
		g.PutOnBattlefield(entry, a.PlayerID())
		g.PutTriggersOnStack()
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(entry.ID()))
			g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			g.PutOnBattlefield(entry, a.PlayerID())
		}
		clone := g.Clone()
		clone.ResolveStack()
		if got := g.FindPermanent(entry.ID()).Counters[P1P1]; got != 0 {
			t.Fatal("clone changed original")
		}
		g.ResolveStack()
		want := 2
		if returned {
			want = 0
		}
		if got := int(g.FindPermanent(entry.ID()).Counters[P1P1]); got != want {
			t.Fatalf("returned=%v: counters=%d, want %d", returned, got, want)
		}
	}
}

func TestEnchantmentYouControlEntersPhasedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		AddCountersTo(From(entered), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	entry := NewEnchantment("Entry", "")
	entry.SetOwner(b.PlayerID())
	g.PutOnBattlefield(entry, a.PlayerID())
	g.PutTriggersOnStack()
	g.PhaseOut(entry.ID())
	g.ResolveStack()
	g.PhaseIn([]uuid.UUID{entry.ID()})
	if got := g.FindPermanent(entry.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("phased entry received %d counters", got)
	}
}

// --- from composable_exile_each_test.go ---
func TestExileEachUsesOneZoneChangeBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", ""), b.PlayerID())
	observer := NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(first.ID()) == nil && game.FindPermanent(second.ID()) != nil
		})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := ExileEach(MatchingPermanents(IsArtifact)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.GetExile()) != 2 || g.FindPermanent(observer.ID()) == nil {
		t.Fatal("an intermediate exile was visible")
	}
}

func TestExileEachIdentityAndRegistration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Artifact", "")
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	if err := ExileEach(fixedTapSet{[]*Permanent{old, current, current}}).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetExile()) != 1 || g.FindPermanent(current.ID()) == nil {
		t.Fatal("exile changed the wrong incarnation or shared clone state")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid exile each",
			Card: NewSorcery("Good", "", Spell(ExileEach(MatchingPermanents(IsArtifact)))),
		},
		{
			Name:    "nil set",
			Card:    NewSorcery("Bad", "", Spell(ExileEach(nil))),
			WantErr: true,
		},
	})
}

func TestExileEachSkipsPhasedAndReturnedTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), b.PlayerID())
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}, {id: other.ID(), incarnation: other.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	g.PhaseOut(other.ID())
	if err := ExileEach(targets).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(current.ID()) == nil || g.FindPermanentIncludingPhased(other.ID()) == nil || len(g.GetExile()) != 0 {
		t.Fatal("an absent or phased target was exiled")
	}
}

func TestExileEachQueuesSourceLeaveTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "", WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(2)))))
	second := NewArtifact("Second", "", WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(3)))))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	if err := ExileEach(ControlledPermanents(IsArtifact)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatalf("life after both leave triggers = %d", a.Life())
	}
}

func TestExileEachTokenDoesNotRemainInExile(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	token := g.PutOnBattlefield(NewArtifact("Token", ""), a.PlayerID())
	token.IsToken = true
	card := g.PutOnBattlefield(NewArtifact("Card", ""), b.PlayerID())
	if err := ExileEach(MatchingPermanents(IsArtifact)).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(token.ID()) != nil || g.FindPermanent(card.ID()) != nil || len(g.GetExile()) != 1 || g.GetExile()[0].Card.ID() != card.ID() {
		t.Fatal("token stayed in exile or nontoken card was lost")
	}
}

// --- from composable_exile_metadata_test.go ---
func TestComposableExileMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	controller := BindPlayer("controller")
	power := BindNumber("power")
	for _, effect := range []ResolvingEffect{
		Exile(target),
		Sequence(SnapshotPlayer(ControllerOf(target), controller), SnapshotNumber(PowerOf(target), power), Exile(target), GainLifeFor(FromPlayer(controller), FromNumber(power))),
	} {
		props := effect.Properties()
		if props.TargetPurposeOverride != AITargetExile || props.Outcome != OutcomeDetriment || props.Destroys {
			t.Fatalf("incorrect exile metadata: %+v", props)
		}
	}
}

// --- from composable_exile_result_test.go ---
func TestExileResultIncarnationAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Subject", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindNumber("exiled")
	body := Sequence(ExileWithResult(This, ExiledThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	clone := g.Clone()
	for _, tc := range []struct {
		name string
		game *Game
		move func(*Game)
		want int
	}{
		{"present", clone, nil, 1},
		{"returned", g, func(game *Game) {
			game.RemoveFromBattlefield(game.FindPermanent(p.ID()))
			game.PutOnBattlefield(card, a.PlayerID())
		}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.move != nil {
				tc.move(tc.game)
			}
			if err := body.Apply(&EffectContext{Game: tc.game, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
				t.Fatal(err)
			}
			if got := tc.game.resolution.numberBindings[result]; got != tc.want {
				t.Fatalf("result = %d, want %d", got, tc.want)
			}
			if got := tc.game.GetPlayer(a.PlayerID()).Life(); got != 20+2*tc.want {
				t.Fatalf("life = %d", got)
			}
		})
	}
}

func TestExileResultDeclaration(t *testing.T) {
	b := BindNumber("exiled")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid exile with result",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(At(YourUpkeep, Sequence(ExileWithResult(This, ExiledThisWayInto(b)), GainLifeFor(You, FromNumber(b)))))),
		},
		{
			Name:          "nil binding",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, ExileWithResult(This, ExiledThisWayInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty binding name",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, ExileWithResult(This, ExiledThisWayInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate result binding",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, ExileWithResult(This, ExiledThisWayInto(b), ExiledThisWayInto(b))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, Sequence(GainLifeFor(You, FromNumber(b)), ExileWithResult(This, ExiledThisWayInto(b)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestExileResultAbsentPhasedAndTarget(t *testing.T) {
	for _, state := range []string{"absent", "phased", "returned", "present"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := NewCreature("Victim", "", 2, 2)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			result := BindNumber("exiled")
			target := DeclarePermanentTarget("target", IsCreature)
			spell := NewInstant("Exile Test", "", Spell(Sequence(ExileWithResult(target, ExiledThisWayInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2)))), WithTargets(target)))
			if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			a.AddToHand(spell)
			g.SetStep(PrecombatMain)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(victim, b.PlayerID())
			}
			g.ResolveStack()
			want := 0
			if state == "present" {
				want = 1
			}
			if a.Life() != 20+2*want {
				t.Fatalf("life = %d, want %d", a.Life(), 20+2*want)
			}
			if state == "present" && g.FindPermanent(p.ID()) != nil {
				t.Fatal("subject remains on battlefield")
			}
		})
	}
}

func TestExileResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	result := BindNumber("exiled")
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Sequence(
		ExileWithResult(target, ExiledThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target))
	spell := NewInstant("Exile Copy", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopySpellOnStack(spell.ID(), a.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("spell copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(p.ID()) != nil {
		t.Fatalf("copy and original did not keep independent outcomes: life = %d", a.Life())
	}
}

// --- from composable_fight_test.go ---
func TestFightTypedTargetsAndIdentity(t *testing.T) {
	for _, changed := range []string{"none", "first returned", "second returned", "second noncreature"} {
		t.Run(changed, func(t *testing.T) {
			g, a, b := newTriggerTargetGame()
			first := NewCreature("First", "", 2, 4)
			first.SetOwner(a.PlayerID())
			firstPerm := g.PutOnBattlefield(first, a.PlayerID())
			second := NewCreature("Second", "", 3, 4)
			second.SetOwner(b.PlayerID())
			secondPerm := g.PutOnBattlefield(second, b.PlayerID())
			firstRef := DeclarePermanentTarget("first", IsCreature)
			secondRef := DeclarePermanentTarget("second", IsCreature)
			ability := Spell(Fight(firstRef, secondRef), WithTargets(firstRef, secondRef))
			card := NewInstant("Fight", "", WithAbility(ability))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition("Fight", card); err != nil {
				t.Fatal(err)
			}
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			switch changed {
			case "first returned":
				g.RemoveFromBattlefield(firstPerm)
				firstPerm = g.PutOnBattlefield(first, a.PlayerID())
			case "second returned":
				g.RemoveFromBattlefield(secondPerm)
				secondPerm = g.PutOnBattlefield(second, b.PlayerID())
			case "second noncreature":
				secondPerm.RevokeBaseAttr(AttrIsCreature)
			}
			g.ResolveStackObject(obj)
			wantFirst, wantSecond := 0, 0
			if changed == "none" {
				wantFirst, wantSecond = 3, 2
			}
			if firstPerm.Damage != wantFirst || secondPerm.Damage != wantSecond {
				t.Fatalf("damage = (%d, %d), want (%d, %d)", firstPerm.Damage, secondPerm.Damage, wantFirst, wantSecond)
			}
		})
	}
}

func TestFightSelfDealsTwicePower(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	c := NewCreature("Self", "", 2, 5)
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	if err := ApplyEffect(g, Fight(This, This), c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.Damage != 4 {
		t.Fatalf("damage = %d, want 4", p.Damage)
	}
}

func TestFightDamageObserversSeeBothResults(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("First", "", 2, 5)
	first.SetOwner(a.PlayerID())
	firstPerm := g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "", 3, 5)
	second.SetOwner(b.PlayerID())
	secondPerm := g.PutOnBattlefield(second, b.PlayerID())
	observations := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observations++
		if firstPerm.Damage != 3 || secondPerm.Damage != 2 {
			t.Errorf("observer saw damage (%d, %d), want (3, 2)", firstPerm.Damage, secondPerm.Damage)
		}
	})
	fightPermanents(&EffectContext{Game: g, Controller: a.PlayerID()}, firstPerm, secondPerm)
	if observations != 2 {
		t.Fatalf("observations = %d, want 2", observations)
	}
}

func TestFightBindingValidation(t *testing.T) {
	first := DeclarePermanentTarget("first", IsCreature)
	second := DeclarePermanentTarget("second", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil first",
			Card:          NewInstant("Invalid Fight", "", WithAbility(Spell(Fight(nil, second), WithTargets(second)))),
			WantErr:       true,
			WantErrSubstr: ".first",
		},
		{
			Name:          "undeclared second",
			Card:          NewInstant("Invalid Fight", "", WithAbility(Spell(Fight(first, second), WithTargets(first)))),
			WantErr:       true,
			WantErrSubstr: ".second",
		},
		{
			Name:          "nil second",
			Card:          NewInstant("Invalid Fight", "", WithAbility(Spell(Fight(first, nil), WithTargets(first)))),
			WantErr:       true,
			WantErrSubstr: ".second",
		},
	})
}

func TestFightTypedCopyAndClone(t *testing.T) {
	g, a, b := newTriggerTargetGame()
	first := NewCreature("First", "", 2, 5)
	first.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "", 3, 5)
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	firstRef := DeclarePermanentTarget("first", IsCreature)
	secondRef := DeclarePermanentTarget("second", IsCreature)
	ability := Spell(Fight(firstRef, secondRef), WithTargets(firstRef, secondRef))
	card := NewInstant("Fight", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	copyObj := clone.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	clone.ResolveStackObject(copyObj)
	if clone.FindPermanent(first.ID()).Damage != 3 || clone.FindPermanent(second.ID()).Damage != 2 {
		t.Fatal("copied fight lost its target bindings")
	}
	if g.FindPermanent(first.ID()).Damage != 0 || g.FindPermanent(second.ID()).Damage != 0 {
		t.Fatal("cloned fight changed original game")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()).Damage != 3 || g.FindPermanent(second.ID()).Damage != 2 {
		t.Fatal("original fight lost its target bindings")
	}
}

// --- from composable_gain_life_each_test.go ---
func TestGainLifeEachFinalTotalsAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	seen := 0
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtLifeGained, false, GainLifeFor(You, Fixed(1))).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if a.Life() != 26 || b.Life() != 26 || event.Amount != 6 {
			t.Errorf("event saw life %d/%d and amount %d", a.Life(), b.Life(), event.Amount)
		}
		seen++
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.AddLifeGainModifier(watcher.ID(), func(_ *Game, _ uuid.UUID, amount int) int {
		if a.Life() != 20 || b.Life() != 20 {
			t.Errorf("modifier saw a partial batch: %d/%d", a.Life(), b.Life())
		}
		return amount * 2
	})
	if err := ApplyEffect(g, GainLifeEach(EachPlayer, Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 26 || b.Life() != 26 || seen != 2 {
		t.Fatalf("life %d/%d, events %d", a.Life(), b.Life(), seen)
	}
}

func TestGainLifeEachZeroAbsentAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	clone := g.Clone()
	for _, effect := range []ResolvingEffect{
		GainLifeEach(EachPlayer, Fixed(0)),
		GainLifeEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), Fixed(4)),
		GainLifeEach(OpponentsOf(You), Fixed(2)),
	} {
		if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("wrong selected players or clone changed original")
	}
}

func TestGainLifeEachReplacementOutcomes(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		amount  int
		consume bool
	}{
		{"changed", 5, false},
		{"consumed", 0, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.effects.AddReplacement(&lifeOutcomeReplacement{amount: scenario.amount, consume: scenario.consume})
			if err := ApplyEffect(g, GainLifeEach(EachPlayer, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			for _, player := range []Player{a, b} {
				if player.Life() != 20+scenario.amount || g.PlayerLifeGainedThisTurn(player.PlayerID()) != scenario.amount {
					t.Errorf("life or event outcome for %s: %d/%d", player.Name(), player.Life(), g.PlayerLifeGainedThisTurn(player.PlayerID()))
				}
			}
		})
	}
}

func TestGainLifeEachDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(GainLifeEach(EachPlayer, Fixed(2)))),
		},
		{
			Name:    "nil players",
			Card:    NewInstant("Invalid", "", Spell(GainLifeEach(nil, Fixed(1)))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewInstant("Invalid", "", Spell(GainLifeEach(EachPlayer, nil))),
			WantErr: true,
		},
		{
			Name:    "missing player binding",
			Card:    NewInstant("Invalid", "", Spell(GainLifeEach(OpponentsOf(FromPlayer(BindPlayer("missing"))), Fixed(1)))),
			WantErr: true,
		},
	})
}

// --- from composable_gain_life_result_test.go ---
func TestGainLifeEachActualResult(t *testing.T) {
	for _, amount := range []int{-1, 0, 3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		result := BindNumber("life gained")
		body := Sequence(
			GainLifeEach(EachPlayer, Fixed(amount), LifeGainedInto(result)),
			LoseLifeFor(You, FromNumber(result)),
		)
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := max(0, amount)
		if a.Life() != 20-want || b.Life() != 20+want {
			t.Fatalf("amount %d: life %d/%d", amount, a.Life(), b.Life())
		}
	}
}

func TestGainLifeEachResultReplacementAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	result := BindNumber("life gained")
	g.resolution.numberBindings = map[*NumberBinding]int{result: 99}
	clone := g.Clone()
	if err := GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 10 || g.resolution.numberBindings[result] != 99 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("replacement result or clone isolation failed")
	}
	if err := GainLifeEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), Fixed(2), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 0 {
		t.Fatal("absent selection kept a stale result")
	}
}

func TestGainLifeEachResultCanReadPriorValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("life gained")
	body := Sequence(
		GainLifeEach(OpponentsOf(You), Fixed(2), LifeGainedInto(result)),
		GainLifeEach(OpponentsOf(You), FromNumber(result), LifeGainedInto(result)),
		LoseLifeFor(You, FromNumber(result)),
	)
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 24 {
		t.Fatalf("life %d/%d, want 18/24", a.Life(), b.Life())
	}
}

func TestGainLifeEachResultDeclaration(t *testing.T) {
	result := BindNumber("life gained")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(
				GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result)),
				LoseLifeFor(You, FromNumber(result)),
			)))),
		},
		{
			Name: "nil binding",
			Card: NewArtifact("Invalid", "", WithAbility(Activate(Costs(),
				GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name: "empty binding name",
			Card: NewArtifact("Invalid", "", WithAbility(Activate(Costs(),
				GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name: "duplicate result binding",
			Card: NewArtifact("Invalid", "", WithAbility(Activate(Costs(),
				GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result), LifeGainedInto(result))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name: "missing producer",
			Card: NewArtifact("Invalid", "", WithAbility(Activate(Costs(),
				Sequence(LoseLifeFor(You, FromNumber(result)), GainLifeEach(EachPlayer, Fixed(2), LifeGainedInto(result)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_gathered_tap_test.go ---
func TestComposableOpponentChosenTap(t *testing.T) {
	for _, many := range []bool{false, true} {
		var binding TargetBinding
		if many {
			binding = DeclarePermanentTargets("opponent", 1, 2, IsCreature, TargetOpponentControls(), TargetChosenByOpponent())
		} else {
			binding = DeclarePermanentTarget("opponent", IsCreature, TargetOpponentControls(), TargetChosenByOpponent())
		}
		spec := binding.targetSpec()
		if !isOpponentChosenTarget(spec) {
			t.Fatal("typed declaration lost the opponent chooser")
		}
	}
}

func TestComposableSavedTapLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Saved", "", 2, 2)
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := TapSubject(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				clone.PhaseIn([]uuid.UUID{c.ID()})
			}
			want := change == "none" || change == "phase cycle" || change == "control"
			if got := clone.FindPermanent(c.ID()); got != nil && got.Tapped != want {
				t.Fatalf("tapped = %v, want %v", got.Tapped, want)
			}
			if got := g.FindPermanent(c.ID()); got != nil && got.Tapped {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableTapBindingValidation(t *testing.T) {
	saved := BindPermanent("saved")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewCreature("Invalid tap", "", 1, 1, WithAbility(Activate(Costs(), TapSubject(nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound permanent",
			Card:    NewCreature("Invalid tap", "", 1, 1, WithAbility(Activate(Costs(), TapSubject(From(saved))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewCreature("Invalid tap", "", 1, 1, WithAbility(Activate(Costs(), TapSubject(DeclarePermanentTarget("undeclared", IsCreature))))),
			WantErr: true,
		},
		{
			Name:    "producer after reader",
			Card:    NewCreature("Invalid tap", "", 1, 1, WithAbility(Activate(Costs(), Sequence(TapSubject(From(saved)), Snapshot(This, saved))))),
			WantErr: true,
		},
	})
}

// --- from composable_graveyard_entry_tapped_test.go ---
func TestReturnCardToBattlefieldTappedAtEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Guest", "", 2, 2,
		WithAbility(WhenEvent(Enters(This), IfState(SourceTappedCondition(), GainLifeFor(You, Fixed(1))))),
		WithAbility(WhenEvent(BecomesTapped(This), GainLifeFor(You, Fixed(3)))))
	card.SetOwner(b.PlayerID())
	b.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterTapped()))
	obj.Targets = []uuid.UUID{card.ID()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if a.Life() != 20 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("clone changed the original game")
	}
	g.ResolveTopOfStack()
	g.PutTriggersOnStack()
	g.ResolveStack()
	perm := g.FindPermanent(card.ID())
	if perm == nil || !perm.Tapped || perm.ControllerID() != a.PlayerID() || perm.Card.Owner() != b.PlayerID() {
		t.Fatal("card did not enter tapped under the chosen controller")
	}
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("entry trigger did not observe tapped state, or a tap transition fired")
	}
}

func TestReturnCardToBattlefieldTappedStaleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Guest")
	card.SetOwner(b.PlayerID())
	b.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterTapped()))
	obj.Targets = []uuid.UUID{card.ID()}
	g.pushStack(obj)
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("could not copy the return spell")
	}
	g.ResolveTopOfStack()
	first := g.FindPermanent(card.ID())
	if first == nil || !first.Tapped || g.StackSize() != 1 {
		t.Fatal("copy did not return the card tapped")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) != first || !first.Tapped {
		t.Fatal("original spell followed the card out of the graveyard")
	}

	card2 := NewLand("Stale")
	card2.SetOwner(b.PlayerID())
	b.AddToGraveyard(card2)
	_, stale := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You, EnterTapped()))
	stale.Targets = []uuid.UUID{card2.ID()}
	g.pushStack(stale)
	moved, ok := g.MoveFromGraveyard(b.PlayerID(), card2.ID(), ZoneHand)
	if !ok {
		t.Fatal("could not move the target")
	}
	b.AddToHand(moved)
	returned, ok := b.RemoveFromHand(card2.ID())
	if !ok {
		t.Fatal("could not return the target")
	}
	b.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if g.FindPermanent(card2.ID()) != nil {
		t.Fatal("stale graveyard target entered the battlefield")
	}
}

func TestReturnCardToBattlefieldTappedValidation(t *testing.T) {
	target := DeclareGraveyardCardTarget("guest")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewSorcery("Return", "", WithAbility(Spell(ReturnCardToBattlefield(target, You, EnterTapped()), WithTargets(target)))),
		},
		{
			Name:          "duplicate",
			Card:          NewSorcery("Return", "", WithAbility(Spell(ReturnCardToBattlefield(target, You, EnterTapped(), EnterTapped()), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: "one entry option",
		},
		{
			Name:          "nil option",
			Card:          NewSorcery("Return", "", WithAbility(Spell(ReturnCardToBattlefield(target, You, nil), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: "non-nil entry option",
		},
		{
			Name:          "missing card",
			Card:          NewSorcery("Return", "", WithAbility(Spell(ReturnCardToBattlefield(nil, You, EnterTapped()), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: ".card",
		},
	})
}

func TestCapturedDiscardedLandEntersTapped(t *testing.T) {
	a := &cardSetChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Chosen land")
	land.SetOwner(b.PlayerID())
	b.AddToHand(land)
	artifact := NewArtifact("Other card", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToHand(artifact)
	a.choice = land.ID()
	discarded, picked := BindCards("discarded"), BindCard("picked")
	player := DeclarePlayerTarget("player")
	spell := NewInstant("Synthetic return", "", WithAbility(Spell(Sequence(
		Discard(player, Fixed(2), DiscardCardsInto(discarded)),
		ChooseUpToOneCard(You, FromCards(discarded), IsLandCard, picked),
		ReturnCardToBattlefield(FromCard(picked), You, EnterTapped()),
	), WithTargets(player))))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	perm := g.FindPermanent(land.ID())
	if perm == nil || !perm.Tapped || perm.ControllerID() != a.PlayerID() || perm.Card.Owner() != b.PlayerID() {
		t.Fatal("chosen discarded land did not enter tapped under the spell controller")
	}
	if !slices.Contains(a.seen, land.ID()) || slices.Contains(a.seen, artifact.ID()) {
		t.Fatal("choice did not offer the discarded land")
	}
}

// --- from composable_graveyard_optional_test.go ---
func TestOptionalGraveyardTargetTriggerWithoutCard(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne(), GraveyardCardFilter(IsCreatureCard))
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		Sequence(GainLifeFor(You, Fixed(1)), ReturnCardToBattlefield(target, You)), WithTargets(target))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if obj := g.StackPeek(); obj == nil || len(obj.Targets) != 0 {
		t.Fatalf("zero-target trigger = %v", obj)
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("zero-target trigger life = %d", a.Life())
	}
}

func TestOptionalGraveyardTargetMayChooseZeroWithLegalCard(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	card := NewCreature("Creature", "", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne())
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		GainLifeFor(You, Fixed(1)), WithTargets(target))))
	watcher.SetOwner(a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{}}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if obj := g.StackPeek(); obj == nil || len(obj.Targets) != 0 {
		t.Fatalf("voluntary zero-target trigger = %v", obj)
	}
	copyObj := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copyObj == nil || len(copyObj.Targets) != 0 {
		t.Fatalf("zero-target copy = %v", copyObj)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("copied zero-target trigger life = %d", a.Life())
	}
}

func TestOptionalGraveyardTargetSelectedCopyAndRecheck(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "{1}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne(), GraveyardCardManaValueAtMost(FixedTargetNumber(1)))
	ability := Spell(ReturnCardToHand(target), WithTargets(target))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copyObj == nil || len(copyObj.Targets) != 1 || copyObj.Targets[0] != card.ID() {
		t.Fatalf("selected target copy = %v", copyObj)
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 0 {
		t.Fatal("clone changed original target")
	}
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	if !ok {
		t.Fatal("could not move target")
	}
	a.AddToHand(moved)
	returned, _ := a.RemoveFromHand(card.ID())
	a.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 {
		t.Fatal("old target followed a new graveyard visit")
	}
}

func TestOptionalGraveyardTargetRejectsDuplicateOption(t *testing.T) {
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne(), GraveyardUpToOne())
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "duplicate option",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToHand(target), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: "up to one",
		},
	})
}

// --- from composable_graveyard_test.go ---
func graveyardTestSpell(owner uuid.UUID, target *CardTargetBinding, effects ResolvingEffect) (Card, *StackObject) {
	ability := Spell(effects, WithTargets(target))
	card := NewSorcery("Recover", "", WithAbility(ability))
	card.SetOwner(owner)
	return card, &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: owner, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
}

func TestTypedGraveyardReturnUsesOwnerAndTargetBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	player := DeclarePlayerTarget("player")
	cardTarget := DeclareGraveyardCardTarget("card", InAnyGraveyard(), GraveyardCardFilter(IsCreatureCard))
	ability := Spell(Sequence(GainLifeFor(player, Fixed(1)), ReturnCardToHand(cardTarget)), WithTargets(player, cardTarget))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Recover", spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{a.PlayerID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if a.Life() != 21 || len(a.Hand()) != 0 || len(b.Hand()) != 1 || b.Hand()[0].ID() != second.ID() {
		t.Fatal("return used the wrong target or destination")
	}
	if !slices.ContainsFunc(a.Graveyard(), func(card Card) bool { return card.ID() == first.ID() }) {
		t.Fatal("return moved an unrelated card")
	}
}

func TestTypedGraveyardTargetVisitAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(a.PlayerID())
	a.AddToGraveyard(victim)
	target := DeclareGraveyardCardTarget("card")
	spell, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToHand(target))
	obj.Targets = []uuid.UUID{victim.ID()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 0 {
		t.Fatal("clone changed the original game")
	}
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), victim.ID(), ZoneHand)
	if !ok {
		t.Fatal("could not move target")
	}
	a.AddToHand(moved)
	removed, ok := a.RemoveFromHand(victim.ID())
	if !ok {
		t.Fatal("could not return target")
	}
	a.AddToGraveyard(removed)
	if !slices.Contains(target.Possible(a.PlayerID(), spell, g), victim.ID()) {
		t.Fatal("card is no longer targetable")
	}
	g.ResolveTopOfStack()
	if slices.ContainsFunc(a.Hand(), func(card Card) bool { return card.ID() == victim.ID() }) || !slices.ContainsFunc(a.Graveyard(), func(card Card) bool { return card.ID() == victim.ID() }) {
		t.Fatal("stale target followed a new graveyard visit")
	}
}

func TestTypedGraveyardTargetScopeAndFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToGraveyard(artifact)
	yours := DeclareGraveyardCardTarget("yours", GraveyardCardFilter(IsCreatureCard))
	fromAny := DeclareGraveyardCardTarget("any", InAnyGraveyard())
	if slices.Contains(yours.Possible(a.PlayerID(), nil, g), artifact.ID()) || !slices.Contains(fromAny.Possible(a.PlayerID(), nil, g), artifact.ID()) {
		t.Fatal("graveyard scope or card filter is wrong")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil",
			Card:          NewSorcery("Test", "", WithAbility(Spell(ReturnCardToHand(nil)))),
			WantErr:       true,
			WantErrSubstr: ".card",
		},
		{
			Name:          "undeclared",
			Card:          NewSorcery("Test", "", WithAbility(Spell(ReturnCardToHand(fromAny)))),
			WantErr:       true,
			WantErrSubstr: ".card",
		},
		{
			Name: "valid",
			Card: NewSorcery("Test", "", WithAbility(Spell(ReturnCardToHand(fromAny), WithTargets(fromAny)))),
		},
	})
}

func TestTypedGraveyardReturnToBattlefieldUsesBindingAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 1, 1)
	first.SetOwner(a.PlayerID())
	second := NewCreature("Second", "", 2, 2, WithKeyword(EntersTapped))
	second.SetOwner(b.PlayerID())
	b.AddToGraveyard(second)
	a.AddToGraveyard(first)
	player := DeclarePlayerTarget("player")
	cardTarget := DeclareGraveyardCardTarget("card", InAnyGraveyard(), GraveyardCardFilter(IsCreatureCard))
	ability := Spell(Sequence(GainLifeFor(player, Fixed(1)), ReturnCardToBattlefield(cardTarget, You)), WithTargets(player, cardTarget))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Recover", spell); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveTopOfStack()
	perm := g.FindPermanent(second.ID())
	if perm == nil || perm.ControllerID() != a.PlayerID() || perm.Card.Owner() != b.PlayerID() || !perm.Tapped || b.Life() != 21 {
		t.Fatal("return used the wrong card, controller, or entry path")
	}
	if g.FindPermanent(first.ID()) != nil || !slices.ContainsFunc(a.Graveyard(), func(card Card) bool { return card.ID() == first.ID() }) {
		t.Fatal("return moved an unrelated card")
	}
}

func TestTypedGraveyardReturnToBattlefieldUsesChosenController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Guest", "", 2, 2)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	cardTarget := DeclareGraveyardCardTarget("card")
	controller := DeclarePlayerTarget("controller")
	ability := Spell(ReturnCardToBattlefield(cardTarget, controller), WithTargets(cardTarget, controller))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID(), b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveTopOfStack()
	perm := g.FindPermanent(card.ID())
	if perm == nil || perm.ControllerID() != b.PlayerID() || perm.Card.Owner() != a.PlayerID() {
		t.Fatal("return did not use the bound controller")
	}
}

func TestTypedGraveyardReturnToBattlefieldWhenLandCannotEnter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Blocked Land")
	land.SetOwner(a.PlayerID())
	a.AddToGraveyard(land)
	g.effects.Rules.SetLandsCantEnter(true)
	target := DeclareGraveyardCardTarget("card")
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You))
	obj.Targets = []uuid.UUID{land.ID()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if g.FindPermanent(land.ID()) != nil || !slices.ContainsFunc(a.Graveyard(), func(card Card) bool { return card.ID() == land.ID() }) {
		t.Fatal("land left the graveyard despite the entry prohibition")
	}
}

func TestTypedGraveyardReturnToBattlefieldRejectsStaleVisit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Victim", "", 2, 2)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card")
	_, obj := graveyardTestSpell(a.PlayerID(), target, ReturnCardToBattlefield(target, You))
	obj.Targets = []uuid.UUID{card.ID()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if clone.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()) != nil {
		t.Fatal("clone did not isolate the return")
	}
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	if !ok {
		t.Fatal("could not move target")
	}
	a.AddToHand(moved)
	removed, ok := a.RemoveFromHand(card.ID())
	if !ok {
		t.Fatal("could not return target")
	}
	a.AddToGraveyard(removed)
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) != nil || !slices.ContainsFunc(a.Graveyard(), func(found Card) bool { return found.ID() == card.ID() }) {
		t.Fatal("stale target followed a new graveyard visit")
	}
}

func TestTypedGraveyardReturnToBattlefieldValidation(t *testing.T) {
	target := DeclareGraveyardCardTarget("card")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil card",
			Card:          NewSorcery("Test", "", WithAbility(Spell(ReturnCardToBattlefield(nil, You)))),
			WantErr:       true,
			WantErrSubstr: ".card",
		},
		{
			Name:          "nil controller",
			Card:          NewSorcery("Test", "", WithAbility(Spell(ReturnCardToBattlefield(target, nil), WithTargets(target)))),
			WantErr:       true,
			WantErrSubstr: ".controller",
		},
		{
			Name:          "undeclared",
			Card:          NewSorcery("Test", "", WithAbility(Spell(ReturnCardToBattlefield(target, You)))),
			WantErr:       true,
			WantErrSubstr: ".card",
		},
		{
			Name: "valid",
			Card: NewSorcery("Test", "", WithAbility(Spell(ReturnCardToBattlefield(target, You), WithTargets(target)))),
		},
	})
}

// --- from composable_land_entry_actor_test.go ---
func TestLandEntryActorDiffersFromController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("entry actor")
	controller := BindPlayer("entry controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().OfSubtype("Swamp").WithActor(actor).WithController(controller),
		Sequence(GainLifeFor(FromPlayer(actor), Fixed(2)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Swamp", WithSubTypes("Swamp"))
	land.SetOwner(b.PlayerID())
	g.PutOnBattlefieldBy(land, b.PlayerID(), a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefieldBy(land, a.PlayerID(), b.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 24 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

func TestLandEntryActorDeclaration(t *testing.T) {
	actor := BindPlayer("actor")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil actor",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithActor(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty actor name",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithActor(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate actor",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithActor(actor).WithActor(actor), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "actor same as controller",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithActor(actor).WithController(actor), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound actor read",
			Card:    NewArtifact("Unbound", "", WithAbility(WhenEvent(LandEnters(), GainLifeFor(FromPlayer(actor), Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestLandEntryActorRequiresKnownEntryPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().WithActor(actor), GainLifeFor(FromPlayer(actor), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(NewLand("Setup land"), b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("entry without a known actor triggered")
	}
	g.SetStep(PrecombatMain)
	land := NewLand("Played land")
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	if err := g.playLandCore(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 {
		t.Fatalf("played land stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

// --- from composable_land_entry_subtype_test.go ---
func TestLandEntrySubtypeAtEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("entering controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().OfSubtype("Swamp").WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	wantLife := 20
	for _, tc := range []struct {
		subtype string
		want    bool
	}{
		{"Forest", false}, {"Swamp", true}, {"Mountain", false}, {"Swamp", true},
	} {
		land := NewLand("Land", WithSubTypes(tc.subtype))
		land.SetOwner(a.PlayerID())
		g.PutOnBattlefield(land, b.PlayerID())
		g.PutTriggersOnStack()
		want := 0
		if tc.want {
			want = 1
		}
		if g.StackSize() != want {
			t.Fatalf("%s: stack=%d, want %d", tc.subtype, g.StackSize(), want)
		}
		if tc.want {
			g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
			clone := g.Clone()
			clone.ResolveStack()
			if b.Life() != wantLife {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantLife++
		}
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

func TestLandEntrySubtypeDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "empty subtype",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().OfSubtype(""), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "invalid subtype",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().OfSubtype("Goblin"), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate subtype",
			Card:    NewArtifact("Duplicate", "", WithAbility(WhenEvent(LandEnters().OfSubtype("Swamp").OfSubtype("Forest"), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

// --- from composable_land_mana_amount_test.go ---
func TestLandTappedForManaAmountCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("produced")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithAmount(amount), GainLifeFor(You, FromNumber(amount)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(2)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("clone lost amount or changed original")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestLandTappedForManaAmountDeclaration(t *testing.T) {
	amount := BindNumber("produced")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil amount binding",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithAmount(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty amount name",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithAmount(BindNumber("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate amount binding",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithAmount(amount).WithAmount(amount), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound read",
			Card: NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
				GainLifeFor(You, FromNumber(amount))))),
			WantErr: true,
		},
	})
}

// --- from composable_land_mana_land_test.go ---
func TestLandTappedForManaEventKeepsIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	var observed int
	watcher := NewEnchantment("Observer", "", WithAbility(NewTriggered(EvtTappedForMana, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == land.ID() && event.SourceIncarnation == g.FindPermanent(land.ID()).incarnationID {
			observed++
		}
		return false
	})))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if observed != 1 {
		t.Fatalf("observed matching incarnation %d times, want 1", observed)
	}
}

func TestLandTappedForManaLandBindingIdentityAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))))
	bound := BindPermanent("producing land")
	player := BindPlayer("activator")
	amount := BindNumber("amount")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithLand(bound).WithPlayer(player).WithAmount(amount),
		AddCountersTo(From(bound), P1P1, FromNumber(amount)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.FindPermanent(land.ID()).Counters[P1P1]; got != 1 {
		t.Fatalf("clone counters = %d, want 1", got)
	}
	if got := g.FindPermanent(land.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("original counters before resolution = %d", got)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, b.PlayerID())
	g.ResolveStack()
	if got := g.FindPermanent(land.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("returned land counters = %d, want 0", got)
	}
}

func TestLandTappedForManaLandBindingDeclaration(t *testing.T) {
	land := BindPermanent("land")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil land binding",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithLand(nil), AddCountersTo(From(land), P1P1, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty land name",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithLand(BindPermanent("")), AddCountersTo(From(land), P1P1, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate land binding",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(LandTappedForMana().WithLand(land).WithLand(land), AddCountersTo(From(land), P1P1, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound read",
			Card: NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
				AddCountersTo(From(land), P1P1, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestLandTappedForManaLandBindingPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))))
	bound := BindPermanent("producing land")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithLand(bound), AddCountersTo(From(bound), P1P1, Fixed(1)))))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutOnBattlefield(land, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.PhaseOut(land.ID())
	g.ResolveStack()
	g.PhaseIn([]uuid.UUID{land.ID()})
	if got := g.FindPermanent(land.ID()).Counters[P1P1]; got != 0 {
		t.Fatalf("phased land counters = %d, want 0", got)
	}
}

// --- from composable_land_sweeper_test.go ---
func TestDestroyEachMatchingLands(t *testing.T) {
	for _, subtype := range []string{"", "Island", "Plains", "Forest"} {
		name := "all lands"
		filter := IsLand
		if subtype != "" {
			name = subtype
			filter = HasSubType(subtype)
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := DestroyEach(MatchingPermanents(filter))
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			var destroyed, survivors []*Permanent
			for _, owner := range []*BasePlayer{a, b} {
				for _, scenario := range []string{"ordinary", "indestructible", "regenerate", "phased", "Mountain", "creature"} {
					card := NewLand(scenario, WithSubTypes(subtype))
					if scenario == "Mountain" {
						card = NewLand("Mountain", WithSubTypes("Mountain"))
					}
					if scenario == "creature" {
						card = NewCreature("Creature", "", 2, 2)
					}
					card.SetOwner(owner.PlayerID())
					p := g.PutOnBattlefield(card, owner.PlayerID())
					switch scenario {
					case "indestructible":
						p.GrantBaseAttr(Indestructible)
					case "regenerate":
						g.AddRegenerationShield(p.ID())
					case "phased":
						p.PhasedOut = true
					}
					if scenario == "ordinary" || (scenario == "Mountain" && subtype == "") {
						destroyed = append(destroyed, p)
					} else {
						survivors = append(survivors, p)
					}
				}
			}
			clone := g.Clone()
			if err := effect.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			for _, p := range destroyed {
				if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
					t.Fatalf("wrong destruction or clone isolation for %s", p.Name())
				}
			}
			for _, p := range survivors {
				got := clone.FindPermanentIncludingPhased(p.ID())
				if got == nil {
					t.Fatalf("destroyed %s", p.Name())
				}
				if p.Name() == "regenerate" && (!got.Tapped || p.Tapped) {
					t.Fatal("regeneration did not tap only the cloned permanent")
				}
			}
		})
	}
}

// --- from composable_layers_test.go ---
func TestComposableTypeAndColorCapture(t *testing.T) {
	g, player, _, land, printed := basicLandTypeTestGame(t)
	colors := []Color{Blue, Green}
	subtypes := []string{"Forest"}
	effect := Until(EndOfTurn,
		SetSubtypeFamily(This, SubtypeLand, subtypes...),
		SetColors(This, colors...),
		GrantKeywordModification(This, Flying),
	)
	colors[0] = Red
	subtypes[0] = "Island"
	if err := ApplyEffect(g, effect, land.ID(), player.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	land = g.FindPermanent(land.ID())
	if !land.HasSubType("Forest") || land.HasSubType("Mountain") || hasRuntimeAbility(land, printed) || countManaAbilityColor(land, Green) != 1 {
		t.Fatal("subtype replacement did not use the basic land rules")
	}
	if !slices.Equal(g.EffectiveColors(land.ID()), []Color{Blue, Green}) || !land.HasAttr(Flying) {
		t.Fatal("compound effect lost its color or later-layer keyword")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(land.ID()).HasSubType("Mountain") || !g.FindPermanent(land.ID()).HasSubType("Forest") {
		t.Fatal("expiration leaked between clones")
	}
	g.RemoveFromBattlefield(land)
	g.PutOnBattlefield(land.Card, player.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(land.ID()).HasSubType("Forest") || slices.Contains(g.EffectiveColors(land.ID()), Blue) {
		t.Fatal("captured modification followed a new incarnation")
	}
}

func TestComposableAddedSubtypePreservesAbilities(t *testing.T) {
	g, player, _, land, printed := basicLandTypeTestGame(t)
	if err := ApplyEffect(g, Until(EndOfTurn, AddSubtypeFamily(This, SubtypeLand, "Forest")), land.ID(), player.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	land = g.FindPermanent(land.ID())
	if !land.HasSubType("Mountain") || !land.HasSubType("Forest") || !hasRuntimeAbility(land, printed) || countManaAbilityColor(land, Red) != 1 || countManaAbilityColor(land, Green) != 1 {
		t.Fatal("adding a land subtype did not preserve old types and abilities")
	}
}

func TestComposableStaticColorUsesLayerMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(Static(While(Controls(IsCreature),
		SetEachColors(ControlledPermanents(IsCreature), Blue),
		GrantEachKeyword(ControlledPermanents(HasColorFilter(Blue)), Flying),
	))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	later := NewCreature("Later", "{R}", 2, 2)
	later.SetOwner(a.PlayerID())
	g.PutOnBattlefield(later, a.PlayerID())
	g.ApplyContinuousEffects()
	if !slices.Equal(g.EffectiveColors(later.ID()), []Color{Blue}) || !g.FindPermanent(later.ID()).HasAttr(Flying) {
		t.Fatal("static set omitted a later member")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ApplyContinuousEffects()
	if !slices.Equal(g.EffectiveColors(later.ID()), []Color{Red}) || g.FindPermanent(later.ID()).HasAttr(Flying) {
		t.Fatal("static effect remained after source exit")
	}
}

func TestComposableLayerValidation(t *testing.T) {
	for name, makeModification := range map[string]func() ContinuousModification{
		"invalid color":      func() ContinuousModification { return SetColors(This, AnyColor) },
		"colorless sentinel": func() ContinuousModification { return SetColors(This, Colorless) },
		"invalid family":     func() ContinuousModification { return SetSubtypeFamily(This, SubtypeFamily(255), "Human") },
		"wrong family":       func() ContinuousModification { return AddSubtypeFamily(This, SubtypeCreature, "Forest") },
		"nil subject":        func() ContinuousModification { return SetColors(nil, Blue) },
		"nil set":            func() ContinuousModification { return SetEachColors(nil, Blue) },
	} {
		t.Run(name, func(t *testing.T) {
			err := validateFactory(name, func() Card {
				return NewCreature(name, "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, makeModification()))))
			})
			if err == nil {
				t.Fatal("invalid modification accepted")
			}
		})
	}
}

func TestComposableColorlessAndSubtypeApplicability(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "{R}", 2, 2, WithSubTypes("Goblin"))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	effect := Until(EndOfTurn, SetColors(This), SetSubtypeFamily(This, SubtypeLand, "Forest"))
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	p := g.FindPermanent(card.ID())
	if len(g.EffectiveColors(card.ID())) != 0 || !p.HasSubType("Goblin") || p.HasSubType("Forest") {
		t.Fatal("colorless or subtype applicability contract failed")
	}
	phased := g.PhaseOut(card.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if len(g.EffectiveColors(card.ID())) != 0 {
		t.Fatal("phasing invalidated the captured reference")
	}
}

// --- from composable_life_amount_test.go ---
func TestTypedCollectionLifeSavedAmount(t *testing.T) {
	for _, power := range []int{-2, 0, 3} {
		t.Run(Fixed(power).Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Amount", "", power, 4)
			g.PutOnBattlefield(source, a.PlayerID())
			saved := BindPermanent("amount")
			body := Sequence(Snapshot(This, saved), LoseLifeEach(EachPlayer, PowerOf(From(saved))))
			card := NewCreature("Definition", "", 1, 1, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			events := 0
			watcher := NewTriggered(EvtLifeLost, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, game GameReader, _, _ uuid.UUID) bool {
				events++
				want := 20 - max(0, power)
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != want || evt.Amount != power {
					t.Error("event observed an incomplete life action")
				}
				return true
			})
			g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
			clone := g.Clone()
			if err := ApplyEffect(clone, body, source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 20 - max(0, power)
			if clone.GetPlayer(a.PlayerID()).Life() != want || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("saved amount was not applied to both players")
			}
			wantEvents := 0
			if power > 0 {
				wantEvents = 2
			}
			if events != wantEvents {
				t.Fatalf("events = %d, want %d", events, wantEvents)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
		})
	}
}

func TestTypedCollectionLifeTargetAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	player := DeclarePlayerTarget("player")
	amount := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(LoseLifeEach(OpponentsOf(player), PowerOf(amount)), WithTargets(player, amount))
	card := NewSorcery("Life", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 20 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

type countedLifeAmount struct{ calls *int }

func (countedLifeAmount) Text() string { return "counted amount" }
func (v countedLifeAmount) resolveValue(ctx *EffectContext) int {
	*v.calls++
	return ctx.Game.GetPlayer(ctx.Controller).Life()
}

func TestTypedCollectionLifeAmountCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	amount := countedLifeAmount{&calls}
	if err := ApplyEffect(g, LoseLifeEach(OpponentsOf(ControllerOf(This)), amount), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("empty set evaluated its amount")
	}
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || a.Life() != 0 || b.Life() != 0 {
		t.Fatalf("calls=%d, life=%d/%d", calls, a.Life(), b.Life())
	}
}

func TestTypedCollectionLifeAmountValidation(t *testing.T) {
	saved := BindPermanent("missing")
	target := DeclarePermanentTarget("undeclared", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil amount",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeEach(EachPlayer, nil)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].amount",
		},
		{
			Name:          "power of nil",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeEach(EachPlayer, PowerOf(nil))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].amount",
		},
		{
			Name:          "power of missing binding",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeEach(EachPlayer, PowerOf(From(saved)))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].amount",
		},
		{
			Name:          "power of undeclared target",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeEach(EachPlayer, PowerOf(target))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].amount",
		},
		{
			Name:          "value at resolution nil",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeEach(EachPlayer, ValueAtResolution(nil))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].amount",
		},
	})
	info := InspectDefinition(LoseLifeEach(EachPlayer, PowerOf(This)))
	if info.Properties.Outcome != OutcomeDetriment || !info.Properties.Mass || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}

func TestTypedCollectionLifeCopiedAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	amount := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(LoseLifeEach(OpponentsOf(You), PowerOf(amount)), WithTargets(amount))
	card := NewSorcery("Life", "", WithAbility(ability))
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	clone := g.Clone()
	copied := clone.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	clone.ResolveStackObject(copied)
	if clone.GetPlayer(a.PlayerID()).Life() != 17 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("copy lost its controller or amount target")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("copy changed original game")
	}
	g.ResolveStackObject(original)
	if a.Life() != 20 || b.Life() != 17 {
		t.Fatal("original lost its controller or amount target")
	}
}

// --- from composable_life_gain_amount_test.go ---
func TestControllerLifeGainAmountCapturesFinalEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("gained")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerGainsLife(player).WithAmount(amount),
		LoseLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	g.gainLifeWithEvent(a, 2)
	g.gainLifeWithEvent(b, 2)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 20 {
			t.Fatalf("life = %d, want 20", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 25 {
			t.Fatalf("opponent life = %d, want 25", got)
		}
	}
}

func TestControllerLifeGainAmountKeepsEachTriggerAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("gainer"), BindNumber("gained")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ControllerGainsLife(player).WithAmount(amount),
		LoseLifeFor(You, FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.gainLifeWithEvent(a, 2)
	g.gainLifeWithEvent(a, 4)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	objects := g.StackObjects()
	g.CopyStackObjectDirect(objects[len(objects)-1], b.PlayerID(), false)
	g.ResolveTopOfStack()
	if got := b.Life(); got != 18 {
		t.Fatalf("copy controller life = %d, want 18", got)
	}
	g.ResolveTopOfStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("first trigger life = %d, want 24", got)
	}
	g.ResolveTopOfStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("second trigger life = %d, want 20", got)
	}
}

func TestControllerLifeGainAmountDeclaration(t *testing.T) {
	player, amount := BindPlayer("gainer"), BindNumber("gained")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(player).WithAmount(amount),
				LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
		},
		{
			Name:          "nil",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(player).WithAmount(nil), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(player).WithAmount(BindNumber("")), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(player).WithAmount(BindNumber("gainer")), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "repeated",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(player).WithAmount(amount).WithAmount(amount), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "one amount",
		},
		{
			Name:          "missing",
			Card:          NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(player), LoseLifeFor(FromPlayer(player), FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "scope validation",
			Card: NewEnchantment("Scope", "", WithAbility(WhenEvent(
				ControllerGainsLife(player).WithAmount(amount), LoseLifeFor(FromPlayer(player), FromNumber(amount)))),
				WithAbility(At(YourUpkeep, LoseLifeFor(You, FromNumber(amount))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	if got := InspectDefinition(ControllerGainsLife(player).WithAmount(amount)); got.Role != "event-pattern" || len(got.Children) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}

// --- from composable_life_loss_result_test.go ---
func TestLoseLifeEachActualResult(t *testing.T) {
	for _, amount := range []int{-1, 0, 4, 25} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		result := BindNumber("life lost")
		body := Sequence(
			LoseLifeEach(OpponentsOf(You), Fixed(amount), LifeLostInto(result)),
			GainLifeFor(You, FromNumber(result)),
		)
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := max(0, amount)
		if got := a.Life(); got != 20+want {
			t.Errorf("amount %d: gained %d, want %d", amount, got-20, want)
		}
		if got := b.Life(); got != 20-want {
			t.Errorf("amount %d: opponent lost %d, want %d", amount, 20-got, want)
		}
	}
}

func TestLoseLifeEachResultAbsentAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("life lost")
	g.resolution.numberBindings = map[*NumberBinding]int{result: 99}
	clone := g.Clone()
	if err := LoseLifeEach(OpponentsOf(FromPlayer(BindPlayer("absent"))), Fixed(4), LifeLostInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 0 || g.resolution.numberBindings[result] != 99 {
		t.Fatal("absent result changed the original frame or kept a stale value")
	}
}

func TestLoseLifeEachResultSumsSimultaneousLoss(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("life lost")
	if err := Sequence(
		LoseLifeEach(EachPlayer, Fixed(4), LifeLostInto(result)),
		GainLifeFor(You, FromNumber(result)),
	).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 24 || b.Life() != 16 || g.resolution.numberBindings[result] != 8 {
		t.Fatalf("life totals %d/%d, result %d; want 24/16, 8", a.Life(), b.Life(), g.resolution.numberBindings[result])
	}
}

func TestLoseLifeEachResultDefinition(t *testing.T) {
	result := BindNumber("life lost")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid declaration",
			Card: NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(
				LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(result)),
				GainLifeFor(You, FromNumber(result)),
			)))),
		},
		{
			Name:          "nil binding",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate option",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(result), LifeLostInto(result))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(result))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "late producer",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Sequence(GainLifeFor(You, FromNumber(result)), LoseLifeEach(OpponentsOf(You), Fixed(4), LifeLostInto(result)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_life_outcome_test.go ---
type lifeOutcomeReplacement struct {
	replacementBase
	amount  int
	consume bool
}

func (*lifeOutcomeReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*LifeGainAction)
	return ok
}
func (*lifeOutcomeReplacement) IsActive(GameReader) bool   { return true }
func (r *lifeOutcomeReplacement) Clone() ReplacementEffect { cloned := *r; return &cloned }
func (r *lifeOutcomeReplacement) Replace(a Action, _ *Game) Action {
	if r.consume {
		return nil
	}
	gain := a.(*LifeGainAction)
	return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), r.amount)
}

func TestLifeGainOutcomeEvents(t *testing.T) {
	for _, adapter := range []string{"typed", "legacy-target", "variable"} {
		for _, scenario := range []struct {
			name                                   string
			requested, modifier, replacement, want int
			replace, consume, lich                 bool
		}{
			{name: "normal", requested: 3, modifier: 1, want: 3},
			{name: "double", requested: 3, modifier: 2, want: 6},
			{name: "suppressed-modifier", requested: 3, modifier: 0},
			{name: "negative-modifier", requested: 3, modifier: -1},
			{name: "zero", requested: 0, modifier: 1},
			{name: "negative", requested: -2, modifier: 1},
			{name: "replacement-amount", requested: 3, modifier: 2, replace: true, replacement: 7, want: 7},
			{name: "replacement-zero", requested: 3, modifier: 1, replace: true},
			{name: "replacement-negative", requested: 3, modifier: 1, replace: true, replacement: -1},
			{name: "consumed", requested: 3, modifier: 1, replace: true, consume: true},
			{name: "lich", requested: 3, modifier: 1, lich: true},
		} {
			t.Run(adapter+"/"+scenario.name, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				watcher := NewEnchantment("Watcher", "", WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, ValueAtResolution(EventAmountValue())), false)))
				g.PutOnBattlefield(watcher, b.PlayerID())
				if scenario.replace {
					g.effects.AddReplacement(&lifeOutcomeReplacement{amount: scenario.replacement, consume: scenario.consume})
				}
				if scenario.lich {
					g.SetLichActive(b.PlayerID(), watcher.ID())
					for range 3 {
						b.AddToLibrary(NewLand("Card"))
					}
				}
				g.AddLifeGainModifier(watcher.ID(), func(_ *Game, _ uuid.UUID, amount int) int { return amount * scenario.modifier })
				clone := g.Clone()
				ctx := &EffectContext{Game: clone, SourceID: watcher.ID(), Vars: make(map[string]any), Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}}
				var effect Effect
				switch adapter {
				case "typed":
					effect = GainLifeFor(ControllerOf(This), Fixed(scenario.requested))
				case "legacy-target":
					effect = GainLifeTarget(Fixed(scenario.requested))
				case "variable":
					ctx.SetInt("amount", scenario.requested)
					ctx.SetUUID("player", b.PlayerID())
					effect = GainLifeFromVar("player", "amount")
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if got := clone.GetPlayer(b.PlayerID()).Life(); got != 20+scenario.want {
					t.Errorf("life = %d, want %d", got, 20+scenario.want)
				}
				if got := clone.PlayerLifeGainedThisTurn(b.PlayerID()); got != scenario.want {
					t.Errorf("event amount = %d, want %d", got, scenario.want)
				}
				clone.PutTriggersOnStack()
				wantTriggers := 0
				if scenario.want > 0 {
					wantTriggers = 1
				}
				if got := clone.stack.Size(); got != wantTriggers {
					t.Errorf("triggers = %d, want %d", got, wantTriggers)
				}
				if wantTriggers == 1 && clone.stack.Size() == 1 {
					clone.ResolveStackObject(clone.stack.Pop())
					if clone.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Error("trigger did not capture the actual gain amount")
					}
				}
				if scenario.lich && len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 {
					t.Error("replacement did not draw three cards")
				}
				if a.Life() != 20 || b.Life() != 20 || g.PlayerLifeGainedThisTurn(b.PlayerID()) != 0 {
					t.Error("clone changed original life or tracker")
				}
			})
		}
	}
}

// --- from composable_life_test.go ---
func TestComposableLifeDynamicMetadata(t *testing.T) {
	effect := GainLifeFor(You, ValueAtResolution(CountBattlefield(SelectController(), IsCreature)))
	info := InspectDefinition(effect)
	if info.Properties.LifeGain != 0 || !strings.Contains(effect.Text(), "creature") {
		t.Fatalf("dynamic life metadata = %+v, text = %q", info, effect.Text())
	}
	if got := GainLifeFor(You, Fixed(3)).Properties().LifeGain; got != 3 {
		t.Fatalf("fixed life gain = %d", got)
	}
}

func TestComposableLifeValueContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Value source", "{4}", 4, 4)
	g.PutOnBattlefield(creature, b.PlayerID())
	effect := GainLifeFor(You, ValueAtResolution(TargetPermanentManaValue()))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	if got := a.Life(); got != 24 {
		t.Fatalf("life = %d, want 24", got)
	}
}

func TestComposableLifeNonpositive(t *testing.T) {
	for _, amount := range []int{0, -2} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, Fixed(1)), false)))
		g.PutOnBattlefield(watcher, a.PlayerID())
		if err := ApplyEffect(g, Sequence(GainLifeFor(You, Fixed(amount)), LoseLifeFor(You, Fixed(amount))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.PutTriggersOnStack()
		if a.Life() != 20 || g.stack.Size() != 0 {
			t.Fatalf("amount %d: life = %d, stack = %d", amount, a.Life(), g.stack.Size())
		}
	}
}

func TestComposableLifeControllerAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(source, b.PlayerID())
	effect := Sequence(LoseLifeFor(You, ValueAtResolution(HalfRoundUp(PlayerLifeValue()))), GainLifeFor(You, Fixed(2)))
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		if err := ApplyEffect(game, effect, source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if game.GetPlayer(a.PlayerID()).Life() != 12 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("life instruction used the source controller or positional target")
		}
	}
	if a.Life() != 12 {
		t.Fatal("clone changed original life")
	}
}

func TestComposableLifeAbsentAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, effect := range []ResolvingEffect{GainLifeFor(ControllerOf(This), Fixed(3)), LoseLifeFor(ControllerOf(This), Fixed(3))} {
		if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("absent recipient fell back to controller")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil gain player",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(GainLifeFor(nil, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
		{
			Name:          "nil gain amount",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(GainLifeFor(You, nil)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
		{
			Name:          "nil lose player",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeFor(nil, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
		{
			Name:          "nil lose amount",
			Card:          NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeFor(You, nil)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
	})
}

func TestComposableLifeGainReplacement(t *testing.T) {
	for _, lich := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Watcher", "", 1, 1, WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, Fixed(1)), false)))
		g.PutOnBattlefield(source, a.PlayerID())
		if lich {
			g.SetLichActive(a.PlayerID(), source.ID())
			a.AddToLibrary(NewLand("One"))
			a.AddToLibrary(NewLand("Two"))
		}
		if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.PutTriggersOnStack()
		if lich {
			if a.Life() != 20 || len(a.Hand()) != 2 || g.stack.Size() != 0 {
				t.Fatal("replaced life gain must draw without a life-gain trigger")
			}
		} else {
			if a.Life() != 22 || g.stack.Size() != 1 {
				t.Fatal("life gain must emit one trigger")
			}
			g.ResolveStackObject(g.stack.Pop())
			if a.Life() != 21 {
				t.Fatal("triggered life loss did not resolve")
			}
		}
	}
}

// --- from composable_loop_test.go ---
func TestComposableLoopRestoresOuterFrameOnError(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Member", "", 1, 1)
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	member, saved := BindPermanent("member"), BindPermanent("outer")
	outer := capturedPermanent{card.ID(), permanent.incarnationID}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: outer}
	want := errors.New("body error")
	body := FuncEffect("fail", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		delete(g.resolution.permanentBindings, saved)
		return want
	})
	err := ForEach(MatchingPermanents(IsCreature), member, body).Apply(&EffectContext{Game: g, Controller: a.PlayerID()})
	if !errors.Is(err, want) || g.resolution.permanentBindings[saved] != outer || len(g.resolution.permanentBindings) != 1 {
		t.Fatal("loop did not restore the outer frame on error")
	}
}

func TestComposableLoopIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	member := BindPermanent("member")
	visited := 0
	effect := ForEach(MatchingPermanents(IsCreature), member, Sequence(
		FuncEffect("return other member", EffectProperties{}, func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
			visited++
			if visited == 1 {
				current := g.resolution.permanentBindings[member].id
				other := first
				if current == first.ID() {
					other = second
				}
				g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
				g.PutOnBattlefield(other, controller)
			}
			return nil
		}),
		Destroy(From(member)),
	))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if visited != 2 || len(g.FilterBattlefield(IsCreature)) != 1 {
		t.Fatal("loop followed a later incarnation or changed its member list")
	}
}

func TestComposableLoopScope(t *testing.T) {
	member, saved := BindPermanent("member"), BindPermanent("saved")
	loop := ForEach(MatchingPermanents(IsCreature), member, Sequence(Snapshot(From(member), saved), Destroy(From(saved))))
	card := func(name string, effect ResolvingEffect) Card {
		return NewCreature(name, "", 1, 1, WithAbility(Activate(Costs(), effect)))
	}
	outer := Sequence(Snapshot(This, saved), ForEach(MatchingPermanents(IsCreature), member, Destroy(From(saved))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid loop",
			Card: card("Loop", loop),
		},
		{
			Name:    "escaped member",
			Card:    card("Scope", Sequence(loop, Destroy(From(member)))),
			WantErr: true,
		},
		{
			Name:    "escaped saved",
			Card:    card("Scope", Sequence(loop, Destroy(From(saved)))),
			WantErr: true,
		},
		{
			Name: "valid outer",
			Card: card("Outer", outer),
		},
	})
}

// --- from composable_main_phase_test.go ---
func TestTypedMainPhaseTriggers(t *testing.T) {
	for _, first := range []bool{true, false} {
		pattern := YourFirstMainPhase
		if !first {
			pattern = YourPostcombatMainPhase
		}
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			trigger := At(pattern, GainLifeFor(You, Fixed(2)))
			if modal {
				trigger = AtModes(pattern, ModeOf("Two", GainLifeFor(You, Fixed(2))), ModeOf("Three", GainLifeFor(You, Fixed(3))))
			}
			source := NewEnchantment("Main phase", "", WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			for _, event := range []GameEvent{
				{Type: EvtMainPhase, PlayerID: b.PlayerID(), Flag: first},
				{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: !first},
				{Type: EvtUpkeep, PlayerID: a.PlayerID(), Flag: first},
			} {
				g.FireEvent(event)
			}
			if len(g.triggers.Pending()) != 0 {
				t.Fatal("unrelated phase matched")
			}
			g.FireEvent(GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: first})
			if len(g.triggers.Pending()) != 1 || a.Life() != 20 {
				t.Fatal("matching phase did not queue one trigger")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("main phase trigger did not use the stack")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatalf("life after source departure = %d", a.Life())
			}
		}
	}
}

func TestTypedDelayedMainPhase(t *testing.T) {
	for _, first := range []bool{true, false} {
		pattern := YourFirstMainPhase
		if !first {
			pattern = YourPostcombatMainPhase
		}
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		event := GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: first}
		g.FireEvent(event)
		if err := ApplyEffect(g, AtNext(pattern, GainLifeFor(You, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		for _, wrong := range []GameEvent{
			{Type: EvtMainPhase, PlayerID: b.PlayerID(), Flag: first},
			{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: !first},
		} {
			g.FireEvent(wrong)
		}
		if len(g.DelayedTriggers()) != 1 || len(g.triggers.Pending()) != 0 {
			t.Fatal("wrong main phase consumed delayed trigger")
		}
		clone := g.Clone()
		clone.FireEvent(event)
		if len(clone.triggers.Pending()) != 1 || clone.StackSize() != 0 {
			t.Fatal("delayed trigger bypassed pending placement")
		}
		clone.ResolveStack()
		clone.FireEvent(event)
		if clone.GetPlayer(a.PlayerID()).Life() != 22 || len(clone.triggers.Pending()) != 0 || len(clone.DelayedTriggers()) != 0 {
			t.Fatal("delayed trigger did not fire exactly once")
		}
		if a.Life() != 20 || len(g.DelayedTriggers()) != 1 {
			t.Fatal("clone consumed original trigger")
		}
		g.FireEvent(event)
		g.ResolveStack()
		if a.Life() != 22 {
			t.Fatal("original delayed trigger did not resolve")
		}
	}
}

// --- from composable_mana_amount_test.go ---
func TestTypedManaAmountActivation(t *testing.T) {
	for _, n := range []int{0, 3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		saved := BindPermanent("source")
		ability := Activate(Costs(TapCost(This)), Sequence(Snapshot(This, saved), AddManaAmount(Black, PowerOf(From(saved)))))
		source := NewCreature("Mana", "", n, 4, WithAbility(ability), WithKeyword(Haste))
		p := g.PutOnBattlefield(source, a.PlayerID())
		p.GrantBaseAttr(AttrCantActivateNonManaAbilities)
		events, amount := 0, 0
		watcher := NewTriggered(EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool { events++; amount = evt.Amount; return false })
		g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		if got := ManaProductionsForAbility(ability); got != nil {
			t.Fatalf("dynamic amount advertised fixed production: %v", got)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.StackSize() != 0 || a.ManaPool().Count(Black) != n {
			t.Fatal("mana ability did not resolve immediately with saved amount")
		}
		wantEvents := 0
		if n > 0 {
			wantEvents = 1
		}
		if events != wantEvents || amount != n {
			t.Fatalf("events=%d amount=%d", events, amount)
		}
	}
}

func TestTypedManaAmountMainPhaseCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	source := NewEnchantment("Phase mana", "", WithAbility(At(YourFirstMainPhase, AddManaAmount(Black, countedLifeAmount{&calls}))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: true})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || calls != 0 {
		t.Fatal("main-phase mana bypassed stack or evaluated early")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone := g.Clone()
	clone.ResolveStack()
	if calls != 2 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Black) != 20 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(Black) != 20 {
		t.Fatal("copied mana lost controller or evaluated amount twice")
	}
	if a.ManaPool().Count(Black) != 0 || b.ManaPool().Count(Black) != 0 {
		t.Fatal("clone changed original pools")
	}
}

func TestTypedManaAmountValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil amount",
			Card:    NewSorcery("Invalid mana", "", WithAbility(Spell(AddManaAmount(Black, nil)))),
			WantErr: true,
		},
		{
			Name:    "nil value at resolution",
			Card:    NewSorcery("Invalid mana", "", WithAbility(Spell(AddManaAmount(Black, ValueAtResolution(nil))))),
			WantErr: true,
		},
		{
			Name:    "missing number binding",
			Card:    NewSorcery("Invalid mana", "", WithAbility(Spell(AddManaAmount(Black, FromNumber(BindNumber("missing")))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewSorcery("Invalid mana", "", WithAbility(Spell(AddManaAmount(Black, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))))),
			WantErr: true,
		},
		{
			Name:    "any color",
			Card:    NewSorcery("Invalid color", "", WithAbility(Spell(AddManaAmount(AnyColor, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "negative color",
			Card:    NewSorcery("Invalid color", "", WithAbility(Spell(AddManaAmount(Color(-1), Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "out-of-range color",
			Card:    NewSorcery("Invalid color", "", WithAbility(Spell(AddManaAmount(Color(99), Fixed(1))))),
			WantErr: true,
		},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.ManaPool().Add(Black, 2)
	for _, n := range []int{-2, 0} {
		if err := ApplyEffect(g, AddManaAmount(Black, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.ManaPool().Count(Black) != 2 {
		t.Fatal("nonpositive production changed pool")
	}
}

func TestTypedManaAmountPlanning(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddManaAmount(Blue, Fixed(1)), AddManaAmount(Green, Fixed(2))))
	source := NewLand("Fixed mana", WithAbility(ability))
	p := g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 1, Green: 2}); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(p.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Green) != 2 || p.Tapped || a.ManaPool().Count(Green) != 0 {
		t.Fatal("fixed planning or clone isolation failed")
	}
	calls := 0
	dynamic := Activate(Costs(TapCost(This)), AddManaAmount(Black, countedLifeAmount{&calls}))
	other := NewLand("Dynamic mana", WithAbility(dynamic))
	q := g.PutOnBattlefield(other, a.PlayerID())
	if _, ok := manaSourceAbilityForPlanning(dynamic, q.ID(), 0, g); ok || calls != 0 {
		t.Fatal("planner evaluated or advertised an unknown amount")
	}
}

func TestTypedManaAmountNestedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	nested := &StackObject{ID: uuid.New(), Controller: b.PlayerID(), IsAbility: true, Effects: []Effect{AddManaAmount(Blue, Fixed(7))}}
	body := Sequence(AddManaAmount(Black, Fixed(2)), FuncEffect("resolve nested mana", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error { game.ResolveStackObject(nested); return nil }), AddManaAmount(Black, Fixed(3)))
	outer := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Effects: []Effect{body}}
	if got, colors := g.resolveStackObject(outer); got != 5 || len(colors) != 1 || colors[0] != Black {
		t.Fatalf("outer production=%d, colors=%v, want 5 black", got, colors)
	}
	if a.ManaPool().Count(Black) != 5 || b.ManaPool().Count(Blue) != 7 || g.resolution.manaProduced != 0 || len(g.resolution.manaColors) != 0 {
		t.Fatal("nested production escaped its resolution")
	}
}

func TestTypedManaAmountDirectResolutionHasNoTrackingState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, AddManaAmount(Black, Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Black) != 3 || g.resolution.manaProduced != 0 || g.resolution.manaTracking {
		t.Fatal("direct effect retained production tracking")
	}
}

func TestIsActivatedManaAbility(t *testing.T) {
	target := DeclarePlayerTarget("player")
	for _, tc := range []struct {
		name    string
		ability Ability
		want    bool
	}{
		{"legacy", NewManaAbility(Green), true},
		{"fixed", Activate(Costs(TapCost(This)), AddManaAmount(Green, Fixed(1))), true},
		{"dynamic", Activate(Costs(TapCost(This)), AddManaAmount(Green, PowerOf(This))), true},
		{"mixed", Activate(Costs(TapCost(This)), Sequence(AddManaAmount(Green, Fixed(1)), GainLifeFor(You, Fixed(1)))), true},
		{"targeted", Activate(Costs(TapCost(This)), AddManaAmount(Green, Fixed(1)), WithTargets(target)), false},
		{"non-mana", Activate(Costs(TapCost(This)), GainLifeFor(You, Fixed(1))), false},
		{"spell", Spell(AddManaAmount(Green, Fixed(1))), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsActivatedManaAbility(tc.ability); got != tc.want {
				t.Fatalf("classification = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- from composable_mana_test.go ---
func TestComposableManaProductionAndResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	addTappedForManaWatcher(g, a)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), Sequence(AddMana(Green, 2))))
	card := NewLand("Composed mana", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	got := ManaProductionsForAbility(ability)
	if len(got) != 2 || got[0].Color != Blue || got[0].Amount != 1 || got[1].Color != Green || got[1].Amount != 2 {
		t.Errorf("productions = %+v", got)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.GetStack().Size() != 0 || a.ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Green) != 2 {
		t.Fatal("composed mana ability did not resolve immediately")
	}
	resolvePendingTriggers(g)
	if a.Life() != 21 {
		t.Fatal("composed mana activation did not emit exactly one tapped-for-mana event")
	}
}

func TestComposableManaPlanningAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), AddMana(Green, 1)))
	card := NewLand("Composed mana", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	cost := ManaCost{Blue: 1, Green: 1}
	if !g.CanAfford(a.PlayerID(), cost, nil) {
		t.Fatal("planner did not find composed production")
	}
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), cost); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(permanent.ID()).Tapped || g.FindPermanent(permanent.ID()).Tapped || a.ManaPool().Count(Blue) != 0 {
		t.Fatal("planned activation leaked between clones")
	}
	cp := clone.GetPlayer(a.PlayerID())
	if cp.ManaPool().Count(Blue) != 1 || cp.ManaPool().Count(Green) != 1 {
		t.Fatal("planned output differs from metadata")
	}
}

func TestComposableManaTargetAndMixedEffects(t *testing.T) {
	target := DeclarePlayerTarget("player")
	targeted := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1)), WithTargets(target))
	if got := ManaProductionsForAbility(targeted); got != nil {
		t.Fatalf("targeted ability advertised mana: %+v", got)
	}
	mixed := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1), GainLifeFor(You, Fixed(2))))
	if got := ManaProductionsForAbility(mixed); got != nil {
		t.Fatalf("planner omitted a non-mana instruction: %+v", got)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Mixed mana", WithAbility(mixed))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || a.ManaPool().Count(Blue) != 1 || g.GetStack().Size() != 0 {
		t.Fatal("mixed mana ability did not resolve all instructions immediately")
	}
	targetedCard := NewLand("Targeted mana", WithAbility(targeted))
	targetedCard.SetOwner(a.PlayerID())
	targetedPermanent := g.PutOnBattlefield(targetedCard, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), targetedPermanent.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.GetStack().Size() != 1 || a.ManaPool().Count(Blue) != 1 {
		t.Fatal("targeted mana instruction bypassed the stack")
	}
	g.ResolveTopOfStack()
	if a.ManaPool().Count(Blue) != 2 {
		t.Fatal("targeted instruction did not resolve normally")
	}
}

func TestComposableManaPlanningUntapRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1)), OnlyDuring(Untap))
	card := NewLand("Restricted mana", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.turns.SetStep(Upkeep)
	if _, ok := manaSourceAbilityForPlanning(ability, p.ID(), 0, g); ok {
		t.Fatal("planner ignored the untap-step restriction")
	}
}

func TestComposableManaWithoutTapCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	addTappedForManaWatcher(g, a)
	ability := Activate(Costs(ManaPayment("{1}")), Sequence(AddMana(Blue, 2)))
	card := NewArtifact("Mana filter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.GetStack().Size() != 0 || a.ManaPool().Count(Blue) != 2 || a.ManaPool().Count(Colorless) != 0 || g.FindPermanent(p.ID()).Tapped {
		t.Fatal("mana ability without a tap cost did not pay and resolve immediately")
	}
	resolvePendingTriggers(g)
	if a.Life() != 20 {
		t.Fatal("untapped mana ability emitted a tapped-for-mana event")
	}
}

func TestComposableManaNonManaRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddMana(Blue, 1)))
	card := NewLand("Mana source", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	p.GrantBaseAttr(AttrCantActivateNonManaAbilities)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Blue) != 1 {
		t.Fatal("mana production was blocked by a non-mana restriction")
	}
}

func TestComposableManaChoiceGroups(t *testing.T) {
	for _, separate := range []bool{false, true} {
		t.Run(map[bool]string{false: "one color", true: "independent colors"}[separate], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := Sequence(AddAnyMana(2, AnyColor))
			if separate {
				effect = Sequence(AddAnyMana(1, AnyColor), Sequence(AddAnyMana(1, AnyColor)))
			}
			card := NewLand("Choice source", WithAbility(Activate(Costs(TapCost(This)), effect)))
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			cost := ManaCost{Blue: 1, Green: 1}
			if got := g.CanAfford(a.PlayerID(), cost, nil); got != separate {
				t.Fatalf("can pay two colors = %v, want %v", got, separate)
			}
			if separate {
				if err := g.AutoTapForCost(a.PlayerID(), cost); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Green) != 1 {
					t.Fatal("planned independent choices were not preserved")
				}
			}
		})
	}
}

func TestComposableManaCostFailureLeavesState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Costed mana", WithAbility(Activate(Costs(TapCost(This), ManaPayment("{2}")), Sequence(AddMana(Blue, 3)))))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation succeeded")
	}
	if g.FindPermanent(p.ID()).Tapped || a.ManaPool().Count(Colorless) != 1 || a.ManaPool().Count(Blue) != 0 || g.GetStack().Size() != 0 {
		t.Fatal("failed mana activation changed resources")
	}
}

func TestComposableManaBonuses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		amount int
		tap    bool
		want   int
	}{
		{"one mana", 1, true, 3},
		{"multiple mana", 3, true, 5},
		{"zero mana", 0, true, 0},
		{"no tap cost", 1, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			bonus := NewArtifact("Bonuses", "{0}", WithAbility(NewManaFlareAbility(IsLand)), WithAbility(NewManaBonusAbility(IsLand, Red)))
			bonus.SetOwner(b.PlayerID())
			g.PutOnBattlefield(bonus, b.PlayerID())
			cost := Costs(ManaPayment("{0}"))
			if tc.tap {
				cost = Costs(TapCost(This))
			}
			card := NewLand("Source", WithAbility(Activate(cost, AddManaAmount(Red, Fixed(tc.amount)))))
			card.SetOwner(a.PlayerID())
			perm := g.PutOnBattlefield(card, a.PlayerID())
			clone := g.Clone()
			if err := clone.ActivateAbilityByIndex(a.PlayerID(), perm.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if got := clone.GetPlayer(a.PlayerID()).ManaPool().Count(Red); got != tc.want {
				t.Fatalf("red mana = %d, want %d", got, tc.want)
			}
			if a.ManaPool().Count(Red) != 0 || g.FindPermanent(perm.ID()).Tapped {
				t.Fatal("activation changed original game")
			}
			if clone.GetStack().Size() != 0 {
				t.Fatal("mana ability used the stack")
			}
		})
	}
}

func TestComposableManaBonusUsesProducedColor(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit", true: "automatic"}[automatic], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			addTappedForManaWatcher(g, a)
			bonus := NewArtifact("Bonus", "{0}", WithAbility(NewManaFlareAbility(IsLand)))
			bonus.SetOwner(b.PlayerID())
			g.PutOnBattlefield(bonus, b.PlayerID())
			card := NewLand("Source", WithAbility(Activate(Costs(TapCost(This)), Sequence(AddManaAmount(Blue, Fixed(1)), AddManaAmount(Blue, Fixed(1))))))
			card.SetOwner(a.PlayerID())
			perm := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(Red, 4)
			var err error
			if automatic {
				err = g.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 3})
			} else {
				err = g.ActivateAbilityByIndex(a.PlayerID(), perm.ID(), 0, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(Blue) != 3 || a.ManaPool().Count(Red) != 4 {
				t.Fatal("bonus must apply once to the produced color")
			}
			resolvePendingTriggers(g)
			if a.Life() != 21 {
				t.Fatal("expected one tapped-for-mana event")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), perm.ID(), 0, nil); err == nil {
				t.Fatal("tapped source activated")
			}
			if a.ManaPool().Count(Blue) != 3 {
				t.Fatal("failed activation added mana")
			}
		})
	}
}

func TestComposableManaOrdinaryActivationChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Choices",
		WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))),
		WithAbility(Activate(Costs(TapCost(This)), GainLifeFor(You, Fixed(1)))),
		WithAbility(Activate(Costs(SacrificeSourceCost()), AddManaAmount(Red, Fixed(2)))),
	)
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	choices := g.GetActivatableAbilities(a.PlayerID())
	if len(choices) != 2 {
		t.Fatalf("ordinary activation choices = %d, want utility and sacrifice mana only", len(choices))
	}
}

// --- from composable_mill_test.go ---
func TestComposableMillContextAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")}
	a.SetLibrary(cards)
	creature := NewCreature("Amount", "{2}", 2, 2)
	g.PutOnBattlefield(creature, b.PlayerID())
	body := Mill(You, ValueAtResolution(TargetPermanentManaValue()))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	ca := clone.GetPlayer(a.PlayerID())
	if len(ca.Library()) != 1 || ca.Library()[0].ID() != cards[2].ID() || len(ca.Graveyard()) != 2 {
		t.Fatal("mill lost the outer value context or library order")
	}
	if len(a.Library()) != 3 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
		t.Fatal("mill changed another player or original game")
	}
}

func TestComposableMillAmountsAndModifiers(t *testing.T) {
	for _, amount := range []int{-1, 0, 1, 4} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLibrary([]Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")})
		source := NewArtifact("Modifier", "")
		g.PutOnBattlefield(source, b.PlayerID())
		g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), b.PlayerID()))
		if err := ApplyEffect(g, Mill(You, Fixed(amount)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := min(3, max(0, amount*2))
		if len(a.Graveyard()) != want || len(a.Library()) != 3-want {
			t.Fatalf("amount %d: graveyard=%d library=%d", amount, len(a.Graveyard()), len(a.Library()))
		}
	}
}

func TestComposableMillModifierBeforeClone(t *testing.T) {
	for _, amount := range []int{-1, 0, 1, 4} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		cards := []Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")}
		a.SetLibrary(cards)
		source := NewArtifact("Modifier", "")
		g.PutOnBattlefield(source, b.PlayerID())
		g.AddMillReplacement(source.ID(), DoubleMillForOpponents(source.ID(), b.PlayerID()))
		clone := g.Clone()
		if err := ApplyEffect(clone, Mill(You, Fixed(amount)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		player := clone.GetPlayer(a.PlayerID())
		want := min(3, max(0, amount*2))
		if len(player.Graveyard()) != want || len(player.Library()) != 3-want {
			t.Fatalf("amount %d: graveyard=%d library=%d", amount, len(player.Graveyard()), len(player.Library()))
		}
		for i, card := range player.Graveyard() {
			if card.ID() != cards[i].ID() {
				t.Fatal("clone milled the wrong card")
			}
		}
		if len(a.Library()) != 3 || len(a.Graveyard()) != 0 || len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
			t.Fatal("mill changed the original game or the other player")
		}
		if player.DrewFromEmpty() {
			t.Fatal("mill counted as a draw")
		}
	}
}

func TestComposableMillAbsenceAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLibrary([]Card{NewArtifact("Card", "")})
	if err := ApplyEffect(g, Mill(ControllerOf(This), Fixed(1)), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 {
		t.Fatal("absent player fell back to controller")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil player",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(Mill(nil, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(Mill(You, nil)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target in amount",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(Mill(You, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))))),
			WantErr: true,
		},
	})
	info := InspectDefinition(Mill(You, Fixed(2)))
	if len(info.Children) != 2 || info.Properties.Outcome != OutcomeDetriment {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestMillTargetPlayerUsesLibraryTop(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	top, middle, bottom := NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")
	a.SetLibrary([]Card{top, middle, bottom})
	if err := ApplyEffect(g, MillTargetPlayer(Fixed(2)), uuid.Nil, b.PlayerID(), []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 || a.Library()[0].ID() != bottom.ID() || len(a.Graveyard()) != 2 {
		t.Fatal("mill did not remove the top two cards")
	}
	if a.Graveyard()[0].ID() != top.ID() || a.Graveyard()[1].ID() != middle.ID() {
		t.Fatal("wrong milled cards")
	}
	if a.DrewFromEmpty() {
		t.Fatal("mill counted as a draw")
	}
}

func TestComposableMillSavedValueInLoop(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	for range 2 {
		g.PutOnBattlefield(NewCreature("Amount", "", 1, 1), a.PlayerID())
	}
	a.SetLibrary([]Card{NewArtifact("top", ""), NewArtifact("middle", ""), NewArtifact("bottom", "")})
	saved := BindPermanent("member")
	body := ForEach(ControlledPermanents(IsCreature), saved, Mill(You, PowerOf(From(saved))))
	card := NewSorcery("Loop", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 2 || len(a.Library()) != 1 || a.Library()[0].Name() != "bottom" {
		t.Fatal("mill lost the saved value in the loop frame")
	}
}

// --- from composable_modal_cast_test.go ---
func modalValidationSpell(typed bool) *BaseCard {
	var ability Ability
	if typed {
		red := DeclarePermanentTarget("red", HasColorFilter(Red))
		mountain := DeclarePermanentTarget("mountain", HasSubType("Mountain"))
		ability = SpellModes(ModeOf("Destroy red", Destroy(red), WithTargets(red)), ModeOf("Return Mountain", ReturnToHand(mountain), WithTargets(mountain)))
	} else {
		ability = NewModalSpell([]Mode{{Label: "Destroy red", Targets: []Target{TargetPermanent(HasColorFilter(Red))}, Effects: []Effect{DestroyTarget()}}, {Label: "Return Mountain", Targets: []Target{TargetPermanent(HasSubType("Mountain"))}, Effects: []Effect{ReturnToHandTarget()}}})
	}
	return NewInstant("Modal validation", "{1}", WithAbility(ability))
}

func TestModalCastRejectsTargetsBeforePayment(t *testing.T) {
	for _, typed := range []bool{false, true} {
		for _, alternate := range []bool{false, true} {
			for _, mode := range []int{0, 1} {
				for _, prompted := range []bool{false, true} {
					g, a, b := newModalGame()
					card := modalValidationSpell(typed)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(Colorless, 1)
					victim := NewCreature("Wrong target", "{U}", 2, 2)
					victim.SetOwner(b.PlayerID())
					g.PutOnBattlefield(victim, b.PlayerID())
					a.modes = []int{mode}
					targets := []uuid.UUID{victim.ID()}
					if prompted {
						a.targets = [][]uuid.UUID{targets}
						targets = nil
					}
					var err error
					if alternate {
						err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), ZoneHand, ManaCost{Generic: 1}, targets, 0)
					} else {
						err = g.CastSpellByName(a.PlayerID(), card.Name(), targets)
					}
					if err == nil {
						t.Errorf("typed=%v alternate=%v mode=%d prompted=%v: illegal cast accepted", typed, alternate, mode, prompted)
					}
					if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || !g.stack.IsEmpty() {
						t.Errorf("illegal cast changed resources or stack")
					}
				}
			}
		}
	}
}

func TestModalCastKeepsModeAndTargets(t *testing.T) {
	for _, typed := range []bool{false, true} {
		for _, mode := range []int{0, 1} {
			g, a, b := newModalGame()
			card := modalValidationSpell(typed)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(Colorless, 1)
			var victim Card = NewCreature("Red target", "{R}", 2, 2)
			if mode == 1 {
				victim = NewLand("Mountain", WithSubTypes("Mountain"))
			}
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			a.modes = []int{mode, 1 - mode}
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			cp := g.CopySpellOnStack(card.ID(), a.PlayerID(), false)
			if cp == nil || cp.ModeChoice != mode || len(cp.TargetSpecs) != 1 || len(cp.ModalTargets[mode]) != 1 {
				t.Fatal("copy lost selected mode or targets")
			}
			if len(a.modes) != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("mode prompted twice or payment incorrect")
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("selected effect did not resolve")
			}
			if mode == 1 && len(b.Hand()) != 1 {
				t.Fatal("return mode did not return target")
			}
			if mode == 0 && len(b.Graveyard()) != 1 {
				t.Fatal("destroy mode did not destroy target")
			}
		}
	}
}

func TestModalCastTargetFailuresLeaveManaSources(t *testing.T) {
	for _, scenario := range []string{"missing", "duplicate", "extra", "shroud", "phased", "invalid mode", "wrong land subtype"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newModalGame()
			land := addLand(t, g, a.PlayerID(), "Plains", White)
			card := modalValidationSpell(true)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := NewCreature("Red target", "{R}", 2, 2)
			victim.SetOwner(b.PlayerID())
			perm := g.PutOnBattlefield(victim, b.PlayerID())
			targets := make([]uuid.UUID, 1, 2)
			targets[0] = victim.ID()
			switch scenario {
			case "missing":
				g.PutPermanentIntoGraveyard(perm)
				targets = targets[:0]
			case "duplicate":
				targets = append(targets, victim.ID())
			case "extra":
				targets = append(targets, b.PlayerID())
			case "shroud":
				perm.GrantBaseAttr(Shroud)
			case "phased":
				g.PhaseOut(perm.ID())
			case "invalid mode":
				a.modes = []int{2}
			case "wrong land subtype":
				island := NewLand("Island", WithSubTypes("Island"))
				island.SetOwner(b.PlayerID())
				g.PutOnBattlefield(island, b.PlayerID())
				targets = []uuid.UUID{island.ID()}
				a.modes = []int{1}
			}
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), targets); err == nil {
				t.Fatal("illegal cast accepted")
			}
			if land.Tapped || a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 1 || !g.stack.IsEmpty() {
				t.Fatal("illegal cast changed resources")
			}
		})
	}
}

func TestModalCastTargetlessMode(t *testing.T) {
	for _, extra := range []bool{false, true} {
		g, a, b := newModalGame()
		target := DeclarePermanentTarget("creature", IsCreature)
		card := NewInstant("Targetless mode", "{1}", WithAbility(SpellModes(ModeOf("Destroy", Destroy(target), WithTargets(target)), ModeOf("Gain", GainLifeFor(You, Fixed(3))))))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(Colorless, 1)
		a.modes = []int{1}
		var targets []uuid.UUID
		if extra {
			targets = []uuid.UUID{b.PlayerID()}
		}
		err := g.CastSpellByName(a.PlayerID(), card.Name(), targets)
		if extra {
			if err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("targetless mode accepted target")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		before := a.Life()
		g.ResolveStack()
		if a.Life() != before+3 {
			t.Fatal("targetless mode did not resolve")
		}
	}
}

func TestModalCastVariableTargetPrompt(t *testing.T) {
	g, a, b := newModalGame()
	card := NewInstant("Variable mode", "{X}", WithAbility(NewModalSpell([]Mode{{Label: "Destroy X", Targets: []Target{TargetXCreatures()}, Effects: []Effect{DestroyTarget()}}, {Label: "Gain", Effects: []Effect{GainLifeFor(You, Fixed(1))}}})))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Colorless, 2)
	for range 2 {
		c := NewCreature("Target", "", 2, 2)
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil, 2); err != nil {
		t.Fatal(err)
	}
	obj := g.stack.Peek()
	if len(obj.Targets) != 2 || obj.XValue != 2 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("modal prompt did not use declared X")
	}
}

// --- from composable_modal_restriction_test.go ---
func TestModalActivationRestrictions(t *testing.T) {
	for _, granted := range []bool{false, true} {
		for _, scenario := range []string{"allowed", "wrong turn", "wrong step", "condition", "unpaid"} {
			t.Run(map[bool]string{false: "direct", true: "grant"}[granted]+"/"+scenario, func(t *testing.T) {
				g, a, b := newModalGame()
				options := []ModalActivationOption{
					ModeOf("one", GainLifeFor(You, Fixed(1))),
					OnlyAsSorcery(), ActivationIf(SourceHasCounters(P1P1, 1)), ActivationLimit(2),
					ModeOf("three", GainLifeFor(You, Fixed(3))),
				}
				recipient := NewArtifact("Recipient", "")
				donor := NewEnchantment("Donor", "")
				if granted {
					donor = NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), options...))))
				} else {
					recipient = NewArtifact("Recipient", "", WithAbility(ActivateModes(Costs(ManaPayment("{1}")), options...)))
				}
				g.PutOnBattlefield(donor, a.PlayerID())
				g.PutOnBattlefield(recipient, b.PlayerID())
				g.MutablePermanent(donor.ID()).AddCounter(P1P1, 1)
				if scenario != "condition" {
					g.MutablePermanent(recipient.ID()).AddCounter(P1P1, 1)
				}
				g.turns.SetActivePlayerIndex(1)
				g.turns.SetStep(PrecombatMain)
				if scenario == "wrong turn" {
					g.turns.SetActivePlayerIndex(0)
				}
				if scenario == "wrong step" {
					g.turns.SetStep(Upkeep)
				}
				if scenario != "unpaid" {
					b.ManaPool().Add(Colorless, 3)
				}
				g.ApplyContinuousEffects()
				b.modes = []int{1}
				err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
				if scenario != "allowed" {
					expectedMana := 3
					if scenario == "unpaid" {
						expectedMana = 0
					}
					if err == nil || g.StackSize() != 0 || b.ManaPool().TotalMana() != expectedMana {
						t.Fatal("invalid activation changed payment or stack", err)
					}
					g.turns.SetActivePlayerIndex(1)
					g.turns.SetStep(PrecombatMain)
					g.MutablePermanent(recipient.ID()).AddCounter(P1P1, 1)
					if scenario == "unpaid" {
						b.ManaPool().Add(Colorless, 3)
					}
					b.modes = []int{1}
					if err = g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				if g.Stack().Peek().ModeChoice != 1 {
					t.Fatal("options changed mode indices")
				}
				if g.CopyStackObjectDirect(g.Stack().Peek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				clone.turns.SetStep(EndStep)
				clone.MutablePermanent(recipient.ID()).RemoveCounter(P1P1, 1)
				clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
				clone.ApplyContinuousEffects()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 26 || b.Life() != 20 {
					t.Fatal("resolution rechecked restrictions or clone changed original")
				}
				g.ResolveStack()
				b.modes = []int{0}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
					t.Fatal("failed activation or copy consumed limit", err)
				}
				g.ResolveStack()
				g.ApplyContinuousEffects()
				if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 1 {
					t.Fatal("modes did not share activation limit", err)
				}
				if b.Life() != 27 {
					t.Fatal("wrong mode effects")
				}
				g.doCleanupActions()
				b.ManaPool().Add(Colorless, 1)
				b.modes = []int{0}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
					t.Fatal("turn retained limit", err)
				}
			})
		}
	}
}

func TestModalActivationRestrictionValidation(t *testing.T) {
	for _, optionSet := range [][]ModalActivationOption{
		{nil}, {OnlyAsSorcery(), OnlyDuring(Upkeep)}, {ActivationLimit(0)},
		{ActivationLimit(1), ActivationLimit(2)}, {ActivationIf(nil)}, {ActivationIf(AllConditions())},
	} {
		options := append([]ModalActivationOption{ModeOf("one", GainLifeFor(You, Fixed(1))), ModeOf("two", GainLifeFor(You, Fixed(2)))}, optionSet...)
		for _, granted := range []bool{false, true} {
			err := validateFactory("Invalid modal restrictions", func() Card {
				if granted {
					return NewEnchantment("Invalid modal restrictions", "", WithAbility(Static(GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(), options...))))
				}
				return NewArtifact("Invalid modal restrictions", "", WithAbility(ActivateModes(Costs(), options...)))
			})
			if err == nil || !strings.Contains(err.Error(), "Invalid modal restrictions") {
				t.Fatal("missing definition diagnostic", err)
			}
		}
	}
}

func TestModalActivationTimingOptions(t *testing.T) {
	for _, timing := range []struct {
		name   string
		option ModalActivationOption
		step   PhaseStep
	}{{"sorcery", OnlyAsSorcery(), PrecombatMain}, {"step", OnlyDuring(Upkeep), Upkeep}, {"own turn", OnlyOnYourTurn(), Upkeep}} {
		for _, scenario := range []string{"allowed", "wrong turn", "wrong step", "stack occupied"} {
			t.Run(timing.name+"/"+scenario, func(t *testing.T) {
				g, a, b := newModalGame()
				donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
					MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), timing.option,
					ModeOf("one", GainLifeFor(You, Fixed(1))), ModeOf("two", GainLifeFor(You, Fixed(2))),
				))))
				recipient := NewArtifact("Recipient", "")
				g.PutOnBattlefield(donor, a.PlayerID())
				g.PutOnBattlefield(recipient, b.PlayerID())
				g.ApplyContinuousEffects()
				g.turns.SetActivePlayerIndex(1)
				g.turns.SetStep(timing.step)
				if scenario == "wrong turn" {
					g.turns.SetActivePlayerIndex(0)
				}
				if scenario == "wrong step" {
					g.turns.SetStep(EndStep)
				}
				if scenario == "stack occupied" {
					filler := NewEnchantment("Filler", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
					g.PutOnBattlefield(filler, b.PlayerID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), filler.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				before := g.StackSize()
				b.ManaPool().Add(Colorless, 1)
				b.modes = []int{0}
				allowed := (scenario != "wrong turn" || timing.name == "step") && (scenario != "wrong step" || timing.name == "own turn") && (scenario != "stack occupied" || timing.name != "sorcery")
				err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
				if (err == nil) != allowed {
					t.Fatalf("allowed = %v, error = %v", allowed, err)
				}
				if !allowed && (g.StackSize() != before || b.ManaPool().TotalMana() != 1 || len(b.modes) != 1) {
					t.Fatal("restriction checked after choice or payment")
				}
			})
		}
	}
}

// --- from composable_no_regeneration_sweeper_test.go ---
func TestCollectionNoRegenerationPreservesCharacteristics(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Victim", "", 2, 2)
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	g.AddRegenerationShield(c.ID())
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("regeneration prevented destruction")
	}
	lki := g.LookupObject(c.ID())
	if lki == nil || lki.ViewHasKeyword(CantRegenerate) {
		t.Fatal("destruction changed the victim's last-known characteristics")
	}
}

func TestDestroyEachWithoutRegenerationCollections(t *testing.T) {
	for _, artifactsOnly := range []bool{false, true} {
		name := "creatures"
		filter := IsCreature
		if artifactsOnly {
			name = "artifacts"
			filter = IsArtifact
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := DestroyEachWithoutRegeneration(MatchingPermanents(filter))
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			var destroyed, survivors []*Permanent
			for _, owner := range []*BasePlayer{a, b} {
				for _, scenario := range []string{"ordinary", "indestructible", "regenerate", "phased", "excluded"} {
					card := NewCreature(scenario, "", 2, 2)
					if artifactsOnly {
						card = NewArtifact(scenario, "")
					}
					if scenario == "excluded" {
						card = NewEnchantment("excluded", "")
					}
					card.SetOwner(owner.PlayerID())
					p := g.PutOnBattlefield(card, owner.PlayerID())
					switch scenario {
					case "indestructible":
						p.GrantBaseAttr(Indestructible)
					case "regenerate":
						g.AddRegenerationShield(p.ID())
					case "phased":
						p.PhasedOut = true
					}
					if scenario == "ordinary" || scenario == "regenerate" {
						destroyed = append(destroyed, p)
					} else {
						survivors = append(survivors, p)
					}
				}
			}
			clone := g.Clone()
			if err := effect.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			for _, p := range destroyed {
				if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
					t.Fatalf("wrong destruction or clone isolation for %s", p.Name())
				}
			}
			for _, p := range survivors {
				got := clone.FindPermanentIncludingPhased(p.ID())
				if got == nil {
					t.Fatalf("destroyed %s", p.Name())
				}
			}
		})
	}
}

func TestNoRegenerationSimultaneousDeathObservation(t *testing.T) {
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
	clone.AddRegenerationShield(first.ID())
	clone.AddRegenerationShield(second.ID())
	if err := DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
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

// --- from composable_opponent_damage_history_test.go ---
func TestSourceDamagedOpponentThisTurnIdentityAndOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 4)
	old := g.PutOnBattlefield(card, a.PlayerID())
	condition := SourceDamagedOpponentThisTurnCondition()
	ctx := StateContext{SourceID: old.ID(), SourceIncarnation: old.incarnationID, ControllerID: a.PlayerID()}
	if condition.checkState(g, ctx) {
		t.Fatal("matched before damage")
	}
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, old.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("prevented damage entered history")
	}
	g.DealDamageToPlayer(a, 1, old.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("damage to controller matched opponent")
	}
	g.DealDamageToPlayer(b, 1, old.ID())
	if !condition.checkState(g, ctx) {
		t.Fatal("opponent damage was not recorded")
	}
	clone := g.Clone()
	clone.damage.ClearEndOfTurn()
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone shared turn history")
	}
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.zones.ClearLKI()
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: returned.ID(), SourceIncarnation: returned.incarnationID, ControllerID: a.PlayerID()}) {
		t.Fatal("returned source inherited old history")
	}
	g.DealDamageToPlayer(b, 1, returned.ID())
	newCtx := StateContext{SourceID: returned.ID(), SourceIncarnation: returned.incarnationID, ControllerID: a.PlayerID()}
	if !condition.checkState(g, newCtx) {
		t.Fatal("new incarnation did not record damage")
	}
	returned.PhasedOut = true
	if condition.checkState(g, newCtx) {
		t.Fatal("phased source matched")
	}
	returned.PhasedOut = false
	if !condition.checkState(g, newCtx) {
		t.Fatal("phasing erased source identity")
	}
	g.damage.ClearEndOfTurn()
	if condition.checkState(g, newCtx) {
		t.Fatal("history survived turn cleanup")
	}
}

func TestSourceDamagedOpponentThisTurnInterveningIf(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := SourceDamagedOpponentThisTurnCondition()
	card := NewCreature("Watcher", "", 1, 4, WithAbility(At(EachEndStep, GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Invalid", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" {
		t.Fatalf("inspection = %+v", meta)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("triggered before damage")
	}
	g.DealDamageToPlayer(b, 1, card.ID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("did not trigger after damage")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != a.Life()+1 {
		t.Fatal("clone trigger did not resolve independently")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("returned source passed resolution recheck")
	}
}

func TestSourceDamagedOpponentThisTurnBatchAndCleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 4)
	permanent := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(permanent.ID())
	condition := SourceDamagedOpponentThisTurnCondition()
	ctx := StateContext{SourceID: permanent.ID(), SourceIncarnation: permanent.incarnationID, ControllerID: a.PlayerID()}
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealSimultaneousDamage(
		PlayerDamageAssignment(a, 1, source),
		PlayerDamageAssignment(b, 1, source),
	)
	if condition.checkState(g, ctx) {
		t.Fatal("fully prevented batch recipient entered history")
	}
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 1, source))
	if !condition.checkState(g, ctx) {
		t.Fatal("batch damage did not enter turn history")
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	if !g.doCleanupActions() || !condition.checkState(g, ctx) {
		t.Fatal("cleanup priority removed history early")
	}
	g.ResolveStack()
	g.RunStepWithPriority(Cleanup)
	if condition.checkState(g, ctx) {
		t.Fatal("final cleanup kept turn history")
	}
}

// --- from composable_opponent_life_test.go ---
func TestComposableOpponentLifeSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	observed := 0
	watcher := NewTriggered(EvtLifeLost, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, game GameReader, _, _ uuid.UUID) bool {
		observed++
		if game.GetPlayer(a.PlayerID()).Life() != 0 || game.GetPlayer(b.PlayerID()).Life() != 0 || evt.Amount != 20 {
			t.Error("life-loss event observed an incomplete action")
		}
		return true
	})
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, ValueAtResolution(PlayerLifeValue())), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if observed != 2 || g.stack.Size() != 2 {
		t.Fatalf("events = %d, triggers = %d", observed, g.stack.Size())
	}
}

func TestComposableOpponentLife(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		controller, opponent := a, b
		if reverse {
			controller, opponent = b, a
		}
		source := NewCreature("Source", "", 1, 1)
		g.PutOnBattlefield(source, opponent.PlayerID())
		effect := Sequence(LoseLifeEach(OpponentsOf(You), Fixed(3)), GainLifeFor(You, Fixed(3)))
		clone := g.Clone()
		if err := ApplyEffect(clone, effect, source.ID(), controller.PlayerID(), []uuid.UUID{controller.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if clone.GetPlayer(controller.PlayerID()).Life() != 23 || clone.GetPlayer(opponent.PlayerID()).Life() != 17 {
			t.Fatal("life action used the active player, source controller, or positional target")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("clone changed original life")
		}
	}
}

func TestComposableOpponentLifeAmountAndOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := NewCreature("Amount", "{3}", 1, 1)
	g.PutOnBattlefield(target, b.PlayerID())
	effect := Sequence(LoseLifeEach(OpponentsOf(You), ValueAtResolution(TargetPermanentManaValue())), GainLifeFor(You, ValueAtResolution(PlayerLifeValue())))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 17 || a.Life() != 40 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
	if err := ApplyEffect(g, Sequence(LoseLifeEach(EachPlayer, Fixed(3)), GainLifeFor(You, ValueAtResolution(PlayerLifeValue()))), uuid.Nil, a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 14 || a.Life() != 74 {
		t.Fatalf("loss must precede gain: life = %d/%d", a.Life(), b.Life())
	}
}

func TestComposableOpponentLifeAbsenceAndNonpositive(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, effect := range []ResolvingEffect{
		LoseLifeEach(OpponentsOf(ControllerOf(This)), Fixed(3)),
		LoseLifeEach(OpponentsOf(You), Fixed(0)),
		LoseLifeEach(OpponentsOf(You), Fixed(-3)),
	} {
		if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("absent or nonpositive instruction changed life")
	}
}

func TestComposableOpponentLifeReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Lich", "", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	g.SetLichActive(a.PlayerID(), source.ID())
	for range 3 {
		a.AddToLibrary(NewLand("Card"))
	}
	if err := ApplyEffect(g, Sequence(LoseLifeEach(OpponentsOf(You), Fixed(3)), GainLifeFor(You, Fixed(3))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 17 || len(a.Hand()) != 3 {
		t.Fatal("life gain replacement did not preserve opponent life loss")
	}
}

func TestComposableOpponentLifeValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil set",
			Card:          NewSorcery("Invalid opponents", "", WithAbility(Spell(LoseLifeEach(nil, Fixed(3))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
		{
			Name:          "opponents of nil",
			Card:          NewSorcery("Invalid opponents", "", WithAbility(Spell(LoseLifeEach(OpponentsOf(nil), Fixed(3))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
		{
			Name:          "nil amount",
			Card:          NewSorcery("Invalid opponents", "", WithAbility(Spell(LoseLifeEach(OpponentsOf(You), nil)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].",
		},
	})
	info := InspectDefinition(LoseLifeEach(OpponentsOf(You), Fixed(3)))
	if info.Properties.Outcome != OutcomeDetriment || !info.Properties.Mass || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}

// --- from composable_optional_reveal_test.go ---
func TestOptionalRevealPaymentInstances(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalRevealFromHandCost(IsCreatureCard, "Reveal a creature", "Reveal?", paid, revealed)),
		Sequence(GainLifeFor(You, FromNumber(paid)),
			IfCardHasType(FromCard(revealed), TypeCreature, GainLifeFor(You, Fixed(2)))))))
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewCreature("Creature", "", 1, 1)
	a.AddToHand(card)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.accept = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := len(g.PublicReveals()); got != 1 {
		t.Fatalf("public reveals = %d", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatalf("optional reveal values escaped their stack or game instances: clone A=%d B=%d live A=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("optional reveal values changed after clone resolution")
	}
}

func TestOptionalRevealPaymentVisitAndRollback(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	card := NewCreature("Creature", "", 1, 1)
	a.AddToHand(card)
	cost := OptionalRevealFromHandCost(IsCreatureCard, "Reveal a creature", "Reveal?", paid, revealed)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("card did not leave hand")
	}
	a.AddToHand(card)
	if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 || len(g.PublicReveals()) != 0 {
		t.Fatalf("stale optional reveal committed: %v", err)
	}
	if a.Life() != 20 {
		t.Fatal("failed payment changed life")
	}
	failed, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{cost, LifePayCost(25)}})
	if err == nil || failed != nil || len(g.PublicReveals()) != 0 {
		t.Fatalf("failed total payment changed live state: %v", err)
	}
}

func TestOptionalRevealSpellAndMissingCard(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	makeSpell := func(name string) Card {
		return NewInstant(name, "",
			WithAbility(Spell(Sequence(
				GainLifeFor(You, FromNumber(paid)),
				IfCardHasType(FromCard(revealed), TypeCreature, GainLifeFor(You, Fixed(2)))))),
			WithAdditionalCost(OptionalRevealFromHandCost(IsCreatureCard, "Reveal a creature", "Reveal?", paid, revealed)))
	}
	first := makeSpell("First")
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	if err := ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), first.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || len(g.PublicReveals()) != 0 {
		t.Fatal("missing matching card paid an optional reveal")
	}
	second := makeSpell("Second")
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	a.AddToHand(NewCreature("Creature", "", 1, 1))
	if err := g.CastSpellByName(a.PlayerID(), second.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 23 || len(g.PublicReveals()) != 1 {
		t.Fatalf("spell reveal result: life=%d reveals=%d", a.Life(), len(g.PublicReveals()))
	}
}

func TestOptionalRevealPaymentDeclaration(t *testing.T) {
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Test", "", WithAbility(Activate(Costs(OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, revealed)), GainLifeFor(You, Fixed(1))))),
		},
		{
			Name:          "nil paid",
			Card:          NewArtifact("Test", "", WithAbility(Activate(Costs(OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", nil, revealed)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil revealed",
			Card:          NewArtifact("Test", "", WithAbility(Activate(Costs(OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, nil)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty text",
			Card:          NewArtifact("Test", "", WithAbility(Activate(Costs(OptionalRevealFromHandCost(IsCreatureCard, "", "Reveal?", paid, revealed)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "text",
		},
		{
			Name:          "empty prompt",
			Card:          NewArtifact("Test", "", WithAbility(Activate(Costs(OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "", paid, revealed)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "prompt",
		},
		{
			Name: "duplicate binding",
			Card: NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
				OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, revealed),
				OptionalRevealFromHandCost(IsCreatureCard, "Reveal", "Reveal?", paid, BindCard("second"))), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
	})
}

// --- from composable_owner_test.go ---
func TestOwnerOfPermanentIdentityAndSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Owned", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	owner := OwnerOf(This)
	if id, ok := owner.resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("owner followed controller")
	}
	saved := BindPlayer("owner")
	if err := SnapshotPlayer(owner, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(p.ID())
	if _, ok := owner.resolvePlayer(ctx); ok {
		t.Fatal("phased permanent supplied owner")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if id, ok := owner.resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("phase-in lost owner")
	}
	clone := g.Clone()
	cloneCtx := &EffectContext{Game: clone, SourceID: p.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	clone.PutOnBattlefield(card, b.PlayerID())
	if _, ok := owner.resolvePlayer(cloneCtx); ok {
		t.Fatal("returned source reused old incarnation")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("snapshot changed with clone")
	}
	if err := SnapshotPlayer(owner, saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("absent owner retained snapshot in clone")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("clone changed original snapshot")
	}
}

func TestOwnerOfTargetAndRegistration(t *testing.T) {
	target := DeclarePermanentTarget("permanent", AnyPermanent)
	saved := BindPlayer("owner")
	ability := Spell(
		Sequence(SnapshotPlayer(OwnerOf(target), saved), GainLifeFor(FromPlayer(saved), Fixed(1))),
		WithTargets(target))
	valid := NewInstant("Owner", "", WithAbility(ability))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		subject PermanentRef
		want    string
	}{
		{"nil", nil, "non-nil"},
		{"undeclared", target, "target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(OwnerOf(tc.subject), Fixed(1)))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation = %v, want %s", err, tc.want)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Target", "", 1, 1)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: valid.ID(), Card: valid, Controller: b.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	phased := g.Clone()
	phased.PhaseOut(p.ID())
	phased.ResolveStack()
	if phased.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("phased target supplied owner")
	}
	returned := g.Clone()
	returned.RemoveFromBattlefield(returned.FindPermanent(p.ID()))
	returned.PutOnBattlefield(card, b.PlayerID())
	returned.ResolveStack()
	if returned.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("returned target supplied owner")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("target owner did not gain life")
	}
}

// --- from composable_permanent_tapped_test.go ---
func TestPermanentTappedAttachmentState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := NewCreature("Host", "", 2, 2)
	other := NewCreature("Other", "", 2, 2)
	aura := NewAura("Aura", "")
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	source := g.FindPermanent(aura.ID())
	ctx := StateContext{SourceID: aura.ID(), ControllerID: a.PlayerID(), SourceIncarnation: source.incarnationID}
	untapped := PermanentIsUntapped(AttachedPermanent)
	if untapped.checkState(g, ctx) {
		t.Fatal("unattached source matched")
	}
	g.Attach(aura.ID(), host.ID())
	if !untapped.checkState(g, ctx) || PermanentIsTapped(AttachedPermanent).checkState(g, ctx) {
		t.Fatal("untapped host did not match")
	}
	clone := g.Clone()
	clone.MutablePermanent(host.ID()).Tapped = true
	if !PermanentIsTapped(AttachedPermanent).checkState(clone, ctx) || !untapped.checkState(g, ctx) {
		t.Fatal("clone shares host tapped state")
	}
	g.MutablePermanent(host.ID()).Tapped = true
	if !PermanentIsTapped(AttachedPermanent).checkState(g, ctx) || untapped.checkState(g, ctx) {
		t.Fatal("tapped host did not match")
	}
	g.Attach(aura.ID(), other.ID())
	if !untapped.checkState(g, ctx) {
		t.Fatal("condition did not follow reattachment")
	}
	g.MutablePermanent(other.ID()).PhasedOut = true
	if untapped.checkState(g, ctx) || PermanentIsTapped(AttachedPermanent).checkState(g, ctx) {
		t.Fatal("phased host matched")
	}
	g.MutablePermanentIncludingPhased(other.ID()).PhasedOut = false
	if !untapped.checkState(g, ctx) {
		t.Fatal("phased-in host lost identity")
	}
	g.PhaseOut(aura.ID())
	if untapped.checkState(g, ctx) || PermanentIsTapped(AttachedPermanent).checkState(g, ctx) {
		t.Fatal("phased source matched")
	}
	g.PhaseIn([]uuid.UUID{aura.ID()})
	if !untapped.checkState(g, ctx) {
		t.Fatal("phased-in source lost identity")
	}
	g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	g.PutOnBattlefield(other, b.PlayerID())
	if untapped.checkState(g, ctx) {
		t.Fatal("returned host matched old attachment")
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), other.ID())
	if untapped.checkState(g, ctx) {
		t.Fatal("returned source matched old incarnation")
	}
}

func TestPermanentTappedAttachmentStaticAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	aura := NewAura("Aura", "", WithAbility(Static(While(PermanentIsUntapped(AttachedPermanent), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition("Aura", aura); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCardDefinition("Source", NewCreature("Source", "", 1, 1,
		WithAbility(WhenState(PermanentIsTapped(This), GainLifeFor(You, Fixed(1)))))); err != nil {
		t.Fatal(err)
	}
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(host, b.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("untapped host did not gain shroud")
	}
	g.MutablePermanent(host.ID()).Tapped = true
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).HasKeyword(Shroud) {
		t.Fatal("tapped host retained shroud")
	}
	target := DeclarePermanentTarget("target", IsCreature)
	invalid := NewAura("Invalid", "", WithAbility(Static(While(PermanentIsTapped(target), GrantKeywordModification(AttachedPermanent, Shroud)))))
	if err := ValidateCardDefinition("Invalid", invalid); err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("unsupported state reference accepted: %v", err)
	}
}

// --- from composable_phased_damage_cleanup_test.go ---
func TestDamageCleanupPhasedPermanentClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Phased survivor", "", 2, 3)
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, victim.ID())
	phased := g.PhaseOut(victim.ID())
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	if p := clone.FindPermanentIncludingPhased(victim.ID()); p == nil || !p.PhasedOut || p.Damage != 0 {
		t.Fatal("cleanup must remove damage and preserve phased-out status")
	}
	if p := g.FindPermanentIncludingPhased(victim.ID()); p == nil || !p.PhasedOut || p.Damage != 2 {
		t.Fatal("clone cleanup changed the original permanent")
	}
	clone.PhaseIn(phased)
	clone.DealDamageToPermanent(clone.FindPermanent(victim.ID()), 1, victim.ID())
	clone.checkStateBasedActions()
	if p := clone.FindPermanent(victim.ID()); p == nil || p.Damage != 1 {
		t.Fatal("damage from before cleanup must not kill the phased-in creature")
	}
	g.RunStepWithPriority(Cleanup)
	if g.FindPermanentIncludingPhased(victim.ID()).Damage != 0 || clone.FindPermanent(victim.ID()).Damage != 1 {
		t.Fatal("original cleanup did not remove damage independently")
	}
}

func TestDamageCleanupPhasedPermanentPriorityRounds(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Phased survivor", "", 2, 3)
	g.PutOnBattlefield(victim, b.PlayerID())
	for range 2 {
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, victim.ID())
		phased := g.PhaseOut(victim.ID())
		g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
		if !g.doCleanupActions() {
			t.Fatal("cleanup did not request priority")
		}
		if p := g.FindPermanentIncludingPhased(victim.ID()); p == nil || !p.PhasedOut || p.Damage != 0 {
			t.Fatal("each cleanup pass must remove damage before priority")
		}
		if !g.GetDamageSources(victim.ID())[victim.ID()] {
			t.Fatal("damage removal cleared source history before final cleanup")
		}
		g.ResolveStack()
		g.PhaseIn(phased)
	}
	g.RunStepWithPriority(Cleanup)
	if len(g.GetDamageSources(victim.ID())) != 0 || a.Life() != 22 {
		t.Fatal("final cleanup must clear history after both cleanup triggers resolve")
	}
}

// --- from composable_player_damage_source_test.go ---
func TestPlayerDamageSourceCaptureLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	source := BindDamageSource("source")
	watcher := NewCreature("Watcher", "", 1, 3, WithKeyword(Lifelink), WithAbility(WhenEvent(
		DealsDamageToPlayer(This, player).WithSource(source),
		DealDamageFrom(FromDamageSource(source), FromPlayer(player), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	g.DealDamageToPlayer(b, 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage made a trigger")
	}
	g.DealDamageToPlayer(b, 1, watcher.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	old := g.FindPermanent(watcher.ID())
	g.RemoveFromBattlefield(old)
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PhaseOut(watcher.ID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 17 {
			t.Fatalf("source capture lost: life = %d", game.GetPlayer(b.PlayerID()).Life())
		}
		if game.GetPlayer(a.PlayerID()).Life() != 23 {
			t.Fatalf("old source controller lost lifelink: life = %d", game.GetPlayer(a.PlayerID()).Life())
		}
	}
}

func TestPlayerDamageSourceDeclaration(t *testing.T) {
	player := BindPlayer("damaged")
	source := BindDamageSource("source")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToPlayer(This, player).WithSource(source),
				DealDamageFrom(FromDamageSource(source), FromPlayer(player), Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(DealsDamageToPlayer(This, player).WithSource(source)); got.Role != "event-pattern" || len(got.Children) != 2 {
					t.Fatalf("inspection = %#v", got)
				}
			},
		},
		{
			Name:    "nil source",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSource(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty source name",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSource(BindDamageSource("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate source",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, player).WithSource(source).WithSource(source), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name: "unbound source read",
			Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToPlayer(This, player), DealDamageFrom(FromDamageSource(source), You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_receives_damage_test.go ---
func TestReceivesDamageCapturesActualAmountAndSourceController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	victim := NewCreature("Victim", "", 2, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	g.PutOnBattlefield(victim, a.PlayerID())
	source := NewCreature("Source", "", 2, 2)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 1)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 3, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 22 {
			t.Fatalf("source controller life = %d", got)
		}
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 20 {
			t.Fatalf("other life = %d", got)
		}
	}
}

func TestReceivesDamageAbsentPhasedAndPrevented(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	victim := NewCreature("Victim", "", 2, 5, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	g.PutOnBattlefield(victim, a.PlayerID())
	source := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddPreventionShield(victim.ID(), 2)
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("prevented damage triggered")
	}
	permanent := g.FindPermanent(victim.ID())
	permanent.PhasedOut = true
	g.DealDamageToPermanent(permanent, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased victim triggered")
	}
	permanent.PhasedOut = false
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
	g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
	g.PutOnBattlefield(victim, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("departed victim triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if got := b.Life(); got != 21 {
		t.Fatalf("source controller life = %d", got)
	}
}

func TestReceivesDamageDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Test", "", 2, 5, WithAbility(WhenEvent(ReceivesDamage(This, BindNumber("amount")).WithSourceController(BindPlayer("controller")), GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(ReceivesDamage(This, BindNumber("amount")).WithSourceController(BindPlayer("controller"))); got.Role != "event-pattern" || len(got.Children) != 2 {
					t.Fatalf("inspection = %#v", got)
				}
			},
		},
		{
			Name:          "nil amount",
			Card:          NewCreature("Test", "", 2, 5, WithAbility(WhenEvent(ReceivesDamage(This, nil).WithSourceController(BindPlayer("controller")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty amount",
			Card:          NewCreature("Test", "", 2, 5, WithAbility(WhenEvent(ReceivesDamage(This, BindNumber("")).WithSourceController(BindPlayer("controller")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "nil controller",
			Card:          NewCreature("Test", "", 2, 5, WithAbility(WhenEvent(ReceivesDamage(This, BindNumber("amount")).WithSourceController(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty controller",
			Card:          NewCreature("Test", "", 2, 5, WithAbility(WhenEvent(ReceivesDamage(This, BindNumber("amount")).WithSourceController(BindPlayer("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "bad zone",
			Card:          NewInstant("Instant", "", WithAbility(WhenEvent(ReceivesDamage(This, BindNumber("amount")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
		{
			Name:          "unbound",
			Card:          NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(Enters(This), GainLifeFor(FromPlayer(BindPlayer("controller")), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestReceivesDamageSimultaneousSources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount, controller := BindNumber("amount"), BindPlayer("controller")
	victim := NewCreature("Victim", "", 1, 8, WithAbility(WhenEvent(
		ReceivesDamage(This, amount).WithSourceController(controller),
		GainLifeFor(FromPlayer(controller), FromNumber(amount)))))
	g.PutOnBattlefield(victim, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	second := NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.DealSimultaneousDamage(
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 2, g.CaptureDamageSource(first.ID())),
		PermanentDamageAssignment(g.FindPermanent(victim.ID()), 3, g.CaptureDamageSource(second.ID())),
	)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if g.CopyStackObjectDirect(obj, uuid.Nil, false) == nil {
			t.Fatal("copy failed")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 24 || game.GetPlayer(b.PlayerID()).Life() != 26 {
			t.Fatalf("captured controllers and amounts = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

// --- from composable_regeneration_ai_test.go ---
func TestRegeneratesAttachment(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(*PermanentBinding, *PermanentBinding) *ActionDefinition
		want  bool
	}{
		{"direct", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(AttachedPermanent))
		}, true},
		{"source", func(a, b *PermanentBinding) *ActionDefinition { return Activate(Costs(), Regenerate(This)) }, false},
		{"snapshot", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Snapshot(AttachedPermanent, a), Regenerate(From(a))))
		}, true},
		{"capture chain", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(From(b)), CapturePermanentBeforePayment(AttachedPermanent, a), CapturePermanentBeforePayment(From(a), b))
		}, true},
		{"capture overwrite", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(From(a)), CapturePermanentBeforePayment(AttachedPermanent, a), CapturePermanentBeforePayment(This, a))
		}, false},
		{"snapshot overwrite", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Snapshot(This, a), Regenerate(From(a))), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
		{"snapshot replaces capture", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Snapshot(AttachedPermanent, a), Regenerate(From(a))), CapturePermanentBeforePayment(This, a))
		}, true},
		{"same name different binding", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(From(b)), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
		{"nested sequence writes persist", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Sequence(Snapshot(This, a)), Regenerate(From(a))), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
		{"nested sequence", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Sequence(Snapshot(AttachedPermanent, a)), Regenerate(From(a))))
		}, true},
		{"unknown instruction", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Destroy(AttachedPermanent), Regenerate(From(a))), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ability := tc.build(BindPermanent("host"), BindPermanent("host"))
			if got := RegeneratesAttachment(ability); got != tc.want {
				t.Fatalf("RegeneratesAttachment = %v, want %v", got, tc.want)
			}
			if got := RegeneratesAttachment(ability); got != tc.want {
				t.Fatalf("repeated inspection = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- from composable_regeneration_test.go ---
func TestTypedRegenerationConsumption(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 3, 4)
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	p.Damage = 2
	g.combat.AddAttacker(c.ID(), b.PlayerID())
	body := Sequence(Regenerate(This), Regenerate(This))
	if !IsRegenerationEffect(body) {
		t.Fatal("missing regeneration metadata")
	}
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.Tapped || p.Damage != 2 {
		t.Fatal("installation regenerated immediately")
	}
	clone := g.Clone()
	for range 2 {
		clone.DestroyPermanent(clone.FindPermanent(c.ID()))
		survivor := clone.FindPermanent(c.ID())
		if survivor == nil || !survivor.Tapped || survivor.Damage != 0 || clone.combat.IsAttacking(c.ID()) {
			t.Fatal("shield did not replace destruction")
		}
	}
	clone.DestroyPermanent(clone.FindPermanent(c.ID()))
	if clone.FindPermanent(c.ID()) != nil {
		t.Fatal("consumed shields still active")
	}
	if !g.HasRegenerationShield(c.ID()) || p.Tapped || p.Damage != 2 {
		t.Fatal("clone changed original")
	}
	g.effects.ClearReplacementsEndOfTurn()
	g.DestroyPermanent(p)
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("shield did not expire")
	}
}

func TestTypedRegenerationSavedLifetime(t *testing.T) {
	for _, change := range []string{"same", "return", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewArtifact("Subject", "")
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			if err := Regenerate(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{c.ID()})
			}
			want := change != "return" && change != "phase"
			if g.HasRegenerationShield(c.ID()) != want {
				t.Fatal("incorrect saved subject lifetime")
			}
			if want {
				g.PhaseIn(g.PhaseOut(c.ID()))
				g.DestroyPermanent(g.FindPermanent(c.ID()))
				if g.FindPermanent(c.ID()) == nil {
					t.Fatal("phasing lost installed shield")
				}
			}
		})
	}
}

func TestTypedRegenerationCopyTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	target := DeclarePermanentTarget("subject", IsCreature)
	ability := Spell(Regenerate(target), WithTargets(target))
	card := NewInstant("Shield", "", WithAbility(ability))
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != second.ID() {
		t.Fatal("copy did not retarget")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ResolveStack()
	if g.HasRegenerationShield(first.ID()) || !g.HasRegenerationShield(second.ID()) {
		t.Fatal("incorrect copy or target identity")
	}
}

func TestTypedRegenerationValidation(t *testing.T) {
	saved := BindPermanent("missing")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(Regenerate(nil)))),
			WantErr: true,
		},
		{
			Name:    "this from instant",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(Regenerate(This)))),
			WantErr: true,
		},
		{
			Name:    "missing saved",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(Regenerate(From(saved))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("Invalid shield", "", WithAbility(Spell(Regenerate(DeclarePermanentTarget("undeclared", IsCreature))))),
			WantErr: true,
		},
	})
}

func TestTypedRegenerationSourceActivation(t *testing.T) {
	for _, scenario := range []string{"present", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			returned := scenario == "returned"
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 2, 2, WithAbility(Activate(Costs(ManaPayment("{G}")), Regenerate(This))))
			g.PutOnBattlefield(c, a.PlayerID())
			a.ManaPool().Add(Green, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || g.HasRegenerationShield(c.ID()) || a.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect payment or stack timing")
			}
			if returned {
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
				g.PutOnBattlefield(c, a.PlayerID())
			}
			g.ResolveStack()
			if g.HasRegenerationShield(c.ID()) == returned {
				t.Fatal("source identity was not retained")
			}
		})
	}
}

func TestTypedRegenerationDoesNotReplaceOtherRemoval(t *testing.T) {
	for _, action := range []string{"no regeneration", "sacrifice", "exile", "prohibited"} {
		t.Run(action, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 2, 2)
			c.SetOwner(a.PlayerID())
			if action == "prohibited" {
				WithKeyword(CantRegenerate)(c)
			}
			g.PutOnBattlefield(c, a.PlayerID())
			if err := ApplyEffect(g, Regenerate(This), c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !g.HasRegenerationShield(c.ID()) {
				t.Fatal("shield installation failed")
			}
			var body Effect
			switch action {
			case "no regeneration":
				body = DestroyWithoutRegeneration(This)
			case "sacrifice":
				body = Sacrifice(This)
			case "exile":
				body = Exile(This)
			case "prohibited":
				body = Destroy(This)
			}
			if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("shield prevented removal")
			}
		})
	}
}

// --- from composable_return_each_test.go ---
func TestReturnEachToHandOwnerAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "")
	first.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, b.PlayerID())
	second := NewArtifact("Second", "")
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	other := NewArtifact("Other", "")
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	phased := NewArtifact("Phased", "")
	phased.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(phased, a.PlayerID())
	g.PhaseOut(q.ID())
	clone := g.Clone()
	if err := ReturnEachToHand(PermanentsOwnedBy(You, IsArtifact)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(p.ID()) != nil || clone.FindPermanent(second.ID()) != nil || clone.FindPermanent(other.ID()) == nil || clone.FindPermanentIncludingPhased(q.ID()) == nil {
		t.Fatal("wrong battlefield membership")
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 2 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
		t.Fatal("returned cards did not reach their owner's hand")
	}
	if g.FindPermanent(p.ID()) == nil || len(a.Hand()) != 0 {
		t.Fatal("batch changed original game")
	}
	clone.PutOnBattlefield(first, b.PlayerID())
	if clone.FindPermanent(first.ID()) == nil || clone.FindPermanent(first.ID()).incarnationID == p.incarnationID {
		t.Fatal("returned card reused its old incarnation")
	}
}

func TestReturnEachToHandIsOneBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "")
	second := NewArtifact("Second", "")
	observer := NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(first.ID()) == nil && game.FindPermanent(second.ID()) != nil
		})))
	for _, card := range []Card{first, second, observer} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ReturnEachToHand(MatchingPermanents(IsArtifact)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(observer.ID()) == nil {
		t.Fatal("state trigger observed an intermediate return")
	}
}

func TestReturnEachToHandDeclaration(t *testing.T) {
	player := DeclarePlayerTarget("player")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(ReturnEachToHand(PermanentsOwnedBy(player, IsArtifact)), WithTargets(player))),
		},
		{
			Name:    "nil set",
			Card:    NewInstant("Bad", "", Spell(ReturnEachToHand(nil))),
			WantErr: true,
		},
		{
			Name:    "undeclared target in set",
			Card:    NewInstant("Bad", "", Spell(ReturnEachToHand(PermanentsOwnedBy(player, IsArtifact)))),
			WantErr: true,
		},
	})
}

func TestBouncePermanentsToHandSkipsDuplicateAndStaleObjects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Card", "")
	card.SetOwner(a.PlayerID())
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.BouncePermanentToHand(old)
	a.RemoveFromHand(card.ID())
	current := g.PutOnBattlefield(card, a.PlayerID())
	token := NewArtifact("Token", "")
	token.SetOwner(a.PlayerID())
	tokenPerm := g.PutOnBattlefield(token, a.PlayerID())
	tokenPerm.IsToken = true
	g.BouncePermanentsToHand([]*Permanent{old, current, current, tokenPerm})
	if g.FindPermanent(card.ID()) != nil || g.FindPermanent(token.ID()) != nil {
		t.Fatal("current objects did not leave")
	}
	if got := len(a.Hand()); got != 1 {
		t.Fatalf("hand size = %d, want one card and no token", got)
	}
}

// --- from composable_return_hand_test.go ---
func TestComposableReturnToHandSavedLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Saved", "", 2, 2)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: b.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := ReturnToHand(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if change == "none" || change == "phase cycle" || change == "control" {
				want = 1
			}
			if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != want {
				t.Fatalf("owner hand = %d, want %d", got, want)
			}
			if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
				t.Fatal("returned card to controller")
			}
			if len(a.Hand()) != 0 || (change != "absent" && change != "phase" && g.FindPermanent(c.ID()) == nil) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableReturnToHandValidation(t *testing.T) {
	saved := BindPermanent("saved")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(ReturnToHand(nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound permanent",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(ReturnToHand(From(saved))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(ReturnToHand(DeclarePermanentTarget("undeclared", IsCreature))))),
			WantErr: true,
		},
		{
			Name:    "producer after reader",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(Sequence(ReturnToHand(From(saved)), Snapshot(This, saved))))),
			WantErr: true,
		},
	})
}

func TestComposableReturnToHandLeaveTriggerAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Leave", "", 2, 2, WithAbility(OnLeaveZone(ZoneBattlefield, ZoneHand, GainLifeFor(You, Fixed(2)), false)))
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	if err := ApplyEffect(g, Sequence(ReturnToHand(This), ReturnToHand(This), GainLifeFor(You, Fixed(1))), c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	resolvePendingTriggers(g)
	if len(a.Hand()) != 1 || a.Life() != 23 {
		t.Fatalf("hand = %d, life = %d; want one return, one trigger, and sequence continuation", len(a.Hand()), a.Life())
	}
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	g.PutOnBattlefield(token, a.PlayerID())
	if err := ApplyEffect(g, ReturnToHand(This), token.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(token.ID()) != nil || len(a.Hand()) != 1 {
		t.Fatal("returned token remained on the battlefield or entered the hand")
	}
}

func TestComposableReturnToHandTargets(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle", "absent"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, c := range []Card{first, second} {
				g.PutOnBattlefield(c, b.PlayerID())
				c.SetOwner(b.PlayerID())
			}
			x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
			ability := Spell(Sequence(ReturnToHand(x), ReturnToHand(y)), WithTargets(x, y))
			card := NewInstant("Return", "", WithAbility(ability))
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
			case "phase":
				g.PhaseOut(first.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			}
			clone := g.Clone()
			clone.ResolveStack()
			if change == "phase" {
				clone.PhaseIn([]uuid.UUID{first.ID()})
			}
			wantHand := 1
			if change == "none" || change == "phase cycle" {
				wantHand = 2
			}
			if got := len(clone.GetPlayer(b.PlayerID()).Hand()); got != wantHand {
				t.Fatalf("hand = %d, want %d", got, wantHand)
			}
			if clone.FindPermanent(second.ID()) != nil {
				t.Fatal("legal second target stayed on battlefield")
			}
			if g.FindPermanent(second.ID()) == nil || len(b.Hand()) != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableReturnToHandCopyRetarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, b.PlayerID())
		c.SetOwner(b.PlayerID())
	}
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Spell(ReturnToHand(target), WithTargets(target))
	card := NewInstant("Return", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	g.ResolveStackObject(copied)
	if g.FindPermanent(second.ID()) != nil || g.FindPermanent(first.ID()) == nil {
		t.Fatal("copy used original binding")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()) != nil {
		t.Fatal("copy changed original binding")
	}
}

// --- from composable_reveal_card_test.go ---
func TestRevealSavedHandCardPublicRecord(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("Forest")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	target := DeclarePlayerTarget("drawer")
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: a.PlayerID()}}}
	if err := Sequence(DrawFor(target, Fixed(1), DrawCardInto(binding)), RevealCard(FromCard(binding))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	reveals := g.PublicReveals()
	if len(reveals) != 1 || reveals[0].CardID != card.ID() || reveals[0].Owner != a.PlayerID() || reveals[0].Name != "Forest" || len(reveals[0].Types) != 1 || reveals[0].Types[0] != TypeLand {
		t.Fatalf("public reveal = %+v", reveals)
	}
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() {
		t.Fatal("reveal moved the card")
	}
	reveals[0].Types[0] = TypeArtifact
	reveals[0].Card.AddType(TypeArtifact)
	if g.PublicReveals()[0].Types[0] != TypeLand || g.PublicReveals()[0].Card.HasType(TypeArtifact) {
		t.Fatal("caller mutated public history")
	}
	clone := g.Clone()
	if err := RevealCard(FromCard(binding)).Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 1 || len(clone.PublicReveals()) != 2 {
		t.Fatal("clone reveal history was shared")
	}
}

func TestRevealSavedHandCardAbsenceAndVisit(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Drawn", "")
	a.AddToLibrary(card)
	binding := BindCard("drawn")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 {
		t.Fatal("absent binding revealed a card")
	}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("drawn card missing")
	}
	a.AddToHand(card)
	if err := RevealCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 {
		t.Fatal("old hand visit was revealed")
	}
	if err := DrawFor(You, Fixed(1), DrawCardInto(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := RevealCard(FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 0 {
		t.Fatal("empty draw reused an old card")
	}
}

func TestRevealCardDeclaration(t *testing.T) {
	binding := BindCard("drawn")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(Sequence(DrawFor(You, Fixed(1), DrawCardInto(binding)), RevealCard(FromCard(binding))))),
			Inspect: func(t *testing.T) {
				if info := InspectDefinition(RevealCard(FromCard(binding))); info.Role != "resolving-effect" {
					t.Fatalf("inspection = %+v", info)
				}
			},
		},
		{
			Name:          "nil card",
			Card:          NewInstant("Invalid", "", Spell(RevealCard(nil))),
			WantErr:       true,
			WantErrSubstr: "card",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Invalid", "", Spell(RevealCard(FromCard(binding)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "conditional producer",
			Card:          NewInstant("Invalid", "", Spell(Sequence(IfState(Controls(IsCreature), DrawFor(You, Fixed(1), DrawCardInto(binding))), RevealCard(FromCard(binding))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_reverse_damage_turn_set_test.go ---
func TestReverseTurnDamageSetRecipientDeathAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipientCard := NewCreature("Recipient", "", 1, 6)
	recipient := g.PutOnBattlefield(recipientCard, a.PlayerID())
	firstCard := NewCreature("First", "", 1, 6)
	first := g.PutOnBattlefield(firstCard, b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 6), b.PlayerID())
	set := PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))
	ctx := &EffectContext{Game: g, SourceID: recipient.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: recipient.incarnationID}
	check := func(want ...*Permanent) {
		t.Helper()
		got := set.resolvePermanents(ctx)
		if len(got) != len(want) {
			t.Fatalf("selected %v, want %v", got, want)
		}
		for i := range got {
			if got[i].ID() != want[i].ID() || got[i].incarnationID != want[i].incarnationID {
				t.Fatalf("selected %v, want %v", got, want)
			}
		}
	}
	check()
	g.AddPreventionShield(recipient.ID(), 1)
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(first.ID()))
	check()
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(first.ID()))
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(second.ID()))
	check(first, second)
	g.PhaseOut(first.ID())
	check(second)
	g.PhaseIn([]uuid.UUID{first.ID()})
	check(first, second)
	g.RemoveFromBattlefield(recipient)
	g.zones.ClearLKI()
	check(first, second)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(first.ID()))
	if got := set.resolvePermanents(&EffectContext{Game: clone, SourceID: recipient.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: recipient.incarnationID}); len(got) != 1 || got[0].ID() != second.ID() {
		t.Fatalf("clone selection = %v", got)
	}
	check(first, second)
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(firstCard, b.PlayerID())
	check(second)
	newRecipient := g.PutOnBattlefield(recipientCard, a.PlayerID())
	ctx.sourceIncarnation = newRecipient.incarnationID
	check()
	g.damage.ClearEndOfTurn()
	ctx.sourceIncarnation = recipient.incarnationID
	check()
}

func TestReverseTurnDamageSetDefinition(t *testing.T) {
	set := PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 5, WithAbility(WhenEvent(Dies(This), TapEach(set)))),
			Inspect: func(t *testing.T) {
				if InspectDefinition(set).Role != "permanent-set" {
					t.Fatal("missing permanent-set inspection role")
				}
			},
		},
		{
			Name:    "nil recipient",
			Card:    NewInstant("Bad", "", Spell(TapEach(PermanentsThatDealtDamageToThisTurn(nil, MatchingPermanents(IsCreature))))),
			WantErr: true,
		},
		{
			Name:    "nil filter",
			Card:    NewInstant("Bad", "", Spell(TapEach(PermanentsThatDealtDamageToThisTurn(This, nil)))),
			WantErr: true,
		},
		{
			Name:    "target recipient",
			Card:    NewInstant("Bad", "", Spell(TapEach(PermanentsThatDealtDamageToThisTurn(DeclarePermanentTarget("recipient", IsCreature), MatchingPermanents(IsCreature))))),
			WantErr: true,
		},
	})
}

func TestReverseTurnDamageSetDeathTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 1, 4, WithAbility(WhenEvent(Dies(This),
		TapEach(PermanentsThatDealtDamageToThisTurn(This, MatchingPermanents(IsCreature))))))
	card.SetOwner(a.PlayerID())
	recipient := g.PutOnBattlefield(card, a.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 4), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 4), b.PlayerID())
	g.DealDamageToPermanentFromSource(recipient, 1, g.CaptureDamageSource(attacker.ID()))
	g.DestroyPermanent(recipient)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending death triggers = %d", g.StackSize())
	}
	g.PutOnBattlefield(card, a.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	if !clone.FindPermanent(attacker.ID()).Tapped || g.FindPermanent(attacker.ID()).Tapped {
		t.Fatal("copy did not keep the death recipient or changed the original")
	}
	g.ResolveStack()
	if !g.FindPermanent(attacker.ID()).Tapped || g.FindPermanent(other.ID()).Tapped {
		t.Fatal("death trigger selected the wrong damage sources")
	}
}

// --- from composable_sacrifice_result_test.go ---
func TestSacrificeAmountResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("sacrificed")
	card := NewCreature("Victim", "", 2, 2)
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	effect := Sequence(Sacrifice(This, SacrificeAmountInto(result)), GainLifeFor(You, FromNumber(result)))
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.Life() != 21 {
		t.Fatal("successful sacrifice result was not observed")
	}
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatal("an absent permanent reported a sacrifice")
	}
	g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || g.FindPermanent(card.ID()) == nil {
		t.Fatal("another player's permanent was sacrificed")
	}
}

func TestSacrificeAmountResultPhasingReturnAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("sacrificed")
	card := NewCreature("Victim", "", 2, 2)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := Sequence(Sacrifice(This, SacrificeAmountInto(result)), GainLifeFor(You, FromNumber(result)))
	p.PhasedOut = true
	if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanentIncludingPhased(card.ID()) == nil {
		t.Fatal("phased permanent reported a sacrifice")
	}
	p.PhasedOut = false
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 || g.FindPermanent(card.ID()) == nil {
		t.Fatal("clone shared sacrifice state")
	}
	old := p.incarnationID
	g.DoSacrifice(p)
	g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: old}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || g.FindPermanent(card.ID()) == nil {
		t.Fatal("returned permanent reported a sacrifice")
	}
}

func TestSacrificeAmountResultClearsAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindNumber("sacrificed")
	ability := Activate(Costs(), Sequence(
		Sacrifice(This, SacrificeAmountInto(result)),
		GainLifeFor(You, FromNumber(result)),
		Sacrifice(This, SacrificeAmountInto(result)),
		GainLifeFor(You, FromNumber(result)),
	))
	card := NewCreature("Victim", "", 2, 2, WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 20 || a.Life() != 20 {
		t.Fatal("copy or clone result escaped its resolution")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("a copied ability reused a prior sacrifice result")
	}
}

func TestSacrificeAmountResultDeclaration(t *testing.T) {
	result := BindNumber("sacrificed")
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(Sacrifice(This, SacrificeAmountInto(result)), GainLifeFor(You, FromNumber(result))), ""},
		{"missing producer", GainLifeFor(You, FromNumber(result)), "producer"},
		{"before producer", Sequence(GainLifeFor(You, FromNumber(result)), Sacrifice(This, SacrificeAmountInto(result))), "producer"},
		{"nil result", Sacrifice(This, SacrificeAmountInto(nil)), "non-nil"},
		{"duplicate result", Sacrifice(This, SacrificeAmountInto(result), SacrificeAmountInto(result)), "duplicate"},
		{"nil option", Sacrifice(This, nil), "non-nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCreature("Synthetic", "", 1, 1, WithAbility(Activate(Costs(), tc.body)))
			err := ValidateCardDefinition(c.Name(), c)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

// --- from composable_sacrifice_subtype_result_test.go ---
func TestSacrificeSubtypeResultChosenLand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		island   bool
		wantLife int
	}{
		{"island", true, 17},
		{"other land", false, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			chosen := BindPermanent("chosen land")
			matched := BindNumber("was Island")
			body := Sequence(ChoosePermanent(You, ControlledPermanents(IsLand), chosen),
				Sacrifice(From(chosen), SacrificeSubtypeInto("Island", matched)),
				IfPositive(FromNumber(matched), DealDamageFrom(This, You, Fixed(3))))
			source := NewCreature("Watcher", "", 2, 2)
			g.PutOnBattlefield(source, a.PlayerID())
			land := NewLand("Land")
			if tc.island {
				land = NewLand("Island", WithSubTypes("Island"))
			}
			g.PutOnBattlefield(land, a.PlayerID())
			opponentLand := NewLand("Opponent Island", WithSubTypes("Island"))
			g.PutOnBattlefield(opponentLand, b.PlayerID())
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != tc.wantLife || g.FindPermanent(land.ID()) != nil || g.FindPermanent(opponentLand.ID()) == nil {
				t.Fatalf("life = %d, land = %v", a.Life(), g.FindPermanent(land.ID()))
			}
		})
	}
}

func TestSacrificeSubtypeResultPendingCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	matched := BindNumber("was Island")
	source := NewLand("Island", WithSubTypes("Island"), WithAbility(Activate(Costs(), Sequence(
		Sacrifice(This, SacrificeSubtypeInto("Island", matched)),
		IfPositive(FromNumber(matched), GainLifeFor(You, Fixed(3)))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 20 || a.Life() != 20 {
		t.Fatal("clone or copied ability reused a sacrifice result")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("pending abilities shared a sacrifice result")
	}
}

func TestSacrificeSubtypeResultFailureClearsPriorValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Island", WithSubTypes("Island"))
	p := g.PutOnBattlefield(land, a.PlayerID())
	matched := BindNumber("was Island")
	body := Sacrifice(This, SacrificeSubtypeInto("Island", matched))
	ctx := &EffectContext{Game: g, SourceID: land.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	for _, state := range []string{"phased", "returned", "other controller", "absent"} {
		g.resolution.numberBindings = map[*NumberBinding]int{matched: 1}
		switch state {
		case "phased":
			p.PhasedOut = true
		case "returned":
			p.PhasedOut = false
			g.RemoveFromBattlefield(p)
			p = g.PutOnBattlefield(land, a.PlayerID())
		case "other controller":
			ctx.sourceIncarnation = p.incarnationID
			ctx.Controller = b.PlayerID()
		case "absent":
			g.RemoveFromBattlefield(p)
		}
		if err := body.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if g.resolution.numberBindings[matched] != 0 {
			t.Fatalf("%s kept a stale subtype result", state)
		}
	}
}

func TestSacrificeSubtypeResultDeclaration(t *testing.T) {
	matched := BindNumber("matched")
	info := InspectDefinition(Sacrifice(This, SacrificeSubtypeInto("Island", matched)))
	if info.Role != "resolving-effect" || len(info.Children) != 2 || info.Children[1].Role != "number-binding" {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(Sacrifice(This, SacrificeSubtypeInto("Island", matched)), GainLifeFor(You, FromNumber(matched))), ""},
		{"empty subtype", Sacrifice(This, SacrificeSubtypeInto("", matched)), "subtype"},
		{"nil binding", Sacrifice(This, SacrificeSubtypeInto("Island", nil)), "binding"},
		{"duplicate output", Sacrifice(This, SacrificeSubtypeInto("Island", matched), SacrificeAmountInto(matched)), "duplicate"},
		{"read before output", Sequence(GainLifeFor(You, FromNumber(matched)), Sacrifice(This, SacrificeSubtypeInto("Island", matched))), "producer"},
		{"missing producer", GainLifeFor(You, FromNumber(matched)), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewLand("Test", WithAbility(Activate(Costs(), tc.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

// --- from composable_singular_life_amount_test.go ---
func TestTypedSingularLifeAmountContext(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Amount", "", 3, 4)
		g.PutOnBattlefield(source, a.PlayerID())
		saved := BindPermanent("amount")
		amount := SubtractValues(PowerOf(From(saved)), Fixed(1))
		effect := LoseLifeFor(ActivePlayer, amount)
		want := 18
		if gain {
			effect = GainLifeFor(ActivePlayer, amount)
			want = 22
		}
		body := Sequence(Snapshot(This, saved), effect)
		clone := g.Clone()
		if err := ApplyEffect(clone, body, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if clone.GetPlayer(a.PlayerID()).Life() != want || clone.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("saved amount or recipient was lost")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("clone changed original life")
		}
	}
}

func TestTypedSingularLifeTargetCopy(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Amount", "", 3, 4)
		g.PutOnBattlefield(source, a.PlayerID())
		player := DeclarePlayerTarget("player")
		amount := DeclarePermanentTarget("amount", IsCreature)
		effect := LoseLifeFor(player, ToughnessOf(amount))
		want := 16
		if gain {
			effect = GainLifeFor(player, ToughnessOf(amount))
			want = 24
		}
		ability := Spell(effect, WithTargets(player, amount))
		card := NewSorcery("Life", "", WithAbility(ability))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
		g.pushStack(original)
		clone := g.Clone()
		copied := clone.CopySpellOnStack(card.ID(), b.PlayerID(), false)
		if copied == nil {
			t.Fatal("copy missing")
		}
		clone.ResolveStackObject(copied)
		if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != want {
			t.Fatal("copy lost amount or recipient target")
		}
		if a.Life() != 20 || b.Life() != 20 {
			t.Fatal("copy changed original game")
		}
		g.ResolveStackObject(original)
		if a.Life() != 20 || b.Life() != want {
			t.Fatal("original lost amount or recipient target")
		}
	}
}

func TestTypedSingularLifeAmountValidationAndCapture(t *testing.T) {
	cases := make([]DefinitionValidationCase, 0, 10)
	for _, constructor := range []struct {
		name string
		fn   func(PlayerRef, ResolutionValue) ResolvingEffect
	}{
		{"gain life", GainLifeFor},
		{"lose life", LoseLifeFor},
	} {
		for _, amount := range []struct {
			name string
			val  ResolutionValue
		}{
			{"nil", nil},
			{"power of nil", PowerOf(nil)},
			{"missing binding", PowerOf(From(BindPermanent("missing")))},
			{"undeclared target", PowerOf(DeclarePermanentTarget("undeclared", IsCreature))},
			{"nil resolution value", ValueAtResolution(nil)},
		} {
			cases = append(cases, DefinitionValidationCase{
				Name:          constructor.name + " " + amount.name,
				Card:          NewSorcery("Invalid life", "", WithAbility(Spell(constructor.fn(You, amount.val)))),
				WantErr:       true,
				WantErrSubstr: "abilities[0].effects[0].amount",
			})
		}
	}
	RunDefinitionValidationTable(t, cases)
	for _, constructor := range []func(PlayerRef, ResolutionValue) ResolvingEffect{GainLifeFor, LoseLifeFor} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		calls := 0
		amount := countedLifeAmount{&calls}
		InspectDefinition(constructor(You, amount))
		if err := ApplyEffect(g, constructor(ControllerOf(This), amount), uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if calls != 0 {
			t.Fatal("inspection or absent recipient evaluated amount")
		}
		if err := ApplyEffect(g, constructor(You, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if calls != 1 || a.Life() == 20 || b.Life() != 20 {
			t.Fatal("amount was not captured exactly once")
		}
	}
}

func TestTypedSingularLifeDelayedAmount(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewCreature("Amount", "", 3, 4)
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		saved := BindPermanent("amount")
		effect := LoseLifeFor(You, PowerOf(From(saved)))
		want := 17
		if gain {
			effect = GainLifeFor(You, PowerOf(From(saved)))
			want = 23
		}
		body := Sequence(Snapshot(This, saved), AtNext(EachEndStep, effect))
		if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		clone.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		clone.PutTriggersOnStack()
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != want || a.Life() != 20 {
			t.Fatal("delayed amount lost its saved frame or clone isolation")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		g.PutOnBattlefield(source, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.Life() != 20 {
			t.Fatal("delayed amount followed a new incarnation")
		}
	}
}

// --- from composable_singular_life_gain_result_test.go ---
func TestSingularLifeGainResultUsesFinalOutcome(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	result := BindNumber("gained")
	body := Sequence(GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)), LoseLifeFor(You, FromNumber(result)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("life=%d/%d", a.Life(), b.Life())
	}
	clone := g.Clone()
	if err := GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 5 || clone.GetPlayer(a.PlayerID()).Life() != 25 || a.Life() != 20 {
		t.Fatal("replacement result or clone isolation failed")
	}
	if err := GainLifeForWithResult(FromPlayer(BindPlayer("absent")), Fixed(3), LifeGainedInto(result)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[result] != 0 {
		t.Fatal("absent recipient or clone changed a prior result")
	}
}

func TestSingularLifeGainResultDeclaration(t *testing.T) {
	result := BindNumber("gained")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid declaration",
			Card: NewInstant("Good", "", WithAbility(Spell(Sequence(
				GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)),
				LoseLifeFor(You, FromNumber(result)),
			)))),
		},
		{
			Name:          "no option",
			Card:          NewInstant("Bad", "", WithAbility(Spell(GainLifeForWithResult(You, Fixed(2))))),
			WantErr:       true,
			WantErrSubstr: "option",
		},
		{
			Name:          "nil binding",
			Card:          NewInstant("Bad", "", WithAbility(Spell(GainLifeForWithResult(You, Fixed(2), LifeGainedInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name",
			Card:          NewInstant("Bad", "", WithAbility(Spell(GainLifeForWithResult(You, Fixed(2), LifeGainedInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate option",
			Card:          NewInstant("Bad", "", WithAbility(Spell(GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result), LifeGainedInto(result))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Bad", "", WithAbility(Spell(Sequence(LoseLifeFor(You, FromNumber(result)), GainLifeForWithResult(You, Fixed(2), LifeGainedInto(result)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestSingularLifeGainResultNonpositive(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	result := BindNumber("gained")
	for _, amount := range []int{0, -2} {
		g.resolution.numberBindings = map[*NumberBinding]int{result: 7}
		if err := GainLifeForWithResult(You, Fixed(amount), LifeGainedInto(result)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if g.resolution.numberBindings[result] != 0 || a.Life() != 20 {
			t.Fatalf("amount %d left result %d and life %d", amount, g.resolution.numberBindings[result], a.Life())
		}
	}
}

// --- from composable_singular_life_loss_result_test.go ---
func TestSingularLifeLossResult(t *testing.T) {
	for _, amount := range []int{-1, 0, 4, 25} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		lost := BindNumber("lost")
		body := Sequence(LoseLifeForWithResult(You, Fixed(amount), LifeLostInto(lost)), GainLifeFor(ActivePlayer, FromNumber(lost)))
		if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := max(0, amount)
		if b.Life() != 20-want || a.Life() != 20+want {
			t.Fatalf("amount %d: life %d/%d, want %d/%d", amount, a.Life(), b.Life(), 20+want, 20-want)
		}
	}
}

func TestSingularLifeLossResultAbsentAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lost := BindNumber("lost")
	g.resolution.numberBindings = map[*NumberBinding]int{lost: 99}
	clone := g.Clone()
	if err := LoseLifeForWithResult(FromPlayer(BindPlayer("absent")), Fixed(4), LifeLostInto(lost)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[lost] != 0 || g.resolution.numberBindings[lost] != 99 {
		t.Fatal("absent result changed the original frame or kept a stale value")
	}
}

func TestSingularLifeLossResultTargetAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lost := BindNumber("lost")
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	ability := Spell(Sequence(
		LoseLifeForWithResult(target, Fixed(3), LifeLostInto(lost)),
		GainLifeFor(You, FromNumber(lost)),
	), WithTargets(target))
	card := NewSorcery("Life", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	clone := g.Clone()
	copied := clone.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	clone.ResolveStackObject(copied)
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 17 {
		t.Fatal("copied instruction lost its target or result")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original life totals")
	}
	g.ResolveStackObject(original)
	if a.Life() != 23 || b.Life() != 17 {
		t.Fatal("original instruction lost its target or result")
	}
}

func TestSingularLifeLossResultDefinition(t *testing.T) {
	lost := BindNumber("lost")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid declaration",
			Card: NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(
				LoseLifeForWithResult(You, Fixed(4), LifeLostInto(lost)), GainLifeFor(You, FromNumber(lost)))))),
		},
		{
			Name:          "no option",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeForWithResult(You, Fixed(4))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "nil binding",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeForWithResult(You, Fixed(4), LifeLostInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty name",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeForWithResult(You, Fixed(4), LifeLostInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate option",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), LoseLifeForWithResult(You, Fixed(4), LifeLostInto(lost), LifeLostInto(lost))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "late producer",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Sequence(GainLifeFor(You, FromNumber(lost)), LoseLifeForWithResult(You, Fixed(4), LifeLostInto(lost)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_singular_no_regeneration_test.go ---
func TestSingularNoRegenerationPreservesCharacteristics(t *testing.T) {
	for _, kind := range []string{"typed", "positional", "step", "saved"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Victim", "", 2, 2)
			c.SetOwner(a.PlayerID())
			g.PutOnBattlefield(c, a.PlayerID())
			g.AddRegenerationShield(c.ID())
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}}
			var effect Effect
			switch kind {
			case "typed":
				effect = DestroyWithoutRegeneration(This)
			case "positional":
				effect = DestroyTargetNoRegen()
			case "step":
				effect = DestroyTargetNoRegenStep()
			case "saved":
				victim := BindPermanent("victim")
				effect = Sequence(Snapshot(This, victim), DestroyWithoutRegeneration(From(victim)))
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("regeneration prevented destruction")
			}
			lki := g.LookupObject(c.ID())
			if lki == nil || lki.ViewHasKeyword(CantRegenerate) {
				t.Fatal("destruction changed the victim's last-known characteristics")
			}
		})
	}
}

func TestSingularNoRegenerationSavedLifetime(t *testing.T) {
	for _, change := range []string{"same", "return", "absent", "phase", "phase cycle", "control", "indestructible"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 2, 2)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.AddRegenerationShield(c.ID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "indestructible":
				p.GrantBaseAttr(Indestructible)
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := DestroyWithoutRegeneration(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			survives := change == "return" || change == "phase" || change == "indestructible"
			got := clone.FindPermanentIncludingPhased(c.ID())
			if (got != nil) != survives {
				t.Fatal("incorrect saved subject lifetime or destruction")
			}
			if got != nil && got.HasKeyword(CantRegenerate) {
				t.Fatal("destruction changed survivor characteristics")
			}
			if change != "absent" && g.FindPermanentIncludingPhased(c.ID()) == nil {
				t.Fatal("clone destruction changed original")
			}
			if change == "indestructible" {
				if !clone.HasRegenerationShield(c.ID()) {
					t.Fatal("indestructible destruction consumed shield")
				}
			}
		})
	}
}

// --- from composable_source_duration_test.go ---
func TestComposableSourceDurationLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	host := NewCreature("Host", "{R}", 2, 2)
	source.SetOwner(a.PlayerID())
	host.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	g.MutablePermanent(source.ID()).AttachedTo = host.ID()
	effect := UntilSourceLeaves(This,
		ModifyPT(AttachedPermanent, Fixed(2), Fixed(3)),
		SetColors(AttachedPermanent, Blue),
		GrantKeywordModification(AttachedPermanent, Flying),
	)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check := func(game *Game, active bool) {
		t.Helper()
		p := game.FindPermanent(host.ID())
		wantPower, wantToughness := 2, 2
		if active {
			wantPower, wantToughness = 4, 5
		}
		if p.CurrentPower(game) != wantPower || p.CurrentToughness(game) != wantToughness || p.HasAttr(Flying) != active || slices.Contains(game.EffectiveColors(host.ID()), Blue) != active {
			t.Fatalf("active %v: power %d, toughness %d, flying %v, blue %v", active, p.CurrentPower(game), p.CurrentToughness(game), p.HasAttr(Flying), slices.Contains(game.EffectiveColors(host.ID()), Blue))
		}
	}
	check(g, true)
	phased := g.PhaseOut(source.ID())
	g.effects.RemoveEndOfCombat()
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(g, true)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, true)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	clone.PutOnBattlefield(source, a.PlayerID())
	clone.ApplyContinuousEffects()
	check(clone, false)
	check(g, true)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ApplyContinuousEffects()
	check(g, false)
}

func TestComposableSourceDurationTriggerContext(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		body := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(2), Fixed(0)))
		source := NewCreature("Source", "", 1, 1, WithAbility(At(YourUpkeep, body)))
		host := NewCreature("Host", "", 2, 2)
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutOnBattlefield(host, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		if g.stack.Size() != 1 {
			t.Fatal("missing source trigger")
		}
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(source, b.PlayerID())
		} else {
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.PhaseOut(source.ID())
		}
		g.ResolveStack()
		want := 4
		if returned {
			want = 2
		}
		if got := g.FindPermanent(host.ID()).CurrentPower(g); got != want {
			t.Fatalf("returned %v: power = %d, want %d", returned, got, want)
		}
	}
}

func TestComposableSourceDurationDirectPhasedContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	g.PhaseOut(source.ID())
	effect := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(0)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(host.ID()).CurrentPower(g); got != 3 {
		t.Fatalf("power = %d, want 3", got)
	}
}

func TestComposableSourceDurationBeforeResolution(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", 1, 1)
			host := NewCreature("Host", "", 2, 2)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(host, a.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: g.FindPermanent(source.ID()).incarnationID}
			if state == "phased" {
				g.PhaseOut(source.ID())
			} else {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					g.PutOnBattlefield(source, a.PlayerID())
				}
			}
			effect := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(3), Fixed(0)))
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 2
			if state == "phased" {
				want = 5
			}
			if got := g.FindPermanent(host.ID()).CurrentPower(g); got != want {
				t.Fatalf("power = %d, want %d", got, want)
			}
		})
	}
}

func TestComposableSourceDurationCapturedMembers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	host := NewCreature("Host", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(host, a.PlayerID())
	effect := UntilSourceLeaves(This, ModifyEachPT(ControlledPermanents(IsCreature), ValueAtResolution(CountBattlefield(SelectController(), IsCreature)), Fixed(0)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	later := NewCreature("Later", "", 2, 2)
	g.PutOnBattlefield(later, a.PlayerID())
	phased := g.PhaseOut(host.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CurrentPower(g) != 4 || g.FindPermanent(later.ID()).CurrentPower(g) != 2 {
		t.Fatal("membership, amount, or phasing did not retain the captured value")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(host, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CurrentPower(g) != 2 {
		t.Fatal("modification followed a returned member")
	}
}

func TestComposableSourceDurationValidation(t *testing.T) {
	saved := BindPermanent("missing")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil effect",
			Card:          NewCreature("Invalid duration", "", 1, 1, WithAbility(Activate(Costs(), UntilSourceLeaves(This)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0]",
		},
		{
			Name:          "explicit nil effect",
			Card:          NewCreature("Invalid duration", "", 1, 1, WithAbility(Activate(Costs(), UntilSourceLeaves(This, nil)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0]",
		},
		{
			Name:          "unbound effect",
			Card:          NewCreature("Invalid duration", "", 1, 1, WithAbility(Activate(Costs(), UntilSourceLeaves(This, ModifyPT(From(saved), Fixed(1), Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0]",
		},
		{
			Name:    "spell-only source",
			Card:    NewSorcery("Invalid source", "", WithAbility(Spell(UntilSourceLeaves(This, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(1)))))),
			WantErr: true,
		},
	})
}

// --- from composable_source_tapped_test.go ---
func TestSourceTappedState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("State source", "", 1, 1)
	g.PutOnBattlefield(card, a.PlayerID())
	p := g.FindPermanent(card.ID())
	ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
	for _, tc := range []struct{ tapped, want bool }{{false, false}, {true, true}} {
		g.MutablePermanent(card.ID()).Tapped = tc.tapped
		if got := SourceTappedCondition().checkState(g, ctx); got != tc.want {
			t.Fatalf("tapped=%v: got %v", tc.tapped, got)
		}
		if got := SourceTappedCondition().checkState(g.Clone(), ctx); got != tc.want {
			t.Fatalf("clone tapped=%v: got %v", tc.tapped, got)
		}
		if got := SourceUntappedCondition().checkState(g, ctx); got == tc.want {
			t.Fatalf("untapped=%v: got %v", !tc.tapped, got)
		}
	}
	clone := g.Clone()
	clone.MutablePermanent(card.ID()).Tapped = false
	if !SourceTappedCondition().checkState(g, ctx) || SourceTappedCondition().checkState(clone, ctx) {
		t.Fatal("clone shares tapped state")
	}
	g.MutablePermanent(card.ID()).PhasedOut = true
	if SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("phased-out source matched")
	}
	if SourceUntappedCondition().checkState(g, ctx) {
		t.Fatal("phased-out source matched untapped")
	}
	g.MutablePermanentIncludingPhased(card.ID()).PhasedOut = false
	if !SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("phased-in source lost identity")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
	if SourceUntappedCondition().checkState(g, ctx) {
		t.Fatal("absent source matched untapped")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(card.ID()).Tapped = true
	if SourceTappedCondition().checkState(g, ctx) {
		t.Fatal("returned source matched old incarnation")
	}
	if SourceTappedCondition().checkState(g, StateContext{SourceID: uuid.Nil}) {
		t.Fatal("missing source matched")
	}
}

func TestSourceTappedStateRegistration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Tapped state", "", 1, 1, WithAbility(WhenState(SourceTappedCondition(), Sacrifice(This)))),
		},
		{
			Name:          "invalid state on spell",
			Card:          NewInstant("Invalid state", "", WithEffect(IfState(SourceTappedCondition(), Sacrifice(This)))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
}

func TestSourceTappedStaticAndInterveningIf(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Tapped source", "", 1, 1,
		WithAbility(Static(While(SourceUntappedCondition(), ModifyPT(This, Fixed(0), Fixed(3))))),
		WithAbility(WhenEvent(Enters(This), GainLifeFor(You, Fixed(2)), WithInterveningIf(SourceUntappedCondition()))))
	if err := ValidateCardDefinition("Tapped source", card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 4 {
		t.Fatalf("untapped toughness = %d", got)
	}
	g.PutTriggersOnStack()
	g.MutablePermanent(card.ID()).Tapped = true
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 1 {
		t.Fatalf("tapped toughness = %d", got)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("intervening condition failed recheck: life = %d", a.Life())
	}
}

// --- from composable_spell_cast_caster_test.go ---
func TestSpellCastCasterEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "")
	spell.SetOwner(b.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: a.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("own cast triggered: stack=%d", g.StackSize())
	}
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("opponent cast trigger: stack=%d, want 3", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original caster")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatalf("life=(%d,%d), want (20,21)", a.Life(), b.Life())
	}
}

func TestSpellCastCasterEventDeclaration(t *testing.T) {
	caster := BindPlayer("caster")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(
				SpellCast().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1))))),
			Inspect: func(t *testing.T) {
				node := InspectDefinition(SpellCast().ByOpponent().WithCaster(caster))
				if node.Role != "event-pattern" || !strings.Contains(node.Text, "opponent") {
					t.Fatalf("inspection=%+v", node)
				}
			},
		},
		{
			Name:    "missing binding",
			Card:    NewArtifact("Missing", "", WithAbility(WhenEvent(SpellCast().WithCaster(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound read",
			Card:    NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(FromPlayer(caster), Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "wrong source zone",
			Card:    NewInstant("Wrong", "", WithAbility(WhenEvent(SpellCast().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate capture",
			Card:    NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().WithCaster(caster).WithCaster(BindPlayer("second")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestSpellCastCasterEventCopyAndSeparateCasts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, player := range []*BasePlayer{a, b} {
		spell := NewInstant("Spell", "")
		spell.SetOwner(player.PlayerID())
		player.AddToHand(spell)
		if err := g.CastSpellByName(player.PlayerID(), spell.Name(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d, want 4", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatalf("life=(%d,%d), want (22,21)", a.Life(), b.Life())
	}
}

func TestSpellCastCasterEventPhasingAndFailedCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithCaster(caster), GainLifeFor(FromPlayer(caster), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PhaseOut(watcher.ID())
	spell := NewInstant("Unpaid", "{U}")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without payment succeeded")
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || b.Life() != 20 {
		t.Fatal("phased source or failed cast created a trigger")
	}
}

func TestSpellCastCasterEventColorComposition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().OfColor(Blue).WithCaster(caster),
		GainLifeFor(FromPlayer(caster), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, tc := range []struct {
		controller uuid.UUID
		cost       string
	}{
		{b.PlayerID(), "{R}"},
		{a.PlayerID(), "{U}"},
		{b.PlayerID(), "{U}"},
	} {
		spell := NewInstant("Spell", tc.cost)
		spell.SetOwner(tc.controller)
		object := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: tc.controller}
		g.pushStack(object)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: object.ID, PlayerID: tc.controller})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("stack=%d, want 4", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatalf("life=(%d,%d), want (20,21)", a.Life(), b.Life())
	}
}

// --- from composable_spell_cast_color_test.go ---
func TestSpellCastColorEvent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cost    string
		change  Color
		castByB bool
		phase   bool
		want    int
	}{
		{name: "blue", cost: "{U}", want: 1},
		{name: "opponent blue", cost: "{U}", castByB: true, want: 1},
		{name: "multicolor blue", cost: "{U}{R}", want: 1},
		{name: "red", cost: "{R}"},
		{name: "changed red to blue", cost: "{R}", change: Blue, want: 1},
		{name: "changed blue to red", cost: "{U}", change: Red},
		{name: "phased source", cost: "{U}", phase: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			caster := a.PlayerID()
			if tc.castByB {
				caster = b.PlayerID()
			}
			spell := NewInstant("Spell", tc.cost)
			spell.SetOwner(caster)
			g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: caster})
			if tc.change != Colorless && !g.ChangeObjectColor(spell.ID(), tc.change) {
				t.Fatal("could not change spell color")
			}
			g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: caster})
			g.PutTriggersOnStack()
			if got := g.StackSize() - 1; got != tc.want {
				t.Fatalf("trigger count=%d, want %d", got, tc.want)
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

func TestSpellCastColorEventPendingController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Blue Spell", "{U}")
	spell.SetOwner(b.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()})
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestSpellCastColorEventRealCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Blue Spell", "")
	spell.SetColorOverride([]Color{Blue})
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size=%d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life=%d, want 21", a.Life())
	}
}

func TestSpellCastColorEventFailedCastDoesNotTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Unpaid", "{U}")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without mana succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed cast created a trigger")
	}
}

func TestSpellCastColorEventUsesCastStackIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Recast", "{R}")
	spell.SetOwner(b.PlayerID())
	blue := []Color{Blue}
	oldCopy := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID(), ColorOverride: &blue, IsCopy: true}
	cast := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()}
	g.pushStack(oldCopy)
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack size=%d, want 2; cast red spell matched blue copy", g.StackSize())
	}
}

func TestSpellCastColorEventDeclaration(t *testing.T) {
	pattern := SpellCast().OfColor(Blue)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(pattern); got.Role != "event-pattern" || got.Text != "a blue spell is cast" {
					t.Fatalf("inspection=%+v", got)
				}
			},
		},
		{
			Name:          "colorless",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfColor(Colorless), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
		{
			Name:          "any color",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfColor(AnyColor), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
		{
			Name:          "duplicate color",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfColor(Blue).OfColor(Red), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "color",
		},
		{
			Name:          "invalid source zone",
			Card:          NewInstant("Invalid Source", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
}

// --- from composable_spell_cast_mana_test.go ---
func TestSpellCastManaSpentCapturedBeforeSpellLeaves(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spent := BindNumber("spent")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithManaSpent(spent), GainLifeFor(You, FromNumber(spent)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "", Spell(DrawFor(You, Fixed(0))))
	spell.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(),
		CastContext: &CastContext{ColorsSpent: map[Color]int{Red: 2, Colorless: 3}}}
	g.pushStack(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	obj.CastContext.ColorsSpent[Red] = 10
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.stack.RemoveByID(obj.ID)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 25 {
			t.Fatalf("captured mana = %d", got)
		}
	}
}

func TestSpellCastManaSpentZeroAndPhasedSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spent := BindNumber("spent")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithManaSpent(spent), GainLifeFor(You, FromNumber(spent)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	permanent := g.FindPermanent(watcher.ID())
	spell := NewInstant("Free", "", Spell(DrawFor(You, Fixed(0))))
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(), CastContext: &CastContext{}}
	g.pushStack(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("phased source triggered: stack = %d", g.StackSize())
	}
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("free cast trigger: stack = %d", g.StackSize())
	}
	g.stack.RemoveByID(obj.ID)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("zero mana capture = %d", a.Life())
	}
}

func TestSpellCastManaSpentDeclaration(t *testing.T) {
	spent := BindNumber("spent")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(spent), GainLifeFor(You, FromNumber(spent))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(SpellCast().WithManaSpent(spent)); got.Role != "event-pattern" || len(got.Children) != 1 {
					t.Fatalf("inspection = %#v", got)
				}
			},
		},
		{
			Name:          "nil",
			Card:          NewArtifact("Nil", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(nil), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "unnamed",
			Card:          NewArtifact("Unnamed", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(BindNumber("")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate",
			Card:          NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().WithManaSpent(spent).WithManaSpent(BindNumber("other")), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "one",
		},
		{
			Name:          "unbound",
			Card:          NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, FromNumber(spent))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_spell_cast_owner_test.go ---
func TestSpellCastNotOwnedByCasterUsesExactSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().NotOwnedByCaster().ByOpponent().FromZone(ZoneExile),
		GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Shared", "")
	card.SetOwner(a.PlayerID())
	foreign := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), CastZone: ZoneExile}
	owned := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), CastZone: ZoneExile}
	g.pushStack(foreign)
	g.pushStack(owned)
	ownedByB := NewInstant("Owned by caster", "")
	ownedByB.SetOwner(b.PlayerID())
	ownedByBObject := &StackObject{ID: uuid.New(), Card: ownedByB, SourceID: ownedByB.ID(), Controller: b.PlayerID(), CastZone: ZoneExile}
	g.pushStack(ownedByBObject)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: ownedByB.ID(), TargetID: ownedByBObject.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("caster-owned spell matched: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: owned.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("owned spell matched: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: ownedByB.ID(), TargetID: foreign.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("mismatched source matched: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: foreign.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatalf("foreign spell missed: stack=%d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.PhaseOut(watcher.ID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 22 {
		t.Fatalf("copied trigger lost caster ownership result: life=%d", a.Life())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: foreign.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("phased watcher triggered: stack=%d", g.StackSize())
	}
}

func TestSpellCastNotOwnedByCasterRejectsInvalidDeclaration(t *testing.T) {
	pattern := SpellCast().NotOwnedByCaster()
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(pattern); !strings.Contains(got.Text, "not owned") {
					t.Fatalf("inspection=%+v", got)
				}
			},
		},
		{
			Name:    "duplicate ownership filter",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(pattern.NotOwnedByCaster(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestSpellCastNotOwnedByCasterRealCastAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().NotOwnedByCaster(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	unpaid := NewInstant("Unpaid", "{U}")
	unpaid.SetOwner(b.PlayerID())
	b.AddToHand(unpaid)
	if err := g.CastSpellByName(b.PlayerID(), unpaid.Name(), nil); err == nil {
		t.Fatal("unpaid spell was cast")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed cast produced a trigger")
	}
	foreign := NewInstant("Foreign", "")
	foreign.SetOwner(a.PlayerID())
	g.ExileCard(foreign, uuid.Nil)
	if err := g.CastCardFromExileWithoutPaying(b.PlayerID(), foreign.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("foreign cast stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("foreign cast life=%d", a.Life())
	}
}

// --- from composable_spell_cast_subtype_test.go ---
func TestSpellCastSubtypeUsesCastStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		SpellCast().OfType(TypeEnchantment).OfEnchantmentSubtype("Aura").ByOpponent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewAura("Aura spell", "")
	spell.SetOwner(b.PlayerID())
	other := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()}
	g.pushStack(other)
	current := &StackObject{ID: uuid.New(), Card: NewEnchantment("Current form", ""), SourceID: spell.ID(), Controller: b.PlayerID()}
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: current.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("stack size=%d, want 2", got)
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: other.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 3 {
		t.Fatalf("stack size=%d, want 3", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("original life after clone resolution=%d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("life=%d, want 21", got)
	}
}

func TestSpellCastSubtypeSourceLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		phase bool
		leave bool
		want  int
	}{
		{name: "present", want: 1},
		{name: "phased", phase: true},
		{name: "left", leave: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(SpellCast().OfEnchantmentSubtype("Aura"), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			if tc.leave {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			}
			spell := NewAura("Aura", "")
			spell.SetOwner(b.PlayerID())
			object := &StackObject{ID: uuid.New(), Card: spell, SourceID: spell.ID(), Controller: b.PlayerID()}
			g.pushStack(object)
			g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: object.ID, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if got := g.StackSize() - 1; got != tc.want {
				t.Fatalf("triggers=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestSpellCastSubtypeDeclaration(t *testing.T) {
	pattern := SpellCast().OfEnchantmentSubtype("Aura")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(pattern); got.Role != "event-pattern" || !strings.Contains(got.Text, "Aura") {
					t.Fatalf("inspection=%+v", got)
				}
			},
		},
		{
			Name:          "empty subtype",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfEnchantmentSubtype(""), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name:          "unknown subtype",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfEnchantmentSubtype("Bogus"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name:          "duplicate subtype",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfEnchantmentSubtype("Aura").OfEnchantmentSubtype("Aura"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
		{
			Name:          "mismatched type and subtype",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfType(TypeCreature).OfEnchantmentSubtype("Aura"), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "subtype",
		},
	})
}

// --- from composable_spell_cast_type_test.go ---
func TestSpellCastTypeEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("artifact caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeArtifact).ByOpponent().WithCaster(caster),
		GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Cast artifact", "")
	artifact.SetOwner(b.PlayerID())
	instant := NewInstant("Cast instant", "")
	instant.SetOwner(b.PlayerID())
	for _, card := range []Card{instant, artifact} {
		object := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
		g.pushStack(object)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: object.ID, PlayerID: b.PlayerID()})
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack=%d, want two spells and one trigger", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original caster")
	}
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatalf("caster life=%d, want 21", b.Life())
	}
}

func TestSpellCastTypeEventUsesStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Shared card", "")
	artifact.SetOwner(b.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: artifact, SourceID: artifact.ID(), Controller: b.PlayerID()}
	current := &StackObject{ID: uuid.New(), Card: NewInstant("Current form", ""), SourceID: artifact.ID(), Controller: b.PlayerID()}
	g.pushStack(old)
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: artifact.ID(), TargetID: current.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, type matched another object with the same source card", g.StackSize())
	}
}

func TestSpellCastTypeEventDeclaration(t *testing.T) {
	pattern := SpellCast().OfType(TypeArtifact)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(pattern); got.Role != "event-pattern" || !strings.Contains(got.Text, "artifact") {
					t.Fatalf("inspection=%+v", got)
				}
			},
		},
		{
			Name:          "land type",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfType(TypeLand), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "type",
		},
		{
			Name:          "invalid card type enum",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfType(CardType(-1)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "type",
		},
		{
			Name:          "multiple types",
			Card:          NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().OfType(TypeArtifact).OfType(TypeCreature), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "type",
		},
	})
}

func TestSpellCastTypeEventLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phase  bool
		remove bool
		want   int
	}{
		{name: "present", want: 1},
		{name: "phased source", phase: true},
		{name: "absent source", remove: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				SpellCast().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phase {
				g.PhaseOut(watcher.ID())
			}
			if tc.remove {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
			}
			spell := NewArtifact("Spell", "")
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			g.turns.SetActivePlayerIndex(1)
			g.turns.SetStep(PrecombatMain)
			if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if got := g.StackSize() - 1; got != tc.want {
				t.Fatalf("trigger count=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestSpellCastTypeEventFailedCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeArtifact), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewArtifact("Unpaid", "{2}")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without mana succeeded")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed cast created a trigger")
	}
}

// --- from composable_spell_cast_x_test.go ---
func TestSpellCastWithXInManaCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		SpellCast().WithXInManaCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, tc := range []struct {
		cost string
		x    int
	}{
		{"{X}{R}", 0},
		{"{X}{X}{U}", 2},
		{"{2}{R}", 0},
	} {
		spell := NewInstant("Spell", tc.cost)
		obj := &StackObject{ID: spell.ID(), SourceID: spell.ID(), Card: spell, XValue: tc.x, Controller: b.PlayerID()}
		g.stack.Push(obj)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
		g.stack.RemoveByID(obj.ID)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want 2", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 22 {
			t.Fatalf("life = %d, want 22", got)
		}
	}
}

func TestSpellCastWithXInManaCostSourceLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		SpellCast().WithXInManaCost(), GainLifeFor(You, Fixed(1)))))
	permanent := g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "{X}{R}")
	obj := &StackObject{ID: spell.ID(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID()}
	g.stack.Push(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.stack.RemoveByID(obj.ID)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d, want 1", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

func TestSpellCastWithXInManaCostDeclaration(t *testing.T) {
	pattern := SpellCast().WithXInManaCost()
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Valid", "", 2, 2, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(pattern); got.Role != "event-pattern" {
					t.Fatalf("inspection = %#v", got)
				}
			},
		},
		{
			Name:          "wrong zone",
			Card:          NewInstant("Wrong zone", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Duplicate", "", 2, 2, WithAbility(WhenEvent(pattern.WithXInManaCost(), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "spell cast",
		},
	})
}

// --- from composable_spell_cast_zone_test.go ---
func TestSpellCastFromZoneUsesExactStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	caster := BindPlayer("caster")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().FromZone(ZoneExile).ByOpponent().WithCaster(caster),
		GainLifeFor(FromPlayer(caster), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Shared", "")
	card.SetOwner(b.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), CastZone: ZoneExile}
	current := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), CastZone: ZoneHand}
	g.pushStack(old)
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: current.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("matched another cast of the same card: stack=%d", g.StackSize())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: old.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("exile cast did not trigger: stack=%d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original caster")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.PhaseOut(watcher.ID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if b.Life() != 22 {
		t.Fatalf("copied event caster lost: life=%d", b.Life())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: old.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("phased source triggered: stack=%d", g.StackSize())
	}
}

func TestSpellCastFromZoneRealCastAndFailedCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().FromZone(ZoneHand), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	unpaid := NewInstant("Unpaid", "{U}")
	unpaid.SetOwner(b.PlayerID())
	b.AddToHand(unpaid)
	if err := g.CastSpellByName(b.PlayerID(), unpaid.Name(), nil); err == nil {
		t.Fatal("unpaid spell was cast")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed cast produced a trigger")
	}
	free := NewInstant("Free", "")
	free.SetOwner(b.PlayerID())
	b.AddToHand(free)
	if err := g.CastSpellByName(b.PlayerID(), free.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("hand cast stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("hand cast life=%d", a.Life())
	}
}

func TestSpellCastFromZoneRealExileCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().FromZone(ZoneExile), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Exiled", "{1}")
	card.SetOwner(b.PlayerID())
	g.ExileCard(card, uuid.Nil)
	if err := g.CastCardFromZoneWithoutPaying(b.PlayerID(), card.ID(), ZoneExile, nil, 0); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("exile cast stack=%d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("exile cast life=%d", a.Life())
	}
}

func TestSpellCastFromZoneDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(SpellCast().FromZone(ZoneGraveyard), GainLifeFor(You, Fixed(1))))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(SpellCast().FromZone(ZoneExile)); !strings.Contains(got.Text, "exile") {
					t.Fatalf("inspection=%+v", got)
				}
			},
		},
		{
			Name:          "zone any",
			Card:          NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().FromZone(ZoneAny), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "zone",
		},
		{
			Name:          "zone stack",
			Card:          NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().FromZone(ZoneStack), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "zone",
		},
		{
			Name:          "invalid zone enum",
			Card:          NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().FromZone(Zone(-1)), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "zone",
		},
		{
			Name:          "duplicate zone",
			Card:          NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().FromZone(ZoneHand).FromZone(ZoneExile), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "zone",
		},
	})
}

// --- from composable_spell_controller_test.go ---
func TestControllerOfSpellPaymentUsesStackController(t *testing.T) {
	a := NewBasePlayer("A")
	b := &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	victim := stackTestSpell("Victim", a.PlayerID())
	target := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(target)
	spell := DeclareSpellTarget("spell")
	body := UnlessPaid(ControllerOfSpell(spell), Costs(LifePayCost(3)), Counter(spell))
	ability := Spell(body, WithTargets(spell))
	card := NewInstant("Test", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Test", card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if got := b.Life(); got != 17 || a.Life() != 20 || g.stack.FindByID(target.ID) == nil {
		t.Fatalf("payment used wrong player: A=%d B=%d targetPresent=%t", a.Life(), got, g.stack.FindByID(target.ID) != nil)
	}
}

func TestControllerOfSpellAbsenceCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := stackTestSpell("First", a.PlayerID())
	secondCard := stackTestSpell("Second", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: a.PlayerID()}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	spell := DeclareSpellTarget("spell")
	ref := ControllerOfSpell(spell)
	ctx := &EffectContext{Game: g, targetBindings: map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: first.ID}}}}
	if got, ok := ref.resolvePlayer(ctx); !ok || got != a.PlayerID() {
		t.Fatalf("first controller = %v, %t", got, ok)
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	cloneCtx.targetBindings = map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: second.ID}}}
	if got, ok := ref.resolvePlayer(&cloneCtx); !ok || got != b.PlayerID() {
		t.Fatalf("clone controller = %v, %t", got, ok)
	}
	if got, ok := ref.resolvePlayer(ctx); !ok || got != a.PlayerID() {
		t.Fatalf("original controller changed = %v, %t", got, ok)
	}
	g.stack.RemoveByID(first.ID)
	recast := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: b.PlayerID()}
	g.pushStack(recast)
	if _, ok := ref.resolvePlayer(ctx); ok {
		t.Fatal("old stack reference resolved after same card was cast again")
	}
}

func TestControllerOfSpellCopiedRetargetedFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := stackTestSpell("First", a.PlayerID())
	secondCard := stackTestSpell("Second", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: a.PlayerID()}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	spell := DeclareSpellTarget("spell")
	ability := Spell(GainLifeFor(ControllerOfSpell(spell), Fixed(1)), WithTargets(spell))
	card := NewInstant("Reward controller", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = g.zoneOfTarget(second.ID)
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 21 || obj.Targets[0] != first.ID {
		t.Fatal("copied stack frame did not use its retargeted spell controller")
	}
	g.ResolveTopOfStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("original stack frame lost its spell controller")
	}
}

func TestControllerOfSpellTargetLegalityAtResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewArtifact("Victim", "")
	victim.SetOwner(b.PlayerID())
	target := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(target)
	allowed := true
	filter := NewCardFilter("artifact", func(card Card) bool { return allowed && card.HasType(TypeArtifact) })
	spell := DeclareSpellTarget("spell", filter)
	ability := Spell(GainLifeFor(ControllerOfSpell(spell), Fixed(2)), WithTargets(spell))
	card := NewInstant("Test", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	allowed = false
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("illegal spell target supplied a controller at resolution")
	}
}

func TestControllerOfSpellDeclarationValidation(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil",
			Card:          NewInstant("Test", "", WithAbility(Spell(GainLifeFor(ControllerOfSpell(nil), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: ".spell",
		},
		{
			Name:          "undeclared",
			Card:          NewInstant("Test", "", WithAbility(Spell(GainLifeFor(ControllerOfSpell(spell), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: ".spell",
		},
		{
			Name: "valid",
			Card: NewInstant("Test", "", WithAbility(Spell(GainLifeFor(ControllerOfSpell(spell), Fixed(1)), WithTargets(spell)))),
		},
	})
}

// --- from composable_stack_test.go ---
func stackTestSpell(name string, owner uuid.UUID) Card {
	card := NewInstant(name, "")
	card.SetOwner(owner)
	return card
}

func TestTypedCounterUsesStackIdentityAndTargetOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := stackTestSpell("Victim", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(first)
	copyObj := g.CopyStackObjectDirect(first, b.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	if got := g.zoneOfTarget(copyObj.ID); got != ZoneStack {
		t.Fatalf("copy target zone = %v, want stack", got)
	}
	player := DeclarePlayerTarget("player")
	spell := DeclareSpellTarget("spell")
	body := Sequence(GainLifeFor(player, Fixed(1)), Counter(spell))
	ability := Spell(body, WithTargets(player, spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Counter", card); err != nil {
		t.Fatal(err)
	}
	possible := spell.Possible(a.PlayerID(), card, g)
	if !slices.Contains(possible, first.ID) || !slices.Contains(possible, copyObj.ID) {
		t.Fatalf("possible stack objects = %v", possible)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{a.PlayerID(), copyObj.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if g.stack.FindByID(copyObj.ID) != nil || g.stack.FindByID(first.ID) == nil || len(b.Graveyard()) != 0 || a.Life() != 21 {
		t.Fatal("counter changed the wrong stack object or lost the earlier effect")
	}
}

func TestTypedCounterLegalityAndUncounterable(t *testing.T) {
	for _, change := range []string{"none", "removed", "uncounterable", "filter changed"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			options := []CardOption{}
			if change == "uncounterable" {
				options = append(options, WithUncounterable())
			}
			victim := NewArtifact("Victim", "", options...)
			victim.SetOwner(b.PlayerID())
			first := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
			g.pushStack(first)
			allow := true
			filter := NewCardFilter("artifact", func(c Card) bool { return allow && c.HasType(TypeArtifact) })
			spell := DeclareSpellTarget("spell", filter)
			ability := Spell(Sequence(Counter(spell), GainLifeFor(You, Fixed(1))), WithTargets(spell))
			card := NewInstant("Counter", "", WithAbility(ability))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition("Counter", card); err != nil {
				t.Fatal(err)
			}
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			if change == "removed" {
				g.stack.RemoveByID(first.ID)
			}
			if change == "filter changed" {
				allow = false
			}
			g.ResolveTopOfStack()
			wantLife := 21
			if change == "removed" || change == "filter changed" {
				wantLife = 20
			}
			if a.Life() != wantLife {
				t.Fatalf("life = %d, want %d", a.Life(), wantLife)
			}
			countered := change == "none"
			if (g.stack.FindByID(first.ID) == nil) != (countered || change == "removed") {
				t.Fatal("wrong stack object status")
			}
			if (len(b.Graveyard()) == 1) != countered {
				t.Fatal("wrong graveyard status")
			}
		})
	}
}

func TestTypedCounterBindingValidation(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil",
			Card:          NewInstant("Test", "", WithAbility(Spell(Counter(nil)))),
			WantErr:       true,
			WantErrSubstr: ".spell",
		},
		{
			Name:          "undeclared",
			Card:          NewInstant("Test", "", WithAbility(Spell(Counter(spell)))),
			WantErr:       true,
			WantErrSubstr: ".spell",
		},
		{
			Name: "valid",
			Card: NewInstant("Test", "", WithAbility(Spell(Counter(spell), WithTargets(spell)))),
		},
	})
}

func TestTypedCounterRecastAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := stackTestSpell("Victim", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(first)
	spell := DeclareSpellTarget("spell")
	ability := Spell(Counter(spell), WithTargets(spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if clone.stack.FindByID(first.ID) != nil || g.stack.FindByID(first.ID) == nil {
		t.Fatal("clone changed original stack")
	}
	g.stack.RemoveByID(first.ID)
	second := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(second)
	if slices.Contains(spell.Possible(a.PlayerID(), card, g), first.ID) || !slices.Contains(spell.Possible(a.PlayerID(), card, g), second.ID) {
		t.Fatal("spell target did not distinguish casts")
	}
	g.stack.RemoveByID(obj.ID)
	g.ResolveStackObject(obj)
	if g.stack.FindByID(second.ID) == nil || len(b.Graveyard()) != 0 {
		t.Fatal("old target countered a later cast")
	}
}

func TestTypedCounterRejectsInvalidAnnouncement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := DeclareSpellTarget("spell", NewCardFilter("artifact", func(c Card) bool { return c.HasType(TypeArtifact) }))
	ability := Spell(Counter(spell), WithTargets(spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	wrong := stackTestSpell("Wrong", b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: wrong.ID(), Card: wrong, Controller: b.PlayerID()}
	g.pushStack(obj)
	if err := g.validateActionTargets(a.PlayerID(), card, ability.Targets(), []uuid.UUID{obj.ID}, 0, "spell"); err == nil {
		t.Fatal("accepted a spell outside the target filter")
	}
	if err := g.validateActionTargets(a.PlayerID(), card, ability.Targets(), []uuid.UUID{uuid.New()}, 0, "spell"); err == nil {
		t.Fatal("accepted an absent stack object")
	}
	if g.stack.FindByID(obj.ID) == nil {
		t.Fatal("invalid announcement changed the stack")
	}
}

func TestTypedCounterCopiedRetargetedInstance(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := stackTestSpell("First", b.PlayerID())
	secondCard := stackTestSpell("Second", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: b.PlayerID()}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	spell := DeclareSpellTarget("spell")
	ability := Spell(Counter(spell), WithTargets(spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = g.zoneOfTarget(second.ID)
	g.ResolveTopOfStack()
	if g.stack.FindByID(second.ID) != nil || g.stack.FindByID(first.ID) == nil || obj.Targets[0] != first.ID {
		t.Fatal("copied counter did not use its own target frame")
	}
	g.ResolveTopOfStack()
	if g.stack.FindByID(first.ID) != nil || len(b.Graveyard()) != 2 {
		t.Fatal("original counter did not retain its target")
	}
}

func TestTypedCounterModeScope(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Modes", "", WithAbility(SpellModes(ModeOf("counter", Counter(spell), WithTargets(spell)), ModeOf("life", GainLifeFor(You, Fixed(1)))))),
		},
		{
			Name:    "binding from another mode",
			Card:    NewInstant("Modes", "", WithAbility(SpellModes(ModeOf("counter", Counter(spell)), ModeOf("life", GainLifeFor(You, Fixed(1)), WithTargets(spell))))),
			WantErr: true,
		},
	})
}

func TestCounterSpellOnStackPreservesAbilityCountering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Controller: b.PlayerID(), IsAbility: true}
	g.pushStack(ability)
	g.CounterSpellOnStack(ability.SourceID)
	if g.stack.FindByID(ability.ID) != nil {
		t.Fatal("legacy counter did not remove the ability")
	}
}

// --- from composable_tap_each_test.go ---
func TestTapEachTransitionsTogether(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	already := g.PutOnBattlefield(NewCreature("Already", "", 1, 1), b.PlayerID())
	g.TapPermanent(already)
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
	g.PhaseOut(phased.ID())
	var events, intermediate int
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(EvtTapped, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(event *GameEvent, game GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == first.ID() || event.SourceID == second.ID() {
			events++
			if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
				intermediate++
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := TapEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if events != 2 || intermediate != 0 || !first.Tapped || !second.Tapped || !already.Tapped || phased.Tapped {
		t.Fatalf("events %d, intermediate %d, tapped %v/%v/%v/%v", events, intermediate, first.Tapped, second.Tapped, already.Tapped, phased.Tapped)
	}
}

func TestTapEachCloneAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Blue", "{U}", 1, 1)
	g.PutOnBattlefield(card, a.PlayerID())
	clone := g.Clone()
	if err := TapEach(MatchingPermanents(HasColorFilter(Blue))).Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(card.ID()).Tapped || g.FindPermanent(card.ID()).Tapped {
		t.Fatal("tap changed original game or missed clone")
	}
	valid := NewInstant("Valid", "", Spell(TapEach(MatchingPermanents(HasColorFilter(Blue)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: valid,
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(valid.Abilities()[0]); len(got.Children) == 0 {
					t.Fatal("tap collection missing from inspection")
				}
			},
		},
		{
			Name:    "nil permanent set",
			Card:    NewInstant("Bad", "", Spell(TapEach(nil))),
			WantErr: true,
		},
	})
}

type fixedTapSet struct{ permanents []*Permanent }

func (fixedTapSet) permanentSet()                                   {}
func (s fixedTapSet) resolvePermanents(*EffectContext) []*Permanent { return s.permanents }

func TestTapEachSkipsStaleAndDuplicateObjects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	if err := TapEach(fixedTapSet{[]*Permanent{old, current, current, other}}).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if !current.Tapped || !other.Tapped || old.Tapped {
		t.Fatal("stale or duplicate member changed tap outcome")
	}
}

func TestTapEachTargetGroupKeepsAnnouncementIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Target", "", 1, 1)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	if err := TapEach(targets).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if current.Tapped {
		t.Fatal("returned target was tapped through an old target group")
	}
}

// --- from composable_tap_result_test.go ---
func TestTapResultTransitionAndBranch(t *testing.T) {
	for _, alreadyTapped := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		p := NewCreature("Subject", "", 2, 2)
		g.PutOnBattlefield(p, a.PlayerID())
		if alreadyTapped {
			g.TapPermanent(g.FindPermanent(p.ID()))
		}
		result := BindNumber("tap result")
		body := Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(3))))
		if err := body.Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		want := 23
		if alreadyTapped {
			want = 20
		}
		wantResult := 1
		if alreadyTapped {
			wantResult = 0
		}
		if a.Life() != want || g.resolution.numberBindings[result] != wantResult {
			t.Fatalf("already tapped %v: life %d, result %d", alreadyTapped, a.Life(), g.resolution.numberBindings[result])
		}
	}
}

func TestTapResultAbsentPhasedAndReturned(t *testing.T) {
	for _, state := range []string{"absent", "phased", "returned"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := NewCreature("Subject", "", 2, 2)
			g.PutOnBattlefield(p, a.PlayerID())
			result := BindNumber("tap result")
			g.resolution.numberBindings = map[*NumberBinding]int{result: 1}
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: g.FindPermanent(p.ID()).incarnationID}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(p, a.PlayerID())
			}
			body := Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(3))))
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || g.resolution.numberBindings[result] != 0 {
				t.Fatal("absent incarnation reused a tap result")
			}
		})
	}
}

func TestTapResultValidationAndClone(t *testing.T) {
	result := BindNumber("tap result")
	inner := BindNumber("inner")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2)))))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: valid,
		},
		{
			Name:          "nil binding",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), TapSubject(This, TapAmountInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "duplicate",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), TapSubject(This, TapAmountInto(result), TapAmountInto(result))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "missing producer",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil value",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfPositive(nil, GainLifeFor(You, Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil body",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfPositive(Fixed(1), nil)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "branch escape",
			Card:          NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Sequence(TapSubject(This, TapAmountInto(result)), IfPositive(FromNumber(result), SnapshotNumber(Fixed(3), inner)), GainLifeFor(You, FromNumber(inner)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})

	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(valid, a.PlayerID())
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), valid.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 || g.FindPermanent(valid.ID()).Tapped {
		t.Fatal("clone shared tap state or result")
	}
}

func TestTapResultTargetedSpellAndStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	result := BindNumber("tap result")
	body := Sequence(TapSubject(target, TapAmountInto(result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	ability := Spell(body, WithTargets(target))
	spell := NewInstant("Synthetic tap", "", WithAbility(ability))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if copied == nil {
		t.Fatal("stack copy failed")
	}
	if len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
		t.Fatalf("copy targets = %v", copied.Targets)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("copied or cloned result used the wrong controller: clone A=%d B=%d, original A=%d B=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life(), b.Life())
	}
	if !clone.FindPermanent(first.ID()).Tapped || !clone.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copied or cloned target changed the original game")
	}

	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("returned target used a stale tap result")
	}
}

// --- from composable_token_result_test.go ---
func TestCreateOneTokenBindsCreatedIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	created := BindPermanent("created token")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenSubtypes("Cat"), TokenColors(White), TokenPT(2, 2))
	body := Sequence(CreateOneToken(You, def, created), AddCountersTo(From(created), P1P1, Fixed(1)))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, body, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("clone changed original game")
	}
	if err := ApplyEffect(g, body, a.PlayerID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 1 || tokens[0].ControllerID() != b.PlayerID() || tokens[0].Counters[P1P1] != 1 {
		t.Fatalf("created tokens: %+v", tokens)
	}
}

func TestCreateOneTokenResultAbsentAndCleared(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	created := BindPermanent("created token")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(2, 2))
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	effect := Sequence(Snapshot(This, created), CreateOneToken(ControllerOf(This), def, created), AddCountersTo(From(created), P1P1, Fixed(1)))
	g.PhaseOut(source.ID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 0 || g.resolution.permanentBindings[created].id != (capturedPermanent{}).id {
		t.Fatal("absent player retained stale token binding")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	token := g.FilterBattlefield(IsToken)[0]
	if token.Counters[P1P1] != 1 {
		t.Fatal("token result was absent")
	}
	g.RemoveFromBattlefield(token)
	if _, ok := From(created).resolvePermanent(&EffectContext{Game: g}); ok {
		t.Fatal("departed token remained present")
	}
}

func TestCreateOneTokenResultDeclaration(t *testing.T) {
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(2, 2))
	result := BindPermanent("token")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid inspection",
			Card: NewInstant("Valid", "", Spell(CreateOneToken(You, def, result))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(CreateOneToken(You, def, result)); got.Role != "resolving-effect" || len(got.Children) != 3 {
					t.Fatalf("inspection: %+v", got)
				}
			},
		},
		{
			Name:    "nil result",
			Card:    NewInstant("Bad", "", Spell(CreateOneToken(You, def, nil))),
			WantErr: true,
		},
		{
			Name:    "unnamed result",
			Card:    NewInstant("Bad", "", Spell(CreateOneToken(You, def, BindPermanent("")))),
			WantErr: true,
		},
		{
			Name:    "nil player",
			Card:    NewInstant("Bad", "", Spell(CreateOneToken(nil, def, result))),
			WantErr: true,
		},
		{
			Name:    "nil definition",
			Card:    NewInstant("Bad", "", Spell(CreateOneToken(You, nil, result))),
			WantErr: true,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Bad", "", Spell(AddCountersTo(From(result), P1P1, Fixed(1)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestCreateOneTokenResultClearsRejectedEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Old", "", 1, 1), a.PlayerID())
	result := BindPermanent("created land")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{result: {old.ID(), old.incarnationID}}
	g.effects.Rules.SetLandsCantEnter(true)
	def := DefineToken("Land", TokenTypes(TypeLand))
	if err := CreateOneToken(You, def, result).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.permanentBindings[result]; ok || len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("rejected entry retained a token result")
	}
}

// --- from composable_token_set_test.go ---
func TestCreateTokenCopiesIntoKeepsCreatedSet(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Cat", "", 1, 1), b.PlayerID())
	result := BindCreatedTokens("cats")
	member := BindPermanent("cat")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(0, 0))
	body := Sequence(CreateTokenCopiesInto(You, Fixed(3), def, result), ForEach(FromCreatedTokens(result), member, AddCountersTo(From(member), P1P1, Fixed(1))))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, a.PlayerID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.CheckStateBasedActions()
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 3 || old.Counters[P1P1] != 0 {
		t.Fatal("result set included old permanent or missed created tokens")
	}
	for _, token := range tokens {
		if token.ControllerID() != b.PlayerID() || token.Counters[P1P1] != 1 {
			t.Fatal("created token identity or controller changed")
		}
	}
}

func TestCreatedTokenSetAbsentMembersAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(1, 1))
	if err := CreateTokenCopiesInto(You, Fixed(2), def, result).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	values := g.resolution.createdTokenBindings[result]
	if len(values) != 2 {
		t.Fatal("wrong number of saved tokens")
	}
	clone := g.Clone()
	clone.resolution.createdTokenBindings[result][0] = capturedPermanent{}
	if g.resolution.createdTokenBindings[result][0] != values[0] {
		t.Fatal("clone shared token result storage")
	}
	first := g.FindPermanent(values[0].id)
	second := g.FindPermanent(values[1].id)
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(first.Card, a.PlayerID())
	g.PhaseOut(second.ID())
	if got := FromCreatedTokens(result).resolvePermanents(&EffectContext{Game: g}); len(got) != 0 {
		t.Fatal("returned or phased token remained in result set")
	}
	if got := FromCreatedTokens(result).resolvePermanents(&EffectContext{Game: clone}); len(got) != 1 {
		t.Fatal("clone did not keep independent token identities")
	}
}

func TestCreatedTokenSetRegistration(t *testing.T) {
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(1, 1))
	member := BindPermanent("member")
	read := ForEach(FromCreatedTokens(result), member, GainLifeFor(You, Fixed(1)))
	branch := Sequence(IfState(PermanentIsTapped(This), CreateTokenCopiesInto(You, Fixed(2), def, result)), read)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid inspection",
			Card: NewInstant("Valid", "", Spell(CreateTokenCopiesInto(You, Fixed(2), def, result))),
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(CreateTokenCopiesInto(You, Fixed(2), def, result)); got.Role != "resolving-effect" || len(got.Children) != 4 {
					t.Fatalf("inspection: %+v", got)
				}
			},
		},
		{
			Name:    "nil player",
			Card:    NewInstant("Bad", "", Spell(CreateTokenCopiesInto(nil, Fixed(2), def, result))),
			WantErr: true,
		},
		{
			Name:    "nil count",
			Card:    NewInstant("Bad", "", Spell(CreateTokenCopiesInto(You, nil, def, result))),
			WantErr: true,
		},
		{
			Name:    "nil def",
			Card:    NewInstant("Bad", "", Spell(CreateTokenCopiesInto(You, Fixed(2), nil, result))),
			WantErr: true,
		},
		{
			Name:    "nil result",
			Card:    NewInstant("Bad", "", Spell(CreateTokenCopiesInto(You, Fixed(2), def, nil))),
			WantErr: true,
		},
		{
			Name:    "empty result name",
			Card:    NewInstant("Bad", "", Spell(CreateTokenCopiesInto(You, Fixed(2), def, BindCreatedTokens("")))),
			WantErr: true,
		},
		{
			Name:          "missing producer",
			Card:          NewInstant("Bad", "", Spell(read)),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "non-dominating producer",
			Card:          NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), branch))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestCreatedTokenSetClearsFailedBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(1, 1))
	effect := CreateTokenCopiesInto(You, Fixed(2), def, result)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.createdTokenBindings[result]) != 2 {
		t.Fatal("missing first batch")
	}
	ctx.Controller = b.PlayerID()
	if err := CreateTokenCopiesInto(You, Fixed(0), def, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.createdTokenBindings[result]) != 0 {
		t.Fatal("zero batch retained stale tokens")
	}
	land := DefineToken("Land", TokenTypes(TypeLand))
	g.effects.Rules.SetLandsCantEnter(true)
	if err := CreateTokenCopiesInto(You, Fixed(2), land, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.resolution.createdTokenBindings[result]) != 0 {
		t.Fatal("rejected entries appeared in result")
	}
}

func TestCreatedTokenSetDelayedCaptureAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("tokens")
	def := DefineToken("Cat", TokenTypes(TypeCreature), TokenPT(1, 1))
	effect := Sequence(CreateTokenCopiesInto(You, Fixed(2), def, result), AtNext(EachEndStep, ReturnEachToHand(FromCreatedTokens(result))))
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", Spell(effect))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("tokens absent before delayed step")
	}
	clone := g.Clone()
	clone.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if len(clone.FilterBattlefield(IsToken)) != 0 || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("delayed result set was not captured or clone was not isolated")
	}
}

// --- from composable_token_test.go ---
func TestComposableTokenCreationAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	types := []CardType{TypeCreature}
	subtypes := []string{"Spirit"}
	colors := []Color{White}
	keywords := []Keyword{Flying}
	ability := Activate(Costs(), GainLifeFor(You, Fixed(1)))
	def := DefineToken("Spirit", TokenTypes(types...), TokenSubtypes(subtypes...), TokenColors(colors...), TokenPT(2, 3), TokenKeywords(keywords...), TokenAbilities(ability))
	types[0], subtypes[0], colors[0], keywords[0] = TypeArtifact, "Goblin", Red, Trample
	effect := CreateTokenCopies(ActivePlayer, Fixed(2), def)
	if err := ValidateCardDefinition("Maker", NewInstant("Maker", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, a.PlayerID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens", len(tokens))
	}
	for _, token := range tokens {
		if token.ControllerID() != a.PlayerID() || token.Card.Owner() != a.PlayerID() || !token.HasType(TypeCreature) || !token.HasSubType("Spirit") || !slices.Contains(token.Colors(), White) || token.CurrentPower(g) != 2 || token.CurrentToughness(g) != 3 || !token.HasKeyword(Flying) {
			t.Fatalf("wrong token characteristics: %+v", token)
		}
	}
	if tokens[0].RuntimeAbilities[0] == tokens[1].RuntimeAbilities[0] || tokens[0].RuntimeAbilities[0].AbilityID() == tokens[1].RuntimeAbilities[0].AbilityID() {
		t.Fatal("token abilities share identity")
	}
	clone := g.Clone()
	if len(clone.FilterBattlefield(IsToken)) != 2 || len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("clone changed tokens")
	}
	if err := ApplyEffect(g, CreateTokenCopies(You, Fixed(0), def), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, CreateTokenCopies(You, Fixed(-1), def), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("nonpositive count created tokens")
	}
}

func TestComposableTokenDefinitionValidation(t *testing.T) {
	outer := DeclarePermanentTarget("outer", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "name",
			Card:    NewInstant("name", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("", TokenTypes(TypeCreature), TokenPT(1, 1))), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "types",
			Card:    NewInstant("types", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad")), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "pt",
			Card:    NewInstant("pt", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad", TokenTypes(TypeCreature))), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "color",
			Card:    NewInstant("color", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad", TokenTypes(TypeArtifact), TokenColors(AnyColor))), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "option",
			Card:    NewInstant("option", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad", TokenTypes(TypeArtifact), nil)), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "duplicate",
			Card:    NewInstant("duplicate", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad", TokenTypes(TypeArtifact), TokenTypes(TypeCreature))), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "nil ability",
			Card:    NewInstant("nil ability", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad", TokenTypes(TypeArtifact), TokenAbilities(nil))), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:    "outer binding",
			Card:    NewInstant("outer binding", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), DefineToken("Bad", TokenTypes(TypeCreature), TokenPT(1, 1), TokenAbilities(Activate(Costs(), Destroy(outer))))), WithTargets(outer)))),
			WantErr: true,
		},
		{
			Name:          "player",
			Card:          NewInstant("player", "", WithAbility(Spell(CreateTokenCopies(nil, Fixed(1), DefineToken("Stone", TokenTypes(TypeArtifact)))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "count",
			Card:          NewInstant("count", "", WithAbility(Spell(CreateTokenCopies(You, nil, DefineToken("Stone", TokenTypes(TypeArtifact)))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "token",
			Card:          NewInstant("token", "", WithAbility(Spell(CreateTokenCopies(You, Fixed(1), nil)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
	})
}

func TestComposableTokenDynamicCountAndEntryObservation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 3, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	seen := 0
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(NewTriggered(EvtZoneChange, false).SetCondition(func(evt *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.ToZone == ZoneBattlefield {
			if p := reader.FindPermanent(evt.SourceID); p != nil && p.IsToken && len(reader.FilterBattlefield(IsToken)) == 3 {
				seen++
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	def := DefineToken("Bird", TokenTypes(TypeCreature), TokenSubtypes("Bird"), TokenColors(Blue), TokenPT(1, 1), TokenKeywords(Flying))
	if err := ApplyEffect(g, CreateTokenCopies(You, PowerOf(This), def), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 3 || seen == 0 {
		t.Fatalf("tokens = %d, observations = %d", len(g.FilterBattlefield(IsToken)), seen)
	}
}

// --- from composable_union_destruction_test.go ---
func TestUnionDestructionRegeneratesOnce(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Artifact enchantment creature", "", 2, 2)
	card.AddType(TypeArtifact)
	card.AddType(TypeEnchantment)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.AddRegenerationShield(p.ID())
	clone := g.Clone()
	if err := DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment))).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	got := clone.FindPermanent(p.ID())
	if got == nil || !got.Tapped {
		t.Fatal("one shield must replace the entire destruction of a permanent with three matching types")
	}
	if g.FindPermanent(p.ID()) == nil || p.Tapped {
		t.Fatal("destruction changed the original game")
	}
	clone.DestroyPermanent(got)
	if clone.FindPermanent(p.ID()) != nil {
		t.Fatal("regeneration shield was not consumed")
	}
}

func TestUnionDestructionIsSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Creature", "", 2, 2)
	artifact := NewArtifact("Artifact", "")
	observer := NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(creature.ID()) == nil && game.FindPermanent(artifact.ID()) != nil
		})))
	for _, card := range []Card{creature, artifact, observer} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(creature.ID()) != nil || g.FindPermanent(artifact.ID()) != nil {
		t.Fatal("matching permanents survived")
	}
	if g.FindPermanent(observer.ID()) == nil {
		t.Fatal("state trigger observed separate destruction batches")
	}
}

func TestUnionDestructionMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment)))
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	var destroyed, survivors []*Permanent
	for _, owner := range []*BasePlayer{a, b} {
		for _, typ := range []CardType{TypeArtifact, TypeCreature, TypeEnchantment, TypeLand, TypePlaneswalker} {
			for _, status := range []string{"ordinary", "indestructible", "phased"} {
				card := NewArtifact(status, "")
				card.types = []CardType{typ}
				card.power, card.toughness = 2, 2
				card.SetOwner(owner.PlayerID())
				p := g.PutOnBattlefield(card, owner.PlayerID())
				if status == "indestructible" {
					p.GrantBaseAttr(Indestructible)
				}
				if status == "phased" {
					p.PhasedOut = true
				}
				if status == "ordinary" && typ != TypeLand && typ != TypePlaneswalker {
					destroyed = append(destroyed, p)
				} else {
					survivors = append(survivors, p)
				}
			}
		}
	}
	clone := g.Clone()
	if err := effect.Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	for _, p := range destroyed {
		if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
			t.Fatal("wrong membership or clone isolation")
		}
	}
	for _, p := range survivors {
		if clone.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatal("destroyed an excluded permanent")
		}
	}
}

// --- from composable_untap_each_test.go ---
func TestUntapEachBatchAndStun(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	stunned := g.PutOnBattlefield(NewCreature("Stunned", "", 1, 1), a.PlayerID())
	already := g.PutOnBattlefield(NewCreature("Already", "", 1, 1), b.PlayerID())
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
	for _, p := range []*Permanent{first, second, stunned, phased} {
		g.TapPermanent(p)
	}
	stunned.AddCounter(Stun, 1)
	g.PhaseOut(phased.ID())
	var events, intermediate int
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(EvtBecameUntapped, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(event *GameEvent, game GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == first.ID() || event.SourceID == second.ID() {
			events++
			if game.FindPermanent(first.ID()).Tapped || game.FindPermanent(second.ID()).Tapped {
				intermediate++
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := UntapEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if events != 2 || intermediate != 0 || first.Tapped || second.Tapped || !stunned.Tapped || stunned.Counters[Stun] != 0 || already.Tapped || !phased.Tapped {
		t.Fatalf("events %d, intermediate %d, state %v/%v/%v/%d/%v/%v", events, intermediate, first.Tapped, second.Tapped, stunned.Tapped, stunned.Counters[Stun], already.Tapped, phased.Tapped)
	}
}

func TestUntapEachIdentityCloneAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.TapPermanent(old)
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	g.TapPermanent(current)
	g.TapPermanent(other)
	clone := g.Clone()
	if err := UntapEach(fixedTapSet{[]*Permanent{old, current, current, other}}).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(current.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped || !current.Tapped || !other.Tapped {
		t.Fatal("stale or duplicate member changed the original or clone")
	}
	valid := NewInstant("Valid", "", Spell(UntapEach(ControlledPermanents(IsCreature))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: valid,
			Inspect: func(t *testing.T) {
				if got := InspectDefinition(valid.Abilities()[0]); len(got.Children) == 0 {
					t.Fatal("untap collection missing from inspection")
				}
			},
		},
		{
			Name:    "nil permanent set",
			Card:    NewInstant("Bad", "", Spell(UntapEach(nil))),
			WantErr: true,
		},
	})
}

func TestUntapEachTargetGroupKeepsAnnouncementIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Target", "", 1, 1)
	old := g.PutOnBattlefield(card, b.PlayerID())
	g.TapPermanent(old)
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	g.TapPermanent(current)
	if err := UntapEach(targets).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !current.Tapped {
		t.Fatal("returned target was untapped through an old target group")
	}
}

// --- from composable_untap_result_test.go ---
func TestUntapResultTransitionAndStun(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Subject", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindNumber("untapped")
	body := Sequence(UntapPermanentWithResult(This, UntappedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	for _, tc := range []struct {
		tapped, stun bool
		want         int
	}{
		{true, false, 1}, {false, false, 0}, {true, true, 0}, {true, false, 1},
	} {
		p.Tapped = tc.tapped
		if tc.stun {
			p.AddCounter(Stun, 1)
		}
		before := a.Life()
		if err := body.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if got := g.resolution.numberBindings[result]; got != tc.want || a.Life() != before+2*tc.want {
			t.Fatalf("result=%d life=%d, want %d", got, a.Life(), tc.want)
		}
		if tc.stun && (p.Counters[Stun] != 0 || !p.Tapped) {
			t.Fatal("stun did not replace untap")
		}
	}
}

func TestUntapResultTargetAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	p.Tapped = true
	result := BindNumber("untapped")
	target := DeclarePermanentTarget("target", IsCreature)
	spell := NewInstant("Untap test", "", Spell(Sequence(
		UntapPermanentWithResult(target, UntappedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target)))
	a.AddToHand(spell)
	g.SetStep(PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).AddCounter(Stun, 1)
	if g.FindPermanent(p.ID()).Counters[Stun] != 0 {
		t.Fatal("clone counter reached original")
	}
	clone.ResolveStack()
	g.ResolveStack()
	if a.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 20 || g.FindPermanent(p.ID()).Tapped || !clone.FindPermanent(p.ID()).Tapped {
		t.Fatalf("target untap outcome: original life=%d clone life=%d original tapped=%v clone tapped=%v", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), g.FindPermanent(p.ID()).Tapped, clone.FindPermanent(p.ID()).Tapped)
	}
}

func TestUntapResultAbsentPhasedAndReturnedTarget(t *testing.T) {
	for _, state := range []string{"absent", "phased", "returned", "present"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			victim := NewCreature("Victim", "", 2, 2)
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			p.Tapped = true
			result := BindNumber("untapped")
			target := DeclarePermanentTarget("target", IsCreature)
			spell := NewInstant("Untap", "", Spell(Sequence(
				UntapPermanentWithResult(target, UntappedThisWayInto(result)),
				IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
			), WithTargets(target)))
			a.AddToHand(spell)
			g.SetStep(PrecombatMain)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(victim, b.PlayerID()).Tapped = true
			}
			g.ResolveStack()
			want := 20
			if state == "present" {
				want = 22
			}
			if a.Life() != want {
				t.Fatalf("life=%d, want %d", a.Life(), want)
			}
		})
	}
}

func TestUntapResultStackCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	result := BindNumber("untapped")
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Sequence(
		UntapPermanentWithResult(target, UntappedThisWayInto(result)),
		IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))),
	), WithTargets(target))
	spell := NewInstant("Untap copy", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	p.Tapped = true
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 22 || g.FindPermanent(p.ID()).Tapped {
		t.Fatalf("life=%d tapped=%v", a.Life(), g.FindPermanent(p.ID()).Tapped)
	}
}

func TestUntapResultDeclaration(t *testing.T) {
	b := BindNumber("untapped")
	good := NewArtifact("Good", "", WithAbility(Activate(Costs(), Sequence(UntapPermanentWithResult(This, UntappedThisWayInto(b)), GainLifeFor(You, FromNumber(b))))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: good,
		},
		{
			Name:          "missing result option",
			Card:          NewArtifact("Bad", "", WithAbility(Activate(Costs(), UntapPermanentWithResult(This)))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "nil subject",
			Card:          NewArtifact("Bad", "", WithAbility(Activate(Costs(), UntapPermanentWithResult(nil, UntappedThisWayInto(b))))),
			WantErr:       true,
			WantErrSubstr: "subject",
		},
		{
			Name:          "nil binding in option",
			Card:          NewArtifact("Bad", "", WithAbility(Activate(Costs(), UntapPermanentWithResult(This, UntappedThisWayInto(nil))))),
			WantErr:       true,
			WantErrSubstr: "binding",
		},
		{
			Name:          "empty binding name",
			Card:          NewArtifact("Bad", "", WithAbility(Activate(Costs(), UntapPermanentWithResult(This, UntappedThisWayInto(BindNumber("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "duplicate result option",
			Card:          NewArtifact("Bad", "", WithAbility(Activate(Costs(), UntapPermanentWithResult(This, UntappedThisWayInto(b), UntappedThisWayInto(b))))),
			WantErr:       true,
			WantErrSubstr: "once",
		},
		{
			Name:          "used before produced",
			Card:          NewArtifact("Bad", "", WithAbility(Activate(Costs(), Sequence(GainLifeFor(You, FromNumber(b)), UntapPermanentWithResult(This, UntappedThisWayInto(b)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_untap_test.go ---
func TestComposableSourceUntapLifetime(t *testing.T) {
	for _, change := range []string{"unchanged", "absent", "returned", "phased", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", 1, 1, WithAbility(At(YourUpkeep, UntapPermanent(This))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.TapPermanent(g.FindPermanent(source.ID()))
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.stack.Size() != 1 {
				t.Fatal("missing trigger")
			}
			switch change {
			case "absent", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returned" {
					g.PutOnBattlefield(source, a.PlayerID())
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			clone.ResolveStack()
			if change == "absent" {
				return
			}
			if change == "phased" {
				clone.PhaseIn([]uuid.UUID{source.ID()})
				g.PhaseIn([]uuid.UUID{source.ID()})
			}
			acted := change != "returned" && change != "phased"
			if clone.FindPermanent(source.ID()).Tapped == acted {
				t.Fatal("untap used the wrong source lifetime")
			}
			if !g.FindPermanent(source.ID()).Tapped {
				t.Fatal("clone untapped original source")
			}
		})
	}
}

func TestComposableSourceUntapStunAndEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1)
	target := NewCreature("Unrelated target", "", 1, 1)
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtBecameUntapped, false, GainLifeFor(You, Fixed(1)))))
	for _, card := range []Card{source, target, watcher} {
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.TapPermanent(g.FindPermanent(source.ID()))
	g.TapPermanent(g.FindPermanent(target.ID()))
	g.MutablePermanent(source.ID()).AddCounter(Stun, 1)
	effect := UntapPermanent(This)
	for i := range 3 {
		if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		resolvePendingTriggers(g)
		wantLife := 21
		if i == 0 {
			wantLife = 20
		}
		if a.Life() != wantLife || g.FindPermanent(source.ID()).Tapped != (i == 0) || g.FindPermanent(source.ID()).Counters[Stun] != 0 {
			t.Fatalf("attempt %d: life %d, source %+v", i, a.Life(), g.FindPermanent(source.ID()))
		}
		if !g.FindPermanent(target.ID()).Tapped {
			t.Fatal("untapped positional target")
		}
	}
}

func TestComposableSourceUntapValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "spell source untap this",
			Card:          NewSorcery("Invalid untap", "", WithAbility(Spell(UntapPermanent(This)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].subject",
		},
		{
			Name:          "spell source untap nil",
			Card:          NewSorcery("Invalid untap", "", WithAbility(Spell(UntapPermanent(nil)))),
			WantErr:       true,
			WantErrSubstr: "abilities[0].effects[0].subject",
		},
	})
	effect := UntapPermanent(This)
	if !strings.Contains(effect.Text(), "this") || effect.Properties().Outcome != OutcomeBenefit {
		t.Fatal("missing untap diagnostics or benefit metadata")
	}
}

// --- from composable_your_turn_test.go ---
func TestIsYourTurnResolutionUsesCurrentTurnAndStoredController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewInstant("Turn check", "")
	effect := IfState(IsYourTurn(), GainLifeFor(You, Fixed(2)))
	g.SetActivePlayerIndex(1)
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("effect ran during opponent's turn")
	}
	clone := g.Clone()
	clone.SetActivePlayerIndex(0)
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("clone did not use its own active player")
	}
	if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 22 {
		t.Fatal("effect did not use the stored controller")
	}
}

func TestIsYourTurnDefinitionAndAbsence(t *testing.T) {
	condition := IsYourTurn()
	card := NewInstant("Turn check", "", Spell(IfState(condition, GainLifeFor(You, Fixed(1)))))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: card,
			Inspect: func(t *testing.T) {
				meta := InspectDefinition(condition)
				if meta.Role != "state-condition" || meta.Text != "it is your turn" {
					t.Fatalf("condition metadata = %+v", meta)
				}
			},
		},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if !condition.checkState(g, StateContext{SourceID: uuid.New(), ControllerID: a.PlayerID()}) {
		t.Fatal("absent spell source changed turn check")
	}
	if condition.checkState(g, StateContext{ControllerID: uuid.Nil}) {
		t.Fatal("missing controller matched active player")
	}
}

// --- from composable_zone_lki_cleanup_test.go ---
func TestZoneLKICleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Departed source", "{3}", 2, 2)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(source.ID()))
	register := func() {
		g.RegisterDelayedTrigger(&DelayedTrigger{
			EventType: EvtCleanup, TargetID: source.ID(), Controller: a.PlayerID(),
			Effects: []Effect{GainLifeTarget(TargetPermanentManaValue())},
		})
	}
	register()
	for round := range 2 {
		g.zones.AddExileInstead(source.ID(), source.ID())
		if !g.doCleanupActions() {
			t.Fatal("cleanup did not request priority")
		}
		if g.LKI(source.ID()) == nil {
			t.Fatal("cleanup priority lost departure information")
		}
		if g.zones.IsExileInstead(source.ID()) {
			t.Fatal("cleanup priority retained an expired exile effect")
		}
		if round == 0 {
			clone := g.Clone()
			clone.ResolveStack()
			clone.RunStepWithPriority(Cleanup)
			if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 || g.LKI(source.ID()) == nil {
				t.Fatal("cloned cleanup trigger lost LKI or changed the original game")
			}
		}
		g.ResolveStack()
		if a.Life() != 20+3*(round+1) {
			t.Fatalf("cleanup trigger life = %d", a.Life())
		}
		if round == 0 {
			register()
		}
	}
	clone := g.Clone()
	clone.RunStepWithPriority(Cleanup)
	if clone.LKI(source.ID()) != nil || g.LKI(source.ID()) == nil {
		t.Fatal("clone cleanup did not clear its cache independently")
	}
	g.RunStepWithPriority(Cleanup)
	if g.LKI(source.ID()) != nil {
		t.Fatal("final cleanup retained the legacy cache")
	}
}

func TestZoneLKICleanupStateBasedDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Temporary survivor", "{4}", 1, 0)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(0), Fixed(1))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	added := false
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		if game.LKI(source.ID()) == nil {
			t.Fatal("cleanup death lost LKI before priority ended")
		}
		if !added {
			added = true
			game.RegisterDelayedTrigger(&DelayedTrigger{
				EventType: EvtCleanup, TargetID: source.ID(), Controller: a.PlayerID(),
				Effects: []Effect{GainLifeTarget(TargetPermanentManaValue())},
			})
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Cleanup)
	if !added || a.Life() != 24 || g.GetCleanupPriorityRounds() != 2 {
		t.Fatalf("priority = %v, life = %d, rounds = %d", added, a.Life(), g.GetCleanupPriorityRounds())
	}
	if g.LKI(source.ID()) != nil {
		t.Fatal("final cleanup retained the cleanup death cache")
	}
}
