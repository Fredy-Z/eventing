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

package v1alpha1

import (
	"testing"

	eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	"github.com/knative/pkg/apis"

	"github.com/google/go-cmp/cmp"
	duckv1alpha1 "github.com/knative/eventing/pkg/apis/duck/v1alpha1"
	pkgduckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	duckv1beta1 "github.com/knative/pkg/apis/duck/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var pipelineConditionReady = apis.Condition{
	Type:   PipelineConditionReady,
	Status: corev1.ConditionTrue,
}

var pipelineConditionChannelsReady = apis.Condition{
	Type:   PipelineConditionChannelsReady,
	Status: corev1.ConditionTrue,
}

var pipelineConditionSubscriptionsReady = apis.Condition{
	Type:   PipelineConditionSubscriptionsReady,
	Status: corev1.ConditionTrue,
}

func getSubscription(name string, ready bool) *eventingv1alpha1.Subscription {
	s := eventingv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "eventing.knative.dev/v1alpha1",
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "testns",
		},
		Status: eventingv1alpha1.SubscriptionStatus{},
	}
	if ready {
		s.Status.MarkChannelReady()
		s.Status.MarkReferencesResolved()
		s.Status.MarkAddedToChannel()
	} else {
		s.Status.MarkChannelNotReady("testInducedFailure", "Test Induced failure")
		s.Status.MarkReferencesNotResolved("testInducedFailure", "Test Induced failure")
		s.Status.MarkNotAddedToChannel("testInducedfailure", "Test Induced failure")
	}
	return &s
}

func getChannelable(ready bool) *duckv1alpha1.Channelable {
	URL, _ := apis.ParseURL("http://example.com")
	s := duckv1alpha1.Channelable{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "messaging.knative.dev/v1alpha1",
			Kind:       "InMemoryChannel",
		},
		ObjectMeta: metav1.ObjectMeta{},
		Status:     duckv1alpha1.ChannelableStatus{},
	}

	if ready {
		s.Status.Address = &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{URL}, ""}
	}

	return &s
}

func TestPipelineGetCondition(t *testing.T) {
	tests := []struct {
		name      string
		ss        *PipelineStatus
		condQuery apis.ConditionType
		want      *apis.Condition
	}{{
		name: "single condition",
		ss: &PipelineStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{
					pipelineConditionReady,
				},
			},
		},
		condQuery: apis.ConditionReady,
		want:      &pipelineConditionReady,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.ss.GetCondition(test.condQuery)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected condition (-want, +got) = %v", diff)
			}
		})
	}
}

func TestPipelineInitializeConditions(t *testing.T) {
	tests := []struct {
		name string
		ts   *PipelineStatus
		want *PipelineStatus
	}{{
		name: "empty",
		ts:   &PipelineStatus{},
		want: &PipelineStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   PipelineConditionAddressable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionChannelsReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionSubscriptionsReady,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
	}, {
		name: "one false",
		ts: &PipelineStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   PipelineConditionChannelsReady,
					Status: corev1.ConditionFalse,
				}},
			},
		},
		want: &PipelineStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   PipelineConditionAddressable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionChannelsReady,
					Status: corev1.ConditionFalse,
				}, {
					Type:   PipelineConditionReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionSubscriptionsReady,
					Status: corev1.ConditionUnknown,
				}},
			},
		},
	}, {
		name: "one true",
		ts: &PipelineStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   PipelineConditionSubscriptionsReady,
					Status: corev1.ConditionTrue,
				}},
			},
		},
		want: &PipelineStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:   PipelineConditionAddressable,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionChannelsReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionReady,
					Status: corev1.ConditionUnknown,
				}, {
					Type:   PipelineConditionSubscriptionsReady,
					Status: corev1.ConditionTrue,
				}},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.ts.InitializeConditions()
			if diff := cmp.Diff(test.want, test.ts, ignoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}

func TestPipelinePropagateSubscriptionStatuses(t *testing.T) {
	tests := []struct {
		name string
		subs []*eventingv1alpha1.Subscription
		want corev1.ConditionStatus
	}{{
		name: "empty",
		subs: []*eventingv1alpha1.Subscription{},
		want: corev1.ConditionFalse,
	}, {
		name: "empty status",
		subs: []*eventingv1alpha1.Subscription{{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "eventing.knative.dev/v1alpha1",
				Kind:       "Subscription",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sub",
				Namespace: "testns",
			},
			Status: eventingv1alpha1.SubscriptionStatus{},
		},
		},
		want: corev1.ConditionFalse,
	}, {
		name: "one subscription not ready",
		subs: []*eventingv1alpha1.Subscription{getSubscription("sub0", false)},
		want: corev1.ConditionFalse,
	}, {
		name: "one subscription ready",
		subs: []*eventingv1alpha1.Subscription{getSubscription("sub0", true)},
		want: corev1.ConditionTrue,
	}, {
		name: "one subscription ready, one not",
		subs: []*eventingv1alpha1.Subscription{getSubscription("sub0", true), getSubscription("sub1", false)},
		want: corev1.ConditionFalse,
	}, {
		name: "two subscriptions ready",
		subs: []*eventingv1alpha1.Subscription{getSubscription("sub0", true), getSubscription("sub1", true)},
		want: corev1.ConditionTrue,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ps := PipelineStatus{}
			ps.PropagateSubscriptionStatuses(test.subs)
			got := ps.GetCondition(PipelineConditionSubscriptionsReady).Status
			want := test.want
			if want != got {
				t.Errorf("unexpected conditions (-want, +got) = %v %v", want, got)
			}
		})
	}
}

