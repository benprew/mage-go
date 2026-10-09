package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type typedSacrificePlayer struct {
	*BasePlayer
	accept bool
	offers int
	before func()
}

func (p *typedSacrificePlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.before != nil {
		p.before()
	}
	return p.accept
}

func TestResolutionSacrificeIdentity(t *testing.T) {
	for _, branch := range []bool{false, true} {
		for _, scenario := range []string{"pay", "decline", "missing", "returned", "phased", "phase-cycle", "opponent", "replaced", "regeneration", "indestructible", "stale", "clone"} {
			t.Run(scenario+map[bool]string{true: "/paid", false: "/unpaid"}[branch], func(t *testing.T) {
				a := &typedSacrificePlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline"}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				c := NewArtifact("Subject", "")
				c.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				ref := BindPermanent("subject")
				body := GainLifeFor(You, Fixed(3))
				effect := UnlessSacrificePaid(You, From(ref), body)
				if branch {
					effect = IfSacrificePaid(You, From(ref), body)
				}
				ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
				sequence := Sequence(Snapshot(This, ref), effect)
				ctx.SourceID = p.ID()
				ctx.sourceIncarnation = p.incarnationID
				if err := Snapshot(This, ref).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				switch scenario {
				case "missing", "returned":
					g.RemoveFromBattlefield(p)
					if scenario == "returned" {
						g.PutOnBattlefield(c, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "opponent":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "regeneration":
					g.AddRegenerationShield(p.ID())
				case "indestructible":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
				case "replaced":
					g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
				case "stale":
					a.before = func() { g.RemoveFromBattlefield(g.FindPermanent(p.ID())); g.PutOnBattlefield(c, a.PlayerID()) }
				}
				clone := g.Clone()
				if scenario == "clone" {
					if err := ApplyEffect(clone, sequence, p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					cloneLife := 20
					if !branch {
						cloneLife = 23
					}
					if clone.GetPlayer(a.PlayerID()).Life() != cloneLife || clone.FindPermanent(p.ID()) == nil {
						t.Fatal("clone payment branch")
					}
					if g.FindPermanent(p.ID()) == nil || a.Life() != 20 {
						t.Fatal("clone changed original")
					}
					return
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				paid := scenario == "pay" || scenario == "phase-cycle" || scenario == "replaced" || scenario == "regeneration" || scenario == "indestructible"
				want := 20
				if paid == branch {
					want += 3
				}
				if a.Life() != want {
					t.Fatalf("life %d want %d", a.Life(), want)
				}
				offers := 1
				if scenario == "missing" || scenario == "returned" || scenario == "phased" || scenario == "opponent" {
					offers = 0
				}
				if a.offers != offers {
					t.Fatalf("offers %d want %d", a.offers, offers)
				}
				if paid && g.FindPermanentIncludingPhased(p.ID()) != nil {
					t.Fatal("subject not sacrificed")
				}
				if scenario == "replaced" && len(g.GetExile()) != 1 {
					t.Fatal("replacement lost")
				}
				if scenario == "pay" && len(b.Graveyard()) != 1 {
					t.Fatal("wrong owner")
				}
				if scenario == "stale" && g.FindPermanent(p.ID()) == nil {
					t.Fatal("returned subject consumed")
				}
			})
		}
	}
}

func TestResolutionSacrificeDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{IfSacrificePaid(You, nil, GainLifeFor(You, Fixed(1))), IfSacrificePaid(nil, This, GainLifeFor(You, Fixed(1))), UnlessSacrificePaid(You, This, nil), IfSacrificePaid(You, From(BindPermanent("unbound")), GainLifeFor(You, Fixed(1)))} {
		c := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestRequiredTypedSacrificeIdentity(t *testing.T) {
	for _, paidBranch := range []bool{false, true} {
		for _, state := range []string{"present", "missing", "returned", "phased", "phase-cycle", "control", "replaced", "prevented", "regeneration", "indestructible"} {
			t.Run(fmt.Sprintf("%s/%t", state, paidBranch), func(t *testing.T) {
				a := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				c := NewArtifact("Subject", "")
				c.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				binding := BindPermanent("subject")
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID}
				if err := Snapshot(This, binding).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "missing", "returned":
					g.RemoveFromBattlefield(p)
					if state == "returned" {
						g.PutOnBattlefield(c, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "replaced":
					g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), uuid.New())
				case "prevented":
					g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
				case "regeneration":
					g.AddRegenerationShield(p.ID())
				case "indestructible":
					g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
				}
				effect := UnlessRequiredSacrificePaid(You, From(binding), GainLifeFor(You, Fixed(3)))
				if paidBranch {
					effect = IfRequiredSacrificePaid(You, From(binding), GainLifeFor(You, Fixed(3)))
				}
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				paid := state == "present" || state == "phase-cycle" || state == "replaced" || state == "prevented" || state == "regeneration" || state == "indestructible"
				want := 20
				if paid == paidBranch {
					want += 3
				}
				if a.Life() != want || a.offers != 0 {
					t.Fatalf("life=%d want=%d offers=%d", a.Life(), want, a.offers)
				}
				if paid && state != "prevented" && g.FindPermanentIncludingPhased(p.ID()) != nil {
					t.Fatal("subject remains")
				}
				if state == "returned" && g.FindPermanent(p.ID()) == nil {
					t.Fatal("returned subject consumed")
				}
				if state == "present" && len(b.Graveyard()) != 1 {
					t.Fatal("wrong owner destination")
				}
				if state == "replaced" && len(g.GetExile()) != 1 {
					t.Fatal("replacement lost")
				}
			})
		}
	}
}

func TestRequiredTypedSacrificeDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		IfRequiredSacrificePaid(nil, This, GainLifeFor(You, Fixed(1))),
		IfRequiredSacrificePaid(You, nil, GainLifeFor(You, Fixed(1))),
		UnlessRequiredSacrificePaid(You, This, nil),
		IfRequiredSacrificePaid(You, From(BindPermanent("unbound")), GainLifeFor(You, Fixed(1))),
	} {
		c := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestRequiredTypedSacrificeReplacementOrdering(t *testing.T) {
	for _, mode := range []int{0, 1} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			a := &choiceRecordingPlayer{BasePlayer: NewBasePlayer("A"), chosenMode: mode}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))), WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, Fixed(10)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(2)))))
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, a.PlayerID())
			first, second := uuid.New(), uuid.New()
			g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), first)
			g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), second)
			g.AddRegenerationShield(p.ID())
			effect := IfRequiredSacrificePaid(You, This, GainLifeFor(You, Fixed(3)))
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := first
			if mode == 1 {
				want = second
			}
			if len(g.GetExile()) != 1 || g.GetExile()[0].ExiledBy != want || len(b.Graveyard()) != 0 || a.Life() != 23 {
				t.Fatal("replacement ordering or payment")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("sacrifice LKI triggers or death exclusion")
			}
			g.ResolveStack()
			if a.Life() != 26 {
				t.Fatal("wrong LKI trigger resolution")
			}
		})
	}
}

type requiredTypedSacrificeChooser struct {
	*requiredPaymentPlayer
	choice *Permanent
}

func (p *requiredTypedSacrificeChooser) ChoosePermanent([]*Permanent, string, GameReader) *Permanent {
	return p.choice
}

func TestRequiredTypedSacrificeRollback(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprint(invalid), func(t *testing.T) {
			a := &requiredTypedSacrificeChooser{requiredPaymentPlayer: &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}}
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
			other := g.PutOnBattlefield(NewArtifact("Other", ""), a.PlayerID())
			if invalid {
				a.choice = other
			}
			effect := UnlessRequiredSacrificePaid(You, This, GainLifeFor(You, Fixed(3)))
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 23 || a.offers != 0 || g.FindPermanent(p.ID()) == nil || g.FindPermanent(other.ID()) == nil || len(a.Graveyard()) != 0 {
				t.Fatal("failed choice changed payment state")
			}
		})
	}
}
