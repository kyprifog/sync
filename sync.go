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
	"sort"
	"strings"
	"sync"
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

func max_length() int {
	return 10
}

func drawButton(s tcell.Screen, row, column, x_spacing int, text string, outofdate, local_changes,
	check,
	sync bool) {
	green := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	red := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	blue := tcell.StyleDefault.Foreground(tcell.ColorBlue).Background(tcell.ColorBlack)
	yellow := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	max_length := max_length()

	x0 := 1 + ((column - 1) * (max_length + 5 + x_spacing))
	y0 := 1 + ((row - 1) * 4)
	x1 := x0 + (max_length + 5)
	y1 := y0 + 2

	color := green
	message := text

	if local_changes {
		color = yellow
	}
	if outofdate {
		color = red
	}
	if check {
		color = blue
	}
	if sync {
		message = "syncing..."
	}

	l := len(message)

	drawBox(s, x0, y0, x1, y1, color, ' ')
	if len(message) <= max_length {
		padding := (max_length - l) / 2
		padded_message := strings.Repeat(" ", padding) + message + strings.Repeat(" ", padding)
		emitStr(s, x0+3, y0+1, color, padded_message)
	} else {
		truncated_message := message[0:max_length-3] + "..."
		emitStr(s, x0+3, y0+1, color, truncated_message)
	}
}

func repos_path() string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	return filepath.Join(dir, "/.repos.yaml")
}

func get_repos(path string, check bool) ([]map[string]interface{}, error) {
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
		el["out_of_date"] = false
		el["local_changes"] = false
		el["sync"] = false
		el["check"] = check
	}

	sort_repos(all_repos)
	return all_repos, err
}

func sort_repos(repos []map[string]interface{}) {
	sort.Slice(repos, func(i, j int) bool {
		return repos[i]["name"].(string) < repos[j]["name"].(string)
	})
}

func run_command(app_command []string) string {
	stdout := ""
	if len(app_command) >= 2 {
		if len(app_command) == 3 {
			cmd := exec.Command(app_command[0], app_command[1], app_command[2])
			stdout, err := cmd.Output()
			if err != nil {
				print(err.Error())
			}
			return string(stdout)
		} else if len(app_command) == 2 {
			cmd := exec.Command(app_command[0], app_command[1])
			stdout, err := cmd.Output()
			if err != nil {
				print(err.Error())
			}
			return string(stdout)
		}
	}
	return stdout
}

func click_repos(x, y int, repos []map[string]interface{}) []map[string]interface{} {
	column := 1
	row := 1
	max_columns := 3
	x_spacing := 3

	max_length := max_length()

	for _, el := range repos {
		x0 := 1 + ((column - 1) * (max_length + 5 + x_spacing))
		y0 := 1 + ((row - 1) * 4)
		x1 := x0 + (max_length + 5)
		y1 := y0 + 2
		if (x < x1) && (x > x0) {
			if (y < y1) && (y > y0) {
				el["sync"] = true
			}
		}

		column += 1
		if column > max_columns {
			row += 1
			column = 1
		}
	}
	return repos
}

func render_repos(s tcell.Screen, repos []map[string]interface{}) []string {
	column := 1
	row := 1
	run_app := []string{}
	max_columns := 3
	x_spacing := 3

	for _, el := range repos {
		name := el["name"].(string)
		out_of_date := el["out_of_date"].(bool)
		local_changes := el["local_changes"].(bool)
		sync := el["sync"].(bool)
		check := el["check"].(bool)

		drawButton(s, row, column, x_spacing, name, out_of_date, local_changes, check, sync)

		column += 1
		if column > max_columns {
			row += 1
			column = 1
		}
	}
	return run_app
}

func run_action(wg *sync.WaitGroup, el map[string]interface{}, out chan map[string]interface{}) {
	defer wg.Done()
	cmd := []string{}
	sync := el["sync"].(bool)
	check := el["check"].(bool)
	path := el["path"].(string)
	if sync == true {
		push := el["push"].(bool)
		cmd = append(cmd, "./sync_cmd")
		cmd = append(cmd, path)
		if push == true {
			cmd = append(cmd, "push")
			el["local_changes"] = false
		}
		run_command(cmd)
		el["sync"] = false
		el["out_of_date"] = false
		out <- el
	} else if check == true {
		cmd = append(cmd, "./status_cmd")
		cmd = append(cmd, path)
		output := run_command(cmd)

		if strings.Contains(output, "branch is behind") {
			el["out_of_date"] = true
		}

		if strings.Contains(output, "Changes not staged") || strings.Contains(output, "Changes to be committed") {
			el["local_changes"] = true
		}

		el["check"] = false
		out <- el
	} else {
		out <- el
	}
}

func run_actions(repos []map[string]interface{}) []map[string]interface{} {
	var new_repos = []map[string]interface{}{}
	var wg sync.WaitGroup

	out := make(chan map[string]interface{})

	for _, el := range repos {
		wg.Add(1)
		go run_action(&wg, el, out)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for i := range out {
		 new_repos = append(new_repos, i)
	}

	sort_repos(new_repos)

	return new_repos
}

var defStyle tcell.Style

func main() {
	ecnt := 0
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

	repos, _ := get_repos(path, true)
	render_repos(s, repos)
	s.Show()
	new_repos := run_actions(repos)
	s.Clear()
	render_repos(s, new_repos)
	s.Show()

	go func() {

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
					clicked_repos := click_repos(x, y, new_repos)
					s.Clear()
					render_repos(s, clicked_repos)
					s.Show()
					runned_repos := run_actions(clicked_repos)
					s.Clear()
					render_repos(s, runned_repos)
					s.Show()
				}
			}
		}

	}()

	t := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-t.C:
			for _, el := range new_repos {
				el["check"] = true
			}
			runned_repos := run_actions(new_repos)
			s.Clear()
			render_repos(s, runned_repos)
			s.Show()
		}
	}

}
