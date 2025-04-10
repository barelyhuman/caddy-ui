package domains

import (
	"database/sql"
	"time"
)

type Domains struct {
	Domain    string
	AppID     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DomainsWithIdentifier struct {
	ID int64
	Domains
}

func New() *Domains {
	return &Domains{}
}

func FindById(db *sql.DB) *DomainsWithIdentifier {
	return &DomainsWithIdentifier{}
}

func (a *Domains) Save(db *sql.DB) (*DomainsWithIdentifier, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `insert into domains(domain,app_id) values(?,?)`

	res, err := tx.Exec(query,
		a.Domain,
		a.AppID,
	)

	if err != nil {
		return nil, err
	}

	result := &DomainsWithIdentifier{
		Domains: *a,
	}
	result.ID, err = res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return result, nil
}
