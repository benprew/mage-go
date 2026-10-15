package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryHistoryLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewArtifact("Source", "")
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsEnteredWithSource(This, MatchingPermanents(AnyPermanent))
	check := func(game *Game, want int) {
		t.Helper()
		local := *ctx
		local.Game = game
		for range 2 {
			if got := len(set.resolvePermanents(&local)); got != want {
				t.Fatalf("entry history = %d, want %d", got, want)
			}
		}
	}
	guest := NewCreature("Guest", "", 3, 4)
	guest.SetOwner(b.PlayerID())
	b.AddToGraveyard(guest)
	target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
	ctx.Targets = []uuid.UUID{guest.ID()}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{guest.ID(), g.graveyardVisit(guest.ID())}}}
	check(g, 0)
	if err := ReturnCardToBattlefield(target, You, RecordEntryFrom(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	check(g, 1)
	clone := g.Clone()
	g.PhaseOut(guest.ID())
	check(g, 0)
	check(clone, 1)
	g.PhaseIn([]uuid.UUID{guest.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	returnedSource := g.PutOnBattlefield(sourceCard, b.PlayerID())
	g.RunStepWithPriority(core.Cleanup)
	check(g, 1)
	fresh := *ctx
	fresh.sourceIncarnation = returnedSource.incarnationID
	if len(set.resolvePermanents(&fresh)) != 0 {
		t.Fatal("returned source inherited entry history")
	}
	g.RemoveFromBattlefield(g.FindPermanent(guest.ID()))
	g.PutOnBattlefield(guest, b.PlayerID())
	check(g, 0)
	check(clone, 1)
	second := NewCreature("Second", "", 2, 2)
	second.SetOwner(b.PlayerID())
	clone.GetPlayer(b.PlayerID()).AddToGraveyard(second)
	local := *ctx
	local.Game = clone
	local.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{second.ID(), clone.graveyardVisit(second.ID())}}}
	if err := ReturnCardToBattlefield(target, You, RecordEntryFrom(This)).Apply(&local); err != nil {
		t.Fatal(err)
	}
	check(clone, 2)
	check(g, 0)
}

func TestEntryHistoryDeclarations(t *testing.T) {
	target := DeclareGraveyardCardTarget("guest")
	for _, body := range []ResolvingEffect{
		TapEach(PermanentsEnteredWithSource(nil, MatchingPermanents(AnyPermanent))),
		TapEach(PermanentsEnteredWithSource(This, nil)),
		TapEach(PermanentsEnteredWithSource(From(BindPermanent("missing")), MatchingPermanents(AnyPermanent))),
		ReturnCardToBattlefield(target, You, RecordEntryFrom(nil)),
		ReturnCardToBattlefield(target, You, RecordEntryFrom(From(BindPermanent("missing")))),
		ReturnCardToBattlefield(target, You, RecordEntryFrom(This), RecordEntryFrom(This)),
	} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), body, WithTargets(target))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid entry history declaration accepted")
		}
	}
}

