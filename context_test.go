package pipeline

import "testing"

func TestContextTotal(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *Context
		pipeline *Pipeline
		value    int
		expected int
	}{
		{"nil pipeline", nil, nil, 0, 0},
		{"valid pipeline", nil, NewPipeline("pipeline"), 100, 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Context{test.ctx, test.pipeline}
			c.Total(test.value)
			if test.pipeline != nil && test.expected != test.pipeline.unitTotal {
				t.Errorf("unit total not set correctly, expected: %d actual: %d", test.expected, test.pipeline.unitTotal)
			}
		})
	}
}

func TestContextInc(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *Context
		pipeline *Pipeline
		value    int
		expected int
	}{
		{"nil pipeline", nil, nil, 0, 0},
		{"valid pipeline zero val", nil, NewPipeline("pipeline"), 0, 0},
		{"valid pipeline", nil, NewPipeline("pipeline"), 100, 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &Context{test.ctx, test.pipeline}
			c.Total(test.value)
			for i := 0; i < test.value; i++ {
				c.Inc()
			}
			if test.pipeline != nil && test.expected != test.pipeline.unitCount {
				t.Errorf("unit total not set correctly, expected: %d actual: %d", test.expected, test.pipeline.unitCount)
			}
		})
	}
}
