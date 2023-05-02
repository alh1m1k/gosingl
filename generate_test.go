package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"testing"
)

//generate gosingl test file use with caution
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/interfaceType interfaceType
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/mapType mapType
//go:generate ./gosingl -w --variable cbInstance --comment "random comment" github.com/alh1m1k/gosingl/test/callbackType CallbackType
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/arraySliceType _arraySlice_
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/empty empty
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/composition composition
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/split split
//go:generate ./gosingl -w --suffix "_1_singleton.go" --deep 1 github.com/alh1m1k/gosingl/test/deep deep
//go:generate ./gosingl -w --suffix "_2_singleton.go" --deep 2 github.com/alh1m1k/gosingl/test/deep deep
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/deep deep
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/mapTarget mapTarget
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/arrayTarget arrayTarget
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/sliceTarget sliceTarget
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/interfaceValidation interfaceValidationValid1
//go:generate ./gosingl -w github.com/alh1m1k/gosingl/test/interfaceValidation interfaceValidationInvalid1
//go:generate  ./gosingl -w --variable "g[int, bool, *os.File]" github.com/alh1m1k/gosingl/test/generics generics

func TestInterface(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/interfaceType",
		Target:   "interfaceType",
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/interfaceType/interfaceType_singleton.go")
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestMap(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/mapType",
		Target:   "mapType",
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/mapType/mapType_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestCallback(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/callbackType",
		Target:   "CallbackType", //public Structure //value Structure(no ref)
		Variable: "cbInstance",
		Comment:  "random comment",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/callbackType/callbackType_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestSlice(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/arraySliceType",
		Target:   "_arraySlice_",
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/arraySliceType/_arraySlice__singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestEmpty(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/empty",
		Target:   "empty", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/empty/empty_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestComposition(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/composition",
		Target:   "composition", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/composition/composition_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestSplit(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/split",
		Target:   "split", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/split/split_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestDeep1(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/deep",
		Target:   "deep", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     1,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/deep/deep_1_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestDeep2(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/deep",
		Target:   "deep", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     2,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/deep/deep_2_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestDeepInf(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/deep",
		Target:   "deep", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/deep/deep_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestMapTarget(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/mapTarget",
		Target:   "mapTarget", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/mapTarget/mapTarget_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestArrayTarget(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/arrayTarget",
		Target:   "arrayTarget", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/arrayTarget/arrayTarget_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestSliceTarget(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/sliceTarget",
		Target:   "sliceTarget", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/sliceTarget/sliceTarget_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestInterfaceValidation1(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/interfaceValidation",
		Target:   "interfaceValidationInvalid1", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/interfaceValidation/interfaceValidationInvalid1_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func TestInterfaceValidation2(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Package:  "github.com/alh1m1k/gosingl/test/interfaceValidation",
		Target:   "interfaceValidationValid1", //public Structure //value Structure(no ref)
		Variable: "Instance",
		Comment:  "Code generated by <git repo>. DO NOT EDIT.",
		Write:    true,
		Deep:     0,
	}

	b := &bytes.Buffer{}
	if err := ParsePackage(context.WithValue(ctx, "writer", b), cfg); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile("./test/interfaceValidation/interfaceValidationValid1_singleton.go")
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != string(result) {
		t.Fatalf("result is not the same as expected : %s\n %s", diff(b.String(), string(result), 10), b.String())
	}

}

func diff(generated, reference string, offset int) string {
	log.Println("str1:", len(generated), "str2:", len(reference))
	for i := 0; i < len(generated); i++ {
		if len(reference) <= i {
			continue
		}
		if generated[i] != reference[i] {
			offset = min[int](offset, len(generated)-i)
			offset = min[int](offset, len(reference)-i)
			return fmt.Sprintf("symbol %d: gen: \"%s\" ref: \"%s\"", i, generated[i:i+offset], reference[i:i+offset])
		}
	}
	return ""
}

func min[T int](s ...T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if m > v {
			m = v
		}
	}
	return m
}
