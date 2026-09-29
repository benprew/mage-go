package mage

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_announced_target_count_test.go ---
func TestAnnouncedTargetCountKeepsIllegalTargetsAndCopyFrame(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareUpToXPermanentTargets("creatures", IsCreature)
	spell := NewInstant("Count targets", "{X}", WithAbility(Spell(
		GainLifeFor(You, AnnouncedTargetCount(targets)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(0, 2)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(second.ID())
	clone.ResolveStackObject(copyObj)
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 22 {
		t.Fatalf("copy life = %d, want 22", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	g.PutOnBattlefield(second.Card, b.PlayerID())
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("original life = %d, want 22", got)
	}
}

func TestAnnouncedTargetCountZeroAndRegistration(t *testing.T) {
	targets := DeclareUpToXPermanentTargets("creatures", IsCreature)
	value := AnnouncedTargetCount(targets)
	if got := value.resolveValue(&EffectContext{}); got != 0 {
		t.Fatalf("absent group count = %d, want 0", got)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Good", "{X}", WithAbility(Spell(GainLifeFor(You, value), WithTargets(targets)))),
		},
		{
			Name:    "undeclared",
			Card:    NewInstant("Undeclared", "", WithAbility(Spell(GainLifeFor(You, value)))),
			WantErr: true,
		},
		{
			Name:    "nil targets binding",
			Card:    NewInstant("Nil", "", WithAbility(Spell(GainLifeFor(You, AnnouncedTargetCount((*PermanentTargetsBinding)(nil)))))),
			WantErr: true,
		},
	})
}

func TestAnnouncedTargetCountUsesOneRecordedGroup(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	first := DeclarePermanentTargets("first", 0, 2, IsCreature)
	second := DeclarePermanentTargets("second", 0, 2, IsCreature)
	obj := &StackObject{
		Targets:     []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		TargetSpecs: []Target{first.targetDeclaration, first.targetDeclaration, second.targetDeclaration},
	}
	defer g.resolution.Begin(obj)()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if got := AnnouncedTargetCount(first).resolveValue(ctx); got != 2 {
		t.Fatalf("first group count = %d, want 2", got)
	}
	if got := AnnouncedTargetCount(second).resolveValue(ctx); got != 1 {
		t.Fatalf("second group count = %d, want 1", got)
	}
}

// --- from composable_arithmetic_test.go ---
func TestComposableArithmeticSignedValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		value ResolutionValue
		want  int
	}{
		{"multiply", MultiplyValues(Fixed(2), Fixed(3)), 6},
		{"multiply negative", MultiplyValues(Fixed(-2), Fixed(3)), -6},
		{"multiply negatives", MultiplyValues(Fixed(-2), Fixed(-3)), 6},
		{"multiply zero", MultiplyValues(Fixed(0), Fixed(3)), 0},
		{"subtract", SubtractValues(Fixed(-2), Fixed(3)), -5},
		{"subtract negative", SubtractValues(Fixed(2), Fixed(-3)), 5},
		{"minimum", MinValue(Fixed(-2), Fixed(-3)), -3},
		{"maximum", MaxValue(Fixed(-2), Fixed(-3)), -2},
		{"equal", MinValue(Fixed(3), Fixed(3)), 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.value.resolveValue(nil); got != test.want {
				t.Fatalf("value = %d, want %d", got, test.want)
			}
		})
	}
}

func TestComposableArithmeticCapture(t *testing.T) {
	for _, toughness := range []int{-2, 0, 1, 2, 4, 5, 8} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		c := NewCreature("Subject", "", 2, toughness)
		g.PutOnBattlefield(c, a.PlayerID())
		saved := BindPermanent("subject")
		change := MinValue(Fixed(0), MaxValue(Fixed(-4), SubtractValues(Fixed(1), ToughnessOf(From(saved)))))
		effect := Sequence(Snapshot(This, saved), Until(EndOfTurn, ModifyPT(From(saved), Fixed(4), change)))
		if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := toughness
		if toughness > 1 {
			want = max(1, toughness-4)
		}
		for range 3 {
			g.ApplyContinuousEffects()
		}
		p := g.FindPermanent(c.ID())
		if p.CurrentPower(g) != 6 || p.CurrentToughness(g) != want {
			t.Fatalf("initial toughness %d: got %d/%d, want 6/%d", toughness, p.CurrentPower(g), p.CurrentToughness(g), want)
		}
		clone := g.Clone()
		clone.effects.RemoveEndOfTurn()
		clone.ApplyContinuousEffects()
		if clone.FindPermanent(c.ID()).CurrentToughness(clone) != toughness || p.CurrentToughness(g) != want {
			t.Fatal("expiration changed the original game or retained the modification")
		}
	}
}

func TestComposableArithmeticStaticEvaluation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	value := MaxValue(Fixed(0), SubtractValues(ToughnessOf(This), Fixed(1)))
	c := NewCreature("Source", "", 2, 3, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, value, Fixed(0))))))
	g.PutOnBattlefield(c, a.PlayerID())
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 4 {
		t.Fatalf("power = %d, want 4", got)
	}
	g.FindPermanent(c.ID()).Counters[P1P1] = 1
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 6 {
		t.Fatalf("power = %d, want 6", got)
	}
}

func TestComposableArithmeticValidation(t *testing.T) {
	missing := BindPermanent("missing")
	undeclared := DeclarePermanentTarget("undeclared", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "multiply nil right",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MultiplyValues(Fixed(1), nil), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "multiply nil left",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MultiplyValues(nil, Fixed(1)), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "multiply undeclared target",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MultiplyValues(Fixed(2), PowerOf(undeclared)), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "multiply missing producer",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MultiplyValues(Fixed(2), ToughnessOf(From(missing))), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "subtract nil right",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, SubtractValues(Fixed(1), nil), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "min nil left",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MinValue(nil, Fixed(0)), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "max nil right",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MaxValue(Fixed(0), nil), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "min max nested missing producer",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MinValue(Fixed(0), MaxValue(Fixed(-4), SubtractValues(Fixed(1), ToughnessOf(From(missing))))), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "max undeclared target",
			Card:    NewCreature("Invalid arithmetic", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, MaxValue(Fixed(0), PowerOf(undeclared)), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:          "missing producer substring",
			Card:          NewCreature("Missing producer", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, SubtractValues(Fixed(1), ToughnessOf(From(missing))), Fixed(0)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestComposableMultiplySavedValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindNumber("amount")
	body := Sequence(SnapshotNumber(PowerOf(This), saved), AddCountersTo(This, Stun, MultiplyValues(Fixed(2), FromNumber(saved))))
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(c.ID()).Counters[Stun]; got != 4 {
		t.Fatalf("counters = %d, want 4", got)
	}
}

// --- from composable_attach_choice_test.go ---
func TestMayAttachChoiceAndIncarnation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		accept bool
		change string
		move   bool
		prompt bool
	}{
		{"accept", true, "", true, true},
		{"decline", false, "", false, true},
		{"returned aura", true, "return aura", false, false},
		{"returned host", true, "return host", false, false},
		{"phased aura", true, "phase aura", false, false},
		{"phased host", true, "phase host", false, false},
		{"same host", true, "same host", false, false},
		{"illegal host", true, "illegal host", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			b := &tapChoicePlayer{BasePlayer: NewBasePlayer("B"), accept: true}
			g := NewGame(a, b)
			old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), a.PlayerID())
			auraCard := NewAura("Aura", "{W}")
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), old.ID())
			hostCard := NewCreature("Host", "", 2, 2)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			source, destination := BindPermanent("aura"), BindPermanent("host")
			body := Sequence(Snapshot(This, source),
				FuncEffect("change", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
					if g.resolution.permanentBindings == nil {
						g.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
					}
					g.resolution.permanentBindings[destination] = capturedPermanent{host.ID(), host.incarnationID}
					switch tc.change {
					case "return aura":
						g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
						g.PutOnBattlefield(auraCard, b.PlayerID())
					case "return host":
						g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
						g.PutOnBattlefield(hostCard, a.PlayerID())
					case "phase aura":
						g.PhaseOut(aura.ID())
					case "phase host":
						g.PhaseOut(host.ID())
					case "same host":
						g.resolution.permanentBindings[destination] = capturedPermanent{old.ID(), old.incarnationID}
					case "illegal host":
						g.MutablePermanent(host.ID()).GrantBaseAttr(AttrCantBeEnchanted)
					}
					return nil
				}), MayAttach(From(source), From(destination)))
			if err := ApplyEffect(g, body, aura.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if got := len(a.prompts) == 1; got != tc.prompt || len(b.prompts) != 0 {
				t.Fatalf("choice prompt: A=%v B=%v", a.prompts, b.prompts)
			}
			if p := g.FindPermanent(aura.ID()); p != nil && (p.AttachedTo == host.ID()) != tc.move {
				t.Fatalf("attachment host = %v", p.AttachedTo)
			}
		})
	}
}

func TestMayAttachDeclaration(t *testing.T) {
	aura, host := BindPermanent("aura"), BindPermanent("host")
	good := Sequence(Snapshot(This, aura), Snapshot(This, host), MayAttach(From(aura), From(host)))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Good", "", 1, 1, WithAbility(Activate(Costs(), good))),
		},
		{
			Name:    "nil source",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), MayAttach(nil, This)))),
			WantErr: true,
		},
		{
			Name:    "nil host",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), MayAttach(This, nil)))),
			WantErr: true,
		},
		{
			Name:    "unbound host",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), MayAttach(From(aura), This)))),
			WantErr: true,
		},
		{
			Name:    "unbound source",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), MayAttach(This, From(host))))),
			WantErr: true,
		},
	})
	if got := InspectDefinition(MayAttach(This, This)); got.Role == "" || !strings.Contains(got.Text, "attach") {
		t.Fatalf("inspection: %+v", got)
	}
}

func TestMayAttachCreatedTokenTrigger(t *testing.T) {
	a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered enchantment")
	created := BindPermanent("created token")
	cat := DefineToken("Cat", TokenTypes(TypeCreature), TokenSubtypes("Cat"), TokenColors(White), TokenPT(2, 2))
	watcher := NewCreature("Watcher", "", 3, 3, WithAbility(WhenEvent(
		EnchantmentYouControlEnters().WithPermanent(entered),
		Sequence(CreateOneToken(You, cat, created), IfPermanentHasEnchantmentSubtype(From(entered), "Aura",
			MayAttach(From(entered), From(created)))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	old := g.PutOnBattlefield(NewCreature("Old", "", 1, 1), a.PlayerID())
	aura := NewAura("Aura", "{W}")
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), old.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	g.ResolveStack()
	if got := g.FindPermanent(aura.ID()).AttachedTo; got != g.FilterBattlefield(IsToken)[0].ID() {
		t.Fatalf("attachment = %v", got)
	}
	if len(clone.FilterBattlefield(IsToken)) != 0 || clone.FindPermanent(aura.ID()).AttachedTo != old.ID() {
		t.Fatal("original resolution changed clone")
	}
}

// --- from composable_attachment_target_test.go ---
func TestAttachmentTargetResolution(t *testing.T) {
	for _, change := range []string{"none", "opponent aura", "host control", "host type", "move to own", "move to opponent", "detach", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), a.PlayerID())
			opponent := g.PutOnBattlefield(NewCreature("Opponent", "", 2, 2), b.PlayerID())
			aura := NewAura("Aura", "")
			aura.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "opponent aura" {
				controller = b.PlayerID()
			}
			p := g.PutOnBattlefield(aura, controller)
			p.AttachedTo = host.ID()
			target := DeclarePermanentTarget("aura", And(IsEnchantment, HasSubType("Aura"), IsAttachedTo(IsCreature)), TargetAttachedToYouControl())
			ability := Spell(ReturnToHand(target), WithTargets(target))
			spell := NewInstant("Return", "", WithAbility(ability))
			g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			clone := g.Clone()
			cp := clone.MutablePermanent(p.ID())
			switch change {
			case "host control":
				clone.AddControlEffect(ControlEffectSpec{SourceID: spell.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "host type":
				cp.AttachedTo = clone.PutOnBattlefield(NewArtifact("Noncreature", ""), a.PlayerID()).ID()
			case "move to own":
				cp.AttachedTo = other.ID()
			case "move to opponent":
				cp.AttachedTo = opponent.ID()
			case "detach":
				cp.AttachedTo = uuid.Nil
			case "phase":
				clone.PhaseOut(p.ID())
			case "return":
				clone.RemoveFromBattlefield(cp)
				clone.PutOnBattlefield(cp.Card, controller).AttachedTo = host.ID()
			}
			clone.ResolveStack()
			want := 0
			if change == "none" || change == "opponent aura" || change == "move to own" {
				want = 1
			}
			if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != want {
				t.Fatalf("hand = %d, want %d", got, want)
			}
			if g.FindPermanent(p.ID()).AttachedTo != host.ID() || len(a.Hand()) != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestAttachmentTargetCopyController(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := make([]uuid.UUID, 0, 2)
	for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), player)
		aura := NewAura("Aura", "")
		aura.SetOwner(player)
		g.PutOnBattlefield(aura, player).AttachedTo = host.ID()
		targets = append(targets, aura.ID())
	}
	target := DeclarePermanentTarget("aura", And(IsEnchantment, HasSubType("Aura"), IsAttachedTo(IsCreature)), TargetAttachedToYouControl())
	ability := Spell(ReturnToHand(target), WithTargets(target))
	spell := NewInstant("Return", "", WithAbility(ability))
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: targets[:1], TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{targets[1]}}
	cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != targets[1] || obj.Targets[0] != targets[0] {
		t.Fatal("copy did not use its controller for attachment targeting")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("copies did not return their legal targets")
	}
}

// --- from composable_attachments_set_test.go ---
func TestAttachedAurasDynamicStaticCount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	value := MultiplyValues(Fixed(2), CountPermanents(AttachedAuras(This)))
	card := NewCreature("Synthetic wombat", "", 0, 1, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, value, value)))))
	host := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
	equipment := g.PutOnBattlefield(NewArtifact("Equipment", "", WithSubTypes("Equipment")), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.Attach(equipment.ID(), host.ID())
	if power, toughness := g.FindPermanent(host.ID()).CurrentPower(g), g.FindPermanent(host.ID()).CurrentToughness(g); power != 2 || toughness != 3 {
		t.Fatalf("one Aura and one Equipment give %d/%d", power, toughness)
	}
	g.Attach(aura.ID(), other.ID())
	if got := g.FindPermanent(host.ID()).CurrentPower(g); got != 0 {
		t.Fatalf("power after reattachment = %d", got)
	}
	g.Attach(aura.ID(), host.ID())
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
	if got := clone.FindPermanent(host.ID()).CurrentPower(clone); got != 0 {
		t.Fatalf("clone power = %d", got)
	}
	if got := g.FindPermanent(host.ID()).CurrentPower(g); got != 2 {
		t.Fatalf("original power = %d", got)
	}
}

func TestAttachedAurasIncarnationPhasingAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	count := CountPermanents(AttachedAuras(This))
	ctx := &EffectContext{Game: g, SourceID: host.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: host.incarnationID}
	check := func(want int) {
		t.Helper()
		if got := count.resolveValue(ctx); got != want {
			t.Fatalf("attached Aura count = %d, want %d", got, want)
		}
	}
	check(1)
	g.PhaseOut(aura.ID())
	check(0)
	g.PhaseIn([]uuid.UUID{aura.ID()})
	check(1)
	phased := g.PhaseOut(host.ID())
	check(0)
	g.PhaseIn(phased)
	check(1)
	g.RemoveFromBattlefield(host)
	g.PutOnBattlefield(host.Card, a.PlayerID())
	check(0)
	if got := count.resolveValue(&EffectContext{Game: g, SourceID: host.ID(), Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("returned host inherited %d Auras", got)
	}
	target := DeclarePermanentTarget("host", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, CountPermanents(AttachedAuras(This)), Fixed(0)))))),
		},
		{
			Name:    "nil attachment host",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(AttachedAuras(nil)))))),
			WantErr: true,
		},
		{
			Name:    "undeclared attachment host target",
			Card:    NewInstant("Undeclared", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(AttachedAuras(target)))))),
			WantErr: true,
		},
	})
}

// --- from composable_binding_test.go ---
func TestPermanentBindingLifetime(t *testing.T) {
	for _, change := range []string{"source leaves", "host returns", "phasing", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source, host := NewCreature("Source", "", 1, 1), NewCreature("Host", "", 1, 1)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(host, a.PlayerID())
			g.FindPermanent(source.ID()).AttachedTo = host.ID()
			saved := BindPermanent("host")
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
			if err := Snapshot(AttachedPermanent, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "host returns":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(host, a.PlayerID())
			case "phasing":
				ids := g.PhaseOut(host.ID())
				g.PhaseIn(ids)
			case "clone":
				g = g.Clone()
				ctx.Game = g
			}
			if err := Sacrifice(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if exists := g.FindPermanent(host.ID()) != nil; exists != (change == "host returns") {
				t.Fatalf("host exists = %v", exists)
			}
		})
	}
}

