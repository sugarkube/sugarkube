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
func RenderTemplate(inputTemplate string, vars map[string]interface{}) (string, error) {
	tpl := template.Must(
		template.New("gotpl").Funcs(
			sprig.TxtFuncMap()).Funcs(CustomFunctions).Parse(inputTemplate))

	buf := bytes.NewBuffer(nil)
	err := tpl.Execute(buf, vars)
	if err != nil {
		return "", errors.Wrapf(err, "Error executing template: %s", inputTemplate)
	}

	return buf.String(), nil
}

// Renders a template from a template file to a buffer
func TemplateFile(src string, outBuf *bytes.Buffer, vars map[string]interface{}) error {

	// verify that the input template exists
	if _, err := os.Stat(src); err != nil {
		return errors.Wrapf(err, "Source template '%s' doesn't exist", src)
	}

	srcTemplate, err := ioutil.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "Error reading source template file %s", src)
	}

	log.Logger.Debugf("Rendering template in '%s' with vars: %#v", src, vars)

	rendered, err := RenderTemplate(string(srcTemplate[:]), vars)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = outBuf.Write([]byte(rendered))
	if err != nil {
		return errors.WithStack(err)
	}

	log.Logger.Debugf("Successfully rendered input template '%s'", src)

	return nil
}
