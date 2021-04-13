/*
 *
 * Copyright 2021 The Vitess Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 */

package ansible

import (
	"context"
	"errors"
	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/google/uuid"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"path"
)

const (
	// KeyExecUUID is the name of a key passed to each Ansible playbook
	// the value of the key points to an Execution UUID.
	KeyExecUUID = "arewefastyet_exec_uuid"

	ErrorPathUnknown = "path does not exist"

	flagAnsibleRoot    = "ansible-root-directory"
	flagInventoryFiles = "ansible-inventory-files"
	flagPlaybookFiles  = "ansible-playbook-files"
)

type Config struct {
	RootDir        string
	InventoryFiles []string
	PlaybookFiles  []string

	stdout io.Writer
	stderr io.Writer
}

func (c *Config) AddToViper(v *viper.Viper) {
	_ = v.UnmarshalKey(flagAnsibleRoot, &c.RootDir)
	_ = v.UnmarshalKey(flagInventoryFiles, &c.InventoryFiles)
	_ = v.UnmarshalKey(flagPlaybookFiles, &c.PlaybookFiles)
}

func (c *Config) AddToPersistentCommand(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&c.RootDir, flagAnsibleRoot, "", "Root directory of Ansible")
	cmd.PersistentFlags().StringSliceVar(&c.InventoryFiles, flagInventoryFiles, []string{}, "List of inventory files used by Ansible")
	cmd.PersistentFlags().StringSliceVar(&c.PlaybookFiles, flagPlaybookFiles, []string{}, "List of playbook files used by Ansible")

	_ = viper.BindPFlag(flagAnsibleRoot, cmd.Flags().Lookup(flagAnsibleRoot))
	_ = viper.BindPFlag(flagInventoryFiles, cmd.Flags().Lookup(flagInventoryFiles))
	_ = viper.BindPFlag(flagPlaybookFiles, cmd.Flags().Lookup(flagPlaybookFiles))
}

func applyRootToFiles(root string, files *[]string) {
	for i, file := range *files {
		if !path.IsAbs(file) {
			(*files)[i] = path.Join(root, file)
		}
	}
}

func inventoryFilesToString(invFiles []string) string {
	var res string
	for i, inv := range invFiles {
		if i > 0 {
			res = res + ", "
		}
		res = res + inv
	}
	return res
}

func Run(c *Config, execUUID uuid.UUID) error {
	applyRootToFiles(c.RootDir, &c.PlaybookFiles)
	applyRootToFiles(c.RootDir, &c.InventoryFiles)

	ansiblePlaybookConnectionOptions := &options.AnsibleConnectionOptions{
		User:          "root",
		SSHCommonArgs: "-o StrictHostKeyChecking=no",
	}

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		Inventory: inventoryFilesToString(c.InventoryFiles),
		ExtraVars: map[string]interface{}{
			KeyExecUUID: execUUID.String(),
		},
	}

	ansiblePlaybookPrivilegeEscalationOptions := &options.AnsiblePrivilegeEscalationOptions{
		Become: true,
	}

	plb := &playbook.AnsiblePlaybookCmd{
		Playbooks:                  c.PlaybookFiles,
		ConnectionOptions:          ansiblePlaybookConnectionOptions,
		PrivilegeEscalationOptions: ansiblePlaybookPrivilegeEscalationOptions,
		Options:                    ansiblePlaybookOptions,
		Exec: execute.NewDefaultExecute(
			execute.WithShowDuration(),
			execute.WithWrite(c.stdout),
			execute.WithWriteError(c.stderr),
		),
	}

	err := plb.Run(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) CopyRootDirectory(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return errors.New(ErrorPathUnknown)
	}

	err := copy.Copy(c.RootDir, directory)
	if err != nil {
		return err
	}
	c.RootDir = directory
	return nil
}

func (c *Config) SetStdout(stdout *os.File) {
	c.stdout = stdout
}

func (c *Config) SetStderr(stderr *os.File) {
	c.stderr = stderr
}

func (c *Config) SetOutputs(stdout, stderr *os.File) {
	c.stdout = stdout
	c.stderr = stderr
}