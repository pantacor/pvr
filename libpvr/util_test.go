//
// Copyright 2017-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package libpvr

import (
	"fmt"
)

type testStruct struct {
	Field1 string `json:"field1"`
}

var (
	testMap1 = map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	testStruct1 = testStruct{
		Field1: "Test1234",
	}
)

func ExampleSprintTmplBasic() {
	s, _ := SprintTmpl("Basic", testMap1)
	fmt.Println(s)
	// Output: Basic
}

func ExampleSprintTmplSimple1() {
	s, _ := SprintTmpl("Basic: {{ .key1 }}", testMap1)
	fmt.Println(s)
	// Output: Basic: value1
}

func ExampleSprintTmplSprintf1() {
	s, _ := SprintTmpl("Basic: {{ \"test\" | sprintf \"%.3s\" }}", testMap1)
	fmt.Println(s)
	// Output: Basic: tes
}

func ExampleSprintTmplSprintf2() {
	s, _ := SprintTmpl("Basic: {{ .key1 | sprintf \"%.3s\" }}", testMap1)
	fmt.Println(s)
	// Output: Basic: val
}

func ExampleSprintTmplStruct() {
	s, _ := SprintTmpl("Basic: {{ .Field1 | sprintf \"%.3s\" }}", testStruct1)
	fmt.Println(s)
	// Output: Basic: Tes
}

func ExampleFixupRef1() {
	uristring, _ := FixupRepoRef("192.168.1.3")
	fmt.Println(uristring)
	// Output: http://192.168.1.3:12368/cgi-bin/pvr
}

func ExampleFixupRef2() {
	uristring, _ := FixupRepoRef("192.168.1.3#hash,some")
	fmt.Println(uristring)
	// Output: http://192.168.1.3:12368/cgi-bin/pvr#hash,some
}

func ExampleFixupRef3() {
	uristring, _ := FixupRepoRef("asacasa/test1234")
	fmt.Println(uristring)
	// Output: https://pvr.pantahub.com/asacasa/test1234
}

func ExampleFixupRef4() {
	uristring, _ := FixupRepoRef("asacasa/test1234#something,fun")
	fmt.Println(uristring)
	// Output: https://pvr.pantahub.com/asacasa/test1234#something,fun
}

func ExampleTestIsValidUrl1() {
	if IsValidUrl("asacasa/test") {
		fmt.Println("yes")
	} else {
		fmt.Println("no")
	}
	// Output: no
}

func ExampleTestIsValidUrl2() {
	if IsValidUrl("192.168.1.1") {
		fmt.Println("yes")
	} else {
		fmt.Println("no")
	}
	// Output: no
}

func ExampleTestIsValidUrl3() {
	if IsValidUrl("http://192.168.1.1") {
		fmt.Println("yes")
	} else {
		fmt.Println("no")
	}
	// Output: yes
}
