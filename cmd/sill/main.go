// Command sill is a thin CLI over the sill library — see
// landmarks/design/engineering/DEC-5xwjKMhHQC. No logic here beyond flag
// parsing, output formatting, and a direct call into the sill package.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/enyhma/sill"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "sill",
		Short: "A headless evaluator for Obsidian Bases (.base) files",
	}
	root.AddCommand(newQueryCmd(), newLintCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "sill:", err)
		os.Exit(1)
	}
}

func newQueryCmd() *cobra.Command {
	var vault, view, format string
	cmd := &cobra.Command{
		Use:   "query <file.base>",
		Short: "Run a view's filters against a vault and print matching notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := sill.ParseFile(args[0])
			if err != nil {
				return err
			}
			results, warnings, err := sill.Query(base, vaultOrDefault(vault, args[0]), view)
			if err != nil {
				return err
			}
			printWarnings(warnings)
			out, err := sill.FormatQueryResults(results, format)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&vault, "vault", "", "vault directory to evaluate against (default: the .base file's directory)")
	cmd.Flags().StringVar(&view, "view", "", "view to run (default: the base's first view, if any)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: json, csv, or table")
	return cmd
}

func newLintCmd() *cobra.Command {
	var vault string
	cmd := &cobra.Command{
		Use:   "lint <file.base>",
		Short: "Evaluate a base's sillAssert extension; exits non-zero below threshold",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := sill.ParseFile(args[0])
			if err != nil {
				return err
			}
			res, warnings, err := sill.Lint(base, vaultOrDefault(vault, args[0]))
			if err != nil {
				return err
			}
			printWarnings(warnings)
			fmt.Printf("%d/%d matched (%.1f%%) — %s\n",
				res.SatisfiedCount, res.PopulationCount, res.Percentage*100, passLabel(res.Passed))
			os.Exit(res.ExitCode())
			return nil
		},
	}
	cmd.Flags().StringVar(&vault, "vault", "", "vault directory to evaluate against (default: the .base file's directory)")
	return cmd
}

func vaultOrDefault(vault, basePath string) string {
	if vault != "" {
		return vault
	}
	return filepath.Dir(basePath)
}

// printWarnings reports non-fatal per-note load problems (e.g. an
// Obsidian template file with placeholder syntax that isn't valid YAML)
// without failing the command — see LoadVault.
func printWarnings(warnings []error) {
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "sill: warning:", w)
	}
}

func passLabel(passed bool) string {
	if passed {
		return "passed"
	}
	return "failed"
}
