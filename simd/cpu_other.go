//go:build !amd64

package simd

func useSIMD() bool {
	return false
}
