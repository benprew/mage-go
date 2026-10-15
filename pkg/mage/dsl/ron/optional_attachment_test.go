package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronOptionalAttachmentPlayer struct {
	*ronPermanentChoicePlayer
	accept bool
	offers int
}

func (p *ronOptionalAttachmentPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONOptionalAttachmentKudzu(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"accept", "decline", "no-host", "shroud", "hexproof", "protection", "indestructible", "same-host", "regenerate", "reattach", "host-control", "aura-control", "host-return", "host-phase", "phase-cycle", "source-left", "source-return", "source-phase", "copy", "clone", "countered"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				factory, err := Compile(chosenDiscardRecord(t, set, "Kudzu"))
				if err != nil {
					t.Fatal(err)
				}
				g, baseA, baseB := ronGame()
				a := &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: baseA}, accept: state != "decline"}
				b := &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: baseB}, accept: state != "decline"}
				g.SetPlayerAt(0, a)
				g.SetPlayerAt(1, b)
				hostCard := mage.NewLand("Host")
				hostCard.SetOwner(b.PlayerID())
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				next := g.PutOnBattlefield(mage.NewLand("Next"), a.PlayerID())
				last := g.PutOnBattlefield(mage.NewLand("Last"), b.PlayerID())
				choose := func(candidates []*mage.Permanent) *mage.Permanent {
					for _, p := range candidates {
						if p.ID() == next.ID() {
							return p
						}
					}
					for _, p := range candidates {
						if p.ID() == last.ID() {
							return p
						}
					}
					t.Fatal("legal host missing")
					return nil
				}
				a.choose = choose
				b.choose = choose
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
					t.Fatal("unpaid Aura cast")
				}
				a.ManaPool().Add(core.Green, 2)
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				aura := g.FindPermanent(card.ID())
				if aura == nil || aura.AttachedTo != host.ID() {
					t.Fatal("Aura entry")
				}
				if state == "shroud" {
					next.GrantBaseAttr(core.Shroud)
				}
				if state == "hexproof" {
					next.GrantBaseAttr(core.Hexproof)
				}
				if state == "indestructible" || state == "same-host" {
					host.GrantBaseAttr(core.Indestructible)
				}
				if state == "regenerate" {
					g.AddRegenerationShield(host.ID())
				}
				if state == "protection" {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantProtectionColors(mage.This, core.Green)), next.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					a.choose = func([]*mage.Permanent) *mage.Permanent { return last }
					b.choose = a.choose
				}
				g.TapPermanent(host)
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || a.offers+b.offers != 0 {
					t.Fatal("trigger placement or early choice")
				}
				switch state {
				case "same-host":
					g.RemoveFromBattlefield(next)
					g.RemoveFromBattlefield(last)
				case "no-host":
					g.RemoveFromBattlefield(next)
					g.RemoveFromBattlefield(last)
				case "reattach":
					g.Attach(aura.ID(), last.ID())
				case "host-control":
					g.AddControlEffect(mage.ControlEffectSpec{TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "aura-control":
					g.AddControlEffect(mage.ControlEffectSpec{TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "host-return":
					g.RemoveFromBattlefield(host)
					g.PutOnBattlefield(hostCard, b.PlayerID())
				case "host-phase":
					g.PhaseOut(host.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "source-left", "source-return":
					g.RemoveFromBattlefield(aura)
					if state == "source-return" {
						p := g.PutOnBattlefield(card, a.PlayerID())
						g.Attach(p.ID(), last.ID())
					}
				case "source-phase":
					g.PhaseOut(aura.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "clone":
					clone := g.Clone()
					g.DestroyPermanent(g.FindPermanent(aura.ID()))
					g = clone
					a = &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: g.GetPlayer(baseA.PlayerID()), choose: choose}, accept: true}
					b = &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: g.GetPlayer(baseB.PlayerID()), choose: choose}, accept: true}
					g.SetPlayerAt(0, a)
					g.SetPlayerAt(1, b)
				case "countered":
					g.CounterStackObject(g.StackPeek().ID)
				}
				g.ResolveStack()
				survives := state == "same-host" || state == "indestructible" || state == "regenerate" || state == "host-return" || state == "host-phase" || state == "countered"
				if (g.FindPermanentIncludingPhased(host.ID()) != nil) != survives {
					t.Fatal("event host destruction identity")
				}
				present := g.FindPermanentIncludingPhased(aura.ID())
				switch state {
				case "decline", "no-host", "source-left":
					if present != nil {
						t.Fatal("unattached Aura survived final state actions")
					}
				case "source-return":
					if present == nil || present.AttachedTo != last.ID() {
						t.Fatal("returned Aura inherited move")
					}
				case "same-host", "host-phase", "source-phase", "countered":
					if present == nil || present.AttachedTo != host.ID() {
						t.Fatal("suspended attachment moved")
					}
				default:
					want := next.ID()
					if state == "protection" {
						want = last.ID()
					}
					if present == nil || present.AttachedTo != want {
						t.Fatal("optional attachment result")
					}
				}
				wantOffers := 1
				if state == "no-host" || state == "host-phase" || state == "source-phase" || state == "source-left" || state == "source-return" || state == "countered" {
					wantOffers = 0
				}
				if state == "copy" {
					wantOffers = 2
				}
				if a.offers+b.offers != wantOffers {
					t.Fatalf("offers %d/%d, want total %d", a.offers, b.offers, wantOffers)
				}
				if state == "host-control" {
					if a.offers != 1 {
						t.Fatal("did not read host controller")
					}
				} else if a.offers != 0 {
					t.Fatal("Aura or trigger controller made host choice")
				}
			})
		}
	}
}

