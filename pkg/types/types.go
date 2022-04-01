package types

type Schema struct {
	Entries []SchemaEntry
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
