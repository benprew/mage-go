package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCommandDamageSourceVisits(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Command object", "{R}")
	card.SetOwner(a.PlayerID())
	visit, err := g.AddCommandObject(card, false)
	if err != nil {
		t.Fatal(err)
	}
	sources := g.CommandDamageSources()
	if len(sources) != 1 || sources[0].CommandVisit() != visit || sources[0].Incarnation() != uuid.Nil || sources[0].StackID() != uuid.Nil || !sources[0].HasColor(core.Red) {
		t.Fatal("command source identity or information")
	}
	saved := sources[0]
	if saved.SameObject(DamageSource{id: saved.ID(), incarnation: visit}) || saved.SameObject(DamageSource{id: saved.ID(), stackID: visit}) {
		t.Fatal("zone identities merged")
	}
	if _, err := g.AddCommandObject(card, false); err == nil {
		t.Fatal("duplicate command object accepted")
	}
	if _, err := g.AddCommandObject(nil, false); err == nil {
		t.Fatal("nil command object accepted")
	}
	invalid := card.Copy()
	invalid.SetID(uuid.Nil)
	if _, err := g.AddCommandObject(invalid, false); err == nil {
		t.Fatal("missing object ID accepted")
	}
	card.AddType(core.TypeEnchantment)
	sources[0].card.AddType(core.TypeCreature)
	if fresh := g.CommandDamageSources(); fresh[0].HasType(core.TypeEnchantment) || fresh[0].HasType(core.TypeCreature) {
		t.Fatal("storage exposed card data")
	}
	clone := g.Clone()
	if !g.SetCommandObjectFaceDown(visit, true) || len(g.CommandDamageSources()) != 0 || len(clone.CommandDamageSources()) != 1 {
		t.Fatal("hidden objects or clone isolation")
	}
	if g.SetCommandObjectFaceDown(uuid.New(), false) {
		t.Fatal("unknown visit changed visibility")
	}
	if !g.SetCommandObjectFaceDown(visit, false) || !g.CommandDamageSources()[0].SameObject(saved) {
		t.Fatal("face change replaced identity")
	}
	removed, ok := g.RemoveCommandObject(visit)
	if !ok || removed.ID() != card.ID() || len(g.CommandDamageSources()) != 0 {
		t.Fatal("departure failed")
	}
	next, err := g.AddCommandObject(removed, false)
	if err != nil {
		t.Fatal(err)
	}
	if next == visit || g.CommandDamageSources()[0].SameObject(saved) || !clone.CommandDamageSources()[0].SameObject(saved) {
		t.Fatal("return or clone lost visit identity")
	}
	if _, ok := g.RemoveCommandObject(visit); ok {
		t.Fatal("stale visit removed returned object")
	}
	if !g.damageReferenceInformation(saved).SameObject(saved) {
		t.Fatal("departure lost retained information")
	}
	clone.zones.commandObjects[0].card.AddType(core.TypeCreature)
	if g.CommandDamageSources()[0].HasType(core.TypeCreature) {
		t.Fatal("clone shared command card")
	}
}

func TestCommandDamageSourceChoiceAndPrevention(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a}
	g := NewGame(chooser, b)
	hidden := NewArtifact("Hidden", "{R}")
	if _, err := g.AddCommandObject(hidden, true); err != nil {
		t.Fatal(err)
	}
	card := NewArtifact("Visible", "{R}")
	visit, err := g.AddCommandObject(card, false)
	if err != nil {
		t.Fatal(err)
	}
	saved := g.CommandDamageSources()[0]
	one, two := BindDamageSource("one"), BindDamageSource("two")
	choice := ChooseDamageSourceFromCommandObjects(You, DamageSourceHasColor(core.Red), one)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	body := Sequence(choice, PreventNextDamageFromSelectedSource(FromDamageSource(one), You, DamageSourceHasColor(core.Red)))
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 || len(chooser.seen) != 1 || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("choice included hidden object or lost result")
	}
	clone := g.Clone()
	removed, _ := g.RemoveCommandObject(visit)
	if _, err := g.AddCommandObject(removed, false); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayerFromSource(a, 2, g.CommandDamageSources()[0])
	if a.Life() != 18 {
		t.Fatal("returned command object inherited shield")
	}
	g.DealDamageToPlayerFromSource(a, 2, saved)
	if a.Life() != 18 {
		t.Fatal("original command damage not prevented")
	}
	g.DealDamageToPlayerFromSource(a, 1, saved)
	if a.Life() != 17 {
		t.Fatal("shield did not consume")
	}
	clone.DealDamageToPlayerFromSource(clone.GetPlayer(a.PlayerID()), 3, saved)
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone lost shield")
	}
	if err := ChooseDamageSourceFromCommandObjects(You, DamageSourceHasColor(core.Blue), two).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(two).resolveDamageSource(ctx).Valid() || !FromDamageSource(one).resolveDamageSource(ctx).SameObject(saved) {
		t.Fatal("bindings merged")
	}
	g.SetCommandObjectFaceDown(g.CommandDamageSources()[0].CommandVisit(), true)
	if err := choice.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromDamageSource(one).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice kept old result")
	}
}

