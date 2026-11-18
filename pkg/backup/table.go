package backup

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"

	"xorm.io/xorm"
	"xorm.io/xorm/convert"
	"xorm.io/xorm/schemas"

	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

type columnCodec int

const (
	columnCodecInt columnCodec = iota + 1
	columnCodecUint
	columnCodecFloat
	columnCodecBool
	columnCodecString
	columnCodecBytes // raw database bytes (blob, xorm conversion or json columns), base64 encoded in backup
)

var conversionType = reflect.TypeOf((*convert.Conversion)(nil)).Elem()

// tableColumn represents a column of a registered table
type tableColumn struct {
	name       string
	fieldIndex []int
	fieldType  reflect.Type
	codec      columnCodec
	pointer    bool // whether the field is a pointer to a basic type (nullable column)
}

// tableSchema represents the schema of a registered table
type tableSchema struct {
	registration  *datastore.TableRegistration
	name          string
	fullName      string
	beanType      reflect.Type
	columns       []*tableColumn
	columnsByName map[string]*tableColumn
	primaryKeys   []string
	database      datastore.Database
	accessor      datastore.RawDatabaseAccessor
}

func (t *tableSchema) columnNames() []string {
	names := make([]string, len(t.columns))

	for i, column := range t.columns {
		names[i] = column.name
	}

	return names
}

func (t *tableSchema) engine() *xorm.Engine {
	return t.accessor.MasterEngine()
}

// loadTableSchemas returns the schemas of all the registered tables
func loadTableSchemas() ([]*tableSchema, error) {
	registrations := datastore.GetAllTables()
	schemas := make([]*tableSchema, 0, len(registrations))

	for _, registration := range registrations {
		store := registration.Store()

		if store == nil || store.Count() < 1 {
			return nil, fmt.Errorf("data store of %s table is not initialized", registration.Description)
		}

		if store.Count() > 1 {
			return nil, fmt.Errorf("backup of sharded data store (%s table) is not supported", registration.Description)
		}

		database := store.Get(0)
		accessor, ok := database.(datastore.RawDatabaseAccessor)

		if !ok {
			return nil, fmt.Errorf("database of %s table does not support backup", registration.Description)
		}

		schema, err := newTableSchema(registration, database, accessor)

		if err != nil {
			return nil, err
		}

		schemas = append(schemas, schema)
	}

	return schemas, nil
}

func newTableSchema(registration *datastore.TableRegistration, database datastore.Database, accessor datastore.RawDatabaseAccessor) (*tableSchema, error) {
	engine := accessor.MasterEngine()
	bean := registration.NewBean()
	tableInfo, err := engine.TableInfo(bean)

	if err != nil {
		return nil, err
	}

	beanType := reflect.TypeOf(bean).Elem()
	schema := &tableSchema{
		registration:  registration,
		name:          tableInfo.Name,
		fullName:      engine.TableName(bean, true),
		beanType:      beanType,
		columnsByName: make(map[string]*tableColumn),
		primaryKeys:   tableInfo.PrimaryKeys,
		database:      database,
		accessor:      accessor,
	}

	for _, column := range tableInfo.Columns() {
		if column.MapType == schemas.ONLYFROMDB || len(column.FieldIndex) < 1 {
			continue
		}

		fieldType := beanType.FieldByIndex(column.FieldIndex).Type

		if fieldType.Kind() == reflect.Interface {
			// embedded interfaces (e.g. core.TokenRecord in models.TokenRecord) never hold data in database
			continue
		}

		codec, pointer, err := getColumnCodec(fieldType)

		if err != nil {
			return nil, fmt.Errorf("%w: column \"%s\" of table \"%s\"", err, column.Name, tableInfo.Name)
		}

		tableColumn := &tableColumn{
			name:       column.Name,
			fieldIndex: column.FieldIndex,
			fieldType:  fieldType,
			codec:      codec,
			pointer:    pointer,
		}

		schema.columns = append(schema.columns, tableColumn)
		schema.columnsByName[column.Name] = tableColumn
	}

	return schema, nil
}

