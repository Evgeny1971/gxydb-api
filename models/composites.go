// Code generated by SQLBoiler 3.6.1 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"github.com/volatiletech/sqlboiler/queries/qmhelper"
	"github.com/volatiletech/sqlboiler/strmangle"
)

// Composite is an object representing the database table.
type Composite struct {
	ID          int64       `boil:"id" json:"id" toml:"id" yaml:"id"`
	Name        string      `boil:"name" json:"name" toml:"name" yaml:"name"`
	Description null.String `boil:"description" json:"description,omitempty" toml:"description" yaml:"description,omitempty"`

	R *compositeR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L compositeL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var CompositeColumns = struct {
	ID          string
	Name        string
	Description string
}{
	ID:          "id",
	Name:        "name",
	Description: "description",
}

// Generated where

type whereHelperint64 struct{ field string }

func (w whereHelperint64) EQ(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperint64) NEQ(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.NEQ, x) }
func (w whereHelperint64) LT(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperint64) LTE(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LTE, x) }
func (w whereHelperint64) GT(x int64) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperint64) GTE(x int64) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GTE, x) }
func (w whereHelperint64) IN(slice []int64) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereIn(fmt.Sprintf("%s IN ?", w.field), values...)
}

type whereHelperstring struct{ field string }

func (w whereHelperstring) EQ(x string) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.EQ, x) }
func (w whereHelperstring) NEQ(x string) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.NEQ, x) }
func (w whereHelperstring) LT(x string) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.LT, x) }
func (w whereHelperstring) LTE(x string) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.LTE, x) }
func (w whereHelperstring) GT(x string) qm.QueryMod  { return qmhelper.Where(w.field, qmhelper.GT, x) }
func (w whereHelperstring) GTE(x string) qm.QueryMod { return qmhelper.Where(w.field, qmhelper.GTE, x) }
func (w whereHelperstring) IN(slice []string) qm.QueryMod {
	values := make([]interface{}, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return qm.WhereIn(fmt.Sprintf("%s IN ?", w.field), values...)
}

type whereHelpernull_String struct{ field string }

func (w whereHelpernull_String) EQ(x null.String) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, false, x)
}
func (w whereHelpernull_String) NEQ(x null.String) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, true, x)
}
func (w whereHelpernull_String) IsNull() qm.QueryMod    { return qmhelper.WhereIsNull(w.field) }
func (w whereHelpernull_String) IsNotNull() qm.QueryMod { return qmhelper.WhereIsNotNull(w.field) }
func (w whereHelpernull_String) LT(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpernull_String) LTE(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpernull_String) GT(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpernull_String) GTE(x null.String) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

var CompositeWhere = struct {
	ID          whereHelperint64
	Name        whereHelperstring
	Description whereHelpernull_String
}{
	ID:          whereHelperint64{field: "\"composites\".\"id\""},
	Name:        whereHelperstring{field: "\"composites\".\"name\""},
	Description: whereHelpernull_String{field: "\"composites\".\"description\""},
}

// CompositeRels is where relationship names are stored.
var CompositeRels = struct {
	CompositesRooms string
}{
	CompositesRooms: "CompositesRooms",
}

// compositeR is where relationships are stored.
type compositeR struct {
	CompositesRooms CompositesRoomSlice
}

// NewStruct creates a new relationship struct
func (*compositeR) NewStruct() *compositeR {
	return &compositeR{}
}

// compositeL is where Load methods for each relationship are stored.
type compositeL struct{}

var (
	compositeAllColumns            = []string{"id", "name", "description"}
	compositeColumnsWithoutDefault = []string{"name", "description"}
	compositeColumnsWithDefault    = []string{"id"}
	compositePrimaryKeyColumns     = []string{"id"}
)

