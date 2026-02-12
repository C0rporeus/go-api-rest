#!/usr/bin/env bash
set -euo pipefail

MIN_COVERAGE="${MIN_COVERAGE:-80}"
PROFILE_FILE="coverage.out"

TARGET_PACKAGES=(
  "./src/api/services"
  "./src/api/middlewares"
  "./src/pkg/apiresponse"
  "./src/pkg/telemetry"
  "./src/pkg/utils"
)

go test "${TARGET_PACKAGES[@]}" -coverprofile="${PROFILE_FILE}" >/dev/null

TOTAL_COVERAGE=$(go tool cover -func="${PROFILE_FILE}" | awk '/^total:/{print $3}' | tr -d '%')

if [[ -z "${TOTAL_COVERAGE}" ]]; then
  echo "No se pudo calcular la cobertura total."
  exit 1
fi

awk -v total="${TOTAL_COVERAGE}" -v min="${MIN_COVERAGE}" 'BEGIN {
  if (total + 0 < min + 0) {
    printf "Cobertura insuficiente: %.1f%% (minimo %.1f%%)\n", total, min
    exit 1
  }
}'

echo "Cobertura OK: ${TOTAL_COVERAGE}% (minimo ${MIN_COVERAGE}%)"
