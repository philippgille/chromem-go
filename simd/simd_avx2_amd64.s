/* This file is a partial copy of https://github.com/viterin/vek/blob/v0.4.2/internal/functions/accel_avx2_amd64.s.
Here is its license, which only applies to the copied parts and not to the rest of chromem-go,
which is licensed under the GNU Affero General Public License.

MIT License

Copyright (c) 2022 viterin

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

#include "textflag.h"

// func Dot_AVX2_F32(x []float32, y []float32) float32
// Requires: AVX, FMA3, SSE
TEXT ·Dot_AVX2_F32(SB), NOSPLIT, $0-52
	MOVQ   x_base+0(FP), DI
	MOVQ   y_base+24(FP), SI
	MOVQ   x_len+8(FP), DX
	TESTQ  DX, DX
	JE     LBB1_1
	CMPQ   DX, $0x20
	JAE    LBB1_4
	VXORPS X0, X0, X0
	XORL   AX, AX
	JMP    LBB1_7

LBB1_1:
	VXORPS X0, X0, X0
	MOVSS  X0, ret+48(FP)
	RET

LBB1_4:
	MOVQ   DX, AX
	ANDQ   $-32, AX
	VXORPS X0, X0, X0
	XORL   CX, CX
	VXORPS X1, X1, X1
	VXORPS X2, X2, X2
	VXORPS X3, X3, X3

LBB1_5:
	VMOVUPS      (SI)(CX*4), Y4
	VMOVUPS      32(SI)(CX*4), Y5
	VMOVUPS      64(SI)(CX*4), Y6
	VMOVUPS      96(SI)(CX*4), Y7
	VFMADD231PS  (DI)(CX*4), Y4, Y0
	VFMADD231PS  32(DI)(CX*4), Y5, Y1
	VFMADD231PS  64(DI)(CX*4), Y6, Y2
	VFMADD231PS  96(DI)(CX*4), Y7, Y3
	ADDQ         $0x20, CX
	CMPQ         AX, CX
	JNE          LBB1_5
	VADDPS       Y0, Y1, Y0
	VADDPS       Y0, Y2, Y0
	VADDPS       Y0, Y3, Y0
	VEXTRACTF128 $0x01, Y0, X1
	VADDPS       X1, X0, X0
	VPERMILPD    $0x01, X0, X1
	VADDPS       X1, X0, X0
	VMOVSHDUP    X0, X1
	VADDSS       X1, X0, X0
	CMPQ         AX, DX
	JE           LBB1_8

LBB1_7:
	VMOVSS      (SI)(AX*4), X1
	VFMADD231SS (DI)(AX*4), X1, X0
	ADDQ        $0x01, AX
	CMPQ        DX, AX
	JNE         LBB1_7

LBB1_8:
	VZEROUPPER
	MOVSS X0, ret+48(FP)
	RET
