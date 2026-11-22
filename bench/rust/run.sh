#!/bin/sh
# Compares fat's Uint128 and Int128 + - * / % with Rust's u128 and i128 on
# the same inputs, needs go, cargo and benchstat:
#
#	bench/rust/run.sh [count] [milliseconds]
#
# count is the rounds per case (6), milliseconds the time of one (200).
# The results land in bench/results/rust, the summary also on stdout
set -eu

cd "$(dirname "$0")/.."
count=${1:-6}
ms=${2:-200}
out=results/rust
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out"

# the inputs and fat's results for the Rust side, then fat's own run
RUST_DATA=$tmp go test -run '^TestDumpRust$' -count 1 .
RUST_DATA=$tmp go test -run '^$' -bench '^BenchmarkRust' \
	-benchtime "${ms}ms" -count "$count" . >"$out/go.txt"

CARGO_TARGET_DIR=$tmp/target cargo build --quiet --release --manifest-path rust/Cargo.toml
"$tmp/target/release/fat-rust-bench" "$tmp" "$count" "$ms" >"$out/rust.txt"

grep -v '^PASS$\|^ok ' "$out/go.txt" >"$out/all.txt"
cat "$out/rust.txt" >>"$out/all.txt"
benchstat -col /impl "$out/all.txt" | tee "$out/summary.txt"
