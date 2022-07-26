package dataConversion

import (
	// "encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"

	sb "github.com/onosproject/gnmi-netconf-adapter/pkg/southbound"
	"github.com/onosproject/gnmi-netconf-adapter/pkg/types"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// TODO: Make it work for more than only the first path per request.
func ConvertAndSendGetReq(req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	datastore, err := getRequestedDatastore(req)
	if err != nil {
		log.Errorf("Failed to get datastore requested, %v", err)
	}

	// startTimeReq := time.Now().UnixNano()
	xmlRequests := getXMLRequests(req.Path, datastore, req.Type)
	// log.Infof("Time to create xmlReq: %v\n", time.Now().UnixNano()-startTimeReq)

	// startTimeGetConf := time.Now().UnixNano()

	// log.Infof("Should have sent the following reqs to %s : %v", req.Path[0].Target, xmlRequests)
	// reply := ""
	reply, err := sb.GetConfig(xmlRequests, req.Path[0].Target)
	// log.Infof("Time to receive conf/counter(s): %v\n", time.Now().UnixNano()-startTimeGetConf)
	if err != nil {
		log.Errorf("Failed to get response from switch: %v\n", err)

		notifications := make([]*gnmi.Notification, 1)
		ts := time.Now().UnixNano()

		notifications[0] = &gnmi.Notification{
			Timestamp: ts,
		}

		return &gnmi.GetResponse{Notification: notifications}, err
	}

	return convertXMLtoGnmiResponse(reply), nil
}

func getRequestedDatastore(req *gnmi.GetRequest) (string, error) {
	requestedDatastore := ""

	// TODO: Test all types of requests.
	switch req.Type {
	case gnmi.GetRequest_ALL:
		// log.Info("Type: ALL")
		requestedDatastore = "running"

	case gnmi.GetRequest_CONFIG:
		// log.Info("Type: CONFIG")
		requestedDatastore = "running"

	case gnmi.GetRequest_STATE:
		// log.Info("Type: STATE")

	case gnmi.GetRequest_OPERATIONAL:
		// log.Info("Type: OPERATIONAL")
		requestedDatastore = "running"

	default:
		log.Warn("Request type not recognized!")
	}

	return requestedDatastore, nil
}

func getXMLRequests(paths []*gnmi.Path, datastore string, reqType gnmi.GetRequest_DataType) []string {
	var cmds []string
	var cmd string
	var endOfCmd string

	for pathIndex, path := range paths {
		cmd = ""
		endOfCmd = ""

		if pathIndex == 0 {
			// If there are no elements requested, the full configuration + state data is requested.
			if len(path.Elem) == 0 {
				return []string{"<get/>"}
			}

			appendXMLTagOnType(&cmd, datastore, reqType, true)
			// TODO: Look into filter types: <filter type="subtree"> etc.
			cmd += "<filter type='subtree'>"
		}

		if pathIndex == len(paths)-1 {
			endOfCmd = "</filter>"
		}

		for _, elem := range path.Elem {
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
		if pathIndex == len(paths)-1 {
			appendXMLTagOnType(&cmd, datastore, reqType, false)
		}
		cmds = append(cmds, cmd)
	}

	// log.Info(cmds)

	return cmds
}

func appendXMLTagOnType(cmd *string, datastore string,
	reqType gnmi.GetRequest_DataType, startTags bool) {

	switch reqType {
	case gnmi.GetRequest_CONFIG:
		if startTags {
			*cmd += fmt.Sprintf("<get-config><source><%s/></source>", datastore)
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
			*cmd += fmt.Sprintf("<get-config><source><%s/></source>", datastore)
		} else {
			*cmd += "</get-config>"
		}

	case gnmi.GetRequest_OPERATIONAL:
		log.Info("Requests of type GetRequest_OPERATIONAL is not yet tested, runs same as GetRequest_CONFIG!")
		if startTags {
			*cmd += fmt.Sprintf("<get-config><source><%s/></source>", datastore)
		} else {
			*cmd += "</get-config>"
		}

	default:
		log.Warn("Did not recognize request type!")
	}
}

func convertXMLtoGnmiResponse(xml string /*, path *gnmi.Path*/) *gnmi.GetResponse {
	// log.Infof("XML string: %v\n", xml)
	adapterResponse := netconfConv(xml /*, path*/)
	adapterResponse.Timestamp = time.Now().UnixNano()

	serializedData, err := proto.Marshal(adapterResponse)
	if err != nil {
		fmt.Printf("error marshaling response using proto: %v", err)
	}

	notifications := []*gnmi.Notification{
		{
			Timestamp: time.Now().UnixNano(),
			Update: []*gnmi.Update{
				{Val: &gnmi.TypedValue{Value: &gnmi.TypedValue_ProtoBytes{ProtoBytes: serializedData}}},
			},
		},
	}

	// log.Infof("Notifications: %v\n", notifications)

	return &gnmi.GetResponse{Notification: notifications}
}

// Converts XML to a Schema containing a slice of all the elements with namespaces and values.
// Consider reworking this to send back the original type, not a types.Schema.
// TODO: Add "searching" to filter out all data except for what the path is requesting.
func netconfConv(xmlString string /*, path *gnmi.Path*/) *types.AdapterResponse {
	// startTime := time.Now().UnixNano()
	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	adapterResponse := &types.AdapterResponse{}

	var newEntry *types.SchemaEntry
	var nsParser *types.NamespaceParser

	index := 0
	for {
		tok, _ := decoder.Token()

		if tok == nil {
			// fmt.Println("")
			// log.Infof("Time to convert from xml: %v\n", time.Now().UnixNano()-startTime)
			return adapterResponse
		}

		switch tokType := tok.(type) {
		case xml.StartElement:
			newEntry = &types.SchemaEntry{}
			newEntry.Name = tokType.Name.Local

			if index > 0 {
				newNsParser := &types.NamespaceParser{
					Parent:              nsParser,
					LastParentNamespace: nsParser.LastParentNamespace,
				}

				// Could be used to add attribute namespaces, NOT tested.
				if len(tokType.Attr) > 0 {
					// fmt.Print(tokType.Name.Local)
					// fmt.Printf(" - %s , %s", tokType.Attr[0].Name, tokType.Attr[0].Value)

					if nsParser.LastParentNamespace != tokType.Attr[0].Value {
						// newNsParser.LastParentNamespace = tokType.Attr[0].Value
						// newEntry.Namespace = tokType.Attr[0].Value
					}
				}

				if nsParser.LastParentNamespace != tokType.Name.Space {
					newNsParser.LastParentNamespace = tokType.Name.Space
					newEntry.Namespace = tokType.Name.Space
				}

				nsParser.Children = append(nsParser.Children, newNsParser)
				nsParser = newNsParser
				newEntry.Tag = "start"
			} else {
				nsParser = &types.NamespaceParser{
					LastParentNamespace: tokType.Name.Space,
					Parent:              nil,
				}

				newEntry.Namespace = tokType.Name.Space
			}

			adapterResponse.Entries = append(adapterResponse.Entries, *newEntry)
			index++

		case xml.EndElement:
			nsParser = nsParser.Parent

			newEntry = &types.SchemaEntry{}
			newEntry.Name = tokType.Name.Local
			newEntry.Tag = "end"

			adapterResponse.Entries = append(adapterResponse.Entries, *newEntry)
			index++

		case xml.CharData:
			bytes := xml.CharData(tokType)
			adapterResponse.Entries[index-1].Value = string([]byte(bytes))

		default:
			fmt.Printf("Token type was not recognized with type: %v", tokType)
		}
	}
}
