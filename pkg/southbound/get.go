package southbound

import (
	"fmt"

	"github.com/Juniper/go-netconf/netconf"
	"github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/crypto/ssh"
)

// Requests the full configuration for the target "running"
func GetFullConfig() *netconf.RPCReply {
	reply := sendRPCRequest(netconf.MethodGetConfig("running"))

	return reply
}

// // Requests partial configuration according to the xmlRequest for the target "running"
// func GetConfig(xmlRequest string) *netconf.RPCReply {
// 	reply := sendRPCRequest(netconf.Meth("<running /></source><filter><interfaces xmlns=\"urn:ietf:params:xml:ns:yang:ietf-interfaces\"><interface/></interfaces></filter>"))

// 	return reply
// }

// GetConfig returns the full configuration, or configuration starting at <section>.
// <format> can be one of "text" or "xml." You can do sub-sections by separating the
// <section> path with a ">" symbol, i.e. "system>login" or "protocols>ospf>area".
// func GetConfig(section, format string) (string, error) {
// 	secs := strings.Split(section, ">")
// 	nSecs := len(secs)
// 	command := fmt.Sprintf("<get-config><source><%s/>", format)
// 	if section == "full" {
// 		command += "</source></get-config>"
// 	}
// 	// if section == "interfaces" {
// 	// command += "</source><filter><interfaces xmlns=\"urn:ietf:params:xml:ns:yang:ietf-interfaces\"><interface/></interfaces></filter></get-config>"
// 	// }
// 	// fmt.Println("number of secs = " + strconv.Itoa(nSecs))
// 	if nSecs > 1 {
// 		// fmt.Println("in the loop")
// 		command += "</source><filter>"
// 		for i := 0; i < nSecs-1; i++ {
// 			command += fmt.Sprintf("<%s xmlns=\"urn:ietf:params:xml:ns:yang:ietf-interfaces\">", secs[i])
// 		}
// 		command += fmt.Sprintf("<%s/>", secs[nSecs-1])

// 		for j := nSecs - 2; j >= 0; j-- {
// 			command += fmt.Sprintf("</%s>", secs[j])
// 		}
// 		command += fmt.Sprint("</filter></get-config>")
// 	}

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "root",
// 		Auth:            []ssh.AuthMethod{ssh.Password("")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}
// 	switchAddr := "192.168.0.1"
// 	//  Start connection to network device
// 	s, err := netconf.DialSSH(switchAddr, sshConfig)

// 	if err != nil {
// 		log.Warn(err)
// 	}

// 	// Close connetion to network device when this function is done executing
// 	defer s.Close()

// 	r := netconf.RawMethod(command)
// 	// fmt.Println(r)
// 	reply, err := s.Exec(r)
// 	if err != nil {
// 		return "", err
// 	}

// 	return reply.Data, nil
// }
func GetConfig(paths []*gnmi.Path, format string, reqType gnmi.GetRequest_DataType) (string, error) {
	// secs := strings.Split(section, ">")
	// nSecs := len(secs)

	var cmd string
	var endOfCmd string
	appendXMLTagOnType(&cmd, format, reqType, true)

	for _, path := range paths {
		for index, elem := range path.Elem {
			if index == 0 {
				cmd += "<filter>"
			}
			cmd += fmt.Sprintf("<%s", elem.Name)
			endOfCmd = fmt.Sprintf("</%s>", elem.Name) + endOfCmd
			// for _, key := range elem.Key {
			// 	log.Info(key)
			// }
			if namespace, ok := elem.Key["namespace"]; ok {
				cmd += fmt.Sprintf(" xmlns=\"%s\">", namespace)
			} else if name, ok := elem.Key["name"]; ok {
				cmd += fmt.Sprintf("<name>%s</name>", name)
			} else {
				cmd += ">"
			}
		}
		cmd += endOfCmd
	}

	appendXMLTagOnType(&cmd, format, reqType, false)

	log.Info(cmd)

	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	switchAddr := "192.168.0.1"
	//  Start connection to network device
	s, err := netconf.DialSSH(switchAddr, sshConfig)

	if err != nil {
		log.Warn(err)
	}

	// Close connetion to network device when this function is done executing
	defer s.Close()

	r := netconf.RawMethod(cmd)
	// fmt.Println(r)
	reply, err := s.Exec(r)
	if err != nil {
		return "", err
	}

	return reply.Data, nil
	// return "", nil
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

	default:
		log.Warn("Did not recognize request type!")
	}
}
