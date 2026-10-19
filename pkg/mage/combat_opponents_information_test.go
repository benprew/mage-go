package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatOpponentsInformation(t *testing.T) {
	for _, role := range []string{"attacker", "blocker"} {
		for _, change := range []string{"live", "source-phase", "source-phase-cycle", "departed", "source-return", "opponent-return", "phase", "phase-cycle", "control", "noncreature", "removed-before-death"} {
			t.Run(role+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				sourceCard := NewCreature("Source", "", 1, 10)
				source := g.PutOnBattlefield(sourceCard, a.PlayerID())
				opponentCard := NewCreature("Opponent", "", 1, 10)
				opponent := g.PutOnBattlefield(opponentCard, b.PlayerID())
				if role == "attacker" {
					g.combat.AddAttackerIncarnation(source.ID(), source.incarnationID, b.PlayerID())
					g.combat.AddBlockerIncarnation(opponent.ID(), opponent.incarnationID, source.ID())
				} else {
					g.combat.AddAttackerIncarnation(opponent.ID(), opponent.incarnationID, a.PlayerID())
					g.combat.AddBlockerIncarnation(source.ID(), source.incarnationID, opponent.ID())
				}
				binding := BindPermanent("dying")
				ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
				if err := Snapshot(This, binding).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if change == "removed-before-death" {
					g.RemoveFromCombat(source.ID())
				}
				if change == "source-phase" {
					g.PhaseOut(source.ID())
				}
				if change == "source-phase-cycle" {
					g.PhaseIn(g.PhaseOut(source.ID()))
				}
				if change != "live" && change != "source-phase" && change != "source-phase-cycle" {
					g.BouncePermanentToHand(source)
				}
				switch change {
				case "source-return":
					a.RemoveFromHand(sourceCard.ID())
					g.PutOnBattlefield(sourceCard, a.PlayerID())
				case "opponent-return":
					g.BouncePermanentToHand(opponent)
					b.RemoveFromHand(opponentCard.ID())
					g.PutOnBattlefield(opponentCard, b.PlayerID())
				case "phase":
					g.PhaseOut(opponent.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(opponent.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: opponent.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "noncreature":
					g.AddContinuousEffect(TargetEffect(core.LayerType, core.Indefinite, opponent.ID(), func(game *Game, p *Permanent) error {
						game.GetEffects().RevokeAttr(p.ID(), core.AttrIsCreature)
						return nil
					}))
				}
				want := 1
				if change == "opponent-return" || change == "phase" || change == "noncreature" || change == "removed-before-death" || change == "source-phase-cycle" {
					want = 0
				}
				for _, game := range []*Game{g, g.Clone()} {
					local := *ctx
					local.Game = game
					for _, ref := range []PermanentRef{This, From(binding)} {
						got := CombatOpponentsWithLKI(ref).resolvePermanents(&local)
						if len(got) != want || (want == 1 && got[0].ID() != opponent.ID()) {
							t.Fatalf("got %v, want %d opponents", got, want)
						}
					}
				}
			})
		}
	}
}

func TestCombatOpponentsInformationDefinitions(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "source", Card: NewArtifact("Good", "", WithAbility(Activate(Costs(), DestroyEach(CombatOpponentsWithLKI(This)))))},
		{Name: "target", Card: NewInstant("Good", "", Spell(DestroyEach(CombatOpponentsWithLKI(target)), WithTargets(target)))},
		{Name: "nil", Card: NewInstant("Bad", "", Spell(DestroyEach(CombatOpponentsWithLKI(nil)))), WantErr: true},
		{Name: "unbound target", Card: NewInstant("Bad", "", Spell(DestroyEach(CombatOpponentsWithLKI(target)))), WantErr: true},
		{Name: "unbound saved", Card: NewInstant("Bad", "", Spell(DestroyEach(CombatOpponentsWithLKI(From(BindPermanent("missing")))))), WantErr: true},
	})
}
