package app_ports

import (
	"database/sql"
	"time"
)

type AppPorts struct {
	Port      string
	AppId     int64
	DomainId  sql.NullInt64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AppPortsWithIdentifier struct {
	ID int64
	AppPorts
}

func New() *AppPorts {
	return &AppPorts{}
}

func FindById(db *sql.DB) *AppPortsWithIdentifier {
	return &AppPortsWithIdentifier{}
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

	return result, nil
}
