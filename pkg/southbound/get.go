package southbound

import (
	"time"

	"github.com/Juniper/go-netconf/netconf"
	"golang.org/x/crypto/ssh"
)

// Requests the full configuration for the target "running"
func GetFullConfig(switchAddr string) *netconf.RPCReply {
	reply := sendRPCRequest(netconf.MethodGetConfig("running"), switchAddr)

	return reply
}

func GetConfig(req []string, target string) (string, error) {
	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	//  Start connection to network device
	s, err := netconf.DialSSH(target, sshConfig)

	if err != nil {
		log.Warnf("Response from switch was: %v", err)

		if s != nil {
			defer s.Close()
		}

		return "", err
	}

	// Close connetion to network device when this function is done executing
	defer s.Close()

	var requests []netconf.RPCMethod
	for _, r := range req {
		requests = append(requests, netconf.RawMethod(r))
		log.Infof("r = %v", r)
	}

	startTimeGetConf := time.Now().UnixNano()
	reply, err := s.Exec(requests...)
	log.Infof("Time to get response without session creation is: %v\n", time.Now().UnixNano()-startTimeGetConf)
	if err != nil {
		return "", err
	}

	return reply.Data, nil
}
