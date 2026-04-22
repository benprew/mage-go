#ifndef MAGE_PYLIB_ABI_H
#define MAGE_PYLIB_ABI_H

#include <stdint.h>

typedef struct {
    int64_t n;
    const int64_t* handles;
    const int64_t* perspective_player_idx;
} MageBatchRequest;

typedef struct {
    int64_t max_options;
    int64_t max_targets_per_option;
    int64_t max_cached_choices;
    int64_t zone_slot_count;
    int64_t game_info_dim;
    int64_t option_scalar_dim;
    int64_t target_scalar_dim;
    int64_t decision_capacity;
} MageEncodeConfig;

typedef struct {
    int64_t* trace_kind_id;
    int64_t* slot_card_rows;
    float* slot_occupied;
    float* slot_tapped;
    float* game_info;
    int64_t* pending_kind_id;
    int64_t* num_present_options;
    int64_t* option_kind_ids;
    float* option_scalars;
    float* option_mask;
    int64_t* option_ref_slot_idx;
    int64_t* option_ref_card_row;
    float* target_mask;
    int64_t* target_type_ids;
    float* target_scalars;
    float* target_overflow;
    int64_t* target_ref_slot_idx;
    uint8_t* target_ref_is_player;
    uint8_t* target_ref_is_self;
    uint8_t* may_mask;
    int64_t* decision_start;
    int64_t* decision_count;
    int64_t* decision_option_idx;
    int64_t* decision_target_idx;
    uint8_t* decision_mask;
    uint8_t* uses_none_head;
} MageEncodeOutputs;

typedef struct {
    int64_t decision_rows_written;
    int64_t error_code;
    char* error_message;
} MageEncodeResult;

#endif
