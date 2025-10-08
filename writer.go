// Copyright 2014 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo
// +build cgo

package webp

import (
	"image"
	"image/color"
	"io"
	"os"
	"reflect"
)

const DefaulQuality = 90

// Options are the encoding parameters.
// Advanced options (Method and below) are only used when encoding RGBA images.
// For Gray and RGB images, only Lossless and Quality are used.
type Options struct {
	Lossless bool
	Quality  float32 // 0 ~ 100

	// Advanced encoding options (RGBA only)
	// Quality/speed trade-off (0=fast, 6=slower-better). Default: 4
	Method int
	// Hint for image type (lossless: 0=default, 1=picture, 2=photo, 3=graphic; lossy: ignored). Default: 0
	ImageHint int
	// If non-zero, set the desired target size in bytes.
	// Takes precedence over the 'compression' parameter. Default: 0
	TargetSize int
	// If non-zero, specifies the minimal distortion to
	// try to achieve. Takes precedence over target_size. Default: 0
	TargetPsnr float32
	// Maximum number of segments to use, in [1..4]. Default: 2
	Segments int
	// Spatial Noise Shaping. 0=off, 100=maximum. Default: 100
	SnsStrength int
	// Range: [0 = off .. 100 = strongest]. Default: 100
	FilterStrength int
	// Range: [0 = off .. 7 = least sharp]. Default: 0
	FilterSharpness int
	// Filtering type: 0 = simple, 1 = strong (only used
	// if filter_strength > 0 or autofilter > 0). Default: 1
	FilterType int
	// Auto adjust filter's strength [0 = off, 1 = on]. Default: false
	Autofilter bool
	// Algorithm for encoding the alpha plane (0 = none,
	// 1 = compressed with WebP lossless). Default: 1
	AlphaCompression int
	// Predictive filtering method for alpha plane.
	// 0: none, 1: fast, 2: best. Default: 1
	AlphaFiltering int
	// Number of entropy-analysis passes (in [1..10]). Default: 1
	Pass int
	// If true, export the compressed picture back.
	// In-loop filtering is not applied. Default: false
	ShowCompressed bool
	// Preprocessing filter (0=none, 1=segment-smooth). Default: 1
	Preprocessing int
	// Log2(number of token partitions) in [0..3]. Default: 0
	Partitions int
	// Quality degradation allowed to fit the 512k limit on
	// prediction modes coding (0: no degradation,
	// 100: maximum possible degradation). Default: 0
	PartitionLimit int
	// If true, compression parameters will be remapped
	// to better match the expected output size from
	// JPEG compression. Generally, the output size will
	// be similar but the degradation will be lower. Default: false
	EmulateJpegSize bool
	// If true, try and use multi-threaded encoding. Default: false
	ThreadLevel bool
	// If set, reduce memory usage (but increase CPU use). Default: false
	LowMemory bool
	// Near lossless encoding [0 = max loss .. 100 = off (default)]. Lossless mode only. Default: 100
	NearLossless int
	// If non-zero, preserve the exact RGB values under transparent area.
	// Otherwise, discard this invisible RGB information for better compression.
	// Lossless mode only. The default value is 0.
	Exact int
	// Reserved for future lossless feature. Default: false
	UseDeltaPalette bool
	// If needed, use sharp (and slow) RGB->YUV conversion. Default: true
	UseSharpYuv bool
}

type colorModeler interface {
	ColorModel() color.Model
}

// applyDefaults fills in default values for unset fields in Options.
// Note: Zero values (0, false) are treated as "unset" and will be replaced with defaults.
// If you need to explicitly set a parameter to 0, this function will override it.
// This is a known limitation of the current design.
func applyDefaults(opt *Options) *Options {
	if opt == nil {
		opt = &Options{}
	}
	// Apply defaults based on WebP library defaults
	if opt.Quality == 0 {
		opt.Quality = DefaulQuality
	}
	if opt.Method == 0 {
		opt.Method = 4
	}
	if opt.Segments == 0 {
		opt.Segments = 2
	}
	if opt.SnsStrength == 0 {
		opt.SnsStrength = 100
	}
	if opt.FilterStrength == 0 {
		opt.FilterStrength = 100
	}
	if opt.FilterType == 0 {
		opt.FilterType = 1
	}
	if opt.AlphaCompression == 0 {
		opt.AlphaCompression = 1
	}
	if opt.AlphaFiltering == 0 {
		opt.AlphaFiltering = 1
	}
	if opt.Pass == 0 {
		opt.Pass = 1
	}
	if opt.Preprocessing == 0 {
		opt.Preprocessing = 1
	}
	if opt.NearLossless == 0 {
		opt.NearLossless = 100
	}
	if !opt.UseSharpYuv {
		opt.UseSharpYuv = true
	}
	return opt
}

// hasAdvancedOptions checks if Options has any advanced encoding parameters set.
// It compares against default values to determine if advanced configuration is needed.
func hasAdvancedOptions(opt *Options) bool {
	if opt == nil {
		return false
	}
	// Check if any advanced option differs from its default value
	// Note: Zero values for numeric fields are treated as "not set" and will be filled by applyDefaults
	return opt.Method != 0 || opt.ImageHint != 0 || opt.TargetSize != 0 || opt.TargetPsnr != 0 ||
		opt.Segments != 0 || opt.SnsStrength != 0 || opt.FilterStrength != 0 || opt.FilterSharpness != 0 ||
		opt.FilterType != 0 || opt.Autofilter || opt.AlphaCompression != 0 || opt.AlphaFiltering != 0 ||
		opt.Pass != 0 || opt.ShowCompressed || opt.Preprocessing != 0 || opt.Partitions != 0 ||
		opt.PartitionLimit != 0 || opt.EmulateJpegSize || opt.ThreadLevel || opt.LowMemory ||
		opt.NearLossless != 0 || opt.Exact != 0 || opt.UseDeltaPalette || opt.UseSharpYuv
}

