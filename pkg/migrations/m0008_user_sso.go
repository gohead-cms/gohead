package migrations

import "gorm.io/gorm"

// Migration 0008 links accounts to an identity provider (sign-in with
// Google).

type user0008 struct {
	SSOProvider string `gorm:"size:32;index:idx_users_sso,priority:1"`
	SSOSubject  string `gorm:"size:191;index:idx_users_sso,priority:2"`
}

func (user0008) TableName() string { return "users" }

func userSSOUp(tx *gorm.DB) error {
	m := tx.Migrator()
	for _, col := range []string{"SSOProvider", "SSOSubject"} {
		if !m.HasColumn(&user0008{}, col) {
			if err := m.AddColumn(&user0008{}, col); err != nil {
				return err
			}
		}
	}
	if !m.HasIndex(&user0008{}, "idx_users_sso") {
		return m.CreateIndex(&user0008{}, "idx_users_sso")
	}
	return nil
}

func userSSODown(tx *gorm.DB) error {
	m := tx.Migrator()
	if m.HasIndex(&user0008{}, "idx_users_sso") {
		if err := m.DropIndex(&user0008{}, "idx_users_sso"); err != nil {
			return err
		}
	}
	for _, col := range []string{"SSOSubject", "SSOProvider"} {
		if m.HasColumn(&user0008{}, col) {
			if err := m.DropColumn(&user0008{}, col); err != nil {
				return err
			}
		}
	}
	return nil
}
