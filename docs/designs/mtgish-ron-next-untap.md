# RON next-untap restriction

Supported shape: action tuple
`PermanentDoesntUntapDuringControllersNextUntap(reference)/1`.
Its operand is a symbol accepted by `compilePermanentRef(Node, scope)`:
`ThisPermanent` for a permanent source; `HostPermanent` for an Aura;
`Ref_TargetPermanent` for the existing single permanent target declaration.
The shared `compileEffect(Node, scope) ([]effectFactory, error)` builds
`mage.SkipNextUntap(subject.build(instanceBindings))`. Spelling and scope
validation stay in RON; all execution stays in the engine.

Contexts: resolving actions in `SpellActions`, `Activated`,
`ActivatedWithModifiers` and supported `TriggerA`/`TriggerI` action declarations.
The action is not a rule, cost, static modification, condition or entry operation.
Each constructor has exact arity. Symbol operands cannot be tuples. Unbound,
wrong-type and cross-rule target references remain errors. There are no numbered
or multiple target groups, event references or action-result references added.
The collection spelling `EachPermanentDoesntUntapDuringControllersNextUntap`
and static `DoesntUntapDuringControllersUntap` remain separate unsupported forms.

Each factory gets fresh typed target bindings. Definition metadata includes the
read so engine validation rejects a missing target declaration. The engine
resolves source incarnations, current Aura attachment or departure attachment
LKI, and typed targets. An absent or phased permanent receives no restriction.
Departure and return never redirect the reference to a new incarnation.
Stack copies retarget independently and game clones retain independent state.

The effect installs the existing TurnSystem restriction at resolution. It does
not tap the subject or prevent untap actions outside an untap step. The next
untap step of the subject's current controller consumes it, even when the subject
is already untapped. Repeated restrictions expire together. An inactive player's
untap step does not consume it. Phasing after installation preserves it until
the subject phases in before its controller's untap. Control changes move the
applicable next untap to the new controller. Zone departure clears the restriction.

Real full-card tests cover Barl's Cage, Elvish Hunter and Homarid Warrior. The
last card also verifies action order, temporary shroud expiration and self tap.
Deep Spawn remains rejected because optional mill payment is unsupported.
Spore Cloud needs collection restriction, combat tap and prevention adapters;
this action alone does not accept its complete record.

Evidence: `skip_untap_test.go` and the typed-read test in
`permanent_scope_test.go`. Synthetic fixtures cover spell, source-entry trigger,
Aura host and departed Aura attachment LKI contexts. These fixtures do not count
as additional verified collection cards. CR 400.7, 502.3, 608.2b and 702.26b/d
apply. No engine API, LKI characteristic query, replacement ordering or batch
execution changes. Production registration and Go definitions remain unchanged.
