package domains

import (
	"database/sql"
	"strings"
	"time"

	"github.com/blockloop/scan/v2"
)

type Domains struct {
	Domain    string `db:"domains.domain"`
	AppID     int64  `db:"domains.app_id"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DomainsWithIdentifier struct {
	ID int64 `db:"domains.id"`
	Domains
}

func New() *Domains {
	return &Domains{}
}

func FindByAppId(db *sql.DB, id string) (*DomainsWithIdentifier, error) {
	var record DomainsWithIdentifier
	res, err := db.Query(`select * from domains where app_id = ? limit 1`, id)

	if err != nil {
		return nil, err
	}
	defer res.Close()

	err = scan.Row(&record, res)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return &record, nil
		}
		return nil, err
	}
	return &record, nil
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
