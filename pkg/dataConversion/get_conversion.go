package dataConversion

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	sb "github.com/onosproject/gnmi-netconf-adapter/pkg/southbound"
	types "github.com/onosproject/gnmi-netconf-adapter/pkg/types"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// TODO: Make it work for more than one path per request.
func ConvertAndSendReq(req *gnmi.GetRequest) *gnmi.GetResponse {
	datastore, err := getRequestedDatastore(req)
	if err != nil {
		log.Warnf("Failed to get datastore requested, %v", err)
	}

	xmlRequest := getXMLRequest(req.Path, datastore, req.Type)
	// log.Info(xmlRequest)

	reply, err := sb.GetConfig(xmlRequest)

	// log.Info(reply)

	// If southbound fails to get config, return empty response
	if err != nil {
		log.Warnf("Response from switch was: %v", err)
		notifications := make([]*gnmi.Notification, 1)
		ts := time.Now().UnixNano()

		notifications[0] = &gnmi.Notification{
			Timestamp: ts,
		}

		return &gnmi.GetResponse{Notification: notifications}
	}

	return convertXMLtoGnmiResponse(reply /*, req.Path[0]*/)
}

func getRequestedDatastore(req *gnmi.GetRequest) (string, error) {
	requestedDatastore := ""

	// TODO: Implement all types of requests
	switch req.Type {
	case gnmi.GetRequest_ALL:
		log.Info("Type: ALL")
		requestedDatastore = "running"

	case gnmi.GetRequest_CONFIG:
		log.Info("Type: CONFIG")
		requestedDatastore = "running"

	case gnmi.GetRequest_STATE:
		log.Info("Type: STATE")

	case gnmi.GetRequest_OPERATIONAL:
		log.Info("Type: OPERATIONAL")
		requestedDatastore = "running"

	default:
		log.Warn("Request type not recognized!")
	}

	return requestedDatastore, nil
}

func getXMLRequest(paths []*gnmi.Path, format string, reqType gnmi.GetRequest_DataType) string {
	var cmd string
	var endOfCmd string
	appendXMLTagOnType(&cmd, format, reqType, true)

	for _, path := range paths {
		for index, elem := range path.Elem {
			if index == 0 {
				// TODO: Look into filter types: <filter type="subtree"> etc.
				cmd += "<filter>"
				endOfCmd = "</filter>"
			}
			cmd += fmt.Sprintf("<%s", elem.Name)
			endOfCmd = fmt.Sprintf("</%s>", elem.Name) + endOfCmd

			// TODO: Add more keys if there are more, don't know yet.
			// Checks if namespace or name is defined before adding them to xml request.
			if namespace, ok := elem.Key["namespace"]; ok {
				cmd += fmt.Sprintf(" xmlns=\"%s\">", namespace)
			} else {
				cmd += ">"
			}

			if len(elem.Key) > 0 {
				for key, value := range elem.Key {
					if key != "namespace" {
						cmd += fmt.Sprintf("<%s>%s</%s>", key, value, key)
					}
				}
			}
			// else if name, ok := elem.Key["name"]; ok {
			// 	cmd += fmt.Sprintf("><name>%s</name>", name)
			// }
		}
		cmd += endOfCmd
	}

	appendXMLTagOnType(&cmd, format, reqType, false)

	// log.Info(cmd)

	return cmd
}

func appendXMLTagOnType(cmd *string, format string,
	reqType gnmi.GetRequest_DataType, startTags bool) {

	switch reqType {
	case gnmi.GetRequest_CONFIG:
		if startTags {
			*cmd += fmt.Sprintf("<get-config><source><%s/></source>", format)
		} else {
			*cmd += "</get-config>"
		}

	case gnmi.GetRequest_STATE:
		if startTags {
			*cmd += "<get>"
		} else {
			*cmd += "</get>"
		}

	case gnmi.GetRequest_ALL:
		log.Info("Requests of type GetRequest_ALL is not yet tested, runs same as GetRequest_CONFIG!")
		if startTags {
			*cmd += fmt.Sprintf("<get-config><source><%s/></source>", format)
		} else {
			*cmd += "</get-config>"
		}

	case gnmi.GetRequest_OPERATIONAL:
		log.Info("Requests of type GetRequest_OPERATIONAL is not yet tested, runs same as GetRequest_CONFIG!")
		if startTags {
			*cmd += fmt.Sprintf("<get-config><source><%s/></source>", format)
		} else {
			*cmd += "</get-config>"
		}

	default:
		log.Warn("Did not recognize request type!")
	}
}

