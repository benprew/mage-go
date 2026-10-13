package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentTypeComparison(t *testing.T) {
	for _, scenario := range []string{"live", "restricted", "excluded", "empty-types", "layer", "leave", "return", "phase", "cycle", "unknown", "member-phase", "control"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Reference", "", 2, 5, WithCardType(core.TypeArtifact))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			creature := g.PutOnBattlefield(NewCreature("Creature", "", 1, 5), b.PlayerID())
			artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
			land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
			var kinds []core.CardType
			wantCreature, wantArtifact, wantLand := true, true, false
			switch scenario {
			case "restricted":
				kinds = []core.CardType{core.TypeArtifact}
				wantCreature = false
			case "excluded":
				kinds = []core.CardType{core.TypeLand}
				wantCreature, wantArtifact = false, false
			case "empty-types":
				source.Card.(*BaseCard).types = nil
				source.RevokeBaseAttr(core.AttrIsCreature)
				source.RevokeBaseAttr(core.AttrIsArtifact)
				wantCreature, wantArtifact = false, false
			case "layer":
				if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, core.TypeLand)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantLand = true
			case "leave", "return":
				g.BouncePermanentToHand(source)
				if scenario == "return" {
					a.RemoveFromHand(card.ID())
					card.types = []core.CardType{core.TypeLand}
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "unknown":
				ctx.sourceIncarnation = uuid.New()
				wantCreature, wantArtifact = false, false
			case "member-phase":
				g.PhaseOut(artifact.ID())
				wantArtifact = false
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: artifact.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			set := PermanentsSharingCardTypes(This, MatchingPermanents(AnyPermanent), kinds...)
			for _, game := range []*Game{g, g.Clone()} {
				local := *ctx
				local.Game = game
				game.SetRandomResults([]int{7})
				got := set.resolvePermanents(&local)
				for _, check := range []struct {
					id   uuid.UUID
					want bool
				}{{creature.ID(), wantCreature}, {artifact.ID(), wantArtifact}, {land.ID(), wantLand}} {
					found := false
					for _, p := range got {
						found = found || p.ID() == check.id
					}
					if found != check.want {
						t.Fatalf("%s: member %s = %v, want %v", scenario, check.id, found, check.want)
					}
				}
				if game.RandIntn(10) != 7 || creature.Tapped || artifact.Tapped {
					t.Fatal("query changed state")
				}
			}
		})
	}
}

func TestPermanentTypeComparisonDeclarations(t *testing.T) {
	target := DeclarePermanentTarget("target", AnyPermanent)
	choice := BindPermanent("choice")
	for _, set := range []PermanentSet{
		PermanentsSharingCardTypes(nil, MatchingPermanents(AnyPermanent)),
		PermanentsSharingCardTypes(This, nil),
		PermanentsSharingCardTypes(This, MatchingPermanents(AnyPermanent), core.CardType(-1)),
		PermanentsSharingCardTypes(This, MatchingPermanents(AnyPermanent), core.TypeLand, core.TypeLand),
		PermanentsSharingCardTypes(target, MatchingPermanents(AnyPermanent)),
		PermanentsSharingCardTypes(From(choice), MatchingPermanents(AnyPermanent)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), TapEach(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid comparison accepted")
		}
	}
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), TapEach(PermanentsSharingCardTypes(target, MatchingPermanents(AnyPermanent))), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Reference", "", 1, 5, WithCardType(core.TypeArtifact)), a.PlayerID())
	kinds := []core.CardType{core.TypeArtifact}
	set := PermanentsSharingCardTypes(This, MatchingPermanents(AnyPermanent), kinds...)
	kinds[0] = core.TypeLand
	if len(set.resolvePermanents(&EffectContext{Game: g, SourceID: p.ID()})) != 1 {
		t.Fatal("constructor shares type list storage")
	}
}

func TestPermanentTypeComparisonSavedAndWaiting(t *testing.T) {
	for _, refKind := range []string{"source", "target", "saved", "host"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "face-down"} {
			t.Run(refKind+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewArtifact("Reference", "")
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				artifact := g.PutOnBattlefield(NewArtifact("Member", ""), b.PlayerID())
				creature := g.PutOnBattlefield(NewCreature("Creature", "", 1, 5), b.PlayerID())
				target := DeclarePermanentTarget("reference", AnyPermanent)
				binding := BindPermanent("saved")
				ctx := &EffectContext{Game: g, SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				if err := Snapshot(target, binding).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				var ref PermanentRef = This
				switch refKind {
				case "target":
					ref = target
				case "saved":
					ref = From(binding)
				case "host":
					aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
					g.Attach(aura.ID(), p.ID())
					ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
					ref = AttachedPermanent
				}
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						card.types = []core.CardType{core.TypeCreature}
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "face-down":
					p.FaceDown = true
				}
				set := PermanentsSharingCardTypes(ref, MatchingPermanents(AnyPermanent))
				for _, game := range []*Game{g, g.Clone()} {
					local := *ctx
					local.Game = game
					if err := TapEach(set).Apply(&local); err != nil {
						t.Fatal(err)
					}
					if game.FindPermanent(artifact.ID()).Tapped != (state != "face-down") || game.FindPermanent(creature.ID()).Tapped != (state == "face-down") {
						t.Fatal("saved reference changed visits or types")
					}
				}
			})
		}
	}
	for _, operand := range []string{"reference", "subjects"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		reads := 0
		filter := NewPermanentFilter("must not run", func(*Permanent, *Game) bool { reads++; return true })
		ref, subjects := PermanentRef(This), MatchingPermanents(filter)
		if operand == "subjects" {
			ref, subjects = AttachedPermanent, FilterPermanentSet(SingletonPermanent(This), filter)
		}
		set := PermanentsSharingCardTypes(ref, subjects)
		if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(set)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || p.Tapped {
				t.Fatal("waiting query executed comparison or lost source")
			}
		}
	}
}
