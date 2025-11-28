package main

import (
	"fmt"
	"log"

	"github.com/go-jet/jet/v2/generator/metadata"
	"github.com/go-jet/jet/v2/generator/postgres"
	"github.com/go-jet/jet/v2/generator/template"
	postgres2 "github.com/go-jet/jet/v2/postgres"
	_ "github.com/lib/pq"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	dsnKey    = "dsn"
	schemaKey = "schema"
	destKey   = "dest"
)

func main() {
	pflag.String(dsnKey, "", "DSN for database connection")
	pflag.String(schemaKey, "public", "Schema name for tables")
	pflag.String(destKey, "", "Destination dir for generated files")

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Fatalf("falied to bind flags: %v", err)
	}

	var (
		dsn    = viper.GetString(dsnKey)
		schema = viper.GetString(schemaKey)
		dest   = viper.GetString(destKey)
	)
	log.Printf("dsn: %s", dsn)
	log.Printf("schema: %s", schema)
	log.Printf("dest: %s", dest)

	err := postgres.GenerateDSN(dsn, schema, dest,
		template.Default(postgres2.Dialect).
			UseSchema(func(schemaMetaData metadata.Schema) template.Schema {
				return template.DefaultSchema(schemaMetaData).
					UseModel(template.DefaultModel().
						UseTable(func(table metadata.Table) template.TableModel {
							return template.DefaultTableModel(table).
								UseField(func(columnMetaData metadata.Column) template.TableModelField {
									defaultTableModelField := template.DefaultTableModelField(columnMetaData)
									// Добавляем тег db с именем столбца

									return defaultTableModelField.UseTags(
										fmt.Sprintf(`db:"%s.%s"`, table.Name, columnMetaData.Name),
									)
								})
						}),
					)
			}))

	if err != nil {
		log.Fatalf("generate models failed: %v", err)
	}
}
