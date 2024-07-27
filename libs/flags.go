package libs

import (
	"errors"
	"flag"
)

const (
	ApiSeverFlag       = "api-sever"
	ProjectNameFlag    = "project-name"
	ProjectVersionFlag = "project-version"
	FilenameFormatFlag = "filename-format"
)

var apiServer = "https://api.papermc.io"
var projectName = ""
var projectVersion = ""
var filenameFormat = "{project-name}-{project-version}.jar"

func ParseFlags() error {
	flag.StringVar(&apiServer, ApiSeverFlag, apiServer, "The url of PaperMC API")
	flag.StringVar(&projectName, ProjectNameFlag, projectName, "The project name such as paper or velocity.")
	flag.StringVar(&projectVersion, ProjectVersionFlag, projectVersion, "The project version such as 1.20.6, 1.21")
	flag.StringVar(&filenameFormat, FilenameFormatFlag, filenameFormat, "The filename to write downloaded jar data.")

	flag.Parse()

	if apiServer == "" {
		return errors.New("api-server is empty")
	}

	if projectName == "" {
		return errors.New("project-name is required")
	}

	if projectVersion == "" {
		return errors.New("project-version is required")
	}

	if filenameFormat == "" {
		return errors.New("filename-format is empty")
	}

	return nil
}

func GetApiServer() string {
	return apiServer
}

func GetProjectName() string {
	return projectName
}

func GetProjectVersion() string {
	return projectVersion
}

func GetFilenameFormat() string {
	return filenameFormat
}
