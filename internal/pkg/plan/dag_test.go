/*
 * Copyright 2019 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plan

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gonum.org/v1/gonum/graph"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func getDescriptors() map[string]nodeDescriptor {
	return map[string]nodeDescriptor{
		// this depends on nothing and nothing depends on it
		"independent":     {"independent", nil, nil},
		"cluster":         {"cluster", nil, nil},
		"tiller":          {"tiller", []string{"cluster"}, nil},
		"externalIngress": {"externalIngress", []string{"tiller"}, nil},
		"sharedRds":       {"sharedRds", nil, nil},
		"wordpress1":      {"wordpress1", []string{"sharedRds", "externalIngress"}, nil},
		"wordpress2":      {"wordpress2", []string{"sharedRds", "externalIngress"}, nil},
		"varnish":         {"varnish", []string{"wordpress2"}, nil},
	}
}

// Tests that DAGs are created correctly
func TestBuildDag(t *testing.T) {
	input := getDescriptors()
	dag, err := BuildDAG(input)
	assert.Nil(t, err)

	for _, descriptor := range input {
		log.Logger.Debugf("Descriptor %s has node ID %v", descriptor.id, *descriptor.node)
	}

	nodes := dag.graph.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		log.Logger.Debugf("DAG contains node %+v", node)
	}

	// assert that each descriptor has edges from any dependencies to itself
	for _, descriptor := range input {
		node := *descriptor.node
		to := dag.graph.To(node.ID())

		if descriptor.dependsOn == nil || len(descriptor.dependsOn) == 0 {
			assert.Equal(t, 0, to.Len())
			log.Logger.Debugf("'%s' (node %v) has no dependencies", descriptor.id, *descriptor.node)
		} else {
			// convert the iterator of nodes to a map of nodes (which are just IDs)
			actualDependencies := make(map[graph.Node]bool, 0)
			for to.Next() {
				dep := to.Node()
				actualDependencies[dep] = true
			}

			log.Logger.Debugf("Actual dependencies for '%s' (node %v) are: %v",
				descriptor.id, *descriptor.node, actualDependencies)

			// make sure the lists are the same length
			assert.Equal(t, len(descriptor.dependsOn), len(actualDependencies))

			// make sure each dependency is an actual dependency
			for _, dependencyName := range descriptor.dependsOn {
				dependentEntry := input[dependencyName]
				dn := *dependentEntry.node
				_, ok := actualDependencies[dn]
				assert.True(t, ok, fmt.Sprintf("'%s' is missing a dependency: '%+v' not found in "+
					"in %v", descriptor.id, *dependentEntry.node, actualDependencies))
			}
		}
	}
}

// Makes sure an error is returned when trying to create loops
func TestBuildDagLoops(t *testing.T) {
	input := map[string]nodeDescriptor{
		"entry1": {"entry1", []string{"entry1"}, nil},
	}

	_, err := BuildDAG(input)
	assert.Error(t, err)
}

// Tests that we can spot a cyclic graph
func TestIsAcyclic(t *testing.T) {
	input := map[string]nodeDescriptor{
		"entry1": {"entry1", []string{"entry2"}, nil},
		"entry2": {"entry2", []string{"entry1"}, nil},
		"entry3": {"entry3", nil, nil},
	}

	_, err := BuildDAG(input)
	assert.NotNil(t, err)
}

func TestTraverse(t *testing.T) {
	input := getDescriptors()
	dag, err := BuildDAG(input)
	assert.Nil(t, err)

	processCh := make(chan nodeDescriptor)
	doneCh := make(chan nodeDescriptor)

	go func() {
		for descriptor := range processCh {
			log.Logger.Infof("Processing '%s' in goroutine...", descriptor.id)
			doneCh <- descriptor
		}
	}()

	dag.traverse(processCh, doneCh)
}
