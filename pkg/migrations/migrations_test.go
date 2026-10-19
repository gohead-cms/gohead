package migrations

import (
	"errors"
	"testing"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// withMigrations replaces All for one test.
func withMigrations(t *testing.T, list ...Migration) {
	t.Helper()
	prev := All
	All = list
	t.Cleanup(func() { All = prev })
}

func TestBaselineAndStatus(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	require.Error(t, Check(db), "fresh database: baseline pending")

	applied, err := Up(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"0001_baseline"}, applied)
	assert.True(t, db.Migrator().HasTable(&models.Item{}))
	assert.NoError(t, Check(db))

	applied, err = Up(db)
	require.NoError(t, err)
	assert.Empty(t, applied, "runs once")

	states, err := Status(db)
	require.NoError(t, err)
	require.Len(t, states, len(All))
	assert.NotNil(t, states[0].AppliedAt)
	assert.False(t, states[0].Reversible)

	_, err = Down(db, 1)
	assert.Error(t, err, "the baseline cannot be undone")
}

type widget struct {
	ID   uint
	Name string
}

func TestUpDownInOrder(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	var order []string
	withMigrations(t,
		Migration{ID: "0002_column", Description: "add a column",
			Up: func(tx *gorm.DB) error {
				order = append(order, "0002")
				return tx.Migrator().AddColumn(&widget{}, "Name")
			},
			Down: func(tx *gorm.DB) error { return tx.Migrator().DropColumn(&widget{}, "Name") },
		},
		// Declared out of order on purpose: IDs decide.
		Migration{ID: "0001_table", Description: "create a table",
			Up: func(tx *gorm.DB) error {
				order = append(order, "0001")
				return tx.Exec("CREATE TABLE widgets (id integer primary key)").Error
			},
			Down: func(tx *gorm.DB) error { return tx.Migrator().DropTable("widgets") },
		},
	)

	applied, err := Up(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"0001_table", "0002_column"}, applied)
	assert.Equal(t, []string{"0001", "0002"}, order)
	assert.True(t, db.Migrator().HasColumn(&widget{}, "Name"))

	reverted, err := Down(db, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"0002_column"}, reverted)
	assert.False(t, db.Migrator().HasColumn(&widget{}, "Name"))
	assert.True(t, errors.Is(Check(db), ErrPending))

	applied, err = Up(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"0002_column"}, applied, "re-applied after the revert")
}

func TestFailedMigrationIsNotRecorded(t *testing.T) {
	_, db := testutils.SetupTestServer()
	defer testutils.CleanupTestDB()

	withMigrations(t, Migration{ID: "0001_broken", Up: func(tx *gorm.DB) error { return errors.New("boom") }})
	_, err := Up(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0001_broken")
	assert.True(t, errors.Is(Check(db), ErrPending), "retried on the next run")
}
