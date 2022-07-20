package dataConversion

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/openconfig/gnmi/proto/gnmi"
)

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
