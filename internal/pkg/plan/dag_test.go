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

func TestBuildDag(t *testing.T) {
	input := map[string]graphEntry{
		"cluster":         {"cluster", nil, nil},
		"tiller":          {"tiller", []string{"cluster"}, nil},
		"externalIngress": {"externalIngress", []string{"tiller"}, nil},
		"sharedRds":       {"sharedRds", nil, nil},
		"wordpress1":      {"wordpress1", []string{"sharedRds", "externalIngress"}, nil},
		"wordpress2":      {"wordpress2", []string{"sharedRds", "externalIngress"}, nil},
		"varnish":         {"varnish", []string{"wordpress2"}, nil},
	}

	dag, err := BuildDirectedGraph(input)
	assert.Nil(t, err)

	for _, entry := range input {
		log.Logger.Debugf("Entry %s has node ID %v", entry.id, *entry.node)
	}

	nodes := dag.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		log.Logger.Debugf("DAG contains node %+v", node)
	}

	// assert that each entry has edges from any dependencies to itself
	for _, entry := range input {
		node := *entry.node
		to := dag.To(node.ID())

		if entry.dependsOn == nil || len(entry.dependsOn) == 0 {
			assert.Equal(t, 0, to.Len())
			log.Logger.Debugf("'%s' (node %v) has no dependencies", entry.id, *entry.node)
		} else {
			// convert the iterator of nodes to a map of nodes (which are just IDs)
			actualDependencies := make(map[graph.Node]bool, 0)
			for to.Next() {
				dep := to.Node()
				actualDependencies[dep] = true
			}

			log.Logger.Debugf("Actual dependencies for '%s' (node %v) are: %v",
				entry.id, *entry.node, actualDependencies)

			// make sure the lists are the same length
			assert.Equal(t, len(entry.dependsOn), len(actualDependencies))

			// make sure each dependency is an actual dependency
			for _, dependencyName := range entry.dependsOn {
				dependentEntry := input[dependencyName]
				dn := *dependentEntry.node
				_, ok := actualDependencies[dn]
				assert.True(t, ok, fmt.Sprintf("'%s' is missing a dependency: '%+v' not found in "+
					"in %v", entry.id, *dependentEntry.node, actualDependencies))
			}
		}
	}
}
