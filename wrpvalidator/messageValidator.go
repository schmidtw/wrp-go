// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpvalidator

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/multierr"
)

var (
	ErrorNotSimpleResponseRequestType = NewValidatorError(errors.New("not simple response request message type"), "", []string{"Type"})
	ErrorNotSimpleEventType           = NewValidatorError(errors.New("not simple event message type"), "", []string{"Type"})
	ErrorInvalidSpanLength            = NewValidatorError(errors.New("invalid span length"), "", []string{"Spans"})
	ErrorInvalidSpanFormat            = NewValidatorError(errors.New("invalid span format"), "", []string{"Spans"})
)

// spanFormat is a simple map of allowed span format.
var spanFormat = map[int]string{
	// parent is the root parent for the spans below to link to
	0: "parent",
	// name is the name of the operation
	1: "name",
	// start time of the operation.
	2: "start time",
	// duration is how long the operation took.
	3: "duration",
	// status of the operation
	4: "status",
}

// SimpleEventValidators ensures messages are valid based on
// each validator in the list. SimpleEventValidators validates the following:
// UTF8 (all string fields), MessageType is valid, Source, Destination, MessageType is of SimpleEventMessageType.
func SimpleEventValidators(f *touchstone.Factory, labelNames ...string) (Validators, error) {
	var errs error
	sv, err := SpecValidators(f, labelNames...)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	stv, err := NewSimpleEventTypeValidator(f, labelNames...)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	return sv.AddFunc(stv), errs
}

// SimpleResponseRequestValidators ensures messages are valid based on
// each validator in the list. SimpleResponseRequestValidators validates the following:
// UTF8 (all string fields), MessageType is valid, Source, Destination, Spans, MessageType is of
// SimpleRequestResponseMessageType.
func SimpleResponseRequestValidators(f *touchstone.Factory, labelNames ...string) (Validators, error) {
	var errs error
	sv, err := SpecValidators(f, labelNames...)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	stv, err := NewSimpleResponseRequestTypeValidator(f, labelNames...)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	spv, err := NewSpansValidator(f, labelNames...)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	return sv.AddFunc(stv, spv), errs
}

// NewSimpleResponseRequestTypeValidator is the metric variant of SimpleResponseRequestTypeValidator
func NewSimpleResponseRequestTypeValidator(f *touchstone.Factory, labelNames ...string) (ValidatorFunc, error) {
	m, err := newSimpleRequestResponseMessageTypeValidatorErrorTotal(f, labelNames...)

	return func(msg wrp.Message, ls prometheus.Labels) error {
		err := SimpleResponseRequestTypeValidator(msg)
		if err != nil {
			m.With(ls).Add(1.0)
		}

		return err
	}, err
}

// NewSimpleEventTypeValidator is the metric variant of SimpleEventTypeValidator
func NewSimpleEventTypeValidator(f *touchstone.Factory, labelNames ...string) (ValidatorFunc, error) {
	m, err := newSimpleEventTypeValidatorErrorTotal(f, labelNames...)

	return func(msg wrp.Message, ls prometheus.Labels) error {
		err := SimpleEventTypeValidator(msg)
		if err != nil {
			m.With(ls).Add(1.0)
		}

		return err
	}, err
}

// NewSpansValidator is the metric variant of SpansValidator
func NewSpansValidator(f *touchstone.Factory, labelNames ...string) (ValidatorFunc, error) {
	m, err := newSpansValidatorErrorTotal(f, labelNames...)

	return func(msg wrp.Message, ls prometheus.Labels) error {
		err := SpansValidator(msg)
		if err != nil {
			m.With(ls).Add(1.0)
		}

		return err
	}, err
}

// SimpleResponseRequestTypeValidator takes messages and validates their Type is of SimpleRequestResponseMessageType.
func SimpleResponseRequestTypeValidator(m wrp.Message) error {
	if m.Type != wrp.SimpleRequestResponseMessageType {
		return ErrorNotSimpleResponseRequestType
	}

	return nil
}

// SimpleEventTypeValidator takes messages and validates their Type is of SimpleEventMessageType.
func SimpleEventTypeValidator(m wrp.Message) error {
	if m.Type != wrp.SimpleEventMessageType {
		return ErrorNotSimpleEventType
	}

	return nil
}

// TODO Do we want to include SpanParentValidator? SpanParent currently doesn't exist in the Message Struct

// SpansValidator takes messages and validates their Spans.
func SpansValidator(m wrp.Message) error {
	var err error
	// Spans consist of individual Span(s), arrays of timing values.
	for _, s := range m.Spans {
		if len(s) != len(spanFormat) {
			err = multierr.Append(err, ErrorInvalidSpanLength)
			continue
		}

		for i, j := range spanFormat {
			switch j {
			// Any nonempty string is valid
			case "parent", "name":
				if len(s[i]) == 0 {
					err = multierr.Append(err, fmt.Errorf("%w %v: invalid %v component '%v'", ErrorInvalidSpanFormat, s, j, s[i]))
				}
			// Must be an integer
			case "start time", "duration", "status":
				if _, atoiErr := strconv.Atoi(s[i]); atoiErr != nil {
					err = multierr.Append(err, fmt.Errorf("%w %v: invalid %v component '%v': %v", ErrorInvalidSpanFormat, s, j, s[i], atoiErr))
				}
			}
		}
	}

	return err
}