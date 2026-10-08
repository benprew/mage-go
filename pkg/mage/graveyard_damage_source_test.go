package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardDamageSourceVisits(t *testing.T) {
	for _, removal := range []string{"single", "clear", "replace"} {
		t.Run(removal, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewCreature("Graveyard source", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			visit := g.graveyardVisit(card.ID())
			saved := g.CaptureGraveyardDamageSource(visit)
			if !saved.hasExactIdentity() || saved.GraveyardVisit() != visit || saved.ID() != card.ID() || saved.Controller() != uuid.Nil || saved.Owner() != a.PlayerID() || !saved.HasColor(core.Red) || !saved.HasType(core.TypeCreature) {
				t.Fatal("graveyard source lost identity or public facts")
			}
			if g.CaptureGraveyardDamageSource(card.ID()).Valid() || g.CaptureGraveyardDamageSource(uuid.Nil).Valid() || saved.SameObject(DamageSource{id: card.ID(), exileVisit: visit}) {
				t.Fatal("physical ID or another zone matched")
			}
			clone := g.Clone()
			card.AddType(core.TypeArtifact)
			if clone.CaptureGraveyardDamageSource(visit).HasType(core.TypeArtifact) || saved.HasType(core.TypeArtifact) {
				t.Fatal("query or clone shared mutable card facts")
			}
			switch removal {
			case "single":
				a.RemoveFromGraveyard(card.ID())
			case "clear":
				a.ClearGraveyard()
			case "replace":
				a.SetGraveyard(nil)
			}
			if g.CaptureGraveyardDamageSource(visit).Valid() {
				t.Fatal("departed visit remained live")
			}
			a.AddToGraveyard(card)
			current := g.CaptureGraveyardDamageSource(g.graveyardVisit(card.ID()))
			old := g.damageReferenceInformation(saved)
			if current.SameObject(saved) || !old.SameObject(saved) || !old.HasType(core.TypeArtifact) {
				t.Fatal("return replaced departure information")
			}
			old.card.AddType(core.TypeLand)
			if g.damageReferenceInformation(saved).HasType(core.TypeLand) {
				t.Fatal("query changed departure facts")
			}
			after := g.Clone()
			a.ClearGraveyard()
			if !after.CaptureGraveyardDamageSource(current.GraveyardVisit()).Valid() || !after.damageReferenceInformation(saved).HasType(core.TypeArtifact) {
				t.Fatal("departure changed clone state")
			}
		})
	}
}

func TestGraveyardDamageSourceStackReferences(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Referred card", "{R}")
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	visit := g.graveyardVisit(card.ID())
	saved := g.CaptureGraveyardDamageSource(visit)
	if len(g.StackGraveyardSources()) != 0 {
		t.Fatal("unreferred graveyard card supplied a source")
	}
	object := &StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{card.ID(), card.ID()},
		capturedCards:    map[*CardBinding]capturedCard{BindCard("card"): {card.ID(), visit, core.ZoneGraveyard}},
		capturedCardSets: map[*CardsBinding][]capturedCard{BindCards("cards"): {{card.ID(), visit, core.ZoneGraveyard}}},
		Effects:          []Effect{Sequence(Sequence(&eventCardCapture{binding: BindCard("event"), value: capturedCard{card.ID(), visit, core.ZoneGraveyard}}))},
	}
	g.PushStack(object)
	card.AddType(core.TypeCreature)
	a.RemoveFromGraveyard(card.ID())
	a.AddToGraveyard(card)
	copied := g.CopyStackObjectDirect(object, a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	for range 3 {
		sources := g.StackGraveyardSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) || !sources[0].HasType(core.TypeCreature) {
			t.Fatalf("wrong source or duplicate: %+v", sources)
		}
		sources[0].card.AddType(core.TypeLand)
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(object.ID)
	if len(g.StackGraveyardSources()) != 1 {
		t.Fatal("copy lost saved reference")
	}
	g.GetStack().RemoveByID(copied.ID)
	if len(g.StackGraveyardSources()) != 0 || len(clone.StackGraveyardSources()) != 1 {
		t.Fatal("reference lifetime or clone isolation failed")
	}
}

func TestGraveyardDamageSourcePayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCards("discarded cards")
	keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(DiscardCostWithCards(2, paid)), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	one, two := NewArtifact("One", "{R}"), NewArtifact("Two", "{U}")
	a.AddToHand(one)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err == nil || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || len(g.StackGraveyardSources()) != 0 {
		t.Fatal("failed payment changed live state")
	}
	a.AddToHand(two)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	sources := g.StackGraveyardSources()
	if len(sources) != 2 || len(a.Hand()) != 0 || len(a.Graveyard()) != 2 {
		t.Fatal("payment did not retain both new visits")
	}
	for _, source := range sources {
		if source.GraveyardVisit() != g.graveyardVisit(source.ID()) {
			t.Fatal("payment kept the old zone identity")
		}
	}
	g.ResolveTopOfStack()
	if len(g.StackGraveyardSources()) != 0 {
		t.Fatal("resolved payment kept category availability")
	}
}

func TestGraveyardDamageSourceInvalid(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewArtifact("Unreferred", "{R}")
	a.AddToGraveyard(card)
	visit := g.graveyardVisit(card.ID())
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{
		nil:                    {card.ID(), visit, core.ZoneGraveyard},
		BindCard("wrong ID"):   {uuid.New(), visit, core.ZoneGraveyard},
		BindCard("wrong zone"): {card.ID(), visit, core.ZoneHand},
		BindCard("no visit"):   {card.ID(), uuid.Nil, core.ZoneGraveyard},
		BindCard("absent"):     {card.ID(), uuid.New(), core.ZoneGraveyard},
	}})
	if len(g.StackGraveyardSources()) != 0 || g.damageReferenceInformation(DamageSource{id: card.ID(), graveyardVisit: visit, stackID: uuid.New()}).Valid() {
		t.Fatal("invalid reference supplied a source")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackGraveyardReferences(nil, DamageSourceFilter{}, BindDamageSource("choice")),
		ChooseDamageSourceFromStackGraveyardReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackGraveyardReferences(You, DamageSourceHasColor(core.Colorless), BindDamageSource("invalid filter")),
		Sequence(DealDamageFrom(FromDamageSource(BindDamageSource("unbound")), You, Fixed(1)), ChooseDamageSourceFromStackGraveyardReferences(You, DamageSourceFilter{}, BindDamageSource("later"))),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestGraveyardDamageSourceRechecksAndBatch(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(map[int]string{0: "source first", 1: "finite first"}[choice], func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
			g := NewGame(a, b)
			card := NewCreature("Source", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			saved := g.CaptureGraveyardDamageSource(g.graveyardVisit(card.ID()))
			g.PushStack(boundDamageObject(saved))
			binding := BindDamageSource("selected")
			body := Sequence(ChooseDamageSourceFromStackDamageBindings(You, DamageSourceHasColor(core.Red), binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)))
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			card.SetColorOverride([]core.Color{core.Blue})
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != 18 {
				t.Fatal("shield used saved color for a live graveyard object")
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
			a.RemoveFromGraveyard(card.ID())
			card.SetColorOverride([]core.Color{core.Blue})
			a.AddToGraveyard(card)
			returned := g.CaptureGraveyardDamageSource(g.graveyardVisit(card.ID()))
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

func TestGraveyardDamageSourceTargetsAndRetarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewArtifact("First", "{R}"), NewArtifact("Second", "{U}")
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	original := g.CaptureGraveyardDamageSource(g.graveyardVisit(first.ID()))
	referer := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Targets: []uuid.UUID{first.ID()}, TargetSpecs: []Target{TargetCardInAnyGraveyard()}}
	g.PushStack(referer)
	first.SetColorOverride([]core.Color{core.Green})
	a.RemoveFromGraveyard(first.ID())
	first.SetColorOverride([]core.Color{core.Black})
	a.AddToGraveyard(first)
	unchanged := g.CopyStackObjectDirect(referer, a.PlayerID(), false)
	if unchanged == nil {
		t.Fatal("copy failed")
	}
	sources := g.StackGraveyardSources()
	if len(sources) != 1 || !sources[0].SameObject(original) || !sources[0].HasColor(core.Green) {
		t.Fatal("copy followed the returned target")
	}
	chooser := newRecordingPlayer("Retarget controller")
	chooser.chooseQueue = [][]uuid.UUID{{second.ID()}}
	g.players[1] = chooser
	changed := g.CopyStackObjectDirect(referer, chooser.PlayerID(), true)
	if changed == nil || len(changed.Targets) != 1 || changed.Targets[0] != second.ID() {
		t.Fatal("retarget failed")
	}
	sources = g.StackGraveyardSources()
	if len(sources) != 2 || !sources[0].SameObject(original) || sources[1].ID() != second.ID() {
		t.Fatal("retarget lost source order or identity")
	}
	g.GetStack().RemoveByID(referer.ID)
	g.GetStack().RemoveByID(unchanged.ID)
	sources = g.StackGraveyardSources()
	if len(sources) != 1 || sources[0].ID() != second.ID() {
		t.Fatal("unused saved target supplied a source")
	}
}

func TestGraveyardDamageSourceSingularPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("discarded card")
	keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(DiscardCardCostInto(paid)), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	card := NewArtifact("Payment", "{R}")
	a.AddToHand(card)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	sources := g.StackGraveyardSources()
	if len(sources) != 1 || sources[0].ID() != card.ID() || sources[0].GraveyardVisit() != g.graveyardVisit(card.ID()) {
		t.Fatal("singular payment lost its new visit")
	}
}

func TestGraveyardDamageSourceCloneTargetStorage(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	a.AddToGraveyard(first)
	a.AddToGraveyard(second)
	object := &StackObject{ID: uuid.New(), IsAbility: true, Targets: []uuid.UUID{first.ID()}, ModalTargets: [][]uuid.UUID{{first.ID()}, {}, {second.ID()}}}
	g.PushStack(object)
	clone := g.Clone()
	cloneObject := clone.StackPeek()
	cloneObject.ModalTargets[0][0] = second.ID()
	cloneObject.ModalTargets[0] = append(cloneObject.ModalTargets[0], first.ID())
	cloneObject.ModalTargets[2][0] = first.ID()
	if object.ModalTargets[0][0] != first.ID() || object.ModalTargets[2][0] != second.ID() || cloneObject.Targets[0] != first.ID() || len(cloneObject.ModalTargets[1]) != 0 {
		t.Fatal("clone target storage changed another context")
	}
}

func TestGraveyardDamageSourcePendingEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dead := BindCard("dead card")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureDies().WithCard(dead), ReturnCardToHand(FromCard(dead)))))
	subject := NewCreature("Subject", "{R}", 1, 1)
	subject.SetOwner(b.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(subject, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(subject.ID()))
	saved := g.CaptureGraveyardDamageSource(g.graveyardVisit(subject.ID()))
	if !saved.Valid() || len(g.StackGraveyardSources()) != 0 {
		t.Fatal("pending event supplied a stack reference")
	}
	b.RemoveFromGraveyard(subject.ID())
	b.AddToGraveyard(subject)
	g.PutTriggersOnStack()
	sources := g.StackGraveyardSources()
	if len(sources) != 1 || !sources[0].SameObject(saved) {
		t.Fatal("pending event followed the returned card")
	}
	g.ResolveStack()
	if len(b.Graveyard()) != 1 || len(b.Hand()) != 0 || len(g.StackGraveyardSources()) != 0 {
		t.Fatal("event body or reference lifetime followed the new visit")
	}
}

