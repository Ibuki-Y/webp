// Copyright 2014 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webp

import (
	"bytes"
	_ "image/png"
	"testing"
)

type tTester struct {
	Filename string
	Lossless bool
	Quality  float32 // 0 ~ 100
	MaxDelta int
	MinDelta int
	Exact    int
}

var tTesterList = []tTester{
	tTester{
		Filename: "video-001.png",
		Lossless: false,
		Quality:  90,
		MaxDelta: 5,
	},
	tTester{
		Filename: "1_webp_ll.png",
		Lossless: false,
		Quality:  90,
		MaxDelta: 5,
	},
	tTester{
		Filename: "2_webp_ll.png",
		Lossless: true,
		Exact:    1,
		Quality:  90,
		MaxDelta: 0,
	},
	tTester{
		Filename: "3_webp_ll.png",
		Lossless: false,
		Quality:  90,
		MaxDelta: 5,
	},
	tTester{
		Filename: "4_webp_ll.png",
		Lossless: true,
		Exact:    1,
		Quality:  90,
		MaxDelta: 0,
	},
	tTester{
		Filename: "5_webp_ll.png",
		Lossless: false,
		Quality:  75,
		MaxDelta: 15,
	},
	tTester{
		Filename: "4_webp_ll.png",
		Lossless: true,
		Quality:  90,
		Exact:    0,
		MaxDelta: 13,
		MinDelta: 10,
	},
}

// Advanced options test cases for EncodeRGBAWithConfig
type tAdvancedTester struct {
	Filename string
	Options  *Options
	MaxDelta int
}

var tAdvancedTesterList = []tAdvancedTester{
	// Test with Method parameter (quality/speed trade-off)
	{
		Filename: "video-001.png",
		Options: &Options{
			Lossless: false,
			Quality:  85,
			Method:   6, // slower-better quality
		},
		MaxDelta: 8,
	},
	// Test with TargetSize parameter
	{
		Filename: "1_webp_ll.png",
		Options: &Options{
			Lossless:   false,
			Quality:    80,
			TargetSize: 10000, // target 10KB
		},
		MaxDelta: 15,
	},
	// Test with advanced filtering options
	{
		Filename: "3_webp_ll.png",
		Options: &Options{
			Lossless:        false,
			Quality:         90,
			FilterStrength:  80,
			FilterSharpness: 3,
			FilterType:      1,
		},
		MaxDelta: 5,
	},
	// Test with alpha compression options
	{
		Filename: "2_webp_ll.png",
		Options: &Options{
			Lossless:         true,
			Quality:          90,
			AlphaCompression: 1,
			AlphaFiltering:   2,
			Exact:            1,
		},
		MaxDelta: 0,
	},
	// Test with multiple advanced options combined
	{
		Filename: "5_webp_ll.png",
		Options: &Options{
			Lossless:        false,
			Quality:         75,
			Method:          4,
			Segments:        4,
			SnsStrength:     50,
			FilterStrength:  60,
			Autofilter:      true,
			Pass:            6,
			Preprocessing:   1,
		},
		MaxDelta: 20,
	},
}

func TestEncode(t *testing.T) {
	for i, v := range tTesterList {
		img0, err := loadImage(v.Filename)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		buf := new(bytes.Buffer)
		err = Encode(buf, img0, &Options{
			Lossless: v.Lossless,
			Quality:  v.Quality,
			Exact:    v.Exact,
		})
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		img1, err := Decode(buf)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		// Compare the average delta to the tolerance level.
		var want int
		if !v.Lossless || v.Exact == 0 {
			want = v.MaxDelta
		}
		got := averageDelta(img0, img1)
		if got > want {
			t.Fatalf("%d: average delta too high; got %d, want <= %d", i, got, want)
		}
		if v.MinDelta > 0 && got < v.MinDelta {
			t.Fatalf("%d: average delta too low; got %d; want >= %d", i, got, v.MinDelta)
		}
	}
}

// TestEncodeAdvanced tests the new EncodeRGBAWithConfig functionality
// with various advanced WebP encoding options.
func TestEncodeAdvanced(t *testing.T) {
	for i, v := range tAdvancedTesterList {
		img0, err := loadImage(v.Filename)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		buf := new(bytes.Buffer)
		err = Encode(buf, img0, v.Options)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		img1, err := Decode(buf)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		// Compare the average delta to the tolerance level.
		var want int
		if !v.Options.Lossless || v.Options.Exact == 0 {
			want = v.MaxDelta
		}
		got := averageDelta(img0, img1)
		if got > want {
			t.Fatalf("%d: average delta too high; got %d, want <= %d", i, got, want)
		}
	}
}
