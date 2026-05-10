"""Python bindings for the mage-go engine via cffi.

Build the shared library first:

    go build -buildmode=c-shared -o libmage.dylib ./cmd/pylib   # macOS
    go build -buildmode=c-shared -o libmage.so    ./cmd/pylib   # linux

Then:

    import mage
    deck = {"name": "A", "cards": [{"name": "Mountain", "count": 20},
                                    {"name": "Lightning Bolt", "count": 40}]}
    game = mage.new_game(deck, deck, seed=42, shuffle=True)
    while not game.is_over:
        game.step(pick_action(game.pending))
    print("winner:", game.winner)

Override the library location with the MAGE_LIB env var or mage.load("/path").
"""

from __future__ import annotations

import os
import platform
from typing import Any

import orjson
from cffi import FFI


_CDEF = """
struct MageNewGame_return {
    int64_t r0;
    char *r1;
};
typedef struct {
    int64_t n;
    const int64_t *handles;
    const int64_t *perspective_player_idx;
} MageBatchRequest;
typedef struct {
    int64_t *ready;
    int64_t *game_over;
    int64_t *pending_player_idx;
    int64_t *winner_player_idx;
} MageBatchPollOutputs;
typedef struct {
    int64_t n;
    int64_t max_options;
    int64_t max_targets_per_option;
    const int64_t *handles;
    const int64_t *decision_start;
    const int64_t *decision_count;
    const int64_t *selected_choice_cols;
    const int64_t *may_selected;
} MageStepChoiceRequest;
typedef struct {
    int64_t max_options;
    int64_t max_targets_per_option;
    int64_t max_cached_choices;
    int64_t zone_slot_count;
    int64_t game_info_dim;
    int64_t option_scalar_dim;
    int64_t target_scalar_dim;
    int64_t decision_capacity;
    int64_t emit_render_plan;
    int64_t render_plan_capacity;
    int64_t dedup_card_bodies;
} MageEncodeConfig;
typedef struct {
    int64_t *trace_kind_id;
    int64_t *slot_card_rows;
    float *slot_occupied;
    float *slot_tapped;
    float *game_info;
    int64_t *pending_kind_id;
    int64_t *num_present_options;
    int64_t *option_kind_ids;
    float *option_scalars;
    float *option_mask;
    int64_t *option_ref_slot_idx;
    int64_t *option_ref_card_row;
    float *target_mask;
    int64_t *target_type_ids;
    float *target_scalars;
    float *target_overflow;
    int64_t *target_ref_slot_idx;
    uint8_t *target_ref_is_player;
    uint8_t *target_ref_is_self;
    uint8_t *may_mask;
    int64_t *decision_start;
    int64_t *decision_count;
    int64_t *decision_option_idx;
    int64_t *decision_target_idx;
    uint8_t *decision_mask;
    uint8_t *uses_none_head;
    int32_t *render_plan;
    int64_t *render_plan_lengths;
    int64_t *render_plan_overflow;
} MageEncodeOutputs;
typedef struct {
    int64_t decision_rows_written;
    int64_t error_code;
    char *error_message;
} MageEncodeResult;
struct MageNewGame_return MageNewGame(char *cfgJSON);
char *MageState(int64_t id);
char *MageLegal(int64_t id);
char *MageStep(int64_t id, char *actionJSON);
char *MageSetCardNameRows(char *cardNameRowsJSON);
MageEncodeResult MageBatchPoll(
    MageBatchRequest *req,
    MageBatchPollOutputs *out
);
MageEncodeResult MageBatchStepByChoice(
    MageStepChoiceRequest *req
);
typedef struct {
    int64_t n;
    int64_t max_decode_len;
    int64_t max_anchors;
    const int64_t *handles;
    const int32_t *decision_type;
    const int32_t *output_token_ids;
    const int32_t *output_pointer_subjects;
    const uint8_t *output_is_pointer;
    const int32_t *output_lens;
    const int32_t *pointer_anchor_handles;
    const int32_t *pointer_anchor_count;
} MageDecoderStepRequest;
MageEncodeResult MageBatchStepByDecoderAction(
    MageDecoderStepRequest *req
);
MageEncodeResult MageEncodeBatch(
    MageBatchRequest *req,
    MageEncodeConfig *cfg,
    MageEncodeOutputs *out
);
int64_t MagePendingPlayer(int64_t id);
int64_t MageIsOver(int64_t id);
char *MageWinner(int64_t id);
void MageFree(int64_t id);
void MageFreeString(char *s);
char *MageRegisteredCards(void);
char *MageRegisteredManaCosts(void);

typedef struct {
    int32_t fragment_count;
    const int32_t *structural_tokens;
    const int32_t *structural_offsets;
    int32_t turn_min;
    int32_t turn_max;
    int32_t step_count;
    const int32_t *turn_step_tokens;
    const int32_t *turn_step_offsets;
    int32_t life_min;
    int32_t life_max;
    int32_t owner_count;
    const int32_t *life_owner_tokens;
    const int32_t *life_owner_offsets;
    int32_t ability_min;
    int32_t ability_max;
    const int32_t *ability_tokens;
    const int32_t *ability_offsets;
    int32_t count_min;
    int32_t count_max;
    const int32_t *count_tokens;
    const int32_t *count_offsets;
    int32_t zone_count;
    const int32_t *zone_open_tokens;
    const int32_t *zone_open_offsets;
    const int32_t *zone_close_tokens;
    const int32_t *zone_close_offsets;
    int32_t action_verb_count;
    const int32_t *action_verb_tokens;
    const int32_t *action_verb_offsets;
    int32_t mana_color_count;
    const int32_t *mana_glyph_tokens;
    const int32_t *mana_glyph_offsets;
    int32_t card_ref_count;
    const int32_t *card_ref_ids;
    int32_t pad_id;
    int32_t option_id;
    int32_t target_open_id;
    int32_t target_close_id;
    int32_t tapped_id;
    int32_t untapped_id;
    int32_t card_closer_len;
    const int32_t *card_closer;
    int32_t status_tapped_len;
    const int32_t *status_tapped;
    int32_t status_untapped_len;
    const int32_t *status_untapped;
    int32_t card_row_count;
    const int32_t *card_body_tokens;
    const int64_t *card_body_offsets;
    const int32_t *card_name_tokens;
    const int64_t *card_name_offsets;
    int32_t dict_open_id;
    int32_t dict_close_id;
    int32_t card_open_id;
    const int32_t *dict_entry_ids;
    int32_t self_id;
    int32_t opp_id;
    int32_t stack_open_id;
    int32_t stack_close_id;
    int32_t command_open_id;
    int32_t command_close_id;
} MageTokenTables;

int32_t MageRegisterTokenTables(MageTokenTables *tables);
char *MageTokenTableSummary(void);
char *MageTokenTableLookup(int32_t kind, int32_t k0, int32_t k1);
char *MageEncodeTimingSummary(int32_t reset);
char *MageNativeTimingSummary(int32_t reset);

typedef struct {
    int32_t max_tokens;
    int32_t max_options;
    int32_t max_targets;
    int32_t max_card_refs;
} MageTokenAssemblerConfig;

typedef struct {
    int32_t *token_ids;
    int32_t *cu_seqlens;
    int32_t *seq_lengths;
    int32_t *state_positions;
    int32_t *card_ref_positions;
    int32_t *token_overflow;
} MagePackedTokenAssemblerOutputs;

MageEncodeResult MageEncodeTokensPacked(
    MageBatchRequest *req,
    MageEncodeConfig *cfg,
    MageEncodeOutputs *out,
    MageTokenAssemblerConfig *tok_cfg,
    MagePackedTokenAssemblerOutputs *packed_out
);

typedef struct {
    int32_t *spec_tokens;
    int32_t *spec_lens;
    int32_t *decision_type;
    int32_t *pointer_anchor_positions;
    int32_t *pointer_anchor_kinds;
    int32_t *pointer_anchor_subjects;
    int32_t *pointer_anchor_handles;
    int32_t *pointer_anchor_counts;
    uint8_t *legal_edge_bitmap;
    int32_t *legal_edge_n_blockers;
    int32_t *legal_edge_n_attackers;
    int32_t T_spec_max;
    int32_t N_anchors_max;
    int32_t N_blockers_max;
    int32_t N_attackers_max;
    int32_t spec_overflow;
} MagePackedSpecOutputs;

typedef struct {
    int32_t spec_open_id;
    int32_t spec_close_id;
    int32_t decision_type_id;
    int32_t legal_attacker_id;
    int32_t legal_blocker_id;
    int32_t legal_target_id;
    int32_t legal_action_id;
    int32_t for_action_id;
    int32_t max_value_open_id;
    int32_t max_value_close_id;
    int32_t player_ref0_id;
    int32_t player_ref1_id;
    const int32_t *dt_name_ids;
    const int32_t *stack_ref_ids;
    int32_t max_value_digit_max;
    const int32_t *max_value_digits;
    const int32_t *max_value_digit_offsets;
} MageDecisionSpecTokens;

int32_t MageRegisterDecisionSpecTokens(MageDecisionSpecTokens *tables);

MageEncodeResult MageEncodeDecisionSpec(
    MageBatchRequest *req,
    int32_t *state_token_lens,
    MagePackedSpecOutputs *spec_out,
    int64_t *handle_out
);

int32_t MagePackCombinedTokens(
    int32_t n,
    int32_t token_capacity,
    int32_t max_card_refs,
    MagePackedTokenAssemblerOutputs *packed,
    MagePackedSpecOutputs *spec
);

int32_t MageDecisionMaskNext(
    int64_t batch_handle,
    int32_t *prefix_tokens,
    int32_t *prefix_pointers,
    int32_t *prefix_lens,
    int32_t batch_size,
    int32_t prefix_len_max,
    int32_t grammar_vocab_size,
    int32_t n_anchors_max,
    uint8_t *out_vocab_mask,
    uint8_t *out_pointer_mask
);

void MageReleaseBatchHandle(int64_t handle);
"""