func Save(name string, m image.Image, opt *Options) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()

	return encode(f, m, opt)
}

// Encode writes the image m to w in WEBP format.
func Encode(w io.Writer, m image.Image, opt *Options) (err error) {
	return encode(w, m, opt)
}

func encode(w io.Writer, m image.Image, opt *Options) (err error) {
	var output []byte

	// If advanced options are provided, use the detailed encoding function for RGBA
	if opt != nil && hasAdvancedOptions(opt) {
		opt = applyDefaults(opt)
		switch m := adjustImage(m).(type) {
		case *image.RGBA:
			if output, err = EncodeRGBAWithConfig(m, opt); err != nil {
				return
			}
		case *image.Gray:
			// Fall back to simple encoding for Gray images.
			// Advanced options are not supported for grayscale images.
			if opt.Lossless {
				if output, err = EncodeLosslessGray(m); err != nil {
					return
				}
			} else {
				if output, err = EncodeGray(m, opt.Quality); err != nil {
					return
				}
			}
		case *RGBImage:
			// Fall back to simple encoding for RGB images.
			// Advanced options are not supported for RGB images.
			if opt.Lossless {
				if output, err = EncodeLosslessRGB(m); err != nil {
					return
				}
			} else {
				if output, err = EncodeRGB(m, opt.Quality); err != nil {
					return
				}
			}
		default:
			panic("image/webp: Encode, unreachable!")
		}
	} else if opt != nil && opt.Lossless {
		switch m := adjustImage(m).(type) {
		case *image.Gray:
			if output, err = EncodeLosslessGray(m); err != nil {
				return
			}
		case *RGBImage:
			if output, err = EncodeLosslessRGB(m); err != nil {
				return
			}
		case *image.RGBA:
			if opt.Exact != 0 {
				output, err = EncodeExactLosslessRGBA(m)
			} else {
				output, err = EncodeLosslessRGBA(m)
			}
			if err != nil {
				return
			}
		default:
			panic("image/webp: Encode, unreachable!")
		}
	} else {
		quality := float32(DefaulQuality)
		if opt != nil {
			quality = opt.Quality
		}

		switch m := adjustImage(m).(type) {
		case *image.Gray:
			if output, err = EncodeGray(m, quality); err != nil {
				return
			}
		case *RGBImage:
			if output, err = EncodeRGB(m, quality); err != nil {
				return
			}
		case *image.RGBA:
			if output, err = EncodeRGBA(m, quality); err != nil {
				return
			}
		default:
			panic("image/webp: Encode, unreachable!")
		}
	}
	_, err = w.Write(output)
	return
}

func adjustImage(m image.Image) image.Image {
	if p, ok := AsMemPImage(m); ok {
		switch {
		case p.XChannels == 1 && p.XDataType == reflect.Uint8:
			m = &image.Gray{
				Pix:    p.XPix,
				Stride: p.XStride,
				Rect:   p.XRect,
			}
		case p.XChannels == 1 && p.XDataType == reflect.Uint16:
			m = toGrayImage(m) // MemP is little endian
		case p.XChannels == 3 && p.XDataType == reflect.Uint8:
			m = &RGBImage{
				XPix:    p.XPix,
				XStride: p.XStride,
				XRect:   p.XRect,
			}
		case p.XChannels == 3 && p.XDataType == reflect.Uint16:
			m = NewRGBImageFrom(m) // MemP is little endian
		case p.XChannels == 4 && p.XDataType == reflect.Uint8:
			m = &image.RGBA{
				Pix:    p.XPix,
				Stride: p.XStride,
				Rect:   p.XRect,
			}
		case p.XChannels == 4 && p.XDataType == reflect.Uint16:
			m = toRGBAImage(m) // MemP is little endian
		}
	}
	switch m := m.(type) {
	case *image.Gray:
		return m
	case *RGBImage:
		return m
	case *RGB48Image:
		return NewRGBImageFrom(m)
	case *image.RGBA:
		return m
	case *image.YCbCr:
		return NewRGBImageFrom(m)

	case *image.Gray16:
		return toGrayImage(m)
	case *image.RGBA64:
		return toRGBAImage(m)
	case *image.NRGBA:
		return toRGBAImage(m)
	case *image.NRGBA64:
		return toRGBAImage(m)

	default:
		return toRGBAImage(m)
	}
}

func toGrayImage(m image.Image) *image.Gray {
	if m, ok := m.(*image.Gray); ok {
		return m
	}
	b := m.Bounds()
	gray := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
			gray.SetGray(x, y, c)
		}
	}
	return gray
}

func toRGBAImage(m image.Image) *image.RGBA {
	if m, ok := m.(*image.RGBA); ok {
		return m
	}
	b := m.Bounds()
	rgba := image.NewRGBA(b)
	dstColorRGBA64 := &color.RGBA64{}
	dstColor := color.Color(dstColorRGBA64)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			pr, pg, pb, pa := m.At(x, y).RGBA()
			dstColorRGBA64.R = uint16(pr)
			dstColorRGBA64.G = uint16(pg)
			dstColorRGBA64.B = uint16(pb)
			dstColorRGBA64.A = uint16(pa)
			rgba.Set(x, y, dstColor)
		}
	}
	return rgba
}
