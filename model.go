package main

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/watch"
)

type model struct {
	width, height int

	workflows     map[string]*v1alpha1.Workflow
	workflowTable table.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case workflowEventMsg:
		switch msg.eventType {
		case watch.Added, watch.Modified:
			m.workflows[msg.workflow.Name] = msg.workflow
		case watch.Deleted:
			delete(m.workflows, msg.workflow.Name)
		}
	}

	workflows := slices.Collect(maps.Values(m.workflows))

	var longestNameLength int
	for _, wf := range workflows {
		if l := len(wf.Name); l > longestNameLength {
			longestNameLength = l
		}
	}

	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].Status.StartedAt.After(workflows[j].Status.StartedAt.Time)
	})

	columns := []table.Column{
		{Title: "Name", Width: longestNameLength},
		{Title: "Namespace", Width: 9},
		{Title: "Started", Width: 9},
		{Title: "Finished", Width: 9},
		{Title: "Duration", Width: 8},
		{Title: "Progress", Width: 8},
		{Title: "Completed", Width: 9},
	}

	rows := make([]table.Row, len(workflows))
	for i, wfItem := range workflows {
		rows[i] = table.Row{
			wfItem.Name,
			wfItem.Namespace,
			fmt.Sprintf("%s ago", time.Since(wfItem.Status.StartedAt.Time).Truncate(time.Second)),
			fmt.Sprintf("%s ago", time.Since(wfItem.Status.FinishedAt.Time).Truncate(time.Second)),
			wfItem.Status.GetDuration().String(),
			string(wfItem.Status.Progress),
			strconv.FormatBool(wfItem.Status.Phase.Completed()),
		}
	}

	m.workflowTable = table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height),
	)

	var cmd tea.Cmd
	m.workflowTable, cmd = m.workflowTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.workflowTable.View()
}

// Cmds
type (
	workflowEventMsg struct {
		workflow  *v1alpha1.Workflow
		eventType watch.EventType
	}
)
