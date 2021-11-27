package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gotk3/gotk3/gtk"

	"golang.design/x/clipboard"
)

type CopyPipeline interface {
	IsMatch(data []byte) bool
	Process(data []byte) (bool, []byte)
	Run(self *struct{}, data []byte) (bool, []byte)
}

type JSONHandler struct {
	name string
}

func (j JSONHandler) IsMatch(data []byte) bool {
	return json.Valid(data)
}

func (j JSONHandler) Process(data []byte) (bool, []byte) {
	prettyJSON, err := indentJSON(data)
	if err != nil {
		return false, []byte{}
	}
	return true, prettyJSON
}

func indentJSON(str []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, str, "", "  ")
	if err != nil {
		return nil, err
	}
	return prettyJSON.Bytes(), nil
}

func (j JSONHandler) Run(self JSONHandler, data []byte) (bool, []byte) {
	h1Res := false
	out := []byte{}

	h1Res = self.IsMatch(data)
	if h1Res {
		log.Println("Match step taken for", self.name)
		h1Res, out = self.Process(data)
		if h1Res {
			log.Println("Process step taken for", self.name)
		} else {
			log.Println("Process step not taken for", self.name)
		}
	} else {
		log.Println("Match step not taken for", self.name)
	}
	return h1Res, out
}

func RunPipeline(data []byte, out chan<- []byte) {
	h1 := JSONHandler{"JSON Handler"}
	_, newData := h1.Run(h1, data)

	out <- newData
}

func CancelAll(cancel []context.CancelFunc) {
	for _, c := range cancel {
		c()
	}
}

func _ConsumeSignals(cancel []context.CancelFunc, sigchan <-chan os.Signal) {
	// TODO: Should we loop over the signals or just break after the first?
	// Maybe we should have an if condition for SIGINT
	for range sigchan {
		CancelAll(cancel)
	}
	gtk.MainQuit()
	os.Exit(0)
}

func StartSignalHandlers(cancel []context.CancelFunc) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	go _ConsumeSignals(cancel, sigchan)
}

type GUI struct {
	ClipboardContent *widget.Entry
}

func main() {
	gui := GUI{}
	rootCtx := context.Background()
	clipCtx, clipCancel := context.WithCancel(rootCtx)

	StartSignalHandlers([]context.CancelFunc{clipCancel})

	in := clipboard.Watch(clipCtx, clipboard.FmtText)
	out := make(chan []byte)
	go func() {
		for data := range in {
			RunPipeline(data, out)
		}
	}()

	go func() {
		for t := range out {
			if len(t) > 0 {
				gui.ClipboardContent.SetText(string(t))
				clipboard.Write(clipboard.FmtText, t)
			}
		}
	}()

	a := app.New()
	w := a.NewWindow("Hello")

	//hello := widget.NewLabel("Hello Fyne!")
	hello := widget.NewMultiLineEntry()
	hello.SetText("hello world")

	gui.ClipboardContent = hello
	w.SetContent(container.NewMax(
		hello,
	))

	w.ShowAndRun()
}

// {"hello": "world"}
