package pipeline

import "testing"

func TestStatusString(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"invalid status", Status(100), ""},
		{"pipeline started", StatusPipelineStarted, "pipeline started"},
		{"pipeline finished", StatusPipelineFinished, "pipeline finished"},
		{"stage started", StatusStageStarted, "stage started"},
		{"stage finished", StatusStageFinished, "stage finished"},
		{"step started", StatusStepStarted, "step started"},
		{"step finished", StatusStepFinished, "step finished"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := test.status.String(); actual != test.expected {
				t.Errorf("incorrect status string, expected: %s actual: %s", test.expected, actual)
			}
		})
	}
}
