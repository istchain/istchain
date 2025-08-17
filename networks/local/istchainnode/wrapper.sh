#!/usr/bin/env sh

BINARY=/istd/linux/${BINARY:-istd}
echo "binary: ${BINARY}"
ID=${ID:-0}
LOG=${LOG:-istd.log}

if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'istd' E.g.: -e BINARY=istd_my_test_version"
	exit 1
fi

BINARY_CHECK="$(file "$BINARY" | grep 'ELF 64-bit LSB executable, x86-64')"

if [ -z "${BINARY_CHECK}" ]; then
	echo "Binary needs to be OS linux, ARCH amd64"
	exit 1
fi

export ISTDHOME="/istd/node${ID}/istd"

if [ -d "$(dirname "${ISTDHOME}"/"${LOG}")" ]; then
  "${BINARY}" --home "${ISTDHOME}" "$@" | tee "${ISTDHOME}/${LOG}"
else
  "${BINARY}" --home "${ISTDHOME}" "$@"
fi
