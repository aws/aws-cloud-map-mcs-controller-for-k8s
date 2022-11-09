package common

import (
	"errors"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "trueCase",
			args: struct{ err error }{err: NotFoundError("1")},
			want: true,
		},
		{
			name: "falseCase",
			args: struct{ err error }{err: errors.New("test")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.args.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnknown(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "trueCase",
			args: struct{ err error }{err: errors.New("test")},
			want: true,
		},
		{
			name: "falseCase",
			args: struct{ err error }{err: NotFoundError("1")},
			want: false,
		},
		{
			name: "nilCase",
			args: struct{ err error }{err: nil},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnknown(tt.args.err); got != tt.want {
				t.Errorf("IsUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{
			name: "happyCase",
			arg:  "arg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NotFoundError(tt.arg); !IsNotFound(err) {
				t.Errorf("NotFoundError() error = %v, containsErr = %v", err, notFound)
			}
		})
	}
}
