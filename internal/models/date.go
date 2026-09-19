package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DateLayout is the only date format accepted from clients and in the database.
const DateLayout = "2006-01-02"

// Date is a calendar date without a time or a timezone, stored in a Postgres
// `date` column and marshalled as "YYYY-MM-DD" rather than the RFC 3339
// timestamp that time.Time would produce.
type Date time.Time

func (d Date) String() string {
	return time.Time(d).Format(DateLayout)
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// Value sends the date as a plain string so Postgres never has to convert a
// timestamp, which would shift the day by one in non-UTC server timezones.
func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *Date) Scan(value any) error {
	if value == nil {
		return nil
	}
	parsed, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("cannot scan %T into Date", value)
	}
	*d = Date(parsed)
	return nil
}
