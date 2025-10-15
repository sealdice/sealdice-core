package schema

import "encoding/json"

type MessageChain []Message

func (m MessageChain) Append(msg Message) MessageChain {
	return append(m, msg)
}

func (m MessageChain) Text(text string) MessageChain {
	data, err := json.Marshal(Text{
		Text: text,
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "text",
		Data: data,
	})
}

func (m MessageChain) Br() MessageChain {
	return m.Text("\n")
}

func (m MessageChain) Face(id string) MessageChain {
	data, err := json.Marshal(Face{
		Id: id,
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "face",
		Data: data,
	})
}

func (m MessageChain) At(qq string) MessageChain {
	data, err := json.Marshal(At{
		QQ: qq,
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "at",
		Data: data,
	})
}

func (m MessageChain) Reply(id int) MessageChain {
	data, err := json.Marshal(Reply{
		Id: id,
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "reply",
		Data: data,
	})
}

// such as:
// network URL: https://www.example.com/image.png
// local file:///C:\\Users\Richard\Pictures\1.png see rfc 8089
// base64: base64://9j/4AAQSkZJRgABAQEAAAAAAAD/...
func (m MessageChain) Image(file string) MessageChain {
	data, err := json.Marshal(Image{
		CommonFile: CommonFile{
			File: file,
		},
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "image",
		Data: data,
	})
}

// such as:
// network URL: https://www.example.com/image.png
// local file:///C:\\Users\Richard\Pictures\1.png see rfc 8089
func (m MessageChain) File(file string) MessageChain {
	data, err := json.Marshal(CommonFile{
		File: file,
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "file",
		Data: data,
	})
}

func (m MessageChain) Record(file string) MessageChain {
	data, err := json.Marshal(CommonFile{
		File: file,
	})
	if err != nil {
		panic(err)
	}
	return m.Append(Message{
		Type: "record",
		Data: data,
	})
}
