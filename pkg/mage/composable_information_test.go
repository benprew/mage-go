package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentInformationLifecycle(t *testing.T) {
	for _, reference := range []string{"target", "saved", "source", "attachment"} {
		for _, scenario := range []string{"live", "control", "leave", "return", "second-departure", "phase", "phase-cycle", "clear-lki", "unknown"} {
			t.Run(reference+"/"+scenario, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Information", "{3}{G}", 2, 8)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				p.AddCounter(P1P1, 3)
				target := DeclarePermanentTarget("target", AnyPermanent)
				saved := BindPermanent("saved")
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				if err := Snapshot(target, saved).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				var subject PermanentRef = target
				switch reference {
				case "saved":
					subject = From(saved)
				case "source":
					subject = This
				case "attachment":
					aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
					g.Attach(aura.ID(), p.ID())
					ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
					subject = AttachedPermanent
				}
				switch scenario {
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
				case "leave", "return", "second-departure", "clear-lki":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if scenario == "return" || scenario == "second-departure" {
						a.RemoveFromHand(card.ID())
						q := g.PutOnBattlefield(card, a.PlayerID())
						q.AddCounter(P1P1, 9)
						if scenario == "second-departure" {
							g.BouncePermanentToHand(q)
						}
					}
					if scenario == "clear-lki" {
						g.zones.ClearLKI()
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "unknown":
					ctx.targetBindings[target.targetDeclaration][0].incarnation = uuid.New()
					g.resolution.permanentBindings[saved] = capturedPermanent{p.ID(), uuid.New()}
					ctx.sourceIncarnation = uuid.New()
				}
				wantPlayer := b.PlayerID()
				if scenario == "control" {
					wantPlayer = a.PlayerID()
				}
				wantPower, wantMana := 5, 4
				if scenario == "unknown" {
					wantPower, wantMana = 0, 0
				}
				for _, game := range []*Game{g, g.Clone()} {
					local := *ctx
					local.Game = game
					who, ok := ControllerOfWithLKI(subject).resolvePlayer(&local)
					if scenario == "unknown" {
						if ok {
							t.Fatal("unknown incarnation has controller")
						}
					} else if !ok || who != wantPlayer {
						t.Fatal("wrong information controller")
					}
					if got := PowerOfWithLKI(subject).resolveValue(&local); got != wantPower {
						t.Fatalf("power=%d want=%d", got, wantPower)
					}
					if got := ManaValueOfWithLKI(subject).resolveValue(&local); got != wantMana {
						t.Fatalf("mana=%d want=%d", got, wantMana)
					}
					if scenario == "leave" || scenario == "return" || scenario == "second-departure" || scenario == "phase" {
						if _, ok := subject.resolvePermanent(&local); ok {
							t.Fatal("information read made mutation reference live")
						}
					}
				}
			})
		}
	}
}

func TestPermanentInformationDeclaration(t *testing.T) {
	for _, effect := range []ResolvingEffect{GainLifeFor(ControllerOfWithLKI(nil), Fixed(1)), GainLifeFor(You, PowerOfWithLKI(nil)), GainLifeFor(You, ManaValueOfWithLKI(nil))} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("nil information subject accepted")
		}
	}
	target := DeclarePermanentTarget("target", AnyPermanent)
	card := NewInstant("Invalid", "", Spell(GainLifeFor(ControllerOfWithLKI(target), PowerOfWithLKI(target))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("undeclared information target accepted")
	}
}

func TestPermanentInformationPhaseSnapshotAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Host", "{2}{G}", 2, 8), b.PlayerID())
	aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura"), WithStaticAbility(BoostAttached(3, 0, AttachAura))), a.PlayerID())
	g.Attach(aura.ID(), p.ID())
	g.ApplyContinuousEffects()
	target := DeclarePermanentTarget("host", AnyPermanent)
	ctx := &EffectContext{Game: g, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
	phased := g.PhaseOut(p.ID())
	if len(phased) != 2 {
		t.Fatal("attachment did not phase with host")
	}
	clone := g.Clone()
	g.PhaseIn(phased)
	g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
	if PowerOfWithLKI(target).resolveValue(ctx) != 2 {
		t.Fatal("phase-in did not restore current power")
	}
	local := *ctx
	local.Game = clone
	if !clone.FindPermanentIncludingPhased(aura.ID()).PhasedOut {
		t.Fatal("phase-in changed the clone's attachment")
	}
	if PowerOfWithLKI(target).resolveValue(&local) != 5 {
		t.Fatal("phase snapshot used inactive layers or shared clone state")
	}
	if _, ok := target.resolvePermanent(&local); ok {
		t.Fatal("phased target made live")
	}
}

func TestPermanentInformationAttachmentDepartureAndUnboundSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	hostCard := NewCreature("Host", "{3}{G}", 4, 8)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	auraCard := NewEnchantment("Aura", "", WithSubTypes("Aura"))
	auraCard.SetOwner(a.PlayerID())
	aura := g.PutOnBattlefield(auraCard, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	ctx := &EffectContext{Game: g, SourceID: aura.ID(), sourceCaptured: true, sourceIncarnation: aura.incarnationID}
	g.BouncePermanentToHand(aura)
	g.BouncePermanentToHand(host)
	b.RemoveFromHand(hostCard.ID())
	g.PutOnBattlefield(hostCard, a.PlayerID())
	a.RemoveFromHand(auraCard.ID())
	g.PutOnBattlefield(auraCard, b.PlayerID())
	who, ok := ControllerOfWithLKI(AttachedPermanent).resolvePlayer(ctx)
	if !ok || who != b.PlayerID() || PowerOfWithLKI(AttachedPermanent).resolveValue(ctx) != 4 {
		t.Fatal("old attachment followed new source or host")
	}
	ctx.sourceCaptured = false
	g.BouncePermanentToHand(g.FindPermanent(auraCard.ID()))
	if _, ok := ControllerOfWithLKI(This).resolvePlayer(ctx); ok {
		t.Fatal("uncaptured source used old incarnation")
	}
	if PowerOfWithLKI(This).resolveValue(ctx) != 0 {
		t.Fatal("unknown source supplied power")
	}
}

func TestPermanentInformationCloneAfterPhaseIn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Host", "", 2, 8), b.PlayerID())
	target := DeclarePermanentTarget("host", AnyPermanent)
	ctx := &EffectContext{Game: g, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
	g.PhaseIn(g.PhaseOut(p.ID()))
	clone := g.Clone()
	g.MutablePermanent(p.ID()).AddCounter(P1P1, 3)
	g.PhaseOut(p.ID())
	clone.PhaseOut(p.ID())
	local := *ctx
	local.Game = clone
	if PowerOfWithLKI(target).resolveValue(ctx) != 5 || PowerOfWithLKI(target).resolveValue(&local) != 2 {
		t.Fatal("empty phase map was shared by clones")
	}
}
