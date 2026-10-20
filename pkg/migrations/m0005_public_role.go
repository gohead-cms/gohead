package migrations

import (
	"fmt"

	"github.com/gohead-cms/gohead/pkg/logger"

	"gorm.io/gorm"
)

// Migration 0005 makes room for the public role (anonymous visitors): a
// custom role named "public" created before it existed is renamed, so its
// accounts and tokens keep their permissions and anonymous visitors do not
// inherit them.

type role0005 struct {
	ID   uint
	Name string
}

func (role0005) TableName() string { return "user_roles" }

func publicRoleUp(db *gorm.DB) error {
	if !db.Migrator().HasTable("user_roles") {
		return nil
	}
	var role role0005
	err := db.Where("name = ?", "public").Limit(1).Find(&role).Error
	if err != nil || role.ID == 0 {
		return err
	}
	name := "public-legacy"
	for i := 2; ; i++ {
		var n int64
		if err := db.Table("user_roles").Where("name = ?", name).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			break
		}
		name = fmt.Sprintf("public-legacy-%d", i)
	}
	if err := db.Table("user_roles").Where("id = ?", role.ID).Update("name", name).Error; err != nil {
		return err
	}
	if db.Migrator().HasTable("api_tokens") {
		if err := db.Table("api_tokens").Where("role = ?", "public").Update("role", name).Error; err != nil {
			return err
		}
	}
	logger.Log.Warnf("The custom role \"public\" was renamed to %q: \"public\" is now the role of anonymous visitors. Its users and API tokens keep their permissions.", name)
	return nil
}

// publicRoleDown keeps the renamed role: renaming it back would give its
// permissions to anonymous visitors.
func publicRoleDown(*gorm.DB) error { return nil }
