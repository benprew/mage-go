package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentOwnerInformation(t *testing.T) {
	for _, reference := range []string{"source", "target", "saved", "attachment"} {
		for _, state := range []string{"live", "control", "ownership", "leave", "return", "phase", "phase-cycle"} {
			t.Run(reference+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g, err := NewGameWithAnte(a, b, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				card := NewCreature("Subject", "", 1, 8)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				target := DeclarePermanentTarget("subject", AnyPermanent)
				ctx := &EffectContext{Game: g, Controller: b.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				var ref PermanentRef = This
				switch reference {
				case "target":
					ref = target
				case "saved":
					binding := BindPermanent("saved")
					if err := Snapshot(target, binding).Apply(ctx); err != nil {
						t.Fatal(err)
					}
					ref = From(binding)
				case "attachment":
					aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), b.PlayerID())
					g.Attach(aura.ID(), p.ID())
					ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
					ref = AttachedPermanent
				}
				want := a.PlayerID()
				switch state {
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "ownership":
					if err := g.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
						t.Fatal(err)
					}
					want = b.PlayerID()
				case "leave", "return":
					g.BouncePermanentToHand(p)
					if state == "return" {
						a.RemoveFromHand(card.ID())
						card.SetOwner(b.PlayerID())
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				for _, game := range []*Game{g, g.Clone()} {
					local := *ctx
					local.Game = game
					got, ok := OwnerOfWithLKI(ref).resolvePlayer(&local)
					if !ok || got != want {
						t.Fatalf("owner=%v ok=%v want=%v", got, ok, want)
					}
					recipient, ok := OwnerOfWithLKI(ref).resolveRecipient(&local)
					if !ok || recipient != want {
						t.Fatal("recipient lost owner")
					}
					if state == "leave" || state == "return" || state == "phase" {
						if _, ok := ref.resolvePermanent(&local); ok {
							t.Fatal("information read made absent visit available")
						}
					}
				}
			})
		}
	}
}

func TestPermanentOwnerInformationInvalid(t *testing.T) {
	target := DeclarePermanentTarget("unbound", AnyPermanent)
	for _, ref := range []PermanentRef{nil, target, From(BindPermanent("unbound"))} {
		c := NewInstant("Invalid", "", Spell(GainLifeFor(OwnerOfWithLKI(ref), Fixed(1))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid owner read accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	ctx := &EffectContext{Game: NewGame(a, b)}
	if _, ok := OwnerOfWithLKI(This).resolvePlayer(ctx); ok {
		t.Fatal("unknown source supplied owner")
	}
}

func TestPermanentOwnerInformationCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	card := NewArtifact("Subject", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true}
	clone := g.Clone()
	if err := g.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if owner, ok := OwnerOfWithLKI(This).resolvePlayer(ctx); !ok || owner != b.PlayerID() {
		t.Fatal("live read lost ownership change")
	}
	local := *ctx
	local.Game = clone
	if owner, ok := OwnerOfWithLKI(This).resolvePlayer(&local); !ok || owner != a.PlayerID() {
		t.Fatal("ownership change reached clone")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	clone.PhaseOut(p.ID())
	if owner, ok := OwnerOfWithLKI(This).resolvePlayer(ctx); !ok || owner != b.PlayerID() {
		t.Fatal("departure lost current owner")
	}
	if owner, ok := OwnerOfWithLKI(This).resolvePlayer(&local); !ok || owner != a.PlayerID() {
		t.Fatal("phase information lost clone owner")
	}
}
