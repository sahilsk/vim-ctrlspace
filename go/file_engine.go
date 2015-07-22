package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Context struct {
	SearchModeEnabled int
	SearchText        string
	SearchResonators  string
	Columns           int
	MaxHeight         int
	MaxSearchedItems  int
	MaxDisplayedItems int
	Path              string
	Dots              string
	DotsSize          int
	SearchLowerRunes  []rune
	ResonatorRunes    []rune
}

type FileItem struct {
	Index      int
	Name       string
	Noise      int
	Pattern    string
	Runes      []rune
	LowerRunes []rune
}

var (
	context Context
	items   []*FileItem
)

func Init(input *os.File) error {
	r := bufio.NewReader(input)
	line, _, err := r.ReadLine()

	if err != nil {
		return err
	}

	if err = initContext(line); err != nil {
		return err
	}

	return initFileItems()
}

func initContext(line []byte) (err error) {
	if err = json.Unmarshal(line, &context); err == nil {
		context.SearchLowerRunes = []rune(strings.ToLower(context.SearchText))
		context.ResonatorRunes = []rune(context.SearchResonators)
	}

	return
}

func initFileItems() error {
	file, err := os.Open(string(context.Path))

	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	idx := 0

	for scanner.Scan() {
		text := scanner.Text()
		items = append(items, &FileItem{
			Index:      idx,
			Name:       text,
			Runes:      []rune(text),
			LowerRunes: []rune(strings.ToLower(text)),
		})
		idx++
	}

	return scanner.Err()
}

func findSubsequence(item *FileItem, offset int) (int, []int) {
	positions := make([]int, 0, len(item.LowerRunes))
	noise := 0

	for _, sl := range context.SearchLowerRunes {
		pos := -1

		for i, l := range item.LowerRunes[offset:] {
			if l == sl {
				pos = i + offset
				break
			}
		}

		if pos == -1 {
			return -1, nil
		} else {
			if len(positions) > 0 {
				n := pos - positions[len(positions)-1]

				if n < 0 {
					n = -n
				}

				noise += n - 1
			}

			positions = append(positions, pos)
			offset = pos + 1
		}
	}

	return noise, positions
}

func findLowestSearchNoise(item *FileItem) (int, string) {
	noise := -1
	matched := ""

	if len(context.SearchLowerRunes) == 1 {
		for i, l := range item.LowerRunes {
			if l == context.SearchLowerRunes[0] {
				noise = i
				break
			}
		}

		if noise > -1 {
			matched = context.SearchText
		}
	} else {
		offset := 0

		for offset < len(item.Runes) {
			n, p := findSubsequence(item, offset)

			if n == -1 {
				break
			} else if noise == -1 || n < noise {
				noise = n
				offset = p[0] + 1
				matched = string(item.Runes[p[0] : p[len(p)-1]+1])

				if len(context.ResonatorRunes) > 0 {
					if p[0] != 0 {
						noise++
						moreNoise := true

						for _, r := range context.ResonatorRunes {
							if r == item.Runes[p[0]-1] {
								moreNoise = false
								break
							}
						}

						if moreNoise {
							noise++
						}
					}

					if p[len(p)-1] != len(item.Runes)-1 {
						noise++
						moreNoise := true

						for _, r := range context.ResonatorRunes {
							if r == item.Runes[p[len(p)-1]+1] {
								moreNoise = false
								break
							}
						}

						if moreNoise {
							noise++
						}
					}
				}
			} else {
				offset++
			}
		}
	}

	pattern := ""

	if noise > -1 && matched != "" {
		pattern = matched
	}

	return noise, pattern
}

func maxNoise() (index int, noise int) {
	index = -1
	noise = -1

	for i, item := range items {
		if item.Noise >= noise {
			noise = item.Noise
			index = i
		}
	}

	return
}

func computeLowestNoises() {
	unprocessed := items
	items = make([]*FileItem, 0, context.MaxSearchedItems)

	for _, item := range unprocessed {
		noise, pattern := findLowestSearchNoise(item)

		if noise == -1 {
			continue
		}

		item.Noise = noise
		item.Pattern = pattern

		if len(items) < context.MaxSearchedItems {
			items = append(items, item)
		} else {
			maxIndex, maxNoise := maxNoise()

			if maxNoise > noise {
				items[maxIndex] = item
			}
		}
	}
}

type SortItems struct {
	items []*FileItem
}

func (s *SortItems) Len() int {
	return len(s.items)
}

func (s *SortItems) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

type SortByNoiseAndText struct {
	SortItems
}

func (s *SortByNoiseAndText) Less(i, j int) bool {
	if s.items[i].Noise < s.items[j].Noise {
		return false
	} else if s.items[i].Noise > s.items[j].Noise {
		return true
	} else if len(s.items[i].Runes) < len(s.items[j].Runes) {
		return false
	} else if len(s.items[i].Runes) > len(s.items[j].Runes) {
		return true
	} else {
		ss := sort.StringSlice{s.items[i].Name, s.items[j].Name}
		return ss.Less(0, 1)
	}
}

type SortByText struct {
	SortItems
}

func (s *SortByText) Less(i, j int) bool {
	ss := sort.StringSlice{s.items[i].Name, s.items[j].Name}
	return ss.Less(0, 1)
}

func PrepareContent() ([]string, []string, string, []string) {
	if context.SearchText != "" {
		computeLowestNoises()
		sort.Sort(&SortByNoiseAndText{SortItems{items}})
	} else {
		if len(items) > context.MaxDisplayedItems {
			items = items[0:context.MaxDisplayedItems]
		}

		sort.Sort(&SortByText{SortItems{items}})
	}

	if context.SearchModeEnabled == 1 {
		if len(items) > context.MaxHeight {
			items = items[len(items)-context.MaxHeight : len(items)]
		}
	}

	itemSpace := 5

	content := make([]string, 0, len(items))
	indices := make([]string, 0, len(items))
	patterns := make(map[string]bool)

	for _, item := range items {
		line := append(make([]rune, 0, context.Columns), ' ', ' ')

		if len(item.Runes)+itemSpace > context.Columns {
			line = append(line, []rune(context.Dots)...)
			line = append(line, item.Runes[len(item.Runes)-context.Columns+itemSpace+context.DotsSize:]...)
		} else {
			line = append(line, item.Runes...)
		}

		for len(line) < context.Columns {
			line = append(line, ' ')
		}

		content = append(content, string(line))

		if len(item.Pattern) > 0 {
			patterns[item.Pattern] = true
		}

		indices = append(indices, strconv.Itoa(item.Index))
	}

	patternKeys := make([]string, 0, len(patterns))

	for k := range patterns {
		patternKeys = append(patternKeys, fmt.Sprintf("%q", k))
	}

	return patternKeys, indices, strconv.Itoa(len(items)), content
}

func main() {
	if err := Init(os.Stdin); err != nil {
		log.Fatalf("%#v", err)
	}

	patterns, indices, size, content := PrepareContent()

	fmt.Printf("[%s]\n", strings.Join(patterns, ","))
	fmt.Printf("[%s]\n", strings.Join(indices, ","))
	fmt.Println(size)

	for _, line := range content {
		fmt.Println(line)
	}
}
