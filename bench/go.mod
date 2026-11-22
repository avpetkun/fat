module github.com/avpetkun/fat/bench

go 1.24

toolchain go1.27.4

require (
	github.com/avpetkun/fat v0.0.0
	github.com/holiman/uint256 v1.3.3-0.20261117190858-760655f5bc8b
	lukechampine.com/uint128 v1.3.1-0.20241116023720-689f0db84623
)

replace github.com/avpetkun/fat => ../