func getColumnCodec(fieldType reflect.Type) (columnCodec, bool, error) {
	if fieldType.Implements(conversionType) || reflect.PointerTo(fieldType).Implements(conversionType) {
		return columnCodecBytes, false, nil
	}

	if fieldType.Kind() == reflect.Pointer {
		codec, pointer, err := getColumnCodec(fieldType.Elem())

		if err != nil || pointer || codec == columnCodecBytes {
			return 0, false, ErrBackupUnsupportedColumnType
		}

		return codec, true, nil
	}

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return columnCodecInt, false, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return columnCodecUint, false, nil
	case reflect.Float32, reflect.Float64:
		return columnCodecFloat, false, nil
	case reflect.Bool:
		return columnCodecBool, false, nil
	case reflect.String:
		return columnCodecString, false, nil
	case reflect.Slice, reflect.Map:
		return columnCodecBytes, false, nil
	case reflect.Struct:
		if fieldType.PkgPath() == "time" {
			return 0, false, ErrBackupUnsupportedColumnType
		}

		return columnCodecBytes, false, nil
	default:
		return 0, false, ErrBackupUnsupportedColumnType
	}
}

// normalizeDatabaseValue converts the raw value returned by the database driver to the canonical typed value
// (int64, uint64, float64, bool, string, []byte or nil) of the column
func normalizeDatabaseValue(column *tableColumn, value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch column.codec {
	case columnCodecInt:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int32:
			return int64(v), nil
		case int:
			return int64(v), nil
		case uint64:
			if v > math.MaxInt64 {
				return nil, fmt.Errorf("value of column \"%s\" overflows int64", column.name)
			}
			return int64(v), nil
		case bool:
			if v {
				return int64(1), nil
			}
			return int64(0), nil
		case float64:
			return int64(v), nil
		case []byte:
			return strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
		case string:
			return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		}
	case columnCodecUint:
		switch v := value.(type) {
		case int64:
			if v < 0 {
				return nil, fmt.Errorf("value of column \"%s\" is negative", column.name)
			}
			return uint64(v), nil
		case uint64:
			return v, nil
		case []byte:
			return strconv.ParseUint(strings.TrimSpace(string(v)), 10, 64)
		case string:
			return strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		}
	case columnCodecFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case []byte:
			return strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		case string:
			return strconv.ParseFloat(strings.TrimSpace(v), 64)
		}
	case columnCodecBool:
		switch v := value.(type) {
		case bool:
			return v, nil
		case int64:
			return v != 0, nil
		case []byte:
			return parseDatabaseBool(string(v))
		case string:
			return parseDatabaseBool(v)
		}
	case columnCodecString:
		switch v := value.(type) {
		case string:
			return v, nil
		case []byte:
			return string(v), nil
		case int64:
			return strconv.FormatInt(v, 10), nil
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		case bool:
			return strconv.FormatBool(v), nil
		}
	case columnCodecBytes:
		switch v := value.(type) {
		case []byte:
			return append([]byte{}, v...), nil
		case string:
			return []byte(v), nil
		}
	}

	return nil, fmt.Errorf("unexpected value type %T of column \"%s\"", value, column.name)
}

func parseDatabaseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes":
		return true, nil
	case "0", "f", "false", "n", "no", "":
		return false, nil
	}

	return false, fmt.Errorf("invalid boolean value \"%s\"", value)
}

// encodeRow writes a row (canonical typed values in column order) as a single json line
func encodeRow(buffer *bytes.Buffer, columns []*tableColumn, values []any) error {
	buffer.WriteByte('{')

	for i, column := range columns {
		if i > 0 {
			buffer.WriteByte(',')
		}

		name, _ := json.Marshal(column.name)
		buffer.Write(name)
		buffer.WriteByte(':')

		switch v := values[i].(type) {
		case nil:
			buffer.WriteString("null")
		case int64:
			buffer.WriteString(strconv.FormatInt(v, 10))
		case uint64:
			buffer.WriteString(strconv.FormatUint(v, 10))
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("value of column \"%s\" is not a finite number", column.name)
			}
			buffer.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
		case bool:
			buffer.WriteString(strconv.FormatBool(v))
		case string:
			data, err := json.Marshal(v)

			if err != nil {
				return err
			}

			buffer.Write(data)
		case []byte:
			data, _ := json.Marshal(base64.StdEncoding.EncodeToString(v))
			buffer.Write(data)
		default:
			return fmt.Errorf("unexpected value type %T of column \"%s\"", v, column.name)
		}
	}

	buffer.WriteString("}\n")

	return nil
}

