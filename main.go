package main

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/gambol99/go-marathon"
	"github.com/mrkm4ntr/marathon-lb-ddns/store"
	"github.com/mrkm4ntr/marathon-lb-ddns/store/file"
	"log"
	"net/url"
	"regexp"
	"strings"
	"net/http"
)

var (
	marathonUrl = flag.String("m", "http://marathon.mesos:8080", "URL of Marathon")
	group       = flag.String("group", "external", "")
	domain      = flag.String("domain", "marathon-lb.example.com.", "")
	zoneId      = flag.String("z", "", "zoneId")
	lbId        = flag.String("lb-id", "/marathon-lb", "Id of marathon-lb")
)

func remove(arr []string, target string) ([]string, bool) {
	for i, s := range arr {
		if target == s {
			return append(arr[:i], arr[i+1:]...), true
		}
	}
	return arr, false
}

func isHAProxyVHost(label string) bool {
	r := regexp.MustCompile(`HAPROXY_[0-9]*_VHOST`)
	return r.MatchString(label)
}

func getLBGroup(args *[]string) string {
	for i, arg := range *args {
		if arg == "--group" {
			return (*args)[i+1]
		}
	}
	return ""
}

func parse(apps []marathon.Application, store store.Store) ([]string, []string, []string) {
	newCNames := []string{}
	ipAddresses := []string{}

	prevCNames, err := store.ListCNames()
	if err != nil {
		log.Fatal(err)
	}

	for _, app := range apps {
		if app.ID == *lbId && getLBGroup(app.Args) == *group {
			for _, task := range app.Tasks {
				ipAddresses = append(ipAddresses, task.IPAddresses[0].IPAddress)
			}
		}
		if (*app.Labels)["HAPROXY_GROUP"] == *group {
			for k, v := range *app.Labels {
				if isHAProxyVHost(k) {
					var removed bool
					prevCNames, removed = remove(prevCNames, v)
					if !removed {
						newCNames = append(newCNames, v)
					}

				}
			}
		}
	}
	return newCNames, prevCNames, ipAddresses
}

func changeDNSRecords(newCNames []string, removedCNames []string, ipAddresses []string) error {
	changes := []*route53.Change{}

	resourceRecordSet := &route53.ResourceRecordSet{
		Name: domain,
		Type: aws.String("A"),
		TTL:  aws.Int64(300),
	}

	resourceRecords := []*route53.ResourceRecord{}
	for _, ipAddress := range ipAddresses {
		v := ipAddress
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{Value: &v})
	}

	resourceRecordSet.SetResourceRecords(resourceRecords)
	changes = append(changes, &route53.Change{
		Action:            aws.String("UPSERT"),
		ResourceRecordSet: resourceRecordSet,
	})

	for _, cName := range newCNames {
		v := cName
		changes = append(changes, &route53.Change{
			Action: aws.String("UPSERT"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: &v,
				Type: aws.String("CNAME"),
				TTL:  aws.Int64(300),
				ResourceRecords: []*route53.ResourceRecord{
					{
						Value: domain,
					},
				},
			},
		})
	}

	for _, cName := range removedCNames {
		v := cName
		changes = append(changes, &route53.Change{
			Action: aws.String("DELETE"),
			ResourceRecordSet: &route53.ResourceRecordSet{
				Name: &v,
				Type: aws.String("CNAME"),
				TTL:  aws.Int64(300),
				ResourceRecords: []*route53.ResourceRecord{
					{
						Value: domain,
					},
				},
			},
		})
	}
	log.Printf("Change request: %v", changes)

	svc := route53.New(session.Must(session.NewSession()))
	o, err := svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: zoneId,
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
	})
	log.Printf("Change response: %v", o)

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: zoneId,
	}
	resp, err := svc.ListResourceRecordSets(params)

	if err != nil {
		log.Fatalf("%s", err)
		return err
	}
	log.Printf("Current record sets: %v", resp.ResourceRecordSets)
	return err
}

func execute(applications *marathon.Applications) {
	store := file.New()
	newCNames, removedCNames, ipAddresses := parse(applications.Apps, store)

	changeDNSRecords(newCNames, removedCNames, ipAddresses)

	for _, cName := range newCNames {
		store.AddCName(cName)
	}
	for _, cName := range removedCNames {
		store.RemoveCName(cName)
	}
}

func main() {
	flag.Parse()

	marathonConfig := marathon.NewDefaultConfig()
	marathonConfig.URL = *marathonUrl
	marathonConfig.EventsTransport = marathon.EventsTransportSSE

	client, err := marathon.NewClient(marathonConfig)
	if err != nil {
		log.Fatalf("Failed to create a client for marathon, error: %s", err)
	}

	store := file.New()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cNames, err := store.ListCNames()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		for _, cName := range cNames {
			w.Write([]byte(cName + "\n"))
		}
	})
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	var values url.Values
	values = url.Values{"embed": []string{"app.tasks"}}

	events, err := client.AddEventsListener(marathon.EventIDApplications)
	if err != nil {
		log.Fatalf("Failed to register for events, %s", err)
	}
	defer client.RemoveEventsListener(events)

	for {
		select {
		case event := <-events:
			log.Printf("Received event: %s", event)
			applications, err := client.Applications(values)
			if err != nil {
				log.Fatal(err)
			}
			execute(applications)
		}
	}
}
