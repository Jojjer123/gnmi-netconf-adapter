package types

type AdapterResponse struct {
	Entries   []SchemaEntry
	Timestamp int64
}

type SchemaEntry struct {
	Name      string
	Tag       string
	Namespace string
	Value     string
}

type NamespaceParser struct {
	Parent              *NamespaceParser
	Children            []*NamespaceParser
	LastParentNamespace string
}
