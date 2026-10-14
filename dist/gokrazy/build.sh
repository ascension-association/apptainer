#!/bin/sh
# Copyright (c) Contributors to the Apptainer project, established as
#   Apptainer a Series of LF Projects LLC.
#   For website terms of use, trademark policy, privacy policy and other
#   project policies see https://lfprojects.org/policies
#
# Build the gokrazy extrafiles tarballs containing a statically linked
# apptainer installation, for each architecture given as argument
# (default: amd64 arm64).
#
# Usage: dist/gokrazy/build.sh [amd64] [arm64]

set -e

here=$(cd "$(dirname "$0")" && pwd)
toplevel=$(cd "${here}/../.." && pwd)
outdir="${here}/apptainer/_gokrazy"

cd "${toplevel}"

version=$(scripts/get-version)
golang_version=$(sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' go.mod)

if [ $# -eq 0 ]; then
	set -- amd64 arm64
fi

mkdir -p "${outdir}"

for arch in "$@"; do
	echo "Building apptainer ${version} for gokrazy linux/${arch}"
	tmp=$(mktemp -d)
	docker buildx build \
		--platform "linux/${arch}" \
		--build-arg "GOLANG_VERSION=${golang_version}" \
		--build-arg "VERSION=${version}" \
		--file "${here}/Dockerfile" \
		--output "type=local,dest=${tmp}" \
		.
	mv "${tmp}/extrafiles.tar" "${outdir}/extrafiles_${arch}.tar"
	rm -rf "${tmp}"
	echo "Wrote ${outdir}/extrafiles_${arch}.tar"
done
