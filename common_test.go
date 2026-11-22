package fat

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/avpetkun/fat/internal/require"
	. "github.com/avpetkun/fat/internal/testutil"
)

// TestMontgomery checks montMul, montSqr and the forms on 64 and 128 bits
// with big.Int: a * b / R % m for odd m of every length, near R too,
// and a, b at random and next to m, where the last subtraction of m comes
// and goes and the sum carries past R
func TestMontgomery(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(3))
	one := big.NewInt(1)
	for _, w := range []int{64, 128, 256} {
		R := new(big.Int).Lsh(one, uint(w))
		for range 3000 {
			n := 2 + r.Intn(w-1)
			m := new(big.Int).Rand(r, new(big.Int).Lsh(one, uint(n)))
			m.SetBit(m, n-1, 1).SetBit(m, 0, 1)
			if r.Intn(4) == 0 {
				// R - 1 - 2j
				m.Sub(R, big.NewInt(1+2*r.Int63n(1000)))
			}
			rinv := new(big.Int).ModInverse(R, m)
			pick := func() *big.Int {
				switch r.Intn(5) {
				case 0:
					return new(big.Int).Sub(m, one)
				case 1:
					return new(big.Int).Sub(m, big.NewInt(2))
				case 2:
					return big.NewInt(int64(r.Intn(2)))
				}
				return new(big.Int).Rand(r, m)
			}
			a, b := pick(), pick()
			want := func(a, b *big.Int) *big.Int {
				v := new(big.Int).Mul(a, b)
				return v.Mod(v.Mul(v, rinv), m)
			}
			t := On(t, w, m, a, b)
			switch w {
			case 64:
				mp := montInv(m.Uint64())
				require.Big(t, want(a, b), new(big.Int).SetUint64(montMul64(a.Uint64(), b.Uint64(), m.Uint64(), mp)))
			case 128:
				x, y, mm := MustUint128FromBig(a), MustUint128FromBig(b), MustUint128FromBig(m)
				mp := montInv(mm.L0)
				require.Big(t, want(a, b), montMul128(x, y, mm, mp).Big())
				require.Big(t, want(a, a), montSqr128(x, mm, mp).Big())
			default:
				x, y, mm := MustUint256FromBig(a), MustUint256FromBig(b), MustUint256FromBig(m)
				mp := montInv(mm.L0)
				require.Big(t, want(a, b), montMul(x, y, mm, mp).Big())
				require.Big(t, want(a, a), montSqr(x, mm, mp).Big())
			}
		}
	}
}
