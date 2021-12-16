/*
Kil is a tool to kill processes by name.
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func main() {
	var search string
	l := flag.Bool("l", false, "Only list matches")
	flag.Parse()
	args := flag.Args()
	flag.Usage = usage

	switch len(args) {
	case 0:
		if !*l {
			usage()
		}
	case 1:
		search = args[0]
	default:
		usage()
	}

	i, err := strconv.Atoi(search)
	if err == nil {
		killPort(i)
		return
	}

	findProcesses(search, !*l)
}

func usage() {
	fmt.Println("Kill process by name or by port")
	fmt.Println("If name is a number it will kil the process listening to that port.")
	fmt.Printf("usage: %s name\n", os.Args[0])
	fmt.Println("      -l Only list matches")
	fmt.Println("      -p port")
	os.Exit(1)
}

func killPort(p int) {
	exec.Command("fuser", "-kn", "tcp", strconv.Itoa(p)).Run()
}

func findProcesses(name string, kill bool) {
	pr, err := processes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// filter matches
	for i := len(pr) - 1; i >= 0; i-- {
		if !strings.Contains(pr[i].name, name) {
			pr = append(pr[:i], pr[i+1:]...)
		}
	}

	if len(pr) == 0 {
		fmt.Fprintln(os.Stderr, "No matches found")
		os.Exit(1)
	}

	sort.Sort(byName(pr))

	for _, p := range pr {
		fmt.Printf("%-7d %s\n", p.id, p.name)
	}

	if kill {
		fmt.Print("kill matching processes? (y/N) ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n') // this will prompt the user for input
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if input == "y\n" {
			killAll(pr)
		}
	}
}

func killAll(pr []process) {
	for _, p := range pr {
		exec.Command("kill", "-9", strconv.Itoa(p.id)).Run()
	}
}

func processes() ([]process, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	results := make([]process, 0, 50)
	for {
		fis, err := d.Readdir(10)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, fi := range fis {
			// We only care about directories, since all pids are dirs
			if !fi.IsDir() {
				continue
			}

			// We only care if the name starts with a numeric
			name := fi.Name()
			if name[0] < '0' || name[0] > '9' {
				continue
			}

			// From this point forward, any errors we just ignore, because
			// it might simply be that the process doesn't exist anymore.
			pid, err := strconv.ParseInt(name, 10, 0)
			if err != nil {
				continue
			}

			p := process{id: int(pid)}
			procName, err := getName(p.id)
			if err != nil {
				return nil, fmt.Errorf("Error parsing name of pid %d", p.id)
			}

			p.name = procName

			results = append(results, p)
		}
	}

	return results, nil
}

type process struct {
	name string
	id   int
}

type byName []process

func (s byName) Len() int {
	return len(s)
}
func (s byName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byName) Less(i, j int) bool {
	return s[i].name < s[j].name
}

func getName(id int) (string, error) {
	statPath := fmt.Sprintf("/proc/%d/stat", id)
	dataBytes, err := ioutil.ReadFile(statPath)
	if err != nil {
		return "", err
	}

	// First, parse out the image name
	data := string(dataBytes)
	binStart := strings.IndexRune(data, '(') + 1
	binEnd := strings.IndexRune(data[binStart:], ')')
	return data[binStart : binStart+binEnd], nil
}
