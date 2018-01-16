package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/generated/models"
)

func TestConfigNode(t *testing.T) {
	t.Parallel()
	db := testDB
	common.UseTable(db, "metadata")
	common.UseTable(db, "config_node")
	defer func() {
		common.ClearTable(db, "config_node")
		common.ClearTable(db, "metadata")
		if p := recover(); p != nil {
			panic(p)
		}
	}()
	model := models.MakeConfigNode()
	model.UUID = "config_node_dummy_uuid"
	model.FQName = []string{"default", "default-domain", "config_node_dummy"}
	model.Perms2.Owner = "admin"

	err := common.DoInTransaction(db, func(tx *sql.Tx) error {
		return CreateConfigNode(tx, model)
	})
	if err != nil {
		t.Fatal("create failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListConfigNode(tx, &common.ListSpec{Limit: 1})
		if err != nil {
			return err
		}
		if len(models) != 1 {
			return fmt.Errorf("expected one element")
		}
		return nil
	})
	if err != nil {
		t.Fatal("list failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		return DeleteConfigNode(tx, model.UUID,
			common.NewAuthContext("default", "demo", "demo", []string{}),
		)
	})
	if err == nil {
		t.Fatal("auth failed")
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		return DeleteConfigNode(tx, model.UUID, nil)
	})
	if err != nil {
		t.Fatal("delete failed", err)
	}

	err = common.DoInTransaction(db, func(tx *sql.Tx) error {
		models, err := ListConfigNode(tx, &common.ListSpec{Limit: 1})
		if err != nil {
			return err
		}
		if len(models) != 0 {
			return fmt.Errorf("expected no element")
		}
		return nil
	})
	if err != nil {
		t.Fatal("list failed", err)
	}
	return
}