func convertXMLtoGnmiResponse(xml string /*, path *gnmi.Path*/) *gnmi.GetResponse {
	schema := netconfConv(xml /*, path*/)

	jsonDump, err := json.Marshal(schema)
	if err != nil {
		log.Warn("Failed to serialize schema!", err)
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
// TODO: Add "searching" to filter out all data except for what the path is requesting.
func netconfConv(xmlString string /*, path *gnmi.Path*/) *types.Schema {
	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	schema := &types.Schema{}

	var newEntry *types.SchemaEntry
	// var lastNamespace string

	var nsParser *types.NamespaceParser

	// elementsToBeFound := path.Elem
	// elementToBeFoundIndex := 0
	index := 0
	for {
		tok, _ := decoder.Token()

		if tok == nil {
			fmt.Println("")
			return schema
		}

		switch tokType := tok.(type) {
		case xml.StartElement:
			// fmt.Println(tokType.Name.Local)
			newEntry = &types.SchemaEntry{}
			newEntry.Name = tokType.Name.Local

			if index > 0 {
				newNsParser := &types.NamespaceParser{
					Parent:              nsParser,
					LastParentNamespace: nsParser.LastParentNamespace,
				}

				// fmt.Print(tokType.Name.Local)
				// fmt.Printf(" - %v", tokType.Attr)

				if nsParser.LastParentNamespace != tokType.Name.Space {
					newNsParser.LastParentNamespace = tokType.Name.Space
					newEntry.Namespace = tokType.Name.Space
					// fmt.Printf(" - %s\n", newNsParser.LastParentNamespace)
				} else {
					// fmt.Print("\n")
				}

				nsParser.Children = append(nsParser.Children, newNsParser)

				nsParser = newNsParser

				newEntry.Tag = "start"
			} else {
				nsParser = &types.NamespaceParser{
					LastParentNamespace: tokType.Name.Space,
					Parent:              nil,
				}

				// fmt.Printf("%s - %s\n", newEntry.Name, nsParser.LastParentNamespace)e
				newEntry.Namespace = tokType.Name.Space
			}

			// var key string
			// if elementToBeFoundIndex > 0 {
			// 	for key, _ := range elementsToBeFound[elementToBeFoundIndex-1].Key {
			// 		if key != "namespace" {
			// 			break
			// 		}
			// 	}
			// }

			// if elementsToBeFound[elementToBeFoundIndex].Name == newEntry.Name || newEntry.Name == "data" || key == newEntry.Name {
			schema.Entries = append(schema.Entries, *newEntry)
			// 	if key == newEntry.Name {
			// 		elementToBeFoundIndex -= 2
			// 	}
			// 	elementToBeFoundIndex++
			// }
			index++

		case xml.EndElement:
			// fmt.Printf("Exiting %s now\n", tokType.Name.Local)
			nsParser = nsParser.Parent

			newEntry = &types.SchemaEntry{}
			newEntry.Name = tokType.Name.Local
			newEntry.Tag = "end"

			// var key string
			// if elementToBeFoundIndex > 0 {
			// 	for key, _ := range elementsToBeFound[elementToBeFoundIndex].Key {
			// 		if key != "namespace" {
			// 			break
			// 		}
			// 	}
			// }

			// var searchIndex int
			// if elementToBeFoundIndex-1 <= 0 {
			// 	searchIndex = 0
			// } else {
			// 	searchIndex = elementToBeFoundIndex - 1
			// }

			// if elementsToBeFound[searchIndex].Name == newEntry.Name || newEntry.Name == "data" || key == newEntry.Name {
			schema.Entries = append(schema.Entries, *newEntry)
			// }
			index++

		case xml.CharData:
			bytes := xml.CharData(tokType)
			schema.Entries[index-1].Value = string([]byte(bytes))

		default:
			fmt.Printf("Token type was not recognized with type: %v", tokType)
		}
	}
}

// Converts XML to a Schema containing a slice of all the elements with namespaces and values.
// TODO: Add "searching" to filter out all data except for what the path is requesting.
// func netconfConv(xmlString string) *types.Schema {
// 	decoder := xml.NewDecoder(strings.NewReader(xmlString))
// 	schema := &types.Schema{}

// 	var newEntry *types.SchemaEntry
// 	var lastNamespace string

// 	var nsParser *types.NamespaceParser

// 	index := 0
// 	for {
// 		tok, _ := decoder.Token()

// 		if tok == nil {
// 			return schema
// 		}

// 		switch tokType := tok.(type) {
// 		case xml.StartElement:
// 			newEntry = &types.SchemaEntry{}
// 			newEntry.Name = tokType.Name.Local

// 			if index > 0 {
// 				newNsParser := &types.NamespaceParser{
// 					Parent: nsParser,
// 				}

// 				if nsParser.LastParentNamespace != tokType.Name.Space {
// 					newNsParser.LastParentNamespace = tokType.Name.Space
// 					fmt.Printf("%s - %s", newEntry.Name, newNsParser.LastParentNamespace)
// 				}

// 				nsParser.Children = append(nsParser.Children, newNsParser)

// 				nsParser = newNsParser

// 				// TODO: Fix namespaces, it currently won't add modules-state which is a module
// 				// with exactly the same namespace as the previous module yang-library for state-data
// 				if tokType.Name.Space != lastNamespace {
// 					lastNamespace = tokType.Name.Space
// 					newEntry.Namespace = lastNamespace
// 				}
// 				newEntry.Tag = "start"
// 			} else {
// 				nsParser.LastParentNamespace = tokType.Name.Space
// 				nsParser.Parent = nil
// 				fmt.Printf("%s - %s", newEntry.Name, nsParser.LastParentNamespace)

// 				lastNamespace = tokType.Name.Space
// 				newEntry.Namespace = lastNamespace
// 			}

// 			schema.Entries = append(schema.Entries, *newEntry)
// 			index++

// 		case xml.EndElement:
// 			newEntry = &types.SchemaEntry{}
// 			newEntry.Name = tokType.Name.Local
// 			newEntry.Tag = "end"
// 			schema.Entries = append(schema.Entries, *newEntry)
// 			index++

// 		case xml.CharData:
// 			bytes := xml.CharData(tokType)
// 			schema.Entries[index-1].Value = string([]byte(bytes))

// 		default:
// 			log.Warnf("Token type was not recognized with type: %v", tokType)
// 		}
// 	}
// }
