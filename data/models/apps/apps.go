package apps

import (
	"database/sql"
	"time"
)

type Apps struct {
	Name       string
	InstanceID int64
	Type       sql.NullString
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AppsWithIdentifier struct {
	ID int64
	Apps
}

func New() *Apps {
	return &Apps{}
}

func FindById(db *sql.DB) *AppsWithIdentifier {
	return &AppsWithIdentifier{}
}

func (a *Apps) Save(db *sql.DB) (*AppsWithIdentifier, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `insert into apps (name,instance_id,type) values (?,?,?)`

	res, err := tx.Exec(query,
		a.Name,
		a.InstanceID,
		a.Type,
	)

	if err != nil {
		return nil, err
	}

	result := &AppsWithIdentifier{
		Apps: *a,
	}
	result.ID, err = res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return result, nil
}