func TestPermanentBindingValidation(t *testing.T) {
	saved, sameName := BindPermanent("host"), BindPermanent("host")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sequence(Snapshot(This, saved), Sacrifice(From(saved))), ManaCostOf("{0}"))),
		},
		{
			Name:          "missing",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sacrifice(From(saved)), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "before write",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sequence(Sacrifice(From(saved)), Snapshot(This, saved)), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "branch escape",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sequence(IfState(Controls(IsCreature), Snapshot(This, saved)), Sacrifice(From(saved))), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "inside branch",
			Card: NewCreature("Binding", "", 1, 1, WithActivatedAbility(IfState(Controls(IsCreature), Sequence(Snapshot(This, saved), Sacrifice(From(saved)))), ManaCostOf("{0}"))),
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sequence(Snapshot(This, saved), Snapshot(This, sameName)), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "delayed escape",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sequence(Snapshot(This, saved), RegisterDelayedTriggerStep(EvtTapped, "", Sacrifice(From(saved)))), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "empty label",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Snapshot(This, BindPermanent("")), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "nil binding",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Snapshot(This, nil), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "loop escape",
			Card:          NewCreature("Binding", "", 1, 1, WithActivatedAbility(Sequence(ForEachPermanent(IsCreature, Snapshot(This, saved), "save"), Sacrifice(From(saved))), ManaCostOf("{0}"))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestPermanentBindingFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("saved")
	ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
	if err := Snapshot(This, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	delete(clone.resolution.permanentBindings, saved)
	if _, ok := From(saved).resolvePermanent(ctx); !ok {
		t.Fatal("clone changed original frame")
	}
	before := g.FindPermanent(c.ID())
	if err := ApplyEffect(g, Sacrifice(From(saved)), c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != before {
		t.Fatal("binding leaked into another resolution")
	}
}

func TestPermanentBindingNestedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(This, saved), FuncEffect("nested resolution", EffectProperties{}, func(g *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
		if err := ApplyEffect(g, Sacrifice(From(saved)), source, controller, nil); err != nil {
			return err
		}
		if g.FindPermanent(source) == nil {
			t.Fatal("nested resolution read the outer frame")
		}
		return nil
	}), Sacrifice(From(saved)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("nested resolution lost the outer frame")
	}
	if len(g.resolution.permanentBindings) != 0 {
		t.Fatal("resolution retained its frame")
	}
}

func TestPermanentBindingAbsentSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(This, saved), Snapshot(AttachedPermanent, saved), Sacrifice(From(saved)), GainLifeFor(You, Fixed(2)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("absent snapshot retained its previous subject")
	}
	if a.Life() != 22 {
		t.Fatal("absent subject stopped the sequence")
	}
}

func TestPermanentBindingSnapshotReadsBeforeWrite(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("host")
	effect := Sequence(Snapshot(This, saved), Snapshot(From(saved), saved), Sacrifice(From(saved)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("snapshot erased its input before reading it")
	}
}

// --- from composable_card_set_choice_test.go ---
type cardSetChoicePlayer struct {
	*BasePlayer
	accept bool
	choice uuid.UUID
	seen   []uuid.UUID
}

func (p *cardSetChoicePlayer) ChooseMayAbility(string) bool { return p.accept }
func (p *cardSetChoicePlayer) ChooseCardFromGraveyard(cards []Card, _ string, _ GameReader) Card {
	p.seen = nil
	for _, card := range cards {
		p.seen = append(p.seen, card.ID())
		if card.ID() == p.choice {
			return card
		}
	}
	return nil
}

func TestChooseUpToOneCardCapturedSet(t *testing.T) {
	baseA, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardSetChoicePlayer{BasePlayer: baseA, accept: true}
	g := NewGame(a, b)
	land := NewLand("Land")
	artifact := NewArtifact("Artifact", "")
	other := NewLand("Other")
	for _, card := range []Card{land, artifact, other} {
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
	}
	set, picked := BindCards("discarded"), BindCard("picked")
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{set: {
		{id: land.ID(), visit: g.graveyardVisit(land.ID()), zone: ZoneGraveyard},
		{id: artifact.ID(), visit: g.graveyardVisit(artifact.ID()), zone: ZoneGraveyard},
	}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choose := ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked)
	a.choice = land.ID()
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.seen) != 1 || a.seen[0] != land.ID() {
		t.Fatalf("offered %v", a.seen)
	}
	if card, ok := FromCard(picked).resolveCard(ctx); !ok || card.ID() != land.ID() {
		t.Fatal("wrong chosen visit")
	}
	clone := g.Clone()
	b.RemoveFromGraveyard(land.ID())
	b.AddToHand(land)
	b.RemoveFromHand(land.ID())
	b.AddToGraveyard(land)
	if _, ok := FromCard(picked).resolveCard(ctx); ok {
		t.Fatal("old visit survived return")
	}
	if card, ok := FromCard(picked).resolveCard(&EffectContext{Game: clone, Controller: a.PlayerID()}); !ok || card.ID() != land.ID() {
		t.Fatal("clone lost chosen visit")
	}
	a.choice = other.ID()
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(picked).resolveCard(ctx); ok {
		t.Fatal("out-of-set choice was saved")
	}
	a.accept = false
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(picked).resolveCard(ctx); ok {
		t.Fatal("declined choice kept old result")
	}
}

func TestChooseUpToOneCardDeclaration(t *testing.T) {
	set, picked := BindCards("discarded"), BindCard("picked")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewSorcery("Choice", "", WithAbility(Spell(Sequence(
				Discard(You, Fixed(2), DiscardCardsInto(set)),
				ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked),
				ReturnCardToHand(FromCard(picked)),
			)))),
		},
		{
			Name:          "producer",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "nil player",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ChooseUpToOneCard(nil, FromCards(set), IsLandCard, picked)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil cards",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ChooseUpToOneCard(You, nil, IsLandCard, picked)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil picked",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ChooseUpToOneCard(You, FromCards(set), IsLandCard, nil)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty name",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(Sequence(Discard(You, Fixed(2), DiscardCardsInto(set)), ChooseUpToOneCard(You, FromCards(set), IsLandCard, BindCard("")))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
	})
}

func TestChooseUpToOneCardSpellCopy(t *testing.T) {
	a := &cardSetChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	b.AddToHand(land)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToHand(artifact)
	a.choice = land.ID()
	set, picked := BindCards("discarded"), BindCard("picked")
	target := DeclarePlayerTarget("player")
	spell := NewInstant("Choice", "", WithAbility(Spell(Sequence(
		Discard(target, Fixed(2), DiscardCardsInto(set)),
		ChooseUpToOneCard(You, FromCards(set), IsLandCard, picked),
		ReturnCardToHand(FromCard(picked)),
	), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	copyObj := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	g.ResolveStackObject(copyObj)
	if len(b.Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 {
		t.Fatal("copy choice or clone isolation failed")
	}
	g.ResolveStack()
	if len(b.Hand()) != 1 || b.Hand()[0].ID() != land.ID() {
		t.Fatal("spell did not choose the discarded land")
	}
}

// --- from composable_choice_test.go ---
func TestTypedPermanentChoice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	small := NewCreature("Small", "", 1, 1)
	large := NewCreature("Large", "", 4, 4)
	for _, c := range []Card{small, large} {
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	selected := BindPermanent("least-power creature")
	effect := Sequence(ChoosePermanent(You, LowestPower(MatchingPermanents(IsCreature)), selected), Destroy(From(selected)))
	if err := ValidateCardDefinition("Choice", NewSorcery("Choice", "", WithAbility(Spell(effect)))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(small.ID()) != nil || g.FindPermanent(large.ID()) == nil {
		t.Fatal("choice did not use least power")
	}
}

func TestTypedChoiceClearsAbsentResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("chosen")
	effect := Sequence(Snapshot(This, saved), ChoosePermanent(You, MatchingPermanents(IsArtifact), saved), Destroy(From(saved)))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("empty choice retained an earlier result")
	}
}

// --- from composable_controlled_by_set_test.go ---
func TestPermanentsControlledByCurrentControllerAndLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	foreign := NewArtifact("Foreign", "")
	foreign.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(foreign, b.PlayerID())
	owned := NewArtifact("Owned", "")
	owned.SetOwner(b.PlayerID())
	g.PutOnBattlefield(owned, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Creature", "", 1, 1), b.PlayerID())
	set := PermanentsControlledBy(You, IsArtifact)
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	count := CountPermanents(set)
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("controlled artifacts = %d, want 1", got)
	}
	g.PhaseOut(p.ID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("phased count = %d, want 0", got)
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if got := count.resolveValue(&EffectContext{Game: clone, Controller: b.PlayerID()}); got != 0 {
		t.Fatalf("clone count = %d, want 0", got)
	}
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("original count = %d, want 1", got)
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: owned.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
	g.ApplyContinuousEffects()
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("changed control count = %d, want 0", got)
	}
	if got := CountPermanents(PermanentsControlledBy(You, IsArtifact)).resolveValue(&EffectContext{Game: g, Controller: a.PlayerID()}); got != 2 {
		t.Fatalf("new controller artifact count = %d, want 2", got)
	}
	g.RemoveFromBattlefield(p)
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("departed count = %d, want 0", got)
	}
	g.PutOnBattlefield(foreign, a.PlayerID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("new controller count = %d, want 0", got)
	}
}

func TestPermanentsControlledByTargetAndDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("player")
	body := GainLifeFor(You, CountPermanents(PermanentsControlledBy(target, IsArtifact)))
	spell := Spell(body, WithTargets(target))
	card := NewInstant("Count controlled", "", WithAbility(spell))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: card,
		},
		{
			Name:    "nil player target",
			Card:    NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(PermanentsControlledBy(nil, IsArtifact)))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(PermanentsControlledBy(target, IsArtifact)))))),
			WantErr: true,
		},
	})
	missing := PermanentsControlledBy(FromPlayer(BindPlayer("missing")), IsArtifact)
	if got := CountPermanents(missing).resolveValue(&EffectContext{Game: NewGame(NewBasePlayer("C"), NewBasePlayer("D"))}); got != 0 {
		t.Fatalf("absent player count = %d, want 0", got)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(a.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: spell.Targets(), Effects: spell.Effects()}
	g.pushStack(obj)
	g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("copy lives = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_counter_amount_test.go ---
func TestTypedCounterAmountContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Recipient", "{1}", 1, 4)
	amount := NewCreature("Amount", "{3}", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(amount, b.PlayerID())
	saved := BindPermanent("recipient")
	body := Sequence(Snapshot(This, saved), AddCountersTo(From(saved), P1P1, ValueAtResolution(TargetPermanentManaValue())))
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), []uuid.UUID{amount.ID()}); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(source.ID()).Counters[P1P1]; got != 3 {
		t.Fatalf("counters = %d, want 3 from outer target", got)
	}
	clone := g.Clone()
	body = Sequence(Snapshot(This, saved), AddCountersTo(From(saved), P1P1, PowerOf(From(saved))))
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(source.ID()).Counters[P1P1] != 7 || g.FindPermanent(source.ID()).Counters[P1P1] != 3 {
		t.Fatal("saved amount or clone isolation changed")
	}
}

func TestTypedCounterAmountReplacementAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Recipient", "", 1, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddCounterDoubler(source.ID(), P1P1, IsCreature)
	calls := 0
	effect := AddCountersTo(This, P1P1, MinValue(Fixed(2), countedLifeAmount{&calls}))
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("absent subject evaluated amount")
	}
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || g.FindPermanent(source.ID()).Counters[P1P1] != 4 {
		t.Fatal("amount capture or replacement failed")
	}
	for _, n := range []int{0, -2} {
		if err := ApplyEffect(g, AddCountersTo(This, P1P1, Fixed(n)), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.FindPermanent(source.ID()).Counters[P1P1] != 4 {
		t.Fatal("nonpositive amount changed counters")
	}
}

func TestTypedCounterAmountValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil amount",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AddCountersTo(This, P1P1, nil)))),
			WantErr: true,
		},
		{
			Name:    "power of nil",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AddCountersTo(This, P1P1, PowerOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AddCountersTo(This, P1P1, PowerOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AddCountersTo(This, P1P1, PowerOf(DeclarePermanentTarget("undeclared", IsCreature)))))),
			WantErr: true,
		},
		{
			Name:    "value at resolution nil",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), AddCountersTo(This, P1P1, ValueAtResolution(nil))))),
			WantErr: true,
		},
	})
	calls := 0
	info := InspectDefinition(AddCountersTo(This, P1P1, countedLifeAmount{&calls}))
	if calls != 0 || info.Properties.Outcome != OutcomeUnknown {
		t.Fatal("inspection evaluated dynamic amount")
	}
}

func TestTypedCounterAmountCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := NewCreature("Recipient", "", 2, 8)
	g.PutOnBattlefield(recipient, a.PlayerID())
	target := DeclarePermanentTarget("recipient", IsCreature)
	ability := Spell(AddCountersTo(target, P1P1, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Counters", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{recipient.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if got := g.FindPermanent(recipient.ID()).Counters[P1P1]; got != 2 {
		t.Fatalf("copy counters = %d, want 2", got)
	}
	g.ResolveStackObject(original)
	if got := g.FindPermanent(recipient.ID()).Counters[P1P1]; got != 6 {
		t.Fatalf("original counters = %d, want 6 from current power", got)
	}
}

// --- from composable_counter_each_test.go ---
func TestAddCountersEachReplacementAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
	g.PhaseOut(phased.ID())
	g.AddCounterDoubler(first.ID(), P1P1, IsCreature)
	clone := g.Clone()
	if err := AddCountersEach(ControlledPermanents(IsCreature), P1P1, Fixed(2)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID(), SourceID: first.ID()}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(first.ID()).Counters[P1P1] != 4 || clone.FindPermanent(second.ID()).Counters[P1P1] != 4 || phased.Counters[P1P1] != 0 {
		t.Fatal("batch replacement or phasing failed")
	}
	if first.Counters[P1P1] != 0 || second.Counters[P1P1] != 0 {
		t.Fatalf("clone changed original counters: %d, %d", first.Counters[P1P1], second.Counters[P1P1])
	}
}

func TestAddCountersEachDeclaration(t *testing.T) {
	good := NewSorcery("Good", "", WithAbility(Spell(AddCountersEach(ControlledPermanents(IsCreature), P1P1, Fixed(2)))))
	if len(InspectDefinition(good.Abilities()[0]).Children) == 0 {
		t.Fatal("batch missing from inspection")
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: good,
		},
		{
			Name:    "nil set",
			Card:    NewSorcery("Bad", "", WithAbility(Spell(AddCountersEach(nil, P1P1, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewSorcery("Bad", "", WithAbility(Spell(AddCountersEach(ControlledPermanents(IsCreature), P1P1, nil)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target group",
			Card:    NewSorcery("Bad", "", WithAbility(Spell(AddCountersEach(DeclarePermanentTargets("missing", 1, 2, IsCreature), P1P1, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "invalid counter type NumCounters",
			Card:    NewSorcery("Bad", "", WithAbility(Spell(AddCountersEach(ControlledPermanents(IsCreature), NumCounters, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestAddCountersEachPreparesBeforePlacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 3), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 3), a.PlayerID())
	source := g.PutOnBattlefield(NewArtifact("Doubler", ""), a.PlayerID())
	g.AddCounterDoubler(source.ID(), P1P1, NewPermanentFilter("second while first has no counters", func(p *Permanent, game *Game) bool {
		return p.ID() == second.ID() && game.FindPermanent(first.ID()).Counters[P1P1] == 0
	}))
	if err := AddCountersEach(fixedTapSet{[]*Permanent{first, second}}, P1P1, Fixed(1)).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if first.Counters[P1P1] != 1 || second.Counters[P1P1] != 2 {
		t.Fatalf("placements observed partial state: %d, %d", first.Counters[P1P1], second.Counters[P1P1])
	}
}

func TestAddCountersEachStaleDuplicateAndTargetGroup(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 2)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 2), b.PlayerID())
	if err := AddCountersEach(fixedTapSet{[]*Permanent{old, current, current, other}}, P1P1, Fixed(1)).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if current.Counters[P1P1] != 1 || other.Counters[P1P1] != 1 {
		t.Fatal("stale or duplicate member changed result")
	}
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	if err := AddCountersEach(targets, P1P1, Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if current.Counters[P1P1] != 1 {
		t.Fatal("returned target inherited old announcement")
	}
}

func TestAddCountersEachAmountReadOnce(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 3), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 3), a.PlayerID())
	calls := 0
	effect := AddCountersEach(fixedTapSet{[]*Permanent{first, second}}, P1P1, MinValue(Fixed(2), countedLifeAmount{&calls}))
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || first.Counters[P1P1] != 2 || second.Counters[P1P1] != 2 {
		t.Fatal("batch amount was not shared")
	}
	if err := AddCountersEach(fixedTapSet{}, P1P1, countedLifeAmount{&calls}).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("empty set evaluated amount")
	}
}

// --- from composable_counter_metadata_test.go ---
func TestComposableCounterMetadata(t *testing.T) {
	for _, tc := range []struct {
		name    string
		counter CounterType
		amount  ValueSource
		outcome Outcome
	}{
		{"positive", P1P1, Fixed(2), OutcomeBenefit},
		{"negative", M1M1, Fixed(2), OutcomeDetriment},
		{"zero", P1P1, Fixed(0), OutcomeUnknown},
		{"less than zero", P1P1, Fixed(-1), OutcomeUnknown},
		{"dynamic", P1P1, XValue(), OutcomeUnknown},
		{"other", Loyalty, Fixed(2), OutcomeUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := DeclarePermanentTarget("creature", IsCreature)
			for _, effect := range []ResolvingEffect{AddCountersTo(target, tc.counter, counterTestAmount(tc.amount)), AddCounters(tc.counter, tc.amount)} {
				if got := effect.Properties().Outcome; got != tc.outcome {
					t.Fatalf("outcome = %v, want %v", got, tc.outcome)
				}
			}
		})
	}
}

func counterTestAmount(value ValueSource) ResolutionValue {
	if fixed, ok := value.(fixedValue); ok {
		return fixed
	}
	return ValueAtResolution(value)
}

// --- from composable_counter_removal_test.go ---
func TestTypedCounterRemovalAmounts(t *testing.T) {
	for _, amount := range []int{-2, 0, 2, 5, 6, 300} {
		t.Run(Fixed(amount).Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", 2, 3)
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 5)
			g.MutablePermanent(source.ID()).AddCounter(Tide, 2)
			body := Sequence(RemoveCountersFrom(This, Charge, Fixed(amount)), GainLifeFor(You, Fixed(1)))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			p := g.FindPermanent(source.ID())
			if got, want := int(p.Counters[Charge]), 5-min(max(amount, 0), 5); got != want {
				t.Fatalf("counters = %d, want %d", got, want)
			}
			if p.Counters[Tide] != 2 || a.Life() != 21 {
				t.Fatal("other counters or following instruction changed")
			}
		})
	}
}

func TestTypedCounterRemovalContextAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 2, 3)
	outer := NewCreature("Outer target", "{3}", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(outer, b.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 5)
	clone := g.Clone()
	saved := BindPermanent("subject")
	body := Sequence(Snapshot(This, saved), RemoveCountersFrom(From(saved), Charge, ValueAtResolution(TargetPermanentManaValue())))
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), []uuid.UUID{outer.ID()}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(source.ID()).Counters[Charge] != 2 || g.FindPermanent(source.ID()).Counters[Charge] != 5 {
		t.Fatal("outer amount context or clone isolation changed")
	}
}

func TestTypedCounterRemovalIdentity(t *testing.T) {
	for _, scenario := range []string{"present", "absent", "returned", "phased", "phase cycle"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 2, 3)
			g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(c.ID()).AddCounter(Charge, 5)
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "absent", "returned":
				g.ExilePermanent(g.FindPermanent(c.ID()))
				if scenario == "returned" {
					g.RemoveFromExile(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(Charge, 5)
				}
			case "phased":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			}
			calls := 0
			if err := RemoveCountersFrom(From(saved), Charge, countedLifeAmount{&calls}).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			wantCalls := 0
			if scenario == "present" || scenario == "phase cycle" {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("amount evaluations = %d, want %d", calls, wantCalls)
			}
			if p := g.FindPermanentIncludingPhased(c.ID()); p != nil && wantCalls == 0 && p.Counters[Charge] != 5 {
				t.Fatal("changed unavailable subject")
			}
		})
	}
}

func TestTypedCounterRemovalValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), RemoveCountersFrom(nil, Charge, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), RemoveCountersFrom(From(BindPermanent("missing")), Charge, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), RemoveCountersFrom(DeclarePermanentTarget("missing", IsCreature), Charge, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), RemoveCountersFrom(This, Charge, nil)))),
			WantErr: true,
		},
		{
			Name:    "missing producer in amount",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), RemoveCountersFrom(This, Charge, PowerOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
	})
	calls := 0
	info := InspectDefinition(RemoveCountersFrom(This, Charge, countedLifeAmount{&calls}))
	if calls != 0 || !reflect.DeepEqual(info.Properties, EffectProperties{}) {
		t.Fatal("inspection evaluated amount or changed legacy metadata")
	}
}

func TestTypedCounterRemovalCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := NewCreature("Recipient", "", 2, 8)
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.MutablePermanent(recipient.ID()).AddCounter(Charge, 5)
	target := DeclarePermanentTarget("recipient", IsCreature)
	ability := Spell(RemoveCountersFrom(target, Charge, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Counters", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{recipient.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if got := g.FindPermanent(recipient.ID()).Counters[Charge]; got != 3 {
		t.Fatalf("copy counters = %d, want 3", got)
	}
	g.ExilePermanent(g.FindPermanent(recipient.ID()))
	g.RemoveFromExile(recipient.ID())
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.MutablePermanent(recipient.ID()).AddCounter(Charge, 5)
	g.ResolveStackObject(original)
	if got := g.FindPermanent(recipient.ID()).Counters[Charge]; got != 5 {
		t.Fatalf("returned target counters = %d, want 5", got)
	}
}

