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
)

// Defines a node that should be created in the graph, along with parent dependencies
type graphEntry struct {
	id        string
	dependsOn []string
	node      *graph.Node
}

func BuildDirectedGraph(entries map[string]graphEntry) (*simple.DirectedGraph, error) {
	graphObj := simple.NewDirectedGraph()

	// add each entry to the graph
	for entryId, entry := range entries {
		addNode(graphObj, entries, &entry)
		entries[entryId] = entry

		if entry.dependsOn != nil {
			// add each dependency to the graph if it's not yet in it
			for _, dependencyId := range entry.dependsOn {
				dependency, ok := entries[dependencyId]
				if !ok {
					return nil, fmt.Errorf("entry '%s' depends on a graph entry that doesn't "+
						"exist: %s", entry.id, dependencyId)
				}

				addNode(graphObj, entries, &dependency)
				// update the entry in the map since it's a map of objects not pointers
				entries[dependencyId] = dependency

				log.Logger.Debugf("Creating edge from %+v to %+v", dependency, entry)

				// return an error instead of creating a loop
				if dependency.node == entry.node {
					return nil, fmt.Errorf("Node %s is not allowed to depend on itself", entry.id)
				}

				// now we have both nodes in the graph, create a directed edge between them
				edge := graphObj.NewEdge(*dependency.node, *entry.node)
				graphObj.SetEdge(edge)
			}
		}
	}

	return graphObj, nil
}

// Adds a node to the graph if the entry isn't already in it. Also adds a reference to the
// node on the graph entry instance
func addNode(graphObj *simple.DirectedGraph, nodes map[string]graphEntry, entry *graphEntry) {
	existing := nodes[entry.id]

	if existing.node != nil {
		if cmp.Equal(existing, *entry, cmp.Option(cmp.AllowUnexported(graphEntry{}))) {
			log.Logger.Debugf("Node '%s' is already in the graph", entry.id)
			return
		}
	}

	log.Logger.Debugf("Creating node '%s'", entry.id)

	node := graphObj.NewNode()
	graphObj.AddNode(node)
	// associate the node with the entry
	entry.node = &node
}

// Returns a boolean indicating whether the given directed graph is acyclic or not
func isAcyclic(graphObj *simple.DirectedGraph) bool {
	// Tarjan's strongly connected components algorithm can only be run on acyclic graphs
	_, err := topo.Sort(graphObj)
	return err == nil
}
