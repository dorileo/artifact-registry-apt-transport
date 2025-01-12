//  Copyright 2021 Google LLC
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package apt

import (
	"bufio"
	"bytes"
	"context"
	"testing"
)

func TestAptMessageGet(t *testing.T) {
	var tests = []struct {
		message  Message
		expected string
	}{
		{
			// Happy case.
			Message{code: 123, description: "Fake", fields: map[string][]string{"key": {"val1", "val2"}}},
			"val1",
		},
		{
			// Missing key.
			Message{code: 123, description: "Fake", fields: map[string][]string{"some-other-key": {"val1", "val2"}}},
			"",
		},
		{
			// Missing value.
			Message{code: 123, description: "Fake", fields: map[string][]string{"key": {}}},
			"",
		},
	}

	for _, tt := range tests {
		if res := tt.message.Get("key"); res != tt.expected {
			t.Errorf("failed, expected: %q got: %q", tt.expected, res)
		}
	}
}

func TestAptWriterWriteMessage(t *testing.T) {
	var tests = []struct {
		message  Message
		expected string
	}{
		{
			Message{
				code:        123,
				description: "Fake",
				fields: map[string][]string{
					"akey": {"val1", "val2"},
					"zkey": {"val4"},
					"Zkey": {"val3"},
				},
			},
			// Capital letters before lowercase, then alphabetical.
			"123 Fake\nZkey: val3\nakey: val1\nakey: val2\nzkey: val4\n\n",
		},
		{
			Message{
				code: 123,
				// Missing description.
				fields: map[string][]string{
					"akey": {"val1"},
				},
			},
			"123 \nakey: val1\n\n",
		},
		{
			Message{
				// missing code.
				description: "Fake",
				fields: map[string][]string{
					"akey": {"val1"},
				},
			},
			"0 Fake\nakey: val1\n\n",
		},
		{
			Message{
				code:        123,
				description: "Fake",
				// missing fields.
			},
			"123 Fake\n\n",
		},
		{
			Message{
				code:        123,
				description: "Fake",
				fields: map[string][]string{
					// Missing field value.
					"akey": {},
				},
			},
			"123 Fake\n\n",
		},
		{
			Message{
				code:        123,
				description: "Fake",
				fields: map[string][]string{
					"akey": {"val ending with newline\n"},
					"zkey": {"val containing \n\n double newlines"},
				},
			},
			"123 Fake\nakey: val ending with newline \nzkey: val containing    double newlines\n\n",
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		writer := NewAptMessageWriter(&buffer)
		err := writer.WriteMessage(tt.message)
		if err != nil || buffer.String() != tt.expected {
			t.Errorf("failed, expected:\n%q\ngot:\n%q", tt.expected, buffer.String())
		}
	}
}

func TestAptWriterSendCapabilities(t *testing.T) {
	var buffer bytes.Buffer
	writer := NewAptMessageWriter(&buffer)
	expected := "100 Capabilities\nSend-Config: true\nVersion: 1.0\n\n"
	if err := writer.SendCapabilities(); err != nil || buffer.String() != expected {
		t.Errorf("failed, expected:\n%q\ngot:\n%q", expected, buffer.String())
	}
}

func TestAptWriterLog(t *testing.T) {
	var tests = []struct {
		msg, expected string
	}{
		{
			"some log message",
			"101 Log\nMessage: some log message\n\n",
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		writer := NewAptMessageWriter(&buffer)
		if err := writer.Log(tt.msg); err != nil || buffer.String() != tt.expected {
			t.Errorf("failed, expected:\n%q\ngot:\n%q", tt.expected, buffer.String())
		}
	}
}
func TestAptWriterURIStart(t *testing.T) {
	var tests = []struct {
		uri, size, lastModified, expected string
	}{
		{
			"http://fake.uri/debian/",
			"419304",
			"Mon, 01 Mar 2021 03:05:06 GMT",
			"200 URI Start\nLast-Modified: Mon, 01 Mar 2021 03:05:06 GMT\nResume-Point: 0\nSize: 419304\nURI: http://fake.uri/debian/\n\n",
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		writer := NewAptMessageWriter(&buffer)
		if err := writer.URIStart(tt.uri, tt.size, tt.lastModified); err != nil || buffer.String() != tt.expected {
			t.Errorf("failed, expected:\n%q\ngot:\n%q", tt.expected, buffer.String())
		}
	}
}

// func URIDone(uri, size, lastModified, md5Hash, filename string, ims bool)
func TestAptWriterURIDone(t *testing.T) {
	var tests = []struct {
		uri, size, lastModified, md5Hash, filename, expected string
		ims                                                  bool
	}{
		{
			"http://fake.uri/debian/",
			"419304",
			"Mon, 01 Mar 2021 03:05:06 GMT",
			"ABCDEFGHIJKL",
			"/some/local/filename",
			"201 URI Done\nFilename: /some/local/filename\nLast-Modified: Mon, 01 Mar 2021 03:05:06 GMT\nMD5-Hash: ABCDEFGHIJKL\nSize: 419304\nURI: http://fake.uri/debian/\n\n",
			false,
		},
		{
			"http://fake.uri/debian/",
			"419304",
			"Mon, 01 Mar 2021 03:05:06 GMT",
			"ABCDEFGHIJKL",
			"/some/local/filename",
			"201 URI Done\nFilename: /some/local/filename\nIMS-Hit: true\nLast-Modified: Mon, 01 Mar 2021 03:05:06 GMT\nURI: http://fake.uri/debian/\n\n",
			true,
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		writer := NewAptMessageWriter(&buffer)
		if err := writer.URIDone(tt.uri, tt.size, tt.lastModified, tt.md5Hash, tt.filename, tt.ims); err != nil || buffer.String() != tt.expected {
			t.Errorf("failed, expected:\n%q\ngot:\n%q", tt.expected, buffer.String())
		}
	}
}

func TestAptWriterFailURI(t *testing.T) {
	var tests = []struct {
		uri, msg, expected string
	}{
		{
			"http://fake.uri/debian/",
			"uri failure message",
			"400 URI Failure\nMessage: uri failure message\nURI: http://fake.uri/debian/\n\n",
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		writer := NewAptMessageWriter(&buffer)
		if err := writer.FailURI(tt.uri, tt.msg); err != nil || buffer.String() != tt.expected {
			t.Errorf("failed, expected:\n%q\ngot:\n%q", tt.expected, buffer.String())
		}
	}
}

func TestAptWriterFail(t *testing.T) {
	var tests = []struct {
		msg, expected string
	}{
		{
			"general failure message",
			"401 General Failure\nMessage: general failure message\n\n",
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		writer := NewAptMessageWriter(&buffer)
		if err := writer.Fail(tt.msg); err != nil || buffer.String() != tt.expected {
			t.Errorf("failed, expected:\n%q\ngot:\n%q", tt.expected, buffer.String())
		}
	}
}

func compareFields(first, second map[string][]string) bool {
	if len(first) != len(second) {
		return false
	}
	for key, firstVals := range first {
		secondVals, ok := second[key]
		if !ok || len(firstVals) != len(secondVals) {
			return false
		}
		for idx, firstVal := range firstVals {
			if firstVal != secondVals[idx] {
				return false
			}
		}
	}
	return true
}

func TestAptReaderReadMessage(t *testing.T) {
	var tests = []struct {
		msg      string
		expected Message
	}{
		{
			"123 Fake Header\nField1: val1\nField2: val2\nField2: val3\nField1: val4\n\n",
			Message{
				code:        123,
				description: "Fake Header",
				fields: map[string][]string{
					"Field1": {"val1", "val4"},
					"Field2": {"val2", "val3"},
				},
			},
		},
	}

	for _, tt := range tests {
		var buffer bytes.Buffer
		buffer.WriteString(tt.msg)
		reader := NewAptMessageReader(bufio.NewReader(&buffer))
		res, err := reader.ReadMessage(context.Background())
		if err != nil {
			t.Errorf("failed: %v", err)
		}
		if res.code != tt.expected.code ||
			res.description != tt.expected.description ||
			!compareFields(res.fields, tt.expected.fields) {
			t.Errorf("failed, expected: %v got: %v", tt.expected, res)
		}
	}
}

func TestAptReaderParseHeader(t *testing.T) {
	var tests = []struct {
		message  Message
		header   string
		expected Message
	}{
		{
			Message{},
			"123 Fake Code",
			Message{code: 123, description: "Fake Code"},
		},
	}

	for _, tt := range tests {
		reader := MessageReader{message: &Message{}}
		if err := reader.parseHeader(tt.header); err != nil {
			t.Errorf("failed, %v", err)
		}
		if reader.message.code != tt.expected.code ||
			reader.message.description != tt.expected.description {
			t.Errorf("failed, expected: %v got: %v", tt.expected, reader.message)
		}
	}
}

func TestAptReaderParseHeaderFail(t *testing.T) {
	var tests = []struct {
		message Message
		header  string
	}{
		{
			Message{},
			"123FakeCode", // Invalid format.
		},
		{
			Message{},
			"Xxx Fake Code", // Invalid format.
		},
		{
			Message{},
			"", // Empty header.
		},
		{
			Message{code: 123},
			"600 URI Acquire", // Valid header, existing message.
		},
		{
			Message{description: "blah"},
			"600 URI Acquire", // Valid header, existing message.
		},
	}

	for idx, tt := range tests {
		reader := MessageReader{message: &tt.message}
		if err := reader.parseHeader(tt.header); err == nil {
			t.Errorf("validation failed test %d", idx)
		}
	}
}

func TestAptReaderParseField(t *testing.T) {
	var tests = []struct {
		message  Message
		field    string
		expected map[string][]string
	}{
		{
			// Test initial fields.
			Message{},
			"Field1: val1",
			map[string][]string{"Field1": {"val1"}},
		},
		{
			// Test appending fields.
			Message{fields: map[string][]string{"Field1": {"val1"}}},
			"Field1: val2",
			map[string][]string{"Field1": {"val1", "val2"}},
		},
	}

	for _, tt := range tests {
		reader := MessageReader{message: &tt.message}
		if err := reader.parseField(tt.field); err != nil {
			t.Errorf("failed, %v", err)
		}
		if !compareFields(tt.message.fields, tt.expected) {
			t.Errorf("failed, expected: %v got %v", tt.expected, tt.message.fields)
		}
	}
}

func TestAptReaderParseFieldFail(t *testing.T) {
	var tests = []struct {
		field string
	}{
		{
			"Field1 = val1",
		},
		{
			"",
		},
		{
			":val1",
		},
		{
			" : val1",
		},
	}

	for idx, tt := range tests {
		reader := MessageReader{message: &Message{}}
		if err := reader.parseField(tt.field); err == nil {
			t.Errorf("validation failed test %d", idx)
		}
	}
}
