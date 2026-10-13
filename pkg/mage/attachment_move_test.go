package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type attachmentMovePlayer struct {
	*BasePlayer
	choose func([]*Permanent) *Permanent
	calls  int
}

func (p *attachmentMovePlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	p.calls++
	return p.choose(candidates)
}

func TestAttachmentMove(t *testing.T) {
	for _, change := range []string{"choose", "nil", "invalid", "shroud", "hexproof", "protection", "cant-enchant", "creature-aura", "aura-phase", "host-phase", "cycle", "aura-return", "host-return", "during-aura-return", "during-host-return", "during-protection"} {
		t.Run(change, func(t *testing.T) {
			a := &attachmentMovePlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			old := g.PutOnBattlefield(NewCreature("Old", "", 2, 4), b.PlayerID())
			first := g.PutOnBattlefield(NewCreature("First", "", 2, 4), b.PlayerID())
			hostCard := NewCreature("Second", "", 2, 4)
			second := g.PutOnBattlefield(hostCard, b.PlayerID())
			land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
			auraCard := NewAura("Aura", "{U}", WithCastTarget(TargetCreature()))
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), old.ID())
			ref := BindPermanent("attachment")
			ctx := &EffectContext{Game: g, SourceID: aura.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, ref).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			a.choose = func(candidates []*Permanent) *Permanent {
				if slices.ContainsFunc(candidates, func(p *Permanent) bool { return p.ID() == old.ID() || p.ID() == land.ID() || p.ID() == aura.ID() }) {
					t.Fatal("illegal host offered")
				}
				switch change {
				case "nil":
					return nil
				case "invalid":
					return land
				case "during-aura-return":
					g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
					g.PutOnBattlefield(auraCard, b.PlayerID())
				case "during-host-return":
					g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
					g.PutOnBattlefield(hostCard, b.PlayerID())
				case "during-protection":
					if err := ApplyEffect(g, Until(core.EndOfTurn, GrantProtectionColors(This, core.Blue)), second.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				return second
			}
			want, calls := second.ID(), 1
			switch change {
			case "nil", "invalid":
				want = first.ID()
			case "shroud":
				second.GrantBaseAttr(core.Shroud)
			case "hexproof":
				second.GrantBaseAttr(core.Hexproof)
			case "protection":
				if err := ApplyEffect(g, Until(core.EndOfTurn, GrantProtectionColors(This, core.Blue)), second.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want, calls = first.ID(), 0
			case "cant-enchant":
				second.GrantBaseAttr(core.AttrCantBeEnchanted)
				want, calls = first.ID(), 0
			case "creature-aura":
				if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, core.TypeCreature)), aura.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want, calls = old.ID(), 0
			case "aura-phase":
				g.PhaseOut(aura.ID())
				want, calls = old.ID(), 0
			case "host-phase":
				g.PhaseOut(second.ID())
				want, calls = first.ID(), 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(second.ID()))
			case "aura-return":
				g.RemoveFromBattlefield(aura)
				g.PutOnBattlefield(auraCard, b.PlayerID())
				want, calls = uuid.Nil, 0
			case "host-return":
				g.RemoveFromBattlefield(second)
				second = g.PutOnBattlefield(hostCard, b.PlayerID())
			case "during-aura-return":
				want = uuid.Nil
			case "during-host-return", "during-protection":
				want = old.ID()
			}
			if err := AttachPermanentToChoice(You, From(ref), MatchingPermanents(AnyPermanent)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			present := g.FindPermanentIncludingPhased(aura.ID())
			if present.AttachedTo != want || a.calls != calls {
				t.Fatalf("host %v, calls %d; want %v, %d", present.AttachedTo, a.calls, want, calls)
			}
			if present.ControllerID() != b.PlayerID() || present.Card.Owner() != auraCard.Owner() {
				t.Fatal("movement changed control or owner")
			}
			if want != uuid.Nil && !slices.Contains(g.FindPermanentIncludingPhased(want).Attachments, aura.ID()) {
				t.Fatal("host lost attachment link")
			}
			if want != old.ID() && slices.Contains(g.FindPermanentIncludingPhased(old.ID()).Attachments, aura.ID()) {
				t.Fatal("old host kept attachment link")
			}
		})
	}
}

func TestAttachmentMoveKindsAndTimestamp(t *testing.T) {
	for _, kind := range []string{"Equipment", "Fortification", "ordinary", "Aura"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var oldCard, hostCard Card = NewCreature("Old", "", 2, 4), NewCreature("Host", "", 2, 4)
			if kind == "Fortification" {
				oldCard, hostCard = NewLand("Old"), NewLand("Host")
			}
			old, host := g.PutOnBattlefield(oldCard, b.PlayerID()), g.PutOnBattlefield(hostCard, b.PlayerID())
			card := NewArtifact("Attachment", "", WithSubTypes(kind))
			if kind == "Aura" {
				card = NewAura("Attachment", "", WithAbility(Static(While(Controls(AnyPermanent), GrantKeywordModification(AttachedPermanent, core.Flying)))))
			}
			attachment := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(attachment.ID(), old.ID())
			later := g.PutOnBattlefield(NewAura("Later", "", WithAbility(Static(While(Controls(AnyPermanent), GrantKeywordModification(AttachedPermanent, core.Trample))))), b.PlayerID())
			g.Attach(later.ID(), host.ID())
			if err := ApplyEffect(g, AttachPermanentToChoice(You, This, MatchingPermanents(AnyPermanent)), attachment.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := host.ID()
			if kind == "ordinary" {
				want = uuid.Nil
			}
			if g.FindPermanent(attachment.ID()).AttachedTo != want {
				t.Fatal("attachment kind legality")
			}
			if kind == "Aura" {
				if old.HasAttr(core.Flying) || !host.HasAttr(core.Flying) || !host.HasAttr(core.Trample) {
					t.Fatal("move did not update static subjects")
				}
				last := g.effects.effects[len(g.effects.effects)-1]
				if last.SourceID() != attachment.ID() {
					t.Fatal("attachment did not receive a later timestamp")
				}
			}
		})
	}
}

func TestAttachmentMoveWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	old := g.PutOnBattlefield(NewCreature("Old", "", 2, 4), b.PlayerID())
	attachment := g.PutOnBattlefield(NewAura("Aura", ""), a.PlayerID())
	g.Attach(attachment.ID(), old.ID())
	reads := 0
	filter := NewPermanentFilter("query guard", func(*Permanent, *Game) bool { reads++; return true })
	body := AttachPermanentToChoice(You, This, MatchingPermanents(filter))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), attachment.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SetRandomResults([]int{7})
	if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || g.RandIntn(10) != 7 || attachment.AttachedTo != old.ID() {
		t.Fatal("waiting query executed attachment action")
	}
}

func TestAttachmentMoveDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		AttachPermanentToChoice(nil, This, MatchingPermanents(AnyPermanent)),
		AttachPermanentToChoice(You, nil, MatchingPermanents(AnyPermanent)),
		AttachPermanentToChoice(You, This, nil),
		AttachPermanentToChoice(You, From(BindPermanent("missing")), MatchingPermanents(AnyPermanent)),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), AttachPermanentToChoice(You, This, MatchingPermanents(AnyPermanent)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