// --- from composable_counter_reset_test.go ---
func TestCounterStateCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 2, 2)
	g.PutOnBattlefield(c, a.PlayerID())
	condition := SourceHasCounters(Tide, 4)
	ctx := StateContext{SourceID: c.ID(), ControllerID: a.PlayerID()}
	for _, n := range []int{0, 1, 3, 4, 5} {
		g.MutablePermanent(c.ID()).Counters[Tide] = uint8(n)
		for range 2 {
			if got := condition.checkState(g, ctx); got != (n >= 4) {
				t.Fatalf("count %d: condition = %v", n, got)
			}
		}
		if got := condition.checkState(g.Clone(), ctx); got != (n >= 4) {
			t.Fatal("clone condition differs")
		}
	}
	g.PhaseOut(c.ID())
	if condition.checkState(g, ctx) {
		t.Fatal("phased source matched")
	}
	g.PhaseIn([]uuid.UUID{c.ID()})
	g.ExilePermanent(g.FindPermanent(c.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
}

func TestRemoveAllTypedCounters(t *testing.T) {
	for _, scenario := range []string{"present", "absent", "returned", "phased", "phase cycle"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 2, 2)
			g.PutOnBattlefield(c, a.PlayerID())
			p := g.MutablePermanent(c.ID())
			p.AddCounter(Tide, 5)
			p.AddCounter(Charge, 2)
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "absent", "returned":
				g.ExilePermanent(g.FindPermanent(c.ID()))
				if scenario == "returned" {
					g.RemoveFromExile(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(Tide, 7)
				}
			case "phased":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			}
			clone := g.Clone()
			child := *ctx
			child.Game = clone
			if err := RemoveAllCounters(From(saved), Tide).Apply(&child); err != nil {
				t.Fatal(err)
			}
			if p := clone.FindPermanentIncludingPhased(c.ID()); p != nil {
				want := 0
				if scenario == "returned" {
					want = 7
				}
				if scenario == "phased" {
					want = 5
				}
				if int(p.Counters[Tide]) != want {
					t.Fatalf("tide = %d, want %d", p.Counters[Tide], want)
				}
				if scenario != "returned" && p.Counters[Charge] != 2 {
					t.Fatal("removed another counter kind")
				}
			}
			if p := g.FindPermanentIncludingPhased(c.ID()); p != nil && p.Counters[Tide] == 0 {
				t.Fatal("clone mutated original")
			}
		})
	}
}

func TestCounterResetValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, RemoveAllCounters(nil, Tide)))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, RemoveAllCounters(From(BindPermanent("missing")), Tide)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, RemoveAllCounters(DeclarePermanentTarget("missing", IsCreature), Tide)))),
			WantErr: true,
		},
		{
			Name:    "negative threshold",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(SourceHasCounters(Tide, -1), RemoveAllCounters(This, Tide)))),
			WantErr: true,
		},
	})
}

// --- from composable_counter_value_test.go ---
func TestCounterValueResolutionAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(c.ID()).AddCounter(Charge, 3)
	g.MutablePermanent(c.ID()).AddCounter(Tide, 7)
	saved := BindPermanent("saved")
	body := Sequence(Snapshot(This, saved), AddCountersTo(This, Charge, Fixed(2)), GainLifeFor(You, CountersOn(From(saved), Charge)), RemoveCountersFrom(This, Charge, CountersOn(This, Charge)))
	clone := g.Clone()
	if err := ApplyEffect(clone, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 25 || a.Life() != 20 {
		t.Fatal("counter value did not read resolution state or changed original")
	}
	if clone.FindPermanent(c.ID()).Counters[Charge] != 0 || g.FindPermanent(c.ID()).Counters[Charge] != 3 || clone.FindPermanent(c.ID()).Counters[Tide] != 7 {
		t.Fatal("counter kind or clone isolation changed")
	}
	ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
	if got := CountersOn(This, Spore).resolveValue(ctx); got != 0 {
		t.Fatalf("missing kind = %d", got)
	}
}

func TestCounterValueIdentity(t *testing.T) {
	for _, scenario := range []string{"present", "absent", "returned", "phased", "phase cycle", "control"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 2, 3)
			g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(c.ID()).AddCounter(Charge, 4)
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 4
			switch scenario {
			case "absent", "returned":
				g.ExilePermanent(g.FindPermanent(c.ID()))
				want = 0
				if scenario == "returned" {
					g.RemoveFromExile(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
					g.MutablePermanent(c.ID()).AddCounter(Charge, 9)
				}
			case "phased":
				g.PhaseOut(c.ID())
				want = 0
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			}
			if got := CountersOn(From(saved), Charge).resolveValue(ctx); got != want {
				t.Fatalf("count = %d, want %d", got, want)
			}
		})
	}
}

func TestCounterValueCopyAndTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(c.ID()).AddCounter(Charge, 3)
	target := DeclarePermanentTarget("subject", IsCreature)
	ability := Spell(GainLifeFor(You, CountersOn(target, Charge)), WithTargets(target))
	card := NewInstant("Read counters", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.MutablePermanent(c.ID()).AddCounter(Charge, 2)
	g.ResolveStackObject(copied)
	if b.Life() != 25 || a.Life() != 20 {
		t.Fatal("copy used wrong amount or controller")
	}
	g.ExilePermanent(g.FindPermanent(c.ID()))
	g.RemoveFromExile(c.ID())
	g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(c.ID()).AddCounter(Charge, 9)
	g.ResolveStackObject(original)
	if a.Life() != 20 {
		t.Fatal("read returned target")
	}
}

func TestCounterValueValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, CountersOn(nil, Charge))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, CountersOn(From(BindPermanent("missing")), Charge))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, CountersOn(DeclarePermanentTarget("undeclared", IsCreature), Charge))))),
			WantErr: true,
		},
		{
			Name: "invalid counter negative one",
			Factory: func() Card {
				return NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, CountersOn(This, CounterType(-1))))))
			},
			WantErr: true,
		},
		{
			Name: "invalid counter NumCounters",
			Factory: func() Card {
				return NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, CountersOn(This, NumCounters)))))
			},
			WantErr: true,
		},
	})
	info := InspectDefinition(CountersOn(This, Charge))
	if info.Role != "counter-value" || info.Text != "Charge counters on subject" || len(info.Children) != 1 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestCounterValueContinuousTiming(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		name := "temporary"
		if dynamic {
			name = "static"
		}
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			modification := ModifyPT(This, CountersOn(This, Charge), Fixed(0))
			var opts []CardOption
			if dynamic {
				opts = append(opts, WithAbility(Static(While(Controls(IsCreature), modification))))
			}
			c := NewCreature("Subject", "", 1, 1, opts...)
			g.PutOnBattlefield(c, a.PlayerID())
			g.MutablePermanent(c.ID()).AddCounter(Charge, 2)
			if !dynamic {
				if err := ApplyEffect(g, Until(EndOfTurn, modification), c.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ApplyContinuousEffects()
			if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 3 {
				t.Fatalf("initial power = %d, want 3", got)
			}
			g.MutablePermanent(c.ID()).AddCounter(Charge, 4)
			for range 3 {
				g.ApplyContinuousEffects()
			}
			want := 3
			if dynamic {
				want = 7
			}
			if got := g.FindPermanent(c.ID()).CurrentPower(g); got != want {
				t.Fatalf("later power = %d, want %d", got, want)
			}
		})
	}
}

// --- from composable_creature_entry_binding_test.go ---
func TestCreatureEntryPermanentBindingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		controllerB bool
		depart      bool
		phase       bool
		want        int
	}{
		{"controlled", false, false, false, 1},
		{"opponent controlled", true, false, false, 0},
		{"returned", false, true, false, 0},
		{"phased", false, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			entered := BindPermanent("entered")
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().WithPermanent(entered),
				AddCountersTo(From(entered), P1P1, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			creature := NewCreature("Creature", "", 2, 2)
			creature.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.controllerB {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(creature, controller)
			g.PutTriggersOnStack()
			if tc.controllerB {
				if g.StackSize() != 0 {
					t.Fatal("opponent entry triggered")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("controlled entry did not trigger")
			}
			if tc.depart {
				g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
				g.PutOnBattlefield(creature, a.PlayerID())
			}
			if tc.phase {
				g.PhaseOut(creature.ID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if !tc.phase {
				if got := g.FindPermanent(creature.ID()).Counters[P1P1]; got != 0 {
					t.Fatalf("clone changed original: %d counters", got)
				}
			}
			g.ResolveStack()
			if tc.phase {
				g.PhaseIn([]uuid.UUID{creature.ID()})
			}
			if got := g.FindPermanent(creature.ID()).Counters[P1P1]; int(got) != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestCreatureEntryPermanentBindingDeclaration(t *testing.T) {
	entered := BindPermanent("entered")
	good := NewCreature("Good", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().WithPermanent(entered), AddCountersTo(From(entered), P1P1, Fixed(1)))))
	meta := InspectDefinition(good.Abilities()[0])
	if meta.Role != "event-trigger" || len(meta.Children) == 0 || len(meta.Children[0].Children) == 0 {
		t.Fatalf("entry binding missing from inspection: %+v", meta)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: good,
		},
		{
			Name:    "nil permanent binding",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureYouControlEnters().WithPermanent(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty permanent binding name",
			Card:    NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureYouControlEnters().WithPermanent(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound event read",
			Card:    NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(AnotherCreatureYouControlEnters(), AddCountersTo(From(entered), P1P1, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestCreatureEntryPermanentBindingStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	entered := BindPermanent("entered")
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().WithPermanent(entered),
		AddCountersTo(From(entered), P1P1, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	creature := NewCreature("Creature", "", 2, 2)
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("entry did not trigger")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.ResolveStack()
	if got := g.FindPermanent(creature.ID()).Counters[P1P1]; got != 2 {
		t.Fatalf("copies used different entered objects: %d counters", got)
	}
}

// --- from composable_damage_targets_test.go ---
func TestDamageTargetGroupMixedBatchAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Mixed damage", "", WithAbility(Spell(
		DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 3, 4), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 4), b.PlayerID())
	a.AddToHand(spell)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), creature.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.RemoveFromBattlefield(clone.FindPermanent(creature.ID()))
	clone.PutOnBattlefield(creature.Card, b.PlayerID())
	clone.ResolveStackObject(copyObj)
	if clone.GetPlayer(b.PlayerID()).Life() != 18 || clone.FindPermanent(creature.ID()).Damage != 0 || clone.FindPermanent(other.ID()).Damage != 0 {
		t.Fatal("copy followed a returned target or damaged an unselected one")
	}
	if b.Life() != 20 || creature.Damage != 0 || len(original.Targets) != 2 {
		t.Fatal("clone changed the original stack or battlefield")
	}
	g.ResolveStackObject(original)
	if b.Life() != 18 || g.FindPermanent(creature.ID()).Damage != 2 || g.FindPermanent(other.ID()).Damage != 0 {
		t.Fatalf("mixed batch: life=%d creature=%d other=%d", b.Life(), g.FindPermanent(creature.ID()).Damage, g.FindPermanent(other.ID()).Damage)
	}
}

func TestDamageTargetGroupAnnouncementAndValidation(t *testing.T) {
	targets := DeclareDamageTargets("recipients", 1, 2)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "count negative min",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(DealDamageEach(This, DeclareDamageTargets("invalid", -1, 2), Fixed(1)), WithTargets(DeclareDamageTargets("invalid", -1, 2))))),
			WantErr: true,
		},
		{
			Name:    "count min greater than max",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(DealDamageEach(This, DeclareDamageTargets("invalid", 2, 1), Fixed(1)), WithTargets(DeclareDamageTargets("invalid", 2, 1))))),
			WantErr: true,
		},
		{
			Name:    "count zero min zero max",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(DealDamageEach(This, DeclareDamageTargets("invalid", 0, 0), Fixed(1)), WithTargets(DeclareDamageTargets("invalid", 0, 0))))),
			WantErr: true,
		},
		{
			Name:          "undeclared group",
			Card:          NewInstant("Missing", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "target binding requires",
		},
	})
	g, a, b := newSpellCopyGame()
	card := NewInstant("Validated", "{1}", WithAbility(Spell(DealDamageEach(This, targets, Fixed(1)), WithTargets(targets))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(0, 1)
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
	for _, chosen := range [][]uuid.UUID{nil, {b.PlayerID(), b.PlayerID()}, {artifact.ID()}, {uuid.New()}, {creature.ID(), artifact.ID()}} {
		before := a.ManaPool().TotalMana()
		if err := g.CastSpellByName(a.PlayerID(), card.Name(), chosen); err == nil {
			t.Fatalf("accepted invalid targets %v", chosen)
		}
		if a.ManaPool().TotalMana() != before || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("failed announcement changed payment or stack")
		}
	}
}

func TestDamageTargetGroupPreventionAndPhasing(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Batch", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 3, 4), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.AddPreventionShield(b.PlayerID(), 1)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if b.Life() != 19 || g.FindPermanent(creature.ID()).Damage != 2 {
			t.Errorf("damage callback saw life=%d damage=%d", b.Life(), g.FindPermanent(creature.ID()).Damage)
		}
	})
	g.ResolveStack()
	if observed != 2 || b.Life() != 19 || g.FindPermanent(creature.ID()).Damage != 2 {
		t.Fatal("prevention changed the wrong recipient or broke the batch")
	}
	g.damage.SetOnDamageDealt(nil)

	second := NewInstant("Second", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	if err := g.CastSpellByName(a.PlayerID(), second.Name(), []uuid.UUID{b.PlayerID(), creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(creature.ID())
	g.ResolveStack()
	if b.Life() != 17 || g.FindPermanentIncludingPhased(creature.ID()).Damage != 2 {
		t.Fatal("phased recipient received damage or legal player was lost")
	}
}

func TestDamageTargetGroupPlaneswalker(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Batch", "", WithAbility(Spell(DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{a.PlayerID(), walker.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 || g.FindPermanent(walker.ID()).Counters[Loyalty] != 3 {
		t.Fatal("player or planeswalker received incorrect damage")
	}
}

// --- from composable_enchanted_target_test.go ---
func TestEnchantedTargetResolution(t *testing.T) {
	for _, change := range []string{"none", "opponent aura", "remove aura", "phase aura", "return host", "second aura"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := randomTestGame()
			host := addRandomTargetCreature(g, b, "Host")
			auraCard := NewAura("Aura", "{W}")
			auraCard.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "opponent aura" {
				controller = b.PlayerID()
			}
			aura := g.PutOnBattlefield(auraCard, controller)
			aura.AttachedTo = host.ID()
			host.Attachments = append(host.Attachments, aura.ID())
			target := DeclarePermanentTarget("creature", And(IsCreature, IsEnchanted))
			ability := Activate(Costs(TapCost(This)), Destroy(target), WithTargets(target))
			source := g.PutOnBattlefield(NewCreature("Assassin", "", 4, 3, WithAbility(ability)), a.PlayerID())
			g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{host.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			clone := g.Clone()
			switch change {
			case "remove aura":
				clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
			case "phase aura":
				clone.PhaseOut(aura.ID())
			case "return host":
				p := clone.FindPermanent(host.ID())
				clone.RemoveFromBattlefield(p)
				clone.PutOnBattlefield(p.Card, b.PlayerID())
			case "second aura":
				card := NewAura("Second Aura", "{W}")
				card.SetOwner(b.PlayerID())
				p := clone.PutOnBattlefield(card, b.PlayerID())
				p.AttachedTo = host.ID()
				h := clone.MutablePermanent(host.ID())
				h.Attachments = append(h.Attachments, p.ID())
				clone.PhaseOut(aura.ID())
			}
			clone.ResolveStack()
			wantDestroyed := change == "none" || change == "opponent aura" || change == "second aura"
			if gotDestroyed := clone.FindPermanent(host.ID()) == nil; gotDestroyed != wantDestroyed {
				t.Fatalf("destroyed = %v, want %v", gotDestroyed, wantDestroyed)
			}
			if g.FindPermanent(host.ID()) == nil || !IsEnchanted.Match(g.FindPermanent(host.ID()), g) {
				t.Fatal("clone changed original target or attachment")
			}
		})
	}
}

// --- from composable_exact_stats_target_test.go ---
func TestExactStatsTargetResolution(t *testing.T) {
	for _, change := range []string{"none", "power", "toughness", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := randomTestGame()
			card := NewCreature("Recipient", "", 1, 1)
			card.SetOwner(b.PlayerID())
			recipient := g.PutOnBattlefield(card, b.PlayerID())
			target := DeclarePermanentTarget("creature", And(IsCreature, HasPowerEqual(1), HasToughnessEqual(1)))
			ability := Activate(Costs(TapCost(This)), Until(EndOfTurn, ModifyPT(target, Fixed(1), Fixed(2))), WithTargets(target))
			source := g.PutOnBattlefield(NewLand("Booster", WithAbility(ability)), a.PlayerID())
			g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{recipient.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
			clone := g.Clone()
			power, toughness := 1, 1
			switch change {
			case "power", "toughness":
				dp, dt := 1, 0
				if change == "toughness" {
					dp, dt = 0, 1
				}
				effect := Until(EndOfTurn, ModifyPT(This, Fixed(dp), Fixed(dt)))
				if err := ApplyEffect(clone, effect, recipient.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				power, toughness = power+dp, toughness+dt
			case "return":
				p := clone.FindPermanent(recipient.ID())
				clone.RemoveFromBattlefield(p)
				clone.PutOnBattlefield(p.Card, b.PlayerID())
			default:
				power, toughness = 2, 3
			}
			clone.ResolveStack()
			p := clone.FindPermanent(recipient.ID())
			if p.CurrentPower(clone) != power || p.CurrentToughness(clone) != toughness {
				t.Fatalf("got %d/%d, want %d/%d", p.CurrentPower(clone), p.CurrentToughness(clone), power, toughness)
			}
			if recipient.CurrentPower(g) != 1 || recipient.CurrentToughness(g) != 1 {
				t.Fatal("clone changed original")
			}
		})
	}
}

// --- from composable_filtered_destruction_test.go ---
func TestDestroyEachMatchingBlackCreatures(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := DestroyEach(MatchingPermanents(And(IsCreature, HasColorFilter(Black))))
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	var destroyed, survivors []*Permanent
	for _, owner := range []*BasePlayer{a, b} {
		for _, scenario := range []string{"black", "multicolor", "white", "colorless", "artifact", "indestructible", "regenerate", "phased"} {
			card := NewCreature(scenario, "{B}", 2, 2)
			switch scenario {
			case "multicolor":
				card = NewCreature(scenario, "{W}{B}", 2, 2)
			case "white":
				card = NewCreature(scenario, "{W}", 2, 2)
			case "colorless":
				card = NewCreature(scenario, "{2}", 2, 2)
			case "artifact":
				card = NewArtifact(scenario, "{B}")
			}
			card.SetOwner(owner.PlayerID())
			p := g.PutOnBattlefield(card, owner.PlayerID())
			switch scenario {
			case "indestructible":
				p.GrantBaseAttr(Indestructible)
			case "regenerate":
				g.AddRegenerationShield(p.ID())
			case "phased":
				p.PhasedOut = true
			}
			if scenario == "black" || scenario == "multicolor" {
				destroyed = append(destroyed, p)
			} else {
				survivors = append(survivors, p)
			}
		}
	}
	clone := g.Clone()
	if err := effect.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	for _, p := range destroyed {
		if clone.FindPermanent(p.ID()) != nil || g.FindPermanent(p.ID()) == nil {
			t.Fatalf("wrong destruction or clone isolation for %s", p.Name())
		}
	}
	for _, p := range survivors {
		got := clone.FindPermanentIncludingPhased(p.ID())
		if got == nil {
			t.Fatalf("destroyed %s", p.Name())
		}
		if p.Name() == "regenerate" && (!got.Tapped || p.Tapped) {
			t.Fatal("regeneration did not tap only the cloned permanent")
		}
	}
}

