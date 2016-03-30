package ecsdog

import (
	"fmt"
	golog "log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var log = golog.New(os.Stderr, "", golog.LstdFlags)

const DefaultNamespace = "aws.ecs"

// Scraper scrapes metrics from ecs.
type Scraper struct {
	Cluster string

	ecs    *ecs.ECS
	statsd *statsd.Client

	sync.Mutex
	services []string

	events map[string]bool
}

// Start starts a new Scraper.
func Scrape(cluster, addr string) error {
	ecs := ecs.New(session.New())
	statsd, err := statsd.New(addr)
	if err != nil {
		return err
	}
	defer statsd.Close()

	s := &Scraper{
		Cluster: cluster,
		ecs:     ecs,
		statsd:  statsd,
		events:  make(map[string]bool),
	}

	return s.Start()
}

func (s *Scraper) Start() error {
	if err := s.updateServices(); err != nil {
		return err
	}

	t := time.Tick(time.Second * 20)

	for range t {
		if err := s.Scrape(); err != nil {
			return err
		}
	}

	return nil
}

// Scrape updates the list of known services, then scrapes metrics from them.
func (s *Scraper) Scrape() error {
	s.Lock()
	defer s.Unlock()

	s.gauge("services", float64(len(s.services)), []string{}, 1)

	for _, services := range chunk(s.services) {
		var pServices []*string
		for _, s := range services {
			ss := s
			pServices = append(pServices, &ss)
		}

		resp, err := s.ecs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(s.Cluster),
			Services: pServices,
		})
		if err != nil {
			return err
		}

		for _, failure := range resp.Failures {
			log.Printf("Failed to describe %s: %s\n", *failure.Arn, *failure.Reason)
		}

		for _, service := range resp.Services {
			if err := s.scrape(service); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Scraper) scrape(service *ecs.Service) error {
	log.Printf("Scraping metrics from %s\n", *service.ServiceName)

	tags := []string{fmt.Sprintf("service_name:%s", *service.ServiceName), fmt.Sprintf("service_status:%s", *service.Status)}

	s.gauge("service.desired", float64(*service.DesiredCount), tags, 1)
	s.gauge("service.pending", float64(*service.PendingCount), tags, 1)
	s.gauge("service.running", float64(*service.RunningCount), tags, 1)

	counts := make(map[string]int)
	for _, deployment := range service.Deployments {
		counts[*deployment.Status] += 1
		dtags := append(tags, []string{fmt.Sprintf("deployment:%s", *deployment.Id), fmt.Sprintf("deployment_status:%s", *deployment.Status)}...)
		s.gauge("service.deployment.desired", float64(*deployment.DesiredCount), dtags, 1)
		s.gauge("service.deployment.pending", float64(*deployment.PendingCount), dtags, 1)
		s.gauge("service.deployment.running", float64(*deployment.RunningCount), dtags, 1)
	}

	s.gauge("service.deployments", float64(len(service.Deployments)), tags, 1)
	for status, count := range counts {
		s.gauge(fmt.Sprintf("service.deployments.%s", strings.ToLower(status)), float64(count), tags, 1)
	}

	for _, event := range service.Events {
		if _, ok := s.events[*event.Id]; !ok {
			s.statsd.Event(&statsd.Event{
				Title:          *event.Message,
				Text:           *event.Message,
				Timestamp:      *event.CreatedAt,
				AggregationKey: *service.ServiceName,
				Tags:           append(tags, "ecs"),
			})
			s.events[*event.Id] = true
		}
	}

	return nil
}

func (s *Scraper) updateServices() (err error) {
	s.Lock()
	defer s.Unlock()

	s.services, err = s.fetchServices()
	return err
}

func (s *Scraper) fetchServices() ([]string, error) {
	var services []string

	err := s.ecs.ListServicesPages(&ecs.ListServicesInput{
		Cluster: aws.String(s.Cluster),
	}, func(p *ecs.ListServicesOutput, lastPage bool) bool {
		for _, arn := range p.ServiceArns {
			services = append(services, *arn)
		}
		return true
	})

	return services, err
}

func (s *Scraper) gauge(name string, value float64, tags []string, rate float64) error {
	tags = append(tags, fmt.Sprintf("cluster_name:%s", s.Cluster))
	return s.statsd.Gauge(strings.Join([]string{DefaultNamespace, name}, "."), value, tags, rate)
}

func chunk(arr []string) [][]string {
	lim := 10
	var chunk []string
	chunks := make([][]string, 0, len(arr)/lim+1)
	for len(arr) >= lim {
		chunk, arr = arr[:lim], arr[lim:]
		chunks = append(chunks, chunk)
	}
	if len(arr) > 0 {
		chunks = append(chunks, arr[:len(arr)])
	}
	return chunks
}
