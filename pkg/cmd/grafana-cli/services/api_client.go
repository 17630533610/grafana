package services

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"

	"github.com/grafana/grafana/pkg/cmd/grafana-cli/logger"
	"github.com/grafana/grafana/pkg/cmd/grafana-cli/models"
	"github.com/grafana/grafana/pkg/util/errutil"
)

type GrafanaComClient struct {
	retryCount int
}

func (client *GrafanaComClient) GetPlugin(pluginID, repoURL string) (models.Plugin, error) {
	logger.Debugf("Getting plugin metadata from %v, plugin ID: %v", repoURL, pluginID)
	body, err := sendRequestGetBytes(HttpClient, repoURL, "repo", pluginID)
	if err != nil {
		if errors.Is(err, ErrNotFoundError) {
			return models.Plugin{}, errutil.Wrap("failed to find requested plugin, check if the plugin ID is correct", err)
		}
		return models.Plugin{}, errutil.Wrap("failed to send request", err)
	}

	var data models.Plugin
	if err := json.Unmarshal(body, &data); err != nil {
		logger.Error("Failed to unmarshal plugin repo response:", err)
		return models.Plugin{}, err
	}

	return data, nil
}

func (client *GrafanaComClient) DownloadFile(pluginName string, tmpFile *os.File, url string, checksum string) (err error) {
	// Try handling URL as a local file path first
	if _, err := os.Stat(url); err == nil {
		// We can ignore this gosec G304 warning since `url` stems from command line flag "pluginUrl". If the
		// user shouldn't be able to read the file, it should be handled through filesystem permissions.
		// nolint:gosec
		f, err := os.Open(url)
		if err != nil {
			return errutil.Wrap("failed to read plugin archive", err)
		}
		_, err = io.Copy(tmpFile, f)
		if err != nil {
			return errutil.Wrap("failed to copy plugin archive", err)
		}
		return nil
	}

	client.retryCount = 0

	defer func() {
		if r := recover(); r != nil {
			client.retryCount++
			if client.retryCount < 3 {
				logger.Info("Failed downloading. Will retry once.")
				err = tmpFile.Truncate(0)
				if err != nil {
					return
				}
				_, err = tmpFile.Seek(0, 0)
				if err != nil {
					return
				}
				err = client.DownloadFile(pluginName, tmpFile, url, checksum)
			} else {
				client.retryCount = 0
				failure := fmt.Sprintf("%v", r)
				if failure == "runtime error: makeslice: len out of range" {
					err = fmt.Errorf("corrupt HTTP response from source, please try again")
				} else {
					panic(r)
				}
			}
		}
	}()

	// Using no timeout here as some plugins can be bigger and smaller timeout would prevent to download a plugin on
	// slow network. As this is CLI operation hanging is not a big of an issue as user can just abort.
	bodyReader, err := sendRequest(HttpClientNoTimeout, url)
	if err != nil {
		return errutil.Wrap("failed to send request", err)
	}
	defer func() {
		if err := bodyReader.Close(); err != nil {
			logger.Warn("Failed to close body", "err", err)
		}
	}()

	w := bufio.NewWriter(tmpFile)
	h := md5.New()
	if _, err = io.Copy(w, io.TeeReader(bodyReader, h)); err != nil {
		return errutil.Wrap("failed to compute MD5 checksum", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to write to %q: %w", tmpFile.Name(), err)
	}
	if len(checksum) > 0 && checksum != fmt.Sprintf("%x", h.Sum(nil)) {
		return fmt.Errorf("expected MD5 checksum does not match the downloaded archive - please contact security@grafana.com")
	}

	return nil
}

func (client *GrafanaComClient) ListAllPlugins(repoURL string) (models.PluginRepo, error) {
	body, err := sendRequestGetBytes(HttpClient, repoURL, "repo")

	if err != nil {
		logger.Error("Failed to send request", "error", err)
		return models.PluginRepo{}, errutil.Wrap("Failed to send request", err)
	}

	var data models.PluginRepo
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error("Failed to unmarshal plugin repo response", "error", err)
		return models.PluginRepo{}, err
	}

	return data, nil
}

func sendRequestGetBytes(client http.Client, repoURL string, subPaths ...string) ([]byte, error) {
	bodyReader, err := sendRequest(client, repoURL, subPaths...)
	if err != nil {
		return []byte{}, err
	}
	defer func() {
		if err := bodyReader.Close(); err != nil {
			logger.Warn("Failed to close stream", "err", err)
		}
	}()
	return ioutil.ReadAll(bodyReader)
}

func sendRequest(client http.Client, repoURL string, subPaths ...string) (io.ReadCloser, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}

	for _, v := range subPaths {
		u.Path = path.Join(u.Path, v)
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("grafana-version", grafanaVersion)
	req.Header.Set("grafana-os", runtime.GOOS)
	req.Header.Set("grafana-arch", runtime.GOARCH)
	req.Header.Set("User-Agent", "grafana "+grafanaVersion)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return handleResponse(res)
}

func handleResponse(res *http.Response) (io.ReadCloser, error) {
	if res.StatusCode == 404 {
		return nil, ErrNotFoundError
	}

	if res.StatusCode/100 != 2 && res.StatusCode/100 != 4 {
		return nil, fmt.Errorf("API returned invalid status: %s", res.Status)
	}

	if res.StatusCode/100 == 4 {
		body, err := ioutil.ReadAll(res.Body)
		defer func() {
			if err := res.Body.Close(); err != nil {
				logger.Warn("Failed to close response body", "err", err)
			}
		}()
		if err != nil || len(body) == 0 {
			return nil, &BadRequestError{Status: res.Status}
		}
		var message string
		var jsonBody map[string]string
		err = json.Unmarshal(body, &jsonBody)
		if err != nil || len(jsonBody["message"]) == 0 {
			message = string(body)
		} else {
			message = jsonBody["message"]
		}
		return nil, &BadRequestError{Status: res.Status, Message: message}
	}

	return res.Body, nil
}