func TestEntryHistoryEntryCommit(t *testing.T) {
	for _, mode := range []string{"normal", "blocked", "stale", "cancel", "attached", "illegal-host", "copy", "source-return", "source-phase"} {
		t.Run(mode, func(t *testing.T) {
			a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewArtifact("Source", "")
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
			set := PermanentsEnteredWithSource(This, MatchingPermanents(AnyPermanent))
			donorCard, err := entryCopyPlan(t, "Donor", func() []CardOption { return nil }).Build()
			if err != nil {
				t.Fatal(err)
			}
			donor := g.PutOnBattlefield(donorCard, a.PlayerID())
			guest := NewCreature("Guest", "", 3, 4)
			options := []BattlefieldEntryOption{RecordEntryFrom(This), EnterTapped()}
			switch mode {
			case "blocked":
				guest = NewLand("Land")
				g.effects.Rules.SetLandsCantEnter(true)
			case "cancel":
				guest.AddAbility(&graveyardRejectEntry{})
			case "attached", "illegal-host":
				guest = NewAura("Aura", "", WithCastTarget(CreatureEnchantTarget()))
				var host PermanentRef = This
				if mode == "attached" {
					bound := BindPermanent("host")
					g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: {donor.ID(), donor.incarnationID}}
					host = From(bound)
				}
				options = append(options, EnterAttachedTo(host))
			case "copy":
				card, buildErr := entryCopyPlan(t, "Guest", func() []CardOption {
					return []CardOption{WithAbility(MayCopyAsEnters(IsCreature))}
				}).Build()
				if buildErr != nil {
					t.Fatal(buildErr)
				}
				guest = card.(*BaseCard)
			case "source-return", "source-phase":
				guest.AddAbility(&optionalEntryProbe{run: func(game *Game, _ *Permanent) bool {
					if mode == "source-return" {
						game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
						game.PutOnBattlefield(sourceCard, a.PlayerID())
					} else {
						game.PhaseOut(source.ID())
					}
					return true
				}})
			}
			guest.SetOwner(b.PlayerID())
			b.AddToGraveyard(guest)
			target := DeclareGraveyardCardTarget("guest", InAnyGraveyard())
			ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{guest.ID(), g.graveyardVisit(guest.ID())}}}
			if mode == "stale" {
				card, _ := g.MoveFromGraveyard(b.PlayerID(), guest.ID(), core.ZoneHand)
				b.AddToGraveyard(card)
			}
			observed := 0
			watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.SourceID == guest.ID() && e.ToZone == core.ZoneBattlefield {
					observed++
					if len(set.resolvePermanents(ctx)) != 1 {
						t.Error("entry observation preceded history")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
			if mode == "blocked" {
				g.effects.Rules.SetLandsCantEnter(true)
			}
			clone := g.Clone()
			if err := ReturnCardToBattlefield(target, You, options...).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			failed := mode == "blocked" || mode == "stale" || mode == "cancel" || mode == "illegal-host"
			got := set.resolvePermanents(ctx)
			if failed {
				if len(got) != 0 || len(g.zones.entryHistory) != 0 || observed != 0 {
					t.Fatal("failed entry recorded history")
				}
			} else if len(got) != 1 || got[0].ID() != guest.ID() || !got[0].Tapped || observed != 1 {
				t.Fatalf("successful entry lost its exact source or entry options: selected=%v observed=%d live=%v", got, observed, g.FindPermanent(guest.ID()))
			}
			if mode == "copy" && g.FindPermanent(guest.ID()).Name() != "Donor" {
				t.Fatal("copy entry did not run")
			}
			if len(clone.zones.entryHistory) != 0 {
				t.Fatal("entry changed a pending clone")
			}
		})
	}
}

func TestEntryHistoryExileBatchAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	set := PermanentsEnteredWithSource(This, MatchingPermanents(AnyPermanent))
	guests := make(savedTransportVisits, 0, 2)
	for range 2 {
		card := NewCreature("Guest", "", 2, 2)
		card.SetOwner(b.PlayerID())
		g.ExileCard(card, source.ID())
		guests = append(guests, capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile})
	}
	observations := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneExile && e.ToZone == core.ZoneBattlefield {
			observations++
			if len(set.resolvePermanents(ctx)) != 2 {
				t.Error("entry observation preceded simultaneous history")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(guests, RecordEntryFrom(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if observations != 2 {
		t.Fatal("batch did not enter")
	}
	if err := AtNext(EachEndStep, TapEach(set)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.RunStepWithPriority(core.Cleanup)
	clone := g.Clone()
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || refs[0].Incarnation() != source.incarnationID || len(g.zones.entryHistory) != 2 {
			t.Fatal("waiting query lost source or changed history")
		}
	}
	g.PhaseOut(guests[0].id)
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if !clone.FindPermanent(guests[0].id).Tapped || g.FindPermanentIncludingPhased(guests[0].id).Tapped || !g.FindPermanent(guests[1].id).Tapped {
		t.Fatal("delayed history lost exact visits or clone isolation")
	}
}
