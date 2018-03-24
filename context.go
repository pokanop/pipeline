package pipeline

import (
	"context"
)

// Context wrapper for pipelines
type Context struct {
	context.Context
	pipeline *Pipeline
}

// Total sets the unit total for the amount of work to expect
func (c *Context) Total(value int) {
	if c.pipeline != nil {
		c.pipeline.Total(value)
	}
}

// Inc increments the unit count and sends progress updates to listeners
func (c *Context) Inc() {
	if c.pipeline != nil {
		c.pipeline.Inc()
	}
}
