package main

import (
	"context"
	"fmt"
	"log/slog"
	"surface-api/models"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) GetMappings(ctx context.Context, site string) ([]models.Mapping, error) {
	sp := sentry.StartSpan(ctx, "repo.GetMappings")
	defer sp.Finish()

	var result = []models.Mapping{}
	var table = site + "_mappings"

	site = "xx"

	err := r.DB.WithContext(ctx).Raw(fmt.Sprintf(`SELECT
		"%s" as site,
		%s.location,
		%s.surface_id,
		surfaces.name as surface_name
		FROM %s
		LEFT JOIN surfaces ON %s.surface_id = surfaces.id`,
		site, table, table, table, table)).Scan(&result).Error

	slog.Debug("in repo", slog.Int("number", 110), slog.String("name", "holla"))
	if err != nil {
		return result, errors.Wrap(err, "failed to get mappings")
	}
	return result, err
}
