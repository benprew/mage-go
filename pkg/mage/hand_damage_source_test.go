package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestHandDamageSourceVisits(t *testing.T) {
	for _, removal := range []string{"single", "replace"} {
		t.Run(removal, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			g.PutOnBattlefield(NewEnchantment("Public hand", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
			card := NewCreature("Hand source", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			visit := g.handVisit(card.ID())
			saved := g.CaptureHandDamageSource(visit)
			a.SetHand([]Card{card})
			if a.handVisit(card.ID()) != visit {
				t.Fatal("hand arrangement changed visit")
			}
			if !saved.hasExactIdentity() || saved.HandVisit() != visit || saved.ID() != card.ID() || saved.Controller() != uuid.Nil || saved.Owner() != a.PlayerID() || !saved.HasColor(core.Red) || !saved.HasType(core.TypeCreature) {
				t.Fatal("hand source lost identity or public facts")
			}
			if g.CaptureHandDamageSource(card.ID()).Valid() || g.CaptureHandDamageSource(uuid.Nil).Valid() || saved.SameObject(DamageSource{id: card.ID(), exileVisit: visit}) {
				t.Fatal("physical ID or another zone matched")
			}
			clone := g.Clone()
			card.AddType(core.TypeArtifact)
			if clone.CaptureHandDamageSource(visit).HasType(core.TypeArtifact) || saved.HasType(core.TypeArtifact) {
				t.Fatal("query or clone shared mutable card facts")
			}
			switch removal {
			case "single":
				a.RemoveFromHand(card.ID())
			case "replace":
				a.SetHand(nil)
			}
			if g.CaptureHandDamageSource(visit).Valid() {
				t.Fatal("departed visit remained live")
			}
			a.AddToHand(card)
			current := g.CaptureHandDamageSource(g.handVisit(card.ID()))
			old := g.damageReferenceInformation(saved)
			if old.card.ID() != card.ID() || current.SameObject(saved) || !old.SameObject(saved) || !old.HasType(core.TypeArtifact) {
				t.Fatal("return replaced departure information")
			}
			old.card.AddType(core.TypeLand)
			if g.damageReferenceInformation(saved).HasType(core.TypeLand) {
				t.Fatal("query changed departure facts")
			}
			after := g.Clone()
			a.SetHand(nil)
			if !after.CaptureHandDamageSource(current.HandVisit()).Valid() || !after.damageReferenceInformation(saved).HasType(core.TypeArtifact) {
				t.Fatal("departure changed clone state")
			}
		})
	}
}

func TestHandDamageSourceStackReferences(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(NewEnchantment("Public hand", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
	card := NewArtifact("Referred card", "{R}")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	visit := g.handVisit(card.ID())
	saved := g.CaptureHandDamageSource(visit)
	if len(g.StackHandSources()) != 0 {
		t.Fatal("unreferred hand card supplied a source")
	}
	object := &StackObject{ID: uuid.New(), IsAbility: true,
		capturedCards:    map[*CardBinding]capturedCard{BindCard("card"): {card.ID(), visit, core.ZoneHand}},
		capturedCardSets: map[*CardsBinding][]capturedCard{BindCards("cards"): {{card.ID(), visit, core.ZoneHand}}},
		Effects:          []Effect{Sequence(Sequence(&eventCardCapture{binding: BindCard("event"), value: capturedCard{card.ID(), visit, core.ZoneHand}}))},
	}
	g.PushStack(object)
	card.AddType(core.TypeCreature)
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	copied := g.CopyStackObjectDirect(object, a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	for range 3 {
		sources := g.StackHandSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) || !sources[0].HasType(core.TypeCreature) {
			t.Fatalf("wrong source or duplicate: %+v", sources)
		}
		sources[0].card.AddType(core.TypeLand)
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(object.ID)
	if len(g.StackHandSources()) != 1 {
		t.Fatal("copy lost saved reference")
	}
	g.GetStack().RemoveByID(copied.ID)
	if len(g.StackHandSources()) != 0 || len(clone.StackHandSources()) != 1 {
		t.Fatal("reference lifetime or clone isolation failed")
	}
}

func TestHandDamageSourceInvalid(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(NewEnchantment("Public hand", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
	card := NewArtifact("Unreferred", "{R}")
	a.AddToHand(card)
	visit := g.handVisit(card.ID())
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{
		nil:                    {card.ID(), visit, core.ZoneHand},
		BindCard("wrong ID"):   {uuid.New(), visit, core.ZoneHand},
		BindCard("wrong zone"): {card.ID(), visit, core.ZoneGraveyard},
		BindCard("no visit"):   {card.ID(), uuid.Nil, core.ZoneHand},
		BindCard("absent"):     {card.ID(), uuid.New(), core.ZoneHand},
	}})
	if len(g.StackHandSources()) != 0 || g.damageReferenceInformation(DamageSource{id: card.ID(), handVisit: visit, stackID: uuid.New()}).Valid() {
		t.Fatal("invalid reference supplied a source")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackHandReferences(nil, DamageSourceFilter{}, BindDamageSource("choice")),
		ChooseDamageSourceFromStackHandReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackHandReferences(You, DamageSourceHasColor(core.Colorless), BindDamageSource("invalid filter")),
		Sequence(DealDamageFrom(FromDamageSource(BindDamageSource("unbound")), You, Fixed(1)), ChooseDamageSourceFromStackHandReferences(You, DamageSourceFilter{}, BindDamageSource("later"))),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestHandDamageSourceRechecksAndBatch(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(map[int]string{0: "source first", 1: "finite first"}[choice], func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
			g := NewGame(a, b)
			g.PutOnBattlefield(NewEnchantment("Public hand", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
			card := NewCreature("Source", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			saved := g.CaptureHandDamageSource(g.handVisit(card.ID()))
			g.PushStack(boundDamageObject(saved))
			binding := BindDamageSource("selected")
			body := Sequence(ChooseDamageSourceFromStackDamageBindings(You, DamageSourceHasColor(core.Red), binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)))
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			card.SetColorOverride([]core.Color{core.Blue})
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != 18 {
				t.Fatal("shield used saved color for a live hand object")
			}
			card.SetColorOverride([]core.Color{core.Red})
			g.DealDamageToPlayerFromSource(b, 0, saved)
			g.AddPreventionShield(b.PlayerID(), 1)
			g.DealDamageToPlayerFromSource(b, 1, saved)
			if b.Life() != 18 || b.prompts != 1 {
				t.Fatal("competing replacement choice was bypassed")
			}
			g.DealDamageToPlayerFromSource(b, 2, saved)
			want := 17
			if choice == 1 {
				want = 18
			}
			if b.Life() != want {
				t.Fatal("wrong shield was consumed")
			}
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			a.RemoveFromHand(card.ID())
			card.SetColorOverride([]core.Color{core.Blue})
			a.AddToHand(card)
			returned := g.CaptureHandDamageSource(g.handVisit(card.ID()))
			outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(b, 2, saved), PlayerDamageAssignment(b, 3, saved), PlayerDamageAssignment(b, 1, returned))
			if b.Life() != want-1 || len(outcomes) != 1 || outcomes[0].amount != 1 || !outcomes[0].source.SameObject(returned) {
				t.Fatalf("simultaneous life=%d want=%d, outcomes=%+v, retained red=%v", b.Life(), want-1, outcomes, g.damageReferenceInformation(saved).HasColor(core.Red))
			}
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.effects.ClearReplacementsEndOfTurn()
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != want-3 {
				t.Fatal("cleanup retained the shield")
			}
		})
	}
}

func TestHandDamageSourceHiddenVisibility(t *testing.T) {
	for _, publicDeparture := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Private", "{R}", 1, 1)
		a.AddToHand(card)
		visit := a.handVisit(card.ID())
		hidden := g.CaptureHandDamageSource(visit)
		checkHidden := func(source DamageSource) {
			t.Helper()
			if !source.SameObject(hidden) || source.Name() != "" || source.HasType(core.TypeCreature) || source.HasColor(core.Red) || source.card != nil || source.Controller() != uuid.Nil {
				t.Fatal("private characteristics escaped")
			}
		}
		checkHidden(hidden)
		binding := BindCard("revealed payment")
		keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(RevealFromHandCostWithCard(CardFilter{}, "reveal", binding)), GainLifeFor(You, Fixed(1)))))
		g.PutOnBattlefield(keeper, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if len(g.PublicReveals()) != 1 || len(g.StackHandSources()) != 1 {
			t.Fatal("real reveal payment did not save reference")
		}
		if !g.StackHandSources()[0].HasColor(core.Red) {
			t.Fatal("reveal payment ended before stack departure")
		}
		g.GetStack().Pop()
		g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{binding: {card.ID(), visit, core.ZoneHand}}})
		checkHidden(g.StackHandSources()[0])
		reveal := NewEnchantment("Public hand", "", WithAbility(Static(RevealAllHands())))
		p := g.PutOnBattlefield(reveal, a.PlayerID())
		if !g.StackHandSources()[0].HasColor(core.Red) {
			t.Fatal("public hand lost characteristics")
		}
		clone := g.Clone()
		g.PhaseOut(p.ID())
		g.ApplyContinuousEffects()
		checkHidden(g.StackHandSources()[0])
		if !clone.StackHandSources()[0].HasColor(core.Red) {
			t.Fatal("visibility crossed clone")
		}
		if publicDeparture {
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.ApplyContinuousEffects()
		}
		a.SetHand(nil)
		departed := g.StackHandSources()[0]
		if publicDeparture {
			if !departed.HasColor(core.Red) {
				t.Fatal("public departure facts lost")
			}
		} else {
			checkHidden(departed)
		}
		a.AddToHand(card)
		if g.CaptureHandDamageSource(a.handVisit(card.ID())).SameObject(hidden) {
			t.Fatal("returned hand inherited visit")
		}
		g.GetStack().Pop()
		if len(g.StackHandSources()) != 0 {
			t.Fatal("departure history made a choice available")
		}
	}
}

func TestHandDamageSourcePaymentRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	binding := BindCard("paid")
	keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(RevealFromHandCostWithCard(CardFilter{}, "reveal", binding), ManaCostOf("{1}")), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	card := NewArtifact("Payment", "{R}")
	a.AddToHand(card)
	visit := a.handVisit(card.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err == nil || len(g.PublicReveals()) != 0 || len(g.StackHandSources()) != 0 || a.handVisit(card.ID()) != visit {
		t.Fatal("failed reveal payment changed state")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(g.StackHandSources()) != 1 || g.StackHandSources()[0].HandVisit() != visit {
		t.Fatal("paid reveal lost visit")
	}
	g.ResolveStack()
	if len(g.StackHandSources()) != 0 {
		t.Fatal("resolution retained stack category")
	}
}

func TestHandDamageSourcePendingDraw(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawn := BindCard("drawn")
	who := BindPlayer("drawer")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(PlayerDrawsCard(who).WithCard(drawn), RevealCard(FromCard(drawn)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewCreature("Drawn", "{R}", 1, 1)
	b.SetLibrary([]Card{card})
	g.PlayerDrawCard(b)
	saved := g.CaptureHandDamageSource(b.handVisit(card.ID()))
	if !saved.Valid() || len(g.StackHandSources()) != 0 {
		t.Fatal("pending event supplied stack source")
	}
	b.RemoveFromHand(card.ID())
	b.AddToHand(card)
	g.PutTriggersOnStack()
	if sources := g.StackHandSources(); len(sources) != 1 || !sources[0].SameObject(saved) {
		t.Fatal("draw capture followed returned card")
	}
	g.ResolveStack()
	if len(g.PublicReveals()) != 0 || len(g.StackHandSources()) != 0 {
		t.Fatal("event resolution followed new visit")
	}
}

func TestHandDamageSourceCloneTargetStorage(t *testing.T) {
	one, two := uuid.New(), uuid.New()
	original := &StackObject{Targets: []uuid.UUID{one, two}, ModalTargets: [][]uuid.UUID{{one}, {two}}}
	clone := cloneStackObject(original)
	clone.Targets[0] = uuid.New()
	if clone.ModalTargets[0][0] != one || original.Targets[0] != one {
		t.Fatal("flat targets shared storage")
	}
	clone.Targets = append(clone.Targets, uuid.New())
	clone.ModalTargets[0] = append(clone.ModalTargets[0], uuid.New())
	if clone.ModalTargets[1][0] != two || original.ModalTargets[0][0] != one {
		t.Fatal("target append crossed groups")
	}
}

func TestHandDamageSourceCastBoundary(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Hand subject", "{R}", 1, 1, WithKeyword(core.Flash))
	a.AddToHand(card)
	saved := g.CaptureHandDamageSource(a.handVisit(card.ID()))
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{BindCard("reference"): {card.ID(), saved.HandVisit(), core.ZoneHand}}})
	binding := BindDamageSource("chosen")
	body := Sequence(ChooseDamageSourceFromStackHandReferences(You, DamageSourceFilter{}, binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceFilter{}))
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spell := g.CaptureSpellDamageSource(g.StackPeek().ID)
	if spell.SameObject(saved) {
		t.Fatal("hand identity became spell identity")
	}
	g.DealDamageToPlayerFromSource(b, 2, spell)
	g.ResolveTopOfStack()
	permanent := g.CaptureDamageSource(card.ID())
	g.DealDamageToPlayerFromSource(b, 2, permanent)
	if b.Life() != 16 {
		t.Fatal("hand shield followed spell or permanent")
	}
	g.DealDamageToPlayerFromSource(b, 2, saved)
	if b.Life() != 16 {
		t.Fatal("hidden source choice lost exact shield")
	}
	g.DealDamageToPlayerFromSource(b, 1, saved)
	if b.Life() != 15 {
		t.Fatal("hidden source shield did not consume")
	}
}

func TestHandDamageSourceRevealStackLifetime(t *testing.T) {
	for _, bound := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		card := NewArtifact("Revealed", "{R}")
		a.AddToHand(card)
		visit := a.handVisit(card.ID())
		cost := RevealFromHandCost(CardFilter{}, "reveal")
		if bound {
			cost = RevealFromHandCostWithCard(CardFilter{}, "reveal", BindCard("paid"))
		}
		keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		g.PutOnBattlefield(keeper, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		original := g.StackPeek()
		copied := g.CopyStackObjectDirect(original, a.PlayerID(), false)
		if copied == nil || !g.CaptureHandDamageSource(visit).HasColor(core.Red) {
			t.Fatal("live reveal lost public characteristics")
		}
		g.GetStack().RemoveByID(original.ID)
		if g.CaptureHandDamageSource(visit).HasColor(core.Red) {
			t.Fatal("copy extended original cost reveal")
		}
		if len(g.StackHandSources()) != 1 {
			t.Fatal("copy lost cost reference")
		}
		g.GetStack().RemoveByID(copied.ID)
		if len(g.StackHandSources()) != 0 {
			t.Fatal("final departure retained reveal source")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.StackPeek().Effects = []Effect{FuncEffect("probe", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
			if !game.CaptureHandDamageSource(visit).HasColor(core.Red) || len(game.StackHandSources()) != 1 {
				t.Fatal("reveal ended during resolution")
			}
			clone := game.Clone()
			if !clone.CaptureHandDamageSource(visit).HasColor(core.Red) {
				t.Fatal("resolution clone lost reveal")
			}
			return nil
		})}
		g.ResolveTopOfStack()
		if g.CaptureHandDamageSource(visit).HasColor(core.Red) || len(g.StackHandSources()) != 0 {
			t.Fatal("resolved object retained reveal")
		}
	}
}

func TestHandDamageSourceRevealSpellPaths(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		subject := NewCreature("Revealed", "{R}", 1, 1)
		a.AddToHand(subject)
		visit := a.handVisit(subject.ID())
		spell := NewInstant("Reveal spell", "", WithAdditionalCost(RevealFromHandCost(IsCreatureCard, "reveal")), GainLifeFor(You, Fixed(1)))
		var err error
		if alternate {
			a.AddToGraveyard(spell)
			err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), core.ZoneGraveyard, core.ManaCost{}, nil, 0)
		} else {
			a.AddToHand(spell)
			err = g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0)
		}
		if err != nil {
			t.Fatal(err)
		}
		if sources := g.StackHandSources(); len(sources) != 1 || sources[0].HandVisit() != visit || !sources[0].HasColor(core.Red) {
			t.Fatal("spell reveal did not keep exact public source")
		}
		clone := g.Clone()
		a.RemoveFromHand(subject.ID())
		a.AddToHand(subject)
		current := g.CaptureHandDamageSource(a.handVisit(subject.ID()))
		if current.HasColor(core.Red) || current.HandVisit() == visit || !g.StackHandSources()[0].HasColor(core.Red) {
			t.Fatal("reveal followed returned visit or lost departure facts")
		}
		if !clone.CaptureHandDamageSource(visit).HasColor(core.Red) {
			t.Fatal("clone lost original reveal")
		}
		g.ResolveStack()
		if len(g.StackHandSources()) != 0 || g.CaptureHandDamageSource(a.handVisit(subject.ID())).HasColor(core.Red) {
			t.Fatal("resolution retained visibility")
		}
	}
}

