// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package pgsql implements gdb.Driver, which supports operations for database PostgreSQL.
//
// Note:
// 1. It does not support Save/Replace features.
// 2. It does not support Insert Ignore features.
package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/lib/pq"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/gutil"
)

// Driver is the driver for postgresql database.
type Driver struct {
	*gdb.Core
}

const (
	internalPrimaryKeyInCtx gctx.StrKey = "primary_key"
	defaultSchema           string      = "public"
	quoteChar               string      = `"`
)

func init() {
	if err := gdb.Register(`pgsql`, New()); err != nil {
		panic(err)
	}
}

// New create and returns a driver that implements gdb.Driver, which supports operations for PostgreSql.
func New() gdb.Driver {
	return &Driver{}
}

// New creates and returns a database object for postgresql.
// It implements the interface of gdb.Driver for extra database driver installation.
func (d *Driver) New(core *gdb.Core, node *gdb.ConfigNode) (gdb.DB, error) {
	return &Driver{
		Core: core,
	}, nil
}

// Open creates and returns an underlying sql.DB object for pgsql.
// https://pkg.go.dev/github.com/lib/pq
func (d *Driver) Open(config *gdb.ConfigNode) (db *sql.DB, err error) {
	var (
		source               string
		underlyingDriverName = "postgres"
	)
	if config.Link != "" {
		// ============================================================================
		// Deprecated from v2.2.0.
		// ============================================================================
		source = config.Link
		// Custom changing the schema in runtime.
		if config.Name != "" {
			source, _ = gregex.ReplaceString(`dbname=([\w\.\-]+)+`, "dbname="+config.Name, source)
		}
	} else {
		if config.Name != "" {
			source = fmt.Sprintf(
				"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
				config.User, config.Pass, config.Host, config.Port, config.Name,
			)
		} else {
			source = fmt.Sprintf(
				"user=%s password=%s host=%s port=%s sslmode=disable",
				config.User, config.Pass, config.Host, config.Port,
			)
		}

		if config.Namespace != "" {
			source = fmt.Sprintf("%s search_path=%s", source, config.Namespace)
		}

		if config.Timezone != "" {
			source = fmt.Sprintf("%s timezone=%s", source, config.Timezone)
		}

		if config.Extra != "" {
			var extraMap map[string]interface{}
			if extraMap, err = gstr.Parse(config.Extra); err != nil {
				return nil, err
			}
			for k, v := range extraMap {
				source += fmt.Sprintf(` %s=%s`, k, v)
			}
		}
	}

	if db, err = sql.Open(underlyingDriverName, source); err != nil {
		err = gerror.WrapCodef(
			gcode.CodeDbOperationError, err,
			`sql.Open failed for driver "%s" by source "%s"`, underlyingDriverName, source,
		)
		return nil, err
	}
	return
}

// GetChars returns the security char for this type of database.
func (d *Driver) GetChars() (charLeft string, charRight string) {
	return quoteChar, quoteChar
}

// CheckLocalTypeForField checks and returns corresponding local golang type for given db type.
func (d *Driver) CheckLocalTypeForField(ctx context.Context, fieldType string, fieldValue interface{}) (gdb.LocalType, error) {
	var typeName string
	match, _ := gregex.MatchString(`(.+?)\((.+)\)`, fieldType)
	if len(match) == 3 {
		typeName = gstr.Trim(match[1])
	} else {
		typeName = fieldType
	}
	typeName = strings.ToLower(typeName)
	switch typeName {
	case
		// For pgsql, int2 = smallint.
		"int2",
		// For pgsql, int4 = integer
		"int4":
		return gdb.LocalTypeInt, nil

	case
		// For pgsql, int8 = bigint
		"int8":
		return gdb.LocalTypeInt64, nil

	case
		"_int2",
		"_int4":
		return gdb.LocalTypeIntSlice, nil

	case
		"_int8":
		return gdb.LocalTypeInt64Slice, nil

	default:
		return d.Core.CheckLocalTypeForField(ctx, fieldType, fieldValue)
	}
}

// ConvertValueForLocal converts value to local Golang type of value according field type name from database.
// The parameter `fieldType` is in lower case, like:
// `float(5,2)`, `unsigned double(5,2)`, `decimal(10,2)`, `char(45)`, `varchar(100)`, etc.
func (d *Driver) ConvertValueForLocal(ctx context.Context, fieldType string, fieldValue interface{}) (interface{}, error) {
	typeName, _ := gregex.ReplaceString(`\(.+\)`, "", fieldType)
	typeName = strings.ToLower(typeName)
	switch typeName {
	// For pgsql, int2 = smallint and int4 = integer.
	case "int2", "int4":
		return gconv.Int(gconv.String(fieldValue)), nil

	// For pgsql, int8 = bigint.
	case "int8":
		return gconv.Int64(gconv.String(fieldValue)), nil

	// Int32 slice.
	case
		"_int2", "_int4":
		return gconv.Ints(
			gstr.ReplaceByMap(gconv.String(fieldValue),
				map[string]string{
					"{": "[",
					"}": "]",
				},
			),
		), nil

	// Int64 slice.
	case
		"_int8":
		return gconv.Int64s(
			gstr.ReplaceByMap(gconv.String(fieldValue),
				map[string]string{
					"{": "[",
					"}": "]",
				},
			),
		), nil

	default:
		return d.Core.ConvertValueForLocal(ctx, fieldType, fieldValue)
	}
}

