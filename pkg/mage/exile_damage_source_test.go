package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileDamageSourceVisits(t *testing.T) {
	for _, removal := range []string{"single", "source", "reexile"} {
		t.Run(removal, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewCreature("Exiled source", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			exiler := uuid.New()
			g.ExileCard(card, exiler)
			visit := g.FindExiledCard(card.ID()).Visit
			saved := g.CaptureExiledDamageSource(visit)
			if !saved.hasExactIdentity() || saved.ExileVisit() != visit || saved.ID() != card.ID() || saved.Controller() != uuid.Nil || saved.Owner() != a.PlayerID() || !saved.HasColor(core.Red) || !saved.HasType(core.TypeCreature) {
				t.Fatal("exile source lost identity or public information")
			}
			if g.CaptureExiledDamageSource(card.ID()).Valid() || g.CaptureExiledDamageSource(uuid.Nil).Valid() || saved.SameObject(DamageSource{id: card.ID(), stackID: visit}) {
				t.Fatal("physical ID or another identity category matched the visit")
			}
			clone := g.Clone()
			g.FindExiledCard(card.ID()).Card.AddType(core.TypeArtifact)
			if clone.CaptureExiledDamageSource(visit).HasType(core.TypeArtifact) || saved.HasType(core.TypeArtifact) {
				t.Fatal("exile card mutation crossed a clone or query")
			}
			switch removal {
			case "single":
				g.RemoveFromExile(card.ID())
				g.ExileCard(card, exiler)
			case "source":
				g.RemoveExiledCardBySource(exiler)
				g.ExileCard(card, exiler)
			case "reexile":
				g.ExileCard(card, exiler)
			}
			current := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
			old := g.damageReferenceInformation(saved)
			if current.SameObject(saved) || !old.SameObject(saved) || !old.HasType(core.TypeArtifact) || len(g.GetExile()) != 1 {
				t.Fatal("new visit replaced departure information or duplicated the object")
			}
			old.card.AddType(core.TypeLand)
			if g.damageReferenceInformation(saved).HasType(core.TypeLand) {
				t.Fatal("query mutated departure information")
			}
		})
	}
}

