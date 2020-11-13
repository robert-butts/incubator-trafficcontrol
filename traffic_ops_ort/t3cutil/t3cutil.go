package t3cutil

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CmdDetermineRestart is a helper for the sub-command t3c-determine-restart.
func CmdDetermineRestart(mode Mode, pluginPackagesInstalled []string, changedConfigFiles []string) (ServiceNeeds, error) {
	cmdName := `t3c-determine-restart`
	cmdPath, err := makePath(cmdName)
	if err != nil {
		// fall back to PATH
		cmdPath = cmdName // TODO warn?
	}

	modeStr := mode.String()
	cmd := exec.Command(cmdPath, "--run-mode="+modeStr, "--plugin-packages-installed="+strings.Join(pluginPackagesInstalled, ","), "--changed-config-paths="+strings.Join(changedConfigFiles, ","))
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	errBuf := &bytes.Buffer{}
	cmd.Stderr = errBuf
	if err := cmd.Run(); err != nil {
		err := errors.New("stdout '" + buf.String() + "' stderr '" + errBuf.String() + "' err: " + err.Error())
		return ServiceNeeds(""), err
	}
	needs := StrToServiceNeeds(strings.TrimSpace(buf.String()))
	if needs == ServiceNeedsInvalid {
		return ServiceNeeds(""), errors.New("t3c-determine-restart returned unknown string '" + buf.String() + "'")
	}
	return needs, nil
}

// makePath takes the command to run, and returns an absolute path to it if it's inside the running executable's directory.
// If it isn't, cmd is returned unchanged.
//
// This allows executing the command to work even if it isn't in the PATH, if it's in the same dir as t3c. And if it isn't, fall back to PATH.
//
// Note this is not guaranteed to work. For example, if the executable is a symlink.
// If it fails to get the path, it simply returns the cmd, and then the cmd will have to be in the PATH to be executed successfully.
//
func makePath(cmd string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return cmd, errors.New("getting executable path: " + err.Error())
	}
	if exePath, err = filepath.EvalSymlinks(exePath); err != nil {
		return cmd, errors.New("evaluating executable path symlinks: " + err.Error())
	}
	fullPath := filepath.Join(filepath.Dir(exePath), cmd)
	if _, err := os.Stat(fullPath); err != nil {
		return cmd, err // likely to be os.IsNotExist(err)
	}
	return fullPath, nil
}