// DoFilter deals with the sql string before commits it to underlying sql driver.
func (d *Driver) DoFilter(ctx context.Context, link gdb.Link, sql string, args []interface{}) (newSql string, newArgs []interface{}, err error) {
	var index int
	// Convert placeholder char '?' to string "$x".
	newSql, _ = gregex.ReplaceStringFunc(`\?`, sql, func(s string) string {
		index++
		return fmt.Sprintf(`$%d`, index)
	})
	// Handle pgsql jsonb feature support, which contains place-holder char '?'.
	// Refer:
	// https://github.com/gogf/gf/issues/1537
	// https://www.postgresql.org/docs/12/functions-json.html
	newSql, _ = gregex.ReplaceStringFuncMatch(`(::jsonb([^\w\d]*)\$\d)`, newSql, func(match []string) string {
		return fmt.Sprintf(`::jsonb%s?`, match[2])
	})
	newSql, _ = gregex.ReplaceString(` LIMIT (\d+),\s*(\d+)`, ` LIMIT $2 OFFSET $1`, newSql)
	return d.Core.DoFilter(ctx, link, newSql, args)
}

// Tables retrieves and returns the tables of current schema.
// It's mainly used in cli tool chain for automatically generating the models.
func (d *Driver) Tables(ctx context.Context, schema ...string) (tables []string, err error) {
	var (
		result     gdb.Result
		usedSchema = gutil.GetOrDefaultStr(d.GetConfig().Namespace, schema...)
	)
	if usedSchema == "" {
		usedSchema = defaultSchema
	}
	// DO NOT use `usedSchema` as parameter for function `SlaveLink`.
	link, err := d.SlaveLink(schema...)
	if err != nil {
		return nil, err
	}

	useRelpartbound := ""
	if gstr.CompareVersion(d.version(ctx, link), "10") >= 0 {
		useRelpartbound = "AND c.relpartbound IS NULL"
	}

	var query = fmt.Sprintf(`
SELECT
	c.relname
FROM
	pg_class c
INNER JOIN pg_namespace n ON
	c.relnamespace = n.oid
WHERE
	n.nspname = '%s'
	AND c.relkind IN ('r', 'p')
	%s
ORDER BY
	c.relname`,
		usedSchema,
		useRelpartbound,
	)

	query, _ = gregex.ReplaceString(`[\n\r\s]+`, " ", gstr.Trim(query))
	result, err = d.DoSelect(ctx, link, query)
	if err != nil {
		return
	}
	for _, m := range result {
		for _, v := range m {
			tables = append(tables, v.String())
		}
	}
	return
}

// version checks and returns the database version.
func (d *Driver) version(ctx context.Context, link gdb.Link) string {
	result, err := d.DoSelect(ctx, link, "SELECT version();")
	if err != nil {
		return ""
	}
	if len(result) > 0 {
		if v, ok := result[0]["version"]; ok {
			matches := regexp.MustCompile(`PostgreSQL (\d+\.\d+)`).FindStringSubmatch(v.String())
			if len(matches) >= 2 {
				return matches[1]
			}
		}
	}
	return ""
}

// TableFields retrieves and returns the fields' information of specified table of current schema.
func (d *Driver) TableFields(ctx context.Context, table string, schema ...string) (fields map[string]*gdb.TableField, err error) {
	var (
		result     gdb.Result
		link       gdb.Link
		usedSchema = gutil.GetOrDefaultStr(d.GetSchema(), schema...)
		// TODO duplicated `id` result?
		structureSql = fmt.Sprintf(`
SELECT a.attname AS field, t.typname AS type,a.attnotnull as null,
    (case when d.contype is not null then 'pri' else '' end)  as key
      ,ic.column_default as default_value,b.description as comment
      ,coalesce(character_maximum_length, numeric_precision, -1) as length
      ,numeric_scale as scale
FROM pg_attribute a
         left join pg_class c on a.attrelid = c.oid
         left join pg_constraint d on d.conrelid = c.oid and a.attnum = d.conkey[1]
         left join pg_description b ON a.attrelid=b.objoid AND a.attnum = b.objsubid
         left join pg_type t ON a.atttypid = t.oid
         left join information_schema.columns ic on ic.column_name = a.attname and ic.table_name = c.relname
WHERE c.relname = '%s' and a.attisdropped is false and a.attnum > 0
ORDER BY a.attnum`,
			table,
		)
	)
	if link, err = d.SlaveLink(usedSchema); err != nil {
		return nil, err
	}
	structureSql, _ = gregex.ReplaceString(`[\n\r\s]+`, " ", gstr.Trim(structureSql))
	result, err = d.DoSelect(ctx, link, structureSql)
	if err != nil {
		return nil, err
	}
	fields = make(map[string]*gdb.TableField)
	var (
		index = 0
		name  string
		ok    bool
	)
	for _, m := range result {
		name = m["field"].String()
		// Filter duplicated fields.
		if _, ok = fields[name]; ok {
			continue
		}
		fields[name] = &gdb.TableField{
			Index:   index,
			Name:    name,
			Type:    m["type"].String(),
			Null:    !m["null"].Bool(),
			Key:     m["key"].String(),
			Default: m["default_value"].Val(),
			Comment: m["comment"].String(),
		}
		index++
	}
	return fields, nil
}