class MageError(RuntimeError):
    pass


_ffi: Any = None  # set by load()
_lib: Any = None  # set by load()
_lib_path_used: str | None = None


def _default_lib_path() -> str:
    override = os.environ.get("MAGE_LIB")
    if override:
        return override
    here = os.path.dirname(os.path.abspath(__file__))
    suffix = {"Darwin": ".dylib", "Linux": ".so", "Windows": ".dll"}.get(
        platform.system(), ".so"
    )
    candidate = os.path.join(here, "libmage" + suffix)
    if os.path.exists(candidate):
        return candidate
    for name in os.listdir(here):
        if name.startswith("libmage.") and name.endswith((".so", ".dylib", ".dll")):
            return os.path.join(here, name)
    raise FileNotFoundError(
        f"libmage not found next to {__file__}; "
        f"build with `go build -buildmode=c-shared -o cmd/pylib/libmage{suffix} ./cmd/pylib` "
        f"or set MAGE_LIB=/path/to/libmage{suffix}"
    )


def load(lib_path: str | None = None) -> None:
    """Explicitly (re)load the shared library. Called lazily by first API use."""
    global _ffi, _lib, _lib_path_used
    ffi = FFI()
    ffi.cdef(_CDEF)
    path = lib_path or _default_lib_path()
    _lib_path_used = os.path.abspath(path)
    _lib = ffi.dlopen(_lib_path_used)
    _ffi = ffi