type (
	// CompositeSlice is an alias for a slice of pointers to Composite.
	// This should generally be used opposed to []Composite.
	CompositeSlice []*Composite

	compositeQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	compositeType                 = reflect.TypeOf(&Composite{})
	compositeMapping              = queries.MakeStructMapping(compositeType)
	compositePrimaryKeyMapping, _ = queries.BindMapping(compositeType, compositeMapping, compositePrimaryKeyColumns)
	compositeInsertCacheMut       sync.RWMutex
	compositeInsertCache          = make(map[string]insertCache)
	compositeUpdateCacheMut       sync.RWMutex
	compositeUpdateCache          = make(map[string]updateCache)
	compositeUpsertCacheMut       sync.RWMutex
	compositeUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// One returns a single composite record from the query.
func (q compositeQuery) One(exec boil.Executor) (*Composite, error) {
	o := &Composite{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(nil, exec, o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for composites")
	}

	return o, nil
}

// All returns all Composite records from the query.
func (q compositeQuery) All(exec boil.Executor) (CompositeSlice, error) {
	var o []*Composite

	err := q.Bind(nil, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to Composite slice")
	}

	return o, nil
}

// Count returns the count of all Composite records in the query.
func (q compositeQuery) Count(exec boil.Executor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRow(exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count composites rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q compositeQuery) Exists(exec boil.Executor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRow(exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if composites exists")
	}

	return count > 0, nil
}

// CompositesRooms retrieves all the composites_room's CompositesRooms with an executor.
func (o *Composite) CompositesRooms(mods ...qm.QueryMod) compositesRoomQuery {
	var queryMods []qm.QueryMod
	if len(mods) != 0 {
		queryMods = append(queryMods, mods...)
	}

	queryMods = append(queryMods,
		qm.Where("\"composites_rooms\".\"composite_id\"=?", o.ID),
	)

	query := CompositesRooms(queryMods...)
	queries.SetFrom(query.Query, "\"composites_rooms\"")

	if len(queries.GetSelect(query.Query)) == 0 {
		queries.SetSelect(query.Query, []string{"\"composites_rooms\".*"})
	}

	return query
}

// LoadCompositesRooms allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for a 1-M or N-M relationship.
func (compositeL) LoadCompositesRooms(e boil.Executor, singular bool, maybeComposite interface{}, mods queries.Applicator) error {
	var slice []*Composite
	var object *Composite

	if singular {
		object = maybeComposite.(*Composite)
	} else {
		slice = *maybeComposite.(*[]*Composite)
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &compositeR{}
		}
		args = append(args, object.ID)
	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &compositeR{}
			}

			for _, a := range args {
				if a == obj.ID {
					continue Outer
				}
			}

			args = append(args, obj.ID)
		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(qm.From(`composites_rooms`), qm.WhereIn(`composites_rooms.composite_id in ?`, args...))
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.Query(e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load composites_rooms")
	}

	var resultSlice []*CompositesRoom
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice composites_rooms")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results in eager load on composites_rooms")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for composites_rooms")
	}

	if singular {
		object.R.CompositesRooms = resultSlice
		for _, foreign := range resultSlice {
			if foreign.R == nil {
				foreign.R = &compositesRoomR{}
			}
			foreign.R.Composite = object
		}
		return nil
	}

	for _, foreign := range resultSlice {
		for _, local := range slice {
			if local.ID == foreign.CompositeID {
				local.R.CompositesRooms = append(local.R.CompositesRooms, foreign)
				if foreign.R == nil {
					foreign.R = &compositesRoomR{}
				}
				foreign.R.Composite = local
				break
			}
		}
	}

	return nil
}

// AddCompositesRooms adds the given related objects to the existing relationships
// of the composite, optionally inserting them as new records.
// Appends related to o.R.CompositesRooms.
// Sets related.R.Composite appropriately.
func (o *Composite) AddCompositesRooms(exec boil.Executor, insert bool, related ...*CompositesRoom) error {
	var err error
	for _, rel := range related {
		if insert {
			rel.CompositeID = o.ID
			if err = rel.Insert(exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "failed to insert into foreign table")
			}
		} else {
			updateQuery := fmt.Sprintf(
				"UPDATE \"composites_rooms\" SET %s WHERE %s",
				strmangle.SetParamNames("\"", "\"", 1, []string{"composite_id"}),
				strmangle.WhereClause("\"", "\"", 2, compositesRoomPrimaryKeyColumns),
			)
			values := []interface{}{o.ID, rel.CompositeID, rel.RoomID, rel.GatewayID}

			if boil.DebugMode {
				fmt.Fprintln(boil.DebugWriter, updateQuery)
				fmt.Fprintln(boil.DebugWriter, values)
			}
			if _, err = exec.Exec(updateQuery, values...); err != nil {
				return errors.Wrap(err, "failed to update foreign table")
			}

			rel.CompositeID = o.ID
		}
	}

	if o.R == nil {
		o.R = &compositeR{
			CompositesRooms: related,
		}
	} else {
		o.R.CompositesRooms = append(o.R.CompositesRooms, related...)
	}

	for _, rel := range related {
		if rel.R == nil {
			rel.R = &compositesRoomR{
				Composite: o,
			}
		} else {
			rel.R.Composite = o
		}
	}
	return nil
}

// Composites retrieves all the records using an executor.
func Composites(mods ...qm.QueryMod) compositeQuery {
	mods = append(mods, qm.From("\"composites\""))
	return compositeQuery{NewQuery(mods...)}
}

