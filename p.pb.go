// Code generated by protoc-gen-go.
// source: p.proto
// DO NOT EDIT!

package reg

import proto "code.google.com/p/goprotobuf/proto"
import json "encoding/json"
import math "math"

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type XLRegMsg_Tag int32

const (
	XLRegMsg_Hello       XLRegMsg_Tag = 1
	XLRegMsg_HelloReply  XLRegMsg_Tag = 2
	XLRegMsg_User        XLRegMsg_Tag = 3
	XLRegMsg_UserOK      XLRegMsg_Tag = 4
	XLRegMsg_Create      XLRegMsg_Tag = 5
	XLRegMsg_CreateReply XLRegMsg_Tag = 6
	XLRegMsg_Join        XLRegMsg_Tag = 7
	XLRegMsg_JoinReply   XLRegMsg_Tag = 8
	XLRegMsg_Get         XLRegMsg_Tag = 9
	XLRegMsg_Members     XLRegMsg_Tag = 10
	XLRegMsg_Bye         XLRegMsg_Tag = 13
	XLRegMsg_Ack         XLRegMsg_Tag = 14
	XLRegMsg_Error       XLRegMsg_Tag = 15
)

var XLRegMsg_Tag_name = map[int32]string{
	1:  "Hello",
	2:  "HelloReply",
	3:  "User",
	4:  "UserOK",
	5:  "Create",
	6:  "CreateReply",
	7:  "Join",
	8:  "JoinReply",
	9:  "Get",
	10: "Members",
	13: "Bye",
	14: "Ack",
	15: "Error",
}
var XLRegMsg_Tag_value = map[string]int32{
	"Hello":       1,
	"HelloReply":  2,
	"User":        3,
	"UserOK":      4,
	"Create":      5,
	"CreateReply": 6,
	"Join":        7,
	"JoinReply":   8,
	"Get":         9,
	"Members":     10,
	"Bye":         13,
	"Ack":         14,
	"Error":       15,
}

func (x XLRegMsg_Tag) Enum() *XLRegMsg_Tag {
	p := new(XLRegMsg_Tag)
	*p = x
	return p
}
func (x XLRegMsg_Tag) String() string {
	return proto.EnumName(XLRegMsg_Tag_name, int32(x))
}
func (x XLRegMsg_Tag) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *XLRegMsg_Tag) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(XLRegMsg_Tag_value, data, "XLRegMsg_Tag")
	if err != nil {
		return err
	}
	*x = XLRegMsg_Tag(value)
	return nil
}

type XLRegMsg struct {
	Op               *XLRegMsg_Tag     `protobuf:"varint,1,opt,enum=reg.XLRegMsg_Tag" json:"Op,omitempty"`
	AesIV            []byte            `protobuf:"bytes,2,opt" json:"AesIV,omitempty"`
	AesKey           []byte            `protobuf:"bytes,3,opt" json:"AesKey,omitempty"`
	Salt1            []byte            `protobuf:"bytes,4,opt" json:"Salt1,omitempty"`
	Salt2            []byte            `protobuf:"bytes,5,opt" json:"Salt2,omitempty"`
	Version          *uint32           `protobuf:"varint,6,opt" json:"Version,omitempty"`
	Size             *uint32           `protobuf:"varint,7,opt" json:"Size,omitempty"`
	MySpecs          *XLRegMsg_Token   `protobuf:"bytes,8,opt" json:"MySpecs,omitempty"`
	ClusterID        []byte            `protobuf:"bytes,9,opt" json:"ClusterID,omitempty"`
	ClusterName      *string           `protobuf:"bytes,10,opt" json:"ClusterName,omitempty"`
	Which            *uint64           `protobuf:"varint,11,opt" json:"Which,omitempty"`
	Tokens           []*XLRegMsg_Token `protobuf:"bytes,12,rep" json:"Tokens,omitempty"`
	ErrDesc          *string           `protobuf:"bytes,15,opt" json:"ErrDesc,omitempty"`
	XXX_unrecognized []byte            `json:"-"`
}

