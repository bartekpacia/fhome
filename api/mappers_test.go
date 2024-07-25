package api

import "testing"

func TestMapLighting(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  string
	}{
		{
			name:  "Min value",
			value: 0,
			want:  "0x6000",
		},
		{
			name:  "Max value",
			value: 100,
			want:  "0x6064",
		},

		{
			name:  "Some value",
			value: 50,
			want:  "0x6032",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapLighting(tt.value)
			if got != tt.want {
				t.Errorf("MapLighting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemapLighting(t *testing.T) {
	tests := []struct {
		name  string
		want  int
		value string
	}{
		{
			name:  "Min value",
			value: "0x6000",
			want:  0,
		},
		{
			name:  "Max value",
			value: "0x6064",
			want:  100,
		},

		{
			name:  "Some value",
			value: "0x6032",
			want:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RemapLighting(tt.value)
			if err != nil {
				t.Errorf("RemapLighting() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("RemapLighting() = %v, want %v", got, tt.want)
			}
		})
	}
}
