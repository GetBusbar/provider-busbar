#!/bin/sh
# Build the exact upstream source used by schema acquisition and provider images.
set -eu
export GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=0

: "${TERRAFORM_PROVIDER_REPO:?required}"
: "${TERRAFORM_PROVIDER_REF:?required}"
: "${TERRAFORM_PROVIDER_VERSION:?required}"
: "${TERRAFORM_PROVIDER_GO_VERSION:?required}"
source_dir=${1:?source checkout directory required}
binary_path=${2:?output binary path required}

test "${#TERRAFORM_PROVIDER_REF}" = 40
case "${TERRAFORM_PROVIDER_REF}" in *[!0-9a-f]*) exit 1 ;; esac
case "${source_dir}" in /*) ;; *) echo "source directory must be absolute" >&2; exit 1 ;; esac
case "${binary_path}" in /*) ;; *) echo "binary path must be absolute" >&2; exit 1 ;; esac
test "$(go env GOVERSION)" = "go${TERRAFORM_PROVIDER_GO_VERSION}"

mkdir -p "${source_dir}"
if ! test -d "${source_dir}/.git"; then
    git -C "${source_dir}" init
    git -C "${source_dir}" remote add origin "${TERRAFORM_PROVIDER_REPO}"
fi
test "$(git -C "${source_dir}" remote get-url origin)" = "${TERRAFORM_PROVIDER_REPO}"
test -z "$(git -C "${source_dir}" status --porcelain --untracked-files=normal)"
if test "$(git -C "${source_dir}" config --get core.sparseCheckout || true)" = true; then
    git -C "${source_dir}" sparse-checkout disable
fi
if ! git -C "${source_dir}" cat-file -e "${TERRAFORM_PROVIDER_REF}^{commit}" 2>/dev/null; then
    git -C "${source_dir}" fetch --depth=1 origin "${TERRAFORM_PROVIDER_REF}"
fi
git -C "${source_dir}" checkout --detach "${TERRAFORM_PROVIDER_REF}"
test "$(git -C "${source_dir}" rev-parse HEAD)" = "${TERRAFORM_PROVIDER_REF}"
mkdir -p "$(dirname "${binary_path}")"
GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=0 \
    go -C "${source_dir}" build -mod=readonly -trimpath \
    -ldflags "-s -w -X main.version=${TERRAFORM_PROVIDER_VERSION}" \
    -o "${binary_path}" .
git -C "${source_dir}" diff --exit-code -- go.mod go.sum