func TestGraveyardDamageSourceReselectReturnedTarget(t *testing.T) {
	for _, reselect := range []bool{false, true} {
		t.Run(map[bool]string{false: "keep", true: "reselect"}[reselect], func(t *testing.T) {
			a := newRecordingPlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewArtifact("Target", "{R}")
			a.AddToGraveyard(card)
			old := g.CaptureGraveyardDamageSource(g.graveyardVisit(card.ID()))
			referer := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Targets: []uuid.UUID{card.ID()}, TargetSpecs: []Target{TargetCardInAnyGraveyard()}}
			g.PushStack(referer)
			a.RemoveFromGraveyard(card.ID())
			a.AddToGraveyard(card)
			current := g.CaptureGraveyardDamageSource(g.graveyardVisit(card.ID()))
			chosen := []uuid.UUID{}
			if reselect {
				chosen = append(chosen, card.ID())
			}
			a.chooseQueue = [][]uuid.UUID{chosen}
			copied := g.CopyStackObjectDirect(referer, a.PlayerID(), true)
			if copied == nil {
				t.Fatal("copy failed")
			}
			g.GetStack().RemoveByID(referer.ID)
			sources := g.StackGraveyardSources()
			want := old
			if reselect {
				want = current
			}
			if len(sources) != 1 || !sources[0].SameObject(want) {
				t.Fatal("copy confused a retained target with a newly chosen visit")
			}
		})
	}
}
