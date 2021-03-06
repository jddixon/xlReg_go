package reg

// xlreg_error_str.go

var XLREG_ERRORS = []string{
	"badly formed attrs line",
	"badly formated VERSION",
	"cannot find cluster with this ID",
	"cannot find cluster with this name",
	"client must have at least one endPoint",
	"cluster members must have at least one endPoint",
	"cluster must have a member!",
	"cluster must have at least two members",
	"ID already in use",
	"ill-formed cluster serialization",
	"ill-formed cluster member serialization",
	"ill-formed regCred serialization",
	"missing closing brace",
	"missing cluster name or ID",
	"missing endPoints section",
	"missing members list",
	"missing private key line",
	"missing regNode line",
	"missing server info",
	"name already in use",
	"nil cluster argument",
	"nil node argument",
	"nil private key argument",
	"nil registry argument",
	"nil RegNode argument",
	"nil XLRegMsg_Token argument",
	"no node and no keys to build one",
	"invalid msg type for current state",
	"message tag out of range",
	"unexpected message type",
	"client unknown, not in registry",
	"wrong number of bytes in attrs",
}
