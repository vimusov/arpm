package main

import (
	"github.com/spf13/cobra"
	"os"
)

func main() {
	initSettings := func(cmd *cobra.Command, args []string) error {
		return loadConfig()
	}

	var rootCmd = &cobra.Command{
		Use:               "arpm <command>",
		Short:             "ArchLinux repository management tool.",
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	}

	var serverCmd = &cobra.Command{
		Use:   "server <dir>",
		Short: "Lunch the repository management server.",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runServer(args[0]) },
	}
	serverCmd.Flags().BoolVarP(
		&debugMode,
		"debug", "d", false,
		"Enable debug mode.",
	)

	var branchesCmd = &cobra.Command{
		Use:   "branches",
		Short: "Manage branches on the server.",
	}
	var listBranchesCmd = &cobra.Command{
		Use:     "ls",
		Short:   "List branches on the server.",
		PreRunE: initSettings,
		RunE:    func(cmd *cobra.Command, args []string) error { return listBranches() },
	}
	var createBranchCmd = &cobra.Command{
		Use:     "mk <name>",
		Short:   "Create a new branch on the server.",
		Args:    cobra.ExactArgs(1),
		PreRunE: initSettings,
		RunE:    func(cmd *cobra.Command, args []string) error { return createBranch(args[0]) },
	}
	branchesCmd.AddCommand(listBranchesCmd)
	branchesCmd.AddCommand(createBranchCmd)

	var pkgsCommands = &cobra.Command{
		Use:   "pkgs",
		Short: "Manage packages in the branch.",
	}
	var listPkgsCmd = &cobra.Command{
		Use:     "ls <branch>",
		Short:   "List packages in the branch.",
		Args:    cobra.ExactArgs(1),
		PreRunE: initSettings,
		RunE:    func(cmd *cobra.Command, args []string) error { return listPackages(args[0]) },
	}
	var getPkgCmd = &cobra.Command{
		Use:     "get <branch> <name> [names...]",
		Short:   "Get package(s) from the branch.",
		Args:    cobra.ExactArgs(2),
		PreRunE: initSettings,
		RunE:    func(cmd *cobra.Command, args []string) error { return getPackage(args[0], args[1]) },
	}
	var putPkgCmd = &cobra.Command{
		Use:     "put <branch> <name> [names...]",
		Short:   "Put package(s) to the server.",
		Args:    cobra.MinimumNArgs(2),
		PreRunE: initSettings,
		RunE:    func(cmd *cobra.Command, args []string) error { return putPackages(args[0], args[1:]) },
	}
	var rmPkgCmd = &cobra.Command{
		Use:     "rm <branch> <name> [names...]",
		Short:   "Remove package(s) from the server.",
		Args:    cobra.MinimumNArgs(2),
		PreRunE: initSettings,
		RunE:    func(cmd *cobra.Command, args []string) error { return rmPackages(args[0], args[1:]) },
	}
	pkgsCommands.AddCommand(listPkgsCmd)
	pkgsCommands.AddCommand(getPkgCmd)
	pkgsCommands.AddCommand(putPkgCmd)
	pkgsCommands.AddCommand(rmPkgCmd)

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(branchesCmd)
	rootCmd.AddCommand(pkgsCommands)

	if execErr := rootCmd.Execute(); execErr != nil {
		logError(execErr, "Failed to execute command")
		os.Exit(1)
	}
}
