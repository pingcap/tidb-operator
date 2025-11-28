package volumes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestHasVAC(t *testing.T) {
	testcases := []struct {
		volumes  []DesiredVolume
		expected bool
	}{
		{
			volumes:  []DesiredVolume{},
			expected: false,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: nil,
				},
			},
			expected: false,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: pointer.String(""),
				},
			},
			expected: false,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			expected: true,
		},
		{
			volumes: []DesiredVolume{
				{
					VolumeAttributesClassName: nil,
				},
				{
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
				{
					VolumeAttributesClassName: pointer.String(""),
				},
			},
			expected: true,
		},
	}

	for _, tt := range testcases {
		actual := hasVAC(tt.volumes)
		assert.Equal(t, tt.expected, actual)
	}
}