func TestPipelinePropagateChannelStatuses(t *testing.T) {
	tests := []struct {
		name     string
		channels []*duckv1alpha1.Channelable
		want     corev1.ConditionStatus
	}{{
		name:     "empty",
		channels: []*duckv1alpha1.Channelable{},
		want:     corev1.ConditionFalse,
	}, {
		name:     "one channelable not ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(false)},
		want:     corev1.ConditionFalse,
	}, {
		name:     "one channelable ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(true)},
		want:     corev1.ConditionTrue,
	}, {
		name:     "one channelable ready, one not",
		channels: []*duckv1alpha1.Channelable{getChannelable(true), getChannelable(false)},
		want:     corev1.ConditionFalse,
	}, {
		name:     "two channelables ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(true), getChannelable(true)},
		want:     corev1.ConditionTrue,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ps := PipelineStatus{}
			ps.PropagateChannelStatuses(test.channels)
			got := ps.GetCondition(PipelineConditionChannelsReady).Status
			want := test.want
			if want != got {
				t.Errorf("unexpected conditions (-want, +got) = %v %v", want, got)
			}
		})
	}
}

func TestPipelineReady(t *testing.T) {
	tests := []struct {
		name     string
		subs     []*eventingv1alpha1.Subscription
		channels []*duckv1alpha1.Channelable
		want     bool
	}{{
		name:     "empty",
		subs:     []*eventingv1alpha1.Subscription{},
		channels: []*duckv1alpha1.Channelable{},
		want:     false,
	}, {
		name:     "one channelable not ready, one subscription ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(false)},
		subs:     []*eventingv1alpha1.Subscription{getSubscription("sub0", true)},
		want:     false,
	}, {
		name:     "one channelable ready, one subscription not ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(true)},
		subs:     []*eventingv1alpha1.Subscription{getSubscription("sub0", false)},
		want:     false,
	}, {
		name:     "one channelable ready, one subscription ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(true)},
		subs:     []*eventingv1alpha1.Subscription{getSubscription("sub0", true)},
		want:     true,
	}, {
		name:     "one channelable ready, one not, two subsriptions ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(true), getChannelable(false)},
		subs:     []*eventingv1alpha1.Subscription{getSubscription("sub0", true), getSubscription("sub1", true)},
		want:     false,
	}, {
		name:     "two channelables ready, one subscription ready, one not",
		channels: []*duckv1alpha1.Channelable{getChannelable(true), getChannelable(true)},
		subs:     []*eventingv1alpha1.Subscription{getSubscription("sub0", true), getSubscription("sub1", false)},
		want:     false,
	}, {
		name:     "two channelables ready, two subscriptions ready",
		channels: []*duckv1alpha1.Channelable{getChannelable(true), getChannelable(true)},
		subs:     []*eventingv1alpha1.Subscription{getSubscription("sub0", true), getSubscription("sub1", true)},
		want:     true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ps := PipelineStatus{}
			ps.PropagateChannelStatuses(test.channels)
			ps.PropagateSubscriptionStatuses(test.subs)
			got := ps.IsReady()
			want := test.want
			if want != got {
				t.Errorf("unexpected conditions (-want, +got) = %v %v", want, got)
			}
		})
	}
}

func TestPipelinePropagateSetAddress(t *testing.T) {
	URL, _ := apis.ParseURL("http://example.com")
	tests := []struct {
		name       string
		address    *pkgduckv1alpha1.Addressable
		want       *pkgduckv1alpha1.Addressable
		wantStatus corev1.ConditionStatus
	}{{
		name:       "nil",
		address:    nil,
		want:       &pkgduckv1alpha1.Addressable{},
		wantStatus: corev1.ConditionFalse,
	}, {
		name:       "empty",
		address:    &pkgduckv1alpha1.Addressable{},
		want:       &pkgduckv1alpha1.Addressable{},
		wantStatus: corev1.ConditionFalse,
	}, {
		name:       "URL",
		address:    &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{URL}, ""},
		want:       &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{URL}, "example.com"},
		wantStatus: corev1.ConditionTrue,
	}, {
		name:       "hostname",
		address:    &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{}, "myhostname"},
		want:       &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{}, "myhostname"},
		wantStatus: corev1.ConditionTrue,
	}, {
		name:       "nil",
		address:    &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{nil}, ""},
		want:       &pkgduckv1alpha1.Addressable{duckv1beta1.Addressable{}, ""},
		wantStatus: corev1.ConditionFalse,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ps := PipelineStatus{}
			ps.setAddress(test.address)
			got := ps.Address
			if diff := cmp.Diff(test.want, &got, ignoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected address (-want, +got) = %v", diff)
			}
			gotStatus := ps.GetCondition(PipelineConditionAddressable).Status
			if test.wantStatus != gotStatus {
				t.Errorf("unexpected conditions (-want, +got) = %v %v", test.wantStatus, gotStatus)
			}
		})
	}
}
