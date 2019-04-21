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
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
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

// Wrapper around a directed graph so we can define our own methods on it
type dag struct {
	graph     *simple.DirectedGraph
	sleepTime time.Duration
}

// Defines a node that should be created in the graph, along with parent dependencies. This is
// just a descriptor of a node, not an actual graph node
type nodeDescriptor struct {
	dependsOn      []string
	installableObj interfaces.IInstallable
}

// A node in a graph that also has a string name
type namedNode struct {
	name           string // must be unique across all nodes in the graph
	node           graph.Node
	installableObj interfaces.IInstallable
}

func (n namedNode) Name() string {
	return n.name
}

func (n namedNode) ID() int64 {
	return n.node.ID()
}

// Used to track whether a node has been processed
type nodeStatus struct {
	node   namedNode
	status int
}

// Builds a graph from a map of descriptors that contain a string node ID plus a list of
// IDs of nodes that node depends on (i.e. parents).
// An error will be returned if the resulting graph is cyclical.
func BuildDAG(descriptors map[string]nodeDescriptor) (*dag, error) {
	graphObj := simple.NewDirectedGraph()
	nodesByName := make(map[string]namedNode, 0)

	// add each descriptor to the graph
	for descriptorId, descriptor := range descriptors {
		descriptorNode := addNode(graphObj, nodesByName, descriptorId, descriptor.installableObj)

		if descriptor.dependsOn != nil {
			// add each dependency to the graph if it's not yet in it
			for _, dependencyId := range descriptor.dependsOn {
				_, ok := descriptors[dependencyId]
				if !ok {
					return nil, fmt.Errorf("descriptor '%s' depends on a graph "+
						"descriptor that doesn't exist: %s", descriptorId, dependencyId)
				}

				dependency := descriptors[dependencyId]
				parentNode := addNode(graphObj, nodesByName, dependencyId, dependency.installableObj)

				log.Logger.Debugf("Creating edge from  '%s' to '%s'", dependencyId, descriptorId)

				// return an error instead of creating a loop
				if parentNode.node == descriptorNode.node {
					return nil, fmt.Errorf("Node %s is not allowed to depend on itself",
						descriptorNode.name)
				}

				// now we have both nodes in the graph, create a directed edge between them
				edge := graphObj.NewEdge(parentNode, descriptorNode)
				graphObj.SetEdge(edge)
			}
		}
	}

	if !isAcyclic(graphObj) {
		return nil, fmt.Errorf("Cyclical dependencies detected")
	}

	dag := dag{
		graph:     graphObj,
		sleepTime: 500 * time.Millisecond,
	}

	return &dag, nil
}

// Adds a node to the graph if the entry isn't already in it. Also adds a reference to the
// node on the graph entry instance
func addNode(graphObj *simple.DirectedGraph, nodes map[string]namedNode, nodeName string,
	installableObj interfaces.IInstallable) namedNode {
	existing, ok := nodes[nodeName]

	if ok {
		log.Logger.Debugf("Node '%s' already exists... won't recreate", nodeName)
		return existing
	}

	log.Logger.Debugf("Creating node '%s'", nodeName)

	namedNode := namedNode{
		name:           nodeName,
		node:           graphObj.NewNode(),
		installableObj: installableObj,
	}
	graphObj.AddNode(namedNode)
	nodes[nodeName] = namedNode
	return namedNode
}

// Returns a boolean indicating whether the given directed graph is acyclic or not
func isAcyclic(graphObj *simple.DirectedGraph) bool {
	// Tarjan's strongly connected components algorithm can only be run on acyclic graphs,
	// so if it doesn't return an error we have an acyclic graph.
	_, err := topo.Sort(graphObj)
	return err == nil
}

