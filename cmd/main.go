// +build js

package main

import (
	"syscall/js"
)

/*
WHAT ARE THE FACTS?

o  Pixel addresses refer to the boundaries between rows of pixels, not
   the centre of the row.
o  So to address one row exactly coords must be something.5
o  If your (float) line width does not line up with available pixel
   rows - it uses adjacent rows partially and mucks about with the
   colour you specify to create the right illusion of width.
o  If you ask for a line thickness less than 1.0 it mucks about with
   the colour you specify - as above.
o  If you ask for a thickness of one CSS pixel, it renders on the real
   screen canvas across two device pixels (on Retina, where PR = 2.0).

WHAT ARE THE IMPLICATIONS?

o  To prevent anti aliasing:
	o  never specify a thickness < 1.0
	o  always specify integer thickness
	o  snap you coordinates to target actual pixels perfectly
o  To get lines on the screen that are single device-pixel wide, you
   must draw into a canvas that is twice the size and then composit that image 
   to the screen canvas at scale 0.5.

*/

func main() {
	global := js.Global()
	doc := global.Get("document")
	canv := doc.Call("getElementById", "myCanvas")
	ctx := canv.Call("getContext", "2d", map[string]interface{}{"alpha": false})

	/*
	pr := global.Get("devicePixelRatio").Float()
	w := int64(canv.Get("width").Int())
	h := int64(canv.Get("height").Int())
	*/

	// canv.Set("width", 100)
	// canv.Set("height", 100)

	ctx.Set("strokeStyle", "#FFFFFF")
	ctx.Set("lineWidth", 1)

	ctx.Call("beginPath")

	ctx.Call("moveTo", 10, 5.5)
	ctx.Call("lineTo", 50, 5.5)

	ctx.Call("stroke")
}
