/*
Copyright 2024 Nokia.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/ // Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: github.com/pkgserver-dev/pkgserver/apis/pkgid/generated.proto

package pkgrevid

import (
	fmt "fmt"

	io "io"
	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"

	proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

func (m *Downstream) Reset()      { *m = Downstream{} }
func (*Downstream) ProtoMessage() {}
func (*Downstream) Descriptor() ([]byte, []int) {
	return fileDescriptor_a84e3191c6a7706e, []int{0}
}
func (m *Downstream) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Downstream) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Downstream) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Downstream.Merge(m, src)
}
func (m *Downstream) XXX_Size() int {
	return m.Size()
}
func (m *Downstream) XXX_DiscardUnknown() {
	xxx_messageInfo_Downstream.DiscardUnknown(m)
}

var xxx_messageInfo_Downstream proto.InternalMessageInfo

func (m *PackageRevID) Reset()      { *m = PackageRevID{} }
func (*PackageRevID) ProtoMessage() {}
func (*PackageRevID) Descriptor() ([]byte, []int) {
	return fileDescriptor_a84e3191c6a7706e, []int{1}
}
func (m *PackageRevID) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PackageRevID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *PackageRevID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PackageID.Merge(m, src)
}
func (m *PackageRevID) XXX_Size() int {
	return m.Size()
}
func (m *PackageRevID) XXX_DiscardUnknown() {
	xxx_messageInfo_PackageID.DiscardUnknown(m)
}

var xxx_messageInfo_PackageID proto.InternalMessageInfo

func (m *Upstream) Reset()      { *m = Upstream{} }
func (*Upstream) ProtoMessage() {}
func (*Upstream) Descriptor() ([]byte, []int) {
	return fileDescriptor_a84e3191c6a7706e, []int{2}
}
func (m *Upstream) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Upstream) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *Upstream) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Upstream.Merge(m, src)
}
func (m *Upstream) XXX_Size() int {
	return m.Size()
}
func (m *Upstream) XXX_DiscardUnknown() {
	xxx_messageInfo_Upstream.DiscardUnknown(m)
}

var xxx_messageInfo_Upstream proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Downstream)(nil), "github.com.pkgserver_dev.pkgserver.apis.pkgid.Downstream")
	proto.RegisterType((*PackageRevID)(nil), "github.com.pkgserver_dev.pkgserver.apis.pkgid.PackageID")
	proto.RegisterType((*Upstream)(nil), "github.com.pkgserver_dev.pkgserver.apis.pkgid.Upstream")
}

func init() {
	proto.RegisterFile("github.com/pkgserver-dev/pkgserver/apis/pkgid/generated.proto", fileDescriptor_a84e3191c6a7706e)
}

var fileDescriptor_a84e3191c6a7706e = []byte{
	// 365 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xd4, 0x93, 0xbb, 0x4e, 0xfb, 0x30,
	0x18, 0xc5, 0xe3, 0xde, 0x63, 0xe9, 0x7f, 0xc1, 0x53, 0xc4, 0xe0, 0xa2, 0x22, 0x21, 0x90, 0x68,
	0x22, 0xb1, 0xb3, 0x54, 0x5d, 0xd8, 0x90, 0x01, 0x21, 0xb1, 0x20, 0xb7, 0xf9, 0x08, 0x51, 0x69,
	0x6d, 0x39, 0x6e, 0x2a, 0x36, 0x1e, 0x81, 0x89, 0x67, 0x61, 0xe0, 0x01, 0x3a, 0x76, 0xec, 0x54,
	0xd1, 0xf0, 0x22, 0xa8, 0xae, 0xdb, 0x94, 0x09, 0x3a, 0xb2, 0xe5, 0x9c, 0xf3, 0x3b, 0x9f, 0x72,
	0x14, 0x05, 0x9f, 0x46, 0xb1, 0xbe, 0x1f, 0x76, 0xfc, 0xae, 0xe8, 0x07, 0xb2, 0x17, 0x25, 0xa0,
	0x52, 0x50, 0xcd, 0x10, 0xd2, 0x5c, 0x05, 0x5c, 0xc6, 0xc9, 0x42, 0xc6, 0x61, 0x10, 0xc1, 0x00,
	0x14, 0xd7, 0x10, 0xfa, 0x52, 0x09, 0x2d, 0x48, 0x33, 0xaf, 0xfb, 0xeb, 0xc2, 0x6d, 0x08, 0x69,
	0xae, 0xfc, 0x45, 0xdd, 0x37, 0xf5, 0xdd, 0x0d, 0x3c, 0x88, 0x44, 0x24, 0x02, 0x73, 0xa5, 0x33,
	0xbc, 0x33, 0xca, 0x08, 0xf3, 0xb4, 0xbc, 0xde, 0x78, 0x45, 0x18, 0xb7, 0xc5, 0x68, 0x90, 0x68,
	0x05, 0xbc, 0x4f, 0x0e, 0x70, 0x45, 0x73, 0x15, 0x81, 0xf6, 0xd0, 0x1e, 0x3a, 0x74, 0x5b, 0x7f,
	0xc7, 0xb3, 0xba, 0x93, 0xcd, 0xea, 0x95, 0x4b, 0xe3, 0x32, 0x9b, 0x92, 0x13, 0x8c, 0x15, 0x48,
	0x91, 0xc4, 0x5a, 0xa8, 0x47, 0xaf, 0x60, 0x58, 0x62, 0x59, 0xcc, 0xd6, 0x09, 0xdb, 0xa0, 0xc8,
	0x3e, 0x2e, 0x2b, 0xe0, 0x0f, 0x7d, 0xaf, 0x68, 0xf0, 0x3f, 0x16, 0x2f, 0xb3, 0x85, 0xc9, 0x96,
	0x19, 0x39, 0xc2, 0x55, 0xc9, 0xbb, 0x3d, 0x1e, 0x81, 0x57, 0x32, 0xd8, 0x3f, 0x8b, 0x55, 0xcf,
	0x97, 0x36, 0x5b, 0xe5, 0x8d, 0x97, 0x02, 0x76, 0xad, 0x79, 0xd6, 0xfe, 0x4d, 0x6f, 0x4e, 0x8e,
	0x71, 0x4d, 0x41, 0x1a, 0x27, 0xb1, 0x18, 0x78, 0x65, 0xc3, 0xfe, 0xb7, 0x6c, 0x8d, 0x59, 0x9f,
	0xad, 0x09, 0x12, 0x60, 0x77, 0x24, 0x54, 0x2f, 0x91, 0xbc, 0x0b, 0x5e, 0xc5, 0xe0, 0x3b, 0x16,
	0x77, 0xaf, 0x57, 0x01, 0xcb, 0x99, 0xc6, 0x1b, 0xc2, 0xb5, 0x2b, 0x69, 0xbf, 0xe8, 0xd7, 0xbd,
	0x68, 0xbb, 0xbd, 0x85, 0x9f, 0xed, 0x2d, 0x6e, 0xb1, 0xb7, 0xf4, 0xdd, 0xde, 0xd6, 0xc5, 0x78,
	0x4e, 0x9d, 0xc9, 0x9c, 0x3a, 0xd3, 0x39, 0x75, 0x9e, 0x32, 0x8a, 0xc6, 0x19, 0x45, 0x93, 0x8c,
	0xa2, 0x69, 0x46, 0xd1, 0x7b, 0x46, 0xd1, 0xf3, 0x07, 0x75, 0x6e, 0x9a, 0x5b, 0xfd, 0x55, 0x9f,
	0x01, 0x00, 0x00, 0xff, 0xff, 0x8d, 0x64, 0x5f, 0x27, 0x85, 0x03, 0x00, 0x00,
}

func (m *Downstream) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Downstream) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Downstream) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	i -= len(m.Package)
	copy(dAtA[i:], m.Package)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Package)))
	i--
	dAtA[i] = 0x22
	i -= len(m.Realm)
	copy(dAtA[i:], m.Realm)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Realm)))
	i--
	dAtA[i] = 0x1a
	i -= len(m.Repository)
	copy(dAtA[i:], m.Repository)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Repository)))
	i--
	dAtA[i] = 0x12
	i -= len(m.Target)
	copy(dAtA[i:], m.Target)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Target)))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *PackageRevID) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PackageRevID) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PackageRevID) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	i -= len(m.Workspace)
	copy(dAtA[i:], m.Workspace)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Workspace)))
	i--
	dAtA[i] = 0x32
	i -= len(m.Revision)
	copy(dAtA[i:], m.Revision)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Revision)))
	i--
	dAtA[i] = 0x2a
	i -= len(m.Package)
	copy(dAtA[i:], m.Package)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Package)))
	i--
	dAtA[i] = 0x22
	i -= len(m.Realm)
	copy(dAtA[i:], m.Realm)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Realm)))
	i--
	dAtA[i] = 0x1a
	i -= len(m.Repository)
	copy(dAtA[i:], m.Repository)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Repository)))
	i--
	dAtA[i] = 0x12
	i -= len(m.Target)
	copy(dAtA[i:], m.Target)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Target)))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *Upstream) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Upstream) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Upstream) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	i -= len(m.Revision)
	copy(dAtA[i:], m.Revision)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Revision)))
	i--
	dAtA[i] = 0x22
	i -= len(m.Package)
	copy(dAtA[i:], m.Package)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Package)))
	i--
	dAtA[i] = 0x1a
	i -= len(m.Realm)
	copy(dAtA[i:], m.Realm)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Realm)))
	i--
	dAtA[i] = 0x12
	i -= len(m.Repository)
	copy(dAtA[i:], m.Repository)
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Repository)))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintGenerated(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenerated(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Downstream) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Target)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Repository)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Realm)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Package)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func (m *PackageRevID) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Target)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Repository)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Realm)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Package)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Revision)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Workspace)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func (m *Upstream) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Repository)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Realm)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Package)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Revision)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func sovGenerated(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenerated(x uint64) (n int) {
	return sovGenerated(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Downstream) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Downstream{`,
		`Target:` + fmt.Sprintf("%v", this.Target) + `,`,
		`Repository:` + fmt.Sprintf("%v", this.Repository) + `,`,
		`Realm:` + fmt.Sprintf("%v", this.Realm) + `,`,
		`Package:` + fmt.Sprintf("%v", this.Package) + `,`,
		`}`,
	}, "")
	return s
}
func (this *PackageRevID) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&PackageID{`,
		`Target:` + fmt.Sprintf("%v", this.Target) + `,`,
		`Repository:` + fmt.Sprintf("%v", this.Repository) + `,`,
		`Realm:` + fmt.Sprintf("%v", this.Realm) + `,`,
		`Package:` + fmt.Sprintf("%v", this.Package) + `,`,
		`Revision:` + fmt.Sprintf("%v", this.Revision) + `,`,
		`Workspace:` + fmt.Sprintf("%v", this.Workspace) + `,`,
		`}`,
	}, "")
	return s
}
func (this *Upstream) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Upstream{`,
		`Repository:` + fmt.Sprintf("%v", this.Repository) + `,`,
		`Realm:` + fmt.Sprintf("%v", this.Realm) + `,`,
		`Package:` + fmt.Sprintf("%v", this.Package) + `,`,
		`Revision:` + fmt.Sprintf("%v", this.Revision) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringGenerated(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Downstream) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Downstream: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Downstream: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Target", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Target = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Repository", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Repository = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Realm", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Realm = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Package", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Package = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *PackageRevID) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PackageID: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PackageID: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Target", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Target = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Repository", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Repository = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Realm", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Realm = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Package", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Package = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Revision", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Revision = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Workspace", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Workspace = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Upstream) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Upstream: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Upstream: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Repository", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Repository = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Realm", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Realm = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Package", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Package = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Revision", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Revision = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGenerated(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthGenerated
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenerated
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenerated
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenerated        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenerated          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenerated = fmt.Errorf("proto: unexpected end of group")
)