func TestExileDamageSourceStackBindings(t *testing.T) {
	for _, plural := range []bool{false, true} {
		t.Run(map[bool]string{false: "card", true: "cards"}[plural], func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			one, many := BindCard("paid"), BindCards("paid set")
			cost := ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", one)
			if plural {
				cost = ExileFromGraveyardCostWithCards(2, many)
			}
			keeper := NewArtifact("Reference keeper", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(keeper, a.PlayerID())
			count := 1
			if plural {
				count = 2
			}
			for range count {
				card := NewCreature("Payment", "{R}", 1, 1)
				card.SetOwner(a.PlayerID())
				a.AddToGraveyard(card)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			object := g.StackPeek()
			copied := g.CopyStackObjectDirect(object, a.PlayerID(), false)
			if copied == nil {
				t.Fatal("copy failed")
			}
			sources := g.StackBoundExileSources()
			if len(sources) != count {
				t.Fatalf("sources = %d, want %d", len(sources), count)
			}
			for _, source := range sources {
				g.RemoveFromExile(source.ID())
			}
			clone := g.Clone()
			if got := g.StackBoundExileSources(); len(got) != count || !got[0].SameObject(sources[0]) || !got[0].HasColor(core.Red) {
				t.Fatal("departure lost stack reference")
			}
			g.GetStack().RemoveByID(object.ID)
			if len(g.StackBoundExileSources()) != count {
				t.Fatal("copy lost references")
			}
			g.GetStack().RemoveByID(copied.ID)
			if len(g.StackBoundExileSources()) != 0 || len(clone.StackBoundExileSources()) != count {
				t.Fatal("reference lifetime crossed a clone")
			}
		})
	}
}

func TestExileDamageSourceHiddenAndDeclarations(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Secret", "{R}", 1, 1, WithSubTypes("Wizard"))
	card.SetOwner(a.PlayerID())
	g.ExileCardFaceDown(card, uuid.Nil, a.PlayerID())
	visit := g.FindExiledCard(card.ID()).Visit
	source := g.CaptureExiledDamageSource(visit)
	if !source.hasExactIdentity() || source.Name() != "" || source.HasColor(core.Red) || source.HasType(core.TypeCreature) || source.HasSubType("Wizard") {
		t.Fatal("private inspection exposed source characteristics")
	}
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, Effects: []Effect{Sequence(&eventCardCapture{binding: BindCard("event"), value: capturedCard{card.ID(), visit, core.ZoneExile}})}})
	if len(g.StackBoundExileSources()) != 1 {
		t.Fatal("hidden referred object lost identity")
	}
	chosen := BindDamageSource("choice")
	if err := ApplyEffect(g, ChooseDamageSourceFromStackExileBindings(You, DamageSourceHasColor(core.Red), chosen), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[chosen].Valid() {
		t.Fatal("hidden characteristics passed color filter")
	}
	g.RemoveFromExile(card.ID())
	if got := g.damageReferenceInformation(source); got.Name() != "" || got.HasColor(core.Red) {
		t.Fatal("departure revealed hidden data")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackExileBindings(nil, DamageSourceFilter{}, chosen),
		ChooseDamageSourceFromStackExileBindings(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackExileBindings(You, DamageSourceHasColor(core.Colorless), chosen),
		Sequence(DealDamageFrom(FromDamageSource(chosen), You, Fixed(1)), ChooseDamageSourceFromStackExileBindings(You, DamageSourceFilter{}, chosen)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestExileDamageSourceCastBoundary(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	card := NewCreature("Permitted source", "{R}", 1, 4, WithKeyword(core.Flash))
	card.SetOwner(a.PlayerID())
	g.ExileCard(card, uuid.Nil)
	visit := g.FindExiledCard(card.ID()).Visit
	saved := g.CaptureExiledDamageSource(visit)
	binding := BindDamageSource("chosen")
	g.PushStack(&StackObject{ID: uuid.New(), Controller: b.PlayerID(), IsAbility: true, capturedCards: map[*CardBinding]capturedCard{BindCard("reference"): {card.ID(), visit, core.ZoneExile}}})
	body := Sequence(ChooseDamageSourceFromStackExileBindings(You, DamageSourceHasColor(core.Red), binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)))
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.GrantCastFromExile(a.PlayerID(), card.ID(), false)
	if err := g.CastExiledCardWithPermission(a.PlayerID(), card.ID(), nil, 0); err == nil || g.FindExiledCard(card.ID()).Visit != visit || !g.CaptureExiledDamageSource(visit).SameObject(saved) || g.CastFromExilePermissionFor(a.PlayerID(), card.ID()) == nil {
		t.Fatal("failed payment changed the exile visit or permission")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastExiledCardWithPermission(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spell := g.CaptureSpellDamageSource(g.StackPeek().ID)
	if spell.SameObject(saved) || g.CaptureExiledDamageSource(visit).Valid() || !g.StackBoundExileSources()[0].SameObject(saved) {
		t.Fatal("casting changed the referred exile identity")
	}
	g.DealDamageToPlayerFromSource(b, 1, spell)
	g.ResolveTopOfStack()
	permanent := g.CaptureDamageSource(card.ID())
	g.DealDamageToPlayerFromSource(b, 1, permanent)
	if b.Life() != 18 {
		t.Fatal("exile choice followed a spell or permanent")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		player := game.GetPlayer(b.PlayerID())
		game.DealDamageToPlayerFromSource(player, 2, saved)
		if player.Life() != 18 {
			t.Fatal("saved exile source did not use its own shield")
		}
		game.DealDamageToPlayerFromSource(player, 1, saved)
		if player.Life() != 17 {
			t.Fatal("shield consumption crossed a clone")
		}
	}
}

func TestExileDamageSourcePermissionVisit(t *testing.T) {
	for _, removal := range []string{"single", "source", "reexile", "face-down"} {
		t.Run(removal, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewInstant("Permitted card", "", Spell(GainLifeFor(You, Fixed(1))))
			card.SetOwner(b.PlayerID())
			exiler := uuid.New()
			g.GrantCastFromExile(a.PlayerID(), card.ID(), true)
			if g.CastFromExilePermissionFor(a.PlayerID(), card.ID()) != nil {
				t.Fatal("absent exile object received permission")
			}
			g.ExileCard(card, exiler)
			g.GrantCastFromExile(a.PlayerID(), card.ID(), true)
			clone := g.Clone()
			switch removal {
			case "single":
				g.RemoveFromExile(card.ID())
				g.ExileCard(card, exiler)
			case "source":
				g.RemoveExiledCardBySource(exiler)
				g.ExileCard(card, exiler)
			case "reexile":
				g.ExileCard(card, exiler)
			case "face-down":
				g.ExileCardFaceDown(card, exiler, a.PlayerID())
			}
			if g.CastFromExilePermissionFor(a.PlayerID(), card.ID()) != nil || len(g.zones.CastFromExilePermissions()) != 0 || clone.CastFromExilePermissionFor(a.PlayerID(), card.ID()) == nil {
				t.Fatal("permission followed a returned visit or crossed a clone")
			}
			if err := g.CastExiledCardWithPermission(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("old permission cast a new visit")
			}
			g.GrantCastFromExile(a.PlayerID(), card.ID(), true)
			if err := g.CastExiledCardWithPermission(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 || g.CastFromExilePermissionFor(a.PlayerID(), card.ID()) != nil {
				t.Fatal("fresh permission lost its caster or lifetime")
			}
		})
	}
}

func TestExileDamageSourceRechecksAndBatch(t *testing.T) {
	for _, choice := range []int{0, 1} {
		t.Run(map[int]string{0: "source first", 1: "finite first"}[choice], func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
			g := NewGame(a, b)
			card := NewCreature("Source", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			g.ExileCard(card, uuid.Nil)
			saved := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
			g.PushStack(boundDamageObject(saved))
			binding := BindDamageSource("selected")
			body := Sequence(ChooseDamageSourceFromStackDamageBindings(You, DamageSourceHasColor(core.Red), binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)))
			if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			card.SetColorOverride([]core.Color{core.Blue})
			g.DealDamageToPlayerFromSource(b, 2, saved)
			if b.Life() != 18 {
				t.Fatal("shield used saved color for a live exile object")
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
			g.RemoveFromExile(card.ID())
			card.SetColorOverride([]core.Color{core.Blue})
			g.ExileCard(card, uuid.Nil)
			returned := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
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

func TestExileDamageSourceInvalidReferences(t *testing.T) {
	g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
	card := NewArtifact("Unreferred", "{R}")
	g.ExileCard(card, uuid.Nil)
	visit := g.FindExiledCard(card.ID()).Visit
	if len(g.StackBoundExileSources()) != 0 {
		t.Fatal("unreferred exile object was enumerated")
	}
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true,
		capturedCards: map[*CardBinding]capturedCard{
			nil:                      {card.ID(), visit, core.ZoneExile},
			BindCard("wrong ID"):     {uuid.New(), visit, core.ZoneExile},
			BindCard("wrong zone"):   {card.ID(), visit, core.ZoneHand},
			BindCard("no visit"):     {card.ID(), uuid.Nil, core.ZoneExile},
			BindCard("absent visit"): {card.ID(), uuid.New(), core.ZoneExile},
		},
	})
	if len(g.StackBoundExileSources()) != 0 || g.damageReferenceInformation(DamageSource{id: card.ID(), exileVisit: visit, stackID: uuid.New()}).Valid() {
		t.Fatal("invalid identity supplied a source")
	}
}