// --- from composable_graveyard_mana_value_test.go ---
func TestGraveyardManaValueBoundTracksControllerHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cheap := NewCreature("Cheap", "{1}", 1, 1)
	cheap.SetOwner(a.PlayerID())
	expensive := NewCreature("Expensive", "{3}", 3, 3)
	expensive.SetOwner(a.PlayerID())
	other := NewCreature("Other", "{1}", 1, 1)
	other.SetOwner(b.PlayerID())
	a.AddToGraveyard(cheap)
	a.AddToGraveyard(expensive)
	b.AddToGraveyard(other)
	target := DeclareGraveyardCardTarget("card", GraveyardCardFilter(IsCreatureCard), GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn()))
	if got := target.Possible(a.PlayerID(), nil, g); len(got) != 0 {
		t.Fatalf("zero gain candidates = %v", got)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := target.Possible(a.PlayerID(), nil, g); !slices.Equal(got, []uuid.UUID{cheap.ID()}) {
		t.Fatalf("one gain candidates = %v", got)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, GainLifeFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := target.Possible(a.PlayerID(), nil, clone); !slices.Contains(got, expensive.ID()) {
		t.Fatalf("clone gain candidates = %v", got)
	}
	if got := target.Possible(a.PlayerID(), nil, g); slices.Contains(got, expensive.ID()) {
		t.Fatalf("clone changed original candidates = %v", got)
	}
	if got := target.Possible(b.PlayerID(), nil, g); len(got) != 0 {
		t.Fatalf("wrong controller candidates = %v", got)
	}
	fromAny := DeclareGraveyardCardTarget("any", InAnyGraveyard(), GraveyardCardManaValueAtMost(FixedTargetNumber(1)))
	if got := fromAny.Possible(a.PlayerID(), nil, g); len(got) != 2 || !slices.Contains(got, other.ID()) {
		t.Fatalf("any-graveyard fixed candidates = %v", got)
	}
}

func TestGraveyardManaValueBoundRechecksTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "{2}", 2, 2)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn()))
	ability := Spell(ReturnCardToHand(target), WithTargets(target))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Recover", spell); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(target.Possible(a.PlayerID(), spell, g), card.ID()) {
		t.Fatal("card was not legal at announcement")
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	if !g.isTargetStillLegal(card.ID(), spell, a.PlayerID(), g.zoneOfTarget(card.ID()), target) {
		t.Fatal("target became illegal without a state change")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if g.isTargetStillLegal(card.ID(), spell, a.PlayerID(), g.zoneOfTarget(card.ID()), target) {
		t.Fatal("target remained legal after the value changed")
	}
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 {
		t.Fatal("illegal target was returned")
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	obj = &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	// A new graveyard visit must not restore the old target.
	moved, _ := g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand)
	a.AddToHand(moved)
	returned, _ := a.RemoveFromHand(card.ID())
	a.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 {
		t.Fatal("old graveyard visit was returned")
	}
}

func TestGraveyardManaValueBoundTriggerAnnouncement(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	cheap := NewCreature("Cheap", "{1}", 1, 1)
	cheap.SetOwner(a.PlayerID())
	expensive := NewCreature("Expensive", "{3}", 3, 3)
	expensive.SetOwner(a.PlayerID())
	a.AddToGraveyard(cheap)
	a.AddToGraveyard(expensive)
	target := DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn()))
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This), ReturnCardToHand(target), WithTargets(target))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.chooseQueue = [][]uuid.UUID{{expensive.ID()}}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackPeek() != nil {
		t.Fatal("trigger accepted an illegal target")
	}
	a.chooseQueue = [][]uuid.UUID{{cheap.ID()}}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if obj := g.StackPeek(); obj == nil || !slices.Equal(obj.Targets, []uuid.UUID{cheap.ID()}) {
		t.Fatalf("trigger target = %v", obj)
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != cheap.ID() {
		t.Fatal("trigger did not use its legal card target")
	}
}

func TestGraveyardManaValueBoundRejectsInvalidDeclaration(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil bound",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToHand(DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(nil))), WithTargets(DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(nil)))))),
			WantErr:       true,
			WantErrSubstr: "mana value",
		},
		{
			Name:          "negative bound",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToHand(DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(FixedTargetNumber(-1)))), WithTargets(DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(FixedTargetNumber(-1))))))),
			WantErr:       true,
			WantErrSubstr: "mana value",
		},
		{
			Name:          "duplicate bound options",
			Card:          NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToHand(DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(FixedTargetNumber(1)), GraveyardCardManaValueAtMost(FixedTargetNumber(2)))), WithTargets(DeclareGraveyardCardTarget("card", GraveyardCardManaValueAtMost(FixedTargetNumber(1)), GraveyardCardManaValueAtMost(FixedTargetNumber(2))))))),
			WantErr:       true,
			WantErrSubstr: "mana value",
		},
	})
}

// --- from composable_hand_size_value_test.go ---
func TestHandSizeOfReadsCurrentSelectedPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Own card", ""))
	for range 3 {
		b.AddToHand(NewArtifact("Opposing card", ""))
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if got := HandSizeOf(You).resolveValue(ctx); got != 1 {
		t.Fatalf("controller hand size = %d, want 1", got)
	}
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 1 {
		t.Fatalf("active player hand size = %d, want 1", got)
	}
	g.SetActivePlayerIndex(1)
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 3 {
		t.Fatalf("opponent hand size = %d, want 3", got)
	}
	b.AddToHand(NewArtifact("Late card", ""))
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 4 {
		t.Fatalf("late hand size = %d, want 4", got)
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).RemoveFromHand(b.Hand()[0].ID())
	if got := HandSizeOf(ActivePlayer).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 3 {
		t.Fatalf("clone hand size = %d, want 3", got)
	}
	if got := HandSizeOf(ActivePlayer).resolveValue(ctx); got != 4 {
		t.Fatalf("original hand size = %d, want 4", got)
	}
	if got := HandSizeOf(ControllerOf(This)).resolveValue(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("absent player hand size = %d, want 0", got)
	}
}

func TestHandSizeOfTargetAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Discard", ""))
	for range 3 {
		b.AddToHand(NewArtifact("Opposing card", ""))
		a.AddToLibrary(NewArtifact("Draw", ""))
	}
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	ability := Spell(Sequence(DiscardEntireHand(You), DrawFor(You, HandSizeOf(target))), WithTargets(target))
	card := NewSorcery("Synthetic hand spell", "", WithAbility(ability))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: card,
		},
		{
			Name:    "nil player ref",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DrawFor(You, HandSizeOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DrawFor(You, HandSizeOf(DeclarePlayerTarget("undeclared")))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(DrawFor(You, HandSizeOf(FromPlayer(BindPlayer("missing"))))))),
			WantErr: true,
		},
	})
	info := InspectDefinition(HandSizeOf(target))
	if info.Role != "resolution-value" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	obj := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).RemoveFromHand(b.Hand()[0].ID())
	clone.ResolveStack()
	if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != 2 {
		t.Fatalf("clone drew %d cards, want 2", got)
	}
	g.ResolveStack()
	if got := len(a.Hand()); got != 3 {
		t.Fatalf("original drew %d cards, want 3", got)
	}
}

