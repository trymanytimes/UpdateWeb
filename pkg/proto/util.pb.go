// Code generated by protoc-gen-go. DO NOT EDIT.
// source: util.proto

package proto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type OperResult struct {
	RetCode              int32    `protobuf:"varint,1,opt,name=ret_code,json=retCode,proto3" json:"ret_code,omitempty"`
	RetMsg               string   `protobuf:"bytes,2,opt,name=ret_msg,json=retMsg,proto3" json:"ret_msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OperResult) Reset()         { *m = OperResult{} }
func (m *OperResult) String() string { return proto.CompactTextString(m) }
func (*OperResult) ProtoMessage()    {}
func (*OperResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_170ba741606d8a4c, []int{0}
}

func (m *OperResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OperResult.Unmarshal(m, b)
}
func (m *OperResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OperResult.Marshal(b, m, deterministic)
}
func (m *OperResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OperResult.Merge(m, src)
}
func (m *OperResult) XXX_Size() int {
	return xxx_messageInfo_OperResult.Size(m)
}
func (m *OperResult) XXX_DiscardUnknown() {
	xxx_messageInfo_OperResult.DiscardUnknown(m)
}

var xxx_messageInfo_OperResult proto.InternalMessageInfo

func (m *OperResult) GetRetCode() int32 {
	if m != nil {
		return m.RetCode
	}
	return 0
}

func (m *OperResult) GetRetMsg() string {
	if m != nil {
		return m.RetMsg
	}
	return ""
}

type NULLMsgReq struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NULLMsgReq) Reset()         { *m = NULLMsgReq{} }
func (m *NULLMsgReq) String() string { return proto.CompactTextString(m) }
func (*NULLMsgReq) ProtoMessage()    {}
func (*NULLMsgReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_170ba741606d8a4c, []int{1}
}

func (m *NULLMsgReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NULLMsgReq.Unmarshal(m, b)
}
func (m *NULLMsgReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NULLMsgReq.Marshal(b, m, deterministic)
}
func (m *NULLMsgReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NULLMsgReq.Merge(m, src)
}
func (m *NULLMsgReq) XXX_Size() int {
	return xxx_messageInfo_NULLMsgReq.Size(m)
}
func (m *NULLMsgReq) XXX_DiscardUnknown() {
	xxx_messageInfo_NULLMsgReq.DiscardUnknown(m)
}

var xxx_messageInfo_NULLMsgReq proto.InternalMessageInfo

func init() {
	proto.RegisterType((*OperResult)(nil), "proto.OperResult")
	proto.RegisterType((*NULLMsgReq)(nil), "proto.NULLMsgReq")
}

func init() { proto.RegisterFile("util.proto", fileDescriptor_170ba741606d8a4c) }

var fileDescriptor_170ba741606d8a4c = []byte{
	// 123 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2a, 0x2d, 0xc9, 0xcc,
	0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53, 0x4a, 0x0e, 0x5c, 0x5c, 0xfe, 0x05,
	0xa9, 0x45, 0x41, 0xa9, 0xc5, 0xa5, 0x39, 0x25, 0x42, 0x92, 0x5c, 0x1c, 0x45, 0xa9, 0x25, 0xf1,
	0xc9, 0xf9, 0x29, 0xa9, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0xac, 0x41, 0xec, 0x45, 0xa9, 0x25, 0xce,
	0xf9, 0x29, 0xa9, 0x42, 0xe2, 0x5c, 0x20, 0x66, 0x7c, 0x6e, 0x71, 0xba, 0x04, 0x93, 0x02, 0xa3,
	0x06, 0x67, 0x10, 0x5b, 0x51, 0x6a, 0x89, 0x6f, 0x71, 0xba, 0x12, 0x0f, 0x17, 0x97, 0x5f, 0xa8,
	0x8f, 0x8f, 0x6f, 0x71, 0x7a, 0x50, 0x6a, 0x61, 0x12, 0x1b, 0xd8, 0x58, 0x63, 0x40, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x85, 0x99, 0x91, 0x76, 0x6b, 0x00, 0x00, 0x00,
}
