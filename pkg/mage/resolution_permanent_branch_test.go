package mage

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentPropertyBranchSelection(t *testing.T) {
	for _, tapped := range []bool{false, true} {
		for _, unless := range []bool{false, true} {
			g := NewGame(NewBasePlayer("A"), NewBasePlayer("B"))
			a := g.PlayerAt(0)
			p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), a.PlayerID())
			if tapped {
				g.TapPermanent(p)
			}
			matching := Sequence(UntapPermanent(This), GainLifeFor(You, Fixed(2)))
			nonmatching := Sequence(TapSubject(This), GainLifeFor(You, Fixed(7)))
			effect := IfPermanentMatchesElse(This, IsTapped, matching, nonmatching)
			want := 27
			if tapped {
				want = 22
			}
			if unless {
				effect = UnlessPermanentMatches(This, IsTapped, nonmatching)
				if tapped {
					want = 20
				}
			}
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != want {
				t.Fatalf("tapped=%v unless=%v life=%d want=%d", tapped, unless, a.Life(), want)
			}
		}
	}
}

func TestPermanentPropertyBranchInformation(t *testing.T) {
	for _, reference := range []string{"source", "target", "saved", "attachment"} {
		for _, change := range []string{"live", "false", "leave", "return", "phase", "phase-cycle", "unknown", "control"} {
			t.Run(reference+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Subject", "", 2, 3, WithKeyword(Flying))
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				target := DeclarePermanentTarget("subject", AnyPermanent)
				saved := BindPermanent("saved")
				ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceIncarnation: p.incarnationID, sourceCaptured: true, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
				if err := Snapshot(target, saved).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				var ref PermanentRef = This
				switch reference {
				case "target":
					ref = target
				case "saved":
					ref = From(saved)
				case "attachment":
					aura := g.PutOnBattlefield(NewEnchantment("Aura", "", WithSubTypes("Aura")), a.PlayerID())
					g.Attach(aura.ID(), p.ID())
					ctx.SourceID, ctx.sourceIncarnation = aura.ID(), aura.incarnationID
					ref = AttachedPermanent
				}
				switch change {
				case "false":
					g.MutablePermanent(p.ID()).RevokeBaseAttr(Flying)
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "return" {
						a.RemoveFromHand(card.ID())
						q := g.PutOnBattlefield(card, b.PlayerID())
						q.RevokeBaseAttr(Flying)
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "unknown":
					ctx.sourceIncarnation = uuid.New()
					ctx.targetBindings[target.targetDeclaration][0].incarnation = uuid.New()
					g.resolution.permanentBindings[saved] = capturedPermanent{p.ID(), uuid.New()}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				}
				body := Sequence(GainLifeFor(You, Fixed(2)), TapSubject(ref))
				effect := IfPermanentMatchesElse(ref, HasKeywordFilter(Flying), body, GainLifeFor(You, Fixed(7)))
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					local := *ctx
					local.Game = game
					if err := effect.Apply(&local); err != nil {
						t.Fatal(err)
					}
					want := 22
					if change == "false" {
						want = 27
					}
					if change == "unknown" {
						want = 20
					}
					if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("wrong branch or controller")
					}
					if change == "return" && game.FindPermanent(p.ID()).Tapped {
						t.Fatal("retained information permitted an action on a new incarnation")
					}
				}
				if change == "unknown" {
					if err := UnlessPermanentMatches(ref, AnyPermanent, GainLifeFor(You, Fixed(10))).Apply(ctx); err != nil {
						t.Fatal(err)
					}
					if a.Life() != 20 {
						t.Fatal("missing information satisfied a negated condition")
					}
				}
			})
		}
	}
}

func TestPermanentPropertyBranchDefinitions(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		IfPermanentMatchesElse(nil, AnyPermanent, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		IfPermanentMatchesElse(This, AnyPermanent, nil, GainLifeFor(You, Fixed(2))),
		IfPermanentMatchesElse(This, AnyPermanent, GainLifeFor(You, Fixed(1)), nil),
		IfPermanentMatchesElse(This, AnyPermanent, GainLifeFor(You, Fixed(1)), TapSubject(DeclarePermanentTarget("unbound", AnyPermanent))),
		UnlessPermanentMatches(This, AnyPermanent, nil),
		UnlessPermanentMatches(From(BindPermanent("unbound")), AnyPermanent, GainLifeFor(You, Fixed(1))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid branch accepted", effect.Text())
		}
	}
	target := DeclarePermanentTarget("bound", AnyPermanent)
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), IfPermanentMatchesElse(target, AnyPermanent, TapSubject(target), UntapPermanent(target)), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestPermanentPropertyBranchErrorAndSequence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), a.PlayerID())
	failure := errors.New("branch failed")
	bad := FuncEffect("failure", EffectProperties{}, func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return failure })
	effect := Sequence(TapSubject(This), IfPermanentMatchesElse(This, IsTapped, bad, GainLifeFor(You, Fixed(7))), GainLifeFor(You, Fixed(10)))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); !errors.Is(err, failure) {
		t.Fatal("selected branch error lost", err)
	}
	if a.Life() != 20 {
		t.Fatal("unselected branch or later instruction ran after error")
	}
}

