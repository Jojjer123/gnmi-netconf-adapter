package southbound

import (
	"fmt"
	"time"

	"github.com/Juniper/go-netconf/netconf"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"golang.org/x/crypto/ssh"
)

// const switchAddr = "192.168.0.1"

const editConfigXml = `<edit-config>
	<target><%s/></target>
	<default-operation>merge</default-operation>
	<error-option>rollback-on-error</error-option>
	<config>%s</config>
	</edit-config>`

var log = logging.GetLogger("main")

// Takes in an RPCMethod function and executes it, then returns the reply from the network device
func sendRPCRequest(fn netconf.RPCMethod, switchAddr string) *netconf.RPCReply {
	//  Define config for connection to network device
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 5,
	}

	// Start connection to network device
	s, err := netconf.DialSSH(switchAddr, sshConfig)

	var reply *netconf.RPCReply
	if err != nil {
		log.Warn(err)
		return nil
	} else {
		// Executes the function passed as fn
		reply, err = s.Exec(fn)

		if err != nil {
			log.Warn(err)
		}

		// Close connetion to network device when this function is done executing
		defer s.Close()
	}

	return reply
}

// Method is necessary for version v.0.1.1 of https://pkg.go.dev/github.com/juniper/go-netconf/netconf as the code implementing it is not in release
// MethodEditConfig sends a NETCONF edit-config request to the network device
func methodEditConfig(database string, dataXml string) netconf.RawMethod {
	// log.Infof("methodEditconfig/sb/utils.go")
	return netconf.RawMethod(fmt.Sprintf(editConfigXml, database, dataXml))
}
