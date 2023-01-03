package main

import (
	"log"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"gopkg.in/DATA-DOG/go-sqlmock.v2"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

//go:generate go run main.go ../../../docs/dev/database_schema.md

func main() {
	w, err := os.Create(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer func() { _ = w.Close() }()

	conn, _, err := sqlmock.New()
	if err != nil {
		log.Fatalf("Failed to get mock connection: %v", err)
	}
	defer func() { _ = conn.Close() }()

	dialectors := []gorm.Dialector{
		postgres.New(postgres.Config{
			Conn: conn,
		}),
		mysql.New(mysql.Config{
			Conn:                      conn,
			SkipInitializeWithVersion: true,
		}),
	}
	collected := make([][]*tableInfo, 0, len(dialectors))
	for i, dialector := range dialectors {
		tableInfos, err := generate(dialector)
		if err != nil {
			log.Fatalf("Failed to get table info of %d: %v", i, err)
		}

		collected = append(collected, tableInfos)
	}

	for i, ti := range collected[0] {
		_, _ = w.WriteString(`# Table "` + ti.Name + `"`)
		_, _ = w.WriteString("\n\n")

		_, _ = w.WriteString("```\n")

		table := tablewriter.NewWriter(w)
		table.SetHeader([]string{"Field", "Column", "PostgreSQL", "MySQL", "SQLite3"})
		table.SetBorder(false)
		for j, f := range ti.Fields {
			table.Append([]string{
				f.Name, f.Column,
				strings.ToUpper(f.Type),                         // PostgreSQL
				strings.ToUpper(collected[1][i].Fields[j].Type), // MySQL
				strings.ToUpper(collected[2][i].Fields[j].Type), // SQLite3
			})
		}
		table.Render()
		_, _ = w.WriteString("\n")

		_, _ = w.WriteString("Primary keys: ")
		_, _ = w.WriteString(strings.Join(ti.PrimaryKeys, ", "))
		_, _ = w.WriteString("\n")

		_, _ = w.WriteString("```\n\n")
	}
}

type tableField struct {
	Name   string
	Column string
	Type   string
}

type tableInfo struct {
	Name        string
	Fields      []*tableField
	PrimaryKeys []string
}

// This function is derived from gorm.io/gorm/migrator/migrator.go:Migrator.CreateTable.
func generate(dialector gorm.Dialector) ([]*tableInfo, error) {
	conn, err := gorm.Open(dialector, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "open database")
	}

	m := conn.Migrator().(interface {
		RunWithValue(value interface{}, fc func(*gorm.Statement) error) error
		FullDataTypeOf(*schema.Field) clause.Expr
	})
	tableInfos := make([]*tableInfo, 0, len(db.Tables))
	for _, table := range db.Tables {
		err = m.RunWithValue(table, func(stmt *gorm.Statement) error {
			fields := make([]*tableField, 0, len(stmt.Schema.DBNames))
			for _, field := range stmt.Schema.Fields {
				if field.DBName == "" {
					continue
				}

				fields = append(fields, &tableField{
					Name:   field.Name,
					Column: field.DBName,
					Type:   m.FullDataTypeOf(field).SQL,
				})
			}

			primaryKeys := make([]string, 0, len(stmt.Schema.PrimaryFields))
			if len(stmt.Schema.PrimaryFields) > 0 {
				for _, field := range stmt.Schema.PrimaryFields {
					primaryKeys = append(primaryKeys, field.DBName)
				}
			}

			tableInfos = append(tableInfos, &tableInfo{
				Name:        stmt.Table,
				Fields:      fields,
				PrimaryKeys: primaryKeys,
			})
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "gather table information")
		}
	}

	return tableInfos, nil
}
