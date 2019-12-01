/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go"

	"knative.dev/eventing/test/base/resources"
)

var (
	filter bool
)

func init() {
	flag.BoolVar(&filter, "filter", false, "Whether to filter the event")
}

func gotEvent(event cloudevents.Event, resp *cloudevents.EventResponse) error {
	ctx := event.Context.AsV03()

	dataBytes, err := event.DataBytes()
	if err != nil {
		log.Printf("Got Data Error: %s\n", err.Error())
		return err
	}
	log.Println("Received a new event: ")
	log.Printf("[%s] %s %s: %s", ctx.Time.String(), ctx.GetSource(), ctx.GetType(), dataBytes)

	if filter {
		log.Println("Filter event")
		resp.Status = 200
	} else {
		event.SetDataContentType(cloudevents.ApplicationJSON)
		log.Println("Reply with event")
		resp.RespondWith(200, &event)
	}
	return nil
}

func main() {
	// parse the command line flags
	flag.Parse()

	c, err := cloudevents.NewDefaultClient()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	go func() {
		log.Printf("start receiver")
		if err := c.StartReceiver(context.Background(), gotEvent); err != nil {
			log.Fatalf("failed to start receiver: %v", err)
		}
	}()

	time.Sleep(10 * time.Second)
	log.Printf("start health check")
	http.HandleFunc(resources.HealthCheckEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	if err := http.ListenAndServe(resources.HealthCheckAddr, nil); err != nil {
		log.Fatalf("failed to start health check endpoint: %v", err)
	}
}
