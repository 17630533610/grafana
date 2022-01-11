package models

type QueryHistory struct {
	Id            int64
	Uid           string
	DatasourceUid string
	OrgId         int64
	CreatedBy     int64
	CreatedAt     int64
	Comment       string
	Queries       string
}
