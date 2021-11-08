package main

import (
	//"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"os"

	"go.samhza.com/gifbench/internal/gif"

	"github.com/ericpauley/go-quantize/quantize"
	"go.samhza.com/esammy/memegen"
)

func main() {
	err := caption(os.Stdin, os.Stdout, func(w, h int) (image.Image, image.Point, bool) {
		img, pt := memegen.Caption(w, h, os.Args[1])
		return img, pt, true
	})
	if err != nil {
		log.Fatalln(err)
	}
}

type compositeFunc func(int, int) (image.Image, image.Point, bool)

func caption(rdr io.Reader, wtr io.Writer, fn compositeFunc) error {
	r := gif.NewReader(rdr)
	w := gif.NewWriter(wtr)
	cfg, err := r.ReadHeader()
	if err != nil {
		return err
	}
	inbounds := image.Rect(0, 0, cfg.Width, cfg.Height)
	img, pt, under := fn(cfg.Width, cfg.Height)
	outbounds := img.Bounds()
	cfg.Height = outbounds.Max.Y
	if err = w.WriteHeader(*cfg); err != nil {
		return err
	}

	const concurrency = 8
	type bruh struct {
		n       int
		in, out *image.Paletted
		m       draw.Image
	}
	in, out := make(chan bruh, concurrency), make(chan bruh, concurrency)
	defer close(in)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			//defer fmt.Println("worker exit", i)
			for b := range in {
				if under {
					//fmt.Println("KEK")
					draw.Draw(b.m, outbounds, img, image.Point{}, draw.Src)
					draw.Draw(b.m, outbounds, b.in, pt, draw.Over)
				} else {
					draw.Draw(b.m, outbounds, b.in, image.Point{}, draw.Src)
					draw.Draw(b.m, outbounds, img, pt, draw.Over)
				}
				b.out.Rect = outbounds
				b.out.Palette = append(quantize.MedianCutQuantizer{}.Quantize(make([]color.Color, 0, 255), b.m), color.RGBA{})
				draw.Draw(b.out, outbounds, b.m, image.Point{}, draw.Src)
				out <- b
			}
		}(i)
	}
	firstBatch := true
	scratch := make(chan bruh, concurrency)

	frames := make([]gif.ImageBlock, concurrency)
	readall := false
	for {
		if readall {
			break
		}
		var n int
		for n = 0; n < concurrency; n++ {
			block, err := r.ReadImage()
			if err != nil {
				return err
			}
			if block == nil {
				readall = true
				break
			}
			frames[n] = *block
			var b bruh
			if firstBatch {
				in := image.NewPaletted(inbounds, nil)
				b = bruh{n, in, in, image.NewRGBA(outbounds)}
				if inbounds != outbounds {
					b.out = image.NewPaletted(outbounds, nil)
				}
			} else {
				b = <-scratch
				b.n = n
			}
			copyPaletted(b.in, block.Image)
			in <- b
		}
		firstBatch = false
		for j := 0; j < n; j++ {
			b := <-out
			frames[b.n].Image = b.out
			scratch <- b
		}
		for j := 0; j < n; j++ {
			err := w.WriteFrame(frames[j])
			if err != nil {
				return err
			}
		}
	}
	return w.Close()
}

func copyPaletted(dst, src *image.Paletted) {
	copy(dst.Pix, src.Pix)
	dst.Stride = src.Stride
	dst.Rect = src.Rect
	dst.Palette = make(color.Palette, len(src.Palette))
	copy(dst.Palette, src.Palette)
}
