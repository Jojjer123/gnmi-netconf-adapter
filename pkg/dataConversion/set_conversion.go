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

	// Get switch requests from list of switches with updates
	switchRequests, err := getSwitchRequests(switches)
	if err != nil {
		log.Errorf("Failed getting module updates: %v", err)
		return &gnmi.SetResponse{}, err
	}

	log.Infof("Requests: %v", switchRequests)

	// TODO: Update config for every switch, and use target as switch address

	response := sb.UpdateConfig(switchRequests[0])

	gnmiResponse := &gnmi.SetResponse{
		Response: []*gnmi.UpdateResult{
			{
				Path: &gnmi.Path{},
			},
		},
	}

	// Convert XML response to gNMI
	// If response.Data contains "<ok/>" or rather "ok" then it was successful, otherwise error occurred
	if strings.Contains(response.Data, "ok") {
		log.Info("Set request was successful")
		gnmiResponse.Response[0].Op = gnmi.UpdateResult_UPDATE
	} else {
		log.Errorf("Set request failed in switch with error(s): %v", response.Errors)
		gnmiResponse.Response[0].Op = gnmi.UpdateResult_INVALID
	}

	return gnmiResponse, nil
}

func getSwitchRequests(switches map[string][]*gnmi.Update) ([]string, error) {
	var switchRequests []string

	// For every switch with at least one update, convert to XML.
	for _, updateList := range switches {
		var addModuleStartTag bool
		var addModuleEndTag bool
		var switchSetReq string

		// Map for each module, where the updates for the module is stored as xml in the value-string
		var moduleUpdates = make(map[string]string)

		for _, update := range updateList {
			addModuleStartTag = false
			addModuleEndTag = false
			var ok bool

			var existingVal string

			if existingVal, ok = moduleUpdates[update.Path.Elem[0].Name]; !ok {
				// Add with start tag
				addModuleStartTag = true
			} else {
				// Add without start tag
				addModuleStartTag = false
			}

			xmlReq, err := getXmlReq(update, addModuleStartTag, addModuleEndTag)
			if err != nil {
				log.Errorf("Failed converting update to xml: %v", err)
				return []string{}, err
			}

			moduleUpdates[update.Path.Elem[0].Name] = existingVal + xmlReq
		}

		addModuleStartTag = false
		addModuleEndTag = true

		for module, update := range moduleUpdates {
			switchSetReq += update + fmt.Sprintf("</%s>", module)
		}

		switchRequests = append(switchRequests, switchSetReq)
	}

	return switchRequests, nil
}

func getXmlReq(update *gnmi.Update, addTopLevelStartTag bool, addTopLevelEndTag bool) (string, error) {
	var xmlReqStart string
	var xmlReqEnd string

	for index, elem := range update.Path.Elem {
		if index == 0 && addTopLevelStartTag {
			// log.Infof("Adding top level start tag: %s", elem.Name)
			xmlReqStart += fmt.Sprintf("<%s", elem.Name)

			if namespace, ok := elem.Key["namespace"]; ok {
				xmlReqStart += fmt.Sprintf(" xmlns=\"%s\">", namespace)
			} else {
				xmlReqStart += ">"
			}
		}

		if index == 0 && addTopLevelEndTag {
			// log.Infof("Adding top level end tag: %s", elem.Name)
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
}
