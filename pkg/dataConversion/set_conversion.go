package dataConversion

import (
	"fmt"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"

	sb "github.com/onosproject/gnmi-netconf-adapter/pkg/southbound"
)

var log = logging.GetLogger("main")

// TODO: Make it work for more than only the first path per request.
func ConvertAndSendSetReq(req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	// Create a map holding groups of update requests for every switch targeted.
	switches := make(map[string][]*gnmi.Update)

	// For every update in set request, store updates for every switch (to group updates).
	for _, update := range req.Update {
		switches[update.Path.Target] = append(switches[update.Path.Target], update)
	}

	var switchRequests []string

	// For every switch with at least one update, convert to XML.
	for _, s := range switches {
		var switchSetReq string

		// TODO: Make target dynamic, currently only changes the running configuration.
		// Create default start structure of XML.
		// switchSetReq = "<edit-config>"
		// switchSetReq += fmt.Sprintf("<target><%s/></target>", "running")
		// switchSetReq += fmt.Sprintf("<test-option>%s</test-option>", "test-then-set")
		// switchSetReq += fmt.Sprintf("<error-option>%s</error-option>", "rollback-on-error")
		// switchSetReq += "<config>"

		// For every update for a given switch, create partial XML req.
		for _, update := range s {
			xmlReq, err := getXmlReq(update)
			if err != nil {
				log.Errorf("Failed converting update to xml: %v", err)
				return &gnmi.SetResponse{}, err
			}

			switchSetReq += xmlReq
		}

		// switchSetReq += "</config></edit-config>"

		switchRequests = append(switchRequests, switchSetReq)
	}

	log.Infof("Switch requests: %v", switchRequests)

	response := sb.UpdateConfig(switchRequests[0])

	log.Infof("Response: %v", response)

	// if _, ok := switches[update.Path.Target]; !ok {
	// 	switches[update.Path.Target] = getXMLRequests([]*gnmi.Path{update.Path}, "", gnmi.GetRequest_CONFIG)
	// } else {

	// 	// updatedPaths := switches[update.Path.Target]
	// 	// updatedPaths = append(updatedPaths, getXMLRequests([]*gnmi.Path{update.Path}, "", gnmi.GetRequest_CONFIG))
	// }

	// xmlRequests := getXMLRequests(req.Path, datastore, req.Type)

	// reply, err := sb.GetConfig(xmlRequests, req.Path[0].Target)
	// if err != nil {
	// 	log.Errorf("Failed to get response from switch: %v\n", err)

	// 	notifications := make([]*gnmi.Notification, 1)
	// 	ts := time.Now().UnixNano()

	// 	notifications[0] = &gnmi.Notification{
	// 		Timestamp: ts,
	// 	}

	// 	return &gnmi.SetResponse{}, err
	// }

	// return convertXMLtoGnmiResponse(reply), nil
	return &gnmi.SetResponse{}, nil
}

func getXmlReq(update *gnmi.Update) (string, error) {
	var xmlReqStart string
	var xmlReqEnd string

	for _, elem := range update.Path.Elem {
		// log.Infof("elem: %v", elem)

		xmlReqStart += fmt.Sprintf("<%s", elem.Name)

		if namespace, ok := elem.Key["namespace"]; ok {
			xmlReqStart += fmt.Sprintf(" xmlns=\"%s\">", namespace)
		} else {
			xmlReqStart += ">"
		}

		if len(elem.Key) > 0 {
			for key, value := range elem.Key {
				if key != "namespace" {
					xmlReqStart += fmt.Sprintf("<%s>%s</%s>", key, value, key)
				}
			}
		}

		xmlReqEnd = fmt.Sprintf("</%s>", elem.Name) + xmlReqEnd
	}

	// TODO: Get any kind of value, not just string values.
	return xmlReqStart + update.Val.GetStringVal() + xmlReqEnd, nil
}

// // Takes in a gnmi get request and returns a gnmi get response.
// func ConvertSetReqtoXML(req *gnmi.SetRequest) { //*gnmi.GetRequest, typeOfRequest string) {

// 	/************************************************************
// 	Implementation of data conversion should be implemented here.
// 	*************************************************************/
// 	log.Infof(req.String())
// 	global_counter := -1
// 	var xmlPath string
// 	for _, upd := range req.GetUpdate() {
// 		for i, e := range upd.GetPath().Elem {
// 			fmt.Println(i, e.GetName())
// 			fmt.Println(i, e.GetKey())
// 		}
// 		calculateXmlPath(&xmlPath, &global_counter, upd, upd.GetPath().Elem)
// 	}
// 	fmt.Println(xmlPath)

// 	// Initiate southbound NETCONF client, sending the xml
// 	reply := sb.UpdateConfig(xmlPath).Data

// 	// Logs the reply, before sending back the response a conversion from xml to json should be implemented.
// 	log.Infof(reply)

// 	// Simulated response.
// 	//notifications := make([]*gnmi.Notification, 1)
// 	//prefix := req.GetPrefix()
// 	//ts := time.Now().UnixNano()

// 	//notifications[0] = &gnmi.Notification{
// 	//	Timestamp: ts,
// 	//	Prefix:    prefix,
// 	//}

// 	//resp := &gnmi.GetResponse{Notification: notifications}
// 	//return resp
// 	//return reply
// }

// func GetValue(upd *gnmi.Update) string {

// 	fmt.Println(upd.GetVal().String())
// 	bool_val := upd.GetVal().GetBoolVal()
// 	fmt.Println(bool_val)
// 	// log.Infof(string(upd.GetVal().GetJsonIetfVal()))
// 	// log.Infof(upd.GetVal().GetStringVal())
// 	// var editValue interface{}
// 	// editValue = make(map[string]interface{})
// 	// err := json.Unmarshal(upd.GetVal().GetJsonVal(), &editValue)
// 	// if err != nil {
// 	// 	status.Errorf(codes.Unknown, "invalid value %s", err)
// 	// }

// 	//return upd.GetVal().String()
// 	return strconv.FormatBool(bool_val)
// }

// func addMapValues(count int, path *string, elem []*gnmi.PathElem) {

// 	for key, value := range elem[count].GetKey() {
// 		*path += `<` + key + `>` + value + `</` + key + `>`
// 	}
// }

// func addNamespace(count int, path *string, elem []*gnmi.PathElem) {

// 	switch elem[count].GetName() {
// 	case "interfaces":
// 		*path += ` xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"`

// 	case "max-sdu-table":
// 		*path += ` xmlns="urn:ieee:std:802.1Q:yang:ieee802-dot1q-sched"`

// 	default:
// 		return
// 	}
// }

// func calculateXmlPath(path *string, global_counter *int, upd *gnmi.Update, elem []*gnmi.PathElem) {

// 	*global_counter++
// 	if *global_counter >= len(elem) {
// 		return
// 	}

// 	local_counter := *global_counter
// 	*path += `<` + elem[local_counter].GetName()
// 	addNamespace(local_counter, path, elem)
// 	*path += `>`
// 	if len(elem[local_counter].GetKey()) > 0 {
// 		addMapValues(local_counter, path, elem)
// 	}
// 	if *global_counter == len(elem)-1 {
// 		*path += GetValue(upd)
// 	}
// 	calculateXmlPath(path, global_counter, upd, elem)
// 	*path += `</` + elem[local_counter].GetName() + `>`

// }

// // func gnmiFullPath(prefix, path *gnmi.Path) *gnmi.Path {
// // 	fullPath := &gnmi.Path{Origin: path.Origin}
// // 	if path.GetElem() != nil {
// // 		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
// // 	}
// // 	return fullPath
// // }
