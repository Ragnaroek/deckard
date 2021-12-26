package deckard

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DeckardUI struct {
	app      *tview.Application
	projects *tview.TextView
	status   *tview.TextView

	config *Config

	state *uiState
}

type uiState struct {
	selectedProject int
	status          string
}

func newDeckardUi(app *tview.Application, state *uiState, config *Config) *DeckardUI {
	return &DeckardUI{app: app, state: state, config: config}
}

func (ui *DeckardUI) Run() error {
	return ui.app.Run()
}

func (ui *DeckardUI) SelectProject(id int) {
	ui.state.selectedProject = id
	updateProjectText(ui.projects, ui.state, ui.config)
}

// TODO Render spinner (optional parameter)
func (ui *DeckardUI) UpdateStatus(text string) {
	ui.state.status = text
	updateStatusText(ui.status, ui.state)
}

func (ui *DeckardUI) ClearStatus() {
	ui.state.status = ""
	updateStatusText(ui.status, ui.state)
}

func BuildUI(config *Config) (*DeckardUI, error) {

	initialState := &uiState{}

	header := tview.NewFlex().SetDirection(tview.FlexColumn)
	header.SetTitle("Deckard").SetTitleAlign(tview.AlignLeft).SetBorder(true)

	projects := buildProjects(initialState, config)
	status := buildStatus(initialState)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header.
			AddItem(projects, 0, 50, false).
			AddItem(status, 0, 50, false),
			3, 100, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox().SetBorder(true).SetTitle("Commits"), 0, 66, false).
			AddItem(tview.NewBox().SetBorder(true).SetTitle("Statistics"), 0, 33, false),
			0, 100, false)

	app := tview.NewApplication().SetRoot(flex, true)
	ui := newDeckardUi(app, initialState, config)
	ui.projects = projects
	ui.status = status

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return handleInput(ui, config, event)
	})

	return ui, nil
}

func handleInput(ui *DeckardUI, config *Config, event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune {
		if event.Rune() >= '0' && event.Rune() <= '9' {
			id, err := strconv.Atoi(string(event.Rune()))
			if err != nil {
				panic(err) //should not happen
			}
			if id > len(config.Projects) {
				return event
			}
			ui.SelectProject(id)
			return event
		}
	}
	return event
}

// ## project view

func buildProjects(state *uiState, config *Config) *tview.TextView {
	text := tview.NewTextView()
	text.SetRegions(true)
	updateProjectText(text, state, config)
	return text
}

type project struct {
	name string
	icon string
}

const SelectionMarker = "sel"

func updateProjectText(text *tview.TextView, state *uiState, config *Config) {

	projects := make([]project, 0)
	projects = append(projects, project{
		name: "all",
		icon: "",
	})
	for id, data := range config.Projects {
		projects = append(projects, project{name: id, icon: data.Icon})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].name < projects[j].name
	})

	prjText := ""
	num := 0
	for _, prj := range projects {

		sel := ""
		if state.selectedProject == num {
			sel = SelectionMarker
		}

		prjText += fmt.Sprintf(` | ["%s"](%d) %s %s[""]`, sel, num, prj.name, prj.icon)
		num++
	}

	text.SetText(prjText)
	text.Highlight(SelectionMarker)
}

// ## status view

func buildStatus(state *uiState) *tview.TextView {
	text := tview.NewTextView()
	text.SetDynamicColors(true)
	text.SetRegions(true)
	updateStatusText(text, state)
	return text
}

func updateStatusText(text *tview.TextView, state *uiState) {
	text.SetText(fmt.Sprintf("[yellow]%s[-]", state.status))
}
