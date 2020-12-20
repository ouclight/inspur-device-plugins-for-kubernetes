// Copyright 2019 Intel Corporation. All Rights Reserved.
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

package bitstream

import (
	"testing"
)

func TestGetFPGABitstream(t *testing.T) {
	var fpgaBitStreamDir = "testdata/intel.com/fpga"

	tcases := []struct {
		name         string
		bitstreamDir string
		region       string
		afu          string
		expectedErr  bool
	}{
		{
			name:         "Correct OPAE bitstream file",
			bitstreamDir: fpgaBitStreamDir,
			region:       "69528db6eb31577a8c3668f9faa081f6",
			afu:          "d8424dc4a4a3c413f89e433683f9040b",
			expectedErr:  false,
		},
		{
			name:         "Get broken OPAE bistream file",
			bitstreamDir: fpgaBitStreamDir,
			region:       "ce48969398f05f33946d560708be108a",
			afu:          "f7df405cbd7acf7222f144b0b93acd18",
			expectedErr:  true,
		},
		{
			name:         "Get broken OpenCL bistream file",
			bitstreamDir: fpgaBitStreamDir,
			region:       "ce48969398f05f33946d560708be108a",
			afu:          "d7724dc4a4a3c413f89e433683f9040b",
			expectedErr:  true,
		},
		{
			name:         "Bitstream not found",
			bitstreamDir: fpgaBitStreamDir,
			region:       "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			afu:          "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
			expectedErr:  true,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetFPGABitstream(tc.bitstreamDir, tc.region, tc.afu)
			if err != nil {
				if !tc.expectedErr {
					t.Errorf("unexpected error occurred: %+v", err)
				}
				return
			}
		})
	}
}

func TestOpen(t *testing.T) {
	tcases := []struct {
		name          string
		fname         string
		expectedError bool
	}{
		{
			name:  "correct GBS",
			fname: "testdata/intel.com/fpga/69528db6eb31577a8c3668f9faa081f6/d8424dc4a4a3c413f89e433683f9040b.gbs",
		},
		{
			name:          "Unsupported file format",
			fname:         "test.unsupported",
			expectedError: true,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Open(tc.fname)
			if tc.expectedError && err == nil {
				t.Error("unexpected success")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error: %+v", err)
			}
		})
	}
}
