package app_ports

import (
	"database/sql"
	"time"

	"github.com/barelyhuman/caddy-ui/utils"
	"github.com/blockloop/scan/v2"
)

type AppPorts struct {
	Port      string        `db:"app_ports.port"`
	AppId     int64         `db:"app_ports.app_id"`
	DomainId  sql.NullInt64 `db:"app_ports.domain_id"`
	CreatedAt time.Time     `db:"app_ports.created_at"`
	UpdatedAt time.Time     `db:"app_ports.updated_at"`
}

type AppPortsWithIdentifier struct {
	ID int64 `db:"app_ports.id"`
	AppPorts
}

func New() *AppPorts {
	return &AppPorts{}
}

func DeleteByAppId(db *sql.DB, appId string) error {
	_, err := db.Exec("delete from app_ports where app_id = ?", appId)
	return err
}

func FindByAppId(db *sql.DB, appId string) (*AppPortsWithIdentifier, error) {
	var record AppPortsWithIdentifier

	cols, _ := scan.Columns(&record)

	res, err := db.Query(`select `+utils.Join(cols, ",")+` from app_ports where app_id = ? limit 1`, appId)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	err = scan.Row(&record, res)

	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (a *AppPorts) Save(db *sql.DB) (*AppPortsWithIdentifier, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `insert into app_ports (port,app_id,domain_id) values(?,?,?)`

	res, err := tx.Exec(query,
		a.Port,
		a.AppId,
		a.DomainId,
	)

	if err != nil {
		return nil, err
	}

	result := &AppPortsWithIdentifier{
		AppPorts: *a,
	}
	result.ID, err = res.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return result, nil
}
