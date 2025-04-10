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

func FindAll(db *sql.DB) ([]AppsWithIdentifier, error) {
	res, err := db.Query(`
		select id,name,instance_id,type,created_at,updated_at from apps
	`)
	if err != nil {
		return []AppsWithIdentifier{}, err
	}

	if res == nil {
		return []AppsWithIdentifier{}, nil
	}

	collection := []AppsWithIdentifier{}
	for res.Next() {
		x := AppsWithIdentifier{}
		res.Scan(
			&x.ID,
			&x.Name,
			&x.InstanceID,
			&x.Type,
			&x.CreatedAt,
			&x.UpdatedAt,
		)
		collection = append(collection, x)
	}
	return collection, nil
}

func FindById(db *sql.DB) *AppsWithIdentifier {
	return &AppsWithIdentifier{}
}

func (a *Apps) Save(db *sql.DB) (*AppsWithIdentifier, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	stmt, _ := tx.Prepare(`insert into apps (name,instance_id,type) values (?,?,?)`)

	res, err := stmt.Exec(
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

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return result, nil
}
