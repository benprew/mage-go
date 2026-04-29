# Phase 4: native text-encoder assembler port

## Goal

Port `magic_ai/text_encoder/assembler.py:_assemble_one` (the opcode-stream
walker that produces `token_ids` / `attention_mask` / `option_positions` /
`target_positions` / `card_ref_positions`) to Go, so the Python side does
zero per-env tokenization work on the rollout hot path.

Phase 3 already shipped the closed-vocabulary token tables across the FFI
(`MageRegisterTokenTables` / `tokenTables` accessors). Phase 4 builds the
emitter that consumes those tables and writes a dense `(B, max_tokens)`
int64 token tensor + the anchor arrays directly.

## FFI surface to add

```c
typedef struct {
    int32_t batch_size;
    int32_t max_tokens;
    int32_t max_options;
    int32_t max_targets;
    int32_t max_card_refs;
} MageEncodeTokensConfig;

typedef struct {
    int64_t* token_ids;          // (B, max_tokens) int64
    int32_t* attention_mask;     // (B, max_tokens) int32
    int32_t* seq_lengths;        // (B,) int32
    int32_t* option_positions;   // (B, max_options) int32 (-1 = absent)
    int32_t* option_mask;        // (B, max_options) int32 (1 = present)
    int32_t* target_positions;   // (B, max_options, max_targets) int32 (-1 = absent)
    int32_t* target_mask;        // (B, max_options, max_targets) int32
    int32_t* card_ref_positions; // (B, max_card_refs) int32 (-1 = absent)
    int32_t* token_overflow;     // (B,) int32 (1 if truncated)
} MageEncodeTokensOutputs;

extern MageEncodeResult MageEncodeTokens(
    MageBatchRequest* req,
    MageEncodeConfig* cfg,
    MageEncodeTokensConfig* tokens_cfg,
    MageEncodeTokensOutputs* out
);
```

`MageEncodeConfig` already covers the existing per-env state (decision
arrays, scalar features, etc). The new call is the encoder + the
assembler in one pass: existing per-env work fills the scalar/decision
outputs; the new tokens-cfg branch fills the token tensor.

## Go-side architecture

`encoder.go` already calls `fillRenderPlan` per env. Replace with
`fillTokenStream` when `tokens_cfg != nil`:

```go
type tokenWriter struct {
    buf            []int64
    cursor         int32
    cap            int32
    overflow       bool
    optionPos      []int32
    optionMask     []int32
    targetPos      []int32  // (max_options * max_targets) flattened
    targetMask     []int32
    cardRefPos     []int32  // (max_card_refs,)
    optionCursor   int32
    cardRefSeen    map[int32]bool
}

func (w *tokenWriter) writeI32Span(span []int32) {
    n := int32(len(span))
    if w.cursor+n > w.cap {
        w.overflow = true
        return
    }
    for _, v := range span {
        w.buf[w.cursor] = int64(v)
        w.cursor++
    }
}
```

The walker mirrors `_assemble_one` 1:1, dispatching every emit through
`tokenTables` accessors:

| Python emit                                     | Go equivalent                                  |
| ----------------------------------------------- | ---------------------------------------------- |
| `out.extend(frag_table[Frag.X])`                | `w.writeI32Span(t.fragmentSpan(int32(fragX)))` |
| `out.extend(turn_step_table[(turn, step_id)])`  | `w.writeI32Span(t.turnStepSpan(turn, stepID))` |
| `out.extend(life_owner_table[(life, owner)])`   | `w.writeI32Span(t.lifeOwnerSpan(life, owner))` |
| `out.extend(action_verb_table[kind_id])`        | `w.writeI32Span(t.actionVerbSpan(kindID))`     |
| `out.extend(mana_glyph_ids[color] * amount)`    | loop `amount` times calling glyphSpan          |
| `out.extend(name_lists[row])`                   | `w.writeI32Span(t.cardNameSpan(row))`          |
| `out.extend(body_lists[row])`                   | `w.writeI32Span(t.cardBodySpan(row))`          |
| `out.append(toks.option_id)` + record pos       | direct write of `int64(t.optionID)` + record   |
| status prefix on `STATUS_TAPPED`                | `w.writeI32Span(t.statusTapped)`               |
| `card_closer_ids` after place / end-card        | `w.writeI32Span(t.cardCloser)`                 |

Anchor recovery (option/target/card-ref positions) writes to the
preallocated `option_positions` / `target_positions` / `card_ref_positions`
slices inline at the same dispatch site that emits the structural token.

The walker iterates the same game state that `fillRenderPlan` does — reuse
`buildRenderPlanIndex` and the per-zone walk. **Do not** generate the
opcode stream first and then re-walk it; emit tokens directly while
walking the snapshot, which skips the int32 buffer entirely.

## Parity test infrastructure (the gate)

A Phase-4 PR is unsafe without a parity test that runs both paths against
the same game state and asserts byte-equal outputs. Suggested layout in
magic-ai:

`tests/test_native_assembler_parity.py`:
1. Spin up a small handful of game handles via `mage.new_game(...)` with
   pre-seeded decks (the existing `tests/golden/` fixtures should cover
   most cases).
2. Drive each game forward through the priority loop, snapshotting at
   every `pending` state.
3. At each snapshot, call:
   - `native_encoder.encode_handles(...)` → `assemble_batch(...)` (Python path)
   - `native_encoder.encode_tokens(...)` (new Go path)
4. Assert byte-equal on `token_ids`, `attention_mask`, `seq_lengths`,
   `option_positions`, `option_mask`, `target_positions`, `target_mask`,
   `card_ref_positions`.

Driving this test requires the existing native FFI + the new
`MageEncodeTokens` export — once the test is green for both paths, the
cutover (Phase 5/6) is safe.

## Phasing within Phase 4

| Sub-phase | Scope                                                       | LOC est |
| --------- | ----------------------------------------------------------- | ------- |
| 4a        | `tokenWriter` + structural opcode dispatch (no place_card)  | ~250    |
| 4b        | `OP_PLACE_CARD` body memcpy + status prefix + closer        | ~80     |
| 4c        | `OP_OPTION` / `OP_TARGET` with anchor position recording    | ~150    |
| 4d        | Truncation (preserve K↔K invariant: store -1 sentinels)     | ~60     |
| 4e        | `MageEncodeTokens` Go export + Python ctypes wrapper        | ~120    |
| 4f        | parity test infrastructure                                  | ~250    |

Total ~900 LOC. Each sub-phase is independently testable via the
`MageTokenTableLookup`-style debug helpers.

## Open questions

- **Body memcpy width**: `tokenTables.cardBodySpan` returns `[]int32`,
  but the output is int64. We widen during memcpy. Worth measuring
  whether int32 output (matching the source dtype) is acceptable; the
  Python path uses int64 for tensor indexing convenience but nothing
  downstream actually requires 64-bit token ids (vocab is 50K).

- **Output ownership**: who allocates the `(B, max_tokens)` torch tensor?
  Cleanest is Python, same pattern as the existing `MageEncodeBatch`
  outputs (raw pointers). The `NativeBatchEncoder._scratch_buffers`
  helper already handles size-up reallocation; extend it to add the
  token tensor.

- **Mana glyph repeat**: the `for _ in range(amount)` loop in the Python
  walker is the only place a single opcode emits a variable-length
  token stream that isn't a single span. The Go side can either loop
  the same way or precompute `mana_repeat_table[(color, amount)]` for
  bounded amounts. Worth measuring; if mana counts top out at ~12 the
  table is 6×12 = 72 entries.
