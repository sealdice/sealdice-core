package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	"sealdice-core/dice/utils/onebot/schema"
)

var (
	ErrNoAvailable = errors.New("no replyer available")
	ErrNotFound    = errors.New("not found")
)

type Messager interface {
	Eventer
	TextFirst() (*schema.Text, error)
	Texts() ([]schema.Text, int)
	FaceFirst() (*schema.Face, error)
	Faces() ([]schema.Face, int)
	AtFirst() (*schema.At, error)
	Ats() ([]schema.At, int)
	ImageFirst() (*schema.Image, error)
	Images() ([]schema.Image, int)
}

type CommonMessage struct {
	SubType    string           `json:"sub_type"`
	MessageId  int              `json:"message_id"`
	UserId     int64            `json:"user_id"`
	Messages   []schema.Message `json:"message"`
	RawMessage string           `json:"raw_message"`
	Font       int              `json:"font"`
	Sender     schema.Sender    `json:"sender"`
}

func (cm CommonMessage) Id() int {
	return cm.MessageId
}
func (cm CommonMessage) Reply(replyer Replyer, text string) error {
	if replyer == nil {
		return ErrNoAvailable
	}
	data := struct {
		Reply string `json:"reply"`
	}{Reply: text}
	return replyer.Reply(data)
}

// yield the rawMessage by type
func (cm CommonMessage) FilterType(Type string) iter.Seq[json.RawMessage] {
	return func(yield func(json.RawMessage) bool) {
		for _, msg := range cm.Messages {
			if msg.Type == Type {
				if !yield(msg.Data) {
					return
				}
			}
		}
	}
}

// yield all messages use type and raw
func (cm CommonMessage) All() iter.Seq2[string, json.RawMessage] {
	return func(yield func(string, json.RawMessage) bool) {
		for _, msg := range cm.Messages {
			if !yield(msg.Type, msg.Data) {
				return
			}
		}
	}
}

func (cm CommonMessage) Texts() ([]schema.Text, int) {
	return all[schema.Text]("text", cm.Messages)
}

func (cm CommonMessage) TextFirst() (*schema.Text, error) {
	return first[schema.Text]("text", cm.Messages)
}

func (cm CommonMessage) Faces() ([]schema.Face, int) {
	return all[schema.Face]("face", cm.Messages)
}

func (cm CommonMessage) FaceFirst() (*schema.Face, error) {
	return first[schema.Face]("face", cm.Messages)
}

func (cm CommonMessage) Ats() ([]schema.At, int) {
	return all[schema.At]("at", cm.Messages)
}

func (cm CommonMessage) AtFirst() (*schema.At, error) {
	return first[schema.At]("at", cm.Messages)
}

func (cm CommonMessage) Images() ([]schema.Image, int) {
	return all[schema.Image]("image", cm.Messages)
}

func (cm CommonMessage) ImageFirst() (*schema.Image, error) {
	return first[schema.Image]("image", cm.Messages)
}

func (cm CommonMessage) File() (*schema.CommonFile, error) {
	return first[schema.CommonFile]("file", cm.Messages)
}

func (em CommonMessage) Record() (*schema.Record, error) {
	return first[schema.Record]("record", em.Messages)
}

func first[T any](msgType string, msg []schema.Message) (*T, error) {
	for _, msg := range msg {
		if msg.Type == msgType {
			var data T
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				return nil, err
			}
			return &data, nil
		}
	}
	return nil, ErrNotFound
}

func all[T any](msgType string, msg []schema.Message) ([]T, int) {
	var data []T
	for _, msg := range msg {
		if msg.Type == msgType {
			var d T
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				continue
			}
			data = append(data, d)
		}
	}
	return data, len(data)
}

type PrivateMessage struct {
	CommonMessage
}

func (pm PrivateMessage) Type() string {
	return "message:private"
}

func (pm PrivateMessage) SessionKey() string {
	return fmt.Sprintf("%s:%d", pm.Type(), pm.UserId)
}

type GroupMessage struct {
	CommonMessage
	GroupId   int64     `json:"group_id"`
	Anonymous Anonymous `json:"anonymous"`
}

type Anonymous struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
	Flag string `json:"flag"`
}

func (gm GroupMessage) Type() string {
	return "message:group"
}

func (gm GroupMessage) SessionKey() string {
	return fmt.Sprintf("%s:%d", gm.Type(), gm.GroupId)
}

type AllMessage struct {
	CommonMessage
	GroupId   int64     `json:"group_id"`
	Anonymous Anonymous `json:"anonymous"`
}

func (am AllMessage) Type() string {
	return "message"
}

func (am AllMessage) SessionKey() string {
	return fmt.Sprintf("%s:%s:%d", am.Type(), am.SubType, am.GroupId)
}
