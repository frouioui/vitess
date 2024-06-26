/*
Copyright 2019 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vttest

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// FieldGenerator is a callback that generates the value of a random field in
// when seeding the database with random data. `name` is the name of the column
// where the field belongs, `t` is its SQL tyoe, and `rng` is the RNG currently
// being used, as to ensure reproducible generation between runs.
// A FieldGenerator must return the raw SQL data for the field, ready to be
// placed into a SQL statement. The returned value will _NOT_ be escaped.
type FieldGenerator func(name, t string) (string, error)

// SeedConfig are the settings to enable the initialization of the
// local cluster with random data. This struct must be set in Config
// before Setup() is called.
type SeedConfig struct {
	// RngSeed is the seed uset to initialize the random number
	// generator that will be used to fill the database with
	// random data. Multiple runs with the same seed will result
	// in the same initial data
	RngSeed int

	// MinSize is the minimum number of initial rows in each tale shard
	MinSize int

	// MaxSize is the maximum number of initial rows in each table shard
	MaxSize int

	// NullProbability is the chance to initialize a field a NULL value.
	// Only applies to fields that can contain NULL values
	NullProbability float64

	// RandomField is a callback to generate the value of a random field
	RandomField FieldGenerator
}

const batchInsertSize = 1000

func (db *LocalCluster) batchInsert(dbname, table string, fields []string, rows [][]string) error {
	var (
		fieldNames = strings.Join(fields, ",")
		values     []string
		sql        string
	)

	for _, row := range rows {
		values = append(values, "("+strings.Join(row, ",")+")")
	}

	sql = fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s",
		table, fieldNames, strings.Join(values, ","),
	)

	return db.Execute([]string{sql}, dbname)
}

func (db *LocalCluster) randomField(name, t string, allowNull bool) (string, error) {
	if allowNull && rand.Float64() < db.Seed.NullProbability {
		return "NULL", nil
	}
	return db.Seed.RandomField(name, t)
}

func (db *LocalCluster) populateTable(dbname, table string) error {
	fieldInfo, err := db.Query(fmt.Sprintf("DESCRIBE %s", table), dbname, 1024)
	if err != nil {
		return err
	}

	var (
		minRows    = db.Seed.MinSize
		maxRows    = db.Seed.MaxSize
		numRows    = rand.IntN(maxRows-minRows) + minRows
		rows       [][]string
		fieldNames []string
	)

	for i := 0; i < numRows; i++ {
		var fields []string
		for _, row := range fieldInfo.Rows {
			fieldName := row[0].ToString()
			fieldType := row[1].ToString()
			allowNull := row[2].ToString() == "YES"

			f, err := db.randomField(fieldName, fieldType, allowNull)
			if err != nil {
				return err
			}
			fields = append(fields, f)
		}
		rows = append(rows, fields)
	}

	for _, row := range fieldInfo.Rows {
		fieldNames = append(fieldNames, row[0].ToString())
	}

	for i := 0; i < len(rows); i += batchInsertSize {
		if err := db.batchInsert(dbname, table, fieldNames, rows); err != nil {
			return err
		}
	}

	return nil
}

func (db *LocalCluster) populateShard(dbname string) error {
	q, err := db.Query("SHOW TABLES", dbname, 1024)
	if err != nil {
		return err
	}

	for _, row := range q.Rows {
		if err := db.populateTable(dbname, row[0].ToString()); err != nil {
			return err
		}
	}
	return nil
}

func (db *LocalCluster) populateWithRandomData() error {
	for _, kpb := range db.Topology.Keyspaces {
		for _, dbname := range db.shardNames(kpb) {
			if err := db.populateShard(dbname); err != nil {
				return err
			}
		}
	}
	return nil
}
