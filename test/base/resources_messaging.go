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

package base

// resources_messaging contains functions that construct Messaging resources.

import (
	kafkamessagingv1alpha1 "github.com/knative/eventing/contrib/kafka/pkg/apis/messaging/v1alpha1"
	messagingv1alpha1 "github.com/knative/eventing/pkg/apis/messaging/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaChannel returns a KafkaChannel resource.
func KafkaChannel(name string) *kafkamessagingv1alpha1.KafkaChannel {
	return &kafkamessagingv1alpha1.KafkaChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// InMemoryChannel returns an InMemoryChannel resource.
func InMemoryChannel(name string) *messagingv1alpha1.InMemoryChannel {
	return &messagingv1alpha1.InMemoryChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}