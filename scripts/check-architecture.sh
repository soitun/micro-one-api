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
      relative="${package_path#${module_path}/app/}"
      service_root="${module_path}/app/$(printf '%s' "${relative}" | cut -d/ -f1,2)"
      ;;
  esac

  for imported in ${imports}; do
    # Rule 1: an app subtree must not import another app subtree's implementation.
    # Exception: testutil packages are explicitly designed for cross-app test
    # sharing; they re-export a curated subset of types for integration tests.
    if [[ -n "${service_root}" \
          && "${imported}" == "${module_path}/app/"* \
          && "${imported}" != "${service_root}" \
          && "${imported}" != "${service_root}/"* \
          && "${imported}" != */testutil \
          && "${imported}" != */testutil/* ]]; then
      echo "${package_path} imports another app implementation: ${imported}"
      violations=1
    fi

    # Rule 2: migrated app code must not re-import legacy root-level internal/<service>.
    if [[ "${package_path}" == "${module_path}/app/"* \
          && "${imported}" =~ ^${module_path}/internal/(admin|billing|channel|config|identity|log|monitor|notify|relay|subscription)(/|$) ]]; then
      echo "${package_path} imports legacy service implementation: ${imported}"
      violations=1
    fi

    # Rule 3: platform/pkg/domain must not reverse-import app.
    if [[ "${package_path}" =~ ^${module_path}/(platform|pkg|domain)/ \
          && "${imported}" == "${module_path}/app/"* ]]; then
      echo "${package_path} has reverse dependency on app: ${imported}"
      violations=1
    fi

    # Rule 4: pkg must remain a pure utility layer (no platform/domain imports).
    if [[ "${package_path}" == "${module_path}/pkg/"* \
          && "${imported}" =~ ^${module_path}/(platform|domain)/ ]]; then
      echo "${package_path} is not a pure utility package: ${imported}"
      violations=1
    fi
  done
done < <(go list -e -f '{{.ImportPath}}|{{join .Imports " "}}' ./app/... ./platform/... ./pkg/... ./domain/... 2>/dev/null || true)

# Guard the Phase 0 boundary for any remaining legacy internal/ packages.
while IFS='|' read -r package_path imports; do
  for imported in ${imports}; do
    if [[ "${package_path}" =~ ^${module_path}/internal/(admin|channel|monitor|pkg)(/|$) \
          && "${imported}" =~ ^${module_path}/internal/(billing|relay|subscription)(/|$) ]]; then
      echo "${package_path} cross-domain import of service implementation: ${imported}"
      violations=1
    fi
  done
done < <(go list -e -f '{{.ImportPath}}|{{join .Imports " "}}' ./internal/... 2>/dev/null || true)

exit "${violations}"