// --- from composable_highest_life_player_test.go ---
func TestPlayerWithUniqueHighestLifeResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ref := PlayerWithUniqueHighestLife()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if _, ok := ref.resolvePlayer(ctx); ok {
		t.Fatal("tied life totals selected a player")
	}
	if err := ApplyEffect(g, Sequence(LoseLifeFor(You, Fixed(1)), GainLifeFor(ref, Fixed(2))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 22 {
		t.Fatalf("resolution-time selection changed life totals to %d, %d", a.Life(), b.Life())
	}
	if err := ApplyEffect(g, LoseLifeFor(ref, Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("highest-life selection did not use the current game state")
	}
	if err := ApplyEffect(g, GainLifeFor(ref, Fixed(4)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 19 {
		t.Fatal("tied selection was not absent")
	}
}

func TestPlayerWithUniqueHighestLifeCloneAndDefinition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ref := PlayerWithUniqueHighestLife()
	if info := InspectDefinition(ref); info.Role != "player-ref" || info.Text == "" {
		t.Fatalf("selector inspection = %+v", info)
	}
	card := NewArtifact("Watcher", "", WithAbility(Activate(Costs(), GainLifeFor(ref, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).GainLife(2)
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone did not select its own highest-life player")
	}
	a.GainLife(2)
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("pending activation did not select at resolution")
	}
}

// --- from composable_land_tap_binding_test.go ---
func TestOpponentLandTapBindingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		returnLand bool
		phaseLand  bool
		want       int
	}{
		{"present", false, false, 1},
		{"returned", true, false, 0},
		{"phased", false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			landBinding := BindPermanent("tapped land")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().OfSubtype("Forest").WithLand(landBinding),
				AddCountersTo(From(landBinding), Charge, Fixed(1)))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			land := NewLand("Forest", WithSubTypes("Forest"))
			g.PutOnBattlefield(land, b.PlayerID())
			g.TapPermanent(g.FindPermanent(land.ID()))
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatalf("stack=%d, want 1", g.StackSize())
			}
			if tc.returnLand {
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				g.PutOnBattlefield(land, b.PlayerID())
			}
			if tc.phaseLand {
				g.PhaseOut(land.ID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if got := g.FindPermanentIncludingPhased(land.ID()).Counters[Charge]; got != 0 {
				t.Fatalf("clone changed original: %d counters", got)
			}
			g.ResolveStack()
			if got := int(g.FindPermanentIncludingPhased(land.ID()).Counters[Charge]); got != tc.want {
				t.Fatalf("counters=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestOpponentLandTapBindingDeclaration(t *testing.T) {
	land := BindPermanent("tapped land")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewEnchantment("Valid", "", WithAbility(WhenEvent(
				OpponentLandBecomesTapped().WithLand(land), AddCountersTo(From(land), Charge, Fixed(1))))),
		},
		{
			Name:    "nil land binding",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(OpponentLandBecomesTapped().WithLand(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty land binding name",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(OpponentLandBecomesTapped().WithLand(BindPermanent("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate land binding",
			Card:    NewEnchantment("Bad", "", WithAbility(WhenEvent(OpponentLandBecomesTapped().WithLand(land).WithLand(land), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "unbound land read",
			Card:    NewEnchantment("Unbound", "", WithAbility(WhenEvent(OpponentLandBecomesTapped(), AddCountersTo(From(land), Charge, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestOpponentLandTapBindingPendingCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	landBinding := BindPermanent("tapped land")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentLandBecomesTapped().WithLand(landBinding),
		AddCountersTo(From(landBinding), Charge, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Forest", WithSubTypes("Forest"))
	g.PutOnBattlefield(land, b.PlayerID())
	g.TapPermanent(g.FindPermanent(land.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("pending stack=%d, want 1", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if got := g.FindPermanent(land.ID()).Counters[Charge]; got != 2 {
		t.Fatalf("counters=%d, want 2", got)
	}
}

// --- from composable_library_choice_test.go ---
func TestComposableLibraryChoiceValidation(t *testing.T) {
	for _, build := range []struct {
		name  string
		build func(ResolutionValue) ResolvingEffect
	}{
		{"ScryAtResolution", ScryAtResolution},
		{"SurveilAtResolution", SurveilAtResolution},
	} {
		t.Run(build.name, func(t *testing.T) {
			target := DeclarePermanentTarget("creature", IsCreature)
			saved := BindPermanent("saved")
			RunDefinitionValidationTable(t, []DefinitionValidationCase{
				{
					Name:          "nil amount",
					Card:          NewSorcery("Invalid choice", "", WithAbility(Spell(build.build(nil)))),
					WantErr:       true,
					WantErrSubstr: ".amount",
				},
				{
					Name:          "power of undeclared target",
					Card:          NewSorcery("Invalid choice", "", WithAbility(Spell(build.build(PowerOf(target))))),
					WantErr:       true,
					WantErrSubstr: ".amount.subject",
				},
				{
					Name:          "power of missing producer",
					Card:          NewSorcery("Invalid choice", "", WithAbility(Spell(build.build(PowerOf(From(saved)))))),
					WantErr:       true,
					WantErrSubstr: ".amount.subject",
				},
				{
					Name:          "nil nested right",
					Card:          NewSorcery("Invalid choice", "", WithAbility(Spell(build.build(MaxValue(Fixed(1), nil))))),
					WantErr:       true,
					WantErrSubstr: ".amount.right",
				},
				{
					Name: "valid choice with saved target",
					Card: NewSorcery("Choice", "", WithAbility(Spell(Sequence(Snapshot(target, saved), build.build(PowerOf(From(saved)))), WithTargets(target)))),
				},
			})
			info := InspectDefinition(build.build(MaxValue(Fixed(1), PowerOf(target))))
			if len(info.Children) != 1 || info.Children[0].Role != "arithmetic-value" || info.Properties.Outcome != OutcomeBenefit {
				t.Fatalf("metadata = %+v", info)
			}
		})
	}
}

type libraryChoiceValue struct{ calls int }

func (*libraryChoiceValue) Text() string { return "dynamic amount" }
func (v *libraryChoiceValue) Resolve(GameReader, uuid.UUID, uuid.UUID, []uuid.UUID) int {
	v.calls++
	return 2
}

func TestLibraryChoiceInspectionDoesNotResolve(t *testing.T) {
	for _, build := range []struct {
		name  string
		build func(ValueSource) Effect
	}{
		{"Scry", Scry},
		{"Surveil", Surveil},
	} {
		t.Run(build.name, func(t *testing.T) {
			value := &libraryChoiceValue{}
			effect := build.build(value)
			info := InspectDefinition(effect)
			if value.calls != 0 || len(info.Children) != 1 || !strings.Contains(info.Text, "dynamic amount") {
				t.Fatalf("calls = %d, metadata = %+v", value.calls, info)
			}
			RunDefinitionValidationTable(t, []DefinitionValidationCase{
				{
					Name:          "nil amount",
					Card:          NewSorcery("Nil choice", "", WithAbility(Spell(build.build(nil)))),
					WantErr:       true,
					WantErrSubstr: ".amount",
				},
				{
					Name:          "typed nil amount",
					Card:          NewSorcery("Nil choice", "", WithAbility(Spell(build.build((*libraryChoiceValue)(nil))))),
					WantErr:       true,
					WantErrSubstr: ".amount",
				},
			})
		})
	}
}

func TestComposableLibraryChoiceCloneAndAbsentBinding(t *testing.T) {
	for _, build := range []func(ResolutionValue) ResolvingEffect{ScryAtResolution, SurveilAtResolution} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		top := NewLand("Top")
		a.SetLibrary([]Card{top})
		clone := g.Clone()
		saved := BindPermanent("absent")
		if err := ApplyEffect(clone, Sequence(Snapshot(This, saved), build(PowerOf(From(saved)))), uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(clone.GetPlayer(a.PlayerID()).Library()) != 1 {
			t.Fatal("absent binding changed library")
		}
		if err := ApplyEffect(clone, Sequence(build(Fixed(4)), DrawFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 0 || len(a.Library()) != 1 {
			t.Fatal("clone or sequence lost isolation")
		}
		if err := ApplyEffect(clone, build(Fixed(4)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from composable_life_gain_history_value_test.go ---
func TestLifeGainedThisTurnValueActualOutcomeAndPlayer(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewInstant("Gain", "")
	value := LifeGainedThisTurn(You)
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("initial gain = %d", got)
	}
	g.effects.AddReplacement(&lifeOutcomeReplacement{consume: true})
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(3)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("consumed gain = %d", got)
	}
	g.effects.replacements = nil
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(0)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 5})
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, LoseLifeFor(You, Fixed(4)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.replacements = nil
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := value.resolveValue(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); got != 7 {
		t.Fatalf("controller gain after source departure = %d, want 7", got)
	}
	if got := value.resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}); got != 0 {
		t.Fatalf("opponent gain = %d", got)
	}
	if got := value.resolveValue(&EffectContext{Game: g}); got != 0 {
		t.Fatalf("absent player gain = %d", got)
	}
	clone := g.Clone()
	clone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if got := value.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("clone gain after reset = %d", got)
	}
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("original gain after clone reset = %d", got)
	}
}

func TestLifeGainedThisTurnValueResolutionAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Synthetic life history", "", WithAbility(Spell(GainLifeFor(You, LifeGainedThisTurn(You)))))
	target := DeclarePlayerTarget("recipient")
	targetAbility := Spell(GainLifeFor(You, LifeGainedThisTurn(target)), WithTargets(target))
	targeted := NewInstant("Target history", "", WithAbility(targetAbility))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: card,
		},
		{
			Name: "valid targeted",
			Card: targeted,
		},
		{
			Name:    "nil player ref",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeGainedThisTurn(nil))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeGainedThisTurn(DeclarePlayerTarget("undeclared")))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeGainedThisTurn(FromPlayer(BindPlayer("missing"))))))),
			WantErr: true,
		},
	})
	info := InspectDefinition(LifeGainedThisTurn(You))
	if info.Role != "resolution-value" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	source := NewCreature("History source", "", 1, 1, WithAbility(At(EachEndStep, GainLifeFor(You, LifeGainedThisTurn(You)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing triggered ability")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ExilePermanent(g.FindPermanent(source.ID()))
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(3)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, GainLifeFor(You, Fixed(2)), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := LifeGainedThisTurn(target).resolveValue(&EffectContext{
		Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
			target.targetDeclaration: {{id: b.PlayerID()}},
		},
	}); got != 2 {
		t.Fatalf("target player's gain = %d, want 2", got)
	}
	if got := LifeGainedThisTurn(target).resolveValue(&EffectContext{Game: g, Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("absent target gain = %d", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 26 || clone.GetPlayer(b.PlayerID()).Life() != 24 {
		t.Fatalf("clone life = %d, %d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
	if a.Life() != 23 || b.Life() != 22 {
		t.Fatal("clone changed original game")
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 24 {
		t.Fatalf("original life = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_life_total_value_test.go ---
func TestLifeTotalOfReadsCurrentPlayerAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	a.SetLife(7)
	b.SetLife(13)
	if got := LifeTotalOf(You).resolveValue(ctx); got != 7 {
		t.Fatalf("controller life = %d, want 7", got)
	}
	g.SetActivePlayerIndex(1)
	if got := LifeTotalOf(ActivePlayer).resolveValue(ctx); got != 13 {
		t.Fatalf("active player life = %d, want 13", got)
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).SetLife(4)
	if got := LifeTotalOf(ActivePlayer).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 4 {
		t.Fatalf("clone life = %d, want 4", got)
	}
	if got := LifeTotalOf(ActivePlayer).resolveValue(ctx); got != 13 {
		t.Fatalf("original life = %d, want 13", got)
	}
	if got := LifeTotalOf(ControllerOf(This)).resolveValue(&EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("absent player life = %d, want 0", got)
	}
}

func TestLifeTotalOfSequenceTargetAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	ability := Spell(Sequence(LoseLifeFor(target, Fixed(3)), GainLifeFor(You, LifeTotalOf(target))), WithTargets(target))
	card := NewSorcery("Synthetic life total", "", WithAbility(ability))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: card,
		},
		{
			Name:    "nil player ref",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeTotalOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeTotalOf(DeclarePlayerTarget("undeclared")))))),
			WantErr: true,
		},
		{
			Name:    "missing producer",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(GainLifeFor(You, LifeTotalOf(FromPlayer(BindPlayer("missing"))))))),
			WantErr: true,
		},
	})
	info := InspectDefinition(LifeTotalOf(target))
	if info.Role != "resolution-value" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	obj := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveStack()
	if got := a.Life(); got != 37 {
		t.Fatalf("life after ordered actions = %d, want 37", got)
	}
	if got := b.Life(); got != 17 {
		t.Fatalf("target life = %d, want 17", got)
	}
}

// --- from composable_mana_value_set_test.go ---
func TestManaValueAtMostSetPaidXAndSimultaneousDestroy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid")
	spell := NewInstant("Sweep", "", WithAbility(Spell(DestroyEach(ManaValueAtMost(MatchingPermanents(Or(IsArtifact, IsCreature)), FromNumber(paid))))), WithAdditionalCost(LifePayXCostInto(paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	makeCard := func(name, cost string, types ...CardType) *Permanent {
		card := NewArtifact(name, cost)
		card.types = types
		card.power, card.toughness = 2, 2
		card.SetOwner(b.PlayerID())
		return g.PutOnBattlefield(card, b.PlayerID())
	}
	zero := makeCard("Zero", "", TypeArtifact)
	boundary := makeCard("Boundary", "{2}", TypeArtifact, TypeCreature)
	above := makeCard("Above", "{3}", TypeCreature)
	land := makeCard("Land", "", TypeLand)
	phased := makeCard("Phased", "{1}", TypeArtifact)
	phased.PhasedOut = true
	g.AddRegenerationShield(boundary.ID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 2); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if !clone.HasRegenerationShield(boundary.ID()) {
		t.Fatal("clone lost regeneration shield")
	}
	clone.ResolveStack()
	if clone.FindPermanent(zero.ID()) != nil || clone.FindPermanent(boundary.ID()) == nil || !clone.FindPermanent(boundary.ID()).Tapped {
		t.Fatalf("paid threshold or regeneration result is wrong: zero=%v boundary=%v", clone.FindPermanent(zero.ID()), clone.FindPermanent(boundary.ID()))
	}
	for _, p := range []*Permanent{above, land, phased} {
		if clone.FindPermanentIncludingPhased(p.ID()) == nil {
			t.Fatalf("excluded permanent %s was destroyed", p.Name())
		}
	}
	if g.FindPermanent(zero.ID()) == nil || g.FindPermanent(boundary.ID()).Tapped {
		t.Fatal("clone changed original game")
	}
	g.ResolveStack()
	if g.FindPermanent(zero.ID()) != nil || g.FindPermanent(boundary.ID()) == nil || !g.FindPermanent(boundary.ID()).Tapped {
		t.Fatal("original resolution differed from clone")
	}
}

func TestManaValueAtMostSetValidation(t *testing.T) {
	paid := BindNumber("paid")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "nil set",
			Card:          NewSorcery("nil set", "", WithAbility(Spell(DestroyEach(ManaValueAtMost(nil, Fixed(2)))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil limit",
			Card:          NewSorcery("nil limit", "", WithAbility(Spell(DestroyEach(ManaValueAtMost(MatchingPermanents(IsCreature), nil))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "unbound",
			Card:          NewSorcery("unbound", "", WithAbility(Spell(DestroyEach(ManaValueAtMost(MatchingPermanents(IsCreature), FromNumber(paid)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_mana_value_test.go ---
func TestManaValueOfPermanent(t *testing.T) {
	for _, tc := range []struct {
		cost string
		want int
	}{{"{3}{W}", 4}, {"{X}{2}", 2}, {"", 0}, {"{W/U}{B/R}", 2}} {
		t.Run(tc.cost, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Subject", tc.cost)
			g.PutOnBattlefield(card, a.PlayerID())
			saved, number := BindPermanent("subject"), BindNumber("mana value")
			ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
			if err := Sequence(Snapshot(This, saved), SnapshotNumber(ManaValueOf(From(saved)), number)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			for _, change := range []string{"live", "phase", "phase in", "return"} {
				switch change {
				case "phase":
					g.PhaseOut(card.ID())
				case "phase in":
					g.PhaseIn([]uuid.UUID{card.ID()})
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
				}
				want := tc.want
				if change == "phase" || change == "return" {
					want = 0
				}
				if got := ManaValueOf(From(saved)).resolveValue(ctx); got != want {
					t.Fatalf("%s: mana value = %d, want %d", change, got, want)
				}
				if got := FromNumber(number).resolveValue(ctx); got != tc.want {
					t.Fatalf("snapshot = %d, want %d", got, tc.want)
				}
			}
		})
	}
}

func TestManaValueOfValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil",
			Card:    NewInstant("nil", "", WithAbility(Spell(GainLifeFor(You, ManaValueOf(nil))))),
			WantErr: true,
		},
		{
			Name:    "missing",
			Card:    NewInstant("missing", "", WithAbility(Spell(GainLifeFor(You, ManaValueOf(From(BindPermanent("missing"))))))),
			WantErr: true,
		},
		{
			Name:    "undeclared",
			Card:    NewInstant("undeclared", "", WithAbility(Spell(GainLifeFor(You, ManaValueOf(DeclarePermanentTarget("undeclared", IsArtifact)))))),
			WantErr: true,
		},
	})
}

func TestManaValueOfTargetClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewArtifact("First", "{1}"), NewArtifact("Second", "{4}")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	x, y := DeclarePermanentTarget("first", IsArtifact), DeclarePermanentTarget("second", IsArtifact)
	n := BindNumber("value")
	body := Sequence(SnapshotNumber(ManaValueOf(y), n), Destroy(y), GainLifeFor(You, FromNumber(n)))
	ability := Spell(body, WithTargets(x, y))
	card := NewInstant("Read second", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.FindPermanent(second.ID()) != nil || clone.FindPermanent(first.ID()) == nil {
		t.Fatal("wrong target or captured amount")
	}
	if a.Life() != 20 || g.FindPermanent(second.ID()) == nil {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("original lost binding state")
	}
}

// --- from composable_mode_test.go ---
func TestTypedModalTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()
	creature := DeclarePermanentTarget("creature", IsCreature)
	player := DeclarePlayerTarget("player")
	ability := SpellModes(ModeOf("Destroy", Destroy(creature), WithTargets(creature)), ModeOf("Draw", DrawFor(player, Fixed(1)), WithTargets(player)))
	card := NewSorcery("Modes", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Modes", card); err != nil {
		t.Fatal(err)
	}
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	obj, err := g.pushCastSpellObject(castStackObjectOptions{Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID()}})
	if err != nil {
		t.Fatal(err)
	}
	if len(obj.TargetSpecs) != 1 {
		t.Fatal("modal stack object lost target declaration")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("modal target binding did not resolve")
	}
}

func TestTypedModalScope(t *testing.T) {
	first := DeclarePlayerTarget("first")
	second := DeclarePlayerTarget("second")
	ability := SpellModes(ModeOf("First", DrawFor(first, Fixed(1)), WithTargets(first)), ModeOf("Second", DrawFor(first, Fixed(1)), WithTargets(second)))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "target escaped its mode",
			Card:    NewSorcery("Scope", "", WithAbility(ability)),
			WantErr: true,
		},
	})
}

// --- from composable_number_snapshot_test.go ---
func TestNumberSnapshotDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("count")
	body := Sequence(SnapshotNumber(CountersOn(This, Charge), n),
		RemoveAllCounters(This, Charge),
		AtNext(EachEndStep, GainLifeFor(You, FromNumber(n))),
		SnapshotNumber(Fixed(99), n))
	c := NewCreature("Source", "", 2, 2, WithAbility(Activate(Costs(), body)))
	c.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(c, a.PlayerID())
	for _, count := range []int{3, 5} {
		g.MutablePermanent(c.ID()).AddCounter(Charge, count)
		if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatalf("stack = %d", game.StackSize())
		}
		for _, object := range game.StackObjects() {
			game.CopyStackObjectDirect(object, b.PlayerID(), false)
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 28 || game.GetPlayer(b.PlayerID()).Life() != 28 {
			t.Fatalf("life A=%d B=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
		if FromNumber(n).resolveValue(&EffectContext{Game: game}) != 0 {
			t.Fatal("snapshot escaped resolution")
		}
		if game == clone && a.Life() != 20 {
			t.Fatal("clone changed original")
		}
	}
}

func TestNumberSnapshotValidation(t *testing.T) {
	n := BindNumber("number")
	read := GainLifeFor(You, FromNumber(n))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(SnapshotNumber(Fixed(2), n), read)))),
		},
		{
			Name:          "read before write",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(read, SnapshotNumber(Fixed(2), n))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "self read",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), SnapshotNumber(FromNumber(n), n)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name: "overwrite",
			Card: NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(SnapshotNumber(Fixed(2), n), SnapshotNumber(SubtractValues(FromNumber(n), Fixed(1)), n), read)))),
		},
		{
			Name:          "branch escape",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(IfState(Controls(IsCreature), SnapshotNumber(Fixed(2), n)), read)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "loop escape",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), SnapshotNumber(Fixed(2), n)), read)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "delayed escape",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(AtNext(EachEndStep, SnapshotNumber(Fixed(2), n)), read)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), Sequence(SnapshotNumber(Fixed(2), n), SnapshotNumber(Fixed(3), BindNumber("number")))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "nil value",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), SnapshotNumber(nil, n)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil binding",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), SnapshotNumber(Fixed(2), nil)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty name",
			Card:          NewCreature("Snapshot", "", 1, 1, WithAbility(Activate(Costs(), SnapshotNumber(Fixed(2), BindNumber(""))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
	})
}

func TestNumberSnapshotOverwriteAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("number")
	ctx := &EffectContext{Game: g}
	for _, amount := range []int{7, -3, 0} {
		if err := SnapshotNumber(Fixed(amount), n).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if got := FromNumber(n).resolveValue(ctx); got != amount {
			t.Fatalf("saved = %d, want %d", got, amount)
		}
	}
	if err := SnapshotNumber(Fixed(4), n).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := SnapshotNumber(Fixed(8), n).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromNumber(n).resolveValue(ctx) != 4 {
		t.Fatal("clone shared binding")
	}
	body := Sequence(SnapshotNumber(SubtractValues(FromNumber(n), Fixed(1)), n), GainLifeFor(You, FromNumber(n)))
	if err := ApplyEffect(g, body, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || FromNumber(n).resolveValue(ctx) != 4 {
		t.Fatal("nested resolution failed to isolate and restore binding")
	}
}

func TestNumberSnapshotLoopIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 3, 3), a.PlayerID())
	n, member := BindNumber("amount"), BindPermanent("member")
	body := Sequence(SnapshotNumber(Fixed(4), n),
		ForEach(MatchingPermanents(IsCreature), member, Sequence(
			SnapshotNumber(PowerOf(From(member)), n), GainLifeFor(You, FromNumber(n)))),
		GainLifeFor(You, FromNumber(n)))
	if err := ApplyEffect(g, body, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 29 {
		t.Fatalf("life = %d", a.Life())
	}
}

// --- from composable_opponent_target_test.go ---
func TestTypedOpponentTargetCandidatesAndActivation(t *testing.T) {
	for _, self := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
		source := NewArtifact("Drain", "", WithAbility(Activate(Costs(), Sequence(LoseLifeFor(target, Fixed(1)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		candidates := target.Possible(a.PlayerID(), source, g)
		if len(candidates) != 1 || candidates[0] != b.PlayerID() {
			t.Fatal("incorrect opponent candidates")
		}
		id := b.PlayerID()
		if self {
			id = a.PlayerID()
		}
		err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
		if self {
			if err == nil || g.StackSize() != 0 {
				t.Fatal("accepted controller target")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != 19 {
			t.Fatal("incorrect drain")
		}
	}
}

func TestTypedOpponentTargetCopy(t *testing.T) {
	for _, choice := range []string{"keep", "retarget", "invalid"} {
		t.Run(choice, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
			ability := Activate(Costs(), Sequence(LoseLifeFor(target, Fixed(1)), GainLifeFor(You, Fixed(1))), WithTargets(target))
			source := NewCreature("Drain", "", 1, 1, WithAbility(ability))
			original := &StackObject{ID: uuid.New(), SourceID: source.ID(), TargetSource: source, Controller: a.PlayerID(), IsAbility: true, Effects: ability.Effects(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets()}
			g.pushStack(original)
			id := a.PlayerID()
			if choice == "invalid" {
				id = b.PlayerID()
			}
			b.chooseQueue = [][]uuid.UUID{{id}}
			cp := g.CopyStackObjectDirect(original, b.PlayerID(), choice != "keep")
			want := b.PlayerID()
			if choice == "retarget" {
				want = a.PlayerID()
			}
			if cp.Targets[0] != want || original.Targets[0] != b.PlayerID() {
				t.Fatal("incorrect copy targets")
			}
			clone := g.Clone()
			clone.ResolveTopOfStack()
			wantA, wantB := 20, 20
			if choice == "retarget" {
				wantA, wantB = 19, 21
			}
			if clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("copy did not recheck opponent restriction")
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveTopOfStack()
			g.ResolveTopOfStack()
			if a.Life() != wantA+1 || b.Life() != wantB-1 {
				t.Fatal("original lost its target or controller")
			}
		})
	}
}

func TestTypedOpponentTargetTriggerController(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(CreatureDies(), Sequence(LoseLifeFor(target, Fixed(1)), GainLifeFor(You, Fixed(1))), WithTargets(target))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	g.AddControlEffect(ControlEffectSpec{SourceID: victim.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() || g.StackPeek().Targets[0] != b.PlayerID() {
		t.Fatal("source control changed trigger controller or target")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 19 {
		t.Fatal("trigger used source controller at resolution")
	}
}

// --- from composable_owned_set_test.go ---
func TestPermanentsOwnedByOwnerAndLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	owned := NewArtifact("Owned", "")
	owned.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(owned, b.PlayerID())
	other := NewArtifact("Other", "")
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, b.PlayerID())
	set := PermanentsOwnedBy(You, IsArtifact)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	count := CountPermanents(set)
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("owned artifacts = %d, want 1", got)
	}
	g.PhaseOut(p.ID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("phased count = %d, want 0", got)
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	if got := count.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("clone count = %d, want 0", got)
	}
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("original count = %d, want 1", got)
	}
	g.RemoveFromBattlefield(p)
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("departed count = %d, want 0", got)
	}
	g.PutOnBattlefield(owned, b.PlayerID())
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("returned count = %d, want 1", got)
	}
}

func TestPermanentsOwnedByTargetAndDeclaration(t *testing.T) {
	target := DeclarePlayerTarget("player")
	body := GainLifeFor(You, CountPermanents(PermanentsOwnedBy(target, IsArtifact)))
	spell := Spell(body, WithTargets(target))
	card := NewInstant("Count owned", "", WithAbility(spell))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: card,
		},
		{
			Name:    "nil player target",
			Card:    NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(PermanentsOwnedBy(nil, IsArtifact)))))),
			WantErr: true,
		},
		{
			Name:    "undeclared player target",
			Card:    NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(PermanentsOwnedBy(target, IsArtifact)))))),
			WantErr: true,
		},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(artifact, a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: spell.Targets(), Effects: spell.Effects()}
	g.pushStack(obj)
	g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("copy lives = %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_permanent_count_test.go ---
func TestCountPermanentsResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	value := CountPermanents(ControlledPermanents(And(IsCreature, HasKeywordFilter(Flying))))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if value.resolveValue(ctx) != 0 {
		t.Fatal("empty set must count as zero")
	}
	flyer := NewCreature("Flyer", "", 1, 1, WithKeyword(Flying))
	flyer.SetOwner(a.PlayerID())
	g.PutOnBattlefield(flyer, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Other flyer", "", 1, 1, WithKeyword(Flying)), b.PlayerID())
	g.PutOnBattlefield(NewCreature("Ground creature", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Artifact", "", WithKeyword(Flying)), a.PlayerID())
	if value.resolveValue(ctx) != 1 {
		t.Fatal("count must filter type, keyword, and controller")
	}
	permanent := g.MutablePermanent(flyer.ID())
	permanent.PhasedOut = true
	if value.resolveValue(ctx) != 0 {
		t.Fatal("phased-out members must be absent")
	}
	permanent.PhasedOut = false
	clone := g.Clone()
	clone.DestroyPermanent(clone.FindPermanent(flyer.ID()))
	if value.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 0 || value.resolveValue(ctx) != 1 {
		t.Fatal("count must read its own game")
	}
	saved := BindNumber("before")
	body := Sequence(SnapshotNumber(value, saved), Destroy(This), GainLifeFor(You, value), GainLifeFor(You, FromNumber(saved)))
	if err := ApplyEffect(g, body, flyer.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 {
		t.Fatalf("life = %d; count must update after destruction while snapshot stays fixed", a.Life())
	}
	g.PutOnBattlefield(flyer, a.PlayerID())
	if value.resolveValue(ctx) != 1 {
		t.Fatal("dynamic collection must include returned objects")
	}
}

func TestCountPermanentsValidation(t *testing.T) {
	member := BindPermanent("member")
	value := CountPermanents(WithoutPermanent(MatchingPermanents(IsCreature), From(member)))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "unbound member",
			Card:    NewCreature("Count", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, value)))),
			WantErr: true,
		},
		{
			Name: "bound member",
			Card: NewCreature("Count", "", 1, 1, WithAbility(Activate(Costs(), Sequence(Snapshot(This, member), GainLifeFor(You, value))))),
		},
		{
			Name:    "nil collection",
			Card:    NewInstant("Nil count", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(nil))))),
			WantErr: true,
		},
	})
}

func TestCountPermanentsCopiedController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(At(EachEndStep,
		GainLifeFor(You, CountPermanents(ControlledPermanents(IsCreature))))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("expected one trigger")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.PutOnBattlefield(NewCreature("Later creature", "", 1, 1), a.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("count must use resolution-time membership and the copy controller")
	}
}

// --- from composable_player_binding_test.go ---
func TestRandomPlayerBindingResolution(t *testing.T) {
	for _, index := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLibrary([]Card{NewArtifact("A top", "")})
		b.SetLibrary([]Card{NewArtifact("B top", "")})
		saved := BindPlayer("recipient")
		body := Sequence(ChooseRandomPlayer(EachPlayer, saved), Mill(FromPlayer(saved), Fixed(1)))
		g.SetRandomResults([]int{index})
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Graveyard()) != 1-index || len(b.Graveyard()) != index {
			t.Fatal("random choice milled the wrong player")
		}
		if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: g}); ok {
			t.Fatal("binding leaked past resolution")
		}
	}
}

func TestRandomPlayerBindingAbsenceAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.SetRandomResults([]int{1, 0})
	if err := ChooseRandomPlayer(EachPlayer, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if err := ChooseRandomPlayer(OpponentsOf(ControllerOf(This)), saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("empty set retained prior binding")
	}
	if got, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || got != b.PlayerID() {
		t.Fatal("clone changed original binding")
	}
	if clone.RandIntn(2) != 0 || g.RandIntn(2) != 0 {
		t.Fatal("empty set consumed randomness or clone shared queue")
	}
	clone.resolution.Clear()
	if len(clone.resolution.playerBindings) != 0 {
		t.Fatal("clear retained player bindings")
	}
}

func TestRandomPlayerBindingNestedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	body := Sequence(ChooseRandomPlayer(OpponentsOf(You), saved),
		FuncEffect("nested", EffectProperties{}, func(g *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
			return ApplyEffect(g, GainLifeFor(FromPlayer(saved), Fixed(7)), source, controller, nil)
		}), GainLifeFor(FromPlayer(saved), Fixed(2)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("nested frame leaked: %d %d", a.Life(), b.Life())
	}
}

func TestPlayerBindingValidation(t *testing.T) {
	saved := BindPlayer("saved")
	choose := ChooseRandomPlayer(EachPlayer, saved)
	read := Mill(FromPlayer(saved), Fixed(1))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(choose, read)))),
		},
		{
			Name:          "missing",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), read))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "before",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(read, choose)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "branch",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(IfState(Controls(IsCreature), choose), read)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "duplicate",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(choose, ChooseRandomPlayer(EachPlayer, BindPlayer("saved")))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "cross category duplicate",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(choose, Snapshot(This, BindPermanent("saved")))))),
			WantErr:       true,
			WantErrSubstr: "duplicate",
		},
		{
			Name:          "nil set",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), ChooseRandomPlayer(nil, saved)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil binding",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), ChooseRandomPlayer(EachPlayer, nil)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil read",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Mill(FromPlayer(nil), Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty name",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), ChooseRandomPlayer(EachPlayer, BindPlayer(""))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "loop escape",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), choose), read)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "delayed escape",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), Sequence(AtNext(EachEndStep, choose), read)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "set dependency",
			Card:          NewCreature("Binding", "", 1, 1, WithAbility(Activate(Costs(), ChooseRandomPlayer(OpponentsOf(FromPlayer(saved)), saved)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
	info := InspectDefinition(choose)
	if info.Role != "random-player-choice" || len(info.Children) != 2 {
		t.Fatalf("metadata = %+v", info)
	}
}

func TestRandomPlayerBindingLoopAndDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	saved := BindPlayer("recipient")
	body := Sequence(ChooseRandomPlayer(OpponentsOf(You), saved),
		ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), Sequence(
			GainLifeFor(FromPlayer(saved), Fixed(1)),
			ChooseRandomPlayer(OpponentsOf(FromPlayer(saved)), saved))),
		AtNext(EachEndStep, GainLifeFor(FromPlayer(saved), Fixed(3))),
		ChooseRandomPlayer(OpponentsOf(FromPlayer(saved)), saved))
	card := NewSorcery("Capture", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("loop failed to restore the outer player binding")
	}
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 25 {
			t.Fatal("delayed trigger lost its captured player")
		}
		if game == clone && b.Life() != 22 {
			t.Fatal("delayed resolution changed original game")
		}
	}
}

func TestRandomPlayerBindingRepeatedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	body := Sequence(ChooseRandomPlayer(EachPlayer, saved), GainLifeFor(FromPlayer(saved), Fixed(1)))
	g.SetRandomResults([]int{0, 1})
	card := NewSorcery("Random", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	InspectDefinition(body)
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("inspection consumed randomness or repeated resolution reused a binding")
	}
}

func TestRandomPlayerBindingSpellCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("recipient")
	ability := Spell(Sequence(ChooseRandomPlayer(OpponentsOf(You), saved), GainLifeFor(FromPlayer(saved), Fixed(2))))
	card := NewSorcery("Random", "", WithAbility(ability))
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("copy did not choose from its controller's opponents")
	}
	g.ResolveStackObject(original)
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("original reused copy binding")
	}
}

// --- from composable_player_planeswalker_target_test.go ---
func TestPlayerOrPlaneswalkerDamageTargetContract(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 4), b.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
	target := DeclarePlayerOrPlaneswalkerTarget("recipient")
	possible := target.spec.Possible(a.PlayerID(), source.Card, g)
	for _, id := range possible {
		if id == creature.ID() {
			t.Fatal("creature was a legal player-or-planeswalker target")
		}
	}
	if !slices.Contains(possible, a.PlayerID()) || !slices.Contains(possible, b.PlayerID()) || !slices.Contains(possible, walker.ID()) {
		t.Fatalf("legal recipients missing: %v", possible)
	}
	g.resolution.targetSpecs = []Target{target.targetDeclaration}
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(2)), source.ID(), a.PlayerID(), []uuid.UUID{walker.ID()}); err != nil {
		t.Fatal(err)
	}
	if walker.Counters[Loyalty] != 3 || creature.Damage != 0 {
		t.Fatalf("loyalty %d, creature damage %d", walker.Counters[Loyalty], creature.Damage)
	}
	g.PhaseOut(walker.ID())
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(1)), source.ID(), a.PlayerID(), []uuid.UUID{walker.ID()}); err != nil {
		t.Fatal(err)
	}
	if walker.Counters[Loyalty] != 3 {
		t.Fatal("phased planeswalker received damage")
	}
	g.PhaseIn([]uuid.UUID{walker.ID()})
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{walker.ID(): walker.incarnationID}
	g.RemoveFromBattlefield(walker)
	returned := g.PutOnBattlefield(walker.Card, b.PlayerID())
	if err := ApplyEffect(g, DealDamageFrom(This, target, Fixed(1)), source.ID(), a.PlayerID(), []uuid.UUID{returned.ID()}); err != nil {
		t.Fatal(err)
	}
	if returned.Counters[Loyalty] != 5 {
		t.Fatal("damage followed returned planeswalker")
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, DealDamageFrom(This, target, Fixed(2)), source.ID(), a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone damage or player target failed")
	}
}

func TestPlayerOrPlaneswalkerDamageTargetDeclaration(t *testing.T) {
	target := DeclarePlayerOrPlaneswalkerTarget("recipient")
	binding := DeclarePlayerOrPlaneswalkerTarget("")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Valid", "", Spell(DealDamageFrom(This, target, Fixed(2)), WithTargets(target))),
		},
		{
			Name:          "missing target",
			Card:          NewInstant("Missing", "", Spell(DealDamageFrom(This, target, Fixed(2)))),
			WantErr:       true,
			WantErrSubstr: "target binding requires",
		},
		{
			Name:    "empty target label",
			Card:    NewInstant("Bad", "", Spell(DealDamageFrom(This, binding, Fixed(2)), WithTargets(binding))),
			WantErr: true,
		},
	})
}

// --- from composable_player_snapshot_test.go ---
func TestPlayerSnapshotLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	saved := BindPlayer("controller")
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	if err := SnapshotPlayer(ControllerOf(This), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(source.Card, a.PlayerID())
	if err := PreventNextDamage(FromPlayer(saved), Fixed(3)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 5, uuid.Nil)
	if clone.GetPlayer(b.PlayerID()).Life() != 18 || b.Life() != 20 {
		t.Fatal("captured player or clone shield changed")
	}
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	g.DealDamageToPlayer(b, 2, uuid.Nil)
	g.DealDamageToPlayer(b, 3, uuid.Nil)
	if a.Life() != 19 || b.Life() != 18 {
		t.Fatal("shield did not retain the captured controller")
	}
	if err := SnapshotPlayer(FromPlayer(saved), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("self capture lost its input")
	}
	cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if err := SnapshotPlayer(ControllerOf(This), saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("absent subject retained the old player")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("clone changed the original binding")
	}
}

func TestPlayerSnapshotDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("player")
	target := DeclarePlayerTarget("target")
	body := Sequence(SnapshotPlayer(target, saved), AtNext(EachEndStep, GainLifeFor(FromPlayer(saved), Fixed(3))), SnapshotPlayer(You, saved))
	ability := Spell(body, WithTargets(target))
	card := NewInstant("Snapshot", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveStack()
	if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: g}); ok {
		t.Fatal("binding escaped resolution")
	}
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("delayed trigger lost its snapshot")
		}
		if game == clone && b.Life() != 20 {
			t.Fatal("clone changed the original player")
		}
	}
}

func TestPlayerSnapshotValidation(t *testing.T) {
	saved := BindPlayer("saved")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewInstant("Snapshot", "", WithAbility(Spell(Sequence(SnapshotPlayer(You, saved), GainLifeFor(FromPlayer(saved), Fixed(1)))))),
		},
		{
			Name:          "nil subject",
			Card:          NewInstant("Snapshot", "", WithAbility(Spell(SnapshotPlayer(nil, saved)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "nil binding",
			Card:          NewInstant("Snapshot", "", WithAbility(Spell(SnapshotPlayer(You, nil)))),
			WantErr:       true,
			WantErrSubstr: "non-nil",
		},
		{
			Name:          "empty name",
			Card:          NewInstant("Snapshot", "", WithAbility(Spell(SnapshotPlayer(You, BindPlayer(""))))),
			WantErr:       true,
			WantErrSubstr: "name",
		},
		{
			Name:          "read before write",
			Card:          NewInstant("Snapshot", "", WithAbility(Spell(SnapshotPlayer(FromPlayer(saved), saved)))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
		{
			Name:          "undeclared target",
			Card:          NewInstant("Snapshot", "", WithAbility(Spell(SnapshotPlayer(DeclarePlayerTarget("target"), saved)))),
			WantErr:       true,
			WantErrSubstr: "target",
		},
		{
			Name:          "optional producer",
			Card:          NewInstant("Snapshot", "", WithAbility(Spell(Sequence(IfState(Controls(IsCreature), SnapshotPlayer(You, saved)), GainLifeFor(FromPlayer(saved), Fixed(1)))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestPlayerSnapshotSpellCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("controller")
	ability := Spell(Sequence(SnapshotPlayer(You, saved), GainLifeFor(FromPlayer(saved), Fixed(2))))
	card := NewInstant("Snapshot", "", WithAbility(ability))
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("copy captured the wrong controller")
	}
	g.ResolveStackObject(original)
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("original reused the copy binding")
	}
}

func TestPlayerSnapshotFrameIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("Member", "", 1, 1), b.PlayerID())
	saved := BindPlayer("player")
	member := BindPermanent("member")
	body := Sequence(SnapshotPlayer(You, saved),
		ForEach(MatchingPermanents(IsCreature), member, Sequence(SnapshotPlayer(ControllerOf(From(member)), saved), GainLifeFor(FromPlayer(saved), Fixed(1)))),
		FuncEffect("nested resolution", EffectProperties{}, func(g *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
			return ApplyEffect(g, Sequence(SnapshotPlayer(You, saved), GainLifeFor(FromPlayer(saved), Fixed(2))), source, b.PlayerID(), nil)
		}), GainLifeFor(FromPlayer(saved), Fixed(3)))
	g.SetRandomResults([]int{1})
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 26 || b.Life() != 26 {
		t.Fatalf("frame leaked: life = %d/%d", a.Life(), b.Life())
	}
	if g.RandIntn(2) != 1 {
		t.Fatal("snapshot consumed randomness")
	}
}

// --- from composable_source_counters_test.go ---
func TestSourceCountersLifetime(t *testing.T) {
	for _, change := range []string{"current", "leave", "sacrifice", "return", "return twice", "cleanup", "phase", "phase cycle", "control", "clone before", "clone after"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewEnchantment("Counter mana", "", WithAbility(At(YourFirstMainPhase, AddManaAmount(Black, SourceCounters(Charge)))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
			g.MutablePermanent(source.ID()).AddCounter(Tide, 9)
			g.FireEvent(GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: true})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.ManaPool().Count(Black) != 0 {
				t.Fatal("first-main trigger must use the stack")
			}
			g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
			original := g
			if change == "clone before" {
				g = g.Clone()
				g.MutablePermanent(source.ID()).AddCounter(Charge, 1)
			}
			switch change {
			case "sacrifice":
				g.DoSacrifice(g.FindPermanent(source.ID()))
			case "leave", "return", "return twice", "cleanup", "clone before", "clone after":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" || change == "return twice" || change == "clone after" {
					if change == "clone after" {
						g = g.Clone()
					}
					g.PutOnBattlefield(source, a.PlayerID())
					g.MutablePermanent(source.ID()).AddCounter(Charge, 17)
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "return" {
						g.PutOnBattlefield(source, a.PlayerID())
					}
				}
				if change == "cleanup" {
					g.zones.ClearEndOfTurn()
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
				g.MutablePermanent(source.ID()).AddCounter(Charge, 1)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy missing")
			}
			g.ResolveStack()
			want := 5
			if change == "clone before" || change == "phase cycle" {
				want = 6
			}
			for _, player := range []*BasePlayer{a, b} {
				if got := g.GetPlayer(player.PlayerID()).ManaPool().Count(Black); got != want {
					t.Fatalf("mana = %d, want %d", got, want)
				}
			}
			if g != original {
				if a.ManaPool().Count(Black) != 0 || b.ManaPool().Count(Black) != 0 {
					t.Fatal("clone changed original mana")
				}
				original.ResolveStack()
				if a.ManaPool().Count(Black) != 5 {
					t.Fatal("clone changed original counter history")
				}
			}
		})
	}
}

func TestSourceCountersDelayedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := AtNext(EachEndStep, GainLifeFor(You, SourceCounters(Charge)))
	source := NewCreature("Delayed counters", "", 2, 2, WithAbility(Activate(Costs(), body)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.zones.ClearEndOfTurn()
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 11)
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatalf("delayed gain = %d, want 5", a.Life()-20)
	}
}

func TestSourceCountersValidation(t *testing.T) {
	value := SourceCounters(Charge)
	card := NewSorcery("Invalid source", "", WithAbility(Spell(GainLifeFor(You, value))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("spell accepted battlefield source value")
	}
	for _, kind := range []CounterType{-1, NumCounters} {
		t.Run(kind.String(), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid counter accepted")
				}
			}()
			SourceCounters(kind)
		})
	}
	info := InspectDefinition(value)
	if info.Role != "source-counter-value" || info.Text != "Charge counters on source, with last-known information" {
		t.Fatalf("metadata = %+v", info)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if got := value.resolveValue(&EffectContext{Game: g}); got != 0 {
		t.Fatalf("missing source count = %d", got)
	}
}

func TestSourceCountersDeathAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := Sequence(GainLifeFor(You, SourceCounters(Charge)), GainLifeFor(You, SourceCounters(Tide)))
	source := NewCreature("Death counters", "", 2, 2, WithAbility(WhenEvent(Dies(This), body)))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	g.DoSacrifice(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("death life = %d, want 22", a.Life())
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).AddCounter(Charge, 4)
	clone.DoSacrifice(clone.FindPermanent(source.ID()))
	g.DoSacrifice(g.FindPermanent(source.ID()))
	for _, game := range []*Game{g, clone} {
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if a.Life() != 25 || clone.GetPlayer(a.PlayerID()).Life() != 29 {
		t.Fatal("games shared counter history for the same source incarnation")
	}
	if got := SourceCounters(Charge).resolveValue(&EffectContext{Game: g, SourceID: source.ID()}); got != 0 {
		t.Fatalf("uncaptured source read history: %d", got)
	}
}

// --- from composable_source_targeted_test.go ---
func TestSourceTargetedBySpellEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargetOfSpell(This), Sacrifice(This))))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	other := NewCreature("Other", "", 2, 2)
	g.PutOnBattlefield(other, b.PlayerID())
	for _, evt := range []GameEvent{
		{Type: EvtBecomesTarget, TargetID: other.ID()},
		{Type: EvtBecomesTarget, TargetID: card.ID(), Flag: true},
	} {
		g.FireEvent(evt)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("nonmatching target event triggered")
	}
	g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: card.ID(), SourceID: uuid.New(), PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("spell target triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || clone.FindPermanent(card.ID()) != nil {
		t.Fatal("clone did not isolate trigger resolution")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("source did not sacrifice")
	}
}

func TestSourceTargetedBySpellCast(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
		BecomesTargetOfSpell(This), Sacrifice(This))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("target", IsCreature)
	spell := NewInstant("Tap spell", "", WithAbility(Spell(TapSubject(target), WithTargets(target))))
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByName(b.PlayerID(), spell.Name(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("cast stack size = %d", g.StackSize())
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("target trigger did not sacrifice source")
	}
}

func TestSourceTargetedBySpellDepartureAndPhasing(t *testing.T) {
	for _, tc := range []string{"phased", "absent", "left", "returned"} {
		t.Run(tc, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(
				BecomesTargetOfSpell(This), GainLifeFor(You, Fixed(1)))))
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			switch tc {
			case "phased":
				g.PhaseOut(card.ID())
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			}
			g.FireEvent(GameEvent{Type: EvtBecomesTarget, TargetID: card.ID()})
			if tc == "phased" || tc == "absent" {
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("unavailable source triggered")
				}
				return
			}
			g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
			if tc == "returned" {
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("pending trigger lost its controller")
			}
		})
	}
}

func TestSourceTargetedBySpellDeclaration(t *testing.T) {
	if info := InspectDefinition(BecomesTargetOfSpell(This)); info.Role != "event-pattern" {
		t.Fatalf("inspection role = %q", info.Role)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(BecomesTargetOfSpell(This), Sacrifice(This)))),
		},
		{
			Name:          "instant with battlefield trigger",
			Card:          NewInstant("Invalid", "", WithAbility(WhenEvent(BecomesTargetOfSpell(This), GainLifeFor(You, Fixed(1))))),
			WantErr:       true,
			WantErrSubstr: "battlefield",
		},
	})
}

// --- from composable_spell_cast_binding_test.go ---
func TestSpellCastBindingUsesCastStackIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := BindSpell("cast spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell),
		Sequence(GainLifeFor(You, ManaValueOfSpell(FromSpell(spell))), Counter(FromSpell(spell))))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Cast card", "{X}{U}")
	card.SetOwner(b.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), XValue: 1}
	cast := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID(), XValue: 3}
	g.pushStack(old)
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack=%d, want 3", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if a.Life() != 20 || g.StackSize() != 3 {
		t.Fatal("clone changed original game")
	}
	g.ResolveTopOfStack()
	if a.Life() != 24 || g.stack.FindByID(cast.ID) != nil || g.stack.FindByID(old.ID) == nil {
		t.Fatalf("life=%d, cast=%v, old=%v", a.Life(), g.stack.FindByID(cast.ID), g.stack.FindByID(old.ID))
	}
}

func TestSpellCastBindingAbsentStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := BindSpell("cast spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell), GainLifeFor(You, ManaValueOfSpell(FromSpell(spell))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Cast card", "{3}{U}")
	card.SetOwner(b.PlayerID())
	cast := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.stack.RemoveByID(cast.ID)
	g.pushStack(&StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatalf("returned stack card supplied old binding: life=%d", a.Life())
	}
}

func TestSpellCastBindingCopyAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := BindSpell("cast spell")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithSpell(spell), GainLifeFor(You, ManaValueOfSpell(FromSpell(spell))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	card := NewInstant("Cast card", "{2}{U}")
	card.SetOwner(b.PlayerID())
	cast := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
	g.pushStack(cast)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.PhaseOut(watcher.ID())
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatalf("copied captures: life A=%d B=%d", a.Life(), b.Life())
	}
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: cast.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("phased watcher created trigger: stack=%d", g.StackSize())
	}
}

