package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingExileSourcesPermanent(t *testing.T) {
	for _, state := range []string{"live", "phase", "cycle", "control", "depart", "return", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Subject", "{R}")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
			g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
			clone := g.Clone()
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				g.zones.ClearLKI()
				if state == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "cleanup":
				g.ClearExileInsteadOfGraveyardForTurn()
			}
			want := 1
			if state == "cleanup" {
				want = 0
			}
			refs := g.WaitingExileSources()
			if len(refs) != want {
				t.Fatalf("sources=%d want=%d", len(refs), want)
			}
			if want == 1 {
				if !refs[0].SameObject(saved) {
					t.Fatal("query followed a returned object")
				}
				if state == "control" && refs[0].Controller() != b.PlayerID() {
					t.Fatal("stale controller")
				}
				refs[0].card.AddType(core.TypeCreature)
				if g.WaitingExileSources()[0].HasType(core.TypeCreature) {
					t.Fatal("mutable query result")
				}
			}
			if len(clone.WaitingExileSources()) != 1 {
				t.Fatal("clone changed")
			}
			if live := g.FindPermanent(p.ID()); live != nil {
				g.DoSacrifice(live)
				replaced := state != "return" && state != "cleanup"
				if (g.FindExiledCard(card.ID()) != nil) != replaced {
					t.Fatal("replacement identity disagrees with query")
				}
			}
		})
	}
}

func TestWaitingExileSourcesSpell(t *testing.T) {
	for _, state := range []string{"live", "color", "control", "counter", "recast", "copy", "continued", "return", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "{R}", 2, 2)
			card.SetOwner(a.PlayerID())
			obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()}
			g.stack.Push(obj)
			saved := g.CaptureSpellDamageSource(obj.ID)
			g.AddExileIfWouldGoToGraveyardThisTurn(card.ID(), uuid.New())
			clone := g.Clone()
			switch state {
			case "color":
				g.ChangeObjectColor(obj.ID, core.Blue)
			case "control":
				obj.Controller = b.PlayerID()
			case "counter", "recast":
				g.CounterStackObject(obj.ID)
				if state == "recast" {
					g.stack.Push(&StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()})
				}
			case "copy":
				g.CopyStackObjectDirect(obj, a.PlayerID(), false)
			case "continued", "return":
				g.ResolveTopOfStack()
				if state == "return" {
					p := g.FindPermanent(card.ID())
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "cleanup":
				g.ClearExileInsteadOfGraveyardForTurn()
			}
			refs := g.WaitingExileSources()
			want := 1
			if state == "continued" || state == "return" {
				want = 2
			}
			if state == "cleanup" {
				want = 0
			}
			if len(refs) != want {
				t.Fatalf("sources=%d want=%d", len(refs), want)
			}
			if want > 0 && !refs[0].SameObject(saved) {
				t.Fatal("spell identity changed")
			}
			if state == "color" && !refs[0].HasColor(core.Blue) {
				t.Fatal("stale spell color")
			}
			if state == "control" && refs[0].Controller() != b.PlayerID() {
				t.Fatal("stale spell controller")
			}
			if len(clone.WaitingExileSources()) != 1 {
				t.Fatal("clone changed")
			}
			if state == "continued" || state == "return" {
				p := g.FindPermanent(card.ID())
				if state == "continued" && !refs[1].SameObject(g.CaptureDamageSource(p.ID())) {
					t.Fatal("missing continuation")
				}
				if state == "return" && refs[1].SameObject(g.CaptureDamageSource(p.ID())) {
					t.Fatal("return inherited continuation")
				}
				g.DoSacrifice(p)
				if (g.FindExiledCard(card.ID()) != nil) != (state == "continued") {
					t.Fatal("wrong continued replacement")
				}
			}
		})
	}
}

func TestWaitingExileSourcesStackExit(t *testing.T) {
	for _, counter := range []bool{false, true} {
		t.Run(map[bool]string{false: "resolve", true: "counter"}[counter], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewInstant("Subject", "{R}")
			card.SetOwner(a.PlayerID())
			obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()}
			g.stack.Push(obj)
			g.AddExileIfWouldGoToGraveyardThisTurn(card.ID(), uuid.New())
			if counter {
				g.CounterStackObject(obj.ID)
			} else {
				g.ResolveTopOfStack()
			}
			if g.FindExiledCard(card.ID()) == nil || len(a.Graveyard()) != 0 {
				t.Fatal("exact spell did not go to exile")
			}
			g.RemoveFromExile(card.ID())
			later := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()}
			g.stack.Push(later)
			if counter {
				g.CounterStackObject(later.ID)
			} else {
				g.ResolveTopOfStack()
			}
			if g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("later cast inherited old rule")
			}
		})
	}
}

func TestWaitingExileSourcesOriginDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	origin := g.PutOnBattlefield(NewArtifact("Origin", ""), a.PlayerID())
	subject := g.PutOnBattlefield(NewArtifact("Subject", "{R}"), a.PlayerID())
	g.AddExileIfWouldGoToGraveyardThisTurn(subject.ID(), origin.ID())
	g.RemoveFromBattlefield(origin)
	if len(g.WaitingExileSources()) != 1 {
		t.Fatal("resolved rule ended on origin departure")
	}
	g.DoSacrifice(subject)
	if g.FindExiledCard(subject.Card.ID()) == nil {
		t.Fatal("origin departure stopped exile")
	}
}

func TestWaitingExileSourcesEntryTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "{R}", 2, 2, WithAbility(ETBEffect(&waitingExileEntrySacrifice{})))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()}
	g.stack.Push(obj)
	g.AddExileIfWouldGoToGraveyardThisTurn(card.ID(), card.ID())
	g.ResolveTopOfStack()
	if g.FindExiledCard(card.ID()) == nil || len(g.WaitingExileSources()) != 2 {
		t.Fatalf("entry: exiled=%v refs=%d live=%v grave=%d", g.FindExiledCard(card.ID()) != nil, len(g.WaitingExileSources()), g.FindPermanent(card.ID()) != nil, len(a.Graveyard()))
	}
}

func TestWaitingExileSourcesChoices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Exile subject", "{R}"), a.PlayerID())
	g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), uuid.New())
	g.PutOnBattlefield(NewArtifact("Unrelated", "{R}"), a.PlayerID())
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		choice := ChooseDamageSourceFromWaitingExileReferences(You, DamageSourceHasColor(core.Red), binding)
		if err := ValidateCardDefinition("Choice", NewArtifact("Choice", "", WithAbility(Activate(Costs(), choice)))); err != nil {
			t.Fatal(err)
		}
		if err := choice.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if !g.resolution.damageSourceBindings[binding].SameObject(g.CaptureDamageSource(source.ID())) {
			t.Fatal("wrong choice")
		}
	}
	if err := ChooseDamageSourceFromWaitingExileReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("binding isolation")
	}
	ctx.Controller = uuid.New()
	if err := ChooseDamageSourceFromWaitingExileReferences(You, DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("absent player retained choice")
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingExileReferences(nil, DamageSourceFilter{}, first),
		ChooseDamageSourceFromWaitingExileReferences(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingExileReferences(You, DamageSourceFilter{}, BindDamageSource("")),
		Sequence(DealDamageFrom(FromDamageSource(first), You, Fixed(1)), ChooseDamageSourceFromWaitingExileReferences(You, DamageSourceFilter{}, first)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingExileSourcesLegacyAndInvalid(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.AddExileIfWouldGoToGraveyardThisTurn(uuid.New(), uuid.New())
	p := g.PutOnBattlefield(NewArtifact("Subject", "{R}"), a.PlayerID())
	for _, subject := range []DamageSource{{id: p.ID()}, {id: p.ID(), incarnation: p.incarnationID, stackID: uuid.New()}} {
		g.effects.AddReplacement(&exileInsteadOfGraveyardReplacement{subject: subject})
	}
	if len(g.WaitingExileSources()) != 0 {
		t.Fatal("legacy or invalid reference became exact")
	}
	g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
	rule := g.effects.replacements[len(g.effects.replacements)-1]
	g.effects.cycleReplacements = append(g.effects.cycleReplacements, rule.Clone())
	if len(g.WaitingExileSources()) != 1 {
		t.Fatal("duplicate exact subjects")
	}
	g.effects.RemoveReplacements(p.ID())
	if len(g.WaitingExileSources()) != 0 {
		t.Fatal("explicit removal kept rule")
	}
}

type waitingExileEntrySacrifice struct{}

func (*waitingExileEntrySacrifice) Text() string                 { return "entry sacrifice probe" }
func (*waitingExileEntrySacrifice) Properties() EffectProperties { return EffectProperties{} }
func (*waitingExileEntrySacrifice) Apply(ctx *EffectContext) error {
	ctx.Game.DoSacrifice(ctx.Game.FindPermanent(ctx.SourceID))
	return nil
}

func TestWaitingExileSourcesRetainedDepartureInformation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Subject", "{R}"), a.PlayerID())
	g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
	g.MutablePermanent(p.ID()).ColorOverride = &[]core.Color{core.Blue}
	g.RemoveFromBattlefield(p)
	g.zones.ClearLKI()
	refs := g.WaitingExileSources()
	if len(refs) != 1 || !refs[0].HasColor(core.Blue) || refs[0].HasColor(core.Red) {
		t.Fatal("departure data was replaced by registration data")
	}
}

func TestWaitingExileSourcesLegacyMarkerQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Subject", "{R}"), a.PlayerID())
	g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
	g.RemoveFromBattlefield(p)
	if !g.IsCardMarkedExileInsteadOfGraveyard(p.Card.ID()) {
		t.Fatal("card marker query lost registration")
	}
	g.ClearExileInsteadOfGraveyardForTurn()
	if g.IsCardMarkedExileInsteadOfGraveyard(p.Card.ID()) {
		t.Fatal("cleanup retained registration")
	}
}