func TestPermanentPropertyBranchReadsOnce(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), a.PlayerID())
	calls := 0
	filter := NewPermanentFilter("count reads", func(*Permanent, *Game) bool { calls++; return false })
	effect := IfPermanentMatchesElse(This, filter, GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(7)))
	if effect.Properties().LifeGain != 0 {
		t.Fatal("AI shortcut bypasses condition")
	}
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || a.Life() != 27 {
		t.Fatal("branch did not read its condition exactly once")
	}
	effect = UnlessPermanentMatches(This, PermanentFilter{}, GainLifeFor(You, Fixed(10)))
	if effect.Properties().LifeGain != 0 {
		t.Fatal("AI shortcut bypasses negated condition")
	}
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 27 {
		t.Fatal("zero filter did not match known information")
	}
}

func TestPermanentPropertyBranchStack(t *testing.T) {
	for _, scenario := range []string{"copy", "clone", "illegal", "return", "phase", "unpaid"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(PrecombatMain)
			target := DeclarePermanentTarget("subject", IsCreature)
			card := NewInstant("Branch", "{R}", Spell(IfPermanentMatchesElse(target, IsTapped, Sequence(UntapPermanent(target), GainLifeFor(You, Fixed(2))), Sequence(TapSubject(target), GainLifeFor(You, Fixed(7)))), WithTargets(target)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(NewCreature("Subject", "", 2, 3), b.PlayerID())
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if scenario != "unpaid" {
				a.ManaPool().Add(Red, 1)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0)
			if scenario == "unpaid" {
				if err == nil || len(a.Hand()) != 1 || !g.Stack().IsEmpty() || p.Tapped || a.Life() != 20 {
					t.Fatal("failed cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || p.Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("branch ran before paid spell resolved")
			}
			switch scenario {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "illegal":
				g.MutablePermanent(p.ID()).GrantBaseAttr(Shroud)
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				b.RemoveFromHand(p.ID())
				g.PutOnBattlefield(p.Card, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.TapPermanent(clone.FindPermanent(p.ID()))
			}
			g.ResolveStack()
			clone.ResolveStack()
			wantA, wantB := 27, 20
			if scenario == "copy" {
				wantA, wantB = 22, 27
			}
			if scenario == "illegal" || scenario == "return" || scenario == "phase" {
				wantA = 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatal("stack lost legality, timing, identity or controller", a.Life(), b.Life())
			}
			cloneA := wantA
			if scenario == "clone" {
				cloneA = 22
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("clone reused another game's branch result")
			}
			if scenario == "return" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("returned target received the branch")
			}
		})
	}
}

func TestPermanentPropertyBranchBindingScope(t *testing.T) {
	saved := BindPermanent("branch result")
	read := TapSubject(From(saved))
	produce := Snapshot(This, saved)
	for _, effect := range []ResolvingEffect{
		IfPermanentMatchesElse(This, AnyPermanent, produce, read),
		Sequence(IfPermanentMatchesElse(This, AnyPermanent, produce, produce), read),
		Sequence(UnlessPermanentMatches(This, IsTapped, produce), read),
	} {
		card := NewArtifact("Invalid scope", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("branch result escaped its producer path")
		}
	}
	effect := IfPermanentMatchesElse(This, IsTapped, Sequence(produce, UntapPermanent(From(saved))), Sequence(produce, read))
	card := NewArtifact("Local scope", "", WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(card, a.PlayerID())
	for _, tapped := range []bool{true, false} {
		if tapped {
			g.TapPermanent(g.FindPermanent(p.ID()))
		} else {
			g.UntapPermanent(g.FindPermanent(p.ID()))
		}
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(p.ID()).Tapped == tapped {
			t.Fatal("local branch binding did not supply its action")
		}
	}
}
