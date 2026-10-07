package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStackAbilityPermanentSources(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "face-down"} {
		t.Run(change, func(t *testing.T) {
			g, a, _, spell := spellSourceFixture(t)
			card := NewCreature("Source", "{U}", 1, 1, WithSubTypes("Wizard"))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			ability := &StackObject{ID: uuid.New(), SourceID: p.ID(), Controller: a.PlayerID(), IsAbility: true}
			g.PushStack(ability)
			if g.CopyStackObjectDirect(ability, a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
					g.ChangeObjectColor(card.ID(), core.Red)
				}
			case "phased", "phase-cycle":
				ids := g.PhaseOut(p.ID())
				if change == "phase-cycle" {
					g.PhaseIn(ids)
				}
			case "recolored":
				g.ChangeObjectColor(p.ID(), core.Red)
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			}
			before := g.StackSize()
			sources := g.StackAbilityPermanentSources()
			if len(sources) != 1 || !sources[0].SameObject(original) || g.StackSize() != before {
				t.Fatal("source identity, duplicate removal or pure query")
			}
			if change == "face-down" {
				if sources[0].Name() != "" || sources[0].HasColor(core.Blue) || sources[0].HasSubType("Wizard") {
					t.Fatal("hidden source characteristics exposed")
				}
			} else if sources[0].HasColor(core.Red) != (change == "recolored") || sources[0].HasColor(core.Blue) == (change == "recolored") {
				t.Fatal("current and retained characteristics merged")
			}
			clone := g.Clone()
			sources[0].card.AddType(core.TypeArtifact)
			if clone.StackAbilityPermanentSources()[0].HasType(core.TypeArtifact) || g.StackAbilityPermanentSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("query exposed mutable game data")
			}
			g.GetStack().RemoveByID(ability.ID)
			g.GetStack().Pop()
			if len(g.StackAbilityPermanentSources()) != 0 || len(clone.StackAbilityPermanentSources()) != 1 || g.StackPeek().ID != spell.ID {
				t.Fatal("resolved references remained eligible or clone changed")
			}
		})
	}
}

func TestStackAbilityPermanentSourceIncarnations(t *testing.T) {
	g, a, b, _ := spellSourceFixture(t)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	old := g.CaptureDamageSource(p.ID())
	first := &StackObject{ID: uuid.New(), SourceID: p.ID(), IsAbility: true, Controller: a.PlayerID()}
	g.PushStack(first)
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	p = g.PutOnBattlefield(card, b.PlayerID())
	g.ChangeObjectColor(p.ID(), core.Red)
	second := &StackObject{ID: uuid.New(), SourceID: p.ID(), IsAbility: true, Controller: b.PlayerID()}
	g.PushStack(second)
	sources := g.StackAbilityPermanentSources()
	if len(sources) != 2 || !sources[0].SameObject(old) || !sources[1].SameObject(g.CaptureDamageSource(p.ID())) || sources[0].Controller() != a.PlayerID() || sources[1].Controller() != b.PlayerID() || sources[0].HasColor(core.Red) || !sources[1].HasColor(core.Red) {
		t.Fatal("separate source visits merged or insertion order changed")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if got := g.StackAbilityPermanentSources(); got[1].Controller() != a.PlayerID() || got[0].Controller() != a.PlayerID() {
		t.Fatal("live source did not read current controller")
	}
	g.GetStack().RemoveByID(first.ID)
	if got := g.StackAbilityPermanentSources(); len(got) != 1 || !got[0].SameObject(sources[1]) {
		t.Fatal("removing one reference removed a separate incarnation")
	}
}

func TestStackAbilityPermanentSourceDeathTrigger(t *testing.T) {
	g, a, _, _ := spellSourceFixture(t)
	card := NewCreature("Dying source", "{U}", 1, 1, WithAbility(WhenEvent(Dies(This), GainLifeFor(You, Fixed(1)))))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	original := g.CaptureDamageSource(p.ID())
	g.DestroyPermanent(p)
	if len(g.StackAbilityPermanentSources()) != 0 {
		t.Fatal("pending trigger was treated as a stack object")
	}
	g.PutTriggersOnStack()
	if got := g.StackAbilityPermanentSources(); len(got) != 1 || !got[0].SameObject(original) {
		t.Fatal("death trigger lost departed source identity")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if len(g.StackAbilityPermanentSources()) != 0 || len(clone.StackAbilityPermanentSources()) != 1 || a.Life() != 21 {
		t.Fatal("resolved trigger remained eligible or changed clone")
	}
}

func TestStackAbilityPermanentSourceMissingIdentity(t *testing.T) {
	g, a, _, _ := spellSourceFixture(t)
	p := g.PutOnBattlefield(NewArtifact("Unreferenced", ""), a.PlayerID())
	g.GetStack().Push(&StackObject{ID: uuid.New(), SourceID: p.ID(), IsAbility: true})
	g.GetStack().Push(&StackObject{ID: uuid.New(), SourceID: p.ID(), sourceIncarnation: uuid.New(), IsAbility: true})
	if len(g.StackAbilityPermanentSources()) != 0 {
		t.Fatal("missing or stale incarnation used current card")
	}
	g.PushStack(&StackObject{ID: uuid.New(), SourceID: p.ID(), IsAbility: true})
	if len(g.StackAbilityPermanentSources()) != 1 {
		t.Fatal("object without damage ability was excluded")
	}
}

func TestStackAbilityPermanentSourceChoice(t *testing.T) {
	g, a, _, _ := spellSourceFixture(t)
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g.players[0] = chooser
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: p.ID(), IsAbility: true}
	g.PushStack(obj)
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choice := ChooseDamageSourceFromStackAbilities(You, DamageSourceHasType(core.TypeArtifact), one)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || len(chooser.seen) != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(g.CaptureDamageSource(p.ID())) {
		t.Fatal("filtered source choice failed")
	}
	if err := ChooseDamageSourceFromStackAbilities(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice changed independent result")
	}
	clone := g.Clone()
	g.GetStack().RemoveByID(obj.ID)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("empty list retained result or changed clone")
	}
	if err := ChooseDamageSourceFromStackAbilities(FromPlayer(BindPlayer("absent")), DamageSourceFilter{}, one).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(&EffectContext{Game: clone}).Valid() {
		t.Fatal("absent player retained result")
	}
}

func TestStackAbilityPermanentSourceDeclarations(t *testing.T) {
	binding := BindDamageSource("source")
	valid := Sequence(ChooseDamageSourceFromStackAbilities(You, DamageSourceFilter{}, binding), DealDamageFrom(FromDamageSource(binding), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromStackAbilities(nil, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromStackAbilities(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromStackAbilities(You, DamageSourceFilter{}, BindDamageSource("")),
		ChooseDamageSourceFromStackAbilities(You, DamageSourceHasColor(core.AnyColor), binding),
		Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), ChooseDamageSourceFromStackAbilities(You, DamageSourceFilter{}, binding)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}
