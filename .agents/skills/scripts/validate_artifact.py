#!/usr/bin/env python3

import json
import sys
from pathlib import Path


def matches_type(value, expected):
    if expected == "object":
        return isinstance(value, dict)
    if expected == "array":
        return isinstance(value, list)
    if expected == "string":
        return isinstance(value, str)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if expected == "boolean":
        return isinstance(value, bool)
    if expected == "null":
        return value is None
    return False


def validate_node(schema, value, path, errors):
    if "const" in schema and value != schema["const"]:
        errors.append(f"{path}: expected {schema['const']!r}")
    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{path}: value is not in enum")

    if "type" in schema:
        types = schema["type"] if isinstance(schema["type"], list) else [schema["type"]]
        if not any(matches_type(value, expected) for expected in types):
            errors.append(f"{path}: expected type {' or '.join(types)}")
            return

    if isinstance(value, dict):
        for key in schema.get("required", []):
            if key not in value:
                errors.append(f"{path}: missing required property {key}")
        properties = schema.get("properties", {})
        for key, child in value.items():
            if key in properties:
                validate_node(properties[key], child, f"{path}.{key}", errors)
            elif schema.get("additionalProperties") is False:
                errors.append(f"{path}: unexpected property {key}")

    if isinstance(value, list):
        if "minItems" in schema and len(value) < schema["minItems"]:
            errors.append(f"{path}: expected at least {schema['minItems']} items")
        if "items" in schema:
            for index, child in enumerate(value):
                validate_node(schema["items"], child, f"{path}[{index}]", errors)

    if isinstance(value, str) and "minLength" in schema and len(value) < schema["minLength"]:
        errors.append(f"{path}: string is too short")
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        if "minimum" in schema and value < schema["minimum"]:
            errors.append(f"{path}: value is below minimum")
        if "maximum" in schema and value > schema["maximum"]:
            errors.append(f"{path}: value is above maximum")


def reconcile_card_summary(data, errors):
    if not isinstance(data.get("cards"), list) or not isinstance(data.get("summary"), dict):
        return
    counts = {}
    for card in data["cards"]:
        if isinstance(card, dict):
            status = card.get("status")
            counts[status] = counts.get(status, 0) + 1
    if data["summary"].get("total") != len(data["cards"]):
        errors.append("$.summary.total: does not match cards")
    for status in ["implemented", "blocked_engine_gap", "accepted_unsupported", "defect", "stub"]:
        if data["summary"].get(status) != counts.get(status, 0):
            errors.append(f"$.summary.{status}: does not match cards")


def reconcile_finding_summary(data, errors):
    if not isinstance(data.get("findings"), list) or not isinstance(data.get("summary"), dict):
        return
    counts = {}
    for finding in data["findings"]:
        if isinstance(finding, dict):
            severity = finding.get("severity")
            counts[severity] = counts.get(severity, 0) + 1
    if data["summary"].get("findings_total") != len(data["findings"]):
        errors.append("$.summary.findings_total: does not match findings")
    for severity in ["blocker", "error", "warning", "info"]:
        if data["summary"].get(severity) != counts.get(severity, 0):
            errors.append(f"$.summary.{severity}: does not match findings")


def reconcile_run_totals(data, errors):
    if not isinstance(data.get("runs"), list):
        return
    for index, run in enumerate(data["runs"]):
        if not isinstance(run, dict):
            continue
        total = sum(run.get(key, 0) for key in ["candidate_wins", "baseline_wins", "draws"])
        if run.get("total_games") != total:
            errors.append(f"$.runs[{index}].total_games: does not match results")


def main():
    if len(sys.argv) != 3:
        print("usage: validate_artifact.py SCHEMA ARTIFACT", file=sys.stderr)
        return 2
    try:
        schema = json.loads(Path(sys.argv[1]).read_text())
        artifact = json.loads(Path(sys.argv[2]).read_text())
    except (OSError, json.JSONDecodeError) as error:
        print(error, file=sys.stderr)
        return 2

    errors = []
    validate_node(schema, artifact, "$", errors)
    if schema.get("x-reconcile-card-summary"):
        reconcile_card_summary(artifact, errors)
    if schema.get("x-reconcile-finding-summary"):
        reconcile_finding_summary(artifact, errors)
    if schema.get("x-reconcile-run-totals"):
        reconcile_run_totals(artifact, errors)

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print("valid")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
