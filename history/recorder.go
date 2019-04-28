package history

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
)

const timeFormat = "2006-01-02 15:04:05"

type Recorder struct {
	file   *os.File
	unique map[string]int
	sorted []Item
}

type Item struct {
	Time time.Time
	Text string
}

func NewRecorder() *Recorder {
	return &Recorder{
		unique: map[string]int{},
	}
}

func rcfile() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".0x81"), nil
}

func (item *Item) String() string {
	return fmt.Sprintf("[green]%s[white]  %s", item.Time.Format(timeFormat), item.Text)
}

func (r *Recorder) Open() error {
	path, err := rcfile()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	r.file = file
	return nil
}

func (r Recorder) Close() error {
	path, err := rcfile()
	if err != nil {
		return err
	}
	buffer := &bytes.Buffer{}
	for _, item := range r.Items() {
		_, _ = buffer.WriteString(fmt.Sprintf("%d|%s\n", item.Time.Unix(), item.Text))
	}
	return ioutil.WriteFile(path, buffer.Bytes(), os.ModePerm)
}

func (r *Recorder) find(text string) int {
	index, found := r.unique[text]
	if !found {
		return -1
	}
	return index
}

func (r *Recorder) Delete(index int) bool {
	if index >= len(r.sorted) {
		return false
	}
	item := r.sorted[index]
	delete(r.unique, item.Text)
	if index < len(r.sorted)-1 {
		copy(r.sorted[index:], r.sorted[index+1:])
	}
	r.sorted = r.sorted[:len(r.sorted)-1]
	r.Resort()
	return true
}

func (r *Recorder) Record(now time.Time, text string) {
	if index := r.find(text); index < 0 {
		r.sorted = append(r.sorted, Item{Text: text, Time: now})
	} else {
		r.sorted[index].Time = now
	}
	r.Resort()
}

func (r *Recorder) Resort() {
	items := r.sorted
	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.Unix() > items[j].Time.Unix()
	})
	for index, item := range items {
		r.unique[item.Text] = index
	}
}

func (r *Recorder) Items() []Item {
	return r.sorted
}

func (r *Recorder) Load() error {
	path, err := rcfile()
	if err != nil {
		return err
	}
	content, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var items []Item
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		ts, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if index, found := r.unique[parts[1]]; found {
			if items[index].Time.Unix() < ts {
				items[index].Time = time.Unix(ts, 0)
			}
			continue
		}
		r.unique[parts[1]] = len(items)
		items = append(items, Item{Time: time.Unix(ts, 0), Text: parts[1]})
	}
	r.sorted = items
	r.Resort()
	return nil
}
