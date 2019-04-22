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
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"io"
	"strings"
	"time"
)

const (
	unprocessed = iota
	finished
)

// Wrapper around a directed graph so we can define our own methods on it
type Dag struct {
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
type NamedNode struct {
	name           string // must be unique across all nodes in the graph
	node           graph.Node
	installableObj interfaces.IInstallable
	marked         bool // indicates whether this node was specifically marked for processing (e.g.
	// installing/deleting, etc. )
}

func (n NamedNode) Name() string {
	return n.name
}

func (n NamedNode) ID() int64 {
	return n.node.ID()
}

// Used to track whether a node has been processed
type nodeStatus struct {
	node   NamedNode
	status int
}

// Creates a DAG for installables in the given manifests. If a list of selected installable IDs is
// given a subgraph will be returned containing only those installables and their ancestors.
func Create(manifests []interfaces.IManifest, selectedInstallableIds []string) (*Dag, error) {
	descriptors := findDependencies(manifests)
	dag, err := build(descriptors)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dag, err = dag.subGraph(selectedInstallableIds)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dag, nil
}

// Builds a graph from a map of descriptors that contain a string node ID plus a list of
// IDs of nodes that node depends on (i.e. parents).
// An error will be returned if the resulting graph is cyclical.
func build(descriptors map[string]nodeDescriptor) (*Dag, error) {
	graphObj := simple.NewDirectedGraph()
	nodesByName := make(map[string]NamedNode, 0)

	// add each descriptor to the graph
	for descriptorId, descriptor := range descriptors {
		descriptorNode := addNode(graphObj, nodesByName, descriptorId,
			descriptor.installableObj, true)

		if descriptor.dependsOn != nil {
			// add each dependency to the graph if it's not yet in it
			for _, dependencyId := range descriptor.dependsOn {
				_, ok := descriptors[dependencyId]
				if !ok {
					return nil, fmt.Errorf("descriptor '%s' depends on a graph "+
						"descriptor that doesn't exist: %s", descriptorId, dependencyId)
				}

				dependency := descriptors[dependencyId]
				parentNode := addNode(graphObj, nodesByName, dependencyId,
					dependency.installableObj, true)

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

	dag := Dag{
		graph:     graphObj,
		sleepTime: 500 * time.Millisecond,
	}

	return &dag, nil
}

// Adds a node to the graph if the entry isn't already in it. Also adds a reference to the
// node on the graph entry instance
func addNode(graphObj *simple.DirectedGraph, nodes map[string]NamedNode, nodeName string,
	installableObj interfaces.IInstallable, shouldProcess bool) NamedNode {
	existing, ok := nodes[nodeName]

	if ok {
		// if the existing node was added but wasn't marked for processing, and now
		// we would create it as a processable node, toggle the flag
		if !existing.marked && shouldProcess {
			existing.marked = shouldProcess
			nodes[nodeName] = existing
		}

		log.Logger.Debugf("Node '%s' already exists... won't recreate", nodeName)
		return existing
	}

	log.Logger.Debugf("Creating node '%s'", nodeName)

	// note - we don't create separate nodes for post actions because whether we actually run them
	// or not depends on if we're installing or deleting the installable
	namedNode := NamedNode{
		name:           nodeName,
		node:           graphObj.NewNode(),
		installableObj: installableObj,
		marked:         shouldProcess,
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

// Returns a new DAG comprising the nodes in the given input list and all their
// ancestors. The returned graph is guaranteed to be a DAG. All nodes in the input list will be
// marked for processing in the returned subgraph.
func (g *Dag) subGraph(nodeNames []string) (*Dag, error) {
	outputGraph := simple.NewDirectedGraph()

	inputGraphNodesByName := g.nodesByName()
	ogNodesByName := make(map[string]NamedNode, 0)

	// find each named node along with all its ancestors and add them to the sub-graph
	for _, nodeName := range nodeNames {
		inputGraphNode, ok := inputGraphNodesByName[nodeName]
		if !ok {
			return nil, fmt.Errorf("Graph doesn't contain a node called '%s'", nodeName)
		}

		// mark that we should process this node
		ogNode := addNode(outputGraph, ogNodesByName, inputGraphNode.Name(),
			inputGraphNode.installableObj, true)
		addAncestors(g.graph, outputGraph, ogNodesByName, inputGraphNode, ogNode)
	}

	dag := Dag{
		graph:     outputGraph,
		sleepTime: 500 * time.Millisecond,
	}

	return &dag, nil
}

func addAncestors(inputGraph *simple.DirectedGraph, outputGraph *simple.DirectedGraph,
	ogNodes map[string]NamedNode, igNode NamedNode, ogNode NamedNode) {
	igParents := inputGraph.To(igNode.ID())

	for igParents.Next() {
		igParentNode := igParents.Node().(NamedNode)

		// we don't want to process ancestors, only use them to grab their outputs
		ogParentNode := addNode(outputGraph, ogNodes, igParentNode.name,
			igParentNode.installableObj, false)

		// now we have parent and child nodes in the output graph , create a directed
		// edge between them
		edge := outputGraph.NewEdge(ogParentNode, ogNode)
		outputGraph.SetEdge(edge)

		// now recurse to the parent of the parent node
		addAncestors(inputGraph, outputGraph, ogNodes, igParentNode, ogParentNode)
	}
}

// Returns a map of nodeStatuses for each node in the graph keyed by node ID
func (g *Dag) nodeStatusesById() map[int64]nodeStatus {
	nodeMap := make(map[int64]nodeStatus, 0)

	nodes := g.graph.Nodes()

	for nodes.Next() {
		node := nodes.Node()
		nodeMap[node.ID()] = nodeStatus{
			node:   node.(NamedNode),
			status: unprocessed,
		}
	}

	return nodeMap
}

// Returns a map of nodes keyed by node name
func (g *Dag) nodesByName() map[string]NamedNode {
	nodeMap := make(map[string]NamedNode, 0)

	nodes := g.graph.Nodes()

	for nodes.Next() {
		node := nodes.Node().(NamedNode)
		nodeMap[node.name] = node
	}

	return nodeMap
}

// Traverses the graph from the root to leaves. Nodes will only be processed once their
// dependencies have been processed. Not having dependencies is a special case of this.
// The size of the processCh buffer determines the level of parallelisation
func (g *Dag) WalkDown(processCh chan<- NamedNode, doneCh chan NamedNode) chan bool {

	log.Logger.Info("Starting DAG traversal...")
	nodeStatusesById := g.nodeStatusesById()
	log.Logger.Tracef("Node statuses by ID: %+v", nodeStatusesById)

	numNodes := g.graph.Nodes().Len()
	log.Logger.Debugf("Graph has %d nodes", numNodes)

	// spawn a goroutine to listen to the doneCh to update the statuses of completed nodes
	go func() {
		for namedNode := range doneCh {
			log.Logger.Infof("Finished processing '%s'", namedNode.name)
			nodeItem := nodeStatusesById[namedNode.node.ID()]
			nodeItem.status = finished
			nodeStatusesById[namedNode.node.ID()] = nodeItem
		}
	}()

	finishedCh := make(chan bool)

	go func() {
		// loop until there are no nodes left which haven't been processed
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
				close(finishedCh)
				// closing the other channels seems to make go send a load of empty
				// instances to the receivers which messes things up
				break
			} else {
				// sleep a little bit to give jobs a chance to complete
				log.Logger.Tracef("DAG still processing. Sleeping for %s...", g.sleepTime)
				time.Sleep(g.sleepTime)
			}
		}
	}()

	return finishedCh
}

// todo - implement
func (g *Dag) WalkUp(processCh chan<- NamedNode, doneCh chan NamedNode) chan bool {
	panic("Not implemented")
	finishedCh := make(chan bool)
	return finishedCh
}

// Prints out the DAG to the writer
func (g *Dag) Print(writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "Created the DAG: \n")
	if err != nil {
		return errors.WithStack(err)
	}

	processCh := make(chan NamedNode, parallelisation)
	doneCh := make(chan NamedNode, parallelisation)
	finishedCh := g.WalkDown(processCh, doneCh)

	// temporarily reduce the sleep time
	originalSleepTime := g.sleepTime
	g.sleepTime = 1 * time.Millisecond

	go func() {
		for {
			select {
			case node := <-processCh:
				log.Logger.Debugf("Visited node: %+v", node)
				parents := g.graph.To(node.ID())

				parentNames := make([]string, 0)
				for parents.Next() {
					parent := parents.Node().(NamedNode)
					parentNames = append(parentNames, parent.name)
				}

				_, err := fmt.Fprintf(writer, "%s - depends on: %s\n", node.Name(),
					strings.Join(parentNames, ", "))
				if err != nil {
					panic(err)
				}
				doneCh <- node
			}
		}
	}()

	<-finishedCh

	g.sleepTime = originalSleepTime
	log.Logger.Debug("DAG printed")

	return nil
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
