// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"text/template"
)

const (
	copyright = "" +
		"// Copyright 2016 The Go Authors. All rights reserved.\n" +
		"// Use of this source code is governed by a BSD-style\n" +
		"// license that can be found in the LICENSE file.\n"

	doNotEdit = "// generated by go run gen.go; DO NOT EDIT\n"

	dashDashDash = "// --------"
)

func main() {
	tmpl, err := ioutil.ReadFile("gen_acc_amd64.s.tmpl")
	if err != nil {
		log.Fatalf("ReadFile: %v", err)
	}
	if !bytes.HasPrefix(tmpl, []byte(copyright)) {
		log.Fatal("source template did not start with the copyright header")
	}
	tmpl = tmpl[len(copyright):]

	preamble := []byte(nil)
	if i := bytes.Index(tmpl, []byte(dashDashDash)); i < 0 {
		log.Fatalf("source template did not contain %q", dashDashDash)
	} else {
		preamble, tmpl = tmpl[:i], tmpl[i:]
	}

	t, err := template.New("").Parse(string(tmpl))
	if err != nil {
		log.Fatalf("Parse: %v", err)
	}

	out := bytes.NewBuffer(nil)
	out.WriteString(doNotEdit)
	out.Write(preamble)

	for i, v := range instances {
		if i != 0 {
			out.WriteString("\n")
		}
		if err := t.Execute(out, v); err != nil {
			log.Fatalf("Execute(%q): %v", v.ShortName, err)
		}
	}

	if err := ioutil.WriteFile("acc_amd64.s", out.Bytes(), 0666); err != nil {
		log.Fatalf("WriteFile: %v", err)
	}
}

var instances = []struct {
	LongName       string
	ShortName      string
	FrameSize      string
	SrcType        string
	XMM3           string
	XMM4           string
	XMM5           string
	XMM8           string
	XMM9           string
	XMM10          string
	Setup          string
	Cleanup        string
	LoadXMMRegs    string
	Add            string
	ClampAndScale  string
	ConvertToInt32 string
	Store4         string
	Store1         string
}{{
	LongName:       "fixedAccumulateOpOver",
	ShortName:      "fxAccOpOver",
	FrameSize:      fxFrameSize,
	SrcType:        fxSrcType,
	XMM3:           fxXMM3,
	XMM4:           fxXMM4,
	XMM5:           fxXMM5_65536,
	XMM8:           opOverXMM8,
	XMM9:           opOverXMM9,
	XMM10:          opOverXMM10,
	Setup:          fxSetup,
	LoadXMMRegs:    fxLoadXMMRegs65536 + "\n" + opOverLoadXMMRegs,
	Cleanup:        fxCleanup,
	Add:            fxAdd,
	ClampAndScale:  fxClampAndScale65536,
	ConvertToInt32: fxConvertToInt32,
	Store4:         opOverStore4,
	Store1:         opOverStore1,
}, {
	LongName:       "fixedAccumulateOpSrc",
	ShortName:      "fxAccOpSrc",
	FrameSize:      fxFrameSize,
	SrcType:        fxSrcType,
	XMM3:           fxXMM3,
	XMM4:           fxXMM4,
	XMM5:           fxXMM5_256,
	XMM8:           opSrcXMM8,
	XMM9:           opSrcXMM9,
	XMM10:          opSrcXMM10,
	Setup:          fxSetup,
	LoadXMMRegs:    fxLoadXMMRegs256 + "\n" + opSrcLoadXMMRegs,
	Cleanup:        fxCleanup,
	Add:            fxAdd,
	ClampAndScale:  fxClampAndScale256,
	ConvertToInt32: fxConvertToInt32,
	Store4:         opSrcStore4,
	Store1:         opSrcStore1,
}, {
	LongName:       "floatingAccumulateOpOver",
	ShortName:      "flAccOpOver",
	FrameSize:      flFrameSize,
	SrcType:        flSrcType,
	XMM3:           flXMM3_65536,
	XMM4:           flXMM4,
	XMM5:           flXMM5,
	XMM8:           opOverXMM8,
	XMM9:           opOverXMM9,
	XMM10:          opOverXMM10,
	Setup:          flSetup,
	LoadXMMRegs:    flLoadXMMRegs65536 + "\n" + opOverLoadXMMRegs,
	Cleanup:        flCleanup,
	Add:            flAdd,
	ClampAndScale:  flClampAndScale65536,
	ConvertToInt32: flConvertToInt32,
	Store4:         opOverStore4,
	Store1:         opOverStore1,
}, {
	LongName:       "floatingAccumulateOpSrc",
	ShortName:      "flAccOpSrc",
	FrameSize:      flFrameSize,
	SrcType:        flSrcType,
	XMM3:           flXMM3_256,
	XMM4:           flXMM4,
	XMM5:           flXMM5,
	XMM8:           opSrcXMM8,
	XMM9:           opSrcXMM9,
	XMM10:          opSrcXMM10,
	Setup:          flSetup,
	LoadXMMRegs:    flLoadXMMRegs256 + "\n" + opSrcLoadXMMRegs,
	Cleanup:        flCleanup,
	Add:            flAdd,
	ClampAndScale:  flClampAndScale256,
	ConvertToInt32: flConvertToInt32,
	Store4:         opSrcStore4,
	Store1:         opSrcStore1,
}}