def _ensure_loaded() -> None:
    if _lib is None:
        load()


def _take_raw(cstr) -> Any:
    if cstr == _ffi.NULL:
        raise MageError("null response from Go")
    try:
        raw = _ffi.string(cstr)
    finally:
        _lib.MageFreeString(cstr)
    return orjson.loads(raw)


def _take(cstr) -> dict[str, Any]:
    resp = _take_raw(cstr)
    if isinstance(resp, dict) and not resp.get("ok", True):
        raise MageError(resp.get("error", "unknown error"))
    return resp


def registered_cards() -> list[str]:
    _ensure_loaded()
    return _take_raw(_lib.MageRegisteredCards())


def registered_mana_costs() -> list[str]:
    _ensure_loaded()
    return _take_raw(_lib.MageRegisteredManaCosts())


def resolved_library_path() -> str:
    _ensure_loaded()
    assert _lib_path_used is not None
    return _lib_path_used


def native_timing_summary(reset: bool = False) -> dict[str, Any]:
    _ensure_loaded()
    return _take_raw(_lib.MageNativeTimingSummary(1 if reset else 0))


def encode_decision_spec(
    handles,
    state_token_lens,
    spec_out,
):
    """Run MageEncodeDecisionSpec for the given batch.

    ``handles`` is a sequence of int64 game handles. ``state_token_lens`` is
    a sequence of int32 state-text lengths (one per row) used to shift
    pointer-anchor positions into the combined stream. ``spec_out`` is a
    pre-built cffi ``MagePackedSpecOutputs *`` with all output buffers
    bound; the caller owns those buffers.

    Returns ``(rows_written, batch_handle)``. The handle is freed by
    ``release_batch_handle`` after all decoder steps for this batch are
    done.
    """

    _ensure_loaded()
    n = len(handles)
    handles_buf = _ffi.new("int64_t[]", list(handles))
    lens_buf = _ffi.new("int32_t[]", list(state_token_lens))
    req = _ffi.new("MageBatchRequest *")
    req.n = n
    req.handles = handles_buf
    req.perspective_player_idx = _ffi.NULL
    handle_out = _ffi.new("int64_t *")
    res = _lib.MageEncodeDecisionSpec(req, lens_buf, spec_out, handle_out)
    if res.error_code != 0:
        msg = _ffi.string(res.error_message).decode("utf-8") if res.error_message else ""
        if res.error_message:
            _lib.MageFreeString(res.error_message)
        raise MageError(f"MageEncodeDecisionSpec failed (code {res.error_code}): {msg}")
    if res.error_message:
        _lib.MageFreeString(res.error_message)
    return int(res.decision_rows_written), int(handle_out[0])


