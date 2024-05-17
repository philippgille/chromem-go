package simd

// cpuid is implemented in cpu_amd64.s
func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)

// xgetbv with ecx = 0 is implemented in cpu_amd64.s
func xgetbv() (eax, edx uint32)

// useSIMD reports the availability of FMA and AVX2 cpu features
func useSIMD() bool {
	var (
		hasAVX2 bool
		hasFMA  bool

		isSet = func(bitpos uint, value uint32) bool {
			return value&(1<<bitpos) != 0
		}
	)

	if maxID, _, _, _ := cpuid(0, 0); maxID < 7 {
		return false
	}
	_, _, ecx1, _ := cpuid(1, 0)

	hasFMA = isSet(12, ecx1)

	if hasOSXSAVE := isSet(27, ecx1); hasOSXSAVE {
		eax, _ := xgetbv() // For XGETBV, OSXSAVE bit is required and sufficient.
		_, ebx7, _, _ := cpuid(7, 0)
		hasAVX2 = isSet(1, eax) && isSet(2, eax) && isSet(5, ebx7)
	}

	return hasAVX2 && hasFMA
}
