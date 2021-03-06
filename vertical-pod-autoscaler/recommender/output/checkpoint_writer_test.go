/*
Copyright 2018 The Kubernetes Authors.

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

package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/vertical-pod-autoscaler/recommender/model"
)

// TODO: Extract these constants to a common test module.
var (
	testPodID1       = model.PodID{"namespace-1", "pod-1"}
	testContainerID1 = model.ContainerID{testPodID1, "container-1"}
	testLabels       = map[string]string{"label-1": "value-1"}
	testRequest      = model.Resources{
		model.ResourceCPU:    model.CPUAmountFromCores(3.14),
		model.ResourceMemory: model.MemoryAmountFromBytes(3.14e9),
	}
)

func TestMergeContainerStateForCheckpointDropsRecentMemoryPeak(t *testing.T) {
	cluster := model.NewClusterState()
	cluster.AddOrUpdatePod(testPodID1, testLabels, apiv1.PodRunning)
	assert.NoError(t, cluster.AddOrUpdateContainer(testContainerID1, testRequest))
	container := cluster.GetContainer(testContainerID1)

	timeNow := time.Unix(1, 0)
	container.AddSample(&model.ContainerUsageSample{
		timeNow, model.MemoryAmountFromBytes(1024 * 1024 * 1024), model.ResourceMemory})
	vpa := &model.Vpa{Pods: cluster.Pods}

	// Verify that the current peak is excluded from the aggregation.
	aggregateContainerStateMap := buildAggregateContainerStateMap(vpa, timeNow)
	if assert.Contains(t, aggregateContainerStateMap, "container-1") {
		assert.True(t, aggregateContainerStateMap["container-1"].AggregateMemoryPeaks.IsEmpty(),
			"Current peak was not excluded from the aggregation.")
	}
	// Verify that an old peak is not excluded from the aggregation.
	timeNow = timeNow.Add(model.MemoryAggregationInterval)
	aggregateContainerStateMap = buildAggregateContainerStateMap(vpa, timeNow)
	if assert.Contains(t, aggregateContainerStateMap, "container-1") {
		assert.False(t, aggregateContainerStateMap["container-1"].AggregateMemoryPeaks.IsEmpty(),
			"Old peak should not be excluded from the aggregation.")
	}
}
