package satori

import (
	"encoding/xml"
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

type Dict map[string]interface{}

func ElementParse(text string) *Element {
	decoder := xml.NewDecoder(strings.NewReader(text))

	var elStack []*Element
	// 添加一个临时的根节点，这样逻辑上更容易处理
	elStack = append(elStack, &Element{Type: "root"})

	appendToChild := func(el *Element) *Element {
		top := elStack[len(elStack)-1]
		top.Children = append(top.Children, el)
		return el
	}

	appendToChildAndPush := func(el *Element) {
		appendToChild(el)
		elStack = append(elStack, el)
	}

	popElement := func() {
		elStack = elStack[:len(elStack)-1]
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			// 对tag弄一下白名单？
			attrs := Dict{}
			for _, attr := range se.Attr {
				attrs[attr.Name.Local] = attr.Value
			}

			appendToChildAndPush(&Element{
				Type:  se.Name.Local,
				Attrs: attrs,
			})
		case xml.EndElement:
			popElement()
		case xml.CharData:
			appendToChild(&Element{
				Type:  "text",
				Attrs: Dict{"content": string(se)},
			})
			// case xml.Comment:
			//	fmt.Printf("Comment: %s\n", se)
			// case xml.ProcInst:
			//	fmt.Printf("ProcInst: %s %s\n", se.Target, se.Inst)
			// case xml.Directive:
			//	fmt.Printf("Directive: %s\n", se)
			// default:
			//	fmt.Printf("Unknown element\n")
		}
	}

	return elStack[0]
}
