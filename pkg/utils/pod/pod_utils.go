/*
Copyright 2023.

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

package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	leaderworkerset "sigs.k8s.io/lws/api/leaderworkerset/v1"
)

// ContainerRestarted return true when there is any container in the pod that gets restarted
func ContainerRestarted(pod corev1.Pod) bool {
	if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
		for j := range pod.Status.InitContainerStatuses {
			stat := pod.Status.InitContainerStatuses[j]
			if stat.RestartCount > 0 {
				return true
			}
		}
		for j := range pod.Status.ContainerStatuses {
			stat := pod.Status.ContainerStatuses[j]
			if stat.RestartCount > 0 {
				return true
			}
		}
	}
	return false
}

// PodDeleted checks if the worker pod has been deleted
func PodDeleted(pod corev1.Pod) bool {
	return pod.DeletionTimestamp != nil
}

// LeaderPod check is the pod is a leader pod
func LeaderPod(pod corev1.Pod) bool {
	return pod.Labels[leaderworkerset.WorkerIndexLabelKey] == "0"
}

// PodRunningAndReady checks if the pod condition is running and marked as ready.
func PodRunningAndReady(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning && podReady(pod)
}

func podReady(pod corev1.Pod) bool {
	return podReadyConditionTrue(pod.Status)
}

func podReadyConditionTrue(status corev1.PodStatus) bool {
	condition := getPodReadyCondition(status)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func getPodReadyCondition(status corev1.PodStatus) *corev1.PodCondition {
	_, condition := getPodCondition(&status, corev1.PodReady)
	return condition
}

func getPodCondition(status *corev1.PodStatus, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	return getPodConditionFromList(status.Conditions, conditionType)
}

func getPodConditionFromList(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) (int, *corev1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

func addEnvVarIfNotExists(c *corev1.Container, e corev1.EnvVar) {
	for _, env := range c.Env {
		if env.Name == e.Name {
			return
		}
	}
	c.Env = append([]corev1.EnvVar{e}, c.Env...)
}

// AddLWSVariables adds LWS_LEADER_ADDRESS environment variable to every container.
func AddLWSVariables(pod *corev1.Pod) error {
	lwsName, found := pod.Labels[leaderworkerset.SetNameLabelKey]
	if !found {
		return fmt.Errorf("Failure constructing environment variables, no name label found for pod %v", pod.Name)
	}

	groupIndex, found := pod.Labels[leaderworkerset.GroupIndexLabelKey]
	if !found {
		return fmt.Errorf("Failure constructing environment variables, no group index label found for pod %v", pod.Name)
	}

	// The headless service name is assumed to be the same as the LWS name.
	// See function [createHeadlessServiceIfNotExists](sigs.k8s.io/lws/pkg/controllers/leaderworkerset_controller.go).
	leaderAddressEnvVar := corev1.EnvVar{
		Name:  leaderworkerset.LwsLeaderAddress,
		Value: fmt.Sprintf("%s-%s.%s.%s", lwsName, groupIndex, lwsName, pod.ObjectMeta.Namespace),
	}

	for i := range pod.Spec.Containers {
		addEnvVarIfNotExists(&pod.Spec.Containers[i], leaderAddressEnvVar)
	}
	for i := range pod.Spec.InitContainers {
		addEnvVarIfNotExists(&pod.Spec.InitContainers[i], leaderAddressEnvVar)
	}

	return nil
}
