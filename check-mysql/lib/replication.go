package checkmysql

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx"
	"github.com/mackerelio/checkers"
)

type replicationOpts struct {
	mysqlSetting
	Crit int64 `short:"c" long:"critical" default:"250" description:"critical if the seconds behind master is over"`
	Warn int64 `short:"w" long:"warning" default:"200" description:"warning if the seconds behind master is over"`
}

type slaveStatus struct {
	SlaveIORunning      string        `db:"Slave_IO_Running"`
	SlaveSQLRunning     string        `db:"Slave_SQL_Running"`
	SecondsBehindMaster sql.NullInt64 `db:"Seconds_Behind_Master"`
}

func (r *slaveStatus) ioRunning() string {
	return r.SlaveIORunning
}

func (r *slaveStatus) sqlRunning() string {
	return r.SlaveSQLRunning
}

func (r *slaveStatus) secondsBehind() sql.NullInt64 {
	return r.SecondsBehindMaster
}

func checkReplication(args []string) *checkers.Checker {
	opts := replicationOpts{}
	psr := flags.NewParser(&opts, flags.Default)
	psr.Usage = "replication [OPTIONS]"
	_, err := psr.ParseArgs(args)
	if err != nil {
		os.Exit(exitCode(err))
	}
	db, err := newDB(opts.mysqlSetting)
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Couldn't open DB: %s", err))
	}
	defer db.Close()

	sqlxDb := sqlx.NewDb(db, "mysql")
	defer sqlxDb.Close()

	// MariaDB names it SHOW REPLICA STATUS only since 10.5.1; SHOW SLAVE STATUS
	// works everywhere. Ignore columns which do not exist in the struct.
	rows, err := sqlxDb.Unsafe().Queryx("SHOW SLAVE STATUS")
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Couldn't execute query: %s", err))
	}
	defer rows.Close()

	if !rows.Next() {
		return checkers.Ok("MySQL is not a replica")
	}

	status := &slaveStatus{}

	err = rows.StructScan(status)
	if err != nil {
		return checkers.Unknown(fmt.Sprintf("Couldn't scan row: %s", err))
	}

	if !(status.ioRunning() == "Yes" && status.sqlRunning() == "Yes") {
		return checkers.Critical("MySQL replication has been stopped")
	}

	checkSt := checkers.OK
	secondsBehind := status.secondsBehind()
	if !secondsBehind.Valid {
		return checkers.Unknown("Unknown seconds behind in MySQL replication")
	}

	msg := fmt.Sprintf("MySQL replication behind master %d seconds", secondsBehind.Int64)
	if secondsBehind.Int64 > opts.Crit {
		checkSt = checkers.CRITICAL
	} else if secondsBehind.Int64 > opts.Warn {
		checkSt = checkers.WARNING
	}
	return checkers.NewChecker(checkSt, msg)
}
