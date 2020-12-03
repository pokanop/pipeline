[![Build Status](https://travis-ci.org/pokanop/pipeline.svg?branch=master)](https://travis-ci.org/pokanop/pipeline)
[![GoDoc](https://godoc.org/github.com/pokanop/pipeline?status.svg)](https://godoc.org/github.com/pokanop/pipeline)
[![Go Report Card](https://goreportcard.com/badge/github.com/pokanop/pipeline)](https://goreportcard.com/report/github.com/pokanop/pipeline)
[![Coverage Status](https://coveralls.io/repos/github/pokanop/pipeline/badge.svg?branch=master)](https://coveralls.io/github/pokanop/pipeline?branch=master)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/containous/traefik/blob/master/LICENSE.md)

# pipeline

pipeline is a package that simplifies creating staged pipelines in `go`.

![Pipeline](/images/pipeline.png)

It provides a simple API to construct **stages** and **steps** to execute complex tasks. The packages supports many features including concurrent stages and configurable workers as well as buffered steps.

## A Basic Pipeline

At minimum to create a `Pipeline`, you'll need to create a single `Stage` with a single `Step`.

Every `Step` in the `Pipeline` requires input and provides output via channels.

```go
func main() {
    // Create a step with a function that does the work
    step := pipeline.NewStep("the step", func(ctx *pipeline.Context, in <-chan interface{}, out chan interface{}) error {
        // Take the input
        input := (<-in).(string)

        // Do something with it
        input += " pipeline"

        // Pass it along
        out<-input
        return nil
    })

    // Create the pipeline with one stage that contains the step
    p := pipeline.NewPipeline("the pipeline")
    p.AddStage(pipeline.NewStage("the stage", step))

    // Run the pipeline
    in := make(chan interface{})
    go func() {
        in<-"hello"
    }()
    out := p.Process(nil, in)

    // Get the output
    fmt.Println((<-out).(string))
}
```

## Stages

A stage is a collection of steps that can be run serially or concurrently.

It can be created using `pipeline.NewStage(...)` and by default it will create a new serial stage.

### Serial Stages

A serial stage will run all the steps in the stage serially, or one after another. The output from one step will flow into the input of the next step.

Serial stages can be created with:

```go
stage := pipeline.NewSerialStage(name, step1, ...)
```

### Concurrent Stages

A concurrent stage will run all the steps in the stage concurrently, or together. The input for each step will come from the output of the last stage and all the output of the steps in this stage will be provided to the next stage.

Concurrent stages can be created with:

```go
stage := pipeline.NewConcurrentStage(name, step1, ...)
```

## Steps

A step is a single processing unit of a pipeline. It takes input in the form of an `interface{}` from a channel, does some work on it, and then should provide the output to another channel.

Steps can be configured in various ways and can modify the pipeline to provide flexibility. The kinds of steps are:

- worker
- buffered
- fan out

Steps can be created using `pipeline.NewStep(...)` and by default it will create a new worker step with a worker count of 1.

### Worker Step

A worker step lets you define the number of workers to spawn in go routines to handle this step. This effectively makes the step concurrent by allowing multiple routines process input.

Worker steps can be created with:

```go
step := pipeline.NewWorkerStep(name, workerCount, stepFn)
```

### Buffered Steps

A buffered step creates a step with a buffered output channel allowing data to be buffered while processing. Sending to the output channel doesn't block if the buffer still has space and allows the step to continue reading from the input channel.

Buffered steps can be created with:

```go
step := pipeline.NewBufferedStep(name, stepFn)
```

### Fan Out Steps

A fan out step creates a step that replicates or fans out the input channel across a number of workers. In this model, the worker count indicates how many concurrent steps to replicate the input data on. Note that this will process redundant data streams.

Fan out steps can be created with:

```go
step := pipeline.NewFanOutStep(name, workerCount, stepFn)
```

## Tracking Progress

Progress of the pipeline can be tracked in a few ways:

- elapsed time
- state changes
- alternate progress updates

### Elapsed Time

Calling `ElapsedTime()` returns a `time.Duration` based on the start time of the pipeline.

```go
// Get elapsed time
t := pipeline.ElapsedTime()
fmt.Printf("pipeline has been running for %f seconds\n" + t.Seconds())
```

### State Changes

Pipelines emit a `State` for several changes on a channel. These include a `Status` and progress percentages. Progress is based on the total number of steps in the pipeline.

```go
// Listen to state changes
for {
    select {
        case state := <-pipeline.State():
            fmt.Printf("name: %s status: %s progress: %f", state.Name, state.Status.String(), state.Progress)
        case <-pipeline.Dead()
            return
    }
}
```

### Alternate Progress Updates

Pipelines also provide an alternate approach to measuring progress. In this model, the `pipeline.Context` provided in each `Step` can be used to configure the total and increment units of work.

```go
pipeline.Total(unitsOfWork)

func step(ctx *pipeline.Context, in <-chan interface{}, out chan interface{}) error {
    // Do some work
    ...
    // Update progress markers
    ctx.Inc()
}
```

## Contibuting

Contributions are what makes the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

Distributed under the MIT License.

## If You ♥️ What We Do

<a href="https://www.buymeacoffee.com/pokanopapps" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" height="60" width="217"></a>
