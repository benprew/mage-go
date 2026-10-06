package mage

import (
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
