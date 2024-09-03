package converters

import (
	"testing"
)

func TestUnsafeStringToBytes(t *testing.T) {
	want := "hello world"
	got := string(UnsafeStringToBytes("hello world"))
	if string(got) != string(want) {
		t.Errorf("error, should be %s, but got %s", want, got)
	}
}

func TestUnsafeBytesToString(t *testing.T) {
	want := "hello world"
	got := UnsafeBytesToString([]byte("hello world"))
	if string(got) != string(want) {
		t.Errorf("error, should be %s, but got %s", want, got)
	}
}

func TestUnsafeUTF16BytesToString(t *testing.T) {
	want := "hello"
	got := UnsafeUTF16BytesToString([]byte{
		0x00,
		'h',
		'e',
		'l',
		'l',
		'o',
		0x00})
	if string(got) != string(want) {
		t.Errorf("error, should be " + want + ", but got " + got)
	}
}
