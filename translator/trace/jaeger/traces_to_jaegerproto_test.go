// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaeger

import (
	"testing"

	"github.com/jaegertracing/jaeger/model"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func TestGetTagFromStatusCode(t *testing.T) {
	tests := []struct {
		name string
		code pdata.StatusCode
		tag  model.KeyValue
	}{
		{
			name: "ok",
			code: pdata.StatusCode(otlptrace.Status_Ok),
			tag: model.KeyValue{
				Key:    tracetranslator.TagStatusCode,
				VInt64: int64(otlptrace.Status_Ok),
				VType:  model.ValueType_INT64,
			},
		},

		{
			name: "unknown",
			code: pdata.StatusCode(otlptrace.Status_UnknownError),
			tag: model.KeyValue{
				Key:    tracetranslator.TagStatusCode,
				VInt64: int64(otlptrace.Status_UnknownError),
				VType:  model.ValueType_INT64,
			},
		},

		{
			name: "not-found",
			code: pdata.StatusCode(otlptrace.Status_NotFound),
			tag: model.KeyValue{
				Key:    tracetranslator.TagStatusCode,
				VInt64: int64(otlptrace.Status_NotFound),
				VType:  model.ValueType_INT64,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := getTagFromStatusCode(test.code)
			assert.True(t, ok)
			assert.EqualValues(t, test.tag, got)
		})
	}
}

func TestGetErrorTagFromStatusCode(t *testing.T) {
	errTag := model.KeyValue{
		Key:   tracetranslator.TagError,
		VBool: true,
		VType: model.ValueType_BOOL,
	}

	got, ok := getErrorTagFromStatusCode(pdata.StatusCode(otlptrace.Status_Ok))
	assert.False(t, ok)

	got, ok = getErrorTagFromStatusCode(pdata.StatusCode(otlptrace.Status_UnknownError))
	assert.True(t, ok)
	assert.EqualValues(t, errTag, got)

	got, ok = getErrorTagFromStatusCode(pdata.StatusCode(otlptrace.Status_NotFound))
	assert.True(t, ok)
	assert.EqualValues(t, errTag, got)
}

func TestGetTagFromStatusMsg(t *testing.T) {
	got, ok := getTagFromStatusMsg("")
	assert.False(t, ok)

	got, ok = getTagFromStatusMsg("test-error")
	assert.True(t, ok)
	assert.EqualValues(t, model.KeyValue{
		Key:   tracetranslator.TagStatusMsg,
		VStr:  "test-error",
		VType: model.ValueType_STRING,
	}, got)
}

func TestGetTagFromSpanKind(t *testing.T) {
	tests := []struct {
		name string
		kind pdata.SpanKind
		tag  model.KeyValue
		ok   bool
	}{
		{
			name: "unspecified",
			kind: pdata.SpanKindUNSPECIFIED,
			tag:  model.KeyValue{},
			ok:   false,
		},

		{
			name: "client",
			kind: pdata.SpanKindCLIENT,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindClient),
			},
			ok: true,
		},

		{
			name: "server",
			kind: pdata.SpanKindSERVER,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindServer),
			},
			ok: true,
		},

		{
			name: "producer",
			kind: pdata.SpanKindPRODUCER,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindProducer),
			},
			ok: true,
		},

		{
			name: "consumer",
			kind: pdata.SpanKindCONSUMER,
			tag: model.KeyValue{
				Key:   tracetranslator.TagSpanKind,
				VType: model.ValueType_STRING,
				VStr:  string(tracetranslator.OpenTracingSpanKindConsumer),
			},
			ok: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := getTagFromSpanKind(test.kind)
			assert.Equal(t, test.ok, ok)
			assert.EqualValues(t, test.tag, got)
		})
	}
}

