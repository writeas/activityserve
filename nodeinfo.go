package activityserve

import (
	"strings"

	"github.com/writefreely/go-nodeinfo"
)

type nodeInfoResolver struct {
	actors int
}

func nodeInfoConfig(baseURL string) *nodeinfo.Config {
	name := "Pherephone"
	desc := "An ActivityPub repeater."
	return &nodeinfo.Config{
		BaseURL: baseURL,
		InfoURL: "/api/nodeinfo",

		Metadata: nodeinfo.Metadata{
			NodeName:        name,
			NodeDescription: desc,
			Software: nodeinfo.SoftwareMeta{
				HomePage: "https://pherephone.org",
				GitHub:   "https://github.com/writeas/pherephone",
			},
		},
		Protocols: []nodeinfo.NodeProtocol{
			nodeinfo.ProtocolActivityPub,
		},
		Services: nodeinfo.Services{
			Inbound:  []nodeinfo.NodeService{},
			Outbound: []nodeinfo.NodeService{},
		},
		Software: nodeinfo.SoftwareInfo{
			Name:    strings.ToLower(libName),
			Version: version,
		},
	}
}

func (r nodeInfoResolver) IsOpenRegistration() (bool, error) {
	return false, nil
}

func (r nodeInfoResolver) Usage() (nodeinfo.Usage, error) {
	return nodeinfo.Usage{
		Users: nodeinfo.UsageUsers{
			Total: r.actors,
		},
	}, nil
}
