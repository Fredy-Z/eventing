// +build e2e

/*
Copyright 2019 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/knative/eventing/pkg/apis/eventing/v1alpha1"

	"github.com/knative/eventing/test"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// This test annotates the testing namespace so that a default broker is created.
// It then binds one trigger with default filters to that default broker.
// The trigger sends the received events to a subscriber, which replies the events directly back.
// Then it sends one single event to the broker's address.
// Finally, it verifies that the subscriber receives exactly 10 such events.
func TestBrokerTriggerWithLoop(t *testing.T) {
	const (
		maxDuplicateEventCount = 10

		defaultBrokerName   = "default"
		senderName          = "end2end-test-sender"
		triggerName         = "end2end-test-trigger"
		subscriberPodName   = "end2end-test-subscriber"
		subscriberRouteName = "end2end-test-router"
		selectorKey         = "end2end-test-broker-trigger-loop"

		any         = v1alpha1.TriggerAnyFilter
		eventType   = "type"
		eventSource = "source"
	)

	clients, cleaner := Setup(t, t.Logf)

	// Verify namespace exists.
	ns, cleanupNS := CreateNamespaceIfNeeded(t, clients, t.Logf)

	defer cleanupNS()
	defer TearDown(clients, cleaner, t.Logf)

	t.Logf("Labeling namespace %s", ns)
	// Label namespace so that it creates the default broker.
	if err := LabelNamespace(clients, t.Logf, map[string]string{"knative-eventing-injection": "enabled"}); err != nil {
		t.Fatalf("Error annotating namespace: %v", err)
	}
	t.Logf("Namespace %s annotated", ns)

	// Wait for default broker ready.
	t.Logf("Waiting for default broker to be ready")
	defaultBroker := test.Broker(defaultBrokerName, ns)
	if err := WaitForBrokerReady(clients, defaultBroker); err != nil {
		t.Fatalf("Error waiting for default broker to become ready: %v", err)
	}
	defaultBrokerUrl := fmt.Sprintf("http://%s", defaultBroker.Status.Address.Hostname)
	t.Logf("Default broker ready: %q", defaultBrokerUrl)

	// Create Subscriber pod and wait for it to become running
	t.Logf("Creating Subscriber pod that does event transformation")
	selector := map[string]string{"e2etest": string(uuid.NewUUID())}
	subscriberPod := test.EventTransformationPod(subscriberPodName, ns, selector, "")
	subscriberPod, err := CreatePodAndServiceReady(clients, subscriberPod, subscriberRouteName, ns, selector, t.Logf, cleaner)
	if err != nil {
		t.Fatalf("Failed to create subscriber pod and service, and get them ready: %v", err)
	}
	t.Logf("Subscriber service created")

	// Create Trigger and wait for it to become ready
	t.Logf("Creating Trigger")
	trigger := test.NewTriggerBuilder(triggerName, ns).
		EventType(any).
		EventSource(any).
		// Don't need to set the broker as we use the default one
		// but wanted to be more explicit.
		Broker(defaultBrokerName).
		SubscriberSvc(subscriberRouteName).
		Build()
	if err := CreateTrigger(clients, trigger, t.Logf, cleaner); err != nil {
		t.Fatalf("Error creating trigger: %v", err)
	}
	t.Logf("Triggers created")

	t.Logf("Waiting for triggers to become ready")
	// Wait for all of the triggers in the namespace to be ready.
	if err := WaitForAllTriggersReady(clients, t.Logf, ns); err != nil {
		t.Fatalf("Error waiting for triggers to become ready: %v", err)
	}
	t.Logf("Triggers ready")

	// We notice some crashLoopBacks in the filter and ingress pod creation.
	// We then delay the creation of the sender pods in order not to miss events.
	// TODO improve this
	t.Logf("Waiting for filter and ingress pods to become running")
	time.Sleep(waitForFilterPodRunning)

	t.Logf("Creating event sender pod")
	// send fake CloudEvent to the Broker
	body := fmt.Sprintf("TestBrokerTriggerLoop %s", uuid.NewUUID())
	if err := SendFakeEventToAddressable(clients, senderName, body, test.CloudEventDefaultType, test.CloudEventDefaultEncoding, defaultBrokerUrl, ns, t.Logf, cleaner); err != nil {
		t.Fatal("Failed to send fake CloudEvent to the Broker: %v", err)
	}

	// check if the logging service receives the correct number of event messages
	if err := WaitForLogContentCount(clients, subscriberPodName, subscriberPod.Spec.Containers[0].Name, body, maxDuplicateEventCount); err != nil {
		t.Fatalf("String %q not found in logs of subscriber pod %q: %v", body, subscriberPodName, err)
	}
}
