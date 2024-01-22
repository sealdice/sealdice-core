package satori

import (
	"fmt"
	"strings"
)

type Element struct {
	// KElement bool      `json:"kElement"`
	Type     string     `json:"type"`
	Attrs    Dict       `json:"attrs"`
	Children []*Element `json:"children"`
	// Source   string    `json:"source"` // js版本自己也不写source，所以姑且注释掉
}

func (el *Element) Traverse(fn func(el *Element)) {
	fn(el)
	for _, child := range el.Children {
		child.Traverse(fn)
	}
}

func (el *Element) Traverse2(fn, fn2 func(el *Element)) {
	fn(el)
	for _, child := range el.Children {
		child.Traverse2(fn, fn2)
	}
	fn2(el)
}

func (el *Element) ToString() string {
	var sb strings.Builder
	el.Traverse2(func(el *Element) {
		switch el.Type {
		case "root":
			break
		case "text":
			sb.WriteString(el.Attrs["content"].(string))
		default:
			sb.WriteString(fmt.Sprintf("<%s", el.Type))
			for k, v := range el.Attrs {
				sb.WriteString(fmt.Sprintf(" %s=\"%s\"", k, v))
			}
			sb.WriteString(">")
		}
	}, func(el *Element) {
		switch el.Type {
		case "root", "text":
			break
		default:
			sb.WriteString(fmt.Sprintf("</%s>", el.Type))
		}
	})
	return sb.String()
}

func FromCQCode(text string) *Element {
	// TODO
	return nil
}

func (el *Element) ToCQCode() string {
	// TODO
	return ""
}