func TestAttributesToJaegerProtoTags(t *testing.T) {

	attributes := pdata.NewAttributeMap()
	attributes.InsertBool("bool-val", true)
	attributes.InsertInt("int-val", 123)
	attributes.InsertString("string-val", "abc")
	attributes.InsertDouble("double-val", 1.23)
	attributes.InsertString(conventions.AttributeServiceName, "service-name")

	expected := []model.KeyValue{
		{
			Key:   "bool-val",
			VType: model.ValueType_BOOL,
			VBool: true,
		},
		{
			Key:    "int-val",
			VType:  model.ValueType_INT64,
			VInt64: 123,
		},
		{
			Key:   "string-val",
			VType: model.ValueType_STRING,
			VStr:  "abc",
		},
		{
			Key:      "double-val",
			VType:    model.ValueType_FLOAT64,
			VFloat64: 1.23,
		},
		{
			Key:   conventions.AttributeServiceName,
			VType: model.ValueType_STRING,
			VStr:  "service-name",
		},
	}

	got := appendTagsFromAttributes(make([]model.KeyValue, 0, len(expected)), attributes)
	require.EqualValues(t, expected, got)

	// The last item in expected ("service-name") must be skipped in resource tags translation
	got = appendTagsFromResourceAttributes(make([]model.KeyValue, 0, len(expected)-1), attributes)
	require.EqualValues(t, expected[:4], got)
}

func TestInternalTracesToJaegerProto(t *testing.T) {

	tests := []struct {
		name string
		td   pdata.Traces
		jb   model.Batch
		err  error
	}{
		{
			name: "empty",
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},

		{
			name: "no-spans",
			td:   generateTraceDataResourceOnly(),
			jb: model.Batch{
				Process: generateProtoProcess(),
			},
			err: nil,
		},

		{
			name: "one-span-no-resources",
			td:   generateTraceDataOneSpanNoResource(),
			jb: model.Batch{
				Spans: []*model.Span{
					generateProtoSpan(),
				},
			},
			err: nil,
		},
		{
			name: "two-spans-child-parent",
			td:   generateTraceDataTwoSpansChildParent(),
			jb: model.Batch{
				Spans: []*model.Span{
					generateProtoSpan(),
					generateProtoChildSpanWithErrorTags(),
				},
			},
			err: nil,
		},

		{
			name: "two-spans-with-follower",
			td:   generateTraceDataTwoSpansWithFollower(),
			jb: model.Batch{
				Spans: []*model.Span{
					generateProtoSpan(),
					generateProtoFollowerSpan(),
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jbs, err := InternalTracesToJaegerProto(test.td)
			assert.EqualValues(t, test.err, err)
			if test.name == "empty" {
				assert.Nil(t, jbs)
			} else {
				assert.Equal(t, 1, len(jbs))
				assert.EqualValues(t, test.jb, *jbs[0])
			}
		})
	}
}

// generateProtoChildSpanWithErrorTags generates a jaeger span to be used in
// internal->jaeger translation test. It supposed to be the same as generateProtoChildSpan
// that used in jaeger->internal, but jaeger->internal translation infers status code from http status if
// status.code is not set, so the pipeline jaeger->internal->jaeger adds two more tags as the result in that case.
func generateProtoChildSpanWithErrorTags() *model.Span {
	span := generateProtoChildSpan()
	span.Tags = append(span.Tags, model.KeyValue{
		Key:    tracetranslator.TagStatusCode,
		VType:  model.ValueType_INT64,
		VInt64: tracetranslator.OCNotFound,
	})
	span.Tags = append(span.Tags, model.KeyValue{
		Key:   tracetranslator.TagError,
		VBool: true,
		VType: model.ValueType_BOOL,
	})
	return span
}

func BenchmarkInternalTracesToJaegerProto(b *testing.B) {
	td := generateTraceDataTwoSpansChildParent()
	resource := generateTraceDataResourceOnly().ResourceSpans().At(0).Resource()
	resource.CopyTo(td.ResourceSpans().At(0).Resource())

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		InternalTracesToJaegerProto(td)
	}
}
