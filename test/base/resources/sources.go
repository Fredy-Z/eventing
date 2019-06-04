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

package resources

// sources contains functions that construct Sources resources.

import (
	sourcesv1alpha1 "github.com/knative/eventing/pkg/apis/sources/v1alpha1"
	pkgTest "github.com/knative/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CronJobSourceOption enables further configuration of a CronJobSource.
type CronJobSourceOption func(*sourcesv1alpha1.CronJobSource)

// ContainerSourceOption enables further configuration of a ContainerSource.
type ContainerSourceOption func(*sourcesv1alpha1.ContainerSource)

// WithSinkServiceForCronJobSource returns an option that adds a Kubernetes Service sink for the given CronJobSource.
func WithSinkServiceForCronJobSource(name string) CronJobSourceOption {
	return func(cjs *sourcesv1alpha1.CronJobSource) {
		cjs.Spec.Sink = pkgTest.CoreV1ObjectReference("Service", "v1", name)
	}
}

// WithServiceAccountForCronJobSource returns an option that adds a ServiceAccount for the given CronJobSource.
func WithServiceAccountForCronJobSource(saName string) CronJobSourceOption {
	return func(cjs *sourcesv1alpha1.CronJobSource) {
		cjs.Spec.ServiceAccountName = saName
	}
}

// CronJobSource returns a CronJob EventSource.
func CronJobSource(
	name,
	schedule,
	data string,
	options ...CronJobSourceOption,
) *sourcesv1alpha1.CronJobSource {
	cronJobSource := &sourcesv1alpha1.CronJobSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: sourcesv1alpha1.CronJobSourceSpec{
			Schedule: schedule,
			Data:     data,
		},
	}
	for _, option := range options {
		option(cronJobSource)
	}
	return cronJobSource
}

// WithArgsForContainerSource returns an option that adds args for the given ContainerSource.
func WithArgsForContainerSource(args []string) ContainerSourceOption {
	return func(cs *sourcesv1alpha1.ContainerSource) {
		cs.Spec.Args = args
	}
}

// WithEnvVarsForContainerSource returns an option that adds environment vars for the given ContainerSource.
func WithEnvVarsForContainerSource(envVars []corev1.EnvVar) ContainerSourceOption {
	return func(cs *sourcesv1alpha1.ContainerSource) {
		cs.Spec.Env = envVars
	}
}

// WithServiceAccountForContainerSource returns an option that adds a ServiceAccount for the given ContainerSource.
func WithServiceAccountForContainerSource(saName string) ContainerSourceOption {
	return func(cs *sourcesv1alpha1.ContainerSource) {
		cs.Spec.ServiceAccountName = saName
	}
}

// WithSinkServiceForContainerSource returns an option that adds a Kubernetes Service sink for the given ContainerSource.
func WithSinkServiceForContainerSource(name string) ContainerSourceOption {
	return func(cs *sourcesv1alpha1.ContainerSource) {
		cs.Spec.Sink = pkgTest.CoreV1ObjectReference("Service", "v1", name)
	}
}

// ContainerSource returns a Container EventSource.
func ContainerSource(
	name,
	imageName string,
	options ...ContainerSourceOption,
) *sourcesv1alpha1.ContainerSource {
	containerSource := &sourcesv1alpha1.ContainerSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: sourcesv1alpha1.ContainerSourceSpec{
			Image: pkgTest.ImagePath(imageName),
		},
	}
	for _, option := range options {
		option(containerSource)
	}
	return containerSource
}
