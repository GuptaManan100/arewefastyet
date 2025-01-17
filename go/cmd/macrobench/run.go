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

package macrobench

import (
	"github.com/spf13/cobra"
	"github.com/vitessio/arewefastyet/go/mysql"
	"github.com/vitessio/arewefastyet/go/tools/macrobench"
)

func run() *cobra.Command {
	mabcfg := macrobench.Config{
		DatabaseConfig: &mysql.ConfigDB{},
	}

	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return macrobench.Run(mabcfg)
		},
	}
	mabcfg.AddToCommand(cmd)
	return cmd
}