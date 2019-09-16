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
	"github.com/sugarkube/sugarkube/internal/pkg/config"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/printer"
	"github.com/sugarkube/sugarkube/internal/pkg/stack"
	"github.com/sugarkube/sugarkube/internal/pkg/utils"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	unprocessed = iota
	running
	finished
)

const markedNodeStr = "*"
const progressInterval = 30 // seconds

// Wrapper around a directed graph so we can define our own methods on it
type Dag struct {
	graph *simple.DirectedGraph
}

// Defines a node that should be created in the graph, along with parent dependencies. This is
// just a descriptor of a node, not an actual graph node
type nodeDescriptor struct {
	installableObj interfaces.IInstallable
}

// A node in a graph that also has a string name
type NamedNode struct {
	name           string // must be unique across all nodes in the graph
	node           graph.Node
	installableObj interfaces.IInstallable
	marked         bool // indicates whether this node was specifically marked for processing (e.g.
	// installing/deleting, etc. )
	conditionsValid bool // will be false if any of the installable's conditions are false
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
func Create(stackObj interfaces.IStack, selectedInstallableIds []string, includeParents bool) (*Dag, error) {

	manifests := stackObj.GetConfig().Manifests()

	manifestIds := make([]string, 0)
	for _, manifest := range manifests {
		manifestIds = append(manifestIds, manifest.Id())
	}

	log.Logger.Debugf("Creating DAG for installables '%s' in manifests %s",
		strings.Join(selectedInstallableIds, ", "), strings.Join(manifestIds, ", "))
	descriptors, err := findDependencies(manifests)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dag, err := build(descriptors, stackObj)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dag, err = dag.subGraph(selectedInstallableIds, includeParents)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Logger.Debugf("Finished creating DAG")

	return dag, nil
}

// Builds a graph from a map of descriptors that contain a string node ID plus a list of
// IDs of nodes that node depends on (i.e. parents).
// An error will be returned if the resulting graph is cyclical.
func build(descriptors map[string]nodeDescriptor, stackObj interfaces.IStack) (*Dag, error) {
	graphObj := simple.NewDirectedGraph()
	nodesByName := make(map[string]NamedNode, 0)

	var shouldProcess bool

	// add each descriptor to the graph
	for descriptorId, descriptor := range descriptors {
		installableObj := descriptor.installableObj

		// template the installable's descriptor
		templatedVars, err := stackObj.GetTemplatedVars(installableObj, map[string]interface{}{})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		err = installableObj.TemplateDescriptor(templatedVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// only process installables whose conditions are all true
		shouldProcess, err = utils.All(installableObj.GetDescriptor().Conditions)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		descriptorNode := addNode(graphObj, nodesByName, descriptorId,
			installableObj, shouldProcess)

		descriptorNode.conditionsValid = shouldProcess

		dependencies := installableObj.GetDescriptor().DependsOn

		if dependencies != nil {
			// add each dependency to the graph if it's not yet in it, provided all its conditions are met (if any)
			for _, dependency := range dependencies {
				// check its conditions are all true if it has any
				if len(dependency.Conditions) > 0 {
					log.Logger.Tracef("Evaluating conditions for dependency '%s': %#v", dependency.Id,
						dependency.Conditions)
					conditionsPassed, err := utils.All(dependency.Conditions)
					if err != nil {
						return nil, errors.WithStack(err)
					}

					if !conditionsPassed {
						log.Logger.Infof("Dependency '%s' has failed conditions. Won't add it to the DAG",
							dependency.Id)
						continue
					}
				}

				_, ok := descriptors[dependency.Id]
				if !ok {
					return nil, fmt.Errorf("descriptor '%s' depends on a graph "+
						"descriptor that doesn't exist: %s", descriptorId, dependency.Id)
				}

				dependentNode := descriptors[dependency.Id]
				parentNode := addNode(graphObj, nodesByName, dependency.Id,
					dependentNode.installableObj, true)

				log.Logger.Debugf("Creating edge from  '%s' to '%s'", dependency.Id, descriptorId)

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
		graph: graphObj,
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
			existing.conditionsValid = shouldProcess
			nodes[nodeName] = existing
			log.Logger.Tracef("Updating node '%s' to: %#v", nodeName, existing)
		}

		log.Logger.Debugf("Node '%s' already exists... won't recreate", nodeName)
		return existing
	}

	log.Logger.Debugf("Creating node '%s'", nodeName)

	// note - we don't create separate nodes for post actions because whether we actually run them
	// or not depends on if we're installing or deleting the installable
	namedNode := NamedNode{
		name:            nodeName,
		node:            graphObj.NewNode(),
		installableObj:  installableObj,
		marked:          shouldProcess,
		conditionsValid: shouldProcess,
	}

	log.Logger.Tracef("Adding node '%s': %#v", nodeName, namedNode)

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

// Returns a list of all marked installables in the DAG (in any order).
func (g *Dag) GetInstallables() []interfaces.IInstallable {
	log.Logger.Debug("Putting all installables in the DAG into a list")

	installables := make([]interfaces.IInstallable, 0)

	for _, node := range g.nodesByName() {
		if node.marked {
			installables = append(installables, node.installableObj)
		}
	}

	return installables
}

// Returns a new DAG comprising the nodes in the given input list and all their
// ancestors. The returned graph is guaranteed to be a DAG. All nodes in the input list will be
// marked for processing in the returned subgraph.
func (g *Dag) subGraph(nodeNames []string, includeParents bool) (*Dag, error) {

	log.Logger.Debugf("Extracting sub-graph for nodes: %s", strings.Join(nodeNames, ", "))

	outputGraph := simple.NewDirectedGraph()

	inputGraphNodesByName := g.nodesByName()
	ogNodesByName := make(map[string]NamedNode, 0)

	var shouldProcess bool
	var err error

	// find each named node along with all its ancestors and add them to the sub-graph
	for _, nodeName := range nodeNames {
		inputGraphNode, ok := inputGraphNodesByName[nodeName]
		if !ok {
			return nil, fmt.Errorf("Graph doesn't contain a node called '%s'", nodeName)
		}

		// only process installables whose conditions are all true
		shouldProcess, err = utils.All(inputGraphNode.installableObj.GetDescriptor().Conditions)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// mark that we should process this node
		ogNode := addNode(outputGraph, ogNodesByName, inputGraphNode.name,
			inputGraphNode.installableObj, shouldProcess)

		ogNode.conditionsValid = shouldProcess

		err = addAncestors(g.graph, outputGraph, ogNodesByName, inputGraphNode, ogNode, includeParents)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	dag := Dag{
		graph: outputGraph,
	}

	log.Logger.Debugf("Finished extracting sub-graph")

	return &dag, nil
}

func addAncestors(inputGraph *simple.DirectedGraph, outputGraph *simple.DirectedGraph,
	ogNodes map[string]NamedNode, igNode NamedNode, ogNode NamedNode, includeParents bool) error {
	igParents := inputGraph.To(igNode.ID())

	var conditionsValid bool
	var err error

	for igParents.Next() {
		igParentNode := igParents.Node().(NamedNode)
		log.Logger.Tracef("Adding ancestors for node '%s'", igParentNode.name)

		conditionsValid, err = utils.All(igParentNode.installableObj.GetDescriptor().Conditions)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger.Tracef("Adding ancestor node '%s' to graph", igParentNode.name)

		// we generally don't want to process ancestors, only use them to grab their
		// outputs, but it depends on `includeParents` and their conditions
		ogParentNode := addNode(outputGraph, ogNodes, igParentNode.name,
			igParentNode.installableObj, includeParents && conditionsValid)

		if includeParents {
			ogParentNode.conditionsValid = includeParents && conditionsValid
		} else {
			ogParentNode.conditionsValid = conditionsValid
		}

		// now we have parent and child nodes in the output graph , create a directed
		// edge between them
		edge := outputGraph.NewEdge(ogParentNode, ogNode)
		outputGraph.SetEdge(edge)

		// now recurse to the parent of the parent node
		err = addAncestors(inputGraph, outputGraph, ogNodes, igParentNode, ogParentNode, includeParents)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
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
func (g *Dag) walkDown(processCh chan<- NamedNode, doneCh chan NamedNode) chan bool {
	return g.walk(true, processCh, doneCh)

}

// Walks the DAG from leaves to root. A node will only be processed once all of its child nodes have been
// processed. A leaf node is a special case of this that has no children.
func (g *Dag) walkUp(processCh chan<- NamedNode, doneCh chan NamedNode) chan bool {
	return g.walk(false, processCh, doneCh)
}

// Walks the DAG in the given direction. If down==true nodes will only be processed if all parents have
// been processed. If down==false it will walk up the DAG from leaves to root, only processing nodes if
// all children have been processed.
func (g *Dag) walk(down bool, processCh chan<- NamedNode, doneCh chan NamedNode) chan bool {

	if down {
		log.Logger.Info("Starting walking down the DAG...")
	} else {
		log.Logger.Info("Starting walking up the DAG...")
	}

	nodeStatusesById := g.nodeStatusesById()
	log.Logger.Tracef("Node statuses by ID: %+v", nodeStatusesById)

	numNodes := g.graph.Nodes().Len()
	log.Logger.Debugf("Graph has %d nodes", numNodes)

	numWorkers := config.CurrentConfig.NumWorkers
	log.Logger.Debugf("Walking the DAG with %d workers", numWorkers)

	finishedCh := make(chan bool)

	channelsClosed := false
	initialiseCh := make(chan bool)

	mutex := &sync.Mutex{}

	go func() {
		// create a ticker to display progress
		progressTicker := time.NewTicker(progressInterval * time.Second)
		defer progressTicker.Stop()

		// loop until there are no nodes left which haven't been processed
		for {
			select {
			case <-initialiseCh:
				go func() {
					g.processEligibleNodes(nodeStatusesById, processCh, mutex, down)
					log.Logger.Debugf("Finished the initial pass processing eligible nodes")
				}()
			case namedNode, ok := <-doneCh:
				// this will be false if the channel has been closed, otherwise it pumps out nil values
				if ok {
					log.Logger.Debugf("Worker informs the DAG it's finished processing node '%s'", namedNode.name)
					nodeItem := nodeStatusesById[namedNode.node.ID()]
					nodeItem.status = finished
					mutex.Lock()
					nodeStatusesById[namedNode.node.ID()] = nodeItem

					// copy the node status map with a mutex so we don't hit concurrent map misuse issues
					nodeStatusesByIdCopy := make(map[int64]nodeStatus)

					for k, v := range nodeStatusesById {
						nodeStatusesByIdCopy[k] = v
					}
					mutex.Unlock()

					go func() {
						// reprocess nodes again since there's been a state change
						g.processEligibleNodes(nodeStatusesById, processCh, mutex, down)

						if allDone(nodeStatusesByIdCopy) {
							log.Logger.Infof("DAG fully processed")
							// keep track of whether we've closed the channels (possibly in another goroutine)
							if !channelsClosed {
								close(finishedCh)
								close(doneCh)
								close(processCh)
								channelsClosed = true
							}
						}
					}()
				}
			case <-progressTicker.C:
				inProgressNodes := make([]string, 0)
				for node, nodeStatus := range nodeStatusesById {
					if nodeStatus.status == running {
						namedNode := nodeStatusesById[node]
						inProgressNodes = append(inProgressNodes, namedNode.node.name)
					}
				}

				if len(inProgressNodes) > 0 {
					_, _ = printer.Fprintf("[yellow]Waiting on: [bold]%s[reset][yellow]...\n", strings.Join(inProgressNodes, ", "))
				}
			}
		}
	}()

	log.Logger.Tracef("Starting initialisation pass over nodes")
	initialiseCh <- true

	return finishedCh
}

// Adds nodes whose dependencies have all been satisfied into a channel for processing by workers
func (g *Dag) processEligibleNodes(nodeStatusesById map[int64]nodeStatus, processCh chan<- NamedNode,
	mutex *sync.Mutex, down bool) {

	// copy the node status map with a mutex so we don't hit concurrent map misuse issues
	nodeStatusesByIdCopy := make(map[int64]nodeStatus)

	mutex.Lock()
	for k, v := range nodeStatusesById {
		nodeStatusesByIdCopy[k] = v
	}
	mutex.Unlock()

	var currentStatus nodeStatus
	for _, nodeStatus := range nodeStatusesByIdCopy {
		// only consider unprocessed nodes
		if nodeStatus.status != unprocessed {
			//log.Logger.Tracef("Skipping node '%s' with status '%v' on this pass...",
			//	namedNode.node.name, nodeStatus.status)
			continue
		}

		var dependencies graph.Nodes
		if down {
			dependencies = g.graph.To(nodeStatus.node.ID())
		} else {
			dependencies = g.graph.From(nodeStatus.node.ID())
		}

		// we have a node that needs to be processed. Check to see if its dependencies have
		// been satisfied
		if dependenciesSatisfied(dependencies, nodeStatusesByIdCopy) {
			log.Logger.Debugf("All dependencies satisfied for node: %+v - Adding it to the "+
				"processing queue", nodeStatus.node)
			// Acquire a mutex and check whether the target node is definitely still unprocessed. If it is update the
			// status to running and process it. Otherwise keep searching.
			mutex.Lock()
			// read from the actual map since we have a mutex
			currentStatus = nodeStatusesById[nodeStatus.node.ID()]

			if currentStatus.status == unprocessed {
				// update the status to running so we don't keep requeuing completed nodes
				nodeStatus.status = running

				log.Logger.Tracef("Updating status of node (with mutex): %v", nodeStatus)
				// update the actual map (not the copy) since we have a mutex
				nodeStatusesById[nodeStatus.node.ID()] = nodeStatus
				mutex.Unlock()
				processCh <- nodeStatus.node
			} else {
				log.Logger.Tracef("Status of node %v is not 'unprocessed'. Will keep searching", nodeStatus)
				mutex.Unlock()
			}
		} else {
			log.Logger.Tracef("Dependencies not satisfied for node: %+v", nodeStatus.node)
		}
	}
}

// Prints the DAG
func (g *Dag) Print() error {
	_, err := printer.Fprintf("Created the following DAG. Only nodes marked with a %s will "+
		"be processed: \n", markedNodeStr)
	if err != nil {
		return errors.WithStack(err)
	}

	numWorkers := config.CurrentConfig.NumWorkers

	processCh := make(chan NamedNode, numWorkers)
	doneCh := make(chan NamedNode, numWorkers)
	finishedCh := g.walkDown(processCh, doneCh)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for node := range processCh {
				log.Logger.Debugf("Print worker received node %+v", node)
				parents := g.graph.To(node.ID())

				parentNames := make([]string, 0)
				for parents.Next() {
					parent := parents.Node().(NamedNode)
					parentNames = append(parentNames, parent.name)
				}

				marked := "  "
				if node.marked {
					marked = fmt.Sprintf("[bold]%s ", markedNodeStr)
				}

				sort.Strings(parentNames)
				parentNamesStr := strings.Join(parentNames, ", ")
				if parentNamesStr == "" {
					parentNamesStr = "<nothing>"
				}

				conditionsStr := ""
				if !node.conditionsValid {
					conditionsStr = " (conditions failed)"
				}

				str := fmt.Sprintf("  %s%s[reset]%s - depends on: %s\n", marked,
					node.name, conditionsStr, parentNamesStr)
				_, err = printer.Fprint(str)
				if err != nil {
					panic(err)
				}

				log.Logger.Tracef("Print worker finished with node: %+v", node)
				doneCh <- node
			}
		}()
	}

	<-finishedCh

	log.Logger.Debug("DAG printed")

	return nil
}

// Returns a definition of the DAG compatible with GraphViz for visualising it
func (g *Dag) Visualise(clusterName string) string {

	numWorkers := config.CurrentConfig.NumWorkers

	processCh := make(chan NamedNode, numWorkers)
	doneCh := make(chan NamedNode, numWorkers)
	finishedCh := g.walkDown(processCh, doneCh)

	// build an array of relationships in the format accepted by graphviz
	graphVizNodes := make([]string, 0)

	// graphViz style to apply to unmarked nodes
	const graphVizNodeNotMarked = ` [fontcolor="#FF0000" color="#FF0000"]`
	hasUnmarkedNodes := false
	var hasParents bool

	// this mutex stops race conditions where different goroutines simultaneously write to the list which causes
	// data loss
	mutex := &sync.Mutex{}

	for i := 0; i < numWorkers; i++ {
		go func() {
			for node := range processCh {
				log.Logger.Debugf("Visualise worker received node: %+v", node)
				parents := g.graph.To(node.ID())

				if !node.marked {
					graphVizNodes = append(graphVizNodes, fmt.Sprintf(`"%s" %s`, node.name, graphVizNodeNotMarked))
					hasUnmarkedNodes = true
				}

				hasParents = false
				for parents.Next() {
					hasParents = true
					parent := parents.Node().(NamedNode)
					mutex.Lock()
					graphVizNodes = append(graphVizNodes,
						fmt.Sprintf(`"%s" -> "%s";`, parent.name, node.name))
					mutex.Unlock()
				}

				if !hasParents {
					mutex.Lock()
					graphVizNodes = append(graphVizNodes, fmt.Sprintf(`"%s";`, node.name))
					mutex.Unlock()
				}

				log.Logger.Tracef("Visualise worker finished with node '%s' (id=%d): %#v", node.name, node.ID(), node)
				doneCh <- node
			}
		}()
	}

	<-finishedCh

	graphVizDigraph := fmt.Sprintf(`digraph {
label = "Cluster: %s"
labelloc = "t";
node [shape=box,style="rounded,bold"]
%%s
%%s
}`, clusterName)

	unmarkedLabel := ""

	if hasUnmarkedNodes {
		unmarkedLabel = `{
        notelabel [
          shape=plain
          label = "Nodes in red will not be processed"
        ]
    }`
	}

	graphVizDot := fmt.Sprintf(graphVizDigraph,
		strings.Join(graphVizNodes, "\n"), unmarkedLabel)

	log.Logger.Debugf("DAG visualisation spec produced: %s", graphVizDot)

	return graphVizDot
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
		dependency := dependencies.Node().(NamedNode)
		nodeStatus := nodeStatuses[dependency.ID()]
		if nodeStatus.status != finished {
			log.Logger.Tracef("Dependent node '%s' hasn't finished (status=%+v)", dependency.name,
				nodeStatus.status)
			return false
		}
	}

	return true
}

// Creates a DAG for installables matched by selectors. If an optional state (e.g. present, absent, etc.) is
// provided, only installables with the same state will be included in the returned DAG
func BuildDagForSelected(stackObj interfaces.IStack, workspaceDir string, includeSelector []string,
	excludeSelector []string, includeParents bool) (*Dag, error) {
	// load configs for all installables in the stack
	err := stackObj.LoadInstallables(workspaceDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// selected kapps will be returned in the order in which they appear in manifests, not the order
	// they're specified in selectors
	selectedInstallables, err := stack.SelectInstallables(stackObj.GetConfig().Manifests(),
		includeSelector, excludeSelector)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	selectedInstallableIds := make([]string, 0)

	for _, installableObj := range selectedInstallables {
		selectedInstallableIds = append(selectedInstallableIds,
			installableObj.FullyQualifiedId())
	}

	dagObj, err := Create(stackObj, selectedInstallableIds, includeParents)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = dagObj.Print()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dagObj, nil
}
