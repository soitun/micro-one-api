#!/usr/bin/env bash
set -euo pipefail

module_path="$(go list -m)"
violations=0

# Surface any compile errors or internal-visibility violations before the
# import analysis below; do not swallow go list failures.
go list -deps ./... >/dev/null

while IFS='|' read -r package_path imports; do
  service_root=""
  case "${package_path}" in
    "${module_path}/app/"*)
      # Flat structure: app/<service>/ — service root is the first segment.
      relative="${package_path#${module_path}/app/}"
      service_root="${module_path}/app/$(printf '%s' "${relative}" | cut -d/ -f1)"
      ;;
    "${module_path}/internal/"*)
      # Root internal/ belongs to the main project (relay-gateway).
      service_root="${module_path}/internal"
      ;;
  esac

  for imported in ${imports}; do
    # Rule 1: an app subtree must not import another app subtree's implementation.
    # The root internal/ (relay-gateway) is also treated as an app subtree.
    # Exception: testutil packages are explicitly designed for cross-app test
    # sharing; they re-export a curated subset of types for integration tests.
    if [[ -n "${service_root}" \
          && ( "${imported}" == "${module_path}/app/"* || "${imported}" == "${module_path}/internal/"* ) \
          && "${imported}" != "${service_root}" \
          && "${imported}" != "${service_root}/"* \
          && "${imported}" != */testutil \
          && "${imported}" != */testutil/* ]]; then
      echo "${package_path} imports another app implementation: ${imported}"
      violations=1
    fi

    # Rule 2: platform/pkg/domain must not reverse-import app or root internal.
    if [[ "${package_path}" =~ ^${module_path}/(platform|pkg|domain)/ \
          && ( "${imported}" == "${module_path}/app/"* || "${imported}" == "${module_path}/internal/"* ) ]]; then
      echo "${package_path} has reverse dependency on app: ${imported}"
      violations=1
    fi

    # Rule 3: pkg must remain a pure utility layer (no platform/domain imports).
    if [[ "${package_path}" == "${module_path}/pkg/"* \
          && "${imported}" =~ ^${module_path}/(platform|domain)/ ]]; then
      echo "${package_path} is not a pure utility package: ${imported}"
      violations=1
    fi
  done
done < <(go list -e -f '{{.ImportPath}}|{{join .Imports " "}}' ./app/... ./internal/... ./platform/... ./pkg/... ./domain/... 2>/dev/null || true)

exit "${violations}"
