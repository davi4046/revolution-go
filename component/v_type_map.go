package component

var typeMap = map[string]string{
	"int":     "xs:integer",
	"int8":    "xs:byte",
	"int16":   "xs:short",
	"int32":   "xs:int",
	"int64":   "xs:long",
	"uint":    "xs:nonNegativeInteger",
	"uint8":   "xs:unsignedByte",
	"uint16":  "xs:unsignedShort",
	"uint32":  "xs:unsignedInt",
	"uint64":  "xs:unsignedLong",
	"float32": "xs:float",
	"float64": "xs:double",
	"bool":    "xs:boolean",
	"string":  "xs:string",
}
