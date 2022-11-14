package common

import (
	"context"
	"testing"
)

func TestRateLimiter_Wait(t *testing.T) {
	type fields struct {
		RateLimiter RateLimiter
	}
	type args struct {
		ctx   context.Context
		event Event
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "happy",
			fields: fields{RateLimiter: NewDefaultRateLimiter()},
			args: args{
				ctx:   context.TODO(),
				event: ListServices,
			},
			wantErr: false,
		},
		{
			name:   "error_ctx_canceled",
			fields: fields{RateLimiter: NewDefaultRateLimiter()},
			args: args{
				ctx:   ctxCanceled(context.TODO()),
				event: ListNamespaces,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.RateLimiter
			if err := r.Wait(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func ctxCanceled(ctx context.Context) context.Context {
	ret, cancel := context.WithCancel(ctx)
	defer cancel() // cancel after function call
	return ret
}
