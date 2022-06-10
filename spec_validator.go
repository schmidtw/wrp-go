/**
 *  Copyright (c) 2022  Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package wrp

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	serialPrefix = "serial"
	uuidPrefix   = "uuid"
	eventPrefix  = "event"
	dnsPrefix    = "dns"
)

var (
	ErrorInvalidMessageEncoding = errors.New("invalid message encoding")
	ErrorInvalidMessageType     = errors.New("invalid message type")
	ErrorInvalidSource          = errors.New("invalid Source name")
	ErrorInvalidDestination     = errors.New("invalid Destination name")
	errInvalidUUID              = errors.New("invalid UUID")
	errEmptyAuthority           = errors.New("invalid empty authority (ID)")
	errInvalidMacLength         = errors.New("invalid mac length")
	errInvalidCharacter         = errors.New("invalid character")
	errInvalidLocatorPattern    = errors.New("value given doesn't match expected locator pattern")
)

// locatorPattern is the precompiled regular expression that all source and dest locators must match.
// Matching is partial, as everything after the authority (ID) is ignored. https://xmidt.io/docs/wrp/basics/#locators
var locatorPattern = regexp.MustCompile(
	`^(?P<scheme>(?i)` + macPrefix + `|` + uuidPrefix + `|` + eventPrefix + `|` + dnsPrefix + `|` + serialPrefix + `):(?P<authority>[^/]+)?`,
)

// SpecValidators returns a WRP validator that ensures messages are valid based on
// each spec validator in the list. Only validates the opinionated portions of the spec.
// SpecValidators validates the following fields: UTF8 (all string fields), MessageType, Source, Destination
func SpecValidators() Validators {
	return Validators{UTF8Validator(), MessageTypeValidator(), SourceValidator(), DestinationValidator()}
}

// UTF8Validator returns a WRP validator that takes messages and validates that it contains UTF-8 strings.
func UTF8Validator() ValidatorFunc {
	return func(m Message) error {
		if err := UTF8(m); err != nil {
			return fmt.Errorf("%w: %v", ErrorInvalidMessageEncoding, err)
		}

		return nil
	}
}

// MessageTypeValidator returns a WRP validator that takes messages and validates their Type.
func MessageTypeValidator() ValidatorFunc {
	return func(m Message) error {
		if m.Type < Invalid0MessageType || m.Type > lastMessageType {
			return ErrorInvalidMessageType
		}

		switch m.Type {
		case Invalid0MessageType, Invalid1MessageType, lastMessageType:
			return ErrorInvalidMessageType
		}

		return nil
	}
}

// SourceValidator returns a WRP validator that takes messages and validates their Source.
// Only mac and uuid sources are validated. Serial, event and dns sources are
// not validated.
func SourceValidator() ValidatorFunc {
	return func(m Message) error {
		if err := validateLocator(m.Source); err != nil {
			return fmt.Errorf("%w '%s': %v", ErrorInvalidSource, m.Source, err)
		}

		return nil
	}
}

// DestinationValidator returns a WRP validator that takes messages and validates their Destination.
// Only mac and uuid destinations are validated. Serial, event and dns destinations are
// not validated.
func DestinationValidator() ValidatorFunc {
	return func(m Message) error {
		if err := validateLocator(m.Destination); err != nil {
			return fmt.Errorf("%w '%s': %v", ErrorInvalidDestination, m.Destination, err)
		}

		return nil
	}
}

// validateLocator validates a given locator's scheme and authority (ID).
// Only mac and uuid schemes' IDs are validated. IDs from serial, event and dns schemes are
// not validated.
func validateLocator(l string) error {
	match := locatorPattern.FindStringSubmatch(l)
	if match == nil {
		return errInvalidLocatorPattern
	}

	idPart := match[2]
	if len(idPart) == 0 {
		return errEmptyAuthority
	}

	switch strings.ToLower(match[1]) {
	case macPrefix:
		var invalidCharacter rune = -1
		idPart = strings.Map(
			func(r rune) rune {
				switch {
				case strings.ContainsRune(hexDigits, r):
					return unicode.ToLower(r)
				case strings.ContainsRune(macDelimiters, r):
					return -1
				default:
					invalidCharacter = r
					return -1
				}
			},
			idPart,
		)

		if invalidCharacter != -1 {
			return fmt.Errorf("%w: %v", errInvalidCharacter, strconv.QuoteRune(invalidCharacter))
		} else if len(idPart) != macLength {
			return errInvalidMacLength
		}
	case uuidPrefix:
		if _, err := uuid.Parse(idPart); err != nil {
			return fmt.Errorf("%w: %v", errInvalidUUID, err)
		}
	}

	return nil
}
