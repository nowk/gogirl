package gogirl

import (
	"github.com/nowk/gogirl/sql"
)

type SQLFactory interface {
	Save(sql.DB) (interface{}, error)
}