func TestHandDamageSourceNestedResolution(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	red, blue := NewCreature("Red", "{R}", 1, 1), NewArtifact("Blue", "{U}")
	a.AddToHand(red)
	a.AddToHand(blue)
	redVisit, blueVisit := a.handVisit(red.ID()), a.handVisit(blue.ID())
	keeper := NewArtifact("Keeper", "",
		WithAbility(Activate(Costs(RevealFromHandCost(IsCreatureCard, "reveal creature")), GainLifeFor(You, Fixed(1)))),
		WithAbility(Activate(Costs(RevealFromHandCost(IsArtifactCard, "reveal artifact")), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	inner := g.StackPeek()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.StackPeek().Effects = []Effect{FuncEffect("outer probe", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		game.ResolveTopOfStack()
		if !game.CaptureHandDamageSource(redVisit).HasColor(core.Red) || game.CaptureHandDamageSource(blueVisit).HasColor(core.Blue) || len(game.StackHandSources()) != 1 {
			t.Fatal("nested completion changed outer lifetime")
		}
		return nil
	})}
	inner.Effects = []Effect{FuncEffect("inner probe", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		if !game.CaptureHandDamageSource(redVisit).HasColor(core.Red) || !game.CaptureHandDamageSource(blueVisit).HasColor(core.Blue) || len(game.StackHandSources()) != 2 {
			t.Fatal("nested resolution lost outer references")
		}
		clone := game.Clone()
		if len(clone.StackHandSources()) != 2 || !clone.CaptureHandDamageSource(redVisit).HasColor(core.Red) {
			t.Fatal("nested clone lost active contexts")
		}
		return nil
	})}
	g.ResolveTopOfStack()
	if len(g.StackHandSources()) != 0 || g.CaptureHandDamageSource(redVisit).HasColor(core.Red) {
		t.Fatal("outer completion retained references")
	}
}
