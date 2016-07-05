package cassandra

// Copyright (c) 2012 The gocql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"gopkg.in/inf.v0"
	"strings"
)

// TypeInfo describes a Cassandra specific data type.
type TypeInfo interface {
	Type() Type
	Version() byte
	Custom() string

	// New creates a pointer to an empty version of whatever type
	// is referenced by the TypeInfo receiver
	New() interface{}
}

type NativeType struct {
	proto  byte
	typ    Type
	custom string // only used for TypeCustom
}

func (t NativeType) New() interface{} {
	return reflect.New(goType(t)).Interface()
}

func (s NativeType) Type() Type {
	return s.typ
}

func (s NativeType) Version() byte {
	return s.proto
}

func (s NativeType) Custom() string {
	return s.custom
}

func (s NativeType) String() string {
	switch s.typ {
	case TypeCustom:
		return fmt.Sprintf("%s(%s)", s.typ, s.custom)
	default:
		return s.typ.String()
	}
}

type CollectionType struct {
	NativeType
	Key  TypeInfo // only used for TypeMap
	Elem TypeInfo // only used for TypeMap, TypeList and TypeSet
}

func goType(t TypeInfo) reflect.Type {
	switch t.Type() {
	case TypeVarchar, TypeAscii, TypeInet, TypeText:
		return reflect.TypeOf(*new(string))
	case TypeBigInt, TypeCounter:
		return reflect.TypeOf(*new(int64))
	case TypeTimestamp:
		return reflect.TypeOf(*new(time.Time))
	case TypeBlob:
		return reflect.TypeOf(*new([]byte))
	case TypeBoolean:
		return reflect.TypeOf(*new(bool))
	case TypeFloat:
		return reflect.TypeOf(*new(float32))
	case TypeDouble:
		return reflect.TypeOf(*new(float64))
	case TypeInt:
		return reflect.TypeOf(*new(int))
	case TypeDecimal:
		return reflect.TypeOf(*new(*inf.Dec))
	case TypeUUID, TypeTimeUUID:
		return reflect.TypeOf(*new(UUID))
	case TypeList, TypeSet:
		return reflect.SliceOf(goType(t.(CollectionType).Elem))
	case TypeMap:
		return reflect.MapOf(goType(t.(CollectionType).Key), goType(t.(CollectionType).Elem))
	case TypeVarint:
		return reflect.TypeOf(*new(*big.Int))
	case TypeTuple:
		// what can we do here? all there is to do is to make a list of interface{}
		tuple := t.(TupleTypeInfo)
		return reflect.TypeOf(make([]interface{}, len(tuple.Elems)))
	case TypeUDT:
		return reflect.TypeOf(make(map[string]interface{}))
	default:
		return nil
	}
}

func (t CollectionType) New() interface{} {
	return reflect.New(goType(t)).Interface()
}

func (c CollectionType) String() string {
	switch c.typ {
	case TypeMap:
		return fmt.Sprintf("%s(%s, %s)", c.typ, c.Key, c.Elem)
	case TypeList, TypeSet:
		return fmt.Sprintf("%s(%s)", c.typ, c.Elem)
	case TypeCustom:
		return fmt.Sprintf("%s(%s)", c.typ, c.custom)
	default:
		return c.typ.String()
	}
}

type TupleTypeInfo struct {
	NativeType
	Elems []TypeInfo
}

func (t TupleTypeInfo) New() interface{} {
	return reflect.New(goType(t)).Interface()
}

type UDTField struct {
	Name string
	Type TypeInfo
}

type UDTTypeInfo struct {
	NativeType
	KeySpace string
	Name     string
	Elements []UDTField
}

func (u UDTTypeInfo) New() interface{} {
	return reflect.New(goType(u)).Interface()
}