def decision_mask_next(
    batch_handle: int,
    prefix_tokens,
    prefix_pointers,
    prefix_lens,
    batch_size: int,
    prefix_len_max: int,
    grammar_vocab_size: int,
    n_anchors_max: int,
    out_vocab_mask,
    out_pointer_mask,
) -> int:
    """Compute per-row vocab + pointer masks for one decoder step.

    All array arguments are cffi pointers (``int32_t *`` / ``uint8_t *``)
    backed by caller-owned numpy / torch buffers. Returns 0 on success.
    """

    _ensure_loaded()
    return int(
        _lib.MageDecisionMaskNext(
            batch_handle,
            prefix_tokens,
            prefix_pointers,
            prefix_lens,
            batch_size,
            prefix_len_max,
            grammar_vocab_size,
            n_anchors_max,
            out_vocab_mask,
            out_pointer_mask,
        )
    )


def release_batch_handle(batch_handle: int) -> None:
    """Drop a batch handle previously returned by ``encode_decision_spec``."""

    _ensure_loaded()
    _lib.MageReleaseBatchHandle(batch_handle)


def _as_ptr(buf, ctype: str):
    """Best-effort coercion of a numpy / torch / list buffer to a cffi pointer.

    For numpy arrays and torch tensors with __array_interface__ /
    data_ptr(), wrap the existing memory as a cffi pointer (no copy). For
    plain Python sequences, allocate a fresh cffi buffer (the caller must
    keep the returned keepalive alive for the duration of the cgo call).
    Returns ``(ptr, keepalive)``.
    """

    if buf is None:
        return _ffi.NULL, None
    # numpy array
    if hasattr(buf, "__array_interface__"):
        addr = buf.__array_interface__["data"][0]
        return _ffi.cast(ctype, addr), buf
    # torch tensor (CPU contiguous)
    if hasattr(buf, "data_ptr"):
        return _ffi.cast(ctype, buf.data_ptr()), buf
    # cffi pointer already
    if isinstance(buf, _ffi.CData):
        return _ffi.cast(ctype, buf), buf
    # fallback: allocate a fresh buffer from a Python sequence
    elem = ctype.replace("const ", "").replace(" *", "[]")
    keep = _ffi.new(elem, list(buf))
    return _ffi.cast(ctype, keep), keep


