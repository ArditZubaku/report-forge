package fixtures

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ArditZubaku/async-api/config"
	"github.com/ArditZubaku/async-api/store"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/require"
)

type TestEnv struct {
	DB     *sql.DB
	Config *config.Config
}

func NewTestEnv(t *testing.T) *TestEnv {
	if err := os.Setenv("ENV", string(config.EnvTest)); err != nil {
		t.FailNow()
	}

	conf, err := config.New()
	require.NoError(t, err)

	db, err := store.NewPG(conf.DatabaseURL())
	require.NoError(t, err)

	return &TestEnv{
		DB:     db,
		Config: conf,
	}
}

func (te *TestEnv) SetupDB(t *testing.T) func(t *testing.T) {
	m, err := migrate.New("file://../migrations", te.Config.DatabaseURL())
	require.NoError(t, err)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	return te.tearDownDB
}

func (te *TestEnv) tearDownDB(t *testing.T) {
	_, err := te.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s;", strings.Join([]string{"users", "refresh_tokens", "reports"}, ",")))
	require.NoError(t, err)
}
