package version

import (
	"fmt"
	"runtime"
)

// 版本信息 - 编译时通过ldflags注入
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
	OS        = runtime.GOOS
	Arch      = runtime.GOARCH
)

// Info 版本信息结构
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get 获取版本信息
func Get() *Info {
	return &Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		OS:        OS,
		Arch:      Arch,
	}
}

// String 格式化输出版本信息
func (v *Info) String() string {
	return fmt.Sprintf(`Orbit Version: %s
Git Commit:    %s
Build Date:    %s
Go Version:    %s
OS/Arch:       %s/%s`,
		v.Version,
		v.GitCommit,
		v.BuildDate,
		v.GoVersion,
		v.OS,
		v.Arch,
	)
}
