package healthcheck

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SQLConnectionCheck returns a checking function that checks the connection state of the given sql database connection
func SQLConnectionCheck(db *sql.DB, acceptablePing ...time.Duration) CheckingFunc {
	if db == nil {
		return func(context.Context) (*CheckingState, error) {
			return nil, errors.New("db is nil")
		}
	}

	return PingCheck(PingFunc(func(ctx context.Context) error {
		return db.PingContext(ctx)
	}), acceptablePing...)
}

// PostgresTableFullPermissionCheck returns a function that checks full-access permissions of the given tables
// Full-access: SELECT, DELETE, UPDATE, and INSERT
func PostgresTableFullPermissionCheck(db *sql.DB, checkTables []string) CheckingFunc {
	if db == nil {
		return func(context.Context) (*CheckingState, error) {
			return nil, errors.New("db is nil")
		}
	} else if len(checkTables) <= 0 {
		return func(context.Context) (*CheckingState, error) {
			return &CheckingState{
				State:  StateHealthy,
				Output: "Skip check because of no tables given",
			}, nil
		}
	}

	return func(ctx context.Context) (*CheckingState, error) {

		authorizedTables, err := queryAuthorizedTables(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("failed to query privileges from the database: %w", err)
		}
		isAuthorized := func(name string) bool {
			for _, t := range authorizedTables {
				if name == t {
					return true
				}
			}
			return false
		}

		unauthorisedTables := make([]string, 0, len(checkTables))
		for _, table := range checkTables {
			if !isAuthorized(table) {
				unauthorisedTables = append(unauthorisedTables, table)
			}
		}

		if len(unauthorisedTables) > 0 {
			return &CheckingState{
				State:  StateUnhealthy,
				Output: fmt.Sprintf("The following tables do not have sufficient permissions (SELECT, DELETE, UPDATE, INSERT) or do not exist: %v", unauthorisedTables),
			}, nil
		}
		return &CheckingState{
			State:  StateHealthy,
			Output: "All tables have full access",
		}, nil
	}
}

func queryAuthorizedTables(ctx context.Context, db *sql.DB) ([]string, error) {
	sql := `WITH 
	table_privileges AS
		(SELECT table_name, array_agg(privilege_type::text) AS privileges
		FROM information_schema.table_privileges
		WHERE grantee IN (SELECT CURRENT_USER)
		GROUP BY table_name), 
	full_privileges_table AS 
		(SELECT table_name,
		'SELECT' = any(privileges) as selectable,
		'DELETE' = any(privileges) as deletable,
		'UPDATE' = any(privileges) as updatable,
		'INSERT' = any(privileges) as insertable
		FROM table_privileges)
	SELECT table_name 
	FROM full_privileges_table
	WHERE selectable AND deletable AND updatable AND insertable`

	rows, err := db.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}