// FindComposite retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindComposite(exec boil.Executor, iD int64, selectCols ...string) (*Composite, error) {
	compositeObj := &Composite{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"composites\" where \"id\"=$1", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(nil, exec, compositeObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from composites")
	}

	return compositeObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Composite) Insert(exec boil.Executor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no composites provided for insertion")
	}

	var err error

	nzDefaults := queries.NonZeroDefaultSet(compositeColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	compositeInsertCacheMut.RLock()
	cache, cached := compositeInsertCache[key]
	compositeInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			compositeAllColumns,
			compositeColumnsWithDefault,
			compositeColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(compositeType, compositeMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(compositeType, compositeMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"composites\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"composites\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRow(cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.Exec(cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into composites")
	}

	if !cached {
		compositeInsertCacheMut.Lock()
		compositeInsertCache[key] = cache
		compositeInsertCacheMut.Unlock()
	}

	return nil
}

// Update uses an executor to update the Composite.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Composite) Update(exec boil.Executor, columns boil.Columns) (int64, error) {
	var err error
	key := makeCacheKey(columns, nil)
	compositeUpdateCacheMut.RLock()
	cache, cached := compositeUpdateCache[key]
	compositeUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			compositeAllColumns,
			compositePrimaryKeyColumns,
		)

		if len(wl) == 0 {
			return 0, errors.New("models: unable to update composites, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"composites\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, compositePrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(compositeType, compositeMapping, append(wl, compositePrimaryKeyColumns...))
		if err != nil {
			return 0, err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, values)
	}
	var result sql.Result
	result, err = exec.Exec(cache.query, values...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update composites row")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by update for composites")
	}

	if !cached {
		compositeUpdateCacheMut.Lock()
		compositeUpdateCache[key] = cache
		compositeUpdateCacheMut.Unlock()
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values.
func (q compositeQuery) UpdateAll(exec boil.Executor, cols M) (int64, error) {
	queries.SetUpdate(q.Query, cols)

	result, err := q.Query.Exec(exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all for composites")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected for composites")
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o CompositeSlice) UpdateAll(exec boil.Executor, cols M) (int64, error) {
	ln := int64(len(o))
	if ln == 0 {
		return 0, nil
	}

	if len(cols) == 0 {
		return 0, errors.New("models: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), compositePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"composites\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, compositePrimaryKeyColumns, len(o)))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}
	result, err := exec.Exec(sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all in composite slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected all in update all composite")
	}
	return rowsAff, nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Composite) Upsert(exec boil.Executor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no composites provided for upsert")
	}

	nzDefaults := queries.NonZeroDefaultSet(compositeColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	compositeUpsertCacheMut.RLock()
	cache, cached := compositeUpsertCache[key]
	compositeUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			compositeAllColumns,
			compositeColumnsWithDefault,
			compositeColumnsWithoutDefault,
			nzDefaults,
		)
		update := updateColumns.UpdateColumnSet(
			compositeAllColumns,
			compositePrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert composites, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(compositePrimaryKeyColumns))
			copy(conflict, compositePrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"composites\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(compositeType, compositeMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(compositeType, compositeMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}
	if len(cache.retMapping) != 0 {
		err = exec.QueryRow(cache.query, vals...).Scan(returns...)
		if err == sql.ErrNoRows {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.Exec(cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert composites")
	}

	if !cached {
		compositeUpsertCacheMut.Lock()
		compositeUpsertCache[key] = cache
		compositeUpsertCacheMut.Unlock()
	}

	return nil
}

// Delete deletes a single Composite record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Composite) Delete(exec boil.Executor) (int64, error) {
	if o == nil {
		return 0, errors.New("models: no Composite provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), compositePrimaryKeyMapping)
	sql := "DELETE FROM \"composites\" WHERE \"id\"=$1"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}
	result, err := exec.Exec(sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete from composites")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by delete for composites")
	}

	return rowsAff, nil
}

// DeleteAll deletes all matching rows.
func (q compositeQuery) DeleteAll(exec boil.Executor) (int64, error) {
	if q.Query == nil {
		return 0, errors.New("models: no compositeQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	result, err := q.Query.Exec(exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from composites")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for composites")
	}

	return rowsAff, nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o CompositeSlice) DeleteAll(exec boil.Executor) (int64, error) {
	if len(o) == 0 {
		return 0, nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), compositePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"composites\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, compositePrimaryKeyColumns, len(o))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}
	result, err := exec.Exec(sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from composite slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for composites")
	}

	return rowsAff, nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Composite) Reload(exec boil.Executor) error {
	ret, err := FindComposite(exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *CompositeSlice) ReloadAll(exec boil.Executor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := CompositeSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), compositePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"composites\".* FROM \"composites\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, compositePrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(nil, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in CompositeSlice")
	}

	*o = slice

	return nil
}

// CompositeExists checks if the Composite row exists.
func CompositeExists(exec boil.Executor, iD int64) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"composites\" where \"id\"=$1 limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, iD)
	}
	row := exec.QueryRow(sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if composites exists")
	}

	return exists, nil
}
