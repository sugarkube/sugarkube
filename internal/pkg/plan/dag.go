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
	"github.com/google/go-cmp/cmp"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"time"
)

// Defines a node that should be created in the graph, along with parent dependencies
type nodeDescriptor struct {
	id        string
	dependsOn []string
	node      *graph.Node
}

func BuildDirectedGraph(descriptors map[string]nodeDescriptor) (*simple.DirectedGraph, error) {
	graphObj := simple.NewDirectedGraph()

	// add each descriptor to the graph
	for descriptorId, descriptor := range descriptors {
		addNode(graphObj, descriptors, &descriptor)
		descriptors[descriptorId] = descriptor

		if descriptor.dependsOn != nil {
			// add each dependency to the graph if it's not yet in it
			for _, dependencyId := range descriptor.dependsOn {
				dependency, ok := descriptors[dependencyId]
				if !ok {
					return nil, fmt.Errorf("descriptor '%s' depends on a graph "+
						"descriptor that doesn't exist: %s", descriptor.id, dependencyId)
				}

				addNode(graphObj, descriptors, &dependency)
				// update the descriptor in the map since it's a map of objects not pointers
				descriptors[dependencyId] = dependency

				log.Logger.Debugf("Creating edge from %+v to %+v", dependency, descriptor)

				// return an error instead of creating a loop
				if dependency.node == descriptor.node {
					return nil, fmt.Errorf("Node %s is not allowed to depend on itself", descriptor.id)
				}

				// now we have both nodes in the graph, create a directed edge between them
				edge := graphObj.NewEdge(*dependency.node, *descriptor.node)
				graphObj.SetEdge(edge)
			}
		}
	}

	return graphObj, nil
}

// Adds a node to the graph if the entry isn't already in it. Also adds a reference to the
// node on the graph entry instance
func addNode(graphObj *simple.DirectedGraph, descriptors map[string]nodeDescriptor, descriptor *nodeDescriptor) {
	existing := descriptors[descriptor.id]

	if existing.node != nil {
		if cmp.Equal(existing, *descriptor, cmp.Option(cmp.AllowUnexported(nodeDescriptor{}))) {
			log.Logger.Debugf("Descriptor '%s' is already in the graph", descriptor.id)
			return
		}
	}

	log.Logger.Debugf("Creating node '%s'", descriptor.id)

	node := graphObj.NewNode()
	graphObj.AddNode(node)
	// associate the node with the descriptor
	descriptor.node = &node
}

// Returns a boolean indicating whether the given directed graph is acyclic or not
func isAcyclic(graphObj *simple.DirectedGraph) bool {
	// Tarjan's strongly connected components algorithm can only be run on acyclic graphs,
	// so if it doesn't return an error we have an acyclic graph.
	_, err := topo.Sort(graphObj)
	return err == nil
}

// Traverses the graph. Nodes will only be processed if their dependencies have been satisfied.
// Not having dependencies is a special case of this.
func traverse(graphObj *simple.DirectedGraph, descriptors map[string]nodeDescriptor,
	processCh chan<- nodeDescriptor, doneCh <-chan nodeDescriptor) {

	// create a map keyed by node where the boolean indicates whether the node has been processed
	descriptorStatuses := make(map[graph.Node]bool, 0)

	// build a map of descriptors keyed by node ID
	descriptorsByNode := make(map[graph.Node]nodeDescriptor, 0)
	for _, descriptor := range descriptors {
		descriptorsByNode[*descriptor.node] = descriptor
	}

	// mark all nodes as unprocessed
	nodes := graphObj.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		descriptorStatuses[node] = false
	}

	// loop until there are no descriptors left which haven't been processed
	for {
		for node, done := range descriptorStatuses {
			if done {
				continue
			}

			descriptor := descriptorsByNode[node]

			// we have a node that hasn't been done. Check to see if its dependencies have
			// been satisfied
			if dependenciesSatisfied(graphObj.To(node.ID()), descriptorStatuses) {
				processCh <- descriptor
			}

			// todo - somehow select on doneCh and use it to update the statuses
		}

		if allDone(descriptorStatuses) {
			break
		} else {
			// sleep a little bit to give jobs a chance to complete
			// todo - this is probably wrong. We may need to block on a select instead...
			time.Sleep(1 * time.Second)
		}
	}
}

// Returns a boolean indicating whether all nodes have been processed
func allDone(descriptorStatuses map[graph.Node]bool) bool {
	for _, done := range descriptorStatuses {
		if !done {
			return false
		}
	}

	return true
}

// Returns a boolean indicating whether all dependencies of a node have been satisfied
func dependenciesSatisfied(dependencies graph.Nodes, descriptorStatuses map[graph.Node]bool) bool {

	for dependencies.Next() {
		dependency := dependencies.Node()

		done, _ := descriptorStatuses[dependency]
		if !done {
			return false
		}
	}

	return true
}