func (m *XLRegMsg) Reset()         { *m = XLRegMsg{} }
func (m *XLRegMsg) String() string { return proto.CompactTextString(m) }
func (*XLRegMsg) ProtoMessage()    {}

func (m *XLRegMsg) GetOp() XLRegMsg_Tag {
	if m != nil && m.Op != nil {
		return *m.Op
	}
	return 0
}

func (m *XLRegMsg) GetAesIV() []byte {
	if m != nil {
		return m.AesIV
	}
	return nil
}

func (m *XLRegMsg) GetAesKey() []byte {
	if m != nil {
		return m.AesKey
	}
	return nil
}

func (m *XLRegMsg) GetSalt1() []byte {
	if m != nil {
		return m.Salt1
	}
	return nil
}

func (m *XLRegMsg) GetSalt2() []byte {
	if m != nil {
		return m.Salt2
	}
	return nil
}

func (m *XLRegMsg) GetVersion() uint32 {
	if m != nil && m.Version != nil {
		return *m.Version
	}
	return 0
}

func (m *XLRegMsg) GetSize() uint32 {
	if m != nil && m.Size != nil {
		return *m.Size
	}
	return 0
}

func (m *XLRegMsg) GetMySpecs() *XLRegMsg_Token {
	if m != nil {
		return m.MySpecs
	}
	return nil
}

func (m *XLRegMsg) GetClusterID() []byte {
	if m != nil {
		return m.ClusterID
	}
	return nil
}

func (m *XLRegMsg) GetClusterName() string {
	if m != nil && m.ClusterName != nil {
		return *m.ClusterName
	}
	return ""
}

func (m *XLRegMsg) GetWhich() uint64 {
	if m != nil && m.Which != nil {
		return *m.Which
	}
	return 0
}

func (m *XLRegMsg) GetTokens() []*XLRegMsg_Token {
	if m != nil {
		return m.Tokens
	}
	return nil
}

func (m *XLRegMsg) GetErrDesc() string {
	if m != nil && m.ErrDesc != nil {
		return *m.ErrDesc
	}
	return ""
}

type XLRegMsg_Token struct {
	Attrs            *uint64  `protobuf:"varint,1,opt" json:"Attrs,omitempty"`
	ID               []byte   `protobuf:"bytes,2,opt" json:"ID,omitempty"`
	CommsKey         []byte   `protobuf:"bytes,3,opt" json:"CommsKey,omitempty"`
	SigKey           []byte   `protobuf:"bytes,4,opt" json:"SigKey,omitempty"`
	MyEnd            []string `protobuf:"bytes,5,rep" json:"MyEnd,omitempty"`
	DigSig           []byte   `protobuf:"bytes,6,opt" json:"DigSig,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *XLRegMsg_Token) Reset()         { *m = XLRegMsg_Token{} }
func (m *XLRegMsg_Token) String() string { return proto.CompactTextString(m) }
func (*XLRegMsg_Token) ProtoMessage()    {}

func (m *XLRegMsg_Token) GetAttrs() uint64 {
	if m != nil && m.Attrs != nil {
		return *m.Attrs
	}
	return 0
}

func (m *XLRegMsg_Token) GetID() []byte {
	if m != nil {
		return m.ID
	}
	return nil
}

func (m *XLRegMsg_Token) GetCommsKey() []byte {
	if m != nil {
		return m.CommsKey
	}
	return nil
}

func (m *XLRegMsg_Token) GetSigKey() []byte {
	if m != nil {
		return m.SigKey
	}
	return nil
}

func (m *XLRegMsg_Token) GetMyEnd() []string {
	if m != nil {
		return m.MyEnd
	}
	return nil
}

func (m *XLRegMsg_Token) GetDigSig() []byte {
	if m != nil {
		return m.DigSig
	}
	return nil
}

func init() {
	proto.RegisterEnum("reg.XLRegMsg_Tag", XLRegMsg_Tag_name, XLRegMsg_Tag_value)
}