func TestSpellCastBindingDeclaration(t *testing.T) {
	spell := BindSpell("spell")
	if got := InspectDefinition(SpellCast().WithSpell(spell)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection=%+v", got)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil spell binding",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().WithSpell(nil), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "empty name spell binding",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().WithSpell(BindSpell("")), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "duplicate spell binding",
			Card:    NewArtifact("Bad", "", WithAbility(WhenEvent(SpellCast().WithSpell(spell).WithSpell(spell), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:          "unbound spell",
			Card:          NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), Counter(FromSpell(spell))))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

// --- from composable_spell_cast_creature_target_test.go ---
func TestSpellCastTargetsCreatureUsesCurrentTargetAndExactSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(),
		GainLifeFor(You, Fixed(1))))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	other := g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID())
	card := NewInstant("Spell", "")
	card.SetOwner(a.PlayerID())
	spell := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{other.ID(), creature.ID()}}
	g.pushStack(spell)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: spell.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d, want spell and trigger", g.StackSize())
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("clone changed original player")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	g.RemoveFromBattlefield(creature)
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if a.Life() != 22 {
		t.Fatalf("pending triggers lost event result: life=%d", a.Life())
	}
	g.PhaseOut(watcher.ID())
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: spell.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phased watcher observed cast")
	}
}

func TestSpellCastTargetsCreatureDeclaration(t *testing.T) {
	pattern := SpellCast().OfInstantOrSorcery().TargetsCreature()
	if got := InspectDefinition(pattern); !strings.Contains(got.Text, "targets a creature") {
		t.Fatalf("inspection=%+v", got)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid",
			Card: NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1))))),
		},
		{
			Name:    "repeated target filter",
			Card:    NewArtifact("Invalid", "", WithAbility(WhenEvent(pattern.TargetsCreature(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "repeated type filter instant",
			Card:    NewArtifact("Bad type", "", WithAbility(WhenEvent(pattern.OfType(TypeInstant), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "repeated type filter instant/sorcery",
			Card:    NewArtifact("Bad type", "", WithAbility(WhenEvent(pattern.OfInstantOrSorcery(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "conflicting caster filter controller/opponent",
			Card:    NewArtifact("Bad caster", "", WithAbility(WhenEvent(pattern.ByController().ByOpponent(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
		{
			Name:    "conflicting caster filter opponent/opponent",
			Card:    NewArtifact("Bad caster", "", WithAbility(WhenEvent(pattern.ByOpponent().ByOpponent(), GainLifeFor(You, Fixed(1))))),
			WantErr: true,
		},
	})
}

func TestSpellCastTargetsCreatureCurrentState(t *testing.T) {
	for _, tc := range []struct {
		name        string
		spellType   CardType
		targetType  CardType
		remove      bool
		phase       bool
		returnSame  bool
		zeroTargets bool
		otherCaster bool
		want        bool
	}{
		{name: "instant creature", spellType: TypeInstant, targetType: TypeCreature, want: true},
		{name: "sorcery creature", spellType: TypeSorcery, targetType: TypeCreature, want: true},
		{name: "artifact spell", spellType: TypeArtifact, targetType: TypeCreature},
		{name: "artifact target", spellType: TypeInstant, targetType: TypeArtifact},
		{name: "no chosen target", spellType: TypeInstant, targetType: TypeCreature, zeroTargets: true},
		{name: "departed target", spellType: TypeInstant, targetType: TypeCreature, remove: true},
		{name: "phased target", spellType: TypeInstant, targetType: TypeCreature, phase: true},
		{name: "returned target", spellType: TypeInstant, targetType: TypeCreature, returnSame: true},
		{name: "other caster", spellType: TypeInstant, targetType: TypeCreature, otherCaster: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
				SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(), GainLifeFor(You, Fixed(1))))), a.PlayerID())
			var target *Permanent
			if tc.targetType == TypeCreature {
				target = g.PutOnBattlefield(NewCreature("Target", "", 2, 2), b.PlayerID())
			} else {
				target = g.PutOnBattlefield(NewArtifact("Target", ""), b.PlayerID())
			}
			var card Card
			switch tc.spellType {
			case TypeInstant:
				card = NewInstant("Spell", "")
			case TypeSorcery:
				card = NewSorcery("Spell", "")
			default:
				card = NewArtifact("Spell", "")
			}
			card.SetOwner(a.PlayerID())
			var targets []uuid.UUID
			if !tc.zeroTargets {
				targets = []uuid.UUID{target.ID()}
			}
			caster := a.PlayerID()
			if tc.otherCaster {
				caster = b.PlayerID()
			}
			spell := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: caster, Targets: targets}
			g.pushStack(spell)
			if tc.remove || tc.returnSame {
				g.RemoveFromBattlefield(target)
			}
			if tc.returnSame {
				g.PutOnBattlefield(target.Card, b.PlayerID())
			}
			if tc.phase {
				g.PhaseOut(target.ID())
			}
			g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: spell.ID, PlayerID: caster})
			g.PutTriggersOnStack()
			if got := g.StackSize() == 2; got != tc.want {
				t.Fatalf("triggered=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestSpellCastTargetsCreatureRealCastAndFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByController().OfInstantOrSorcery().TargetsCreature(),
		GainLifeFor(You, Fixed(1))))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	unpaid := NewInstant("Unpaid", "{U}", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargets(target))))
	unpaid.SetOwner(a.PlayerID())
	a.AddToHand(unpaid)
	if err := g.CastSpellByName(a.PlayerID(), unpaid.Name(), []uuid.UUID{creature.ID()}); err == nil {
		t.Fatal("unpaid spell was cast")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed cast produced a trigger")
	}
	free := NewSorcery("Free", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargets(target))))
	free.SetOwner(a.PlayerID())
	a.AddToHand(free)
	g.turns.SetStep(PrecombatMain)
	if err := g.CastSpellByName(a.PlayerID(), free.Name(), []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("successful cast stack=%d, want two", g.StackSize())
	}
}

func TestSpellCastTargetsCreatureUsesCastStackObject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().TargetsCreature(), GainLifeFor(You, Fixed(1))))), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
	card := NewInstant("Shared", "")
	card.SetOwner(a.PlayerID())
	old := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{creature.ID()}}
	current := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{artifact.ID()}}
	g.pushStack(old)
	g.pushStack(current)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: card.ID(), TargetID: current.ID, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("cast event matched another stack object with the same source card")
	}
}

// --- from composable_spell_mana_value_test.go ---
func TestManaValueOfSpellStackXAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	variable := NewInstant("Variable", "{X}{X}{U}")
	variable.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: variable.ID(), Card: variable, Controller: b.PlayerID(), XValue: 3}
	g.pushStack(obj)
	spell := DeclareSpellTarget("spell")
	value := ManaValueOfSpell(spell)
	ctx := &EffectContext{Game: g, targetBindings: map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: obj.ID}}}}
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("stack mana value = %d, want 7", got)
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	if got := value.resolveValue(&cloneCtx); got != 7 {
		t.Fatalf("clone mana value = %d", got)
	}
	clone.stack.RemoveByID(obj.ID)
	if got := value.resolveValue(&cloneCtx); got != 0 {
		t.Fatalf("departed clone spell = %d", got)
	}
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("original changed = %d", got)
	}
	g.stack.RemoveByID(obj.ID)
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: variable.ID(), Card: variable, Controller: b.PlayerID(), XValue: 5})
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("recast card matched old stack identity: %d", got)
	}
}

func TestManaValueOfSpellResolutionAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewInstant("Victim", "{3}{U}")
	victim.SetOwner(b.PlayerID())
	target := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(target)
	spell := DeclareSpellTarget("spell")
	ability := Spell(GainLifeFor(You, ManaValueOfSpell(spell)), WithTargets(spell))
	card := NewInstant("Reader", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name: "valid reader",
			Card: card,
		},
		{
			Name:          "nil ref",
			Card:          NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, ManaValueOfSpell(nil)), WithTargets(spell)))),
			WantErr:       true,
			WantErrSubstr: "spell",
		},
		{
			Name:          "undeclared ref",
			Card:          NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, ManaValueOfSpell(DeclareSpellTarget("other"))), WithTargets(spell)))),
			WantErr:       true,
			WantErrSubstr: "spell",
		},
	})
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("life = %d, want 24", got)
	}
}

func TestManaValueOfSpellCopiedFrameAndTargetLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := NewInstant("First", "{X}{R}")
	secondCard := NewArtifact("Second", "{4}")
	firstCard.SetOwner(b.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: b.PlayerID(), XValue: 2}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	allowed := true
	filter := NewCardFilter("allowed", func(card Card) bool { return allowed && card.HasType(TypeInstant) })
	spell := DeclareSpellTarget("spell")
	ability := Spell(GainLifeFor(You, ManaValueOfSpell(spell)), WithTargets(spell))
	reader := NewInstant("Reader", "", WithAbility(ability))
	reader.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: reader.ID(), Card: reader, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = g.zoneOfTarget(second.ID)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("copy used wrong spell: life = %d", got)
	}
	g.ResolveTopOfStack()
	if got := a.Life(); got != 27 {
		t.Fatalf("original lost its target: life = %d", got)
	}
	filtered := DeclareSpellTarget("filtered", filter)
	filteredAbility := Spell(GainLifeFor(You, ManaValueOfSpell(filtered)), WithTargets(filtered))
	filteredReader := NewInstant("Filtered reader", "", WithAbility(filteredAbility))
	filteredReader.SetOwner(a.PlayerID())
	filteredObj := &StackObject{ID: uuid.New(), SourceID: filteredReader.ID(), Card: filteredReader, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: filteredAbility.Targets(), Effects: filteredAbility.Effects()}
	g.pushStack(filteredObj)
	allowed = false
	g.ResolveTopOfStack()
	if got := a.Life(); got != 27 {
		t.Fatalf("illegal target supplied mana value: life = %d", got)
	}
}

// --- from composable_tap_choice_test.go ---
type tapChoicePlayer struct {
	*BasePlayer
	accept  bool
	prompts []string
}

func (p *tapChoicePlayer) ChooseMayAbility(prompt string) bool {
	p.prompts = append(p.prompts, prompt)
	return p.accept
}

func TestComposableTapChoice(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		tapped, accept, stun, want bool
	}{
		{"tap", false, true, false, true},
		{"untap", true, true, false, false},
		{"decline tap", false, false, false, false},
		{"decline untap", true, false, false, true},
		{"stun replacement", true, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			b := &tapChoicePlayer{BasePlayer: NewBasePlayer("B"), accept: !tc.accept}
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 1, 1)
			p := g.PutOnBattlefield(c, b.PlayerID())
			p.Tapped = tc.tapped
			if tc.stun {
				p.AddCounter(Stun, 1)
			}
			if err := ApplyEffect(g, MayTapOrUntap(This), c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()).Tapped != tc.want || len(a.prompts) != 1 || len(b.prompts) != 0 {
				t.Fatal("wrong action or choosing player")
			}
			verb := "tap "
			if tc.tapped {
				verb = "untap "
			}
			if !strings.HasPrefix(a.prompts[0], verb) {
				t.Fatalf("prompt = %q", a.prompts[0])
			}
			if tc.stun && g.FindPermanent(c.ID()).Counters[Stun] != 0 {
				t.Fatal("stun replacement bypassed")
			}
		})
	}
}

func TestComposableTapChoiceSavedLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 1, 1)
			g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("subject")
			body := Sequence(Snapshot(This, saved), FuncEffect("change subject", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				switch change {
				case "return", "absent":
					g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
					if change == "return" {
						g.PutOnBattlefield(c, a.PlayerID())
					}
				case "phase":
					g.PhaseOut(c.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(c.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				}
				return nil
			}), MayTapOrUntap(From(saved)))
			if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			acted := change == "none" || change == "phase cycle" || change == "control"
			if (len(a.prompts) == 1) != acted {
				t.Fatal("choice used an absent incarnation")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{c.ID()})
			}
			if p := g.FindPermanent(c.ID()); p != nil && p.Tapped != acted {
				t.Fatal("wrong saved subject")
			}
		})
	}
}

func TestComposableTapChoiceEvents(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		tapped, accept, stun bool
		wantLife             int
	}{
		{"tap", false, true, false, 21},
		{"untap", true, true, false, 22},
		{"decline tap", false, false, false, 20},
		{"decline untap", true, false, false, 20},
		{"replace untap", true, true, true, 20},
		{"decline with stun", true, false, true, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &tapChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			g := NewGame(a, NewBasePlayer("B"))
			c := NewCreature("Subject", "", 1, 1)
			p := g.PutOnBattlefield(c, a.PlayerID())
			p.Tapped = tc.tapped
			if tc.stun {
				p.AddCounter(Stun, 1)
			}
			watcher := NewEnchantment("Watcher", "",
				WithAbility(NewTriggered(EvtTapped, false, GainLifeFor(You, Fixed(1)))),
				WithAbility(NewTriggered(EvtBecameUntapped, false, GainLifeFor(You, Fixed(2)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			if err := ApplyEffect(g, MayTapOrUntap(This), c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			resolvePendingTriggers(g)
			if a.Life() != tc.wantLife {
				t.Fatalf("life = %d, want %d from action events", a.Life(), tc.wantLife)
			}
			wantStun := uint8(0)
			if tc.stun && !tc.accept {
				wantStun = 1
			}
			if got := g.FindPermanent(c.ID()).Counters[Stun]; got != wantStun {
				t.Fatalf("stun counters = %d, want %d", got, wantStun)
			}
		})
	}
}

func TestComposableTapChoiceValidation(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil subject",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(MayTapOrUntap(nil)))),
			WantErr: true,
		},
		{
			Name:    "this on spell",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(MayTapOrUntap(This)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(MayTapOrUntap(DeclarePermanentTarget("undeclared", IsCreature))))),
			WantErr: true,
		},
		{
			Name:    "unwritten binding",
			Card:    NewInstant("Invalid", "", WithAbility(Spell(MayTapOrUntap(From(BindPermanent("unwritten")))))),
			WantErr: true,
		},
	})
}

func TestTypedOpponentTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	own, other := NewArtifact("Own", ""), NewArtifact("Other", "")
	g.PutOnBattlefield(own, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	target := DeclarePermanentTarget("artifact", IsArtifact, TargetOpponentControls())
	ability := Spell(TapSubject(target), WithTargets(target))
	card := NewInstant("Spell", "", WithAbility(ability))
	possible := ability.Targets()[0].Possible(a.PlayerID(), card, g)
	if len(possible) != 1 || possible[0] != other.ID() {
		t.Fatal("wrong controller restriction")
	}
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{other.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.AddControlEffect(ControlEffectSpec{SourceID: own.ID(), TargetID: other.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
	g.ResolveStack()
	if g.FindPermanent(other.ID()).Tapped {
		t.Fatal("target remained legal after control change")
	}
}

func TestComposableTapChoiceTargetClone(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		first, second := NewCreature("First", "", 1, 1), NewCreature("Second", "", 1, 1)
		g.PutOnBattlefield(first, b.PlayerID())
		g.PutOnBattlefield(second, b.PlayerID())
		x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
		ability := Spell(Sequence(MayTapOrUntap(x), MayTapOrUntap(y)), WithTargets(x, y))
		card := NewInstant("Choice", "", WithAbility(ability))
		g.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			g.PutOnBattlefield(first, b.PlayerID())
		}
		clone := g.Clone()
		chooser := &tapChoicePlayer{BasePlayer: extractBasePlayer(clone.GetPlayer(a.PlayerID())), accept: true}
		clone.players[0] = chooser
		clone.TapPermanent(clone.FindPermanent(second.ID()))
		clone.ResolveStack()
		if clone.FindPermanent(first.ID()).Tapped == returned || clone.FindPermanent(second.ID()).Tapped {
			t.Fatal("wrong target incarnation or resolution-time state")
		}
		if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
			t.Fatal("clone changed original")
		}
		want := 2
		if returned {
			want = 1
		}
		if len(chooser.prompts) != want {
			t.Fatal("illegal target received a choice")
		}
	}
}

func TestTypedOpponentTargetConflicts(t *testing.T) {
	for _, collection := range []bool{false, true} {
		t.Run(map[bool]string{false: "singular", true: "collection"}[collection], func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("conflicting restrictions accepted")
				}
			}()
			if collection {
				DeclarePermanentTargets("targets", 0, 2, IsCreature, TargetYouControl(), TargetOpponentControls())
			} else {
				DeclarePermanentTarget("target", IsCreature, TargetOpponentControls(), TargetYouControl())
			}
		})
	}
}

func TestComposableTapChoiceCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	target := DeclarePermanentTarget("artifact", IsArtifact)
	ability := Spell(MayTapOrUntap(target), WithTargets(target))
	card := NewInstant("Choice", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	a.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	chooser := &tapChoicePlayer{BasePlayer: a.BasePlayer, accept: true}
	g.players[0] = chooser
	g.ResolveStackObject(copied)
	if !g.FindPermanent(second.ID()).Tapped || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy lost its new target")
	}
	chooser.accept = false
	g.ResolveStackObject(original)
	if g.FindPermanent(first.ID()).Tapped || len(chooser.prompts) != 2 {
		t.Fatal("copy choice affected original")
	}
}

// --- from composable_target_owner_test.go ---
func TestTypedTargetYouOwnCandidates(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewInstant("Source", "")
	var owned []uuid.UUID
	for _, owner := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			card := NewCreature("Candidate", "", 2, 2)
			card.SetOwner(owner)
			g.PutOnBattlefield(card, controller)
			if owner == a.PlayerID() {
				owned = append(owned, card.ID())
			}
		}
	}
	target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn())
	got := target.Possible(a.PlayerID(), source, g)
	if len(got) != 2 || !slices.Contains(got, owned[0]) || !slices.Contains(got, owned[1]) {
		t.Fatalf("owned candidates = %v", got)
	}
	both := DeclarePermanentTargets("both", 1, 2, AnyPermanent, TargetYouOwn(), TargetYouControl())
	got = both.Possible(a.PlayerID(), source, g)
	if len(got) != 1 || got[0] != owned[0] {
		t.Fatalf("owned and controlled candidates = %v", got)
	}
	opposing := DeclarePermanentTarget("opposing", AnyPermanent, TargetYouOwn(), TargetOpponentControls())
	got = opposing.Possible(a.PlayerID(), source, g)
	if len(got) != 1 || got[0] != owned[1] {
		t.Fatalf("owned opposing candidates = %v", got)
	}
	g.PhaseOut(owned[0])
	if len(both.Possible(a.PlayerID(), source, g)) != 0 {
		t.Fatal("phased target is available")
	}
}

