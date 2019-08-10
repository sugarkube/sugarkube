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
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"sync"
	"testing"
)

func init() {
	log.ConfigureLogger("trace", false)
}

func getDescriptors() map[string]nodeDescriptor {
	return map[string]nodeDescriptor{
		// this depends on nothing and nothing depends on it
		"independent":     {dependsOn: nil},
		"cluster":         {dependsOn: []string{}},
		"tiller":          {dependsOn: []string{"cluster"}},
		"externalIngress": {dependsOn: []string{"tiller"}},
		"sharedRds":       {dependsOn: nil},
		"wordpress1":      {dependsOn: []string{"sharedRds", "externalIngress"}},
		"wordpress2":      {dependsOn: []string{"sharedRds", "externalIngress"}},
		"varnish":         {dependsOn: []string{"wordpress2"}},
	}
}

// Tests that DAGs are created correctly
func TestBuildDag(t *testing.T) {
	input := getDescriptors()
	dag, err := build(input)
	assert.Nil(t, err)

	nodes := dag.graph.Nodes()
	for nodes.Next() {
		node := nodes.Node().(NamedNode)
		log.Logger.Debugf("DAG contains node %+v", node)

		descriptor := input[node.name]

		// assert that each node has edges from any dependencies to itself
		to := dag.graph.To(node.ID())

		if descriptor.dependsOn == nil || len(descriptor.dependsOn) == 0 {
			assert.Equal(t, 0, to.Len())
			log.Logger.Debugf("'%s' (node %v) has no dependencies", node.name, node)
		} else {
			// convert the iterator of nodes to a map of nodes keyed by name
			actualDependencies := make(map[string]NamedNode, 0)
			for to.Next() {
				parent := to.Node().(NamedNode)
				actualDependencies[parent.name] = NamedNode{}
			}

			log.Logger.Debugf("Actual dependencies for '%s' (node %v) are: %v",
				node.name, node, actualDependencies)

			// make sure the lists are the same length
			assert.Equal(t, len(descriptor.dependsOn), len(actualDependencies))

			// make sure each dependency is an actual dependency
			for _, dependencyName := range descriptor.dependsOn {
				_, ok := actualDependencies[dependencyName]
				assert.True(t, ok, fmt.Sprintf("'%s' is missing a dependency: '%+v' not found in "+
					"in %v", node.name, dependencyName, actualDependencies))
			}
		}
	}
}

// Makes sure an error is returned when trying to create loops
func TestBuildDagLoops(t *testing.T) {
	input := map[string]nodeDescriptor{
		"entry1": {dependsOn: []string{"entry1"}},
	}

	_, err := build(input)
	assert.Error(t, err)
}

// Tests that we can spot a cyclic graph
func TestIsAcyclic(t *testing.T) {
	input := map[string]nodeDescriptor{
		"entry1": {dependsOn: []string{"entry2"}},
		"entry2": {dependsOn: []string{"entry1"}},
		"entry3": {dependsOn: nil},
	}

	_, err := build(input)
	assert.NotNil(t, err)
}

func TestTraverse(t *testing.T) {
	input := getDescriptors()
	dag, err := build(input)
	assert.Nil(t, err)

	// IDs of nodes which could be the first to be run
	possibleFirstNodes := []string{
		"independent",
		"cluster",
		"sharedRds",
	}

	// IDs of nodes which could be the last to be run
	possibleLastNodes := []string{
		"independent",
		"varnish",
		"wordpress1",
	}

	processCh := make(chan NamedNode)
	doneCh := make(chan NamedNode)

	mutex := &sync.Mutex{}
	numProcessed := 0
	var lastProcessedId string

	numWorkers := 5

	for i := 0; i < numWorkers; i++ {
		go func() {
			for node := range processCh {
				log.Logger.Infof("Processing '%s' in goroutine...", node.name)

				// make sure the first node we process is one of those marked as being allowed to
				// be processed first
				if numProcessed == 0 {
					assert.True(t, utils.InStringArray(possibleFirstNodes, node.name))
				}

				lastProcessedId = node.name

				mutex.Lock()
				numProcessed++
				mutex.Unlock()

				log.Logger.Debugf("Test worker finished processing node '%s'", node.name)
				doneCh <- node
			}
		}()
	}

	finishedCh := dag.walkDown(processCh, doneCh)

	// wait for traversal to finish
	select {
	case <-finishedCh:
		break
	}

	// make sure the last to be processed is marked as being allowed to be last
	assert.True(t, utils.InStringArray(possibleLastNodes, lastProcessedId))
}

// Test we can extract subgraphs of the node
func TestSubGraph(t *testing.T) {
	input := getDescriptors()
	dag, err := build(input)
	assert.Nil(t, err)

	nodeNames := []string{"wordpress1", "independent"}

	subGraph, err := dag.subGraph(nodeNames, false)
	assert.Nil(t, err)

	nodesByName := subGraph.nodesByName()

	for _, nodeName := range nodeNames {
		assertDependencies(t, subGraph, input, nodesByName, nodeName, true)
	}
}

func assertDependencies(t *testing.T, graphObj *Dag, descriptors map[string]nodeDescriptor,
	nodesByName map[string]NamedNode, nodeName string, shouldProcess bool) {
	node := nodesByName[nodeName]

	assert.Equal(t, shouldProcess, node.marked, "marked for node '%s' is not %v",
		nodeName, shouldProcess)

	parents := graphObj.graph.To(node.ID())

	dependencyNames := descriptors[nodeName].dependsOn

	assert.Equal(t, len(dependencyNames), parents.Len())
	if parents.Len() > 0 {
		// make sure the parents are the ones we want
		for parents.Next() {
			parent := parents.Node().(NamedNode)
			assert.True(t, utils.InStringArray(dependencyNames, parent.name),
				"%s is not in %s", parent.name, dependencyNames)
			assertDependencies(t, graphObj, descriptors, nodesByName, parent.name, false)
		}

	}
}
