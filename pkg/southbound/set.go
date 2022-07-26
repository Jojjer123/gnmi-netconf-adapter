package southbound

import "github.com/Juniper/go-netconf/netconf"

// Updates the configuration accoring to the input xml for the target "running"
func UpdateConfig(xmlChanges string, switchAddr string) *netconf.RPCReply {
	// log.Infof("Should have sent the following req to %s : %s", switchAddr, xmlChanges)

	reply := sendRPCRequest(methodEditConfig("running", xmlChanges), switchAddr)

	return reply

	// return nil
}
