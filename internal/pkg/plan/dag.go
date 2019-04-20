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

const (
	unprocessed = iota
	finished
)

// Encapsulates both a directed graph and descriptions of the nodes and each one's parents/dependencies
type dag struct {
	graph       *simple.DirectedGraph
	descriptors map[string]nodeDescriptor
	sleepTime   time.Duration
}

// Defines a node that should be created in the graph, along with parent dependencies
type nodeDescriptor struct {
	id        string
	dependsOn []string
	node      *graph.Node
}

// Builds a graph from a map of descriptors that contain a string node ID plus a list of
// IDs of nodes that node depends on (i.e. parents).
// An error will be returned if the resulting graph is cyclical.
func BuildDAG(descriptors map[string]nodeDescriptor) (*dag, error) {
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

	if !isAcyclic(graphObj) {
		return nil, fmt.Errorf("Cyclical dependencies detected")
	}

	dag := dag{
		graph:       graphObj,
		descriptors: descriptors,
		sleepTime:   500 * time.Millisecond,
	}

	return &dag, nil
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

// todo - create a method to extract a subtree for specific kapps so we can restrict processing
//  to a subset of the graph to (un)install specific kapps

// Traverses the graph. Nodes will only be processed if their dependencies have been satisfied.
// Not having dependencies is a special case of this.
// The size of the processCh buffer determines the level of parallelisation
func (g *dag) traverse(processCh chan<- nodeDescriptor, doneCh chan nodeDescriptor) {

	log.Logger.Info("Starting DAG traversal...")

	// create a map keyed by node where the boolean indicates whether the node has been processed
	descriptorStatuses := make(map[graph.Node]int, 0)

	// build a map of descriptors keyed by node ID
	descriptorsByNode := make(map[graph.Node]nodeDescriptor, 0)
	for _, descriptor := range g.descriptors {
		descriptorsByNode[*descriptor.node] = descriptor
	}

	// mark all nodes as unprocessed
	nodes := g.graph.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		descriptorStatuses[node] = unprocessed
	}

	// spawn a goroutine to listen on doneCh to update the statuses of completed nodes
	go func() {
		for descriptor := range doneCh {
			log.Logger.Infof("Finished processing '%s'", descriptor.id)
			descriptorStatuses[*descriptor.node] = finished
		}
	}()

	// loop until there are no descriptors left which haven't been processed
	for {
		for node, status := range descriptorStatuses {
			// only consider unprocessed nodes
			if status != unprocessed {
				continue
			}

			descriptor := descriptorsByNode[node]

			// we have a node that needs to be processed. Check to see if its dependencies have
			// been satisfied
			if dependenciesSatisfied(g.graph.To(node.ID()), descriptorStatuses) {
				log.Logger.Debugf("All dependencies satisfied for '%s', adding it to the "+
					"processing queue", descriptor.id)
				processCh <- descriptor
			}
		}

		if allDone(descriptorStatuses) {
			log.Logger.Infof("DAG fully processed")
			close(processCh)
			close(doneCh)
			break
		} else {
			// sleep a little bit to give jobs a chance to complete
			log.Logger.Tracef("DAG still processing. Sleeping for %s...", g.sleepTime)
			time.Sleep(g.sleepTime)
		}
	}
}

// Returns a boolean indicating whether all nodes have been processed
func allDone(descriptorStatuses map[graph.Node]int) bool {
	for _, status := range descriptorStatuses {
		if status != finished {
			return false
		}
	}

	return true
}

// Returns a boolean indicating whether all dependencies of a node have been satisfied
func dependenciesSatisfied(dependencies graph.Nodes, descriptorStatuses map[graph.Node]int) bool {

	for dependencies.Next() {
		dependency := dependencies.Node()

		status := descriptorStatuses[dependency]
		if status != finished {
			return false
		}
	}

	return true
}
