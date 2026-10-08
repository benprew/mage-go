package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolvingObjectSourcesFrames(t *testing.T) {
	for _, kind := range []string{"ability", "permanent target", "permanent binding", "spell target", "spell binding", "damage binding"} {
		t.Run(kind, func(t *testing.T) {
			g, a, _, spell := spellSourceFixture(t)
			card := NewArtifact("Permanent source", "{R}")
			card.SetOwner(a.PlayerID())
			permanent := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(permanent.ID())
			query := (*Game).StackAbilityPermanentSources
			object := &StackObject{ID: uuid.New(), IsAbility: true, Controller: a.PlayerID()}
			switch kind {
			case "ability":
				object.SourceID, object.sourceIncarnation = permanent.ID(), permanent.incarnationID
			case "permanent target":
				query = (*Game).StackTargetPermanentSources
				object.Targets = []uuid.UUID{permanent.ID(), permanent.ID()}
			case "permanent binding":
				query = (*Game).StackBoundPermanentSources
				object.Effects = []Effect{Sequence(Sequence(&eventPermanentCapture{binding: BindPermanent("event"), value: capturedPermanent{permanent.ID(), permanent.incarnationID}}))}
			case "spell target":
				saved = g.CaptureSpellDamageSource(spell.ID)
				query = (*Game).StackTargetSpellSources
				object.Targets = []uuid.UUID{spell.ID, spell.ID}
			case "spell binding":
				saved = g.CaptureSpellDamageSource(spell.ID)
				query = (*Game).StackBoundSpellSources
				object.Effects = boundSpellObject(saved).Effects
			case "damage binding":
				saved = g.CaptureSpellDamageSource(spell.ID)
				query = (*Game).StackBoundDamageSources
				object.Effects = boundDamageObject(saved).Effects
			}
			checks := 0
			check := func(game *Game, color core.Color) {
				t.Helper()
				sources := query(game)
				if len(sources) != 1 || !sources[0].SameObject(saved) || !sources[0].HasColor(color) {
					t.Fatalf("resolving %s lost exact source or facts: %v", kind, sources)
				}
				sources[0].card.AddType(core.TypeLand)
				if query(game)[0].HasType(core.TypeLand) {
					t.Fatal("query mutated retained facts")
				}
			}
			object.Effects = append(object.Effects, FuncEffect("Inspect resolving source", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				checks++
				check(game, core.Red)
				clone := game.Clone()
				check(clone, core.Red)
				want := core.Red
				if saved.StackID() != uuid.Nil {
					game.ChangeObjectColor(spell.ID, core.Blue)
					game.GetStack().RemoveByID(spell.ID)
					want = core.Blue
				} else {
					ids := game.PhaseOut(permanent.ID())
					check(game, core.Red)
					game.PhaseIn(ids)
					game.BouncePermanentToHand(game.FindPermanent(permanent.ID()))
					game.GetPlayer(a.PlayerID()).RemoveFromHand(card.ID())
					game.PutOnBattlefield(card, a.PlayerID())
					game.ChangeObjectColor(card.ID(), core.Blue)
				}
				check(game, want)
				check(clone, core.Red)
				inner := &StackObject{ID: uuid.New(), IsAbility: true, Controller: a.PlayerID(), Effects: []Effect{FuncEffect("Inspect outer source", EffectProperties{}, func(nested *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
					check(nested, want)
					check(nested.Clone(), want)
					return nil
				})}}
				game.PushStack(inner)
				game.ResolveTopOfStack()
				check(game, want)
				return nil
			}))
			g.PushStack(object)
			g.ResolveTopOfStack()
			if checks != 1 || len(query(g)) != 0 {
				t.Fatal("reference escaped final frame")
			}
		})
	}
}

func TestResolvingObjectSourcesInvalidReferences(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Unreferenced", "{R}")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	object := &StackObject{ID: uuid.New(), IsAbility: true, SourceID: p.ID(), sourceIncarnation: uuid.New(), Controller: a.PlayerID(), capturedPermanents: map[*PermanentBinding]capturedPermanent{nil: {p.ID(), p.incarnationID}, BindPermanent("wrong"): {p.ID(), uuid.New()}}}
	object.Effects = []Effect{Sequence(&eventSpellCapture{id: uuid.New()}, &eventDamageSourceCapture{source: g.CaptureDamageSource(p.ID())}), FuncEffect("Inspect invalid references", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		for _, query := range []func(*Game) []DamageSource{(*Game).StackAbilityPermanentSources, (*Game).StackTargetPermanentSources, (*Game).StackBoundPermanentSources, (*Game).StackTargetSpellSources, (*Game).StackBoundSpellSources, (*Game).StackBoundDamageSources} {
			if len(query(game)) != 0 {
				t.Fatal("invalid or unbound reference admitted")
			}
		}
		return nil
	})}
	g.PushStack(object)
	g.ResolveTopOfStack()
}
