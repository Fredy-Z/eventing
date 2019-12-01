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
	eventMsgAppender string
)

func init() {
	flag.StringVar(&eventMsgAppender, "msg-appender", "", "a string we want to append on the event message")
}

func gotEvent(event cloudevents.Event, resp *cloudevents.EventResponse) error {
	ctx := event.Context.AsV03()

	data := &resources.CloudEventBaseData{}
	if err := event.DataAs(data); err != nil {
		log.Printf("Got Data Error: %s\n", err.Error())
		return err
	}

	log.Println("Received a new event: ")
	log.Printf("[%s] %s %s: %+v", ctx.Time.String(), ctx.GetSource(), ctx.GetType(), data)

	// append eventMsgAppender to message of the data
	data.Message = data.Message + eventMsgAppender
	r := cloudevents.Event{
		Context: ctx,
		Data:    data,
	}

	r.SetDataContentType(cloudevents.ApplicationJSON)

	log.Println("Transform the event to: ")
	log.Printf("[%s] %s %s: %+v", ctx.Time.String(), ctx.GetSource(), ctx.GetType(), data)

	resp.RespondWith(200, &r)
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
