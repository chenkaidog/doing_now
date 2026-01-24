package errs

import "github.com/go-sql-driver/mysql"

func IsDuplicatedErr(err error) bool {
	if err == nil {
		return false
	}

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == 1062
	}

	return false
}
