// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"strings"
)

// The example files hold real attachments: documents that open in a PDF viewer and images that open in any image
// viewer. Both generators are deterministic, so regenerating an example gives the same bytes.

// realPDF builds a one-page PDF 1.4 document with a title and some lines of text, with a correct cross-reference table.
func realPDF(title string, lines ...string) []byte {
	escape := strings.NewReplacer(`\`, `\\`, "(", `\(`, ")", `\)`)
	var text strings.Builder
	fmt.Fprintf(&text, "BT /F1 22 Tf 72 760 Td (%s) Tj ET\n", escape.Replace(title))
	for i, line := range lines {
		fmt.Fprintf(&text, "BT /F1 12 Tf 72 %d Td (%s) Tj ET\n", 720-i*20, escape.Replace(line))
	}
	stream := text.String()

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	var offsets []int
	object := func(n int, body string) {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}
	object(1, "<< /Type /Catalog /Pages 2 0 R >>")
	object(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	object(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>")
	object(4, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	object(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	object(6, fmt.Sprintf("<< /Title (%s) /Producer (SereniBase example) >>", escape.Replace(title)))

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R /Info 6 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)
	return buf.Bytes()
}

// realPNG draws a w x h image: a gradient in a colour taken from the name, with a lighter circle and two bands, so each
// file looks different and is easy to recognise.
func realPNG(w, h int, name string) []byte {
	sum := fnv.New32a()
	_, _ = sum.Write([]byte(name))
	seed := sum.Sum32()
	base := color.RGBA{R: uint8(60 + seed%140), G: uint8(60 + (seed>>8)%140), B: uint8(60 + (seed>>16)%140), A: 255}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	cx, cy, radius := w/2, h/2, min(w, h)/3
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / float64(h)
			c := color.RGBA{R: uint8(float64(base.R) * (1 - 0.5*t)), G: uint8(float64(base.G) * (1 - 0.5*t)), B: uint8(float64(base.B) * (1 - 0.5*t)), A: 255}
			if dx, dy := x-cx, y-cy; dx*dx+dy*dy <= radius*radius {
				c = color.RGBA{R: 245, G: 245, B: 245, A: 255}
			}
			if (x+y)%(max(w, h)/4+1) < 3 {
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic(err)
	}
	return out.Bytes()
}

// The files used by the examples.
var (
	invoicePDF = realPDF("Invoice 1", "Customer: Acme", "Amount: 120.50", "Status: New", "Thank you for your order.")
	coverPNG   = realPNG(480, 240, "cover")
)
