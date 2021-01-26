// (c) Copyright IBM Corp. 2021
// (c) Copyright Instana Inc. 2021

package instaawssdk_test

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/sns"
	instana "github.com/instana/go-sensor"
	"github.com/instana/go-sensor/instrumentation/instaawssdk"
	"github.com/instana/testify/assert"
	"github.com/instana/testify/require"
)

func TestStartSNSSpan_WithActiveSpan(t *testing.T) {
	recorder := instana.NewTestRecorder()
	sensor := instana.NewSensorWithTracer(
		instana.NewTracerWithEverything(instana.DefaultOptions(), recorder),
	)

	parentSp := sensor.Tracer().StartSpan("testing")

	req := newSNSRequest()
	req.SetContext(instana.ContextWithSpan(req.Context(), parentSp))

	instaawssdk.StartSNSSpan(req, sensor)

	sp, ok := instana.SpanFromContext(req.Context())
	require.True(t, ok)

	sp.Finish()
	parentSp.Finish()

	spans := recorder.GetQueuedSpans()
	require.Len(t, spans, 2)

	snsSpan, testingSpan := spans[0], spans[1]

	assert.Equal(t, testingSpan.TraceID, snsSpan.TraceID)
	assert.Equal(t, testingSpan.SpanID, snsSpan.ParentID)
	assert.NotEqual(t, testingSpan.SpanID, snsSpan.SpanID)
	assert.NotEmpty(t, snsSpan.SpanID)

	assert.Equal(t, "sns", snsSpan.Name)
	assert.Empty(t, snsSpan.Ec)

	assert.IsType(t, instana.AWSSNSSpanData{}, snsSpan.Data)
}

func TestStartSNSSpan_NoActiveSpan(t *testing.T) {
	recorder := instana.NewTestRecorder()
	sensor := instana.NewSensorWithTracer(
		instana.NewTracerWithEverything(instana.DefaultOptions(), recorder),
	)

	req := newSNSRequest()
	instaawssdk.StartSNSSpan(req, sensor)

	_, ok := instana.SpanFromContext(req.Context())
	require.False(t, ok)
}

func TestFinalizeSNS_NoError(t *testing.T) {
	recorder := instana.NewTestRecorder()
	sensor := instana.NewSensorWithTracer(
		instana.NewTracerWithEverything(instana.DefaultOptions(), recorder),
	)

	sp := sensor.Tracer().StartSpan("sns")

	req := newSNSRequest()
	req.SetContext(instana.ContextWithSpan(req.Context(), sp))

	instaawssdk.FinalizeSNSSpan(req)

	spans := recorder.GetQueuedSpans()
	require.Len(t, spans, 1)

	snsSpan := spans[0]

	assert.IsType(t, instana.AWSSNSSpanData{}, snsSpan.Data)
}

func TestFinalizeSNSSpan_WithError(t *testing.T) {
	recorder := instana.NewTestRecorder()
	sensor := instana.NewSensorWithTracer(
		instana.NewTracerWithEverything(instana.DefaultOptions(), recorder),
	)

	sp := sensor.Tracer().StartSpan("sns")

	req := newSNSRequest()
	req.Error = awserr.New("42", "test error", errors.New("an error occurred"))
	req.SetContext(instana.ContextWithSpan(req.Context(), sp))

	instaawssdk.FinalizeSNSSpan(req)

	spans := recorder.GetQueuedSpans()
	require.Len(t, spans, 1)

	snsSpan := spans[0]

	assert.IsType(t, instana.AWSSNSSpanData{}, snsSpan.Data)
	data := snsSpan.Data.(instana.AWSSNSSpanData)

	assert.Equal(t, instana.AWSSNSSpanTags{
		Error: req.Error.Error(),
	}, data.Tags)
}

func newSNSRequest() *request.Request {
	svc := sns.New(unit.Session)
	req, _ := svc.PublishRequest(&sns.PublishInput{
		Message:     aws.String("message content"),
		PhoneNumber: aws.String("test-phone-no"),
		Subject:     aws.String("test-subject"),
		TargetArn:   aws.String("test-target-arn"),
		TopicArn:    aws.String("test-topic-arn"),
	})

	return req
}