func (u UDTTypeInfo) String() string {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%s.%s{", u.KeySpace, u.Name)
	first := true
	for _, e := range u.Elements {
		if !first {
			fmt.Fprint(buf, ",")
		} else {
			first = false
		}

		fmt.Fprintf(buf, "%s=%v", e.Name, e.Type)
	}
	fmt.Fprint(buf, "}")

	return buf.String()
}

// String returns a human readable name for the Cassandra datatype
// described by t.
// Type is the identifier of a Cassandra internal datatype.
type Type int

const (
	TypeCustom    Type = 0x0000
	TypeAscii     Type = 0x0001
	TypeBigInt    Type = 0x0002
	TypeBlob      Type = 0x0003
	TypeBoolean   Type = 0x0004
	TypeCounter   Type = 0x0005
	TypeDecimal   Type = 0x0006
	TypeDouble    Type = 0x0007
	TypeFloat     Type = 0x0008
	TypeInt       Type = 0x0009
	TypeText      Type = 0x000A
	TypeTimestamp Type = 0x000B
	TypeUUID      Type = 0x000C
	TypeVarchar   Type = 0x000D
	TypeVarint    Type = 0x000E
	TypeTimeUUID  Type = 0x000F
	TypeInet      Type = 0x0010
	TypeDate      Type = 0x0011
	TypeTime      Type = 0x0012
	TypeSmallInt  Type = 0x0013
	TypeTinyInt   Type = 0x0014
	TypeList      Type = 0x0020
	TypeMap       Type = 0x0021
	TypeSet       Type = 0x0022
	TypeUDT       Type = 0x0030
	TypeTuple     Type = 0x0031
)

// String returns the name of the identifier.
func (t Type) String() string {
	switch t {
	case TypeCustom:
		return "custom"
	case TypeAscii:
		return "ascii"
	case TypeBigInt:
		return "bigint"
	case TypeBlob:
		return "blob"
	case TypeBoolean:
		return "boolean"
	case TypeCounter:
		return "counter"
	case TypeDecimal:
		return "decimal"
	case TypeDouble:
		return "double"
	case TypeFloat:
		return "float"
	case TypeInt:
		return "int"
	case TypeText:
		return "text"
	case TypeTimestamp:
		return "timestamp"
	case TypeUUID:
		return "uuid"
	case TypeVarchar:
		return "varchar"
	case TypeTimeUUID:
		return "timeuuid"
	case TypeInet:
		return "inet"
	case TypeDate:
		return "date"
	case TypeTime:
		return "time"
	case TypeSmallInt:
		return "smallint"
	case TypeTinyInt:
		return "tinyint"
	case TypeList:
		return "list"
	case TypeMap:
		return "map"
	case TypeSet:
		return "set"
	case TypeVarint:
		return "varint"
	case TypeTuple:
		return "tuple"
	default:
		return fmt.Sprintf("unknown_type_%d", t)
	}
}

const (
	apacheCassandraTypePrefix = "org.apache.cassandra.db.marshal."
)

// get Apache Cassandra types
func getApacheCassandraType(class string) Type {
	switch strings.TrimPrefix(class, apacheCassandraTypePrefix) {
	case "AsciiType":
		return TypeAscii
	case "LongType":
		return TypeBigInt
	case "BytesType":
		return TypeBlob
	case "BooleanType":
		return TypeBoolean
	case "CounterColumnType":
		return TypeCounter
	case "DecimalType":
		return TypeDecimal
	case "DoubleType":
		return TypeDouble
	case "FloatType":
		return TypeFloat
	case "Int32Type":
		return TypeInt
	case "ShortType":
		return TypeSmallInt
	case "ByteType":
		return TypeTinyInt
	case "DateType", "TimestampType":
		return TypeTimestamp
	case "UUIDType", "LexicalUUIDType":
		return TypeUUID
	case "UTF8Type":
		return TypeVarchar
	case "IntegerType":
		return TypeVarint
	case "TimeUUIDType":
		return TypeTimeUUID
	case "InetAddressType":
		return TypeInet
	case "MapType":
		return TypeMap
	case "ListType":
		return TypeList
	case "SetType":
		return TypeSet
	case "TupleType":
		return TypeTuple
	default:
		return TypeCustom
	}
}

// error Types
type ErrType int

const (
	errServer          ErrType = 0x0000
	errProtocol        ErrType = 0x000A
	errCredentials     ErrType = 0x0100
	errUnavailable     ErrType = 0x1000
	errOverloaded      ErrType = 0x1001
	errBootstrapping   ErrType = 0x1002
	errTruncate        ErrType = 0x1003
	errWriteTimeout    ErrType = 0x1100
	errReadTimeout     ErrType = 0x1200
	errReadFailure     ErrType = 0x1300
	errFunctionFailure ErrType = 0x1400
	errWriteFailure    ErrType = 0x1500
	errSyntax          ErrType = 0x2000
	errUnauthorized    ErrType = 0x2100
	errInvalid         ErrType = 0x2200
	errConfig          ErrType = 0x2300
	errAlreadyExists   ErrType = 0x2400
	errUnprepared      ErrType = 0x2500
)

func (this ErrType) String() string {
	switch this {
	case errUnavailable:
		return "errUnavailable"
	case errWriteTimeout:
		return "errWriteTimeout"
	case errReadTimeout:
		return "errReadTimeout"
	case errAlreadyExists:
		return "errAlreadyExists"
	case errUnprepared:
		return "errUnprepared"
	case errReadFailure:
		return "errReadFailure"
	case errWriteFailure:
		return "errWriteFailure"
	case errFunctionFailure:
		return "errFunctionFailure"
	case errInvalid:
		return "errInvalid"
	case errBootstrapping:
		return "errBootstrapping"
	case errConfig:
		return "errConfig"
	case errCredentials:
		return "errCredentials"
	case errOverloaded:
		return "errOverloaded"
	case errProtocol:
		return "errProtocol"
	case errServer:
		return "errServer"
	case errSyntax:
		return "errSyntax"
	case errTruncate:
		return "errTruncate"
	case errUnauthorized:
		return "errUnauthorized"
	}

	return fmt.Sprintf("ErrUnknown: 0x%x", this)
}

const (
	protoDirectionMask = 0x80
	protoVersionMask   = 0x7F
	protoVersion1      = 0x01
	protoVersion2      = 0x02
	protoVersion3      = 0x03
	protoVersion4      = 0x04

	maxFrameSize = 256 * 1024 * 1024
)

type protoVersion byte

func (p protoVersion) request() bool {
	v := p.version()

	if v < protoVersion1 || v > protoVersion4 {
		logp.Err("unsupported request version: %x", v)
	}

	if v == protoVersion4 {
		return p == 0x04
	}

	if v == protoVersion3 {
		return p == 0x03
	}

	return p == 0x00
}

func (p protoVersion) response() bool {
	v := p.version()

	if v < protoVersion1 || v > protoVersion4 {
		logp.Err("unsupported response version: %x", v)
	}

	if v == protoVersion4 {
		return p == 0x84
	}

	if v == protoVersion3 {
		return p == 0x83
	}

	return p == 0x80
}

func (p protoVersion) version() byte {
	return byte(p) & protoVersionMask
}

func (p protoVersion) String() string {
	dir := "REQ"
	if p.response() {
		dir = "RESP"
	}

	return fmt.Sprintf("[version=%d direction=%s]", p.version(), dir)
}

type frameOp byte

const (
	// header ops
	opError         frameOp = 0x00
	opStartup       frameOp = 0x01
	opReady         frameOp = 0x02
	opAuthenticate  frameOp = 0x03
	opOptions       frameOp = 0x05
	opSupported     frameOp = 0x06
	opQuery         frameOp = 0x07
	opResult        frameOp = 0x08
	opPrepare       frameOp = 0x09
	opExecute       frameOp = 0x0A
	opRegister      frameOp = 0x0B
	opEvent         frameOp = 0x0C
	opBatch         frameOp = 0x0D
	opAuthChallenge frameOp = 0x0E
	opAuthResponse  frameOp = 0x0F
	opAuthSuccess   frameOp = 0x10
)

func (f frameOp) String() string {
	switch f {
	case opError:
		return "ERROR"
	case opStartup:
		return "STARTUP"
	case opReady:
		return "READY"
	case opAuthenticate:
		return "AUTHENTICATE"
	case opOptions:
		return "OPTIONS"
	case opSupported:
		return "SUPPORTED"
	case opQuery:
		return "QUERY"
	case opResult:
		return "RESULT"
	case opPrepare:
		return "PREPARE"
	case opExecute:
		return "EXECUTE"
	case opRegister:
		return "REGISTER"
	case opEvent:
		return "EVENT"
	case opBatch:
		return "BATCH"
	case opAuthChallenge:
		return "AUTH_CHALLENGE"
	case opAuthResponse:
		return "AUTH_RESPONSE"
	case opAuthSuccess:
		return "AUTH_SUCCESS"
	default:
		return fmt.Sprintf("UNKNOWN_OP_%d", f)
	}
}

const (
	// result kind
	resultKindVoid          = 1
	resultKindRows          = 2
	resultKindSetKeyspace   = 3
	resultKindPrepared      = 4
	resultKindSchemaChanged = 5

	// rows flags
	flagGlobalTableSpec int = 0x01
	flagHasMorePages    int = 0x02
	flagNoMetaData      int = 0x04

	// query flags
	flagValues                byte = 0x01
	flagSkipMetaData          byte = 0x02
	flagPageSize              byte = 0x04
	flagWithPagingState       byte = 0x08
	flagWithSerialConsistency byte = 0x10
	flagDefaultTimestamp      byte = 0x20
	flagWithNameValues        byte = 0x40

	// header flags
	flagCompress      byte = 0x01
	flagTracing       byte = 0x02
	flagCustomPayload byte = 0x04
	flagWarning       byte = 0x08
)

type Consistency uint16

const (
	Any         Consistency = 0x00
	One         Consistency = 0x01
	Two         Consistency = 0x02
	Three       Consistency = 0x03
	Quorum      Consistency = 0x04
	All         Consistency = 0x05
	LocalQuorum Consistency = 0x06
	EachQuorum  Consistency = 0x07
	LocalOne    Consistency = 0x0A
)

func (c Consistency) String() string {
	switch c {
	case Any:
		return "ANY"
	case One:
		return "ONE"
	case Two:
		return "TWO"
	case Three:
		return "THREE"
	case Quorum:
		return "QUORUM"
	case All:
		return "ALL"
	case LocalQuorum:
		return "LOCAL_QUORUM"
	case EachQuorum:
		return "EACH_QUORUM"
	case LocalOne:
		return "LOCAL_ONE"
	default:
		return fmt.Sprintf("UNKNOWN_CONS_0x%x", uint16(c))
	}
}

type SerialConsistency uint16

const (
	Serial      SerialConsistency = 0x08
	LocalSerial SerialConsistency = 0x09
)

func (s SerialConsistency) String() string {
	switch s {
	case Serial:
		return "SERIAL"
	case LocalSerial:
		return "LOCAL_SERIAL"
	default:
		return fmt.Sprintf("UNKNOWN_SERIAL_CONS_0x%x", uint16(s))
	}
}
