/*
 * Copyright 2018 The Sugarkube Authors
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
	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"io/ioutil"
	"os"
	"text/template"
)

// Returns a template rendered with the given input variables
func renderTemplate(inputTemplate string, vars map[string]interface{}) (string, error) {
	tpl := template.Must(
		template.New("gotpl").Funcs(sprig.TxtFuncMap()).Parse(inputTemplate))

	buf := bytes.NewBuffer(nil)
	err := tpl.Execute(buf, vars)
	if err != nil {
		return "", errors.Wrapf(err, "Error executing template: %s", inputTemplate)
	}

	return buf.String(), nil
}

// Renders a template from a template file, writing the output to another file
// at a specified path, optionally overwriting it.
func TemplateFile(src string, dest string, vars map[string]interface{},
	overwrite bool) error {

	// verify that the input template exists
	if _, err := os.Stat(src); err != nil {
		return errors.Wrapf(err, "Source template '%s' doesn't exist", src)
	}

	// if the dest path exists, only continue if we're allowed to overwrite it
	if _, err := os.Stat(dest); err == nil && !overwrite {
		return errors.Wrapf(err, "Template destination path '%s' already "+
			"exists, and overwrite=false", dest)
	}

	srcTemplate, err := ioutil.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "Error reading source template file %s", src)
	}

	log.Logger.Debugf("Rendering template in '%s' with vars: %#v", src, vars)

	rendered, err := renderTemplate(string(srcTemplate[:]), vars)
	if err != nil {
		return errors.WithStack(err)
	}

	err = ioutil.WriteFile(dest, []byte(rendered), 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Successfully rendered input template '%s' to '%s", src, dest)

	return nil
}
