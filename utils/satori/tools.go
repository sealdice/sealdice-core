package satori

import (
	"encoding/xml"
	"strings"
)

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
			// 对tag attr弄一下白名单？
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

func ContentEscape(text string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return replacer.Replace(text)
}

func ContentUnescape(str string) string {
	replacer := strings.NewReplacer("&quot;", "\"", "&gt;", ">", "&lt;", "<", "&amp;", "&")
	return replacer.Replace(str)
}
