package types

import "github.com/gogo/protobuf/proto"

type AdapterResponse struct {
	Entries   []SchemaEntry `protobuf:"bytes,1,opt,name=Entries"`
	Timestamp int64         `protobuf:"fixed64,2,opt,name=Timestamp"`
}

func (m *AdapterResponse) Reset()         { *m = AdapterResponse{} }
func (m *AdapterResponse) String() string { return proto.CompactTextString(m) }
func (m *AdapterResponse) ProtoMessage()  {}

type SchemaEntry struct {
	Name      string `protobuf:"bytes,1,req,name=Name"`
	Tag       string `protobuf:"bytes,2,opt,name=Tag"`
	Namespace string `protobuf:"bytes,3,opt,name=Namespace"`
	Value     string `protobuf:"bytes,4,opt,name=Value"`
}

type NamespaceParser struct {
	Parent              *NamespaceParser
	Children            []*NamespaceParser
	LastParentNamespace string
}
