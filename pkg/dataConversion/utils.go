package dataConversion

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Builds XML request from list of updates provided.
func buildXml(updates []Update) string {
	var firstElems = map[string][]Update{}

	for _, update := range updates {
		// Group by the first element(s) and their keys found (not grouping by
		// namespaces, might have to be added later on). If namespaces should be included in grouping, then
		// the if-statement should be removed, with the "keys += fmt.Sprintf(...)" running for all keys.
		if len(update.Update.Path.Elem) > 0 {
			keys := ""
			for key, val := range update.Update.Path.Elem[0].GetKey() {
				if key != "namespace" {
					keys += fmt.Sprintf("%s.%s.", key, val)
				}
			}
			firstElemKey := update.Update.Path.Elem[0].Name + "." + keys
			firstElems[firstElemKey] = append(firstElems[firstElemKey], update)
		}
	}

	var branches = []string{}

	for _, updateGroup := range firstElems {
		startTag, endTag := getTags(updateGroup[0].Update.Path.Elem[0])
		// If an update group only contains one update, check if the update only contains one element
		if len(updateGroup) == 1 {
			// If the update only contains one element, it is a leaf, so try to add value
			if len(updateGroup[0].Update.Path.Elem) == 1 {
				val, err := getValue(updateGroup[0].Update)
				if err != nil {
					log.Errorf("Failed getting value from update: %v", err)
				} else {
					branches = append(branches, fmt.Sprintf("<%s>%s</%s>", startTag, val, endTag))
				}
			}
		} else {
			// Get a list of updates for all the new branches found
			newUpdates := removeFirstElement(updateGroup)
			// Recursively call this function on the new branches
			branches = append(branches, fmt.Sprintf("<%s>%s</%s>", startTag, buildXml(newUpdates), endTag))
		}
	}

	var xml string

	// Add all branches to the tree
	for _, branch := range branches {
		xml += branch
	}

	return xml
}

// Extracts the start- and end-tags, including keys for the start-tag
func getTags(elem *pb.PathElem) (string, string) {
	var startTag = fmt.Sprintf("%s", elem.Name)
	var endTag = startTag
	// If any keys are present, add them
	for key, value := range elem.Key {
		if key == "namespace" {
			startTag += fmt.Sprintf(" xmlns=\"%s\"", value)
		} else {
			startTag += fmt.Sprintf("><%s>%s</%s", key, value, key)
		}
	}
	return startTag, endTag
}

// Removes first element from all updates in the updateGroup
func removeFirstElement(updateGroup []Update) []Update {
	var newUpdates []Update

	for _, update := range updateGroup {
		// Create new element list without the first element of the old list
		newElemList := update.Update.Path.Elem[1:]
		// Create new update with the new element list
		newUpdates = append(newUpdates, Update{Update: &pb.Update{
			Path: &pb.Path{
				Elem: newElemList,
			},
			Val: update.Update.Val,
		}})
	}

	return newUpdates
}

// Takes in a gnmi.Update and converts the value to a string
func getValue(update *gnmi.Update) (string, error) {
	var value string

	// TODO: Get any kind of value, not just decimal values.
	switch update.Val.Value.(type) {
	case *gnmi.TypedValue_AnyVal:
		value = update.GetVal().GetAnyVal().String()
	case *gnmi.TypedValue_AsciiVal:
		value = update.GetVal().GetAsciiVal()
	case *gnmi.TypedValue_BoolVal:
		if update.GetVal().GetBoolVal() {
			value = "true"
		} else {
			value = "false"
		}
	case *gnmi.TypedValue_BytesVal:
		value = string(update.GetVal().GetBytesVal())
	case *gnmi.TypedValue_FloatVal:
		value = fmt.Sprintf("%f", update.GetVal().GetFloatVal())
	case *gnmi.TypedValue_DecimalVal:
		value = strconv.FormatInt(update.GetVal().GetDecimalVal().GetDigits(), 10)
	case *gnmi.TypedValue_IntVal:
		value = strconv.Itoa(int(update.GetVal().GetIntVal()))
	case *gnmi.TypedValue_JsonIetfVal:
		value = string(update.GetVal().GetJsonIetfVal())
	case *gnmi.TypedValue_JsonVal:
		value = string(update.GetVal().GetJsonVal())
	case *gnmi.TypedValue_LeaflistVal:
		value = update.GetVal().GetLeaflistVal().String()
	case *gnmi.TypedValue_ProtoBytes:
		value = string(update.GetVal().GetProtoBytes())
	case *gnmi.TypedValue_StringVal:
		value = update.GetVal().GetStringVal()
	case *gnmi.TypedValue_UintVal:
		value = strconv.FormatUint(update.GetVal().GetUintVal(), 10)
	default:
		log.Errorf("Value \"%v\" is not defined", update.GetValue())
		return "", errors.New("Value not defined")
	}

	return value, nil
}
