// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpendpoint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testContextKey string

var foo testContextKey = "foo"

func TestNew(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedRequest Request = &request{
			note: note{
				contents: []byte("request"),
			},
		}

		expectedResponse Response = &response{
			note: note{
				contents: []byte("response"),
			},
		}

		expectedCtx = context.WithValue(context.Background(), foo, "bar")
		service     = new(mockService)
		endpoint    = New(service)
	)

	service.On("ServeWRP", expectedCtx, expectedRequest).Return(expectedResponse, error(nil)).Once()
	actualResponse, err := endpoint(expectedCtx, expectedRequest)
	assert.Equal(expectedResponse, actualResponse)
	assert.NoError(err)
	service.AssertExpectations(t)
}

func TestWrap(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedRequest Request = &request{
			note: note{
				contents: []byte("request"),
			},
		}

		expectedResponse Response = &response{
			note: note{
				contents: []byte("response"),
			},
		}

		expectedCtx    = context.WithValue(context.Background(), foo, "bar")
		endpointCalled = false
		endpoint       = func(ctx context.Context, value interface{}) (interface{}, error) {
			endpointCalled = true
			assert.Equal(expectedCtx, ctx)
			assert.Equal(expectedRequest, value)
			return expectedResponse, nil
		}

		service = Wrap(endpoint)
	)

	actualResponse, err := service.ServeWRP(expectedCtx, expectedRequest)
	assert.Equal(expectedResponse, actualResponse)
	assert.NoError(err)
	assert.True(endpointCalled)
}