func TestRONOptionalAttachmentInvalid(t *testing.T) {
	good := `PlayerMayAction(You,AttachPermanentToAPermanent(ThisPermanent,AnyPermanent))`
	for _, body := range []string{
		`PlayerMayAction`, `PlayerMayAction(You)`, `PlayerMayAction(You,AttachPermanentToAPermanent(ThisPermanent,AnyPermanent),DrawACard)`,
		`PlayerMayAction(ThisPermanent,AttachPermanentToAPermanent(ThisPermanent,AnyPermanent))`,
		`PlayerMayAction(Trigger_ThatPlayer,AttachPermanentToAPermanent(ThisPermanent,AnyPermanent))`,
		`PlayerMayAction(You,DrawACard)`, `PlayerMayAction(You,AttachPermanentToAPermanent(You,AnyPermanent))`,
		`PlayerMayAction(You,AttachPermanentToAPermanent(TheChosenPermanent,AnyPermanent))`,
		`PlayerMayAction(You,AttachPermanentToAPermanent(ThisPermanent,You))`,
		good + `,TapPermanent(TheChosenPermanent)`,
	} {
		assertConditionalRejected(t, `Activated(PayMana([]),ActionList([`+body+`]))`)
	}
	for _, rule := range []string{`Activated(` + good + `,ActionList([]))`, `PermanentLayerEffect(ThisPermanent,[` + good + `])`, `AsPermanentEnters(ThisPermanent,[` + good + `])`} {
		assertConditionalRejected(t, rule)
	}
	for _, ctx := range []scope{{permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
		nodes, err := Parse(strings.NewReader(good), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestRONOptionalAttachmentInstances(t *testing.T) {
	g, baseA, baseB := ronGame()
	a := &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: baseA}, accept: true}
	b := &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: baseB}, accept: true}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	first := g.PutOnBattlefield(mage.NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewLand("Second"), b.PlayerID())
	nextA := g.PutOnBattlefield(mage.NewLand("A destination"), b.PlayerID())
	nextB := g.PutOnBattlefield(mage.NewLand("B destination"), a.PlayerID())
	a.choose = func(candidates []*mage.Permanent) *mage.Permanent {
		for _, p := range candidates {
			if p.ID() == nextA.ID() {
				return p
			}
		}
		t.Fatal("A destination absent")
		return nil
	}
	b.choose = func(candidates []*mage.Permanent) *mage.Permanent {
		for _, p := range candidates {
			if p.ID() == nextB.ID() {
				return p
			}
		}
		t.Fatal("B destination absent")
		return nil
	}
	factory, err := Compile(chosenDiscardRecord(t, "LEA", "Kudzu"))
	if err != nil {
		t.Fatal(err)
	}
	auraA := g.PutOnBattlefield(factory(), b.PlayerID())
	g.Attach(auraA.ID(), first.ID())
	auraB := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(auraB.ID(), second.ID())
	g.TapPermanent(first)
	g.TapPermanent(second)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent triggers missing")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || g.FindPermanent(auraA.ID()).AttachedTo != nextA.ID() || g.FindPermanent(auraB.ID()).AttachedTo != nextB.ID() || a.offers != 1 || b.offers != 1 {
		t.Fatal("independent source, host or chooser lost")
	}
}

func TestRONOptionalAttachmentDelayedScope(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsEnchantmentType(Aura))],ActionList([PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerMayAction(ActionPlayer,AttachPermanentToAPermanent(Ref_TargetPermanent,IsCardtype(Creature)))]))])])))`)
	g, baseA, baseB := ronGame()
	a := &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: baseA}, accept: true}
	b := &ronOptionalAttachmentPlayer{ronPermanentChoicePlayer: &ronPermanentChoicePlayer{Player: baseB}, accept: true}
	g.SetPlayerAt(0, a)
	g.SetPlayerAt(1, b)
	old := g.PutOnBattlefield(mage.NewCreature("Old", "", 2, 4), a.PlayerID())
	next := g.PutOnBattlefield(mage.NewCreature("Next", "", 2, 4), a.PlayerID())
	aura := g.PutOnBattlefield(mage.NewAura("Aura", "", mage.WithCastTarget(mage.TargetCreature())), b.PlayerID())
	g.Attach(aura.ID(), old.ID())
	b.choose = func(candidates []*mage.Permanent) *mage.Permanent {
		for _, p := range candidates {
			if p.ID() == next.ID() {
				return p
			}
		}
		t.Fatal("delayed destination absent")
		return nil
	}
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 1 || b.offers != 0 {
		t.Fatal("waiting query made a choice")
	}
	g.AddControlEffect(mage.ControlEffectSpec{TargetID: aura.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(aura.ID()).AttachedTo != next.ID() || b.offers != 1 || a.offers != 0 {
		t.Fatal("delayed action lost saved player or target")
	}
}
