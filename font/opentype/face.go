// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package opentype

import (
	"image"
	"image/draw"

	"github.com/joeblubaugh/image/font"
	"github.com/joeblubaugh/image/font/sfnt"
	"github.com/joeblubaugh/image/math/fixed"
	"github.com/joeblubaugh/image/vector"
)

// FaceOptions describes the possible options given to NewFace when
// creating a new font.Face from a sfnt.Font.
type FaceOptions struct {
	Size    float64      // Size is the font size in points
	DPI     float64      // DPI is the dots per inch resolution
	Hinting font.Hinting // Hinting selects how to quantize a vector font's glyph nodes
}

func defaultFaceOptions() *FaceOptions {
	return &FaceOptions{
		Size:    12,
		DPI:     72,
		Hinting: font.HintingNone,
	}
}

// Face implements the font.Face interface for sfnt.Font values.
type Face struct {
	f       *sfnt.Font
	hinting font.Hinting
	scale   fixed.Int26_6

	metrics    font.Metrics
	metricsSet bool

	buf  sfnt.Buffer
	rast vector.Rasterizer
	mask image.Alpha
}

// NewFace returns a new font.Face for the given sfnt.Font.
// if opts is nil, sensible defaults will be used.
func NewFace(f *sfnt.Font, opts *FaceOptions) (font.Face, error) {
	if opts == nil {
		opts = defaultFaceOptions()
	}
	face := &Face{
		f:       f,
		hinting: opts.Hinting,
		scale:   fixed.Int26_6(0.5 + (opts.Size * opts.DPI * 64 / 72)),
	}
	return face, nil
}

// Close satisfies the font.Face interface.
func (f *Face) Close() error {
	return nil
}

// Metrics satisfies the font.Face interface.
func (f *Face) Metrics() font.Metrics {
	if !f.metricsSet {
		var err error
		f.metrics, err = f.f.Metrics(&f.buf, f.scale, f.hinting)
		if err != nil {
			f.metrics = font.Metrics{}
		}
		f.metricsSet = true
	}
	return f.metrics
}

// Kern satisfies the font.Face interface.
func (f *Face) Kern(r0, r1 rune) fixed.Int26_6 {
	x0 := f.index(r0)
	x1 := f.index(r1)
	k, err := f.f.Kern(&f.buf, x0, x1, fixed.Int26_6(f.f.UnitsPerEm()), f.hinting)
	if err != nil {
		return 0
	}
	return k
}

// Glyph satisfies the font.Face interface.
func (f *Face) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {

	x, err := f.f.GlyphIndex(&f.buf, r)
	if err != nil {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}

	segments, err := f.f.LoadGlyph(&f.buf, x, f.scale, nil)
	if err != nil {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}

	bounds, advance, ok := f.GlyphBounds(r)

	// Calculate the integer pixel bounds for rasterization.
	xmin := bounds.Min.X.Floor()
	ymin := bounds.Min.Y.Floor()
	xmax := bounds.Max.X.Ceil()
	ymax := bounds.Max.Y.Ceil()

	width, height := xmax-xmin, ymax-ymin

	// Rasterizer always starts at (0,0). Shift the
	// origin so that xmin,ymin is 0,0
	originX := float32(0 - xmin)
	originY := float32(0 - ymin)

	f.rast.Reset(width, height)
	f.rast.DrawOp = draw.Src
	for _, seg := range segments {
		// The divisions by 64 below is because the seg.Args values have type
		// fixed.Int26_6, a 26.6 fixed point number, and 1<<6 == 64.
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			f.rast.MoveTo(
				originX+float32(seg.Args[0].X)/64,
				originY+float32(seg.Args[0].Y)/64,
			)
		case sfnt.SegmentOpLineTo:
			f.rast.LineTo(
				originX+float32(seg.Args[0].X)/64,
				originY+float32(seg.Args[0].Y)/64,
			)
		case sfnt.SegmentOpQuadTo:
			f.rast.QuadTo(
				originX+float32(seg.Args[0].X)/64,
				originY+float32(seg.Args[0].Y)/64,
				originX+float32(seg.Args[1].X)/64,
				originY+float32(seg.Args[1].Y)/64,
			)
		case sfnt.SegmentOpCubeTo:
			f.rast.CubeTo(
				originX+float32(seg.Args[0].X)/64,
				originY+float32(seg.Args[0].Y)/64,
				originX+float32(seg.Args[1].X)/64,
				originY+float32(seg.Args[1].Y)/64,
				originX+float32(seg.Args[2].X)/64,
				originY+float32(seg.Args[2].Y)/64,
			)
		}
	}

	npix := width * height
	if cap(f.mask.Pix) < npix {
		f.mask.Pix = make([]uint8, 2*npix)
	}
	f.mask.Pix = f.mask.Pix[:npix]
	f.mask.Stride = width
	f.mask.Rect.Min.X = 0
	f.mask.Rect.Min.Y = 0
	f.mask.Rect.Max.X = width
	f.mask.Rect.Max.Y = height

	f.rast.Draw(&f.mask, f.mask.Bounds(), image.Opaque, image.Point{})

	// get the integer part of the dot.
	ix := dot.X.Floor()
	iy := dot.Y.Floor()
	dr.Min = image.Point{X: ix + xmin, Y: iy + ymin}
	dr.Max = image.Point{X: dr.Min.X + width, Y: dr.Min.Y + height}

	return dr, &f.mask, f.mask.Rect.Min, advance, true
}

// GlyphBounds satisfies the font.Face interface.
func (f *Face) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	idx := f.index(r)
	bounds, advance, err := f.f.GlyphBounds(&f.buf, idx, f.scale, f.hinting)
	return bounds, advance, err == nil
}

// GlyphAdvance satisfies the font.Face interface.
func (f *Face) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	idx := f.index(r)
	advance, err := f.f.GlyphAdvance(&f.buf, idx, f.scale, f.hinting)
	return advance, err == nil
}

func (f *Face) index(r rune) sfnt.GlyphIndex {
	x, _ := f.f.GlyphIndex(&f.buf, r)
	return x
}
