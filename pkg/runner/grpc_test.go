/*
MIT License
Copyright (c) 2023 API Testing Authors.
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package runner

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/h2non/gock"
	testsrv "github.com/linuxsuren/api-testing/pkg/runner/grpc_test"
	atest "github.com/linuxsuren/api-testing/pkg/testing"
	fakeruntime "github.com/linuxsuren/go-fake-runtime"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var cache sync.Map

const (
	unary        = "/grpctest.Main/Unary"
	basic        = "/grpctest.Main/TestBasicType"
	advanced     = "/grpctest.Main/TestAdvancedType"
	clienSstream = "/grpctest.Main/ClientStream"
	serverStream = "/grpctest.Main/ServerStream"
	bidStream    = "/grpctest.Main/BidStream"

	unknownRPC = "/grpctest.Main/UnknownName"

	pburi = "http://localhost/pb"
)

type testUnit struct {
	name     string
	execer   fakeruntime.Execer
	testCase *atest.TestCase
	desc     *atest.GRPCDesc
	ctx      any
	prepare  func()
	verify   func(t *testing.T, output any, err error)
}

func TestGRPCTestCase(t *testing.T) {
	s := grpc.NewServer()
	testServer := &testsrv.TestServer{}
	testsrv.RegisterMainServer(s, testServer)

	l := runServer(t, s)

	doGRPCTest(t, l, nil, &atest.GRPCDesc{
		ImportPath: []string{"grpc_test"},
		ProtoFile:  "test.proto",
	})
	s.Stop()
}
func TestGRPCTestCaseWithSecure(t *testing.T) {
	creds, err := credentials.NewServerTLSFromFile("grpc_test/testassets/server.pem", "grpc_test/testassets/server.key")
	assert.Nil(t, err)

	s := grpc.NewServer(grpc.Creds(creds))
	testServer := &testsrv.TestServer{}
	testsrv.RegisterMainServer(s, testServer)

	l := runServer(t, s)

	doGRPCTest(t, l,
		&atest.Secure{
			Insecure:   false,
			CertFile:   "grpc_test/testassets/server.pem",
			ServerName: "atest",
		},
		&atest.GRPCDesc{
			ImportPath: []string{"grpc_test"},
			ProtoFile:  "test.proto",
		})

	s.Stop()
}

func TestGRPCProtoSetTestCase(t *testing.T) {
	s := grpc.NewServer()
	testServer := &testsrv.TestServer{}
	testsrv.RegisterMainServer(s, testServer)

	l := runServer(t, s)

	addition := []testUnit{
		{
			name: "test get protoset from url but url is error",
			desc: &atest.GRPCDesc{
				ProtoSet: pburi,
			},
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test get protoset from url",
			desc: &atest.GRPCDesc{
				ProtoSet: "http://localhost/pb",
			},
			prepare: func() {
				gock.New("http://localhost/pb").
					Reply(200).
					File("grpc_test/test.pb")
			},
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
				Expect: atest.Response{
					Body: getJSONOrCache("unary", &testsrv.HelloReply{
						Message: "Hello!",
					}),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
	}

	doGRPCTest(t, l, nil, &atest.GRPCDesc{
		ProtoSet: "grpc_test/test.pb",
	},
		addition...)
	s.Stop()
}

func TestGRPCReflectTestCase(t *testing.T) {
	s := grpc.NewServer()
	testServer := &testsrv.TestServer{}
	testsrv.RegisterMainServer(s, testServer)
	reflection.RegisterV1(s)

	l := runServer(t, s)

	doGRPCTest(t, l, nil, &atest.GRPCDesc{
		ServerReflection: true,
	})
	s.Stop()
}

func TestGRPCTestError(t *testing.T) {
	s := grpc.NewServer()
	testServer := &testsrv.TestServer{}
	testsrv.RegisterMainServer(s, testServer)

	l := runServer(t, s)

	tests := []testUnit{
		{
			name: "test proto not found",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
			},
			desc: &atest.GRPCDesc{
				ProtoFile: "unknown",
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test proto set found",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
			},
			desc: &atest.GRPCDesc{
				ProtoSet: "unknown",
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test reflect on unsupported server",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
			},
			desc: &atest.GRPCDesc{
				ServerReflection: true,
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test missing descriptor source",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
			},
			desc: &atest.GRPCDesc{},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test server is closed",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
				Expect: atest.Response{
					Body: getJSONOrCache("unary", &testsrv.HelloReply{
						Message: "Hello!",
					}),
				},
			},
			prepare: func() {
				s.Stop()
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
	}

	runUnits(tests, t, l, nil, &atest.GRPCDesc{})
}

func runServer(t *testing.T, s *grpc.Server) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err, "Listen port")

	log.Println("listening at", l.Addr().(*net.TCPAddr).Port)
	go s.Serve(l)
	return l
}

func doGRPCTest(t *testing.T, l net.Listener, sec *atest.Secure, desc *atest.GRPCDesc, addition ...testUnit) {
	tests := []testUnit{
		{
			name: "test unary rpc",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
				Expect: atest.Response{
					Body: getJSONOrCache("unary", &testsrv.HelloReply{
						Message: "Hello!",
					}),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "test unary rpc not equal",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unary,
					Body: "{}",
				},
				Expect: atest.Response{
					Body: getJSONOrCache(nil, &testsrv.HelloReply{
						Message: "Happy!",
					}),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test client stream rpc",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API: clienSstream,
					Body: getJSONOrCache("stream", []*testsrv.StreamMessage{
						{MsgID: 1, ExpectLen: 3},
						{MsgID: 2, ExpectLen: 3},
						{MsgID: 3, ExpectLen: 3},
					}),
				},
				Expect: atest.Response{
					Body: getJSONOrCache("streamRepeted", &testsrv.StreamMessageRepeated{
						Data: []*testsrv.StreamMessage{
							{MsgID: 1, ExpectLen: 3},
							{MsgID: 2, ExpectLen: 3},
							{MsgID: 3, ExpectLen: 3},
						},
					}),
					Verify: []string{`len(data[0].data) == 3`},
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "test server stream rpc",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  serverStream,
					Body: getJSONOrCache("streamRepeted", nil),
				},
				Expect: atest.Response{
					Body:   getJSONOrCache("stream", nil),
					Verify: []string{`len(data) == 3`},
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "test bid stream rpc",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  bidStream,
					Body: getJSONOrCache("stream", nil),
				},
				Expect: atest.Response{
					Body: getJSONOrCache("stream", nil),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "test bid stream rpc len not equal",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  bidStream,
					Body: getJSONOrCache("stream", nil),
				},
				Expect: atest.Response{
					Body: getJSONOrCache(nil, []*testsrv.StreamMessage{
						{MsgID: 1, ExpectLen: 2},
						{MsgID: 2, ExpectLen: 2},
					}),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test bid stream rpc content not equal",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  bidStream,
					Body: getJSONOrCache("stream", nil),
				},
				Expect: atest.Response{
					Body: getJSONOrCache(nil, []*testsrv.StreamMessage{
						{MsgID: 4, ExpectLen: 3},
						{MsgID: 5, ExpectLen: 3},
						{MsgID: 6, ExpectLen: 3},
					}),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test basic type",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API: basic,
					Body: getJSONOrCache("basic",
						&testsrv.BasicType{
							Int32:   rand.Int31(),
							Int64:   rand.Int63(),
							Uint32:  rand.Uint32(),
							Uint64:  rand.Uint64(),
							Float32: rand.Float32(),
							Float64: rand.Float64(),
							String_: time.Now().Format(time.RFC3339),
							Bool:    true,
						}),
				},
				Expect: atest.Response{
					Body: getJSONOrCache("basic", nil),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "test advanced type",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API: advanced,
					Body: getJSONOrCache("advanced",
						&testsrv.AdvancedType{
							Int32Array:   []int32{rand.Int31(), rand.Int31()},
							Int64Array:   []int64{rand.Int63(), rand.Int63()},
							Uint32Array:  []uint32{rand.Uint32(), rand.Uint32()},
							Uint64Array:  []uint64{rand.Uint64(), rand.Uint64()},
							Float32Array: []float32{rand.Float32(), rand.Float32()},
							Float64Array: []float64{rand.NormFloat64(), rand.NormFloat64()},
							StringArray:  []string{time.Now().Format(time.RFC3339), time.Now().Format(time.RFC822)},
							BoolArray:    []bool{true, false},
							HelloReplyMap: map[string]*testsrv.HelloReply{"key": {
								Message: "Hello",
							}},
						}),
				},
				Expect: atest.Response{
					Body: getJSONOrCache("advanced", nil),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "test advanced type not equal",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API: advanced,
					Body: getJSONOrCache("advanced",
						&testsrv.AdvancedType{
							Int32Array:   []int32{rand.Int31(), rand.Int31()},
							Int64Array:   []int64{rand.Int63(), rand.Int63()},
							Uint32Array:  []uint32{rand.Uint32(), rand.Uint32()},
							Uint64Array:  []uint64{rand.Uint64(), rand.Uint64()},
							Float32Array: []float32{rand.Float32(), rand.Float32()},
							Float64Array: []float64{rand.NormFloat64(), rand.NormFloat64()},
							StringArray:  []string{time.Now().Format(time.RFC3339), time.Now().Format(time.RFC822)},
							BoolArray:    []bool{true, false},
							HelloReplyMap: map[string]*testsrv.HelloReply{"key": {
								Message: "Hello",
							}},
						}),
				},
				Expect: atest.Response{
					Body: getJSONOrCache(nil,
						&testsrv.AdvancedType{
							Int32Array:   []int32{rand.Int31(), rand.Int31()},
							Int64Array:   []int64{rand.Int63(), rand.Int63()},
							Uint32Array:  []uint32{rand.Uint32(), rand.Uint32()},
							Uint64Array:  []uint64{rand.Uint64(), rand.Uint64()},
							Float32Array: []float32{rand.Float32(), rand.Float32()},
							Float64Array: []float64{rand.NormFloat64(), rand.NormFloat64()},
							StringArray:  []string{time.Now().Format(time.RFC3339), time.Now().Format(time.RFC822)},
							BoolArray:    []bool{true, false},
							HelloReplyMap: map[string]*testsrv.HelloReply{"key": {
								Message: "Happy",
							}},
						}),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test unknown rpc",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  unknownRPC,
					Body: "{}",
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "not found", "UnknownName")
			},
		},
		{
			name: "test invalid input",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  basic,
					Body: "{",
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			name: "test wrong input type",
			testCase: &atest.TestCase{
				Request: atest.Request{
					API:  basic,
					Body: getJSONOrCache("unary", nil),
				},
			},
			verify: func(t *testing.T, output any, err error) {
				assert.NotNil(t, err)
			},
		},
	}
	tests = append(tests, addition...)
	runUnits(tests, t, l, sec, desc)
}

func runUnits(tests []testUnit, t *testing.T, l net.Listener, sec *atest.Secure, desc *atest.GRPCDesc) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer gock.Clean()
			if tt.prepare != nil {
				tt.prepare()
			}
			if tt.verify == nil {
				tt.verify = hasError
			}
			if tt.desc != nil {
				desc = tt.desc
			}
			runner := NewGRPCTestCaseRunner(l.Addr().String(), *desc)
			runner.WithOutputWriter(os.Stdout)
			if tt.execer != nil {
				runner.WithExecer(tt.execer)
			}
			runner.WithSecure(sec)
			tt.testCase.Request.API = l.Addr().String() + tt.testCase.Request.API
			output, err := runner.RunTestCase(tt.testCase, tt.ctx, context.TODO())
			tt.verify(t, output, err)

			getter, ok := runner.(ResponseRecord)
			assert.True(t, ok)
			assert.NotNil(t, getter.GetResponseRecord())
		})
	}
}

func TestAPINameMatch(t *testing.T) {
	qn, err := splitFullQualifiedName("127.0.0.1:7070/server.Runner/GetVersion")
	assert.NoError(t, err)
	assert.Equal(t,
		protoreflect.FullName("server.Runner.GetVersion"),
		qn,
		"match full qualified name",
	)

	qn, err = splitFullQualifiedName("127.0.0.1:7070/server.v1.service/method")
	assert.NoError(t, err)
	assert.Equal(t,
		protoreflect.FullName("server.v1.service.method"),
		qn,
		"match full qualified name long",
	)

	qn, err = splitFullQualifiedName("127.0.0.1:7070//server.Runner/GetVersion")
	assert.NotNil(t,
		err,
		"unexpect leading character",
	)

	qn, err = splitFullQualifiedName("127.0.0.1:7070/server.Runner/GetVersion/")
	assert.NotNil(t,
		err,
		"unexpect trailing character",
	)
}

// getJSONOrCache can store the JSON string of value.
//
// Let key be nil represent not using cache.
func getJSONOrCache(key any, value any) (msg string) {
	v, ok := cache.Load(key)
	if ok && key != nil {
		return v.(string)
	}
	b, _ := json.Marshal(value)
	msg = string(b)
	if key != nil {
		cache.Store(key, msg)
	}
	return
}
