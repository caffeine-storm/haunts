package main

import (
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

type SaveWidget struct {
	*gui.VerticalTable
	filename *gui.TextEditLine
	on_save  func(string)
}

func MakeSaveWidget(on_save func(string)) *SaveWidget {
	var sw SaveWidget

	sw.VerticalTable = gui.MakeVerticalTable()
	sw.on_save = on_save
	sw.AddChild(gui.MakeTextLine("standard_18", "Enter Filename", 300, 1, 1, 1, 1))
	sw.filename = gui.MakeTextEditLine("standard_18", "filename", 300, 1, 1, 1, 1)
	sw.AddChild(sw.filename)
	sw.AddChild(gui.MakeButton("standard_18", "Save!", 300, 1, 1, 1, 1, func(int64) {
		sw.on_save(sw.filename.GetText())
	}))

	return &sw
}

func (sw *SaveWidget) Respond(ui *gui.Gui, event_group gui.EventGroup) bool {
	if found, event := event_group.FindEvent(gin.AnyReturn); found && event.Type == gin.Press {
		sw.on_save(sw.filename.GetText())
		return true
	}
	sw.VerticalTable.Respond(ui, event_group)
	return true
}