func TestTypedTargetYouOwnResolution(t *testing.T) {
	for _, change := range []string{"none", "owner", "owner restored", "control", "phase cycle", "return"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, err := NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			card := NewCreature("Owned", "", 2, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn())
			ability := Spell(ReturnToHand(target), WithTargets(target))
			spell := NewInstant("Return", "", WithAbility(ability))
			obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			clone := g.Clone()
			switch change {
			case "owner", "owner restored":
				if err := clone.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
					t.Fatal(err)
				}
				if change == "owner restored" {
					if err := clone.ChangeOwner(card.ID(), a.PlayerID()); err != nil {
						t.Fatal(err)
					}
				}
			case "control":
				clone.AddControlEffect(ControlEffectSpec{SourceID: spell.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phase cycle":
				clone.PhaseIn(clone.PhaseOut(card.ID()))
			case "return":
				clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
				clone.PutOnBattlefield(card, a.PlayerID())
			}
			clone.ResolveStack()
			want := 1
			if change == "owner" || change == "return" {
				want = 0
			}
			if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != want {
				t.Fatalf("hand = %d, want %d", got, want)
			}
			if g.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()).Card.Owner() != a.PlayerID() || len(a.Hand()) != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestTypedTargetYouOwnCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn())
	ability := Spell(ReturnToHand(target), WithTargets(target))
	card := NewInstant("Return", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != second.ID() || obj.Targets[0] != first.ID() {
		t.Fatal("copy did not use its controller for ownership")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("copies did not return their respective owned targets")
	}
}

func TestTypedTargetYouOwnActivation(t *testing.T) {
	for _, scenario := range []string{"paid", "not owned", "not controlled", "no mana", "control lost", "source exit"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn(), TargetYouControl())
			source := NewArtifact("Return source", "", WithAbility(Activate(Costs(ManaPayment("{6}"), TapCost(This)), ReturnToHand(target), WithTargets(target))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			card := NewCreature("Owned", "", 2, 2)
			owner, controller := a.PlayerID(), a.PlayerID()
			if scenario == "not owned" {
				owner = b.PlayerID()
			}
			if scenario == "not controlled" {
				controller = b.PlayerID()
			}
			card.SetOwner(owner)
			g.PutOnBattlefield(card, controller)
			mana := 6
			if scenario == "no mana" {
				mana = 0
			}
			a.ManaPool().Add(Colorless, mana)
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{card.ID()})
			if scenario == "not owned" || scenario == "not controlled" || scenario == "no mana" {
				if err == nil || a.ManaPool().TotalMana() != mana || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("invalid activation succeeded or paid costs")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 1 || g.FindPermanent(card.ID()) == nil {
				t.Fatal("incorrect payment or stack timing")
			}
			if scenario == "control lost" {
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			if scenario == "source exit" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			want := 1
			if scenario == "control lost" {
				want = 0
			}
			if len(a.Hand()) != want {
				t.Fatalf("hand = %d, want %d", len(a.Hand()), want)
			}
		})
	}
}

// --- from composable_target_test.go ---
func TestTargetIncarnationAfterReturn(t *testing.T) {
	for _, copySpell := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		c := NewCreature("Target", "", 2, 2)
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
		obj := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, TargetSource: c, Targets: []uuid.UUID{c.ID()}, TargetSpecs: []Target{TargetCreature()}, Effects: []Effect{DestroyTarget()}}
		g.pushStack(obj)
		g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
		g.PutOnBattlefield(c, b.PlayerID())
		if copySpell {
			obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
		}
		g.ResolveStackObject(obj)
		if g.FindPermanent(c.ID()) == nil {
			t.Fatalf("returned object destroyed (copy=%v)", copySpell)
		}
	}
}

func TestTypedTargetGroups(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Target", "", 2, 2)
	c.SetOwner(b.PlayerID())
	g.PutOnBattlefield(c, b.PlayerID())
	creature := DeclarePermanentTarget("creature", IsCreature)
	player := DeclarePlayerTarget("player")
	ability := Spell(Sequence(Destroy(creature), DrawFor(player, Fixed(1))), WithTargets(creature, player))
	card := NewSorcery("Test", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Test", card); err != nil {
		t.Fatal(err)
	}
	drawn := NewCreature("Drawn", "", 1, 1)
	b.AddToLibrary(drawn)
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID(), b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	g.ResolveStackObject(obj)
	if len(b.Hand()) != 1 {
		t.Fatal("legal player binding lost after creature became illegal")
	}
	if len(a.Hand()) != 0 {
		t.Fatal("player binding fell back to controller")
	}
}

func TestTypedTargetValidation(t *testing.T) {
	target := DeclarePermanentTarget("target", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "undeclared",
			Card:    NewSorcery("Validation", "", WithAbility(Spell(Destroy(target)))),
			WantErr: true,
		},
		{
			Name:    "duplicate name",
			Card:    NewSorcery("Validation", "", WithAbility(Spell(Destroy(target), WithTargets(target, DeclarePlayerTarget("target"))))),
			WantErr: true,
		},
		{
			Name:    "duplicate declaration",
			Card:    NewSorcery("Validation", "", WithAbility(Spell(Destroy(target), WithTargets(target, target)))),
			WantErr: true,
		},
		{
			Name:    "nil amount",
			Card:    NewSorcery("Validation", "", WithAbility(Spell(DrawFor(You, nil)))),
			WantErr: true,
		},
		{
			Name: "valid",
			Card: NewSorcery("Validation", "", WithAbility(Spell(Destroy(target), WithTargets(target)))),
		},
	})
}

func TestTypedTargetCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Destroy(target), WithTargets(target))
	card := NewSorcery("Spell", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	copied := clone.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	clone.ResolveStackObject(copied)
	if clone.FindPermanent(victim.ID()) != nil {
		t.Fatal("copy lost target binding")
	}
	if g.FindPermanent(victim.ID()) == nil {
		t.Fatal("clone changed original")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("original lost binding after clone resolution")
	}
}

func TestTypedTargetCannotEscapeDelayedScope(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Spell(RegisterDelayedTriggerStep(EvtTapped, "", Destroy(target)), WithTargets(target))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "delayed trigger read outer target binding",
			Card:    NewSorcery("Delayed", "", WithAbility(ability)),
			WantErr: true,
		},
	})
}

func TestComposedRemovalMetadata(t *testing.T) {
	effect := Sequence(Destroy(This), DrawFor(You, Fixed(2)))
	m := InspectDefinition(effect)
	if len(m.Children) != 2 || !m.Children[0].Properties.Destroys || m.Children[1].Properties.DrawCount != 2 || !m.Properties.Destroys {
		t.Fatalf("metadata lost removal: %+v", m)
	}
}

func TestTypedTargetCollectionPartialLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	creatures := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	player := DeclarePlayerTarget("player")
	ability := Spell(Sequence(DestroyEach(creatures), DrawFor(player, Fixed(1))), WithTargets(creatures, player))
	card := NewSorcery("Collection", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	b.AddToLibrary(NewLand("Land"))
	targets := []uuid.UUID{first.ID(), second.ID(), b.PlayerID()}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: targets, TargetSpecs: expandTargetSpecs(ability.Targets(), targets, 0), Effects: ability.Effects()}
	g.pushStack(obj)
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, b.PlayerID())
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || len(b.Hand()) != 1 {
		t.Fatal("collection failed to retain its legal members and adjacent player group")
	}
}

func TestCopyCannotChangeTargetCount(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclarePermanentTargets("creatures", 0, 2, IsCreature)
	ability := Spell(DestroyEach(targets), WithTargets(targets))
	spell := NewSorcery("Copy test", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	c := NewCreature("Creature", "", 2, 2)
	c.SetOwner(b.PlayerID())
	g.PutOnBattlefield(c, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: expandTargetSpecs(ability.Targets(), []uuid.UUID{c.ID()}, 0), Effects: ability.Effects()}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{}}
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if len(copied.Targets) != 1 || copied.Targets[0] != c.ID() {
		t.Fatal("copy changed the number of targets")
	}
}

func TestCopyRetargetsDividedValues(t *testing.T) {
	g, a, b := newSpellCopyGame()
	spell := NewInstant("Divided", "", NewTargetedSpell(TargetPlayer(), DealDamage(Fixed(3))))
	spell.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, DamageDistribution: map[uuid.UUID]int{b.PlayerID(): 3}, CounterDistribution: map[uuid.UUID]int{b.PlayerID(): 2}}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	cp := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if cp.DamageDistribution[a.PlayerID()] != 3 || cp.CounterDistribution[a.PlayerID()] != 2 {
		t.Fatal("retargeting lost divided amounts")
	}
	if obj.DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("copy mutated original distribution")
	}
}

func TestTypedVariableGroupsRequireBoundaries(t *testing.T) {
	first := DeclarePermanentTargets("first", 0, 2, IsCreature)
	second := DeclarePermanentTargets("second", 0, 2, IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "ambiguous variable target groups",
			Card:    NewSorcery("Ambiguous", "", WithAbility(Spell(DestroyEach(first), WithTargets(first, second)))),
			WantErr: true,
		},
	})
}

// --- from composable_target_untap_test.go ---
func TestComposableTargetUntapPartialAndLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle", "absent"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, c := range []Card{first, second} {
				g.PutOnBattlefield(c, b.PlayerID())
				g.TapPermanent(g.FindPermanent(c.ID()))
			}
			x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
			ability := Spell(Sequence(UntapPermanent(x), UntapPermanent(y)), WithTargets(x, y))
			card := NewInstant("Untap", "", WithAbility(ability))
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
				g.TapPermanent(g.FindPermanent(first.ID()))
			case "phase":
				g.PhaseOut(first.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
			}
			clone := g.Clone()
			clone.ResolveStack()
			if change == "phase" {
				clone.PhaseIn([]uuid.UUID{first.ID()})
			}
			wantTapped := change == "return" || change == "phase" || change == "absent"
			if (clone.FindPermanent(first.ID()) != nil && clone.FindPermanent(first.ID()).Tapped != wantTapped) || clone.FindPermanent(second.ID()).Tapped {
				t.Fatalf("incorrect target binding or legality: first=%+v second tapped=%v", clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID()).Tapped)
			}
			if !g.FindPermanent(second.ID()).Tapped {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestComposableRandomUntapBinding(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, a.PlayerID())
		g.TapPermanent(g.FindPermanent(c.ID()))
	}
	saved := BindPermanent("random")
	body := Sequence(ChooseRandomPermanent(MatchingPermanents(And(IsCreature, IsTapped)), saved), UntapPermanent(From(saved)))
	card := NewSorcery("Random", "", WithAbility(Spell(body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.SetRandomResults([]int{1, 0, 7})
	clone := g.Clone()
	if err := ApplyEffect(clone, body, card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(second.ID()).Tapped || !clone.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("random choice or clone isolation failed")
	}
	for range 3 {
		if err := ApplyEffect(g, body, card.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped || g.RandIntn(10) != 7 {
		t.Fatal("empty choice consumed randomness or reused a binding")
	}
}

func TestComposableLegacyTriggerTypedUntap(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	trigger := NewTriggered(EvtUpkeep, false, UntapPermanent(target)).WithTargets(target)
	card := NewCreature("Trigger", "", 1, 1, WithAbility(trigger))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(card, a.PlayerID())
	g.TapPermanent(g.FindPermanent(card.ID()))
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	resolvePendingTriggers(g)
	if g.FindPermanent(card.ID()).Tapped {
		t.Fatal("trigger lost typed target")
	}
}

func TestComposableTargetUntapCopyRetarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, b.PlayerID())
		g.TapPermanent(g.FindPermanent(c.ID()))
	}
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Spell(UntapPermanent(target), WithTargets(target))
	card := NewInstant("Untap", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{second.ID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	g.ResolveStackObject(copied)
	if g.FindPermanent(second.ID()).Tapped || !g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy used original binding")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()).Tapped {
		t.Fatal("copy changed original binding")
	}
}

func TestComposableRandomUntapValidationAndAbsence(t *testing.T) {
	saved := BindPermanent("chosen")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "unbound read before choose",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(Sequence(UntapPermanent(From(saved)), ChooseRandomPermanent(MatchingPermanents(IsCreature), saved))))),
			WantErr: true,
		},
		{
			Name:    "nil filter set",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(Sequence(ChooseRandomPermanent(nil, saved), UntapPermanent(From(saved)))))),
			WantErr: true,
		},
		{
			Name:    "nil binding",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(ChooseRandomPermanent(MatchingPermanents(IsCreature), nil)))),
			WantErr: true,
		},
		{
			Name:    "undeclared target",
			Card:    NewSorcery("Invalid", "", WithAbility(Spell(UntapPermanent(DeclarePermanentTarget("undeclared", IsCreature))))),
			WantErr: true,
		},
	})
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	g.TapPermanent(g.FindPermanent(c.ID()))
	body := Sequence(Snapshot(This, saved), ChooseRandomPermanent(MatchingPermanents(IsLand), saved), UntapPermanent(From(saved)))
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(c.ID()).Tapped {
		t.Fatal("empty random choice retained previous binding")
	}
}

// --- from composable_unique_highest_life_test.go ---
func TestUniqueHighestLifeStateAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := HasUniqueHighestLife()
	ctx := StateContext{ControllerID: a.PlayerID()}
	if condition.checkState(g, ctx) {
		t.Fatal("tied life totals have no unique leader")
	}
	g.PlayerGainLife(a, 1)
	if !condition.checkState(g, ctx) || !condition.checkState(g, StateContext{ControllerID: b.PlayerID()}) {
		t.Fatal("unique leader depended on the ability controller")
	}
	clone := g.Clone()
	clone.PlayerGainLife(clone.GetPlayer(b.PlayerID()), 2)
	if !condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone did not read independent life totals")
	}
	clone.PlayerLoseLife(clone.GetPlayer(b.PlayerID()), 1)
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("tie or clone changed original state")
	}
}

func TestUniqueHighestLifeInterveningIfAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := HasUniqueHighestLife()
	source := NewCreature("Life watcher", "", 2, 2, WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(condition); got.Role != "state-condition" || got.Text != "one player has the highest life total" {
		t.Fatalf("inspection = %+v", got)
	}
}

// --- from composable_value_test.go ---
func TestComposablePermanentValueCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("subject")
	effect := Sequence(Snapshot(This, saved), Until(EndOfTurn, ModifyPT(From(saved), PowerOf(From(saved)), ToughnessOf(From(saved)))))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	p := g.FindPermanent(c.ID())
	if p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 6 {
		t.Fatal("values were not captured once")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(c.ID()))
	clone.PutOnBattlefield(c, a.PlayerID())
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(c.ID()).CurrentPower(clone) != 2 || p.CurrentPower(g) != 4 {
		t.Fatal("capture followed a returned card or changed original")
	}
}

func TestComposablePermanentValueValidation(t *testing.T) {
	saved := BindPermanent("missing")
	effect := Until(EndOfTurn, ModifyPT(This, PowerOf(From(saved)), Fixed(0)))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:          "unbound producer",
			Card:          NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect))),
			WantErr:       true,
			WantErrSubstr: "producer",
		},
	})
}

func TestComposablePermanentValueLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle", "absent", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", -2, 3)
			g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
				g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			}
			wantPower, wantToughness := -2, 3
			if change == "return" || change == "phase" || change == "absent" {
				wantPower, wantToughness = 0, 0
			}
			if got := PowerOf(From(saved)).resolveValue(ctx); got != wantPower {
				t.Fatalf("power = %d, want %d", got, wantPower)
			}
			if got := ToughnessOf(From(saved)).resolveValue(ctx); got != wantToughness {
				t.Fatalf("toughness = %d, want %d", got, wantToughness)
			}
		})
	}
}

func TestComposablePermanentValueTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 3), NewCreature("Second", "", 4, 5)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
	ability := Spell(Until(EndOfTurn, ModifyPT(x, PowerOf(y), ToughnessOf(y))), WithTargets(x, y))
	card := NewInstant("Read second", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveStack()
	p := clone.FindPermanent(first.ID())
	if p.CurrentPower(clone) != 6 || p.CurrentToughness(clone) != 8 {
		t.Fatal("value did not read the second target")
	}
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 {
		t.Fatal("clone changed original")
	}
}

func TestComposablePermanentValueInvalidDefinitions(t *testing.T) {
	target := DeclarePermanentTarget("undeclared", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "nil PowerOf",
			Card:    NewCreature("nil PowerOf", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, PowerOf(nil), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "nil ToughnessOf",
			Card:    NewCreature("nil ToughnessOf", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, ToughnessOf(nil), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "undeclared target PowerOf",
			Card:    NewCreature("undeclared target PowerOf", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, PowerOf(target), Fixed(0)))))),
			WantErr: true,
		},
		{
			Name:    "nil ValueAtResolution",
			Card:    NewCreature("nil ValueAtResolution", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, ValueAtResolution(nil), Fixed(0)))))),
			WantErr: true,
		},
	})
}

func TestComposablePermanentValueStaticEvaluation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 2, 3, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, PowerOf(This), Fixed(0))))))
	g.PutOnBattlefield(c, a.PlayerID())
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 4 {
		t.Fatalf("power = %d, want 4", got)
	}
	g.FindPermanent(c.ID()).Counters[P1P1] = 1
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 6 {
		t.Fatalf("power = %d, want 6", got)
	}
}

// --- from composable_x_targets_test.go ---
func TestTypedXTargetsSpellAnnouncementAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXPermanentTargets("creatures", IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("X tap", "{X}", WithAbility(Spell(ForEach(targets, member, TapSubject(From(member))), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	for _, chosen := range [][]uuid.UUID{{first.ID()}, {first.ID(), first.ID()}, {first.ID(), second.ID(), uuid.New()}} {
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), chosen, 2); err == nil {
			t.Fatalf("accepted invalid X targets %v", chosen)
		}
		if g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("invalid announcement changed state")
		}
	}
	a.ManaPool().Add(0, 4)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(second.ID())
	clone.ResolveStackObject(copyObj)
	if !clone.FindPermanent(first.ID()).Tapped || clone.FindPermanentIncludingPhased(second.ID()).Tapped {
		t.Fatal("copy lost legal target or affected phased target")
	}
	if first.Tapped || second.Tapped || len(original.TargetSpecs) != 2 {
		t.Fatal("clone changed original or lost target count")
	}
	g.ResolveStackObject(original)
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("X targets did not resolve")
	}
}

func TestTypedXTargetsZeroAndReturnedTarget(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXPermanentTargets("creatures", IsCreature)
	member := BindPermanent("member")
	spell := NewInstant("X tap", "{X}", WithAbility(Spell(ForEach(targets, member, TapSubject(From(member))), WithTargets(targets))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	second := NewInstant("X tap", "{X}", WithAbility(Spell(ForEach(targets, member, TapSubject(From(member))), WithTargets(targets))))
	second.SetOwner(a.PlayerID())
	a.AddToHand(second)
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 2), b.PlayerID())
	a.ManaPool().Add(0, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{old.ID()}, 1); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(old.ID()))
	g.PutOnBattlefield(old.Card, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(old.ID()).Tapped {
		t.Fatal("target followed a returned incarnation")
	}
}

func TestTypedXTargetsActivationAndUpToX(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareUpToXPermanentTargets("lands", IsLand)
	member := BindPermanent("member")
	source := g.PutOnBattlefield(NewArtifact("X untap", "", WithAbility(Activate(Costs(ManaPayment("{X}"), TapCost(This)),
		ForEach(targets, member, UntapPermanent(From(member))), WithTargets(targets)))), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source.Card); err != nil {
		t.Fatal(err)
	}
	land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	g.TapPermanent(land)
	g.SetXValue(1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID(), uuid.New()}); err == nil {
		t.Fatal("accepted more than X targets")
	}
	if source.Tapped || !land.Tapped || g.StackSize() != 0 {
		t.Fatal("invalid activation changed state")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err == nil {
		t.Fatal("accepted unpaid X activation")
	}
	if source.Tapped || !land.Tapped || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(0, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if land.Tapped || !source.Tapped {
		t.Fatal("activation lost announced target")
	}
}

func TestTypedXTargetsDeclaration(t *testing.T) {
	targets := DeclareXPermanentTargets("creatures", IsCreature)
	unnamed := DeclareXPermanentTargets("", IsCreature)
	first := DeclareUpToXPermanentTargets("first", IsCreature)
	second := DeclareUpToXPermanentTargets("second", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{
			Name:    "undeclared X targets",
			Card:    NewSorcery("Missing", "", WithAbility(Spell(DestroyEach(targets)))),
			WantErr: true,
		},
		{
			Name:    "unnamed X targets",
			Card:    NewSorcery("Unnamed", "", WithAbility(Spell(DestroyEach(unnamed), WithTargets(unnamed)))),
			WantErr: true,
		},
		{
			Name: "grouped X targets",
			Card: NewInstant("Grouped", "{X}", WithAbility(Spell(DestroyEach(first), WithTargetGroups(first, second)))),
		},
	})
}
