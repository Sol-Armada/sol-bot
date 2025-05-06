package utils

import (
	"testing"
	"time"
)

func TestStringToDuration(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Duration
		wantErr bool
	}{
		{
			name: "valid duration in days, hours, minutes, and seconds",
			args: args{
				str: "1d2h3m4s",
			},
			want:    1*24*time.Hour + 2*time.Hour + 3*time.Minute + 4*time.Second,
			wantErr: false,
		},
		{
			name: "valid duration by number only",
			args: args{
				str: "60",
			},
			want:    60 * time.Minute,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToDuration(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("StringToDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StringToDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
