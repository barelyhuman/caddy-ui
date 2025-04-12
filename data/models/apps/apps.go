package apps

import (
	"database/sql"
	"time"
)

type Apps struct {
	Name       string         `db:"apps.name"`
	InstanceID int64          `db:"apps.instance_id"`
	Type       sql.NullString `db:"apps.type"`
	CreatedAt  time.Time      `db:"apps.created_at"`
	UpdatedAt  time.Time      `db:"apps.updated_at"`
}

type AppsWithIdentifier struct {
	ID int64 `db:"apps.id"`
	Apps
}

func New() *Apps {
	return &Apps{}
}

func DeleteById(db *sql.DB, id int64) error {
	_, err := db.Exec("delete from apps where id = ?", id)
	return err
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

func FindById(db *sql.DB, id string) (*AppsWithIdentifier, error) {
	var x AppsWithIdentifier
	res, err := db.Query(`
		select id,name,instance_id,type,created_at,updated_at from apps where id = ?
	`, id)
	if err != nil {
		return &x, err
	}
	defer res.Close()

	if res == nil {
		return &x, nil
	}

	for res.Next() {
		res.Scan(
			&x.ID,
			&x.Name,
			&x.InstanceID,
			&x.Type,
			&x.CreatedAt,
			&x.UpdatedAt,
		)
		return &x, nil
	}

	return &x, nil
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
