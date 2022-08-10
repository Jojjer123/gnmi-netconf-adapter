package dataConversion

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Juniper/go-netconf/netconf"
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
		log.Infof("Update: %v", update)
		// log.Infof("Update.Path.Target: %s", update.Path.Target)
		switches[update.Path.Target] = append(switches[update.Path.Target], update)
	}

	var switchRequests = make(map[string]string)

	// For each switch, create a switchRequest entry using switch address and switch request
	for switchAddr, switchUpdates := range switches {
		// log.Infof("Switch address: %v", switchAddr)
		// Get switch request in XML using all the updates for a switch
		switchUpdateRequest, err := getSwitchRequest(switchUpdates)
		if err != nil {
			log.Errorf("Failed getting module updates: %v", err)
			return &gnmi.SetResponse{}, err
		}

		// log.Infof("Request: %v", switchUpdateRequest)
		switchRequests[switchAddr] = switchUpdateRequest
	}

	var responses = make(map[string]*netconf.RPCReply)
	var wg sync.WaitGroup

	// Update config for every switch
	for addr, req := range switchRequests {
		// log.Infof("Addr: %s", addr)
		// TODO: Make multithreaded
		wg.Add(1)
		go sendUpdate(req, addr, responses, &wg)
		// responses = append(responses, sb.UpdateConfig(req, addr))
	}

	wg.Wait()

	gnmiResponse := &gnmi.SetResponse{
		Response: []*gnmi.UpdateResult{},
	}

	// TODO: Check all responses if they were successful or not
	for switchAddr, response := range responses {
		if response == nil {
			continue
		}
		// TODO: Check if response is ok, then get paths to all updates for that switch, add them
		// into the gnmiResponse as separate updates

		// Convert XML response to gNMI
		// If response.Data contains "<ok/>" or rather "ok" then it was successful, otherwise error occurred
		if strings.Contains(response.Data, "ok") {
			log.Infof("Set request for switch %s was successful", switchAddr)

			// TODO: For every path updated in the switch, get it, and build gnmi a new gnmi.UpdateResult
			for _, update := range switches[switchAddr] {
				gnmiResponse.Response = append(gnmiResponse.Response, &gnmi.UpdateResult{
					Path: update.Path,
					Op:   gnmi.UpdateResult_UPDATE,
				})
			}
		} else {
			log.Errorf("Set request failed in switch %s with error(s): %v", switchAddr, response.Errors)
			for _, update := range switches[switchAddr] {
				gnmiResponse.Response = append(gnmiResponse.Response, &gnmi.UpdateResult{
					Path: update.Path,
					Op:   gnmi.UpdateResult_INVALID,
				})
			}
		}
	}

	// log.Infof("gnmiResponse: %v", gnmiResponse)
	log.Info("gnmiResponse arrived now")

	// TODO: Send back one response for all set requests

	return gnmiResponse, nil
}

func sendUpdate(req string, addr string, responses map[string]*netconf.RPCReply, wg *sync.WaitGroup) {
	defer wg.Done()

	// log.Infof("Sending update to switch: %s", addr)

	log.Infof("Request sent: %v", req)

	response := sb.UpdateConfig(req, addr)

	responses[addr] = response
}

func getSwitchRequest(switcheUpdates []*gnmi.Update) (string, error) {
	// var switchRequests string

	// For every switch with at least one update, convert to XML.
	// for _, updateList := range switches {
	var addModuleStartTag bool
	var addModuleEndTag bool
	var switchSetReq string

	// Map for each module, where the updates for the module is stored as xml in the value-string
	var moduleUpdates = make(map[string]string)

	for _, update := range switcheUpdates {
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
			return "", err
		}

		moduleUpdates[update.Path.Elem[0].Name] = existingVal + xmlReq
	}

	addModuleStartTag = false
	addModuleEndTag = true

	for module, update := range moduleUpdates {
		switchSetReq += update + fmt.Sprintf("</%s>", module)
	}

	// switchRequests = append(switchRequests, switchSetReq)
	// }

	return switchSetReq, nil
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
