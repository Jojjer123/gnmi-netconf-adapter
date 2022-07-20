package dataConversion

import (
	"fmt"
	"strings"

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
	for _, updateList := range switches {
		var switchSetReq string

		// TODO: Make target dynamic, currently only changes the running configuration.
		// Create default start structure of XML.
		// switchSetReq = "<edit-config>"
		// switchSetReq += fmt.Sprintf("<target><%s/></target>", "running")
		// switchSetReq += fmt.Sprintf("<test-option>%s</test-option>", "test-then-set")
		// switchSetReq += fmt.Sprintf("<error-option>%s</error-option>", "rollback-on-error")
		// switchSetReq += "<config>"

		// For every update for a given switch, create partial XML req.
		for index, update := range updateList {
			var addTopLevelStartTag bool
			var addTopLevelEndTag bool

			// For first index, always add top level start tag, otherwise set only when new one occurs
			if index == 0 {
				addTopLevelStartTag = true
				log.Infof("START TAG - Name: %s", updateList[index].Path.Elem[0].Name)
			} else {
				if update.Path.Elem[0].Name == updateList[index-1].Path.Elem[0].Name {
					log.Infof("START TAG - Names: %s == %s", update.Path.Elem[0].Name, updateList[index-1].Path.Elem[0].Name)
					addTopLevelStartTag = false
				} else {
					addTopLevelStartTag = true
				}
			}

			// If there are more updates
			if index < len(updateList)-1 {
				// If the current and next paths have the same top level element
				if update.Path.Elem[0].Name == updateList[index+1].Path.Elem[0].Name {
					log.Infof("END TAG - Names: %s == %s", update.Path.Elem[0].Name, updateList[index+1].Path.Elem[0].Name)
					addTopLevelEndTag = false
				} else {
					addTopLevelEndTag = true
				}
			} else if index == len(updateList)-1 {
				addTopLevelEndTag = true
			}

			// TODO: If next update request (if there any) uses the same "top-level" element, skip the end-tag, else use end-tag
			xmlReq, err := getXmlReq(update, addTopLevelStartTag, addTopLevelEndTag)
			if err != nil {
				log.Errorf("Failed converting update to xml: %v", err)
				return &gnmi.SetResponse{}, err
			}

			switchSetReq += xmlReq
		}

		// switchSetReq += "</config></edit-config>"

		switchRequests = append(switchRequests, switchSetReq)
	}

	log.Infof("Requests: %v", switchRequests)

	response := sb.UpdateConfig(switchRequests[0])

	gnmiResponse := &gnmi.SetResponse{
		Response: []*gnmi.UpdateResult{
			{
				Path: &gnmi.Path{},
			},
		},
	}

	// TODO: Convert XML response to gNMI
	// If response.Data contains "<ok/>" or rather "ok" then it was successful, otherwise error occurred
	if strings.Contains(response.Data, "ok") {
		log.Info("Set request was successful")
		gnmiResponse.Response[0].Op = gnmi.UpdateResult_UPDATE
	} else {
		log.Errorf("Set request failed in switch with error(s): %v", response.Errors)
		gnmiResponse.Response[0].Op = gnmi.UpdateResult_INVALID
	}

	// log.Infof("Response: %v", response)

	// gnmiResp := netconfConv(response.Data)
	// log.Infof("adapter-response: %v", gnmiResp.Entries)

	return gnmiResponse, nil
}

func getXmlReq(update *gnmi.Update, addTopLevelStartTag bool, addTopLevelEndTag bool) (string, error) {
	var xmlReqStart string
	var xmlReqEnd string

	for index, elem := range update.Path.Elem {
		if index == 0 && addTopLevelStartTag {
			log.Infof("Adding top level start tag: %s", elem.Name)
			xmlReqStart += fmt.Sprintf("<%s", elem.Name)

			if namespace, ok := elem.Key["namespace"]; ok {
				xmlReqStart += fmt.Sprintf(" xmlns=\"%s\">", namespace)
			} else {
				xmlReqStart += ">"
			}
		}

		if index == 0 && addTopLevelEndTag {
			log.Infof("Adding top level end tag: %s", elem.Name)
			xmlReqEnd = fmt.Sprintf("</%s>", elem.Name) + xmlReqEnd
		}

		// If not top level element, add element and namespace if it has one specified.
		if index != 0 {
			xmlReqStart += fmt.Sprintf("<%s", elem.Name)

			if namespace, ok := elem.Key["namespace"]; ok {
				xmlReqStart += fmt.Sprintf(" xmlns=\"%s\">", namespace)
			} else {
				xmlReqStart += ">"
			}

			xmlReqEnd = fmt.Sprintf("</%s>", elem.Name) + xmlReqEnd
		}

		if len(elem.Key) > 0 {
			for key, value := range elem.Key {
				if key != "namespace" {
					xmlReqStart += fmt.Sprintf("<%s>%s</%s>", key, value, key)
				}
			}
		}

	}

	val, err := getValue(update)
	if err != nil {
		log.Errorf("Failed getting value for path: %v", update.GetPath())
		return "", err
	}

	return xmlReqStart + val + xmlReqEnd, nil
	// return xmlReqStart + fmt.Sprintf("%d", update.Val.GetDecimalVal().GetDigits()) + xmlReqEnd, nil
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
