package repository

import "fmt"

type PluginArchiveInfo struct {
	ID           string
	Version      string
	Dependencies map[string]*PluginArchiveInfo
	Path         string
}

type PluginDownloadOptions struct {
	Version      string
	PluginZipURL string
}

type InstalledPlugin struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	Info         PluginInfo   `json:"info"`
	Dependencies Dependencies `json:"dependencies"`
}

type Dependencies struct {
	GrafanaVersion string             `json:"grafanaVersion"`
	Plugins        []PluginDependency `json:"plugins"`
}

type PluginDependency struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PluginInfo struct {
	Version string `json:"version"`
	Updated string `json:"updated"`
}

type Plugin struct {
	ID       string    `json:"id"`
	Category string    `json:"category"`
	Versions []Version `json:"versions"`
}

type Version struct {
	Commit  string              `json:"commit"`
	URL     string              `json:"repoURL"`
	Version string              `json:"version"`
	Arch    map[string]ArchMeta `json:"arch"`
}

type ArchMeta struct {
	SHA256 string `json:"sha256"`
}

type PluginRepo struct {
	Plugins []Plugin `json:"plugins"`
	Version string   `json:"version"`
}

type Response4xxError struct {
	Message    string
	StatusCode int
	SystemInfo string
}

func (e Response4xxError) Error() string {
	if len(e.Message) > 0 {
		if len(e.SystemInfo) > 0 {
			return fmt.Sprintf("%s (%s)", e.Message, e.SystemInfo)
		}
		return fmt.Sprintf("%d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%d", e.StatusCode)
}

type ErrVersionUnsupported struct {
	PluginID         string
	RequestedVersion string
	SystemInfo       string
}

func (e ErrVersionUnsupported) Error() string {
	return fmt.Sprintf("%s v%s is not supported on your system (%s)", e.PluginID, e.RequestedVersion, e.SystemInfo)
}

type ErrVersionNotFound struct {
	PluginID         string
	RequestedVersion string
	SystemInfo       string
}

func (e ErrVersionNotFound) Error() string {
	return fmt.Sprintf("%s v%s either does not exist or is not supported on your system (%s)", e.PluginID, e.RequestedVersion, e.SystemInfo)
}

type ErrPermissionDenied struct {
	Path string
}

func (e ErrPermissionDenied) Error() string {
	return fmt.Sprintf("could not create %q, permission denied, make sure you have write access to plugin dir", e.Path)
}