func TestCommandDamageSourceDeclarations(t *testing.T) {
	binding := BindDamageSource("command")
	choice := ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, binding)
	valid := Sequence(choice, DealDamageFrom(FromDamageSource(binding), You, Fixed(1)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		ChooseDamageSourceFromCommandObjects(nil, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, BindDamageSource("")),
		ChooseDamageSourceFromCommandObjects(You, AnyOfDamageSources(), binding),
		Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), choice),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestCommandDamageSourceRetainedVisibility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Secret", "{R}")
	visit, err := g.AddCommandObject(card, false)
	if err != nil {
		t.Fatal(err)
	}
	saved := g.CommandDamageSources()[0]
	g.SetCommandObjectFaceDown(visit, true)
	g.RemoveCommandObject(visit)
	information := g.damageReferenceInformation(saved)
	if !information.SameObject(saved) || information.Name() != "" || information.HasColor(core.Red) || information.HasType(core.TypeArtifact) {
		t.Fatal("departed hidden object exposed face-up information")
	}
	clone := g.Clone()
	if !clone.damageReferenceInformation(saved).SameObject(information) || clone.damageReferenceInformation(saved).HasColor(core.Red) {
		t.Fatal("clone lost departure information")
	}
	if _, err := g.AddCommandObject(card, false); err != nil {
		t.Fatal(err)
	}
	if g.damageReferenceInformation(saved).HasColor(core.Red) {
		t.Fatal("return replaced departure information")
	}
}

func TestCommandDamageSourceBatchAndOrder(t *testing.T) {
	for choice := range 2 {
		a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		if _, err := g.AddCommandObject(NewArtifact("Command", "{R}"), false); err != nil {
			t.Fatal(err)
		}
		saved := g.CommandDamageSources()[0]
		binding := BindDamageSource("command")
		if err := ApplyEffect(g, Sequence(ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, binding), PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceFilter{})), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.AddPreventionShield(a.PlayerID(), 1)
		g.DealDamageToPlayerFromSource(a, 1, saved)
		if a.Life() != 20 || a.prompts != 1 {
			t.Fatal("replacement choice changed")
		}
		g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, saved), PlayerDamageAssignment(a, 3, saved))
		want := 16
		if choice == 1 {
			want = 20
		}
		if a.Life() != want {
			t.Fatalf("batch life=%d want=%d", a.Life(), want)
		}
	}
}

func TestCommandDamageSourceRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if _, err := g.AddCommandObject(NewArtifact("Command", ""), false); err != nil {
		t.Fatal(err)
	}
	saved := g.CommandDamageSources()[0]
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
	binding := BindDamageSource("command")
	if err := ApplyEffect(g, Sequence(ChooseDamageSourceFromCommandObjects(You, DamageSourceFilter{}, binding), RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceFilter{})), target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.DealSimultaneousDamage(PermanentDamageAssignment(game.FindPermanent(target.ID()), 2, saved), PermanentDamageAssignment(game.FindPermanent(target.ID()), 3, saved))
		if game.GetPlayer(a.PlayerID()).Life() != 15 || game.FindPermanent(target.ID()).Damage != 0 {
			t.Fatal("command redirect or batch failed")
		}
	}
}

func TestCommandDamageSourceZoneLookup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Command", "{R}")
	visit, err := g.AddCommandObject(card, false)
	if err != nil {
		t.Fatal(err)
	}
	found := g.FindCardAnywhere(card.ID())
	if found == nil || found.ID() != card.ID() {
		t.Fatal("command zone missing from object lookup")
	}
	found.AddType(core.TypeCreature)
	if g.CommandDamageSources()[0].HasType(core.TypeCreature) {
		t.Fatal("lookup exposed command storage")
	}
	g.RemoveCommandObject(visit)
	if g.FindCardAnywhere(card.ID()) != nil {
		t.Fatal("lookup retained departed command object")
	}
}

func TestCommandDamageSourceCapturedReferences(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Command", "{R}")
	visit, err := g.AddCommandObject(card, false)
	if err != nil {
		t.Fatal(err)
	}
	saved := g.CommandDamageSources()[0]
	binding := BindDamageSource("command")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := (&eventDamageSourceCapture{binding: binding, source: saved}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := AtNext(EachEndStep, DealDamageFrom(FromDamageSource(binding), You, Fixed(2))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.Clear()
	g.RemoveCommandObject(visit)
	if _, err := g.AddCommandObject(card, false); err != nil {
		t.Fatal(err)
	}
	g.zones.ClearLKI()
	for _, game := range []*Game{g, g.Clone()} {
		sources := game.WaitingDelayedDamageSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) {
			t.Fatal("captured command visit not retained")
		}
		game.SetStep(core.EndStep)
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		if len(game.WaitingDelayedDamageSources()) != 0 {
			t.Fatal("triggered command reference still waiting")
		}
		sources = game.StackBoundDamageSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) {
			t.Fatal("stack lost exact command visit")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 18 {
			t.Fatal("captured command damage failed")
		}
	}
}