// decodeRow parses a json line into canonical typed values by column name, it returns an error if the line contains unknown columns
func decodeRow(schema *tableSchema, line []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.UseNumber()

	raw := make(map[string]any)

	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}

	if decoder.More() {
		return nil, fmt.Errorf("unexpected data after json object")
	}

	values := make(map[string]any, len(raw))

	for name, value := range raw {
		column, exists := schema.columnsByName[name]

		if !exists {
			return nil, fmt.Errorf("%w: unknown column \"%s\" of table \"%s\" (the backup was probably created by a newer version)", ErrBackupIncompatible, name, schema.name)
		}

		typedValue, err := decodeJsonValue(column, value)

		if err != nil {
			return nil, err
		}

		values[name] = typedValue
	}

	return values, nil
}

func decodeJsonValue(column *tableColumn, value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	invalidValueError := fmt.Errorf("invalid value of column \"%s\"", column.name)

	switch column.codec {
	case columnCodecInt:
		if number, ok := value.(json.Number); ok {
			return strconv.ParseInt(number.String(), 10, 64)
		}
	case columnCodecUint:
		if number, ok := value.(json.Number); ok {
			return strconv.ParseUint(number.String(), 10, 64)
		}
	case columnCodecFloat:
		if number, ok := value.(json.Number); ok {
			return strconv.ParseFloat(number.String(), 64)
		}
	case columnCodecBool:
		if boolValue, ok := value.(bool); ok {
			return boolValue, nil
		}
	case columnCodecString:
		if stringValue, ok := value.(string); ok {
			return stringValue, nil
		}
	case columnCodecBytes:
		if stringValue, ok := value.(string); ok {
			data, err := base64.StdEncoding.DecodeString(stringValue)

			if err != nil {
				return nil, invalidValueError
			}

			return data, nil
		}
	}

	return nil, invalidValueError
}

// newBeanFromValues creates a new model instance and fills the fields with the canonical typed values
func newBeanFromValues(schema *tableSchema, values map[string]any) (any, error) {
	beanValue := reflect.New(schema.beanType)

	for _, column := range schema.columns {
		value, exists := values[column.name]

		if !exists || value == nil {
			continue
		}

		field := beanValue.Elem().FieldByIndex(column.fieldIndex)

		if err := setFieldValue(field, column, value); err != nil {
			return nil, err
		}
	}

	return beanValue.Interface(), nil
}

func setFieldValue(field reflect.Value, column *tableColumn, value any) error {
	if column.pointer {
		field.Set(reflect.New(field.Type().Elem()))
		field = field.Elem()
	}

	switch column.codec {
	case columnCodecInt:
		v := value.(int64)

		if field.OverflowInt(v) {
			return fmt.Errorf("value of column \"%s\" overflows", column.name)
		}

		field.SetInt(v)
	case columnCodecUint:
		v := value.(uint64)

		if field.OverflowUint(v) {
			return fmt.Errorf("value of column \"%s\" overflows", column.name)
		}

		field.SetUint(v)
	case columnCodecFloat:
		field.SetFloat(value.(float64))
	case columnCodecBool:
		field.SetBool(value.(bool))
	case columnCodecString:
		field.SetString(value.(string))
	case columnCodecBytes:
		data := value.([]byte)

		if field.Kind() == reflect.Pointer && field.Type().Implements(conversionType) {
			field.Set(reflect.New(field.Type().Elem()))
			return field.Interface().(convert.Conversion).FromDB(data)
		} else if field.Addr().Type().Implements(conversionType) {
			return field.Addr().Interface().(convert.Conversion).FromDB(data)
		} else if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			field.SetBytes(data)
		} else if len(data) > 0 {
			return json.Unmarshal(data, field.Addr().Interface())
		}
	}

	return nil
}

// getFieldValues returns the canonical typed values of all the columns of a model instance
func getFieldValues(schema *tableSchema, bean any) ([]any, error) {
	beanValue := reflect.ValueOf(bean).Elem()
	values := make([]any, len(schema.columns))

	for i, column := range schema.columns {
		field := beanValue.FieldByIndex(column.fieldIndex)

		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				values[i] = nil
				continue
			}

			if column.pointer {
				field = field.Elem()
			}
		}

		switch column.codec {
		case columnCodecInt:
			values[i] = field.Int()
		case columnCodecUint:
			values[i] = field.Uint()
		case columnCodecFloat:
			values[i] = field.Float()
		case columnCodecBool:
			values[i] = field.Bool()
		case columnCodecString:
			values[i] = field.String()
		case columnCodecBytes:
			if field.Kind() == reflect.Pointer && field.Type().Implements(conversionType) {
				data, err := field.Interface().(convert.Conversion).ToDB()

				if err != nil {
					return nil, err
				}

				values[i] = data
			} else if field.Addr().Type().Implements(conversionType) {
				data, err := field.Addr().Interface().(convert.Conversion).ToDB()

				if err != nil {
					return nil, err
				}

				values[i] = data
			} else if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
				values[i] = field.Bytes()
			} else {
				data, err := json.Marshal(field.Interface())

				if err != nil {
					return nil, err
				}

				values[i] = data
			}
		}
	}

	return values, nil
}

