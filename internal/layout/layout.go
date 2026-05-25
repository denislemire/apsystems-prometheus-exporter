package layout

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Panel struct {
	UID     string `json:"uid"`
	Channel int    `json:"channel"`
	Array   string `json:"array"`
	Row     int    `json:"row"`
	Col     int    `json:"col"`
}

type File struct {
	Panels map[string]Panel `json:"panels"`
}

type Labels struct {
	UID     string
	Channel string
	Array   string
	Row     string
	Col     string
}

func Load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read layout: %w", err)
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("parse layout: %w", err)
	}
	if f.Panels == nil {
		f.Panels = map[string]Panel{}
	}
	return f, nil
}

func (f File) Labels(key string) Labels {
	if p, ok := f.Panels[key]; ok {
		return Labels{
			UID:     p.UID,
			Channel: strconv.Itoa(p.Channel),
			Array:   p.Array,
			Row:     strconv.Itoa(p.Row),
			Col:     strconv.Itoa(p.Col),
		}
	}
	uid, channel, ok := splitKey(key)
	if !ok {
		return Labels{UID: key, Channel: "0", Array: "unknown", Row: "0", Col: "0"}
	}
	return Labels{UID: uid, Channel: channel, Array: "unknown", Row: "0", Col: "0"}
}

func splitKey(key string) (uid, channel string, ok bool) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '-' {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}
