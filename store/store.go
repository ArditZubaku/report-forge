package store

import "database/sql"

type Store struct {
	Users         *UserStore
	RefreshTokens *RefreshTokenStore
	Reports       *ReportStore
}

func New(db *sql.DB) *Store {
	return &Store{
		Users:         NewUserStore(db),
		RefreshTokens: NewRefreshTokenStore(db),
		Reports:       NewReportStore(db),
	}
}
