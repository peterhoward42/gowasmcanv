// +build js

package main

import (
	_ "fmt"
	"math"
	"syscall/js"
)

/*

Source  https://github.com/golang/go/wiki/WebAssembly

WASM DEVELOPMENT FILES
----------------------

$(go env GOROOT)/misc/wasm/wasm_exec.js"
	The go run() javascript package for a wasm file that you include as
	a <script> in a web page. Ships with Go.

Note about *go run/test*
	The default <go run> / <go test> command runs the compiled executable
	directly. However it accepts  an -exec argument to specify a intermediary
	command to which running the compiled executable is delegated.
	Ships with Go.

$(go env GOROOT)/misc/wasm/go_js_wasm_exec"

	go_js_wasm_exec is such an -exec runner command that runs the (WASM)
	compiled executable inside Node.js from the command line. Thus good
	for unit tests without browser dependency.

"$GOPATH/bin/wasmbrowsertest"

	wasmbrowsertest is another -exec runner command that runs the (WASM)
	compiled executable from an html page context inside a headless Chrome
	browser. Get it from github.com/agnivade/wasmbrowsertest.


RUNNING IN A WEB PAGE
---------------------

o  See source (above) for step by step instructions
o  Write a main function
o  Write an index.html page that runs it
o  Compile yourwasm
	cd ./cmd
	GOOS=js GOARCH=wasm go build -o main.wasm
o  Serve: 3 files with a web server: index.html, main.wasm, wasm_exec.js
o  Nb. There's a single-line solution:
o  goexec 'http.ListenAndServe(`:8080`, http.FileServer(http.Dir(`.`)))'
o  Get it from: github.com/shurcooL/goexec by doing
	go get -u github.com/shurcooL/goexec
o  Browse to localhost:8080/index.html
o  Move the mouse to see graphics compositing based on mouse movement.

RUNNING *go run/test* in a Node.js ENVIRONMENT
----------------------------------------------

o  Install Node
o  GOOS=js GOARCH=wasm go run -exec="$(go env GOROOT)/misc/wasm/go_js_wasm_exec" .
o  Or substitute *go test* for *go run*
o  Shortcut
	o  This to be confirmed by testing...
	o  *go run* knows it should use the go_js_wasm_exec command when it is
	   targeting wasm, but doesn't know where to find it. Hence if you put its
	   location on your $PATH you don't need to specify the -exec part.

RUNNING *go run/test* in a HEADLESS CHROME
------------------------------------------

o  go get github.com/agnivade/wasmbrowsertest
o  ensure $PATH includes it $GOPATH/bin/wasmbrowsertest
o  GOOS=js GOARCH=wasm go test -exec="$GOPATH/bin/wasmbrowsertest" .

*/

func main() {
	r := NewRenderer()

	// We draw the background canvas just once.
	r.GetBackgroundCanvasReady()

	// Everything else happens in response to mouse movements.
	r.RealCanvas.Set("onmousemove", js.FuncOf(r.OnMoveHandler))
	wait := make(chan bool)
	<-wait
}

// Renderer is a thing that knows how to operate and scale backing canvases,
// draw into them, and composit them into a "real" on-screen canvas.
type Renderer struct {
	RealCanvas  js.Value
	RealContext js.Value

	PixelRatio float64
	Document   js.Value
	Global     js.Value

	BackgroundCanvas  js.Value
	BackgroundContext js.Value

	ForegroundCanvas  js.Value
	ForegroundContext js.Value

	WidthCSS  int64
	HeightCSS int64
}

// NewRenderer provides a Renderer ready to use.
func NewRenderer() *Renderer {
	// Access the "real" canvas.
	r := &Renderer{}
	r.Global = js.Global()
	r.Document = r.Global.Get("document")
	r.RealCanvas = r.Document.Call("getElementById", "myCanvas")

	// Create two scaled-up off-screen canvases - so that we can work in
	// device pixels - and hence get single-pixel line thicknesses.
	r.PixelRatio = r.Global.Get("devicePixelRatio").Float()
	r.WidthCSS = int64(r.RealCanvas.Get("width").Int())
	r.HeightCSS = int64(r.RealCanvas.Get("height").Int())

	r.BackgroundCanvas = r.offScreenCanvas()
	r.BackgroundContext = r.BackgroundCanvas.Call(
		"getContext", "2d", map[string]interface{}{"alpha": false})

	r.ForegroundCanvas = r.offScreenCanvas()
	r.ForegroundContext = r.ForegroundCanvas.Call(
		"getContext", "2d", map[string]interface{}{"alpha": false})

	// Set the transform on the real canvas to invert the scaling up
	// of the off-screen canvases - so that when we composit the offscreen
	// canvases onto it they come out the right size.
	r.RealContext = r.RealCanvas.Call(
		"getContext", "2d", map[string]interface{}{"alpha": false})
	r.RealContext.Call("setTransform", 1/r.PixelRatio, 0, 0, 1/r.PixelRatio, 0, 0)

	return r
}

