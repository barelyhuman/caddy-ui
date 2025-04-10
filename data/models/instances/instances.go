package instances

import (
	"database/sql"
	"time"
)

type Instances struct {
	Password   string
	BaseDomain sql.NullString
	IsPrimary  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type InstancesWithIdentifier struct {
	ID int64
	Instances
}

func New() *Instances {
	return &Instances{}
}

func FindById(db *sql.DB) *InstancesWithIdentifier {
	return &InstancesWithIdentifier{}
}

func (a *Instances) Save(db *sql.DB) (*InstancesWithIdentifier, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `insert into (password,is_primary,base_domain) values(?,?,?)`

	res, err := tx.Exec(query,
		a.Password,
		a.IsPrimary,
		a.BaseDomain,
	)

	if err != nil {
		return nil, err
	}

	result := &InstancesWithIdentifier{
		Instances: *a,
	}

	result.ID, err = res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return result, nil
}
