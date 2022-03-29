package dataConversion

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	sb "github.com/onosproject/gnmi-netconf-adapter/pkg/southbound"
	"github.com/openconfig/gnmi/proto/gnmi"
)

func ConvertAndSendReq(req *gnmi.GetRequest) *gnmi.GetResponse { //*gnmi.GetRequest, typeOfRequest string) {

	/************************************************************
	Implementation of data conversion should be implemented here.
	*************************************************************/
	//GetConfig("interfaces>interface", "running")
	// fmt.Println(sb.GetFullConfig())

	reply := sb.GetFullConfig()

	// If southbound fails to get config, return empty response
	if reply == nil {
		notifications := make([]*gnmi.Notification, 1)
		ts := time.Now().UnixNano()

		notifications[0] = &gnmi.Notification{
			Timestamp: ts,
		}

		return &gnmi.GetResponse{Notification: notifications}
	}

	return convertXMLtoGnmiResponse(reply.Data)
}

type Schema struct {
	Entries []SchemaEntry
}

type SchemaEntry struct {
	Name      string
	Tag       string
	Namespace string
	Value     string
}

func convertXMLtoGnmiResponse(xml string) *gnmi.GetResponse {
	log.Info("Converting XML to GNMI response...")
	schema := netconfConv(xml)

	jsonDump, err := json.Marshal(schema)
	if err != nil {
		fmt.Println("Failed to serialize schema!")
		fmt.Println(err)
	}

	notifications := []*gnmi.Notification{
		{
			Timestamp: time.Now().UnixNano(),
			Update: []*gnmi.Update{
				{Val: &gnmi.TypedValue{Value: &gnmi.TypedValue_BytesVal{BytesVal: jsonDump}}},
			},
		},
	}

	return &gnmi.GetResponse{Notification: notifications}
}

// Converts XML to a Schema containing a slice of all the elements with namespaces and values.
// TODO: Add "searching" to filter out all data except what the path is requesting.
func netconfConv(xmlString string) *Schema {
	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	schema := &Schema{}

	var newEntry *SchemaEntry
	var lastNamespace string

	index := 0
	for {
		tok, _ := decoder.Token()

		if tok == nil {
			return schema
		}

		switch tokType := tok.(type) {
		case xml.StartElement:
			newEntry = &SchemaEntry{}
			newEntry.Name = tokType.Name.Local

			if index > 0 {
				if tokType.Name.Space != lastNamespace {
					lastNamespace = tokType.Name.Space
					newEntry.Namespace = lastNamespace
				}
				newEntry.Tag = "start"
			} else {
				lastNamespace = tokType.Name.Space
				newEntry.Namespace = lastNamespace
			}

			schema.Entries = append(schema.Entries, *newEntry)
			index++

		case xml.EndElement:
			newEntry = &SchemaEntry{}
			newEntry.Name = tokType.Name.Local
			newEntry.Tag = "end"
			schema.Entries = append(schema.Entries, *newEntry)
			index++

		case xml.CharData:
			bytes := xml.CharData(tokType)
			schema.Entries[index-1].Value = string([]byte(bytes))

		default:
			fmt.Print(", was not recognized with type: ")
			fmt.Println(tokType)
		}
	}
}
