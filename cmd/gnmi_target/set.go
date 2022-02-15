// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/google/gnxi/utils/credentials"
	//dataConv "github.com/onosproject/gnmi-netconf-adapter/pkg/dataConversion"
	"fmt"

	"github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Set overrides the Set func of gnmi.Target to provide user auth.
func (s *server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	//checking pull behavior
	msg, ok := credentials.AuthorizeUser(ctx)
	if !ok {
		log.Infof("denied a Set request: %v", msg)
		return nil, status.Error(codes.PermissionDenied, msg)
	}
	log.Infof("allowed a Set request: %v", msg)
	//log.Infof("Allowed set req..")

	fmt.Println("ext number = ", len(req.GetExtension()))
	for i, e := range req.GetExtension() {
		fmt.Println(i, e.String())
	}

	prefix := req.GetPrefix()

	log.Infof(req.String())
	for _, upd := range req.GetUpdate() {
		for i, e := range upd.GetPath().Elem {
			fmt.Println(i, e.GetName())
			fmt.Println(i, e.GetKey())
		}

		path := upd.GetPath()
		fullPath := path
		if prefix != nil {
			fmt.Println("prefix exists")
			fullPath = gnmiFullPath(prefix, path)
		}
		fmt.Println(fullPath)

		//log.Infof(string(upd.GetVal().GetJsonIetfVal()))
		//log.Infof(string(upd.GetVal().GetJsonIetfVal()))
		//log.Infof(upd.getva)
	}

	//dataConv.Convert(req)
	// log.Infof(req.String())

	setResponse, err := s.Server.Set(ctx, req)
	return setResponse, err
	//	return nil, nil
}

func gnmiFullPath(prefix, path *gnmi.Path) *gnmi.Path {
	fullPath := &gnmi.Path{Origin: path.Origin}
	if path.GetElem() != nil {
		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
	}
	return fullPath
}
