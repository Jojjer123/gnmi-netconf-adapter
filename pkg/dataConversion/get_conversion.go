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

func ConvertAndSendReq(req *gnmi.GetRequest) *gnmi.GetResponse { //*gnmi.GetRequest, typeOfRequest string) {

	/************************************************************
	Implementation of data conversion should be implemented here.
	*************************************************************/
	//GetConfig("interfaces>interface", "running")
	// fmt.Println(sb.GetFullConfig())

	// TODO: Parse req and send path to sb.GetConfig(), might be good to change the input-params
	// in order to be more general and with less conversion of path.
	path, datastore, err := getRequestedPath(req)
	if err != nil {
		log.Warnf("Failed to get request path and datastore, %v", err)
	}

	xmlRequest := getXMLRequest(path, datastore, req.Type)

	// reply, err := sb.GetConfig(path, datastore, req.Type)
	// log.Info(reply)

	reply, err := sb.GetConfig(xmlRequest)

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

	return convertXMLtoGnmiResponse(reply)
}

func getRequestedPath(req *gnmi.GetRequest) ([]*gnmi.Path, string, error) {
	// requestedPath := ""
	requestedDatastore := ""

	// TODO: Implement all types of requests

	switch req.Type {
	case gnmi.GetRequest_ALL:
		log.Infof("Type: ALL")

	case gnmi.GetRequest_CONFIG:
		log.Infof("Type: CONFIG")
		requestedDatastore = "running"

	case gnmi.GetRequest_STATE:
		log.Infof("Type: STATE")

	case gnmi.GetRequest_OPERATIONAL:
		log.Infof("Type: OPERATIONAL")

	default:
		log.Warn("Request type not recognized!")
	}

	// for _, path := range req.Path {
	// 	for _, pathElem := range path.Elem {
	// 		// log.Info(pathElem.Name)
	// 		if requestedPath != "" {
	// 			requestedPath += ">"
	// 		}
	// 		requestedPath += pathElem.Name
	// 	}
	// }

	// return requestedPath, requestedDatastore, nil
	return req.Path, requestedDatastore, nil
}

func getXMLRequest(paths []*gnmi.Path, format string, reqType gnmi.GetRequest_DataType) string {
	var cmd string
	var endOfCmd string
	appendXMLTagOnType(&cmd, format, reqType, true)

	for _, path := range paths {
		for index, elem := range path.Elem {
			if index == 0 {
				cmd += "<filter>" // TODO: Look into filter types: <filter type="subtree"> etc.
				endOfCmd = "</filter>"
			}
			cmd += fmt.Sprintf("<%s", elem.Name)
			endOfCmd = fmt.Sprintf("</%s>", elem.Name) + endOfCmd

			// TODO: Add more keys if there are more, don't know yet.
			// Checks if namespace or name is defined before adding them to xml request.
			if namespace, ok := elem.Key["namespace"]; ok {
				cmd += fmt.Sprintf(" xmlns=\"%s\">", namespace)
			} else if name, ok := elem.Key["name"]; ok {
				cmd += fmt.Sprintf("><name>%s</name>", name)
			} else {
				cmd += ">"
			}
		}
		cmd += endOfCmd
	}

	appendXMLTagOnType(&cmd, format, reqType, false)

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
		log.Info("Requests of type GetRequest_ALL is not yet implemented!")

	case gnmi.GetRequest_OPERATIONAL:
		log.Info("Requests of type GetRequest_OPERATIONAL is not yet implemented!")

	default:
		log.Warn("Did not recognize request type!")
	}
}

func convertXMLtoGnmiResponse(xml string) *gnmi.GetResponse {
	// log.Info("Converting XML to GNMI response...")
	schema := netconfConv(xml)

	jsonDump, err := json.Marshal(schema)
	if err != nil {
		log.Warn("Failed to serialize schema!", err)
		// fmt.Println(err)
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
func netconfConv(xmlString string) *types.Schema {
	decoder := xml.NewDecoder(strings.NewReader(xmlString))
	schema := &types.Schema{}

	var newEntry *types.SchemaEntry
	var lastNamespace string

	index := 0
	for {
		tok, _ := decoder.Token()

		if tok == nil {
			return schema
		}

		switch tokType := tok.(type) {
		case xml.StartElement:
			newEntry = &types.SchemaEntry{}
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
			newEntry = &types.SchemaEntry{}
			newEntry.Name = tokType.Name.Local
			newEntry.Tag = "end"
			schema.Entries = append(schema.Entries, *newEntry)
			index++

		case xml.CharData:
			bytes := xml.CharData(tokType)
			schema.Entries[index-1].Value = string([]byte(bytes))

		default:
			log.Warnf("Token type was not recognized with type: %v", tokType)
			// fmt.Println(tokType)
		}
	}
}
