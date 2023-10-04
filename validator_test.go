package main

import (
	"math/rand"
	"testing"
	"time"
)

// from https://www.calhoun.io/creating-random-strings-in-go/
func randomStr(charset string, length int) string {
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	o := make([]byte, length)
	for i := range o {
		o[i] = charset[seed.Intn(len(charset))]
	}
	return string(o)
}

func generatePositiveInt(minValue uint, maxValue uint) int {
	if minValue == 0 {
		return 0
	}
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	o := seed.Intn(int(maxValue))
	if o < int(minValue) {
		return int(minValue)
	}
	return o
}

func TestValidateObjectID(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "shall be valid",
			args: args{
				randomStr("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", generatePositiveInt(1, 20)),
			},
			wantErr: false,
		},
		{
			name: "shall be invalid: too long",
			args: args{
				randomStr("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", generatePositiveInt(50, 55)),
			},
			wantErr: true,
		},
		{
			name: "shall be invalid: too short",
			args: args{
				randomStr("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", generatePositiveInt(0, 0)),
			},
			wantErr: true,
		},
		{
			name: "shall be invalid: wrong charset",
			args: args{
				randomStr("!-/$%", generatePositiveInt(1, 20)),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateObjectID(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("ValidateObjectID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
