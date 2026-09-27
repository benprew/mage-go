package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

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
						g.MutablePermanent(host.ID()).GrantBaseAttr(core.AttrCantBeEnchanted)
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
	if err := ValidateCardDefinition("Good", NewCreature("Good", "", 1, 1, WithAbility(Activate(Costs(), good)))); err != nil {
		t.Fatal(err)
	}
	for _, body := range []ResolvingEffect{MayAttach(nil, This), MayAttach(This, nil), MayAttach(From(aura), This), MayAttach(This, From(host))} {
		if err := ValidateCardDefinition("Bad", NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), body)))); err == nil {
			t.Fatal("accepted invalid attachment declaration")
		}
	}
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
	cat := DefineToken("Cat", TokenTypes(core.TypeCreature), TokenSubtypes("Cat"), TokenColors(core.White), TokenPT(2, 2))
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