// Returns a new DAG comprising the nodes in the given input descriptors and all their
// ancestors. The returned graph is guaranteed to be a DAG.
func (g *dag) subGraph(descriptors []string) (*dag, error) {
	outputGraph := simple.NewDirectedGraph()

	inputGraphNodesByName := g.nodesByName()
	ogNodesByName := make(map[string]namedNode, 0)

	// find each named node along with all its ancestors and add them to the sub-graph
	for _, descriptorId := range descriptors {
		inputGraphNode, ok := inputGraphNodesByName[descriptorId]
		if !ok {
			return nil, fmt.Errorf("Graph doesn't contain a node called '%s'", descriptorId)
		}

		ogNode := addNode(outputGraph, ogNodesByName, inputGraphNode.Name(), inputGraphNode.installableObj)
		addAncestors(g.graph, outputGraph, ogNodesByName, inputGraphNode, ogNode)
	}

	dag := dag{
		graph:     outputGraph,
		sleepTime: 500 * time.Millisecond,
	}

	return &dag, nil
}

func addAncestors(inputGraph *simple.DirectedGraph, outputGraph *simple.DirectedGraph,
	ogNodes map[string]namedNode, igNode namedNode, ogNode namedNode) {
	igParents := inputGraph.To(igNode.ID())

	for igParents.Next() {
		igParentNode := igParents.Node().(namedNode)
		ogParentNode := addNode(outputGraph, ogNodes, igParentNode.name, igParentNode.installableObj)

		// now we have parent and child nodes in the output graph , create a directed
		// edge between them
		edge := outputGraph.NewEdge(ogParentNode, ogNode)
		outputGraph.SetEdge(edge)

		// now recurse to the parent of the parent node
		addAncestors(inputGraph, outputGraph, ogNodes, igParentNode, ogParentNode)
	}
}

// Returns a map of nodeStatuses for each node in the graph keyed by node ID
func (g *dag) nodeStatusesById() map[int64]nodeStatus {
	nodeMap := make(map[int64]nodeStatus, 0)

	nodes := g.graph.Nodes()

	for nodes.Next() {
		node := nodes.Node()
		nodeMap[node.ID()] = nodeStatus{
			node:   node.(namedNode),
			status: unprocessed,
		}
	}

	return nodeMap
}

// Returns a map of nodes keyed by node name
func (g *dag) nodesByName() map[string]namedNode {
	nodeMap := make(map[string]namedNode, 0)

	nodes := g.graph.Nodes()

	for nodes.Next() {
		node := nodes.Node().(namedNode)
		nodeMap[node.name] = node
	}

	return nodeMap
}

// Traverses the graph. Nodes will only be processed if their dependencies have been satisfied.
// Not having dependencies is a special case of this.
// The size of the processCh buffer determines the level of parallelisation
func (g *dag) traverse(processCh chan<- namedNode, doneCh chan namedNode) {

	log.Logger.Info("Starting DAG traversal...")

	nodeStatusesById := g.nodeStatusesById()

	// spawn a goroutine to listen on doneCh to update the statuses of completed nodes
	go func() {
		for namedNode := range doneCh {
			log.Logger.Infof("Finished processing '%s'", namedNode.name)
			nodeItem := nodeStatusesById[namedNode.node.ID()]
			nodeItem.status = finished
			nodeStatusesById[namedNode.node.ID()] = nodeItem
		}
	}()

	// loop until there are no descriptors left which haven't been processed
	for {
		for node, nodeStatus := range nodeStatusesById {
			// only consider unprocessed nodes
			if nodeStatus.status != unprocessed {
				continue
			}

			namedNode := nodeStatusesById[node]

			// we have a node that needs to be processed. Check to see if its dependencies have
			// been satisfied
			if dependenciesSatisfied(g.graph.To(nodeStatus.node.ID()), nodeStatusesById) {
				log.Logger.Debugf("All dependencies satisfied for '%s', adding it to the "+
					"processing queue", namedNode.node.name)
				processCh <- namedNode.node
			}
		}

		if allDone(nodeStatusesById) {
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
func allDone(nodeStatuses map[int64]nodeStatus) bool {
	for _, nodeStatus := range nodeStatuses {
		if nodeStatus.status != finished {
			return false
		}
	}

	return true
}

// Returns a boolean indicating whether all dependencies of a node have been satisfied
func dependenciesSatisfied(dependencies graph.Nodes, nodeStatuses map[int64]nodeStatus) bool {

	for dependencies.Next() {
		dependency := dependencies.Node()

		nodeStatus := nodeStatuses[dependency.ID()]
		if nodeStatus.status != finished {
			return false
		}
	}

	return true
}
