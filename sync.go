package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/encoding"
	"github.com/mattn/go-runewidth"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, r rune) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}
	if y1 != y2 && x1 != x2 {
		// Only add corners if we need to
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		for col := x1 + 1; col < x2; col++ {
			s.SetContent(col, row, r, nil, style)
		}
	}
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		s.SetContent(x, y, c, comb, style)
		x += w
	}
}

func drawButton(s tcell.Screen, x0, y0 , x1, y1 int, text string, outofdate, local_changes bool) {
	green := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	red := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	yellow := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)

	color := green
	if local_changes {
		color = yellow
	}
	if outofdate {
		color = red
	}
	drawBox(s, x0, y0, x1, y1 , color, ' ')
	emitStr(s, x0 + 3, y0 +1, color, text)
}

func repos_path() string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	return filepath.Join(dir, "/.repos.yaml")
}

func get_repos(path string) ([]map[string]interface{}, error) {
	yamlFile, err := ioutil.ReadFile(path)

	type RepoConfig struct {
		Repos []map[string]interface{}
	}

	repos := RepoConfig{}
	if err == nil {
		err = yaml.Unmarshal(yamlFile, &repos)
	}

	all_repos := repos.Repos

	for _, el := range all_repos {
		out_of_date := false
		local_changes := false
		status_cmd := []string{}
		status_cmd = append(status_cmd, "./status_cmd")
		status_cmd = append(status_cmd, el["path"].(string))
		output := run_commands(status_cmd)

		if strings.Contains(output, "branch is behind") {
			out_of_date = true
		}

		if strings.Contains(output, "Changes not staged") || strings.Contains(output,
			"Changes to be committed") {
			local_changes = true
		}


		el["out_of_date"] = out_of_date
		el["local_changes"] = local_changes

	}


	return all_repos, err
}

func num_columns() int {
	return 3
}

func run_commands(app_commands []string) string {
	stdout := ""
	if len(app_commands) >= 2 {
		if len(app_commands) == 3 {
			cmd := exec.Command(app_commands[0], app_commands[1], app_commands[2])
			stdout, err := cmd.Output()
			if err != nil {
				print(err.Error())
			}
			return string(stdout)
		} else if len(app_commands) == 2 {
			cmd := exec.Command(app_commands[0], app_commands[1])
			stdout, err := cmd.Output()
			if err != nil {
				print(err.Error())
			}
			return string(stdout)
		}
	}
	return stdout
}

func render_repos(s tcell.Screen, x, y, x_spacing int, repos []map[string]interface{}) []string {
	column := 1
	row := 1
	syncing_message := "syncing..."
	max_length := len(syncing_message)
	run_app := []string{}

	for _, el :=  range repos {
		name := el["name"].(string)
		path := el["path"].(string)
		push := el["push"].(bool)
		out_of_date := el["out_of_date"].(bool)
		local_changes := el["local_changes"].(bool)
		x0 := 1 + ((column - 1) * (max_length + 5 + x_spacing))
		y0 := 1 + ((row -1) * 4)
		x1 := x0 + (max_length + 5)
		y1 := y0 + 2
		if (x < x1) && (x > x0) {
			if (y < y1) && (y > y0) {
				 name = syncing_message
				 run_app = append(run_app, "./sync_cmd")
				 run_app = append(run_app, path)
				 if push == true {
					  run_app = append(run_app, "push")
				 }
			}
		}

		l := len(name)

		if l <= max_length {
			padding := (max_length - l) / 2
		 	padded_name := strings.Repeat(" ", padding) + name + strings.Repeat(" ", padding)
		 	drawButton(s, x0, y0, x1, y1, padded_name, out_of_date, local_changes)
		} else {
			//truncated_name := name[0:max_length-3] + "..."
			truncated_name := name
			drawButton(s, x0, y0, x1, y1, truncated_name, out_of_date, local_changes)
		}

		column += 1
		if column > num_columns() {
			row += 1
			column = 1
		}
	}
	return run_app

}

var defStyle tcell.Style

func main() {
	path := repos_path()
	if len(os.Args) > 1 {
		path = os.Args[1]
	}


	s, e := tcell.NewScreen()

	encoding.Register()

	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

	s.SetStyle(defStyle)
	s.EnableMouse()

	s.Clear()
	repos, _ := get_repos(path)

	x_spacing := 3

	render_repos(s, 0, 0, x_spacing, repos)

	s.Show()

	go func() {
		ecnt := 0
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {

			case *tcell.EventKey:
				if ev.Key() == tcell.KeyEscape {
					ecnt++
					if ecnt > 1 {
						s.Fini()
						os.Exit(0)
					}
				}
			case *tcell.EventMouse:
				switch ev.Buttons() {
				case tcell.Button1, tcell.Button2, tcell.Button3:
					x, y := ev.Position()
					app_commands := render_repos(s, x, y, x_spacing, repos)
					s.Show()
					run_commands(app_commands)
					s.Clear()
					repos, _ = get_repos(path)
					render_repos(s, 0, 0, x_spacing, repos)
					s.Show()
				}
			}
		}
	}()

	t := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-t.C:
			s.Clear()
			repos, _ = get_repos(path)
			render_repos(s, 0, 0, x_spacing, repos)
			s.Show()
		}
	}

}