def batch_step_by_decoder_action(
    handles,
    decision_type,
    output_token_ids,
    output_pointer_subjects,
    output_is_pointer,
    output_lens,
    pointer_anchor_handles,
    pointer_anchor_count,
) -> None:
    """Apply a batch of decoder-shaped actions to the engine.

    All array arguments must be host-resident, contiguous, and of the dtypes
    declared on the C ABI (see abi.h::MageDecoderStepRequest):

    * ``handles`` — ``[n]`` int64 game handles.
    * ``decision_type`` — ``[n]`` int32 (DecisionType enum, -1 = no-op).
    * ``output_token_ids`` — ``[n, L_max]`` int32 (PAD-padded).
    * ``output_pointer_subjects`` — ``[n, L_max]`` int32 (-1 on vocab steps).
    * ``output_is_pointer`` — ``[n, L_max]`` uint8.
    * ``output_lens`` — ``[n]`` int32.
    * ``pointer_anchor_handles`` — ``[n, N_max]`` int32.
    * ``pointer_anchor_count`` — ``[n]`` int32.

    Per-env errors are logged on the Go side and the env is skipped — the
    rest of the batch advances. See ``docs/decoder_impala_plan.md`` item 1.
    """

    _ensure_loaded()
    n = int(handles.shape[0]) if hasattr(handles, "shape") else len(handles)
    if hasattr(output_token_ids, "shape"):
        l_max = int(output_token_ids.shape[1])
    else:
        l_max = 0
    if hasattr(pointer_anchor_handles, "shape"):
        n_max = int(pointer_anchor_handles.shape[1])
    else:
        n_max = 0

    handles_p, _kh = _as_ptr(handles, "const int64_t *")
    dt_p, _kd = _as_ptr(decision_type, "const int32_t *")
    tok_p, _kt = _as_ptr(output_token_ids, "const int32_t *")
    ptr_p, _kp = _as_ptr(output_pointer_subjects, "const int32_t *")
    is_ptr_p, _ki = _as_ptr(output_is_pointer, "const uint8_t *")
    lens_p, _kl = _as_ptr(output_lens, "const int32_t *")
    anc_p, _ka = _as_ptr(pointer_anchor_handles, "const int32_t *")
    cnt_p, _kc = _as_ptr(pointer_anchor_count, "const int32_t *")

    req = _ffi.new("MageDecoderStepRequest *")
    req.n = n
    req.max_decode_len = l_max
    req.max_anchors = n_max
    req.handles = handles_p
    req.decision_type = dt_p
    req.output_token_ids = tok_p
    req.output_pointer_subjects = ptr_p
    req.output_is_pointer = is_ptr_p
    req.output_lens = lens_p
    req.pointer_anchor_handles = anc_p
    req.pointer_anchor_count = cnt_p

    res = _lib.MageBatchStepByDecoderAction(req)
    if res.error_code != 0:
        msg = (
            _ffi.string(res.error_message).decode("utf-8")
            if res.error_message
            else ""
        )
        if res.error_message:
            _lib.MageFreeString(res.error_message)
        raise MageError(
            f"MageBatchStepByDecoderAction failed (code {res.error_code}): {msg}"
        )
    if res.error_message:
        _lib.MageFreeString(res.error_message)