// offScreenCanvas is a constructor for an off-screen canvas to support
// off-sceen rendering. It is scaled up by a factor equal to the screen's
// device pixel ration. E.g. by 2.0 on a Retina screen.
func (r *Renderer) offScreenCanvas() (canvas js.Value) {
	canvas = r.Document.Call("createElement", "canvas")
	width := int64(r.PixelRatio * float64(r.WidthCSS))
	height := int64(r.PixelRatio * float64(r.HeightCSS))
	canvas.Set("width", width)
	canvas.Set("height", height)
	return canvas
}

// GetBackgroundCanvasReady does all the necessary graphics drawing into
// the background off-screen canvas. This content does not change, so the intent
// is to do it only once.
func (r *Renderer) GetBackgroundCanvasReady() {
	lineWidthForScaledCanvas := r.CalcScaledLineWidth(1)
	context := r.BackgroundContext
	context.Set("strokeStyle", "#FFFFFF")
	context.Set("lineWidth", lineWidthForScaledCanvas)
	context.Call("beginPath")
	r.line(context, lineWidthForScaledCanvas, 10, 10, 990, 590)
	r.line(context, lineWidthForScaledCanvas, 10, 10, 990, 10)
	r.line(context, lineWidthForScaledCanvas, 10, 10, 10, 590)
	context.Call("stroke")
}

// CalcScaledLineWidth receives the requested line width that is desired
// on the screen - in device pixel units. It returns the thickness that
// should be used to set the *lineWidth* attribute on the context that belongs
// to a scaled-up off-screen canvas.
func (r *Renderer) CalcScaledLineWidth(onScreenLineThicknessInDevicePixels int) int {
	return int(math.Round(float64(onScreenLineThicknessInDevicePixels) * r.PixelRatio))
}

// line draws a line into the given drawing context.
func (r *Renderer) line(context js.Value, thick int, x1, y1, x2, y2 int) {
	context.Call("moveTo", nudge(x1, thick)*r.PixelRatio, nudge(y1, thick)*r.PixelRatio)
	context.Call("lineTo", nudge(x2, thick)*r.PixelRatio, nudge(y2, thick)*r.PixelRatio)
}

// OnMoveHandler is an event handler for mouse movement events. It first redraws
// the foreground canvas to represent the new mouse position, and then updates
// the on-screen canvas, by compositing the two off-screen canvases into the
// on-screen one.
func (r *Renderer) OnMoveHandler(canvas js.Value, args []js.Value) interface{} {
	evt := args[0]
	x := evt.Get("offsetX").Int()
	y := evt.Get("offsetY").Int()
	r.drawMouseTrackingGraphics(x, y)
	r.compositToScreen()
	return nil
}

// drawMouseTrackingGraphics paints the foreground canvas with a short
// line stroke going through the mouse position over an otherwise completely
// black background.
func (r *Renderer) drawMouseTrackingGraphics(x, y int) {
	context := r.ForegroundContext
	context.Set("fillStyle", "#000000")
	w := r.ForegroundCanvas.Get("width")
	h := r.ForegroundCanvas.Get("height")
	context.Call("fillRect", 0, 0, w, h)

	context.Set("strokeStyle", "CornFlowerBlue")
	lineWidthForScaledCanvas := r.CalcScaledLineWidth(1)
	context.Set("lineWidth", lineWidthForScaledCanvas)
	context.Call("beginPath")
	r.line(context, lineWidthForScaledCanvas, x-10, y, x+10, y)
	context.Call("stroke")
}

func (r *Renderer) compositToScreen() {
	// Completely overwrite with the background.
	r.RealContext.Set("globalCompositeOperation", "copy")
	r.RealContext.Call("drawImage", r.BackgroundCanvas, 0, 0)
	// Superimpose the foreground - using a mask that selects only
	// pixels that become lighter.
	r.RealContext.Set("globalCompositeOperation", "lighter")
	r.RealContext.Call("drawImage", r.ForegroundCanvas, 0, 0)
}

// nudge adds 0.5 to q if thickDC is even, otherwise it returns it
// unchanged. This makes sure that horizontal and vertical lines end up with
// a centreline that is centrally aligned with a row or column of physical pixels.
func nudge(q int, thickDC int) float64 {
	if thickDC%2 != 0 {
		return float64(q)
	}
	return float64(q) + 0.5
}
