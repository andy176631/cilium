// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

package cmd

import (
	"embed"
	_ "embed"
	"fmt"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/cilium/cilium/pkg/cmdref"
	"github.com/cilium/cilium/pkg/option"
)

//go:embed assets/*.yaml
var cmdrefFlagsFS embed.FS

func validateConfigmapCmd() *cobra.Command {
	var configMapDir string
	cmd := &cobra.Command{
		Use:   "validate-configmap",
		Short: "Validate Cilium ConfigMap for unrecognized keys in the daemon and operator.",
		Long: `Before upgrading Cilium, it is recommended to run this validation checker to 
ensure that the deployed Cilium ConfigMap is valid. The validator verifies that all configuration
keys are recognized by both the daemon and the operator. If any unrecognized keys are found, an
error is printed and the command exits with a non-zero status code.`,
	}

	cmd.Flags().StringVar(&configMapDir, "configmap-dir", "",
		"Path to a directory mounted from a Kubernetes ConfigMap; all files in this directory will be loaded as configuration")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := validateUnrecognizedKeys(configMapDir); err != nil {
			Fatalf("%s", err)
		}
	}
	return cmd
}

func loadCmdRefFlagNames() (map[string]struct{}, error) {
	const dir = "assets"

	files, err := cmdrefFlagsFS.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	flagNames := make(map[string]struct{}, 0)

	for _, e := range files {
		filePath := path.Join(dir, e.Name())
		flags, err := readFlagEntries(filePath)
		if err != nil {
			return nil, err // 已經帶 context
		}
		for _, f := range flags {
			if f.Name == "" {
				return nil, fmt.Errorf("empty flag name in %s", filePath)
			}
			flagNames[f.Name] = struct{}{}
		}
	}

	return flagNames, nil
}

func readFlagEntries(filePath string) ([]cmdref.FlagEntry, error) {
	data, err := cmdrefFlagsFS.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	var flags []cmdref.FlagEntry
	if err := yaml.Unmarshal(data, &flags); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", filePath, err)
	}

	return flags, nil
}

func validateUnrecognizedKeys(configMapDir string) error {
	var err error
	var recognizedKeys map[string]struct{}
	var cm map[string]any
	var unrecognized []string

	if cm, err = option.ReadDirConfig(log, configMapDir); err != nil {
		return err
	}
	if recognizedKeys, err = loadCmdRefFlagNames(); err != nil {
		return err
	}

	for k := range cm {
		if _, ok := recognizedKeys[k]; !ok {
			unrecognized = append(unrecognized, k)
		}
	}

	if len(unrecognized) == 0 {
		fmt.Println("All keys are recognized.")
		return nil
	}

	sort.Strings(unrecognized)
	return fmt.Errorf(
		"unrecognized keys detected:\n  - %s",
		strings.Join(unrecognized, "\n  - "),
	)
}
