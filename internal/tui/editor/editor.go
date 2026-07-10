package editor

import (
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

type FinishedMsg struct {
	Body string
	Err  error
}

func OpenBodyCmd(body string) tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			editor = "vi"
		}
		f, err := os.CreateTemp("", "naitv-mcp-body-*.md")
		if err != nil {
			return FinishedMsg{Body: body, Err: err}
		}
		path := f.Name()
		_, _ = f.WriteString(body)
		f.Close()
		c := exec.Command(editor, path)
		return tea.ExecProcess(c, func(execErr error) tea.Msg {
			defer os.Remove(path)
			if execErr != nil {
				return FinishedMsg{Body: body, Err: execErr}
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return FinishedMsg{Body: body, Err: err}
			}
			return FinishedMsg{Body: string(b)}
		})()
	}
}
