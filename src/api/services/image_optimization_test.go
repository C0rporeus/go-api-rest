package services

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestOptimizePNGIsLosslessAndReducesUncompressedPNG(t *testing.T) {
	originalImage := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			originalImage.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*17 + y*3) % 256),
				G: uint8((y*19 + x*5) % 256),
				B: uint8((x*y + 13) % 256),
				A: uint8(100 + (x+y)%156),
			})
		}
	}
	var input bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&input, originalImage); err != nil {
		t.Fatalf("encode source PNG: %v", err)
	}

	optimized := optimizePNG("image/png", input.Bytes())
	if len(optimized) >= input.Len() {
		t.Fatalf("expected optimized PNG to shrink: before=%d after=%d", input.Len(), len(optimized))
	}
	decoded, err := png.Decode(bytes.NewReader(optimized))
	if err != nil {
		t.Fatalf("decode optimized PNG: %v", err)
	}
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			gotR, gotG, gotB, gotA := decoded.At(x, y).RGBA()
			wantR, wantG, wantB, wantA := originalImage.At(x, y).RGBA()
			if gotR != wantR || gotG != wantG || gotB != wantB || gotA != wantA {
				t.Fatalf("pixel (%d,%d) changed: got=(%d,%d,%d,%d) want=(%d,%d,%d,%d)", x, y, gotR, gotG, gotB, gotA, wantR, wantG, wantB, wantA)
			}
		}
	}
}

func TestOptimizePNGLeavesUnsupportedOrInvalidDataUntouched(t *testing.T) {
	input := []byte("not an image")
	if got := optimizePNG("image/png", input); !bytes.Equal(got, input) {
		t.Fatal("invalid PNG should be passed through unchanged")
	}
	if got := optimizePNG("image/jpeg", input); !bytes.Equal(got, input) {
		t.Fatal("non-PNG image should be passed through unchanged")
	}
}
