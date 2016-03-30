# ECSDog

ECSDog is a standalone Go application that scrapes metrics and events from ECS, and sends them to statsd.

## Usage

The recommended usage is to run it with Docker (e.g. in an ECS service):

```
$ docker run remind101/ecsdog -cluster $CLUSTER -statsd "statsd:8125"
```

## Metrics

```
aws.ecs.services{cluster}
aws.ecs.service.deployments{cluster,service}
aws.ecs.service.desired{cluster,service}
aws.ecs.service.pending{cluster,service}
aws.ecs.service.running{cluster,service}
aws.ecs.service.deployments{cluster,service}
aws.ecs.service.deployment.desired{cluster,service,deployment}
aws.ecs.service.deployment.pending{cluster,service,deployment}
aws.ecs.service.deployment.running{cluster,service,deployment}
```

## Events

ECSDog will scrape all the [ServiceEvents](http://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_ServiceEvent.html) and put them in the DataDog stream so you can monitor and alert on ECS issues like scheduling problems and unhealthy instances.

ECSDog will make a best effort to de-dup events with the same ID, but you may get duplicates across daemon restarts.
