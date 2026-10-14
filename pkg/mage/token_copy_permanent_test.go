package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTokenCopyPermanentBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	plan := entryCopyPlan(t, "Donor", func() []CardOption {
		return []CardOption{WithKeyword(core.EntersTapped), WithAbility(EntersWithNCounters(core.P1P1, 3)), WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1))))}
	})
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, b.PlayerID())
	donor.AddCounter(core.P1P1, 4)
	source := g.PutOnBattlefield(NewArtifact("Maker", ""), a.PlayerID())
	binding := BindPermanent("donor")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {donor.ID(), donor.incarnationID}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	link := NewCreatedTokenLink()
	var seen []int
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, r GameReader, _, _ uuid.UUID) bool {
		if e.ToZone == core.ZoneBattlefield {
			if p := r.FindPermanent(e.SourceID); p != nil && p.IsToken {
				seen = append(seen, len(TokensCreatedWith(link, This).resolvePermanents(ctx)))
			}
		}
		return false
	}))), a.PlayerID())
	effect := CreateTokenCopiesOfPermanentLinked(You, Fixed(2), From(binding), link)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	tokens := TokensCreatedWith(link, This).resolvePermanents(ctx)
	if len(tokens) != 2 || len(seen) != 2 || seen[0] != 2 || seen[1] != 2 {
		t.Fatalf("incomplete simultaneous linked entry: %v", seen)
	}
	for _, p := range tokens {
		if !p.IsToken || !p.Card.IsToken() || p.Card.Owner() != a.PlayerID() || p.ControllerID() != a.PlayerID() || p.Name() != "Donor" || !p.Tapped || p.Counters[core.P1P1] != 3 {
			t.Fatal("copied state instead of characteristics")
		}
		values, err := g.CaptureCopiableValues(p)
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := values.Build()
		if err != nil || fresh.IsToken() || fresh.Name() != "Donor" {
			t.Fatal("lost copy provenance", err)
		}
	}
	if tokens[0].Card.Abilities()[1].AbilityID() == tokens[1].Card.Abilities()[1].AbilityID() {
		t.Fatal("shared token declarations")
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	g.MutablePermanent(tokens[0].ID()).AddCounter(core.P1P1, 1)
	if clone.FindPermanent(tokens[0].ID()).Counters[core.P1P1] != 3 {
		t.Fatal("clone state shared")
	}
	g.BouncePermanentToHand(source)
	returned := g.PutOnBattlefield(source.Card, b.PlayerID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(TokensCreatedWith(link, This).resolvePermanents(ctx)) != 4 || len(TokensCreatedWith(link, This).resolvePermanents(&cloneCtx)) != 2 {
		t.Fatal("departed source or clone lost creation")
	}
	newCtx := &EffectContext{Game: g, SourceID: returned.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: returned.incarnationID}
	if len(TokensCreatedWith(link, This).resolvePermanents(newCtx)) != 0 {
		t.Fatal("returned source inherited link")
	}
}

func TestTokenCopyPermanentIdentity(t *testing.T) {
	for _, state := range []string{"live", "phased", "returned", "departed", "face-down", "zero", "negative", "untracked"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			plan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithKeyword(core.Flying)} })
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			if state == "untracked" {
				card = NewCreature("Untracked", "", 2, 2)
			}
			donor := g.PutOnBattlefield(card, b.PlayerID())
			binding := BindPermanent("donor")
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {donor.ID(), donor.incarnationID}}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			count := 1
			switch state {
			case "phased":
				g.PhaseOut(donor.ID())
			case "departed":
				g.RemoveFromBattlefield(donor)
			case "returned":
				g.RemoveFromBattlefield(donor)
				g.PutOnBattlefield(card, b.PlayerID())
			case "face-down":
				g.MutablePermanent(donor.ID()).FaceDown = true
			case "zero":
				count = 0
			case "negative":
				count = -1
			}
			err = CreateTokenCopiesOfPermanent(You, Fixed(count), From(binding)).Apply(ctx)
			if state == "untracked" {
				if err == nil {
					t.Fatal("untracked copy accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tokens := g.FilterBattlefield(IsToken)
			want := 0
			if state == "live" || state == "face-down" {
				want = 1
			}
			if len(tokens) != want {
				t.Fatalf("tokens: %d want %d", len(tokens), want)
			}
			if state == "face-down" {
				p := tokens[0]
				if p.FaceDown || p.Name() != "" || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || p.HasKeyword(core.Flying) || len(p.Card.Abilities()) != 0 {
					t.Fatal("copied face-down status or hidden rules")
				}
			}
		})
	}
}

func TestTokenCopyPermanentInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		CreateTokenCopiesOfPermanent(nil, Fixed(1), This), CreateTokenCopiesOfPermanent(You, nil, This), CreateTokenCopiesOfPermanent(You, Fixed(1), nil),
		CreateTokenCopiesOfPermanentLinked(You, Fixed(1), This, nil), CreateTokenCopiesOfPermanentLinked(You, Fixed(1), This, &CreatedTokenLink{}),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid copy declaration accepted")
		}
	}
}

func TestTokenCopyPermanentExceptionsAndEntry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	plan := entryCopyPlan(t, "Donor", func() []CardOption {
		return []CardOption{WithKeyword(core.Flying), WithAbility(EntersWithNCounters(core.P1P1, 2))}
	})
	values, err := plan.CopiableValues().WithModifications(CopyAddCardTypes(core.TypeArtifact))
	if err != nil {
		t.Fatal(err)
	}
	card, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, b.PlayerID())
	donor.AddCounter(core.P1P1, 4)
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(7), Fixed(7))), donor.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	binding := BindPermanent("donor")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {donor.ID(), donor.incarnationID}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	observed := 0
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool {
		entry, ok := action.(*battlefieldEntryAction)
		return ok && entry.permanent.IsToken
	}, replaceFunc: func(action Action, _ *Game) Action {
		p := action.(*battlefieldEntryAction).permanent
		if p.Name() != "Donor" || !p.HasType(core.TypeArtifact) || p.Card.Owner() != a.PlayerID() {
			t.Fatal("entry replacement saw uncopied values")
		}
		observed++
		return action
	}})
	if err := CreateTokenCopiesOfPermanent(You, Fixed(2), From(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	if len(tokens) != 2 || observed != 2 {
		t.Fatal("entry replacements missed copied tokens")
	}
	for _, p := range tokens {
		if !p.HasType(core.TypeArtifact) || p.Counters[core.P1P1] != 2 || p.CurrentPower(g) != 4 {
			t.Fatal("ordinary changes copied or copy exception lost")
		}
	}
	landPlan, err := NewCardDefinitionPlan(func() Card { return NewLand("Land donor") })
	if err != nil {
		t.Fatal(err)
	}
	land, err := landPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	landSource := g.PutOnBattlefield(land, b.PlayerID())
	g.resolution.permanentBindings[binding] = capturedPermanent{landSource.ID(), landSource.incarnationID}
	g.PutOnBattlefield(NewEnchantment("Block", "", WithAbility(Static(PreventLandEntry()))), a.PlayerID())
	if err := CreateTokenCopiesOfPermanent(You, Fixed(2), From(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.FilterBattlefield(IsToken)) != 2 {
		t.Fatal("prohibited tokens entered")
	}
}

func TestTokenCopyPermanentBuildFailure(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	builds := 0
	failAt := 0
	plan, err := NewCardDefinitionPlan(func() Card {
		builds++
		if failAt != 0 && builds == failAt {
			panic("builder drift")
		}
		return NewCreature("Donor", "", 2, 2)
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, b.PlayerID())
	binding := BindPermanent("donor")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {donor.ID(), donor.incarnationID}}
	failAt = builds + 2
	if err := CreateTokenCopiesOfPermanent(You, Fixed(3), From(binding)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err == nil {
		t.Fatal("builder failure ignored")
	}
	if len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("failed builder installed partial batch")
	}
}

func TestTokenCopyPermanentWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card, err := entryCopyPlan(t, "Donor", func() []CardOption { return nil }).Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	effect := AtNext(EachEndStep, CreateTokenCopiesOfPermanent(You, Fixed(1), This))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || len(g.FilterBattlefield(IsToken)) != 0 {
			t.Fatal("waiting query lost source or created tokens")
		}
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(card, a.PlayerID())
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.ResolveStack()
	}
	if len(g.FilterBattlefield(IsToken)) != 0 || len(clone.FilterBattlefield(IsToken)) != 1 {
		t.Fatal("delayed copy used returned subject or crossed clone")
	}
}