def register_decision_spec_tokens(
    spec_open_id: int,
    spec_close_id: int,
    decision_type_id: int,
    legal_attacker_id: int,
    legal_blocker_id: int,
    legal_target_id: int,
    legal_action_id: int,
    for_action_id: int,
    max_value_open_id: int,
    max_value_close_id: int,
    player_ref0_id: int,
    player_ref1_id: int,
    dt_name_ids,
    stack_ref_ids,
    max_value_digit_max: int,
    max_value_digits,
    max_value_digit_offsets,
):
    """Register the spec-tag id table + digit-token lookup with the Go side.

    ``dt_name_ids`` is a length-7 int sequence (one per DecisionType).
    ``stack_ref_ids`` is length-16. ``max_value_digits`` is a flat int32
    buffer of all digit-id sequences concatenated; ``max_value_digit_offsets``
    has length ``max_value_digit_max + 2``.

    Returns the cffi keepalive object: callers must hold a reference for
    the lifetime of the registration so the borrowed buffers stay alive.
    """

    _ensure_loaded()
    dt_buf = _ffi.new("int32_t[]", list(dt_name_ids))
    stack_buf = _ffi.new("int32_t[]", list(stack_ref_ids))
    digits_buf = _ffi.new("int32_t[]", list(max_value_digits))
    offsets_buf = _ffi.new("int32_t[]", list(max_value_digit_offsets))
    tables = _ffi.new("MageDecisionSpecTokens *")
    tables.spec_open_id = spec_open_id
    tables.spec_close_id = spec_close_id
    tables.decision_type_id = decision_type_id
    tables.legal_attacker_id = legal_attacker_id
    tables.legal_blocker_id = legal_blocker_id
    tables.legal_target_id = legal_target_id
    tables.legal_action_id = legal_action_id
    tables.for_action_id = for_action_id
    tables.max_value_open_id = max_value_open_id
    tables.max_value_close_id = max_value_close_id
    tables.player_ref0_id = player_ref0_id
    tables.player_ref1_id = player_ref1_id
    tables.dt_name_ids = dt_buf
    tables.stack_ref_ids = stack_buf
    tables.max_value_digit_max = max_value_digit_max
    tables.max_value_digits = digits_buf
    tables.max_value_digit_offsets = offsets_buf
    rc = int(_lib.MageRegisterDecisionSpecTokens(tables))
    if rc != 0:
        raise MageError(f"MageRegisterDecisionSpecTokens failed (code {rc})")
    # Caller must keep the keepalive alive — return the buffers + struct.
    return (tables, dt_buf, stack_buf, digits_buf, offsets_buf)


def new_game(
    deck_a: dict,
    deck_b: dict,
    name_a: str = "",
    name_b: str = "",
    seed: int = 0,
    shuffle: bool = True,
    hand_size: int = 7,
) -> "Game":
    _ensure_loaded()
    cfg = orjson.dumps({
        "player_a": deck_a,
        "player_b": deck_b,
        "name_a": name_a,
        "name_b": name_b,
        "seed": seed,
        "shuffle": shuffle,
        "hand_size": hand_size,
    })
    ret = _lib.MageNewGame(_ffi.new("char[]", cfg))
    resp = _take(ret.r1)
    if ret.r0 < 0:
        raise MageError(resp.get("error", "new_game failed"))
    return Game(int(ret.r0), resp)


class Game:
    """One live game handle. Not thread-safe; create one per worker."""

    def __init__(self, handle: int, initial: dict):
        self._id = handle
        self._last = initial

    @property
    def handle(self) -> int:
        return self._id

    @property
    def state(self) -> dict:
        return self._last.get("state") or {}

    @property
    def pending(self) -> dict | None:
        return self._last.get("pending")

    @property
    def is_over(self) -> bool:
        return bool(self._last.get("game_over"))

    @property
    def winner(self) -> str:
        return self._last.get("winner", "")

    def refresh_state(self) -> dict:
        self._last = _take(_lib.MageState(self._id))
        return self._last

    def legal(self) -> dict | None:
        ret = _lib.MageLegal(self._id)
        if ret == _ffi.NULL:
            return None
        try:
            raw = _ffi.string(ret)
        finally:
            _lib.MageFreeString(ret)
        if not raw or raw == b"null":
            return None
        return orjson.loads(raw)

    def step(self, action: dict) -> dict:
        payload = orjson.dumps(action)
        self._last = _take(_lib.MageStep(self._id, _ffi.new("char[]", payload)))
        return self._last

    def close(self):
        if self._id >= 0 and _lib is not None:
            _lib.MageFree(self._id)
            self._id = -1

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.close()

    def __del__(self):
        try:
            self.close()
        except Exception:
            pass