// DoInsert inserts or updates data forF given table.
func (d *Driver) DoInsert(ctx context.Context, link gdb.Link, table string, list gdb.List, option gdb.DoInsertOption) (result sql.Result, err error) {
	switch option.InsertOption {
	case gdb.InsertOptionSave:
		return nil, gerror.NewCode(
			gcode.CodeNotSupported,
			`Save operation is not supported by pgsql driver`,
		)

	case gdb.InsertOptionReplace:
		return nil, gerror.NewCode(
			gcode.CodeNotSupported,
			`Replace operation is not supported by pgsql driver`,
		)

	case gdb.InsertOptionIgnore:
		return nil, gerror.NewCode(
			gcode.CodeNotSupported,
			`Insert ignore operation is not supported by pgsql driver`,
		)

	case gdb.InsertOptionDefault:
		tableFields, err := d.GetCore().GetDB().TableFields(ctx, table)
		if err == nil {
			for _, field := range tableFields {
				if field.Key == "pri" {
					pkField := *field
					ctx = context.WithValue(ctx, internalPrimaryKeyInCtx, pkField)
					break
				}
			}
		}
	}
	return d.Core.DoInsert(ctx, link, table, list, option)
}

// DoExec commits the sql string and its arguments to underlying driver
// through given link object and returns the execution result.
func (d *Driver) DoExec(ctx context.Context, link gdb.Link, sql string, args ...interface{}) (result sql.Result, err error) {
	var (
		isUseCoreDoExec bool   = false // Check whether the default method needs to be used
		primaryKey      string = ""
		pkField         gdb.TableField
	)

	// Transaction checks.
	if link == nil {
		if tx := gdb.TXFromCtx(ctx, d.GetGroup()); tx != nil {
			// Firstly, check and retrieve transaction link from context.
			link = tx
		} else if link, err = d.MasterLink(); err != nil {
			// Or else it creates one from master node.
			return nil, err
		}
	} else if !link.IsTransaction() {
		// If current link is not transaction link, it checks and retrieves transaction from context.
		if tx := gdb.TXFromCtx(ctx, d.GetGroup()); tx != nil {
			link = tx
		}
	}

	// Check if it is an insert operation with primary key.
	if value := ctx.Value(internalPrimaryKeyInCtx); value != nil {
		var ok bool
		pkField, ok = value.(gdb.TableField)
		if !ok {
			isUseCoreDoExec = true
		}
	} else {
		isUseCoreDoExec = true
	}

	// check if it is an insert operation.
	if !isUseCoreDoExec && pkField.Name != "" && strings.Contains(sql, "INSERT INTO") {
		primaryKey = pkField.Name
		sql += " RETURNING " + primaryKey
	} else {
		// use default DoExec
		return d.Core.DoExec(ctx, link, sql, args...)
	}

	// Only the insert operation with primary key can execute the following code

	if d.GetConfig().ExecTimeout > 0 {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(ctx, d.GetConfig().ExecTimeout)
		defer cancelFunc()
	}

	// Sql filtering.
	sql, args = d.FormatSqlBeforeExecuting(sql, args)
	sql, args, err = d.DoFilter(ctx, link, sql, args)
	if err != nil {
		return nil, err
	}

	// Link execution.
	var out gdb.DoCommitOutput
	out, err = d.DoCommit(ctx, gdb.DoCommitInput{
		Link:          link,
		Sql:           sql,
		Args:          args,
		Stmt:          nil,
		Type:          gdb.SqlTypeQueryContext,
		IsTransaction: link.IsTransaction(),
	})

	if err != nil {
		return nil, err
	}
	affected := len(out.Records)
	if affected > 0 {
		if !strings.Contains(pkField.Type, "int") {
			return Result{
				affected:     int64(affected),
				lastInsertId: 0,
				lastInsertIdError: gerror.NewCodef(
					gcode.CodeNotSupported,
					"LastInsertId is not supported by primary key type: %s", pkField.Type),
			}, nil
		}

		if out.Records[affected-1][primaryKey] != nil {
			lastInsertId := out.Records[affected-1][primaryKey].Int64()
			return Result{
				affected:     int64(affected),
				lastInsertId: lastInsertId,
			}, nil
		}
	}

	return Result{}, nil
}
