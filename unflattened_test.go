package unflattened_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	uf "github.com/x4t3a/unflattened"
)

type (
	XMLObject struct {
		Name              xml.Name     // XML node kind/name.
		Id                string       // XML node id. Let's store it separately from others attributes.
		Attrs             []xml.Attr   // XML node attributes.
		ObjKey, ParObjKey string       // Used to save hierarchical info after data flattening.
		Children          []*XMLObject // Used to store children after Unmarshal'ing. Empty after flattening.
	}
)

func (obj *XMLObject) UFKey() string {
	return obj.ObjKey
}

func (obj *XMLObject) UFParentKey() string {
	return obj.ParObjKey
}

func (obj *XMLObject) UFAppendChild(child uf.UnFlattenable) error {
	if obj == nil {
		return fmt.Errorf("nil receiver")
	}

	if child == nil {
		return fmt.Errorf("nil argument")
	}

	if childObj, castable := child.(*XMLObject); castable {
		obj.Children = append(obj.Children, childObj)
	} else {
		return fmt.Errorf("wrong cast")
	}

	return nil
}

func (obj *XMLObject) UFGetChildren() ([]uf.UnFlattenable, error) {
	children := make([]uf.UnFlattenable, 0, len(obj.Children))
	for _, child := range obj.Children {
		children = append(children, child)
	}

	return children, nil
}

func TestUnFlattenXML(t *testing.T) {
	const input = `
		<a>
			<b id='1' battr='battr-val'>
				<c id='b11'/>
				<c id='b12'/>
				<c id='b13'>
					<d id='c131' dattr='dattr-val'/>
				</c>
			</b>
			<b id='2'>
				<c id='b21'/>
				<c id='b22'/>
			</b>
			<b id='3'/>
		</a>
	`

	var xmlObjModel XMLObject

	if err := xml.Unmarshal([]byte(input), &xmlObjModel); err != nil {
		t.Fatal(err)
	}

	if xmlEntities, err := uf.Flatten(&xmlObjModel); err == nil {
		if unflattened, err := uf.Unflatten(xmlEntities); err == nil {
			if len(unflattened) != 1 {
				t.Fatal("can be only 1 root in this test")
			}

			unflattenedObj := unflattened[0]
			if marshaledBytes, err := xml.MarshalIndent(unflattenedObj, "", "    "); err == nil {
				t.Log(string(marshaledBytes))
			} else {
				t.Fatal(err)
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

func (obj *XMLObject) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	obj.Name = start.Name

	for _, attr := range start.Attr {
		if attr.Name.Local == "id" {
			obj.Id = attr.Value
		} else {
			obj.Attrs = append(obj.Attrs, attr)
		}
	}

	obj.ObjKey = GenerateUniqueKey()

	for {
		if token, err := d.Token(); err == nil {
			switch t := token.(type) {
			case xml.StartElement:
				child := &XMLObject{ParObjKey: obj.ObjKey}

				if err := d.DecodeElement(child, &t); err != nil {
					return err
				}

				obj.Children = append(obj.Children, child)
			case xml.EndElement:
				d.Skip()
			case xml.Comment, xml.CharData:
				continue
			default:
				return fmt.Errorf("unimplemented")
			}
		} else {
			if err == io.EOF {
				return nil
			}

			return err
		}
	}
}

func (obj XMLObject) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	el := xml.StartElement{Name: obj.Name}

	if len(obj.Id) > 0 {
		el.Attr = []xml.Attr{{Name: xml.Name{Local: "id"}, Value: obj.Id}}
	}

	if len(obj.Attrs) > 0 {
		el.Attr = append(el.Attr, obj.Attrs...)
	}

	if err := e.EncodeToken(el); err != nil {
		return err
	}

	for _, chEl := range obj.Children {
		if err := e.EncodeElement(chEl, el); err != nil {
			return err
		}
	}

	if err := e.EncodeToken(xml.EndElement{Name: obj.Name}); err != nil {
		return err
	}

	return nil
}

// The code below is auxiliary. Used to generate pseudo-random unique identifiers used for unflattening.
// Stolen from https://stackoverflow.com/a/31832326 :)

func GenerateUniqueKey() string {
	return RandStringRunes(5)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