// queryer represents a database handle which can run queries (*sql.DB or *sql.Tx)
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// dumpTable writes all the rows of the table as json lines into the writer (ordered by primary keys), and returns the row count.
// The rowHandler (optional) is called for every row with the canonical typed values.
func dumpTable(ctx context.Context, db queryer, schema *tableSchema, writer io.Writer, rowHandler func(values []any) error) (int64, error) {
	engine := schema.engine()
	quotedColumns := make([]string, len(schema.columns))

	for i, column := range schema.columns {
		quotedColumns[i] = engine.Quote(column.name)
	}

	query := "SELECT " + strings.Join(quotedColumns, ", ") + " FROM " + engine.Quote(schema.fullName)

	if len(schema.primaryKeys) > 0 {
		quotedPrimaryKeys := make([]string, len(schema.primaryKeys))

		for i, primaryKey := range schema.primaryKeys {
			quotedPrimaryKeys[i] = engine.Quote(primaryKey)
		}

		query = query + " ORDER BY " + strings.Join(quotedPrimaryKeys, ", ")
	}

	rows, err := db.QueryContext(ctx, query)

	if err != nil {
		return 0, err
	}

	defer rows.Close()

	rawValues := make([]any, len(schema.columns))
	scanTargets := make([]any, len(schema.columns))

	for i := range rawValues {
		scanTargets[i] = &rawValues[i]
	}

	buffer := &bytes.Buffer{}
	rowCount := int64(0)

	for rows.Next() {
		if err := rows.Scan(scanTargets...); err != nil {
			return rowCount, err
		}

		values := make([]any, len(schema.columns))

		for i, column := range schema.columns {
			values[i], err = normalizeDatabaseValue(column, rawValues[i])

			if err != nil {
				return rowCount, fmt.Errorf("table \"%s\": %w", schema.name, err)
			}
		}

		buffer.Reset()

		if err := encodeRow(buffer, schema.columns, values); err != nil {
			return rowCount, fmt.Errorf("table \"%s\": %w", schema.name, err)
		}

		if _, err := writer.Write(buffer.Bytes()); err != nil {
			return rowCount, err
		}

		if rowHandler != nil {
			if err := rowHandler(values); err != nil {
				return rowCount, err
			}
		}

		rowCount++
	}

	if err := rows.Err(); err != nil {
		return rowCount, err
	}

	return rowCount, nil
}

// beginSnapshotTransaction starts a read transaction which provides a consistent snapshot of all the tables in the database
func beginSnapshotTransaction(ctx context.Context, accessor datastore.RawDatabaseAccessor) (*sql.Tx, error) {
	sqlDB := accessor.MasterEngine().DB().DB

	switch accessor.DatabaseType() {
	case settings.MySqlDbType, settings.PostgresDbType:
		return sqlDB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelRepeatableRead,
			ReadOnly:  true,
		})
	default:
		// sqlite holds a shared lock from the first read until the transaction ends, so all the reads see the same snapshot
		return sqlDB.BeginTx(ctx, nil)
	}
}

// countTableRows returns the row count of the table, and -1 if the table does not exist
func countTableRows(schema *tableSchema) (int64, error) {
	engine := schema.engine()
	exists, err := engine.IsTableExist(schema.name)

	if err != nil {
		return 0, err
	}

	if !exists {
		return -1, nil
	}

	results, err := engine.QueryString("SELECT COUNT(*) AS row_count FROM " + engine.Quote(schema.fullName))

	if err != nil {
		return 0, err
	}

	if len(results) < 1 {
		return 0, nil
	}

	for _, value := range results[0] {
		return strconv.ParseInt(value, 10, 64)
	}

	return 0, nil
}