const (
	fxFrameSize = `0`
	flFrameSize = `8`

	fxSrcType = `[]uint32`
	flSrcType = `[]float32`

	fxXMM3       = `-`
	flXMM3_256   = `flAlmost256`
	flXMM3_65536 = `flAlmost65536`

	fxXMM4 = `-`
	flXMM4 = `flOne`

	fxXMM5_256   = `fxAlmost256`
	fxXMM5_65536 = `fxAlmost65536`
	flXMM5       = `flSignMask`

	fxSetup = ``
	flSetup = `
		// Set MXCSR bits 13 and 14, so that the CVTPS2PL below is "Round To Zero".
		STMXCSR mxcsrOrig-8(SP)
		MOVL    mxcsrOrig-8(SP), AX
		ORL     $0x6000, AX
		MOVL    AX, mxcsrNew-4(SP)
		LDMXCSR mxcsrNew-4(SP)
		`

	fxCleanup = `// No-op.`
	flCleanup = `LDMXCSR mxcsrOrig-8(SP)`

	fxLoadXMMRegs256 = `
		// fxAlmost256 := XMM(0x000000ff repeated four times) // Maximum of an uint8.
		MOVOU fxAlmost256<>(SB), X5
		`
	fxLoadXMMRegs65536 = `
		// fxAlmost65536 := XMM(0x0000ffff repeated four times) // Maximum of an uint16.
		MOVOU fxAlmost65536<>(SB), X5
		`
	flLoadXMMRegs256 = `
		// flAlmost256 := XMM(0x437fffff repeated four times) // 255.99998 as a float32.
		// flOne       := XMM(0x3f800000 repeated four times) // 1 as a float32.
		// flSignMask  := XMM(0x7fffffff repeated four times) // All but the sign bit of a float32.
		MOVOU flAlmost256<>(SB), X3
		MOVOU flOne<>(SB), X4
		MOVOU flSignMask<>(SB), X5
		`
	flLoadXMMRegs65536 = `
		// flAlmost65536 := XMM(0x477fffff repeated four times) // 255.99998 * 256 as a float32.
		// flOne         := XMM(0x3f800000 repeated four times) // 1 as a float32.
		// flSignMask    := XMM(0x7fffffff repeated four times) // All but the sign bit of a float32.
		MOVOU flAlmost65536<>(SB), X3
		MOVOU flOne<>(SB), X4
		MOVOU flSignMask<>(SB), X5
		`

	fxAdd = `PADDD`
	flAdd = `ADDPS`

	fxClampAndScale256 = `
		// y = abs(x)
		// y >>= 12 // Shift by 2*ϕ - 8.
		// y = min(y, fxAlmost256)
		//
		// pabsd  %xmm1,%xmm2
		// psrld  $0xc,%xmm2
		// pminud %xmm5,%xmm2
		//
		// Hopefully we'll get these opcode mnemonics into the assembler for Go
		// 1.8. https://golang.org/issue/16007 isn't exactly the same thing, but
		// it's similar.
		BYTE $0x66; BYTE $0x0f; BYTE $0x38; BYTE $0x1e; BYTE $0xd1
		BYTE $0x66; BYTE $0x0f; BYTE $0x72; BYTE $0xd2; BYTE $0x0c
		BYTE $0x66; BYTE $0x0f; BYTE $0x38; BYTE $0x3b; BYTE $0xd5
		`
	fxClampAndScale65536 = `
		// y = abs(x)
		// y >>= 4 // Shift by 2*ϕ - 16.
		// y = min(y, fxAlmost65536)
		//
		// pabsd  %xmm1,%xmm2
		// psrld  $0x4,%xmm2
		// pminud %xmm5,%xmm2
		//
		// Hopefully we'll get these opcode mnemonics into the assembler for Go
		// 1.8. https://golang.org/issue/16007 isn't exactly the same thing, but
		// it's similar.
		BYTE $0x66; BYTE $0x0f; BYTE $0x38; BYTE $0x1e; BYTE $0xd1
		BYTE $0x66; BYTE $0x0f; BYTE $0x72; BYTE $0xd2; BYTE $0x04
		BYTE $0x66; BYTE $0x0f; BYTE $0x38; BYTE $0x3b; BYTE $0xd5
		`
	flClampAndScale256 = `
		// y = x & flSignMask
		// y = min(y, flOne)
		// y = mul(y, flAlmost256)
		MOVOU X5, X2
		ANDPS X1, X2
		MINPS X4, X2
		MULPS X3, X2
		`
	flClampAndScale65536 = `
		// y = x & flSignMask
		// y = min(y, flOne)
		// y = mul(y, flAlmost65536)
		MOVOU X5, X2
		ANDPS X1, X2
		MINPS X4, X2
		MULPS X3, X2
		`

	fxConvertToInt32 = `// No-op.`
	flConvertToInt32 = `CVTPS2PL X2, X2`

	opOverStore4 = `
		// Blend over the dst's prior value. SIMD for i in 0..3:
		//
		// dstA := uint32(dst[i]) * 0x101
		// maskA := z@i
		// outA := dstA*(0xffff-maskA)/0xffff + maskA
		// dst[i] = uint8(outA >> 8)
		//
		// First, set X0 to dstA*(0xfff-maskA).
		MOVL   (DI), X0
		PSHUFB X8, X0
		MOVOU  X9, X11
		PSUBL  X2, X11
		PMULLD X11, X0
		// We implement uint32 division by 0xffff as multiplication by a magic
		// constant (0x800080001) and then a shift by a magic constant (47).
		// See TestDivideByFFFF for a justification.
		//
		// That multiplication widens from uint32 to uint64, so we have to
		// duplicate and shift our four uint32s from one XMM register (X0) to
		// two XMM registers (X0 and X11).
		//
		// Move the second and fourth uint32s in X0 to be the first and third
		// uint32s in X11.
		MOVOU X0, X11
		PSRLQ $32, X11
		// Multiply by magic, shift by magic.
		//
		// pmuludq %xmm10,%xmm0
		// pmuludq %xmm10,%xmm11
		BYTE  $0x66; BYTE $0x41; BYTE $0x0f; BYTE $0xf4; BYTE $0xc2
		BYTE  $0x66; BYTE $0x45; BYTE $0x0f; BYTE $0xf4; BYTE $0xda
		PSRLQ $47, X0
		PSRLQ $47, X11
		// Merge the two registers back to one, X11.
		PSLLQ $32, X11
		XORPS X0, X11
		// Add maskA, shift from 16 bit color to 8 bit color.
		PADDD  X11, X2
		PSRLQ  $8, X2
		// As per opSrcStore4, shuffle and copy the low 4 bytes.
		PSHUFB X6, X2
		MOVL   X2, (DI)
		`
	opSrcStore4 = `
		// z = shuffleTheLowBytesOfEach4ByteElement(z)
		// copy(dst[:4], low4BytesOf(z))
		PSHUFB X6, X2
		MOVL   X2, (DI)
		`

	opOverStore1 = `
		// Blend over the dst's prior value.
		//
		// dstA := uint32(dst[0]) * 0x101
		// maskA := z
		// outA := dstA*(0xffff-maskA)/0xffff + maskA
		// dst[0] = uint8(outA >> 8)
		MOVBLZX (DI), R12
		IMULL   $0x101, R12
		MOVL    X2, R13
		MOVL    $0xffff, AX
		SUBL    R13, AX
		MULL    R12             // MULL's implicit arg is AX, and the result is stored in DX:AX.
		MOVL    $0x80008001, BX // Divide by 0xffff is to first multiply by a magic constant...
		MULL    BX              // MULL's implicit arg is AX, and the result is stored in DX:AX.
		SHRL    $15, DX         // ...and then shift by another magic constant (47 - 32 = 15).
		ADDL    DX, R13
		SHRL    $8, R13
		MOVB    R13, (DI)
		`
	opSrcStore1 = `
		// dst[0] = uint8(z)
		MOVL X2, BX
		MOVB BX, (DI)
		`

	opOverXMM8 = `scatterAndMulBy0x101`
	opSrcXMM8  = `-`

	opOverXMM9 = `fxAlmost65536`
	opSrcXMM9  = `-`

	opOverXMM10 = `inverseFFFF`
	opSrcXMM10  = `-`

	opOverLoadXMMRegs = `
		// scatterAndMulBy0x101 := XMM(see above)                      // PSHUFB shuffle mask.
		// fxAlmost65536        := XMM(0x0000ffff repeated four times) // 0xffff.
		// inverseFFFF          := XMM(0x80008001 repeated four times) // Magic constant for dividing by 0xffff.
		MOVOU scatterAndMulBy0x101<>(SB), X8
		MOVOU fxAlmost65536<>(SB), X9
		MOVOU inverseFFFF<>(SB), X10
		`
	opSrcLoadXMMRegs = ``
)