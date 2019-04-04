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

package templater

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
)

// Iterate over the input variables trying to replace data as if it was a template. Keep iterating up to a maximum
// number of times, or until the size of the input and output remain the same. Doing this allows us to define
// intermediate variables or aliases (e.g. set `cluster_name` = '{{ .stack.region }}-{{ .stack.account }}' then just
// use '{{ .kapp.vars.cluster_name }}'. Templating this requires 2 iterations).
func IterativelyTemplate(vars map[string]interface{}) (map[string]interface{}, error) {

	// maximum number of iterations whils templating variables
	maxIterations := 20

	var previousBytes []byte
	var renderedYaml string

	log.Logger.Tracef("Iteratively templating variables: %+v", vars)

	for i := 0; i < maxIterations; i++ {
		log.Logger.Tracef("Templating variables. Iteration %d of max %d", i, maxIterations)

		// convert the input variables to YAML to simplify templating it
		yamlData, err := yaml.Marshal(&vars)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		//log.Logger.Debugf("Vars to template (raw): %s", vars)
		//log.Logger.Debugf("Vars to template as YAML:\n%s", yamlData)

		renderedYaml, err = RenderTemplate(string(yamlData[:]), vars)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		//log.Logger.Debugf("Variables templated as:\n%s", renderedYaml)

		// unmarshal the rendered template ready for another iteration
		currentBytes := []byte(renderedYaml)
		var renderedVars map[string]interface{}
		err = yaml.UnmarshalStrict(currentBytes, &renderedVars)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		vars = renderedVars
		if previousBytes != nil && bytes.Equal(previousBytes, currentBytes) {
			log.Logger.Debugf("Breaking out of templating variables after %d iterations", i)
			break
		}

		previousBytes = currentBytes
	}

	return vars, nil
}
