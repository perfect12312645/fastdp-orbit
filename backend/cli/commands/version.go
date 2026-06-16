package commands

import (
	"fastdp-orbit/backend/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Get()
	 println(info.String())
	},
}
