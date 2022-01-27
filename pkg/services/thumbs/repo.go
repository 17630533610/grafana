package thumbs

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore"
)

func newThumbnailRepo(store *sqlstore.SQLStore) thumbnailRepo {
	repo := &sqlThumbnailRepository{
		store: store,
	}
	return repo
}

type sqlThumbnailRepository struct {
	store *sqlstore.SQLStore
}

func (r *sqlThumbnailRepository) saveFromFile(filePath string, meta models.DashboardThumbnailMeta, dashboardVersion int) (int64, error) {
	file, err := os.Open(filepath.Clean(filePath))

	if err != nil {
		tlog.Error("error opening file", "dashboardUID", meta.DashboardUID, "err", err)
		return 0, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			tlog.Warn("Failed to close thumbnail file", "path", filePath, "err", err)
		}
	}()

	reader := bufio.NewReader(file)
	content, err := ioutil.ReadAll(reader)

	if err != nil {
		tlog.Error("error reading file", "dashboardUID", meta.DashboardUID, "err", err)
		return 0, err
	}

	return r.saveFromBytes(content, getMimeType(filePath), meta, dashboardVersion)
}

func getMimeType(filePath string) string {
	if strings.HasSuffix(filePath, ".webp") {
		return "image/webp"
	}

	return "image/png"
}

func (r *sqlThumbnailRepository) saveFromBytes(content []byte, mimeType string, meta models.DashboardThumbnailMeta, dashboardVersion int) (int64, error) {
	cmd := &models.SaveDashboardThumbnailCommand{
		DashboardThumbnailMeta: meta,
		Image:                  content,
		MimeType:               mimeType,
		DashboardVersion:       dashboardVersion,
	}

	_, err := r.store.SaveThumbnail(cmd)
	if err != nil {
		tlog.Error("error saving to the db", "dashboardUID", meta.DashboardUID, "err", err)
		return 0, err
	}

	return cmd.Result.Id, nil
}

func (r *sqlThumbnailRepository) updateThumbnailState(state models.ThumbnailState, meta models.DashboardThumbnailMeta) error {
	return r.store.UpdateThumbnailState(&models.UpdateThumbnailStateCommand{
		State:                  state,
		DashboardThumbnailMeta: meta,
	})
}

func (r *sqlThumbnailRepository) getThumbnail(meta models.DashboardThumbnailMeta) (*models.DashboardThumbnail, error) {
	query := &models.GetDashboardThumbnailCommand{
		DashboardThumbnailMeta: meta,
	}
	return r.store.GetThumbnail(query)
}

func (r *sqlThumbnailRepository) findDashboardsWithStaleThumbnails() ([]*models.DashboardWithStaleThumbnail, error) {
	return r.store.FindDashboardsWithStaleThumbnails(&models.FindDashboardsWithStaleThumbnailsCommand{
		IncludeManuallyUploadedThumbnails: false,
	})
}
